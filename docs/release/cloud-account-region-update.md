# 云账号地域差量与原子更新

原实现更新地域时先软删除全部旧地域，再重建列表。现有 `(cloud_account_id, region)` 唯一索引会使保留原地域或重新添加历史地域失败；基本信息与凭据又通过事务外连接先写入，导致失败时部分提交。已使用旧源码 Go overlay 在真实隔离 MySQL 中复现 1062，并通过先红后绿的事务回归确认修复。

更新现在锁定父账户行，在同一事务中写入基本信息、加密后的凭据与地域差量。未修改地域的 ID、状态、创建信息及关联资源保持；地域名称、描述和默认标志按提交集合更新，新增地域才创建新行。历史软删除地域重新添加时恢复原行，保留 ID 与原状态，不通过重建绕开唯一约束。

移除地域前检查未软删除的本地云资源引用，停止状态的资源同样保留关联。存在引用时明确拒绝，整个事务回滚。没有修改生产索引、表结构或创建响应。地域请求仍使用完整选中集合；省略 `regions` 表示保留当前集合，前端未修改地域时应省略，避免提交旧快照。并发完整集合按父账户行锁串行，最后一个显式集合生效，不承诺自动合并两个用户的完整编辑。

原子边界包括服务本次更新中的基本信息、凭据与地域；不执行云验证、同步或云资源修改。已有独立地域管理/同步写入路径仍须遵守各自的关联检查，不将本次账户行锁解释为分布式云端事务。

验证使用 SQLite 与随机限权 MySQL 库，覆盖保留 ID/禁用状态/资源关联、引用拒绝、历史恢复、数据库 CHECK/触发器强制失败后的全部回滚、读取旧快照后省略地域仍保留较新选择。真实 MySQL 另覆盖并发完整集合串行与唯一约束。测试库、用户及连接均 finally 清理为 0，没有调用外部 SDK。

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race ./internal/tree/service ./internal/tree/dao
DEVELOPER_DIR=/Library/Developer/CommandLineTools python3 scripts/integration/schema_smoke.py --mysql-cloud-account
```

同一 MySQL 验收已接入三仓 integration 工作流，普通前端创建接口仍返回现有空成功结果，不新增猜测的 ID 响应。
