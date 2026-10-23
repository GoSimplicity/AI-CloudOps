#!/usr/bin/env bash
# review3g P3-1：.githooks/pre-commit 的 gitleaks 策略（.gitleaks.toml / .gitleaksignore）只取暂存区版本。
# 修复前：钩子用工作区的 .gitleaks.toml，gitleaks 还会自动读取工作区的 .gitleaksignore——未跟踪或未暂存的
# 放行配置/指纹在提交时生效，却不进入提交。
#
#   A1  未跟踪的 .gitleaks.toml（全部放行）+ 暂存合成 PAT            → 拒绝提交（未跟踪），密钥不进提交
#   A2  已跟踪 .gitleaks.toml，工作区改成全部放行但未暂存 + 暂存 PAT  → 拒绝提交（未暂存的修改）
#   A3  未跟踪的 .gitleaksignore（含该 PAT 的指纹）+ 暂存 PAT         → 拒绝提交（未跟踪）
#   A4  已跟踪的空 .gitleaksignore，工作区加入指纹但未暂存 + 暂存 PAT → 拒绝提交（未暂存的修改）
#   A5  已跟踪 .gitleaks.toml 在工作区被删除（未暂存）                → 拒绝提交
#   对照 C1  暂存区 .gitleaks.toml 按值放行该 PAT（工作区一致）       → 提交成功（证明用的是暂存区配置）
#   对照 C2  暂存 .gitleaksignore 指纹（工作区一致）                    → 提交成功
#   对照 C3  无策略文件，暂存 PAT                                      → gitleaks 检出，拒绝
#   对照 C4  干净提交                                                  → 成功
#
# 用法: .github/tests/test-hook-policy-files.sh [--rev <git-rev>]   （对修复前版本运行，A1–A5 应报 FAIL）
# 在 mktemp -d 的临时仓库中运行；合成 PAT 在运行时随机生成，不是真实凭据。
set -euo pipefail
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REV=""
while [ $# -gt 0 ]; do
    case "$1" in
        --rev) REV="$2"; shift 2 ;;
        *) echo "未知参数: $1" >&2; exit 2 ;;
    esac
done
export PATH="$HOME/.local/bin:$PATH"
for c in git gitleaks; do command -v "$c" >/dev/null || { echo "缺少依赖: $c" >&2; exit 2; }; done
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
export GIT_CONFIG_GLOBAL="$TMP/gitconfig" GIT_CONFIG_NOSYSTEM=1
export SKIP_VET=1 SKIP_BUILD=1 SKIP_GENERATE=1
unset GITLEAKS_CONFIG GITLEAKS_CONFIG_TOML || true
git config --global user.name test; git config --global user.email test@example.invalid
git config --global init.defaultBranch main

hook_src() { if [ -n "$REV" ]; then git -C "$REPO" show "$REV:.githooks/pre-commit"; else cat "$REPO/.githooks/pre-commit"; fi; }
fake_token() { printf 'ghp_%s' "$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 36)"; }
ALLOW_ALL="$(printf '[extend]\nuseDefault = true\n\n[[allowlists]]\ndescription = "allow everything"\npaths = [%s]\n' "'''.*'''")"
DEFAULT_CFG="$(printf '[extend]\nuseDefault = true\n')"

