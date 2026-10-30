# 数据库运行角色与迁移角色

`cmd/schema/roles.go` 的 `BuildRolePlan` 只生成离线授权语句。调用者必须显式提供目标数据库名、运行角色的名称和 host、迁移角色的名称和 host；函数不连接数据库、不创建角色或用户、不生成密码、不执行 GRANT，也不自动改变已有账号权限。命令行入口另行接入，不能把构建计划当作授权已经生效。

角色引用必须对应管理员已经创建的 MySQL role；运行账号与迁移账号的角色绑定、默认角色启用、认证方式和私有凭据文件由独立配置处理。不得把两个角色同时授给运行账号。

## 权限清单

| 对象 | 运行角色 | 迁移角色 |
| --- | --- | --- |
| 固定清单的57张业务表 | 逐表 SELECT、INSERT、UPDATE、DELETE | 逐表 SELECT、CREATE、ALTER、INDEX |
| 两项历史修复目标表 | 同上 | 额外 UPDATE，限 `cl_tree_cloud_resource_change_log`、`cl_workorder_notification` |
| 通知时区 clock 表 | SELECT | SELECT、INSERT、UPDATE，以及 CREATE、ALTER、INDEX |
| schema history 账本 | SELECT | SELECT、INSERT、UPDATE、DELETE，以及 CREATE、ALTER、INDEX |
| 目标 schema 元数据 | 通过上述逐表权限查看已知表 | schema 级 REFERENCES、TRIGGER，保证未知旧表及触发器元数据也可见 |

总计58张应用表（含 clock）和一张版本账本。生产授权不解析任意 SQL 猜测对象，使用固定、可审查清单；单测直接对内嵌不可变版本1执行内容做内存捕获，精确比较58个 CREATE TABLE 和两项 UPDATE 修复对象。

运行角色不获得数据库级 DML 权限，避免 `GRANT ... ON database.*` 意外覆盖账本与时区标记。迁移角色也不获得数据库级 SELECT、UPDATE、INSERT、DELETE 或任意表 CREATE；DDL限定到当前清单。没有 DROP、GRANT OPTION、CREATE USER、全局管理权限或 `mysql.*` 数据访问权限，不能把该计划当成回退工具。未来新增迁移对象时必须同步新增并验证授权清单。

clock 的 UPDATE 是 MySQL `SELECT ... FOR UPDATE` 所需权限，真实隔离 MySQL 8.4.11 已验证仅 SELECT+INSERT 会被拒绝。迁移命令仍只在没有历史冲突时插入标记，绝不自动改写既有时区。迁移角色属于受信任维护权限，凭据持有人可以在命令外 UPDATE 标记或 ALTER 表；不能声称其凭据本身不可修改 marker。相关权限要求见 [MySQL 锁定读文档](https://dev.mysql.com/doc/refman/8.4/en/innodb-locking-reads.html)。

## 完整元数据与标识符边界

只授已知表权限会隐藏未知旧表，导致“空库”预检漏判。迁移角色额外获得目标 schema 的 REFERENCES，能够枚举全部表元数据，但不能据此读取或修改未知表的行；隔离测试创建未知旧表，验证迁移角色能发现它，空库预检返回拒绝，直接读取其数据被权限系统拒绝。

已知 a8f1700 升级的 safe-v2 预检另要求实际 `CURRENT_USER()` 具有直接 schema/global TRIGGER 权限，再证明目标 schema 没有触发器；仅 REFERENCES 无法证明触发器不存在，role 继承不代替直接授权证明。运行角色不获得 TRIGGER。MySQL 的 TRIGGER 权限也能创建/删除触发器，并非纯只读权限，因此只授受信任维护账号；该账号及凭据不进入应用。工具不自动删除未知触发器，发现即在账本写入前拒绝。

下面只是维护账号直接元数据授权的示例，不会由 `schema roles` 自动执行，也不创建账号或替代完整最小权限角色；管理员必须确认实际认证账号与目标数据库。

```sql
GRANT REFERENCES, TRIGGER ON `cloudops`.* TO 'cloudops_migrator'@'%';
```

数据库名只允许1至64个 ASCII 英文字母或数字，并拒绝 MySQL 系统 schema。下划线和百分号在数据库级 GRANT 中可能成为通配符，且 `partial_revokes` 会改变解释方式，因此工具直接拒绝这些名称，不猜测服务端模式。已有带下划线的库不能用该计划生成器；未来若支持，必须明确验证服务端模式和精确授权行为。角色名只允许1至32个字母、数字或下划线；host 必须显式提供并通过有限字符校验。host 中通配符由调用者明确选择，工具不自动添加。参见 [MySQL GRANT 文档](https://dev.mysql.com/doc/refman/8.4/en/grant.html)。

## 新库顺序与已有宽权账号

1. 管理员单独创建目标数据库、两个独立角色及其专用账号，检查没有继承或动态宽权限。
2. 应用 `RolePlan.Migrator`，然后使用迁移专用账号执行显式新库迁移与时区标记初始化。迁移计划的逐表授权包含 CREATE，因此可以在目标表尚未存在时授予。
3. 表和账本全部创建成功后再应用 `RolePlan.Runtime`，随后为运行账号绑定、启用运行角色。纯 SELECT/DML 的逐表 GRANT 在表尚不存在时不能作为初始化步骤执行，不应为绕过此限制给运行角色添加 CREATE。
4. 使用真实运行账号验证业务读写、账本与时区只读、DDL拒绝，以及应用启动的只读验证。

新增 GRANT 不会撤销旧权限。不能把此计划直接套到现有 `ALL ON database.*`、全局权限、角色继承或同时持有迁移角色的账号上，并宣称已实现最小权限。旧账号必须先审查完整直接/继承/默认角色授权，再采用独立新账号，或在另行授权与验证后回收宽权；本工具不会自动 REVOKE 或连接现有数据库。

当前只完成离线计划及本地隔离权限验证。真实部署账号分权、Compose 凭据接线、旧库采用、升级及回退仍需后续完成；本阶段未访问或改变生产环境。

## 验证命令

```sh
go test -race ./cmd/schema -run '^TestRolePlan' -count=1
go test -race -tags=integration ./cmd/schema -run '^TestMySQLRolePlan' -count=1 -timeout=180s
```

单测覆盖标识符拒绝、系统 schema 拒绝、角色分离、语句范围和不可变基线清单。集成测试自行启动带唯一标签的 loopback MySQL 8.4.11，创建角色并真实执行计划：迁移角色完成基线与时区初始化，运行角色完成业务 CRUD；运行账号不能写账本、clock 或执行 DDL，迁移账号不能随意创建未知表、访问其他库或读取 `mysql.user`；两个账号都能看见全部58张应用表和账本。测试核对标签后清理自己的容器与数据卷，不接受外部 DSN。

## CLI与结构化分阶段输出

`schema roles` 提供离线计划。显式传入 `--database`、`--runtime-role`、`--runtime-role-host`、`--migrator-role`、`--migrator-role-host`；默认SQL按迁移前/迁移后分段展示。编排工具必须使用 `--section migrator --format json` 或 `--section runtime --format json` 分阶段读取，JSON合同为 `version: 1`、`database`、`section` 及有序 `statements` 字符串数组。JSON不支持混合all阶段，避免在建表前误授运行权限。输出不含凭据，不连接数据库或执行授权。
