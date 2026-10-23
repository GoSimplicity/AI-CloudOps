#!/usr/bin/env bash
# SEC-01A 第三轮补修（review3b）：env.example 与端口 / 认证相关的说明必须和代码一致。
#   端口  PROMETHEUS_HTTP_SD_API、WEBHOOK_BACKEND_DOMAIN 指向主后端端口（SERVER_PORT = config 中 server.port）；
#         WEBHOOK_PORT 若出现，须留空或等于 config/webhook.yaml 的 webhook.port，注释分别说明合入前（不生效）/
#         合入 sched/w1-mantou 后（bb410eb 起生效）的行为与改端口方法；PROMETHEUS_ALERT_WEBHOOK_ADDR 指向 webhook 端口；
#         本人负责的范围（env.example、deploy/、docs/、scripts/、.github/、根目录 README* / compose / Makefile）
#         不得再以“值”的形式出现旧端口 8888（注释里解释历史默认值可以；登记的例外见 ALLOW_8888）。
#   JWT   JWT_EXPIRATION=30（单位分钟），注释写明单位与合入前/后默认值，并与 deploy/release/compose.yaml 一致；
#         本人负责的范围内提到 JWT_EXPIRATION 的每个文档都写明单位为分钟，且不再出现 JWT_EXPIRATION=3600。
#   合入   也可在合入 sched/w1-mantou 后的工作区运行：馒头新增的 w1-*.md 只登记；“8888 改为 8889”同行说明不算旧值。
#   Redis REDIS_PASSWORD 值为空且注释标明 compose 必填。
#
# 用法: scripts/tests/security/test-env-example.sh [--rev <git-rev>]   （不带 --rev 检查工作区；对修复前版本运行应报 FAIL）
# 只读：不修改任何文件；config/ 仅读取作对照。兼容 macOS（bash 3.2 / BSD grep、sed、awk）。
set -eu

REV=""
while [ $# -gt 0 ]; do
    case "$1" in
        --rev) [ $# -ge 2 ] || { echo "--rev 需要参数" >&2; exit 2; }; REV="$2"; shift 2 ;;
        -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
        *) echo "未知参数: $1" >&2; exit 2 ;;
    esac
done

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO="$(git -C "$HERE" rev-parse --show-toplevel)"
if [ -n "$REV" ]; then git -C "$REPO" rev-parse --verify -q "$REV^{commit}" >/dev/null || { echo "无效的 rev: $REV" >&2; exit 2; }; fi
TMP="$(mktemp -d "${TMPDIR:-/tmp}/test-env-example.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
FAILS=0
ok()  { echo "PASS [$1]"; }
bad() { echo "FAIL [$1]"; FAILS=$((FAILS+1)); }

# src <路径>：从 --rev 或工作区取文件内容
src() { if [ -n "$REV" ]; then git -C "$REPO" show "$REV:$1"; else cat "$REPO/$1"; fi; }
src env.example >"$TMP/env"
src config/config.development.yaml >"$TMP/dev.yaml"
src config/webhook.yaml >"$TMP/webhook.yaml"

# val <变量名>：env.example 中该变量的值（只看未注释的行）
val() { sed -n "s/^$1=//p" "$TMP/env" | tail -n 1; }
# block <变量名>：该变量赋值行之前紧挨着的连续注释行
block() { awk -v k="$1=" '/^#/{b=b $0 "\n"; next} index($0,k)==1{printf "%s", b; exit} {b=""}' "$TMP/env"; }

yaml_srv="$(awk '/^server:/{f=1;next} f&&/^[^ #]/{f=0} f&&/^  port:/{sub(/#.*/,"");gsub(/[^0-9]/,"");print;exit}' "$TMP/dev.yaml")"
yaml_wh="$(awk '/^webhook:/{f=1;next} f&&/^[^ #]/{f=0} f&&/^  port:/{sub(/#.*/,"");gsub(/[^0-9]/,"");print;exit}' "$TMP/webhook.yaml")"
srv="$(val SERVER_PORT)"
echo "目标: ${REV:-工作区}  SERVER_PORT=${srv}  config.development.yaml server.port=${yaml_srv}  config/webhook.yaml webhook.port=${yaml_wh}"

