#!/usr/bin/env python3
"""Create local release secrets without printing them or overwriting an existing file."""

import argparse
import os
from pathlib import Path
import re
import secrets
import sys

# Runtime account is explicitly provisioned by release-fresh.py; never MYSQL_USER/ALL.
MYSQL_USER = "cloudops"
MYSQL_DATABASE = "cloudops"
DEFAULT_MYSQL_HOST = "mysql:3306"
# webhook（cmd/webhook）经 gorm.io/driver/mysql 解析 go-sql-driver DSN，参数与 backend 的 MYSQL_ADDR 相同
DSN_PARAMS = "charset=utf8mb4&parseTime=True&loc=UTC"
# 口令直接拼进 DSN 且 .env 不加引号：只接受不含 DSN 分隔符和 Compose 插值字符的值
SAFE_VALUE = re.compile(r"^[A-Za-z0-9._~-]+$")
SAFE_HOST = re.compile(r"^[A-Za-z0-9.-]+:[0-9]{1,5}$")


def webhook_dsn(password, host):
    """Build WEBHOOK_MYSQL_ADDR; never print or log the result."""
    if not SAFE_VALUE.match(password):
        raise ValueError("MYSQL_PASSWORD contains characters unsafe for an unquoted DSN")
    return f"{MYSQL_USER}:{password}@tcp({host})/{MYSQL_DATABASE}?{DSN_PARAMS}"


def read_settings(path):
    settings = {}
    for line in path.read_text().splitlines():
        if line and not line.lstrip().startswith("#") and "=" in line:
            name, value = line.split("=", 1)
            settings[name.strip()] = value.strip()
    return settings


def derived_additions(settings, host):
    """Lines to append. None means refuse and leave the file unchanged."""
    lines = []
    if "WEBHOOK_MYSQL_ADDR" not in settings:
        password = settings.get("MYSQL_PASSWORD", "")
        if not password:
            return None, "MYSQL_PASSWORD is missing or empty; cannot derive WEBHOOK_MYSQL_ADDR."
        try:
            dsn = webhook_dsn(password, host)
        except ValueError as error:
            return None, f"{error}; set WEBHOOK_MYSQL_ADDR manually."
        lines.append("# Webhook (cmd/webhook) MySQL DSN, derived from MYSQL_PASSWORD.")
        lines.append(f"WEBHOOK_MYSQL_ADDR={dsn}")
    elif not settings.get("WEBHOOK_MYSQL_ADDR"):
        return None, "WEBHOOK_MYSQL_ADDR= exists but is empty; remove that line and rerun with --add-missing."
    if "WEBHOOK_BACKEND_DOMAIN" not in settings:
        lines.append("# External base for Feishu card actions. Includes scheme and /api/v1/alerts.")
        lines.append("WEBHOOK_BACKEND_DOMAIN=http://127.0.0.1:8889/api/v1/alerts")
    if "WEBHOOK_ACTION_SIGNING_KEY" not in settings:
        lines.append("# HMAC key for Feishu card action links.")
        lines.append(f"WEBHOOK_ACTION_SIGNING_KEY={secrets.token_hex(32)}")
    if "WEBHOOK_ALERTMANAGER_BEARER_TOKEN" not in settings:
        lines.append("# Empty keeps /receive unauthenticated. Match alertmanager http_config when set.")
        lines.append("WEBHOOK_ALERTMANAGER_BEARER_TOKEN=")
    return lines, ""


