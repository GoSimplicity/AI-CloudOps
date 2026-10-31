"""根目录 env.example 与本分支代码行为的一致性检查（r3inc P3-7 / P3-8）。

只读取仓库里的 env.example、config/webhook.yaml、deploy/release/compose.yaml 和 Go 源码中的默认值 / 拼接格式，
不连网、不启动进程；群机器人 token 用运行时生成的合成值。
"""
import json
import os
import random
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[2]
_DEFAULT_PORTS = {"http": 80, "https": 443}


_GODOTENV_SPACE = "\t\v\f\r \x85\xa0"  # parser.go isSpace：不含换行
_EXPAND_VAR = re.compile(r"(\\)?(\$)(\()?\{?([A-Z0-9_]+)?\}?")


def _go_unicode_isspace(ch):
    # unicode.IsSpace：Latin-1 内只有这 8 个；Python 的 isspace 还把 \x1c-\x1f 算空白，要排除
    return ch in "\t\n\v\f\r \x85\xa0" or (ord(ch) > 0xff and ch.isspace())


def _godotenv_expand(value, seen):
    def repl(m):
        sub = _EXPAND_VAR.search(m.group(0))
        if sub.group(1) == "\\" or sub.group(2) == "(":  # 原实现比较的是第 2 组（恒为 "$"），这一支只会因反斜杠命中
            return m.group(0)[1:]
        if sub.group(4):
            return seen.get(sub.group(4), "")
        return m.group(0)
    return _EXPAND_VAR.sub(repl, value)


def _godotenv_escapes(value):
    value = re.sub(r"\\.", lambda m: {"n": "\n", "r": "\r"}.get(m.group(0)[1], m.group(0)), value)
    return re.sub(r"\\([^$])", r"\1", value)


def _godotenv_parse(text):
    """逐条移植 github.com/joho/godotenv v1.5.1 parser.go（main.go 用 godotenv.Load() 读 .env）。

    重复键后者生效；可带 export 前缀；分隔符 = 或 :；未加引号的值从行尾往前找第一个前面是空白的 #，
    其后为注释（a#b 不是注释）；单引号原样，双引号展开 \\n \\r 和其他反斜杠转义；
    未加引号和双引号的值展开 $NAME / ${NAME}，只看前面已解析的键，找不到为空串。
    格式错误时抛 ValueError（godotenv.Load 同样报错）。
    """
    src = text.replace("\r\n", "\n")
    out = {}
    while True:
        # getStatementStart：跳过空白（含换行）和 # 注释行
        while True:
            pos = next((i for i, ch in enumerate(src) if not _go_unicode_isspace(ch)), -1)
            if pos == -1:
                return out
            src = src[pos:]
            if src[0] != "#":
                break
            nl = src.find("\n")
            if nl == -1:
                return out
            src = src[nl:]
        # locateKeyName：键名按字节逐个检查（与 Go 的 for i, char := range []byte 一致）
        src = src.lstrip(_GODOTENV_SPACE)
        if src.startswith("export") and src[6:7] and src[6] in _GODOTENV_SPACE:
            src = src[6:].lstrip(_GODOTENV_SPACE)
        raw = src.encode("utf-8")
        key, offset = b"", 0
        for i, c in enumerate(raw):
            ch = chr(c)  # rune(byte)：按 Latin-1 解释单个字节
            if ch in _GODOTENV_SPACE:
                continue
            if ch in "=:":
                key, offset = raw[:i], i + 1
                break
            if ch == "_" or ch == "." or ch.isalpha() or ch.isnumeric():
                continue
            raise ValueError(f"unexpected character {ch!r} in variable name")
        if not raw:
            raise ValueError("zero length string")
        key = key.decode("utf-8", "replace").rstrip()
        src = raw[offset:].decode("utf-8").lstrip(_GODOTENV_SPACE)
        # extractVarValue
        if src[:1] in ('"', "'"):
            quote = src[0]
            for i in range(1, len(src)):
                if src[i] != quote or src[i - 1] == "\\":
                    continue
                value = src[:i].strip(quote)
                if quote == '"':
                    value = _godotenv_expand(_godotenv_escapes(value), out)
                out[key] = value
                src = src[i + 1:]
                break
            else:
                raise ValueError("unterminated quoted value")
            continue
        eol = next((i for i, ch in enumerate(src) if ch in "\n\r"), len(src))
        line, end = src[:eol], eol
        for i in range(end - 1, 0, -1):
            if line[i] == "#" and line[i - 1] in _GODOTENV_SPACE:
                end = i
                break
        out[key] = _godotenv_expand(line[:end].strip(_GODOTENV_SPACE), out)
        src = src[eol:]


def _env_example(text=None):
    """env.example 复制成 .env 后 backend 实际读到的值（godotenv v1.5.1 语义，见 _godotenv_parse）"""
    return _godotenv_parse((ROOT / "env.example").read_text(encoding="utf-8") if text is None else text)


def _ascii_port(url_or_netloc_port):
    """URL 里显式写出的端口必须是 ASCII 数字：Python 3.9/3.10 的 urllib 会把全角数字 ８８８９ 当成 8889。"""
    import re
    if url_or_netloc_port is not None and not re.fullmatch(r"[0-9]+", url_or_netloc_port):
        raise ValueError(f"端口只能是 ASCII 数字: {url_or_netloc_port!r}")


