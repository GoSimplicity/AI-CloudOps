# 恢复与租约的独立 MySQL 验收

本阶段使用新建的mysql:8.4.11容器、随机schema/user和动态127.0.0.1端口。拒绝13306/3306和Docker环境覆盖，不接入演示数据库；密码文件、env-file及DSN在700目录内以600权限保存。所有daemon命令固定经已核实的本机Unix socket，容器按完整ID及随机owner label清理，并验证该label剩余数量为0；没有prune或复用已有实例。

```sh
CLOUDOPS_TEST_PROJECTION_RECOVERY_MYSQL=true go test -race -tags=integration \
  ./internal/prometheus/dao/config \
  -run '^TestProjectionRecovery(OwnedMySQL|MySQLOwnedRejectsRemoteOverrides)$' \
  -count=1 -timeout=240s -json
```

实例只初始化五个journal相关模型。两个runtime连接池各MaxOpenConns=1，实际不同CONNECTION_ID为9/10，实读隔离级别REPEATABLE-READ；管理员连接仅用于观察本实例等待与终止精确核实属于该schema/user的租约连接。没有Redis，SQL状态标记用例明确不作为外部投影成功证据。

## 已执行场景

实际12叶全部通过、0fail/skip：两个远程Docker覆盖拒绝，以及十个MySQL场景。

- 当前lastCommitted候选、历史/projected/reserved排除、高水位和坏manifest后续分页。
- family1/3/4中B已经预留、A读到counter之后B提交而counter未再增加，旧A仍无法覆盖新LC。
- Commit锁内LC guard与同nonce已提交事实的幂等重读。
- AM2/5两个方向peer提交使旧向量失效，只有reserved的失败尝试不饿死last-good。
- 两个方向均实际观测2→5锁序、指定peer被指定lease阻塞，直到真实deadline取消；原租约仍有效，释放后peer能重试。
- 捕获租约原始连接ID、终止该连接，另一runtime提交新SQL内容。旧租约Recheck检测并seal之后，Snapshot、MarkComplete及fresh context均不能续权，新SQL连接也不能重放已失效旧receipt。

首轮10/12通过，两个等待用例因要求INNODB_TRX同步出现LOCK WAIT而假阴性；安全诊断已捕获真实peer→lease等待边和正确deadline。最终没有取消阻塞断言，而改为严格连接及表归属的data_lock_waits/data_locks联合正证据：requested为WAITING、blocker为GRANTED，两者均为本schema控制表的RECORD锁，匹配实际双方线程。2→5、deadline、原lease复验及释放后retry断言保持。

MySQL文档说明[事务与锁视图可能不一致](https://dev.mysql.com/doc/refman/8.4/en/innodb-information-schema-internal-data.html)，[data_lock_waits表达实际等待依赖](https://dev.mysql.com/doc/refman/8.4/en/performance-schema-data-lock-waits-table.html)。因此没有用Sleep推测并发，也没有把单个观测视图的缺失当作没有锁。首次失败与诊断日志均保留。

tagged vet、actionlint及来源锁合同检查通过；新用例已接入三仓CI。这里只证明SQL边界，不代表已执行MySQL与Redis组合的端到端交付、集群failover或生产容量验收。生产/Store/Lua/schema未改，性能优化仍待下一阶段；演示服务保持原状态，未上线。
