"""release-env.py 的 WEBHOOK_MYSQL_ADDR 生成与新参数回归：只用临时目录和合成口令，不访问 Docker 或数据库"""

import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/release-env.py"
COMPOSE = ROOT / "deploy/release/compose.yaml"
# 合成口令：仅用于测试，满足 SAFE_VALUE 字符集
FIXTURE_PASSWORD = "fixture-only-pw"
# go-sql-driver DSN：user:passwd@net(addr)/dbname?params（webhook 经 gorm.io/driver/mysql 调用 mysql.ParseDSN）
DSN_PATTERN = re.compile(r"^(?P<user>[^:@/]+):(?P<passwd>[^@/]+)@(?P<net>tcp)\((?P<addr>[^)]+)\)/(?P<db>[^?/]+)\?(?P<params>.+)$")

GO_PARSE_DSN = """package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func main() {
	line, _ := bufio.NewReader(os.Stdin).ReadString('\\n')
	// 与 webhook 加载逻辑一致：strings.TrimSpace(os.Getenv("WEBHOOK_MYSQL_ADDR")) 后交给 mysql.Open
	c, err := mysql.ParseDSN(strings.TrimSpace(line))
	if err != nil {
		fmt.Fprintln(os.Stderr, "DSN_PARSE_FAILED")
		os.Exit(3)
	}
	out, _ := json.Marshal(map[string]any{"user": c.User, "passwd": c.Passwd, "net": c.Net, "addr": c.Addr,
		"db": c.DBName, "parseTime": c.ParseTime, "loc": c.Loc.String(), "charset": c.Params["charset"]})
	fmt.Println(string(out))
}
"""


def read_env(path):
    return dict(line.split("=", 1) for line in path.read_text().splitlines()
                if line and not line.startswith("#") and "=" in line)


def compose_mysql_settings():
    """从发布 compose 读取 mysql 服务的 MYSQL_USER / MYSQL_DATABASE 与 backend 的 MYSQL_ADDR 模板（纯文本解析，无需 PyYAML）"""
    text = COMPOSE.read_text()
    user = re.search(r"^\s+MYSQL_USER:\s*(\S+)\s*$", text, re.M).group(1)
    database = re.search(r"^\s+MYSQL_DATABASE:\s*(\S+)\s*$", text, re.M).group(1)
    backend = re.search(r"^\s+MYSQL_ADDR:\s*(.+?)\s*$", text, re.M).group(1)
    return user, database, backend


class ReleaseEnvTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.dir = Path(self.temp.name)

    def run_script(self, *arguments):
        return subprocess.run([sys.executable, str(SCRIPT), *arguments],
                              capture_output=True, text=True, timeout=30)

    def assert_no_value_leak(self, result, *values):
        output = result.stdout + result.stderr
        for value in values:
            self.assertNotIn(value, output)
        self.assertNotIn("@tcp(", output)

    def generate(self, *extra):
        destination = self.dir / "gen.env"
        result = self.run_script("--env-file", str(destination), *extra)
        self.assertEqual(result.returncode, 0, result.stderr)
        return destination, result

    def test_generated_dsn_matches_mysql_variables_and_backend_dsn(self):
        destination, result = self.generate()
        settings = read_env(destination)
        dsn = settings["WEBHOOK_MYSQL_ADDR"]
        match = DSN_PATTERN.match(dsn)
        self.assertIsNotNone(match, "WEBHOOK_MYSQL_ADDR 不符合 go-sql-driver DSN 格式")
        user, database, backend_template = compose_mysql_settings()
        self.assertEqual(match["user"], user)
        self.assertEqual(match["passwd"], settings["MYSQL_PASSWORD"])
        self.assertEqual(match["addr"], "mysql:3306")
        self.assertEqual(match["db"], database)
        self.assertEqual(dict(p.split("=", 1) for p in match["params"].split("&")),
                         {"charset": "utf8mb4", "parseTime": "True", "loc": "Local"})
        # 与 compose 中 backend 的 MYSQL_ADDR（用同一 MYSQL_PASSWORD 插值）逐字一致
        expanded = re.sub(r"\$\{MYSQL_PASSWORD:\?[^}]*\}", settings["MYSQL_PASSWORD"], backend_template)
        self.assertEqual(dsn, expanded)
        # .env 不加引号：值中不能出现空白、引号或 Compose 插值字符
        self.assertIsNone(re.search(r"[\s'\"$#]", dsn))
        self.assert_no_value_leak(result, dsn, settings["MYSQL_PASSWORD"])

    def test_generated_dsn_parses_with_go_sql_driver(self):
        go = shutil.which("go")
        if go is None:
            self.skipTest("go 不可用，跳过真实驱动解析（正则校验仍在上一用例执行）")
        destination, _ = self.generate()
        settings = read_env(destination)
        module = self.dir / "dsncheck"
        module.mkdir()
        (module / "go.mod").write_text("module dsncheck\n\ngo 1.22\n\nrequire github.com/go-sql-driver/mysql v1.8.1\n")
        (module / "main.go").write_text(GO_PARSE_DSN)
        # 只用本机模块缓存，不联网、不自动切换工具链
        env = {**os.environ, "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOFLAGS": "-mod=mod", "GOSUMDB": "off"}
        result = subprocess.run([go, "run", "."], cwd=module, input=settings["WEBHOOK_MYSQL_ADDR"] + "\n",
                                env=env, capture_output=True, text=True, timeout=180)
        if result.returncode != 0 and "DSN_PARSE_FAILED" not in result.stderr:
            self.skipTest("本机模块缓存缺少 go-sql-driver/mysql v1.8.1，跳过真实驱动解析")
        self.assertEqual(result.returncode, 0, "go-sql-driver 解析 WEBHOOK_MYSQL_ADDR 失败")
        parsed = json.loads(result.stdout)
        user, database, _ = compose_mysql_settings()
        self.assertEqual(parsed, {"user": user, "passwd": settings["MYSQL_PASSWORD"], "net": "tcp",
                                  "addr": "mysql:3306", "db": database, "parseTime": True,
                                  "loc": "Local", "charset": "utf8mb4"})

    def test_env_file_and_mysql_host_parameters(self):
        destination, result = self.generate("--mysql-host", "127.0.0.1:13306")
        self.assertEqual(stat.S_IMODE(destination.stat().st_mode), 0o600)
        match = DSN_PATTERN.match(read_env(destination)["WEBHOOK_MYSQL_ADDR"])
        self.assertEqual(match["addr"], "127.0.0.1:13306")
        self.assertIn(str(destination), result.stdout)
        for host in ("mysql", "mysql:3306/x", "my sql:3306", "mysql:abc", "a@b:3306"):
            with self.subTest(host=host):
                target = self.dir / "rejected.env"
                bad = self.run_script("--env-file", str(target), "--mysql-host", host)
                self.assertEqual(bad.returncode, 2)
                self.assertIn("--mysql-host", bad.stderr)
                self.assertFalse(target.exists())

    def test_existing_file_never_overwritten_without_add_missing(self):
        destination = self.dir / "existing.env"
        destination.write_text(f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nOTHER=keep\n")
        before = destination.read_bytes()
        result = self.run_script("--env-file", str(destination))
        self.assertEqual(result.returncode, 1)
        self.assertIn("--add-missing", result.stderr)
        self.assertEqual(destination.read_bytes(), before)
        self.assert_no_value_leak(result, FIXTURE_PASSWORD)

    def test_add_missing_appends_only_dsn_and_sets_0600(self):
        destination = self.dir / "upgrade.env"
        # 无结尾换行 + 0644，验证追加时补换行且收紧权限
        destination.write_text(f"# note\nMYSQL_PASSWORD={FIXTURE_PASSWORD}\nOTHER=keep")
        destination.chmod(0o644)
        before = destination.read_text()
        result = self.run_script("--env-file", str(destination), "--add-missing")
        self.assertEqual(result.returncode, 0, result.stderr)
        after = destination.read_text()
        self.assertTrue(after.startswith(before + "\n"), "原有内容必须原样保留")
        self.assertEqual(stat.S_IMODE(destination.stat().st_mode), 0o600)
        settings = read_env(destination)
        self.assertEqual(settings["OTHER"], "keep")
        self.assertEqual(settings["MYSQL_PASSWORD"], FIXTURE_PASSWORD)
        expected = f"cloudops:{FIXTURE_PASSWORD}@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=Local"
        self.assertEqual(settings["WEBHOOK_MYSQL_ADDR"], expected)
        self.assertEqual(after.count("WEBHOOK_MYSQL_ADDR="), 1)
        self.assert_no_value_leak(result, FIXTURE_PASSWORD, expected)
        # 再次执行：已有值不覆盖、不重复追加
        again = self.run_script("--env-file", str(destination), "--add-missing", "--mysql-host", "other:3306")
        self.assertEqual(again.returncode, 0, again.stderr)
        self.assertEqual(destination.read_text(), after)
        self.assert_no_value_leak(again, FIXTURE_PASSWORD, expected)

    def test_add_missing_keeps_existing_custom_value(self):
        destination = self.dir / "custom.env"
        custom = "hook:fixture-custom@tcp(db.internal:3306)/hookdb?parseTime=True"
        destination.write_text(f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nWEBHOOK_MYSQL_ADDR={custom}\n")
        destination.chmod(0o600)
        before = destination.read_bytes()
        result = self.run_script("--env-file", str(destination), "--add-missing")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(destination.read_bytes(), before)
        self.assert_no_value_leak(result, FIXTURE_PASSWORD, custom)

    def test_add_missing_refusals_leave_file_unchanged(self):
        cases = (
            ("missing-password", "OTHER=keep\n", "MYSQL_PASSWORD"),
            ("empty-password", "MYSQL_PASSWORD=\n", "MYSQL_PASSWORD"),
            ("empty-placeholder", f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nWEBHOOK_MYSQL_ADDR=\n", "remove that line"),
            ("unsafe-password", "MYSQL_PASSWORD=fixture@pw/x\n", "unsafe"),
        )
        for name, content, message in cases:
            with self.subTest(case=name):
                destination = self.dir / f"{name}.env"
                destination.write_text(content)
                before = destination.read_bytes()
                result = self.run_script("--env-file", str(destination), "--add-missing")
                self.assertEqual(result.returncode, 1)
                self.assertIn(message, result.stderr)
                self.assertEqual(destination.read_bytes(), before)
                self.assert_no_value_leak(result, FIXTURE_PASSWORD, "fixture@pw/x")

    def test_add_missing_requires_existing_file(self):
        destination = self.dir / "absent.env"
        result = self.run_script("--env-file", str(destination), "--add-missing")
        self.assertEqual(result.returncode, 1)
        self.assertIn("does not exist", result.stderr)
        self.assertFalse(destination.exists())


if __name__ == "__main__":
    unittest.main()
