#!/usr/bin/env bash
# SEC-01A 第三轮补修（review3b 第 3 项）：历史中 RFC 6455 示例 nonce（24b8315 / 125bc27）要不要加带提交号的指纹。
# 结论：不加。本脚本把依据固化为可重复的检查：
#   T1  本地全量历史扫描 + HEAD 的 .gitleaks.toml：按值 allowlist 已覆盖历史提交中的 nonce → 不报
#   T2  CI 全量路径（push 默认分支且 before 全零 → 范围 = HEAD）：工作流按设计用默认规则 + 空 ignore，
#       即使 HEAD 的 .gitleaksignore 里有 commit:path:rule:line 指纹也不加载 → 仍报（已知限制，记录在 README-ops）
#   T3  对照：同样的指纹若被加载（默认规则 + 该 ignore）则不报 —— 说明 T2 仍报是因为 CI 不加载，而不是指纹写错
#   T4  目标版本的 .gitleaksignore 没有有效条目，且注释如实写明 CI 全量路径不使用 allowlist/本文件
#
# 用法: .github/tests/test-gitleaks-history-fullscan.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
# 依赖: git、gitleaks、python3。夹具为生成的最小 Go 文件，不读取 internal/ 下的真实文件。
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"
for c in gitleaks python3; do command -v "$c" >/dev/null || { echo "缺少依赖: $c" >&2; exit 2; }; done

LT=internal/middleware/login_test.go
NONCE="$(printf 'the sample nonce' | base64)"   # RFC 6455 示例值，运行时计算
ZERO=0000000000000000000000000000000000000000

gl_new_origin hist
mv "$GL_DEV/.gitleaks.toml" "$GL_TMP/target.gitleaks.toml"
# 第 1 个提交：只有夹具（模拟 allowlist 引入之前的历史提交）
mkdir -p "$GL_DEV/internal/middleware"
{ printf 'package middleware\n\n'
  printf 'func a(h interface{ Set(string, string) }) {\n\th.Set("Sec-WebSocket-Key", "%s")\n}\n' "$NONCE"; } >"$GL_DEV/$LT"
git -C "$GL_DEV" add -A; git -C "$GL_DEV" -c core.hooksPath=/dev/null commit -qm "历史：含示例 nonce"
OLD="$(git -C "$GL_DEV" rev-parse HEAD)"
# 取该提交在默认规则下的精确指纹（commit:path:rule:line）
: >"$GL_TMP/empty.toml"
printf '[extend]\nuseDefault = true\n' >"$GL_TMP/default.toml"
gitleaks git --no-banner --redact --config "$GL_TMP/default.toml" --log-opts="$OLD" \
    --report-format json --report-path "$GL_TMP/fp.json" "$GL_DEV" >/dev/null 2>&1 || true
python3 -c 'import json,sys; [print(f["Fingerprint"]) for f in json.load(open(sys.argv[1])) or []]' "$GL_TMP/fp.json" >"$GL_TMP/fp.txt"
[ -s "$GL_TMP/fp.txt" ] || { echo "夹具在默认规则下未被检出，无法验证" >&2; exit 2; }
# 第 2 个提交：引入目标版本的 .gitleaks.toml，并在 .gitleaksignore 中加入上面的指纹（模拟“加指纹”方案）
cp "$GL_TMP/target.gitleaks.toml" "$GL_DEV/.gitleaks.toml"
{ if gl_has .gitleaksignore; then gl_src .gitleaksignore; fi; cat "$GL_TMP/fp.txt"; } >"$GL_DEV/.gitleaksignore"
git -C "$GL_DEV" add -A; git -C "$GL_DEV" -c core.hooksPath=/dev/null commit -qm "引入配置 + 模拟指纹"
git -C "$GL_DEV" push -q origin HEAD:main
HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"

# T1
rc=0; gitleaks git --no-banner --redact --ignore-gitleaks-allow --config "$GL_DEV/.gitleaks.toml" \
    --gitleaks-ignore-path "$GL_TMP/empty.toml" --log-opts="HEAD" "$GL_DEV" >"$GL_TMP/t1.out" 2>&1 || rc=$?
if [ "$rc" -eq 0 ]; then echo "PASS [T1 全量历史 + HEAD 配置：按值 allowlist 覆盖历史提交中的 nonce]"
else echo "FAIL [T1 全量历史 + HEAD 配置仍报出（rc=${rc}）：allowlist 未覆盖历史 nonce]"; GL_FAILS=$((GL_FAILS+1)); fi

# T2：执行目标版本工作流的 Scan new commits（全量路径）
gl_checkout "$HEAD_SHA"
gl_step scan
gl_expect_block "T2 CI 全量路径（before 全零）：默认规则 + 空 ignore，HEAD 中的指纹不生效，仍报（已知限制）" \
    "leak:generic-api-key:$LT" EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA=$HEAD_SHA DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch
rng="$(grep -m1 'gitleaks scan range' "$GL_TMP/scan.out" | sed 's/.*range: *//' || true)"
case "$rng" in
    *..*|"") echo "FAIL [T2 前提：before 全零时范围应为全量（单个提交号），实际: ${rng:-<无>}]"; GL_FAILS=$((GL_FAILS+1)) ;;
    *) echo "PASS [T2 前提：范围为全量历史（${rng}）]" ;;
esac

# T3
rc=0; gitleaks git --no-banner --redact --ignore-gitleaks-allow --config "$GL_TMP/default.toml" \
    --gitleaks-ignore-path "$GL_DEV/.gitleaksignore" --log-opts="HEAD" "$GL_DEV" >"$GL_TMP/t3.out" 2>&1 || rc=$?
if [ "$rc" -eq 0 ]; then echo "PASS [T3 对照：指纹被加载时默认规则下不报（指纹本身正确）]"
else echo "FAIL [T3 对照：加载指纹后仍报（rc=${rc}）]"; GL_FAILS=$((GL_FAILS+1)); fi

# T4
gl_src .gitleaksignore >"$GL_TMP/target.ign" 2>/dev/null || : >"$GL_TMP/target.ign"
active="$(grep -v -E '^[[:space:]]*(#|$)' "$GL_TMP/target.ign" || true)"
if [ -z "$active" ]; then echo "PASS [T4 .gitleaksignore 无有效条目（未加历史指纹）]"
else echo "FAIL [T4 .gitleaksignore 有有效条目: $(printf '%s' "$active" | tr '\n' ' ')]"; GL_FAILS=$((GL_FAILS+1)); fi
if grep -F -q 'CI 全量' "$GL_TMP/target.ign" && grep -F -q '默认规则' "$GL_TMP/target.ign"; then
    echo "PASS [T4 .gitleaksignore 注释写明 CI 全量路径用默认规则、不加载本文件]"
else echo "FAIL [T4 .gitleaksignore 注释未写明 CI 全量路径的限制（仍声称全量历史扫描不再报出）]"; GL_FAILS=$((GL_FAILS+1)); fi
gl_finish