def _netloc_port(netloc):
    # netloc 里冒号后的端口原文（IPv6 字面量跳过方括号）；没写端口返回 None
    hostport = netloc.rpartition("@")[2]
    tail = hostport.rpartition("]")[2] if hostport.startswith("[") else hostport
    return tail.rpartition(":")[2] if ":" in tail else None


def _has_http_scheme_fold(base):
    """对齐 w1 119c332 alert_cache.go 的 hasPrefixFold(base, "http://") || hasPrefixFold(base, "https://")：
    按字节长度切出前缀再 strings.EqualFold。前缀全是 ASCII，等价于只做 ASCII 大小写折叠；
    ſ（U+017F）、全角字母等占的字节数不同，不会被认成 http(s)://。"""
    b = base.encode("utf-8")
    return b[:7].lower() == b"http://" or b[:8].lower() == b"https://"


def _split_alert_url(url):
    """按 alert_cache.go buildWebhookURL 的规则拆地址（w1 119c332 起）：TrimSpace 后以 http:// 或 https:// 开头
    （不区分大小写，HTTP://、Https:// 也算）才是完整 URL，不含 :// 时按 host:port 走 http；其他带 :// 的写法
    （ftp://、httpſ:// 等）在代码里会拼出 http://ftp://... 的错误地址，这里直接报错。
    注：合入 w1 前本分支 db97f42 的代码仍用区分大小写的 strings.HasPrefix。"""
    base = url.strip()
    if _has_http_scheme_fold(base):
        parts = urlsplit(base)  # urlsplit 会把 scheme 转成小写
    elif "://" in base:
        raise ValueError(f"scheme 只能是 http/https（不区分大小写）: {base.split('://', 1)[0]!r}")
    else:
        parts = urlsplit("http://" + base)
    _ascii_port(_netloc_port(parts.netloc))
    return parts


def _url_port(url):
    parts = _split_alert_url(url)
    scheme = parts.scheme
    port = parts.port if parts.port is not None else _DEFAULT_PORTS[scheme]
    if not 1 <= port <= 65535:
        raise ValueError(f"端口越界: {port}")
    return port


_RECEIVE_PATH_GO = "internal/prometheus/webhook/api/handler.go"
_HTTP_SD_GO = "internal/not_auth/api/handler.go"


def _gin_routes(rel, method):
    """只读解析 Gin 路由：<g> := <engine>.Group("<前缀>") 与 <g>.<METHOD>("<子路径>", ...)，返回完整路径列表。"""
    text = (ROOT / rel).read_text(encoding="utf-8")
    groups = dict(re.findall(r'\b([A-Za-z_][A-Za-z0-9_]*)\s*:?=\s*[A-Za-z_][A-Za-z0-9_.]*\.Group\(\s*"([^"]*)"', text))
    routes = [groups[g] + sub for g, sub in re.findall(r'\b([A-Za-z_][A-Za-z0-9_]*)\.' + method + r'\(\s*"([^"]*)"', text)
              if g in groups]
    if not routes:
        raise AssertionError(f"{rel} 里没解析到 {method} 路由")
    return routes


def _alert_receive_route():
    routes = [r for r in _gin_routes(_RECEIVE_PATH_GO, "POST") if r.endswith("/receive")]
    if len(routes) != 1:
        raise AssertionError(f"webhook 告警接收路由应恰好 1 个，实际 {routes}")
    return routes[0]


def _alert_target(addr):
    """按 alert_cache.go buildWebhookURL 算出 Alertmanager 实际推送的 (scheme, host, port, path)。

    完整 URL：TrimRight "/"，已含 /api/v1/alerts/receive 就原样使用，否则在末尾追加；host:port：http://<base>/api/v1/alerts/receive。
    """
    receive = "/api/v1/alerts/receive"
    base = addr.strip()
    parts = _split_alert_url(base)  # 校验 scheme 与端口
    if "://" in base:
        u = base.rstrip("/")
        full = u if receive in u else u + receive
    else:
        full = "http://" + base + receive
    parts = urlsplit(full)
    return parts.scheme, parts.hostname, _url_port(full), parts.path


def _release_default_alert_addr():
    """发布栈里的告警地址期望值：compose 的 ${PROMETHEUS_ALERT_WEBHOOK_ADDR:-默认}；本分支 compose 没有时用 alertmanager.yml。"""
    compose = (ROOT / "deploy/release/compose.yaml").read_text(encoding="utf-8")
    m = re.search(r"PROMETHEUS_ALERT_WEBHOOK_ADDR:\s*\$\{PROMETHEUS_ALERT_WEBHOOK_ADDR:-([^}]+)\}", compose)
    if m:
        return m.group(1).strip()
    found = re.findall(r"^\s*-?\s*url:\s*(\S+)\s*$", (ROOT / "deploy/release/alertmanager.yml").read_text(encoding="utf-8"), re.M)
    if len(found) != 1:
        raise AssertionError(f"alertmanager.yml 的 webhook url 应恰好 1 个，实际 {len(found)}")
    return found[0]


