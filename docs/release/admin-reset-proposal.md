# admin 口令重置途径方案（SEC-27 后续）

> **状态：草稿，等用户审批。** 本文只比较方案，不包含实现代码；在审批通过前不新增任何命令、环境变量或代码路径。

## 1. 背景

SEC-27 之后，启动时会检测用户名为 `admin` 的所有行（表 `cl_system_users`）是否仍在使用 SEC-27 之前 mock 内置的旧弱口令
（`internal/app/legacy_admin_password.go`）：

- 生产环境（`ENV` 为 `production`/`prod`/`prd`）命中，或检查本身读库失败：`App.Run` 返回错误，进程退出（`internal/app/app.go:142-144`，`main.go:41-44`）。
- 非生产环境（`development`/`dev`、`test`/`testing`）：只打 `[SECURITY]` Warn。

已有的 `BOOTSTRAP_ADMIN_PASSWORD` 只在**用户表为空**时创建 admin，不会重置已有账号（`internal/app/bootstrap.go`）。
因此一旦生产库中的 admin 仍是旧口令，服务起不来，也就无法通过 Web 界面修改口令；目前唯一的处置方式是负责人手工生成 bcrypt 并直接改库，
这一步没有统一流程，也没有审计。本文比较两种受控的重置途径。

## 2. 共同约束（两种方案都必须满足）

1. 口令**不得**出现在命令行参数（会进入 `ps`、shell history、容器 `docker inspect`/K8s 事件）和任何日志、审计记录、错误信息里。
2. 新口令至少 12 字符（与 `MinBootstrapAdminPasswordLen` 一致），并且不得等于旧内置口令（复用 `legacyMockAdminCandidate` 做比对，用后清零）。
3. 只写入 bcrypt 哈希（`bcrypt.DefaultCost`），明文在内存中用完即清零；读取环境变量后立即 `Unsetenv`（复用 `takeSecretEnv`）。
4. 作用范围仅限 `username = 'admin'` 的行；同名多行时全部更新或拒绝执行（审批时二选一，草稿倾向「拒绝执行并提示人工处理」）。
5. 重置后让 admin 已签发的会话失效（复用现有退出会话失效机制），避免旧口令期间签发的 token 继续有效。
6. 审计记录**只记**：操作者、时间（UTC 与 Asia/Shanghai）、主机名/Pod 名、目标用户 id 与 username、动作（`admin_password_reset`）、结果（成功/失败原因分类）、触发途径（CLI / BOOTSTRAP 开关）、变更单号。**不记**口令、哈希、环境变量值。

## 3. 方案 A：一次性 CLI `reset-admin`

### 3.1 触发

- 在与服务相同的镜像中提供子命令，例如 `AI-CloudOps reset-admin --operator <工号> --ticket <变更单号>`（参数只含身份与单号，**不含口令**）。
- 运维在维护窗口用 `kubectl exec`/`docker run --rm` 以一次性任务运行，服务本体无需先启动；命令复用 `config.Load` 的 `ENV` 校验与数据库配置。
- 执行前打印目标行（id、username、account_type、updated_at）并要求交互确认（`--yes` 仅允许在非交互的一次性 Job 中使用）。

### 3.2 口令传入

- 首选：交互式 TTY 关闭回显读取两次（`golang.org/x/term` `ReadPassword`），两次一致才继续。
- 非交互（K8s Job）：只从**文件**读取，例如 `--password-file /run/secrets/admin_reset`（路径是参数，口令不是），文件来自一次性 Secret 挂载；读取后进程内清零，命令结束后由运维删除 Secret。
- 不接受 `--password=...`，也不从普通环境变量读取（避免进入 `/proc/<pid>/environ` 与容器元数据）。

### 3.3 审计

- 在同一事务内更新口令哈希并写入 `cl_system_audit_logs` 一条记录：`UserID`=目标 admin id，`OperationType`=`admin_password_reset`，`TargetType`=`user`，`TargetID`=admin id，`Endpoint`=`cli:reset-admin`，`HttpMethod`=`CLI`，`Description` 中写操作者与变更单号，`IPAddress` 写主机名/Pod 名；`RequestBody`/`ResponseBody` 置空。
- 同时向 stderr 输出一行结构化日志（同样不含口令/哈希），便于平台日志留存。
- 数据库不可写时整体失败，不留下「改了口令但没审计」的状态。

### 3.4 用完如何失效

- 命令本身是一次性进程，退出即结束，没有常驻入口；服务进程不暴露任何 HTTP 重置接口。
- 一次性 Secret（口令文件）在执行后删除；K8s Job 设置 `ttlSecondsAfterFinished`。
- 可选加强：命令只在「检测到旧弱口令」或带 `--force --ticket` 时允许执行，避免被当作日常改密工具。

### 3.5 风险

- 任何能 `exec` 进容器并拿到数据库凭据的人都能执行 → 依赖集群 RBAC；审计记录可追溯但无法阻止。
- 新增二进制入口，需要单独测试与代码评审；误把口令写进 shell history（如 `echo pass > file`）属于操作风险，需在 Runbook 中给出安全写法。
- 同名多个 admin 行时语义不清（见共同约束 4）。

