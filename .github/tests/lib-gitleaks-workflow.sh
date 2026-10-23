#!/usr/bin/env bash
# .github/workflows/gitleaks.yml 回归测试的公共函数（SEC-01A 审查 P2/P3）。由同目录的 test-*.sh source。
# 所有场景在 mktemp -d 的临时仓库中运行；密钥为运行时拼出的合成值，不是真实凭据。
# 兼容 macOS 自带 bash 3.2 与 BSD 工具：不用 \s / grep -P / sed -i / readlink -f / mapfile。
set -euo pipefail

GL_REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GL_REV=""
GL_WORKFLOW=".github/workflows/gitleaks.yml"
GL_STEP="scan"
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
gl_fake_generic() { printf '%s' "$(gl_rand 'A-Za-z0-9' 32)"; }

# 从工作流中提取指定 step 的 run 内容（"run: |" 块或单行 "run: <cmd>"；按缩进解析，不依赖 PyYAML）
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
    m = re.match(r"\s*run:\s*(\S.*)$", lines[k])
    if m:
        print(m.group(1)); sys.exit(0)
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
    mkdir -p "$(dirname "$GL_DEV/$1")"
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
        else cp -R "$GL_REPO/.github/scripts" "$GL_WS/.github/"; fi
    fi
}

# 选择被判定的步骤：scan = "Scan new commits"；tree = 先跑 "Scan new commits"（准备 base 配置，结果不计），
# 再跑并判定 "Scan working tree (HEAD)"
gl_step() { GL_STEP="$1"; }

# 运行目标版本的步骤（与 GitHub 的 bash --noprofile --norc -eo pipefail 一致）；每次使用新的 RUNNER_TEMP
# 用法: gl_run_scan EVENT_NAME=push PUSH_BEFORE=... HEAD_SHA=... DEFAULT_BRANCH=main REF_NAME=main
gl_run_scan() {
    local f="$GL_TMP/scan-step.sh" t="$GL_TMP/tree-step.sh" rc=0
    GL_RT="$GL_TMP/rt-$RANDOM$RANDOM"; mkdir -p "$GL_RT"; rm -f "$GL_TMP/scan-commits.out"
    gl_extract_run "$f" "Scan new commits"
    set +e
    ( cd "$GL_WS" && env -u PR_BASE_SHA -u GITLEAKS_CONFIG -u GITLEAKS_CONFIG_TOML RUNNER_TEMP="$GL_RT" \
        EVENT_NAME= PUSH_BEFORE= PR_BASE_SHA= HEAD_SHA= DEFAULT_BRANCH= REF_NAME= REF_TYPE= "$@" \
        bash --noprofile --norc -eo pipefail "$f" ) >"$GL_TMP/scan.out" 2>&1
    rc=$?
    if [ "$GL_STEP" = "tree" ]; then
        mv "$GL_TMP/scan.out" "$GL_TMP/scan-commits.out"; rm -f "$GL_RT"/gitleaks-commits*.json
        gl_extract_run "$t" "Scan working tree (HEAD)"
        ( cd "$GL_WS" && env -u GITLEAKS_CONFIG -u GITLEAKS_CONFIG_TOML RUNNER_TEMP="$GL_RT" "$@" \
            bash --noprofile --norc -eo pipefail "$t" ) >"$GL_TMP/scan.out" 2>&1
        rc=$?
    fi
    set -e
    return "$rc"
}

# 列出本次运行检出的发现（每行 "RuleID<TAB>File"）：优先读步骤写出的 JSON 报告（$RUNNER_TEMP/gitleaks-*.json），
# 目标版本没有报告文件时（旧版本工作流）回退解析 --verbose 输出中的 RuleID/File 字段
gl_findings() {
    local reports; reports="$(ls "$GL_RT"/gitleaks-*.json 2>/dev/null || true)"
    if [ -n "$reports" ]; then
        python3 -c '
import json, sys
for p in sys.argv[1:]:
    for f in json.load(open(p)) or []:
        print("%s\t%s" % (f.get("RuleID", ""), f.get("File", "")))
' $reports
    else
        awk '$1 == "RuleID:" { r = $2 } $1 == "File:" { print r "\t" $2 }' "$GL_TMP/scan.out"
    fi
}

