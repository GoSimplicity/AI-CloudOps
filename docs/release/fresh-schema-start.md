# 新栈显式初始化与运行角色限权

此入口仅初始化全新的本地隔离栈，不升级既有部署、不认领旧卷、不发布生产环境。版本化旧库采用需要独立核验流程。

## 默认离线计划

```sh
bash scripts/release-start.sh
# 或 python3 scripts/release-fresh.py
```

默认只打印阶段，不读取 `.env`、不运行 Docker、不连接数据库。首次本地初始化需要先生成新私有配置，再明确指定唯一项目：

```sh
python3 scripts/release-env.py
export COMPOSE_PROJECT_NAME="cloudops-fresh-$(openssl rand -hex 6)"
python3 scripts/release-fresh.py --execute-new-stack --project "$COMPOSE_PROJECT_NAME"
```

GitHub CI 另允许 `cloudops-ci-<run id>`。Docker 固定通过本地 Unix socket；默认 `/var/run/docker.sock`，可用 `--docker-socket` 指定另一条本机 socket 路径，不接受 TCP/SSH daemon。初始配置固定为本仓 `deploy/release/.env`，必须由当前用户持有、权限 `0600`、无符号链接与重复键。不会自动创建、覆盖或补写旧配置。

新配置包含三份不同的随机口令：MySQL 管理、专用迁移账户 `cloudops_migrator`、运行账户 `cloudops`。运行口令继续使用 `MYSQL_PASSWORD`；backend 与 webhook 的 DSN 不变。默认关闭通知、留空 webhook 与模型密钥、模型地址指向容器内不可用的回环地址，避免初始化期间触发真实外部服务。

编排子进程只继承 PATH、HOME、临时目录与区域设置等进程基础变量；不继承宿主的任何 Compose 配置覆盖。私有文件与模板默认值是唯一配置来源，宿主 `LLM_API_KEY`、通知开关、webhook、Docker/Compose 目标或镜像变量不会偷偷覆盖它。读取渲染后的实际配置时还会再次严格核验 AIOps 模型密钥为空、地址为回环 9 端口、通知关闭且 webhook 为空。

凭据文件使用 `O_NOFOLLOW | O_NONBLOCK` 打开后，在同一文件描述符上核验 owner、普通文件类型、权限与有界大小；读取前后的 inode 元数据必须一致。FIFO、设备、符号链接和读到一半发生变化的文件均拒绝，不会先阻塞等待 FIFO 写入者。

## 持久私有快照与镜像选择

开始执行后，在三仓共同父目录的 `.cloudops-fresh/<project>-<随机token>/` 保存本轮配置。父目录与本轮目录为 `0700`，所有文件为 `0600`；目录或文件存在就拒绝覆盖。它位于三仓 Git 与 Docker 构建上下文之外，成功和失败都不自动删除。入口只输出安全快照路径。

`settings.env` 直接来自首次安全读取并校验后的内存设置，不再次读取原 `.env`。`compose.yaml` 保存原始文本，后续所有 Compose 命令只使用这两份快照；构建期间修改原文件不会改变最终凭据、模型或通知设置。显式 `--project-directory` 保持原先相对 build/bind 路径，不把 canonical config 输出当新源文件再次插值，从而保留健康检查中的 `$$` 语义。

每个本轮构建服务使用 project 与随机 build token 组成的唯一 tag，并附带 owner、project、build、service 镜像标签。构建后核验这些标签，再取得不可变 image ID；另一项目对共享 `:local` tag 的构建不能进入这轮选择。MCP 等复用同一构建来源的服务共享该 image ID。非构建服务显式 pull 后选定 image ID，最终所有服务以及 schema 工具均使用固定 ID；启动禁用 pull 和重新构建。之后移动 tag 不会切换本轮选定的镜像。

快照中 `images.json` 保存完整选择，`image-records.jsonl` 逐项记录已固定 ID，部分失败也保留已完成的选择；`phases.jsonl` 记录执行阶段。`compose-command.json` 保存不含明文凭据的完整 argv，引用同一私有 env、原始 Compose 与 ID override。后续维护应使用这份命令并保持 `--no-build --pull never`，不要退回原工作区的可变 `.env` 或共享 tag。私有快照包含真实凭据，不属于普通 CI 工件。

此入口固定本次配置与选定镜像，不是完整的不可变 release：构建源码上下文、Dockerfile 和只读 bind 配置文件仍来自原工作区路径。源码构建期间变更、bind 文件后续变更不由这份配置快照冻结；完整候选应从持久的固定 Git archive 构建并管理 bind 文件。不要将本地 fresh 验证等同于生产发布验收。

## 执行顺序与拒绝边界

