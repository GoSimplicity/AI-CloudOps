# 有界 artifact 批量投影

完整发布与AM subset在准备后，将已签发artifact按有界批次写入。Prepare、intent新增、成员集、intent清理及ACK继续各自执行，原ApplyArtifact接口保持；只减少artifact命令的整帧重复传输，不引入server frame引用或新的权限来源。

## 选择与预算

ApplyArtifacts先复制调用方ID列表，完整验证所有ID、重复、材料、action、role与正文SHA1；任何尾部坏输入都在首条EVAL之前拒绝。全部批次先分组完成，再开始发送。

每批最多16项，Go按完整RESP字节计量，至多1MiB；包含EVAL、当前Lua文本、numkeys、所有KEYS、完整wire JSON、selection JSON、全部raw正文及framing。单项、单体过大或frame本身超预算，在发送前选择原fenced单项路径，避免为既有合法配置新增拒收上限。预算只约束batch资格，不能声明大单体具有相同上界或Redis硬阻塞SLA。

ARGV为wireJSON、selectionJSON及逐项rawBody。selection是十进制字符串EventID数组；正文不进JSON，NUL及非标准UTF8字节的协议测试保持原样。Lua另有16项及raw正文合计1MiB限制；Lua不能观察EVAL源文本，因此不声称服务端能核验Go的同等完整RESP预算。

## 拒绝与恢复

每批仍验证原完整帧、所有role/TYPE、pool及**全部namespace向量**、同版本frame硬CAS，再验证本批所有ID/唯一性/action/argc/hash。全部逻辑验证完成后才开始写第一个artifact；不能只核选中namespace。AM subset推进NS5后，旧full的迟到main/batch仍由全向量判为STALE。

ordinary继续单项invalidate/retire路径，不能用batch取得render权限；subset5只处理本plan已签render，不能写main、full成员、SHA或intent。每条实际EVAL之前继续验证原SQL事务连接，不得用健康全局DB续权；空选择也要验证lease。

任何EVAL未知结果或明确拒绝都直接返回，不自动重试，也不在错误后回退单项。服务端已执行而客户端丢失回复时，SQL保持committed未projected，重建后按持久事实幂等恢复。Lua运行错误不会回滚已写内容，因此这里只保证逻辑校验先行，不声称跨SQL/Redis原子提交或客户端timeout能安全中断已运行脚本。

## 实际验收

Root owned Redis最终86叶0fail/skip：原64保持，新增独立及实际批次场景，覆盖尾body/ID/TYPE、duplicate、prepared SHA/ACK/role、未选namespace接管、ordinary/subset、数量/字节预算、raw字节、原事务rollback/退出后零EVAL，以及server完整执行后丢回复不重试且回执可恢复。正例真实完成全部raw、成员、ACK及SQL projected，避免“脚本不认识新mode使负例假绿”。MySQL12叶保持通过；全Go/vet/所有integration/e2e/smoke标签编译及actionlint通过。

相同真实规模夹具的race单次观测，N128相对批量前：

| 指标 | 批量前 | 有界批量后 |
| --- | ---: | ---: |
| EVAL次数 | 131 | 11 |
| JSON帧字节 | 11,859,831 | 995,883 |
| EVAL RESP参数字节 | 16,093,977 | 1,369,684 |
| SQL SELECT | 907 | 788 |
| 完整投影ns | 1,463,941,250 | 456,845,334 |
| Redis EVAL累计微秒 | 657,532 | 87,891 |

raw正文均为微小合成值，累计1408字节保持。测量使用SQLite及专属Redis、开启race；不是MySQL+Redis容量或生产延迟分位数。固定16项下仍重复整帧与全向量，不能称整体复杂度已线性；保留原15项legacy单项微基准作为对照，不能把它冒称当前批量runner总成本。

新的Redis/MySQL实例均精确按随机label/fullID清理；无schema/manifest/SQL权限变化，预览保持CI已验证的67df64e9与原数据。新提交仍需远端CI，未上线。

冻结字节完整cache244main/1175leaf及alert/scrape service129leaf race通过，0fail/skip；Go边界和独立review负控保留。来源锁升级为独立CI成功的Web9339393及AI75d4a87，新组合另验。
