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
#   review3h P3-a（git diff 不比较 skip-worktree / assume-unchanged 文件，旧钩子被绕过）：
#   A6  .gitleaks.toml 标记 skip-worktree，工作区改成全部放行 + 暂存 PAT   → 拒绝提交（索引标记）
#   A7  .gitleaks.toml 标记 assume-unchanged，工作区改成全部放行 + 暂存 PAT → 拒绝提交
#   A8  .gitleaksignore 标记 skip-worktree，工作区加入指纹 + 暂存 PAT      → 拒绝提交
#   A9  .gitleaksignore 标记 assume-unchanged，工作区加入指纹 + 暂存 PAT   → 拒绝提交
#   A10 .gitleaksignore 标记 skip-worktree 但工作区与暂存区一致           → 仍拒绝（发现标记即拒绝）
#   A11 已跟踪 .gitleaksignore 在工作区换成指向放行指纹的符号链接          → 拒绝提交
#   A12 暂存区 .gitleaks.toml 用 [extend] path 引用未跟踪的全部放行配置    → 拒绝（gitleaks-config-check.py）
#   A13 同 A12，且工作区的 gitleaks-config-check.py 被改成直接 exit 0（未暂存）→ 仍拒绝（检查脚本取暂存区版本）
#   A14 同 A12，且 [Extend] 大小写变体                                    → 拒绝
#   对照 C1  暂存区 .gitleaks.toml 按值放行该 PAT（工作区一致）       → 提交成功（证明用的是暂存区配置）
#   对照 C2  暂存 .gitleaksignore 指纹（工作区一致）                    → 提交成功
#   对照 C3  无策略文件，暂存 PAT                                      → gitleaks 检出，拒绝
#   对照 C4  干净提交                                                  → 成功
#   对照 C5  暂存区 .gitleaks.toml 只用 [extend] useDefault（无 path），已跟踪文件无特殊标记 → 提交成功
#   对照 C6  清除 skip-worktree 标记并还原工作区后（A6 的仓库）          → 提交成功
#
# 用法: .github/tests/test-hook-policy-files.sh [--rev <git-rev>]   （对 review3g 版本运行，A6–A14 应报 FAIL）
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
CHECK_PY=.github/scripts/gitleaks-config-check.py
check_src() { if [ -n "$REV" ]; then git -C "$REPO" show "$REV:$CHECK_PY"; else cat "$REPO/$CHECK_PY"; fi; }
fake_token() { printf 'ghp_%s' "$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 36)"; }
ALLOW_ALL="$(printf '[extend]\nuseDefault = true\n\n[[allowlists]]\ndescription = "allow everything"\npaths = [%s]\n' "'''.*'''")"
DEFAULT_CFG="$(printf '[extend]\nuseDefault = true\n')"

