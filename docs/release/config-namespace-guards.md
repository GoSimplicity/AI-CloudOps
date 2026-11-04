# 配置命名空间 SQL 写保护

本阶段增加 `cl_monitor_config_namespaces`，为普通配置及生成器写入提供同一个数据库锁地址。复合主键是配置类型和原始实例键的字节；表中没有配置正文、独立ID或软删除字段。原始键使用 VARBINARY(180)，可以容纳原配置列45个UTF-8字符，不做大小写、空格或IP表示规范化。DAO校验合法UTF-8、非空和最多45个字符，不做截断。

**当前已覆盖普通及生成器的全部SQL配置写入口。** 新建、迁入、批量写和生成器更新/清退在锁内重新检查完整命名空间。SQL提交后的Redis发布窗口仍待后续闭环；SQL保护不等于跨存储原子性。

## 版本与现有数据

独立v2迁移只创建锁表；v1 SQL及执行包装、已发布的已知来源升级实现保持原字节和校验和。没有给历史配置增加唯一约束，没有删除或改写自定义、非激活及重复配置。锁地址会永久保留，避免删除后并发写入者锁住不同记录。

`CurrentManifest` 是现代CLI、运行时及通知时区初始化的完整目标。仅供冻结旧库验证和升级使用的 `Manifest` 保持sole-v1目标。可追加聚合器放在独立文件，避免将来追加v3时改变自绑定的v2校验和。运行时只验证完整历史与通知时区，不执行DDL。

旧库采用及已知来源升级只登记v1并保留原审计来源；随后必须显式追加待迁版本。旧来源物理回滚演练仍验证精确v1账本、完整升级证明和时区，不偷偷应用v2，也不将现代运行时的待迁状态当成演练成功。文本证明按字节比较。

## 明确区分两种迁移授权

先使用 `schema plan` 查看离线计划。新库初始化使用 `migrate --confirm-new-database`；已有干净非空版本历史追加迁移使用 `migrate --confirm-pending-migrations --confirm-no-writers`。两种确认互斥，不将“新库”确认解释成修改已有版本库的授权。

迁移引擎取得同一数据库的迁移锁后，在持锁连接上重新读取并校验模式，再决定是否允许DDL或账本写入。fresh只接受真正无业务表的无历史/空历史库，pending只接受完整、干净、非空的已应用版本前缀。失败、漂移和模式变化拒绝；完整当前版本是只读幂等结果。锁外预检不能替代这次锁内检查。

fresh流程依次执行 migrate、`init-clock --confirm-no-writers --notification-timezone UTC`、`verify-clock --notification-timezone UTC`；按实际要求选择明确的IANA时区。时区初始化也要求完整当前版本，不能在仍待迁v2时插入标记。凭据通过私有DSN文件提供，不写入命令、文档或日志。

运行账号对锁表仅有 SELECT、INSERT、UPDATE，没有DELETE或DDL；迁移账号获得创建该表所需的表级权限。事务保护已经真实双连接验证；跨SQL与Redis的发布窗口仍需后续方案和验收。本阶段未上线或连接生产。

## 验收证据

全Go测试、vet及integration/smoke/schema_smoke标签编译通过；schema与CLI两包新增确认模式的215叶race通过，原stage五包315叶race通过。173项发布工具测试通过，来源锁另核5项。

隔离MySQL8.4.11/RR实际运行8个子场景：fresh/空账本、已有v1升级、采用后显式升级、failed/drift拒绝，以及3个不同CONNECTION_ID连接配合channel的预检后竞争。竞争分别令fresh库变为v1、pending账本清空或消失；取得迁移锁后均拒绝且不创建v2/新账本意图。fresh迁移与init-clock/verify-clock真实链及v1 pending时拒绝初始化也通过。独立Docker角色测试确认锁表读/插入/更新允许，删除/ALTER拒绝。临时库、专属用户、DSN及角色测试容器均清理。

修改后的完整E04物理演练已重新执行：旧49表12行经备份升级至58表，再恢复到隔离新卷；旧模型数据指纹、NULL/软删除、合成密文、Redis队列与标记、知识文件和私有配置一致。物理数据复制观测2.948秒，不是整站RTO，也没有启动应用/worker或重放外部副作用。演练容器与卷已按所有权清理，仅留私有证明。

Web来源锁68bee75492336ed3b5140736dc4a71578517e87d自身远端Verify/gitleaks成功；AI保持ba1876a。后端新提交的三仓组合需另验，不能复用上一0ef/814/ba187集成成功。

## 全部 SQL 写入口接线

普通Create/Update/Delete、BatchCreate/BatchUpdate和GeneratedUpdate/Delete先按类型与原始键的固定顺序，事务内UPSERT永久guard行并取得行锁，再执行完整命名空间的FOR UPDATE当前读。普通单项操作复用相同批量合同；没有遗留可绕过协议的假BatchUpsert入口。

新建或迁入目标只要存在任何未删除配置就拒绝，包含其他池、自定义名称和非激活行；输入批次内重复键也拒绝。原同ID同键编辑保留struct Updates省略零值的行为。历史重复不自动删改，可明确按ID删除或迁出到空键修复。源行在取得guard后按ID重读，归属变化拒绝，不临时追加无序锁。

生成器仍使用不可变Expected owner CAS，并额外要求完整命名空间唯一、归属可信。完成更名后的完全Desired重试仍幂等；首次清退和tombstone重试均检查全命名空间，不能因非激活/新owner行被隐藏而授权删除。

创建批次只有一个外层事务，100只是SQL执行分块。先复制输入，全部成功提交后才回填ID和时间；后段真实SQL错误回滚前段且输入保持原样。取消保留context错误身份，1205/1213归安全重试错误；提交确认异常单独标为不确定，不把它谎称为已证明回滚，也不自动重放最后一条语句。调用方需重新读取后决定重试。

锁内查询保留BINARY/COLLATE BINARY最终字节比较，并增加可索引的普通实例键预过滤。原生测试实际复现BINARY-only扫描令两个不相等键发生1213死锁；修复后原屏障用例通过，EXPLAIN从deleted_at/config_type的index_merge变为instance_ip的ref查找、估计1行。没有新增index/schema，也不宣称所有数据库死锁已消失或给出线上延迟收益。

## 本阶段验证与边界

独立SQLite契约9主40叶race通过，同一测试覆盖回已提交旧实现有30叶业务失败；101条输入的最后真实UNIQUE/PK错误证明整批及输入ID不泄漏。真实publisher在预检后通过普通DAO插入active/status2 owner的2个原红例已转绿。原pager、分页、归属、取消、历史重复和所有恢复断言继续保留。

专属MySQL8.4.11/RR、两个独立连接真实运行7个场景零skip：旧RR快照之后的新owner、1205与重读、取消、源归属漂移、101后段PK失败、非激活重复阻止生成器更新/首次清退及无关键更新。原生成owner的changed-rows/found-rows两个模式再次原生通过。完整cache/DAO race、vet、全Go及全部标签编译通过；专属库、用户和DSN清理。

这些保证要求所有元数据写入者参与同一DAO协议；旧二进制或绕过DAO的直接SQL不受应用锁协议保护，交付时应停旧写入者统一更新。SQL完成后再进行的Redis SET/Del/ACK尚未与普通写操作统一串行化或fencing，此窗口不能用本阶段的SQL回归宣称关闭。未上线。
