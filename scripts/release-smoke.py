#!/usr/bin/env python3
"""Verify the release gateway without printing credentials or changing infrastructure."""

import argparse
import json
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen
from uuid import uuid4


def require(condition, message):
    # 发布门槛在 Python 优化模式下也必须执行
    if not condition:
        raise RuntimeError(message)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://localhost:18080")
    parser.add_argument("--api-only", action="store_true", help="Skip frontend HTML when testing the Go process directly")
    parser.add_argument("--crud", action="store_true", help="Create, update and delete an isolated workorder category")
    parser.add_argument("--env-file", type=Path, default=Path(__file__).resolve().parents[1] / "deploy/release/.env")
    args = parser.parse_args()
    settings = dict(line.split("=", 1) for line in args.env_file.read_text().splitlines() if line and not line.startswith("#") and "=" in line)
    base = args.url.rstrip("/")

    def request(path, data=None, token=None, method=None):
        headers = {"User-Agent": "cloudops-release-smoke/1.0"}
        if data is not None:
            headers["Content-Type"] = "application/json"
        if token:
            headers["Authorization"] = f"Bearer {token}"
        req = Request(base + path, data=json.dumps(data).encode() if data is not None else None, headers=headers, method=method)
        try:
            with urlopen(req, timeout=30) as response:
                return response.status, response.read()
        except HTTPError as error:
            return error.code, error.read()

    status, body = request("/health")
    require(status == 200 and json.loads(body).get("status") == "ok", "Backend health failed")
    print("PASS backend health")
    status, body = request("/ready")
    require(status == 200 and json.loads(body).get("status") == "ready", "Backend readiness failed")
    print("PASS database and Redis readiness")
    if not args.api_only:
        status, body = request("/")
        require(status == 200 and b"<html" in body.lower(), "Frontend HTML unavailable")
        print("PASS frontend HTML")
    for path in ("/api/user/profile", "/api/v1/assistant/ready"):
        status, _ = request(path)
        require(status == 401, f"Anonymous request not rejected: {path} ({status})")
    print("PASS anonymous API rejection")
    status, body = request("/api/user/login", {"username": "admin", "password": settings["BOOTSTRAP_ADMIN_PASSWORD"]})
    require(status == 200, "Login transport failed")
    payload = json.loads(body)
    token = payload.get("data", {}).get("accessToken")
    require(token, "Bootstrap admin login failed")
    print("PASS bootstrap admin login")
    status, body = request("/api/user/profile", token=token)
    require(status == 200 and json.loads(body).get("data"), "Authenticated profile unavailable")
    print("PASS authenticated profile")
    for path in (
        "/api/user/list", "/api/role/list", "/api/apis/list", "/api/audit/list",
        "/api/cron/job/list", "/api/workorder/category/list", "/api/workorder/form-design/list",
        "/api/workorder/process/list", "/api/workorder/template/list", "/api/workorder/instance/list",
    ):
        status, body = request(path + "?page=1&size=10", token=token)
        result = json.loads(body)
        require(status == 200 and result.get("code") == 0, f"Business list failed: {path} (HTTP {status}, code {result.get('code')})")
    print("PASS 10 authenticated business lists")
    if args.crud:
        name = f"release-smoke-{uuid4().hex[:12]}"
        category_id = None
        try:
            status, body = request("/api/workorder/category/create", {"name": name, "status": 1, "operator_id": 999999, "operator_name": "spoofed"}, token=token)
            require(status == 200 and json.loads(body).get("code") == 0, "Category create failed")
            status, body = request(f"/api/workorder/category/list?page=1&size=10&search={name}", token=token)
            data = json.loads(body)["data"]
            rows = data["items"] if "items" in data else data["list"]
            category = next(row for row in rows if row["name"] == name)
            category_id = category["id"]
            require(category["operator_name"] == "admin" and category["operator_id"] == payload["data"]["id"], "Client forged category operator")
            status, body = request(f"/api/workorder/category/update/{category_id}", {"id": 999999, "name": name, "status": 2, "description": "release integration check"}, token=token, method="PUT")
            require(status == 200 and json.loads(body).get("code") == 0, "Category update failed or body ID overrode URL")
            status, body = request(f"/api/workorder/category/detail/{category_id}", token=token)
            require(status == 200 and json.loads(body)["data"]["status"] == 2, "Category update not persisted")
            print("PASS category create/read/update and protected identity fields")
        finally:
            if category_id is not None:
                status, body = request(f"/api/workorder/category/delete/{category_id}", token=token, method="DELETE")
                require(status == 200 and json.loads(body).get("code") == 0, f"Category cleanup failed: id={category_id}")
                print("PASS category cleanup")
    status, body = request("/api/v1/assistant/ready", token=token)
    require(status not in (401, 403, 404, 502, 504), f"AI proxy/authentication failed ({status})")
    require(status in (200, 503), f"Unexpected AI readiness response ({status})")
    json.loads(body)
    print(f"PASS authenticated AI proxy (readiness HTTP {status}; capability readiness must be checked separately)")
    status, body = request("/api/user/refresh_token", {"refreshToken": payload["data"]["refreshToken"]})
    refreshed = json.loads(body)
    require(status == 200 and refreshed.get("code") == 0 and refreshed.get("data"), "Token refresh failed")
    print("PASS token refresh")
    status, body = request("/api/user/logout", {}, token=token)
    require(status == 200 and json.loads(body).get("code") == 0, "Logout failed")
    status, _ = request("/api/user/profile", token=token)
    require(status == 401, "Logged-out access token remains valid")
    status, body = request("/api/user/refresh_token", {"refreshToken": payload["data"]["refreshToken"]})
    require(status == 401 or json.loads(body).get("code") != 0, "Logged-out refresh token remains valid")
    print("PASS logout revokes access and refresh tokens")


if __name__ == "__main__":
    main()