# ---------- 主后端端口 ----------
[ -n "$srv" ] && [ "$srv" = "$yaml_srv" ] && ok "SERVER_PORT=${srv} 与 config.development.yaml 一致" \
    || bad "SERVER_PORT=${srv}，config.development.yaml server.port=${yaml_srv}"
sd="$(val PROMETHEUS_HTTP_SD_API)"
case "$sd" in
    "http://localhost:${srv}/"*) ok "PROMETHEUS_HTTP_SD_API 指向主后端端口 ${srv}" ;;
    *) bad "PROMETHEUS_HTTP_SD_API=${sd} 未指向主后端端口 ${srv}" ;;
esac
bd="$(val WEBHOOK_BACKEND_DOMAIN)"
case "$bd" in
    "http://localhost:${srv}"|"http://localhost:${srv}/"*) ok "WEBHOOK_BACKEND_DOMAIN 指向主后端端口 ${srv}" ;;
    *) bad "WEBHOOK_BACKEND_DOMAIN=${bd} 未指向主后端端口 ${srv}" ;;
esac

# ---------- WEBHOOK_PORT / PROMETHEUS_ALERT_WEBHOOK_ADDR ----------
# 合入前 WEBHOOK_PORT 不生效、端口取自 config/webhook.yaml；合入 sched/w1-mantou（bb410eb 起）后 WEBHOOK_PORT > 文件 > 默认 8889
wh_eff="$yaml_wh"
if grep -q '^WEBHOOK_PORT=' "$TMP/env"; then
    wp="$(val WEBHOOK_PORT)"
    if [ -z "$wp" ] || [ "$wp" = "$yaml_wh" ]; then ok "WEBHOOK_PORT='${wp}'（留空或与 config/webhook.yaml 一致）"
    else bad "WEBHOOK_PORT=${wp}，config/webhook.yaml webhook.port=${yaml_wh}（合入前不生效、合入后会改变监听端口）"; wh_eff="$wp"; fi
    block WEBHOOK_PORT >"$TMP/wp.txt"
    for kw in '合入 sched/w1-mantou 前' '合入 sched/w1-mantou 后' 'config/webhook.yaml' '--config' '根节点' '同一台机器' '优先级' 'PROMETHEUS_ALERT_WEBHOOK_ADDR'; do
        if grep -F -q -e "$kw" "$TMP/wp.txt"; then ok "WEBHOOK_PORT 注释含「${kw}」"
        else bad "WEBHOOK_PORT 注释缺「${kw}」（需分别说明合入前/后、同机冲突、改端口方法与告警地址联动）"; fi
    done
    # 合入前那一段必须写明“不生效”（实测），合入后那一段不得再写“不生效/不改变”（bb410eb 已修复）
    awk '/合入 sched\/w1-mantou 前/{f=1} /合入 sched\/w1-mantou 后/{f=0} f' "$TMP/wp.txt" >"$TMP/wp-pre.txt"
    awk '/合入 sched\/w1-mantou 后/{f=1} /^# 主后端/{f=0} f' "$TMP/wp.txt" >"$TMP/wp-post.txt"
    grep -F -q '不生效' "$TMP/wp-pre.txt" && ok "WEBHOOK_PORT 合入前一段写明不生效" || bad "WEBHOOK_PORT 合入前一段未写明不生效"
    if grep -E -q '不生效|不改变|不受' "$TMP/wp-post.txt"; then bad "WEBHOOK_PORT 合入后一段仍写不生效（与 bb410eb 实测不符）"
    else ok "WEBHOOK_PORT 合入后一段与 bb410eb 行为一致（未写不生效）"; fi
else
    ok "env.example 未设置 WEBHOOK_PORT"
