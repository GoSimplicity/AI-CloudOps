#!/usr/bin/env bash
# 从“基准提交”取出 gitleaks 配置与指纹忽略列表，供 CI 扫描使用（SEC-01A 第三轮 P3-1）。
#
# 用法: bash .github/scripts/gitleaks-base-config.sh <扫描范围> <输出目录>
#   <扫描范围> 为 gitleaks-range.sh 的输出（"A..B" 或 "B"）。
#   输出: <输出目录>/gitleaks.toml、<输出目录>/gitleaksignore（总是生成，供 --config / --gitleaks-ignore-path 使用）
#   环境变量（与 gitleaks-range.sh 相同，来自 GitHub 事件上下文，必填 EVENT_NAME / DEFAULT_BRANCH）：
#     EVENT_NAME、DEFAULT_BRANCH、REF_NAME、REF_TYPE（push 必填，只接受 branch / tag）、PR_BASE_REF（github.base_ref）、REMOTE（默认 origin）
#
# 策略来源（取哪个提交上的 .gitleaks.toml / .gitleaksignore；review3g N2：扫描起点 ≠ 策略来源）：
#   push 到默认分支                    → 范围起点 A（默认分支上一次的状态）；"B"（全量）→ 默认规则、空忽略列表
#   push 到其他分支 / tag              → <REMOTE>/<默认分支> 的最新提交（不用 before：before 是同一分支上一次推送，
#                                         攻击者可以先推一个全部放行的 .gitleaks.toml，再推密钥）
#   pull_request 目标为默认分支        → 范围起点 A（PR base sha，在默认分支上）
#   pull_request 目标为其他分支 / 未知 → <REMOTE>/<默认分支> 的最新提交（同上，非默认分支不受保护）
#   缺少 EVENT_NAME / DEFAULT_BRANCH，或需要 <REMOTE>/<默认分支> 但不存在 → 失败（不回退到范围起点）
#
# 为什么：PR 可以同时提交密钥并修改 .gitleaks.toml / .gitleaksignore 放行它。CI 只信任基准提交上的这两个文件：
#   - 策略提交上没有该文件时使用默认配置（[extend] useDefault = true）/ 空忽略列表，绝不回退到 HEAD 的文件；
#   - gitleaks 总会额外读取“扫描源目录/.gitleaksignore”（即 HEAD 工作区中的文件，8.30.1 实测），
#     因此这里把工作区中的 .gitleaksignore 移走（CI 检出目录是一次性的；该文件的内容仍会被提交扫描覆盖）；
#   - 基准配置中的 [extend] path 会从当前工作区（HEAD）读取文件，视为不可信，直接失败；
#     检查由 gitleaks-config-check.py（tomllib，Python 3.11+）完成，大小写/点号/内联表等写法都算，缺 tomllib 也失败。
# 局限：本脚本与工作流本身仍来自 PR 的 HEAD，修改 .github/ 的 PR 可以绕过——需要 CODEOWNERS/分支保护（见 README-ops 待办）。
set -euo pipefail

die()  { echo "[gitleaks-base-config] ERROR: $*" >&2; exit 1; }
note() { echo "[gitleaks-base-config] $*" >&2; }

[ "$#" -eq 2 ] || die "用法: $0 <扫描范围> <输出目录>"
range="$1"; out="$2"
[ -n "$range" ] || die "扫描范围为空"
REMOTE="${REMOTE:-origin}"
[ -n "${EVENT_NAME:-}" ] || die "缺少 EVENT_NAME，无法确定策略来源（不回退到扫描范围起点）"
[ -n "${DEFAULT_BRANCH:-}" ] || die "缺少 DEFAULT_BRANCH，无法确定策略来源"
mkdir -p "$out"
cfg="$out/gitleaks.toml"; ign="$out/gitleaksignore"

start=""
case "$range" in
    *...*) die "不支持的范围格式: $range" ;;
    *..*)  start="${range%%..*}" ;;
esac
if [ -n "$start" ]; then
    git rev-parse --verify --quiet "${start}^{commit}" >/dev/null || die "范围起点提交不存在: $start"
fi

# 策略提交 base：只取默认分支上的提交（见文件头）
use_default_tip=0
case "$EVENT_NAME" in
    push)
        # review3h P3-i：REF_TYPE 只接受 branch / tag（区分大小写），空值或其他写法失败，不再默认当作 branch
        case "${REF_TYPE:-}" in
            branch|tag) ;;
            *) die "REF_TYPE=${REF_TYPE:-<空>} 无效（只接受 branch / tag），无法确定策略来源" ;;
        esac
        if [ "$REF_TYPE" = "branch" ] && [ "${REF_NAME:-}" = "$DEFAULT_BRANCH" ]; then
            base="$start"
        else
            use_default_tip=1
        fi ;;
    pull_request|pull_request_target)
        if [ "${PR_BASE_REF:-}" = "$DEFAULT_BRANCH" ]; then base="$start"; else use_default_tip=1; fi ;;
    *) die "不支持的事件类型: $EVENT_NAME" ;;
esac
if [ "$use_default_tip" -eq 1 ]; then
    default_ref="refs/remotes/${REMOTE}/${DEFAULT_BRANCH}"
    base="$(git rev-parse --verify -q "${default_ref}^{commit}")" \
        || die "推送/PR 目标不是默认分支，策略须取自 ${REMOTE}/${DEFAULT_BRANCH}，但本地没有该引用（actions/checkout 需 fetch-depth: 0）"
    note "目标不是默认分支（${REF_TYPE:-?} ${REF_NAME:-?}${PR_BASE_REF:+，PR 目标 $PR_BASE_REF}）：策略取自 ${REMOTE}/${DEFAULT_BRANCH}（${base}），不用扫描起点 ${start:-<无>}"
else
    note "策略取自默认分支上的范围起点 ${base:-（无，全量扫描）}"
fi

if [ -n "$base" ] && git cat-file -e "${base}:.gitleaks.toml" 2>/dev/null; then
    git show "${base}:.gitleaks.toml" >"$cfg"
    note "使用基准 ${base} 上的 .gitleaks.toml"
    # [extend] path 会从工作区（HEAD）读取，不可信；用 TOML 解析器按 gitleaks（viper）的规则检查（review3g N1）
    command -v python3 >/dev/null 2>&1 || die "缺少 python3（需要 3.11+ 的 tomllib）检查基准 .gitleaks.toml 的 [extend]，拒绝继续"
    python3 "$(dirname "$0")/gitleaks-config-check.py" "$cfg" \
        || die "基准 .gitleaks.toml 未通过 [extend] 检查（见上一行），拒绝使用"
else
    printf '# 基准提交上没有 .gitleaks.toml：使用 gitleaks 默认规则（不使用 HEAD 的配置）\n[extend]\nuseDefault = true\n' >"$cfg"
    note "基准 ${base:-（无，全量扫描）} 上没有 .gitleaks.toml，使用 gitleaks 默认规则"
fi

if [ -n "$base" ] && git cat-file -e "${base}:.gitleaksignore" 2>/dev/null; then
    git show "${base}:.gitleaksignore" >"$ign"
    note "使用基准 ${base} 上的 .gitleaksignore"
else
    : >"$ign"
    note "基准 ${base:-（无，全量扫描）} 上没有 .gitleaksignore，使用空忽略列表"
fi

if [ -e .gitleaksignore ]; then
    mv .gitleaksignore "$out/head.gitleaksignore"
    note "已移走工作区中的 .gitleaksignore（HEAD 版本，不作为忽略列表使用）"
fi
