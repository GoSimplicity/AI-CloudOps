#!/usr/bin/env bash
# review3f/review3g/review3h：scripts/check-webhook-env.sh 的回归测试（规则对应当前 Go loader 校验）。
# 每个用例分别用 sh、bash、bash --posix 运行被测脚本（机器上没有的解释器跳过），断言退出码与指定字段的结论；
# 另断言输出里不出现任何取值（合成的哨兵数字 / 合成密钥）。只读：样本写在 mktemp 目录；config/ 只读取。
# 用法: scripts/tests/security/test-check-webhook-env.sh      兼容 macOS（bash 3.2 / BSD grep、sed、awk）。
set -eu
HERE="$(cd "$(dirname "$0")" && pwd)"
REPO="$(git -C "$HERE" rev-parse --show-toplevel)"
CHK="$REPO/scripts/check-webhook-env.sh"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/test-check-webhook-env.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT
FAILS=0; N=0
ok()  { echo "PASS [$1]"; }
bad() { echo "FAIL [$1]"; FAILS=$((FAILS+1)); }
SHELLS=""
for s in sh bash; do command -v "$s" >/dev/null 2>&1 && SHELLS="$SHELLS $s"; done
# 合成哨兵：运行时生成，断言不出现在输出里（不是真实密钥）
SEC="sk$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')"
NUM=$((700 + $(od -An -N2 -tu2 /dev/urandom | tr -d ' \n') % 300))

y() { printf '%s\n' "$2" >"$TMP/$1.yaml"; }   # y <名字> <内容>
y repo "webhook:
  common_map_renew_interval_seconds: 10  # 注释
  port: \"8889\"
  default_upgrade_minutes: 30
  im_feishu:
    app_secret: \"$SEC\"
    request_timeout_seconds: 10
mysql:
  addr: \"u:$SEC@tcp(127.0.0.1:3306)/d\""
y flat "port: \"8889\"
fixed_workers: 10
mysql:
  addr: \"\""
y flatPORT "PORT: \"8889\""
y noroot "mysql:
  addr: \"\""
y whnull "webhook:
mysql:
  addr: \"\""
: >"$TMP/empty.yaml"
y q100 "webhook:
  default_upgrade_minutes: \"10.0\""
y uq100 "webhook:
  default_upgrade_minutes: 10.0"
y qsp "webhook:
  fixed_workers: \" 10\""
y neg "webhook:
  common_map_renew_interval_seconds: -5"
y zero "webhook:
  common_map_renew_interval_seconds: 0"
y p0080 "webhook:
  port: \"0080\""
y pboth "webhook:
  port: \"8889\"
  PORT: \"8890\""
y tmo0 "webhook:
  im_feishu:
    request_timeout_seconds: 0"
# review3g：任意仅大小写不同 / 完全相同的同名键
y dupfront "webhook:
  front_domain: a
  port: \"8889\"
  FRONT_DOMAIN: b"
y dupim "webhook:
  im_feishu:
    private_robot_app_id: \"$SEC\"
    request_timeout_seconds: 10
    PRIVATE_ROBOT_APP_ID: \"$SEC\""
y dupimblock "webhook:
  im_feishu:
    request_timeout_seconds: 10
  IM_FEISHU:
    group_message_api: x"
y dupsame "webhook:
  min_workers: 1
  min_workers: 2"
y duproot "webhook:
  port: \"8889\"
Webhook:
  port: \"8890\""
y dupmysql "webhook:
  port: \"8889\"
mysql:
  addr: \"u:$SEC@tcp(h:1)/d\"
  ADDR: \"u:$SEC@tcp(h:1)/d\""
y nodup "webhook:
  port: \"8889\"
  im_feishu:
    port: 1
    request_timeout_seconds: 10
mysql:
  port: 3306
extra:
  - name: a
    port: 1
  - name: b
    port: 2
  NAME: c"
y emptyval "webhook:
  port: \"\""
y sentinel "webhook:
  fixed_workers: $NUM"
