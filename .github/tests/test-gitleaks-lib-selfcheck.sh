#!/usr/bin/env bash
# SEC-01A 第三轮审查 P3-3 自检：lib-gitleaks-workflow.sh 的 gl_expect_block 必须区分“检出密钥”与“步骤出错”，
# 并核对检出的 RuleID / File 与用例预期一致（否则 gitleaks 自身报错、范围脚本出错也会被当成“已拦下”）。
# 用法: .github/tests/test-gitleaks-lib-selfcheck.sh [--rev <git-rev>]   （使用该版本的 lib；修复前版本应报 FAIL）
set -euo pipefail
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SELF_REV=""
while [ $# -gt 0 ]; do case "$1" in --rev) SELF_REV="$2"; shift 2;; *) echo "未知参数: $1" >&2; exit 2;; esac; done
SELF_LIB="$SELF_DIR/lib-gitleaks-workflow.sh"
if [ -n "$SELF_REV" ]; then
    SELF_LIB="$(mktemp)"; git -C "$SELF_DIR" show "$SELF_REV:.github/tests/lib-gitleaks-workflow.sh" >"$SELF_LIB"
fi
# shellcheck source=/dev/null
source "$SELF_LIB"
[ -n "$SELF_REV" ] && rm -f "$SELF_LIB"

# 用自定义步骤脚本替换“从工作流提取的步骤”，只检验判定逻辑本身
SELF_STEP="$GL_TMP/selfcheck-step.sh"
gl_extract_run() { cp "$SELF_STEP" "$1"; }
GL_WS="$GL_TMP/selfcheck-ws"; mkdir -p "$GL_WS"
printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_WS/b.env"

SELF_FAILS=0
check() { # check <用例> <期望判定 PASS|FAIL> <gl_expect_block 参数...>
    local name="$1" want="$2" out verdict; shift 2
    out="$(GL_FAILS=0; gl_expect_block "$@" 2>&1 || true)"
    verdict="$(printf '%s\n' "$out" | grep -m1 -oE '^(PASS|FAIL)' || echo '<无判定>')"
    if [ "$verdict" = "$want" ]; then echo "PASS [${name}] gl_expect_block 判定为 ${verdict}（符合预期）"
    else echo "FAIL [${name}] gl_expect_block 判定为 ${verdict}，应为 ${want} | ${out}"; SELF_FAILS=$((SELF_FAILS+1)); fi
}

# S1 步骤因 gitleaks/脚本出错而失败（没有任何检出），用例期望检出 -> 必须判 FAIL
printf '%s\n' 'echo "ERR gitleaks: failed to load config" >&2; exit 1' >"$SELF_STEP"
check "S1 出错不能当成检出" FAIL "s1" leak:github-pat:b.env
# S2 检出了，但文件与预期不符 -> 必须判 FAIL
printf '%s\n' 'gitleaks dir --redact --no-banner --verbose --exit-code 1 .' >"$SELF_STEP"
check "S2 检出文件不符" FAIL "s2" leak:github-pat:a.env
# S3 检出了，但规则与预期不符 -> 必须判 FAIL
check "S3 检出规则不符" FAIL "s3" leak:aws-access-token:b.env
# S4 规则与文件都符合 -> PASS
check "S4 规则与文件都符合" PASS "s4" leak:github-pat:b.env
# S5 期望出错且出错信息匹配 -> PASS；期望出错但实际检出 -> FAIL
printf '%s\n' 'echo "[gitleaks-range] ERROR: 缺少 origin/main" >&2; exit 1' >"$SELF_STEP"
check "S5 期望的出错" PASS "s5" 'error:缺少 origin/main'
printf '%s\n' 'gitleaks dir --redact --no-banner --verbose --exit-code 1 .' >"$SELF_STEP"
check "S6 期望出错却检出" FAIL "s6" 'error:缺少 origin/main'
echo "目标 lib 版本: ${SELF_REV:-工作区}；失败 ${SELF_FAILS} 项"
[ "$SELF_FAILS" -eq 0 ]
