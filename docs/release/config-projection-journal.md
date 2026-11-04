# 配置发布的持久日志与投影版本

## 当前接线状态（2026-11-05）

PromMain1、AlertRule3、Record4及AM full2/subset5均已接入持久计划、原SQL租约与typed Redis Store。AM使用双source控制与只读coverage，见[AM合同](alertmanager-projection.md)；其它发布器见[Prom合同](prom-main-projection.md)和[规则共享合同](scrape-rule-projection.md)。[独立恢复worker](projection-recovery.md)已接入App，规模性能及隔离MySQL并发验收继续后续推进，不以已接线声明整个1.0完成。当前schema ledger为1/2/3/4；下方保留各阶段当时的交付记录，最新验收见[当前完成度](readiness-current.md)。

## 初始 v3 交付快照

当前已交付 v3 存储结构及日志事务接口；普通配置写入会在同一 SQL 事务记录事件。新日志计划接口尚未接入发布器，Redis fencing 与恢复 worker 尚未交付。已有的两处 SQL 提交后 Redis 竞争仍属于后续验收，不能用建表成功声称发布窗口已经关闭。

## 三种记录的责任

| 表 | 责任 | 保留要求 |
| --- | --- | --- |
| `cl_monitor_config_pool_controls` | 每个发布域/池的 generation counter、当前 attempt、最后 committed 和最后 projected 指针 | 永久锁地址；reserve 不覆盖两个 last 指针 |
| `cl_monitor_config_pool_operations` | 逐代操作、nonce、source hash、manifest、恢复来源与阶段 | append 历史；失败新 attempt 不抹掉旧空操作 |
| `cl_monitor_config_artifact_events` | namespace 单调事件版本、行/owner/body digest、投影授权与交付状态 | 保留退休与 A→B→C 证明；当前阶段无自动 GC |

这三张表不嵌入普通业务 Model，没有软删除或 YAML/凭据正文副本。`manifest` 的 SQL NULL 表示尚未提交，JSON `[]` 表示合法的空 manifest。operation nonce 使用16个二进制字节，digest使用32字节；namespace原始键使用 VARBINARY(512)，保留128个UTF-8字符的字节身份。

发布域包含四类完整发布和独立 AM webhook-only 作用域。后者不表示完整 AM 主配置已经完成。commit 后的 manifest、owner/digest/binding payload 将由后续 DAO 保持不可变，只有交付阶段允许条件推进；模型注释不是这一业务校验已经实现的证据。

## 后续接线必须满足的合同

reserve 在永久 control 行的锁下分配 generation，不能用空集合 `MAX+1` 假装互斥。metadata、artifact events、committed manifest 与 lastCommitted 必须同事务提交；SQL 确认异常按 nonce 重载事实，不能猜失败或重放旧写入。合法空操作同样有独立操作记录，不依赖至少一个 artifact event 才能恢复。

普通 SQL 保存不等待 Redis；对应 journal 表示持久 pending，不承诺已经分发。Redis 交付在业务 SQL 提交后重新取得验证租约，通过 revision/nonce CAS 防止旧发布覆盖或删除新 owner 数据，最后才确认 ACK。全局锁序为 control → 全部有序 namespace → operation/events；ordinary 变更与投影仅 namespace，不倒序追加 control。

未知提交、旧 generation、较新失败 attempt、同 owner 不同 body、空目标恢复和已删除池均须单独覆盖。示例：gen1 空目标已经完成，gen2 只 reserve 后失败；重建仍须定位 gen1 的 committed/projected 记录，必要时用新 gen3恢复，不降 counter、不假 ACK gen2。

## SQL 键不等于 Redis 操作权限

普通 SQL 允许的 `pool:7:ips` 若直接套入主配置 Redis 模板，会碰到成员 SET。45字符的自定义键、type5 retirement intent 也不能被自动视为配置正文。事件明确区分 generated、lineage_only、not_applicable：只有验证过的 generated binding 或可信谱系可授权投影，不能接受自由物理 Redis key。

