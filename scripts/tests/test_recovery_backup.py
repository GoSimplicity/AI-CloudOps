"""Safety regression for the offline backup tool; no Docker or external service."""
import hashlib
import hmac
import io
import json
import os
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

from scripts import recovery_backup as tool


def archive(entries):
    out = io.BytesIO()
    with tarfile.open(fileobj=out, mode="w") as tar:
        for name, kind in entries:
            item = tarfile.TarInfo(name)
            if kind == "dir":
                item.type = tarfile.DIRTYPE
                tar.addfile(item)
            elif kind == "link":
                item.type = tarfile.SYMTYPE
                item.linkname = "../../outside"
                tar.addfile(item)
            else:
                item.size = 4
                tar.addfile(item, io.BytesIO(b"data"))
    out.seek(0)
    return out


class RecoverySafetyTests(unittest.TestCase):
    def test_archive_rejects_traversal_absolute_links_and_duplicates(self):
        for entries in [[("../outside", "file")], [("/absolute", "file")], [("a", "link")], [("./a", "file"), ("a", "file")]]:
            with self.subTest(entries=entries), self.assertRaises(tool.RecoveryError):
                tool.validate_archive(archive(entries))
        tool.validate_archive(archive([(".", "dir"), ("./safe", "file")]))

    def test_private_inputs_reject_symlinks_and_public_mode(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            secret = root / "key"
            secret.write_bytes(b"x" * 32)
            secret.chmod(0o644)
            with self.assertRaises(tool.RecoveryError):
                tool.load_key(secret)
            secret.chmod(0o600)
            link = root / "link"
            link.symlink_to(secret)
            with self.assertRaises(tool.RecoveryError):
                tool.load_key(link)
            self.assertEqual(tool.load_key(secret), b"x" * 32)

    def test_default_dry_run_does_not_touch_sources_or_docker(self):
        with patch("sys.argv", ["recovery_backup.py", "backup", "--source", "/missing", "--key-file", "/missing", "--output", "/missing/new"]), patch.object(tool, "docker", side_effect=AssertionError("must not call docker")):
            tool.main()

    def test_invalid_or_foreign_fixture_is_rejected(self):
        descriptor = {"project": "cloudops-fixture-" + "a" * 32, "containers": {role: "b" * 64 for role in tool.ROLES}, "volumes": {role: "volume-" + role for role in tool.ROLES}}
        with patch.object(tool, "docker", return_value="unrelated"):
            with self.assertRaises(tool.RecoveryError):
                tool.verify_fixture(descriptor)
        descriptor["project"] = "cloudops"
        with patch.object(tool, "docker", side_effect=AssertionError("must not inspect")):
            with self.assertRaises(tool.RecoveryError):
                tool.verify_fixture(descriptor)

    def test_manifest_tampering_rejected_before_docker(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            bundle = root / "bundle"
            bundle.mkdir(mode=0o700)
            key = b"k" * 32
            (bundle / "manifest.json").write_bytes(b"{}")
            (bundle / "manifest.hmac").write_text(hmac.new(key, b"other", hashlib.sha256).hexdigest())
            for name in ("manifest.json", "manifest.hmac"):
                (bundle / name).chmod(0o600)
            with patch.object(tool, "docker", side_effect=AssertionError("must not call docker")), self.assertRaises(tool.RecoveryError):
                tool.load_manifest(bundle, key)

    def test_bundle_file_hash_and_file_mode_are_enforced(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            file = root / "mysql.tar"
            file.write_bytes(b"data")
            file.chmod(0o600)
            good = {"size": 4, "sha256": hashlib.sha256(b"data").hexdigest()}
            with tool.verified_copy(root, "mysql.tar", good) as copy:
                self.assertEqual(copy.read(), b"data")
                copy.seek(0)
                self.assertEqual(os.read(copy.fileno(), 4), b"data")
            with self.assertRaises(tool.RecoveryError):
                tool.verified_copy(root, "mysql.tar", {**good, "sha256": "0" * 64})
            file.chmod(0o644)
            with self.assertRaises(tool.RecoveryError):
                tool.verified_copy(root, "mysql.tar", good)

    def test_new_directory_never_overwrites_existing(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            existing = root / "existing"
            existing.mkdir()
            (existing / "evidence").write_text("keep")
            with self.assertRaises(tool.RecoveryError):
                tool.safe_directory(existing)
            self.assertEqual((existing / "evidence").read_text(), "keep")
            created = tool.safe_directory(root / "new")
            self.assertEqual(created.stat().st_mode & 0o777, 0o700)

    def test_source_descriptor_requires_explicit_distinct_full_ids(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "source.json"
            descriptor = {"version": 1, "project": "cloudops-fixture-" + "a" * 32,
                          "containers": {role: "b" * 64 for role in tool.ROLES},
                          "volumes": {role: "volume-" + role for role in tool.ROLES},
                          "private_config": [{"name": ".env", "path": "/fixture/env"}],
                          "revisions": {role: "c" * 40 for role in ("backend", "web", "aiops")}}
            source.write_text(json.dumps(descriptor))
            source.chmod(0o600)
            with self.assertRaises(tool.RecoveryError):
                tool.load_descriptor(source)
            descriptor["containers"] = {role: str(index) * 64 for index, role in enumerate(tool.ROLES, 1)}
            source.write_text(json.dumps(descriptor))
            self.assertEqual(tool.load_descriptor(source)["private_config"][0]["name"], ".env")

    def test_execute_requires_explicit_local_socket_and_confirmation(self):
        with patch("sys.argv", ["recovery_backup.py", "backup", "--source", "/missing", "--key-file", "/missing", "--output", "/missing/new", "--execute"]), self.assertRaises(SystemExit) as raised:
            tool.main()
        self.assertEqual(raised.exception.code, 1)


if __name__ == "__main__":
    unittest.main()
