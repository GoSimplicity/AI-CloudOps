# 已知起始版本 a8f1700 到 v1 的显式升级

本路径仅支持 `a8f1700675bef0fe950bce5018893b9a2366f3b4` 对应的冻结49表结构。它解决旧49表既不是空库、也尚不等于v1而无法使用fresh/adopt的问题，不代表支持所有历史数据库。源49表和v1均在自有MySQL8.4.11中生成/比对；其他MySQL版本、不同collation、未知对象或结构漂移不会被猜测兼容。

冻结SQL包含9个新表CREATE、4个既有表ALTER（14个新增列均带AFTER，history额外2普通索引）和2项原数据修复，共15条。原49表没有删表、删列、修改既有列、删除索引或FK；34个既有外键保持。直接用当前AutoMigrate更新旧库会使4表列顺序与fresh v1不同，不属于本来源的完整升级证据，工具不会force认领这样的中间结构。

`known_a8f1700.json` 保存源DDL及原model/InitTables/依赖文件SHA来源；`known_a8f1700_sql.json` 保存有序SQL。独立implementation checksum绑定源快照、SQL、升级wrapper/validator、严格parser源码和目标manifest checksum。发布后不得修改这些输入来覆盖已发生历史；新的行为须作为新的受审实现交付。已发布 `baseline_v1` 及其checksum保持不变。

当前 CLI 的离线计划和实际执行统一采用 `KnownA8SafeUpgradeV2`。它保留原15条SQL，额外绑定自身源码和旧 implementation checksum；safe-v2 checksum 为 `94a07ba52e771ce4344e022556ff5d4eb21318cb574eefbda9054f0c833fab52`。旧实现 `44e466f5ff9dce7e1f304107d2e7d150810d61b9c39e957a8a67f1901d901e67` 及其输入保留用于已发生迁移的历史审计，不再作为新CLI执行入口。

```sh
go build -o /tmp/cloudops-schema ./cmd/schema
/tmp/cloudops-schema upgrade-a8f1700
```

命令默认仅输出离线来源/实现/目标hash及步骤说明，不读DSN或证明文件，不连接数据库。以下操作只能由发布者在维护窗口执行，先停全部旧应用、Webhook轮询、worker及其他迁移任务；旧AutoMigrate和不参与锁的管理员不受GET_LOCK保护。升级工具不启动旧整站、通知或云同步来“探测”状态。

```sh
/tmp/cloudops-schema upgrade-a8f1700 --execute --confirm-no-writers \
  --dsn-file "$HOME/.config/ai-cloudops/schema-migrator.dsn" \
  --upgrade-note-file "$HOME/.config/ai-cloudops/upgrade-note" \
  --backup-proof-file "$HOME/.config/ai-cloudops/backup-proof.json" \
  --notification-timezone UTC --attest-historical-time
```

DSN、note和proof均须为当前执行用户拥有的私有常规文件（0600或0400，无组/其他用户权限，拒绝软链接）。note为非空单行UTF8，最多1000字节。proof为单行、单对象JSON，字段严格如下（拒绝重复键、转义等价重复键和未知/大小写别名字段）；先真实生成、保管并人工核对备份，再填写对应SHA，不能把示例占位符当证据：

```json
{"version":1,"source_revision":"a8f1700675bef0fe950bce5018893b9a2366f3b4","backup_sha256":"<64位小写SHA256>","backup_ready":true}
```

proof是操作者对备份就绪及SHA的审计声明。该CLI不会据此自动验证归档内容、HMAC、全站恢复、RPO或RTO；不能因为给了一个SHA就声称已证明可恢复。配置与凭据仍不从应用 `.env` 自动读取。备份应与原MySQL、Redis队列、AIOps数据/私有配置及固定旧构件相互匹配，实际恢复演练属于独立证据。