### 3.6 回滚

- 执行前按 `docs/release/backup-restore.md` 备份 `cl_system_users`（至少导出 admin 行的 `id`、`password`、`updated_at`，备份文件按密钥材料保管）。
- 代码回滚：删除子命令即可，不影响服务主流程与数据结构。
- 数据回滚：用备份的旧哈希写回——但这会恢复旧弱口令，生产环境将再次拒绝启动，因此实际回滚应是「再执行一次 reset-admin 设置另一个新口令」，而不是写回旧哈希。

## 4. 方案 B：BOOTSTRAP 重置开关

### 4.1 触发

- 新增开关，例如 `BOOTSTRAP_ADMIN_RESET=<变更单号>` 配合已有的 `BOOTSTRAP_ADMIN_PASSWORD`：服务启动时若两者都存在，且 admin 行存在，则在 `checkLegacyAdminPassword` **之前**用新口令覆盖 admin 哈希，然后继续正常启动。
- 通过修改部署（Secret/环境变量）并重启服务触发。

### 4.2 口令传入

- 复用 `BOOTSTRAP_ADMIN_PASSWORD`：读取后立即 `Unsetenv`（`takeSecretEnv` 已实现），并在启动 cron/asynq 等子进程前兜底清除（`scrubAdminPasswordEnv`）。
- 风险点：环境变量在进程启动前已进入容器规格、`/proc/1/environ`（`Unsetenv` 只影响进程内副本，不改内核保存的初始环境）以及编排系统元数据；必须用 Secret 引用而非明文写入清单。

### 4.3 审计

- 启动流程在同一事务内更新哈希并写 `cl_system_audit_logs`：`Endpoint`=`bootstrap:admin-reset`，`Description` 写变更单号与 Pod 名；操作者只能从变更单号间接得到（进程内不知道是谁改了部署），需要依赖编排平台的变更审计补充「谁」。
- 启动日志打一条 `[SECURITY] admin 口令已通过 BOOTSTRAP 重置` Warn，不含口令。

### 4.4 用完如何失效

- 关键难点：开关是**状态**而不是**动作**。只要变量未从部署中移除，每次重启都会再次重置口令（会覆盖 admin 之后自行修改的口令）。
- 缓解：重置后在库中记录「该变更单号已使用」（需要新表或配置表一行），再次遇到同一单号时拒绝并在生产环境拒绝启动，提示移除变量；这引入了新的持久化状态。
- 仍需运维在部署中手工删除 `BOOTSTRAP_ADMIN_RESET` 与 `BOOTSTRAP_ADMIN_PASSWORD`。

### 4.5 风险

- 常驻的服务进程获得了「改管理员口令」的能力，攻击面落在服务主流程上；配置漂移（变量忘删、被模板复制到其他环境）会导致重复或越界重置。
- 多副本同时启动会并发执行重置，需要加锁或依赖幂等单号记录。
- 「谁执行」无法在进程内得知，审计完整性依赖外部平台。

### 4.6 回滚

- 代码回滚：去掉开关逻辑；已写入的「单号已使用」记录可保留或清理。
- 数据回滚：同方案 A，不建议写回旧哈希，而是用新的单号再重置一次；同时从部署中移除两个变量后重启。

## 5. 对比

| 维度 | A：一次性 CLI `reset-admin` | B：BOOTSTRAP 重置开关 |
| --- | --- | --- |
| 触发 | 维护窗口内显式执行一次 | 修改部署并重启 |
| 口令传入 | TTY 无回显或一次性 Secret 文件 | 环境变量（进程内可清除，但初始环境与编排元数据仍留存） |
| 审计「谁」 | 进程内可记录 `--operator` + 单号 | 只能记单号，「谁」依赖平台审计 |
| 用完失效 | 进程退出即失效，无常驻入口 | 需手工移除变量，否则每次重启重复执行；需额外持久化防重放 |
| 多副本 | 不涉及 | 需加锁/幂等 |
| 新增攻击面 | 镜像中的一次性子命令 | 服务启动主流程 |
| 实现量 | 子命令 + 事务 + 审计 + 测试 | 启动分支 + 防重放状态 + 审计 + 测试 |

## 6. 推荐（草稿，等用户审批）

**推荐方案 A：一次性 CLI `reset-admin`。** 理由：

1. 天然一次性，用完即失效，不依赖「记得删变量」；
2. 口令可以完全不经过环境变量与命令行参数（TTY 无回显或一次性 Secret 文件）；
3. 审计可以在进程内直接记录操作者与变更单号；
4. 不给常驻服务进程增加改管理员口令的能力，也不涉及多副本并发。

方案 B 的唯一优势是运维流程只需改部署，若审批倾向 B，至少需要「单号防重放 + 生产环境发现变量未移除即拒绝启动」两项加强。

审批前不做任何实现；审批通过后另开单，代码、测试与 Runbook 单独提交评审。
