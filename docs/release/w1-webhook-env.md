# W1：告警 webhook（`cmd/webhook`）支持的环境变量

## 优先级

**环境变量 > 配置文件（`config/webhook.yaml`，可用 `--config` 指定）> 默认值。**

`internal/config/loader.go` 的 `loadWebhookFromFile` 用独立的 viper 实例读取三层来源，逐键取合并后的值，
再用同一份值：

- 解码为返回的 `WebhookConfig`（`cmd/webhook` 用其中的 `Port` 监听）；
- 发布到全局 viper 的 `webhook.*`（webhook 子系统其余代码通过 `viper.Get*("webhook.…")` 读取）。

合并值只取 `webhook.*` 下的**已知叶子键**：用反射 `WebhookConfig` 生成两张表——父键白名单（目前只有 `webhook.im_feishu`）和叶子类型表
（string / int），只遍历叶子表，父键和未知键都不进入合并值；按键名排序后只构造一次，解码与发布按同一顺序写入同一份值。
因此**加载成功时**，`cfg` 的每个字段与全局 viper 按对应类型读取（`GetString` / `GetInt`）的结果一致（有逐字段比对的测试，含嵌套的 `im_feishu`，
并覆盖 `im_feishu:` 为 null 与 `{}` 两种写法各 200 次重复加载）。全局里存的是**规范化后**的值，类型与 `cfg` 字段一致（整数字段为 `int`，即使环境变量给的是字符串 `"3"`；字符串字段为 `string`）；
**加载失败**（类型错误、端口非法、整数字段非法或越界等）时不向全局发布任何键，全局里已有的旧值保持不变（有逐字段断言的测试）。
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

- **整数字段只接受十进制整数**（正则 `^(0|-?[1-9][0-9]*)$`），环境变量、加引号的 YAML 字符串和未加引号的 YAML 数值规则相同；
  YAML 数值按配置文件里的**原始文本**判断（用 yaml.v3 读节点），不经 YAML 的进制/浮点转换。以下写法 webhook **启动失败**：
  `10.5`、`-0.5`、`10.0`、`1e1`、`0x10`、`0o10`、`0b1`、`+3`、`010`（前导零）、`00`、`-0`、`1_0`、`" 10"`、`"10 "`、空串（加引号时）、`true`、`.inf`。
  （之前：未加引号的 `10.5` / `10.0` 被截断为 10，`010` 按八进制得 8，`0x10` 得 16，`+10` 得 10。）
  报错只列出键名和环境变量名（如 `webhook.fixed_workers（环境变量 WEBHOOK_FIXED_WORKERS）应为十进制整数`），**不含取值**。
- **类型错误**：标量叶子（如 `front_domain`、`im_feishu.request_timeout_seconds`、`im_feishu.private_robot_app_secret`）写成 map（含 `{}`）或列表，
  或父键 `webhook.im_feishu` / 根节点 `webhook` 写成标量或列表时，webhook **启动失败**，报错 `Webhook 配置类型错误: …` 只列键名。
  只检查配置文件本身：**文件里写错时，即使该键被环境变量覆盖也会失败**。父键写成 `~` / `null` / 留空 / `{}` 合法，子键取环境变量/默认值；
  叶子写成 `~` 等同未写。YAML 别名和 `<<` 合并键按展开后的结构检查。
- 字符串字段写成未加引号的数字/布尔（如 `port: 8081`、`backend_domain: 007`）时取文件里的原始文本（`007` 不会变成 `7`，`0x50` 不会变成 `80`）。
- 环境变量设为空串视为未设置，回落到配置文件/默认值。
- **端口校验**：`webhook.port` 取合并后的值并去首尾空白，必须是纯数字；先去掉前导零，再要求在 1–65535 之间（`" 9998 "`、`000080` 可以，
  `""`、`abc`、`0`、`0000`、`65536`、`065536`、`-1`、`+80`、`0x50` 不行），否则 webhook 启动失败；报错只列出键名 `webhook.port` 和变量名 `WEBHOOK_PORT`，不含取值。
  去空白、去前导零后的值（如 `000080` → `80`）同时用于监听和全局 `webhook.port`。
