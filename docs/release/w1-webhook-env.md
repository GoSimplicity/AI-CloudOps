# W1：告警 webhook（`cmd/webhook`）支持的环境变量

## 优先级

**环境变量 > 配置文件（`config/webhook.yaml`，可用 `--config` 指定）> 默认值。**

配置文件只支持 YAML：扩展名必须是 `.yaml` / `.yml`；`.json` 作为 YAML 的子集接受，走同一套 YAML 节点检查。
其他扩展名（`.toml`、`.ini`、`.properties`、`.env`、`.hcl`、无扩展名等）webhook **启动失败**，报错
`Webhook 配置文件只支持 .yaml / .yml（.json 按 YAML 子集解析）格式，不支持 PATH`。

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
| `WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API` | `webhook.im_feishu.group_message_api` | `https://open.feishu.cn/open-apis/bot/v2/hook` | 同默认值 |
| `WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS` | `webhook.im_feishu.request_timeout_seconds` | `10` | `10` |
| `WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID` | `webhook.im_feishu.private_robot_app_id` | 空 | 占位 `xxx` |
| `WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET` | `webhook.im_feishu.private_robot_app_secret` | 空 | 占位 `xxx`（**凭据**，请用环境变量注入，不要写进文件） |
| `WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API` | `webhook.im_feishu.tenant_access_token_api` | `https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal` | 同默认值 |

说明：

- **飞书群机器人地址**：`group_message_api` 是群机器人 webhook 的前缀，发送时拼成 `前缀/机器人 token`。默认值与仓库 `webhook.yaml`
  都是 `https://open.feishu.cn/open-apis/bot/v2/hook`（不带结尾斜杠）；拼接前会先去掉前缀结尾的所有 `/`，
  所以写成 `…/bot/v2/hook/` 或 `…/bot/v2/hook//` 也只会得到 `…/bot/v2/hook/<token>`；首尾空白（如 `…/hook/ `）也会先去掉，不会拼出 `hook/ /<token>`。
  （之前默认值是 `https://open.feishu.cn/open-apis/im/v1/messages`，这是开放平台“发送消息”接口，不是群机器人地址，拼上 token 后无法发送；
  仓库 `webhook.yaml` 写的是带结尾斜杠的 `…/bot/v2/hook/`，拼出来是 `hook//<token>`。）
  注意：环境变量 `WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API` 优先于文件和默认值；如果部署里仍设为 `…/im/v1/messages`，群消息仍然发不出去，请改成上面的地址或不设置。
  `env.example` 第 87 行已同步改为 `…/bot/v2/hook`（之前是 `…/im/v1/messages`，从它复制的 `.env` 需要同步修改）。
  主程序工单通知的 `notification.feishu.webhook_url`（`config.development.yaml` / `config.production.yaml` 与 `loader.go` 默认值）本来就是 `…/bot/v2/hook/`，
  属于另一套配置，本次未改。
- **URL 字段去首尾空白**：`front_domain`、`backend_domain`、`alert_manager_api`、`im_feishu.group_message_api`、`im_feishu.tenant_access_token_api`
  无论来自环境变量、配置文件还是默认值，都先去首尾空白（`strings.TrimSpace`）；前四个是拼接前缀，再去掉结尾的 `/`
  （`http://f.example/` → `http://f.example`）；`tenant_access_token_api` 是完整接口地址，只去空白、保留结尾斜杠。去空白后的值同时用于 `cfg` 和全局 viper。
- **整数字段只接受十进制整数**（正则 `^(0|-?[1-9][0-9]*)$`），环境变量、加引号的 YAML 字符串和未加引号的 YAML 数值规则相同；
  YAML 数值按配置文件里的**原始文本**判断（用 yaml.v3 读节点），不经 YAML 的进制/浮点转换。以下写法 webhook **启动失败**：
  `10.5`、`-0.5`、`10.0`、`1e1`、`0x10`、`0o10`、`0b1`、`+3`、`010`（前导零）、`00`、`-0`、`1_0`、`" 10"`、`"10 "`（YAML 引号内带空白）、空串（加引号时）、`true`、`.inf`。
  （之前：未加引号的 `10.5` / `10.0` 被截断为 10，`010` 按八进制得 8，`0x10` 得 16，`+10` 得 10。）
