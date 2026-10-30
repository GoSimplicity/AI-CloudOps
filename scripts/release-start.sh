#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Default is an offline plan. Existing deployments must use a separate adoption
# workflow; this entry point never upgrades or retries an existing volume.
exec python3 "$repo_dir/scripts/release-fresh.py" "$@"
