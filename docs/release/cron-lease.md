# Cron 执行租约与崩溃恢复

当前启动已改为只读schema/clock验证。下文GORM注册与自动迁移描述属于历史实现或模型合同测试；新库由不可变基线显式初始化，旧库须单独预检/认领，运行账号不应授予DDL。见 [schema-cli.md](schema-cli.md) 与 [fresh-schema-start.md](fresh-schema-start.md)。

新增 `cl_cron_jobs.lease_owner VARCHAR(64) NOT NULL DEFAULT ''` 与 `lease_expires_at BIGINT NOT NULL DEFAULT 0` 两列，分别保存执行尝试的随机归属 token 和租约到期 Unix 秒，均不通过任务 JSON 返回。执行器已接入原子领取、续租与归属完成，旧的无归属结果写入入口已移除

租约 30 秒，每 10 秒续租。期限判断与更新在数据库 UPDATE 内使用数据库时钟执行：MySQL `UNIX_TIMESTAMP()`，SQLite `strftime('%s','now')`，不依赖各 worker 的应用时钟，也不在读取时钟后等待行锁再写入旧期限

领取只允许配置启用/运行中的任务，且无主的新任务或已有归属的过期租约。续租与完成同时检查 token 和未过期条件；完成原子写入统计/结果、释放租约，并仅将 `Running` 恢复为 `Enabled`。人工 `Disabled`/`Enabled` 不被完成覆盖，重新启用也不能绕过活跃租约。旧 owner 的结果不得覆盖新 owner

升级与回滚：

- 原有 GORM `InitTables`/`AutoMigrate` 会为现有表增加这两列，默认值不会重置任务状态
- 升级前备份数据库，停止并排空全部旧版 worker，先由单个新实例完成 GORM 增列，再扩容新 worker，避免多个实例同时变更同一旧表，不能混跑不认识租约的旧 worker
- 旧版遗留 `Running` 且无 token 的记录没有可靠归属，不在启动时批量清空或直接抢占。停止旧 worker 后逐项核对远端执行结果，再由管理员重新启用这些任务，启用后下次领取建立新租约
- 已采用租约的新版本 worker 硬崩溃，其他 worker 可在租约过期后原子领取；不需要人工清空 `Running`
- 回滚前必须停止并排空新 worker，确认没有仍在执行的命令并处理 `Running`，再启动旧版本；不支持旧 worker 与租约 worker 并存。新增列可以保留，删除前需单独确认备份和变更窗口

租约提供有界归属与恢复，不能保证外部命令 exactly-once。进程崩溃、网络分区或取消无法撤销已提交到外部系统的副作用；任务必须使用幂等业务键、去重或可重入步骤，并核对不确定的远端结果

第一阶段验证使用真实 SQLite、实际 GORM 增列与 DAO，并通过受控 SQL 将数据库租约回拨到过去，未使用生产数据库：多副本并发领取只有一个成功、活跃租约不抢占、过期可恢复、旧归属续租/完成失败、统计与结果不被旧 owner 覆盖、人工禁用/启用保留、无归属旧 `Running` 不被盲目接管

```sh
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/cron/dao
```


执行器在取得租约后启动心跳，续租查询受 3 秒上下文期限限制。续租失败或失去 owner 即取消当前命令/HTTP/SSH 的执行上下文，不再提交该尝试的结果；让租约自然过期，随后按 Asynq 重试恢复。正常完成停止心跳，并以独立 5 秒清理上下文完成自己仍有效的租约。结果持久化失败或归属失效返回可重试错误，不向队列确认成功

活跃租约上的重投也返回可重试错误，避免原来 `status != Enabled` 直接返回 nil 导致硬崩溃后的任务被永久吞掉。手工禁用或删除的任务依旧按配置跳过，不被恢复流程重新启用

30 秒是数据库执行租约，并非端到端恢复时限。当前 Asynq v0.22.0 的 Redis 任务租约也有独立恢复流程：失联租约过期后额外容忍 30 秒时钟偏差，recoverer 每分钟轮询，再按重试延迟入队；Redis 不可用、worker 容量不足或重试耗尽会进一步影响恢复。应监控重试/归档任务及长期 Running，而不能把30秒解释为用户请求恢复SLA

已执行集成验证使用实际 SQLite、GORM DAO 和 Asynq Task 对象调用生产 `ProcessTask`，通过可控心跳事件和数据库期限回拨模拟故障，无固定 sleep 猜测执行阶段：

- 先复现旧实现对硬崩溃遗留 Running 返回 nil 的失败回归，再验证该记录不会被静默确认成功
- 过期租约的真实 Task 重投执行 HTTP 替身并原子保存结果、释放归属
- 活跃租约重投返回可重试错误，未误执行或误确认
- 心跳实际续租后竞争者不能抢占；模拟数据库过期后新 owner 领取，旧心跳发现失权并取消已启动请求，旧结果和统计不覆盖新 owner
- 续租数据库失败会取消当前执行并保留不确定租约待过期，不提交未确认归属的结果；既有人工启停与请求取消回归继续通过

这些测试未启动或硬杀真实 Redis/Asynq worker 进程，未执行生产外部命令；Redis 队列进程级崩溃恢复和实际远端副作用仍属于部署故障演练范围

```sh
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/cron/dao ./internal/cron/handler
GOTOOLCHAIN=auto go vet ./internal/cron/dao ./internal/cron/handler ./internal/model
GOTOOLCHAIN=auto go test ./...
```