def _go_default(key):
    # internal/config/loader.go 里 <recv>.SetDefault("<key>", "<value>") 的值（只读解析）；
    # 接收者不限于 viper：sched/w1-mantou 的 webhook 默认值写成 v.SetDefault("webhook.port", "8889")
    text = (ROOT / "internal/config/loader.go").read_text(encoding="utf-8")
    found = re.findall(r'\b[A-Za-z_][A-Za-z0-9_]*\.SetDefault\(\s*"' + re.escape(key) + r'",\s*"([^"]*)"\s*\)', text)
    if len(found) != 1:
        raise AssertionError(f"loader.go 中 {key} 默认值应恰好 1 处，实际 {len(found)}")
    return found[0]


# webhook.port 的值：纯 ASCII 数字（可加单/双引号）；行内注释的 # 前必须有空白（YAML 规则，8889#oops 是一个标量）
_PORT_VALUE = r"""(?:"([0-9]+)"|'([0-9]+)'|([0-9]+))(?:[ \t]+#.*)?[ \t]*"""


def _webhook_yaml_port(text=None):
    # config/webhook.yaml 的 webhook.port（webhook: 根节点下第一层的 port）；text 用于合成 YAML 用例
    if text is None:
        text = (ROOT / "config/webhook.yaml").read_text(encoding="utf-8")
    level = None
    in_root = False
    roots = 0
    ports = []
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        indent = len(line) - len(line.lstrip(" "))
        if indent == 0:
            if re.match(r"""(["'])webhook\1[ \t]*:""", line):
                raise AssertionError(f"webhook 根键带引号，测试解析不支持，请写成 webhook: ：{line!r}")
            in_root = re.match(r"webhook[ \t]*:", line) is not None
            roots += in_root
            if roots > 1:
                raise AssertionError("config/webhook.yaml 的 webhook 根节点重复")
            level = None
            continue
        if not in_root:
            continue
        if level is None:
            level = indent
        if indent == level:
            if re.match(r"""(["'])port\1[ \t]*:""", line.strip()):
                raise AssertionError(f"webhook.port 键名带引号，测试解析不支持，请写成 port: ：{line.strip()!r}")
            m = re.fullmatch(r"port[ \t]*:[ \t]*" + _PORT_VALUE, line.strip())
            if m:
                ports.append(int(next(g for g in m.groups() if g)))
            elif re.match(r"port\s*:", line.strip()):
                raise AssertionError(f"webhook.port 不是纯数字: {line.strip()!r}")
    if len(ports) != 1:
        raise AssertionError(f"config/webhook.yaml 的 webhook.port 应恰好 1 个，实际 {len(ports)}")
    return ports[0]


class WebhookPortConsistencyTests(unittest.TestCase):
    """P3-7：按 env.example 启动时 webhook 监听端口须与 PROMETHEUS_ALERT_WEBHOOK_ADDR 的端口一致。"""

    def test_declared_webhook_port_matches_alert_addr_port(self):
        env = _env_example()
        addr_port = _url_port(env["PROMETHEUS_ALERT_WEBHOOK_ADDR"])
        # WEBHOOK_PORT 留空时不能依赖配置文件：取代码默认（本分支 loader.go 的 webhook.port 默认值）
        declared = env.get("WEBHOOK_PORT", "") or _go_default("webhook.port")
        self.assertRegex(declared, r"^[1-9][0-9]{0,4}$")
        self.assertEqual(int(declared), addr_port,
                         "env.example 的 webhook 端口（WEBHOOK_PORT，留空时为代码默认）与告警推送地址端口不一致")

    def test_repo_webhook_yaml_port_matches_alert_addr_port(self):
        # 合入 w1 前 WEBHOOK_PORT 不起作用，从仓库根目录启动时实际端口来自默认 --config config/webhook.yaml；
        # w1 之后 WEBHOOK_PORT 优先，env.example 里两者都是 8889，与告警推送地址端口一致
        self.assertEqual(_webhook_yaml_port(), _url_port(_env_example()["PROMETHEUS_ALERT_WEBHOOK_ADDR"]))

    def test_release_compose_default_alert_addr_port_matches(self):
        compose = (ROOT / "deploy/release/compose.yaml").read_text(encoding="utf-8")
        m = re.search(r"PROMETHEUS_ALERT_WEBHOOK_ADDR:\s*\$\{PROMETHEUS_ALERT_WEBHOOK_ADDR:-([^}]+)\}", compose)
        if m is None:
            self.skipTest("本分支发布 compose 未设置可覆盖的 PROMETHEUS_ALERT_WEBHOOK_ADDR（REL-03A 合入后才有）")
        self.assertEqual(_url_port(m.group(1)), _webhook_yaml_port())

    def test_url_port_boundaries(self):
        self.assertEqual(_url_port("http://h/x"), 80)
        self.assertEqual(_url_port("https://h/x"), 443)
        self.assertEqual(_url_port(" https://h:9443/x \n"), 9443)
        self.assertEqual(_url_port("h:8889"), 8889)
        # w1 119c332 起 scheme 不区分大小写（alert_cache.go hasPrefixFold），大写写法按 scheme 取端口
        self.assertEqual(_url_port("HTTPS://h/x"), 443)
        self.assertEqual(_url_port("HTTP://h:8889/x"), 8889)
        self.assertEqual(_url_port("Http://h:8889/x"), 8889)
        self.assertEqual(_url_port("hTTps://h:9443/x"), 9443)
        # "httpſ://"（U+017F，与 s 大小写折叠）字节长度不同，hasPrefixFold 不认，代码会按 host:port 拼错地址
        for bad in ("ftp://h:21/x", "http://h:0/x", "http://h:8889suffix/x", "http://h:70000/x",
                    "http://h:８８８９/x", "h:８８８９", "httpſ://h:8889/x", "ＨTTP://h:8889/x"):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                _url_port(bad)