# 期望：步骤失败，且失败原因与预期一致
#   gl_expect_block <名称> leak:<RuleID>:<File> <gl_run_scan 参数...>   检出密钥（报告中须有该 RuleID+File）
#   gl_expect_block <名称> error:<ERE>           <gl_run_scan 参数...>   因错误失败（无任何检出，输出匹配 ERE）
gl_expect_block() {
    local name="$1" want="$2" rc=0; shift 2
    gl_run_scan "$@" || rc=$?
    local range; range="$(cat "$GL_TMP/scan-commits.out" "$GL_TMP/scan.out" 2>/dev/null | grep -m1 'gitleaks scan range' || true)"
    local found; found="$(gl_findings | sort -u)"
    local summary; summary="$(printf '%s' "$found" | tr '\t\n' ': ' )"
    local err; err="$(grep -m1 -E 'ERROR|ERR |无法|FTL' "$GL_TMP/scan.out" || true)"
    if [ "$rc" -eq 0 ]; then
        echo "FAIL [$name] 漏扫：步骤 rc=0，未拦下 | ${range:-<无范围输出>}"; GL_FAILS=$((GL_FAILS+1)); return 0
    fi
    case "$want" in
        leak:*)
            local rule file; rule="${want#leak:}"; file="${rule#*:}"; rule="${rule%%:*}"
            if printf '%s\n' "$found" | grep -qxF "$(printf '%s\t%s' "$rule" "$file")"; then
                echo "PASS [$name] 检出 ${rule} @ ${file}（rc=${rc}；全部发现: ${summary}）${range:+ | $range}"
            elif [ -z "$found" ]; then
                echo "FAIL [$name] 步骤失败（rc=${rc}）但不是检出密钥，而是出错: ${err:-<无错误信息>}"; GL_FAILS=$((GL_FAILS+1))
            else
                echo "FAIL [$name] 检出内容与预期不符（期望 ${rule} @ ${file}，实际: ${summary}）"; GL_FAILS=$((GL_FAILS+1))
            fi ;;
        error:*)
            local pat="${want#error:}"
            if [ -n "$found" ]; then
                echo "FAIL [$name] 期望因错误失败，实际检出密钥: ${summary}"; GL_FAILS=$((GL_FAILS+1))
            elif grep -qE "$pat" "$GL_TMP/scan.out"; then
                echo "PASS [$name] 按预期出错失败（rc=${rc}）: $(grep -m1 -E "$pat" "$GL_TMP/scan.out")"
            else
                echo "FAIL [$name] 失败原因不符（期望匹配 /${pat}/）: ${err:-$(tail -1 "$GL_TMP/scan.out")}"; GL_FAILS=$((GL_FAILS+1))
            fi ;;
        *) echo "测试用例写法错误: 期望值须为 leak:<RuleID>:<File> 或 error:<ERE>（${want}）" >&2; exit 2 ;;
    esac
}
gl_expect_pass() { # 对照：无密钥时步骤应成功
    local name="$1" rc=0; shift
    gl_run_scan "$@" || rc=$?
    if [ "$rc" -eq 0 ]; then echo "PASS [$name] 无密钥，步骤成功（rc=0）"
    else echo "FAIL [$name] 无密钥但步骤失败（rc=${rc}）"; tail -5 "$GL_TMP/scan.out" | sed 's/^/      /'; GL_FAILS=$((GL_FAILS+1)); fi
}
gl_finish() { echo "目标版本: ${GL_REV:-工作区}；失败 $GL_FAILS 项"; [ "$GL_FAILS" -eq 0 ]; }
