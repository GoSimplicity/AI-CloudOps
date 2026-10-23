#!/usr/bin/env bash
# SEC-01A 第三轮审查 P3-1：行内 "gitleaks:allow" 注释、以及 PR 自带的 .gitleaks.toml / .gitleaksignore 都不能放行密钥。
#
#   15   pre-commit：暂存 k.env 中的合成 GitHub PAT 行尾带 "# gitleaks:allow"          → 必须拦下
#   15b  pre-commit：app.js 中 "// gitleaks:allow" 变体                                  → 必须拦下
#   16   CI Scan new commits：新提交中带 gitleaks:allow 的合成 PAT                        → 必须拦下
#   17   CI Scan working tree：base 中已存在、带 gitleaks:allow 的合成 PAT（范围内无新密钥）→ 必须拦下
#   18   CI：PR 同时提交密钥并把 .gitleaks.toml 改成全部放行                               → 必须拦下（用 base 的配置）
#   18b  CI：PR 同时提交密钥并在 .gitleaksignore 加入它的指纹                              → 必须拦下（用 base 的忽略列表）
#   19   CI：base 上没有 .gitleaks.toml，PR 新增一个全部放行的 .gitleaks.toml               → 必须拦下（用默认规则）
#   U1/U2  gitleaks-base-config.sh：无基准时用默认规则；基准配置 [extend] path 拒绝（review3h：断言检查器的拒绝原因）
#   对照 U2c：基准配置合法时通过
#   对照 C1/C2：base 上的 allowlist / .gitleaksignore 确实生效（同一合成值在 CI 中放行）
#
# 用法: .github/tests/test-gitleaks-allow-bypass.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
# 合成密钥均在运行时随机生成；所有仓库都在临时目录中创建。
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"
export SKIP_VET=1 SKIP_BUILD=1 SKIP_GENERATE=1

ALLOW_ALL="$(printf '[extend]\nuseDefault = true\n\n[[allowlists]]\ndescription = "allow everything"\npaths = [%s]\n' "'''.*'''")"

# ---------- pre-commit（目标版本的 .githooks/pre-commit） ----------
hook_case() { # hook_case <名称> <文件> <内容>
    local name="$1" f="$2" content="$3" rc=0 before after
    gl_new_origin "hk$RANDOM"
    mkdir -p "$GL_DEV/.githooks"; gl_src .githooks/pre-commit >"$GL_DEV/.githooks/pre-commit"; chmod +x "$GL_DEV/.githooks/pre-commit"
    gl_commit README.md hello init
    git -C "$GL_DEV" config core.hooksPath .githooks
    before="$(git -C "$GL_DEV" rev-parse HEAD)"
    mkdir -p "$(dirname "$GL_DEV/$f")"; printf '%s\n' "$content" >"$GL_DEV/$f"; git -C "$GL_DEV" add "$f"
    git -C "$GL_DEV" commit -qm "$name" >"$GL_TMP/hook.out" 2>&1 || rc=$?
    after="$(git -C "$GL_DEV" rev-parse HEAD)"
    if [ "$rc" -ne 0 ] && [ "$after" = "$before" ] \
        && awk -v f="$f" '$1=="RuleID:"{r=$2} $1=="File:"{ if (r=="github-pat" && $2==f) ok=1 } END{exit !ok}' "$GL_TMP/hook.out"; then
        echo "PASS [$name] pre-commit 拦下（github-pat @ ${f}）"
    elif [ "$rc" -ne 0 ] && [ "$after" = "$before" ]; then
        echo "FAIL [$name] 提交被拒但不是因为检出 github-pat @ ${f}: $(grep -m1 -E 'ERROR|RuleID' "$GL_TMP/hook.out")"; GL_FAILS=$((GL_FAILS+1))
    else
        echo "FAIL [$name] pre-commit 放行（rc=${rc}），带 gitleaks:allow 的密钥进入提交"; GL_FAILS=$((GL_FAILS+1))
    fi
}
hook_case "15 pre-commit：# gitleaks:allow" k.env "GITHUB_TOKEN=$(gl_fake_ghp) # gitleaks:allow"
hook_case "15b pre-commit：// gitleaks:allow（JS）" web/app.js "const token = \"$(gl_fake_ghp)\"; // gitleaks:allow"

# ---------- CI ----------
# pr_case <名称> <base 是否带目标 .gitleaks.toml: yes|no> <base 额外文件 路径|-> <base 额外内容> -- 之后为 PR 中的 文件 内容 对
#   PR 提交：feature 分支上一个提交，事件为 pull_request（PR_BASE_SHA = base）
setup_pr() { # setup_pr <name> <base_cfg yes|no>
    gl_new_origin "$1"
    if [ "$2" = no ]; then rm -f "$GL_DEV/.gitleaks.toml"; fi
}
base_push() {
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base
    git -C "$GL_DEV" push -q origin HEAD:main; BASE_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"
    git -C "$GL_DEV" checkout -q -b feature
}
pr_push() {
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm pr
    git -C "$GL_DEV" push -q origin HEAD:feature; HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"
    gl_checkout "$HEAD_SHA"
    PR_ARGS="EVENT_NAME=pull_request PR_BASE_SHA=$BASE_SHA PR_BASE_REF=main HEAD_SHA=$HEAD_SHA DEFAULT_BRANCH=main REF_NAME=1/merge REF_TYPE=branch"
}

