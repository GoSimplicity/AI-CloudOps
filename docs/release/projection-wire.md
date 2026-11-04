# 投影传输编码

`pw2`只压缩传输字段，不压缩权限证明。每个EVAL仍显式携带全部KEYS、完整namespace向量和角色元数据，没有服务器frame缓存或跨请求引用。ACK和legacy owner intent JSON作为原始字符串传输，不解码重编码。

顶层是15项位置数组：marker、mode、family、pool、target、roles、fences、artifacts、controls、pool_fence、ack、legacy、legacy_sha、ack_receipt、ack_signature。角色行为元组5项，fence4项，artifact6项，control8项；角色的key仅从同索引KEYS恢复，Go编码前先核对原始role.Key与KEYS一致。

Lua在任何Redis操作前验证新格式的精确元组长度、集合形状、标量类型和索引范围，再重建旧canonical对象。随后执行原有全部角色/TYPE、全部namespace fences、prepared frame fingerprint、body、控制和ACK检查。新旧对象格式得到相同fingerprint，编码marker不参与签名；已有持久状态不迁移。大ID和版本保留十进制字符串，不经Lua浮点转换。

未知marker、多余或缺失元组项、null、错误布尔/数值类型都不默认修补。Redis cjson把空对象与空数组都解码为空table；合法空集合槽位保持这一语义等价，不声称能区分二者。固定非空元组仍必须具有精确arity。

新发送默认使用pw2；Lua同时接受旧对象，支持已经排队的旧请求进入相同fencing路径。旧对象兼容不放宽原有业务校验，普通操作仍不具备生成配置写权限，AM subset仍只能更新本plan webhook。

批次继续最多16项、Go计算完整EVAL RESP预算1MiB；实际Lua源码长度计入预算。所有选择的预验、每批全帧检查、超预算发前single选择及未知结果后不重试保持。原始正文仍独立ARGV，任意字节不经JSON。

本阶段不声明任意实例数或任意正文的容量SLA。完整KEYS和全向量验证仍随配置规模增加，过大完整请求仍可能退回single；减少字段重复不能等同于消除所有规模成本。旧布局基准保留对象编码对照，真实新传输的字节与结果由owned Redis场景单独观测。

## 首轮真实投影观测

原84个owned Redis协议叶场景和4个MySQL/Redis组合恢复均实际race通过，没有跳过。另用显式flag观测1024个原始45字节IPv6地址、每项11B正文：

```sh
CLOUDOPS_TEST_PROJECTION_STORE=true CLOUDOPS_TEST_PROJECTION_COMPACT_CAPACITY=true \
  go test -race -json -tags=integration ./internal/prometheus/cache \
  -run '^TestProjectionStoreOwnedRedisProtocol$/^scale_cost$/^N1024_IPv6Raw45$' \
  -count=1 -timeout=240s
```

实际67次EVAL（64个artifact批次与3个控制命令），所有命令均pw2。完整frame35,150,588B，原始正文11,264B，完整EVAL RESP参数52,976,776B；实际Redis INFO计数相符。逐字raw、精确成员集合、canonical ACK、legacy SHA及SQL projected全部核对通过。

单次本机SQLite/race完整投影22,792,527,417ns、6220 SELECT、server EVAL累计5,774,413us，仅作观测，不作为通用延迟承诺。hooks增加了测试侧完整双格式解码，整段时间还包含该观测成本。配置保存、源生成及journal提交不在投影计时内。

此前同规模长地址旧对象布局预测artifact阶段会退回1024次single、总RESP1,221,622,701B；那是未发送的布局测量。本次是实际全控制流程传输，不冒称两边都做过GB真实网络发送或精确同口径耗时比较。较大计划仍有成本边界，不能据1024场景推导无限容量。

## 最终回归与独立审查

最终owned入口107叶场景（原84协议、新21紧凑格式反例、2远端Docker环境拒绝）实际race全部通过，0fail/skip。新增专项包含旧对象prepare与新格式动作互换及反向、相同frame重复ACK/DUMP不变、空集合{}与[]、最后元组的长度/null/类型错误、未知codec、KEYS重定向、未选产物TYPE与更高namespace版本、超过2^53的事件ID、AM非空Add/Remove原字节、subset和ordinary权限边界。负例要求明确Lua回复并核对完整相关DUMP，不能把网络错误当成功拒绝。

四项真实MySQL/Redis组合恢复及1024长地址容量场景也通过。全Go测试、全vet、integration/e2e/smoke标签编译、integration cache vet、完整cache race及5项来源锁检查通过。生产Lua从原role路径开始的后续字节保持一致，只将array校验函数移到解码之前；独立审查确认所有原fencing/fingerprint/写分支保留。

新增专项首轮20/21通过，空集合场景直接将私有p.proof交给MarkComplete，绕过公开Acknowledge对空vector的规范化而被严格SQL比对拒绝。隔离SQL反证确认原因后，测试改为使用公开Acknowledge的返回proof，保留原空集合/跨格式/DUMP断言；最终107全部通过，没有修改生产或放松MarkComplete校验。

来源锁提升到独立CI已成功的Web841a313及AI75d4a876。新的固定组合仍需远端CI验收；本地演示继续使用已验证的67df64e9，数据和服务保持，未上线。
