# Pod 终端输入大小与压缩边界

Pod 终端继续使用 `cloudops-v1` 鉴权与现有 JSON `stdin` / `resize` / `stdout` 协议，同时保留原始消息作为 stdin 的兼容分支，不引入 SSH 的 PTY v1 帧格式。

新增边界按解压后的字节计算：

- 完整输入消息最多 512KiB
- JSON stdin 的 `data` 或 raw stdin 最多 64KiB
- resize 完整消息最多 1024 字节

完整消息预算独立于 stdin 预算，因为 JSON 会把一个控制字符转义成六个字符，64KiB 的合法 stdin 数据可能对应约 384KiB 的消息。现有尺寸零值的默认处理及 resize 队列行为保持不变。

输入使用 `NextReader` 与 `LimitReader`，完整接收并验限后才交给 Kubernetes exec。线路长度另有有界上限，不能仅依赖 `SetReadLimit` 保护压缩场景。超限消息会同步关闭自身 WebSocket、取消 exec、关闭 resize 队列，首个读取返回 0 字节和固定错误；不会部分交付该条消息，也不补传 EOT。已经验收并交付的早先消息不受该原子边界的回溯保证。

已完成先红后绿的真实 WS 回归，覆盖压缩与未压缩 raw/JSON stdin 超限、resize 超限、2MiB 消息拒绝、exec 取消及零部分交付；合法 64KiB raw 和包含 NUL 的 JSON 转义输入均完整保留。旧并发 stdout/错误/心跳、阻塞读取消、重复关闭、Pong、resize 队列、长输入和输入日志回归同时通过。

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race -tags=smoke ./pkg/terminal
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./pkg/terminal
```

本阶段未连接已有集群或生产主机，使用本机隔离的 WebSocket 连接验证传输边界，不将其作为真实容器命令执行验收。
