#!/usr/bin/env python3
"""Create local release secrets without printing them or overwriting an existing file."""

import os
from pathlib import Path
import secrets


def main():
    destination = Path(__file__).resolve().parents[1] / "deploy/release/.env"
    names = (
        "MYSQL_PASSWORD", "MYSQL_ROOT_PASSWORD", "REDIS_PASSWORD", "JWT_KEY1",
        "JWT_KEY2", "TREE_PASSWORD_ENCRYPTION_KEY", "BOOTSTRAP_ADMIN_PASSWORD",
        "AIOPS_API_KEY",
    )
    # Hex avoids shell, Compose interpolation and MySQL DSN delimiters.
    values = {name: secrets.token_hex(24) for name in names}
    # Existing AES encryption requires exactly 32 bytes; preserve this key on upgrades.
    values["TREE_PASSWORD_ENCRYPTION_KEY"] = secrets.token_hex(16)
    content = "# Keep private. Back up encryption keys with the database.\n"
    content += "".join(f"{name}={value}\n" for name, value in values.items())
    content += "PUBLIC_ORIGIN=http://localhost:18080\nHTTP_PORT=18080\n"
    content += "BIND_ADDRESS=127.0.0.1\nRELEASE_VERSION=local\n"
    content += "# Optional AIOps custom bot; keep the webhook credential private.\n"
    content += "NOTIFICATION_ENABLED=true\nFEISHU_WEBHOOK_URL=\n"
    descriptor = os.open(destination, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w") as output:
        output.write(content)
    print(f"Created {destination} with mode 0600. No secrets printed.")


if __name__ == "__main__":
    main()