1. 解析 Compose 配置并验证角色分离、回环网关、资源 owner 标签与内部网络。发现该项目已有容器、卷、网络，或无 Compose 标签但同名的卷/网络，立即拒绝。
2. 使用唯一构建标识构建并核验镜像，显式拉取上游镜像，固定全部 image ID；再检查一次项目仍为空，避免长时间准备期间误接既有资源。
3. 在 `--network none` 的工具容器中读取 `schema roles --section migrator/runtime --format json` 的两个完整计划。它们不含凭据；不拆分人工 SQL 文本，不把 runtime grants 提前执行。
4. 只启动 MySQL、Redis。核验所有项目容器只有这两项且标签匹配。root 在 mysql 容器内完成全 schema 元数据和账号冲突预检，要求业务 schema 无表或视图，两名目标账号均不存在。
5. 使用普通 `CREATE USER` 创建两个全新账号；不使用 `IF NOT EXISTS` 掩盖旧账号及其历史授权。先应用 migrator grants，再显式运行新库迁移。
6. 所有应用写入进程仍停止时，显式登记 UTC 通知 clock。之后才授予 runtime 表级权限。
7. 使用 runtime DSN 分别执行 `schema verify` 和 `schema verify-clock`。通过后才启动 backend、webhook 和其余服务；应用启动继续只读核验，不能依赖 CREATE 权限。

MySQL 官方镜像的 `MYSQL_USER` 初始化会授予数据库级 `ALL`，因此本配置已移除该自动用户机制。给一个已有 ALL 账户追加表级 GRANT 不会收敛权限，此流程不会这样做。MySQL 初始健康探针使用该容器内部管理身份，并确认 PID 1 已是最终 mysqld，避免临时初始化服务假就绪；不依赖尚未创建的运行账号。

每条 GRANT 来自冻结清单的离线结构化计划，迁移角色先有全 schema REFERENCES 元数据可见性，再获得目标表所需 DDL 和两项修复/clock 所需 DML。运行角色只有业务表 DML，clock 与迁移账本只有 SELECT；没有全库 ALL、运行 DDL或账本写权限。此流程只为全新账号建立权限，不能用于收敛已有或继承的宽权限。

## 凭据与失败处理

管理口令仅留在 mysql 容器；root SQL 使用私密 stdin，输出不回显。迁移与运行核验 DSN 通过 stdin 写入 schema 工具容器 `0700` tmpfs 内的 `0600` 文件，既不进入命令行，也不通过工具容器环境变量传递。backend/webhook 仅接收运行凭据；构建上下文继续排除私有 `.env`。

所有 Compose 容器、命名卷与网络带本轮随机 `ai.cloudops.fresh.owner` 标签；临时工具容器也有 project 和 owner 标签。失败只输出固定阶段分类，隐藏 SQL、底层驱动信息和凭据。fresh 脚本不会调用 down、删卷、删账本、撤销授权或自动重试。请保留现场，判断属于空库配置错误、部分 DDL、脏账本还是旧库采用需求，再执行独立人工审核后的修复。

GitHub 集成 job 使用托管临时 runner 和本 job 唯一项目；其外层 `if: always()` 清理仍会回收该 job 的专用 fixture。这是测试生命周期清理，区别于本地 fresh 入口的失败保留，不代表 CI 会持久保存数据库卷。流水线在清理前输出安全阶段/服务状态，不上传私有 `.env`、数据库卷或 Playwright 原始日志。

## 验证与范围

```sh
python3 -m unittest scripts.tests.test_release_fresh scripts.tests.test_release_env scripts.tests.test_release -v
```

离线测试覆盖默认零执行、私有配置、独立口令、阻断官方镜像 ALL 账号机制、容器/存储归属、全元数据预检先于授权、部分迁移失败不启动写入进程、不自动清理、grant/clock 顺序、凭据只走 stdin、宿主环境注入隔离、FIFO 立即拒绝、原路径替换后的安全 FD 读取、快照后原文件编辑隔离，以及构建标签核验和 image ID 选择。

本机演练使用全新随机标签 MySQL 8.4.11/Redis fixture 和独立 schema Linux 工具镜像，运行实际的元数据预检、新账号、pre-grants、58 表迁移、UTC clock、post-grants、runtime 账本与 clock 核验。另检查运行角色业务 DML 和受保护 DDL/账本/clock 写入拒绝、第二次 fresh 启动拒绝。演练只在逐项确认 project 与 owner 标签后回收自己的资源。

另用真实 Compose 的纯配置渲染对照确认初始配置、冻结配置、原文件被改后的冻结配置完全一致，包括 `$$` 健康命令与相对路径。自有小镜像 fixture 实际验证唯一构建标签、上游 pull、别名服务 ID 复用，并在移动 schema tag 后确认直接工具执行和 Compose run 仍使用原 image ID；只回收标签核对通过的本轮资源。

本阶段不重建和启动本机三仓完整业务栈，也不连接既有服务。新提交触发的三仓 CI 将使用同一 fresh 编排入口构建应用并执行 API/浏览器验收；真实生产升级、旧卷采用及上线不属于这条入口的授权范围。