- **整数环境变量先去首尾 ASCII 空白**：整数字段的环境变量取值先去掉首尾的空格、`\t`、`\r`、`\n`，再按上面的正则严格校验，
  所以 `WEBHOOK_FIXED_WORKERS=" 10"`、`"10\n"` 接受为 10；中间的空白（`1 0`）、其他空白（NBSP、`\v`、`\f`、U+0085、全角空格 U+3000）和全角数字仍然启动失败。
  非 URL 的字符串字段（`private_robot_app_id`、`private_robot_app_secret`）的环境变量**原样使用**，不去空白；URL 字段见上条，`webhook.port` 见“端口校验”。
  配置文件里加引号的 `" 10"` 不去空白，仍拒绝。
- **报错按取值的实际来源描述键**，不含取值：值来自配置文件时写 `配置文件中的 KEY`（如 `配置文件中的 webhook.fixed_workers 应为十进制整数`），
  来自环境变量时写 `KEY（环境变量 ENV）`（如 `webhook.fixed_workers（环境变量 WEBHOOK_FIXED_WORKERS）必须在 1 到 1024 之间`），
  只有环境变量来源才提环境变量名；来自内置默认值时写 `KEY（内置默认值）`。
- **写法或类型错误**：标量叶子（如 `front_domain`、`im_feishu.request_timeout_seconds`、`im_feishu.private_robot_app_secret`）写成 map（含 `{}`）或列表，
  或父键 `webhook.im_feishu` / 根节点 `webhook` 写成标量或列表时，webhook **启动失败**，报错以 `Webhook 配置写法或类型错误: ` 开头，只列键名或行号，
  多个问题按键名排序、用 `；` 分隔一次列出。只检查配置文件本身：**文件里写错时，即使该键被环境变量覆盖也会失败**。
  父键写成 `~` / `null` / 留空 / `{}` 合法，子键取环境变量/默认值；叶子写成 `~` 等同未写。
- **父键环境变量**：`WEBHOOK`、`WEBHOOK_IM_FEISHU` 不是配置项。它们设为非空值时，viper 会用这个字符串遮蔽整个 `webhook` / `webhook.im_feishu` 子树，
  所以 webhook **启动失败**，报错只列出变量名（不含取值）。设为空串等同未设置。叶子变量（如 `WEBHOOK_PORT`、`WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS`）不受影响。
  如果某个叶子因父键被遮蔽而取不到值，同样启动失败（`… 取不到值：其父键被环境变量或配置文件遮蔽`）；每个叶子都有内置默认值（有测试保证）。
- **YAML 别名与合并键**：
  - 键本身不能写成别名（`*a: 1`），启动失败。别名用作**值**（`front_domain: *a`）和不带引号、不带 tag 的 `<<` 合并键照常展开，展开后的结构同样受本节全部检查。
  - 合并优先级与 yaml.v3 一致：mapping 里显式写出的键优先于 `<<` 带入的键；`<<: [*a, *b]` 里靠前的来源优先；来源里显式写的键优先于来源自己的 `<<`。
    **只检查最终生效的条目**：被覆盖的值即使写错也不报错（它不会被使用）。
  - 同一层只能写一个 `<<`（yaml.v3 多个时只有最后一个生效），写多个启动失败（viper 读取时即按重复键报解析错误 `配置文件第 N 行: 同一层有重复的键`，结构扫描另有同层多个 `<<` 的检查兜底），请合并成 `<<: [*a, *b]`。
  - **带引号或带 tag 的 `<<` 不被支持**，一律启动失败（报错 `配置文件第 N 行：带引号或带 tag 的 << 不被支持`）：yaml.v3 里带引号的 `"<<"` / `'<<'`（无 tag）、
    `!!str <<` 是普通键，**不会合并**；而 `!!merge "<<"`、`!!merge <<` 会真的合并。写法看起来像合并、实际是否合并取决于引号和 tag，容易误解，所以只支持不带引号、不带 tag 的 `<<`。
  - `<<` 合并链（来源自身再 `<<` 另一个来源）最多 64 层，mapping 嵌套最多 64 层；超过时启动失败（`<< 合并链超过 64 层` / `嵌套超过 64 层`），不再静默截断
    （之前超过 64 层的合并来源被扫描忽略，而 viper 照常合并，`010` 之类的写法会绕过原文校验）。
