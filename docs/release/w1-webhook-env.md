# W1：告警 webhook（`cmd/webhook`）支持的环境变量

## 优先级

**环境变量 > 配置文件（`config/webhook.yaml`，可用 `--config` 指定）> 默认值。**

`internal/config/loader.go` 的 `loadWebhookFromFile` 用独立的 viper 实例读取三层来源，逐键取合并后的值，
再用同一份值：

- 解码为返回的 `WebhookConfig`（`cmd/webhook` 用其中的 `Port` 监听）；
- 发布到全局 viper 的 `webhook.*`（webhook 子系统其余代码通过 `viper.Get*("webhook.…")` 读取）。

合并值只取 `webhook.*` 下的**叶子键**（跳过 `webhook.im_feishu` 这类父键），按键名排序后只构造一次，解码与发布按同一顺序写入同一份值。
因此**加载成功时**，`cfg` 的每个字段与全局 viper 按对应类型读取（`GetString` / `GetInt`）的结果一致（有逐字段比对的测试，含嵌套的 `im_feishu`，
并覆盖 `im_feishu:` 为 null 与 `{}` 两种写法各 200 次重复加载）。注意两点：全局里存的是合并后的原始值（例如环境变量给的是字符串 `"3"`），
直接 `viper.Get` 拿到的类型可能与 `cfg` 字段类型不同；**加载失败**（端口非法、整数字段解析失败等）时不向全局发布任何键。
`mysql.*` 等其他键不发布到全局。

> e1c57de 之前这一点并不总成立：配置文件写成 `im_feishu:`（子键全部注释掉，值为 null）时，父键 `webhook.im_feishu` 的默认值 map
> 会按随机顺序覆盖子键的环境变量，`WEBHOOK_IM_FEISHU_*`（包括 `WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET`）约一半概率丢失，cfg 与全局也可能不一致。

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

- 整数项接受字符串形式（如 `WEBHOOK_FIXED_WORKERS=3`）。**整数字段不再接受 `10.0`、带空格或小数的写法**：环境变量或加引号的 YAML 字符串
  为 `10.0`、`" 10"`、`"10 "`、`10.5`、`1e1` 时 webhook **启动失败**（之前全局侧按 cast 宽松解析：`10.0` 当作 10，带空格或小数静默变成 0）。
  例外与细节（当前行为，有测试记录）：未加引号的 YAML 数值（`fixed_workers: 10.0`、`10.5`）由 YAML 解析成浮点数，仍被接受并**截断**为 10；
  字符串按 Go 的进制前缀解析，`010` 为八进制 8、`0x10` 为 16、`+10` 为 10，`cfg` 与全局两侧结果相同。
  整数解析失败的报错来自解码库，会带出该整数字段的取值（这些字段都不是凭据）。
- 环境变量设为空串视为未设置，回落到配置文件/默认值。
- **端口校验**：`webhook.port` 取合并后的值并去首尾空白，必须是纯数字；先去掉前导零，再要求在 1–65535 之间（`" 9998 "`、`000080` 可以，
  `""`、`abc`、`0`、`0000`、`65536`、`065536`、`-1`、`+80`、`0x50` 不行），否则 webhook 启动失败；报错只列出键名 `webhook.port` 和变量名 `WEBHOOK_PORT`，不含取值。
  去空白、去前导零后的值（如 `000080` → `80`）同时用于监听和全局 `webhook.port`。
- **不要在 `webhook:` 下同时写 `port` 和 `PORT`**（或其他只有大小写不同的同名键）：viper 对键名不区分大小写，同时出现时取哪一个**未定义**。

- `webhook.yaml` 里还有 `alert_receive_queue_size`、`min_workers` 等不在结构体中的键，它们没有 `env` 标签，不在上表中，
  不承诺可用环境变量覆盖，也不做整数范围校验（当前 Go 代码中没有读取它们）。

- **不再支持没有 `webhook` 根节点的旧格式。** 配置文件根节点下出现 `WebhookConfig` 的字段（`port`、`fixed_workers`、`front_domain`、
  `backend_domain`、`default_upgrade_minutes`、`alert_manager_api`、`common_map_renew_interval_seconds`、`im_feishu`）时，webhook **启动失败**，
  报错列出这些键名并提示移到 `webhook:` 下，不含取值。只检查配置文件本身：环境变量（如 `PORT`）和默认值不算；`mysql` 等非 webhook 键不受影响；
  配置文件缺失时不做此检查。

- **配置文件存在但没有 `webhook` 根节点**（含写成 `webhook:` 而值为 null、空文件）时，记一条 `[配置告警]` 并继续启动，`webhook.*` 全部取环境变量/默认值；
  告警只列出文件路径和文件根节点下的键名，不含取值。配置文件缺失时仍按原来的“读取配置文件失败”告警处理，不重复提示。

### 整数字段取值范围