后续 binding 按固定逻辑角色派生物理 key；Lua 必须在任何写入前完成全部 KEY 的角色、TYPE 与 fence 检查。Redis Lua 原子执行不代表运行错误会回滚先前写入。raw YAML 的值类型与字节约定保留，不以没有注册的读取服务代替储存侧防覆盖。

## 迁移与权限

v3 只创建上述三表，已发布 v1/v2 的 SQL、执行包装及校验和保持原样。现代 runtime 和时区初始化要求完整 CurrentManifest；已有 v2 库需先停止写入者，以 `migrate --confirm-pending-migrations --confirm-no-writers` 显式追加。旧来源采用仍只验证/登记冻结 v1，再显式迁到当前版本。

当前表目录为冻结58表、namespace guard与三张 journal 表；legacy 采用检查仍针对冻结58表。runtime对保留表仅有SELECT/INSERT/UPDATE，没有DELETE或DDL。migrator获得必要表级DDL，不获得任意业务数据修复权限。此次没有上线或连接生产。

## 本阶段验收

五包357叶race、vet和标签编译通过；最终全Go测试/vet/全部标签编译通过。MySQL8.4.11/RR的专属空库实际运行4个子场景：fresh+clock/存储约束、v2追加v3保留原数据/guard/校验和且journal为空、dirty拒绝、drift拒绝。验证JSON NULL与[]、nonce/generation唯一性、字节键及旧last指针不被失败新attempt覆盖；临时库/账号/DSN已清。

完整`go test -race -tags=integration ./internal/dbschema`与`./cmd/schema`两组自有Docker原生测试通过，包含冻结旧库、当前模型结构、现代CLI、角色正反权限与时区流程；没有仅跑新例而漏掉旧fixture版本目标。

Web来源锁19ac2716481c44d145193fe24a6e97095e49b97f自身远端Verify/gitleaks成功，AI维持ba1876a；后端本提交的新组合CI另验。本地预览仍运行已验收0618及schema v2，本次开发未对演示库应用v3。

## 日志事务接口验收

`ProjectionJournalDAO` 独立于旧元数据接口：reserve 以永久 control 行锁分配 generation 和追加操作；commit 在 control → namespace → 数据行的顺序下，将业务配置、事件、manifest 和 lastCommitted 同事务提交。业务 SQL 使用私有事务原语，不嵌套调用自行提交的旧 public DAO。创建只在已知 commit 成功后回填输入 ID/时间；未知结果返回固定不确定错误，通过 nonce 重新加载 SQL 事实。

普通单项、批量、迁入/迁出及旧 generated 写口同事务追加 lineage_only 或 not_applicable 事件，不调用 Redis。只有验证过的 generated binding 或同字节 namespace 的可信谱系可授权投影；不穿过最新 N/A 或断链去找更旧权限。`pool:7:ips`、45字符自定义键和 type5 intent 保存成功不等于有 Redis artifact 权限。

已提交 manifest 及事件 payload 不允许被重试改写；MySQL JSON 的字段排序/空格按严格字段语义解码。交付标记从已签发 manifest 核对完整回执与 SourceID、latest revision 和 lastCommitted，不用调用方字段重新“签发”回执。空操作有独立 [] manifest，失败后继 attempt 不覆盖 last-good；必要时新 generation 恢复旧已提交事实。

新增独立 12 叶合同 race 通过。审查中实际复现并修复错误 SourceID 可标 applied，以及 create -1/3 状态可获得 generated receipt 两个缺口，原红例均转绿。另有非空 lineage/条件标记/返回值深拷贝、真实 SQLite commit 成功后故意丢 ACK 的 reserve/business 两分支及 nonce 重载测试通过。

