# 1.0 开发变更日志

## 2026-10-23：发布入口基线修复

- 统一 AIOps 仓库路径大小写，修复 Linux 发布构建、启动预检与备份清单路径
- 新增隔离的发布工具回归，核验私有密钥生成权限、防覆盖、启动顺序与缺失依赖拒绝
- 将工具测试接入 CI/发布流程，更新三仓部署目录说明

本阶段不代表 1.0 生产就绪；其余验收项见 `verification.md`。

- 远端验证发现新增工具测试排在 Checkout 之前，已修正顺序；本地工具测试保持通过

## 2026-10-23：监控采集证书与服务发现

- Kubernetes 指标采集恢复 TLS 证书校验，保留配置的目标 CA 和 token
- Kubernetes 服务发现由 kubeconfig 或集群内身份提供连接配置，移除与 Prometheus 上游契约互斥的自定义 HTTP 设置
- 回归通过：生成的 kubeconfig/集群内两种服务发现配置均可被 Prometheus 解析，实际 TLS 请求拒绝未信任证书
- 自签证书采集目标须配置可信 CA，不再静默跳过校验

## 2026-10-23：定时任务调度一致性

- 跟踪 Asynq 实际返回的条目 ID，修复重载重复注册、立即删除无法注销的问题
- 保留未变化的条目，不重置下一次运行时间；只删除本管理器持有的失效条目
- 使用完整启用列表，移除 1000 条截断；数据库查询失败保留原有调度
- 实际 Asynq 内存注册/注销回归与 race 检查通过，无 Redis 或生产任务执行
- CI/发布冒烟扩展至 cron handler，覆盖实际子进程输出与 HTTP 边界

## 2026-10-23：脚本执行运行时

- Node.js 使用 `-e` 执行脚本，替代仅语法检查的 `-c`，运行错误按真实失败返回
- 发布后端镜像包含 Bash、Python3、Node，匹配已有脚本类型契约
- `go test -race -tags=smoke ./internal/cron/handler` 通过：真实 Bash/Python/Node 输出、Node 运行异常、取消与已有输出边界

## 2026-10-23：依赖就绪与可靠发布验收

- 后端新增并发检查数据库和 Redis 的 `/ready`，共享两秒期限，失败返回通用 503；`/health` 保留独立存活检查
- 注入现有 Redis 连接并启用请求上下文超时，更新 Wire、鉴权放行、Compose 健康探针及网关路由
- 发布 smoke 必须同时通过健康与就绪检查；所有准入判断改为显式运行时检查，避免 Python `-O` 绕过 assert
- 本机假故障 HTTP 服务回归覆盖健康/就绪失败及正常/优化模式，不输出测试密钥

## 2026-10-23：运行中任务与人工停用

- 已推送 `08f1632`：执行权改为状态 CAS，防止重复 claim；运行中仍保留调度条目，完成时不覆盖人工禁用
- 完结状态使用独立有界上下文，避免任务取消后遗留 Running
- 无数据库 schema 变更；进程硬崩溃后的任务状态恢复仍需要租约策略，不能盲目重置多副本运行状态
- CI/标签发布纳入实际 SQLite DAO 与 Asynq 的 scheduler smoke，详见 `cron-state.md`

## 2026-10-23：可重复容器构建与模型配置

- Go 镜像使用 BuildKit 模块/编译缓存，移除每轮强制重编译全部依赖；实际 Linux arm64 镜像构建成功
- 实际镜像中的 Bash/Python/Node 执行输出全部核验成功
- Compose 支持独立配置任务模型与嵌入模型；密钥仅保存在已忽略的私有部署 `.env`
- 硅基流动实际模型目录可访问，所选 Qwen 工具调用成功；项目内助手/RAG 联调结果另记

## 2026-10-23：连接资源与进程回收

- `c5efb05`：Unix 独立进程组取消与有限管道等待，覆盖启动后取消、deadline、父进程提前退出的真实进程回归；边界见 `cron-process.md`
- `f546b13`、`f18f227`：MySQL 有界池与共享启动期限，版本探测保留，显式 DSN 超时优先，启动失败关闭资源，错误不输出 DSN
- Compose 转发六项 MySQL 资源配置，支持按数据库预算和初次建表耗时调整

## 2026-10-23：三仓持续集成

- 新增隔离 Compose 集成工作流，记录实际三仓 SHA，真实执行网关/会话、RBAC、工单全流程和 MCP 认证/计算器验收
- 每次运行生成私有随机配置，不使用生产或模型供应商密钥；结束仅清理当前唯一 Compose 项目
- 源码中不包含凭据，不展开完整 Compose 环境到日志；首次远端执行结果待记录

## 2026-10-23：构建上下文隔离

- 明确递归排除部署私有 `.env`、本地备份和清单，避免进入后端构建上下文与后续缓存
- 使用实际 Docker COPY 构建检查：私有配置与备份目录均不存在，公开 `go.mod` 仍在上下文，检查通过

## 2026-10-23：SSH 输入与连接边界

- 两个 SSH 客户端统一验证/转义公钥，修复注释中引号插入远端 shell 命令的风险，保留合法选项语义
- 拨号期限覆盖握手并正确回收失败连接，使用 IPv6 安全地址拼接；日志移除原始命令与输出
- 新增只读 known_hosts 可选发布模板；真实临时 shell 与静默 TCP 服务专项回归通过
- 范围与未执行的远端主机验收见 `ssh-boundaries.md`

## 2026-10-23：持续交付队列

- 分支 CI 与三仓集成按工作流/分支分组，取消被更新提交取代的运行，避免连续原子推送堆积旧构建
- 标签发布流程保持原有执行方式

## 2026-10-23：跨区域构建修复

- 首轮远端三仓集成在固定 goproxy.cn 下载 Go 依赖时收到 HTTP/2 GOAWAY，构建失败；前端阶段已完成
- Go 模块代理改为可配置，默认使用官方源，区域镜像按部署环境显式选择，不关闭 TLS 或依赖校验

## 2026-10-23：WebSocket 认证传输

- 服务端支持固定 cloudops-v1 + 唯一认证子协议，避免新版浏览器把JWT放入URL，保留真实升级请求的历史兼容格式
- 修复路径含 execute 的普通HTTP任务接口被误判为WebSocket查询认证
- 网关访问日志不记录查询字符串/认证头，实际Nginx配置检查与重载通过
- 解析/登录传输/DI race通过，客户端迁移与真实Pod exec联调随后验收

## 2026-10-23：独立复审后的认证补强

- 代理保留 Host 端口，避免同源非默认端口被当作跨源拒绝
- 浏览器凭据只用于注册终端路由的完整 GET 握手，普通 POST 即使伪造 Upgrade 头也不能改用查询/子协议认证
- 替换默认请求转储 Recovery，panic 不记录认证头、请求体/查询或异常原文；保留安全栈与固定路由
- 真正 101/固定协议协商、伪升级 POST、debug/release 恢复日志隐私及 race 回归通过

## 2026-10-23：AI 用户身份边界

- AI代理从已验证claims注入正整数用户身份，外来同名头不可冒用，缺主体拒绝
- 转发时剥离浏览器JWT所有已知载体，保留公开子协议/业务参数，支持既有assistant stream完整握手
- 实际SSE与WS反向代理身份/凭据隔离race smoke通过；AIOps会话归属配套单独交付

## 2026-10-23：真实崩溃恢复与持续验收门槛

