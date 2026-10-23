#!/usr/bin/env bash
# SEC-01A 审查 P3：.gitignore 中宽泛的 "!*.example" 会把被 ".env.*" 等规则忽略的文件重新放出来
# （如 .env.prod.example、config/.env.local.example 这类常被拿来存真实值的文件名）。
# 修复后只对被跟踪的模板逐个放行。
# 用法: <本脚本> [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
set -euo pipefail
REPO="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
REV=""
while [ $# -gt 0 ]; do case "$1" in --rev) REV="$2"; shift 2;; *) echo "未知参数: $1" >&2; exit 2;; esac; done
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
export GIT_CONFIG_GLOBAL="$TMP/gitconfig" GIT_CONFIG_NOSYSTEM=1; : >"$GIT_CONFIG_GLOBAL"
git init -q "$TMP/r"
if [ -n "$REV" ]; then git -C "$REPO" show "$REV:.gitignore"; else cat "$REPO/.gitignore"; fi >"$TMP/r/.gitignore"
FAILS=0
ign() { git -C "$TMP/r" check-ignore -q --no-index -- "$1"; }
expect_tracked() { if ign "$1"; then echo "FAIL [模板应被跟踪] $1 被忽略"; FAILS=$((FAILS+1)); else echo "PASS [模板应被跟踪] $1"; fi; }
expect_ignored() { if ign "$1"; then echo "PASS [应被忽略] $1"; else echo "FAIL [应被忽略] $1 未被忽略（可被 git add 提交）"; FAILS=$((FAILS+1)); fi; }

# 1) 仓库中当前被跟踪的所有 *.example 模板都不能被忽略
while IFS= read -r f; do expect_tracked "$f"; done < <(git -C "$REPO" ls-files -- '*.example' | sort)
# 2) 被 .env.* 覆盖的"伪模板"必须仍被忽略
expect_ignored ".env.example"
expect_ignored ".env.prod.example"
expect_ignored "config/.env.local.example"
expect_ignored "deploy/.env.staging.example"
# 3) 对照：真实 env / 集群凭据仍被忽略
expect_ignored ".env"
expect_ignored ".env.bak-20261023"
expect_ignored "deploy/kubernetes/config"
echo "目标版本: ${REV:-工作区}；失败 $FAILS 项"; [ "$FAILS" -eq 0 ]