safe-v2 源预检首先证明真实 `CURRENT_USER()` 具有直接 schema/global TRIGGER 权限，再查询本 schema 的 `information_schema.TRIGGERS` 确认零触发器；目标验证也重复此检查。仅 REFERENCES 可使表元数据完整，却会隐藏触发器，不能证明安全；role 继承本身不作为直接授权证明。发现触发器或无法证明可见性均在创建账本前拒绝，不自动删除或禁用触发器。维护账号须另获直接 TRIGGER 授权；运行账号无此权限，迁移凭据不进入应用。外部定时写入者仍属于必须停写的维护窗口前置合同。

其余源预检在同一锁连接内执行，只读确认49个完整BASE TABLE及全部列/索引/FK/选项，无额外表或view，且真实CURRENT_USER具备直接schema/global SELECT或REFERENCES以证明完整metadata可见性。role继承本身不当作该直接证明；迁移账号需要角色计划规定的权限及直接metadata授权。

旧账户中任意 `provider IN (4,5,6) AND status=1`（含仍标enabled的软删除记录）会在写账本前拒绝。必须先另行人工复核、明确停用；工具不代改状态，避免新版意外激活旧版不支持的厂商。原账号加密字段逐字节保留，新增credential_type默认long_term、credential_version默认0；不猜项目ID、operation ID、lease或投递成功历史。

时区必须显式UTC或有效IANA，不能使用Local或依据Git SHA猜历史解释。源49表中任一冻结DATETIME列存在非NULL值时，必须另外给出 `--attest-historical-time`，否则在写任何ledger前拒绝。此声明应基于发布者已有配置与历史证据；工具不转换历史DATETIME，不自动把高级触发的trigger_version改为1，不重放旧通知队列。升级后记录该显式zone，运行时DSN解释也须与它匹配。

引擎先精确源预检，随后创建带审计字段的ledger并持久写started，才执行冻结SQL和zone登记。只有严格58表、原两项修复后条件、完整source/implementation/target/operator/backup审计及clock匹配后才applied；不调用fresh `Migration.Apply`，也不冒称执行过fresh SQL。DDL可能隐式提交，失败或panic记录failed；连接/回执不确定保留started或返回uncertain，后续自动操作全部拒绝，不删证据、不盲重跑。部分DDL需人工核对并向前修复。

既有正常applied目标只核验，不改原来源或审计，也不重跑升级回调；历史旧实现审计不会伪改为safe-v2。CLI成功后还分别验证schema与请求的clock；重复调用若换zone会失败，不能把“schema已应用”误报为时区也可用。

```sh
go test -race ./cmd/schema ./internal/dbschema
go test -race -tags=integration ./internal/dbschema -run '^TestKnownA8' -count=1 -timeout=240s
```

实际自有MySQL验收使用冻结49表、代表业务关联行、软删除、NULL、加密占位字节、旧高级触发与待队列。生产role plan生成的最小迁移角色执行真实CLI，核对原列摘要（仅两项已审查修复值除外）、目标58表、新列默认、来源审计、只读重复与错zone拒绝；另验历史未attest/启用旧unsupported provider零ledger拒绝、部分DDL阻断重试、两个连接同库互斥。fixture不接受已有服务地址，拒Docker endpoint override，固定本地Unix --host及唯一owner标签，结束核对后清理自有容器/卷。

新增 trigger fixture 使用独立本地 MySQL：49表附加 BEFORE UPDATE 触发器会篡改未授权修复的 message_template；REFERENCES-only 看不到触发器时 safe-v2 因缺直接 TRIGGER 证明拒绝，授权后发现触发器同样拒绝，均零账本、原字段不变。移除自有测试触发器后，真实CLI仍按原15条SQL完成升级并记录新实现摘要。fixture只操作本轮拥有的容器，不能据此声称真实旧库已经验收。

本阶段业务fixture只生成并记录自有DB的0600逻辑快照SHA，没有把它当全站备份或回退成功。恢复到匹配旧构件的只读schema/DAO检查证据应独立交付；不得启动未经隔离的旧整站/worker，因为旧Webhook轮询和云账户同步可能主动外发。加列结构无破坏性也不等于旧二进制/队列协议可安全接管新数据；升级后有新写入或外部副作用时不能自动DROP新表/列来假装回退。
