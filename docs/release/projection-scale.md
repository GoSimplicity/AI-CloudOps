# 投影验证成本基线

这是性能优化前的本机基线，不是完整发布吞吐或生产容量结论。实际运行Go1.26.10、SQLite文件数据库、真实journal与sealed lease，每个产物正文固定为微小合成YAML。没有发送Redis命令、执行Lua、取得ACK或标记SQL projected。

```sh
go test ./internal/prometheus/cache -run '^$' \
  -bench '^BenchmarkProjectionScale$' -benchtime=1x -count=1 -benchmem -timeout=15m
```

N取1/4/16/64/128，每个N分别测进入lease、单次Recheck及Store命令所需的检查/帧序列化，共15项全部执行通过；128未超时、未降低样本规模。夹具准备和退出检查不计入命令段计时，实际SELECT用GORM Query/Row回调计数，不输出SQL参数或正文。

## 命令检查与序列化段

| N | 耗时 ns/op | SELECT/op | allocs/op | JSON帧字节/op |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 1,815,542 | 40 | 7,021 | 10,267 |
| 4 | 8,240,833 | 184 | 31,739 | 32,303 |
| 16 | 78,544,916 | 1,660 | 309,929 | 243,825 |
| 64 | 1,654,743,583 | 21,964 | 5,025,265 | 3,079,031 |
| 128 | 9,651,352,708 | 84,876 | 23,974,562 | 11,856,163 |

N128累计分配4,873,801,480 B/op，约4.87GB，**不是峰值常驻内存**。该段按现有Record runner/Prepare路径执行两次真实Snapshot、一次Prepare前Recheck，以及N+3个命令各自的Recheck和真实整帧json.Marshal。N128合计132次显式Recheck、131个JSON帧。帧字节不包含未发送的RESP封装、正文参数、TLS或网络开销。

N128另测进入lease为78,985,917ns/772 SELECT；单次Recheck为72,005,083ns/643 SELECT。退出lease的643 SELECT单列，不混入上述命令段。`benchtime=1x`只用于暴露数量级和实际查询次数，不用于宣称稳定延迟分位数或跨机器性能。

## 已确认的成本来源

- 每条Store命令重新完整读取receipt、全部namespace/归属/正文；N个artifact反复触发全量Recheck，SQL数量随N呈二次增长。
- trusted binding读取同一operation的manifest并逐artifact查找，造成额外解析与分配；查询次数不能完整表达这部分CPU成本。
- 每条命令又序列化整帧，Lua同样需要全帧角色/类型/fence检查；本次未计Lua。只优化SQL后仍不能宣称整个Store线性。

后续先验证真实MySQL原连接失效及双control并发，再考虑在同一已持锁事务内复用完整初验结果。每个外部写之前仍须在原事务连接检查存活，任何错误、取消或回调结束都必须seal，不能借新连接续权。原生迟到请求、权限、AM双向量、Record旧payload和取消反例须保持。当前未实施性能优化；大正文、MySQL锁成本、Redis/Lua及端到端容量还需单独测量。
