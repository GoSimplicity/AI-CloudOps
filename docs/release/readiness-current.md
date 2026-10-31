# 当前开发完成度与验收边界

审计快照：2026-10-30。对照三仓外的《AI-CloudOps 三仓现状与后续工作清单》（2026-10-29），逐项核对当前源码、测试与合同。本文件不是自动更新的 CI 看板，也不以历史“1.0 已验收”替代当前版本验收。

| 仓库 | 本次核对的已提交基线 |
| --- | --- |
| 后端 | `eec9582a0aa1eda3718f7924811c119ef197bcd8` |
| Web | `d0f098e91b26eda8055c5f49cda70530fd32f8fe` |
| AIOps | `c2de23e724168b1d8aab01ea9f0d0e604a3a0306` |

`integration-sources.json` 固定 Web/AIOps SHA，后端由实际 job checkout 决定。[五场景手动集成 41067158747](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41067158747) 已通过，实际后端为 `f195db9fcff5f5af5d512386f3717d814c34e344`；它不能替代后续 eec9582a 的验收。[最终组合 41078544655](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41078544655) 已全部通过，日志确认三仓完整 SHA 与五场景零 skip/retry。它与本地固定源码候选的构建收据分别记录，历史 `release-candidate.json` 不覆盖。

状态含义：**已实现**表示原清单明确范围已有实现及回归；隔离验证与外部验收分别记录，不相互替代。本轮最终离线验证：Go `go test ./...`、`go vet ./...` 通过，脚本 156 项通过；Web 133 文件、2051 项单测、类型检查与生产构建通过；AIOps 2310 通过、33 跳过、3 条既有 SWIG 告警，Ruff 通过。跳过的外部依赖用例不算已验。另有真实隔离 MySQL/Redis、迁移/回退、race 和浏览器回归，按各专项证据判断范围。没有重测真实模型、通知、云、Kubernetes、SSH 或生产服务。

## F01–F14

路径前缀 `Go` 为后端仓库，`Web` 为前端仓库，`AI` 为 AIOps 仓库。

