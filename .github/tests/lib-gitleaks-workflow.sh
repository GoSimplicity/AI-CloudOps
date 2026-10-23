#!/usr/bin/env bash
# .github/workflows/gitleaks.yml 回归测试的公共函数（SEC-01A 审查 P2/P3）。由同目录的 test-*.sh source。
# 所有场景在 mktemp -d 的临时仓库中运行；密钥为运行时拼出的合成值，不是真实凭据。
set -euo pipefail

GL_REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GL_REV=""
GL_WORKFLOW=".github/workflows/gitleaks.yml"
gl_parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --rev) GL_REV="$2"; shift 2 ;;
            *) echo "未知参数: $1（用法: $0 [--rev <git-rev>]，默认测试当前工作区文件）" >&2; exit 2 ;;
        esac
    done
}
export PATH="$HOME/.local/bin:$PATH"
GL_TMP="$(mktemp -d)"
trap 'rm -rf "$GL_TMP"' EXIT
export GIT_CONFIG_GLOBAL="$GL_TMP/gitconfig" GIT_CONFIG_NOSYSTEM=1
git config --global user.name test
git config --global user.email test@example.invalid
git config --global init.defaultBranch main
git config --global advice.detachedHead false
GL_FAILS=0

# 读取目标版本（--rev 或工作区）中的文件
gl_src() { if [ -n "$GL_REV" ]; then git -C "$GL_REPO" show "$GL_REV:$1"; else cat "$GL_REPO/$1"; fi; }
gl_has() { if [ -n "$GL_REV" ]; then git -C "$GL_REPO" cat-file -e "$GL_REV:$1" 2>/dev/null; else [ -e "$GL_REPO/$1" ]; fi; }

# 合成密钥：运行时拼接，源码中不出现完整 token
gl_rand() { LC_ALL=C tr -dc "$1" </dev/urandom | head -c "$2"; }
gl_fake_ghp() { printf 'ghp_%s' "$(gl_rand 'A-Za-z0-9' 36)"; }
gl_fake_aws() { printf 'AKIA%s' "$(gl_rand 'A-Z2-7' 16)"; }

# 从工作流中提取指定 step 的 run 块（按缩进解析，不依赖 PyYAML）
gl_extract_run() {
    gl_src "$GL_WORKFLOW" | python3 -c '
import re, sys
name = sys.argv[1]; lines = sys.stdin.read().splitlines()
i = next(k for k, l in enumerate(lines) if re.match(r"\s*-?\s*name:\s*" + re.escape(name) + r"\s*$", l))
step_indent = len(lines[i]) - len(lines[i].lstrip(" -"))
for k in range(i + 1, len(lines)):
    m = re.match(r"(\s*)run:\s*\|\s*$", lines[k])
    if m:
        ind = len(m.group(1)); out = []
        for l in lines[k + 1:]:
            if l.strip() and len(l) - len(l.lstrip()) <= ind: break
            out.append(l)
        body_ind = min(len(l) - len(l.lstrip()) for l in out if l.strip())
        print("\n".join(l[body_ind:] for l in out)); sys.exit(0)
    if re.match(r"\s*-\s", lines[k]) and len(lines[k]) - len(lines[k].lstrip(" -")) <= step_indent: break
sys.exit("run block not found for step: " + name)
' "$2" >"$1"
}

# 初始化"远端"裸仓库 + 开发者仓库（含目标版本的 .gitleaks.toml）
gl_new_origin() {
    local name="$1"
    GL_ORIGIN="$GL_TMP/$name-origin.git"; GL_DEV="$GL_TMP/$name-dev"
    git init -q --bare "$GL_ORIGIN"
    git init -q "$GL_DEV"; git -C "$GL_DEV" remote add origin "$GL_ORIGIN"
    if gl_has .gitleaks.toml; then gl_src .gitleaks.toml >"$GL_DEV/.gitleaks.toml"; else : >"$GL_DEV/.gitleaks.toml"; fi
}
gl_commit() { # gl_commit <file> <content> <msg>
    printf '%s\n' "$2" >"$GL_DEV/$1"; git -C "$GL_DEV" add -A; git -C "$GL_DEV" commit -qm "$3"
}