fi
if grep -q '^WEBHOOK_MYSQL_ADDR=' "$TMP/env"; then
    block WEBHOOK_MYSQL_ADDR >"$TMP/wm.txt"
    if grep -F -q '合入 sched/w1-mantou 前' "$TMP/wm.txt" && grep -F -q '合入 sched/w1-mantou 后' "$TMP/wm.txt" && grep -F -q '不读此变量' "$TMP/wm.txt"; then
        ok "WEBHOOK_MYSQL_ADDR 注释分别说明合入前（不读）/合入后（必填）"
    else bad "WEBHOOK_MYSQL_ADDR 注释未区分合入前（webhook 不读此变量）与合入后（必填）"; fi
fi
aw="$(val PROMETHEUS_ALERT_WEBHOOK_ADDR)"
case "$aw" in
    "http://localhost:${wh_eff}/api/v1/alerts/receive") ok "PROMETHEUS_ALERT_WEBHOOK_ADDR 指向 webhook 端口 ${wh_eff}" ;;
    *) bad "PROMETHEUS_ALERT_WEBHOOK_ADDR=${aw} 未指向 webhook 端口 ${wh_eff}" ;;
esac
block PROMETHEUS_ALERT_WEBHOOK_ADDR >"$TMP/aw.txt"
grep -F -q 'WEBHOOK_PORT' "$TMP/aw.txt" && ok "PROMETHEUS_ALERT_WEBHOOK_ADDR 注释说明与 WEBHOOK_PORT 联动" \
    || bad "PROMETHEUS_ALERT_WEBHOOK_ADDR 缺注释：指向 webhook 进程、端口须与 WEBHOOK_PORT 一致"

# ---------- 旧端口 8888 ----------
# 登记的例外（不属本人范围或另有归属，见 README-ops）：
#   docs/release/prometheus-scrape-auth.md —— 该行在 sched/w1-mantou 中已由馒头改写，合并时以其版本为准
#   MANTOU_DOCS —— 馒头 sched/w1-mantou 新增的发布说明（合入后才出现），其中引用旧 env.example 的 8888/3600 作为
#     迁移说明，归馒头，只登记不检查
ALLOW_8888='docs/release/prometheus-scrape-auth.md'
MANTOU_DOCS='docs/release/w1-jwt-config.md docs/release/w1-webhook-env.md'
SELF='scripts/tests/security/test-env-example.sh'
if [ -n "$REV" ]; then
    git -C "$REPO" grep -n -I -F 8888 "$REV" -- . ':!internal/' ':!config/' ':!go.sum' 2>/dev/null | sed "s|^$REV:||" >"$TMP/hits" || true
else
    git -C "$REPO" grep -n -I -F 8888 -- . ':!internal/' ':!config/' ':!go.sum' 2>/dev/null >"$TMP/hits" || true
fi
: >"$TMP/bad8888"
while IFS= read -r h; do
    f="${h%%:*}"; rest="${h#*:}"; text="${rest#*:}"
    [ "$f" = "$SELF" ] && continue
    case " $ALLOW_8888 $MANTOU_DOCS " in *" $f "*) echo "  登记例外: ${f}:${rest%%:*}"; continue ;; esac
    # 同一行同时写了 8889 的，是“8888 改为 8889”一类迁移说明，不算旧值
    case "$text" in *8889*) continue ;; esac
    # 注释行（# 或 // 开头）只是解释历史默认值，不算“值”
    stripped="$(printf '%s' "$text" | sed 's/^[[:space:]]*//')"
    case "$stripped" in '#'*|'//'*) continue ;; esac
    echo "$h" >>"$TMP/bad8888"
done <"$TMP/hits"
if [ -s "$TMP/bad8888" ]; then bad "本人范围内仍有非注释的 8888：$(tr '\n' ' ' <"$TMP/bad8888")"
else ok "本人范围内无非注释的 8888（internal/、config/ 不属本人，只登记）"; fi