| 项目 | 开发状态 | 当前行为与源码依据 | 隔离回归依据 | 未验或保留边界 |
| --- | --- | --- | --- | --- |
| F01 周期任务归档后阻塞 | 已实现 | Go `internal/cron/scheduler/cron_scheduler.go` 为调度执行分配实例 ID，业务互斥由任务租约承担。 | `internal/cron/scheduler/periodic_redis_integration_test.go` 的 `TestPeriodicTaskContinuesAfterArchive`；CI 有真实隔离 Redis 步骤。 | 不承诺外部副作用 exactly-once；既有已入队旧任务不被伪装为自动安全回收。 |
| F02 工单事件丢失、删除后无快照 | 已实现 | Go `internal/workorder/dao/notification_events.go`、`service/notification_event_worker.go`；业务事务保存事件和实例快照，worker 按租约及逐通道结果交付。 | `notification_event_lifecycle_test.go`、`notification_event_mysql_integration_test.go`、`notification_event_transport_test.go`、DI 异步 Redis fixture。 | 未在当前版本对真实邮件/飞书实投；超时/未知投递不能盲目重放，备份恢复后需核对外部副作用。 |
| F03 重启追加后旧知识失召回 | 已实现 | AI `app/core/vector/redis_vector_store.py` 的语义读取以持久向量为准，进程内索引映射互相隔离，并绑定 generation/revision。 | `tests/unit/test_rag_ingestion.py` 覆盖旧 A→重启→追加 B 仍召回 A、实例映射隔离、不完整扫描失败；`scripts/manual_checks/rag_refresh_redis.py`。 | 有明确查询数量/时间边界；完整扫描不是分布式共享索引性能保证。已加入目录/文件/读取预算和线程快照；超过边界明确失败。 |
| F04 刷新先清旧索引 | 已实现 | AI `enterprise_assistant.py`、`redis_vector_store.py` 先构建新 generation，再以 CAS 切换活动版本，失败保留旧版本。 | `test_rag_refresh_atomic.py` 覆盖缺目录、嵌入失败、取消、并发修订；`test_generation_gc.py` 与隔离 Redis 刷新脚本覆盖生命周期。 | 发布回执丢失/发布期间取消是 unknown，必须核对；旧固定前缀及备份不等于已物理擦除。 |
| F05 内置 Cron 配置不驱动循环 | 已实现 | Go `internal/cron/manager.go`、`builtin.go` 统一单次执行与调度登记、重载、取消，执行前重读状态。 | `internal/cron/builtin_runtime_test.go`、`internal/cron/dao/builtin_pagination_test.go`。 | 配置约 30 秒内重载；历史表达式开始真正生效，升级前须审核期望频率；未连真实集群。 |
| F06 首次读库失败不恢复 | 已实现 | Go `scheduler/cron_scheduler.go` 退避重试并暴露 ready/连续失败状态；内置调度同样参与 readiness。 | `internal/cron/scheduler/recovery_test.go` 覆盖多次失败后恢复和取消；`internal/app/health_smoke_test.go` 的路由回归。 | `/health` 只证明存活；调度未就绪时 `/ready` 返回失败，不以 HTTP 存活替代调度能力。 |
| F07 Cron 保存及编辑校验缺口 | 已实现 | Go `internal/cron/service/cron_service.go` 保存/启用前校验；Web `Management.ts` 使用当前表单表达式，保留原有效调度。 | Go `service/schedule_test.go`、`schedule_smoke_test.go`；Web `Management.test.ts`、`Management.fullchain.test.ts`。 | 服务端验证是最终门禁，不能因前端校验通过绕过服务端。 |
| F08 HTTP 固定 30 秒截断 | 已实现 | Go `internal/cron/handler/executors.go` 由任务 context 决定总期限，Transport 保留阶段限制。 | `internal/cron/handler/http_timeout_test.go` 覆盖 120 秒 deadline 传递、取消和错误保密。 | 测试核对期限及取消行为，没有用等待 120 秒冒充真实外部请求验收。 |
| F09 RCA 采集失败显示完成 | 已实现 | AI `app/core/rca/rca_engine.py`、`services/rca_service.py`、`api/routes/rca.py` 传递数据质量与采集状态；Web RCA 页面区分失败/部分/空数据。 | `test_rca_collection_contract.py` 的实际 collector→ASGI 链路，`test_rca_metadata.py`；Web `RCADiagnosis.test.ts`、`RcaSafety.test.ts`。 | 覆盖率未知保持未知；没有真实监控/集群来源验收。 |
| F10 精度与置信度无依据 | 已实现 | AI 预测器返回 method/model_version/fallback_reason，缺证据时精度为未知；`app/core/prediction/backtest.py` 与回测 API/Web 入口提供滚动误差和校准。 | `test_prediction_provenance_and_report.py`、`test_prediction_rolling_backtest.py`；Web `PredictionProvenance.test.ts`、`PredictionBacktest.test.ts`。 | 回测输入是用户提供的预测/观测记录，不证明训练集隔离或生产模型质量；未自动虚构 accuracy、模型权重校准或置信度。 |
| F11 报告与扩缩容/成本不一致 | 已实现 | AI `app/services/prediction_service.py` 在单维/多维预测完成异常、扩缩容、成本补强后再生成报告。 | `test_prediction_provenance_and_report.py` 覆盖报告收到最终建议/成本、跨维顺序、补强失败来源。 | 成本是估算；报告有界、归属和有效期不代表真实账单或扩缩容动作已执行。 |
| F12 Cron 编辑缺执行字段 | 已实现 | Web `views/cron/management/Management.vue`、`Management.ts` 按原任务类型编辑 HTTP、SSH、命令/脚本实际配置。 | `CronEditFields.test.ts`、`Management.edit.ui.test.ts`、`Management.localExec.test.ts`。 | 仍遵守本地执行默认关闭及原任务类型约束；不自动开放任意类型切换或真实 SSH。 |
| F13 列表/详情可用动作不同 | 已实现 | Web `views/workorder/Instance.vue` 菜单打开时取 `getAvailableActions`，列表和详情消费同一服务端可用动作口径。 | `Instance.test.ts`；既有工单真实 UI 生命周期用例已接入三仓 CI，固定 f195db9/d0f098e/c2de23e 五场景组合已通过，不外推之后改动。 | 加载失败不展示推测动作；服务端鉴权仍是最终执行边界。 |
| F14 AWS 枚举文档错误 | 已实现 | Go `internal/model/tree_cloud.go` 与 Web `api/core/tree/tree_account.ts` 保持 AWS=3、华为=4，`docs/multicloud-compute.md` 已修正。 | `scripts/integration/cloud_provider_contract.go`、`cloud_provider_contract_test.go` 及集成 CI 跨仓检查。 | 不能从枚举存在推导某厂商全部产品都已实现。 |

## D01–D05 的明确范围

原清单使用了“按实际需求”“按需定义”，没有要求六厂商的全部产品、所有凭据来源和所有同步模式。下表按当前明确交付边界核对；新增产品扩展不得伪装成已经请求的缺陷修复。

