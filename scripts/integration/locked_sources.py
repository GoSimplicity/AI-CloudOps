#!/usr/bin/env python3
"""Read exact secondary repository revisions; never resolve branches over a network."""
from __future__ import annotations

import json
import os
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[2]
LOCK = ROOT / "docs/release/integration-sources.json"
REPOSITORIES = {
    "web": "GoSimplicity/AI-CloudOps-web",
    "aiops": "GoSimplicity/AI-CloudOps-aiops",
}


class SourceLockError(ValueError):
    pass


def unique_object(pairs: list[tuple[str, object]]) -> dict:
    result = {}
    for key, value in pairs:
        if key in result:
            raise SourceLockError("duplicate source lock key")
        result[key] = value
    return result


def load_revisions(path: Path) -> dict[str, str]:
    with path.open("rb") as source:
        raw = source.read(16 * 1024 + 1)
    if len(raw) > 16 * 1024:
        raise SourceLockError("source lock exceeds its bounded format")
    lock = json.loads(raw, object_pairs_hook=unique_object)
    if (not isinstance(lock, dict) or set(lock) != {"schema_version", *REPOSITORIES}
            or type(lock["schema_version"]) is not int or lock["schema_version"] != 1):
        raise SourceLockError("unsupported source lock")
    revisions = {}
    for role, repository in REPOSITORIES.items():
        entry = lock[role]
        if (not isinstance(entry, dict) or set(entry) != {"repository", "revision"}
                or entry["repository"] != repository or not isinstance(entry["revision"], str)
                or re.fullmatch(r"[0-9a-f]{40}", entry["revision"]) is None):
            raise SourceLockError("source lock requires known repositories and full commit hashes")
        revisions[role] = entry["revision"]
    return revisions


def main(arguments: list[str] | None = None) -> int:
    arguments = sys.argv[1:] if arguments is None else arguments
    if arguments not in ([], ["--github-output"]):
        print("FAIL invalid source-lock options; supplied values omitted", file=sys.stderr)
        return 2
    try:
        revisions = load_revisions(LOCK)
        if arguments:
            destination = os.environ.get("GITHUB_OUTPUT")
            if os.environ.get("GITHUB_ACTIONS") != "true" or not destination:
                raise SourceLockError("GitHub output requires the current Actions job")
            with Path(destination).open("a", encoding="utf-8") as output:
                for role, revision in revisions.items():
                    output.write(f"{role}={revision}\n")
        else:
            print(json.dumps(revisions, sort_keys=True))
        return 0
    except (OSError, ValueError, TypeError, KeyError):
        print("FAIL invalid repository source lock; raw values omitted", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
