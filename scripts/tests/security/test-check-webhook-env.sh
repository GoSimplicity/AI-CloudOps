#!/usr/bin/env bash
# review3f：scripts/check-webhook-env.sh 的回归测试（规则对应 sched/w1-mantou 0146077 的 loader 校验）。
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
NUM=7$(od -An -N2 -tu2 /dev/urandom | tr -d ' \n')3

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
y emptyval "webhook:
  port: \"\""
y sentinel "webhook:
  fixed_workers: $NUM"
printf 'webhook:\r\n  port: "8889"\r\n  fixed_workers: 3\r\n' >"$TMP/crlf.yaml"
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
case_ "仓库形态 yaml、无环境变量" 0 "配置文件根节点: 合规|webhook.port: 合规（配置文件）|webhook.fixed_workers: 合规（内置默认值）|webhook.im_feishu.request_timeout_seconds: 合规（配置文件）" -- -- $C "$TMP/repo.yaml"
case_ "仓库 config/webhook.yaml + env.example" 0 "配置文件根节点: 合规|结果: 合规（告警 0 项）" -- -- $C "$REPO/config/webhook.yaml" --env-file "$REPO/env.example"
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
case_ "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=0" 0 "webhook.im_feishu.request_timeout_seconds: 告警（环境变量" -- WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=0 --
case_ "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=-1" 1 "webhook.im_feishu.request_timeout_seconds: 不合规" -- WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS=-1 --
case_ "yaml 加引号 10.0" 1 "webhook.default_upgrade_minutes: 不合规（配置文件：不是十进制整数" -- -- $C "$TMP/q100.yaml"
case_ "yaml 不加引号 10.0" 1 "webhook.default_upgrade_minutes: 不合规（配置文件：不是十进制整数" -- -- $C "$TMP/uq100.yaml"
case_ "yaml 加引号带空格" 1 "webhook.fixed_workers: 不合规（配置文件" -- -- $C "$TMP/qsp.yaml"
case_ "yaml 负数" 1 "webhook.common_map_renew_interval_seconds: 不合规（配置文件" -- -- $C "$TMP/neg.yaml"
case_ "yaml 0" 1 "webhook.common_map_renew_interval_seconds: 不合规（配置文件：小于 1" -- -- $C "$TMP/zero.yaml"
case_ "yaml timeout 0" 0 "webhook.im_feishu.request_timeout_seconds: 告警（配置文件" -- -- $C "$TMP/tmo0.yaml"
case_ "yaml port 0080" 1 "webhook.port: 不合规（配置文件：有前导零" -- -- $C "$TMP/p0080.yaml"
for v in 0080 0 65536 abc " 9998 "; do
    case_ "WEBHOOK_PORT='$v'" 1 "webhook.port: 不合规（环境变量 WEBHOOK_PORT" -- "WEBHOOK_PORT=$v" --
done
case_ "WEBHOOK_PORT=65535" 0 "webhook.port: 合规（环境变量 WEBHOOK_PORT）" -- WEBHOOK_PORT=65535 --
case_ "port 与 PORT 同写" 1 "webhook.port: 不合规（配置文件同一层重复写了 port" -- -- $C "$TMP/pboth.yaml"
case_ "yaml 空值" 1 "webhook.port: 不合规（配置文件里是空值" -- -- $C "$TMP/emptyval.yaml"
case_ "环境变量覆盖文件" 0 "webhook.common_map_renew_interval_seconds: 合规（环境变量" -- WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS=5 -- $C "$TMP/zero.yaml"
case_ "空环境变量视为未设置" 1 "webhook.common_map_renew_interval_seconds: 不合规（配置文件：小于 1" -- WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS= -- $C "$TMP/zero.yaml"
case_ "哨兵值不回显" 0 "webhook.fixed_workers: 合规（配置文件）" -- -- $C "$TMP/sentinel.yaml"
case_ "CRLF 文件" 0 "webhook.port: 合规（配置文件）|webhook.fixed_workers: 合规（配置文件）" -- -- $C "$TMP/crlf.yaml"
case_ "--env-file（引号、export、密钥行）" 0 "webhook.fixed_workers: 合规（环境变量 WEBHOOK_FIXED_WORKERS）|webhook.port: 合规（环境变量 WEBHOOK_PORT）" -- -- --env-file "$TMP/a.env"
case_ "--config 文件不存在" 2 "配置文件不可读" -- -- $C "$TMP/nope.yaml"
echo "共 ${N} 次运行（解释器:${SHELLS} bash --posix）；失败 ${FAILS} 项"
[ "$FAILS" -eq 0 ]