- **显式 tag**：叶子只允许 `!!str`、`!!int`、`!!null` 三种显式 tag；`!!float`、`!!bool`、`!!binary`、`!!timestamp`、自定义 tag 等启动失败，报错写键名和 tag 说明，不含取值。
  不写 tag 时按上面的整数/字符串规则判断。
  **键**上只允许不写 tag 或 `!!str`，`!!binary`、`!!int`、`!!bool`、自定义 tag 等写在键上时启动失败（整份文件，报错只写行号）：
  例如 `!!binary Zml4ZWRfd29ya2Vycw==: 0x10` 会被 viper 解码成 `fixed_workers: 16`，而原文扫描看到的是另一个键名，会绕过十进制整数校验。
  兜底：viper 认为某叶子的值来自配置文件、原文扫描却拿不到这个叶子的原始文本时，也启动失败（`配置文件中的 KEY 无法按原文校验`），不按内置默认值放行。
- **多文档 YAML**：配置文件只能有一个 YAML 文档；出现第二个文档（`---` 之后还有内容）时启动失败（`配置文件包含多个 YAML 文档（---），只支持一个`）。
  之前 viper 只读第一个文档、其余被静默忽略。开头单独一个 `---` 仍然可以。
- **NUL 字符**：键或值里含 NUL（YAML `"\0"`、JSON `"\u0000"`）时启动失败，报错只写行号。
- **JSON 写法**：`.json` 文件按 YAML 子集做原文扫描，扫描前把 YAML 不认的两种合法 JSON 转义换成等价写法：UTF-16 代理对（`"\ud83d\ude00"`）和 `\/`；
  viper 仍按 `encoding/json` 解析原始内容。落单的代理项（`"\ud83d"`）仍按解析错误拒绝。
- **配置文件只读一次**：同一份字节同时交给 viper（`ReadConfig`）和原文扫描，避免两次读取之间文件被替换导致两边看到的内容不一致。
- **键名不能带点号**：根节点下的键（如 `"webhook.port": "8889"`、`"webhook.im_feishu": 5`，即使同时有正常的 `webhook:` 块），
  以及 `webhook:`、`webhook.im_feishu:` 下的键（如 `"im_feishu.request_timeout_seconds": 5`）带 `.` 时启动失败，请改成嵌套写法。
  拼出的路径是 webhook 已知键时报错写键名，否则只写行号（不回显任意键名）。
- **大小写重复键**（口径同 ee753b4 的 `scripts/check-webhook-env.sh`）：同一父节点下小写后相同的键（如 `port` 与 `PORT`、`im_feishu` 与 `IM_FEISHU`，
  或完全相同的键）不合规，检查整份文件（不只 `webhook:` 下，`<<` 展开后的键也算），启动失败，报错只列小写后的键路径；列表项里的 mapping 不参与。
  viper 对键名不区分大小写，同时出现时只会留下其中一个，所以直接拒绝。
  **只写一种大小写**（如只写 `PORT:` 或 `Webhook:`）仍然接受，按小写键名取值；但写在根节点的平铺 `PORT` 仍按“旧格式”拒绝。
