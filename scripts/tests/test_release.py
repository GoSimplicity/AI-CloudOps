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


def _compose_tools():
    return shutil.which("docker") and shutil.which("jq") and subprocess.run(
        ["docker", "compose", "version"], capture_output=True, timeout=30).returncode == 0


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
        # 干净环境：不继承调用者导出的变量，避免覆盖合成值
        self.env = {"PATH": os.environ.get("PATH", ""), "HOME": self.temp.name}
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

    def test_backend_alert_webhook_addr_targets_webhook_port(self):
        from urllib.parse import urlsplit
        import re
        addr = self._jq('.services.backend.environment.PROMETHEUS_ALERT_WEBHOOK_ADDR // ""')
        self.assertTrue(addr, "backend 未设置 PROMETHEUS_ALERT_WEBHOOK_ADDR（代码默认 localhost:8889 在容器里指向 backend 自己）")
        url = urlsplit(addr if "://" in addr else "http://" + addr)
        self.assertEqual(url.hostname, "webhook")
        self.assertEqual(self._jq('.services.webhook | type'), "object", "compose 里没有 webhook 服务")
        self.assertEqual(self._jq('.services.backend.network_mode // ""'), "", "backend 用了 network_mode，服务名不可解析")
        self.assertEqual(self._jq('.services.webhook.environment.WEBHOOK_PORT // ""'), "",
                         "webhook 设了 WEBHOOK_PORT，需同步检查本测试")
        # webhook 实际端口：镜像内 config/webhook.yaml 的 webhook.port
        port, in_webhook = None, False
        for line in (ROOT / "config/webhook.yaml").read_text().splitlines():
            if line and not line[0].isspace():
                in_webhook = line.split("#")[0].strip() == "webhook:"
                continue
            m = re.match(r'^  port:[ \t]*"?(\d+)"?', line) if in_webhook else None
            if m:
                port = int(m.group(1))
        self.assertIsNotNone(port, "config/webhook.yaml 里没找到 webhook.port（必须在 webhook: 根节点下）")
        self.assertEqual(url.port or 80, port)
        self.assertIn(url.path.rstrip("/"), ("", "/api/v1/alerts/receive"))
        # 与 webhook 健康检查、alertmanager.yml 的端口一致
        health = self._jq('.services.webhook.healthcheck.test | join(" ")')
        self.assertIn(f"127.0.0.1:{url.port}/", health)
        self.assertIn(f"http://webhook:{url.port}/api/v1/alerts/receive",
                      (ROOT / "deploy/release/alertmanager.yml").read_text())


if __name__ == "__main__":
    unittest.main()
