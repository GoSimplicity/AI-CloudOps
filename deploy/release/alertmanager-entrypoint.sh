#!/bin/sh
# 把 WEBHOOK_ALERTMANAGER_BEARER_TOKEN 写成 Alertmanager 读取的文件。脚本和配置里都不放真实值。
set -eu
if [ -z "${WEBHOOK_ALERTMANAGER_BEARER_TOKEN:-}" ]; then
  echo "WEBHOOK_ALERTMANAGER_BEARER_TOKEN is required" >&2
  exit 1
fi
umask 077
printf '%s' "$WEBHOOK_ALERTMANAGER_BEARER_TOKEN" > /tmp/webhook_bearer_token
exec /bin/alertmanager "$@"