printf 'webhook:\r\n  port: "8889"\r\n  fixed_workers: 3\r\n' >"$TMP/crlf.yaml"
# review3h P3-d：先去注释再判断（制表符 + 注释、行尾注释），多文档、块标量里的 port 不误报；不支持的写法判不合规/请人工检查
printf 'webhook:\t# 注释\n  port: "8889"\n' >"$TMP/tabc.yaml"
y spc "webhook: # 注释
  port: 8889 # 端口
  im_feishu: # 飞书
    request_timeout_seconds: 10 # 秒"
y multidoc "webhook:
  port: \"8889\"
---
webhook:
  port: abc
  PORT: 1
port: 1"
y leaddoc "---
webhook:
  port: 8889"
y blockscalar "doc: |
  webhook:
  port: 1
webhook:
  port: \"8889\"
  note: |
    port: abc
    PORT: 1

    fixed_workers: x
  fixed_workers: 3
mysql:
  desc: >-
    port: 1
    port: 2"
y qhash "webhook:
  front_domain: \"http://x/#/a\" # 注释
  backend_domain: 'it''s # 不是注释'
  alert_manager_api: http://x/a#b
  port: 8889"
y uflow "webhook:
  im_feishu: {request_timeout_seconds: 0}"
y uflowseq "webhook:
  port: [1]"
y uanchor "base: &b
  port: 1
webhook:
  <<: *b"
y ualias "webhook:
  port: *p"
y uesc "webhook:
  \"po\\x72t\": 1"
y uq "webhook:
  ? port
  : 1"
printf 'webhook:\n\tport: 1\n' >"$TMP/utab.yaml"
printf 'WEBHOOK_FIXED_WORKERS="12"\nexport WEBHOOK_PORT=9001\nWEBHOOK_MYSQL_ADDR=u:%s@tcp(h:1)/d\nWEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET=%s\n' "$SEC" "$SEC" >"$TMP/a.env"

# case_ <名字> <期望退出码> <期望行（“字段: 结论前缀”，可多条用 | 分隔）> -- [一个 VAR=值] -- <参数…>
#   VAR=值 整体作为一个参数传给 env，值里的空格原样保留
case_() {
    name="$1"; want_rc="$2"; want="$3"; shift 3
    E1=""; [ "$1" = "--" ] && shift
    if [ "$1" != "--" ]; then E1="$1"; shift; fi
    shift
    for s in $SHELLS "bash --posix"; do
        case "$s" in "bash --posix") command -v bash >/dev/null 2>&1 || continue ;; esac
        N=$((N+1))
        # 清掉调用方环境里的 WEBHOOK_*，只带用例指定的变量（另带合成密钥变量，验证不被读出）
        rc=0
        env -i PATH="$PATH" WEBHOOK_MYSQL_ADDR="u:$SEC@tcp(h:1)/d" WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET="$SEC" ${E1:+"$E1"} \
            $s "$CHK" "$@" >"$TMP/out" 2>&1 || rc=$?
        msg="${name}（${s}）"
        [ "$rc" -eq "$want_rc" ] || { bad "$msg 退出码 $rc，期望 $want_rc"; sed 's/^/    /' "$TMP/out"; continue; }
        if grep -F -q -e "$SEC" -e "$NUM" "$TMP/out"; then bad "$msg 输出里出现了取值"; continue; fi
        miss=""
        old_ifs="$IFS"; IFS='|'
        for w in $want; do grep -F -q -e "$w" "$TMP/out" || miss="$miss [$w]"; done
        IFS="$old_ifs"
        [ -z "$miss" ] && ok "$msg" || { bad "$msg 缺:$miss"; sed 's/^/    /' "$TMP/out"; }
    done
}
C="--config"
case_ "仓库形态 yaml、无环境变量" 0 "配置文件根节点: 合规|配置文件重复键: 合规|webhook.port: 合规（配置文件）|webhook.fixed_workers: 合规（内置默认值）|webhook.im_feishu.request_timeout_seconds: 合规（配置文件）" -- -- $C "$TMP/repo.yaml"
case_ "仓库 config/webhook.yaml + env.example" 0 "配置文件根节点: 合规|配置文件重复键: 合规|结果: 合规（告警 0 项）" -- -- $C "$REPO/config/webhook.yaml" --env-file "$REPO/env.example"
case_ "平铺 webhook 字段" 1 "配置文件根节点: 不合规（根节点下有 webhook 字段 port, fixed_workers" -- -- $C "$TMP/flat.yaml"
case_ "平铺大写 PORT" 1 "配置文件根节点: 不合规（根节点下有 webhook 字段 port" -- -- $C "$TMP/flatPORT.yaml"
case_ "无根节点只有 mysql" 0 "配置文件根节点: 告警|结果: 合规（告警 1 项）" -- -- $C "$TMP/noroot.yaml"
case_ "webhook: 为空" 0 "配置文件根节点: 告警" -- -- $C "$TMP/whnull.yaml"
case_ "空文件" 0 "配置文件根节点: 告警" -- -- $C "$TMP/empty.yaml"
for v in 10.0 " 10" "10 " 10.5 1e1 -1 +10 0x10; do
    case_ "WEBHOOK_FIXED_WORKERS='$v'" 1 "webhook.fixed_workers: 不合规（环境变量 WEBHOOK_FIXED_WORKERS：不是十进制整数" -- "WEBHOOK_FIXED_WORKERS=$v" --