# 16：新提交中带 gitleaks:allow
setup_pr c16 yes; printf 'hello\n' >"$GL_DEV/README.md"; base_push
printf 'GITHUB_TOKEN=%s # gitleaks:allow\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; pr_push
gl_step scan; gl_expect_block "16 CI Scan new commits：gitleaks:allow 不豁免" leak:github-pat:k.env $PR_ARGS

# 17：base 中已有带 gitleaks:allow 的密钥（绕过钩子提交），PR 只改 README
setup_pr c17 yes; printf 'GITHUB_TOKEN=%s # gitleaks:allow\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; base_push
printf 'hello 2\n' >"$GL_DEV/README.md"; pr_push
gl_step scan; gl_expect_pass "17 前提：范围内无新增密钥，Scan new commits 通过" $PR_ARGS
gl_step tree; gl_expect_block "17 CI Scan working tree：gitleaks:allow 不豁免" leak:github-pat:k.env $PR_ARGS

# 18：PR 把 .gitleaks.toml 改为全部放行
setup_pr c18 yes; printf 'hello\n' >"$GL_DEV/README.md"; base_push
printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/.gitleaks.toml"; printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; pr_push
gl_step scan; gl_expect_block "18 CI：PR 修改 .gitleaks.toml 全部放行（范围扫描）" leak:github-pat:k.env $PR_ARGS
gl_step tree; gl_expect_block "18 CI：PR 修改 .gitleaks.toml 全部放行（工作树扫描）" leak:github-pat:k.env $PR_ARGS

# 18b：PR 在 .gitleaksignore 加入密钥指纹（path:rule:line 对 git 与 dir 扫描都生效）
setup_pr c18b yes; printf 'hello\n' >"$GL_DEV/README.md"; base_push
printf 'k.env:github-pat:1\n' >"$GL_DEV/.gitleaksignore"; printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; pr_push
gl_step scan; gl_expect_block "18b CI：PR 新增 .gitleaksignore 指纹（范围扫描）" leak:github-pat:k.env $PR_ARGS
gl_step tree; gl_expect_block "18b CI：PR 新增 .gitleaksignore 指纹（工作树扫描）" leak:github-pat:k.env $PR_ARGS

# 19：base 无 .gitleaks.toml，PR 新增全部放行的配置
setup_pr c19 no; printf 'hello\n' >"$GL_DEV/README.md"; base_push
printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/.gitleaks.toml"; printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; pr_push
gl_step scan; gl_expect_block "19 CI：base 无配置、PR 新增全部放行配置" leak:github-pat:k.env $PR_ARGS

# 对照 C1：base 的 .gitleaks.toml 按值放行某个合成值 → PR 中出现该值应放行（证明确实加载了 base 的配置）
setup_pr cc1 yes; tok="$(gl_fake_ghp)"
printf '\n[[allowlists]]\ndescription = "test control"\nregexTarget = "secret"\nregexes = [%s]\n' "'''^${tok}\$'''" >>"$GL_DEV/.gitleaks.toml"
printf 'hello\n' >"$GL_DEV/README.md"; base_push
printf 'GITHUB_TOKEN=%s\n' "$tok" >"$GL_DEV/k.env"; pr_push
gl_step scan; gl_expect_pass "对照 C1：base 配置中的 allowlist 生效（范围扫描）" $PR_ARGS
gl_step tree; gl_expect_pass "对照 C1：base 配置中的 allowlist 生效（工作树扫描）" $PR_ARGS

# 对照 C2：base 的 .gitleaksignore 指纹生效
setup_pr cc2 yes; printf 'k.env:github-pat:1\n' >"$GL_DEV/.gitleaksignore"; printf 'hello\n' >"$GL_DEV/README.md"; base_push
printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; pr_push
gl_step scan; gl_expect_pass "对照 C2：base 的 .gitleaksignore 生效（范围扫描）" $PR_ARGS
gl_step tree; gl_expect_pass "对照 C2：base 的 .gitleaksignore 生效（工作树扫描）" $PR_ARGS
gl_step scan