| 项目 | 开发状态 | 已有闭环及依据 | 隔离验证 | 未验/未请求扩展 |
| --- | --- | --- | --- | --- |
| D01 AWS/腾讯管理台 | 已实现 | Go 三家计算客户端及账户/地域/同步服务；Web `CloudAccountManagement.vue`、`CloudResourceManager.vue` 和 `cloudaccountmanagement/provider.ts` 已允许阿里/AWS/腾讯端到端操作。 | Go SDK 内存 HTTP、账户/地域 MySQL 事务回归；Web `CloudAccount.contract.test.ts`、`CloudResource.contract.test.ts` 与 E2E 合同用例。 | 真实账号和授权覆盖未验；五场景门禁包含三家停用账户 CRUD，但不包括真实云。 |
| D02 高级通知与消息中心 | 已实现 | Go `notification_schedule_test.go` 对应的有界 delayed/scheduled/conditional/重复调度、取消、事件历史与人工重放；`dao/notification_inbox.go`、API 与 Web `layouts/basic.vue`/工单通知页接站内已读未读和历史。 | 调度、取消、重放、逐通道及并发站内事务有单元/SQLite/隔离 MySQL 回归；Web `NotificationEvents.test.ts`、`Notification.test.ts`、`notification-schedule.test.ts`。 | 条件采用明确支持字段，不执行任意表达式；未知投递不能自动重试。站内删除保留去重墓碑，外部真实投递和保留政策仍须业务验收。 |
| D03 持久会话/审批与跨进程 | 已实现（单权威 Redis 范围） | AI `assistant_session_redis.py`、`workflow_approval_redis.py` 默认用于服务，Lua 原子领取、owner 绑定、结果查询/历史和崩溃 unknown；Web `ai.vue` 会话恢复及 autofix 恢复页已接。RAG generation 消除共享位置映射错误。 | `tests/integration/test_assistant_session_redis.py`、`scripts/manual_checks/autofix_redis.py`、审批隔离测试；Web `ai-session-recovery.test.ts`、`useAutoFixRecovery.test.ts`。 | Redis Cluster、异步切换或恢复旧备份不保证零丢失；仍需部署副本/容量验收。没有把有限扫描称作分布式索引性能，也不提供外部动作 exactly-once。 |
| D04 新厂商及临时凭据 | 已实现（显式凭据与受限计算同步） | `utils/cloud_client.go` 工厂已接六家；Azure 服务主体/GCP service_account 整组加密、华为逐地域 project_id；AWS/Tencent 显式临时令牌与 UTC 到期检查、轮换 CAS。Web 账户表单和指定 ID 同步入口均已接。 | `cloud_account_provider_http_test.go`、provider/temporary SQLite 与 MySQL 测试；`TestTemporaryFullSyncAtomicAndExpiredCredentials`、`TestMySQLTemporaryFullSyncAtomic`；三家新增 SDK 分页/身份/取消替身及 Web 账户合同。 | 华为/Azure/GCP 只允许明确地域与指定实例 ID 增量，拒绝无法证明完整性的全量/无 ID 同步。AWS/Tencent 临时凭据已支持有地域事务边界的 full；不承诺跨地域原子成功。自动 AssumeRole、令牌自动续期及其他身份链未请求扩展，真实账号仍未验。 |
| D05 非计算资源 | 已实现（首批阿里云三类型） | `utils/aliyun_managed_resources.go` 独立 RDS/传统 CLB/缓存 DTO；`tree_managed_resource_service.go`、独立 DAO/表/API 与 Web `CloudManagedResources.vue` 完成指定 ID 增量、列表/详情/历史。持久 operation_id、范围门禁、按 owner 查询及人工核对恢复已接。 | managed SDK、handler/RBAC、租约/fencing/事务/MySQL clientFoundRows 回归；Web managed 资源、操作恢复、跨 owner 列表组件测试与隔离模拟浏览器。 | 当前不删除未列出的资源；真实云未验。其他厂商托管产品及 ALB/NLB 是尚未定义的扩展，不自动列为必做。旧无 ID pending 不伪造关联，已删除范围不在恢复接口范围；保留治理另需明确政策。 |

D04/D05 的“已实现”不意味着 SDK 拥有全账号权限，也不允许用不完整清单删除资源。账户凭据验证、地域读取、指定 ID 同步、全量完整性证明分别有边界；应保留当前 fail-closed 限制。

## 五项工程交付

