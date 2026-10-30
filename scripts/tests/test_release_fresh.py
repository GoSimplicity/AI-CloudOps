"""Fresh startup safety tests: no Docker, browsers or database connections."""
import copy
import importlib.util
import io
import json
import os
from pathlib import Path
import secrets
import tempfile
import unittest
from contextlib import redirect_stdout, redirect_stderr
from unittest.mock import patch

MODULE = Path(__file__).resolve().parents[1] / "release-fresh.py"
spec = importlib.util.spec_from_file_location("release_fresh", MODULE)
tool = importlib.util.module_from_spec(spec)
spec.loader.exec_module(tool)
PROJECT = "cloudops-fresh-123456789abc"


def settings():
    values = {name: secrets.token_hex(24) for name in ("MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "MYSQL_MIGRATOR_PASSWORD")}
    values.update(FRESH_STACK_OWNER=secrets.token_hex(16), BIND_ADDRESS="127.0.0.1", HTTP_PORT="18080", PUBLIC_ORIGIN="http://localhost:18080", NOTIFICATION_ENABLED="false", FEISHU_WEBHOOK_URL="", LLM_BASE_URL="http://127.0.0.1:9/v1")
    values["WEBHOOK_MYSQL_ADDR"] = f'cloudops:{values["MYSQL_PASSWORD"]}@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC'
    return values


def config(values):
    label = {tool.OWNER_LABEL: values["FRESH_STACK_OWNER"]}
    result = {"services": {name: {"labels": label.copy(), "environment": {}} for name in ("mysql", "redis", "schema", "backend", "webhook", "gateway", "aiops")}, "volumes": {"mysql": {"name": PROJECT + "_mysql", "labels": label.copy()}}, "networks": {"default": {"name": PROJECT + "_default", "labels": label.copy()}}}
    result["services"]["mysql"]["environment"] = {"MYSQL_DATABASE": "cloudops", "MYSQL_ROOT_PASSWORD": values["MYSQL_ROOT_PASSWORD"]}
    result["services"]["backend"]["environment"] = {"MYSQL_ADDR": values["WEBHOOK_MYSQL_ADDR"]}
    result["services"]["webhook"]["environment"] = {"WEBHOOK_MYSQL_ADDR": values["WEBHOOK_MYSQL_ADDR"]}
    result["services"]["schema"]["image"] = "cloudops-schema:fixture"
    result["services"]["aiops"]["environment"] = {"LLM_API_KEY": "", "LLM_BASE_URL": "http://127.0.0.1:9/v1", "NOTIFICATION_ENABLED": "false", "FEISHU_WEBHOOK": ""}
    result["services"]["gateway"]["ports"] = [{"host_ip": "127.0.0.1", "published": "18080", "target": 8080}]
    return result


def plan(section):
    user = "cloudops_migrator" if section == "migrator" else "cloudops"
    return json.dumps({"version": 1, "database": "cloudops", "section": section, "statements": [f"GRANT SELECT ON `cloudops`.`fixture` TO '{user}'@'%'"]}).encode()


class FakeRunner(tool.FreshRunner):
    def __init__(self, root, values):
        super().__init__(root, PROJECT, Path("/fixture.sock"), values)
        self.config = config(values)
        self.commands = []
        self.tools = []
        self.admin_calls = []
        self.existing = False
        self.foreign_name = False
        self.counts = b"0\n0\n"
        self.fail_migrate = False

    def command(self, args, payload=None, timeout=120):
        self.commands.append((args, payload))
        if args[-3:] == ["config", "--format", "json"]:
            return json.dumps(self.config).encode()
        if "--filter" in args and "ps" in args and "-aq" in args:
            return b"prior-container" if self.existing else b""
        if "--format" in args and args[-1] == "{{.Name}}":
            return (PROJECT + "_mysql\n").encode() if self.foreign_name else b""
        if "--format" in args and "ps" in args:
            return f'{"a" * 12} mysql {self.owner}\n{"b" * 12} redis {self.owner}\n'.encode()
        return b""

    def tool(self, args, dsn=None):
        self.tools.append((args, dsn))
        if args[0] == "roles":
            return plan(args[args.index("--section") + 1])
        if args[0] == "migrate" and self.fail_migrate:
            raise tool.FreshError("private-driver-diagnostic")
        return b""

    def admin(self, sql):
        self.admin_calls.append(sql)
        return self.counts if sql.startswith("SELECT COUNT(*)") else b""


class FreshReleaseTests(unittest.TestCase):
    def test_default_plan_never_reads_config_or_calls_docker(self):
        with patch.object(tool, "private_settings") as read, patch.object(tool.FreshRunner, "execute") as execute, redirect_stdout(io.StringIO()):
            self.assertEqual(tool.main([]), 0)
            read.assert_not_called(); execute.assert_not_called()

    def test_invalid_execution_identity_refuses_before_credentials_and_docker(self):
        with patch.object(tool, "private_settings") as read, patch.object(tool.FreshRunner, "execute") as execute, redirect_stderr(io.StringIO()):
            for args in [["--execute-new-stack"], ["--execute-new-stack", "--project", "cloudops"], ["--execute-new-stack", "--project", "production"]]:
                self.assertEqual(tool.main(args), 1)
            read.assert_not_called(); execute.assert_not_called()

    def test_invalid_arguments_never_echo_supplied_values(self):
        output = io.StringIO()
        with redirect_stderr(output), patch.object(tool, "private_settings") as read:
            self.assertEqual(tool.main(["--unsupported=private-argument"]), 2)
            read.assert_not_called()
        self.assertNotIn("private-argument", output.getvalue())

    def test_private_inputs_reject_symlink_public_permissions_duplicate_keys_and_reused_passwords(self):
        values = settings()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "fixture.env"
            def write(data):
                path.write_text("".join(key + "=" + value + "\n" for key, value in data.items())); path.chmod(0o600)
            write(values)
            self.assertEqual(tool.private_settings(path), values)
            path.chmod(0o644)
            with self.assertRaises(tool.FreshError): tool.private_settings(path)
            path.chmod(0o600)
            link = Path(directory) / "link"; link.symlink_to(path)
            with self.assertRaises(tool.FreshError): tool.private_settings(link)
            with path.open("a") as out: out.write("MYSQL_PASSWORD=ignored\n")
            with self.assertRaises(tool.FreshError): tool.private_settings(path)
            write({**values, "MYSQL_MIGRATOR_PASSWORD": values["MYSQL_PASSWORD"]})
            with self.assertRaises(tool.FreshError): tool.private_settings(path)
            write({**values, "NOTIFICATION_ENABLED": "true"})
            with self.assertRaises(tool.FreshError): tool.private_settings(path)

    def test_runtime_all_privileges_external_resources_and_credential_leaks_refused(self):
        values = settings(); original = config(values)
        self.assertEqual(tool.validate_config(original, values, PROJECT)[0], "cloudops-schema:fixture")
        mutations = [lambda data: data["services"]["mysql"]["environment"].update(MYSQL_USER="cloudops"), lambda data: data["services"]["backend"]["environment"].update(ADMIN=values["MYSQL_ROOT_PASSWORD"]), lambda data: data["services"]["webhook"]["environment"].update(MIGRATOR=values["MYSQL_MIGRATOR_PASSWORD"]), lambda data: data["services"]["schema"]["environment"].update(SECRET="value"), lambda data: data["volumes"]["mysql"].update(external=True), lambda data: data["networks"]["default"].update(external=True), lambda data: data["services"]["gateway"]["ports"][0].update(host_ip="0.0.0.0"), lambda data: data["services"]["backend"]["labels"].clear()]
        for mutate in mutations:
            data = copy.deepcopy(original); mutate(data)
            with self.assertRaises(tool.FreshError): tool.validate_config(data, values, PROJECT)

    def test_role_plan_is_structured_and_never_shell_split(self):
        self.assertEqual(len(tool.role_statements(plan("runtime"), "runtime")), 1)
        for mutation in [lambda data: data.update(section="all"), lambda data: data.update(database="mysql"), lambda data: data.update(statements=[]), lambda data: data["statements"].append("GRANT ALL PRIVILEGES ON x.* TO 'cloudops'@'%'"), lambda data: data["statements"].append("GRANT SELECT ON x.y TO 'cloudops'@'%'; DELETE FROM x.y"), lambda data: data.update(statements=["GRANT SELECT ON x.y TO 'other'@'%'"])]:
            data = json.loads(plan("runtime")); mutation(data)
            with self.assertRaises(tool.FreshError): tool.role_statements(json.dumps(data).encode(), "runtime")

    def test_ambient_compose_values_never_override_the_private_file(self):
        ambient = {"LLM_API_KEY": "ambient-private", "LLM_BASE_URL": "https://example.invalid/v1", "NOTIFICATION_ENABLED": "true", "FEISHU_WEBHOOK_URL": "ambient-hook", "MYSQL_ROOT_PASSWORD": "ambient-root", "NPM_REGISTRY": "https://example.invalid", "COMPOSE_FILE": "/another/compose.yml"}
        with patch.dict(os.environ, ambient):
            runner = tool.FreshRunner(Path("/fixture"), PROJECT, Path("/fixture.sock"), settings())
        for name in ambient:
            self.assertNotIn(name, runner.environment)
        self.assertEqual(runner.environment["COMPOSE_PROJECT_NAME"], PROJECT)

    def test_rendered_external_capabilities_are_rejected_even_with_safe_private_file(self):
        values = settings()
        for name, value in (("LLM_API_KEY", "rendered-private"), ("LLM_BASE_URL", "https://example.invalid/v1"), ("NOTIFICATION_ENABLED", "true"), ("FEISHU_WEBHOOK", "rendered-hook")):
            data = config(values)
            data["services"]["aiops"]["environment"][name] = value
            with self.assertRaises(tool.FreshError): tool.validate_config(data, values, PROJECT)

    def run_fixture(self, function):
        with tempfile.TemporaryDirectory() as directory, redirect_stdout(io.StringIO()):
            root = Path(directory) / "AI-CloudOps"; root.mkdir()
            for name in ("AI-CloudOps-web", "AI-CloudOps-aiops"):
                sibling = root.parent / name; sibling.mkdir(); (sibling / "Dockerfile").touch()
            runner = FakeRunner(root, settings())
            return function(runner)

    def test_existing_project_or_foreign_storage_fails_before_build_or_account_mutation(self):
        def check(runner):
            for name in ("existing", "foreign_name"):
                setattr(runner, name, True)
                with self.assertRaises(tool.FreshError): runner.execute()
                self.assertFalse(any("build" in args for args, _ in runner.commands))
                self.assertEqual(runner.admin_calls, [])
                self.assertEqual(runner.tools, [])
                setattr(runner, name, False)
        self.run_fixture(check)

    def test_full_metadata_preflight_precedes_every_mutation(self):
        def check(runner):
            runner.counts = b"1\n0\n"
            with self.assertRaises(tool.FreshError): runner.execute()
            self.assertEqual(len(runner.admin_calls), 1)
            self.assertNotIn("CREATE USER", runner.admin_calls[0])
            self.assertFalse(any(args[0] == "migrate" for args, _ in runner.tools))
        self.run_fixture(check)

    def test_migration_failure_preserves_state_and_never_grants_runtime_or_starts_writers(self):
        def check(runner):
            runner.fail_migrate = True
            with self.assertRaises(tool.FreshError): runner.execute()
            self.assertEqual(runner.phase, "migrate")
            self.assertEqual(len(runner.admin_calls), 3)
            self.assertFalse(any(args[0] == "verify" for args, _ in runner.tools))
            for args, _ in runner.commands:
                self.assertFalse(any(word in args for word in ("down", "rm", "rmi", "prune")))
                self.assertNotIn("--no-build", args)
        self.run_fixture(check)

    def test_fresh_success_obeys_role_and_clock_order(self):
        def check(runner):
            runner.execute()
            self.assertEqual([args[0] for args, _ in runner.tools], ["roles", "roles", "migrate", "init-clock", "verify", "verify-clock"])
            self.assertEqual(len(runner.admin_calls), 4)
            self.assertIn("CREATE USER 'cloudops'@'%'", runner.admin_calls[1])
            self.assertIn("CREATE USER 'cloudops_migrator'@'%'", runner.admin_calls[1])
            self.assertNotIn("IF NOT EXISTS", runner.admin_calls[1])
            self.assertIn("cloudops_migrator", runner.admin_calls[2])
            self.assertNotIn("cloudops_migrator", runner.admin_calls[3])
            self.assertIn("--no-build", runner.commands[-1][0])
        self.run_fixture(check)

    def test_tool_credentials_are_stdin_only_on_private_tmpfs(self):
        values = settings(); runner = tool.FreshRunner(Path("/fixture"), PROJECT, Path("/fixture.sock"), values)
        runner.image = "cloudops-schema:fixture"; runner.network = PROJECT + "_default"
        with patch.object(runner, "command", return_value=b"") as command:
            runner.tool(["verify"], values["WEBHOOK_MYSQL_ADDR"])
        args, payload = command.call_args.args
        self.assertNotIn(values["MYSQL_PASSWORD"], " ".join(args))
        self.assertIn("--tmpfs", args)
        self.assertEqual(payload, (values["WEBHOOK_MYSQL_ADDR"] + "\n").encode())
        self.assertNotIn("MYSQL_ROOT_PASSWORD", runner.environment)
        self.assertNotIn("MYSQL_MIGRATOR_PASSWORD", runner.environment)


if __name__ == "__main__":
    unittest.main()
