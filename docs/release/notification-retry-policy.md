# 通知配置重试执行合同

日期：2026-10-23

## 测试用例矩阵

| 范围 | 场景 | 断言 |
| --- | --- | --- |
| 同步 Manager | 省略、显式 0、显式 N、提前成功 | 省略继承通道默认；0 仅首次发送；N 为额外重试；成功后停止 |
| 参数与取消 | 负值、超过 100、分钟转 duration 溢出、预取消、等待中取消 | 非法值零发送；取消及时返回且不再发送；错误与请求 MessageID 保留 |
| 异步入队 | 0/N/默认、地址标准化、上下文、序列化和队列失败 | MaxRetry、TaskID 和 ProcessIn 正确；队列错误传播；不在入队时发送 |
| Worker | 既有 raw 请求、新 request envelope、send/scheduled/retry | 两种已发布格式可解析；每次 worker 执行仅一次渠道调用，交由 Asynq 重试 |
| 批量 Worker | 部分失败、其余成功 | 尽力处理各项，返回失败与 SkipRetry，避免整批重放重复投递成功项 |
| 服务层 | 实际自动通知、显式测试通知、手动直接发送 | 自动/测试携带配置 0/2/100 和 7 分钟；无配置直接发送不虚构覆盖值 |
| 日志 | 正文、元数据、地址、供应商错误的 canary | Manager/Processor 日志不含敏感值 |

## 前置依赖与环境

复用仓库 Go 1.26.10、Asynq v0.22.0 和 zap，使用原生 testing 与本地接口替身，无新增依赖。新增专项不使用真实 SMTP、飞书、Webhook、Redis、数据库或外部网络。通知包及服务包的原有本地回归仍一起执行

模型与现有 UI 均将 RetryInterval 定义为分钟，SendRequest 使用可选 MaxRetries 与 RetryIntervalMinutes；nil 使用通道原默认，显式 0 不丢失。次数上限与已有请求 DTO 一致为 100，分钟乘 duration 前校验溢出；额外重试次数不包含首次发送

## 完整测试代码

- internal/workorder/notification/retry_policy_test.go：假渠道计数、取消、假队列选项、协议兼容、worker 单层重试、批量失败与日志边界
- internal/workorder/service/notification_retry_test.go：真实服务方法到发送接口的配置透传；公开构造函数仍接受 *notification.Manager，内部最小接口供无 IO 验证

七处分支先在基线实证失败，再实现修复。实际业务链仍为 processNotification → sendChannelNotification → Manager.SendNotification；显式测试同样同步发送。手动直接发送没有配置，保持通道默认

异步 Manager 现在统一生成已存在的 request envelope，并兼容早期直接序列化 SendRequest 的持久化格式。worker 不执行内部重试，Asynq MaxRetry 负责额外执行次数；Manager.RetryDelay 可接到 asynq.Config.RetryDelayFunc，为通知任务应用分钟间隔，其余任务保留 Asynq 默认退避。后续最小接线已将 Processor.RegisterTasks 接到 App.Run 的唯一实际 worker mux，与 Cron 共用原 default 队列；worker config 接入 Manager.RetryDelay，非通知任务保持 Asynq 原生退避，Cron 的 IsFailure、并发 10 与队列权重 1 均保留。工单同步业务入口不改为异步

## 执行与验证命令

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race ./internal/workorder/notification ./internal/workorder/service
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./internal/workorder/notification ./internal/workorder/service
```

此批未使用外部通知，也未更改全局通道 BaseChannelConfig 的旧默认语义。来源 scope 与历史高级触发 helper 由独立批次提供，在 notification_service.go 串行接入；不匹配来源在收件人读取前直接跳过，匹配后再完整校验自动触发参数

## 实际异步接线验收

internal/app/asynq_mux.go 组合原 Cron 与通知四类任务；App.NewApp 注入 Processor 并拒绝缺失依赖。pkg/di/asynq.go 组合现有队列与分类重试，pkg/di/wire.go 注册 provider，wire_gen.go 通过仓库既有 go generate 命令及锁定 Wire v0.6.0 生成，未手改

离线 mux/queues/delay 回归覆盖 Cron 路由和四种通知路由、未知任务、default:1、Cron 原失败分类器、通知 0/1 分钟/省略值，以及非通知原生退避。四个完整包 app/di/notification/service 的 race 通过

真实 Asynq integration（5.84 秒）由测试创建全新 redis:7.4.11-alpine Docker 容器，唯一 cloudops.test.run 标签、随机 loopback 端口、无持久化卷；不接收业务 Redis 地址。localhost httptest 假飞书收件箱一直返回已知失败：配置 override 0 → 实际收件箱 1 次且 archived/retried0；override 2 → 实际 3 次且 archived/retried2。均使用真实生产 RetryDelayFunc 和合法 0 分钟，只缩短测试 worker 的扫描周期，不替换重试间隔函数；另在同一真实生产 hook 上确认 1 分钟返回 time.Minute。TaskID 与请求 MessageID 一致

worker 优雅退出，核对容器 run 标签后精确删除 Redis，复查该测试标签无残留容器。没有访问主业务 Redis，也没有真正外部邮件、飞书、Webhook 或云调用。默认通知通道启用配置不变

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go generate ./pkg/di
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race ./internal/app ./pkg/di ./internal/workorder/notification ./internal/workorder/service
CLOUDOPS_TEST_NOTIFICATION_ASYNC=true DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race -tags=integration ./pkg/di -run '^TestNotificationAsyncIsolatedRedis$' -count=1 -v
```

显式 integration 标志缺失时跳过真实容器测试。该测试使用 --pull=never，要求本机已有指定 Redis 镜像；不会隐式拉取镜像或使用其它 Redis
