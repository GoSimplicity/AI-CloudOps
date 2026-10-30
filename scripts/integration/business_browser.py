#!/usr/bin/env python3
"""Run two selected business browser checks against this CI job's owned stack only."""
from __future__ import annotations

import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[2]
BASE_URL = "http://127.0.0.1:18080"
TEST_TITLES = (
    "cron default retry budget and responsive cancellation use real API",
    "workorder instance completes a real template draft submit and approval lifecycle",
)
REQUIRED_SERVICES = {"mysql", "redis", "backend", "aiops", "web", "gateway"}
MAX_REPORT_BYTES = 4 * 1024 * 1024


class BrowserGateError(RuntimeError):
    """Errors intentionally contain no process output or private configuration."""


def require_owned_ci(environment: dict[str, str]) -> str:
    run_id = environment.get("GITHUB_RUN_ID", "")
    project = environment.get("COMPOSE_PROJECT_NAME", "")
    if (environment.get("GITHUB_ACTIONS") != "true"
            or environment.get("CLOUDOPS_CI_OWNED_STACK") != "1"
            or not re.fullmatch(r"[0-9]+", run_id)
            or project != f"cloudops-ci-{run_id}"):
        raise BrowserGateError("Requires this GitHub job's explicitly owned isolated stack")
    if any(environment.get(name) for name in (
        "DOCKER_HOST", "DOCKER_CONTEXT", "CLOUDOPS_WEB_URL", "CLOUDOPS_TEST_ENV_FILE",
        "CLOUDOPS_TEST_PASSWORD", "CLOUDOPS_TEST_USER", "CLOUDOPS_PREVIEW",
    )):
        raise BrowserGateError("Remote Docker and preselected browser targets are forbidden")
    return project


def validate_private_config(path: Path) -> None:
    info = path.lstat()
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
            or stat.S_IMODE(info.st_mode) != 0o600 or info.st_size > 64 * 1024):
        raise BrowserGateError("Requires the generated owner-only CI configuration")
    settings: dict[str, str] = {}
    for line in path.read_text().splitlines():
        if not line or line.lstrip().startswith("#"):
            continue
        if "=" not in line:
            raise BrowserGateError("Invalid generated CI configuration")
        name, value = line.split("=", 1)
        if name in settings:
            raise BrowserGateError("Duplicate CI configuration key")
        settings[name] = value
    if (not settings.get("BOOTSTRAP_ADMIN_PASSWORD")
            or settings.get("BIND_ADDRESS") != "127.0.0.1"
            or settings.get("HTTP_PORT") != "18080"
            or settings.get("PUBLIC_ORIGIN") != "http://localhost:18080"
            or settings.get("NOTIFICATION_ENABLED") != "false"
            or settings.get("FEISHU_WEBHOOK_URL")
            or settings.get("LLM_API_KEY")
            or settings.get("LLM_BASE_URL") != "http://127.0.0.1:9/v1"):
        raise BrowserGateError("CI target or external capability configuration is unsafe")


def validate_containers(containers: list[dict], project: str) -> None:
    services: set[str] = set()
    for container in containers:
        labels = container.get("labels") or {}
        service = labels.get("com.docker.compose.service")
        if (labels.get("com.docker.compose.project") != project
                or not isinstance(service, str) or service in services
                or container.get("running") is not True):
            raise BrowserGateError("Compose container ownership or state mismatch")
        services.add(service)
        if service in REQUIRED_SERVICES and container.get("health") != "healthy":
            raise BrowserGateError("Required CI service is not healthy")
        ports = container.get("ports") or {}
        published = {key: value for key, value in ports.items() if value}
        if service == "gateway":
            if published != {"8080/tcp": [{"HostIp": "127.0.0.1", "HostPort": "18080"}]}:
                raise BrowserGateError("Gateway must publish only the fixed loopback CI port")
        elif published:
            raise BrowserGateError("Unexpected published service port")
    if not REQUIRED_SERVICES.issubset(services):
        raise BrowserGateError("Missing owned CI services")


