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
    """运行角色来自显式 DSN；镜像不得通过 MYSQL_USER 生成数据库级 ALL。"""
    text = COMPOSE.read_text()
    if re.search(r"^\s+MYSQL_USER:", text, re.M):
        raise AssertionError("MYSQL_USER would implicitly create an ALL-privilege runtime account")
    database = re.search(r"^\s+MYSQL_DATABASE:\s*(\S+)\s*$", text, re.M).group(1)
    backend = re.search(r"^\s+MYSQL_ADDR:\s*(.+?)\s*$", text, re.M).group(1)
    user = backend.split(":", 1)[0]
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

    def test_release_container_issuer_matches_repository_configuration(self):
        # The runtime image only copies the binary: a YAML-only issuer would stop startup.
        compose = COMPOSE.read_text()
        match = re.search(r"^\s+JWT_ISSUER:\s*\$\{JWT_ISSUER:-([^}]+)\}\s*$", compose, re.M)
        self.assertIsNotNone(match, "release backend must receive a nonempty JWT_ISSUER")
        for name in ("config.production.yaml", "config.development.yaml"):
            configured = re.search(r'^\s+issuer:\s*"([^"\n]+)"\s*$',
                                   (ROOT / "config" / name).read_text(), re.M)
            self.assertIsNotNone(configured)
            self.assertEqual(match.group(1), configured.group(1),
                             "changing issuer on container upgrade would invalidate existing tokens")

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
                         {"charset": "utf8mb4", "parseTime": "True", "loc": "UTC"})
        # 与 compose 中 backend 的 MYSQL_ADDR（用同一 MYSQL_PASSWORD 插值）逐字一致
        expanded = re.sub(r"\$\{MYSQL_PASSWORD:\?[^}]*\}", settings["MYSQL_PASSWORD"], backend_template)
        self.assertEqual(dsn, expanded)
        # .env 不加引号：值中不能出现空白、引号或 Compose 插值字符
        self.assertIsNone(re.search(r"[\s'\"$#]", dsn))
        self.assert_no_value_leak(result, dsn, settings["MYSQL_PASSWORD"])

    def test_fresh_credentials_are_separate_and_external_capabilities_start_disabled(self):
        destination, result = self.generate()
        settings = read_env(destination)
        names = ("MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "MYSQL_MIGRATOR_PASSWORD")
        self.assertEqual(len({settings[name] for name in names}), 3)
        for name in names:
            self.assertRegex(settings[name], r"^[a-f0-9]{48}$")
            self.assertNotIn(settings[name], result.stdout + result.stderr)
        self.assertRegex(settings["FRESH_STACK_OWNER"], r"^[a-f0-9]{32}$")
        self.assertEqual(settings["NOTIFICATION_ENABLED"], "false")
        self.assertEqual(settings["LLM_BASE_URL"], "http://127.0.0.1:9/v1")
        self.assertRegex(settings["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assertNotIn(settings["PROMETHEUS_HTTP_SD_TOKEN"], result.stdout + result.stderr)

    def test_generated_webhook_bot_defaults_are_fully_isolated(self):
        destination, result = self.generate()
        values = read_env(destination)
        expected = {
            "WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID": "",
            "WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET": "",
            "WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API": "http://127.0.0.1:9/open-apis/auth/v3/tenant_access_token/internal",
            "WEBHOOK_IM_FEISHU_PRIVATE_MESSAGE_API": "http://127.0.0.1:9/open-apis/im/v1/messages",
            "WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API": "http://127.0.0.1:9/open-apis/bot/v2/hook",
        }
        self.assertEqual({key: values.get(key) for key in expected}, expected)
        self.assertEqual(values["WEBHOOK_BACKEND_DOMAIN"], "http://127.0.0.1:8889/api/v1/alerts")
        self.assertRegex(values["WEBHOOK_ACTION_SIGNING_KEY"], r"^[0-9a-f]{64}$")
        self.assertRegex(values["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], r"^[0-9a-f]{64}$")
        self.assertNotEqual(values["WEBHOOK_ACTION_SIGNING_KEY"], values["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"])
        self.assertNotIn(values["WEBHOOK_ACTION_SIGNING_KEY"], result.stdout + result.stderr)
        self.assertNotIn(values["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], result.stdout + result.stderr)
        self.assertNotIn("@tcp(", result.stdout + result.stderr)

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
                                  "loc": "UTC", "charset": "utf8mb4"})

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
        expected = f"cloudops:{FIXTURE_PASSWORD}@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC"
        self.assertEqual(settings["WEBHOOK_MYSQL_ADDR"], expected)
        self.assertEqual(after.count("WEBHOOK_MYSQL_ADDR="), 1)
        self.assertRegex(settings["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assertEqual(after.count("PROMETHEUS_HTTP_SD_TOKEN="), 1)
        self.assertRegex(settings["WEBHOOK_ACTION_SIGNING_KEY"], r"^[0-9a-f]{64}$")
        self.assertRegex(settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], r"^[0-9a-f]{64}$")
        self.assert_no_value_leak(
            result, FIXTURE_PASSWORD, expected, settings["PROMETHEUS_HTTP_SD_TOKEN"],
            settings["WEBHOOK_ACTION_SIGNING_KEY"], settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"],
            settings["WEBHOOK_BACKEND_DOMAIN"],
        )
        # 再次执行：已有值不覆盖、不重复追加
        again = self.run_script("--env-file", str(destination), "--add-missing", "--mysql-host", "other:3306")
        self.assertEqual(again.returncode, 0, again.stderr)
        self.assertEqual(destination.read_text(), after)
        self.assert_no_value_leak(
            again, FIXTURE_PASSWORD, expected, settings["PROMETHEUS_HTTP_SD_TOKEN"],
            settings["WEBHOOK_ACTION_SIGNING_KEY"], settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"],
        )

    def test_add_missing_keeps_existing_custom_value(self):
        destination = self.dir / "custom.env"
        custom = "hook:fixture-custom@tcp(db.internal:3306)/hookdb?parseTime=True"
        destination.write_text(f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nWEBHOOK_MYSQL_ADDR={custom}\n")
        destination.chmod(0o600)
        before = destination.read_text()
        result = self.run_script("--env-file", str(destination), "--add-missing")
        self.assertEqual(result.returncode, 0, result.stderr)
        after = destination.read_text()
        self.assertTrue(after.startswith(before))
        settings = read_env(destination)
        self.assertEqual(settings["WEBHOOK_MYSQL_ADDR"], custom)
        self.assertEqual(settings["MYSQL_PASSWORD"], FIXTURE_PASSWORD)
        self.assertEqual(settings["WEBHOOK_BACKEND_DOMAIN"], "http://127.0.0.1:8889/api/v1/alerts")
        self.assertRegex(settings["WEBHOOK_ACTION_SIGNING_KEY"], r"^[0-9a-f]{64}$")
        self.assertRegex(settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], r"^[0-9a-f]{64}$")
        self.assertNotEqual(settings["WEBHOOK_ACTION_SIGNING_KEY"], settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"])
        self.assertRegex(settings["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assertEqual(after.count("PROMETHEUS_HTTP_SD_TOKEN="), 1)
        self.assert_no_value_leak(
            result, FIXTURE_PASSWORD, custom, settings["WEBHOOK_ACTION_SIGNING_KEY"],
            settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], settings["PROMETHEUS_HTTP_SD_TOKEN"],
        )
        again = self.run_script("--env-file", str(destination), "--add-missing")
        self.assertEqual(again.returncode, 0, again.stderr)
        self.assertEqual(destination.read_text(), after)
        self.assert_no_value_leak(
            again, FIXTURE_PASSWORD, custom, settings["WEBHOOK_ACTION_SIGNING_KEY"],
            settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], settings["PROMETHEUS_HTTP_SD_TOKEN"],
        )

    def test_add_missing_refusals_leave_file_unchanged(self):
        cases = (
            ("missing-password", "OTHER=keep\n", "MYSQL_PASSWORD"),
            ("empty-password", "MYSQL_PASSWORD=\n", "MYSQL_PASSWORD"),
            ("empty-placeholder", f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nWEBHOOK_MYSQL_ADDR=\n", "remove that line"),
            ("empty-http-sd-token", f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nWEBHOOK_MYSQL_ADDR=cloudops:{FIXTURE_PASSWORD}@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\nPROMETHEUS_HTTP_SD_TOKEN=\n", "PROMETHEUS_HTTP_SD_TOKEN"),
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

    def test_loopback_backend_placeholder_is_called_out_without_leaking_values(self):
        destination, result = self.generate()
        settings = read_env(destination)
        note = "loopback placeholder"
        output = result.stdout + result.stderr
        self.assertIn(note, result.stderr)
        self.assertNotIn("127.0.0.1", output)
        self.assertNotIn(settings["WEBHOOK_BACKEND_DOMAIN"], output)
        self.assertNotIn(settings["WEBHOOK_ACTION_SIGNING_KEY"], output)
        self.assertNotIn(settings["WEBHOOK_ALERTMANAGER_BEARER_TOKEN"], output)
        self.assertNotIn(settings["MYSQL_PASSWORD"], output)
        self.assertIn("Production", result.stderr)

        kept = self.dir / "kept-domain.env"
        kept.write_text(
            f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\n"
            "WEBHOOK_MYSQL_ADDR=cloudops:fixture-only-pw@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\n"
            "WEBHOOK_BACKEND_DOMAIN=https://alerts.example/api/v1/alerts\n"
            "WEBHOOK_ACTION_SIGNING_KEY=ab\n"
            "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=cd\n"
        )
        kept_result = self.run_script("--env-file", str(kept), "--add-missing")
        self.assertEqual(kept_result.returncode, 0, kept_result.stderr)
        self.assertNotIn(note, kept_result.stdout + kept_result.stderr)
        self.assertEqual(read_env(kept)["WEBHOOK_BACKEND_DOMAIN"], "https://alerts.example/api/v1/alerts")
        self.assertRegex(read_env(kept)["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assert_no_value_leak(kept_result, FIXTURE_PASSWORD, read_env(kept)["PROMETHEUS_HTTP_SD_TOKEN"])

        added = self.dir / "need-domain.env"
        added.write_text(
            f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\n"
            "WEBHOOK_MYSQL_ADDR=cloudops:fixture-only-pw@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\n"
            "WEBHOOK_ACTION_SIGNING_KEY=ab\n"
            "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=cd\n"
        )
        added_result = self.run_script("--env-file", str(added), "--add-missing")
        self.assertEqual(added_result.returncode, 0, added_result.stderr)
        added_output = added_result.stdout + added_result.stderr
        self.assertIn(note, added_result.stderr)
        self.assertNotIn("127.0.0.1", added_output)
        self.assertEqual(read_env(added)["WEBHOOK_BACKEND_DOMAIN"], "http://127.0.0.1:8889/api/v1/alerts")
        self.assertRegex(read_env(added)["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assert_no_value_leak(added_result, FIXTURE_PASSWORD, "fixture-only-pw@tcp", read_env(added)["PROMETHEUS_HTTP_SD_TOKEN"])

    def test_webhook_backend_domain_argument(self):
        domain = "http://alerts.ci.invalid/api/v1/alerts"
        created, created_result = self.generate("--webhook-backend-domain", domain)
        self.assertEqual(read_env(created)["WEBHOOK_BACKEND_DOMAIN"], domain)
        created_output = created_result.stdout + created_result.stderr
        self.assertNotIn("loopback placeholder", created_output)
        self.assertNotIn(domain, created_output)
        self.assertNotIn("127.0.0.1", created_output)

        https = "https://alerts.example/api/v1/alerts"
        https_path = self.dir / "https.env"
        https_result = self.run_script("--env-file", str(https_path), "--webhook-backend-domain", https)
        self.assertEqual(https_result.returncode, 0, https_result.stderr)
        self.assertEqual(read_env(https_path)["WEBHOOK_BACKEND_DOMAIN"], https)
        self.assertNotIn("loopback placeholder", https_result.stdout + https_result.stderr)
        self.assertNotIn(https, https_result.stdout + https_result.stderr)

        added = self.dir / "add-domain.env"
        added.write_text(
            f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\n"
            "WEBHOOK_MYSQL_ADDR=cloudops:fixture-only-pw@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\n"
            "WEBHOOK_ACTION_SIGNING_KEY=ab\n"
            "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=cd\n"
        )
        added_result = self.run_script("--env-file", str(added), "--add-missing", "--webhook-backend-domain", domain)
        self.assertEqual(added_result.returncode, 0, added_result.stderr)
        self.assertEqual(read_env(added)["WEBHOOK_BACKEND_DOMAIN"], domain)
        self.assertEqual(added.read_text().count("WEBHOOK_BACKEND_DOMAIN="), 1)
        self.assertRegex(read_env(added)["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        added_output = added_result.stdout + added_result.stderr
        self.assertNotIn("loopback placeholder", added_output)
        self.assertNotIn(domain, added_output)
        self.assert_no_value_leak(added_result, FIXTURE_PASSWORD, read_env(added)["PROMETHEUS_HTTP_SD_TOKEN"])

        # 不传时两条写入路径仍是回环占位，并保留 stderr 提示
        omitted = self.dir / "omitted.env"
        omitted_result = self.run_script("--env-file", str(omitted))
        self.assertEqual(omitted_result.returncode, 0, omitted_result.stderr)
        self.assertEqual(read_env(omitted)["WEBHOOK_BACKEND_DOMAIN"], "http://127.0.0.1:8889/api/v1/alerts")
        self.assertRegex(read_env(omitted)["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assertIn("loopback placeholder", omitted_result.stderr)
        self.assertNotIn("127.0.0.1", omitted_result.stdout + omitted_result.stderr)
        self.assertNotIn(read_env(omitted)["PROMETHEUS_HTTP_SD_TOKEN"], omitted_result.stdout + omitted_result.stderr)

        omitted_add = self.dir / "omitted-add.env"
        omitted_add.write_text(
            f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\n"
            "WEBHOOK_MYSQL_ADDR=cloudops:fixture-only-pw@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\n"
            "WEBHOOK_ACTION_SIGNING_KEY=ab\n"
            "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=cd\n"
        )
        omitted_add_result = self.run_script("--env-file", str(omitted_add), "--add-missing")
        self.assertEqual(omitted_add_result.returncode, 0, omitted_add_result.stderr)
        self.assertEqual(read_env(omitted_add)["WEBHOOK_BACKEND_DOMAIN"], "http://127.0.0.1:8889/api/v1/alerts")
        self.assertRegex(read_env(omitted_add)["PROMETHEUS_HTTP_SD_TOKEN"], r"^[a-f0-9]{48}$")
        self.assertIn("loopback placeholder", omitted_add_result.stderr)
        self.assertNotIn("127.0.0.1", omitted_add_result.stdout + omitted_add_result.stderr)
        self.assertNotIn(read_env(omitted_add)["PROMETHEUS_HTTP_SD_TOKEN"], omitted_add_result.stdout + omitted_add_result.stderr)

        # 已有值不覆盖，即使传入了另一个地址
        kept = self.dir / "keep-custom.env"
        custom = "https://kept.example/api/v1/alerts"
        kept.write_text(
            f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\n"
            "WEBHOOK_MYSQL_ADDR=cloudops:fixture-only-pw@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\n"
            f"WEBHOOK_BACKEND_DOMAIN={custom}\n"
            "WEBHOOK_ACTION_SIGNING_KEY=ab\n"
            "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=cd\n"
            "PROMETHEUS_HTTP_SD_TOKEN=fixture-http-sd-token\n"
        )
        kept_before = kept.read_bytes()
        kept_result = self.run_script("--env-file", str(kept), "--add-missing", "--webhook-backend-domain", domain)
        self.assertEqual(kept_result.returncode, 0, kept_result.stderr)
        self.assertEqual(kept.read_bytes(), kept_before)
        self.assert_no_value_leak(kept_result, FIXTURE_PASSWORD, "fixture-http-sd-token", custom, domain)
        self.assertNotIn("loopback placeholder", kept_result.stdout + kept_result.stderr)
        self.assertNotIn(domain, kept_result.stdout + kept_result.stderr)

        # 显式传入回环地址仍写入，并继续打回环提示，但不回显地址
        loopback = "http://localhost:8889/api/v1/alerts"
        loop_path = self.dir / "explicit-loopback.env"
        loop_result = self.run_script("--env-file", str(loop_path), "--webhook-backend-domain", loopback)
        self.assertEqual(loop_result.returncode, 0, loop_result.stderr)
        self.assertEqual(read_env(loop_path)["WEBHOOK_BACKEND_DOMAIN"], loopback)
        self.assertIn("loopback placeholder", loop_result.stderr)
        self.assertNotIn(loopback, loop_result.stdout + loop_result.stderr)
        self.assertNotIn("localhost", loop_result.stdout + loop_result.stderr)

        rejected = (
            ("space", "http://bad host.example/canary-space-9f3a2c"),
            ("dollar", "http://bad.example/canary-dollar-9f3a2c$secret"),
            ("quote", "http://bad.example/canary-quote-9f3a2c'"),
            ("dquote", 'http://bad.example/canary-dquote-9f3a2c"'),
            ("newline", "http://bad.example/canary-newline-9f3a2c\nINJECTED=1"),
            ("scheme", "ftp://bad.example/canary-scheme-9f3a2c"),
            ("noscheme", "canary-noscheme-9f3a2c.example/api"),
            ("nohost", "http:///canary-nohost-9f3a2c"),
            ("noport", "http://canary-noport-9f3a2c:notaport/api"),
            ("empty", ""),
        )
        for name, value in rejected:
            with self.subTest(case=name):
                marker = f"canary-{name}-9f3a2c" if name != "empty" else ""
                fresh = self.dir / f"rejected-fresh-{name}.env"
                fresh_result = self.run_script("--env-file", str(fresh), "--webhook-backend-domain", value)
                self.assertEqual(fresh_result.returncode, 2)
                self.assertIn("--webhook-backend-domain", fresh_result.stderr)
                self.assertFalse(fresh.exists())
                fresh_output = fresh_result.stdout + fresh_result.stderr
                if value:
                    self.assertNotIn(value, fresh_output)
                if marker:
                    self.assertNotIn(marker, fresh_output)
                    self.assertNotIn("INJECTED", fresh_output)

                existing = self.dir / f"rejected-add-{name}.env"
                existing.write_text(f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nOTHER=keep\n")
                before = existing.read_bytes()
                add_result = self.run_script(
                    "--env-file", str(existing), "--add-missing", "--webhook-backend-domain", value)
                self.assertEqual(add_result.returncode, 2)
                self.assertEqual(existing.read_bytes(), before)
                add_output = add_result.stdout + add_result.stderr
                self.assertIn("--webhook-backend-domain", add_result.stderr)
                if value:
                    self.assertNotIn(value, add_output)
                if marker:
                    self.assertNotIn(marker, add_output)
                self.assertNotIn(FIXTURE_PASSWORD, add_output)

    def test_webhook_backend_domain_rejects_unspecified_addresses(self):
        # 主机是未指定地址才拒绝。路径、查询、以及仅仅包含这些字符的主机名仍然合法。
        rejected = (
            ("v4", "http://0.0.0.0/canary-unspec-v4", ("0.0.0.0",)),
            ("v4-port", "http://0.0.0.0:8889/canary-unspec-v4port", ("0.0.0.0",)),
            ("v4-https", "https://0.0.0.0:443/canary-unspec-v4https", ("0.0.0.0",)),
            ("v4-dot", "http://0.0.0.0./canary-unspec-v4dot", ("0.0.0.0",)),
            ("v4-upper", "HTTP://0.0.0.0/canary-unspec-v4upper", ("0.0.0.0",)),
            ("v4-query", "http://0.0.0.0/canary-unspec-v4query?x=1", ("0.0.0.0",)),
            ("v6", "http://[::]/canary-unspec-v6", ("[::]",)),
            ("v6-https", "https://[::]/canary-unspec-v6https", ("[::]",)),
            ("v6-port", "http://[::]:8889/canary-unspec-v6port", ("[::]",)),
            ("v6-compress", "http://[::0]/canary-unspec-v6comp", ("[::0]",)),
            ("v6-expand", "http://[0::0]/canary-unspec-v6expand", ("[0::0]",)),
            ("v6-full", "http://[0:0:0:0:0:0:0:0]:9/canary-unspec-v6full", ("0:0:0:0:0:0:0:0",)),
            ("v4mapped", "http://[::ffff:0.0.0.0]/canary-unspec-v4mapped", ("::ffff:0.0.0.0", "0.0.0.0")),
            ("v4mapped-port", "http://[::FFFF:0:0]:8889/canary-unspec-v4mappedport", ("::FFFF:0:0", "::ffff:0:0")),
            ("v6-compat", "https://[::0.0.0.0]/canary-unspec-v6compat", ("::0.0.0.0",)),
            ("v6-zone", "http://[::%25lo0]:8889/canary-unspec-v6zone", ("::%25lo0", "[::")),
            ("v6-userinfo", "http://user:canary-unspec-userinfo@[::]:8889/api", ("canary-unspec-userinfo", "[::]")),
            ("v6-bare", "http://::/canary-unspec-v6bare", ("http://::/canary-unspec-v6bare",)),
            ("literal-v4", "0.0.0.0", ("0.0.0.0",)),
            ("literal-v6", "[::]", ("[::]",)),
        )
        for name, value, hidden in rejected:
            with self.subTest(case=name):
                fresh = self.dir / f"unspec-fresh-{name}.env"
                fresh_result = self.run_script("--env-file", str(fresh), "--webhook-backend-domain", value)
                self.assertEqual(fresh_result.returncode, 2)
                self.assertIn("--webhook-backend-domain", fresh_result.stderr)
                self.assertIn("WEBHOOK_BACKEND_DOMAIN", fresh_result.stderr)
                self.assertFalse(fresh.exists())
                fresh_output = fresh_result.stdout + fresh_result.stderr
                self.assertNotIn(value, fresh_output)
                for token in hidden:
                    self.assertNotIn(token, fresh_output)

                existing = self.dir / f"unspec-add-{name}.env"
                existing.write_text(f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\nOTHER=keep\n")
                before = existing.read_bytes()
                add_result = self.run_script(
                    "--env-file", str(existing), "--add-missing", "--webhook-backend-domain", value)
                self.assertEqual(add_result.returncode, 2)
                self.assertEqual(existing.read_bytes(), before)
                add_output = add_result.stdout + add_result.stderr
                self.assertIn("--webhook-backend-domain", add_result.stderr)
                self.assertIn("WEBHOOK_BACKEND_DOMAIN", add_result.stderr)
                self.assertNotIn(value, add_output)
                for token in hidden:
                    self.assertNotIn(token, add_output)
                self.assertNotIn(FIXTURE_PASSWORD, add_output)

        accepted = (
            "http://alerts.ci.invalid/api/v1/alerts",
            "https://alerts.example:8443/api/v1/alerts",
            "http://alerts.ci.invalid/0.0.0.0",
            "http://alerts.ci.invalid/api/v1/alerts?host=0.0.0.0&v6=[::]",
            "http://0.0.0.0.example/api/v1/alerts",
            "http://alerts.example/path/[::]/keep",
            "http://[::1]:8889/api/v1/alerts",
        )
        for domain in accepted:
            with self.subTest(domain=domain):
                destination = self.dir / "accepted.env"
                if destination.exists():
                    destination.unlink()
                result = self.run_script("--env-file", str(destination), "--webhook-backend-domain", domain)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(read_env(destination)["WEBHOOK_BACKEND_DOMAIN"], domain)
                output = result.stdout + result.stderr
                self.assertNotIn(domain, output)
                self.assertNotIn(FIXTURE_PASSWORD, output)

        added = self.dir / "accepted-add.env"
        added.write_text(
            f"MYSQL_PASSWORD={FIXTURE_PASSWORD}\n"
            "WEBHOOK_MYSQL_ADDR=cloudops:fixture-only-pw@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC\n"
            "WEBHOOK_ACTION_SIGNING_KEY=ab\n"
            "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=cd\n"
        )
        domain = "http://alerts.ci.invalid/api/v1/alerts"
        added_result = self.run_script("--env-file", str(added), "--add-missing", "--webhook-backend-domain", domain)
        self.assertEqual(added_result.returncode, 0, added_result.stderr)
        self.assertEqual(read_env(added)["WEBHOOK_BACKEND_DOMAIN"], domain)
        self.assertNotIn(domain, added_result.stdout + added_result.stderr)
        self.assert_no_value_leak(added_result, FIXTURE_PASSWORD)

    def test_fresh_stack_workflows_pass_non_loopback_backend_domain(self):
        domain = "http://alerts.ci.invalid/api/v1/alerts"
        generators = []
        for path in sorted((ROOT / ".github/workflows").glob("*.yml")):
            text = path.read_text()
            if "release-env.py" not in text:
                continue
            generators.append(path.name)
            self.assertIn("release-env.py --webhook-backend-domain " + domain, text, path.name)
        self.assertIn("integration.yml", generators)

    def test_add_missing_requires_existing_file(self):
        destination = self.dir / "absent.env"
        result = self.run_script("--env-file", str(destination), "--add-missing")
        self.assertEqual(result.returncode, 1)
        self.assertIn("does not exist", result.stderr)
        self.assertFalse(destination.exists())


if __name__ == "__main__":
    unittest.main()