| 原清单项 | 状态 | 已交付证据 | 仍需完成或单独验收 |
| --- | --- | --- | --- |
| E01 固定最新三仓发布组合 | 已实现；固定源码构建与本地验收通过 | 来源锁及 `release-candidate.py` 钉住干净全 SHA，以隔离 Git archive 记录每个 blob；fresh 快照私有配置、唯一标签及全部 image ID，拒绝外部覆盖。 | eec9582a/d0f098e/c2de23e 归档源码已实际构建，新隔离栈 schema/clock/分权校验、9服务健康及精确五例零 skip/retry 通过。镜像导出与校验清单见[本轮候选收据](candidates/2026-10-30-eec9582a/review-manifest.json)；不覆盖历史 `release-candidate.json`。 |
| E02 业务浏览器进入三仓 CI | 已实现且最终固定组合通过 | CI 自有栈包含工单生命周期、Cron 重试读取/取消、过期登录、三云停用账户 CRUD、停用 HTTP Cron 创建/编辑五场景；要求恰好 5 pass、零 skip/retry，诊断仅白名单位置/状态。固定 f195db9/d0f098e/c2de23e 已通过。 | eec9582a/d0f098e/c2de23e 最终 run 41078544655 五项通过，零 skip/retry；没有真实云、真实通知或生产业务写入。 |
| E03 全量可恢复备份 | 已实现（隔离演练）；生产指标未验 | `recovery_backup.py` 覆盖停写、MySQL/Redis 全目录、AIOps 知识文件、实际私有配置、HMAC/sha256及新隔离卷恢复。`known_upgrade_rollback.py` 真实执行旧49表→58表→新卷恢复旧49表，12行含软删除/NULL/合成密文指纹一致，HMAC独立核验通过。 | 物理复制观测 2.472 秒，不是整站 RTO；未启动旧应用/worker，也未验证生产外部副作用。生产 RPO/RTO 和凭据权限由部署方验收。 |
| E04 版本化迁移与分权 | 已实现（新库、严格采用、已知历史升级） | 冻结基线、迁移锁/意图/校验和、只读 Verify/clock、显式 CLI 与角色分权。a8f1700 的49表可停写备份后显式升级；safe-v2 在任何 ledger/DDL 前证明 TRIGGER 元数据可见且零触发器，隐藏/可见触发器均真实 MySQL 拒绝。 | 仅支持明确来源指纹；不自动采用未知历史、重试脏 ledger 或回滚非事务 DDL。备份 attestation 不是自动证明所有业务已停写。旧线上账号未访问/收权。 |
| E05 可移植性与技术债 | 原清单目标已实现；渐进规范债保留 | AF_UNIX 短目录、Pydantic ConfigDict、19页 Table.slots/bodyCell 及组件回归已完成；AlertManager/Prometheus 缓存、Webhook API、资产密钥/SSHExecutor 已按 typed cfg 收敛。RAG 目录线程快照、读取/总量预算、完整 stage 校验和 revision-CAS 已提交，真实 Redis 边界回归通过。 | 原清单不要求全仓 Viper/ESLint 清零。2026-10-31 [配置债阶段](configuration-boundary.md)已收敛其余 Webhook 配置读取及 Cron 显式动态 policy，全仓生产 Viper 导入仅留配置边界，Go 全量/vet、相关 race 和全构建标签编译通过。前端全仓 ESLint 清理另行推进，不能据此宣称 lint 绿。 |

## 当前结论与文档口径

F01–F14 的原缺陷和 D01–D05 上表明确范围均已实现。最终固定组合 CI 及本地归档源码候选的五场景验收均通过，具体来源与镜像以[本轮候选清单](candidates/2026-10-30-eec9582a/README.md)为准；真实外部依赖、生产容量/恢复指标、部署和发布保持未验。本轮额外修复通知部分失败/业务码确认与错误保密、Silence 状态 CAS、旧结构隐藏触发器和 SSH 密文容量等可复现问题，详见专项合同与提交。

`cloud-managed-resources.md`、`huawei-client.md`、`azure-client.md`、`gcp-client.md`、`cloud-provider-credentials.md`、`cloud-temporary-credentials.md`、`notification-trigger-boundary.md` 及 `docs/multicloud-compute.md` 含有分阶段历史证据。本次已更新当前入口说明，以及被后续实现推翻的“工厂/前端尚未接入”“临时凭据 full 尚未开放”“只支持 immediate”等结论；历史测试范围保留，但不能覆盖后续交付事实。

真实模型、通知、云/集群、生产备份恢复与上线均需要各自授权及目标配置。没有实际执行的验收保持“未验”，不以代码完成、健康检查、mock、单个绿灯或历史验收替代。