- **配置文件解析失败**（YAML 语法错误、完全重复的键、tab 缩进等）时报错为 `读取Webhook配置文件失败: 配置文件第 N 行: 原因`：
  原因只取白名单里的固定中文说明（如“同一层有重复的键”“缩进里有 Tab 字符”“结构不完整（缺少预期的键、值、结束引号或括号）”“取值类型不符或数值超出范围”），
  匹配不上时写“无法解析（YAML/JSON 语法错误）”，**不回显解析器错误串里的任何原文**（键名、取值片段、数字、Go 字段路径都不输出）；
  行号只取错误串开头或换行后的 `line N:`，解析器没给行号时写 `配置文件解析失败: 原因`。
- 字符串字段写成未加引号的数字/布尔（如 `port: 8081`、`backend_domain: 007`）时取文件里的原始文本（`007` 不会变成 `7`，`0x50` 不会变成 `80`）。
- 环境变量设为空串视为未设置，回落到配置文件/默认值。
- **端口校验**：`webhook.port` 取合并后的值并只去首尾 ASCII 空白（空格、`\t`、`\r`、`\n`，与整数字段一致；NBSP、`\v`、全角空格等不去，按非法端口报错），
  必须是纯数字；先去掉前导零，再要求在 1–65535 之间（`" 9998 "`、`000080` 可以，
  `""`、`abc`、`0`、`0000`、`65536`、`065536`、`-1`、`+80`、`0x50` 不行），否则 webhook 启动失败；报错按来源写 `配置文件中的 webhook.port` 或
  `webhook.port（环境变量 WEBHOOK_PORT）`，不含取值。去空白、去前导零后的值（如 `000080` → `80`）同时用于监听和全局 `webhook.port`。
- **Kubernetes 注意**：如果同一命名空间里有名为 `webhook` 的 Service，kubelet 会给 Pod 注入 `WEBHOOK_PORT=tcp://<ClusterIP>:<端口>`
  （以及 `WEBHOOK_SERVICE_HOST` 等），它会覆盖配置文件里的端口并让 webhook 按“端口非法”启动失败。规避：在 Pod spec 里设 `enableServiceLinks: false`，
  或者把 Service 改名（不叫 `webhook`）。仓库 `deploy/` 当前没有 webhook 的 Kubernetes 清单，本提交未改部署文件。

- **未知键**：`webhook:` 下不在 `WebhookConfig` 中的键**不发布到全局 viper**，也不拒绝启动；每次加载成功时记**一条** `[配置告警]`，
  只列出排序后的完整键名（未知 map 只列到该键本身，不展开子键），不含取值。
  仓库 `webhook.yaml` 原有 7 个这样的键（`alert_receive_queue_size`、`http_request_global_timeout_seconds`、`min_workers`、`max_workers`、
  `scale_threshold`、`scale_interval`、`im_feishu.private_message_api`），Go 代码中没有读取它们，本提交已从文件中删除，仓库文件加载时不再告警。
  把它们接入代码（例如真正实现动态扩缩容工作者）不在本次范围内。
- **已知限制（全局里的旧未知键）**：加载成功时只覆盖 12 个已知叶子键，不会清除全局 viper 里此前已有的其他 `webhook.*` 键
  （例如同一进程中别处 `viper.Set` 写入的键）；这些键仍可被读到。本次只记录，不处理。
- **整份文件的检查范围（有意为之）**：大小写重复键、根节点带点号的键、键上的 tag、别名键、带引号或带 tag 的 `<<`、NUL、合并链/嵌套深度检查整份文件，
  不只 `webhook:` 子树。`LoadWebhook` 只读 `--config` 指定的文件（默认 `config/webhook.yaml`，唯一调用方 `cmd/webhook/webhook.go`），
  不读主程序配置（`config.development.yaml` / `config.production.yaml`），所以整份文件就是 webhook 自己的配置。
  **升级风险**：若部署把 `--config` 指向一份与主程序共用的大配置文件，其中非 webhook 部分的上述写法也会让 webhook 启动失败。

