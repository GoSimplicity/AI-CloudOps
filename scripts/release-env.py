#!/usr/bin/env python3
"""Create local release secrets without printing them or overwriting an existing file."""

import argparse
import os
from pathlib import Path
import re
import secrets
import sys

# 与 deploy/release/compose.yaml 中 mysql 服务的 MYSQL_USER / MYSQL_DATABASE 及服务名保持一致
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


def add_missing(destination, host):
    """Upgrade an existing .env: only append derived keys that are missing or empty."""
    settings = read_settings(destination)
    current = settings.get("WEBHOOK_MYSQL_ADDR", "")
    if current:
        print(f"{destination} already has WEBHOOK_MYSQL_ADDR; nothing changed. No secrets printed.")
        return 0
    password = settings.get("MYSQL_PASSWORD", "")
    if not password:
        print("MYSQL_PASSWORD is missing or empty; cannot derive WEBHOOK_MYSQL_ADDR.", file=sys.stderr)
        return 1
    if "WEBHOOK_MYSQL_ADDR" in settings:
        # 存在空占位行时不改写原行（避免重写整个文件），提示人工删除空行后重试
        print("WEBHOOK_MYSQL_ADDR= exists but is empty; remove that line and rerun with --add-missing.", file=sys.stderr)
        return 1
    try:
        line = f"WEBHOOK_MYSQL_ADDR={webhook_dsn(password, host)}\n"
    except ValueError as error:
        print(f"{error}; set WEBHOOK_MYSQL_ADDR manually.", file=sys.stderr)
        return 1
    raw = destination.read_bytes()
    needs_newline = bool(raw) and not raw.endswith(b"\n")
    descriptor = os.open(destination, os.O_WRONLY | os.O_APPEND)
    with os.fdopen(descriptor, "w") as output:
        output.write(("\n" if needs_newline else "") + "# Webhook (cmd/webhook) MySQL DSN, derived from MYSQL_PASSWORD.\n" + line)
    os.chmod(destination, 0o600)
    print(f"Appended WEBHOOK_MYSQL_ADDR to {destination} (mode 0600). No secrets printed.")
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--env-file", type=Path,
                        default=Path(__file__).resolve().parents[1] / "deploy/release/.env",
                        help="destination .env (default: deploy/release/.env)")
    parser.add_argument("--mysql-host", default=DEFAULT_MYSQL_HOST,
                        help="host:port used in WEBHOOK_MYSQL_ADDR (default: mysql:3306, the Compose service)")
    parser.add_argument("--add-missing", action="store_true",
                        help="for an existing .env, append missing derived keys (WEBHOOK_MYSQL_ADDR) without touching other values")
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
        "MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "REDIS_PASSWORD", "JWT_KEY1",
        "JWT_KEY2", "TREE_PASSWORD_ENCRYPTION_KEY", "BOOTSTRAP_ADMIN_PASSWORD",
        "AIOPS_API_KEY",
    )
    # Hex avoids shell, Compose interpolation and MySQL DSN delimiters.
    values = {name: secrets.token_hex(24) for name in names}
    # Existing AES encryption requires exactly 32 bytes; preserve this key on upgrades.
    values["TREE_PASSWORD_ENCRYPTION_KEY"] = secrets.token_hex(16)
    # Webhook reads only WEBHOOK_MYSQL_ADDR (never the backend MYSQL_ADDR); same account and database.
    values["WEBHOOK_MYSQL_ADDR"] = webhook_dsn(values["MYSQL_PASSWORD"], args.mysql_host)
    content = "# Keep private. Back up encryption keys with the database.\n"
    content += "".join(f"{name}={value}\n" for name, value in values.items())
    content += "PUBLIC_ORIGIN=http://localhost:18080\nHTTP_PORT=18080\n"
    content += "BIND_ADDRESS=127.0.0.1\nRELEASE_VERSION=local\n"
    content += "# Optional AIOps custom bot; keep the webhook credential private.\n"
    content += "NOTIFICATION_ENABLED=true\nFEISHU_WEBHOOK_URL=\n"
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
