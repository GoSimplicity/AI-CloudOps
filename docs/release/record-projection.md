# Record 规则发布与恢复

正常生成、显式停用、空目标与删除池的清退共用持久发布链。渲染器及下载用的原 Redis key 保持，发布代码不再用旧的 `Set → SQL → Del → SHA` 顺序，也不会在缺少 journal/runner 依赖时回退到旧路径。

## 一次发布的边界

1. 在读取池、生命周期或规则之前，从 SQL 读取 Record 控制版本。新 nonce 的版本预留在永久 control 锁内比较 expected counter，旧源快照不能因为读取较慢、较晚预留而取得更高版本去覆盖已经完成的新源。
2. 完整读取源与元数据，构造确定的配置、owner 快照、退役历史和完整成员集合。停用/删除不读取可能已损坏的旧规则，也不会重放旧 render。
3. 将业务配置、artifact 事件、v2 manifest 与 lastCommitted 在一个 SQL 事务中提交。所有退役历史在同一事务、control → 有序 namespace 锁后重验；前置读取不是提交权限。
4. 在另一个 SQL 租约中，从持久事实生成 Store handle。每个 Redis 操作前重验原连接；正文、成员及完整 ACK 经 fencing 执行，最后才标记 SQL projected。

SQL 配置保存成功与 Redis 投影完成是两个不同状态。`committed`、未确认的回执及投影错误不能展示成新版本已生效；旧 SHA 仍可能表示历史版本，不能单独用它判断新发布成功。

## 归属与历史数据

活跃的当前 canonical owner 是退役权限。已删除的 foreign/custom 历史行是 ABA 比较的观察数据，不会因此禁止当前 owner 清退自身，也不会授予删除 foreign 当前正文的权限。提交前出现新增、迁入、删除后再次创建等历史变化，必须在原事务中拒绝，不能靠第二次 public DAO 调用伪造原子性。

没有当前 owner 时，只读历史接口先按 pool/kind 定位旧行，再按精确 namespace 跨所有 pool/status/deletion state 消歧。它覆盖只有真实 SQL 墓碑、raw 仍在、member 和旧 manifest 都已丢失的情况。仅有 member/raw 而无 SQL owner 或可信完成证明时保留原缓存并报告未完成，不猜测 ID 或清删未知正文。

## 接管后的空状态恢复

旧 owner 的 SQL 行已退休，而当前 namespace 已由另一个 generated owner 完整投影时，旧池需要完成自己的空状态，不能一直把新 owner 当作待清退对象。该恢复只清理旧池自己的成员和确认记录，不包含新 owner 的 raw 或 namespace fence 操作。

接受过的 render/retire 使用已签发 op/event/owner 证明；升级前从未产生 pool journal 的真实旧墓碑使用显式 legacy provenance，不制造不存在的 op/event。替代方必须具有匹配的 generated binding、applied artifact、projected operation 及完整 manifest。custom、N/A 或尚未完成的替代方不能冒充该证明。

可选 `record_superseded` 只持久化 ID、owner 与内容摘要，不复制 YAML 或任意 Redis key。事务同时核对替代事实并记录解决结果；恢复加载验证不可变历史，所以进程重建、ACK 前失败或新 owner 以后正常更新正文，不会让旧池遗忘已经解决的归属。旧的无该字段 v1/v2 manifest 保持可读。

## 不确定结果与连接生命周期

SQL COMMIT 回复丢失后按 nonce 核对实际 manifest；reserved NULL 不等于成功。Redis 传输结果不明不会透明重发旧命令；保留 accepted SQL 事实，在新租约中重建回执恢复。原租约在退出、取消或检测到失败后不能换连接续命。

Record 显式接收 journal、history 和 projection runner。正式 runner 固定使用 typed Store；测试替身只替代外部投影边界，SQL 归属和租约仍来自真实数据库。Wire provider 管理独立 Store 连接池，后续构建失败时清理，主程序在应用停止写入者后执行 cleanup，关闭不会影响调用者原 Redis 连接池。

## 验证范围

既有单元和 HTTP 回归迁移到真实 SQLite journal/guard；测试内存映射仅作历史输入和断言视图，不制造租约权限。真实 TCP 原生回归在中间节点截留已经发送的旧 artifact 命令，让调用方先得到 EOF、原 SQL 租约结束，等 B 完整发布后才把旧包送达 Redis；必须实际读到 STALE，并保持 B 正文、成员、回执、fence 与 SQL owner 不变。仅让旧 goroutine 没有执行到写口不算通过。

真实源 DAO 的 pool/rule 读取屏障用于复现旧快照晚预留问题；接受安全 no-op 或显式错误，但不接受新完成状态被旧源改写。历史 guard 回归还以真实 generic commit 作旧行为对照，核对 create→delete 后 namespace 再空的 ABA。

本阶段仍只接入 Record。其他三类发布器、独立恢复 worker 与规模性能验收另行完成；不宣称跨 SQL/Redis 原子可见。预览继续固定已验收的 clean5feb/schema4，演示外部执行关闭；本次开发没有上线或连接生产。


### 本阶段最终结果

全Go测试、vet、integration/e2e/smoke标签编译通过。完整cache、config DAO、DI的race及vet通过，保留原Record与共享场景中其它发布器的业务断言；旧pending字面量检查只在Record分支迁为更强的持久未完成状态，未关闭测试、跳过分支或放宽CI规则。

owned实例最终JSON报告20叶通过、零跳过/失败：12项既有Store协议、2项Docker环境门禁，以及6项真正发布器回归（迟到render/retire两种TCP命令、legacy接纳及B后续清退、真实pool/rule源快照两种竞争、同hash新DB-only stale清退）。每次实例以label+完整ID精确清理，最终容器为零；没有使用演示Redis。

回归过程中实际复现并修复：旧源晚预留将gen1的新完成覆盖为gen2旧数据；旧池off在新owner完成后永久pending；无旧pool-op的legacy墓碑无法结清；当前owner因已删除foreign/custom历史而不能自退；已签retire在Redis失败后被fresh legacy消歧挡住；健康旧receipt掩盖同hash新增stale。原SQL ABA还用真实generic提交作旧行为对照。已有permit只在fresh完整目标均被其覆盖时重放，新stale必须签新的guarded计划，不能为恢复绕过权限或遗漏新元数据。

新后端三仓CI需在本次提交后验证。Web来源cd4f757的独立Verify41844229072及gitleaks41844228940已成功；AI仍ba187。前端最新完整实扫为1130文件、7771条遗留诊断/5条warning，fatal与保护规则0，不冒称全部ESLint已完成。

完整发布工具173项回归和actionlint通过，未执行真实备份或对外连接。
