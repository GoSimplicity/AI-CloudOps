# 当前版本旧库的只读预检

`dbschema.ValidateLegacyCurrent(ctx, conn)` 只检查未版本化数据库是否已经符合冻结的版本 1。它只发出 `SELECT` 和 `SHOW CREATE TABLE`，不会建表、修复数据、创建迁移账本或登记任何版本。

本阶段提供独立预检命令与显式审计认领命令。预检通过也不等于完成认领或允许发布。生产操作由发布负责人执行，需要单独授权。

## 使用边界

调用方必须先停止所有业务 writer，并持有 `schema.WithMigrationLock` 提供的同一主库连接。此锁只能协调遵守该协议的迁移工具，不能阻止其他应用连接写数据或执行 DDL。预检与之后的认领之间不得释放锁；否则结构和数据可能在检查后变化。

此函数只认识当前版本 1，且要求完整的 58 张业务表。以下情况均拒绝：

- 缺表、多表、视图，或已经存在 `cloudops_schema_history`。
- 列顺序、类型、长度、默认值、空值约束、注释、索引、外键或表选项发生变化。
- 数据修复的后置条件尚未满足。
- 元数据可见性不足，或任一查询失败。

已有迁移账本的库应走 `schema.Verify` 检查 applied、pending、dirty 和 checksum，不能通过忽略账本转入旧库认领。真正较旧的结构需要另行设计、验证升级步骤，不能把版本 1 的纯 CREATE 快照直接用于旧库。

未来追加版本 2 时，此函数会主动拒绝工作，直到补充经审阅的验证契约；不能继续用版本 1 预检结果登记更高版本，也不能修改已发布的 baseline 文件来适配旧库。

## 结构比对

期望结构来自冻结的 `baseline_v1.json`：58 条 CREATE、34 条独立 ADD FOREIGN KEY，以及 2 条数据修复语句。生产验证器不引用未来可能变化的 GORM model，也不执行快照中的任何 SQL。

解析器只接受已经冻结的多行 SHOW CREATE 格式，按表合并 CREATE 和后置外键声明。实际结构首先核对完整表与视图清单，再读取这些已知表的 SHOW CREATE。

比对保留列顺序和每条定义的完整内容，只进行两项规范化：

1. 对独立的索引、约束声明排序。索引内部的列顺序、前缀长度、ASC/DESC、唯一性和完整外键动作保持原样。
2. 忽略表选项中的 `AUTO_INCREMENT` 当前计数。业务行和自增序号增长不构成结构漂移；列自身的 AUTO_INCREMENT 定义仍必须一致。

引擎、字符集和 collation 必须与冻结结构一致。其他额外表选项不会被随意忽略。固定格式解析失败时保守拒绝，不猜测 SQL 的含义。

## 元数据权限

MySQL 的 information_schema 会根据权限隐藏对象。只授予若干已知表权限的账号无法证明不存在隐藏的额外表。

验证器要求 `CURRENT_USER()` 具有直接的数据库级或全局 `SELECT` / `REFERENCES` 权限，并查询 information_schema 权限元数据确认这一前提；同时必须能实际读取全部 SHOW CREATE，以及后置条件涉及的业务表。

迁移账号可使用数据库级 REFERENCES 保证完整对象可见性，再配置所需数据权限。只读的数据库级 SELECT 账号也可以执行本预检，无须 DDL 或 DML 权限。

MySQL 8.4.11 的自有 fixture 已验证：激活角色后，ENABLED_ROLES 能证明角色已启用，实际 TABLES 查询也会看到角色权限覆盖的对象；但 SCHEMA_PRIVILEGES 和 USER_PRIVILEGES 并不向该普通账号暴露角色拥有的数据库级 REFERENCES 授权。因此，“角色已启用”不能代替“角色拥有完整 schema 元数据权限”的证明。验证器仍坚持直接授权，既不接受未激活角色，也不会仅凭已激活角色名称放宽检查。

`BuildRolePlan` 的 principal 可以是实际账号：使用 plain CREATE USER 后，直接把生成的权限授给该 principal，即可满足本检查。若选择通过 MySQL role 继承迁移和数据权限，则必须额外直接向实际迁移账号授予目标数据库的 REFERENCES；角色仍须处于激活状态，以提供需要的数据读取权限。这个补充不要求额外授予账号 DDL、DML 或读取 mysql 系统表的权限。不会为了读取角色授权而扩大账号的系统元数据访问范围。

不满足这一直接元数据授权前提时保守拒绝，不把“不看得到对象”解释为“不存在对象”。当前不解析复杂的 SHOW GRANTS ... USING 语法。

## 数据后置条件

预检使用与冻结修复语句完全相同的 WHERE 语义，要求以下两个结果均为 0：