MySQL8.4.11/RR 两独立 session 实际运行6子例、零skip：并发 generation/nonce、非空 JSON receipt/ordinary lineage、empty last-good/new recovery、后段 SQL CHECK 错误整批回滚、commit ACK 丢失独立 session 重载、unsigned counter 耗尽。旧 owner CAS 的 changed/found rows 两模式和7个 namespace 原生回归再次通过。完整 config DAO/cache race、vet、全 Go 与全部标签编译通过；所有专属 schema/user/DSN 已清理。

此提交没有连接发布器、Lua 或 worker。SQL mark API 由内部调用者确认外部投影后使用，测试中调用它不证明 Redis 已写；已复现的旧 A 删除/覆盖新 B 两个窗口仍待后续 fencing 业务闭环。普通 SQL 成功只表示保存与持久 pending，不等于分发成功。Web锁为独立 Verify 已通过的 dd558ce，AIba187；新后端组合CI另验。本地预览仍固定0618/schema2，没有把本次WIP直接应用演示库。


## 完整投影计划与 SQL 租约

`ProjectionPlanJournalDAO` 在同一业务事务中持久化 v2 envelope：artifact 事件向量、完整成员集合、Alertmanager 退役 intent 与 ACK 所需控制摘要。full family 1–4 必须包含全量 members 和 full ACK；Alertmanager 另需 intent 控制。family 5 只允许 webhook subset ACK，不能改写完整 Alertmanager 的成员集合或历史 SHA。合法空发布仍持久化空成员集合及控制证明，不能只留下空 artifact 数组。

旧 v1 数组（包括 `[]`）保持历史可读，不能凭版本升级制造完整控制权限；恢复完整投影需签发新的 v2 操作。旧 artifact-only pool marker 拒绝 v2。重试必须匹配原计划摘要与已签发 SourceID/EventID，不能改写原 manifest。损坏的持久 intent SourceID 即使重算本地摘要，也必须因缺少当前 artifact 向量中的精确 EventID 映射而拒绝。

`WithPoolProjectionLease` 在第二个独立 SQL 事务中按 control → 全部有序 namespace → operation/events 取得锁，callback 内只从该连接重建并复核材料。它不推迟或重做已经成功的业务提交。普通配置事件仅取得 namespace 租约，可 invalidate/retire，不能获得渲染正文、成员控制或 pool ACK。跨普通 N/A 接管断开的历史 binding 不能恢复权限，intent 保留确切旧 owner 快照，不从摘要反推名称或猜测新建 ID。

租约快照是深拷贝，句柄不能在 callback 返回、取消或检测到连接失效后继续使用。检测失败后不会另借连接“续租”。完成标记核对完整 artifact/control 向量；重复相同确认保留原 projected 时间和指针。业务提交确认丢失使用 nonce 读取持久事实，未知结果与已回滚失败明确区分。

SQL 租约不证明 Redis 已投影。本阶段仍未连接发布器、Lua 或恢复 worker；外部写前还需 fenced store 与原租约重验，已复现的两个跨 SQL/Redis 窗口尚未据此消除。

本机演示已在完整备份、停写及显式 pending 迁移后更新到干净 5feb4cc/schema v4；业务数据逐字节保留，临时迁移账号与 DSN 已清理。该更新只发生在本机，演示外部执行及 Cron 保持关闭，没有部署生产。


本阶段最终验收：完整 Go 测试、vet、integration/e2e/smoke 标签编译通过；完整 config DAO 132 叶 race、cache race 与两包 vet 通过，包含真实提交成功后丢 ACK 的 v2 nonce 重载。独立合同与持久 proof 反例通过。专属 MySQL8.4.11/RR 五子零跳过通过；旧错误封闭行为 overlay 在真实 KILL 后实际失败，旧重复 ACK 实际改写 projected 时间，最终同例转绿。原生测试仅使用新空库的专用账号/会话，未终止演示连接；临时库、账号和 DSN 已清理。