done
case_ "WEBHOOK_FIXED_WORKERS=010" 1 "webhook.fixed_workers: 不合规（环境变量 WEBHOOK_FIXED_WORKERS：有前导零" -- WEBHOOK_FIXED_WORKERS=010 --
case_ "WEBHOOK_FIXED_WORKERS=0" 1 "webhook.fixed_workers: 不合规（环境变量 WEBHOOK_FIXED_WORKERS：小于 1" -- WEBHOOK_FIXED_WORKERS=0 --
case_ "WEBHOOK_DEFAULT_UPGRADE_MINUTES=0" 1 "webhook.default_upgrade_minutes: 不合规（环境变量 WEBHOOK_DEFAULT_UPGRADE_MINUTES：小于 1" -- WEBHOOK_DEFAULT_UPGRADE_MINUTES=0 --
case_ "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS=0" 1 "webhook.common_map_renew_interval_seconds: 不合规" -- WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS=0 --
case_ "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=0" 1 "webhook.im_feishu.request_timeout_seconds: 不合规（环境变量" -- WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=0 --
for bound in WEBHOOK_FIXED_WORKERS:fixed_workers:1024 WEBHOOK_DEFAULT_UPGRADE_MINUTES:default_upgrade_minutes:10080 WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS:common_map_renew_interval_seconds:86400 WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS:im_feishu.request_timeout_seconds:600; do
    ev="${bound%%:*}"; rest="${bound#*:}"; field="${rest%:*}"; upper="${rest##*:}"
    case_ "${ev}=upper boundary" 0 "webhook.${field}: 合规（环境变量" -- "${ev}=${upper}" --
    case_ "${ev}=above upper boundary" 1 "webhook.${field}: 不合规（环境变量" -- "${ev}=$((upper+1))" --
