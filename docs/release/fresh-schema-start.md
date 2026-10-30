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

后续所有Compose操作应保留同一COMPOSE_PROJECT_NAME，避免指向默认或其他项目。GitHub CI 另允许 `cloudops-ci-<run id>`。Docker 固定通过本地 Unix socket；默认 `/var/run/docker.sock`，可用 `--docker-socket` 指定另一条本机 socket 路径，不接受 TCP/SSH daemon。配置固定为本仓 `deploy/release/.env`，必须由当前用户持有、权限 `0600`、无符号链接与重复键。不会自动创建、覆盖或补写旧配置。

新配置包含三份不同的随机口令：MySQL 管理、专用迁移账户 `cloudops_migrator`、运行账户 `cloudops`。运行口令继续使用 `MYSQL_PASSWORD`；backend 与 webhook 的 DSN 不变。默认关闭通知、留空 webhook 与模型密钥、模型地址指向容器内不可用的回环地址，避免初始化期间触发真实外部服务。

编排子进程只继承 PATH、HOME、临时目录与区域设置等进程基础变量；不继承宿主的任何 Compose 配置覆盖。私有文件与模板默认值是唯一配置来源，宿主 `LLM_API_KEY`、通知开关、webhook、Docker/Compose 目标或镜像变量不会偷偷覆盖它。读取渲染后的实际配置时还会再次严格核验 AIOps 模型密钥为空、地址为回环 9 端口、通知关闭且 webhook 为空。

## 执行顺序与拒绝边界

1. 解析 Compose 配置并验证角色分离、回环网关、资源 owner 标签与内部网络。发现该项目已有容器、卷、网络，或无 Compose 标签但同名的卷/网络，立即拒绝。
2. 构建镜像，再检查一次项目仍为空，避免长时间构建期间误接既有资源。
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

离线测试覆盖默认零执行、私有配置、独立口令、阻断官方镜像 ALL 账号机制、容器/存储归属、全元数据预检先于授权、部分迁移失败不启动写入进程、不自动清理、grant/clock 顺序、凭据只走 stdin，以及宿主环境注入隔离和渲染配置的双重拒绝。

本机演练使用全新随机标签 MySQL 8.4.11/Redis fixture 和独立 schema Linux 工具镜像，运行实际的元数据预检、新账号、pre-grants、58 表迁移、UTC clock、post-grants、runtime 账本与 clock 核验。另检查运行角色业务 DML 和受保护 DDL/账本/clock 写入拒绝、第二次 fresh 启动拒绝。演练只在逐项确认 project 与 owner 标签后回收自己的资源。

本阶段不重建和启动本机三仓完整业务栈，也不连接既有服务。新提交触发的三仓 CI 将使用同一 fresh 编排入口构建应用并执行 API/浏览器验收；真实生产升级、旧卷采用及上线不属于这条入口的授权范围。
