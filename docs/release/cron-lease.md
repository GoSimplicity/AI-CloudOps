# Cron 执行租约与崩溃恢复

新增 `cl_cron_jobs.lease_owner VARCHAR(64) NOT NULL DEFAULT ''` 与 `lease_expires_at BIGINT NOT NULL DEFAULT 0` 两列，分别保存执行尝试的随机归属 token 和租约到期 Unix 秒，均不通过任务 JSON 返回。当前第一阶段提供原子 DAO；执行器接入与续租验证随下一阶段补齐

租约 30 秒，计划每 10 秒续租。期限判断与更新在数据库 UPDATE 内使用数据库时钟执行：MySQL `UNIX_TIMESTAMP()`，SQLite `strftime('%s','now')`，不依赖各 worker 的应用时钟，也不在读取时钟后等待行锁再写入旧期限

领取只允许配置启用/运行中的任务，且无主的新任务或已有归属的过期租约。续租与完成同时检查 token 和未过期条件；完成原子写入统计/结果、释放租约，并仅将 `Running` 恢复为 `Enabled`。人工 `Disabled`/`Enabled` 不被完成覆盖，重新启用也不能绕过活跃租约。旧 owner 的结果不得覆盖新 owner

升级与回滚：

- 原有 GORM `InitTables`/`AutoMigrate` 会为现有表增加这两列，默认值不会重置任务状态
- 升级前备份数据库，停止并排空全部旧版 worker，再切换新版本，不能混跑不认识租约的旧 worker
- 旧版遗留 `Running` 且无 token 的记录没有可靠归属，不在启动时批量清空或直接抢占。停止旧 worker 后逐项核对远端执行结果，再由管理员重新启用这些任务，启用后下次领取建立新租约
- 已采用租约的新版本 worker 硬崩溃，其他 worker 可在租约过期后原子领取；不需要人工清空 `Running`
- 回滚前必须停止并排空新 worker，确认没有仍在执行的命令并处理 `Running`，再启动旧版本；不支持旧 worker 与租约 worker 并存。新增列可以保留，删除前需单独确认备份和变更窗口

租约提供有界归属与恢复，不能保证外部命令 exactly-once。进程崩溃、网络分区或取消无法撤销已提交到外部系统的副作用；任务必须使用幂等业务键、去重或可重入步骤，并核对不确定的远端结果

第一阶段验证使用真实 SQLite、实际 GORM 增列与 DAO，并通过受控 SQL 将数据库租约回拨到过去，未使用生产数据库：多副本并发领取只有一个成功、活跃租约不抢占、过期可恢复、旧归属续租/完成失败、统计与结果不被旧 owner 覆盖、人工禁用/启用保留、无归属旧 `Running` 不被盲目接管

```sh
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/cron/dao
```