- 固化仅创建随机独立 Compose 项目的故障验收脚本，固定本地镜像 ID，并核对项目标签后才 SIGKILL；默认拒绝执行，finally 仅清理本次项目
- 真实 MySQL/Redis 硬崩溃原型 87 秒恢复；固化脚本再验 88 秒恢复，均确认任务成功完成并精确清理
- 发布工具 17 项通过（含优化模式的执行开关保护）；终端会话隔离、任务租约与 Pod 核心 race/smoke 目标接入分支及标签门槛
- 追加真实模型/RAG、kind 1.35.8、跨账号会话归属、Pod PTY 及46表新快照恢复证据；旧状态文档明确为历史实现

## 2026-10-23：工作区权限边界澄清

- 依据实际全局业务模型及接口级RBAC，修正README中尚未实现的同部署多组织隔离承诺
- 明确AI会话按用户隔离，以及监控配置详情权限包含认证配置，避免将其误授予普通业务只读角色
- 本阶段仅文档变更；接口行为未改变，真实权限与会话验证见verification记录

## 2026-10-23：Kubernetes 列表筛选与安全分页

- Service列表支持status=1/2/3，沿用原有类型派生状态；补齐已声明labels条件，所有过滤均在分页及总数计算前执行
- ConfigMap列表新增data_key，对data及binary_data键名不区分大小写包含匹配，不检索内容，与原名称/namespace/labels条件并用
- 两类列表复用通用分页，先除后乘避免超大页码/页大小溢出导致负切片索引
- service/utils实际专项race通过，包含HTTP查询绑定、无效状态拒绝、跨页总数、二进制键名、标签缺失及MaxInt边界；真实kind联调将在镜像更新后记录

## Cron 重试预算与配置编辑隔离

- 手动与周期任务精确传递 max_retry，创建省略默认 3、显式 0 保留，更新省略保留旧值
- 租约协调等待不消耗失败预算；硬崩溃恢复遵守 SDK 同一有限预算，详见 [cron-retry.md](cron-retry.md)
- 配置编辑限定字段，防止覆盖正在执行的状态、租约与历史输出
- 验证：Cron 与 DI 的 smoke/race、go vet 通过，真实 SQLite 验证显式 0 与运行中编辑

## Prometheus 采集认证与发现配置修复

- 三种发现方式统一生成目标认证和 CA，校验互斥来源；仅操作配置导出恢复真实令牌，不修改 SDK 全局脱敏开关
- 修复 HTTP-SD 节点 ID 保存、地址默认端口和 release 容器地址，保留原有查询参数
- 完整生成配置参与缓存摘要，令牌等作业字段变化能刷新
- 验证：本地真实 HTTPS 认证/CA、配置解析、缓存轮换与日志检查、race/vet、全仓 go test 通过；实际发布 v3.2.1 promtool 无网络临时容器验收通过，见 [prometheus-scrape-auth.md](prometheus-scrape-auth.md)

## 公共 HTTP-SD 输入限量

- 双路由保留 Prometheus raw target-group 成功协议，节点 ID 保序去重后读取，每次最多 100 个不同节点
- 参数和查询串长度有界，无效输入在数据库读取前返回 400；服务层同样校验边界
- 修复 IPv6 目标地址编码，保留稳定目标顺序与请求取消
- 真实 Gin/httptest 与服务回归、race/vet 通过，详见 [http-sd-input.md](http-sd-input.md)

## 监控凭据读取脱敏与明确更新

- list/detail 仅返回 has_bearer_token，旧明文列不迁移，操作配置生成保留真实值
- 更新令牌支持省略保留、空串清除和非空替换，事务锁内合并当前凭据并校验来源互斥，配置字段可实际清空
- 配套前端使用明确 keep/replace/clear 语义；更新省略令牌不会回写并发轮换前的快照
- 先红后绿的真实 SQLite/race 验证清空和来源切换；HTTP回归验证双读取、请求绑定、审计/访问日志无正文；全仓 go test 与 vet 通过，详见 [prometheus-credentials.md](prometheus-credentials.md)

## 数据库日志与启动错误边界

- 主后端及独立webhook统一使用轻量安全GORM logger，只记录固定事件、耗时、行数与错误类型，SQL/参数/驱动错误原文与框架变参不进入日志
- 连接/ORM/迁移/管理员初始化与webhook panic 使用静态阶段和安全类型，同时保留Unwrap供errors.Is/As定位
- 真实SQLite约束/慢查询及Bootstrap错误先红后绿，真实MySQL驱动拨号前错误覆盖webhook；专项race/vet、全仓test/build通过，Wire实际生成，见 [database-logging.md](database-logging.md)

## SSH终端传输基础

- 长消息按reader缓冲区完整分段交付，64KiB消息先验收后执行，旧逐消息换行格式保留
- 两条实现共享串行输出及有界写期限，legacy补上下文取消与断开清理，输入关闭原因不入日志
- 本地真实WS/SSH fixture、race/vet通过，见 [ssh-terminal-transport.md](ssh-terminal-transport.md)

## 2026-10-23：发布验证器与日志隐私门槛

- 分支和标签流程固定拉取实际发布使用的Prometheus v3.2.1，通过CLOUDOPS_PROMTOOL_IMAGE激活真实promtool配置校验，不再让镜像契约测试仅依赖本机手动开关
- 把采集配置本地HTTPS、主后端安全数据库日志及webhook数据库日志smoke包加入发布门槛
- 相同环境开关下三包race/smoke通过，工作流YAML解析通过；未创建标签或发布GitHub Release

## SSH显式PTY模式

- 双终端通过terminal_mode=pty-v1显式使用binary stdin/output与text resize，cloudops-v1鉴权及legacy入口保留
- 远端ECHO=1、尺寸验限与window-change，压缩后输入与控制消息先完整验限再交付
- 真实SSH协议fixture覆盖Ctrl+C/方向键/64KiB/二进制并发输出、resize与关闭/deadline；真实WS压缩/非法帧/日志及双handler回归、race/vet、全仓test/build通过，Wire实际生成，见 [ssh-pty.md](ssh-pty.md)

## Pod终端解压后输入边界

- 保留JSON stdin/resize/stdout与raw兼容格式，限制解压后message512KiB、stdin64KiB、resize1024B
- 完整消息先验限后交付，压缩/未压缩超限关闭自身连接并取消exec，首读零部分交付且无EOT
- 真实WS先红后绿，合法JSON六倍转义、并发/心跳/关闭/resize既有回归及race/vet与全仓build通过，见 [pod-terminal-input-limits.md](pod-terminal-input-limits.md)

## 完整启动迁移与YAML数据库门槛

- 注册YAML模板/任务、通知队列和账号区域；普通AST测试核对所有TableName模型防漏，无需为SQLite索引作用域改合法MySQL schema
- 随机限权MySQL库验证完整/幂等迁移和YAML CRUD，finally清理0；拒绝skip/未匹配测试冒充验收
- 三仓integration增加模型注册、真实隔离MySQL与启动后YAML双list业务码门槛
- 先红后绿、真实MySQL、DI smoke/race/vet及全仓Go测试通过，见 [startup-schema.md](startup-schema.md)；共享镜像由主流程统一更新

## 工单通知渠道数据库筛选

- 通知list新增可选channel枚举email/feishu/sms/webhook，分页保留page_size
- MySQL8.4与SQLite按顶层JSON字符串精确匹配，在count和分页前应用，非LIKE或当前页筛选
- 同数据在真实SQLite与隔离MySQL覆盖多页/总数、大小写/子串/嵌套/非数组/无效JSON与叠加条件；先红后绿、race/vet、全仓test/build通过；integration增加MySQL门槛，见 [notification-channel-filter.md](notification-channel-filter.md)