_GROUP_API_CALL = r'viper\.GetString\(\s*"webhook\.im_feishu\.group_message_api"\s*\)'
_GROUP_API_VALUE = rf'(?:{_GROUP_API_CALL}|\bwc\.groupMessageAPI\b)'
_GO_TRIMS = ("TrimSpace", "TrimRight", "TrimLeft", "TrimSuffix", "TrimPrefix", "Trim")


def _go_string(lit):
    # 只接受不含转义的 Go 解释型字符串字面量；其他写法直接报错，避免静默误判
    m = re.fullmatch(r'"([^"\\]*)"', lit.strip())
    if m is None:
        raise AssertionError(f"不支持的 Go 字符串字面量: {lit!r}")
    return m.group(1)


def _go_call_args(text, open_paren):
    """从 text[open_paren] 的 '(' 起，按顶层逗号切分实参；跳过字符串/字符/反引号字面量。返回 (实参列表, 右括号后位置)。"""
    if text[open_paren] != "(":
        raise AssertionError("期望 '('")
    depth, i, start, args = 0, open_paren, open_paren + 1, []
    while i < len(text):
        c = text[i]
        if c in "\"'":
            i += 1
            while text[i] != c:
                i += 2 if text[i] == "\\" else 1
        elif c == "`":
            i = text.index("`", i + 1)
        elif c in "([{":
            depth += 1
        elif c in ")]}":
            depth -= 1
            if depth == 0:
                args.append(text[start:i].strip())
                return [a for a in args if a], i + 1
        elif c == "," and depth == 1:
            args.append(text[start:i].strip())
            start = i + 1
        i += 1
    raise AssertionError("Go 调用括号不配对")


def _group_base_transform(expr):
    """把 Sprintf 第一个 %s 的实参解释成对 group_message_api 取值的变换（只认历史 Viper 取值或显式 typed 字段，以及 strings.Trim* 包装）。"""
    expr = expr.strip()
    if re.fullmatch(_GROUP_API_VALUE, expr):
        return lambda s: s
    m = re.match(r"strings\.(" + "|".join(_GO_TRIMS) + r")\s*(?=\()", expr)
    if m is None:
        raise AssertionError(f"group_message_api 外层包装不认识，需补充测试解析: {expr!r}")
    args, end = _go_call_args(expr, m.end())
    if expr[end:].strip():
        raise AssertionError(f"包装调用后还有多余内容: {expr!r}")
    fn = m.group(1)
    inner = _group_base_transform(args[0])
    if fn == "TrimSpace":
        if len(args) != 1:
            raise AssertionError(f"strings.TrimSpace 参数个数不对: {expr!r}")
        return lambda s: inner(s).strip()
    if len(args) != 2:
        raise AssertionError(f"strings.{fn} 参数个数不对: {expr!r}")
    cut = _go_string(args[1])
    ops = {
        "TrimRight": lambda s: s.rstrip(cut) if cut else s,
        "TrimLeft": lambda s: s.lstrip(cut) if cut else s,
        "Trim": lambda s: s.strip(cut) if cut else s,
        "TrimSuffix": lambda s: s[:-len(cut)] if cut and s.endswith(cut) else s,
        "TrimPrefix": lambda s: s[len(cut):] if cut and s.startswith(cut) else s,
    }
    op = ops[fn]
    return lambda s: op(inner(s))


def _group_format(text):
    """content.go 中用 group_message_api 拼群消息 URL 的 fmt.Sprintf：返回 (格式串, 对 base 的变换)。

    当前值来自构造器注入的 wc.groupMessageAPI；cfg→字段→实际请求由 Go 的
    TestContentUsesTypedConfigurationSnapshot/FakeRoundTripper 与 Wire 编译验证。
    本检查只验证 env.example 的值与群地址拼接合同，不再要求业务直接读取 Viper。
    兼容：fmt.Sprintf("%s/%s", viper.GetString(...), robotToken)（历史）、
    fmt.Sprintf("%s/%s", strings.TrimRight(viper.GetString(...), "/"), robotToken)（sched/w1-mantou 2026deb）
    以及再套一层 strings.TrimSpace(...) 等 strings.Trim* 包装。
    """
    calls = []
    for m in re.finditer(r"\bfmt\.Sprintf\s*(?=\()", text):
        args, _ = _go_call_args(text, m.end())
        if any(re.search(_GROUP_API_VALUE, arg) for arg in args):
            calls.append(args)
    if len(calls) != 1:
        raise AssertionError(f"content.go 中群消息 URL 拼接应恰好 1 处，实际 {len(calls)}")
    args = calls[0]
    if len(args) != 3 or args[2] != "robotToken":
        raise AssertionError(f"群消息 URL 拼接实参不是 (格式, base, robotToken): {args!r}")
    fmt = _go_string(args[0])
    if fmt.count("%s") != 2 or fmt.replace("%s", "").count("%") != 0:
        raise AssertionError(f"群消息 URL 格式串应恰好两个 %s: {fmt!r}")
    return fmt, _group_base_transform(args[1])