new_repo() { # new_repo <name> [<path> <content>]...：初始提交（不经钩子）
    local d="$TMP/$1"; shift; mkdir -p "$d/.githooks"; cd "$d"; git init -q .
    hook_src >.githooks/pre-commit; chmod +x .githooks/pre-commit
    printf 'hello\n' >README.md
    while [ $# -ge 2 ]; do printf '%s\n' "$2" >"$1"; shift 2; done
    git add -A; git -c core.hooksPath=/dev/null commit -qm init
    git config core.hooksPath .githooks
}
stage_secret() { TOK="$(fake_token)"; printf 'GITHUB_TOKEN=%s\n' "$TOK" >k.env; git add k.env; }
try_commit() { set +e; git commit -qm "$1" >"$TMP/$1.out" 2>&1; RC=$?; set -e; }
leaked() { git log --all -p >"$TMP/log.txt"; grep -qF "$TOK" "$TMP/log.txt"; }   # 不用管道：pipefail + grep -q 会因 SIGPIPE 误判

FAILS=0
fail() { echo "FAIL [$1] $2"; FAILS=$((FAILS+1)); }
expect_refuse() { # expect_refuse <名称> <ERE>
    if leaked; then fail "$1" "密钥进入了提交（commit rc=${RC}）-> 漏"
    elif [ "$RC" -eq 0 ]; then fail "$1" "提交成功（应被拒绝）"
    elif grep -qE "$2" "$TMP/$CASE.out"; then echo "PASS [$1] 拒绝提交（rc=${RC}）: $(grep -m1 -E "$2" "$TMP/$CASE.out")"
    else fail "$1" "提交被拒但原因不符（期望 /$2/）: $(tail -1 "$TMP/$CASE.out")"; fi
}
expect_ok() { # expect_ok <名称>
    if [ "$RC" -eq 0 ] && git log -1 --format=%s | grep -qx "$CASE"; then echo "PASS [$1] 提交成功"
    else fail "$1" "提交未成功（rc=${RC}）: $(tail -2 "$TMP/$CASE.out" | tr '\n' ' ')"; fi
}

CASE=a1; new_repo a1; printf '%s\n' "$ALLOW_ALL" >.gitleaks.toml; stage_secret; try_commit a1
expect_refuse 'A1 未跟踪的全部放行 .gitleaks.toml' '\.gitleaks\.toml 未跟踪'
CASE=a2; new_repo a2 .gitleaks.toml "$DEFAULT_CFG"; printf '%s\n' "$ALLOW_ALL" >.gitleaks.toml; stage_secret; try_commit a2
expect_refuse 'A2 .gitleaks.toml 未暂存的全部放行修改' '\.gitleaks\.toml 有未暂存的修改'
CASE=a3; new_repo a3; printf 'k.env:github-pat:1\n' >.gitleaksignore; stage_secret; try_commit a3
expect_refuse 'A3 未跟踪的 .gitleaksignore 指纹' '\.gitleaksignore 未跟踪'
CASE=a4; new_repo a4 .gitleaksignore ''; printf 'k.env:github-pat:1\n' >.gitleaksignore; stage_secret; try_commit a4
expect_refuse 'A4 .gitleaksignore 未暂存的指纹' '\.gitleaksignore 有未暂存的修改'
CASE=a5; new_repo a5 .gitleaks.toml "$DEFAULT_CFG"; rm .gitleaks.toml; TOK=none-a5; printf 'x\n' >a.txt; git add a.txt; try_commit a5
expect_refuse 'A5 .gitleaks.toml 在工作区被删除（未暂存）' '\.gitleaks\.toml 有未暂存的修改'

CASE=c1; new_repo c1; stage_secret
printf '%s\n\n[[allowlists]]\ndescription = "test control"\nregexTarget = "secret"\nregexes = [%s]\n' "$DEFAULT_CFG" "'''^${TOK}\$'''" >.gitleaks.toml
git add .gitleaks.toml; try_commit c1; expect_ok '对照 C1 暂存区配置按值放行（工作区一致）'
CASE=c2; new_repo c2; stage_secret; printf 'k.env:github-pat:1\n' >.gitleaksignore; git add .gitleaksignore; try_commit c2
expect_ok '对照 C2 暂存的 .gitleaksignore 指纹（工作区一致）'
CASE=c3; new_repo c3; stage_secret; try_commit c3
expect_refuse '对照 C3 无策略文件时暂存 PAT 被检出' 'gitleaks 在暂存内容中发现'
CASE=c4; new_repo c4; printf 'ok\n' >a.txt; git add a.txt; try_commit c4; expect_ok '对照 C4 干净提交'

echo "目标版本: ${REV:-工作区}；失败 $FAILS 项"
[ "$FAILS" -eq 0 ]
