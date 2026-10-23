# W1：告警 webhook（`cmd/webhook`）支持的环境变量

## 优先级

**环境变量 > 配置文件（`config/webhook.yaml`，可用 `--config` 指定）> 默认值。**

`internal/config/loader.go` 的 `loadWebhookFromFile` 用独立的 viper 实例读取三层来源，逐键取合并后的值，
再用同一份值：

- 解码为返回的 `WebhookConfig`（`cmd/webhook` 用其中的 `Port` 监听）；
- 发布到全局 viper 的 `webhook.*`（webhook 子系统其余代码通过 `viper.Get*("webhook.…")` 读取）。

两处取值完全一致（有逐字段比对的测试，含嵌套的 `im_feishu`）。`mysql.*` 等其他键不发布到全局。

## 支持的环境变量

以 `internal/config/config.go` 中 `WebhookConfig` / `ImFeishuConfig` 的 `env` 标签为准。
“webhook.yaml 中的值”一列是仓库当前文件的取值，仅供对照；有该值时它优先于默认值。

| 环境变量 | yaml 键 | 默认值 | 仓库 webhook.yaml 中的值 |
| --- | --- | --- | --- |
| `WEBHOOK_PORT` | `webhook.port` | `8889` | `"8889"` |
| `WEBHOOK_FIXED_WORKERS` | `webhook.fixed_workers` | `10` | 未设置 |
| `WEBHOOK_FRONT_DOMAIN` | `webhook.front_domain` | `http://localhost:3000` | `localhost:3000` |
| `WEBHOOK_BACKEND_DOMAIN` | `webhook.backend_domain` | `http://localhost:8889` | `localhost:8889/api/v1/alerts` |
| `WEBHOOK_DEFAULT_UPGRADE_MINUTES` | `webhook.default_upgrade_minutes` | `60` | `30` |
| `WEBHOOK_ALERT_MANAGER_API` | `webhook.alert_manager_api` | `http://localhost:9093` | `http://localhost:9093` |
| `WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS` | `webhook.common_map_renew_interval_seconds` | `300` | `10` |
| `WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API` | `webhook.im_feishu.group_message_api` | `https://open.feishu.cn/open-apis/im/v1/messages` | `https://open.feishu.cn/open-apis/bot/v2/hook/` |
| `WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS` | `webhook.im_feishu.request_timeout_seconds` | `10` | `10` |
| `WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID` | `webhook.im_feishu.private_robot_app_id` | 空 | 占位 `xxx` |
| `WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET` | `webhook.im_feishu.private_robot_app_secret` | 空 | 占位 `xxx`（**凭据**，请用环境变量注入，不要写进文件） |
| `WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API` | `webhook.im_feishu.tenant_access_token_api` | `https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal` | 同默认值 |

说明：

- 整数项接受字符串形式（如 `WEBHOOK_FIXED_WORKERS=3`）。
- 环境变量设为空串视为未设置，回落到配置文件/默认值。
- **端口校验**：`webhook.port` 取合并后的值并去首尾空白，必须是 1–65535 的纯数字（`" 9998 "` 可以，
  `""`、`abc`、`0`、`65536`、`-1`、`+80` 不行），否则 webhook 启动失败；报错只列出键名 `webhook.port` 和变量名 `WEBHOOK_PORT`，不含取值。
  去空白后的值同时用于监听和全局 `webhook.port`。
- `webhook.yaml` 里还有 `alert_receive_queue_size`、`min_workers` 等不在结构体中的键，它们没有 `env` 标签，不在上表中，
  不承诺可用环境变量覆盖。

- **不再支持没有 `webhook` 根节点的旧格式。** 配置文件根节点下出现 `WebhookConfig` 的字段（`port`、`fixed_workers`、`front_domain`、
  `backend_domain`、`default_upgrade_minutes`、`alert_manager_api`、`common_map_renew_interval_seconds`、`im_feishu`）时，webhook **启动失败**，
  报错列出这些键名并提示移到 `webhook:` 下，不含取值。只检查配置文件本身：环境变量（如 `PORT`）和默认值不算；`mysql` 等非 webhook 键不受影响；
  配置文件缺失时不做此检查。

