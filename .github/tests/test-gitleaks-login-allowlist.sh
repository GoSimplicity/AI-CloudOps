#!/usr/bin/env bash
# SEC-01A 第三轮审查 P2-1：login_test.go 中 RFC 6455 示例 nonce 的误报豁免必须“按值”放行。
# 旧做法是 .gitleaksignore 里的 path:rule:line 指纹（:109 / :130）——同一行换成任何真实密钥都会被放行。
# 新做法：.gitleaks.toml 中 [[allowlists]]（targetRules + paths + regexTarget=secret + regexes，condition=AND）。
#
#   正例  原 nonce 出现在 login_test.go 第 109/130 行：pre-commit 与 CI（范围扫描 + 工作树扫描）均放行
#   反例  同文件同一行换成合成密钥：pre-commit、CI 范围扫描、CI 工作树扫描都必须拦下（generic-api-key @ 该文件）
#   反例  同一个 nonce 值出现在其他文件：必须拦下（allowlist 只限该文件）
#
# 用法: .github/tests/test-gitleaks-login-allowlist.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
# 依赖: git、gofmt、gitleaks、python3。夹具为生成的最小 Go 文件（行号与原文件一致），不读取 internal/ 下的真实文件。
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"
for c in gofmt gitleaks python3; do command -v "$c" >/dev/null || { echo "缺少依赖: $c" >&2; exit 2; }; done
export SKIP_VET=1 SKIP_BUILD=1 SKIP_GENERATE=1 GOTOOLCHAIN=local

LT=internal/middleware/login_test.go
# RFC 6455 第 1.3 节示例值：base64("the sample nonce")，运行时计算，本脚本不含该字面量
NONCE="$(printf 'the sample nonce' | base64)"

# 生成与原文件行号一致的夹具：第 109 行与第 130 行为 Sec-WebSocket-Key 赋值，其余为注释/空行（gofmt 规范）
gen_fixture() { # gen_fixture <109 行的值> <130 行的值>
    local i=3
    printf 'package middleware\n\n'
    while [ "$i" -le 107 ]; do printf '// filler %s\n' "$i"; i=$((i+1)); done
    printf 'func a(h interface{ Set(string, string) }) {\n\th.Set("Sec-WebSocket-Key", "%s")\n}\n\n' "$1"
    i=112
    while [ "$i" -le 128 ]; do printf '// filler %s\n' "$i"; i=$((i+1)); done
    printf 'func b(h interface{ Set(string, string) }) {\n\th.Set("Sec-WebSocket-Key", "%s")\n}\n' "$2"
}

# 开发者仓库：目标版本的 .gitleaks.toml / .gitleaksignore / .githooks/pre-commit + 夹具（原 nonce）
new_dev() {
    gl_new_origin "$1"
    if gl_has .gitleaksignore; then gl_src .gitleaksignore >"$GL_DEV/.gitleaksignore"; fi
    mkdir -p "$GL_DEV/.githooks" "$GL_DEV/internal/middleware"
    gl_src .githooks/pre-commit >"$GL_DEV/.githooks/pre-commit"; chmod +x "$GL_DEV/.githooks/pre-commit"
    printf 'module example.com/t\n\ngo 1.21\n' >"$GL_DEV/go.mod"
    gen_fixture "$NONCE" "$NONCE" >"$GL_DEV/$LT"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" -c core.hooksPath=/dev/null commit -qm init
    git -C "$GL_DEV" config core.hooksPath .githooks
    [ -z "$(gofmt -l "$GL_DEV/$LT")" ] || { echo "夹具格式不规范" >&2; exit 2; }
    [ "$(sed -n 109p "$GL_DEV/$LT" | grep -c "$NONCE")" = 1 ] && [ "$(sed -n 130p "$GL_DEV/$LT" | grep -c "$NONCE")" = 1 ] \
        || { echo "夹具行号不符" >&2; exit 2; }
}

