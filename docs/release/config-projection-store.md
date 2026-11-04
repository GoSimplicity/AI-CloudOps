# 投影存储与 Redis fencing

`ProjectionStore` 接受真实 SQL 租约并从其深拷贝快照生成私有 handle；调用者只能按已签发 EventID/ControlID执行，不能提供任意物理 key 或正文。构造器接收当前实际的 standalone `*redis.Client`，复制 options 创建专用连接池并关闭命令重试；Close 不影响原应用池。本阶段不宣称支持 Redis Cluster/Sentinel/failover。

## 写入顺序与版本合同

Prepare只更新新投影控制 fence，不改旧正文、成员集合、intent 或历史 publication SHA。每次准备、正文/清退、成员、intent 与 ACK 操作前均在原 sealed SQL lease 上重验，连接失效不能另借SQL连接续租。

池 generation 与 namespace EventID 分别使用固定20位十进制字符串比较，支持大于2^53的相邻整数，不转Lua number。低版本在任何写入前拒绝。同版本还必须匹配nonce、完整key/role映射、artifact/action/body、control与ACK内容；SQL signature不是可供调用方换payload的通行证。Lua先检查全frame的参数、角色、TYPE及所有fence，随后才进入写区；Redis Lua错误不会自动回滚前面的写入。

物理key只从typed family/pool/binding导出，保持原YAML字节与publication SHA string。新控制receipt另存。完整ACK要求所有artifact完成且正文/清退结果正确、完整members集合匹配、AM intent控制完成且实为空。family5仅拥有webhook subset ACK，不能碰完整AM成员、intent或SHA；普通lineage只允许invalidate/retire，不能渲染正文或确认pool。

## 不确定结果与恢复

命令没有透明重试。传输结果不明不能当回滚或成功；保持SQL committed事实，通过nonce/lastCommitted重载原计划，在新租约上重建handle并按已存Redis receipts完成恢复。完成后的SQL phase不进入不可变fence签名，所以已投影计划重建仍可幂等执行。

原生测试真实丢弃已执行Lua的Redis回复，观测正文已写而SHA未确认；调用返回不确定错误且只建立一次连接，随后重载SQL计划成功恢复。没有以抛异常代替Redis实际写成功，也没有用关闭旧goroutine代替迟到命令fencing。

## 已执行验收及当前边界

专属Redis7.4.11 standalone实例、动态loopback端口、私有ACL/ownership marker与600连接文件实际运行9个子场景，零跳过：正文与空控制/phase重建、TYPE零partial、同版冲突/role拒绝、真实SQL EventID大于2^53的迟到render与retire、AM intent完整性、webhook subset/ordinary权限、已执行raw丢回复恢复，以及真实SQL generation大于2^53的迟到空ACK。

另3项独立原生审查通过：新版本Prepare后旧prepare/raw/members/ACK四种真实Lua命令全部STALE、所有相关key DUMP保持；同版篡改legacy SHA与重定向artifact至未见namespace均拒绝且零写。旧SHA反例实际返回OK并写入，修复加入完整behavior frame摘要后原预期转绿；重定向反例只验证修复后，不冒称已取得旧红。首次2个大ID用例在SQLite sequence夹具失败而未到协议，改为合法真实事件anchor后实跑通过，不混记成fencing失败。

本阶段仍未将新Store接入现有发布器或恢复worker，不能据此宣称两个已复现的实际publisher窗口已消除。新的opt-in原生测试目前有本机执行证据，CI自动供给隔离实例另需接线。逐步投影仍可能暂时可见，完整确认依靠新receipt/fence，不宣称跨SQL/Redis原子事务。完整计划每步重验的SQL/协议规模尚需真正发布器接线后计数和性能验收。

本机预览继续clean5feb4cc/schema4、18080/8889/18089，演示数据保留，Cron及外部执行关闭；本次Store源码未刷新入预览，没有上线或连接生产。

最终整合检查：全Go测试、vet及integration/e2e/smoke标签编译通过；完整cache/DAO race与vet通过。独立原生overlay文件已逐字节移为永久回归。专属Redis实例以所有权label精确删除，连接文件及ACL清理，未使用演示Redis16379。


## 自动供给 CI 原生实例

`CLOUDOPS_TEST_PROJECTION_STORE=true go test -race -tags=integration ./internal/prometheus/cache -run '^TestProjection(StoreOwnedRedisProtocol|OwnedDockerRejectsRemoteOverrides)$' -count=1 -timeout=120s` 会生成新的专属Redis实例与私有ACL，动态绑定loopback端口，写入owner marker后运行12个真实协议叶测试。它不发现或复用开发机Redis，也不会读取调用者提供的连接文件。关闭实例前核验所有权label和完整container ID，按ID删除后再次确认无遗留；创建结果不确定时也先核身份再清理。

入口在任何daemon操作前拒绝DOCKER_HOST/DOCKER_CONTEXT覆盖，只读取本机Docker context元数据，验证unix绝对地址及实际Unix socket，再对所有daemon命令显式传--host。两项无Docker CLI路径的回归证明远程环境覆盖在调用CLI前被拒绝。工作流在自有fresh stack已经缓存相同7.4.11镜像后执行，使用--pull=never，不拉取新镜像或使用演示16379。

本机实际结果为12项真实协议+2项入口门禁通过、零跳过/失败，tagged vet、actionlint与来源锁测试通过，实例清理为零。验证使用干净62e4c107源码加这两个CI文件的隔离本机checkout，未把并行中的Record接线WIP算入证据；文件SHA与提交候选一致。Linux GitHub运行结果另行跟踪，源码接线不等于已取得远端绿灯。