`WebhookConfig` 全部整数字段如下。合并后的值（环境变量 > 配置文件 > 默认值）低于下限时 webhook **启动失败**，报错一次列出所有越界字段的键名、
环境变量名和下限，**不含取值**；与其他校验失败一样，不向全局 viper 发布任何键。不设上限。仓库 `config/webhook.yaml` 与内置默认值都满足下表。

| yaml 键 | 环境变量 | 默认值 | 允许范围 | 是否允许 0 | 含义与 0 值规则依据 |
| --- | --- | --- | --- | --- | --- |
| `webhook.fixed_workers` | `WEBHOOK_FIXED_WORKERS` | `10`（仓库 yaml 未设置） | `>= 1` | 否 | 固定工作者数量。0 个工作者没有意义。（注：当前 `consumer.go` 只把它存进 `workerCount`，没有实际使用，告警按条起 goroutine 处理） |
| `webhook.default_upgrade_minutes` | `WEBHOOK_DEFAULT_UPGRADE_MINUTES` | `60`（仓库 yaml 为 `30`） | `>= 1` | 否 | 告警升级等待分钟数，发送组 `upgrade_minutes` 为 0（即“用默认值”）时使用。`content.go` 按 `time.Since(StartsAt) > N 分钟` 判断，为 0 时所有 firing 告警立即升级 |
| `webhook.common_map_renew_interval_seconds` | `WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS` | `300`（仓库 yaml 为 `10`） | `>= 1` | 否 | 发送组/用户/值班组/规则缓存的刷新周期（秒）。`cache.go` 对 `<= 0` 告警并改用 60 秒；`wait.UntilWithContext` 周期为 0 会不间断刷库 |
| `webhook.im_feishu.request_timeout_seconds` | `WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS` | `10`（仓库 yaml 为 `10`） | `>= 0` | **是**（0 = 不设超时） | 飞书机器人与 Alertmanager 静默接口 HTTP 请求的超时（秒），用作 `http.Client.Timeout`。Go 中 0 表示不设超时，不会立即超时或除零；负数在 `http.Client` 中同样等于不设超时，但不是有意义的写法，拒绝 |

注意：设为 `0` 的 `request_timeout_seconds` 会让对端无响应时请求一直挂起，生产环境不建议使用。

整数的解析方式（当前行为，不做改动）：
- 未加引号的 YAML 小数会被**截断**：`fixed_workers: 10.5` 得到 `10`（`10.0` 同理）；加引号的 `"10.5"` 或环境变量 `10.5` 则启动失败。
- 字符串按 Go 进制前缀解析：`010` 按**八进制**解析为 `8`，`0x10` 为 `16`；环境变量和 YAML 都一样，不要给整数写前导零。

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
- `webhook.port` 非法（空、非纯数字、去前导零后超出 1–65535）时现在加载配置阶段即失败；之前不做校验，直接拼进监听地址。
  带前导零的合法端口（如 `000080`）规范为 `80`。
- 整数字段（`fixed_workers`、`default_upgrade_minutes`、`common_map_renew_interval_seconds`、`im_feishu.request_timeout_seconds`）
  由宽松改为严格：环境变量或带引号的 YAML 写成 `10.0`、带空格或小数时**启动失败**（见上文“说明”）。升级前请检查 `.env` 与 `webhook.yaml`。
- 整数字段新增范围校验：任何字段为负数，或 `fixed_workers`、`default_upgrade_minutes`、`common_map_renew_interval_seconds` 为 0 时**启动失败**
  （见“整数字段取值范围”）。之前这些值会被直接使用：缓存刷新周期 `<= 0` 回落到 60 秒，升级等待为 0 时立即升级。
- **不再支持没有 `webhook` 根节点的旧格式（平铺写法）**，此类配置文件现在加载即失败，请把 `port`、`fixed_workers` 等键移到 `webhook:` 下。
  排查结论：仓库内唯一的 webhook 配置文件 `config/webhook.yaml` 自引入起（3c47234）就带 `webhook` 根节点；`deploy/`（Kubernetes、release Compose、nginx、
  Prometheus）、`Dockerfile`、`docker-compose.yaml` 中没有 webhook 配置文件，也没有 Helm values 或 ConfigMap 承载 webhook 配置。
  原“兼容旧配置”分支是 f25c744（`pkg/di/viper.go` 迁到 `internal/config/loader.go`）时加入的防御代码；此前全局 viper 一直只读 `webhook.*`，
  从未支持过平铺格式。而且该分支实际不会执行：根节点缺失时 `UnmarshalKey("webhook")` 不返回错误，平铺的键被**静默忽略**、回落到默认值
  （例如端口变成 8889，已在 70bf9fc 与 bb410eb 上用临时文件验证）。本提交把这种静默忽略改为明确报错。
- 主后端（`SERVER_PORT`）和 webhook 默认都是 8889；部署在同一台机器或共享网络命名空间时，**用 `WEBHOOK_PORT` 错开 webhook 端口的做法从本提交起成立**。
