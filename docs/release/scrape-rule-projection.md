# AlertRule 与 Record 的共享发布边界

共享实现只服务 AlertRule（family3）与 Record（family4），两个外层发布器保留各自源读取、渲染和目标生成。公共流程接收已经验证的逐实例字节，统一处理源版本、元数据归属、历史前置条件、已签计划重放与受围栏保护的投影。

## 实际共用的职责

- source 版本必须先于池、生命周期或规则读取；新 nonce 在同一永久 control 锁内比较 expected counter 后才能分配版本。旧源不能晚预留更高 generation 反向覆盖新完成状态。
- 元数据计划完整读取全部目标和旧归属。健康旧回执仅覆盖它已签发的目标；同 hash 下新增的 database-only stale 也必须进入新计划。
- 退役权限来自当前 canonical live owner 或具体历史证明。已删除 foreign/custom 行是必须完整观察和比较的 ABA 前置数据，不能阻止当前 owner 清退自己，也不能授予清理 foreign 当前正文的权限。
- accepted render/retire 或 legacy SQL owner 被另一完整 generated 投影替代时，只恢复旧池自己的 empty bookkeeping。替代方的 raw、成员和 namespace fence 不加入旧池操作。
- metadata、artifact events、manifest 与 lastCommitted 同事务提交；交付在第二个原 SQL 租约中完成，只有完整 Store ACK 才能标 projected。SQL/Redis 不确定结果保留持久事实，不能假称回滚或新完成。

这些算法各有一份 scoped 实现，没有为 AlertRule 复制 Record 的状态机。scope 只接受3/4，物理名称及 key 使用固定映射；不存在通用字符串 key/callback 授权入口，也未提前泛化到 Prometheus/AM。

## 两种发布格式与渲染规则

Record 已发布 public API、DTO、plan digest、manifest 字段及 `record_superseded` 由固定family4外层保留。AlertRule 使用明确的family3接口和独立可选 `rule_superseded`；错误 family 或混用 side proof 必须拒绝。旧无 side proof 的 v1/v2 仍可读。

AlertRule 的 BuildAlertingRule、UTF8 名称、for0s、label/annotation首个等号、稳定数据库 ID group 与按原 PrometheusInstances 顺序分片不进入共享重构。字段值、规则顺序和逐实例 YAML 仍由原 renderer 决定。

## 验证证据与限制

Record 的五组兼容金样由干净、已发布的 `db65def2ed37bca5956065d36a1d2de4bdf21134` 实际 SQL 实现生成，再与当前共享实现比较；包含空计划、render、含已删除 foreign 历史的 live 退役、accepted disposition 和 legacy disposition。比较原 manifest、receipt 表示和计划摘要，不用当前实现生成当前 expected 自证兼容。

真正原生测试对两个family复用同一固定scope场景，而非复制另一份断言：真实 TCP 截留旧EVAL并先向调用方返回EOF，B完成后才转发旧包，必须收到实际Redis STALE且B全key/SQL不变；还验证legacy接纳/B后续off、同hash新stale以及真实pool/rule DAO读取屏障。另在同一SQLite与同一Redis中启停两个family，核对彼此正文、成员、SHA、ACK、fence与SQL owner不跨界。

本阶段完整本机回归已经完成；提交后的远端组合另验。Prometheus/AM 发布器、独立恢复worker和规模性能另行完成，不据此宣称整个1.0就绪或跨SQL/Redis原子可见。本机预览保持已验收的clean5feb/schema4、模拟数据与关闭外部执行，没有上线。


## 最终本机验收

完整 Go 测试、vet、integration/e2e/smoke 标签编译、173 项发布工具回归与 actionlint 通过。完整 cache 的209个测试入口、1095个叶回归以及 alert service 的race通过，零失败/跳过；Alert新增31项合同通过，原Record抽取的9个入口、23项断言完整运行，原Alert renderer文件与db65字节相同。

owned Redis 最终27叶通过、零失败/跳过：原Record20不减，Alert6项真正发布器场景与同一SQL/Redis中两family隔离1项加入同一入口。原生实例全部按label和完整ID核验后清理，最终容器为零，没有使用演示Redis。

跨版本五金样不是本轮新实现重生成：先在独立干净db65 checkout以真实SQL DAO生成，再在共享核心下验证原manifest、receipt和planDigest。金样与生成说明已作为永久测试保留。

来源锁更新至独立远端Verify41857398777/gitleaks41857398891已成功的Web955cbf7，AI仍ba187；本次后端新组合CI另验。尚未把候选代码刷新入本机预览，也没有上线或连接生产。
