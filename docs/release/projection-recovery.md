# 独立配置投影恢复 worker

应用启动后立即扫描已提交、尚未完成投影的当前operation，随后定期重试。它拥有独立App生命周期，不依赖系统Cron刷新或普通Prom/Alert/Record生成开关。恢复调用复用既有publisher、SQL journal、原连接租约和typed Store，不复制渲染器，也不赋予任意Redis键写入能力。

## 候选与公平性

候选必须精确关联control.LastCommittedOperationID，且operation.phase为committed；reserved、新失败attempt、projected及历史操作不作为恢复任务。每轮先确定OperationID高水位，再按keyset分页。达到数量或时间限制后保留cursor，后续轮次从该位置继续；不会追逐本轮新生成的operation，也不会反复停留在第一页的坏池。

枚举只返回locator，不解释manifest或发放权限。单个坏manifest、nonce或source内容在实际加载时拒绝，记录失败后继续后续候选；无效分页顺序或locator结构则整页拒绝。普通缺失行不是删除授权；只有SQL真实墓碑能进入删除分支，且不读取已删除池的失效rules/jobs/groups。

## 恢复与并发保护

每项恢复在读取本池源之前先取source counter或AM双source版本，再使用fresh真实源走现有发布流程。仍有效的已签计划可原样幂等重放；源已变化或AM peer使旧向量失效时，重新生成本family的受保护recovery计划。恢复AM subset不会把full状态当成自己的权限；即使full已经覆盖其输出，当前pending subset也须完成自己的新回执，不能只返回healthy而永久遗留pending。

恢复入口首先确认目标仍是当前committed。需要重新签发时，Reserve及Commit还在control锁内验证ExpectedLastCommittedOperationID，绑定原RecoveryOf目标；入口检查之后出现的新正常发布不能被旧恢复覆盖。可选guard缺省保持旧manifest/digest；已知同nonce提交事实按既有幂等规则读取，不因lastCommitted已指向自身而误拒。

AM恢复读取当前池名、实例和发送组，不把旧main的URLFile与新subset文件名拼接。新操作投影完成后不再入队；peer后续更新不能把历史projected再次当pending，因此恢复自身不会引起generation ping-pong。

## 取消与关闭

每项及每轮都有context时限。恢复入口通过可取消TryLock等待publisher锁，获锁后再次检查context；不会留下后台阻塞Lock goroutine。正常publisher原锁行为保持。App关闭时取消worker并等待其退出，再关闭SQL并返回Wire的共享Store清理；超时返回错误，不把发出cancel当作已退出。

日志仅包含family、pool ID、operation ID和分类，不回显源错误、manifest、正文或凭据。时间限制是协作式context期限，不是已测得的端到端恢复SLA；当前Store的大计划验证开销仍待下一阶段优化。

## 配置

| prometheus配置项 | 环境变量 | 默认 | 范围 |
| --- | --- | --- | --- |
| projection_recovery_interval | PROMETHEUS_PROJECTION_RECOVERY_INTERVAL | 15s | 大于0，至多1h |
| projection_recovery_item_timeout | PROMETHEUS_PROJECTION_RECOVERY_ITEM_TIMEOUT | 30s | 大于0，至多5m |
| projection_recovery_cycle_timeout | PROMETHEUS_PROJECTION_RECOVERY_CYCLE_TIMEOUT | 1m | 大于0，至多5m |
| projection_recovery_page_size | PROMETHEUS_PROJECTION_RECOVERY_PAGE_SIZE | 100 | 1至100 |
| projection_recovery_max_per_cycle | PROMETHEUS_PROJECTION_RECOVERY_MAX_PER_CYCLE | 1000 | 1至10000 |

DI兼容旧配置的零值并应用默认值；零值不表示关闭恢复。非法负数或越界值拒绝构建worker。需要停机时使用正常App关闭流程，Cron开关不会代替此生命周期。

## 本阶段验证

- Root owned Redis最终59叶通过，0fail/skip；原53保持，新增6项验证Prom/Alert/Record真正worker启动与取消、AM双向stale peer按已改名真实源恢复且不混旧文件名、重放旧TCP包不改当前状态，以及坏manifest在第一页且每轮cap为1时健康后页仍完成。使用真实SQLite journal/租约和专属Redis；规则/job定义为固定隔离输入，不连接外部采集目标。
- QA最初完整cache234main/1147leaf及service129leaf race通过。随后发现忙锁不能取消的实际缺口：旧实现6项负控全红，修后相同断言全绿；锁修后新旧worker共9main/16leaf race、完整vet/标签编译及Root全Go/59原生重新通过。没有把此前完整cache race改称为锁修后重跑。
- DAO171leaf、DI45leaf、App55leaf完整race及vet通过，实际Wire生成通过；Record五个已发布payload金样保持。173工具、actionlint和新来源锁合同检查通过。SQLite验证不等同于MySQL并发锁验收，隔离MySQL及规模性能仍需后续补齐。

来源锁为独立CI成功的Web a7fd8ff与AI75d4a87。前一AM提交304f3386的三仓CI 41891975151已53步骤成功，但不替代本次worker新提交的远端验收。

本机预览已从干净源码安全刷新到CI验证的304f3386，schema ledger仍为1/2/3/4，备份前后业务数据哈希一致，外部执行关闭。worker尚未应用到预览；本阶段没有上线或生产连接，整个1.0仍未完成。