def validate_report(report: dict) -> int:
    if report.get("errors"):
        raise BrowserGateError("Browser runner reported infrastructure errors")
    found: list[str] = []
    def visit(suite: dict) -> None:
        for spec in suite.get("specs", []):
            found.append(spec.get("title", ""))
            tests = spec.get("tests", [])
            if (spec.get("ok") is not True or len(tests) != 1
                    or tests[0].get("expectedStatus") != "passed"
                    or tests[0].get("status") != "expected"
                    or len(tests[0].get("results", [])) != 1
                    or tests[0]["results"][0].get("status") != "passed"):
                raise BrowserGateError("Every selected browser test must pass without skips or retries")
        for child in suite.get("suites", []):
            visit(child)
    for suite in report.get("suites", []):
        visit(suite)
    statistics = report.get("stats", {})
    if (sorted(found) != sorted(TEST_TITLES) or statistics.get("expected") != len(TEST_TITLES)
            or any(statistics.get(key) != 0 for key in ("skipped", "unexpected", "flaky"))):
        raise BrowserGateError("Browser report omitted, skipped or added a selected business test")
    return len(found)


def failure_diagnostics(report: object) -> list[str]:
    """Project untrusted report fields onto fixed titles, states and integer counts."""
    known_results = {"passed", "failed", "timedOut", "skipped", "interrupted"}
    known_outcomes = {"expected", "unexpected", "flaky", "skipped"}
    selected: dict[str, list[dict]] = {title: [] for title in TEST_TITLES}

    def visit(suite: object, depth: int = 0) -> None:
        if not isinstance(suite, dict) or depth > 32:
            return
        specs = suite.get("specs", [])
        if isinstance(specs, list):
            for spec in specs:
                if isinstance(spec, dict) and isinstance(spec.get("title"), str) and spec["title"] in selected:
                    selected[spec["title"]].append(spec)
        children = suite.get("suites", [])
        if isinstance(children, list):
            for child in children:
                visit(child, depth + 1)

    if isinstance(report, dict):
        suites = report.get("suites", [])
        if isinstance(suites, list):
            for suite in suites:
                visit(suite)
    lines: list[str] = []
    for title in TEST_TITLES:
        status = outcome = "unknown"
        specs = selected[title]
        if len(specs) == 1:
            tests = specs[0].get("tests", [])
            if isinstance(tests, list) and len(tests) == 1 and isinstance(tests[0], dict):
                test = tests[0]
                if isinstance(test.get("status"), str) and test["status"] in known_outcomes:
                    outcome = test["status"]
                results = test.get("results", [])
                if isinstance(results, list) and len(results) == 1 and isinstance(results[0], dict):
                    candidate = results[0].get("status")
                    if isinstance(candidate, str) and candidate in known_results:
                        status = candidate
        # title comes from the fixed catalogue, never from an error or arbitrary spec.
        lines.append(f"BROWSER {title}: status={status} outcome={outcome}")
    stats = report.get("stats") if isinstance(report, dict) else None
    counts = []
    for key in ("expected", "skipped", "unexpected", "flaky"):
        value = stats.get(key) if isinstance(stats, dict) else None
        count = str(value) if type(value) is int and 0 <= value <= 1_000_000 else "unknown"
        counts.append(f"{key}={count}")
    lines.append("BROWSER stats " + " ".join(counts))
    if isinstance(report, dict) and report.get("errors"):
        lines.append("BROWSER infrastructure=runner_error")
    return lines


def captured(command: list[str], *, environment: dict[str, str], timeout: int) -> str:
    result = subprocess.run(command, cwd=ROOT, env=environment, capture_output=True, timeout=timeout)
    if result.returncode or len(result.stdout) > MAX_REPORT_BYTES:
        raise BrowserGateError("Owned stack inspection failed; private output omitted")
    return result.stdout.decode("utf-8")


