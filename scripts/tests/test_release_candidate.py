"""Source candidate preparation uses temporary Git repositories only; no services."""
import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import stat
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

MODULE = Path(__file__).resolve().parents[1] / "release-candidate.py"
spec = importlib.util.spec_from_file_location("release_candidate", MODULE)
tool = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = tool
spec.loader.exec_module(tool)


def blob(value):
    return hashlib.sha1(("blob " + str(len(value)) + "\0").encode() + value).hexdigest()


def repositories(parent):
    for role, (folder, repository) in tool.REPOSITORIES.items():
        root = parent / folder; root.mkdir()
        tool.git(root, ["init", "--quiet", "--initial-branch=main", "--object-format=sha1"])
        tool.git(root, ["config", "user.name", "fixture"])
        tool.git(root, ["config", "user.email", "fixture@example.invalid"])
        tool.git(root, ["config", "commit.gpgsign", "false"])
        tool.git(root, ["remote", "add", "origin", "https://github.com/" + repository + ".git"])
        (root / ".gitignore").write_text(".env\n.env.*\n")
        (root / "README.md").write_text(role + " committed source\n")
        (root / "run.sh").write_text("#!/bin/sh\nexit 0\n"); (root / "run.sh").chmod(0o755)
        if role == "web":
            for relative in tool.PUBLIC_ENV_PATHS[role]:
                target = root / relative; target.parent.mkdir(parents=True, exist_ok=True); target.write_text("VITE_PUBLIC_SETTING=public\n")
                tool.git(root, ["add", "--force", relative])
        tool.git(root, ["add", "."])
        tool.git(root, ["commit", "--quiet", "-m", "fixture source"])
        (root / ".env").write_text("PRIVATE_SETTING=ignored-private-canary\n")
    return parent / "AI-CloudOps"


class CandidatePreparationTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="candidate-test-")
        self.addCleanup(self.temporary.cleanup)
        self.parent = Path(self.temporary.name)

    def quiet_prepare(self, root, output):
        with contextlib.redirect_stdout(io.StringIO()):
            return tool.prepare(root, output)

    def test_default_plan_and_invalid_arguments_do_not_read_repositories_or_credentials(self):
        output = io.StringIO()
        with patch.object(tool, "prepare") as prepare, patch.object(tool, "collect_sources") as collect, contextlib.redirect_stdout(output), contextlib.redirect_stderr(output):
            self.assertEqual(tool.main([]), 0)
            self.assertEqual(tool.main(["--output", "private-path-canary"]), 0)
            self.assertEqual(tool.main(["--unknown=private-argument-canary"]), 1)
            self.assertEqual(tool.main(["--prepare"]), 1)
            prepare.assert_not_called(); collect.assert_not_called()
        self.assertNotIn("private-path-canary", output.getvalue())
        self.assertNotIn("private-argument-canary", output.getvalue())

    def test_prepared_manifest_and_archives_match_exact_commits_and_public_env(self):
        root = repositories(self.parent); output = self.parent / "candidate"
        result = self.quiet_prepare(root, output)
        self.assertEqual(result["status"], "prepared-not-built")
        self.assertEqual(set(result["sources"]), set(tool.REPOSITORIES))
        self.assertFalse((output / "INCOMPLETE").exists())
        self.assertEqual(result["verification"]["build"], "not-run")
        self.assertEqual(result["verification"]["tests"], "not-run")
        for role, entry in result["sources"].items():
            folder, repository = tool.REPOSITORIES[role]
            self.assertEqual(entry["repository"], repository)
            self.assertEqual(entry["revision"], tool.git(self.parent / folder, ["rev-parse", "HEAD"]).decode().strip())
            self.assertEqual(hashlib.sha256((output / entry["archive"]).read_bytes()).hexdigest(), entry["archive_sha256"])
            self.assertEqual((output / entry["source_directory"] / "README.md").read_text(), role + " committed source\n")
            self.assertFalse((output / entry["source_directory"] / ".env").exists())
            self.assertNotIn(b"ignored-private-canary", (output / entry["archive"]).read_bytes())
            self.assertEqual(stat.S_IMODE((output / entry["source_directory"] / "run.sh").stat().st_mode), 0o755)
        web = output / result["sources"]["web"]["source_directory"]
        for relative in tool.PUBLIC_ENV_PATHS["web"]:
            self.assertEqual((web / relative).read_text(), "VITE_PUBLIC_SETTING=public\n")
        for path in [output, *output.rglob("*")]:
            public_source = output / "sources" in path.parents
            if public_source:
                expected = 0o755 if path.is_dir() or path.name == "run.sh" else 0o644
            else:
                expected = 0o700 if path.is_dir() else 0o600
            self.assertEqual(stat.S_IMODE(path.stat().st_mode), expected)

    def test_dirty_tracked_staged_and_untracked_sources_refuse_before_output_creation(self):
        for kind in ("tracked", "staged", "untracked"):
            with self.subTest(kind=kind):
                parent = self.parent / kind; parent.mkdir(); root = repositories(parent)
                target = root / ("README.md" if kind != "untracked" else "untracked.txt")
                target.write_text("uncommitted work must stay intact\n")
                if kind == "staged": tool.git(root, ["add", "README.md"])
                with self.assertRaises(tool.CandidateError): self.quiet_prepare(root, parent / "output")
                self.assertFalse((parent / "output").exists())
                self.assertEqual(target.read_text(), "uncommitted work must stay intact\n")

    def test_existing_and_nested_outputs_are_never_overwritten(self):
        root = repositories(self.parent)
        existing = self.parent / "existing"; existing.mkdir(); (existing / "keep").write_text("keep")
        unrelated = self.parent / "other-repo"; unrelated.mkdir(); (unrelated / ".git").mkdir()
        link = self.parent / "source-link"; link.symlink_to(root, target_is_directory=True)
        for output in (existing, root / "nested", self.parent / "AI-CloudOps-web" / "nested", link / "nested", unrelated / "nested"):
            with self.subTest(output=output), patch.object(tool, "collect_sources") as collect:
                with self.assertRaises(tool.CandidateError): self.quiet_prepare(root, output)
                collect.assert_not_called()
        self.assertEqual((existing / "keep").read_text(), "keep")

    def test_hidden_index_changes_and_nested_source_repositories_are_refused(self):
        root = repositories(self.parent)
        tool.git(root, ["update-index", "--assume-unchanged", "README.md"])
        (root / "README.md").write_text("hidden unrelated work\n")
        self.assertEqual(tool.git(root, ["status", "--porcelain"]).strip(), b"")
        with self.assertRaises(tool.CandidateError): tool.collect_sources(root)
        self.assertEqual((root / "README.md").read_text(), "hidden unrelated work\n")
        (self.parent / ".git").mkdir()
        with self.assertRaises(tool.CandidateError): tool.collect_sources(root)

    def test_snapshot_still_archives_original_sha_after_head_changes(self):
        root = repositories(self.parent)
        selected = tool.collect_sources(root)[0]
        old = (root / "README.md").read_text()
        (root / "README.md").write_text("new committed HEAD\n")
        tool.git(root, ["commit", "--quiet", "-am", "new HEAD"])
        self.assertNotEqual(tool.git(root, ["rev-parse", "HEAD"]).decode().strip(), selected.revision)
        archive = self.parent / "selected.tar"
        digest = tool.archive_source(selected, archive, self.parent)
        destination = self.parent / "extracted"; destination.mkdir(mode=0o700)
        tool.extract_source(selected, archive, destination)
        self.assertEqual((destination / selected.folder / "README.md").read_text(), old)
        self.assertEqual(hashlib.sha256(archive.read_bytes()).hexdigest(), digest)

    def test_local_archive_attributes_do_not_rewrite_the_selected_tree(self):
        root = repositories(self.parent)
        (root / ".git/info/attributes").write_text("* export-ignore\n")
        output = self.parent / "candidate"
        self.quiet_prepare(root, output)
        self.assertTrue((output / "sources/AI-CloudOps/README.md").is_file())

    def test_committed_archive_transformations_fail_closed_and_preserve_incomplete_output(self):
        root = repositories(self.parent)
        (root / ".gitattributes").write_text("README.md export-ignore\n")
        tool.git(root, ["add", ".gitattributes"]); tool.git(root, ["commit", "--quiet", "-m", "export policy"])
        output = self.parent / "candidate"
        with self.assertRaises(tool.CandidateError): self.quiet_prepare(root, output)
        self.assertTrue((output / "INCOMPLETE").is_file())
        self.assertFalse((output / "candidate.json").exists())

    def test_tracked_personal_env_and_links_are_rejected_before_archiving(self):
        for kind in ("environment", "symlink"):
            parent = self.parent / kind; parent.mkdir(); root = repositories(parent)
            if kind == "environment":
                tool.git(root, ["add", "--force", ".env"])
            else:
                (root / "outside-link").symlink_to("/outside-source")
                tool.git(root, ["add", "outside-link"])
            tool.git(root, ["commit", "--quiet", "-m", "unsupported source"])
            with self.assertRaises(tool.CandidateError): self.quiet_prepare(root, parent / "candidate")
            self.assertFalse((parent / "candidate").exists())

    def test_origin_tokens_and_git_failures_are_never_printed_or_recorded(self):
        root = repositories(self.parent); output = self.parent / "candidate"
        token = "remote-private-canary"
        tool.git(root, ["remote", "set-url", "origin", f"https://user:{token}@github.com/GoSimplicity/AI-CloudOps.git?token={token}"])
        logged = io.StringIO()
        with patch.object(tool, "ROOT", root), contextlib.redirect_stdout(logged), contextlib.redirect_stderr(logged):
            self.assertEqual(tool.main(["--prepare", "--output", str(output)]), 0)
        self.assertNotIn(token, logged.getvalue())
        self.assertNotIn(token, (output / "candidate.json").read_text())
        for archive in (output / "archives").iterdir(): self.assertNotIn(token.encode(), archive.read_bytes())
        logged = io.StringIO()
        with patch.object(tool, "collect_sources", side_effect=tool.CandidateError(token)), contextlib.redirect_stdout(logged), contextlib.redirect_stderr(logged):
            self.assertEqual(tool.main(["--prepare", "--output", str(self.parent / "failed")]), 1)
        self.assertNotIn(token, logged.getvalue())

    def test_git_execution_helpers_are_refused_before_status_can_execute_them(self):
        root = repositories(self.parent); marker = self.parent / "must-not-run"
        tool.git(root, ["config", "filter.untrusted.process", "touch " + str(marker)])
        with self.assertRaises(tool.CandidateError): tool.collect_sources(root)
        self.assertFalse(marker.exists())

    def test_ambient_git_configuration_cannot_redirect_repository_or_lazy_fetch(self):
        with patch.dict(os.environ, {"GIT_DIR": "/unrelated", "GIT_WORK_TREE": "/unrelated", "GIT_SSH_COMMAND": "private-value", "GIT_CONFIG_COUNT": "1"}):
            result = tool.git_environment()
        for key in ("GIT_DIR", "GIT_WORK_TREE", "GIT_SSH_COMMAND", "GIT_CONFIG_COUNT"):
            self.assertNotIn(key, result)
        self.assertEqual(result["GIT_NO_LAZY_FETCH"], "1")
        self.assertEqual(result["GIT_CONFIG_GLOBAL"], os.devnull)


class ArchiveSafetyTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="candidate-tar-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = tool.Source("backend", "AI-CloudOps", "GoSimplicity/AI-CloudOps", self.root, "a" * 40, self.root,
                                  {"safe.txt": tool.Blob(blob(b"safe"), 4, False)})

    def archive(self, members):
        path = self.root / "input.tar"
        with tarfile.open(path, "w", format=tarfile.PAX_FORMAT, pax_headers={"comment": self.source.revision}) as archive:
            for name, kind, content in members:
                member = tarfile.TarInfo(name); member.mode = 0o600
                if kind == "file": member.size = len(content); archive.addfile(member, io.BytesIO(content))
                elif kind == "dir": member.type = tarfile.DIRTYPE; archive.addfile(member)
                elif kind == "symlink": member.type = tarfile.SYMTYPE; member.linkname = content.decode(); archive.addfile(member)
                elif kind == "hardlink": member.type = tarfile.LNKTYPE; member.linkname = content.decode(); archive.addfile(member)
                elif kind == "fifo": member.type = tarfile.FIFOTYPE; archive.addfile(member)
        return path

    def test_malicious_paths_links_duplicates_and_special_files_cannot_extract(self):
        cases = [
            [("AI-CloudOps/../../escape", "file", b"safe")], [("/outside", "file", b"safe")],
            [("AI-CloudOps/..\\escape", "file", b"safe")], [("AI-CloudOps/.git/config", "file", b"safe")],
            [("AI-CloudOps/.env", "file", b"safe")], [("AI-CloudOps/safe.txt", "symlink", b"../../outside")],
            [("AI-CloudOps/safe.txt", "hardlink", b"/outside")], [("AI-CloudOps/safe.txt", "fifo", b"")],
            [("AI-CloudOps/safe.txt", "file", b"safe"), ("AI-CloudOps/safe.txt", "file", b"safe")],
            [("AI-CloudOps/safe.txt", "file", b"safe"), ("AI-CloudOps/SAFE.txt", "file", b"safe")],
            [("AI-CloudOps/extra", "dir", b"")], [("AI-CloudOps/other.txt", "file", b"safe")],
        ]
        for index, members in enumerate(cases):
            destination = self.root / str(index); destination.mkdir(mode=0o700)
            with self.subTest(index=index), self.assertRaises(tool.CandidateError):
                tool.extract_source(self.source, self.archive(members), destination)
            self.assertFalse((destination / self.source.folder).exists())
            self.assertFalse((self.root / "escape").exists())

    def test_matching_names_do_not_allow_changed_blob_content(self):
        destination = self.root / "destination"; destination.mkdir(mode=0o700)
        with self.assertRaises(tool.CandidateError):
            tool.extract_source(self.source, self.archive([("AI-CloudOps/safe.txt", "file", b"evil")]), destination)

    def test_pax_path_override_and_file_budget_are_checked_before_extraction(self):
        path = self.root / "pax.tar"
        with tarfile.open(path, "w", format=tarfile.PAX_FORMAT, pax_headers={"comment": self.source.revision}) as archive:
            item = tarfile.TarInfo("AI-CloudOps/safe.txt")
            item.mode = 0o600; item.size = 4
            item.pax_headers = {"path": "AI-CloudOps/../../escape"}
            archive.addfile(item, io.BytesIO(b"safe"))
        destination = self.root / "destination"; destination.mkdir(mode=0o700)
        with self.assertRaises(tool.CandidateError): tool.extract_source(self.source, path, destination)
        self.assertFalse((destination / self.source.folder).exists())
        normal = self.archive([("AI-CloudOps/safe.txt", "file", b"safe")])
        with patch.object(tool, "MAX_FILE_BYTES", 3), self.assertRaises(tool.CandidateError):
            tool.extract_source(self.source, normal, destination)
        self.assertFalse((destination / self.source.folder).exists())


if __name__ == "__main__":
    unittest.main()
