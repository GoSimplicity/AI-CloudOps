# 配置依赖边界

业务模块通过构造器接收配置，不直接导入 Viper。加载、默认值和环境覆盖保留在 `internal/config`；`TestProductionViperImportsStayInsideConfigBoundary` 用 Go AST 检查 `cmd`、`internal`、`pkg` 的生产源码导入，包含构建标签文件，避免仅靠搜索漏掉别名导入。测试文件可以用隔离配置替身。

## Webhook：构造时复制配置

consumer、公共映射缓存、消息内容和机器人分别接收 `*config.WebhookConfig`，复制各自需要的标量。构造后修改原指针或全局 Viper 不会改变该实例的地址、凭据、超时、worker 数及刷新间隔。配置改动需要重建实例；机器人令牌和公共映射的运行期刷新仍按原合同执行，不把刷新状态当作静态配置。

空或不完整的机器人凭据仍禁止外呼。实际请求地址、超时、失败确认及错误保密的既有规则继续生效，测试用本地替身，不发送真实消息。

## Cron：显式动态门禁

本地命令/脚本许可与普通配置快照不同。三层构造器显式接收同一个动态 policy：服务保存/启用/手动触发、调度重载、执行前检查均读取当前开关。配置边界创建 policy 时绑定当前加载的配置实例，每次检查仍读取该实例的当前配置和环境覆盖。

未注入 policy 时拒绝本地执行，默认开关仍为 false。HTTP/SSH 不因本地命令开关被误拦。此变更没有新增运行中并发写 Viper map 的保证，也没有新增管理接口、文件监听或远程热更新能力；它只保留现有动态读取合同并使依赖可见。

## 验证记录

2026-10-31 已执行 Go 1.26.10：

| 检查 | 结果与范围 |
| --- | --- |
| `go test ./...`、`go vet ./...` | 全仓通过 |
| Cron/config/main DI 相关10包 race | 通过；动态 env、同实例更新、实例隔离、nil 拒绝，以及保存/启用/触发和排队后撤销 |
| Webhook 全包、Prometheus utils、config race | 通过；配置与 global 冲突、构造后原配置修改、空凭据禁外呼、现有消息协议/隐私/部分失败 |
| `go test -tags=integration,smoke,schema_smoke -run '^$' ./...` | 全仓编译通过；不等于实际执行集成测试 |
| 发布工具 | 强制 Compose 的完整脚本158项通过、零跳过；示例地址合同支持 typed 字段且拒绝错误字段/receiver |
| Wire | main 和 Webhook 两处均实际重新生成 |

本轮测试使用仓库既有替身/SQLite 等隔离设施，未调用真实通知、云或生产任务。复验使用既有 Go 模块与工具链，无新增依赖。完整回归源码位于 `internal/config/cron_config_test.go`、Cron 三包既有 local_exec 测试及 Webhook 各包 configuration 测试。源码候选 `eec9582a` 的历史镜像不包含本阶段配置重构，不能拿历史候选替代新提交验收。

```sh
go test ./...
go vet ./...
go test -race ./internal/config ./internal/cron/... ./pkg/di ./internal/prometheus/webhook/... ./internal/prometheus/utils
go test -tags=integration,smoke,schema_smoke -run '^$' ./...
```

## HTTP/WebSocket 来源配置（2026-11-03）

主程序的HTTP CORS和注入的WebSocket升级器现在都接收`*config.Config`，使用`Server.AllowedOrigins`的构造时快照。`SERVER_ALLOWED_ORIGINS`仍通过配置加载边界按环境变量>YAML生效；空环境值沿用既有Viper回退YAML合同。构造后修改原配置或进程环境不会更改这些实例。YAML示例新增空的`server.allowed_origins`，空或nil配置拒绝跨域；WebSocket同源及无Origin客户端的既有合同保留。

通用`pkg/websocket.CheckOrigin`仍作为既有独立调用方的动态环境适配器，不冒称整个代码库已消除所有环境读取。主应用DI改用显式`NewOriginChecker`，没有调用这一适配器；通用库旧API未破坏。配置修改需要重建主应用实例。

| 场景 | 检查 | 结果 |
| --- | --- | --- |
| YAML来源/环境覆盖/空环境回退 | 真实Load路径配置测试 | 通过 |
| 来源精确匹配、端口/子域混淆、nil/空配置 | 真实Gin CORS及WS checker | 通过 |
| 构造后环境/配置改变 | 已构造实例保持快照；legacy adapter仍读取当前env | 通过 |
| 单元与并发 | `go test ./...`、`go vet ./...`；最终配置/DI/WS三包`-race` | 通过；真实加载新增用例首次因夹具缺必需密钥/issuer失败，补足随机测试配置后复验通过 |
| 真实本地WebSocket握手 | `go test -tags=smoke ./pkg/di -run '^TestWebSocketNegotiatesPublicProtocolOnly$'` | 通过，固定公开子协议且不回传凭据 |
| 构建标签与工具 | integration/smoke/schema_smoke全仓只编译；强制Compose脚本158项 | 通过，脚本零跳过 |

使用既有Go 1.26.10、Gin/cors/Gorilla与仓库测试设施，无新依赖；main Wire实际重新生成。未运行生产环境、云请求或上线。完整测试源码见`pkg/di/origin_config_test.go`、`internal/config/load_test.go`及`pkg/websocket/origin_test.go`。
