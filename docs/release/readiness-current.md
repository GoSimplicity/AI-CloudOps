# 当前开发完成度与验收边界

审计快照：2026-10-30。对照三仓外的《AI-CloudOps 三仓现状与后续工作清单》（2026-10-29），逐项核对当前源码、测试与合同。本文件不是自动更新的 CI 看板，也不以历史“1.0 已验收”替代当前版本验收。

| 仓库 | 本次核对的已提交基线 |
| --- | --- |
| 后端 | `799e3ef8792016473fe63bf5f0e365522c428195` |
| Web | `fae499555ee36d13eba6f6f2f317bdb908589c29` |
| AIOps | `cb0d6bee89ca42906dc883cb9476eee1bfc22705` |

`integration-sources.json` 已固定上述 Web/AIOps SHA，后端由实际 job checkout 决定。本次核对时，[固定组合手动集成验收 41042508609](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41042508609) 现已通过（后端799e3ef8、Webfae4995、AIOpscb0d6be这一固定组合）；新发布候选清单及固定 Git archive/镜像组合仍待完成。AIOps 工作树中的知识目录快照、总量预算及异步读取优化尚在开发，未计入上述提交的完成度。

状态含义：**已实现**表示原清单明确范围内已有实现与回归覆盖；**部分**表示该项仍有具体交付工作；**未验**描述真实环境或最新组合的验收状态。代码完成、隔离验证、外部验收是三个维度，不相互替代。本次为只读源码/测试核对，没有重跑三仓全套测试，也没有调用真实模型、通知、云、Kubernetes、SSH 或既有服务；下表中的隔离证据来自已记录的阶段验证及可定位的测试，不能自动推导最新 CI 全绿。

## F01–F14

路径前缀 `Go` 为后端仓库，`Web` 为前端仓库，`AI` 为 AIOps 仓库。

