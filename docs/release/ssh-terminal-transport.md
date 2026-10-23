# SSH 终端传输基础修复

旧 WebSocket 命令格式保持：每个原始输入消息若未以换行结束，仅在消息末尾补一次换行，远端关闭回显。输入现在保留 pending 字节，较小的 `io.Reader` 缓冲区不会截断长输入。每条消息上限 64KiB，先完整接收再交给 SSH；超限时关闭连接，该消息不部分交付执行。超过上限的粘贴应由客户端分片，旧命令格式每个分片会补换行，因此交互式长输入应使用后续显式 PTY 模式。

主客户端和旧 `internal/tree/ssh` 入口复用同一个 reader/writer 实现，stdout/stderr 共享同一 writer 和互斥锁，输出有 10 秒写期限。输入失败不再直接写关闭帧或记录客户端关闭原因，避免额外 writer 竞争和输入内容进入日志。

旧 `Web2SSH` 保留入口，新增 `Web2SSHContext` 供拥有该会话的调用者取消。断开、取消和设置阶段超时会关闭捕获的 SSH/WebSocket 传输，阻塞 I/O 得到唤醒。主客户端继续保持原有上下文取消与 10 秒初始化期限。

已执行真实本地 WS/SSH 协议 fixture 测试：小缓冲分段读完整长消息、超限零交付与取消、两个实现的旧消息格式、断开清理、并发 stdout/stderr 128 帧串行写入及初始化/运行阶段取消。fixture 的 echo 仅验证传输字节，不作为真实远端命令执行的证据。

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race -tags=smoke ./pkg/ssh ./internal/tree/ssh ./internal/tree/api
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./pkg/ssh ./internal/tree/ssh
```

本阶段没有连接生产主机，没有改变鉴权子协议；后续 PTY 模式由显式查询参数选择。