## 云账户地域差量与原子更新

- 父账户行锁内统一更新基本信息/凭据/地域，保留原地域ID/状态/关联，安全恢复历史soft-delete行
- 拒绝移除仍有未软删除云资源关联的地域，失败时整笔回滚；省略regions不覆盖较新集合
- 旧源码overlay在隔离MySQL真实复现1062；修复后MySQL唯一约束/并发/失败回滚、SQLite、race/vet与全仓测试通过，integration增加门槛，见 [cloud-account-region-update.md](cloud-account-region-update.md)

## 2026-10-23：AI 接口细分授权目录

- 将实际 65 项 AI HTTP/WebSocket 路由补入权限目录，分别授权助手、会话和修复确认，不自动分配角色或覆盖既有元数据
- 代理接口参数授权严格匹配单个非空路径段；保留既有显式整模块授权并说明升级时如何收窄
- 三仓 CI 对账实际 AIOps 路由，覆盖隐藏别名和 WebSocket，避免新路由遗漏权限项
- 验证：先红复现目录缺项与参数授权失败，修复后 app/middleware 全包 smoke race 通过，容器实际路由 65 项对账通过

## 2026-10-23：本地主机元数据合同修复

- 局部更新只保存显式字段，支持清空描述/标签、修改用户名，不重置运行状态或覆盖并发轮换的新凭据
- 本地主机 JSON 不再返回私钥；内部 SSH/定时任务凭据读取保留
- 验证：先红复现后 SQLite 持久化与旧快照回归通过，tree service/dao/model/cron executor race 及全仓 build 通过

## 2026-10-23：云资源凭据审计脱敏

- 共用审计 DAO 对 key/password 旧值、新值脱敏，历史记录 JSON 同样不回显凭据
- 启动时清理历史审计副本，保留事件与时间，重复执行幂等，不修改资源实际凭据
- 验证：先红复现两项泄漏，DAO/service/di race 通过，隔离 MySQL 完整迁移与历史清理、普通审计保留和 YAML CRUD 通过

## 2026-10-23：真实 AI 细分角色网关验收

- RBAC 集成流程覆盖实际 AI 权限目录、参数 GET 下游404、越界/模型/修复/通知403和立即撤权
- 账号验收脚本限定回环网关，真实创建和清理唯一临时用户、角色；默认流程不调用模型或基础设施
- 验证：最新隔离三服务镜像上全部通过；另行真实硅基流动→只读MCP计算器→模型14往返通过

## 2026-10-23：通知配置显式零值与清空

- 创建/更新准确保留显式0重试，省略创建默认3；创建零覆盖失败整笔回滚
- 更新描述/主题支持清空，正文不可显式清空，默认标志省略保留且拒绝0，旧非法0升级规范为非默认2
- 验证：SQLite先红后绿、workorder/di race、限权MySQL presence/事务回滚与全量迁移回归通过，临时资源清理完成

## 2026-10-23：阻止未实现的通知高级触发提前发送

- 1.0自动通知只开放immediate，写入未实现的延迟/定时/条件/重复参数明确拒绝
- 历史非即时配置在收件人查询前阻止自动投递；允许禁用与元数据维护，不允许直接重新启用
- 验证：非即时写入/历史自动发送/无效高级参数先红后绿，workorder全模块race与真实本机应用收件箱回归通过

## 2026-10-23：AIOps专用飞书机器人配置接入

- 私有FEISHU_WEBHOOK_URL映射到AIOps服务端，默认模板只包含空占位，不向MCP或前端注入
- 保留显式通知开关，区分机器人Webhook与工单应用消息通道，记录官方HTTP/业务成功合同
- 验证：17项发布工具回归、Compose config与diff检查通过；真实单发验收在最新AIOps镜像重建后单独记录

## 2026-10-23：MCP真实传输UTC时间回归

- 三仓集成smoke增加实际MCP时间工具响应ISO解析与UTC offset检查，防止重复时区标识回归
- 最新AIOps/MCP4dc31ac实际内网鉴权、计算器14、UTC ISO与匿名401全部通过，无模型或基础设施调用
- 专用飞书机器人另经JWT网关唯一实发通过，上游HTTP200/业务code0，详细安全记录在AIOps docs/release/feishu-delivery.md；未重发

## 2026-10-23：通知范围、重试执行与真实 worker

- 工单从已验证表单或模板保存不可修改的来源快照；通知按流程、分类、模板共同匹配，历史未知来源不命中具体范围
- 历史 immediate 配置携带高级触发参数也禁止自动投递和重新启用，避免未实现条件被提前执行
- 0 次重试只投递一次；配置重试次数和分钟间隔贯穿业务发送及真实 Asynq worker，异步仅单层重试；保留 Cron 默认队列、失败分类及退避
- 修复 MySQL 相同值更新被误判不存在；飞书日志及持久化诊断移除消息、地址、URL 和供应商原始正文
- 四包 race、真实隔离 MySQL 来源/幂等回归、独立 Redis 与本地 HTTP 收件箱通过；CI 纳入以上集成测试，未向外部重复发送通知

## 2026-10-23：1.0 交付验收归档

- 58/58 页面统一完成，627 前端单测与类型检查/生产构建通过；同源生产浏览器两轮13+2项、完整56静态入口与独立SSH两入口验证
- 最终 Go 默认全量、race/smoke、17发布工具及 Python488项通过；CI补全真实MySQL来源/幂等、Redis通知worker与全量Python静态检查
- 记录真实模型/RAG、唯一一次飞书机器人接受、真实kind/SSH、独立worker硬崩溃恢复97秒；测试替身和外部服务证据分别说明
- 清理仅本轮SSH/kind及私钥凭据；最终8服务healthy，49表限权恢复成功，应用上一版镜像回滚与返回最终版通过
- 固定产品源码/镜像/CI与支持边界，旧过程验收归档；未部署生产、未发布标签，备份仍明确不包含Redis和上传文档卷

## 2026-10-29：周期任务归档后继续调度

- 移除周期任务固定 TaskID，由 Asynq 为每次入队生成独立 ID，保留重试预算、调度条目注销和数据库租约控制
- 在独立 Redis 验证失败任务归档后下一周期仍能入队，新增回归已接入三仓集成 CI

## 2026-10-29：工单通知持久投递

- 工单状态变更和通知事件快照在同一数据库事务保存，删除事件使用删除前快照；工作器按租约领取并保留首次投递计划和逐收件人结果
- 已确认拒绝按配置预算重试；结果不确定时停止自动重发，通过独立权限和审计记录进行人工核对处置；SMTP错误因回执阶段无法确定均按不确定处理
- 增加受理人清空、事件保留策略、可信操作者和邮件日志脱敏；race、隔离MySQL、真实本地SMTP/HTTP断回执与事务回滚验证通过，CI纳入MySQL outbox回归

## 2026-10-29：HTTP 定时任务超时合同

- 由任务配置的期限控制 HTTP 总超时，移除客户端固定30秒上限，保留默认30秒和调用方取消
- 日志与错误隐藏完整URL及上游异常正文；保留取消语义，本地race回归通过

## 2026-10-29：调度恢复与表达式校验

- 首次加载失败自动退避重试并暴露状态；无效配置先拒绝，替换注册失败尝试恢复原调度
- 创建、更新、启用和预览统一实际 Asynq 表达式和UTC时区合同；新增真实Gin/SQLite与race回归