done
case_ "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=-1" 1 "webhook.im_feishu.request_timeout_seconds: 不合规" -- WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=-1 --
case_ "yaml 加引号 10.0" 1 "webhook.default_upgrade_minutes: 不合规（配置文件：不是十进制整数" -- -- $C "$TMP/q100.yaml"
case_ "yaml 不加引号 10.0" 1 "webhook.default_upgrade_minutes: 不合规（配置文件：不是十进制整数" -- -- $C "$TMP/uq100.yaml"
case_ "yaml 加引号带空格" 1 "webhook.fixed_workers: 不合规（配置文件" -- -- $C "$TMP/qsp.yaml"
case_ "yaml 负数" 1 "webhook.common_map_renew_interval_seconds: 不合规（配置文件" -- -- $C "$TMP/neg.yaml"
case_ "yaml 0" 1 "webhook.common_map_renew_interval_seconds: 不合规（配置文件：小于 1" -- -- $C "$TMP/zero.yaml"
case_ "yaml timeout 0" 1 "webhook.im_feishu.request_timeout_seconds: 不合规（配置文件" -- -- $C "$TMP/tmo0.yaml"
case_ "yaml port 0080" 1 "webhook.port: 不合规（配置文件：有前导零" -- -- $C "$TMP/p0080.yaml"
for v in 0080 0 65536 abc " 9998 "; do
    case_ "WEBHOOK_PORT='$v'" 1 "webhook.port: 不合规（环境变量 WEBHOOK_PORT" -- "WEBHOOK_PORT=$v" --
