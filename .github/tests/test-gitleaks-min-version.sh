#!/usr/bin/env bash
# review3g N6：本地 gitleaks 最低版本 8.25.0（.gitleaks.toml 的多个 [[allowlists]] 自 8.25.0 起支持），
# 版本过低或无法解析时 .githooks/pre-commit 与 scripts/setup-git-hooks.sh 都必须失败；钩子不再回退到 `gitleaks protect --staged`。
# 用假的 gitleaks（只改写 `gitleaks version` 的输出，其余参数交给真实 gitleaks）模拟各版本：
#   钩子 H1 8.24.3（H1b 干净提交同样拒绝）/ H2 v8.18.4（旧版本走 protect 回退的路径）/ H3 无法解析（dev）/ H4 8.25（缺补丁号）
#        H5 空输出 / H6 8.25.0-rc1                                     → 拒绝提交，合成密钥不进入提交
#   对照 HC1 8.25.0 / HC2 v9.0.0 / HC3 真实版本 干净提交成功；HC4 8.25.0 暂存合成 PAT 仍被检出
#   钩子 H7 不含 `gitleaks protect` 调用（回退已删除）
#   setup（不依赖 husky 的仓库）S1 8.24.3 / S2 无法解析 → 非 0 退出；S3 8.25.0 → 成功
#   一致性 K1 钩子与 setup 中的 gitleaks_version_check 逐字一致
# 用法: .github/tests/test-gitleaks-min-version.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
# 在 mktemp -d 的临时仓库中运行；合成 PAT 在运行时随机生成，不是真实凭据。兼容 macOS bash 3.2 / BSD 工具。
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
REAL_GL="$(command -v gitleaks)"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
export GIT_CONFIG_GLOBAL="$TMP/gitconfig" GIT_CONFIG_NOSYSTEM=1
export SKIP_VET=1 SKIP_BUILD=1 SKIP_GENERATE=1
unset GITLEAKS_CONFIG GITLEAKS_CONFIG_TOML || true
git config --global user.name test; git config --global user.email test@example.invalid
git config --global init.defaultBranch main

src() { if [ -n "$REV" ]; then git -C "$REPO" show "$REV:$1"; else cat "$REPO/$1"; fi; }
fake_token() { printf 'ghp_%s' "$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 36)"; }
fake_gl() { # fake_gl <目录名> <version 输出>：返回该假 gitleaks 所在目录
    local d="$TMP/bin-$1"; mkdir -p "$d"
    { printf '#!/bin/sh\n'
      printf 'if [ "$1" = version ]; then printf "%%s\\n" '"'"'%s'"'"'; exit 0; fi\n' "$2"
      printf 'exec "%s" "$@"\n' "$REAL_GL"; } >"$d/gitleaks"
    chmod +x "$d/gitleaks"; echo "$d"
}
new_repo() { # new_repo <name>：初始提交（不经钩子）
    local d="$TMP/$1"; mkdir -p "$d/.githooks" "$d/scripts"; cd "$d"; git init -q .
    src .githooks/pre-commit >.githooks/pre-commit; chmod +x .githooks/pre-commit
    src scripts/setup-git-hooks.sh >scripts/setup-git-hooks.sh; chmod +x scripts/setup-git-hooks.sh
    printf 'hello\n' >README.md
    git add -A; git -c core.hooksPath=/dev/null commit -qm init
    git config core.hooksPath .githooks
}
FAILS=0
fail() { echo "FAIL [$1] $2"; FAILS=$((FAILS+1)); }
TOK=""
hook_case() { # hook_case <名称> <版本输出|REAL> <secret|clean> <期望 refuse|ok> [<ERE>]
    local name="$1" ver="$2" kind="$3" want="$4" ere="${5:-}" bin rc
    CASE="h$RANDOM$RANDOM"; new_repo "$CASE"
    if [ "$ver" = REAL ]; then bin="$(dirname "$REAL_GL")"; else bin="$(fake_gl "$CASE" "$ver")"; fi
    if [ "$kind" = secret ]; then TOK="$(fake_token)"; printf 'GITHUB_TOKEN=%s\n' "$TOK" >k.env; git add k.env
    else TOK=""; printf 'ok\n' >a.txt; git add a.txt; fi
    set +e; PATH="$bin:$PATH" git commit -qm "$CASE" >"$TMP/$CASE.out" 2>&1; rc=$?; set -e
    git log --all -p >"$TMP/log.txt"   # 不用管道：pipefail + grep -q 会因 SIGPIPE 误判
    if [ -n "$TOK" ] && grep -qF "$TOK" "$TMP/log.txt"; then fail "$name" "合成密钥进入了提交（rc=${rc}）"; return; fi
    if [ "$want" = ok ]; then
        git log -1 --format=%s >"$TMP/subj.txt"
        if [ "$rc" -eq 0 ] && grep -qx "$CASE" "$TMP/subj.txt"; then echo "PASS [$name] 提交成功"
        else fail "$name" "提交未成功（rc=${rc}）: $(tail -1 "$TMP/$CASE.out")"; fi
    elif [ "$rc" -eq 0 ]; then fail "$name" "提交成功（应被拒绝）"
    elif grep -qE "$ere" "$TMP/$CASE.out"; then echo "PASS [$name] 拒绝提交（rc=${rc}）: $(grep -m1 -E "$ere" "$TMP/$CASE.out")"
    else fail "$name" "提交被拒但原因不符（期望 /$ere/）: $(tail -1 "$TMP/$CASE.out")"; fi
}
LOW='低于 8\.25\.0'; BAD='无法解析 gitleaks 版本'
hook_case 'H1 gitleaks 8.24.3'                 '8.24.3'      secret refuse "$LOW"
hook_case 'H1b gitleaks 8.24.3 干净提交也拒绝'     '8.24.3'      clean  refuse "$LOW"
hook_case 'H2 gitleaks v8.18.4（旧回退路径）'   'v8.18.4'     secret refuse "$LOW"
hook_case 'H3 版本输出无法解析（dev）'           'dev'         secret refuse "$BAD"
hook_case 'H4 版本缺补丁号（8.25）'             '8.25'        secret refuse "$BAD"
hook_case 'H5 版本输出为空'                     ''            secret refuse "$BAD"
hook_case 'H6 预发布版本（8.25.0-rc1）'         '8.25.0-rc1'  secret refuse "$BAD"
hook_case '对照 HC1 8.25.0 干净提交'            '8.25.0'      clean  ok
hook_case '对照 HC2 v9.0.0 干净提交'            'v9.0.0'      clean  ok
hook_case '对照 HC3 真实 gitleaks 干净提交'      REAL          clean  ok
hook_case '对照 HC4 8.25.0 暂存合成 PAT 仍被检出' '8.25.0'      secret refuse 'gitleaks 在暂存内容中发现'