- Cron 表达式解析补充数据库 varchar(100) 长度边界，避免预览通过但持久化失败；对应race测试通过

## 2026-10-29：云厂商枚举文档修正

- 修正文档中的 AWS 枚举为3、华为云为4，与后端及前端实际合同一致

## 2026-10-29：禁用云账号验证边界

- 禁用账户在凭据解密及云厂商调用前拒绝验证，与同步和实时地域查询的禁用规则一致；SQLite race回归通过

## 2026-10-29：内置任务真实执行与就绪状态

- 手动和定时共用真实值班/Kubernetes/Prometheus执行路径与租约，修复假成功、分页截断和错误传播
- 保留既有启停/表达式并动态重载，停止等待回调；历史非法表达式拒绝且不触发解析panic
- 调度状态纳入 `/ready`，恢复后自动就绪；升级时数据库中既有schedule将实际生效，详见cron-builtin-scheduling.md
- Cron/App/DI race、真实SQLite、分页与健康路由smoke通过，未连接真实集群

## 2026-10-29：云同步地域边界

- 同步前完整验证显式地域ID的存在性、账户归属、启用状态与重复值；任何无效项均在解密及云调用之前拒绝
- 空地域列表仍选择该账户全部启用地域；替身race回归覆盖混合无效输入且确认不进入本地清理

## 2026-10-29：工单编辑动作合同

- 可用动作按更新接口相同的草稿/待处理状态返回 edit，步骤自定义值不得绕过状态限制；实际更新仍由原有RBAC授权
- 真实SQLite及Gin鉴权回归覆盖无写权限、跨受理人及状态约束

## 2026-10-29：工单动作只读查询

- 查询当前步骤及可用动作时不再回写工单；流程状态变化仍由对应事务持久化
- SQLite race验证查询不修改步骤或更新时间，已有18个流程成功/回滚场景继续通过

## 2026-10-29：华为云计算只读客户端

- 新增官方ECS SDK客户端、显式地域/project与可取消HTTP、指定ID分批和失败页拒绝；隔离race及依赖审计通过
- 尚未开放账户/同步入口；企业项目范围无法证明完整时，无ID增量与全量必须保持拒绝

## 2026-10-29：云临时凭据与华为地域接线

- AWS/腾讯临时凭据支持令牌加密、UTC到期和成组CAS轮换；华为通过地域project ID接入指定实例增量查询
- SDK只使用显式凭据，响应/URL/日志不回显密钥；无ID华为同步与临时凭据全量保持拒绝，待完整性和事务化保障后开放
- tree race、SDK签名替身、真实handler/RBAC及隔离MySQL迁移/并发回归通过；升级须先备份迁移并统一版本

## 2026-10-29：高级通知与排期管理

- 延迟、UTC定时、快照条件和有限重复形成持久计划；取消、重发和结果核对使用精确权限、事务CAS与操作者审计
- 历史高级规则默认不执行，需重新验证；配置更新使用版本检查，投递目标仅提供受控脱敏标识
- 启动要求固定MySQL时区并验证数据库标记；已有通知时间数据必须先停写、备份、核对迁移，不能与旧实例混跑，详见通知合同
- 隔离MySQL并发/时区、workorder race及目录回归通过；未向真实通道发送

## 2026-10-29：Azure计算只读客户端

- 新增显式服务主体、限定端点/只读请求、禁止自动注册、完整分页和指定VM ID核验；隔离SDK race及依赖审计通过
- 尚未接入账户/同步入口；未经订阅覆盖证明不得开放全量或无ID同步

## 2026-10-29：GCP计算只读客户端

- 新增显式服务账号、固定令牌端点及只读Compute查询，校验指定实例、完整分页和取消；隔离SDK/REST回归通过
- 未验证项目权限完整范围，无ID与全量同步继续拒绝；账户工厂与前端尚待接线

## 2026-10-29：Azure/GCP账户凭据

- 两家云使用显式完整凭据对象、加密存储与版本CAS轮换，普通响应不回显；旧AK/SK保持兼容
- 隔离MySQL迁移/回滚/并发与Gin鉴权替身测试通过；指定ID同步与前端接线继续单独验证

- 多云枚举合同：三仓集成 CI 对照 Go 实际枚举与 Web `CloudProvider`，发现数值变化、缺项、重复或新增未对齐项即失败；并包含合同检查器自身的负例回归。

- 云资源同步：增加按地域的租约、事务和可核对结果；AWS/腾讯临时凭据全量支持事务回滚，所有同步审计身份取可信登录信息。新增隔离 MySQL 的事务、租约和临时凭据回归。

- 工单站内消息：增加用户隔离的历史、未读、已读及删除，事件投递与账本事务去重；普通评论与通知事件原子保存，并阻止伪造系统评论。

- 预测回测入口按 POST 和实际路径登记独立 AIOps 网关权限；未授权角色不能调用。

- 非计算资源首批只读合同：阿里云 RDS、传统 CLB 和缓存按指定 ID 精确查询，禁止将不完整列表当成全量删除依据；API 与持久化仍在后续阶段。

- 自动修复持久化审批增加历史、操作和计划的独立 GET 网关权限，发布组合明确使用 Redis `noeviction` 策略。

- 托管资源增加独立存储、按地域和指定 ID 的增量对账及用户授权历史；账户与地域删除同步检查托管资产，并保留结果未知的人工核对边界。

- 增加经HMAC认证的停写物理备份和新卷隔离恢复工具，覆盖MySQL、Redis全目录、AIOps数据与私有配置；安全单测及真实无网络数据卷恢复演练接入集成CI。

- 托管同步新增已有历史的人工核对审计接口，独立精确授权与地域租约fencing；服务端阻止未核对历史直接重试，迟到worker不能覆盖审计。

- 托管同步支持用户隔离的持久操作ID、防重复领取及安全结果查询；MySQL开启clientFoundRows时仍按独立持久令牌判定首次执行，不确定操作不自动重放。

- 托管操作核对可为缺失记录写入防重墓碑，保留原结果和关联历史；停用范围无需重新启用即可审计，已领取旧请求在提交前再次校验持久状态。

- 托管同步增加跨用户范围级持久接纳门禁和有界操作列表；默认用户隔离，可信超管可显式跨用户核对并保留原操作人与实际核对人，禁止伪造权限参数。

- 操作列表增加不泄露其他用户标识的scope_blocked汇总，供前端在本人列表为空或恢复指针丢失时继续安全阻断；未知历史状态同样阻断。

- 增加版本化迁移基础引擎：单连接数据库锁、校验和及started/applied/failed状态，脏或不确定结果禁止自动重跑；只读Verify及独立MySQL角色/中断/部分DDL/panic回归接入CI，应用基线与启动接线仍在后续阶段。


## 2026-10-30：清单闭环与最终固定候选

- 完成原 F01–F14 与明确范围的 D01–D05；补齐持久通知/会话、多云、RAG 原子发布和各模块回归，保留实际支持模式及外部验收限制。
- 补充版本化迁移、严格旧库采用与 safe-v2 已知历史升级、迁移/运行分权、全量认证备份及旧库物理恢复演练；拒绝隐藏触发器与不明确迁移来源。
- 完成 typed 配置/密钥接线、通知业务码及部分失败确认、错误保密和 SSH 密文容量校验；不改变既有密文算法。
- 固定 eec9582a/d0f098e/c2de23e：全量离线门禁、手动集成 CI、本机归档源码构建及五场景浏览器通过；导出精确 arm64 镜像并登记校验和。详见[候选收据](candidates/2026-10-30-eec9582a/README.md)。
- 更新当前完成度及中英文入口，历史候选不覆盖。保留全仓 ESLint/渐进配置债，不宣称生产、其它架构或真实外部依赖已验；未上线。


