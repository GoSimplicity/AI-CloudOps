#!/usr/bin/env bash
# review3g N1：基准 .gitleaks.toml 的 [extend] path 检查改用 TOML 解析器（gitleaks-config-check.py，tomllib）。
# gitleaks 8.30.1 实测（见 review3g/n1-gitleaks-extend-probe.log）：键名不区分大小写，带点的键按层级展开，
# 下列写法都会让 gitleaks 读取 path 指向的文件；旧的 awk 只认 "[extend]" + 行首 "path ="，可以绕过。
#
# 每个用例：main 上一次推送的 .gitleaks.toml 用该写法指向 evil.toml；下一次推送加入“全部放行”的 evil.toml
# 和合成 PAT（before = 上一次推送）→ 必须因 [extend] 检查失败（修复前：gitleaks 读入 evil.toml，漏扫）。
#   X1..X13  各种 extend.path 写法（含大小写、点号键、内联表、引号键、\u 转义、子表）
#   M1  多个大小写不同的 extend 表        M2  extend 不是表          M3  TOML 解析失败
#   M4  没有 tomllib（PYTHONPATH 中放一个抛 ModuleNotFoundError 的 tomllib.py 模拟 Python < 3.11）
#   M5  PATH 中没有 python3（直接调用 gitleaks-base-config.sh）
#   对照 P1 目标版本自己的 .gitleaks.toml、P2 useDefault + disabledRules：检查通过，合成 PAT 照常检出
#
# 用法: .github/tests/test-gitleaks-extend-check.sh [--rev <git-rev>]   （对修复前版本运行，X*/M* 应报 FAIL）
# 合成密钥均在运行时随机生成；所有仓库都在临时目录中创建。
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"
ALLOW_ALL="$(printf '[extend]\nuseDefault = true\n\n[[allowlists]]\ndescription = "allow everything"\npaths = [%s]\n' "'''.*'''")"
P='error:\[extend\] 使用了 path'

# setup_case <name> <配置行...>：main 推送该配置（H1），再推 evil.toml(全部放行) + 合成 PAT（H2），检出 H2
setup_case() {
    local name="$1"; shift
    gl_new_origin "$name"
    if [ "$#" -gt 0 ]; then printf '%s\n' "$@" >"$GL_DEV/.gitleaks.toml"; fi
    printf 'hello\n' >"$GL_DEV/README.md"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; git -C "$GL_DEV" push -q origin HEAD:main
    H1="$(git -C "$GL_DEV" rev-parse HEAD)"
    printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/evil.toml"
    printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm secret; git -C "$GL_DEV" push -q origin HEAD:main
    H2="$(git -C "$GL_DEV" rev-parse HEAD)"
    gl_checkout "$H2"
    ARGS="EVENT_NAME=push PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch"
}
xcase() { # xcase <名称> <期望 error:ERE> <配置行...>
    local name="$1" want="$2"; shift 2
    setup_case "x$GL_N" "$@"; GL_N=$((GL_N+1))
    # shellcheck disable=SC2086
    gl_expect_block "$name" "$want" $ARGS
}
GL_N=1
gl_step scan
xcase 'X1 [extend] path' "$P" '[extend]' 'path = "evil.toml"'
xcase 'X2 [extend] 后带注释' "$P" '[extend] # x' 'path = "evil.toml"'
xcase 'X3 点号键 extend.path' "$P" 'extend.path = "evil.toml"'
xcase 'X4 内联表 extend = { path }' "$P" 'extend = { path = "evil.toml" }'
xcase 'X5 引号表名 ["extend"]' "$P" '["extend"]' 'path = "evil.toml"'
xcase 'X6 引号键 "path"' "$P" '[extend]' '"path" = "evil.toml"'
xcase 'X7 [Extend] Path（大小写）' "$P" '[Extend]' 'Path = "evil.toml"'
xcase 'X8 [EXTEND]' "$P" '[EXTEND]' 'path = "evil.toml"'
xcase 'X9 引号点号键 "extend.path"' "$P" '"extend.path" = "evil.toml"'
xcase 'X10 引号点号表 ["extend.path"]' "$P" '["extend.path"]' 'x = 1'
xcase 'X11 子表 [extend.path]' "$P" '[extend.path]' 'x = "evil.toml"'
xcase 'X12 \u 转义表名' "$P" '["\u0065xtend"]' 'path = "evil.toml"'
xcase 'X13 [extend] path.x' "$P" '[extend]' 'path.x = "evil.toml"'
xcase 'M1 多个大小写不同的 extend 表' 'error:多个 extend 写法' '[extend]' 'useDefault = true' '[Extend]' 'useDefault = true'
xcase 'M2 extend 不是表' 'error:extend 不是表' 'extend = "evil.toml"'
xcase 'M3 TOML 解析失败' 'error:无法按 TOML 解析' '[extend' 'path = "evil.toml"'

# M4：模拟没有 tomllib（目标版本自己的配置，本应通过）
SHADOW="$GL_TMP/shadow-py"; mkdir -p "$SHADOW"
printf 'raise ModuleNotFoundError("tomllib shadowed by test (simulating Python < 3.11)")\n' >"$SHADOW/tomllib.py"
setup_case m4
# shellcheck disable=SC2086
gl_expect_block 'M4 没有 tomllib（Python < 3.11）' 'error:缺少 tomllib' $ARGS PYTHONPATH="$SHADOW"

# M5：PATH 中没有 python3（只放 base-config 需要的命令），直接调用脚本
setup_case m5
MINBIN="$GL_TMP/minbin"; mkdir -p "$MINBIN"
for c in git dirname mkdir mv cat; do ln -sf "$(command -v "$c")" "$MINBIN/$c"; done
rc=0; ( cd "$GL_WS" && env PATH="$MINBIN" EVENT_NAME=push DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch \
    "$(command -v bash)" .github/scripts/gitleaks-base-config.sh "$H1..$H2" "$GL_TMP/m5-out" ) >"$GL_TMP/m5.log" 2>&1 || rc=$?
if [ "$rc" -ne 0 ] && grep -q '缺少 python3' "$GL_TMP/m5.log"; then echo "PASS [M5 PATH 中没有 python3 时失败] $(grep -m1 '缺少 python3' "$GL_TMP/m5.log")"
else echo "FAIL [M5 PATH 中没有 python3 时应失败] rc=${rc}: $(tail -1 "$GL_TMP/m5.log")"; GL_FAILS=$((GL_FAILS+1)); fi

# 对照：合法配置通过检查，合成 PAT 照常检出
setup_case p1
# shellcheck disable=SC2086
gl_expect_block '对照 P1 目标版本的 .gitleaks.toml 通过检查' leak:github-pat:k.env $ARGS
setup_case p2 '[extend]' 'useDefault = true' 'disabledRules = ["generic-api-key"]'
# shellcheck disable=SC2086
gl_expect_block '对照 P2 useDefault + disabledRules 通过检查' leak:github-pat:k.env $ARGS
gl_finish
