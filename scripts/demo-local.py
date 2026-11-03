#!/usr/bin/env python3
"""Inspect or resume an already prepared, private local demonstration environment."""

import argparse
import fcntl
import ipaddress
import json
import os
from pathlib import Path
import re
import shlex
import stat
import subprocess
import sys
import time
import urllib.request


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_RUNTIME = Path("/tmp/cloudops-demo-20261103")
PROJECT = "cloudops-demo-20261103"
MYSQL_CONTAINER = PROJECT + "-mysql-1"
REDIS_CONTAINER = PROJECT + "-redis-1"
GO_ROOT = Path.home() / "go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.darwin-arm64"


class DemoError(Exception):
    """Messages are static and never contain subprocess output or configuration."""


def execute(arguments, *, environment=None, content=None, timeout=30):
    result = subprocess.run(arguments, env=environment, input=content,
                            capture_output=True, text=True, timeout=timeout)
    if result.returncode:
        raise DemoError("Local demo command failed; command output was withheld.")
    return result.stdout


def private_text(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600:
        raise DemoError("Prepared credential files must be owned regular files with mode 0600.")
    if info.st_size > 65536:
        raise DemoError("Prepared configuration exceeds the allowed size.")
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(descriptor) as source:
        opened = os.fstat(source.fileno())
        if (opened.st_dev, opened.st_ino) != (info.st_dev, info.st_ino):
            raise DemoError("Prepared configuration changed while being read.")
        return source.read()


def read_environment(path):
    result = {}
    for line in private_text(path).splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        name, separator, raw = line.partition("=")
        if not separator or not re.fullmatch(r"[A-Z][A-Z0-9_]*", name) or name in result:
            raise DemoError("Prepared environment has an invalid or duplicate entry.")
        try:
            parts = shlex.split(raw, comments=False)
        except ValueError:
            raise DemoError("Prepared environment has invalid quoting.") from None
        if len(parts) > 1:
            raise DemoError("Prepared environment values must be quoted when necessary.")
        result[name] = parts[0] if parts else ""
    return result


def demo_database(dsn):
    match = re.fullmatch(r"cloudops_demo:[^@\s]+@tcp\(([^)]+)\)/(cloudops_demo_[A-Za-z0-9_]+)\?charset=utf8mb4&parseTime=True&loc=UTC", dsn)
    if not match:
        raise DemoError("Only the prepared loopback demo database DSN is supported.")
    address = match[1]
    # This prepared Compose overlay publishes MySQL only at this literal address.
    if address != "127.0.0.1:13306" or not ipaddress.ip_address(address.split(":")[0]).is_loopback:
        raise DemoError("The demo database must use the prepared literal loopback port.")
    return match[2]


class LocalDemo:
    def __init__(self, runtime):
        info = runtime.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
            raise DemoError("The prepared runtime directory must be owned and mode 0700.")
        self.runtime = runtime.resolve()
        self.settings = read_environment(self.runtime / "runtime.env")
        self.compose_settings = read_environment(self.runtime / "compose.env")
        self.database = demo_database(self.settings.get("MYSQL_ADDR", ""))
        if private_text(self.runtime / "schema.dsn").strip() != self.settings["MYSQL_ADDR"]:
            raise DemoError("Prepared schema and runtime database configurations differ.")
        required = {"ENV": "development", "SERVER_PORT": "8889", "MOCK_ENABLED": "false",
                    "CRON_ALLOW_LOCAL_EXEC": "false", "NOTIFICATION_EMAIL_ENABLED": "false",
                    "NOTIFICATION_FEISHU_ENABLED": "false", "PROMETHEUS_ENABLE_ALERT": "0",
                    "PROMETHEUS_ENABLE_RECORD": "0", "REDIS_ADDR": "127.0.0.1:16379",
                    "AIOPS_BASE_URL": "http://127.0.0.1:18081"}
        if any(self.settings.get(key) != value for key, value in required.items()):
            raise DemoError("Prepared runtime does not satisfy the local demo safety settings.")
        if not self.compose_settings.get("MYSQL_ROOT_PASSWORD") or not self.settings.get("REDIS_PASSWORD"):
            raise DemoError("Prepared local database credentials are missing.")
        if self.settings["REDIS_PASSWORD"] != self.compose_settings.get("REDIS_PASSWORD"):
            raise DemoError("Prepared Redis credentials differ.")

    def check_docker(self):
        endpoint = os.environ.get("DOCKER_HOST")
        # A named context can override DOCKER_HOST. Reject every remote selection
        # before any command which could contact a daemon.
        context = os.environ.get("DOCKER_CONTEXT")
        if context or not endpoint:
            args = ["docker", "context", "inspect"] + ([context] if context else [])
            contexts = json.loads(execute(args))
            endpoint = contexts[0]["Endpoints"]["docker"]["Host"]
        if not endpoint or not endpoint.startswith("unix:///"):
            raise DemoError("Docker must use a local Unix socket; remote daemons are refused.")
        if not stat.S_ISSOCK(Path(endpoint[7:]).stat().st_mode):
            raise DemoError("The selected local Docker socket is unavailable.")

    def container_states(self):
        states = {}
        for service, name in (("mysql", MYSQL_CONTAINER), ("redis", REDIS_CONTAINER)):
            info = json.loads(execute(["docker", "inspect", name]))[0]
            labels = info["Config"].get("Labels", {})
            if labels.get("com.docker.compose.project") != PROJECT or labels.get("com.docker.compose.service") != service:
                raise DemoError("A demo dependency container has unexpected ownership labels.")
            bindings = info["HostConfig"].get("PortBindings", {})
            port, published = ("3306/tcp", "13306") if service == "mysql" else ("6379/tcp", "16379")
            if bindings.get(port) != [{"HostIp": "127.0.0.1", "HostPort": published}]:
                raise DemoError("Demo dependencies must publish only the prepared loopback ports.")
            state = info["State"]
            states[service] = state.get("Health", {}).get("Status", state["Status"]) if state["Running"] else state["Status"]
        return states

    def owned_pid(self):
        file = self.runtime / "backend.pid"
        if not file.exists():
            return None
        raw = private_text(file).strip()
        if not re.fullmatch(r"[1-9][0-9]*", raw):
            raise DemoError("Prepared backend PID is invalid.")
        pid = int(raw)
        try:
            os.kill(pid, 0)
        except ProcessLookupError:
            return None
        record = execute(["ps", "-p", str(pid), "-o", "uid=,comm="]).strip().split(maxsplit=1)
        if len(record) != 2 or record[0] != str(os.getuid()) or Path(record[1]).resolve() != self.runtime / "backend":
            raise DemoError("The recorded PID belongs to another process; it will not be touched.")
        cwd = execute(["lsof", "-a", "-p", str(pid), "-d", "cwd", "-Fn"])
        if not any(line.startswith("n") and Path(line[1:]).resolve() == self.runtime for line in cwd.splitlines()):
            raise DemoError("The backend process has an unexpected working directory.")
        listeners = execute(["lsof", "-a", "-p", str(pid), "-iTCP:8889", "-sTCP:LISTEN", "-t"])
        if listeners.strip() != str(pid):
            raise DemoError("The owned backend does not own the expected listening port.")
        return pid

    def mysql(self, sql):
        environment = dict(os.environ, MYSQL_PWD=self.compose_settings["MYSQL_ROOT_PASSWORD"])
        return execute(["docker", "exec", "-i", "-e", "MYSQL_PWD", MYSQL_CONTAINER,
                        "mysql", "-uroot", "--batch", "--skip-column-names", self.database],
                       environment=environment, content=sql, timeout=15)

    def cron_count(self):
        return int(self.mysql("SELECT COUNT(*) FROM cl_cron_jobs WHERE status <> 2;").strip())

    def inert_database(self):
        builtins = self.mysql("SELECT COUNT(DISTINCT command) FROM cl_cron_jobs WHERE is_built_in=1 AND job_type=1 AND command IN ('on_duty_history','k8s_status_check','prometheus_config_refresh');").strip()
        targets = self.mysql("SELECT (SELECT COUNT(*) FROM cl_k8s_clusters WHERE COALESCE(TRIM(kube_config_content),'') <> '') + (SELECT COUNT(*) FROM cl_monitor_scrape_pools WHERE COALESCE(TRIM(prometheus_instances),'') NOT IN ('','[]','null'));").strip()
        return builtins == "3" and targets == "0"

    def disable_cron(self):
        self.mysql("UPDATE cl_cron_jobs SET status=2 WHERE status <> 2;")
        if self.cron_count() != 0:
            raise DemoError("Demo Cron jobs were not disabled; backend startup was refused.")
        if not self.inert_database():
            raise DemoError("Prepared demo requires all builtin rows and no K8s or Prometheus targets.")

    @staticmethod
    def health(path):
        try:
            # Never inherit an HTTP proxy or follow redirects to another service.
            class NoRedirect(urllib.request.HTTPRedirectHandler):
                def redirect_request(self, request, fp, code, message, headers, url):
                    return None
            client = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
            with client.open("http://127.0.0.1:8889/" + path, timeout=3) as response:
                body = json.load(response)
                expected = "ok" if path == "health" else "ready"
                return response.status == 200 and body.get("status") == expected
        except (OSError, ValueError):
            return False

    def status(self):
        self.check_docker()
        dependencies = self.container_states()
        pid = self.owned_pid()
        return {"mode": "prepared-resume", "backend_pid": pid, "dependencies": dependencies,
                "database": self.database, "disabled_cron": self.cron_count() == 0,
                "inert_database": self.inert_database(),
                "health": bool(pid) and self.health("health"),
                "ready": bool(pid) and self.health("ready"), "credentials_printed": False}

    def compose_up(self):
        execute(["docker", "compose", "--project-name", PROJECT, "--env-file", str(self.runtime / "compose.env"),
                 "-f", str(ROOT / "deploy/release/compose.yaml"), "-f", str(ROOT / "deploy/release/local-ports.yaml"),
                 "up", "-d", "--pull", "never", "--no-build", "mysql", "redis"], timeout=60)
        for _ in range(30):
            if all(value == "healthy" for value in self.container_states().values()):
                return
            time.sleep(1)
        raise DemoError("Local dependencies did not become healthy.")

    def build(self, target, package):
        destination = self.runtime / target
        if destination.is_symlink() or (destination.exists() and
                (not destination.is_file() or destination.stat().st_uid != os.getuid())):
            raise DemoError("A prepared binary has unexpected ownership or file type.")
        environment = {key: os.environ[key] for key in ("PATH", "HOME", "TMPDIR", "USER") if key in os.environ}
        environment.update(GOROOT=str(GO_ROOT), GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off",
                           DEVELOPER_DIR="/Library/Developer/CommandLineTools")
        execute([str(GO_ROOT / "bin/go"), "-C", str(ROOT), "build", "-o", str(self.runtime / target), package],
                environment=environment, timeout=300)

    def seed(self):
        owner = json.loads(private_text(self.runtime / "owner.json"))
        uid = owner.get("uid")
        if type(uid) is not int or uid <= 0 or owner.get("username") != "admin":
            raise DemoError("Prepared seed owner is invalid.")
        count = self.mysql(f"SELECT COUNT(*) FROM cl_system_users WHERE id={uid} AND username='admin';").strip()
        if count != "1":
            raise DemoError("Prepared seed owner does not match this demo database.")
        self.build("demo-seed", "./cmd/demo-seed")
        environment = {"ENV": "development", "CLOUDOPS_DEMO_DSN": self.settings["MYSQL_ADDR"]}
        execute([str(self.runtime / "demo-seed"), "--apply", "--database", self.database, "--owner-id", str(uid)],
                environment=environment, timeout=60)
        self.disable_cron()

    def start(self, seed=False):
        self.check_docker()
        if self.owned_pid():
            result = self.status()
            if not result["disabled_cron"] or not result["inert_database"] or not result["health"] or not result["ready"]:
                raise DemoError("Running demo requires inspection; it was not restarted or changed.")
            if seed:
                raise DemoError("Stop the owned backend separately before requesting seed; live data was not changed.")
            return result
        # Never stop, replace, or attach to an unrecorded listener.
        listener = subprocess.run(["lsof", "-nP", "-iTCP:8889", "-sTCP:LISTEN", "-t"], capture_output=True, text=True)
        if listener.returncode not in (0, 1) or listener.stdout.strip():
            raise DemoError("Backend port is occupied or cannot be inspected; no process was changed.")
        self.compose_up()
        self.build("schema", "./cmd/schema")
        for command in (("verify",), ("verify-clock", "--notification-timezone", "UTC")):
            execute([str(self.runtime / "schema"), *command, "--dsn-file", str(self.runtime / "schema.dsn")])
        self.disable_cron()
        if seed:
            self.seed()
        self.build("backend", ".")
        environment = {key: os.environ[key] for key in ("PATH", "HOME", "TMPDIR", "USER") if key in os.environ}
        environment.update(self.settings)
        log_path = self.runtime / "backend.log"
        descriptor = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "a") as log:
            os.fchmod(log.fileno(), 0o600)
            process = subprocess.Popen([str(self.runtime / "backend"), "--config", str(ROOT / "config/config.development.yaml")],
                                       cwd=self.runtime, env=environment, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
        pid_path = self.runtime / "backend.pid"
        descriptor = os.open(pid_path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "w") as output:
            os.fchmod(output.fileno(), 0o600)
            output.write(str(process.pid) + "\n")
        for _ in range(30):
            if process.poll() is not None:
                raise DemoError("Backend exited; inspect the private runtime log locally.")
            if self.health("ready"):
                return self.status()
            time.sleep(1)
        raise DemoError("Backend readiness timed out; inspect the owned process and private log.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("status", "start"))
    parser.add_argument("--runtime-dir", type=Path, default=DEFAULT_RUNTIME)
    parser.add_argument("--seed", action="store_true", help="seed only while the prepared backend is stopped")
    arguments = parser.parse_args()
    if arguments.seed and arguments.command != "start":
        parser.error("--seed requires start")
    try:
        demo = LocalDemo(arguments.runtime_dir)
        if arguments.command == "status":
            result = demo.status()
        else:
            descriptor = os.open(demo.runtime / "start.lock", os.O_WRONLY | os.O_CREAT | os.O_NOFOLLOW, 0o600)
            with os.fdopen(descriptor, "w") as lock:
                os.fchmod(lock.fileno(), 0o600)
                try:
                    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                except BlockingIOError:
                    raise DemoError("Another local demo start is running.") from None
                result = demo.start(arguments.seed)
        print(json.dumps(result, ensure_ascii=False))
        return 0
    except DemoError as error:
        print(str(error), file=sys.stderr)
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError):
        print("Local demo inspection or startup failed; sensitive diagnostics were withheld.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
