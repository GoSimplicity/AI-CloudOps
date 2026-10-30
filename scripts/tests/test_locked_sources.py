"""Source-lock contract tests: local files only, no GitHub requests."""
import contextlib
import copy
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from scripts.integration import locked_sources as tool


def manifest():
    return {"schema_version": 1, **{role: {"repository": repository, "revision": "a" * 40}
                                  for role, repository in tool.REPOSITORIES.items()}}


class LockedSourcesTests(unittest.TestCase):
    def test_committed_lock_contains_only_known_repositories_and_full_hashes(self):
        revisions = tool.load_revisions(tool.LOCK)
        self.assertEqual(set(revisions), {"web", "aiops"})
        self.assertTrue(all(len(value) == 40 for value in revisions.values()))

    def test_branches_short_hashes_unknown_repositories_and_format_drift_are_rejected(self):
        mutations = [lambda data: data.update(schema_version=True),
                     lambda data: data.update(schema_version=2), lambda data: data.pop("web"),
                     lambda data: data.update(extra={}),
                     lambda data: data["web"].update(repository="other/private"),
                     lambda data: data["web"].update(extra="unexpected")]
        mutations += [lambda data, revision=revision: data["web"].update(revision=revision)
                      for revision in ("main", "a" * 12, "A" * 40, "a" * 39 + "\n", None, 1)]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "lock.json"
            for mutation in mutations:
                value = copy.deepcopy(manifest()); mutation(value)
                path.write_text(json.dumps(value))
                with self.assertRaises(tool.SourceLockError): tool.load_revisions(path)
            path.write_text('{"schema_version":1,"schema_version":2}')
            with self.assertRaises(tool.SourceLockError): tool.load_revisions(path)
            path.write_bytes(b" " * (16 * 1024 + 1))
            with self.assertRaises(tool.SourceLockError): tool.load_revisions(path)

    def test_github_output_is_exact_and_refuses_non_actions_environment(self):
        with tempfile.TemporaryDirectory() as directory:
            destination = Path(directory) / "output"
            with patch.object(tool, "load_revisions", return_value={"web": "a" * 40, "aiops": "b" * 40}), patch.dict(os.environ, {"GITHUB_ACTIONS": "true", "GITHUB_OUTPUT": str(destination)}, clear=True):
                self.assertEqual(tool.main(["--github-output"]), 0)
            self.assertEqual(destination.read_text(), "web=" + "a" * 40 + "\naiops=" + "b" * 40 + "\n")
            with patch.dict(os.environ, {}, clear=True), contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(tool.main(["--github-output"]), 1)

    def test_workflow_checks_out_secondary_repositories_by_locked_hash(self):
        workflow = (tool.ROOT / ".github/workflows/integration.yml").read_text()
        self.assertIn("python3 scripts/integration/locked_sources.py --github-output", workflow)
        for role in tool.REPOSITORIES:
            self.assertIn("ref: ${{ steps.sources.outputs." + role + " }}", workflow)

    def test_invalid_arguments_and_raw_manifest_values_are_not_echoed(self):
        diagnostic = io.StringIO()
        with contextlib.redirect_stderr(diagnostic):
            self.assertEqual(tool.main(["private-input-canary"]), 2)
            with patch.object(tool, "load_revisions", side_effect=ValueError("private-input-canary")):
                self.assertEqual(tool.main([]), 1)
        self.assertNotIn("private-input-canary", diagnostic.getvalue())
