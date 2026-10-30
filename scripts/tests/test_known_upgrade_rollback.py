"""Offline safety boundaries for the explicit owned E04 restore fixture."""
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import tarfile
import tempfile
import unittest
from contextlib import redirect_stdout, redirect_stderr
from unittest.mock import patch

FILE = Path(__file__).resolve().parents[1] / "integration/known_upgrade_rollback.py"
spec = importlib.util.spec_from_file_location("known_upgrade_rollback", FILE)
tool = importlib.util.module_from_spec(spec)
spec.loader.exec_module(tool)


def runner():
    with patch.dict(os.environ, {}, clear=True):
        return tool.Runner("/fixture/go")


class KnownRollbackSafetyTests(unittest.TestCase):
    def test_dry_run_reads_no_credentials_source_or_docker(self):
        output = io.StringIO()
        with patch.object(tool, "Runner", side_effect=AssertionError("must not construct")), patch.object(tool.subprocess, "run", side_effect=AssertionError("must not execute")), patch.object(Path, "read_bytes", side_effect=AssertionError("must not read")), redirect_stdout(output):
            self.assertEqual(tool.main(["--go", "private-canary"]), 0)
        self.assertIn("DRY RUN", output.getvalue())
        self.assertNotIn("private-canary", output.getvalue())

    def test_every_docker_environment_override_is_rejected(self):
        for key in ("DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CONFIG", "DOCKER_CERT_PATH"):
            with self.subTest(key=key), patch.dict(os.environ, {key: "private-canary"}, clear=True), self.assertRaises(tool.RollbackError) as failure:
                tool.Runner()
            self.assertNotIn("private-canary", str(failure.exception))

    def test_helpers_are_local_named_labelled_and_have_no_anonymous_data_volume(self):
        value = runner()
        command = value.docker_command("run", "--rm", "--network", "none", "--mount", "type=volume,src=fixture,dst=/source,readonly", "redis:7.4.11-alpine", "true")
        self.assertEqual(command[:3], ["docker", "--host", "unix:///var/run/docker.sock"])
        self.assertIn("--pull=never", command)
        self.assertIn("/data:rw,noexec,nosuid", command)
        self.assertIn(f"{tool.OWNER_LABEL}={value.owner}", command)
        self.assertIn("com.docker.compose.project=" + value.project, command)
        self.assertEqual(len(value.containers), 1)
        self.assertRegex(value.project, r"^cloudops-fixture-[a-f0-9]{32}$")
        mounted = value.docker_command("run", "--network", "none", "--mount", "type=volume,src=fixture,dst=/data", "redis:7.4.11-alpine")
        self.assertNotIn("--tmpfs", mounted)
        similar = value.docker_command("run", "--network", "none", "--mount", "type=volume,src=fixture,dst=/data2", "redis:7.4.11-alpine")
        self.assertIn("--tmpfs", similar)

    def test_foreign_names_labels_network_and_pull_fail_before_docker(self):
        value = runner()
        for arguments in [("run", "--network", "host", "redis"), ("run", "--network", "none", "--name", "unrelated", "redis"), ("run", "--network", "none", "--label", f"{tool.OWNER_LABEL}=foreign", "redis"), ("run", "--network", "none", "--pull=always", "redis"), ("volume", "create", "foreign")]:
            with self.subTest(arguments=arguments), self.assertRaises(tool.RollbackError):
                value.docker_command(*arguments)

    def test_restore_volumes_keep_published_restore_labels_and_our_owner(self):
        value = runner()
        name = "cloudops-restore-" + "a" * 32 + "-mysql"
        command = value.docker_command("volume", "create", "--label", "cloudops.restore.id=" + "a" * 32, name)
        self.assertIn("cloudops.restore.id=" + "a" * 32, command)
        self.assertIn(f"{tool.OWNER_LABEL}={value.owner}", command)
        self.assertIn("com.docker.compose.project=" + value.project, command)
        self.assertIn(name, value.volumes)

    def test_cleanup_verifies_all_before_deleting_and_preserves_foreign_resources(self):
        value = runner()
        name = value.project + "-mysql"
        value.containers.add(name)
        commands = []
        def command(*arguments, **kwargs):
            commands.append(arguments)
            if arguments[0] == "ps": return b"a" * 64
            if arguments[:2] == ("inspect", "--format") and arguments[2] == "{{.Name}}": return ("/" + name).encode()
            return ("foreign|" + value.project + "|" + value.project).encode()
        value.command = command
        with self.assertRaises(tool.RollbackError): value.cleanup()
        self.assertFalse(any(arguments[0] == "rm" or arguments[:2] == ("volume", "rm") for arguments in commands))

    def test_failure_does_not_cleanup_or_echo_private_values(self):
        with tempfile.TemporaryDirectory() as directory:
            instance = runner()
            instance.directory = Path(directory)
            instance.phase = "upgrade-through-cli"
            instance.execute = lambda: (_ for _ in ()).throw(tool.RollbackError("private-canary"))
            with patch.object(tool, "Runner", return_value=instance), patch.object(instance, "cleanup") as cleanup, redirect_stderr(io.StringIO()) as output:
                self.assertEqual(tool.main(["--execute"]), 1)
                cleanup.assert_not_called()
            self.assertNotIn("private-canary", output.getvalue())
            self.assertEqual(json.loads((Path(directory) / "failure.json").read_bytes())["state"], "failed-evidence-retained")

    def test_unsafe_old_archives_or_provenance_paths_are_rejected(self):
        for name, kind in (("../outside", "file"), ("/absolute", "file"), ("link", "link")):
            content = io.BytesIO()
            with tarfile.open(fileobj=content, mode="w") as archive:
                item = tarfile.TarInfo(name)
                if kind == "link": item.type = tarfile.SYMTYPE; item.linkname = "../outside"
                else: item.size = 1
                archive.addfile(item, None if kind == "link" else io.BytesIO(b"x"))
            with tempfile.TemporaryDirectory() as directory, self.assertRaises(tool.RollbackError):
                tool.unpack_old_archive(content.getvalue(), Path(directory), {})

    def test_private_stdin_cannot_be_silently_dropped(self):
        value = runner()
        with patch.object(tool.subprocess, "run", side_effect=AssertionError("must not run")):
            for operation in ("run", "exec"):
                with self.subTest(operation=operation), self.assertRaises(tool.RollbackError):
                    value.command(operation, "fixture", data=b"private-canary")

    def test_source_provenance_is_verified_without_following_paths(self):
        content = io.BytesIO()
        data = b"module fixture\n"
        with tarfile.open(fileobj=content, mode="w") as archive:
            item = tarfile.TarInfo("go.mod"); item.size = len(data)
            archive.addfile(item, io.BytesIO(data))
        for provenance, valid in (({"go.mod": hashlib.sha256(data).hexdigest()}, True), ({"go.mod": "0" * 64}, False), ({"../private": "0" * 64}, False)):
            with tempfile.TemporaryDirectory() as directory:
                if valid: tool.unpack_old_archive(content.getvalue(), Path(directory), provenance)
                else:
                    with self.assertRaises(tool.RollbackError): tool.unpack_old_archive(content.getvalue(), Path(directory), provenance)

    def test_seed_client_explicitly_preserves_frozen_unicode_ddl(self):
        value = runner()
        captured = []
        value.owned = lambda *args: None
        value.command = lambda *args, **kwargs: captured.append(args) or b""
        value.sql("owned-fixture", "SELECT 1")
        self.assertIn("--default-character-set=utf8mb4", captured[0])

    def test_frozen_source_inventory_and_cipher_seed_are_explicit(self):
        source = json.loads((tool.ROOT / "internal/dbschema/known_a8f1700.json").read_bytes())
        statements = tool.source_statements(source)
        self.assertEqual(sum(line.startswith("CREATE TABLE") for line in statements), 49)
        self.assertTrue(all("FOREIGN_KEY_CHECKS" not in line for line in statements))
        self.assertNotIn("InitTables", tool.OLD_CHECKER)
        self.assertNotIn("AutoMigrate", tool.OLD_CHECKER)
        self.assertNotIn("NewApp", tool.OLD_CHECKER)
        self.assertIn("ReadOnly:true", tool.OLD_CHECKER)
        self.assertIn("cipher.NewGCM", tool.CIPHER_SOURCE)
        self.assertIn("base64.StdEncoding", tool.CIPHER_SOURCE)
        sql = tool.seed_sql({key: "cipher-'private" for key in ("access", "secret", "password")})
        self.assertNotIn("cipher-'private", sql)
        self.assertIn("NULL,NULL", sql)


if __name__ == "__main__":
    unittest.main()
