#!/usr/bin/env python3
"""Create local release secrets without printing them or overwriting an existing file."""

import argparse
import ipaddress
import os
from pathlib import Path
import re
import secrets
import sys
from urllib.parse import urlparse

# Runtime account is explicitly provisioned by release-fresh.py; never MYSQL_USER/ALL.
MYSQL_USER = "cloudops"
MYSQL_DATABASE = "cloudops"
DEFAULT_MYSQL_HOST = "mysql:3306"
# webhook（cmd/webhook）经 gorm.io/driver/mysql 解析 go-sql-driver DSN，参数与 backend 的 MYSQL_ADDR 相同
DSN_PARAMS = "charset=utf8mb4&parseTime=True&loc=UTC"
# 口令直接拼进 DSN 且 .env 不加引号：只接受不含 DSN 分隔符和 Compose 插值字符的值
SAFE_VALUE = re.compile(r"^[A-Za-z0-9._~-]+$")
SAFE_HOST = re.compile(r"^[A-Za-z0-9.-]+:[0-9]{1,5}$")
# 不传 --webhook-backend-domain 时写入的本机占位。生产 webhook 会拒绝回环主机。
LOOPBACK_BACKEND_DOMAIN = "http://127.0.0.1:8889/api/v1/alerts"
# 校验失败时只打这句，不回显调用方传入的原始值。
BACKEND_DOMAIN_ERROR = (
    "--webhook-backend-domain must be an http or https URL with a hostname "
    "and must not contain whitespace, $, or quotes."
)


def webhook_dsn(password, host):
    """Build WEBHOOK_MYSQL_ADDR; never print or log the result."""
    if not SAFE_VALUE.match(password):
        raise ValueError("MYSQL_PASSWORD contains characters unsafe for an unquoted DSN")
    return f"{MYSQL_USER}:{password}@tcp({host})/{MYSQL_DATABASE}?{DSN_PARAMS}"


def backend_domain_rejected(value):
    """True when value is not an http(s) URL with a hostname, or contains whitespace, $ or quotes."""
    if not value or any(character.isspace() for character in value) or "$" in value or "'" in value or '"' in value:
        return True
    try:
        parsed = urlparse(value)
        host = parsed.hostname
        # 非数字或超出范围的端口在这里抛出 ValueError，不能当成合法地址写入。
        parsed.port
    except (ValueError, UnicodeError):
        return True
    if parsed.scheme.lower() not in ("http", "https") or not host or not host.strip("."):
        return True
    return False


def host_is_loopback(value):
    """Whether the URL host is a loopback name or address. Unspecified addresses are not loopback."""
    try:
        host = urlparse(value).hostname or ""
    except (ValueError, UnicodeError):
        return False
    name = host.strip().rstrip(".").lower()
    if name == "localhost" or name.endswith(".localhost"):
        return True
    try:
        address = ipaddress.ip_address(name.split("%", 1)[0])
    except ValueError:
        return False
    return bool(address.is_loopback)


def backend_domain_header(kind, domain):
    """Comment lines above WEBHOOK_BACKEND_DOMAIN. The loopback wording stays stable when the flag is omitted."""
    if host_is_loopback(domain):
        if kind == "create":
            return (
                "# Feishu card action base (scheme and /api/v1/alerts). Loopback placeholder for local use.\n"
                "# Production, unset, and unrecognized ENV refuse to start until this host is an external address.\n"
            )
        return (
            "# External base for Feishu card actions (scheme and /api/v1/alerts).\n"
            "# Loopback placeholder for local use. Production refuses to start until this is an external address.\n"
        )
    return (
        "# Feishu card action base (scheme and /api/v1/alerts).\n"
        "# Supplied with --webhook-backend-domain. Production refuses loopback and unspecified hosts.\n"
    )


def read_settings(path):
    settings = {}
    for line in path.read_text().splitlines():
        if line and not line.lstrip().startswith("#") and "=" in line:
            name, value = line.split("=", 1)
            settings[name.strip()] = value.strip()
    return settings