- **不再支持没有 `webhook` 根节点的旧格式。** 配置文件根节点下出现 `WebhookConfig` 的字段（`port`、`fixed_workers`、`front_domain`、
  `backend_domain`、`default_upgrade_minutes`、`alert_manager_api`、`common_map_renew_interval_seconds`、`im_feishu`）时，webhook **启动失败**，
  报错列出这些键名并提示移到 `webhook:` 下，不含取值。只检查配置文件本身：环境变量（如 `PORT`）和默认值不算；`mysql` 等非 webhook 键不受影响；
  配置文件缺失时不做此检查。

- **配置文件存在但没有 `webhook` 根节点**（含写成 `webhook:` 而值为 null、空文件）时，记一条 `[配置告警]` 并继续启动，`webhook.*` 全部取环境变量/默认值；
  告警只列出文件路径和文件根节点下的键名，不含取值。配置文件缺失时仍按原来的“读取配置文件失败”告警处理，不重复提示。

### 整数字段取值范围

`WebhookConfig` 全部整数字段如下。合并后的值（环境变量 > 配置文件 > 默认值）不在 `[下限, 上限]` 闭区间时 webhook **启动失败**，
报错一次列出所有越界字段的键名、来源（见上文“报错按取值的实际来源描述键”）和上下限（`…必须在 MIN 到 MAX 之间`），**不含取值**；与其他校验失败一样，不向全局 viper 发布任何键。
超出 int 表示范围的写法（如 `99999999999999999999`）同样按越界报错。仓库 `config/webhook.yaml` 与内置默认值都满足下表。

上限按语义取宽松但合理的值，同时保证乘以 `time.Second` / `time.Minute` 后不会溢出 `time.Duration`（int64 纳秒，约 292 年）：

| yaml 键 | 环境变量 | 默认值 | 允许范围 | 是否允许 0 | 含义与取值规则依据 |
| --- | --- | --- | --- | --- | --- |
| `webhook.fixed_workers` | `WEBHOOK_FIXED_WORKERS` | `10`（仓库 yaml 未设置） | `1`–`1024` | 否 | 固定工作者数量。0 个工作者没有意义。（注：当前 `consumer.go` 只把它存进 `workerCount`，没有实际使用，告警按条起 goroutine 处理）。上限 1024 远超单进程合理并发，防止误写成超大值 |
| `webhook.default_upgrade_minutes` | `WEBHOOK_DEFAULT_UPGRADE_MINUTES` | `60`（仓库 yaml 为 `30`） | `1`–`10080` | 否 | 告警升级等待分钟数，发送组 `upgrade_minutes` 为 0（即“用默认值”）时使用。`content.go` 按 `time.Since(StartsAt) > N 分钟` 判断，为 0 时所有 firing 告警立即升级。上限 10080 = 7 天 |
| `webhook.common_map_renew_interval_seconds` | `WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS` | `300`（仓库 yaml 为 `10`） | `1`–`86400` | 否 | 发送组/用户/值班组/规则缓存的刷新周期（秒）。`<= 0` 在加载阶段拒绝，`cache.go` 保留防御性回退（告警并改用 60 秒）；`wait.UntilWithContext` 周期为 0 会不间断刷库。上限 86400 = 1 天 |
| `webhook.im_feishu.request_timeout_seconds` | `WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS` | `10`（仓库 yaml 为 `10`） | `1`–`600` | 否 | 飞书机器人与 Alertmanager 静默接口 HTTP 请求的超时（秒），用作 `http.Client.Timeout`。Go 中 0 表示不设超时，对端无响应时请求会一直挂起，因此第二轮起拒绝 0；负数同理拒绝。上限 600 = 10 分钟 |

