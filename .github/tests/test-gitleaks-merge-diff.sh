#!/usr/bin/env bash
# SEC-01A 审查 P2-1：merge commit 自身引入的密钥（冲突解决 / evil merge）必须被 CI 扫描检出。
# 用法: .github/tests/test-gitleaks-merge-diff.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"

# 冲突合并：main 与 feat 改同一行，密钥只出现在 merge 提交的冲突解决结果里
gl_new_origin merge
gl_commit app.conf 'mode = base' init
git -C "$GL_DEV" push -q origin HEAD:main
BEFORE="$(git -C "$GL_DEV" rev-parse HEAD)"
git -C "$GL_DEV" checkout -qb feat; gl_commit app.conf 'mode = feat' feat
git -C "$GL_DEV" checkout -q main;  gl_commit app.conf 'mode = main' main
git -C "$GL_DEV" merge -q feat >/dev/null 2>&1 || true
printf 'mode = merged\ngithub_token = "%s"\n' "$(gl_fake_ghp)" >"$GL_DEV/app.conf"
git -C "$GL_DEV" add app.conf; git -C "$GL_DEV" commit -qm "merge feat"
git -C "$GL_DEV" push -q origin main feat
HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"
gl_checkout "$HEAD_SHA"
gl_expect_block "push 到 main，密钥只在冲突合并提交中" EVENT_NAME=push PUSH_BEFORE="$BEFORE" HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=main

# 无冲突的 evil merge：合并时额外加入一个含 AWS key 的文件
gl_new_origin evil
gl_commit a.txt a init; git -C "$GL_DEV" push -q origin HEAD:main
BEFORE="$(git -C "$GL_DEV" rev-parse HEAD)"
git -C "$GL_DEV" checkout -qb feat; gl_commit b.txt b feat
git -C "$GL_DEV" checkout -q main
git -C "$GL_DEV" merge -q --no-ff --no-commit feat >/dev/null 2>&1
printf 'aws_access_key_id = %s\n' "$(gl_fake_aws)" >"$GL_DEV/creds.ini"
git -C "$GL_DEV" add creds.ini; git -C "$GL_DEV" commit -qm "merge feat (evil)"
git -C "$GL_DEV" push -q origin main feat
HEAD_SHA="$(git -C "$GL_DEV" rev-parse HEAD)"; FEAT_SHA="$(git -C "$GL_DEV" rev-parse feat)"
gl_checkout "$HEAD_SHA"
gl_expect_block "push 到 main，无冲突 evil merge 加入密钥文件" EVENT_NAME=push PUSH_BEFORE="$BEFORE" HEAD_SHA="$HEAD_SHA" DEFAULT_BRANCH=main REF_NAME=main
gl_expect_pass "对照：PR 事件 base..feat，无密钥" EVENT_NAME=pull_request PR_BASE_SHA="$BEFORE" HEAD_SHA="$FEAT_SHA" DEFAULT_BRANCH=main REF_NAME=1/merge
gl_finish
