# AlertManager 配置快照与发布

生成器一次读取并复制启用发送组，同一快照用于路由、webhook文件、真实固定Alertmanager v0.31.1解析校验及内容指纹。修改发送组间隔不再被pool-only哈希跳过，禁用组不会生成通知路由。确实没有启用组时发布合法blackhole配置并清除精确归属的旧webhook；读取失败不当作空集合。

GroupByStr补齐SDK实际序列化的group_by字段，保留原IR；自定义分组、纯通配及UTF-8按SDK校验，混用通配等非法组合拒绝。全局从SDK DefaultGlobalConfig起步，不再强制关闭SMTP TLS；HTTP默认及既有时长fallback保持，没有新增SMTP接收器或连接测试。

## 归属与完成状态

任何写入前读取完整artifact元数据并预检归属。当前和退休键都要求PoolID、ConfigType、生成器命名格式及canonical InstanceIP/groupID一致；同单值键有owned与custom/foreign并存时失败关闭，不能因已经有owned行而忽略冲突。跨页扫描不遗漏后页冲突，不通配清Redis、不删除他池、自定义或非canonical alias。

Alert专用元数据保存比较Name/PoolID/Type/正文hash/content/status，修复同正文pool更名仍需更新文件名的问题；未修改共享BatchConfigManager或DAO接口。公开wrapper签名及成功持久化副作用保留，main使用无副作用的读取/渲染流程。

完成状态使用独立publication前缀及固定pool.ID。旧Name哈希原样保留但不信任/不镜像，首轮强制刷新；不会与旧Name恰为pool:ID的键碰撞。首次可见artifact修改前必须可靠写pending:目标SHA，所有必要存储、成员和清理完成后才写纯SHA。失败后源回A或继续B、甚至重建cache实例都必须修复实际Redis与DB，不能误命中旧ACK。

这是本实例串行与幂等恢复，不是数据库/Redis跨存储事务或分布式锁；部分合法新artifact可能已可见而最后确认失败。错误只记录固定阶段和PoolID，不回显组配置、URL或凭据。

## 验证范围

本阶段新增172叶场景，定向race共204项测试/子场景、cache vet及差异检查通过；主线程提取并执行全部25个新增测试函数通过。独立审查发现并转绿的混合归属、源回退及更名反例均保留，覆盖冷启动pending失败零写、当前/退休歧义键、分页、pool改名同正文元数据更新、启用/禁用与blackhole、稳定排序、并发、取消等。

测试使用内存DAO/Redis/FakeDB与真实SDK解析，不连接真实AlertManager、SMTP、数据库、Redis或生产服务。完整Go整合将与其它发布补修冻结后独立执行，不能把本定向结果冒称所有集成链已验。

## 2026-11-04：主配置无索引历史行退役

主配置退役现在完整、严格分页读取type2元数据，复制每页记录，检查总数、唯一正ID、提前空页及类型一致性。在任何写入前，以PoolID/type/IP及`alertmanager_pool_<ID>_<IP>.yaml`规范名称确认归属；pool显示名称变化不作为所有权依据。

已验证的非desired数据库地址与可信成员索引合并为stale集合，当前和旧地址都检查混合归属冲突。即使源ACK未变，新增DB-only规范孤儿也不能走快速完成分支。pending可靠后先把stale地址登记成员，再经SQL条件退役、Redis删除、成员移除，避免DB已软删而Redis失败时失去恢复定位。当前目标、custom/foreign和type5 intent保持，未知Redis-only键不扫描猜删。

新增7主/24子、27叶测试及原始Fake/真实SQLite+HTTP反例全部通过；独立实际DAO/HTTP race验证迁移和更名失败重建、同ACK晚出现的DB-only孤儿及foreign混合零写保护。最终全Go test/vet、cache/DAO完整race、全标签只编译通过。没有修改schema、空AM目标政策、AM池删除或公开构造合同，没有连接真实AlertManager或上线。