class GroupFormatParserTests(unittest.TestCase):
    """_group_format 对几种 Go 写法的解析（合成代码片段）。"""

    BASE = "https://open.feishu.cn/open-apis/bot/v2/hook"
    API = 'viper.GetString("webhook.im_feishu.group_message_api")'

    def _build(self, snippet, base, token="t"):
        fmt, transform = _group_format(snippet)
        return fmt % (transform(base), token)

    def test_old_form(self):
        snippet = f'url := fmt.Sprintf("%s/%s", {self.API}, robotToken)'
        self.assertEqual(self._build(snippet, self.BASE), self.BASE + "/t")
        self.assertEqual(self._build(snippet, self.BASE + "/"), self.BASE + "//t")

    def test_w1_trim_right_form(self):
        snippet = f'url := fmt.Sprintf("%s/%s", strings.TrimRight({self.API}, "/"), robotToken)'
        for base in (self.BASE, self.BASE + "/", self.BASE + "///"):
            with self.subTest(base=base):
                self.assertEqual(self._build(snippet, base), self.BASE + "/t")

    def test_extra_outer_wrapper(self):
        snippet = ('url := fmt.Sprintf(\n\t"%s/%s",\n\tstrings.TrimSpace(strings.TrimRight(\n\t\t'
                   + self.API + ', "/")),\n\trobotToken,\n)')
        self.assertEqual(self._build(snippet, " " + self.BASE + "/"), self.BASE + "/t")
        snippet = f'url := fmt.Sprintf("%s/%s", strings.TrimRight(strings.TrimSpace({self.API}), "/"), robotToken)'
        self.assertEqual(self._build(snippet, self.BASE + "/ \t"), self.BASE + "/t")

    def test_typed_field_preserves_group_endpoint_transforms(self):
        raw = 'url := fmt.Sprintf("%s/%s", wc.groupMessageAPI, robotToken)'
        trimmed = ('url := fmt.Sprintf("%s/%s", '
                   'strings.TrimRight(strings.TrimSpace(wc.groupMessageAPI), "/"), robotToken)')
        self.assertEqual(self._build(raw, self.BASE), self.BASE + "/t")
        self.assertEqual(self._build(trimmed, " \t" + self.BASE + "/// \n"), self.BASE + "/t")
        # 同一输入删去 trim 会暴露多斜杠，不能把字段识别成功当作 URL 合同通过。
        self.assertNotEqual(self._build(raw, self.BASE + "/"), self.BASE + "/t")

    def test_typed_field_mutations_do_not_match_group_endpoint(self):
        for value in ("wc.privateMessageAPI", "wc.groupMessageAPIFallback", "other.groupMessageAPI"):
            snippet = f'url := fmt.Sprintf("%s/%s", {value}, robotToken)'
            with self.subTest(value=value), self.assertRaises(AssertionError):
                _group_format(snippet)

    def test_unknown_or_ambiguous_forms_fail_loudly(self):
        for snippet in (
            f'u := fmt.Sprintf("%s/%s", strings.ToLower({self.API}), robotToken)',
            f'u := fmt.Sprintf("%s/%s", {self.API}, otherToken)',
            f'u := fmt.Sprintf("%s/%s/%s", {self.API}, robotToken)',
            'u := fmt.Sprintf("%s/%s", base, robotToken)',
            f'a := fmt.Sprintf("%s/%s", {self.API}, robotToken)\nb := fmt.Sprintf("%s/%s", {self.API}, robotToken)',
        ):
            with self.subTest(snippet=snippet), self.assertRaises(AssertionError):
                _group_format(snippet)


