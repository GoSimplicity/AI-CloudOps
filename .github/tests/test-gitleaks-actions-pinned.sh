#!/usr/bin/env bash
# SEC-01A 审查 P3：gitleaks 工作流中所有 action 必须固定到完整 commit SHA（注释写版本号），
# actions/checkout 必须设置 persist-credentials: false（不把 GITHUB_TOKEN 留在 .git/config 里）。
# 用法: .github/tests/test-gitleaks-actions-pinned.sh [--rev <git-rev>] [--verify-remote]
#   --verify-remote  额外用 git ls-remote 核对注释中的版本 tag 确实指向该 SHA（需要网络）
# grep/sed 只用 POSIX 字符类 [[:space:]]：BSD grep/sed（macOS）不认识反斜杠 s 简写。
VERIFY_REMOTE=0; ARGS=()
for a in "$@"; do if [ "$a" = --verify-remote ]; then VERIFY_REMOTE=1; else ARGS+=("$a"); fi; done
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "${ARGS[@]+"${ARGS[@]}"}"

gl_src "$GL_WORKFLOW" >"$GL_TMP/wf.yml"
while IFS= read -r line; do
    spec="$(sed -E 's/^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*([^ #]+).*/\1/' <<<"$line")"
    action="${spec%@*}"; ref="${spec#*@}"; tag="$(grep -oE '#[[:space:]]*v[0-9][^ ]*' <<<"$line" | tr -d '# ' || true)"
    if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
        echo "PASS [uses] ${action}@${ref} ${tag:+(${tag})}"
        if [ "$VERIFY_REMOTE" = 1 ]; then
            [ -n "$tag" ] || { echo "FAIL [verify] ${action} 缺少版本注释"; GL_FAILS=$((GL_FAILS+1)); continue; }
            remote="$(git ls-remote "https://github.com/${action}" "refs/tags/${tag}" "refs/tags/${tag}^{}" | awk '{print $1}' | tail -1)"
            if [ "$remote" = "$ref" ]; then echo "PASS [verify] ${action} ${tag} -> ${remote}"
            else echo "FAIL [verify] ${action} ${tag} -> ${remote:-<不存在>}，与 ${ref} 不符"; GL_FAILS=$((GL_FAILS+1)); fi
        fi
    else
        echo "FAIL [uses] ${action}@${ref} 未固定到完整 commit SHA"; GL_FAILS=$((GL_FAILS+1))
    fi
done < <(grep -E '^[[:space:]]*-?[[:space:]]*uses:' "$GL_TMP/wf.yml")

# actions/checkout 步骤的 with: 中需要 persist-credentials: false
if python3 - "$GL_TMP/wf.yml" <<'PY'; then echo "PASS [checkout] persist-credentials: false"; else echo "FAIL [checkout] 未设置 persist-credentials: false"; GL_FAILS=$((GL_FAILS+1)); fi
import re, sys
lines = open(sys.argv[1], encoding="utf-8").read().splitlines()
for i, l in enumerate(lines):
    if re.match(r"\s*-?\s*uses:\s*actions/checkout@", l):
        ind = len(l) - len(l.lstrip(" -"))
        for m in lines[i + 1:]:
            if re.match(r"\s*-\s", m) and len(m) - len(m.lstrip(" -")) <= ind: break
            if re.match(r"\s*persist-credentials:\s*false\s*(#.*)?$", m): sys.exit(0)
        sys.exit(1)
sys.exit(1)
PY
gl_finish
