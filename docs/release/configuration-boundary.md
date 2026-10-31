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
