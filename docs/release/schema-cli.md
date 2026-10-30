# 显式数据库迁移命令

`cmd/schema` 是独立的迁移操作入口，不读取应用 `.env`、Viper 或运行账号配置。默认命令只展示程序内嵌的版本清单，不连接数据库，也不判断目标库是否已完成迁移。

```sh
go build -o /tmp/cloudops-schema ./cmd/schema
/tmp/cloudops-schema
/tmp/cloudops-schema plan
/tmp/cloudops-schema --help
```

清单包含版本号、名称和不可变 checksum，不输出凭据或 SQL。`plan` 是离线清单，不能作为数据库验收结果。

## 凭据与只读检查

数据库命令要求显式指定当前用户拥有的私有 DSN 文件，或从专用环境变量 `CLOUDOPS_SCHEMA_DSN` 读取。指定文件时不会回退到环境变量。文件须为普通文件，禁止软链接和组/其他用户读写权限；可使用 `0600` 或 `0400`。不要把 DSN 放在命令参数、源码、日志或仓库文档中。

DSN 必须指定目标数据库；指向直连同一个 MySQL primary 的地址，不能使用多写节点路由。当前迁移基线经过 MySQL 8.4.11 验证。工具拒绝 `multiStatements`、`allowAllFiles`、明文密码插件和认证降级选项。连接使用独立的静默 MySQL logger，错误只输出固定分类，不输出底层错误、SQL、文件路径或 DSN。

```sh
/tmp/cloudops-schema verify --dsn-file "$HOME/.config/ai-cloudops/schema.dsn" --timeout 2m
```

`verify` 只有 SELECT：账本不存在、版本待执行、未来版本、checksum 不一致、failed/started 状态均失败，不自动创建账本或修改业务数据。它验证版本历史，不能发现任意人工表结构漂移。只授予账本 SELECT 的角色已在隔离 MySQL 中通过该命令验证。

超时默认两分钟，允许 `1s` 至 `30m`；操作上下文和网络连接/读写均有超时。SIGINT、SIGTERM 触发上下文取消。写操作被取消后可能已有 DDL 提交，必须检查状态后再决定后续操作。

## 仅新库初始化

```sh
/tmp/cloudops-schema migrate --confirm-new-database \
  --dsn-file "$HOME/.config/ai-cloudops/schema-migrator.dsn" --timeout 2m
```

必须同时指定 `migrate` 和 `--confirm-new-database` 才能进入写入路径。独立迁移账号需要 DDL、账本读写及数据修复权限，并能看见目标 schema 的完整元数据；不要使用只看见局部表的受限账号认定数据库为空。实际部署的角色创建及 GRANT 仍需单独接入。

命令先只读检查账本。已完整应用当前版本时直接成功，不再写入。dirty、drift 或未知错误直接拒绝。新库或空账本还须通过只读空库预检；任何已有业务表或视图都会阻止写入，旧库不会因该预检失败被新建账本或写入 failed 记录。迁移引擎取得锁后在同一个物理连接内再次进行空库检查，避免命令行预检与实际执行间的并发窗口。

当前只支持版本1的新库基线，不支持未版本化旧库采用、已有业务库升级或自动向后迁移。旧库须经过另外验证的采用流程。版本执行失败或回执不确定时工具不重试、不删除账本、不标记成功；MySQL DDL 可能部分提交，需人工核对并向前修复。命令完成后再次验证账本才返回成功。

应用启动现在只读核验版本账本和通知时区，不再调用GORM AutoMigrate或隐式登记clock。新库须通过显式迁移和时区登记，匹配当前结构的旧库须走完整预检与审计认领；运行账号不需要DDL权限。旧版本应用仍可能执行AutoMigrate，维护期间必须停止所有旧writer，不能把新工具与旧启动方式混跑后宣称迁移锁覆盖全部DDL。

## 本地验证

```sh
go test -race ./cmd/schema -count=1
go test -race -tags=integration ./cmd/schema -count=1 -timeout=180s
```

普通单测覆盖命令缺省行为、写入确认、无效输入脱敏、状态门禁、错误脱敏、关闭连接及上下文 deadline。集成测试还验证私有凭据文件边界，并自行创建唯一标签、仅绑定 loopback 的 MySQL 8.4.11 容器；验证58张业务表及账本初始化、重复执行、只读角色、dirty 不修复、旧库不变。集成测试不接受外部 DSN，不访问既有数据库，核对标签后清理自己的容器和卷。

本阶段没有连接或迁移生产环境，也未验证真实部署数据的升级、回退、运行账号分权或应用启动准入。

## 显式通知时区登记

新库schema迁移成功后，还需在所有业务writer与其他迁移job停止时登记通知DATETIME解释时区。不能从机器时区猜测，也不能为了启动成功修改历史时间。下述写命令必须显式确认停写；本工具不会替操作者停止业务实例。

```sh
/tmp/cloudops-schema init-clock --confirm-no-writers --notification-timezone UTC \
  --dsn-file "$HOME/.config/ai-cloudops/schema-migrator.dsn"
/tmp/cloudops-schema verify-clock --notification-timezone UTC \
  --dsn-file "$HOME/.config/ai-cloudops/schema-runtime.dsn"
```

`init-clock`复用schema迁移锁，在同一连接只读验证当前版本，再使用事务锁住marker、检查通知事件与计划时间；只有无marker且没有这些历史数据时才插入。已有相同时区幂等，不同时区拒绝，缺失业务表也拒绝。它不创建表、不重写marker、不转换历史DATETIME。迁移锁为fail-fast，并发工具须看到明确失败后核查状态，不自动重试写入；业务DML不受advisory lock阻止，停写确认不可省略。

`verify-clock`只读验证版本和时区标记，不写入。时区须显式UTC或合法IANA名称，禁止Local与空值；别名按确切名称核对，不自动等价转换。初始化仍不支持缺marker且有历史通知的旧库，该情况需独立备份和人工核对流程。

隔离MySQL集成覆盖CLI首次登记、重复登记、只读检查及不同zone拒绝；模块额外覆盖历史数据、缺表、并发、只读角色和单连接池。未迁移任何既有部署数据。

## 离线角色授权计划

```sh
/tmp/cloudops-schema roles --database cloudops \
  --runtime-role cloudops_runtime --runtime-role-host '%' \
  --migrator-role cloudops_migrator --migrator-role-host '%'
```

该命令不读取凭据、不连接数据库、不创建账号或执行GRANT，仅输出明确分阶段的授权SQL。先由管理员创建独立角色/账号并审查继承权限，再应用迁移角色计划；基线表全部创建后才应用运行角色计划。不能给已有ALL权限账号叠加这些GRANT后声称已收敛权限。具体最小权限与实证边界见 [schema-roles.md](schema-roles.md)。
