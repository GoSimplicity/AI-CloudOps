# Web 终端会话隔离

本地主机和云 ECS 终端保持原有路由、JWT/RBAC、资源读取与 WebSocket 交互契约。固定子协议沿用公共 `pkg/websocket.Protocol`，即 `cloudops-v1`，未修改升级器或中间件

两个 Handler 均注入无状态 `TerminalClientFactory`，每个 WebSocket 请求创建自己的 SSH 客户端。主机、凭据、会话 map、底层传输与最终关闭均属于该请求，不再向 Handler 注入带连接状态的共享实例。同一用户同时打开不同主机或多个标签页也不会复用同一个 SSH 会话 map

连接建立使用请求上下文。Web 终端新增 `WebTerminalContext`，请求取消时只关闭当前会话捕获的底层 SSH 传输和 WebSocket；取消回调完成后才交给 Handler 执行最终 `Close`，避免中断路径与最终关闭并发修改客户端字段。WebSocket 对端断开会触发当前终端上下文取消，及时回收其 SSH 传输。NewSession、PTY、Shell 初始化阶段另有 10 秒上限，覆盖读取循环尚未启动时对端已断开的情况；shell 成功后取消该计时器，不限制正常交互终端的持续时长

SSH 初始化失败、升级失败、用户断开和请求取消均回收本次创建的客户端。升级后的关闭或执行错误只记录状态日志，不向已接管的连接写入 HTTP JSON。SSH 标准输出和错误输出共享同一带锁 WebSocket writer，遵守单 writer 契约

Wire 已实际重新生成，生成结果为一个无状态工厂供两个 Handler 调用，不再生成用于终端的全局带状态 SSH 客户端

已执行回归：

- 真实 WebSocket 握手，local/cloud 两种 Handler，各覆盖用户 A 主动断开与请求取消两种退出路径
- A/B 同时在线且使用不同主机和凭据，A 清理后 B 仍可正常往返，B 后续取消不会重复关闭 A
- 第三个连接初始化失败只关闭其自身客户端，不影响已在线会话
- 使用真实本机 SSH 协议 fixture 和 WebSocket，验证 SSH 建立会话阻塞时取消、初始化期间对端断开后期限回收、活动 shell 取消、WebSocket 对端断开均及时退出
- 真实 WebSocket 并发输出验证 stdout/stderr 各 64 帧完整接收，`-race` 通过

```sh
GOTOOLCHAIN=auto go generate ./pkg/di
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/tree/api ./pkg/ssh
GOTOOLCHAIN=auto go vet ./internal/tree/api ./pkg/ssh/... ./pkg/di
GOTOOLCHAIN=auto go test ./...
```

测试仅使用回环网络、临时资源及测试凭据，没有连接真实远端或修改共享容器。WebSocket + SSH 断开不保证远端脱离会话的后台进程被终止，具体远端终止语义仍以服务器 shell 和业务任务为准
