# cl_cron_jobs.job_type 去掉列默认值（升级说明）

`internal/model/cron_job.go` 中 `CronJob.JobType` 的 gorm tag 已去掉 `default:2`。原默认值 2 是"命令行任务"，在 `cron.allow_local_exec` 默认关闭（SEC-03A）之后，任何漏传 `job_type` 的写入都会静默落成一个不可执行、也不能编辑的本地任务；去掉默认值后，漏传会由 `NOT NULL` 直接报错。应用层所有写入路径（创建/更新接口、内置任务）都会显式写入 `job_type`，接口层 `job_type` 为必填。

## 重要：启动时的 AutoMigrate 会自动改列

本仓库**没有**独立的迁移体系，后端启动时无条件执行 `pkg/di/gorm.go:74` → `InitTables`（`pkg/di/init.go:41` 的 `db.AutoMigrate(...)`，其中 `pkg/di/init.go:89` 注册了 `&model.CronJob{}`）。

依据 go.mod 中的 `gorm.io/gorm v1.25.11` 与 `gorm.io/driver/mysql v1.5.7` 源码：

- `gorm@v1.25.11/migrator/migrator.go:161`：表已存在时，对每个已存在的列调用 `MigrateColumn`
- `gorm@v1.25.11/migrator/migrator.go:533-543`：`currentDefaultNotNull` 由模型 tag 是否声明 default 决定，`dv, dvNotNull` 来自数据库；数据库有默认值而模型没有（`dvNotNull && !currentDefaultNotNull`）即置 `alterColumn = true`；反之亦然
- `gorm@v1.25.11/migrator/migrator.go:567-569`：`alterColumn` 为真时调用 `AlterColumn`
- `driver/mysql@v1.5.7/migrator.go:313-381`：`ColumnTypes` 从 `information_schema.columns.column_default` 读取默认值（`'2'` → `Valid=true`）
- `driver/mysql@v1.5.7/migrator.go:147-160`：`AlterColumn` 执行 `ALTER TABLE ? MODIFY COLUMN ? <FullDataTypeOf>`

使用 DryRun 调用 MySQL 方言的 `MigrateColumn` 实测（不连接数据库）：

| 模型 tag | 库中列默认值 | AutoMigrate 产生的 SQL |
|---|---|---|
| 含 `default:2`（旧） | 有 DEFAULT 2 | 无 |
| 含 `default:2`（旧） | 无默认值 | ``ALTER TABLE `cl_cron_jobs` MODIFY COLUMN `job_type` tinyint(1) NOT NULL DEFAULT 2 COMMENT '…'`` |
| 不含 default（新） | 有 DEFAULT 2 | ``ALTER TABLE `cl_cron_jobs` MODIFY COLUMN `job_type` tinyint(1) NOT NULL COMMENT '…'`` |
| 不含 default（新） | 无默认值 | 无 |

结论：

1. **只要部署了本改动的后端并启动，AutoMigrate 就会自动对 `cl_cron_jobs.job_type` 执行一次 `ALTER TABLE … MODIFY COLUMN`，去掉 DEFAULT 2。** 这等同于未经审批执行了下文的 DDL。在 REL-01 迁移体系上线、且生产执行获批之前，不得把本改动部署到连接生产库（或任何共享库）的实例。
2. 反过来，如果手工执行了下文 DDL，却仍运行带 `default:2` 的旧二进制，旧二进制启动时会把 DEFAULT 2 自动加回。回滚二进制时须同步执行回滚语句，或接受默认值被加回。
3. MySQL 的 `MODIFY` 会重新定义整列，不写 `COMMENT` 子句会清空列注释，而新版本启动时 AutoMigrate 发现注释与模型不一致会再 `MODIFY COLUMN` 一次。因此下文的升级与回滚语句都带上与模型一致的 `COMMENT`，文字逐字取自 `internal/model/cron_job.go` 中 `JobType` 的 gorm tag（`comment:任务类型 1系统任务 2命令行 3HTTP 4脚本 5SSH`）；执行后列定义与对应版本的模型一致，AutoMigrate 不会再改列（见下文"DryRun 核对"）。

## 待执行的 DDL（REL-01 迁移体系上线后再执行）

执行前先确认当前列定义，确认 `job_type` 为 `tinyint(1) NOT NULL DEFAULT '2'`：

```sql
SHOW CREATE TABLE cl_cron_jobs;
```

升级：

```sql
ALTER TABLE cl_cron_jobs MODIFY job_type tinyint(1) NOT NULL COMMENT '任务类型 1系统任务 2命令行 3HTTP 4脚本 5SSH';
```

回滚：

```sql
ALTER TABLE cl_cron_jobs MODIFY job_type tinyint(1) NOT NULL DEFAULT 2 COMMENT '任务类型 1系统任务 2命令行 3HTTP 4脚本 5SSH';
```

执行要求：

- 等 REL-01 的迁移体系上线后，由迁移体系执行，不手工在启动流程外散落执行
- 生产库执行必须经用户批准；执行前备份，先在库副本上演练
- 执行后再次 `SHOW CREATE TABLE cl_cron_jobs` 核对：`job_type` 无 DEFAULT、仍为 NOT NULL、`COMMENT` 仍为 `'任务类型 1系统任务 2命令行 3HTTP 4脚本 5SSH'`，`idx_job_type` 索引仍在
- `COMMENT` 文字必须与 `internal/model/cron_job.go` 中 `JobType` 的 gorm tag 逐字一致；若模型注释改动，本文语句须同步修改，否则 AutoMigrate 会因注释不一致再次 `MODIFY COLUMN`
- 回滚应用二进制到含 `default:2` 的旧版本时，同步执行回滚语句

## DryRun 核对（带 COMMENT 的语句）

用 DryRun 调用 MySQL 方言的 `MigrateColumn`（不连接数据库，按 `information_schema.columns` 在执行语句后的取值构造列信息：`data_type=tinyint`、`column_type=tinyint(1)`、`is_nullable=NO`、`column_comment` 为上文注释）：

| 模型 | 库中列（执行语句后） | AutoMigrate 产生的 SQL |
|---|---|---|
| 不含 default（新） | 执行升级语句：无默认值、带 COMMENT | 无 |
| 不含 default（新） | 升级语句不带 COMMENT（注释为空） | ``ALTER TABLE `cl_cron_jobs` MODIFY COLUMN `job_type` tinyint(1) NOT NULL COMMENT '…'`` |
| 含 `default:2`（旧） | 执行回滚语句：DEFAULT 2、带 COMMENT | 无 |
| 含 `default:2`（旧） | 回滚语句不带 COMMENT（注释为空） | ``ALTER TABLE `cl_cron_jobs` MODIFY COLUMN `job_type` tinyint(1) NOT NULL DEFAULT 2 COMMENT '…'`` |

结论：按上文带 COMMENT 的语句执行后，对应版本的二进制启动时 AutoMigrate 不再对 `job_type` 生成 ALTER。该核对未在真实 MySQL 上执行，执行后仍须按上文 `SHOW CREATE TABLE` 核对。

## 待决事项

- 在 REL-01 上线前如何阻止 AutoMigrate 自动改列（暂缓合并/部署本改动，或在迁移体系中接管 `cl_cron_jobs`），需由研发指挥官决定。

本改动只修改了代码，没有对任何数据库执行 ALTER。
