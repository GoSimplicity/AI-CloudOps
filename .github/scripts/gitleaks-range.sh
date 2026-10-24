#!/usr/bin/env bash
# 计算 CI 中 gitleaks 要扫描的提交范围（SEC-01A 审查 P2-3），供 .github/workflows/gitleaks.yml 调用。
# stdout 只输出一个 git log 范围参数；说明信息写 stderr；无法确定完整范围时 exit 1（宁可失败，不可漏扫）。
#
# 输入（环境变量，均来自 GitHub 事件上下文）：
#   EVENT_NAME      github.event_name
#   PUSH_BEFORE     github.event.before（push）
#   PR_BASE_SHA     github.event.pull_request.base.sha（pull_request）
#   HEAD_SHA        github.sha
#   DEFAULT_BRANCH  github.event.repository.default_branch
#   REF_NAME        github.ref_name
#   REF_TYPE        github.ref_type（非 PR 事件必填，只接受 branch / tag；空值、Tag 等其他写法一律失败，review3h P3-i）
#   REMOTE          远端名，默认 origin（actions/checkout 使用 origin，且需 fetch-depth: 0）
#
# 规则：
#   pull_request              -> <base.sha>..<HEAD>；base 不在历史中则失败
#   push，before 在历史中     -> <before>..<HEAD>（force-push 但旧提交仍可达时同样适用）
#   push，before 为全零或不在历史中（首推 / force-push 后旧提交不可达）：
#     - 推的是默认分支本身   -> 全量扫描 HEAD 可达的全部历史（无更早的基线可比较）
#     - 其他分支 / tag        -> <merge-base origin/<default> HEAD>..<HEAD>
#     - 缺少默认分支信息、没有 origin/<default>、无共同祖先、浅克隆 -> 失败
# 注意：本脚本只决定“扫描哪些提交”。范围起点（如 push 的 before）不是可信的策略来源——
#   gitleaks 配置 / 忽略列表由 gitleaks-base-config.sh 另行从默认分支选取（review3g N2）。
set -euo pipefail

ZERO=0000000000000000000000000000000000000000
REMOTE="${REMOTE:-origin}"
die()  { echo "[gitleaks-range] ERROR: $*" >&2; exit 1; }
note() { echo "[gitleaks-range] $*" >&2; }
is_commit() { [ -n "${1:-}" ] && [ "$1" != "$ZERO" ] && git cat-file -e "$1^{commit}" 2>/dev/null; }

[ -n "${HEAD_SHA:-}" ] || die "HEAD_SHA 为空"
is_commit "$HEAD_SHA" || die "HEAD_SHA=$HEAD_SHA 不在本地历史中"
[ "$(git rev-parse --is-shallow-repository)" = "false" ] || die "浅克隆无法确定完整范围（actions/checkout 需设置 fetch-depth: 0）"

case "${EVENT_NAME:-}" in
    pull_request|pull_request_target)
        is_commit "${PR_BASE_SHA:-}" || die "PR base sha（${PR_BASE_SHA:-<空>}）不在历史中，无法确定扫描范围"
        echo "${PR_BASE_SHA}..${HEAD_SHA}"
        exit 0 ;;
esac

# review3h P3-i：非 PR 事件按 push 处理，REF_TYPE 只接受 branch / tag（区分大小写）；
# 以前的 ${REF_TYPE:-branch} 会把空值当 branch，把 "Tag" 之类的写法当成分支，现在一律失败
case "${REF_TYPE:-}" in
    branch|tag) ;;
    *) die "REF_TYPE=${REF_TYPE:-<空>} 无效（只接受 branch / tag），无法确定扫描范围" ;;
esac

if is_commit "${PUSH_BEFORE:-}"; then
    echo "${PUSH_BEFORE}..${HEAD_SHA}"
    exit 0
fi
note "before=${PUSH_BEFORE:-<空>} 为全零或不在历史中（首推或 force-push），改用默认分支确定范围"

[ -n "${DEFAULT_BRANCH:-}" ] || die "缺少默认分支信息（DEFAULT_BRANCH 为空），无法确定扫描范围"

if [ "$REF_TYPE" = "branch" ] && [ "${REF_NAME:-}" = "$DEFAULT_BRANCH" ]; then
    note "推送的是默认分支 ${DEFAULT_BRANCH} 本身且无可用 before：全量扫描 HEAD 可达的全部历史"
    echo "${HEAD_SHA}"
    exit 0
fi

default_ref="refs/remotes/${REMOTE}/${DEFAULT_BRANCH}"
git rev-parse --verify -q "${default_ref}^{commit}" >/dev/null \
    || die "缺少 ${REMOTE}/${DEFAULT_BRANCH}，无法确定扫描范围（actions/checkout 需设置 fetch-depth: 0）"
base="$(git merge-base "$default_ref" "$HEAD_SHA" 2>/dev/null)" \
    || die "${HEAD_SHA} 与 ${REMOTE}/${DEFAULT_BRANCH} 没有共同祖先，无法确定扫描范围"
[ "$base" = "$HEAD_SHA" ] && note "HEAD 已包含在 ${REMOTE}/${DEFAULT_BRANCH} 中，没有新增提交"
echo "${base}..${HEAD_SHA}"
