#!/usr/bin/env python3
"""仅在随机独立 Compose 项目中验证 Redis 重投递及任务租约崩溃恢复"""

import argparse
import json
import os
import subprocess
import tempfile
import time
import uuid
from pathlib import Path

root = Path(__file__).resolve().parents[2]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument(
    "--execute", action="store_true", help="允许创建并硬终止本工具的独立测试容器"
)
parser.add_argument(
    "--image",
    default="cloudops-backend:local",
    help="已构建且含 Python3 的后端镜像；启动前固定到本地镜像 ID",
)
parser.add_argument("--recovery-timeout", type=int, default=330)
args = parser.parse_args()
if not args.execute:
    parser.exit(2, "需要 --execute；仅应在独立开发验收机器运行\n")
if not 30 <= args.recovery_timeout <= 600:
    parser.error("recovery-timeout 必须为30至600秒")
try:
    image = subprocess.check_output(
        ["docker", "image", "inspect", "--format", "{{.Id}}", args.image],
        stderr=subprocess.DEVNULL,
        text=True,
    ).strip()
except subprocess.CalledProcessError:
    parser.exit(1, "需先构建可用的后端镜像\n")
if not image.startswith("sha256:") or len(image) != 71:
    parser.exit(1, "无法固定本地镜像 ID\n")
overlay = tempfile.TemporaryDirectory(prefix="cloudops-crash-")
override = Path(overlay.name) / "compose.json"
override.write_text(json.dumps({"services": {"backend": {"image": image}}}))
project = "cloudops-crash-" + uuid.uuid4().hex[:10]
env = dict(os.environ)
compose = [
    "docker",
    "compose",
    "-p",
    project,
    "-f",
    str(root / "deploy/release/compose.yaml"),
    "-f",
    str(override),
]
name = project + "-job"
marker = "/tmp/" + project + "-first"
runner = r"""
import json, os, sys
from urllib.request import Request, urlopen

payload = json.load(sys.stdin)
base = "http://127.0.0.1:8889"
token = None


def call(path, data=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = Request(
        base + path,
        headers=headers,
        data=json.dumps(data).encode() if data is not None else None,
    )
    try:
        with urlopen(req, timeout=12) as response:
            result = json.load(response)
    except Exception:
        raise RuntimeError("Owned acceptance API transport failed") from None
    if result.get("code") != 0:
        raise RuntimeError("Owned acceptance API operation rejected")
    return result.get("data")


token = call(
    "/api/user/login",
    {"username": "admin", "password": os.environ["BOOTSTRAP_ADMIN_PASSWORD"]},
)["accessToken"]
if payload["op"] == "prepare":
    script = (
        "if [ ! -f "
        + payload["marker"]
        + " ]; then touch "
        + payload["marker"]
        + "; sleep 180; fi; printf HA_RECOVERY_OK"
    )
    call(
        "/api/cron/job/create",
        {
            "name": payload["name"],
            "job_type": 2,
            "schedule": "@every 1h",
            "command": "/bin/sh",
            "args": ["-c", script],
            "timeout": 300,
            "max_retry": 3,
        },
    )
    result = call("/api/cron/job/list?page=1&size=100&search=" + payload["name"])
    rows = result["items"]
    owned = [row for row in rows if row["name"] == payload["name"]]
    if len(owned) != 1:
        raise RuntimeError("Owned job lookup was not unique")
    job = owned[0]
    jobid = job["id"]
    if job["status"] != 1:
        call("/api/cron/job/" + str(jobid) + "/enable", {})
    call("/api/cron/job/" + str(jobid) + "/trigger", {})
    print(json.dumps({"id": jobid}))
else:
    job = call("/api/cron/job/" + str(payload["id"]) + "/detail")
    if job["name"] != payload["name"]:
        raise RuntimeError("Owned job name mismatch")
    print(
        json.dumps(
            {
                key: job.get(key)
                for key in (
                    "status",
                    "last_run_status",
                    "last_run_output",
                    "run_count",
                    "success_count",
                )
            }
        )
    )
"""


def run(args, **kwargs):
    result = subprocess.run(args, env=env, capture_output=True, text=True, **kwargs)
    if result.returncode:
        raise RuntimeError("Isolated acceptance command failed; output withheld")
    return result.stdout


def api(container, data):
    return json.loads(
        run(
            ["docker", "exec", "-i", container, "python3", "-c", runner],
            input=json.dumps(data),
            timeout=20,
        )
    )


try:
    run(
        compose
        + [
            "up",
            "-d",
            "--no-build",
            "--wait",
            "--wait-timeout",
            "180",
            "mysql",
            "redis",
            "backend",
        ],
        timeout=210,
    )
    backend = run(compose + ["ps", "-q", "backend"]).strip()
    labels = run(
        [
            "docker",
            "inspect",
            "--format",
            '{{index .Config.Labels "com.docker.compose.project"}}',
            backend,
        ]
    ).strip()
    if labels != project:
        raise RuntimeError("Refusing to fault unrelated container")
    job = api(backend, {"op": "prepare", "name": name, "marker": marker})
    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        probe = subprocess.run(
            ["docker", "exec", backend, "test", "-f", marker], capture_output=True
        )
        if probe.returncode == 0:
            break
        time.sleep(1)
    else:
        raise RuntimeError("Owned script did not enter execution")
    before = api(backend, {"op": "state", "name": name, "id": job["id"]})
    if before["status"] != 3:
        raise RuntimeError("Owned job was not running before crash")
    print(
        "PASS real Redis worker entered owned task; injecting SIGKILL only into temporary backend",
        flush=True,
    )
    killed = time.monotonic()
    run(["docker", "kill", "--signal", "KILL", backend])
    run(["docker", "start", backend])
    deadline = time.monotonic() + args.recovery_timeout
    last = None
    while time.monotonic() < deadline:
        try:
            last = api(backend, {"op": "state", "name": name, "id": job["id"]})
            if (
                last["last_run_status"] == 1
                and last["last_run_output"] == "HA_RECOVERY_OK"
                and last["status"] == 1
            ):
                print(
                    "PASS real worker hard-crash recovery: Redis redelivery, expired DB lease, successful completion; elapsed_seconds="
                    + str(round(time.monotonic() - killed)),
                    flush=True,
                )
                break
        except Exception:
            pass
        time.sleep(3)
    else:
        print("Recovery state (owned metadata only):", json.dumps(last), flush=True)
        raise RuntimeError("Real worker recovery exceeded acceptance deadline")
finally:
    result = subprocess.run(
        compose + ["down", "--volumes", "--remove-orphans"],
        env=env,
        capture_output=True,
        text=True,
        timeout=90,
    )
    if result.returncode:
        raise RuntimeError("Temporary fault-test cleanup failed")
    overlay.cleanup()
    print(
        "Removed only temporary crash project containers, network and volumes",
        flush=True,
    )
