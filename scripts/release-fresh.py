#!/usr/bin/env python3
"""Plan offline, or explicitly initialize a brand-new owned local Compose stack."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import secrets
import signal
import stat
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
OWNER_LABEL = "ai.cloudops.fresh.owner"
RUNTIME_USER = "cloudops"
MIGRATOR_USER = "cloudops_migrator"
DATABASE = "cloudops"
PROJECT_PATTERN = re.compile(r"cloudops-(?:fresh-[0-9a-f]{12,32}|ci-[0-9]{1,24})")
PHASES = (
    "validate-empty-project", "build", "plan-roles", "start-database", "preflight-metadata",
    "create-accounts", "migrator-grants", "migrate", "initialize-clock", "runtime-grants",
    "verify-runtime", "start-services",
)


class FreshError(RuntimeError):
    """Only fixed diagnostics may cross the process boundary."""


class SafeArgumentParser(argparse.ArgumentParser):
    def error(self, message: str) -> None:
        # Even an accidentally pasted secret in an unsupported argument is not
        # echoed back by argparse.
        raise FreshError("invalid-options")


def private_settings(path: Path) -> dict[str, str]:
    info = path.lstat()
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
            or stat.S_IMODE(info.st_mode) != 0o600 or info.st_size > 64 * 1024):
        raise FreshError("private-config")
    result: dict[str, str] = {}
    for line in path.read_text().splitlines():
        if not line or line.lstrip().startswith("#"):
            continue
        if "=" not in line:
            raise FreshError("private-config")
        key, value = line.split("=", 1)
        if not re.fullmatch(r"[A-Z][A-Z0-9_]*", key) or key in result:
            raise FreshError("private-config")
        result[key] = value
    for key in ("MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "MYSQL_MIGRATOR_PASSWORD"):
        if not re.fullmatch(r"[0-9a-f]{48}", result.get(key, "")):
            raise FreshError("fresh-credentials-required")
    if len({result[key] for key in ("MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "MYSQL_MIGRATOR_PASSWORD")}) != 3:
        raise FreshError("separate-credentials-required")
    if (not re.fullmatch(r"[0-9a-f]{32}", result.get("FRESH_STACK_OWNER", ""))
            or result.get("BIND_ADDRESS") != "127.0.0.1"
            or not re.fullmatch(r"[1-9][0-9]{3,4}", result.get("HTTP_PORT", ""))
            or not 1024 <= int(result["HTTP_PORT"]) <= 65535
            or result.get("PUBLIC_ORIGIN") != f'http://localhost:{result["HTTP_PORT"]}'
            or result.get("NOTIFICATION_ENABLED") != "false"
            or result.get("FEISHU_WEBHOOK_URL") or result.get("LLM_API_KEY")
            or result.get("LLM_BASE_URL") != "http://127.0.0.1:9/v1"):
        raise FreshError("fresh-local-config-required")
    expected_dsn = f'cloudops:{result["MYSQL_PASSWORD"]}@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC'
    if result.get("WEBHOOK_MYSQL_ADDR") != expected_dsn:
        raise FreshError("runtime-dsn-mismatch")
    return result


def role_statements(raw: bytes, section: str) -> list[str]:
    plan = json.loads(raw)
    if (not isinstance(plan, dict) or plan.get("version") != 1 or plan.get("database") != DATABASE
            or plan.get("section") != section or not isinstance(plan.get("statements"), list)
            or not 1 <= len(plan["statements"]) <= 256):
        raise FreshError("invalid-role-plan")
    expected_user = MIGRATOR_USER if section == "migrator" else RUNTIME_USER
    for statement in plan["statements"]:
        if (not isinstance(statement, str) or len(statement) > 2048 or ";" in statement
                or "\n" in statement or not statement.startswith("GRANT ")
                or not statement.endswith(f" TO '{expected_user}'@'%'")
                or "WITH GRANT OPTION" in statement or "ALL PRIVILEGES" in statement):
            raise FreshError("invalid-role-plan")
    return plan["statements"]


def validate_config(config: dict, settings: dict[str, str], project: str) -> tuple[str, str]:
    services = config["services"]
    mysql = services["mysql"].get("environment", {})
    if ("MYSQL_USER" in mysql or "MYSQL_PASSWORD" in mysql
            or mysql.get("MYSQL_DATABASE") != DATABASE
            or mysql.get("MYSQL_ROOT_PASSWORD") != settings["MYSQL_ROOT_PASSWORD"]):
        raise FreshError("mysql-default-all-forbidden")
    if services["schema"].get("environment"):
        raise FreshError("schema-environment-forbidden")
    aiops = services["aiops"].get("environment", {})
    if (aiops.get("LLM_API_KEY") != ""
            or aiops.get("LLM_BASE_URL") != "http://127.0.0.1:9/v1"
            or aiops.get("NOTIFICATION_ENABLED") != "false"
            or aiops.get("FEISHU_WEBHOOK") != ""):
        raise FreshError("rendered-external-capabilities-forbidden")
    expected_dsn = settings["WEBHOOK_MYSQL_ADDR"]
    if (services["backend"]["environment"].get("MYSQL_ADDR") != expected_dsn
            or services["webhook"]["environment"].get("WEBHOOK_MYSQL_ADDR") != expected_dsn):
        raise FreshError("runtime-dsn-mismatch")
    sensitive = (settings["MYSQL_ROOT_PASSWORD"], settings["MYSQL_MIGRATOR_PASSWORD"])
    for name, service in services.items():
        if name != "mysql":
            if any(secret in json.dumps(service.get("environment", {})) for secret in sensitive):
                raise FreshError("privileged-runtime-credential-forbidden")
        if service.get("labels", {}).get(OWNER_LABEL) != settings["FRESH_STACK_OWNER"]:
            raise FreshError("resource-owner-label-required")
        for port in service.get("ports", []):
            if (name != "gateway" or port.get("host_ip") != "127.0.0.1"
                    or str(port.get("published")) != settings["HTTP_PORT"] or port.get("target") != 8080):
                raise FreshError("loopback-gateway-required")
        for volume in service.get("volumes", []):
            if volume.get("type") == "volume" and volume.get("source") not in config.get("volumes", {}):
                raise FreshError("untracked-volume-forbidden")
    volumes = config.get("volumes", {})
    for volume in volumes.values():
        if volume.get("external") or volume.get("labels", {}).get(OWNER_LABEL) != settings["FRESH_STACK_OWNER"]:
            raise FreshError("external-volume-forbidden")
    networks = config.get("networks", {})
    if (set(networks) != {"default"} or networks["default"].get("external")
            or networks["default"].get("name") != project + "_default"
            or networks["default"].get("labels", {}).get(OWNER_LABEL) != settings["FRESH_STACK_OWNER"]):
        raise FreshError("owned-network-required")
    image = services["schema"].get("image", "")
    if not re.fullmatch(r"cloudops-schema:[A-Za-z0-9._-]+", image):
        raise FreshError("schema-image-required")
    return image, networks["default"]["name"]


class FreshRunner:
    def __init__(self, root: Path, project: str, socket: Path, settings: dict[str, str]):
        self.root = root
        self.project = project
        self.owner = settings["FRESH_STACK_OWNER"]
        self.settings = settings
        self.docker = ["docker", "--host", "unix://" + str(socket)]
        # Compose gives shell variables precedence over --env-file. Use only
        # process plumbing from the caller; every Compose setting must come from
        # the reviewed private file or the checked-in template's defaults.
        process_keys = {"PATH", "HOME", "USER", "LOGNAME", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "TERM"}
        self.environment = {key: value for key, value in os.environ.items() if key in process_keys}
        self.environment["COMPOSE_PROJECT_NAME"] = project
        self.compose = [*self.docker, "compose", "--project-name", project,
                        "--env-file", str(root / "deploy/release/.env"),
                        "-f", str(root / "deploy/release/compose.yaml")]
        self.phase = "validate-empty-project"
        self.image = ""
        self.network = ""
        self.mysql = ""

    def command(self, arguments: list[str], payload: bytes | None = None, timeout: int = 120) -> bytes:
        process = None
        try:
            process = subprocess.Popen(arguments, cwd=self.root, env=self.environment,
                                       stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                       start_new_session=True)
            output, _ = process.communicate(payload, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired):
            if process is not None:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                process.communicate()
            raise FreshError("command-failed-or-timed-out") from None
        if process.returncode or len(output) > 8 * 1024 * 1024:
            raise FreshError("command-failed-private-output-omitted")
        return output

    def stage(self, phase: str) -> None:
        if phase not in PHASES:
            raise FreshError("invalid-phase")
        self.phase = phase
        print("Fresh phase: " + phase, flush=True)

    def require_empty_project(self, config: dict) -> None:
        for resource, arguments in (
            ("container", ["ps", "-aq"]), ("volume", ["volume", "ls", "-q"]),
            ("network", ["network", "ls", "-q"]),
        ):
            found = self.command([*self.docker, *arguments, "--filter", "label=com.docker.compose.project=" + self.project])
            if found.strip():
                raise FreshError("existing-project-refused-use-adoption")
        # A foreign named volume/network can exist without Compose labels.
        for resource, names in (
            ("volume", [volume["name"] for volume in config.get("volumes", {}).values()]),
            ("network", [self.network]),
        ):
            existing = self.command([*self.docker, resource, "ls", "--format", "{{.Name}}"], timeout=30).decode().splitlines()
            if set(names).intersection(existing):
                raise FreshError("existing-storage-refused-use-adoption")

    def tool(self, arguments: list[str], dsn: str | None = None) -> bytes:
        command = [*self.docker, "run", "--rm", "--name", self.project + "-schema-" + secrets.token_hex(6),
                   "--label", OWNER_LABEL + "=" + self.owner,
                   "--label", "com.docker.compose.project=" + self.project,
                   "--label", "com.docker.compose.service=schema",
                   "--network", "none" if dsn is None else self.network]
        if dsn is None:
            return self.command([*command, self.image, *arguments])
        shell = 'umask 077; cat > /run/schema-secrets/connection; exec /app/schema "$@" --dsn-file /run/schema-secrets/connection'
        return self.command([*command, "--tmpfs", "/run/schema-secrets:rw,nosuid,noexec,size=1m,mode=0700",
                             "-i", "--entrypoint", "sh", self.image, "-c", shell, "schema", *arguments],
                            (dsn + "\n").encode(), timeout=240)

    def admin(self, sql: str) -> bytes:
        # Only the mysql container holds this root credential; stdin keeps SQL and
        # freshly generated account passwords out of command lines and logs.
        shell = 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql --protocol=socket --user=root --batch --raw --skip-column-names'
        return self.command([*self.docker, "exec", "-i", self.mysql, "sh", "-c", shell], sql.encode())

    def require_database_only(self) -> None:
        raw = self.command([*self.docker, "ps", "-a", "--filter", "label=com.docker.compose.project=" + self.project,
                            "--format", '{{.ID}} {{.Label "com.docker.compose.service"}} {{.Label "ai.cloudops.fresh.owner"}}'])
        seen: set[str] = set()
        for line in raw.decode().splitlines():
            fields = line.split()
            if len(fields) != 3 or fields[1] not in {"mysql", "redis"} or fields[2] != self.owner:
                raise FreshError("unexpected-writer-or-owner")
            if fields[1] in seen or not re.fullmatch(r"[0-9a-f]{12,64}", fields[0]):
                raise FreshError("ambiguous-database-service")
            seen.add(fields[1])
            if fields[1] == "mysql":
                self.mysql = fields[0]
        if seen != {"mysql", "redis"}:
            raise FreshError("missing-database-service")

    def initialize_database(self, migrator: list[str], runtime: list[str]) -> None:
        self.stage("start-database")
        self.command([*self.compose, "up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", "mysql", "redis"], timeout=240)
        self.require_database_only()
        self.stage("preflight-metadata")
        # root has full metadata visibility. Inspect every table/view and account
        # before the first CREATE USER or GRANT; never infer emptiness from a role
        # that sees only a subset of the schema.
        counts = self.admin("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='cloudops';\n"
                            "SELECT COUNT(*) FROM mysql.user WHERE User IN ('cloudops','cloudops_migrator');\n")
        if counts.split() != [b"0", b"0"]:
            raise FreshError("legacy-schema-or-role-refused-use-adoption")
        self.stage("create-accounts")
        statements = [f"CREATE USER '{user}'@'%' IDENTIFIED BY '{self.settings[key]}'"
                      for user, key in ((RUNTIME_USER, "MYSQL_PASSWORD"), (MIGRATOR_USER, "MYSQL_MIGRATOR_PASSWORD"))]
        self.admin(";\n".join(statements) + ";\n")
        self.stage("migrator-grants")
        self.admin(";\n".join(migrator) + ";\n")
        migration_dsn = f'{MIGRATOR_USER}:{self.settings["MYSQL_MIGRATOR_PASSWORD"]}@tcp(mysql:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=UTC'
        self.stage("migrate")
        self.tool(["migrate", "--confirm-new-database"], migration_dsn)
        self.require_database_only()
        self.stage("initialize-clock")
        self.tool(["init-clock", "--confirm-no-writers", "--notification-timezone", "UTC"], migration_dsn)
        self.stage("runtime-grants")
        self.admin(";\n".join(runtime) + ";\n")
        self.stage("verify-runtime")
        self.tool(["verify"], self.settings["WEBHOOK_MYSQL_ADDR"])
        self.tool(["verify-clock", "--notification-timezone", "UTC"], self.settings["WEBHOOK_MYSQL_ADDR"])

    def execute(self) -> None:
        for sibling in ("AI-CloudOps-web", "AI-CloudOps-aiops"):
            if not (self.root.parent / sibling / "Dockerfile").is_file():
                raise FreshError("missing-sibling-repository")
        config = json.loads(self.command([*self.compose, "--profile", "maintenance", "config", "--format", "json"]))
        self.image, self.network = validate_config(config, self.settings, self.project)
        self.require_empty_project(config)
        self.stage("build")
        self.command([*self.compose, "--profile", "maintenance", "build"], timeout=1800)
        # Recheck after a potentially long build, before creating any resources.
        self.require_empty_project(config)
        self.stage("plan-roles")
        plans = {}
        for section in ("migrator", "runtime"):
            raw = self.tool(["roles", "--section", section, "--format", "json", "--database", DATABASE,
                             "--runtime-role", RUNTIME_USER, "--runtime-role-host", "%",
                             "--migrator-role", MIGRATOR_USER, "--migrator-role-host", "%"])
            plans[section] = role_statements(raw, section)
        self.initialize_database(plans["migrator"], plans["runtime"])
        self.stage("start-services")
        self.command([*self.compose, "up", "-d", "--no-build", "--wait", "--wait-timeout", "300"], timeout=360)
        print("PASS fresh local stack initialized; runtime credentials only; no existing deployment modified")


def main(argv: list[str] | None = None) -> int:
    parser = SafeArgumentParser(description=__doc__)
    parser.add_argument("--execute-new-stack", action="store_true")
    parser.add_argument("--project", help="unique cloudops-fresh-<12..32 hex> or cloudops-ci-<run id>")
    parser.add_argument("--docker-socket", type=Path, default=Path("/var/run/docker.sock"))
    try:
        options = parser.parse_args(argv)
    except FreshError:
        print("FAIL fresh stack options; supplied values omitted", file=sys.stderr)
        return 2
    if not options.execute_new_stack:
        print("Offline fresh-stack plan only; credentials, Docker and databases were not read.")
        for phase in PHASES:
            print(" - " + phase)
        print("Generate a new private configuration, then explicitly pass --execute-new-stack and a unique --project.")
        print("Existing volumes/accounts/schemas are refused. Failures preserve evidence; use adoption or explicit repair.")
        return 0
    runner = None
    try:
        if not options.project or not PROJECT_PATTERN.fullmatch(options.project):
            raise FreshError("unique-project-required")
        if os.environ.get("DOCKER_HOST") or os.environ.get("DOCKER_CONTEXT"):
            raise FreshError("explicit-remote-docker-forbidden")
        socket = options.docker_socket.resolve(strict=True)
        if not stat.S_ISSOCK(socket.stat().st_mode):
            raise FreshError("local-docker-socket-required")
        settings = private_settings(ROOT / "deploy/release/.env")
        runner = FreshRunner(ROOT, options.project, socket, settings)
        runner.execute()
        return 0
    except (FreshError, OSError, ValueError, KeyError, TypeError, AttributeError):
        phase = runner.phase if runner is not None else "preflight"
        print("FAIL fresh stack at " + phase + "; private diagnostics omitted; resources preserved; automatic retry forbidden", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
