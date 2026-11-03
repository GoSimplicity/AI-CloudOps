# 系统关键指标与健康状态

健康状态只在CPU、内存和磁盘三项关键采集完整且有效时计算。CPU必须有限且位于0—100，CPU核心数及内存/磁盘总量必须为正，已用量不能超过总量；内存和磁盘探针的`total`与`used`都必须明确存在。真实采得的零CPU或零已用量仍合法，缺字段不能补零冒充空闲。

关键探针依次采集、完整校验后才一起提交结果；任何失败或取消都不返回或缓存部分成功。失败消息不回显底层命令输出，公开方法保留`context.Canceled`等错误身份。Windows尚无采集实现，现在明确报不支持，不再返回全零成功；其它不支持平台同样报错，没有承诺Windows能力。

缓存最初以v2隔离旧失败后补零记录，严格底层解析升级后进一步采用`system:info:v3`，不继续读取旧解析语义下看似合法的v2样本；旧key不强制删除，按原TTL过期。缓存还须满足有效关键指标、非未来时间及5分钟有效期。无效或过期缓存重新采集，失败不回填。响应转换对nil或无效快照返回“未知/不可用”，不序列化NaN等无效数值，也不显示健康。

既有有效值的健康阈值保持。CPU型号、负载、运行时间、进程数、系统版本和流量等非关键采集项保留既有可选行为，本阶段不宣称每项都成功或重写所有平台探针。

## 验证

资源失败、缺字段、容量边界、NaN/Infinity、零值、三公开方法取消、缓存时间及阈值测试通过。独立发现的缺`used`和公开错误身份5个原反例保持原预期，修复后全部通过；没有将错误预期调整成通过。

本机Mac真实采集调用通过，返回完整有效的关键指标，未打印主机详情；这不是Linux/Windows实机验收。System service/utils完整race和vet通过；与D1冻结后全Go test/vet、相关五包race、全标签只编译通过。没有修改schema或连接生产，未上线。

## 底层格式完整性：2026-11-04

Linux CPU全部已提供的4—10个计数字段必须为有效无符号整数，总和溢出、计数回退或没有观测区间均拒绝，不把解析失败当作真实空闲。guest/guest_nice已经计入user/nice，总时间不再重复相加；依据[内核CPU记账实现](https://raw.githubusercontent.com/torvalds/linux/master/kernel/sched/cputime.c)。iowait等不可靠或回退样本不强行伪造有效百分比，[内核proc说明](https://docs.kernel.org/filesystems/proc.html)也明确其限制。

Darwin必须读到完整且自洽的user/system/idle三项。兼容常规小数和Apple实际输出的未补零百分位格式，但只有三项能共同建立约100%的样本才接受；依据[Apple top实现](https://raw.githubusercontent.com/apple-oss-distributions/top/main/globalstats.c)及本机实采，不能仅抓第一个百分比当CPU使用率。上述官方来源核验于2026-11-04，不把main分支视为永久固定OS版本。

vm_stat必须包含有效page-size头、free和inactive计数，检查4K/16K等页面单位、重复/缺失字段、溢出及容量。沿用基于free+inactive的内存估算口径，未冒充Activity Monitor各项内存统计的完全等价实现。

原三个假命令/计数拒绝反例保持原预期全部转绿；独立Linux guest记账、批量Apple百分位三元组及4K/16K容量边界通过，最终本机Darwin smoke通过。全Go test/vet、相关七包race、全标签只编译，以及v3缓存调整后的System两包race/vet通过。没有新增依赖、schema或重写可选探针，未上线。