- **不要在 `webhook:` 下同时写 `port` 和 `PORT`**（或其他只有大小写不同的同名键）：viper 对键名不区分大小写，同时出现时取哪一个**未定义**。

- **未知键**：`webhook:` 下不在 `WebhookConfig` 中的键（仓库 `webhook.yaml` 里的 `alert_receive_queue_size`、`http_request_global_timeout_seconds`、
  `min_workers`、`max_workers`、`scale_threshold`、`scale_interval`、`im_feishu.private_message_api`，当前 Go 代码中没有读取它们）
  **不发布到全局 viper**，也不拒绝启动；每次加载成功时记**一条** `[配置告警]`，只列出排序后的完整键名（未知 map 只列到该键本身，不展开子键），不含取值。

- **不再支持没有 `webhook` 根节点的旧格式。** 配置文件根节点下出现 `WebhookConfig` 的字段（`port`、`fixed_workers`、`front_domain`、
  `backend_domain`、`default_upgrade_minutes`、`alert_manager_api`、`common_map_renew_interval_seconds`、`im_feishu`）时，webhook **启动失败**，
  报错列出这些键名并提示移到 `webhook:` 下，不含取值。只检查配置文件本身：环境变量（如 `PORT`）和默认值不算；`mysql` 等非 webhook 键不受影响；
  配置文件缺失时不做此检查。

- **配置文件存在但没有 `webhook` 根节点**（含写成 `webhook:` 而值为 null、空文件）时，记一条 `[配置告警]` 并继续启动，`webhook.*` 全部取环境变量/默认值；
  告警只列出文件路径和文件根节点下的键名，不含取值。配置文件缺失时仍按原来的“读取配置文件失败”告警处理，不重复提示。

### 整数字段取值范围

`WebhookConfig` 全部整数字段如下。合并后的值（环境变量 > 配置文件 > 默认值）不在 `[下限, 上限]` 闭区间时 webhook **启动失败**，
报错一次列出所有越界字段的键名、环境变量名和上下限（`…必须在 MIN 到 MAX 之间`），**不含取值**；与其他校验失败一样，不向全局 viper 发布任何键。
超出 int 表示范围的写法（如 `99999999999999999999`）同样按越界报错。仓库 `config/webhook.yaml` 与内置默认值都满足下表。

上限按语义取宽松但合理的值，同时保证乘以 `time.Second` / `time.Minute` 后不会溢出 `time.Duration`（int64 纳秒，约 292 年）：

| yaml 键 | 环境变量 | 默认值 | 允许范围 | 是否允许 0 | 含义与取值规则依据 |
| --- | --- | --- | --- | --- | --- |
| `webhook.fixed_workers` | `WEBHOOK_FIXED_WORKERS` | `10`（仓库 yaml 未设置） | `1`–`1024` | 否 | 固定工作者数量。0 个工作者没有意义。（注：当前 `consumer.go` 只把它存进 `workerCount`，没有实际使用，告警按条起 goroutine 处理）。上限 1024 远超单进程合理并发，防止误写成超大值 |
| `webhook.default_upgrade_minutes` | `WEBHOOK_DEFAULT_UPGRADE_MINUTES` | `60`（仓库 yaml 为 `30`） | `1`–`10080` | 否 | 告警升级等待分钟数，发送组 `upgrade_minutes` 为 0（即“用默认值”）时使用。`content.go` 按 `time.Since(StartsAt) > N 分钟` 判断，为 0 时所有 firing 告警立即升级。上限 10080 = 7 天 |
| `webhook.common_map_renew_interval_seconds` | `WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS` | `300`（仓库 yaml 为 `10`） | `1`–`86400` | 否 | 发送组/用户/值班组/规则缓存的刷新周期（秒）。`<= 0` 在加载阶段拒绝，`cache.go` 保留防御性回退（告警并改用 60 秒）；`wait.UntilWithContext` 周期为 0 会不间断刷库。上限 86400 = 1 天 |
| `webhook.im_feishu.request_timeout_seconds` | `WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS` | `10`（仓库 yaml 为 `10`） | `0`–`600` | **是**（0 = 不设超时） | 飞书机器人与 Alertmanager 静默接口 HTTP 请求的超时（秒），用作 `http.Client.Timeout`。Go 中 0 表示不设超时，不会立即超时或除零；负数在 `http.Client` 中同样等于不设超时，但不是有意义的写法，拒绝。上限 600 = 10 分钟 |