## 2026-10-31：配置技术债第一阶段

- Webhook consumer/cache/content/robot 显式接收 typed 配置并复制所需标量，移除剩余13处业务 Viper 读取；保留动态令牌刷新、既有请求合同和空凭据禁外呼。
- Cron 服务/调度/执行器显式注入共享动态 policy，逐次读取开关与环境覆盖，nil 拒绝本地执行；删除隐式全局 fallback，保留可撤销门禁。
- 两处 Wire 实际再生，增加生产 Viper 导入架构约束；全 Go 测试/vet、相关 race 和所有集成/冒烟标签编译通过。详见[配置依赖边界](configuration-boundary.md)。未上线，未执行真实通知或生产任务。

- 推送后 Simple CI 发现发布工具仍按旧 Viper 源码取值模式匹配群通知地址；现修为识别显式 typed 字段，保留旧格式/trim/错误地址反例并增加字段变异拒绝。强制 Compose 的完整脚本158项通过、零跳过；实际 Go 配置快照到Fake HTTP请求的 race 复验通过。此修复仅调整测试合同，不恢复业务全局读取。


## 2026-10-31：固定技术债续做后的前端来源

- 将三仓集成前端固定为 `e68fb072a344373f6429b855007c26413c6a2166`，AIOps来源保持不变；历史候选不覆盖，新增组合由本次CI验证。
- 更新当前开发证据与剩余边界：前端2153项完整测试、类型和构建通过，末尾Secret收尾474项另验；ESLint仍有12254错误与5警告，不宣称全仓清零。未上线。

- 最终格式兼容阶段固定前端 `dcaa59f1eb7f78131c707a97cd41b350ebdadeeb`：处理真实Prettier/Vue闭合换行修复冲突，12配置回归与2160全量测试、类型检查通过；完整扫描11968错误/5警告，仍未清零。只有重复格式校验职责调整，语义安全规则保留；无产品源码/生产变更。

- 最后测试标记收尾固定前端 `daec9407c09270a6d2d59d0591b54ef16552bd85`：6处既有any上的无效非空断言移除，7项相关测试、类型及运行代码等价核对通过；完整扫描11962错误/5警告，其余检查保持原阶段范围。

## 2026-11-03：来源配置加载与实例边界

- 主HTTP CORS与WebSocket升级器统一使用已加载的server.allowed_origins，修复YAML字段不生效及实例后续受进程环境改变的问题。
- 保留环境覆盖、空值回退、WS同源/无Origin及通用库独立动态适配合同；nil/空配置拒绝跨域。
- 真实加载优先级、Gin/WS精确匹配和快照回归通过；Go全量、vet、相关三包race、本地WS握手、全标签编译及158项发布工具测试通过，未上线。

## 2026-11-03：配置修复及lint门禁来源锁

- 集成源码锁更新至Web 7af88de（已清四类lint回归门禁、2191项单元测试）与AIOps ba1876a（ENV配置失败边界、生产别名鉴权、provider模型快照、巡检开关闭环）。
- AIOps两模块冻结后全量2498项通过、33项既有条件跳过；Go跨域配置修复与本地回归已在54ae55d0独立推送。
- 此提交触发固定三仓隔离CI，结果尚待验收；不代表浏览器已通过、历史镜像已更新或已上线。完整lint仍11430 errors/5 warnings，Vue TS规则覆盖缺口继续处理。

## 2026-11-03：前端P1闭环整合来源锁

- 固定Web c8d08c3及AI ba1876a；包含告警、规则/配置表单、用户角色、分类、云账户和K8s回滚/历史作用域修复。
- Web冻结后全量2867项、类型检查及11项构建通过，四类lint边界零回归；完整lint仍10503 errors/5 warnings、SFC仍33语法盲点，不宣称全部完成。
- 更新当前验收边界，明确旧中间CI41578362287成功不替代新组合结果；源码锁5项回归通过，未上线。

## 2026-11-03：拒绝实体零ID与请求目标重定向

- 保留原authoritative绑定，修复URI0未被快照保护而借用body合法ID的错误目标更新/回滚；普通及cluster路径实体ID统一正整数。
- 虚拟树父0使用显式非负解析，body/query的部门根/解绑等零值合同不变；补5个工单解析失败空200为规范HTTP400。
- 新53场景、最终Go全量/vet、相关5包race和全标签只编译通过，新增测试无真实网络或回滚；未改变数据表、镜像或上线。

## 2026-11-03：Static采集端口合同

- 修复裸IP/主机保存了独立port却在生成器中被忽略，按标准host:port构造Static目标；已有显式嵌入端口原文优先，0合同不重解释为9090。
- IPv6正确补方括号，不把空/非法主机改成新的本地目标；API/存储/其它SD不改。
- 17表组、真实生成地址→Prometheus解析→本地CA/Bearer/TLS请求闭环、缓存race和最终Go全仓test/vet通过。无真实监控访问或上线。

## 2026-11-03：第二整合检查点来源锁

- 固定Web8daf2be及AIba1876a，纳入auth epoch、流程/采集真实校验与地域独立确认安全闭环；3069全量、类型/11构建通过。
- 完整lint10443/5及SFC11剩余明确记录，不将四规则门禁通过当作full lint清零。
- Go最新业务已含18f8实体路径及1e206 Static独立端口，新的完整来源组合由本提交CI验收；来源锁5项通过，未上线。

## 2026-11-03：服务树显式移根

- DTO用父ID指针区分未提供/null与显式0，恢复原服务已允许的移根；URI实体正数和self/cycle规则不变。
- 补Gin空form隐式转0与坏编码缓存边界，不把空输入误执行为移根；不改global binder/schema。
- 47个真实HTTP/Fake场景、相关race/vet和最终Go全仓test/vet、全标签只编译通过，无真实节点或DB操作。

## 2026-11-03：第三整合检查点来源锁

- 固定Web8be2a2c与AIba1876a；纳入阶段33—46的表单、通知、调班、真实池目录和SFC门禁闭环，历史pool_id保留人工核对边界。
- Web冻结后3389项全量与三类类型检查通过，11项构建无缓存全部通过；普通lint10177/5仍未清零，五保护类及parser/tool0。
- 旧bc2/8daf组合41602391719已核对完整SHA和5浏览器收据；本次新来源独立等CI，不更新历史镜像或部署。

## 2026-11-03：录制规则保存与发布合同

- 统一真实UTF-8名称、vector/scalar及标签校验；标签进入生成配置和版本指纹，同一次DAO快照生成，修复合法分隔符造成的哈希碰撞。
- 坏条目隔离、全坏快照保留旧配置、真空列表显式清除；必要缓存/成员/持久化/清理完成后才确认哈希，失败可重试，不误删当前last-good。
- 新108叶场景、全仓test/vet、相关3包race及全标签只编译通过；全Fake且真实解析器，无数据迁移或上线。详情见[录制规则配置合同](recording-rule-config.md)。
- 第三检查点45d64768/8be2a2c/ba1876a组合41615904232已成功，完整SHA及5浏览器零skip/fail/retry已核验；该证据不外推本次后端新代码，新组合由本提交CI验证。

## 2026-11-03：第四源码检查点