def add_missing(destination, host):
    """Upgrade an existing .env: append derived keys that are absent. Never print secret values."""
    settings = read_settings(destination)
    lines, error = derived_additions(settings, host)
    if error:
        print(error, file=sys.stderr)
        return 1
    if not lines:
        print(f"{destination} already has the derived webhook settings; nothing changed. No secrets printed.")
        return 0
    raw = destination.read_bytes()
    needs_newline = bool(raw) and not raw.endswith(b"\n")
    descriptor = os.open(destination, os.O_WRONLY | os.O_APPEND)
    with os.fdopen(descriptor, "w") as output:
        output.write(("\n" if needs_newline else "") + "\n".join(lines) + "\n")
    os.chmod(destination, 0o600)
    print(f"Appended derived webhook settings to {destination} (mode 0600). No secrets printed.")
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--env-file", type=Path,
                        default=Path(__file__).resolve().parents[1] / "deploy/release/.env",
                        help="destination .env (default: deploy/release/.env)")
    parser.add_argument("--mysql-host", default=DEFAULT_MYSQL_HOST,
                        help="host:port used in WEBHOOK_MYSQL_ADDR (default: mysql:3306, the Compose service)")
    parser.add_argument("--add-missing", action="store_true",
                        help="for an existing .env, append missing derived keys (WEBHOOK_MYSQL_ADDR, card link base, signing key) without touching other values")
    args = parser.parse_args()
    destination = args.env_file
    if not SAFE_HOST.match(args.mysql_host):
        print("--mysql-host must look like host:port", file=sys.stderr)
        return 2

    if args.add_missing:
        if not destination.exists():
            print(f"{destination} does not exist; run without --add-missing first.", file=sys.stderr)
            return 1
        return add_missing(destination, args.mysql_host)

    names = (
        "MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "MYSQL_MIGRATOR_PASSWORD", "REDIS_PASSWORD", "JWT_KEY1",
        "JWT_KEY2", "TREE_PASSWORD_ENCRYPTION_KEY", "BOOTSTRAP_ADMIN_PASSWORD",
        "AIOPS_API_KEY",
    )
    # Hex avoids shell, Compose interpolation and MySQL DSN delimiters.
    values = {name: secrets.token_hex(24) for name in names}
    # Existing AES encryption requires exactly 32 bytes; preserve this key on upgrades.
    values["TREE_PASSWORD_ENCRYPTION_KEY"] = secrets.token_hex(16)
    values["FRESH_STACK_OWNER"] = secrets.token_hex(16)
    # Webhook reads only WEBHOOK_MYSQL_ADDR (never the backend MYSQL_ADDR); same account and database.
    values["WEBHOOK_MYSQL_ADDR"] = webhook_dsn(values["MYSQL_PASSWORD"], args.mysql_host)
    content = "# Keep private. Back up encryption keys with the database.\n"
    content += "".join(f"{name}={value}\n" for name, value in values.items())
    content += "PUBLIC_ORIGIN=http://localhost:18080\nHTTP_PORT=18080\n"
    content += "BIND_ADDRESS=127.0.0.1\nRELEASE_VERSION=local\n"
    content += "# Optional AIOps custom bot; keep the webhook credential private.\n"
    content += "NOTIFICATION_ENABLED=false\nFEISHU_WEBHOOK_URL=\n"
    content += "# Fresh local stacks start without an external model connection.\n"
    content += "LLM_BASE_URL=http://127.0.0.1:9/v1\n"
    content += "# Fresh webhook bot credentials are disabled and every bot destination is loopback.\n"
    content += 'WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID=\n'
    content += 'WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET=\n'
    content += 'WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API=http://127.0.0.1:9/open-apis/auth/v3/tenant_access_token/internal\n'
    content += 'WEBHOOK_IM_FEISHU_PRIVATE_MESSAGE_API=http://127.0.0.1:9/open-apis/im/v1/messages\n'
    content += 'WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API=http://127.0.0.1:9/open-apis/bot/v2/hook\n'
    content += "# Feishu card action base. Replace with the external URL (scheme + /api/v1/alerts) before exposing webhook.\n"
    content += "WEBHOOK_BACKEND_DOMAIN=http://127.0.0.1:8889/api/v1/alerts\n"
    content += f"WEBHOOK_ACTION_SIGNING_KEY={secrets.token_hex(32)}\n"
    content += "# Empty: /receive stays open and webhook logs a startup warning. Set with alertmanager http_config.\n"
    content += "WEBHOOK_ALERTMANAGER_BEARER_TOKEN=\n"
    try:
        descriptor = os.open(destination, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        print(f"{destination} already exists; refusing to overwrite. "
              "Use --add-missing to append only missing derived keys.", file=sys.stderr)
        return 1
    with os.fdopen(descriptor, "w") as output:
        output.write(content)
    print(f"Created {destination} with mode 0600. No secrets printed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
