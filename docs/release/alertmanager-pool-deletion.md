# AlertManager池删除生命周期

## AM-D1：发送组引用与父池锁

服务层原本已检查包括禁用组在内的发送组，本阶段保留该正向保护；修复的是预检后并发插入、迁入已删除父池等交错。最终DAO事务锁定AM父池，检查全部未删除发送组，再执行软删除。

发送组创建或迁移在同一写事务内先锁定存活的目标AM池，再锁定子行及更新三类用户关联。省略PoolID的struct更新继续保留原关联，但在锁内重检初读值未被竞争迁移。Scrape池即使有相同数字ID，也不能充当AM父池。失败或取消回滚定义与关联表，不自动删除用户发送组。

新增14叶场景及相关DAO/service/cache测试通过，原三条定义交错反例和禁用组保护对照保持原预期转绿。独立实际SQLite DAO race验证省略关联竞争迁移及取消时发送组/三表全部回滚。SQLite只证明事务与行为，不冒充MySQL行锁。

本机新建专属MySQL空库与最小权限测试账号，实际运行创建/迁移分别先于删除或晚于删除的4个锁顺序场景，全通过且零skip。临时库、账号和DSN清理，演示数据未改；最终全Go test/vet、相关三包完整race、全标签只编译通过。

## AM-D2：持久删除意图与同步退役

真实AM软删除记录现在提供持久退役授权。普通活动列表不变，专用生命周期查询完整读取并复制所有页，验证总数、唯一ID与删除状态；只有真实AM tombstone可授权，Scrape同号池、普通缺失、旧poolName或读取失败都不能代替。

删除服务在D1最终事务后，同步等待AM type2主配置及type5已证明owned的Webhook输出清退，全部完成才返回成功；部分失败明确pending，同ID重复DELETE可从真实tombstone继续。已删除分支跳过坏历史发送组/渲染，保留用户定义和custom/foreign。publisher mutex等待旧inflight结束再验证授权，删除状态进入独立版本。

主配置成员与owner-only Webhook intent在pending后可靠登记，SQL CAS先于Redis清理，最后移除恢复定位并确认SHA。DB-only历史行、取消/重建、旧intent与新owned同键二次ABA继续遵守已交归属合同；未知无DB/索引Redis键不猜删。

周期管理器先完成四类独立退休批次，各任务使用父超时上下文而不互相取消；随后运行原有正常生成组，enable条件不变，退休与正常错误汇总返回。正常fail-fast不能持续取消退休，AM某个foreign冲突也不能阻断其它Scrape退休；父上下文取消/超时仍生效，没有开启被暂停的正常生成。

新增54叶及相关四包race/vet通过；实际文件SQLite+Gin三场景连续50次race通过，覆盖重复DELETE、周期重建、DB-only type2/5、历史孤儿组及同号Scrape保护。原审计断言仅为新增同步接口补fake适配后保持预期，未谎称旧接口fixture无变也可运行。独立真实DB/HTTP取消→重建且正常Prom错误仍完成2/5清退、原4个二次ABA场景均通过race。最终全Go test/vet、相关四包完整race、全标签只编译通过。

本阶段没有新增schema、修改CAS核心或Scrape发布器，不声称SQL COUNT为一致快照、跨存储事务、多进程exactly-once或真实目标已应用；没有外部重载、通知、生产访问或上线。
