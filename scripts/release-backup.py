#!/usr/bin/env python3
"""Back up MySQL and deployment secrets without exposing them in arguments or output."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import shutil
import subprocess


def main():
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=root / "deploy/release/backups.local" / datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ"))
    args = parser.parse_args()
    os.umask(0o077)
    args.output.mkdir(parents=True, exist_ok=False, mode=0o700)
    compose = ["docker", "compose", "-f", str(root / "deploy/release/compose.yaml")]
    destination = args.output / "mysql.sql"
    try:
        with destination.open("xb") as output:
            subprocess.run(
                compose + ["exec", "-T", "mysql", "sh", "-c", 'MYSQL_PWD="$MYSQL_PASSWORD" exec mysqldump -u cloudops --single-transaction --quick --no-tablespaces cloudops'],
                stdout=output, check=True,
            )
        if destination.stat().st_size == 0:
            raise RuntimeError("Database dump is empty")
        shutil.copyfile(root / "deploy/release/.env", args.output / "deployment.env")
        revisions = {}
        for name, path in (("backend", root), ("web", root.parent / "AI-CloudOps-web"), ("aiops", root.parent / "AI-CloudOps-aiops")):
            revisions[name] = subprocess.check_output(["git", "-C", str(path), "rev-parse", "HEAD"], text=True).strip()
        (args.output / "manifest.json").write_text(json.dumps({
            "created_at": datetime.now(timezone.utc).isoformat(),
            "revisions": revisions,
            "scope": ["mysql", "deployment-secrets"],
            "excluded": ["redis-task-state", "aiops-uploaded-documents", "external-infrastructure"],
        }, indent=2) + "\n")
    except Exception:
        # Incomplete backups are never mistaken for a successful restore point.
        (args.output / "INCOMPLETE").touch()
        raise
    print(f"Database and deployment-secret backup saved privately to {args.output}")
    print("This backup excludes Redis queues and AIOps uploaded documents; preserve their volumes separately.")


if __name__ == "__main__":
    main()
