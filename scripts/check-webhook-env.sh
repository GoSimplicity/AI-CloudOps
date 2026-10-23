#!/bin/sh
# 只读检查：webhook（cmd/webhook）升级到 sched/w1-mantou（0146077 起）前，配置文件 / 环境变量里的 webhook 字段是否合规。
# 只输出字段名和“合规 / 不合规 / 告警”及原因类别，从不输出任何取值；密钥字段（mysql.addr、WEBHOOK_MYSQL_ADDR、
# app_id / app_secret 等）的值在 awk 里直接丢弃、不进入 shell 变量。
#
# 用法: scripts/check-webhook-env.sh [--config <webhook.yaml>] [--env-file <.env>]
#   --config    要检查的 webhook 配置文件（与 cmd/webhook --config 相同）；不给则只检查环境变量 + 内置默认值。
#   --env-file  从 env 文件读取 WEBHOOK_* 的 5 个非密钥变量（不 source、不执行）；不给则读当前进程环境变量。
# 取值优先级与 loader 相同：环境变量（空串视为未设置）> 配置文件 webhook.* > 内置默认值。
# 检查项（比 loader 严格：loader 会静默改写的写法也算不合规）：
#   根节点        根节点下出现 webhook 字段（平铺，含大写 PORT）→ 不合规（loader 拒绝启动）；
#                 没有 webhook 根节点或其值为空 → 告警（loader 只告警、webhook.* 全取环境变量/默认值）
#   webhook.port  不带符号、不带前导零的十进制整数，1-65535（loader 会把 0080 当 80，这里算不合规）
#   fixed_workers / default_upgrade_minutes / common_map_renew_interval_seconds  十进制整数 >= 1
#   im_feishu.request_timeout_seconds  十进制整数 >= 0（0 = 不设超时，合规但生产不推荐，输出“告警”）
#   同一层重复的键（如 port 和 PORT 同写，键名不区分大小写）→ 不合规
# 退出码: 0 = 无不合规（可能有告警），1 = 有不合规，2 = 用法错误或文件不可读。
# 兼容 POSIX sh / bash --posix / macOS（bash 3.2、BSD awk）。
set -u
CFG=""; ENVF=""
while [ $# -gt 0 ]; do
    case "$1" in
        --config) [ $# -ge 2 ] || { echo "--config 需要参数" >&2; exit 2; }; CFG="$2"; shift 2 ;;
        --env-file) [ $# -ge 2 ] || { echo "--env-file 需要参数" >&2; exit 2; }; ENVF="$2"; shift 2 ;;
        -h|--help) sed -n '2,19p' "$0"; exit 0 ;;
        *) echo "未知参数: $1" >&2; exit 2 ;;
    esac
done
[ -z "$CFG" ] || [ -r "$CFG" ] || { echo "配置文件不可读: --config 指定的文件" >&2; exit 2; }
[ -z "$ENVF" ] || [ -r "$ENVF" ] || { echo "env 文件不可读: --env-file 指定的文件" >&2; exit 2; }

NG=0; WARN=0
say() { printf '%s: %s\n' "$1" "$2"; }

# ---------- 环境变量来源 ----------
# envval <变量名>：只允许下面 5 个非密钥变量
envval() {
    case "$1" in
        WEBHOOK_PORT|WEBHOOK_FIXED_WORKERS|WEBHOOK_DEFAULT_UPGRADE_MINUTES|WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS|WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS) ;;
        *) return 1 ;;
    esac
    if [ -n "$ENVF" ]; then
        # 只取 KEY=... 精确匹配的最后一行；去掉 CR、前缀 export、成对的外层引号
        K="$1" awk '{ sub(/\r$/, ""); sub(/^[ \t]*export[ \t]+/, "") }
            index($0, ENVIRON["K"] "=") == 1 { v = substr($0, length(ENVIRON["K"]) + 2); got = 1 }
            END { if (got) { if (v ~ /^".*"$/ || v ~ /^\047.*\047$/) v = substr(v, 2, length(v) - 2); printf "%s", v } }' "$ENVF"
    else
        eval "printf '%s' \"\${$1:-}\""
    fi
}