def derived_additions(settings, host, backend_domain):
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
        lines.extend(backend_domain_header("add", backend_domain).splitlines())
        lines.append(f"WEBHOOK_BACKEND_DOMAIN={backend_domain}")
    if not settings.get("WEBHOOK_ACTION_SIGNING_KEY"):
        if "WEBHOOK_ACTION_SIGNING_KEY" in settings:
            return None, "WEBHOOK_ACTION_SIGNING_KEY= exists but is empty; remove that line and rerun with --add-missing."
        lines.append("# HMAC key for Feishu card action links. Generated; do not copy into git.")
        lines.append(f"WEBHOOK_ACTION_SIGNING_KEY={secrets.token_hex(32)}")
    if not settings.get("WEBHOOK_ALERTMANAGER_BEARER_TOKEN"):
        if "WEBHOOK_ALERTMANAGER_BEARER_TOKEN" in settings:
            return None, "WEBHOOK_ALERTMANAGER_BEARER_TOKEN= exists but is empty; remove that line and rerun with --add-missing."
        lines.append("# Bearer token for Alertmanager /receive. Same value is injected into Alertmanager at start.")
        lines.append(f"WEBHOOK_ALERTMANAGER_BEARER_TOKEN={secrets.token_hex(32)}")
    return lines, ""


def add_missing(destination, host, backend_domain):
    """Upgrade an existing .env: append derived keys that are absent. Never print secret values."""
    settings = read_settings(destination)
    lines, error = derived_additions(settings, host, backend_domain)
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
    if wrote_backend_domain(lines) and host_is_loopback(backend_domain):
        note_loopback_backend_placeholder()
    return 0


def wrote_backend_domain(lines):
    return any(line.startswith("WEBHOOK_BACKEND_DOMAIN=") for line in lines)


def note_loopback_backend_placeholder():
    """Local placeholder only. Do not print the address or any secret."""
    print(
        "WEBHOOK_BACKEND_DOMAIN was filled with a loopback placeholder for local use. "
        "Production, an unset ENV, and an unrecognized ENV refuse to start until it is replaced "
        "with the external card action base. No address or secret was printed.",
        file=sys.stderr,
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--env-file", type=Path,
                        default=Path(__file__).resolve().parents[1] / "deploy/release/.env",
                        help="destination .env (default: deploy/release/.env)")
    parser.add_argument("--mysql-host", default=DEFAULT_MYSQL_HOST,
                        help="host:port used in WEBHOOK_MYSQL_ADDR (default: mysql:3306, the Compose service)")
    parser.add_argument("--add-missing", action="store_true",
                        help="for an existing .env, append missing derived keys (WEBHOOK_MYSQL_ADDR, card link base, signing key) without touching other values")
    parser.add_argument("--webhook-backend-domain",
                        help="WEBHOOK_BACKEND_DOMAIN written for a new file or when --add-missing adds that key. "
                             "Must be an http or https URL with a hostname and no whitespace, $, or quotes. "
                             "Omit to write the local loopback placeholder.")
    args = parser.parse_args()
    destination = args.env_file
    if not SAFE_HOST.match(args.mysql_host):
        print("--mysql-host must look like host:port", file=sys.stderr)
        return 2
    if args.webhook_backend_domain is None:
        backend_domain = LOOPBACK_BACKEND_DOMAIN
    elif backend_domain_rejected(args.webhook_backend_domain):
        print(BACKEND_DOMAIN_ERROR, file=sys.stderr)
        return 2
    else:
        backend_domain = args.webhook_backend_domain

    if args.add_missing:
        if not destination.exists():
            print(f"{destination} does not exist; run without --add-missing first.", file=sys.stderr)
            return 1
        return add_missing(destination, args.mysql_host, backend_domain)

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
    content += backend_domain_header("create", backend_domain)
    content += f"WEBHOOK_BACKEND_DOMAIN={backend_domain}\n"
    signing_key = secrets.token_hex(32)
    bearer_token = secrets.token_hex(32)
    content += f"WEBHOOK_ACTION_SIGNING_KEY={signing_key}\n"
    content += f"WEBHOOK_ALERTMANAGER_BEARER_TOKEN={bearer_token}\n"
    try:
        descriptor = os.open(destination, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        print(f"{destination} already exists; refusing to overwrite. "
              "Use --add-missing to append only missing derived keys.", file=sys.stderr)
        return 1
    with os.fdopen(descriptor, "w") as output:
        output.write(content)
    print(f"Created {destination} with mode 0600. No secrets printed.")
    if host_is_loopback(backend_domain):
        note_loopback_backend_placeholder()
    return 0


if __name__ == "__main__":
    sys.exit(main())
