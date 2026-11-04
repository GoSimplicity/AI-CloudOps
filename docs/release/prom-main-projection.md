# Prometheus 主配置持久发布

PromMain（family1）正常生成、空目标和删除池清退已接入 source版本、完整元数据计划、同事务journal、原SQL租约和typed Store。固定内部scope现在仅为1/3/4；既有public ScrapeRuleScope仍只接受3/4，Prom有明确的固定family1窄接口。没有通过缺少依赖时的旧写路径维持测试兼容，也没有提前泛化到AM。

## 渲染与哈希合同

发布器只替换已经生成并验证逐实例字节之后的写链。原单次任务快照、detached输入、CreateBaseConfig、renderScrapeJobSnapshot、秘密字段真实marshal、pc.Load原生校验、原实例顺序ApplyHashMod、逐实例再次pc.Load均保持。空目标仍读取任务和验证输入；真实deleted池是权威清退，不读取失效的旧任务。

HTTP SD原地址/path/query保留，只设置既有port和tree_node_ids；SD自身默认HTTP client与目标认证/TLS分离，K8s SD使用自身kubeconfig，不继承目标token。静态目标内嵌端口、IPv6、job/relabel/target顺序、remote read/write、external labels和规则路径不在本次重写。

目标哈希仍结合池源和实际逐实例operational bytes。公开Job JSON隐藏token，只能表示凭据是否存在，不能作为真正发布内容的哈希。原生回归只用synthetic marker：公开JSON相同的credential-only轮换，实际YAML、源版本、metadata body/hash必须一起更新，测试不输出正文或使用真实凭据。

## 发布、归属与恢复

source版本在任何pool/job/lifecycle读前取自SQL，新nonce在同control锁内CAS预留，迟到旧快照不能取得更高gen反向覆盖已完成新源。业务配置、事件、manifest和lastCommitted同事务；原始历史全namespace与所有已deleted观察在同tx重验，current owner可以清退自己，缺少owner时不能凭member猜删。

已签计划只有覆盖fresh完整目标时才可在原sealed lease重放；同hash新发现database-only stale必须新guarded plan。legacy或accepted owner已被别的完整generated投影替代时，仅回收旧池自身empty bookkeeping；不能触碰替代方raw/fence。Prom使用独立可选prom_superseded，Record/Alert原字段与digest保持，cross/mixed字段拒绝。旧v1/v2无side数据仍可读。

Redis未知结果不透明重试，按持久SQL事实重建；真正完成需要完整artifact/member/ACK。Store私有连接池由同一Wire cleanup管理，没有为第三个消费者再创建无主池。schema/Store/Lua与main未改变。

## 已执行检查

Root owned Redis最终35叶通过，0skip/fail：原Record+Alert27保持，Prom新增8项真实场景，覆盖迟到render/retire TCP原包、legacy/B接管后empty恢复及重建、同hash新stale、actual pool/job DAO旧读、credential-only真实bytes轮换，以及SQL pool墓碑经公开清退入口不读坏任务。三family在同SQLite和同Redis开关互不跨写；实例均精确核所有权清理，最终容器0。

全Go测试、vet、所有integration/e2e/smoke标签编译、173发布工具回归和actionlint通过。完整cache224组1131叶及alert/scrape service的race通过，0skip/fail；完整DAO/DI race及vet通过。Record已发布五金样在本次共享核心下继续通过。原本机HTTP/TLS信任正反、token日志脱敏、SD、排序分片与双detached单快照回归保持执行；需要额外flag的旧Docker smoke只做标签编译，不冒称已执行。

来源锁为独立Verify41872300109/gitleaks41872300035成功的Webb3ec3e6，AIba187；新后端组合CI另验。AM发布、独立恢复worker、规模性能和剩余ESLint继续后续阶段，不以三类接线宣称整个1.0完成。预览固定clean5feb/schema4、模拟数据及外部执行关闭，本阶段没有上线或生产连接。
