# 配置发布的持久日志与投影版本

当前阶段只交付 v3 存储结构、模型和权限；事务日志业务接口、Redis fencing、发布器及恢复 worker 尚未接入。已有的两处 SQL 提交后 Redis 竞争仍属于后续验收，不能用建表成功声称发布窗口已经关闭。

## 三种记录的责任

| 表 | 责任 | 保留要求 |
| --- | --- | --- |
| `cl_monitor_config_pool_controls` | 每个发布域/池的 generation counter、当前 attempt、最后 committed 和最后 projected 指针 | 永久锁地址；reserve 不覆盖两个 last 指针 |
| `cl_monitor_config_pool_operations` | 逐代操作、nonce、source hash、manifest、恢复来源与阶段 | append 历史；失败新 attempt 不抹掉旧空操作 |
| `cl_monitor_config_artifact_events` | namespace 单调事件版本、行/owner/body digest、投影授权与交付状态 | 保留退休与 A→B→C 证明；当前阶段无自动 GC |

这三张表不嵌入普通业务 Model，没有软删除或 YAML/凭据正文副本。`manifest` 的 SQL NULL 表示尚未提交，JSON `[]` 表示合法的空 manifest。operation nonce 使用16个二进制字节，digest使用32字节；namespace原始键使用 VARBINARY(180)，保留45个UTF-8字符的字节身份。

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
