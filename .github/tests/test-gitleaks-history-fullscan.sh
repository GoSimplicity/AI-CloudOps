#!/usr/bin/env bash
# SEC-01A 第三轮补修（review3b 第 3 项）：历史中 RFC 6455 示例 nonce（24b8315 / 125bc27）要不要加带提交号的指纹。
# 结论：不加。本脚本把依据固化为可重复的检查：
#   T1  本地全量历史扫描 + HEAD 的 .gitleaks.toml：按值 allowlist 已覆盖历史提交中的 nonce → 不报
#       （review3g N5：扫描前移走开发仓库的 .gitleaksignore，否则 gitleaks 自动加载其中的指纹，T1 的通过不能归因于 allowlist）
#   T1b 对照：同样条件下只用默认规则（无 allowlist）→ 必须检出
#   T2  CI 全量路径（push 默认分支且 before 全零 → 范围 = HEAD）：工作流按设计用默认规则 + 空 ignore，
#       即使 HEAD 的 .gitleaksignore 里有 commit:path:rule:line 指纹也不加载 → 仍报（已知限制，记录在 README-ops）
#   T3  对照：同样的指纹若被加载（默认规则 + 该 ignore）则不报 —— 说明 T2 仍报是因为 CI 不加载，而不是指纹写错
#   T4  目标版本的 .gitleaksignore 没有有效条目，且注释如实写明 CI 全量路径不使用 allowlist/本文件
#   T5  目标版本的 .gitleaks.toml 不声称 RFC 6455 allowlist 可删除（review3g：历史中默认规则仍有 4 处命中）
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

# T1（review3g N5）：先把开发仓库中的 .gitleaksignore 移走——gitleaks 总会自动读取“源目录/.gitleaksignore”，
# 而上面第 2 个提交在其中写入了历史指纹，不移走时 T1 的通过可能来自指纹而不是 allowlist
mv "$GL_DEV/.gitleaksignore" "$GL_TMP/dev.gitleaksignore"
rc=0; gitleaks git --no-banner --redact --ignore-gitleaks-allow --config "$GL_DEV/.gitleaks.toml" \
    --gitleaks-ignore-path "$GL_TMP/empty.toml" --log-opts="HEAD" "$GL_DEV" >"$GL_TMP/t1.out" 2>&1 || rc=$?
if [ "$rc" -eq 0 ]; then echo "PASS [T1 全量历史 + HEAD 配置（无任何 .gitleaksignore）：按值 allowlist 覆盖历史提交中的 nonce]"
else echo "FAIL [T1 全量历史 + HEAD 配置仍报出（rc=${rc}）：allowlist 未覆盖历史 nonce]"; GL_FAILS=$((GL_FAILS+1)); fi
# T1b 对照：同样无 .gitleaksignore、只用默认规则（无 allowlist）→ 必须检出，证明 T1 的通过来自 allowlist
rc=0; gitleaks git --no-banner --redact --ignore-gitleaks-allow --config "$GL_TMP/default.toml" \
    --gitleaks-ignore-path "$GL_TMP/empty.toml" --log-opts="HEAD" --report-format json --report-path "$GL_TMP/t1b.json" \
    "$GL_DEV" >"$GL_TMP/t1b.out" 2>&1 || rc=$?
t1b="$(python3 -c 'import json,sys; print(sum(1 for f in json.load(open(sys.argv[1])) or [] if f["File"]==sys.argv[2] and f["RuleID"]=="generic-api-key"))' "$GL_TMP/t1b.json" "$LT" 2>/dev/null || echo 0)"
if [ "$rc" -ne 0 ] && [ "$t1b" -gt 0 ]; then echo "PASS [T1b 对照：无 allowlist、无 .gitleaksignore 时检出 ${t1b} 处（rc=${rc}）]"
else echo "FAIL [T1b 对照：无 allowlist 时应检出（rc=${rc}，命中 ${t1b}）]"; GL_FAILS=$((GL_FAILS+1)); fi
mv "$GL_TMP/dev.gitleaksignore" "$GL_DEV/.gitleaksignore"

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
# T5（review3g 注释修正）：.gitleaks.toml 不得再声称 RFC 6455 allowlist “不再命中任何内容/可删除”——历史仍含该 nonce（T1b 即证）
if grep -F -q '可在之后删除' "$GL_TMP/target.gitleaks.toml" || grep -F -q '不再命中任何内容' "$GL_TMP/target.gitleaks.toml"; then
    echo "FAIL [T5 .gitleaks.toml 仍声称 RFC 6455 allowlist 可删除（历史中仍有命中）]"; GL_FAILS=$((GL_FAILS+1))
elif grep -F -q '必须保留' "$GL_TMP/target.gitleaks.toml"; then echo "PASS [T5 .gitleaks.toml 注释写明历史仍命中、allowlist 必须保留]"
else echo "FAIL [T5 .gitleaks.toml 注释未写明 allowlist 必须保留]"; GL_FAILS=$((GL_FAILS+1)); fi
gl_finish
