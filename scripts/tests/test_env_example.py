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


def _url_port(url):
    parts = urlsplit(url if "://" in url else "http://" + url)
    scheme = parts.scheme.lower()
    if scheme not in _DEFAULT_PORTS:
        raise ValueError(f"不支持的 scheme: {scheme!r}")
    port = parts.port if parts.port is not None else _DEFAULT_PORTS[scheme]
    if not 1 <= port <= 65535:
        raise ValueError(f"端口越界: {port}")
    return port


def _go_default(key):
    # internal/config/loader.go 里 viper.SetDefault("<key>", "<value>") 的值（只读解析）
    text = (ROOT / "internal/config/loader.go").read_text(encoding="utf-8")
    found = re.findall(r'viper\.SetDefault\("' + re.escape(key) + r'",\s*"([^"]*)"\)', text)
    if len(found) != 1:
        raise AssertionError(f"loader.go 中 {key} 默认值应恰好 1 处，实际 {len(found)}")
    return found[0]


def _webhook_yaml_port():
    # config/webhook.yaml 的 webhook.port（webhook: 根节点下第一层的 port）
    level = None
    in_root = False
    ports = []
    for line in (ROOT / "config/webhook.yaml").read_text(encoding="utf-8").splitlines():
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
            m = re.fullmatch(r"port[ \t]*:[ \t]*(?:\"(\d+)\"|'(\d+)'|(\d+))[ \t]*(?:#.*)?", line.strip())
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
        self.assertEqual(_url_port("HTTPS://h/x"), 443)
        self.assertEqual(_url_port("h:8889"), 8889)
        for bad in ("ftp://h:21/x", "http://h:0/x", "http://h:8889suffix/x", "http://h:70000/x"):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                _url_port(bad)


class FeishuGroupMessageUrlTests(unittest.TestCase):
    """P3-8：WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API 按 content.go SentFeishuGroup 的拼接方式须得到群机器人 webhook 地址。"""

    CONTENT_GO = "internal/prometheus/webhook/content/content.go"
    EVENT_GO = "internal/prometheus/service/alert/event_service.go"

    def _group_format(self):
        text = (ROOT / self.CONTENT_GO).read_text(encoding="utf-8")
        m = re.findall(r'fmt\.Sprintf\("([^"]*)",\s*viper\.GetString\("webhook\.im_feishu\.group_message_api"\),\s*robotToken\)', text)
        self.assertEqual(len(m), 1, "content.go 中群消息 URL 拼接应恰好 1 处")
        self.assertEqual(m[0].count("%s"), 2)
        return m[0]

    def _reference_format(self):
        # 后端 event_service.go 用同一个群机器人 token 直接拼出的地址，作为期望值来源
        text = (ROOT / self.EVENT_GO).read_text(encoding="utf-8")
        m = re.findall(r'fmt\.Sprintf\("(https://[^"]*/%s)",\s*sendGroup\.FeiShuQunRobotToken\)', text)
        self.assertEqual(len(m), 1)
        return m[0]

    def test_group_message_url_from_env_example(self):
        token = f"r3inc-synthetic-{os.getpid()}"
        base = _env_example()["WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API"]
        url = self._group_format() % (base, token)
        expected = self._reference_format() % token
        parts = urlsplit(url)
        self.assertNotIn("//", parts.path, f"拼出的路径有重复斜杠: {url}")
        self.assertEqual(url, expected, f"拼出的群消息地址不对: {url}")
        self.assertEqual(parts.path.rsplit("/", 1)[-1], token)

    def test_wrong_group_message_bases_are_detected(self):
        # 负向：旧示例值（im/v1/messages）、带尾斜杠（config/webhook.yaml 现值）、缺 hook 段都拼不出正确地址
        token = f"r3inc-synthetic-{os.getpid()}"
        expected = self._reference_format() % token
        for base in ("https://open.feishu.cn/open-apis/im/v1/messages",
                     "https://open.feishu.cn/open-apis/bot/v2/hook/",
                     "https://open.feishu.cn/open-apis/bot/v2"):
            with self.subTest(base=base):
                url = self._group_format() % (base, token)
                self.assertTrue(url != expected or "//" in urlsplit(url).path)
                self.assertNotEqual(url, expected)


if __name__ == "__main__":
    unittest.main()