- 固定Web6cd88fe/AIba1876a，纳入值班CRUD、本地资源A/B和显式本地演示；3560全量/三类type/11build与27Node专项通过，普通lint10164/5仍未清零。
- 本机独立演示库88条可重复样例，真实消息账本及外部任务不伪造；恢复工具仅prepared环境，健康复用已实测，停服恢复未实测。
- 6e5a3032旧组合41623940913完整SHA和5浏览器收据已核验，本次新来源单独验收；不发布镜像、部署或访问生产。

## 2026-11-03：Prometheus主配置失败重试

- 修复配置/成员/持久化/清理失败仍ACK、当前IP写失败被当stale删除、恢复后永久跳过；保留同一快照与原YAML哈希，成功持久化后才确认版本。
- 真空活动job发布有效空scrape停止旧抓取，读取/生成/consumer校验错误保lastgood，错误不回显配置。
- 新21与旧28场景共49定向race通过，全仓test/vet及全标签只编译通过；未操作真实服务或上线。见[发布边界](prometheus-publication.md)。

## 2026-11-03：告警规则池实体统一

- 保存、池名、前端选择/搜索统一真实ScrapePool，与规则生成实体一致；保留pool_id字段，不按同号自动映射历史AlertManager ID。
- Alert99/Scrape7不重合替身验证拒错实体、存储与生成同实体；Go service/DI、Web164、类型和实际Wire生成通过。
- 只处理实体关联阶段，不借此宣称规则构造/发布其它缺口已修；历史人工核对见[专项说明](alert-rule-pool.md)，未迁移或上线。

## 2026-11-03：告警规则构造与标签协议

- 保存/生成共用真实UTF-8、PromQL结果类型、duration、label/annotation/template合同，保0s及合法{}告警名；首个等号后的内容和合法空值保持，坏历史不猜测转换。
- 同名告警按数据库ID分组，完整文件真实parser校验。首次group身份变化可能重新等待for/firing不直接继承，发布人需审核窗口，未重载真实实例。
- Go构造三包专项/race/vet、Web169/类型及独立映射复验通过；发布/hash循环未改，另阶段处理。见[构造合同](alert-rule-construction.md)。

## 2026-11-03：发布ACK回退与稳定身份

- record/Prom发布前持久pending，完成后SHA，修复A→失败B→回A旧ACK跳过；同实例锁实际接入，取消后不继续操作。
- 内部完成键改为独立publication前缀+pool.ID，修复池改名回退和旧名称碰撞；旧Name键不删/不信任，首次升级重发一次。
- 新30叶与旧76纯内存共106定向race、vet及主线程新增用例复验通过；其它发布阶段仍在独立整合，不冒称当前全部全仓集成通过。

## 2026-11-03：AlertManager快照与安全发布

- 启用组同快照内容hash、禁用过滤、真空blackhole、GroupByStr与SDK默认字段正确生效，保公开wrapper及duration合同。
- artifact归属预检保护他池/自定义/歧义键；更名同正文仍更新元数据，必要失败pending不ACK且可恢复，完成键固定PoolID独立namespace。
- 新172叶、定向race204项/子场景、vet、主线程25函数及独立反例复验通过；未连接真实服务或上线。详见[发布说明](alertmanager-publication.md)。

## 2026-11-03：第五源码检查点

- 固定Web9da4556/AIba1876a，纳入API目录隔离、告警实体/构造与三publisher持久pending/稳定ID。
- 冻结基线全Go/vet/三包完整race/全标签只编译、Web3609/三类type/27demo/11build通过；普通lint10089/5仍保留。
- 告警规则发布/hash下一阶段未包含，README补本地演示入口；不发布、不连接生产，新组合由本次CI独立验收。

## 2026-11-03：告警规则安全发布与退役

- 单次活动规则快照用于完整字段指纹和文件生成；合法空规则发布 `groups: []`，合法空目标退役本池配置，读取/构造失败保留有效旧配置。
- 稳定 PoolID 完成键与旧名称哈希分离；pending 先于可见写入，所有必要步骤完成后才 ACK，支持失败回退、更名和重建恢复。
- 分页预检元数据归属，拒绝覆盖自定义/他池/重复配置，精确清理旧目标与无成员索引的本池孤儿。
- 新 14 主测试/81 子场景、全 Go test/vet、相关三包完整 race、全标签只编译通过；独立反例发现的提前空页现已拒绝，不提供跨进程原子性，不调用真实外部服务或上线。见[发布合同](alert-rule-publication.md)。

## 2026-11-03：录制规则空目标与生成物恢复

- 同源哈希仍核对缓存、成员和元数据正文/哈希/状态，修复丢失及漂移；合法空目标精准退役，读取/构造失败保旧。
- 持久pending先于所有可见写入，分页及canonical归属预检拒绝不确定快照，保护custom/他池数据，收回无成员索引的本池旧生成物；非规范名称不猜测来源。
- 新16主/85子、独立4个race反例、全Go test/vet/相关完整race/全标签只编译通过。Web165文件3609全量及11构建通过，来源锁固定10dbbfd/AIba1876a，5项锁定测试通过。
- 后续继续Prometheus主配置与support开关生命周期；不提供多进程原子性，不上线。详见[录制规则合同](recording-rule-config.md)。

## 2026-11-03：Prometheus主配置空目标与漂移恢复

- 全池分页库存预读失败零发布；合法空目标仍读真实任务并执行原生config.Load，通过后才精准清退本池生成物。
- 同源ACK下缺缓存/成员或元数据漂移重新修复，canonical归属预检保护未归属配置，pending/成员/元数据/Redis/清退/ACK完整可重试。
- 新18主/88子、独立4个race反例、全Go test/vet/相关完整race/全标签只编译通过。来源锁更新Web2cc9269/AIba1876a，Web CI全量3609/27demo/类型/11构建已验；普通源码lint仍9998/5。
- support生命周期后续继续；不改其它三个publisher，不提供跨进程事务，不上线。见[发布合同](prometheus-publication.md)。

## 2026-11-04：告警与录制规则支持开关生命周期

- 完整未过滤采集池库存先读后发布；相关开关1生成、明确2精确退役、未知值保旧报错，关闭不读取坏规则或使用当前坏目标构造键。
- 开关进入稳定PoolID发布版本，pending/归属预检/清退/ACK恢复合同保持，跨类型互不影响。Prometheus只复用库存读取器。
- 新9主/80子、独立4个race反例、完整旧缓存及全Go test/vet/相关完整race/全标签只编译通过。来源锁固定Web9f3b00f/AIba1876a，Web全量3609/27demo/类型/11构建远端成功。
- 不将池缺失推断为关闭、不盲删未知Redis-only键，不声称真实实例已应用或上线。见[支持开关合同](support-lifecycle.md)。

## 2026-11-04：采集池支持状态输入校验

- 创建省略/0继续默认禁用2，更新必须明确两项1/2；JSON绑定及服务入口拒绝非法值，拒绝路径不触DAO或缓存。
- 新5主/64子及model/scrape service完整race/vet通过，普通CRUD、删除、publisher和schema不变；创建默认回调是无I/O MySQL DryRun，不冒称真实数据库写入。
- 已有未知值不猜测迁移，发布器原子归属写保护另阶段整合；没有上线。见[开关输入合同](support-lifecycle.md)。

## 2026-11-04：生成配置的原子归属更新与退役