class AlertAddrSchemeDocTests(unittest.TestCase):
    """env.example 写明告警地址 scheme 建议小写、不区分大小写（w1 119c332 alert_cache.go hasPrefixFold）；
    不再写“必须小写 / 代码区分大小写”这类合入 w1 后会过时的说法。"""

    SCHEME_DOCS = ("env.example", "deploy/release/env.example", "deploy/release/compose.yaml", "docs/release/README.md")
    # 旧文案（7b47940）原句片段；新文案里“合入前的旧代码区分大小写”是如实的历史说明，不算过时
    STALE = ("scheme 必须是小写", "（代码区分大小写", "代码按区分大小写的前缀判断，HTTP://",
             "会被当成 host:port 拼出", "会被当成 host:port，拼出")

    def test_release_docs_drop_stale_case_sensitive_wording(self):
        for rel in self.SCHEME_DOCS:
            text = (ROOT / rel).read_text(encoding="utf-8")
            with self.subTest(file=rel):
                for stale in self.STALE:
                    self.assertFalse(stale in text, f"{rel} 仍有合入 w1 后过时的说法 {stale!r}")
                self.assertTrue("不区分大小写" in text, f"{rel} 没写 scheme 不区分大小写")

    def test_env_example_recommends_lowercase_scheme_case_insensitive(self):
        lines = (ROOT / "env.example").read_text(encoding="utf-8").splitlines()
        idx = next(i for i, line in enumerate(lines) if line.startswith("PROMETHEUS_ALERT_WEBHOOK_ADDR="))
        block = []
        for line in reversed(lines[:idx]):
            if not line.startswith("#"):
                break
            block.append(line)
        text = "\n".join(block)
        self.assertIn("scheme 建议写小写 http:// 或 https://，不区分大小写", text)
        self.assertNotIn("必须是小写", text)
        value = lines[idx].split("=", 1)[1].strip()
        self.assertTrue(value.startswith(("http://", "https://")) or "://" not in value, value)


class WebhookPortDocTests(unittest.TestCase):
    """发布文档必须说明端口覆盖优先级和 Compose 未传入 WEBHOOK_PORT。"""

    PORT_DOCS = ("docs/release/README.md", "deploy/release/compose.yaml")

    def test_release_docs_document_webhook_port_precedence(self):
        for rel in self.PORT_DOCS:
            text = (ROOT / rel).read_text(encoding="utf-8")
            with self.subTest(file=rel):
                self.assertFalse("对监听端口不生效" in text, f"{rel} 仍写 WEBHOOK_PORT 对监听端口不生效")
                self.assertFalse("不合并环境变量覆盖" in text, f"{rel} 仍写 loader 不合并环境变量")
                self.assertTrue("compose 没有传这个变量" in text, f"{rel} 没写 compose 没有传 WEBHOOK_PORT")
                self.assertTrue("WEBHOOK_PORT > 配置文件 > 默认 8889" in text, f"{rel} 没写 WEBHOOK_PORT 优先级")


class WebhookYamlPortValueTests(unittest.TestCase):
    """webhook.port 取值的正负向（合成 YAML）。"""

    def test_accepts_ascii_digits_and_spaced_comment(self):
        for value in ("8889", '"8889"', "'8889'", "8889 # c", '"8889"\t# c', "8889  "):
            with self.subTest(value=value):
                self.assertEqual(_webhook_yaml_port(f"webhook:\n  port: {value}\n"), 8889)

    def test_rejects_duplicate_root_and_quoted_keys(self):
        for text in ('webhook:\n  port: 8889\nwebhook:\n  min_workers: 1\n',
                     'webhook:\n  port: 8889\nserver:\n  port: 1\nwebhook:\n  port: 8890\n',
                     'webhook:\n  port: 8889\n  "port": 9999\n',
                     "webhook:\n  'port': 9999\n",
                     '"webhook":\n  port: 9999\n'):
            with self.subTest(text=text), self.assertRaises(AssertionError):
                _webhook_yaml_port(text)

    def test_rejects_fullwidth_digits_and_unspaced_hash(self):
        for value in ("８８８９", '"８８８９"', "8889#oops", '"8889"#oops', "'8889'#oops"):
            with self.subTest(value=value), self.assertRaises(AssertionError):
                _webhook_yaml_port(f"webhook:\n  port: {value}\n")


