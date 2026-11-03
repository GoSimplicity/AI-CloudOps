# 录制规则配置合同

保存与生成共用 `utils.BuildRecordingRule`，使用当前锁定 Prometheus/Common 的 UTF-8 默认名称方案。保留中文、连字符及其它合法 UTF-8 名称，拒绝空白名称和名称中的花括号；不套用会误拒合法历史名称的 ASCII 正则。表达式必须语法合法并返回即时向量或标量，不能保存后再静默丢弃字符串或矩阵结果。

Labels 按第一个 `=` 拆分，保留空值、后续等号和原始值，重复键最后一个值生效；拒绝缺少分隔符、非法 UTF-8 及保留的 `__name__`。服务持久化的原始数组合同保持，生成时写入真实 `rulefmt.Rule.Labels`。创建省略/零状态仍使用原 ORM 的启用默认1，更新省略/零状态仍交给既有 GORM 略零处理；非法非零状态在绑定和服务边界拒绝。

生成器只读取一次 DAO 快照，指纹与配置使用同一份数据；指纹纳入标签与目标实例，以长度前缀避免名称中合法分隔符造成碰撞，并更换版本强制刷新旧缓存。旧数据中的坏名称、表达式或标签按条目隔离，其余有效规则仍可发布；整份生成结果还须通过真实 rulefmt 解析。非空但全部无效的快照保留旧配置，只有确实没有活动规则时发布显式 `groups: []` 清除。

发布顺序为缓存/成员写入、配置持久化、旧实例清理、版本确认。任一必要步骤失败不确认该池的新哈希，因此同一快照可在恢复后重试；写失败的当前实例不会被误认成旧实例删除。旧缓存删除成功后才移除成员索引，保证清理失败也能重试。刷新返回阶段错误而非假成功；新发布阶段错误仅记录池ID和阶段，不回显配置或后端错误内容。

这不构成 Redis 与数据库之间的事务：持久化失败时，部分合法缓存可能已经可见，随后重试收敛。本次未迁移历史规则或连接真实 Prometheus/Redis/数据库；已有坏配置需通过正常管理流程修正。

## 测试矩阵与环境

使用仓库既有原生 testing、内存 DAO/Redis/配置仓库替身及真实固定版本解析器，无新增依赖。Go 1.26.10；Prometheus v0.311.3，Common v0.67.5。

| 范围 | 新增叶场景 | 验证点 |
| --- | ---: | --- |
| 构建合同 | 27 | UTF-8、vector/scalar、空/非法输入、完整标签值、错误不回显 |
| 保存与绑定 | 53 | 存储前拒绝坏配置、0/1/2状态、空/nil标签、名称长度、存储失败 |
| 真实生成与刷新 | 21 | 坏条目隔离、last-good、显式空清除、标签刷新、碰撞及单次快照 |
| 发布故障恢复 | 6 | Set、SAdd、持久化、Del、SRem的失败与恢复重试 |
| 已取消刷新 | 1 | 不读取或发布配置 |

完整源码见 [构建测试](../../internal/prometheus/utils/recording_rule_test.go)、[服务测试](../../internal/prometheus/service/alert/record_service_validation_test.go)、[生成测试](../../internal/prometheus/cache/record_rule_contract_test.go)、[故障恢复测试](../../internal/prometheus/cache/record_publication_retry_test.go)、[取消测试](../../internal/prometheus/cache/record_cache_context_test.go)。基线生成器的21场景中20个真实业务断言失败，修复后全部转绿；总计108个新增叶场景通过。

## 验证命令

```sh
go test ./...
go vet ./...
go test -race ./internal/prometheus/utils ./internal/prometheus/service/alert ./internal/prometheus/cache
go test -tags=integration,smoke,schema_smoke -run '^$' ./...
```

以上均通过。最后一条仅编译带标签用例，不表示执行了真实依赖集成；未执行上线、通知或主机操作。

## 完成标记补修：回退与更名

仅保留上次A的完成哈希不足以表示失败后的状态：B可能已部分可见，源回A时不能再用旧A标记跳过。现在首个可见写入前必须可靠写pending:目标哈希，全部完成后才写纯SHA；pending失败不改artifact。可信完成键改为固定pool.ID的独立publication命名空间，旧名称键保留但不再读取为完成依据，首次升级会重新发布一次。

新增30个record/Prom共用回归覆盖失败后恢复A/B、池改名后回退旧名、重建cache复用存储、旧Name恰为pool:ID的键碰撞、pending失败零artifact与本实例并发。连76个既有纯内存场景共106个race通过、cache vet通过，主线程另复验新增前缀全部通过。实例锁仅保护该cache实例，不能当作跨进程事务或分布式锁。