- 四个publisher按不可变expected owner做条件SQL更新/删除，byte-exact匹配Name/IP，批次冲突回滚，0行仅在严格幂等条件下成功；普通CRUD不变。
- 条件归属闸门先于Redis清理，AM旧主行软退役，Webhook owner-only持久意图补重建/ABA/更名/删除或索引故障恢复，保护foreign/custom替代行。
- 原真实HTTP+SQLite两反例转绿，独立恢复对照通过；全Go test/vet、相关六包完整race、全标签只编译及隔离真实MySQL changed/found两模式通过，临时测试资源已清理。
- Web来源锁41095cd/AIba1876a，普通源码lint9895/5仍未清零。creation键位、池删除和跨进程保证未混入；未上线。见[归属写合同](generated-config-ownership.md)。

## 2026-11-04：删除采集池的定义引用与父锁

- 最终DAO事务锁父池并检查全部未删除job/alert/record定义（含禁用），三类创建/迁移同事务验证存活父池；record省略关联锁内重检，保凭据和普通更新合同。
- 新7主/27子、独立4个race场景、隔离真实MySQL6个锁先后场景、全Go test/vet/相关race/全标签只编译通过，临时测试资源清理。
- Web锁15d1572/AIba1876a，Web全量3609/类型/11构建通过；D2生成配置退役明确未完成，不把D1当全部删除闭环。见[删除合同](scrape-pool-deletion.md)。

## 2026-11-04：系统关键指标失败与健康判断

- CPU/内存/磁盘完整校验后一起提交，关键错误/取消不缓存半份数据，缺used不补零，底层输出不回显，public错误保留取消身份。
- v2缓存拒绝旧不可靠数据及未来/过期/无效快照；未实现平台明确报错，转换层缺失指标显示未知，有效阈值不变。
- 独立原5反例、System两包race/vet、本机Mac真实采集及冻结后全Go/相关五包race/标签只编译通过；非关键探针未重写，不承诺Windows能力。见[指标合同](system-metrics-health.md)。未上线。

## 2026-11-04：删除池的同步清退与持久恢复

- 真实tombstone授权主配置/告警/录制三类同步退役，部分失败明确pending且同ID可重试，周期重建可恢复；正常库存缺失不猜删除，正常生成暂停不遗失删除意图。
- pending后SQL前可靠登记已证明stale目标，补正常关闭/清空/迁移及deleted路径的无索引DB行重建恢复，不改CAS核心或AM。
- 累计83叶、独立真实DB/HTTP恢复、150次HTTP场景race、全Go test/vet/七包race/全标签只编译通过。未知键、custom/foreign、用户定义保持，不声称真实目标应用或上线。见[删除合同](scrape-pool-deletion.md)。

## 2026-11-04：严格CPU/内存探针格式与缓存版本

- Linux计数完整解析/guest不双算/无效区间拒绝，Darwin完整CPU百分比和实际页面大小vm_stat严格校验，防解析失败被当作合法测量。
- 原3反例、独立数值边界、真实Darwin smoke、全Go test/vet/七包race/标签只编译通过；缓存v3不复用旧解析样本，目标两包再次race/vet通过。
- 官方来源与本机格式依据见[指标合同](system-metrics-health.md)，无新依赖/schema、无生产或上线。

## 2026-11-04：AM无索引主配置清退

- type2全元数据严格分页发现canonical DB-only旧实例，与成员合并退休目标；sameACK不能遗漏晚出现孤儿，pending后先登记再CAS/Redis清理，失败可重建恢复。
- current及混合custom/foreign写前保护、type5 intent保持。新27叶、原HTTP反例、独立实际DAO/HTTP race、全Go test/vet/相关完整race/标签只编译通过。
- Web锁efd0e70/AIba1876a；本批前端186/type/编译proof通过。AM删除/空目标/schema不扩，未上线。见[AM发布合同](alertmanager-publication.md)。

## 2026-11-04：AM池删除与发送组父锁

- 删除最终事务锁AM父池并检查全部未删发送组；新建/迁移同事务父存活guard，省略关联锁内重检，三用户关联失败回滚，保已有disabled组保护。
- 新14叶、原交错反例/独立SQLite race/真实MySQL四锁场景、全Go test/vet/相关race/标签只编译通过；临时测试资源清理，无演示或线上改动。
- AM-D2生成配置退役仍待交付，不把引用保护当全部闭环。见[AM删除合同](alertmanager-pool-deletion.md)，未上线。

## 2026-11-04：AM删除配置清退与恢复任务隔离

- 真实AM tombstone授权type2/5同步退役和幂等重试，失败明确pending，周期/重建不靠一次goroutine。保owned CAS/主成员及Webhook意图，当前/foreign/custom/同号Scrape隔离。
- 四类退休在独立父ctx批次完整等待后执行原normal组，错误汇总，不因某类或正常生成失败取消其它退休；正常enable语义不变。
- 新54叶、独立实际DB/HTTP与ABA对照、150次HTTP race、全Go test/vet/四包race/标签只编译通过；Web23f18d8/AIba1876a锁定，前端3619/类型/11构建/边界过，源lint9682/5仍未全清。见[AM-D2合同](alertmanager-pool-deletion.md)，未上线。

## 2026-11-04：AM sameACK缓存自愈

- desired主配置与启用Webhook真实Redis内容比较后才允许快路径；丢失/漂移重发，GET错误/取消在pending前拒绝，body/secret不记录日志。
- 新27叶、原反例和独立3case race、全Go test/vet/cacheDAO race/全标签只编译通过；原删除、意图、blackhole和回退合同保持。
- Web锁75576e7/AIba1876a，工单纯回调260和映射补全270项/type通过；新组合CI另验，没有上线。见[AM发布合同](alertmanager-publication.md)。

## 2026-11-04：Record/Prom元数据完整读取责任

- 复用完整分页/身份/取消/逐页clone，保持每池fresh读取、类型策略和owner/CAS/hash/写入状态机。
- 新25叶、独立边界与原调用量probe、全Go test/vet/cacheDAO race/标签只编译通过；N²行量未减少，不夸为性能优化。
- 不扩Rule/AM、不改schema或公共接口，未上线。见[读取合同](config-metadata-snapshot.md)。

## 2026-11-04：持续push不再互相取消CI

- 后端Simple CI/三仓集成改为每push或手动run独立组，PR仍替换过时head，补齐与其余工作流一致的完整验收策略。
- actionlint及YAML结构对照通过；测试步骤、权限、触发条件、发布任务不变。三个仓库当前主分支成功/历史红与cancelled已按真实run核对，新配置远端结果另验。
- 不删除历史记录、不放宽门禁、不触Release或上线。见[CI完整性说明](ci-run-continuity.md)。

## 2026-11-04：定向元数据查询能力

- 新private DAO分页提供pool/type全部候选及精确键全Pool命名空间，保foreign/custom/Status2/duplicates，空键零查询，100键/页界限与byte比较明确。
- 新18叶、独立SQLite边界、隔离真实MySQL11 PASS节点零skip、全Go test/vet/DAO-cache race/标签只编译通过；测试资源清理。
- publisher未接线、扫描量未改善、不添加atomic create/schema/index，未上线。见[查询合同](config-metadata-snapshot.md)。

## 2026-11-04：Record/Prom发布元数据返回量减量

- 每池候选仅定位canonical旧键，desired/成员/旧键完整分块读取全namespace；保foreign/custom/inactive/duplicate冲突与两阶段owner漂移拒绝。原写入顺序、CAS和恢复锚点保持，去掉无生产调用的旧wrapper。
- 新38叶、原pager25叶和真实SQLite28叶通过；全Go test/vet、cache/DAO race及标签编译通过。5/10池样本返回量25/100降为10/20行，查询数增加为10/20次；不宣称真实SQL耗时下降。
- Web锁更新已远端验证的814214b，来源锁5测试通过；补齐CI修复后638/d478最终成功收据。新组合另验，atomic create-if-none与Rule/AM减量尚未交付，未上线。