注意：设为 `0` 的 `request_timeout_seconds` 会让对端无响应时请求一直挂起，生产环境不建议使用。

整数的解析方式：
- 只接受十进制整数，未加引号的 YAML 小数**不再截断**：`fixed_workers: 10.5`、`10.0` 启动失败（之前得到 `10`）。
- 不再按进制前缀解析：`010`、`0x10`、`+3` 启动失败（之前分别得到 `8`、`16`、`3`）；环境变量和 YAML 规则相同。
- 解码不走 mapstructure 的 `WeaklyTypedInput` 宽松转换，按叶子类型表逐字段赋值；解码报错只含键名和期望类型（`int` / `string`），不回显取值。

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
- 整数字段新增上限：`fixed_workers > 1024`、`default_upgrade_minutes > 10080`、`common_map_renew_interval_seconds > 86400`、
  `im_feishu.request_timeout_seconds > 600` 时**启动失败**。
- 整数字段严格为十进制整数：未加引号的 YAML `10.5` / `10.0`（之前截断为 10）、`010`（之前八进制 8）、`0x10`、`+3`，以及环境变量 `010` / `0x10` / `+3`
  （之前按进制前缀接受）现在**启动失败**。
- 标量叶子写成 map / 列表（之前静默读成零值，例如 `request_timeout_seconds: {x: -1}` 让 HTTP 客户端没有超时），父键 `im_feishu` 或根节点 `webhook`
  写成标量 / 列表（bb410eb 之前就被静默忽略）现在**启动失败**；配置文件里写错时即使环境变量覆盖了该键也失败。
- 字符串字段写成未加引号的数字时按原始文本取值，例如 `port: 0x50` 之前被 YAML 解析成 80 并通过，现在按 `0x50` 判为非法端口。
- `webhook:` 下的未知键不再发布到全局 viper（之前会以 `webhook.min_workers` 等出现在全局，但没有代码读取），改为一条启动告警。
- 全局 viper 中整数字段的值类型统一为 `int`（之前可能是环境变量的字符串）；通过 `viper.GetInt` / `GetString` 读取的现有代码不受影响。
- **不再支持没有 `webhook` 根节点的旧格式（平铺写法）**，此类配置文件现在加载即失败，请把 `port`、`fixed_workers` 等键移到 `webhook:` 下。
  排查结论：仓库内唯一的 webhook 配置文件 `config/webhook.yaml` 自引入起（3c47234）就带 `webhook` 根节点；`deploy/`（Kubernetes、release Compose、nginx、
  Prometheus）、`Dockerfile`、`docker-compose.yaml` 中没有 webhook 配置文件，也没有 Helm values 或 ConfigMap 承载 webhook 配置。
  原“兼容旧配置”分支是 f25c744（`pkg/di/viper.go` 迁到 `internal/config/loader.go`）时加入的防御代码；此前全局 viper 一直只读 `webhook.*`，
  从未支持过平铺格式。而且该分支实际不会执行：根节点缺失时 `UnmarshalKey("webhook")` 不返回错误，平铺的键被**静默忽略**、回落到默认值
  （例如端口变成 8889，已在 70bf9fc 与 bb410eb 上用临时文件验证）。本提交把这种静默忽略改为明确报错。
- 主后端（`SERVER_PORT`）和 webhook 默认都是 8889；部署在同一台机器或共享网络命名空间时，**用 `WEBHOOK_PORT` 错开 webhook 端口的做法从本提交起成立**。