| 项目 | 开发状态 | 当前行为与源码依据 | 隔离回归依据 | 未验或保留边界 |
| --- | --- | --- | --- | --- |
| F01 周期任务归档后阻塞 | 已实现 | Go `internal/cron/scheduler/cron_scheduler.go` 为调度执行分配实例 ID，业务互斥由任务租约承担。 | `internal/cron/scheduler/periodic_redis_integration_test.go` 的 `TestPeriodicTaskContinuesAfterArchive`；CI 有真实隔离 Redis 步骤。 | 不承诺外部副作用 exactly-once；既有已入队旧任务不被伪装为自动安全回收。 |
| F02 工单事件丢失、删除后无快照 | 已实现 | Go `internal/workorder/dao/notification_events.go`、`service/notification_event_worker.go`；业务事务保存事件和实例快照，worker 按租约及逐通道结果交付。 | `notification_event_lifecycle_test.go`、`notification_event_mysql_integration_test.go`、`notification_event_transport_test.go`、DI 异步 Redis fixture。 | 未在当前版本对真实邮件/飞书实投；超时/未知投递不能盲目重放，备份恢复后需核对外部副作用。 |
| F03 重启追加后旧知识失召回 | 已实现 | AI `app/core/vector/redis_vector_store.py` 的语义读取以持久向量为准，进程内索引映射互相隔离，并绑定 generation/revision。 | `tests/unit/test_rag_ingestion.py` 覆盖旧 A→重启→追加 B 仍召回 A、实例映射隔离、不完整扫描失败；`scripts/manual_checks/rag_refresh_redis.py`。 | 有明确查询数量/时间边界；完整扫描不是分布式共享索引性能保证。当前另做目录预算优化，不把未提交代码算已完成。 |
| F04 刷新先清旧索引 | 已实现 | AI `enterprise_assistant.py`、`redis_vector_store.py` 先构建新 generation，再以 CAS 切换活动版本，失败保留旧版本。 | `test_rag_refresh_atomic.py` 覆盖缺目录、嵌入失败、取消、并发修订；`test_generation_gc.py` 与隔离 Redis 刷新脚本覆盖生命周期。 | 发布回执丢失/发布期间取消是 unknown，必须核对；旧固定前缀及备份不等于已物理擦除。 |
| F05 内置 Cron 配置不驱动循环 | 已实现 | Go `internal/cron/manager.go`、`builtin.go` 统一单次执行与调度登记、重载、取消，执行前重读状态。 | `internal/cron/builtin_runtime_test.go`、`internal/cron/dao/builtin_pagination_test.go`。 | 配置约 30 秒内重载；历史表达式开始真正生效，升级前须审核期望频率；未连真实集群。 |
| F06 首次读库失败不恢复 | 已实现 | Go `scheduler/cron_scheduler.go` 退避重试并暴露 ready/连续失败状态；内置调度同样参与 readiness。 | `internal/cron/scheduler/recovery_test.go` 覆盖多次失败后恢复和取消；`internal/app/health_smoke_test.go` 的路由回归。 | `/health` 只证明存活；调度未就绪时 `/ready` 返回失败，不以 HTTP 存活替代调度能力。 |
| F07 Cron 保存及编辑校验缺口 | 已实现 | Go `internal/cron/service/cron_service.go` 保存/启用前校验；Web `Management.ts` 使用当前表单表达式，保留原有效调度。 | Go `service/schedule_test.go`、`schedule_smoke_test.go`；Web `Management.test.ts`、`Management.fullchain.test.ts`。 | 服务端验证是最终门禁，不能因前端校验通过绕过服务端。 |
| F08 HTTP 固定 30 秒截断 | 已实现 | Go `internal/cron/handler/executors.go` 由任务 context 决定总期限，Transport 保留阶段限制。 | `internal/cron/handler/http_timeout_test.go` 覆盖 120 秒 deadline 传递、取消和错误保密。 | 测试核对期限及取消行为，没有用等待 120 秒冒充真实外部请求验收。 |
| F09 RCA 采集失败显示完成 | 已实现 | AI `app/core/rca/rca_engine.py`、`services/rca_service.py`、`api/routes/rca.py` 传递数据质量与采集状态；Web RCA 页面区分失败/部分/空数据。 | `test_rca_collection_contract.py` 的实际 collector→ASGI 链路，`test_rca_metadata.py`；Web `RCADiagnosis.test.ts`、`RcaSafety.test.ts`。 | 覆盖率未知保持未知；没有真实监控/集群来源验收。 |
| F10 精度与置信度无依据 | 已实现 | AI 预测器返回 method/model_version/fallback_reason，缺证据时精度为未知；`app/core/prediction/backtest.py` 与回测 API/Web 入口提供滚动误差和校准。 | `test_prediction_provenance_and_report.py`、`test_prediction_rolling_backtest.py`；Web `PredictionProvenance.test.ts`、`PredictionBacktest.test.ts`。 | 回测输入是用户提供的预测/观测记录，不证明训练集隔离或生产模型质量；未自动虚构 accuracy、模型权重校准或置信度。 |
| F11 报告与扩缩容/成本不一致 | 已实现 | AI `app/services/prediction_service.py` 在单维/多维预测完成异常、扩缩容、成本补强后再生成报告。 | `test_prediction_provenance_and_report.py` 覆盖报告收到最终建议/成本、跨维顺序、补强失败来源。 | 成本是估算；报告有界、归属和有效期不代表真实账单或扩缩容动作已执行。 |
| F12 Cron 编辑缺执行字段 | 已实现 | Web `views/cron/management/Management.vue`、`Management.ts` 按原任务类型编辑 HTTP、SSH、命令/脚本实际配置。 | `CronEditFields.test.ts`、`Management.edit.ui.test.ts`、`Management.localExec.test.ts`。 | 仍遵守本地执行默认关闭及原任务类型约束；不自动开放任意类型切换或真实 SSH。 |
| F13 列表/详情可用动作不同 | 已实现 | Web `views/workorder/Instance.vue` 菜单打开时取 `getAvailableActions`，列表和详情消费同一服务端可用动作口径。 | `Instance.test.ts`；既有工单真实 UI 生命周期用例已接入三仓 CI，上述799e3ef8固定组合已通过，但不外推之后改动。 | 加载失败不展示推测动作；服务端鉴权仍是最终执行边界。 |
| F14 AWS 枚举文档错误 | 已实现 | Go `internal/model/tree_cloud.go` 与 Web `api/core/tree/tree_account.ts` 保持 AWS=3、华为=4，`docs/multicloud-compute.md` 已修正。 | `scripts/integration/cloud_provider_contract.go`、`cloud_provider_contract_test.go` 及集成 CI 跨仓检查。 | 不能从枚举存在推导某厂商全部产品都已实现。 |

## D01–D05 的明确范围