new_repo() { # new_repo <name> [<path> <content>]...：初始提交（不经钩子）
    local d="$TMP/$1"; shift; mkdir -p "$d/.githooks"; cd "$d"; git init -q .
    hook_src >.githooks/pre-commit; chmod +x .githooks/pre-commit
    mkdir -p .github/scripts; check_src >"$CHECK_PY" 2>/dev/null || rm -f "$CHECK_PY"   # 目标版本的检查脚本（与仓库一致，随初始提交进入索引）
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

# review3h P3-a：skip-worktree / assume-unchanged 绕过
FP='k.env:github-pat:1'
CASE=a6; new_repo a6 .gitleaks.toml "$DEFAULT_CFG"; git update-index --skip-worktree .gitleaks.toml
printf '%s\n' "$ALLOW_ALL" >.gitleaks.toml; stage_secret; try_commit a6
expect_refuse 'A6 .gitleaks.toml skip-worktree + 工作区全部放行' '\.gitleaks\.toml 在索引中带有 skip-worktree.*: S'
CASE=a7; new_repo a7 .gitleaks.toml "$DEFAULT_CFG"; git update-index --assume-unchanged .gitleaks.toml
printf '%s\n' "$ALLOW_ALL" >.gitleaks.toml; stage_secret; try_commit a7
expect_refuse 'A7 .gitleaks.toml assume-unchanged + 工作区全部放行' '\.gitleaks\.toml 在索引中带有 .*assume-unchanged.*: h'
CASE=a8; new_repo a8 .gitleaksignore ''; git update-index --skip-worktree .gitleaksignore
printf '%s\n' "$FP" >.gitleaksignore; stage_secret; try_commit a8
expect_refuse 'A8 .gitleaksignore skip-worktree + 工作区指纹' '\.gitleaksignore 在索引中带有 skip-worktree.*: S'
CASE=a9; new_repo a9 .gitleaksignore ''; git update-index --assume-unchanged .gitleaksignore
printf '%s\n' "$FP" >.gitleaksignore; stage_secret; try_commit a9
expect_refuse 'A9 .gitleaksignore assume-unchanged + 工作区指纹' '\.gitleaksignore 在索引中带有 .*assume-unchanged.*: h'
CASE=a10; new_repo a10 .gitleaksignore ''; git update-index --skip-worktree .gitleaksignore; TOK=none-a10
printf 'x\n' >a.txt; git add a.txt; try_commit a10
expect_refuse 'A10 .gitleaksignore skip-worktree（工作区一致）也拒绝' '\.gitleaksignore 在索引中带有 skip-worktree'
CASE=a11; new_repo a11 .gitleaksignore '' fp.txt "$FP"; rm .gitleaksignore; ln -s fp.txt .gitleaksignore; stage_secret; try_commit a11
expect_refuse 'A11 .gitleaksignore 换成符号链接' '\.gitleaksignore 有未暂存的修改'
EXT_PATH="$(printf '[extend]\npath = "evil.toml"\n')"
CASE=a12; new_repo a12; printf '%s\n' "$ALLOW_ALL" >evil.toml; printf '%s\n' "$EXT_PATH" >.gitleaks.toml; git add .gitleaks.toml; stage_secret; try_commit a12
expect_refuse 'A12 暂存区配置 [extend] path 引用未跟踪的放行配置' 'gitleaks-config-check\] ERROR: .*path'
CASE=a13; new_repo a13; printf '%s\n' "$ALLOW_ALL" >evil.toml; printf '%s\n' "$EXT_PATH" >.gitleaks.toml; git add .gitleaks.toml
if [ -f "$CHECK_PY" ]; then printf 'import sys\nsys.exit(0)\n' >"$CHECK_PY"; fi; stage_secret; try_commit a13
expect_refuse 'A13 工作区检查脚本被改成 exit 0（未暂存）仍拒绝' 'gitleaks-config-check\] ERROR: .*path'
CASE=a14; new_repo a14; printf '%s\n' "$ALLOW_ALL" >evil.toml; printf '[Extend]\nPath = "evil.toml"\n' >.gitleaks.toml; git add .gitleaks.toml; stage_secret; try_commit a14
expect_refuse 'A14 [Extend] Path 大小写变体' 'gitleaks-config-check\] ERROR: .*path'

CASE=c1; new_repo c1; stage_secret
printf '%s\n\n[[allowlists]]\ndescription = "test control"\nregexTarget = "secret"\nregexes = [%s]\n' "$DEFAULT_CFG" "'''^${TOK}\$'''" >.gitleaks.toml
git add .gitleaks.toml; try_commit c1; expect_ok '对照 C1 暂存区配置按值放行（工作区一致）'
CASE=c2; new_repo c2; stage_secret; printf 'k.env:github-pat:1\n' >.gitleaksignore; git add .gitleaksignore; try_commit c2
expect_ok '对照 C2 暂存的 .gitleaksignore 指纹（工作区一致）'
CASE=c3; new_repo c3; stage_secret; try_commit c3
expect_refuse '对照 C3 无策略文件时暂存 PAT 被检出' 'gitleaks 在暂存内容中发现'
CASE=c4; new_repo c4; printf 'ok\n' >a.txt; git add a.txt; try_commit c4; expect_ok '对照 C4 干净提交'
CASE=c5; new_repo c5; printf '%s\n' "$DEFAULT_CFG" >.gitleaks.toml; git add .gitleaks.toml; printf 'ok\n' >a.txt; git add a.txt; try_commit c5
expect_ok '对照 C5 暂存区配置只有 [extend] useDefault，通过检查脚本'
CASE=c6; cd "$TMP/a6"; git update-index --no-skip-worktree .gitleaks.toml; git checkout -q -- .gitleaks.toml; git rm -q --cached k.env; rm -f k.env
printf 'ok\n' >a.txt; git add a.txt; try_commit c6; expect_ok '对照 C6 清除 skip-worktree 并还原后（A6 的仓库）'

echo "目标版本: ${REV:-工作区}；失败 $FAILS 项"
[ "$FAILS" -eq 0 ]