# 通过 pre-commit 钩子提交，判定是否被拦以及拦截原因
hook_commit() { # hook_commit <名称> <期望 block|pass> <期望文件:行>（block 时）
    local name="$1" want="$2" where="${3:-}" rc=0 before after
    before="$(git -C "$GL_DEV" rev-parse HEAD)"
    git -C "$GL_DEV" commit -qm "$name" >"$GL_TMP/hook.out" 2>&1 || rc=$?
    after="$(git -C "$GL_DEV" rev-parse HEAD)"
    if [ "$want" = pass ]; then
        if [ "$rc" -eq 0 ] && [ "$after" != "$before" ]; then echo "PASS [$name] pre-commit 放行，提交成功"
        else echo "FAIL [$name] 应放行但被拦（rc=${rc}）: $(grep -m1 -E 'RuleID|ERROR|发现' "$GL_TMP/hook.out")"; GL_FAILS=$((GL_FAILS+1)); fi
        return 0
    fi
    local file="${where%:*}" line="${where##*:}"
    if [ "$rc" -ne 0 ] && [ "$after" = "$before" ] \
        && awk -v f="$file" -v l="$line" '$1=="RuleID:"{r=$2} $1=="File:"{p=$2} $1=="Line:"{ if (r=="generic-api-key" && p==f && $2==l) ok=1 } END{exit !ok}' "$GL_TMP/hook.out"; then
        echo "PASS [$name] pre-commit 拦下（generic-api-key @ ${file}:${line}）"
    elif [ "$rc" -ne 0 ] && [ "$after" = "$before" ]; then
        echo "FAIL [$name] 提交被拒但不是因为检出 generic-api-key @ ${where}: $(grep -m1 -E 'ERROR|RuleID|gofmt' "$GL_TMP/hook.out")"; GL_FAILS=$((GL_FAILS+1))
    else
        echo "FAIL [$name] pre-commit 放行（rc=${rc}），密钥进入提交"; GL_FAILS=$((GL_FAILS+1))
    fi
}

# ---------- pre-commit ----------
new_dev pc-pos
printf '\n// touch\n' >>"$GL_DEV/$LT"; git -C "$GL_DEV" add "$LT"
hook_commit "正例 pre-commit：原 nonce 在第 109/130 行" pass

new_dev pc-neg109
gen_fixture "$(gl_fake_generic)" "$NONCE" >"$GL_DEV/$LT"; git -C "$GL_DEV" add "$LT"
hook_commit "反例 pre-commit：第 109 行换成合成密钥" block "$LT:109"

new_dev pc-neg130
gen_fixture "$NONCE" "$(gl_fake_generic)" >"$GL_DEV/$LT"; git -C "$GL_DEV" add "$LT"
hook_commit "反例 pre-commit：第 130 行换成合成密钥" block "$LT:130"

new_dev pc-other
mkdir -p "$GL_DEV/internal/other"; sed 's/^package middleware/package other/' "$GL_DEV/$LT" >"$GL_DEV/internal/other/ws_test.go"
git -C "$GL_DEV" add -A
hook_commit "反例 pre-commit：同一 nonce 出现在其他文件" block "internal/other/ws_test.go:109"

# ---------- CI（执行目标版本工作流的 Scan new commits / Scan working tree 步骤） ----------
ci_case() { # ci_case <名称前缀> <109 行的值> <期望 pass|block>
    new_dev "ci$RANDOM"
    git -C "$GL_DEV" push -q origin HEAD:main; local before; before="$(git -C "$GL_DEV" rev-parse HEAD)"
    gen_fixture "$2" "$NONCE" >"$GL_DEV/$LT"; printf '\n// change\n' >>"$GL_DEV/$LT"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" -c core.hooksPath=/dev/null commit -qm change
    git -C "$GL_DEV" push -q origin HEAD:main; local head; head="$(git -C "$GL_DEV" rev-parse HEAD)"
    gl_checkout "$head"
    local args="EVENT_NAME=push PUSH_BEFORE=$before HEAD_SHA=$head DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch"
    if [ "$3" = pass ]; then
        gl_step scan; gl_expect_pass "$1 CI 范围扫描" $args
        gl_step tree; gl_expect_pass "$1 CI 工作树扫描" $args
    else
        gl_step scan; gl_expect_block "$1 CI 范围扫描" "leak:generic-api-key:$LT" $args
        gl_step tree; gl_expect_block "$1 CI 工作树扫描" "leak:generic-api-key:$LT" $args
    fi
    gl_step scan
}
ci_case "正例：原 nonce 保持不变" "$NONCE" pass
ci_case "反例：第 109 行换成合成密钥" "$(gl_fake_generic)" block
gl_finish
