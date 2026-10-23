#!/usr/bin/env bash
# SEC-01A 审查 P2-2：push 事件不得被 concurrency 取消（否则被取消的 push 永远不会补扫），PR 事件可以取消。
# 解析工作流 concurrency.cancel-in-progress，并对 push / pull_request 两种事件求值
# （支持字面量 true/false 和 ${{ github.event_name == '...' }} / != 形式）。
# 另外：GitHub 对同一 concurrency group 只保留 1 个运行中 + 1 个排队中的运行，新排队的会取消旧的
# 排队运行（与 cancel-in-progress 无关）。因此 push 事件的 group 还必须每次运行唯一（含 github.run_id），
# 否则连续 3 次 push 时中间那次仍会被取消、其提交永不补扫。
# 用法: .github/tests/test-gitleaks-concurrency.sh [--rev <git-rev>]   （对修复前版本运行应报 FAIL）
source "$(dirname "${BASH_SOURCE[0]}")/lib-gitleaks-workflow.sh"
gl_parse_args "$@"

eval_cancel() { # eval_cancel <event_name> -> 打印 true/false
    gl_src "$GL_WORKFLOW" | python3 -c '
import re, sys
event = sys.argv[1]; text = sys.stdin.read()
m = re.search(r"^concurrency:\s*\n((?:[ \t]+.*\n?)+)", text, re.M)
if not m: print("false"); sys.exit(0)          # 无 concurrency：不取消
v = re.search(r"^\s+cancel-in-progress:\s*(.+?)\s*$", m.group(1), re.M)
if not v: print("false"); sys.exit(0)
val = v.group(1).strip().strip("\"\x27")
if val in ("true", "false"): print(val); sys.exit(0)
e = re.fullmatch(r"\$\{\{\s*github\.event_name\s*(==|!=)\s*\x27([^\x27]+)\x27\s*\}\}", val)
if not e: sys.exit("无法求值的表达式: " + val)
eq = (event == e.group(2)); print(str(eq if e.group(1) == "==" else not eq).lower())
' "$1"
}
expect() { # expect <event> <期望值>
    local got; got="$(eval_cancel "$1")"
    if [ "$got" = "$2" ]; then echo "PASS [$1] cancel-in-progress=$got"
    else echo "FAIL [$1] cancel-in-progress=$got（期望 $2）"; GL_FAILS=$((GL_FAILS+1)); fi
}
echo "concurrency 段: $(gl_src "$GL_WORKFLOW" | grep -A2 '^concurrency:' | tr -s ' ' | tr '\n' ' ')"
group_unique_for_push() {
    gl_src "$GL_WORKFLOW" | python3 -c '
import re, sys
text = sys.stdin.read()
m = re.search(r"^concurrency:\s*\n((?:[ \t]+.*\n?)+)", text, re.M)
if not m: print("true"); sys.exit(0)            # 无 concurrency：互不影响
g = re.search(r"^\s+group:\s*(.+?)\s*$", m.group(1), re.M).group(1)
# 接受 ... github.event_name == \x27pull_request\x27 && \x27<x>\x27 || github.run_id ... 形式
ok = re.search(r"github\.event_name\s*==\s*\x27pull_request\x27\s*&&\s*\x27[^\x27]*\x27\s*\|\|\s*github\.run_id", g)
print("true" if ok else "false")
'
}
expect push false
expect pull_request true
got="$(group_unique_for_push)"
if [ "$got" = true ]; then echo "PASS [push] concurrency group 每次运行唯一（含 github.run_id），排队中的 push 不会被取消"
else echo "FAIL [push] concurrency group 按 ref 共享：排队中的 push 运行会被后续 push 取消"; GL_FAILS=$((GL_FAILS+1)); fi
gl_finish
