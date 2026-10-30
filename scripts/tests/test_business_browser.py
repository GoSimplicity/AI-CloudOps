"""Offline safety/report regression; never starts Docker, browsers or HTTP."""
import copy
from contextlib import redirect_stderr
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from scripts.integration import business_browser as tool


def environment():
    return {"GITHUB_ACTIONS": "true", "GITHUB_RUN_ID": "123", "COMPOSE_PROJECT_NAME": "cloudops-ci-123", "CLOUDOPS_CI_OWNED_STACK": "1"}


def containers():
    return [{"labels": {"com.docker.compose.project": "cloudops-ci-123", "com.docker.compose.service": service}, "running": True, "health": "healthy", "ports": {"8080/tcp": [{"HostIp": "127.0.0.1", "HostPort": "18080"}]} if service == "gateway" else {}} for service in sorted(tool.REQUIRED_SERVICES)]


def report():
    return {"errors": [], "suites": [{"suites": [{"specs": [{"title": title, "ok": True, "tests": [{"expectedStatus": "passed", "status": "expected", "results": [{"status": "passed"}]}]} for title in tool.TEST_TITLES]}]}], "stats": {"expected": 2, "skipped": 0, "unexpected": 0, "flaky": 0}}


class BusinessBrowserSafetyTests(unittest.TestCase):
    def test_missing_ci_gate_refuses_before_any_process_or_network(self):
        with patch.dict(os.environ, {}, clear=True), patch.object(tool.subprocess, "run") as run:
            with self.assertRaises(tool.BrowserGateError):
                tool.main()
            run.assert_not_called()

    def test_ci_identity_and_remote_or_preselected_target_rejected(self):
        self.assertEqual(tool.require_owned_ci(environment()), "cloudops-ci-123")
        for key, value in [("GITHUB_ACTIONS", "false"), ("GITHUB_RUN_ID", "1;echo"), ("COMPOSE_PROJECT_NAME", "other"), ("CLOUDOPS_CI_OWNED_STACK", "0"), ("DOCKER_HOST", "tcp://remote:2375"), ("DOCKER_CONTEXT", "other"), ("CLOUDOPS_WEB_URL", tool.BASE_URL), ("CLOUDOPS_TEST_ENV_FILE", "/somewhere"), ("CLOUDOPS_TEST_PASSWORD", "configured"), ("CLOUDOPS_TEST_USER", "other"), ("CLOUDOPS_PREVIEW", "true")]:
            with self.subTest(key=key), self.assertRaises(tool.BrowserGateError):
                tool.require_owned_ci({**environment(), key: value})

    def test_private_configuration_refuses_links_public_permissions_duplicates_and_external_capabilities(self):
        good = "BOOTSTRAP_ADMIN_PASSWORD=fixture-only\nBIND_ADDRESS=127.0.0.1\nHTTP_PORT=18080\nPUBLIC_ORIGIN=http://localhost:18080\nNOTIFICATION_ENABLED=false\nFEISHU_WEBHOOK_URL=\nLLM_BASE_URL=http://127.0.0.1:9/v1\n"
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "fixture.env"
            path.write_text(good); path.chmod(0o600)
            tool.validate_private_config(path)
            path.chmod(0o644)
            with self.assertRaises(tool.BrowserGateError): tool.validate_private_config(path)
            path.chmod(0o600)
            link = Path(directory) / "link"; link.symlink_to(path)
            with self.assertRaises(tool.BrowserGateError): tool.validate_private_config(link)
            for content in [good + "HTTP_PORT=18080\n", good.replace("NOTIFICATION_ENABLED=false", "NOTIFICATION_ENABLED=true"), good + "LLM_API_KEY=fixture\n", good.replace("FEISHU_WEBHOOK_URL=\n", "FEISHU_WEBHOOK_URL=https://example.invalid/\n"), good.replace("BIND_ADDRESS=127.0.0.1", "BIND_ADDRESS=0.0.0.0"), good.replace("http://127.0.0.1:9/v1", "https://example.invalid/v1")]:
                path.write_text(content)
                with self.assertRaises(tool.BrowserGateError): tool.validate_private_config(path)

    def test_container_ownership_health_and_exact_loopback_publication(self):
        tool.validate_containers(containers(), "cloudops-ci-123")
        for mutate in [lambda rows: rows.pop(), lambda rows: rows[0]["labels"].update({"com.docker.compose.project": "another"}), lambda rows: rows[0].update(running=False), lambda rows: rows[0].update(health="starting"), lambda rows: rows.append(copy.deepcopy(rows[0])), lambda rows: rows[0].update(ports={"999/tcp": [{"HostIp": "127.0.0.1", "HostPort": "19000"}]}), lambda rows: next(row for row in rows if row["labels"]["com.docker.compose.service"] == "gateway")["ports"]["8080/tcp"][0].update(HostIp="0.0.0.0")]:
            rows = containers(); mutate(rows)
            with self.assertRaises(tool.BrowserGateError): tool.validate_containers(rows, "cloudops-ci-123")

    def test_only_two_selected_passes_accepted(self):
        self.assertEqual(tool.validate_report(report()), 2)
        for mutate in [lambda data: data.update(errors=[{"message": "private"}]), lambda data: data["stats"].update(skipped=1), lambda data: data["stats"].update(unexpected=1), lambda data: data["stats"].update(flaky=1), lambda data: data["stats"].update(expected=0), lambda data: data["suites"].clear(), lambda data: data["suites"][0]["suites"][0]["specs"].pop(), lambda data: data["suites"][0]["suites"][0]["specs"][0].update(title="different")]:
            data = report(); mutate(data)
            with self.assertRaises(tool.BrowserGateError): tool.validate_report(data)

    def test_skips_expected_failures_and_success_after_retry_are_rejected(self):
        for change in [{"expectedStatus": "failed"}, {"status": "skipped"}, {"status": "flaky"}, {"results": [{"status": "failed"}, {"status": "passed"}]}, {"results": [{"status": "skipped"}]}]:
            data = report()
            data["suites"][0]["suites"][0]["specs"][0]["tests"][0].update(change)
            with self.assertRaises(tool.BrowserGateError): tool.validate_report(data)

    def test_process_failure_diagnostics_never_enter_gate_error(self):
        result = tool.subprocess.CompletedProcess([], 1, b"private output", b"private error")
        with patch.object(tool.subprocess, "run", return_value=result):
            with self.assertRaisesRegex(tool.BrowserGateError, "private output omitted") as caught:
                tool.captured(["not-executed"], environment={}, timeout=1)
            self.assertNotIn("private error", str(caught.exception))

    def test_successful_launcher_uses_fixed_owned_target_and_only_selected_tests(self):
        data = report()
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            if "compose" in command:
                return tool.subprocess.CompletedProcess(command, 0, b"a" * 64, b"")
            if "inspect" in command:
                return tool.subprocess.CompletedProcess(command, 0, "\n".join(json.dumps(row) for row in containers()).encode(), b"")
            kwargs["stdout"].write(json.dumps(data).encode())
            return tool.subprocess.CompletedProcess(command, 0)
        with patch.dict(os.environ, environment(), clear=True), patch.object(tool, "validate_private_config"), patch.object(tool.subprocess, "run", side_effect=run), patch("builtins.print"):
            self.assertEqual(tool.main(), 0)
        command, options = calls[-1]
        self.assertEqual(options["env"]["CLOUDOPS_WEB_URL"], tool.BASE_URL)
        self.assertEqual(options["env"]["CLOUDOPS_TEST_WORKORDER_FLOW_READY"], "true")
        self.assertIn("--retries=0", command)
        self.assertIn("--reporter=json", command)
        self.assertNotIn("CLOUDOPS_CI_OWNED_STACK", options["env"])
        self.assertEqual(calls[0][0][:3], ["docker", "--host", "unix:///var/run/docker.sock"])

    def test_failure_projection_never_prints_private_report_fields_or_unknown_values(self):
        data = report()
        private = "private-diagnostic-value"
        data["errors"] = [{"message": private, "stack": private}]
        first = data["suites"][0]["suites"][0]["specs"][0]
        first["tests"][0]["results"][0].update(status="failed", error={"message": private}, attachments=[{"body": private}])
        second = data["suites"][0]["suites"][0]["specs"][1]
        second["tests"][0].update(status=private)
        second["tests"][0]["results"][0]["status"] = private
        data["stats"].update(skipped=private, unexpected=True, flaky=-1)
        data["suites"][0]["suites"][0]["specs"].extend([
            {"title": private, "tests": [{"status": private}]}, {"title": [private]},
        ])
        output = "\n".join(tool.failure_diagnostics(data))
        self.assertNotIn(private, output)
        self.assertIn("status=failed outcome=expected", output)
        self.assertIn("status=unknown outcome=unknown", output)
        self.assertIn("expected=2 skipped=unknown unexpected=unknown flaky=unknown", output)
        self.assertIn("infrastructure=runner_error", output)
        self.assertEqual(len(output.splitlines()), 4)

    def test_nonzero_browser_process_still_emits_safe_status_and_counts(self):
        data = report()
        data["errors"] = [{"message": "private-runner-message", "stack": "private-stack"}]
        data["stats"].update(expected=1, unexpected=1)
        data["suites"][0]["suites"][0]["specs"][0]["tests"][0].update(status="unexpected", results=[{"status": "timedOut", "error": {"message": "private-error"}}])
        def run(command, **kwargs):
            if "compose" in command:
                return tool.subprocess.CompletedProcess(command, 0, b"a" * 64, b"")
            if "inspect" in command:
                return tool.subprocess.CompletedProcess(command, 0, "\n".join(json.dumps(row) for row in containers()).encode(), b"")
            kwargs["stdout"].write(json.dumps(data).encode())
            kwargs["stderr"].write(b"private-stderr")
            return tool.subprocess.CompletedProcess(command, 1)
        output = io.StringIO()
        with patch.dict(os.environ, environment(), clear=True), patch.object(tool, "validate_private_config"), patch.object(tool.subprocess, "run", side_effect=run), redirect_stderr(output):
            with self.assertRaises(tool.BrowserGateError):
                tool.main()
        self.assertIn("status=timedOut outcome=unexpected", output.getvalue())
        self.assertIn("expected=1 skipped=0 unexpected=1 flaky=0", output.getvalue())
        self.assertNotIn("private-", output.getvalue())


if __name__ == "__main__":
    unittest.main()