原清单使用了“按实际需求”“按需定义”，没有要求六厂商的全部产品、所有凭据来源和所有同步模式。下表按当前明确交付边界核对；新增产品扩展不得伪装成已经请求的缺陷修复。

| 项目 | 开发状态 | 已有闭环及依据 | 隔离验证 | 未验/未请求扩展 |
| --- | --- | --- | --- | --- |
| D01 AWS/腾讯管理台 | 已实现 | Go 三家计算客户端及账户/地域/同步服务；Web `CloudAccountManagement.vue`、`CloudResourceManager.vue` 和 `cloudaccountmanagement/provider.ts` 已允许阿里/AWS/腾讯端到端操作。 | Go SDK 内存 HTTP、账户/地域 MySQL 事务回归；Web `CloudAccount.contract.test.ts`、`CloudResource.contract.test.ts` 与 E2E 合同用例。 | 真实账号和授权覆盖未验；本轮 CI 两个浏览器门禁不包括真实云。 |
| D02 高级通知与消息中心 | 已实现 | Go `notification_schedule_test.go` 对应的有界 delayed/scheduled/conditional/重复调度、取消、事件历史与人工重放；`dao/notification_inbox.go`、API 与 Web `layouts/basic.vue`/工单通知页接站内已读未读和历史。 | 调度、取消、重放、逐通道及并发站内事务有单元/SQLite/隔离 MySQL 回归；Web `NotificationEvents.test.ts`、`Notification.test.ts`、`notification-schedule.test.ts`。 | 条件采用明确支持字段，不执行任意表达式；未知投递不能自动重试。站内删除保留去重墓碑，外部真实投递和保留政策仍须业务验收。 |
| D03 持久会话/审批与跨进程 | 已实现（单权威 Redis 范围） | AI `assistant_session_redis.py`、`workflow_approval_redis.py` 默认用于服务，Lua 原子领取、owner 绑定、结果查询/历史和崩溃 unknown；Web `ai.vue` 会话恢复及 autofix 恢复页已接。RAG generation 消除共享位置映射错误。 | `tests/integration/test_assistant_session_redis.py`、`scripts/manual_checks/autofix_redis.py`、审批隔离测试；Web `ai-session-recovery.test.ts`、`useAutoFixRecovery.test.ts`。 | Redis Cluster、异步切换或恢复旧备份不保证零丢失；仍需部署副本/容量验收。没有把有限扫描称作分布式索引性能，也不提供外部动作 exactly-once。 |
| D04 新厂商及临时凭据 | 已实现（显式凭据与受限计算同步） | `utils/cloud_client.go` 工厂已接六家；Azure 服务主体/GCP service_account 整组加密、华为逐地域 project_id；AWS/Tencent 显式临时令牌与 UTC 到期检查、轮换 CAS。Web 账户表单和指定 ID 同步入口均已接。 | `cloud_account_provider_http_test.go`、provider/temporary SQLite 与 MySQL 测试；`TestTemporaryFullSyncAtomicAndExpiredCredentials`、`TestMySQLTemporaryFullSyncAtomic`；三家新增 SDK 分页/身份/取消替身及 Web 账户合同。 | 华为/Azure/GCP 只允许明确地域与指定实例 ID 增量，拒绝无法证明完整性的全量/无 ID 同步。AWS/Tencent 临时凭据已支持有地域事务边界的 full；不承诺跨地域原子成功。自动 AssumeRole、令牌自动续期及其他身份链未请求扩展，真实账号仍未验。 |
| D05 非计算资源 | 已实现（首批阿里云三类型） | `utils/aliyun_managed_resources.go` 独立 RDS/传统 CLB/缓存 DTO；`tree_managed_resource_service.go`、独立 DAO/表/API 与 Web `CloudManagedResources.vue` 完成指定 ID 增量、列表/详情/历史。持久 operation_id、范围门禁、按 owner 查询及人工核对恢复已接。 | managed SDK、handler/RBAC、租约/fencing/事务/MySQL clientFoundRows 回归；Web managed 资源、操作恢复、跨 owner 列表组件测试与隔离模拟浏览器。 | 当前不删除未列出的资源；真实云未验。其他厂商托管产品及 ALB/NLB 是尚未定义的扩展，不自动列为必做。旧无 ID pending 不伪造关联，已删除范围不在恢复接口范围；保留治理另需明确政策。 |

D04/D05 的“已实现”不意味着 SDK 拥有全账号权限，也不允许用不完整清单删除资源。账户凭据验证、地域读取、指定 ID 同步、全量完整性证明分别有边界；应保留当前 fail-closed 限制。

