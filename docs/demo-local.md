# 本地演示环境恢复

`scripts/demo-local.py` 用于检查和恢复**已经准备好**的本机演示环境，不是全新安装器。它不创建数据库、不执行 schema migration、不生成密钥，不启动旧镜像应用，也不删除数据库或 volume。

当前环境使用最新本地 Go 源码后端（8889）、独立 Compose project `cloudops-demo-20261103`、MySQL `127.0.0.1:13306`、Redis `127.0.0.1:16379`。原 `cloudops` volumes 不变。前端和演示 API 代理由各自进程维护，本工具不启动或重启它们。

## 已准备环境的要求

- 私有运行目录默认 `/tmp/cloudops-demo-20261103`，属于当前用户，权限 `0700`。
- `runtime.env`、`compose.env`、`schema.dsn`、已有 `backend.pid` 均为当前用户的普通文件，权限 `0600`，不能是符号链接。`compose.env` 是准备时生成的私有副本，工具不修改 `deploy/release/.env`。
- 运行配置必须是 `ENV=development`、`MOCK_ENABLED=false`，邮件和飞书关闭、本地 exec 关闭、Prometheus alert/record 开关关闭。Redis 必须是上述本机地址；AIOps 地址保留本机 `http://127.0.0.1:18081`。
- MySQL 必须使用 `cloudops_demo` 用户、字面量 `127.0.0.1:13306` 和 `cloudops_demo_` 前缀数据库。schema DSN 与运行 DSN 必须完全一致，包含 `charset=utf8mb4&parseTime=True&loc=UTC`。
- Docker 必须选择本地 Unix socket；拒绝 SSH/TCP daemon。已准备容器须属于上述 project，数据库端口只发布到 loopback。
- schema 和 UTC 通知时区已经通过官方 `cmd/schema` 初始化。已有三个内置 Cron 行必须保留；所有内置和用户 Cron 均禁用。K8s kubeconfig 和 Prometheus instances 必须为空。
- 当前 macOS 机器已缓存 Go `1.26.10` 工具链、依赖以及 MySQL/Redis 镜像。启动构建使用 CLT、`GOPROXY=off` 和 `GOSUMDB=off`；不拉镜像或依赖。

缺少这些文件、schema 或依赖时，工具会失败，不能将其当作 fresh 安装完成。`/tmp` 可能被系统清理；恢复前自行保管需要保留的私有配置，不要提交到 Git。

## 只读检查

在 Go 仓库根目录执行：

```sh
codex-python scripts/demo-local.py status
```

输出只含 PID、依赖健康状态、演示数据库名、Cron/资源检查和健康状态，不含 DSN、口令、token 或子命令原始错误。`status` 不启动依赖、不更新数据库、不发送登录请求、不修改任何进程。

当前管理员用户名为 `admin`；密码仅保存在私有 `/tmp/cloudops-demo-20261103/login.env`。不要把它复制进终端命令参数、日志或仓库。

## 恢复后端

```sh
codex-python scripts/demo-local.py start
codex-python scripts/demo-local.py status
```

已记录 PID 必须确实属于当前用户、执行该目录的 backend、工作目录匹配并监听 8889。若当前后端健康，重复 `start` 直接复用，不重启、不重新编译、不变更数据；不明确归属的 PID 或端口会导致拒绝操作。它不会停止进程，因此也不会自动替换一个仍在运行的旧版本。

后端已停止时，工具只恢复 MySQL/Redis，编译当前 checkout 的 schema 工具并执行 `verify`、`verify-clock`，禁用全部 Cron 并检查无外部 K8s/Prometheus 目标，再编译当前源码后端、写入私有日志/PID 并检查 readiness。启动以文件锁串行执行。凭据只通过私有文件、进程环境或 stdin 交给本地程序。

内置任务在应用启动时可能立即执行，旧 `TREE_CHECK_STATUS_CRON` 等周期变量不能替代数据库禁用。工具拒绝缺少内置任务行的数据库，避免启动过程重新创建默认启用任务。演示资源应保持 TEST-NET/无真实凭据，界面上不要启用 Cron、添加真实集群配置或执行验证云账号等外部动作。

## 可选补齐演示数据

仅在后端已停止且 `owner.json` 已有当前数据库的 `admin` UID 时使用：

```sh
codex-python scripts/demo-local.py start --seed
```

工具核对 owner，编译并调用已有 `cmd/demo-seed`，只补缺失数据，之后再次禁用全部 Cron。运行中的后端不接受 `--seed`，避免悄悄修改用户正在查看的数据。

## 验证范围

工具的自动测试全部使用 Fake 进程、Docker 和数据库，不访问真实依赖：

```sh
codex-python -m unittest discover -s scripts/tests -p test_demo_local.py -v
```

本轮实际执行了只读 `status`，并在健康服务上连续两次执行不带 `--seed` 的 `start`：PID 均保持28356，依赖、禁用Cron和就绪检查通过，没有重启或改变数据。未停服；停服后的依赖恢复、构建和启动路径尚未实际执行，其行为目前由隔离测试覆盖。
