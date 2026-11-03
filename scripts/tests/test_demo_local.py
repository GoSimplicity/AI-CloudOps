"""Local demo guard tests; all process, Docker, and database operations are fake."""

import importlib.util
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch


SCRIPT = Path(__file__).resolve().parents[1] / "demo-local.py"
SPEC = importlib.util.spec_from_file_location("demo_local", SCRIPT)
DEMO = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DEMO)
CANARY = "fixture-private-canary"
DSN = f"cloudops_demo:{CANARY}@tcp(127.0.0.1:13306)/cloudops_demo_fixture?charset=utf8mb4&parseTime=True&loc=UTC"


class DemoLocalTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.directory.chmod(0o700)
        self.settings = {
            "ENV": "development", "SERVER_PORT": "8889", "MYSQL_ADDR": DSN,
            "REDIS_ADDR": "127.0.0.1:16379", "REDIS_PASSWORD": CANARY,
            "MOCK_ENABLED": "false", "CRON_ALLOW_LOCAL_EXEC": "false",
            "NOTIFICATION_EMAIL_ENABLED": "false", "NOTIFICATION_FEISHU_ENABLED": "false",
            "PROMETHEUS_ENABLE_ALERT": "0", "PROMETHEUS_ENABLE_RECORD": "0",
            "AIOPS_BASE_URL": "http://127.0.0.1:18081",
        }
        self.write_environment()
        self.write("compose.env", f"MYSQL_ROOT_PASSWORD={CANARY}\nREDIS_PASSWORD={CANARY}\n")
        self.write("schema.dsn", DSN + "\n")

    def write(self, name, content):
        destination = self.directory / name
        destination.write_text(content)
        destination.chmod(0o600)
        return destination

    def write_environment(self):
        self.write("runtime.env", "".join(f"{key}='{value}'\n" for key, value in self.settings.items()))

    def test_prepared_configuration_accepts_only_exact_local_contract(self):
        self.assertEqual(DEMO.LocalDemo(self.directory).database, "cloudops_demo_fixture")
        for key, value in (("ENV", "production"), ("MOCK_ENABLED", "true"),
                           ("CRON_ALLOW_LOCAL_EXEC", "true"), ("NOTIFICATION_EMAIL_ENABLED", "true"),
                           ("NOTIFICATION_FEISHU_ENABLED", "true"), ("REDIS_ADDR", "example.invalid:6379"),
                           ("AIOPS_BASE_URL", "https://example.invalid")):
            with self.subTest(key=key):
                original = self.settings[key]
                self.settings[key] = value
                self.write_environment()
                with self.assertRaises(DEMO.DemoError):
                    DEMO.LocalDemo(self.directory)
                self.settings[key] = original
        self.write_environment()

    def test_remote_ambiguous_and_non_demo_dsns_are_rejected_without_echo(self):
        for value in (DSN.replace("127.0.0.1", "localhost"), DSN.replace("127.0.0.1", "192.0.2.1"),
                      DSN.replace("13306", "3306"), DSN.replace("cloudops_demo_fixture", "mysql"),
                      DSN + "&multiStatements=true", DSN.replace("cloudops_demo:", "root:")):
            with self.subTest(value=value.split("@")[-1]):
                with self.assertRaises(DEMO.DemoError) as failure:
                    DEMO.demo_database(value)
                self.assertNotIn(CANARY, str(failure.exception))

    def test_credentials_must_be_private_regular_owned_files(self):
        file = self.directory / "runtime.env"
        file.chmod(0o644)
        with self.assertRaises(DEMO.DemoError):
            DEMO.LocalDemo(self.directory)
        file.chmod(0o600)
        file.rename(self.directory / "actual.env")
        file.symlink_to(self.directory / "actual.env")
        with self.assertRaises(DEMO.DemoError):
            DEMO.LocalDemo(self.directory)

    def test_duplicate_environment_entries_fail_closed(self):
        self.write("runtime.env", "ENV=development\nENV=production\n")
        with self.assertRaises(DEMO.DemoError):
            DEMO.LocalDemo(self.directory)

    def test_schema_and_runtime_must_refer_to_the_same_database(self):
        self.write("schema.dsn", DSN.replace("cloudops_demo_fixture", "cloudops_demo_other"))
        with self.assertRaises(DEMO.DemoError):
            DEMO.LocalDemo(self.directory)

    def test_remote_docker_is_rejected_before_daemon_access(self):
        demo = DEMO.LocalDemo(self.directory)
        with patch.dict(os.environ, {"DOCKER_HOST": "ssh://example.invalid"}, clear=True), patch.object(DEMO, "execute") as execute:
            with self.assertRaises(DEMO.DemoError):
                demo.check_docker()
            execute.assert_not_called()
        contexts = [{"Endpoints": {"docker": {"Host": "tcp://example.invalid:2376"}}}]
        with patch.dict(os.environ, {"DOCKER_CONTEXT": "remote"}, clear=True), patch.object(DEMO, "execute", return_value=json.dumps(contexts)) as execute:
            with self.assertRaises(DEMO.DemoError):
                demo.check_docker()
            execute.assert_called_once_with(["docker", "context", "inspect", "remote"])

    def test_recorded_unrelated_pid_is_never_signaled(self):
        self.write("backend.pid", "4321\n")
        demo = DEMO.LocalDemo(self.directory)
        with patch.object(DEMO.os, "kill") as signal, patch.object(DEMO, "execute", return_value=f"{os.getuid()} /another/application"):
            with self.assertRaises(DEMO.DemoError):
                demo.owned_pid()
            signal.assert_called_once_with(4321, 0)

    def test_status_only_inspects_and_never_starts_or_disables(self):
        demo = DEMO.LocalDemo(self.directory)
        with patch.object(demo, "check_docker"), patch.object(demo, "container_states", return_value={"mysql": "healthy", "redis": "healthy"}), \
                patch.object(demo, "owned_pid", return_value=1234), patch.object(demo, "cron_count", return_value=0), \
                patch.object(demo, "inert_database", return_value=True), patch.object(demo, "health", return_value=True), \
                patch.object(demo, "compose_up") as dependencies, patch.object(demo, "disable_cron") as disable, \
                patch.object(DEMO.subprocess, "Popen") as spawn:
            self.assertEqual(demo.status()["backend_pid"], 1234)
            dependencies.assert_not_called()
            disable.assert_not_called()
            spawn.assert_not_called()

    def test_repeated_start_reuses_owned_running_service_without_mutation(self):
        demo = DEMO.LocalDemo(self.directory)
        status = {"backend_pid": 1234, "disabled_cron": True, "inert_database": True, "health": True, "ready": True}
        with patch.object(demo, "check_docker"), patch.object(demo, "owned_pid", return_value=1234), \
                patch.object(demo, "status", return_value=status), patch.object(demo, "compose_up") as dependencies, \
                patch.object(demo, "build") as build, patch.object(demo, "disable_cron") as disable, \
                patch.object(DEMO.subprocess, "Popen") as spawn:
            self.assertEqual(demo.start(), status)
            self.assertEqual(demo.start(), status)
            with self.assertRaises(DEMO.DemoError):
                demo.start(seed=True)
            dependencies.assert_not_called()
            build.assert_not_called()
            disable.assert_not_called()
            spawn.assert_not_called()

    def test_missing_builtins_or_external_targets_prevent_startup(self):
        demo = DEMO.LocalDemo(self.directory)
        for values in (("2", "0"), ("3", "1"), ("3", "0")):
            with self.subTest(values=values), patch.object(demo, "mysql", side_effect=values):
                self.assertEqual(demo.inert_database(), values == ("3", "0"))

    def test_unrecorded_listener_prevents_dependency_or_process_mutation(self):
        demo = DEMO.LocalDemo(self.directory)
        with patch.object(demo, "check_docker"), patch.object(demo, "owned_pid", return_value=None), \
                patch.object(DEMO.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout="9999\n")), \
                patch.object(demo, "compose_up") as dependencies, patch.object(demo, "build") as build:
            with self.assertRaises(DEMO.DemoError):
                demo.start()
            dependencies.assert_not_called()
            build.assert_not_called()

    def test_build_does_not_follow_existing_binary_symlink(self):
        demo = DEMO.LocalDemo(self.directory)
        target = self.write("unrelated", "preserve")
        (self.directory / "backend").symlink_to(target)
        with patch.object(DEMO, "execute") as execute:
            with self.assertRaises(DEMO.DemoError):
                demo.build("backend", ".")
            execute.assert_not_called()
        self.assertEqual(target.read_text(), "preserve")

    def test_subprocess_errors_do_not_echo_secret_output(self):
        result = SimpleNamespace(returncode=1, stdout=CANARY, stderr=DSN)
        with patch.object(DEMO.subprocess, "run", return_value=result):
            with self.assertRaises(DEMO.DemoError) as failure:
                DEMO.execute(["fixture"])
            self.assertNotIn(CANARY, str(failure.exception))
            self.assertNotIn("@tcp(", str(failure.exception))


if __name__ == "__main__":
    unittest.main()