## 2026-11-04：命名空间锁表与原子迁移确认

- 新独立v2字节锁地址，保immutable v1与历史duplicate；CurrentManifest用于现代runtime/CLI/clock，旧来源工具仍只证明v1。新锁表最小运行权限禁止删除与DDL。
- fresh/pending显式确认分流，同持锁连接在任何DDL/ledger写前重新判定；3个真实双连接预检竞态拒绝，无新v2/账本写。修复现代clock遗漏版本与旧回滚演练误用modern verify。
- 全Go test/vet/标签编译、相关race、173发布工具测试、8个MySQL子场景、原生角色权限及完整旧49表→58表→物理恢复49表12行演练通过，资源清理。数据复制2.948秒不代表整站RTO。
- Web锁更新已远端验证68bee75/AIba187；新组合CI另验。业务DAO接线与跨SQL/Redis窗口尚未交付，未上线。

## 2026-11-04：普通与生成配置的 SQL 命名空间保护

- 全部DAO写入口先固定序guard再当前读全namespace，新增/迁入占用含Status2拒绝；历史duplicate保留，按ID删除/迁出可修复。Generated保Expected CAS、更名完全Desired重试及tombstone保护，移除无caller假BatchUpsert。
- 批次一个事务，复制输入、commit成功后回填；101后段真实SQL失败零前段提交/零ID时间泄漏，1205/1213安全重试类别与commit不确定类别分开。
- 独立40叶/旧实现30实红、原publisher2红转绿、完整Go/vet/tag/cache-DAO race、MySQL双连接7场景及原CAS两row-count模式通过；原生无关键1213复现并以IP索引预过滤修复，字节精确检查保留，测试资源清理。
- Web锁更新已验证9bd9b7c，AIba187不变；SQL→Redis窗口另阶段，不宣称跨存储原子或上线。

## 2026-11-04：持久发布日志schema v3基础

- 三表分别保存池counter/last指针、历代操作与namespace事件；NULL保留未提交语义，[]表示合法空manifest；无YAML正文/软删除/自动GC。新增typed binding授权字段，SQL合法key不直接授权Redis操作。
- CurrentManifest追加独立自绑定v3，v1/v2字节不变，统一当前额外表目录与最小S/I/U权限，legacy58不改。
- 357叶race、全Go/vet/tagcompile、MySQL4子例及完整dbschema/CLI Docker原生测试通过，资源清理；Web锁已验证19ac271，AIba187。
- 当前仅存储基础，不假称DAO journal/Redis fencing或两个窗口闭环；preview仍0618/schema2，未上线。

## 2026-11-04：发布日志SQL业务接口

- 独立ProjectionJournalDAO reserve/typed business plan/nonce恢复/last-good/条件mark，metadata+events+manifest+lastCommitted单事务；ordinary写同tx记录lineage/N/A，不依赖Redis。保批次rollback/input回填与owner CAS，禁止自由Redis key投影权。
- 独立12叶及SourceID/状态红转绿、真实commit丢ACK、MySQL双session6子、旧CAS两模式/namespace7子与全Go/vet/tag/cache-DAO race通过，资源清理。
- Web锁已验证dd558ce/AIba187，新CI另验；发布器/Lua/recovery worker及两窗口闭环仍待接线，未上线。

## 2026-11-04：IPv6 webhook与长生成名称容量修复

- 独立v4三个ALTER扩宽Name255/key128字符、guard/event512字节；共享Go/DTO限制同步，原数据/索引/字符属性与V1–V3校验和不变。
- 先验证table default及四列源属性，继承已证相同collation；原模型结构硬比对不放宽，dirty/unsupported source拒绝零ALTER但不谎称零history。
- 47/65合法IPv6 key/106Name原红转绿、五包race/vet/tag、全Go/MySQL六子/完整dbschema-CLI native过，私有资源清理；Web锁已验证192178f，前端表单/权限另阶段。未上线，preview仍0618/schema2。


## 2026-11-04：完整投影计划与 SQL 租约

- 新 v2 manifest 持久完整成员、AM退役 intent、artifact/control ACK向量；旧 v1 保持历史语义，旧artifact-only pool marker拒绝v2，损坏SourceID映射不再获得完整租约。
- 第二事务按control→有序namespace取得sealed callback租约，普通事件仅可invalidate/retire；同连接重验与深拷贝，退出/取消/已知错误封闭，禁止丢连接后另借连接续租。
- MySQL8.4.11/RR五子实际通过：互斥普通writer及新reserve、空发布与重复ACK保原时间、ordinary最小权限、取消、真实KILL自有会话后fail-closed。业务commit丢ACK按nonce恢复完整控制计划。临时资源已清。
- 来源锁更新到独立远端验证通过的Web7e3daa5/AIba187；本机预览无损升级clean5feb/schema4，业务数据保留且Cron/外部执行关闭。新组合CI另验，Redis store/发布器/worker仍未接线，未上线。


## 2026-11-04：typed Redis 投影存储与 fencing

- 私有handle从真实SQL lease取得typed artifacts/controls，所有key按逻辑角色导出；Prepare只新fence，所有参数/TYPE/fence先验后写，uint64以20位字符串精确比较。
- 同版完整behavior frame固定SHA、key/role/绑定、artifact/control/ACK，修复原生同版改legacy SHA可写入的反例；专用无重试client保留真实传输不确定状态，SQL重建可恢复。
- 真实Redis9子和独立3项通过：迟到render/retire与四种同池命令、大于2^53事件/池版本、空发布、TYPE、完整AM/subset/ordinary、真实已执行丢回复恢复；不冒称跨存储原子或任意failover安全。
- 来源锁更新已远端Verify通过Webe124f08/AIba187；发布器/worker与opt-in native CI供给另阶段，演示继续clean5feb/schema4、无外部执行，未上线。


## 2026-11-04：Redis fencing 自动隔离 CI

- 三仓integration新增owned原生入口：全新Redis/private ACL/owner marker/动态loopback，自供给12项真实协议测试并精确清理，不依赖本机连接文件或业务Redis。
- daemon操作前拒DOCKER_HOST/DOCKER_CONTEXT并验证本机Unix socket，所有命令显式固定endpoint；新增2项调用CLI前拒绝回归。
- 干净62e4c107隔离候选本机14叶race零skip/fail、tagged vet/actionlint/来源锁通过，实例清理验证为零；Linux新CI另验。Record并行WIP不在此提交，未上线。


## 2026-11-04：Record 持久发布与故障恢复

- Record正常生成、off/empty/deleted共用journal→sealed lease→Store，移除旧unsafe写序列和隐式fallback；源读前版本/原control CAS拒迟到旧快照，源Hash与DB metadata同时参与完整计划判断。
- 新narrow历史定位覆盖无member/manifest墓碑；同tx全历史guard防create/delete ABA，live owner可观察已deleted foreign/custom而退自己，未授权旧墓碑仍严格消歧。
- accepted render/retire与显式legacy provenance持久记录替代完成事实，A只回收自身empty bookkeeping不碰B；B后更新/退役及多次重建不忘解决结果。已有permit可在原sealed lease恢复，但新增stale必须新guarded plan。
- 全Go/vet/tag与cache/DAO/DI race通过；owned原生20叶0skip/fail含真实TCP/SQL/Redis，实例清理0。Wire实际生成并接pool cleanup。Web锁cd4f757已独立Verify成功，AIba187；新CI另验，未上线。
