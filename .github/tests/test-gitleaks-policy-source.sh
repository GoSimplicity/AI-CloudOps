#!/usr/bin/env bash
# review3g N2：扫描起点 ≠ 策略来源。push 到非默认分支 / tag、PR 进非默认分支时，gitleaks 配置与忽略列表
# 一律取自 origin/<默认分支>，不能取自 push 的 before（同一分支上一次推送，攻击者可控）。
#
#   N2-1  feature 分支：先推全部放行的 .gitleaks.toml（无密钥，通过），再推合成 PAT（before=上一次推送）→ 必须检出
#         （范围扫描 + 工作树扫描各判一次）
#   N2-2  同上，但第一次推送的是带该文件指纹的 .gitleaksignore                                 → 必须检出
#   N2-3  PR 目标是非默认分支（其 tip 已改成全部放行），PR 中加入合成 PAT                         → 必须检出
#   N2-4  push tag（before 全零），tag 指向“全部放行配置 + 合成 PAT”的提交                        → 必须检出
#   对照 C1  origin/main 的配置按值放行某合成值 → feature 推送该值放行（证明策略确实取自 origin/main）
#   对照 C2  push 到 main：before 上的按值 allowlist 生效（默认分支仍用范围起点）
#   E1  push 到 feature 但本地没有 origin/main → 失败（不回退到 before）
#   E2  缺少 EVENT_NAME → 失败
#
# 用法: .github/tests/test-gitleaks-policy-source.sh [--rev <git-rev>]   （对修复前版本运行，N2-1/N2-2/N2-3/E1/E2 应报 FAIL）
# 合成密钥均在运行时随机生成；所有仓库都在临时目录中创建。
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"
ZERO=0000000000000000000000000000000000000000
ALLOW_ALL="$(printf '[extend]\nuseDefault = true\n\n[[allowlists]]\ndescription = "allow everything"\npaths = [%s]\n' "'''.*'''")"

main_base() { # main_base <name>：origin/main 上有目标版本的 .gitleaks.toml，本地切到 feature
    gl_new_origin "$1"; printf 'hello\n' >"$GL_DEV/README.md"
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; git -C "$GL_DEV" push -q origin HEAD:main
    MAIN_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"; git -C "$GL_DEV" checkout -q -b feature
}
push_feature() { # push_feature <msg>：提交工作区全部改动并推到 origin/feature，设置 H
    git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm "$1"; git -C "$GL_DEV" push -q origin HEAD:feature
    H="$(git -C "$GL_DEV" rev-parse HEAD)"
}

# N2-1：先推全部放行的配置，再推密钥
main_base n21; printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/.gitleaks.toml"; push_feature allow-all; H1="$H"
gl_checkout "$H1"; gl_step scan
gl_expect_pass "N2-1 前提：第一次推送只改配置、无密钥" EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA=$H1 DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch
printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; push_feature secret; H2="$H"
ARGS="EVENT_NAME=push PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch"
gl_checkout "$H2"
gl_step scan; gl_expect_block "N2-1 feature 先推全部放行配置再推合成 PAT（范围扫描）" leak:github-pat:k.env $ARGS
gl_step tree; gl_expect_block "N2-1 feature 先推全部放行配置再推合成 PAT（工作树扫描）" leak:github-pat:k.env $ARGS
gl_step scan

# N2-2：先推指纹忽略列表，再推密钥
main_base n22; printf 'k.env:github-pat:1\n' >"$GL_DEV/.gitleaksignore"; push_feature ignore; H1="$H"
printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; push_feature secret; H2="$H"
gl_checkout "$H2"
gl_expect_block "N2-2 feature 先推 .gitleaksignore 指纹再推合成 PAT" leak:github-pat:k.env \
    EVENT_NAME=push PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch

# N2-3：PR 目标为非默认分支 feature（tip 已全部放行）
main_base n23; printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/.gitleaks.toml"; push_feature allow-all; H1="$H"
git -C "$GL_DEV" checkout -q -b topic
printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"; git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm pr
git -C "$GL_DEV" push -q origin HEAD:topic; H2="$(git -C "$GL_DEV" rev-parse HEAD)"
gl_checkout "$H2"
gl_expect_block "N2-3 PR 进非默认分支（其 tip 全部放行）" leak:github-pat:k.env \
    EVENT_NAME=pull_request PR_BASE_SHA=$H1 PR_BASE_REF=feature HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=2/merge REF_TYPE=branch

# N2-4：tag 推送
main_base n24; printf '%s\n' "$ALLOW_ALL" >"$GL_DEV/.gitleaks.toml"; printf 'GITHUB_TOKEN=%s\n' "$(gl_fake_ghp)" >"$GL_DEV/k.env"
push_feature tagged; gl_checkout "$H"
gl_expect_block "N2-4 tag 推送（全部放行配置 + 合成 PAT）" leak:github-pat:k.env \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA=$H DEFAULT_BRANCH=main REF_NAME=v9.9.9 REF_TYPE=tag

# 对照 C1：origin/main 的按值 allowlist 对 feature 推送生效
tok="$(gl_fake_ghp)"
gl_new_origin cc1
printf '\n[[allowlists]]\ndescription = "test control"\nregexTarget = "secret"\nregexes = [%s]\n' "'''^${tok}\$'''" >>"$GL_DEV/.gitleaks.toml"
git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; git -C "$GL_DEV" push -q origin HEAD:main; git -C "$GL_DEV" checkout -q -b feature
printf 'hello\n' >"$GL_DEV/README.md"; push_feature one; H1="$H"
printf 'GITHUB_TOKEN=%s\n' "$tok" >"$GL_DEV/k.env"; push_feature two; H2="$H"
gl_checkout "$H2"
gl_expect_pass "对照 C1：origin/main 配置中的 allowlist 对 feature 推送生效" \
    EVENT_NAME=push PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch

# 对照 C2：push 到 main，before 上的按值 allowlist 生效
gl_new_origin cc2
printf '\n[[allowlists]]\ndescription = "test control"\nregexTarget = "secret"\nregexes = [%s]\n' "'''^${tok}\$'''" >>"$GL_DEV/.gitleaks.toml"
git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm base; git -C "$GL_DEV" push -q origin HEAD:main; H1="$(git -C "$GL_DEV" rev-parse HEAD)"
printf 'GITHUB_TOKEN=%s\n' "$tok" >"$GL_DEV/k.env"; git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm two
git -C "$GL_DEV" push -q origin HEAD:main; H2="$(git -C "$GL_DEV" rev-parse HEAD)"
gl_checkout "$H2"
gl_expect_pass "对照 C2：push 到 main 用 before 上的配置" EVENT_NAME=push PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch

# E1：需要 origin/main 但不存在
main_base e1; printf 'x\n' >"$GL_DEV/a.txt"; push_feature one; H1="$H"
printf 'y\n' >"$GL_DEV/a.txt"; push_feature two; H2="$H"
gl_checkout "$H2"; git -C "$GL_WS" update-ref -d refs/remotes/origin/main
gl_expect_block "E1 push 到 feature、缺少 origin/main 时失败（不回退到 before）" 'error:策略须取自' \
    EVENT_NAME=push PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch

# E2：缺少 EVENT_NAME
gl_checkout "$H2"
gl_expect_block "E2 缺少 EVENT_NAME 时失败" 'error:缺少 EVENT_NAME' \
    EVENT_NAME= PUSH_BEFORE=$H1 HEAD_SHA=$H2 DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch
gl_finish