# U1/U2：gitleaks-base-config.sh 的边界（目标版本没有该脚本时判 FAIL）
if gl_has .github/scripts/gitleaks-base-config.sh; then
    gl_new_origin u1; printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/.gitleaks.toml"; printf 'x:y:1\n' >"$GL_DEV/.gitleaksignore"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; h="$(git -C "$GL_DEV" rev-parse HEAD)"
    gl_src .github/scripts/gitleaks-base-config.sh >"$GL_TMP/bc.sh"
    # review3h P3-g：bc.sh 按 $(dirname "$0")/gitleaks-config-check.py 调检查器，必须一起复制；
    # 否则检查器缺失也会让 bc.sh 失败，U2 会“因为缺文件”而误判通过
    rm -f "$GL_TMP/gitleaks-config-check.py"
    if gl_has .github/scripts/gitleaks-config-check.py; then gl_src .github/scripts/gitleaks-config-check.py >"$GL_TMP/gitleaks-config-check.py"; fi
    rc=0; (cd "$GL_DEV" && EVENT_NAME=push DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch bash "$GL_TMP/bc.sh" "$h" "$GL_TMP/u1out") >"$GL_TMP/u1.log" 2>&1 || rc=$?
    if [ "$rc" -eq 0 ] && ! grep -q allowlists "$GL_TMP/u1out/gitleaks.toml" && grep -q 'useDefault = true' "$GL_TMP/u1out/gitleaks.toml" \
        && [ ! -s "$GL_TMP/u1out/gitleaksignore" ] && [ ! -e "$GL_DEV/.gitleaksignore" ]; then
        echo "PASS [U1 无基准（全量扫描）时用默认规则、空忽略列表，不用 HEAD 的文件，并移走工作区 .gitleaksignore]"
    else echo "FAIL [U1 无基准时应使用默认规则/空忽略列表] rc=${rc}: $(tail -1 "$GL_TMP/u1.log")"; GL_FAILS=$((GL_FAILS+1)); fi
    gl_new_origin u2; printf '[extend]\npath = "evil.toml"\n' >"$GL_DEV/.gitleaks.toml"; printf 'a\n' >"$GL_DEV/a"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; b="$(git -C "$GL_DEV" rev-parse HEAD)"
    printf 'b\n' >"$GL_DEV/a"; git -C "$GL_DEV" commit -qam two; h="$(git -C "$GL_DEV" rev-parse HEAD)"
    rc=0; (cd "$GL_DEV" && EVENT_NAME=push DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch bash "$GL_TMP/bc.sh" "$b..$h" "$GL_TMP/u2out") >"$GL_TMP/u2.log" 2>&1 || rc=$?
    # 断言具体的拒绝原因（检查器的输出），不只看退出码和 "extend" 字样
    if [ ! -f "$GL_TMP/gitleaks-config-check.py" ]; then
        echo "FAIL [U2 基准配置使用 [extend] path 时应失败] 目标版本没有 .github/scripts/gitleaks-config-check.py"; GL_FAILS=$((GL_FAILS+1))
    elif [ "$rc" -ne 0 ] && grep -F -q '配置的 [extend] 使用了 path' "$GL_TMP/u2.log"; then
        echo "PASS [U2 基准配置 [extend] path 指向工作区文件时拒绝（检查器报 [extend] 使用了 path）]"
    else echo "FAIL [U2 基准配置使用 [extend] path 时应失败且由检查器拒绝] rc=${rc}: $(tail -1 "$GL_TMP/u2.log")"; GL_FAILS=$((GL_FAILS+1)); fi
    # 对照 U2c：同样的流程，基准配置合法（useDefault + 一条自定义规则）时通过，输出就是基准上的配置
    gl_new_origin u2c; printf '[extend]\nuseDefault = true\n\n[[rules]]\nid = "u2c-rule"\nregex = %s\n' "'''u2c-[0-9]{8}'''" >"$GL_DEV/.gitleaks.toml"; printf 'a\n' >"$GL_DEV/a"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; b="$(git -C "$GL_DEV" rev-parse HEAD)"
    printf 'b\n' >"$GL_DEV/a"; git -C "$GL_DEV" commit -qam two; h="$(git -C "$GL_DEV" rev-parse HEAD)"
    rc=0; (cd "$GL_DEV" && EVENT_NAME=push DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch bash "$GL_TMP/bc.sh" "$b..$h" "$GL_TMP/u2cout") >"$GL_TMP/u2c.log" 2>&1 || rc=$?
    git -C "$GL_DEV" show "$b:.gitleaks.toml" >"$GL_TMP/u2c-base.toml"
    if [ "$rc" -eq 0 ] && cmp -s "$GL_TMP/u2c-base.toml" "$GL_TMP/u2cout/gitleaks.toml"; then
        echo "PASS [对照 U2c 基准配置合法（无 [extend] path）时通过，使用基准配置]"
    else echo "FAIL [对照 U2c 基准配置合法时应通过] rc=${rc}: $(tail -1 "$GL_TMP/u2c.log")"; GL_FAILS=$((GL_FAILS+1)); fi
else
    echo "FAIL [U1/U2] 目标版本没有 .github/scripts/gitleaks-base-config.sh（CI 仍使用 HEAD 的配置）"; GL_FAILS=$((GL_FAILS+1))
fi
gl_finish