```sql
SELECT COUNT(*)
FROM cl_workorder_notification
WHERE is_default = 0;

SELECT COUNT(*)
FROM cl_tree_cloud_resource_change_log
WHERE LOWER(TRIM(field_name)) IN ('key', 'password')
  AND (old_value <> '[REDACTED]' OR new_value <> '[REDACTED]');
```

查询没有软删除过滤，已软删除的历史记录也参与检查。不会读取或输出敏感审计字段的内容，只读取计数。

SQL NULL 遵循 MySQL 三值逻辑：`(NULL, NULL)`、`(NULL, '[REDACTED]')` 和 `('[REDACTED]', NULL)` 不命中原修复条件，允许保留；任一非 NULL 字段仍包含未脱敏值时会拒绝。此预检不声称历史 NULL 已被改写。若要强化为两个字段都必须非 NULL，应新增数据迁移版本，不能静默改变冻结版本 1 的语义。

原业务行、历史时间和自增序号均可以存在并增长。此检查不证明历史 DATETIME 使用了哪个时区；有历史通知时间的旧库仍需要单独核实其解释方式。`InitializeFreshClock` 不能用于替旧库猜测或转换时区。

## 错误与后续认领

错误仅返回静态类别，不包含驱动原文、SQL、业务值或连接配置：

| 错误 | 含义 |
| --- | --- |
| `ErrLegacyStructure` | 表清单或完整结构不一致，也包括已有账本 |
| `ErrLegacyData` | 冻结修复的后置条件尚未满足 |
| `ErrLegacyVisibility` | 无法确认完整元数据可见性 |
| `ErrLegacyRead` | 无法读取完整元数据或必要数据 |
| `ErrLegacySnapshot` | 冻结格式或当前版本不在验证器支持范围 |

认领必须在停写、备份和核对历史时区后，持同一迁移锁执行预检；只有全部通过才能创建账本并按固定版本、固定 checksum 登记。MySQL 的账本 DDL 可能隐式提交，不能声称整个认领步骤可由数据库事务回滚。中断或回执不确定时必须保留明确的失败/不确定状态并停止，不得自动重写成功记录。

## 已执行验证

独立集成测试只创建并使用自身随机标签的 MySQL 8.4.11 容器。源结构由现有唯一注册入口 `di.InitTables` 生成；测试验证只读角色、完整行指纹不变、账本不存在、结构漂移拒绝，以及软删除和 NULL 边界。验证命令：

```sh
go test -race -tags=integration ./internal/dbschema -run '^TestLegacy' -count=1 -timeout=240s
```

## 显式命令与审计账本

```sh
schema preflight-legacy --dsn-file "$HOME/.config/ai-cloudops/schema-migrator.dsn"
schema adopt --confirm-legacy-schema --confirm-no-writers \
  --adoption-note-file "$HOME/.config/ai-cloudops/adoption-note.txt" \
  --dsn-file "$HOME/.config/ai-cloudops/schema-migrator.dsn"
```

预检命令会短暂持有同一schema advisory lock，执行完整SELECT/SHOW检查；不建账本，不修改业务数据。释放锁后的结果不能代替认领时的再次验证。只有显式 `adopt` 与两个确认flag，以及当前用户拥有的私有审计说明文件，才可进入认领路径。说明文件须单行、有效UTF-8、不超过1000字节，CLI去除首尾空白后保存，内容不输出到日志；不要放入密钥或密码。

认领仅支持与当前版本1结构和修复后置条件完全一致的未版本化库。引擎在同一锁连接中重新验证，通过后才创建带 `origin/adoption_note/adopted_by` 的账本，使用单条参数化INSERT登记applied、固定checksum、数据库CURRENT_USER及时间。不会调用Migration.Apply，不会更改业务表或历史时间。已有完整账本仅只读核验，不重写原审计；已有空账本、dirty/pending/drift均拒绝，不会自动补认领。

账本CREATE可能隐式提交，INSERT回执丢失也可能已经成功；任何不确定结果必须保留现场核查，不自动重试或删除账本。若只创建出空账本，下次执行会拒绝，而不是猜测已验证并填成功。AdoptAPI的可信validator必须只读，不遗留事务；通用锁回调若使用事务必须deferRollback及有界上下文，无法承诺立刻强制关闭违反契约且保留事务的回调。

认领不等于服务可以启动：还需单独只读核对通知clock与运行账号权限。缺marker但有历史通知时间时，`init-clock`会拒绝；必须由发布负责人独立核实或迁移时间数据，不能为绕过启动检查伪造marker。较旧结构升级、生产数据验证、生产发布和自动DDL回滚仍不在已验证范围内。

CLI真实自建MySQL测试覆盖只读预检、结构漂移拒绝且无账本、精确当前库认领、重复执行不改审计、业务行保留以及独立clock登记。