def main() -> int:
    environment = dict(os.environ)
    project = require_owned_ci(environment)
    private_env = ROOT / "deploy/release/.env"
    validate_private_config(private_env)
    # Never use a remote daemon/context, even if the local Docker CLI selected one.
    docker = ["docker", "--host", "unix:///var/run/docker.sock"]
    ids = captured([*docker, "compose", "-f", "deploy/release/compose.yaml", "ps", "-q"], environment=environment, timeout=30).split()
    if not ids or not all(re.fullmatch(r"[0-9a-f]{12,64}", item) for item in ids):
        raise BrowserGateError("No owned running containers found")
    # A selected projection deliberately excludes container environment and credentials.
    projection = '{"labels":{{json .Config.Labels}},"running":{{json .State.Running}},"health":{{if .State.Health}}{{json .State.Health.Status}}{{else}}""{{end}},"ports":{{json .NetworkSettings.Ports}}}'
    containers = [json.loads(line) for line in captured([*docker, "inspect", "--format", projection, *ids], environment=environment, timeout=30).splitlines()]
    validate_containers(containers, project)
    web = ROOT.parent / "AI-CloudOps-web"
    child_env = {key: value for key, value in environment.items() if not key.startswith(("CLOUDOPS_", "PLAYWRIGHT_"))}
    child_env.update({
        "CI": "true", "CLOUDOPS_WEB_URL": BASE_URL,
        "CLOUDOPS_TEST_ENV_FILE": str(private_env), "CLOUDOPS_ROUTER_HISTORY": "hash",
        "CLOUDOPS_TEST_WORKORDER_FLOW_READY": "true", "PLAYWRIGHT_CHANNEL": "chromium",
    })
    pattern = r"(?:^|\s)(?:" + "|".join(re.escape(title) for title in TEST_TITLES) + ")$"
    with tempfile.TemporaryDirectory(prefix="cloudops-business-browser-") as directory:
        os.chmod(directory, 0o700)
        report_path = Path(directory) / "report.json"
        error_path = Path(directory) / "stderr.log"
        with report_path.open("wb") as report, error_path.open("wb") as errors:
            os.chmod(report_path, 0o600)
            os.chmod(error_path, 0o600)
            try:
                result = subprocess.run(
                    ["pnpm", "exec", "playwright", "test", "tests/e2e/console.spec.ts", "--grep", pattern,
                     "--reporter=json", "--retries=0", "--workers=1"],
                    cwd=web, env=child_env, stdout=report, stderr=errors, timeout=480,
                )
            except subprocess.TimeoutExpired:
                print("BROWSER infrastructure=runner_timeout", file=sys.stderr)
                raise BrowserGateError("Browser runner timed out") from None
            except OSError:
                print("BROWSER infrastructure=runner_start_failed", file=sys.stderr)
                raise BrowserGateError("Browser runner could not start") from None
        if report_path.stat().st_size > MAX_REPORT_BYTES:
            print("BROWSER infrastructure=report_oversize", file=sys.stderr)
            raise BrowserGateError("Browser report exceeded its bound")
        try:
            report_data = json.loads(report_path.read_text())
        except (ValueError, OSError):
            print("BROWSER infrastructure=invalid_report", file=sys.stderr)
            raise BrowserGateError("Browser report unavailable") from None
        try:
            if result.returncode:
                raise BrowserGateError("Browser runner failed")
            count = validate_report(report_data)
        except (BrowserGateError, TypeError, KeyError, AttributeError, RecursionError):
            for line in failure_diagnostics(report_data):
                print(line, file=sys.stderr)
            raise BrowserGateError("Business browser checks failed; private output omitted") from None
    print(f"PASS {count} owned CI browser business checks; zero skipped, failed or retried tests")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (BrowserGateError, OSError, ValueError, TypeError, KeyError, AttributeError, RecursionError, subprocess.SubprocessError):
        print("FAIL business browser gate; private diagnostics omitted", file=sys.stderr)
        sys.exit(1)
