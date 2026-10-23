#!/usr/bin/env bash
# SEC-01A 审查 P2-3：首推 / force-push 时扫描范围必须覆盖所有新提交（不能退化成只扫 HEAD 一个提交），
# 无法确定完整范围时必须失败。执行目标版本工作流中 "Scan new commits" 步骤本身（含其调用的 .github/scripts/）。
# 用法: .github/tests/test-gitleaks-range.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
# 每个拦截用例都声明期望的失败原因：leak:<RuleID>:<File>（检出）或 error:<ERE>（无法确定范围而出错）。
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"
ZERO=0000000000000000000000000000000000000000

# 远端已有 main（2 个干净提交）；feature 分支 3 个新提交，密钥只在第 1 个（非 HEAD）提交中
setup_feature() {
    gl_new_origin "$1"
    gl_commit README.md 'hello' init; gl_commit README.md 'hello 2' main-2
    git -C "$GL_DEV" push -q origin HEAD:main
    git -C "$GL_DEV" checkout -qb feature
    gl_commit deploy.env "GITHUB_TOKEN=$(gl_fake_ghp)" "feat-1 (secret)"
    gl_commit deploy.env 'GITHUB_TOKEN=' "feat-2 remove secret"
    gl_commit notes.txt 'done' "feat-3"
    git -C "$GL_DEV" push -q origin feature
    HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"
}

# R1 新分支首推（before 全零）
setup_feature r1; gl_checkout "$HEAD_SHA"
gl_expect_block "R1 首推 feature（3 个提交，密钥在非 HEAD 提交）" leak:github-pat:deploy.env \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch

# R2 force-push（before 是被覆盖的旧提交，checkout 后不在历史中）
setup_feature r2; gl_checkout "$HEAD_SHA"
gl_expect_block "R2 force-push feature（before 不在历史中）" leak:github-pat:deploy.env \
    EVENT_NAME=push PUSH_BEFORE="$(gl_rand 'a-f0-9' 40)" HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch

# R3 无法确定范围：缺少默认分支信息 / 没有 origin/main / 与默认分支无共同祖先
setup_feature r3; gl_checkout "$HEAD_SHA"
gl_expect_block "R3a 首推但 DEFAULT_BRANCH 为空" 'error:DEFAULT_BRANCH 为空' \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH= REF_NAME=feature REF_TYPE=branch
git -C "$GL_WS" update-ref -d refs/remotes/origin/main
gl_expect_block "R3b 首推但缺少 origin/main" 'error:缺少 origin/main' \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch
gl_new_origin r3c
gl_commit a.txt a init; git -C "$GL_DEV" push -q origin HEAD:main
git -C "$GL_DEV" checkout -q --orphan orphan; git -C "$GL_DEV" rm -rqf .
gl_commit .gitleaks.toml '' o-1; gl_commit k.env "AWS_KEY=$(gl_fake_aws)" "o-2 (secret)"; gl_commit k.env 'AWS_KEY=' o-3
git -C "$GL_DEV" push -q origin orphan; HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"; gl_checkout "$HEAD_SHA"
gl_expect_block "R3c 首推与 main 无共同祖先的分支" 'error:没有共同祖先' \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=orphan REF_TYPE=branch

# R4 新仓库首推默认分支本身：密钥在非 HEAD 提交 -> 全量扫描
gl_new_origin r4
gl_commit a.txt a init; gl_commit k.env "GITHUB_TOKEN=$(gl_fake_ghp)" "secret"; gl_commit k.env 'GITHUB_TOKEN=' clean
git -C "$GL_DEV" push -q origin HEAD:main; HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"; gl_checkout "$HEAD_SHA"
gl_expect_block "R4 首推默认分支 main 本身（全量扫描）" leak:github-pat:k.env \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=main REF_TYPE=branch

# R5 PR 事件缺少/无效 base sha
setup_feature r5; gl_checkout "$HEAD_SHA"
gl_expect_block "R5 pull_request 的 base sha 不在历史中" 'error:PR base sha' \
    EVENT_NAME=pull_request PR_BASE_SHA="$(gl_rand 'a-f0-9' 40)" PUSH_BEFORE= HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=1/merge

# 对照：正常 push（before 有效）仍检出；干净的首推 / 首推默认分支不误报
setup_feature c1; BEFORE="$(git -C "$GL_DEV" rev-parse main)"; gl_checkout "$HEAD_SHA"
gl_expect_block "对照 正常 push before..HEAD 含密钥" leak:github-pat:deploy.env \
    EVENT_NAME=push PUSH_BEFORE="$BEFORE" HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=feature REF_TYPE=branch
gl_new_origin c2
gl_commit README.md 'hello' init; git -C "$GL_DEV" push -q origin HEAD:main
git -C "$GL_DEV" checkout -qb clean; gl_commit a.txt a c-1; gl_commit b.txt b c-2
git -C "$GL_DEV" push -q origin clean; HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"; gl_checkout "$HEAD_SHA"
gl_expect_pass "对照 干净 feature 首推" \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=clean REF_TYPE=branch
gl_expect_pass "对照 HEAD 已在 main 上的新分支（空范围）" \
    EVENT_NAME=push PUSH_BEFORE=$ZERO HEAD_SHA="$(git -C "$GL_DEV" rev-parse main)" DEFAULT_BRANCH=main REF_NAME=copy REF_TYPE=branch
gl_finish
