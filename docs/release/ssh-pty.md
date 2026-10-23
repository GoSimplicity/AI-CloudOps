# SSH 显式 PTY 协议

本地和云主机终端均可通过查询参数 `terminal_mode=pty-v1&cols=96&rows=37` 选择交互式 PTY，WebSocket 鉴权子协议仍为 `cloudops-v1`。未声明模式时保留已发布的逐消息命令格式与关闭回显，不根据输入内容猜测 JSON。

## 帧与尺寸

| 方向/类型 | PTY v1 行为 |
| --- | --- |
| 客户端 binary message | 原始 stdin 字节，保持 Ctrl+C、方向键、NUL、非 UTF-8、部分输入，不追加换行 |
| 客户端 text message | 只允许 `{"op":"resize","row_size":42,"col_size":120}`，拒绝未知字段、操作和多段 JSON |
| 服务端 binary message | stdout/stderr 原始字节；客户端可直接交给 xterm 的 Uint8Array 输入 |

初始尺寸省略时为 80 列、25 行；显式尺寸以及 resize 均限制列数 1～1000、行数 1～500。HTTP 查询参数错误会在连接 SSH 和升级 WebSocket 前返回 400。PTY 请求设置远端 `ECHO=1`，前端应直接转发每个 `onData` 片段，停止本地行缓冲与回显。resize 通过 SSH `window-change` 请求生效。

stdin 每条消息最多 64KiB，控制消息最多 1024 字节，均按解压后的字节计算。输入使用 NextReader + LimitReader 完整接收并验限，再交给 SSH；压缩放大或未压缩超限会主动关闭连接，该条消息零部分交付。线路长度额外受有界限制，避免只依赖 Gorilla 对压缩线路长度的上限。64KiB 以内的长消息会在多个小缓冲读取中完整交付。前端需要按 UTF-8 字节分片长粘贴，每条不超过上述上限。

两条 SSH 实现共用 reader/writer；并发 stdout/stderr 通过同一个串行 writer 发送，写期限 10 秒，断开和请求取消会关闭捕获的连接并唤醒阻塞 I/O。父上下文 deadline/cancellation 保持原有错误身份，初始化仍有 10 秒期限。`WebTerminalContext` / `Web2SSHContext` 旧入口委托默认 legacy options；新显式入口接受 TerminalOptions。

## 验证边界

本机真实 SSH 协议 fixture 使用独立主机密钥及生产模式 known_hosts 校验，覆盖两种客户端实现的 PTY/ECHO/尺寸握手、实际 window-change、Ctrl+C/方向键/64KiB 原始输入、binary JSON 不变成控制、并发 stdout/stderr 和非 UTF-8 二进制输出。fixture 记录协议和固定输出，不执行用户命令，不用 echo 冒充真实 shell 执行或真实远端验收。

真实 WS 回归覆盖压缩和未压缩超限零交付、合法压缩输入、非法控制关闭、原始输入不进入日志、断开/deadline回收；双 handler 验证参数传递、非法输入零连接和固定鉴权子协议。先前的会话隔离与 legacy 回归同时保留。

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race -tags=smoke ./pkg/ssh ./internal/tree/ssh ./internal/tree/api ./pkg/di
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./pkg/ssh/... ./internal/tree/ssh ./internal/tree/api
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test ./...
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go build ./...
```

Wire 已实际重新生成且结果无差异。浏览器连接真实 OpenSSH/交互式 shell 的端到端验收由独立本机测试环境执行，本阶段未连接生产主机。