class AlertAddrFullUrlTests(unittest.TestCase):
    """r4 A-4：env.example 的告警地址整体比较（scheme、host、path、port），HTTP SD 端口与 SERVER_PORT 一致。

    期望值来源：scheme 与 path 取发布 compose 默认值（本分支没有时取 deploy/release/alertmanager.yml），path 还须等于
    webhook 路由 internal/prometheus/webhook/api/handler.go 的 POST 接收路由；host 取代码默认
    prometheus.alert_webhook_addr（本机开发，localhost）；port 取 WEBHOOK_PORT（留空为代码默认）并与 config/webhook.yaml 一致。
    """

    def _expected(self):
        env = _env_example()
        route = _alert_receive_route()
        rel_scheme, _, _, rel_path = _alert_target(_release_default_alert_addr())
        code_scheme, code_host, _, code_path = _alert_target(_go_default("prometheus.alert_webhook_addr"))
        self.assertEqual(rel_path, route, "发布默认告警地址的路径与 webhook 接收路由不一致")
        self.assertEqual(code_path, route, "代码默认告警地址的路径与 webhook 接收路由不一致")
        self.assertEqual(rel_scheme, code_scheme)
        port = int(env.get("WEBHOOK_PORT", "") or _go_default("webhook.port"))
        self.assertEqual(port, _webhook_yaml_port(), "WEBHOOK_PORT 与 config/webhook.yaml 的 webhook.port 不一致")
        return rel_scheme, code_host, port, route

    def test_alert_addr_full_url(self):
        addr = _env_example()["PROMETHEUS_ALERT_WEBHOOK_ADDR"]
        scheme, host, port, path = _alert_target(addr)
        e_scheme, e_host, e_port, e_path = self._expected()
        self.assertEqual(scheme, e_scheme, "告警地址 scheme 不对")
        self.assertEqual(host, e_host, "告警地址 host 不对（本机开发模板应为代码默认的 localhost）")
        self.assertEqual(path, e_path, "告警地址实际推送路径与 webhook 接收路由不一致")
        self.assertEqual(port, e_port, "告警地址端口与 webhook 监听端口不一致")
        self.assertEqual(f"{scheme}://{host}:{port}{path}", f"{e_scheme}://{e_host}:{e_port}{e_path}")

    def test_http_sd_api_port_matches_server_port(self):
        env = _env_example()
        sd = env["PROMETHEUS_HTTP_SD_API"]
        server_port = env.get("SERVER_PORT", "") or _go_default("server.port")
        self.assertRegex(server_port, r"^[1-9][0-9]{0,4}$")
        self.assertEqual(_url_port(sd), int(server_port), "PROMETHEUS_HTTP_SD_API 端口与 SERVER_PORT 不一致")
        parts = _split_alert_url(sd)
        self.assertEqual(parts.scheme, "http")
        self.assertIn(parts.path, _gin_routes(_HTTP_SD_GO, "GET"), "PROMETHEUS_HTTP_SD_API 路径不是主后端的 HTTP SD 路由")

    def test_alert_target_follows_go_builder(self):
        self.assertEqual(_alert_target("http://h:1/api/v1/alerts/receive/"), ("http", "h", 1, "/api/v1/alerts/receive"))
        self.assertEqual(_alert_target("http://h:1"), ("http", "h", 1, "/api/v1/alerts/receive"))
        self.assertEqual(_alert_target("h:2"), ("http", "h", 2, "/api/v1/alerts/receive"))
        self.assertEqual(_alert_target("https://h/x"), ("https", "h", 443, "/x/api/v1/alerts/receive"))
        # w1 119c332：大写 scheme 也是完整 URL（不再拼成 http://HTTP://...）
        self.assertEqual(_alert_target("HTTP://h:1/api/v1/alerts/receive"), ("http", "h", 1, "/api/v1/alerts/receive"))
        self.assertEqual(_alert_target(" Https://h/x/ "), ("https", "h", 443, "/x/api/v1/alerts/receive"))
        with self.assertRaises(ValueError):
            _alert_target("httpſ://h:1/api/v1/alerts/receive")


_GODOTENV_CASES = [
    # (内容, godotenv v1.5.1 解析结果)；合成值，与仓库 / 主机上的真实配置无关
    ("export FOO=1\n", {"FOO": "1"}),
    ("FOO = 1\nBAR: two\n", {"FOO": "1", "BAR": "two"}),
    ("FOO=a # 注释\nBAR=a#b\nBAZ=a #b #c\n", {"FOO": "a", "BAR": "a#b", "BAZ": "a #b"}),
    ("FOO=\"a b\" # 注释\nBAR='x # y'\n", {"FOO": "a b", "BAR": "x # y"}),
    ("FOO=\"l1\\nl2\"\nBAR='raw\\n'\n", {"FOO": "l1\nl2", "BAR": "raw\\n"}),
    ("A=1\nB=${A}x\nC=$A\nD=\\$A\nE=${MISSING}\nF='$A'\n", {"A": "1", "B": "1x", "C": "1", "D": "$A", "E": "", "F": "$A"}),
    ("X=${LATER}\nLATER=v\n", {"X": "", "LATER": "v"}),
    ("A=1\nA=2\n", {"A": "2"}),
    ("lower.case=1\n", {"lower.case": "1"}),
    ("  # 注释\n\tFOO=1  \r\nBAR=\n", {"FOO": "1", "BAR": ""}),
]