# ---------- 配置文件解析（只认本仓库 webhook.yaml 的块状写法）----------
# 输出（制表符分隔）：TOP <小写键> | ROOT <empty|block|inline> | WH <小写键> [<值>] | IM <小写键> [<值>]
# 只有 5 个检查字段输出值，其余键（含 mysql.addr、app_secret）只输出键名。
parse_cfg() {
    awk '
    function trim(s) { sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s); return s }
    function val(s,   q, i) {          # 去注释、去外层引号；带引号时保留引号内原样（含空格）
        s = trim(s)
        q = substr(s, 1, 1)
        if (q == "\"" || q == "\047") { i = index(substr(s, 2), q); if (i > 0) return substr(s, 2, i - 1); return substr(s, 2) }
        sub(/[ \t]+#.*$/, "", s); return trim(s)
    }
    function want(k) { return k == "port" || k == "fixed_workers" || k == "default_upgrade_minutes" || k == "common_map_renew_interval_seconds" || k == "request_timeout_seconds" }
    {
        sub(/\r$/, "")
        if ($0 ~ /^[ \t]*(#|$)/) next
        if ($0 ~ /^---/) next
        match($0, /^ */); ind = RLENGTH
        line = substr($0, ind + 1)
        c = index(line, ":"); if (c == 0) next
        k = tolower(trim(substr(line, 1, c - 1))); gsub(/^["\047]|["\047]$/, "", k)
        rest = substr(line, c + 1)
        if (ind == 0) {
            print "TOP\t" k
            inwh = 0; inim = 0
            if (k == "webhook") {
                v = val(rest)
                if (v == "" || v == "~" || v == "null" || v == "{}") { inwh = 1; wi = -1; print "ROOT\tempty" }
                else print "ROOT\tinline"
            }
            next
        }
        if (!inwh) next
        if (wi < 0) { wi = ind; print "ROOT\tblock" }
        if (ind == wi) {
            inim = (k == "im_feishu"); ii = -1
            if (want(k) && k != "request_timeout_seconds") print "WH\t" k "\t" val(rest); else print "WH\t" k
            next
        }
        if (ind > wi && inim) {
            if (ii < 0) ii = ind
            if (ind != ii) next
            if (k == "request_timeout_seconds") print "IM\t" k "\t" val(rest); else print "IM\t" k
        }
    }' "$1"
}

TAB="$(printf '\t')"
P="${TMPDIR:-/tmp}/check-webhook-env.$$"
trap 'rm -f "$P"' EXIT
trap 'rm -f "$P"; exit 2' HUP INT TERM
: >"$P"
if [ -n "$CFG" ]; then parse_cfg "$CFG" >"$P" || { echo "配置文件解析失败" >&2; exit 2; }; fi

# ---------- 根节点 ----------
if [ -n "$CFG" ]; then
    flat=""
    for k in port fixed_workers front_domain backend_domain default_upgrade_minutes alert_manager_api common_map_renew_interval_seconds im_feishu; do
        if grep -q "^TOP${TAB}${k}\$" "$P"; then flat="${flat}${flat:+, }${k}"; fi
    done
    nwh="$(grep -c "^TOP${TAB}webhook\$" "$P")"
    if [ -n "$flat" ]; then say "配置文件根节点" "不合规（根节点下有 webhook 字段 ${flat}，loader 拒绝启动；移到 webhook: 下）"; NG=$((NG+1))
    elif [ "$nwh" -gt 1 ]; then say "配置文件根节点" "不合规（webhook 根节点重复）"; NG=$((NG+1))
    elif grep -q "^ROOT${TAB}inline\$" "$P"; then say "配置文件根节点" "不合规（webhook: 写成了单行值，本脚本只支持块状写法，请人工检查）"; NG=$((NG+1))
    elif grep -q "^ROOT${TAB}block\$" "$P"; then say "配置文件根节点" "合规"
    else say "配置文件根节点" "告警（没有 webhook 根节点或其值为空：loader 只告警，webhook.* 全部取环境变量/默认值）"; WARN=$((WARN+1)); fi
else
    say "配置文件根节点" "未检查（没有 --config）"
fi

# ---------- 字段 ----------
# check <显示名> <环境变量名> <解析行前缀 WH|IM> <yaml 键> <下限> <上限|-> <默认值>
check() {
    name="$1"; ev="$2"; kind="$3"; yk="$4"; lo="$5"; hi="$6"; def="$7"
    n="$(grep -E -c "^${kind}${TAB}${yk}(${TAB}|\$)" "$P")"
    if [ "$n" -gt 1 ]; then say "$name" "不合规（配置文件同一层重复写了 ${yk}，键名不区分大小写，结果未定义）"; NG=$((NG+1)); return; fi
    v="$(envval "$ev")"; src="环境变量 ${ev}"
    if [ -z "$v" ]; then
        if [ "$n" -eq 1 ]; then
            line="$(grep -E "^${kind}${TAB}${yk}(${TAB}|\$)" "$P")"
            v="${line#"${kind}${TAB}${yk}"}"; v="${v#"${TAB}"}"; src="配置文件"
            if [ -z "$v" ]; then say "$name" "不合规（配置文件里是空值，删掉该键或写整数）"; NG=$((NG+1)); return; fi
        else
            v="$def"; src="内置默认值"
        fi
    fi
    case "$v" in
        0) ;;
        *[!0-9]*) say "$name" "不合规（${src}：不是十进制整数，含符号、小数点、空格、字母或进制前缀）"; NG=$((NG+1)); return ;;
        0*) say "$name" "不合规（${src}：有前导零）"; NG=$((NG+1)); return ;;
    esac
    if [ "${#v}" -gt 9 ]; then say "$name" "不合规（${src}：数值过大）"; NG=$((NG+1)); return; fi
    if [ "$v" -lt "$lo" ]; then say "$name" "不合规（${src}：小于 ${lo}）"; NG=$((NG+1)); return; fi
    if [ "$hi" != "-" ] && [ "$v" -gt "$hi" ]; then say "$name" "不合规（${src}：大于 ${hi}）"; NG=$((NG+1)); return; fi
    if [ "$yk" = request_timeout_seconds ] && [ "$v" -eq 0 ]; then say "$name" "告警（${src}：0 表示不设超时，生产不推荐）"; WARN=$((WARN+1)); return; fi
    say "$name" "合规（${src}）"
}
check webhook.port WEBHOOK_PORT WH port 1 65535 8889
check webhook.fixed_workers WEBHOOK_FIXED_WORKERS WH fixed_workers 1 - 10
check webhook.default_upgrade_minutes WEBHOOK_DEFAULT_UPGRADE_MINUTES WH default_upgrade_minutes 1 - 60
check webhook.common_map_renew_interval_seconds WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS WH common_map_renew_interval_seconds 1 - 300
check webhook.im_feishu.request_timeout_seconds WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS IM request_timeout_seconds 0 - 10

if [ "$NG" -gt 0 ]; then echo "结果: 不合规 ${NG} 项，告警 ${WARN} 项"; exit 1; fi
echo "结果: 合规（告警 ${WARN} 项）"
exit 0
