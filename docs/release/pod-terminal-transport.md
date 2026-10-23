# Pod WebSocket 终端传输

传输核心按当前 Gorilla WebSocket 与 client-go 协议实现，不改动现有 shell 探测和 Kubernetes exec 路由

实际 SDK 合同要求一个 reader 与一个 writer，`SetWriteDeadline` 也属于 writer 操作；`Close` 可与其他方法并发。client-go 的 `TerminalSizeQueue.Next` 只在监控停止时返回 nil，队列暂时没有事件时应继续等待

本次修复：

- `Read` 只持有专用 reader 锁，剩余 stdin 字节保存在会话内并跨后续 Read 完整交付，不再丢弃长粘贴超出缓冲区的部分。空缓冲 Read 不消耗消息
- stdout、stderr、错误消息和心跳共享 writer 锁，包含写期限设置，遵守单 writer 合同。Pong 回调在 reader 路径处理，不再重复获取会话锁
- `Close` 原子确认归属后立即关闭底层连接，中断正在阻塞的网络读写，不等待 reader/writer 锁。resize 的发送和关闭由独立锁协调，只有 Close 关闭通道，不靠读取一个待处理值猜测通道状态
- `Next` 在空队列等待 resize，遇到 0×0 继续监控，仅在关闭时返回 nil；单边为零仍使用既有默认行列值
- 请求取消主动关闭所属会话，输入对端断开也取消所属 Kubernetes exec 上下文；重复 Close 保持稳定，等待中的 Read/Next 被唤醒
- 原始 stdin、JSON stdin、未知操作值及客户端 Close 原因不进入日志，错误消息日志只记录长度

保留协议：客户端 JSON `op=stdin`、`op=resize`，服务端 JSON `op=stdout`；非 JSON 文本仍按原始 stdin 处理。关闭后读取仍按既有 EOT/EOF 语义结束。Close 优先即时释放连接，不等待发送 graceful close 控制帧

真实本机 WebSocket 回归先通过追踪底层 `net.Conn.Read` 复现旧锁死：确认已阻塞读后 Close 无法返回，需关闭对端才解除。修复后的 `-race` 测试覆盖：

- 已进入网络读的 Close 与请求取消均及时返回
- stdout/stderr、错误消息和 heartbeat 并发写入，160 个业务帧完整接收，格式保持一致
- 大幅超过单次 17 字节读取缓冲区的 raw 与 JSON frame 混合输入逐字节完整交付，前置 resize 不丢失
- 空 resize 队列继续等待、缓冲有值仍真正关闭、Next 关闭唤醒、重复 Close
- 实际 Ping/Pong 回调与 stdin 并行，取消后 heartbeat 与对端读退出
- 原始输入、未知操作、错误内容及 Close 原因不出现在 Debug 日志

```sh
GOTOOLCHAIN=auto go test -race -tags=smoke ./pkg/terminal
GOTOOLCHAIN=auto go vet ./pkg/terminal
GOTOOLCHAIN=auto go test ./...
```

测试只访问本机回环网络，不访问生产 Kubernetes 集群。实际 Pod WebSocket 与服务器 shell 的部署验收单独记录，不能用本机传输测试替代真实集群验收