class GodotenvParserTests(unittest.TestCase):
    """_env_example 与 backend 读 .env 的方式一致：main.go 调用 godotenv.Load()（go.mod 锁定 v1.5.1）"""

    def test_semantics(self):
        for text, expected in _GODOTENV_CASES:
            with self.subTest(text=text):
                self.assertEqual(_env_example(text), expected)

    def test_rejects_malformed(self):
        for text in ("FOO\n", 'FOO="unterminated\n', "FOO-BAR=1\n"):
            with self.subTest(text=text), self.assertRaises(ValueError):
                _env_example(text)

    def test_env_example_parses(self):
        self.assertIn("PROMETHEUS_ALERT_WEBHOOK_ADDR", _env_example())

    def test_matches_real_godotenv(self):
        # 可选交叉核对：只用本机模块缓存（GOPROXY=off、不切换工具链、不改 go.mod）；缺 go 或缓存时跳过
        go = shutil.which("go")
        if go is None:
            self.skipTest("go 不可用，跳过与真实 godotenv 的交叉核对")
        sums = [l for l in (ROOT / "go.sum").read_text(encoding="utf-8").splitlines()
                if l.startswith("github.com/joho/godotenv v1.5.1")]
        if len(sums) != 2:
            self.skipTest("go.sum 里没有 godotenv v1.5.1")
        rnd = random.Random(20261024)
        alphabet = ["A", "B", "_", "x", "1", "=", ":", " ", "\t", "#", "\n", "\r", "\r\n", '"', "'", "\\", "$",
                    "{", "}", "(", "export ", "\\n", "${A}", "$B", "é", "\xa0", "\x85", "\u3000", ".", "-", "１"]
        cases = [t for t, _ in _GODOTENV_CASES] + [(ROOT / "env.example").read_text(encoding="utf-8")]
        cases += ["".join(rnd.choice(alphabet) for _ in range(rnd.randint(0, 24))) for _ in range(3000)]
        with tempfile.TemporaryDirectory() as tmp:
            mod = Path(tmp) / "godotenvcheck"
            mod.mkdir()
            (mod / "go.mod").write_text("module godotenvcheck\n\ngo 1.22\n\nrequire github.com/joho/godotenv v1.5.1\n")
            (mod / "go.sum").write_text("\n".join(sums) + "\n")
            (mod / "main.go").write_text(_GODOTENV_GO)
            env = {**os.environ, "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off", "GOFLAGS": "",
                   "GOCACHE": os.environ.get("GOCACHE") or str(Path(tmp) / "gocache")}
            r = subprocess.run([go, "run", "."], cwd=mod, input=json.dumps(cases), env=env,
                               capture_output=True, text=True, timeout=300)
            if r.returncode != 0:
                self.skipTest("本机模块缓存缺少 github.com/joho/godotenv v1.5.1，跳过交叉核对")
        mismatches = []
        for text, got in zip(cases, json.loads(r.stdout)):
            want = {"err": True} if got.get("err") else {"ok": got.get("ok") or {}}
            try:
                mine = {"ok": _env_example(text)}
            except ValueError:
                mine = {"err": True}
            if mine != want:
                mismatches.append(text)
        self.assertEqual(mismatches[:5], [], f"{len(mismatches)} / {len(cases)} 条与 godotenv 结果不同")


_GODOTENV_GO = """package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type result struct {
	OK  map[string]string `json:"ok,omitempty"`
	Err bool              `json:"err,omitempty"`
}

func main() {
	var cases []string
	if err := json.NewDecoder(os.Stdin).Decode(&cases); err != nil {
		panic(err)
	}
	out := make([]result, 0, len(cases))
	for _, c := range cases {
		m, err := godotenv.Parse(strings.NewReader(c))
		if err != nil {
			out = append(out, result{Err: true})
			continue
		}
		out = append(out, result{OK: m})
	}
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
"""


class FeishuGroupMessageUrlTests(unittest.TestCase):
    """P3-8：WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API 按 content.go SentFeishuGroup 的拼接方式须得到群机器人 webhook 地址。"""

    CONTENT_GO = "internal/prometheus/webhook/content/content.go"
    EVENT_GO = "internal/prometheus/service/alert/event_service.go"

    def _build(self, base, token):
        fmt, transform = _group_format((ROOT / self.CONTENT_GO).read_text(encoding="utf-8"))
        return fmt % (transform(base), token)

    def _strips_trailing_slash(self):
        _, transform = _group_format((ROOT / self.CONTENT_GO).read_text(encoding="utf-8"))
        return transform("x/") == "x"

    def _reference_format(self):
        # 后端 event_service.go 用同一个群机器人 token 直接拼出的地址，作为期望值来源
        text = (ROOT / self.EVENT_GO).read_text(encoding="utf-8")
        m = re.findall(r'fmt\.Sprintf\("(https://[^"]*/%s)",\s*sendGroup\.FeiShuQunRobotToken\)', text)
        self.assertEqual(len(m), 1)
        return m[0]

    def test_group_message_url_from_env_example(self):
        token = f"r4-synthetic-{os.getpid()}"
        base = _env_example()["WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API"]
        url = self._build(base, token)
        expected = self._reference_format() % token
        parts = urlsplit(url)
        self.assertNotIn("//", parts.path, f"拼出的路径有重复斜杠: {url}")
        self.assertEqual(url, expected, f"拼出的群消息地址不对: {url}")
        self.assertEqual(parts.path.rsplit("/", 1)[-1], token)

    def test_wrong_group_message_bases_are_detected(self):
        # 负向：旧示例值（im/v1/messages）、缺 hook 段在任何写法下都拼不出正确地址
        token = f"r4-synthetic-{os.getpid()}"
        expected = self._reference_format() % token
        for base in ("https://open.feishu.cn/open-apis/im/v1/messages",
                     "https://open.feishu.cn/open-apis/bot/v2"):
            with self.subTest(base=base):
                self.assertNotEqual(self._build(base, token), expected)

    def test_trailing_slash_base_follows_code_semantics(self):
        # 当前实现必须先去首尾空白、再去所有尾斜杠；不能接受历史上的 hook//<token>。
        token = f"r4-synthetic-{os.getpid()}"
        expected = self._reference_format() % token
        self.assertTrue(self._strips_trailing_slash())
        for base in ("https://open.feishu.cn/open-apis/bot/v2/hook/",
                     " \thttps://open.feishu.cn/open-apis/bot/v2/hook/// \n"):
            with self.subTest(base=base):
                self.assertEqual(self._build(base, token), expected)


if __name__ == "__main__":
    unittest.main()
