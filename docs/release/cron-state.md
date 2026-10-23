# 定时任务启用与执行状态（历史阶段）

> 本文记录早期条件更新实现，已由 [执行租约与崩溃恢复](cron-lease.md) 替代；当前版本新增租约字段并支持过期执行权恢复。以下无迁移及硬退出不恢复的描述仅适用于早期实现

本模块沿用 `cl_cron_jobs.status` 的既有枚举与数据库结构，无新增列，无需数据库迁移

- `Enabled` 与 `Running` 都属于需要保留调度条目的状态，执行中重载不重置 `@every` 周期
- 执行器通过数据库条件更新 `Enabled → Running` 获取执行权，避免读状态后被停用或被其他 worker 抢占时仍执行
- 完成时只允许 `Running → Enabled`，人工改为 `Disabled` 后不会被完成处理重新启用
- 请求取消后，结果写入与状态释放共用独立的 5 秒清理期限

已执行验证：

```sh
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go vet ./internal/cron/...
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/cron/handler ./internal/cron/scheduler
```

纯内存执行器替身验证正常完成、运行中停用、读取后停用、其他 worker 竞争、领取失败与任务取消。SQLite 内存数据库与实际 DAO、Asynq 调度器验证运行中条目保持、重复领取失败、人工停用后恢复失败及共享调度器外部条目不受影响

已知边界：进程硬退出后的持久 `Running` 不会自动复位。没有执行租约时，自动复位可能与其他副本仍在执行的任务冲突；应先核实执行结果，再由管理员恢复启用。已有运行任务在人工停用后继续完成，本模块阻止未来调度，不将“禁用”解释为立即取消进程
