#!/usr/bin/env bash
# 从“基准提交”取出 gitleaks 配置与指纹忽略列表，供 CI 扫描使用（SEC-01A 第三轮 P3-1）。
#
# 用法: bash .github/scripts/gitleaks-base-config.sh <扫描范围> <输出目录>
#   <扫描范围> 为 gitleaks-range.sh 的输出：
#     "A..B" → 基准为 A（PR 的 base sha / push 的 before / 与默认分支的 merge-base）
#     "B"    → 无基准（默认分支全量扫描）→ 使用 gitleaks 默认规则、空忽略列表
#   输出: <输出目录>/gitleaks.toml、<输出目录>/gitleaksignore（总是生成，供 --config / --gitleaks-ignore-path 使用）
#
# 为什么：PR 可以同时提交密钥并修改 .gitleaks.toml / .gitleaksignore 放行它。CI 只信任基准提交上的这两个文件：
#   - 基准上没有该文件时使用默认配置（[extend] useDefault = true）/ 空忽略列表，绝不回退到 HEAD 的文件；
#   - gitleaks 总会额外读取“扫描源目录/.gitleaksignore”（即 HEAD 工作区中的文件，8.30.1 实测），
#     因此这里把工作区中的 .gitleaksignore 移走（CI 检出目录是一次性的；该文件的内容仍会被提交扫描覆盖）；
#   - 基准配置中的 [extend] path 会从当前工作区（HEAD）读取文件，视为不可信，直接失败。
# 局限：本脚本与工作流本身仍来自 PR 的 HEAD，修改 .github/ 的 PR 可以绕过——需要 CODEOWNERS/分支保护（见 README-ops 待办）。
set -euo pipefail

die()  { echo "[gitleaks-base-config] ERROR: $*" >&2; exit 1; }
note() { echo "[gitleaks-base-config] $*" >&2; }

[ "$#" -eq 2 ] || die "用法: $0 <扫描范围> <输出目录>"
range="$1"; out="$2"
[ -n "$range" ] || die "扫描范围为空"
mkdir -p "$out"
cfg="$out/gitleaks.toml"; ign="$out/gitleaksignore"

base=""
case "$range" in
    *...*) die "不支持的范围格式: $range" ;;
    *..*)  base="${range%%..*}" ;;
esac

if [ -n "$base" ]; then
    git rev-parse --verify --quiet "${base}^{commit}" >/dev/null || die "基准提交不存在: $base"
fi

if [ -n "$base" ] && git cat-file -e "${base}:.gitleaks.toml" 2>/dev/null; then
    git show "${base}:.gitleaks.toml" >"$cfg"
    note "使用基准 ${base} 上的 .gitleaks.toml"
    # [extend] 段中的 path = ... 会从工作区（HEAD）读取，不可信
    if awk '/^[[:space:]]*\[/{ sec=$0; gsub(/[[:space:]]/, "", sec) } sec=="[extend]" && /^[[:space:]]*path[[:space:]]*=/{ f=1 } END{ exit !f }' "$cfg"; then
        die "基准 .gitleaks.toml 的 [extend] 使用了 path（会读取 HEAD 工作区中的文件），拒绝使用"
    fi
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