### `WEBHOOK_MYSQL_ADDR`（单独说明）

- webhook 进程的完整 MySQL DSN（含账号密码），**必填，没有默认值**。
- 缺失或只有空白时 webhook **启动失败**，报错只列出变量名。
- 不读取配置文件中的 `mysql.addr`（文件里只保留空占位），也不使用主程序的 `MYSQL_ADDR`。
- 只经 `loadWebhookFromFile` 的返回值传给数据库初始化，**不会写入全局 viper**（全局中没有 `mysql.addr`）。

## 整合说明

- **基线（188e907）**：全局 viper 的 `webhook.*` 已经支持环境变量覆盖（`AutomaticEnv` + 按 `env` 标签 `BindEnv`），
  但返回的 `cfg` 用 `UnmarshalKey("webhook")` 解码，只拿到配置文件里的 `webhook` 子树，环境变量和默认值都进不了 `cfg`。
  `cmd/webhook` 用 `cfg.Port` 监听，因此 **`WEBHOOK_PORT` 从来没有改变过监听端口**；`cfg.FixedWorkers` 等未写在文件里的字段在 `cfg` 中为零值
  （子系统读的是全局值，不受影响）。**本提交修复了这一点**：`cfg` 改为从逐键合并后的值解码。
- **`WEBHOOK_MYSQL_ADDR`** 由 w1 链上的 2a350c3 引入（DSN 移出 `webhook.yaml`）；38b0356 改为独立 viper 实例、
  DSN 只经返回值传递、不写入全局 `mysql.addr`。基线不认这个变量，所以 **REL-03A 注入的 `WEBHOOK_MYSQL_ADDR` 要等 w1 合入后才会生效**。

## 发布说明（行为变化）

- **之前设置了 `WEBHOOK_PORT` 但没生效的环境，升级后会开始生效，监听端口可能改变。** 例如从 `env.example` 复制的 `.env`
  含 `WEBHOOK_PORT=8888`：升级前实际监听 `webhook.yaml` 的 8889，升级后改为监听 **8888**。
  请同步检查 Prometheus/Alertmanager 推送告警的地址（如 `PROMETHEUS_ALERT_WEBHOOK_ADDR`，`env.example` 中为 `localhost:8889`）
  是否指向 webhook 的实际端口，否则告警可能发到主后端（主后端默认也是 8889）。
- 配置文件存在但没写 `port` 时，之前 `cfg.Port` 为空串（监听地址为 `":"`），现在取 `WEBHOOK_PORT`，未设置则取默认值 **8889**。
  配置文件整个缺失时，之前已经取默认值 8889（但同样忽略 `WEBHOOK_PORT`），现在 `WEBHOOK_PORT` 也会生效。
- `webhook.port` 非法（空、非纯数字、超出 1–65535）时现在加载配置阶段即失败；之前不做校验，直接拼进监听地址。
- **不再支持没有 `webhook` 根节点的旧格式（平铺写法）**，此类配置文件现在加载即失败，请把 `port`、`fixed_workers` 等键移到 `webhook:` 下。
  排查结论：仓库内唯一的 webhook 配置文件 `config/webhook.yaml` 自引入起（3c47234）就带 `webhook` 根节点；`deploy/`（Kubernetes、release Compose、nginx、
  Prometheus）、`Dockerfile`、`docker-compose.yaml` 中没有 webhook 配置文件，也没有 Helm values 或 ConfigMap 承载 webhook 配置。
  原“兼容旧配置”分支是 f25c744（`pkg/di/viper.go` 迁到 `internal/config/loader.go`）时加入的防御代码；此前全局 viper 一直只读 `webhook.*`，
  从未支持过平铺格式。而且该分支实际不会执行：根节点缺失时 `UnmarshalKey("webhook")` 不返回错误，平铺的键被**静默忽略**、回落到默认值
  （例如端口变成 8889，已在 70bf9fc 与 bb410eb 上用临时文件验证）。本提交把这种静默忽略改为明确报错。
- 主后端（`SERVER_PORT`）和 webhook 默认都是 8889；部署在同一台机器或共享网络命名空间时，**用 `WEBHOOK_PORT` 错开 webhook 端口的做法从本提交起成立**。
