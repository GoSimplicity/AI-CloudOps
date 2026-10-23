# 定时任务进程取消

命令与脚本复用同一执行入口，保留既有超时时间和输出限额

Linux、macOS 等 Unix 系统为任务创建独立进程组。请求取消或期限到达时，向该进程组发送 `SIGKILL`，清理普通继承进程组的子进程。`WaitDelay` 设为 2 秒，限制父进程结束后继承输出管道的等待；失败返回时再次清理进程组，覆盖父进程提前退出而子进程持有管道的场景

非 Unix 系统保留 Go `CommandContext` 对直接子进程的取消行为，并启用相同 `WaitDelay`。完整进程组终止保障以生产 Linux 与开发 macOS 为准。主动创建新 session 或更换进程组的程序超出本机制保障，应使用独立容器等隔离方式执行此类任务

实际子进程测试通过输出 PID 握手后再触发取消，检查错误返回、期限及子进程消失；分别覆盖运行后取消、期限到达、父进程提前退出而子进程仍持有输出管道，不依赖固定 sleep 猜测启动完成

验证命令：

```sh
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/cron/handler
GOTOOLCHAIN=auto go vet ./internal/cron/...
```