src .githooks/pre-commit >"$TMP/hook.txt"
if grep -E '^[[:space:]]*gitleaks[[:space:]]+protect' "$TMP/hook.txt" >/dev/null; then fail 'H7 钩子不再回退到 gitleaks protect' "仍有: $(grep -m1 -E '^[[:space:]]*gitleaks[[:space:]]+protect' "$TMP/hook.txt")"
else echo "PASS [H7 钩子不再回退到 gitleaks protect]"; fi

# setup：依赖 husky 的仓库（web）由 scripts/tests/security/test-setup-git-hooks.sh 覆盖
src scripts/setup-git-hooks.sh >"$TMP/setup.txt"
if grep -qF 'node_modules/.bin/husky' "$TMP/setup.txt"; then echo "SKIP [S1–S3 setup 依赖 husky，见 scripts/tests/security/test-setup-git-hooks.sh 情况5]"
else
    setup_case() { # setup_case <名称> <版本输出> <期望 fail|ok>
        local d rc; d="s$RANDOM$RANDOM"; new_repo "$d"; git config --unset core.hooksPath
        set +e; PATH="$(fake_gl "$d" "$2"):$PATH" bash scripts/setup-git-hooks.sh >"$TMP/$d.out" 2>&1; rc=$?; set -e
        if { [ "$3" = fail ] && [ "$rc" -ne 0 ]; } || { [ "$3" = ok ] && [ "$rc" -eq 0 ]; }; then echo "PASS [$1] rc=${rc}"
        else fail "$1" "rc=${rc}（期望 $3）: $(tail -1 "$TMP/$d.out")"; fi
    }
    setup_case 'S1 setup：gitleaks 8.24.3' '8.24.3' fail
    setup_case 'S2 setup：版本无法解析'     'dev'    fail
    setup_case '对照 S3 setup：gitleaks 8.25.0' '8.25.0' ok
fi

fn_of() { awk '/^gitleaks_version_check\(\) \{/{p=1} p{print} p&&/^}/{exit}' "$1"; }
fn_of "$TMP/hook.txt" >"$TMP/fn-hook.txt"; fn_of "$TMP/setup.txt" >"$TMP/fn-setup.txt"
if [ -s "$TMP/fn-hook.txt" ] && cmp -s "$TMP/fn-hook.txt" "$TMP/fn-setup.txt"; then echo "PASS [K1 钩子与 setup 的 gitleaks_version_check 一致]"
else fail 'K1 钩子与 setup 的 gitleaks_version_check 一致' "缺失或不一致"; fi

echo "目标版本: ${REV:-工作区}；失败 $FAILS 项"
[ "$FAILS" -eq 0 ]
