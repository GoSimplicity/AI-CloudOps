#!/usr/bin/env bash
# SEC-01A 审查 P1 复现/回归测试：.githooks/pre-commit 不得把未经 gitleaks 扫描的内容带进提交。
#
#   G1  部分暂存的 .go 文件：暂存版本无密钥，工作区未暂存部分含密钥（暂存版本格式不规范 / 规范两种）
#   G2  从未暂存过的 .go 文件（含密钥、格式不规范）不得被钩子自动加入提交
#
# 用法: scripts/tests/security/test-precommit-hook.sh [--rev <git-rev>]
#   --rev  测试指定版本的 .githooks/pre-commit（默认：当前工作区文件）。对修复前版本运行应报 FAIL。
# 依赖: git、go/gofmt、gitleaks（PATH 或 ~/.local/bin）。在 mktemp -d 的临时仓库中运行，不改动本仓库。
# 密钥为运行时拼出的合成 GitHub PAT（ghp_ + 36 位随机字符），不是真实凭据。
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
REV=""
while [ $# -gt 0 ]; do
    case "$1" in
        --rev) REV="$2"; shift 2 ;;
        *) echo "未知参数: $1" >&2; exit 2 ;;
    esac
done
export PATH="$HOME/.local/bin:$PATH"
for c in git go gofmt gitleaks; do command -v "$c" >/dev/null || { echo "缺少依赖: $c" >&2; exit 2; }; done

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
export GIT_CONFIG_GLOBAL="$TMP/gitconfig" GIT_CONFIG_NOSYSTEM=1 GOTOOLCHAIN=local GOFLAGS=-mod=mod
export SKIP_VET=1 SKIP_BUILD=1 SKIP_GENERATE=1   # 只关心暂存/格式化逻辑
git config --global user.name test; git config --global user.email test@example.invalid
git config --global init.defaultBranch main

hook_src() {
    if [ -n "$REV" ]; then git -C "$REPO" show "$REV:.githooks/pre-commit"; else cat "$REPO/.githooks/pre-commit"; fi
}
fake_token() { printf 'ghp_%s' "$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 36)"; }

new_repo() {
    local d="$TMP/$1"; mkdir -p "$d/.githooks"; cd "$d"; git init -q .
    hook_src >.githooks/pre-commit; chmod +x .githooks/pre-commit
    printf 'module example.com/t\n\ngo 1.21\n' >go.mod
    printf 'package main\n\nfunc main() {}\n' >main.go
    git add -A; git -c core.hooksPath=/dev/null commit -qm init
    git config core.hooksPath .githooks
}

FAILS=0
# 判定：提交历史中出现合成密钥 = 漏（LEAK）
check() {
    local name="$1" tok="$2" rc="$3" hits
    hits="$(git log --all -p | grep -cF "$tok" || true)"
    if [ "$hits" -gt 0 ]; then
        echo "FAIL [$name] 密钥进入了提交（commit rc=$rc, 命中 $hits 处） -> 漏"; FAILS=$((FAILS+1))
    else
        echo "PASS [$name] 密钥未进入提交（commit rc=$rc）"
    fi
}

# G1a：暂存版本格式不规范且无密钥；工作区追加未暂存的密钥
new_repo g1a; TOK="$(fake_token)"
printf 'package main\n\nfunc   main()  {\n}\n' >main.go; git add main.go
printf 'package main\n\nvar token = "%s"\n\nfunc   main()  {\n}\n' "$TOK" >main.go
set +e; git commit -qm g1a >"$TMP/g1a.out" 2>&1; rc=$?; set -e
check G1a-部分暂存-暂存版不规范 "$TOK" "$rc"
[ "$rc" -ne 0 ] && grep -q 'gofmt' "$TMP/g1a.out" && echo "      (提交被拒并提示 gofmt)"

# G1b：暂存版本规范；工作区未暂存部分含密钥且不规范
new_repo g1b; TOK="$(fake_token)"
printf 'package main\n\nfunc main() { println("x") }\n' >main.go; git add main.go
printf 'package main\n\nvar   token = "%s"\n\nfunc main() { println("x") }\n' "$TOK" >main.go
set +e; git commit -qm g1b >"$TMP/g1b.out" 2>&1; rc=$?; set -e
check G1b-部分暂存-暂存版规范 "$TOK" "$rc"

# G2：从未暂存的 .go 文件（含密钥、不规范）；同时暂存另一个规范的 .go 改动
new_repo g2; TOK="$(fake_token)"
printf 'package main\n\nvar   leaked = "%s"\n' "$TOK" >secret.go
printf 'package main\n\nfunc main() { println("y") }\n' >main.go; git add main.go
set +e; git commit -qm g2 >"$TMP/g2.out" 2>&1; rc=$?; set -e
check G2-未暂存文件被自动加入 "$TOK" "$rc"

# 负向对照：暂存版本含密钥时必须被 gitleaks 拦下
new_repo ctl; TOK="$(fake_token)"
printf 'package main\n\nvar token = "%s"\n\nfunc main() {}\n' "$TOK" >main.go; git add main.go
set +e; git commit -qm ctl >"$TMP/ctl.out" 2>&1; rc=$?; set -e
check 对照-暂存密钥被gitleaks拦截 "$TOK" "$rc"

echo "目标版本: ${REV:-工作区}；失败 $FAILS 项"
[ "$FAILS" -eq 0 ]
