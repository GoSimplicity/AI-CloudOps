"""发布入口回归：仅使用临时目录，不启动服务或访问外部基础设施"""

import os
from pathlib import Path
import shutil
import stat
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.parent = Path(self.temp.name)
        self.root = self.parent / "AI-CloudOps"
        (self.root / "scripts").mkdir(parents=True)
        (self.root / "deploy/release").mkdir(parents=True)
        for name in ("release-start.sh", "release-env.py"):
            shutil.copyfile(ROOT / "scripts" / name, self.root / "scripts" / name)

    def test_secrets_are_private_and_never_overwritten(self):
        import sys

        script = self.root / "scripts/release-env.py"
        result = subprocess.run([sys.executable, str(script)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        destination = self.root / "deploy/release/.env"
        content = destination.read_text()
        self.assertEqual(stat.S_IMODE(destination.stat().st_mode), 0o600)
        settings = dict(line.split("=", 1) for line in content.splitlines() if "=" in line)
        self.assertEqual(len(settings["TREE_PASSWORD_ENCRYPTION_KEY"]), 32)
        for name in ("MYSQL_PASSWORD", "JWT_KEY1", "JWT_KEY2", "AIOPS_API_KEY"):
            self.assertNotIn(settings[name], result.stdout + result.stderr)
        again = subprocess.run([sys.executable, str(script)], capture_output=True, text=True)
        self.assertNotEqual(again.returncode, 0)
        self.assertEqual(destination.read_text(), content)

    def test_start_rejects_missing_siblings_before_docker(self):
        result = subprocess.run(["bash", str(self.root / "scripts/release-start.sh")], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Missing sibling repository", result.stderr)

    def test_start_accepts_exact_github_directory_names(self):
        for name in ("AI-CloudOps-web", "AI-CloudOps-aiops"):
            sibling = self.parent / name
            sibling.mkdir()
            (sibling / "Dockerfile").touch()
        # 不生成或输出真实密钥，Docker 替身只记录调用参数
        (self.root / "deploy/release/.env").touch()
        commands = self.parent / "commands"
        commands.mkdir()
        docker = commands / "docker"
        docker.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$RELEASE_TEST_CALLS"\n')
        docker.chmod(0o700)
        calls = self.parent / "calls"
        env = {**os.environ, "PATH": str(commands) + os.pathsep + os.environ["PATH"], "RELEASE_TEST_CALLS": str(calls)}
        result = subprocess.run(["bash", str(self.root / "scripts/release-start.sh")], env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        lines = calls.read_text().splitlines()
        self.assertEqual(len(lines), 3)
        self.assertEqual(lines[0], "info")
        self.assertTrue(lines[1].endswith("config --quiet"))
        self.assertTrue(lines[2].endswith("up -d --build --wait --wait-timeout 300"))

    def test_smoke_rejects_unhealthy_service_in_optimized_python(self):
        import sys
        from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
        import threading

        class Unhealthy(BaseHTTPRequestHandler):
            def do_GET(self):
                failed = self.path == self.server.fail_path
                self.send_response(503 if failed else 200)
                self.end_headers()
                status = "unavailable" if failed else ("ok" if self.path == "/health" else "ready")
                self.wfile.write(('{"status":"' + status + '"}').encode())

            def log_message(self, *_args):
                pass

        server = ThreadingHTTPServer(("127.0.0.1", 0), Unhealthy)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            env_file = self.parent / "smoke.env"
            env_file.write_text("BOOTSTRAP_ADMIN_PASSWORD=private-test-value\n")
            for path, message in (("/health", "Backend health failed"), ("/ready", "Backend readiness failed")):
                server.fail_path = path
                for flags in ([], ["-O"]):
                    with self.subTest(path=path, flags=flags):
                        result = subprocess.run(
                            [sys.executable, *flags, str(ROOT / "scripts/release-smoke.py"),
                             "--url", f"http://127.0.0.1:{server.server_port}", "--env-file", str(env_file)],
                            capture_output=True, text=True, timeout=10,
                        )
                        self.assertNotEqual(result.returncode, 0)
                        self.assertIn(message, result.stderr)
                        self.assertNotIn("PASS bootstrap", result.stdout)
                        self.assertNotIn("private-test-value", result.stdout + result.stderr)
        finally:
            server.shutdown()
            server.server_close()
            thread.join()

    def test_release_paths_match_github_case(self):
        for name in ("scripts/release-start.sh", "scripts/release-backup.py", "deploy/release/compose.yaml"):
            content = (ROOT / name).read_text()
            self.assertIn("AI-CloudOps-aiops", content, name)
            self.assertNotIn("Ai-CloudOps-aiops", content, name)

    def test_crash_smoke_guards_do_not_call_docker(self):
        import sys

        script = ROOT / "scripts/integration/cron_crash_smoke.py"
        # 空 PATH 下 Docker 不可执行，帮助及拒绝路径仍应正常退出
        env = {**os.environ, "PATH": self.temp.name}
        for arguments, code, message in (
            ([], 2, "需要 --execute"),
            (["--execute", "--recovery-timeout", "0"], 2, "30至600秒"),
            (["--help"], 0, "--execute"),
        ):
            with self.subTest(arguments=arguments):
                result = subprocess.run(
                    [sys.executable, "-O", str(script), *arguments], env=env,
                    capture_output=True, text=True, timeout=10,
                )
                self.assertEqual(result.returncode, code, result.stderr)
                self.assertIn(message, result.stdout + result.stderr)


# 告警地址只允许 http/https；没写端口时按 scheme 取默认端口
_DEFAULT_PORTS = {"http": 80, "https": 443}


def _alert_addr_target(addr):
    """解析 PROMETHEUS_ALERT_WEBHOOK_ADDR，返回 (主机名, 端口, 路径)。

    代码接受完整 URL 和 host:port 两种写法，后者按 http 处理。scheme 不是 http/https、端口不是 1-65535 的纯数字
    （如 8889suffix）时抛 ValueError（端口格式由 urllib 校验）。
    """
    from urllib.parse import urlsplit
    url = urlsplit(addr if "://" in addr else "http://" + addr)
    scheme = url.scheme.lower()
    if scheme not in _DEFAULT_PORTS:
        raise ValueError(f"告警地址只允许 http/https，实际 scheme={url.scheme!r}")
    port = url.port
    if port is None:
        port = _DEFAULT_PORTS[scheme]
    if not 1 <= port <= 65535:
        raise ValueError(f"告警地址端口超出 1-65535: {port}")
    return url.hostname, port, url.path


def _webhook_root_port(text):
    """从 webhook.yaml 文本取 webhook 根节点下的 port，没有时返回 None。

    不依赖固定缩进：以 webhook: 下第一个子键的缩进为准，只认这一层的 port（更深层的嵌套键不算）。
    值只能是纯数字（可加单/双引号，后面只能跟空白或 # 注释），8889suffix 这类写法、重复的 port、
    超出 1-65535 都抛 ValueError。不依赖 PyYAML，macOS 自带 python3 也能跑。
    """
    import re
    ports, in_root, child = [], False, None
    for raw in text.splitlines():
        body = raw.lstrip(" \t")
        if not body or body.startswith("#"):
            continue
        indent = len(raw) - len(body)
        if indent == 0:  # 顶层键
            in_root = re.match(r"webhook[ \t]*:[ \t]*(#.*)?$", raw) is not None
            child = None
            continue
        if not in_root:
            continue
        if "\t" in raw[:indent]:
            raise ValueError("webhook.yaml 缩进里有 Tab")
        if child is None:
            child = indent
        if indent != child or not re.match(r"port[ \t]*:", body):
            continue
        m = re.fullmatch(r"port[ \t]*:[ \t]*(?:\"(\d+)\"|'(\d+)'|(\d+))[ \t]*(?:#.*)?", body)
        if not m:
            raise ValueError(f"webhook.port 不是纯数字: {body!r}")
        ports.append(int(next(g for g in m.groups() if g)))
    if len(ports) > 1:
        raise ValueError(f"webhook.port 重复 {len(ports)} 次")
    if not ports:
        return None
    if not 1 <= ports[0] <= 65535:
        raise ValueError(f"webhook.port 超出 1-65535: {ports[0]}")
    return ports[0]


def _compose_env(home):
    """compose 测试的隔离环境，探测（_compose_tools）和执行（ReleaseComposeWebhookAddrTests）共用。

    只保留 PATH，HOME 指向临时目录，不继承调用者导出的其他变量（避免覆盖合成值）。
    DOCKER_CONFIG 透传；未设置时指向隔离前的 ~/.docker：docker CLI 在 $DOCKER_CONFIG/cli-plugins
    （默认 ~/.docker/cli-plugins）里找 compose 插件，Mac 上 OrbStack / Docker Desktop 就装在这里，
    只换 HOME 会让执行时找不到插件（退出码 125）。
    """
    real_home = os.environ.get("HOME") or os.path.expanduser("~")
    docker_config = os.environ.get("DOCKER_CONFIG") or os.path.join(real_home, ".docker")
    return {"PATH": os.environ.get("PATH", ""), "HOME": str(home), "DOCKER_CONFIG": docker_config}


def _compose_tools():
    # 在导入阶段调用：任何探测失败（超时、无法执行）都只让相关用例 skip，不能让模块导入报错
    if not (shutil.which("docker") and shutil.which("jq")):
        return False
    try:
        with tempfile.TemporaryDirectory() as home:
            return subprocess.run(["docker", "compose", "version"], env=_compose_env(home),
                                  capture_output=True, timeout=30).returncode == 0
    except (subprocess.TimeoutExpired, OSError):
        return False


@unittest.skipUnless(_compose_tools(), "需要 docker compose 和 jq（只做 compose config，不连 Docker 守护进程、不启动服务）")
class ReleaseComposeWebhookAddrTests(unittest.TestCase):
    """backend 的 PROMETHEUS_ALERT_WEBHOOK_ADDR 必须指向 webhook 服务、端口与 webhook 实际监听端口一致"""

    COMPOSE = ROOT / "deploy/release/compose.yaml"

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        # 合成 .env：compose 中所有 ${X:?…} 必填变量都给运行时拼出的占位值，不读仓库或主机上的真实 .env
        import re
        names = sorted(set(re.findall(r"\$\{([A-Z0-9_]+):\?", self.COMPOSE.read_text())))
        self.env_file = Path(self.temp.name) / "synthetic.env"
        self.env_file.write_text("".join(f"{n}={'fixture'}-{n.lower()}-{os.getpid()}\n" for n in names))
        # 干净环境：不继承调用者导出的变量，避免覆盖合成值；与探测用同一套隔离规则（见 _compose_env）
        self.env = _compose_env(self.temp.name)
        self.base = ["docker", "compose", "--project-directory", str(self.COMPOSE.parent),
                     "-f", str(self.COMPOSE), "--env-file", str(self.env_file)]

    def _jq(self, expr):
        # 只经 jq 取指定字段，完整 config 输出不进入测试日志
        config = subprocess.run(self.base + ["config", "--format", "json"], env=self.env,
                                capture_output=True, timeout=60)
        self.assertEqual(config.returncode, 0, "compose config 失败（输出已省略）")
        result = subprocess.run(["jq", "-r", expr], input=config.stdout, capture_output=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stderr.decode(errors="replace"))
        return result.stdout.decode().strip()

    def test_compose_config_quiet(self):
        result = subprocess.run(self.base + ["config", "--quiet"], env=self.env, capture_output=True, timeout=60)
        self.assertEqual(result.returncode, 0, "compose config --quiet 失败（输出已省略）")

    DEFAULT_ALERT_ADDR = "http://webhook:8889/api/v1/alerts/receive"
    ALERT_ADDR_JQ = '.services.backend.environment.PROMETHEUS_ALERT_WEBHOOK_ADDR // ""'

    def test_alert_webhook_addr_default_when_unset_or_empty(self):
        # r3inc P3-4：compose 写成 ${PROMETHEUS_ALERT_WEBHOOK_ADDR:-默认}；未设置和设为空串都取默认
        self.assertEqual(self._jq(self.ALERT_ADDR_JQ), self.DEFAULT_ALERT_ADDR)
        with self.env_file.open("a") as f:
            f.write("PROMETHEUS_ALERT_WEBHOOK_ADDR=\n")
        self.assertEqual(self._jq(self.ALERT_ADDR_JQ), self.DEFAULT_ALERT_ADDR)

    def test_alert_webhook_addr_override_from_env_file(self):
        # Compose 网络外的 Alertmanager：在 .env 里覆盖（合成地址，.test 保留域名）
        custom = f"https://alerts-{os.getpid()}.example.test:9443/api/v1/alerts/receive"
        with self.env_file.open("a") as f:
            f.write(f"PROMETHEUS_ALERT_WEBHOOK_ADDR={custom}\n")
        self.assertEqual(self._jq(self.ALERT_ADDR_JQ), custom)
        self.assertEqual(_alert_addr_target(custom)[1:], (9443, "/api/v1/alerts/receive"))

    def test_alert_webhook_addr_override_from_shell_env(self):
        # shell 里导出的同名变量同样生效（优先于 .env）：发布前要确认没有误带本机开发用的 localhost 值
        custom = f"http://198.51.100.{os.getpid() % 250 + 1}:18889/api/v1/alerts/receive"
        self.env["PROMETHEUS_ALERT_WEBHOOK_ADDR"] = custom
        self.assertEqual(self._jq(self.ALERT_ADDR_JQ), custom)

    def test_backend_alert_webhook_addr_targets_webhook_port(self):
        addr = self._jq('.services.backend.environment.PROMETHEUS_ALERT_WEBHOOK_ADDR // ""')
        self.assertTrue(addr, "backend 未设置 PROMETHEUS_ALERT_WEBHOOK_ADDR（代码默认 localhost:8889 在容器里指向 backend 自己）")
        host, port, path = _alert_addr_target(addr)
        self.assertEqual(host, "webhook")
        self.assertEqual(self._jq('.services.webhook | type'), "object", "compose 里没有 webhook 服务")
        self.assertEqual(self._jq('.services.backend.network_mode // ""'), "", "backend 用了 network_mode，服务名不可解析")
        self.assertEqual(self._jq('.services.webhook.environment.WEBHOOK_PORT // ""'), "",
                         "webhook 设了 WEBHOOK_PORT，需同步检查本测试")
        # webhook 实际端口：镜像内 config/webhook.yaml 的 webhook.port
        listen = _webhook_root_port((ROOT / "config/webhook.yaml").read_text())
        self.assertIsNotNone(listen, "config/webhook.yaml 里没找到 webhook.port（必须在 webhook: 根节点下）")
        self.assertEqual(port, listen)
        self.assertIn(path.rstrip("/"), ("", "/api/v1/alerts/receive"))
        # 与 webhook 健康检查、alertmanager.yml 的端口一致
        health = self._jq('.services.webhook.healthcheck.test | join(" ")')
        self.assertIn(f"127.0.0.1:{port}/", health)
        self.assertIn(f"http://webhook:{port}/api/v1/alerts/receive",
                      (ROOT / "deploy/release/alertmanager.yml").read_text())

class AlertAddrBoundaryTests(unittest.TestCase):
    """告警地址解析边界（r3inc P3-3）：scheme 白名单、按 scheme 取默认端口。不需要 Docker"""

    def test_accepts_http_https_and_host_port(self):
        self.assertEqual(_alert_addr_target("http://webhook:8889/api/v1/alerts/receive"),
                         ("webhook", 8889, "/api/v1/alerts/receive"))
        self.assertEqual(_alert_addr_target("HTTPS://webhook:9443/x")[1], 9443)
        self.assertEqual(_alert_addr_target("webhook:8889"), ("webhook", 8889, ""))

    def test_rejects_non_http_scheme(self):
        for addr in ("ftp://webhook:8889/api/v1/alerts/receive", "ws://webhook:8889/api/v1/alerts/receive",
                     "file://webhook:8889/api/v1/alerts/receive", "unix://webhook:8889/x"):
            with self.subTest(addr=addr), self.assertRaises(ValueError):
                _alert_addr_target(addr)

    def test_default_port_follows_scheme(self):
        self.assertEqual(_alert_addr_target("http://webhook/api/v1/alerts/receive")[1], 80)
        self.assertEqual(_alert_addr_target("https://webhook/api/v1/alerts/receive")[1], 443)

    def test_https_without_port_is_not_80(self):
        # 负向：https 不写端口时不能当成 80，否则 webhook 监听 80 时会被误判为一致
        self.assertNotEqual(_alert_addr_target("https://webhook/api/v1/alerts/receive")[1], 80)

    def test_rejects_bad_port(self):
        for addr in ("http://webhook:8889suffix/x", "http://webhook:0/x", "http://webhook:65536/x"):
            with self.subTest(addr=addr), self.assertRaises(ValueError):
                _alert_addr_target(addr)


class WebhookYamlPortBoundaryTests(unittest.TestCase):
    """webhook.yaml 根节点端口解析边界（r3inc P3-3）。全部是运行时拼出的合成 YAML，不读主机文件"""

    def test_two_space_indent(self):
        text = 'webhook:\n  front_domain: "x"\n  port: "8889"  # HTTP 服务端口\n'
        self.assertEqual(_webhook_root_port(text), 8889)

    def test_quote_styles_and_comment(self):
        for value in ("8889", '"8889"', "'8889'", '"8889" # c', "8889\t# c"):
            with self.subTest(value=value):
                self.assertEqual(_webhook_root_port(f"webhook:\n  port: {value}\n"), 8889)

    def test_four_space_indent(self):
        # 负向：缩进不是两个空格时也必须找到 webhook.port
        text = 'webhook:\n    im_feishu:\n      port: "1"\n    port: "9999"\n'
        self.assertEqual(_webhook_root_port(text), 9999)

    def test_nested_port_is_not_root_port(self):
        text = 'webhook:\n  im_feishu:\n    port: "1"\n  alert_manager_api: "x"\n'
        self.assertIsNone(_webhook_root_port(text))

    def test_port_outside_webhook_root_ignored(self):
        text = 'server:\n  port: "1234"\nwebhook:\n  min_workers: 1\n'
        self.assertIsNone(_webhook_root_port(text))

    def test_rejects_suffix_after_digits(self):
        # 负向：8889suffix 不能被截成 8889
        for value in ("8889suffix", '"8889"suffix', "8889 suffix", '"8889suffix"', "88 89"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                _webhook_root_port(f"webhook:\n  port: {value}\n")

    def test_rejects_tab_indent(self):
        with self.assertRaises(ValueError):
            _webhook_root_port("webhook:\n\tport: 8889\n")

    def test_rejects_duplicate_and_out_of_range(self):
        for text in ('webhook:\n  port: "8889"\n  port: "8890"\n', "webhook:\n  port: 0\n", "webhook:\n  port: 65536\n"):
            with self.subTest(text=text), self.assertRaises(ValueError):
                _webhook_root_port(text)


class ComposeProbeEnvTests(unittest.TestCase):
    """P2-1 回归：compose 探测与执行必须用同一个隔离环境。

    模拟 Mac（OrbStack）布局：compose 插件只在 $HOME/.docker/cli-plugins，系统目录里没有。
    假 docker 按真实 CLI 的规则只在 ${DOCKER_CONFIG:-$HOME/.docker}/cli-plugins 找插件，找不到时退出 125。
    全部用运行时生成的 /bin/sh 替身，不需要真实 Docker，也不读取主机的 ~/.docker。
    """

    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.base = Path(temp.name)
        self.bin = self.base / "bin"
        self.bin.mkdir()
        self.home = self.base / "real-home"  # 隔离前的 HOME（相当于 Mac 上的 /Users/<name>）
        self.calls = self.base / "plugin-calls"
        self.calls.touch()
        self._script(self.bin / "docker", (
            '#!/bin/sh\n'
            'if [ "$1" = compose ]; then\n'
            '  p="${DOCKER_CONFIG:-$HOME/.docker}/cli-plugins/docker-compose"\n'
            '  if [ -x "$p" ]; then exec "$p" "$@"; fi\n'
            '  echo "docker: \'compose\' is not a docker command." >&2\n'
            '  exit 125\n'
            'fi\n'
            'exit 0\n'))
        self._script(self.bin / "jq", '#!/bin/sh\ncat >/dev/null\necho "{}"\n')
        self.environ = {"PATH": os.pathsep.join([str(self.bin), "/usr/bin", "/bin"]), "HOME": str(self.home)}

    @staticmethod
    def _script(path, body):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(body)
        path.chmod(0o700)

    def _plugin(self, directory):
        # 假插件：记录参数（不含任何值），version / config 一律成功
        self._script(directory / "docker-compose", (
            '#!/bin/sh\n'
            f'printf "%s\\n" "$*" >> "{self.calls}"\n'
            'case "$*" in *"--format json"*) echo "{}";; esac\n'
            'exit 0\n'))

    def _probe_then_run(self, extra=None):
        from unittest import mock
        with mock.patch.dict(os.environ, {**self.environ, **(extra or {})}, clear=True):
            self.assertTrue(_compose_tools(), "探测应找到 compose 插件")
            case = ReleaseComposeWebhookAddrTests("test_compose_config_quiet")
            case.setUp()  # 直接调用，绕过类级 skip，保证在没有真实 Docker 的机器上也会执行
            try:
                case.test_compose_config_quiet()
            finally:
                case.doCleanups()
        self.assertIn("config --quiet", self.calls.read_text())

    def test_plugin_only_in_home_docker_cli_plugins(self):
        self._plugin(self.home / ".docker/cli-plugins")
        self._probe_then_run()

    def test_docker_config_is_passed_through(self):
        custom = self.base / "custom-docker-config"
        self._plugin(custom / "cli-plugins")
        self._probe_then_run({"DOCKER_CONFIG": str(custom)})

    def test_probe_oserror_skips_instead_of_raising(self):
        from unittest import mock
        # 有执行位、shutil.which 能找到，但解释器不存在：exec 失败，subprocess 抛 OSError（FileNotFoundError）
        self._script(self.bin / "docker", "#!/nonexistent-r3inc/interpreter\n")
        # PATH 只留替身目录：否则 subprocess 遇到 ENOENT 会继续试 PATH 里后面的真实 docker
        with mock.patch.dict(os.environ, {**self.environ, "PATH": str(self.bin)}, clear=True):
            self.assertFalse(_compose_tools())

    def test_probe_timeout_skips_instead_of_raising(self):
        from unittest import mock
        timeout = subprocess.TimeoutExpired(["docker", "compose", "version"], 30)
        with mock.patch.dict(os.environ, self.environ, clear=True), \
                mock.patch.object(subprocess, "run", side_effect=timeout):
            self.assertFalse(_compose_tools())


if __name__ == "__main__":
    unittest.main()
