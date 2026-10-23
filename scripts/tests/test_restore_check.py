"""恢复验收工具回归：只用替身与临时文件，不访问 Docker 或外部数据库"""

import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("restore_check", Path(__file__).resolve().parents[1] / "release-restore-check.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class FakeClient:
    def __init__(self, fail_restore=False, fail_create=False):
        self.fail_restore = fail_restore
        self.fail_create = fail_create
        self.cleaned = []
        self.commands = []
        self.restored = False

    def verify_container(self):
        pass

    def admin(self, sql):
        self.commands.append(sql)
        if sql.startswith("CREATE") and self.fail_create:
            raise MODULE.RestoreError("create failed")
        return b"0\n"

    def restore(self, user, database, password, dump):
        self.restored = True
        if self.fail_restore:
            raise MODULE.RestoreError("restore failed")

    def query(self, *args):
        return b""

    def cleanup(self, database, user):
        self.cleaned.append((database, user))


class RestoreCheckTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.dump = Path(self.temp.name) / "mysql.sql"
        self.dump.write_text("SELECT 1;\n")

    def run_check(self, client, *, schema=None, data=None):
        with patch.object(MODULE, "schema_fingerprint", side_effect=schema or [(46, "schema"), (46, "schema")]), patch.object(MODULE, "critical_data", side_effect=data or [("admin", "permissions"), ("admin", "permissions")]):
            return MODULE.restore_check(client, self.dump)

    def test_success_uses_unique_restricted_database_and_cleans_up(self):
        client = FakeClient()
        self.assertEqual(self.run_check(client), 46)
        database, user = client.cleaned[0]
        self.assertNotEqual(database, user)
        MODULE.identifier(database)
        MODULE.identifier(user)
        create = next(sql for sql in client.commands if sql.startswith("CREATE"))
        self.assertIn(f"ON `{database}`.*", create)
        for privilege in (" ON *.*", "SUPER", "FILE,", "GRANT OPTION", "SYSTEM_USER"):
            self.assertNotIn(privilege, create)
        self.assertNotIn("`cloudops`", create)

    def test_failures_always_clean_up_owned_resources(self):
        for client in (FakeClient(fail_restore=True), FakeClient(fail_create=True)):
            with self.subTest(client=type(client).__name__):
                with self.assertRaises(MODULE.RestoreError):
                    self.run_check(client)
                self.assertEqual(len(client.cleaned), 1)

    def test_schema_and_data_mismatch_fail_closed(self):
        for options in ({"schema": [(46, "expected"), (45, "different")]}, {"data": [("admin", "permissions"), ("changed", "permissions")]}):
            client = FakeClient()
            with self.assertRaises(MODULE.RestoreError):
                self.run_check(client, **options)
            self.assertEqual(len(client.cleaned), 1)

    def test_empty_backup_rejected_before_any_container_operation(self):
        self.dump.write_text("")
        client = FakeClient()
        with self.assertRaises(MODULE.RestoreError):
            MODULE.restore_check(client, self.dump)
        self.assertFalse(client.commands)
        self.assertFalse(client.restored)

    def test_name_collision_does_not_clean_or_change_existing_resources(self):
        client = FakeClient()
        with patch.object(client, "admin", return_value=b"1\n") as admin:
            with self.assertRaises(MODULE.RestoreError):
                self.run_check(client)
            self.assertEqual(admin.call_count, 1)
        self.assertFalse(client.cleaned)
        self.assertFalse(client.restored)

    def test_cleanup_rejects_arbitrary_identifiers(self):
        client = MODULE.DockerMySQL("cloudops-mysql-1", 10)
        with patch.object(client, "admin") as admin:
            with self.assertRaises(MODULE.RestoreError):
                client.cleanup("cloudops", "root")
            admin.assert_not_called()

    def test_restore_client_disables_local_commands_and_infile(self):
        command = MODULE.DockerMySQL("cloudops-mysql-1", 10).restricted_command("restore_" + "a" * 24, "restore_" + "b" * 24)
        self.assertEqual(command[2:4], ["--user", "mysql"])
        text = " ".join(command)
        for option in ("--skip-commands", "--local-infile=0", "--binary-mode", "--skip-named-commands"):
            self.assertIn(option, text)
        self.assertNotIn("--user=root", text)
        self.assertNotIn("MYSQL_ROOT_PASSWORD", text)

    def test_password_only_enters_stdin_and_driver_errors_are_hidden(self):
        client = MODULE.DockerMySQL("cloudops-mysql-1", 10)
        password = "private-password-test"
        with patch.object(MODULE.subprocess, "run", return_value=subprocess.CompletedProcess([], 1, b"", password.encode())) as run:
            with self.assertRaises(MODULE.RestoreError) as caught:
                client.query("restore_" + "a" * 24, "restore_" + "b" * 24, password, "SELECT 1;")
            self.assertNotIn(password, str(caught.exception))
            self.assertNotIn(password, " ".join(run.call_args.args[0]))
            self.assertTrue(run.call_args.kwargs["input"].startswith(password.encode() + b"\n"))

    def test_other_container_labels_rejected(self):
        client = MODULE.DockerMySQL("wrong", 10)
        with patch.object(client, "run", return_value=b"other mysql healthy"):
            with self.assertRaises(MODULE.RestoreError):
                client.verify_container()

    def test_cleanup_continues_after_session_race_and_separates_drop_steps(self):
        client = MODULE.DockerMySQL("cloudops-mysql-1", 10)
        user, database = "restore_" + "a" * 24, "restore_" + "b" * 24
        with patch.object(client, "admin", side_effect=[b"123\n", MODULE.RestoreError("session gone"), b"", b"", b"0\n"]) as admin:
            client.cleanup(database, user)
            statements = [call.args[0] for call in admin.call_args_list]
            self.assertTrue(any(sql.startswith("DROP DATABASE") for sql in statements))
            self.assertTrue(any(sql.startswith("DROP USER") for sql in statements))

    def test_cleanup_attempts_drop_user_even_if_database_drop_fails(self):
        client = MODULE.DockerMySQL("cloudops-mysql-1", 10)
        with patch.object(client, "admin", side_effect=[b"", MODULE.RestoreError("drop failed"), b""]) as admin:
            with self.assertRaises(MODULE.RestoreError):
                client.cleanup("restore_" + "a" * 24, "restore_" + "b" * 24)
            self.assertTrue(admin.call_args.args[0].startswith("DROP USER"))


if __name__ == "__main__":
    unittest.main()
