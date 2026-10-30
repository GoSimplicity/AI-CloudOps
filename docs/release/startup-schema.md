# 启动迁移清单与 YAML 数据库门槛

当前启动已改为只读schema/clock验证。下文GORM注册与自动迁移描述属于历史实现或模型合同测试；新库由不可变基线显式初始化，旧库须单独预检/认领，运行账号不应授予DDL。见 [schema-cli.md](schema-cli.md) 与 [fresh-schema-start.md](fresh-schema-start.md)。

`InitTables` 补齐 `K8sYamlTemplate`、`K8sYamlTask`、`WorkorderNotificationQueue`，并显式注册原本通过云账号关联隐式迁移的 `CloudAccountRegion`。已有 MySQL 的合法索引定义保持不变，迁移仅创建缺失的表/列，不删除业务数据；重复启动仍使用 GORM 幂等迁移。回滚旧二进制不要求删除新增表，不能为了回滚清空已有 YAML 数据。

原始故障是启动和健康检查成功，但 `/api/k8s/yaml_template/1/list` 与 `/api/k8s/yaml_task/1/list` 返回业务错误，底层为 MySQL 1146。启动就绪本身不能证明每个功能表均已注册。

防回归分三层：

1. 普通 Go 测试使用 AST 读取 `internal/model` 的真实 `TableName` 方法，核对 `InitTables` 的显式 `AutoMigrate` 参数；新增持久化模型忘记注册将直接失败，不需要数据库或 smoke 标签
2. `schema_smoke.py --mysql-migration` 在本轮 healthy cloudops MySQL 容器中创建随机数据库和仅限该库的临时用户，运行真实 Go 迁移测试，核对每张声明表、重复迁移、YAML 模板/任务 DAO CRUD 及数据保持
3. 三仓 integration 工作流启动新镜像后真实登录，调用两个 YAML list API 并检查业务 `code=0`，避免 HTTP 200 掩盖数据库错误

隔离工具不修改 `cloudops` 业务库，不重启共享服务。管理 SQL 仅包含工具生成的随机标识，测试 SQL 由限权用户执行；密码只经容器环境和子进程环境传递，不进入 argv 或日志。成功、失败和超时均在 finally 回收自身连接、数据库和用户，最后核对遗留资源为 0。Go JSON 测试事件必须确认目标测试实际 pass，skip/没有匹配测试不能冒充成功。

完整生产模型不在单个 SQLite 数据库中 AutoMigrate：MySQL 的索引名按表分隔，SQLite 的索引名在整个数据库中共享，合法的同名索引会形成测试方言冲突；不为这个冲突改变生产 schema。实际完整迁移在 MySQL 上执行。

已执行先红后绿：默认 AST 检查首先明确报出四个未注册类型，修复后通过；真实本机 MySQL 随机限权库的全量迁移、两次幂等运行和 YAML CRUD 已通过，清理核对为 0。未在本阶段部署共享镜像，因此现有旧镜像的 YAML API 恢复由主验收流程在更新镜像后核实。

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test ./pkg/di -run TestStartupRegistersEveryPersistentModel -count=1
DEVELOPER_DIR=/Library/Developer/CommandLineTools python3 scripts/integration/schema_smoke.py --mysql-migration
# 启动新镜像后，将测试管理员密码通过既有环境变量提供
python3 scripts/integration/schema_smoke.py --url http://localhost:18080
```

新增 CI 步骤使用 Go 的实际版本文件与真实 MySQL。没有调用云 SDK、Kubernetes apply 或生产服务。
