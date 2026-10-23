#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release_dir="$repo_dir/deploy/release"

for sibling in AI-CloudOps-web AI-CloudOps-aiops; do
  if [[ ! -f "$repo_dir/../$sibling/Dockerfile" ]]; then
    echo "Missing sibling repository: $repo_dir/../$sibling" >&2
    exit 1
  fi
done

docker info >/dev/null
if [[ ! -f "$release_dir/.env" ]]; then
  python3 "$repo_dir/scripts/release-env.py"
fi

docker compose -f "$release_dir/compose.yaml" config --quiet
docker compose -f "$release_dir/compose.yaml" up -d --build --wait --wait-timeout 300
echo "CloudOps started. Default URL: http://localhost:18080 (see HTTP_PORT/PUBLIC_ORIGIN in deploy/release/.env)."
echo "Initial account: admin. Read BOOTSTRAP_ADMIN_PASSWORD privately from deploy/release/.env."
echo "Run python3 scripts/release-smoke.py from the backend repository to verify the gateway."