# ---------- JWT_EXPIRATION ----------
je="$(val JWT_EXPIRATION)"
[ "$je" = 30 ] && ok "JWT_EXPIRATION=30" || bad "JWT_EXPIRATION=${je}（应为 30 分钟；3600 实为 60 小时，合入后生产上限 1440）"
block JWT_EXPIRATION >"$TMP/je.txt"
for kw in '单位是分钟' '合入 sched/w1-mantou 前' '合入 sched/w1-mantou 后' '1440'; do
    if grep -F -q -e "$kw" "$TMP/je.txt"; then ok "JWT_EXPIRATION 注释含「${kw}」"
    else bad "JWT_EXPIRATION 注释缺「${kw}」"; fi
done
src deploy/release/compose.yaml >"$TMP/compose.yaml" 2>/dev/null || : >"$TMP/compose.yaml"
cj="$(sed -n 's/^[[:space:]]*JWT_EXPIRATION:[[:space:]]*"\{0,1\}\([0-9]*\)"\{0,1\}.*/\1/p' "$TMP/compose.yaml" | head -n 1)"
[ "$cj" = "$je" ] && ok "deploy/release/compose.yaml JWT_EXPIRATION=${cj} 与 env.example 一致" \
    || bad "deploy/release/compose.yaml JWT_EXPIRATION=${cj}，env.example 为 ${je}"
if [ -n "$REV" ]; then
    git -C "$REPO" grep -n -I -F JWT_EXPIRATION "$REV" -- . ':!internal/' ':!config/' 2>/dev/null | sed "s|^$REV:||" >"$TMP/jhits" || true
else
    git -C "$REPO" grep -n -I -F JWT_EXPIRATION -- . ':!internal/' ':!config/' 2>/dev/null >"$TMP/jhits" || true
fi
: >"$TMP/badjwt"
# 单位按文件判断：一个 .md 提到 JWT_EXPIRATION 时，其中至少一处提到它的行写明分钟/minute
for f in $(cut -d: -f1 "$TMP/jhits" | sort -u); do
    case "$f" in *.md) ;; *) continue ;; esac
    case " $MANTOU_DOCS " in *" $f "*) continue ;; esac
    grep "^${f}:" "$TMP/jhits" | grep -E -q '分钟|minute' || echo "${f}（提到 JWT_EXPIRATION 但未写单位）" >>"$TMP/badjwt"
done
while IFS= read -r h; do
    f="${h%%:*}"
    case "$f" in "$SELF"|env.example|deploy/release/compose.yaml|*.go) continue ;; esac
    case " $MANTOU_DOCS " in *" $f "*) echo "  登记（馒头）: ${f}:$(printf '%s' "${h#*:}" | cut -d: -f1)"; continue ;; esac
    if printf '%s' "$h" | grep -E -q 'JWT_EXPIRATION[=:][[:space:]]*"?3600'; then echo "${h}（3600）" >>"$TMP/badjwt"; fi
done <"$TMP/jhits"
if [ -s "$TMP/badjwt" ]; then bad "本人范围内 JWT_EXPIRATION 说明不符：$(tr '\n' ' ' <"$TMP/badjwt")"
else ok "本人范围内提到 JWT_EXPIRATION 的文档均写明分钟、无 3600（internal/、config/ 只登记）"; fi

# ---------- Redis ----------
rp="$(val REDIS_PASSWORD)"
[ -z "$rp" ] && ok "REDIS_PASSWORD 值为空" || bad "REDIS_PASSWORD 不为空"
block REDIS_PASSWORD >"$TMP/rp.txt"
grep -F -q '必填' "$TMP/rp.txt" && ok "REDIS_PASSWORD 注释标明 compose 必填" || bad "REDIS_PASSWORD 注释未标明必填"
if grep -F -q '默认空（无认证）' "$TMP/rp.txt"; then bad "REDIS_PASSWORD 仍写着“可选：默认空”"; fi

echo "目标: ${REV:-工作区}；失败 ${FAILS} 项"
[ "$FAILS" -eq 0 ]