## 五项工程交付

| 原清单项 | 状态 | 已交付证据 | 仍需完成或单独验收 |
| --- | --- | --- | --- |
| E01 固定最新三仓发布组合 | 部分 | `integration-sources.json`、`scripts/integration/locked_sources.py` 固定三仓来源；`release-fresh.py` 持久私有 env/Compose 快照、唯一构建标签/来源标识、全部 image ID 固定，已有原配置编辑和 tag 替换隔离回归。 | 当前组合手动 CI 未通过前不能算验收；固定 Git archive、持久 bind/构建来源及新候选清单仍待固化。`release-candidate.json` 是 10-23 历史快照，不能覆盖冒充当前版本。 |
| E02 业务浏览器进入三仓 CI | 部分（两场景已接）；最新组合未验 | `scripts/integration/business_browser.py` 默认拒绝已有服务，仅 CI 自有栈；工单真实 UI 草稿→提交→审批→完成与 Cron 读取/取消两例，强制 2 pass、零 skip/retry/失败，诊断白名单脱敏。 | 手动 run 41042508609 进行中；两个用例不等于审批、会话过期、多云、Cron 全部操作都已经浏览器验收，更不包含真实外部依赖。 |
| E03 全量可恢复备份 | 已实现（隔离演练）；生产指标未验 | `scripts/recovery_backup.py`、安全单测及 `scripts/integration/recovery_backup_fixture.py` 覆盖干净停写、MySQL/Redis 全目录和多 DB、AIOps 知识文件、私有配置、HMAC/sha256、全新隔离卷恢复。 | 已记录恢复数据/状态核对；private_config 应包含实际生效的 fresh 私有快照，不能在原 `.env` 被改后仍以它替代真实配置。真实部署 RPO 取决于最后有效快照与业务写入，RTO 包含应用及外部副作用核查，尚未约定/实测生产 SLA。不能用复制耗时冒充业务恢复耗时。 |
| E04 版本化迁移与分权 | 已实现（新库+严格当前旧库采用）；部署未验 | `pkg/schema` 迁移锁/意图/校验和/只读 Verify，`internal/dbschema` 冻结 58 表基线、clock 与严格旧库预检；`cmd/schema` 显式 CLI、逐表 GRANT；DI startup 只读核验，fresh 移除镜像默认 ALL，真实自有 MySQL 角色拒绝路径已验。 | 不支持未知历史结构自动认领；部分 DDL/脏 ledger 要人工向前修复，不声明 MySQL DDL可事务回滚。实际旧部署未访问，不因离线 GRANT 计划就声称现有账号已收权。 |
| E05 可移植性与技术债 | 部分 | AF_UNIX 测试已用自有短目录（`test_sec33_l3_file_boundary.py`）；Pydantic 改 `ConfigDict`；19 页 Table.slots 改 bodyCell、157 分支 AST 对照及组件回归；Unicorn 过期注释规则保留且兼容；AlertManagerConfigCache 已按 typed cfg 收敛。 | 其余 viper 需按实际可达模块继续收敛，不能无依据全仓替换。全仓 ESLint 曾记录 34645 errors/439 warnings 的既有规范债务，仅配置包检查通过，不得宣称全仓 lint 绿。RAG 目录预算/非阻塞读取仍在开发。 |

## 当前结论与文档口径

F01–F14 的原缺陷均有实现及回归；D01–D05 已按上表明确范围交付，实际外部验收仍未执行。当前不能宣布整个根目标 100% 完成：固定组合 CI/候选、正在开发的 RAG 目录治理及工程余项仍须继续记录和验证。

`cloud-managed-resources.md`、`huawei-client.md`、`azure-client.md`、`gcp-client.md`、`cloud-provider-credentials.md`、`cloud-temporary-credentials.md`、`notification-trigger-boundary.md` 及 `docs/multicloud-compute.md` 含有分阶段历史证据。本次已更新当前入口说明，以及被后续实现推翻的“工厂/前端尚未接入”“临时凭据 full 尚未开放”“只支持 immediate”等结论；历史测试范围保留，但不能覆盖后续交付事实。

真实模型、通知、云/集群、生产备份恢复与上线均需要各自授权及目标配置。没有实际执行的验收保持“未验”，不以代码完成、健康检查、mock、单个绿灯或历史验收替代。