# 模拟 actions/checkout(fetch-depth: 0)：fetch refs/heads/* 到 origin/*，不设置 origin/HEAD，detached 到 HEAD_SHA
gl_checkout() { # gl_checkout <head-sha> [extra refspec]
    GL_WS="$GL_TMP/ws-$RANDOM$RANDOM"; git init -q "$GL_WS"
    git -C "$GL_WS" remote add origin "$GL_ORIGIN"
    git -C "$GL_WS" fetch -q origin '+refs/heads/*:refs/remotes/origin/*' ${2:-}
    git -C "$GL_WS" checkout -q --detach "$1"
    # 目标版本的辅助脚本（如 .github/scripts/）放到工作区（未跟踪，不影响 gitleaks git 扫描的提交）
    if gl_has .github/scripts; then
        mkdir -p "$GL_WS/.github"
        if [ -n "$GL_REV" ]; then git -C "$GL_REPO" archive "$GL_REV" .github/scripts | tar -x -C "$GL_WS"
        else cp -r "$GL_REPO/.github/scripts" "$GL_WS/.github/"; fi
    fi
}

# 运行目标版本 "Scan new commits" 步骤（与 GitHub 的 bash --noprofile --norc -eo pipefail 一致）
# 用法: gl_run_scan EVENT_NAME=push PUSH_BEFORE=... HEAD_SHA=... DEFAULT_BRANCH=main REF_NAME=main
gl_run_scan() {
    local f="$GL_TMP/scan-step.sh"; gl_extract_run "$f" "Scan new commits"
    ( cd "$GL_WS" && env -u PR_BASE_SHA EVENT_NAME= PUSH_BEFORE= PR_BASE_SHA= HEAD_SHA= DEFAULT_BRANCH= REF_NAME= REF_TYPE= "$@" \
        bash --noprofile --norc -eo pipefail "$f" ) >"$GL_TMP/scan.out" 2>&1
}

# 期望：扫描步骤失败（检出密钥或无法确定范围而失败）
gl_expect_block() { # gl_expect_block <名称> <gl_run_scan 参数...>
    local name="$1" rc; shift
    set +e; gl_run_scan "$@"; rc=$?; set -e
    local range; range="$(grep -m1 'gitleaks scan range' "$GL_TMP/scan.out" || true)"
    local leaks; leaks="$(grep -Eo 'leaks found: [0-9]+' "$GL_TMP/scan.out" || true)"
    if [ "$rc" -ne 0 ]; then
        echo "PASS [$name] 被拦（rc=$rc ${leaks:-$(grep -m1 -E 'ERROR|无法' "$GL_TMP/scan.out" || true)}）${range:+ | $range}"
    else
        echo "FAIL [$name] 漏扫：步骤 rc=0，密钥未检出 | ${range:-<无范围输出>}"; GL_FAILS=$((GL_FAILS+1))
    fi
}
gl_expect_pass() { # 对照：无密钥时步骤应成功
    local name="$1" rc; shift
    set +e; gl_run_scan "$@"; rc=$?; set -e
    if [ "$rc" -eq 0 ]; then echo "PASS [$name] 无密钥，步骤成功（rc=0）"
    else echo "FAIL [$name] 无密钥但步骤失败（rc=$rc）"; tail -5 "$GL_TMP/scan.out" | sed 's/^/      /'; GL_FAILS=$((GL_FAILS+1)); fi
}
gl_finish() { echo "目标版本: ${GL_REV:-工作区}；失败 $GL_FAILS 项"; [ "$GL_FAILS" -eq 0 ]; }
