#!/usr/bin/env python3
"""Verify the release gateway without printing credentials or changing infrastructure."""

import argparse
import json
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://localhost:18080")
    parser.add_argument("--env-file", type=Path, default=Path(__file__).resolve().parents[1] / "deploy/release/.env")
    args = parser.parse_args()
    settings = dict(line.split("=", 1) for line in args.env_file.read_text().splitlines() if line and not line.startswith("#") and "=" in line)
    base = args.url.rstrip("/")

    def request(path, data=None, token=None):
        headers = {"User-Agent": "cloudops-release-smoke/1.0", "Content-Type": "application/json"}
        if token:
            headers["Authorization"] = f"Bearer {token}"
        req = Request(base + path, data=json.dumps(data).encode() if data is not None else None, headers=headers)
        try:
            with urlopen(req, timeout=30) as response:
                return response.status, response.read()
        except HTTPError as error:
            return error.code, error.read()

    status, body = request("/health")
    assert status == 200 and json.loads(body).get("status") == "ok", "Backend health failed"
    print("PASS backend health")
    status, body = request("/")
    assert status == 200 and b"<html" in body.lower(), "Frontend HTML unavailable"
    print("PASS frontend HTML")
    for path in ("/api/user/profile", "/api/v1/assistant/ready"):
        status, _ = request(path)
        assert status == 401, f"Anonymous request not rejected: {path} ({status})"
    print("PASS anonymous API rejection")
    status, body = request("/api/user/login", {"username": "admin", "password": settings["BOOTSTRAP_ADMIN_PASSWORD"]})
    assert status == 200, "Login transport failed"
    payload = json.loads(body)
    token = payload.get("data", {}).get("accessToken")
    assert token, "Bootstrap admin login failed"
    print("PASS bootstrap admin login")
    status, body = request("/api/user/profile", token=token)
    assert status == 200 and json.loads(body).get("data"), "Authenticated profile unavailable"
    print("PASS authenticated profile")
    status, body = request("/api/v1/assistant/ready", token=token)
    assert status not in (401, 403, 404, 502, 504), f"AI proxy/authentication failed ({status})"
    assert status in (200, 503), f"Unexpected AI readiness response ({status})"
    json.loads(body)
    print(f"PASS authenticated AI proxy (readiness HTTP {status}; capability readiness must be checked separately)")


if __name__ == "__main__":
    main()
