"""根目录 env.example 与本分支代码行为的一致性检查（r3inc P3-7 / P3-8）。

只读取仓库里的 env.example、config/webhook.yaml、deploy/release/compose.yaml 和 Go 源码中的默认值 / 拼接格式，
不连网、不启动进程；群机器人 token 用运行时生成的合成值。
"""
import os
import re
import unittest
from pathlib import Path
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[2]
_DEFAULT_PORTS = {"http": 80, "https": 443}


def _env_example():
    values = {}
    for line in (ROOT / "env.example").read_text(encoding="utf-8").splitlines():
        m = re.fullmatch(r"([A-Z][A-Z0-9_]*)=(.*)", line.strip())
        if m:
            values[m.group(1)] = m.group(2).strip()
    return values


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


def _split_alert_url(url):
    """按 alert_cache.go buildWebhookURL 的规则拆地址：TrimSpace 后以小写 http:// 或 https:// 开头才算完整 URL
    （区分大小写的前缀判断），不含 :// 时按 host:port 走 http；其他带 :// 的写法（含大写 HTTPS://）在代码里会拼出
    http://HTTPS://... 的错误地址，这里直接报错，不做小写化。"""
    base = url.strip()
    if base.startswith(("http://", "https://")):
        parts = urlsplit(base)
    elif "://" in base:
        raise ValueError(f"scheme 必须是小写 http/https（代码区分大小写）: {base.split('://', 1)[0]!r}")
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
    parts = _split_alert_url(base)  # 校验 scheme 大小写与端口
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
    ports = []
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        indent = len(line) - len(line.lstrip(" "))
        if indent == 0:
            in_root = re.match(r"webhook\s*:", line) is not None
            level = None
            continue
        if not in_root:
            continue
        if level is None:
            level = indent
        if indent == level:
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
        # 本分支 WEBHOOK_PORT 不生效，从仓库根目录启动时实际端口来自默认 --config config/webhook.yaml
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
        for bad in ("ftp://h:21/x", "http://h:0/x", "http://h:8889suffix/x", "http://h:70000/x",
                    "http://h:８８８９/x", "h:８８８９",
                    "HTTPS://h/x", "HTTP://h:8889/x", "Http://h:8889/x", "hTTps://h:9443/x"):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                _url_port(bad)


_GROUP_API_CALL = r'viper\.GetString\(\s*"webhook\.im_feishu\.group_message_api"\s*\)'
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
    """把 Sprintf 第一个 %s 的实参解释成对 group_message_api 取值的变换（只认 viper.GetString 与 strings.Trim* 包装）。"""
    expr = expr.strip()
    if re.fullmatch(_GROUP_API_CALL, expr):
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

    兼容：fmt.Sprintf("%s/%s", viper.GetString(...), robotToken)（本分支）、
    fmt.Sprintf("%s/%s", strings.TrimRight(viper.GetString(...), "/"), robotToken)（sched/w1-mantou 2026deb）
    以及再套一层 strings.TrimSpace(...) 等 strings.Trim* 包装。
    """
    calls = []
    for m in re.finditer(r"\bfmt\.Sprintf\s*(?=\()", text):
        args, _ = _go_call_args(text, m.end())
        if any("group_message_api" in a for a in args):
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
    """env.example 须写明告警地址的 scheme 必须小写（代码按区分大小写的前缀判断）。"""

    def test_env_example_documents_lowercase_scheme(self):
        lines = (ROOT / "env.example").read_text(encoding="utf-8").splitlines()
        idx = next(i for i, line in enumerate(lines) if line.startswith("PROMETHEUS_ALERT_WEBHOOK_ADDR="))
        block = []
        for line in reversed(lines[:idx]):
            if not line.startswith("#"):
                break
            block.append(line)
        text = "\n".join(block)
        self.assertIn("scheme 必须是小写 http:// 或 https://", text)
        value = lines[idx].split("=", 1)[1].strip()
        self.assertTrue(value.startswith(("http://", "https://")) or "://" not in value, value)


class WebhookYamlPortValueTests(unittest.TestCase):
    """webhook.port 取值的正负向（合成 YAML）。"""

    def test_accepts_ascii_digits_and_spaced_comment(self):
        for value in ("8889", '"8889"', "'8889'", "8889 # c", '"8889"\t# c', "8889  "):
            with self.subTest(value=value):
                self.assertEqual(_webhook_yaml_port(f"webhook:\n  port: {value}\n"), 8889)

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
        with self.assertRaises(ValueError):
            _alert_target("HTTP://h:1/api/v1/alerts/receive")


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
        # 带尾斜杠（config/webhook.yaml 现值）：本分支原样拼接得到 hook//<token>（错误）；
        # sched/w1-mantou 的 strings.TrimRight(..., "/") 会去掉尾斜杠，得到正确地址
        token = f"r4-synthetic-{os.getpid()}"
        expected = self._reference_format() % token
        url = self._build("https://open.feishu.cn/open-apis/bot/v2/hook/", token)
        if self._strips_trailing_slash():
            self.assertEqual(url, expected)
        else:
            self.assertNotEqual(url, expected)
            self.assertIn("//", urlsplit(url).path)


if __name__ == "__main__":
    unittest.main()
