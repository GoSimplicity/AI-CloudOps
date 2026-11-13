"""Makefile 入口回归。历史 Docker 目标必须在调用 docker 之前失败。

`make -n` 只打印配方，不算执行。退役目标的断言会真正运行 make，
并用替身 docker 记录是否被调用；替身被调用即失败。
"""

import os
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
RETIRED = (
    "docker-build",
    "docker-start-env",
    "docker-start-server",
    "docker-stop-env",
    "docker-stop-server",
    "docker-net-remove",
    "dev",
    "stop",
)


class MakefileEntryTests(unittest.TestCase):
    def test_help_names_current_plan_and_refuses_to_execute_a_stack(self):
        result = subprocess.run(["make", "-s", "help"], cwd=ROOT, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        output = result.stdout + result.stderr
        self.assertIn("bash scripts/release-start.sh", output)
        self.assertIn("deploy/release/compose.yaml", output)
        self.assertNotIn("--execute-new-stack", output)
        self.assertNotIn("Bamboo/gomodd", output)

    def test_goimports_uses_module_path(self):
        dry = subprocess.run(["make", "-n", "fmt-imports"], cwd=ROOT, capture_output=True, text=True)
        self.assertEqual(dry.returncode, 0, dry.stderr)
        self.assertIn("goimports -w -local github.com/GoSimplicity/AI-CloudOps", dry.stdout)
        self.assertNotIn("github.com/yourusername", dry.stdout)
        self.assertNotIn("Bamboo/gomodd", (ROOT / "Makefile").read_text(encoding="utf-8"))

    def test_generate_and_tool_install_dry_run_does_not_execute(self):
        for target, needle in (
            ("generate", "go generate ./..."),
            ("install-dev-tools", "go install github.com/air-verse/air@latest"),
        ):
            with self.subTest(target=target):
                dry = subprocess.run(["make", "-n", target], cwd=ROOT, capture_output=True, text=True)
                self.assertEqual(dry.returncode, 0, dry.stderr)
                self.assertIn(needle, dry.stdout)

    def test_retired_docker_targets_fail_before_invoking_docker(self):
        with tempfile.TemporaryDirectory() as directory:
            bindir = Path(directory) / "bin"
            bindir.mkdir()
            marker = Path(directory) / "docker-invoked"
            stub = "#!/bin/sh\nprintf '%s\\n' \"$@\" >> " + repr(str(marker)) + "\nexit 86\n"
            for name in ("docker", "docker-compose"):
                path = bindir / name
                path.write_text(stub, encoding="utf-8")
                path.chmod(path.stat().st_mode | stat.S_IEXEC)
            env = os.environ.copy()
            env["PATH"] = str(bindir) + os.pathsep + env.get("PATH", "")
            for target in RETIRED:
                with self.subTest(target=target):
                    if marker.exists():
                        marker.unlink()
                    result = subprocess.run(
                        ["make", target], cwd=ROOT, env=env, capture_output=True, text=True
                    )
                    output = result.stdout + result.stderr
                    self.assertNotEqual(result.returncode, 0, output)
                    self.assertIn("已停用", output)
                    self.assertIn("bash scripts/release-start.sh", output)
                    self.assertIn("deploy/release/compose.yaml", output)
                    self.assertFalse(marker.exists(), output)
                    self.assertNotIn("--execute-new-stack", output)


if __name__ == "__main__":
    unittest.main()
