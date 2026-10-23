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


if __name__ == "__main__":
    unittest.main()