注意：之前允许的 `request_timeout_seconds: 0`（不设超时）现在启动失败，请改成 1–600 之间的值。

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
  由宽松改为严格：环境变量或带引号的 YAML 写成 `10.0`、小数，或带引号的 YAML 里带空格时**启动失败**（见上文“说明”）；
  整数环境变量首尾的空格、`\t`、`\r`、`\n` 会先去掉。升级前请检查 `.env` 与 `webhook.yaml`。
- 整数字段新增范围校验：任何字段为负数，或 `fixed_workers`、`default_upgrade_minutes`、`common_map_renew_interval_seconds` 为 0 时**启动失败**
  （见“整数字段取值范围”）。之前这些值会被直接使用：缓存刷新周期 `<= 0` 回落到 60 秒，升级等待为 0 时立即升级。
- `im_feishu.request_timeout_seconds` 的允许范围是 1–600：写 `0`（之前表示不设超时）现在**启动失败**。
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
- 配置文件只接受 `.yaml` / `.yml` / `.json`；`--config` 指向 `.toml` 等其他格式现在**启动失败**（仓库与 `deploy/` 中没有这样的用法）。
- 环境变量 `WEBHOOK`、`WEBHOOK_IM_FEISHU` 设为非空值现在**启动失败**（之前会静默遮蔽整个子树，叶子全部回落为零值或默认值）。
- 配置文件里的以下写法现在**启动失败**：键写成别名、同一层多个 `<<`、带引号或带 tag 的 `<<`（含 `!!merge <<`、`!!merge "<<"`）、
  键上 `!!str` 以外的显式 tag（含 `!!binary`）、叶子上的 `!!float` / `!!bool` 等显式 tag、键或值含 NUL、多个 YAML 文档、`<<` 合并链或嵌套超过 64 层、
  带点号的键名（根节点下，或 `webhook` / `im_feishu` 下）、同一父节点下大小写重复的键（整份文件；若 `--config` 指向与主程序共用的配置文件，非 webhook 部分也会触发）。
- 配置文件解析失败的报错只保留行号和白名单里的固定原因，不回显任何原文（之前会原样带出 viper/yaml 的报错，可能含取值）。
- `webhook.port` 只去首尾 ASCII 空白：首尾是 NBSP、`\v`、全角空格等的端口（之前被 `TrimSpace` 去掉后接受）现在**启动失败**。
- URL 字段（`front_domain`、`backend_domain`、`alert_manager_api`、`im_feishu.group_message_api`、`im_feishu.tenant_access_token_api`）去首尾空白，
  前四个再去结尾 `/`；依赖结尾斜杠或首尾空白的部署需检查（拼接处本来就以 `/` 开头，去掉后不会少斜杠）。
- `.json` 配置中的 `\/` 与 UTF-16 代理对转义现在可以正常加载（之前原文扫描按解析错误拒绝）。
- 报错描述改为按来源：配置文件里写错时写 `配置文件中的 KEY`，不再写成 `KEY（环境变量 ENV）`；报错前缀由 `Webhook 配置类型错误` 改为 `Webhook 配置写法或类型错误`。
  依赖报错原文的告警规则需要同步。
- 仓库 `webhook.yaml` 删除了 7 个没有代码读取的未知键，启动不再出现对应的 `[配置告警]`。
- 飞书群机器人地址默认值改为 `https://open.feishu.cn/open-apis/bot/v2/hook`，拼接时去掉结尾斜杠：之前用默认值时群消息发往错误的 `im/v1/messages/<token>`，
  用仓库 `webhook.yaml` 时拼成 `hook//<token>`。通过 `WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API` 设置了其他地址的部署不受影响（但若设为 `im/v1/messages` 仍发不出去）。
- Kubernetes 中同命名空间有名为 `webhook` 的 Service 时，注入的 `WEBHOOK_PORT=tcp://…` 会让 webhook 启动失败（见上文“Kubernetes 注意”）。
- 主后端（`SERVER_PORT`）和 webhook 默认都是 8889；部署在同一台机器或共享网络命名空间时，**用 `WEBHOOK_PORT` 错开 webhook 端口的做法从本提交起成立**。
