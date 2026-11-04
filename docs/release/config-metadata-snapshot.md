# 生成配置元数据的完整读取责任

Record和Prometheus主配置共用私有读取器，只集中完整分页、总数一致性、提前空页/短页、正且唯一ID、取消及每页数据复制。每个池、每次调用仍重新读取，不跨池或轮次缓存。wrong-kind处理保留在原publisher，仍在全页身份校验后跳过；归属判定、生成和发布状态机不合并。

输入源复用上一页对象时，已捕获的元数据不会随下一页改变。DAO返回同时发生取消也不返回部分成功；错误仍由各publisher固定阶段对外呈现，不暴露内容。没有改变owner CAS、创建预检、hash、原生配置解析或SQL/Redis写入次序，未迁移Rule或AM。

新增25叶读取测试、完整cache race/vet、原量化probe及独立分页缓冲复用/每次fresh/末页取消反例通过。最终全Go test/vet、cache+DAO race、全标签只编译通过。没有新依赖、schema或公共接口变更，未上线。

## 量化边界

内存Fake稳态量化：每池1实例，AM每池1启用组，5池时Prom/Rule/Record各读5次返回25行，AM读10次返回50行；10池时前三各10次返回100行，AM20次返回200行。另有AM desired点查2N次。所有情境发布写入为0；这是精确调用/行数而不是真实SQL耗时基准。

本阶段提取校验责任后这些数量保持不变，**没有降低二次方行量**。减量应另设计按池owned库存及目标/旧键的完整归属查询，不能用长寿命全表缓存弱化并发检查。AM复合键、名称、Webhook意图和ABA逻辑继续留域内，不由相似plan字段推导通用发布框架。

## 定向DAO能力：2026-11-04

新增两个仅供控制器使用的分页入口：按PoolID/type读取全部未删除候选，保留custom及Status2；按type和最多100个精确实例键读取全Pool命名空间，保留foreign/custom/duplicate行。页码至少1、每页1—100，非法参数拒绝不截断，空键集合零SQL。键不trim或normalize，MySQL使用字节比较，SQLite使用BINARY排序规则，结果按ID稳定排列。

池候选不是归属授权；减量接线必须将desired、可信成员及canonical owned旧键合并后，再读取它们的完整命名空间，否则会漏foreign冲突。COUNT与页读取分开，消费者仍需完整校验每页；没有把读取变成事务一致快照或atomic create。普通CRUD及旧GetMonitorConfigsByInstances行为保持，schema/index未变。

新增18叶、完整DAO/cache race与vet、独立实际SQLite跨页101行/51候选、byte sibling、空键零查询和取消反例通过。隔离本机MySQL实际运行5组合同及byte子例，共11个PASS节点、零skip，验证CI/PAD SPACE列、所有者和非激活行、边界与旧API；临时库、专属账号和DSN已清理，演示数据未改。全Go test/vet/相关race/全标签只编译通过。

**发布器尚未接线，生产扫描量未减少。** 后续接线须重新验证归属保护、调用量和故障恢复，不能拿这项能力测试作为性能收益证据。