done
case_ "WEBHOOK_PORT=65535" 0 "webhook.port: 合规（环境变量 WEBHOOK_PORT）" -- WEBHOOK_PORT=65535 --
case_ "port 与 PORT 同写" 1 "webhook.port: 不合规（配置文件同一层重复写了 port|配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：webhook.port；" -- -- $C "$TMP/pboth.yaml"
case_ "port 与 PORT 同写 + WEBHOOK_PORT 已设" 1 "webhook.port: 不合规（配置文件同一层重复写了 port|配置文件重复键: 不合规" -- WEBHOOK_PORT=9001 -- $C "$TMP/pboth.yaml"
case_ "front_domain 与 FRONT_DOMAIN 同写" 1 "配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：webhook.front_domain；|webhook.port: 合规（配置文件）" -- -- $C "$TMP/dupfront.yaml"
case_ "im_feishu 下 app_id 大小写重复（值不回显）" 1 "配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：webhook.im_feishu.private_robot_app_id；" -- -- $C "$TMP/dupim.yaml"
case_ "im_feishu 与 IM_FEISHU 两个块" 1 "配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：webhook.im_feishu；" -- -- $C "$TMP/dupimblock.yaml"
case_ "完全相同的键写两次" 1 "配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：webhook.min_workers；" -- -- $C "$TMP/dupsame.yaml"
case_ "webhook 与 Webhook 两个根" 1 "配置文件根节点: 不合规（webhook 根节点重复）|配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：webhook, webhook.port；" -- -- $C "$TMP/duproot.yaml"
case_ "mysql 下 addr 与 ADDR（值不回显）" 1 "配置文件重复键: 不合规（同一层有仅大小写不同或完全相同的键：mysql.addr；" -- -- $C "$TMP/dupmysql.yaml"
case_ "不同父节点同名、列表项同名不算重复" 0 "配置文件重复键: 合规|结果: 合规" -- -- $C "$TMP/nodup.yaml"
case_ "yaml 空值" 1 "webhook.port: 不合规（配置文件里是空值" -- -- $C "$TMP/emptyval.yaml"
case_ "环境变量覆盖文件" 0 "webhook.common_map_renew_interval_seconds: 合规（环境变量" -- WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS=5 -- $C "$TMP/zero.yaml"
case_ "空环境变量视为未设置" 1 "webhook.common_map_renew_interval_seconds: 不合规（配置文件：小于 1" -- WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS= -- $C "$TMP/zero.yaml"
case_ "哨兵值不回显" 0 "webhook.fixed_workers: 合规（配置文件）" -- -- $C "$TMP/sentinel.yaml"
case_ "CRLF 文件" 0 "webhook.port: 合规（配置文件）|webhook.fixed_workers: 合规（配置文件）" -- -- $C "$TMP/crlf.yaml"
case_ "--env-file（引号、export、密钥行）" 0 "webhook.fixed_workers: 合规（环境变量 WEBHOOK_FIXED_WORKERS）|webhook.port: 合规（环境变量 WEBHOOK_PORT）" -- -- --env-file "$TMP/a.env"
case_ "--config 文件不存在" 2 "配置文件不可读" -- -- $C "$TMP/nope.yaml"
# review3h P3-d
case_ "webhook:<TAB># 注释" 0 "配置文件根节点: 合规|配置文件写法: 合规|webhook.port: 合规（配置文件）" -- -- $C "$TMP/tabc.yaml"
case_ "webhook: # 注释 与行尾注释" 0 "配置文件根节点: 合规|webhook.port: 合规（配置文件）|webhook.im_feishu.request_timeout_seconds: 合规（配置文件）" -- -- $C "$TMP/spc.yaml"
case_ "多文档 --- 拒绝启动" 1 "配置文件根节点: 合规|配置文件重复键: 合规|webhook.port: 合规（配置文件）|配置文件写法: 不合规（多文档" -- -- $C "$TMP/multidoc.yaml"
case_ "开头的 --- 不算多文档" 0 "配置文件根节点: 合规|配置文件写法: 合规|webhook.port: 合规（配置文件）|结果: 合规（告警 0 项）" -- -- $C "$TMP/leaddoc.yaml"
case_ "块标量里的 port/webhook 不算键" 0 "配置文件根节点: 合规|配置文件重复键: 合规|配置文件写法: 合规|webhook.port: 合规（配置文件）|webhook.fixed_workers: 合规（配置文件）" -- -- $C "$TMP/blockscalar.yaml"
case_ "引号内与非空白后的 # 不算注释（对照）" 0 "配置文件写法: 合规|配置文件重复键: 合规|webhook.port: 合规（配置文件）|结果: 合规（告警 0 项）" -- -- $C "$TMP/qhash.yaml"
case_ "内联表 im_feishu: {...}" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：内联表/数组 第 2 行；请人工检查）" -- -- $C "$TMP/uflow.yaml"
case_ "内联数组 port: [1]" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：内联表/数组 第 2 行；请人工检查）" -- -- $C "$TMP/uflowseq.yaml"
case_ "锚点 + 合并键 <<: *b" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：锚点/别名 第 1 行, 合并键 << 第 4 行, 锚点/别名 第 4 行；请人工检查）" -- -- $C "$TMP/uanchor.yaml"
case_ "别名 port: *p" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：锚点/别名 第 2 行；请人工检查）" -- -- $C "$TMP/ualias.yaml"
case_ "转义键 \"po\\x72t\"" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：转义键 第 2 行；请人工检查）" -- -- $C "$TMP/uesc.yaml"
case_ "复杂键 ? port" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：复杂键 ? 第 2 行；请人工检查）" -- -- $C "$TMP/uq.yaml"
case_ "制表符缩进" 1 "配置文件写法: 不合规（含本脚本不支持的 YAML 写法：制表符缩进 第 2 行；请人工检查）" -- -- $C "$TMP/utab.yaml"
# review3h P3-e：临时文件用 mktemp 创建并检查；TMPDIR 不可用时报错退出 2，不得继续出结论；运行后不留临时文件
case_ "TMPDIR 不存在" 2 "无法创建临时文件" -- "TMPDIR=$TMP/no-such-dir" -- $C "$TMP/repo.yaml"
case_ "TMPDIR 不存在（无 --config）" 2 "无法创建临时文件" -- "TMPDIR=$TMP/no-such-dir" --
mkdir "$TMP/td"
for s in $SHELLS "bash --posix"; do
    N=$((N+1)); rc=0
    env -i PATH="$PATH" TMPDIR="$TMP/td" $s "$CHK" $C "$TMP/repo.yaml" >"$TMP/out" 2>&1 || rc=$?
    left="$(ls -A "$TMP/td" | wc -l | tr -d ' ')"
    if [ "$rc" -eq 0 ] && [ "$left" = 0 ]; then ok "运行后 TMPDIR 无残留（${s}）"; else bad "运行后 TMPDIR 残留 ${left} 个 / 退出码 ${rc}（${s}）"; fi
done
echo "共 ${N} 次运行（解释器:${SHELLS} bash --posix）；失败 ${FAILS} 项"
[ "$FAILS" -eq 0 ]
