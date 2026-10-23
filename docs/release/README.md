# 1.0 部署与验收

本目录记录三个仓库的集成交付。版本通过验收前，不将构建成功标记为生产就绪。

## 部署目录

三个仓库需要同级放置，前端和 AIOps 目录名与下面一致。后端仓库使用 GitHub 原名 `AI-CloudOps`，也可改名。

```text
project/
  AI-CloudOps/
  AI-CloudOps-web/
  AI-CloudOps-aiops/
```

在后端仓库执行：

```bash
bash scripts/release-start.sh
```

该入口检查同级仓库和 Docker，按需生成私有配置，构建并等待服务健康。需要分别执行时：

```bash
python3 scripts/release-env.py
docker compose -f deploy/release/compose.yaml config --quiet
docker compose -f deploy/release/compose.yaml up -d --build --wait --wait-timeout 300
python3 scripts/release-smoke.py
```

在隔离验收环境增加 `--crud`，会创建唯一命名的工单分类，核验持久化、URL ID 和操作者防伪造，然后删除该测试分类。它会在审计日志中留下真实操作记录，不应当作无副作用的生产探活。

本机运行 Go/Python 进程联调时，可叠加 `-f deploy/release/local-ports.yaml` 暴露回环地址上的 MySQL `13306` 与 Redis `16379`。对此环境后续所有 Compose 操作都应使用相同文件集合，否则 Compose 会重新创建服务并移除端口。测试直连 Go 使用 `release-smoke.py --api-only --url http://127.0.0.1:17889`。

密钥生成器创建权限为 `0600` 的 `deploy/release/.env`，文件已被 Git 忽略；重复执行会拒绝覆盖。首次管理员为 `admin`，密码取该文件中的 `BOOTSTRAP_ADMIN_PASSWORD`。已有数据库的管理员不会被重置。检查日志时不要粘贴完整环境变量或展开后的 Compose 配置。

默认访问地址为 `http://localhost:18080`，只绑定回环地址。对外服务应通过有 TLS 的反向代理，配置 `PUBLIC_ORIGIN` 为实际 HTTPS 源站。不要直接暴露 MySQL、Redis 或 AIOps。浏览器所有 `/api/v1/*` 请求经过 Go 的 JWT/RBAC 检查，再由 Go 注入内部密钥。

## 真实依赖

- MySQL 保存用户、权限、资产、工单等业务数据。使用独立 `cloudops` 用户，初始化权限只作用于其数据库。
- Redis 提供缓存、任务队列与向量数据持久化；当前 RAG 检索索引由应用内 FAISS 管理。
- Prometheus 默认只监控自身。生产指标需要配置实际采集目标，不能将空数据当作健康状态。
- AIOps 启动和 HTTP 健康检查不要求模型提供商在线。模型推理、知识库嵌入和自动修复需要分别配置与验收；`/health` 通过不代表这些能力就绪。
- `LLM_API_KEY`、`LLM_BASE_URL`、`LLM_MODEL` 可在本地 `.env` 中配置。不得将提供商密钥放入前端的 `VITE_*` 配置。
- Kubernetes 与 SSH 使用组织提供的最小权限凭据。部署模板不挂载开发者本机 kubeconfig，不连接已有集群执行修复。

## 升级与回滚

升级前备份 MySQL、Redis、AIOps 数据卷及 `.env`，保存三仓 commit SHA 和镜像 digest。尤其保留 `TREE_PASSWORD_ENCRYPTION_KEY`，更换它会影响已存资产密码的解密。先在数据库副本上演练升级；当前 GORM 自动迁移并不等价于可逆迁移。

使用固定 `RELEASE_VERSION` 构建镜像，保留上一版镜像。应用回滚可指定上一版镜像，但数据库结构变化必须依据升级前备份和演练结果处理。`docker compose down` 保留数据卷；不要在生产使用 `down -v`。

`python3 scripts/release-backup.py` 创建权限受限、不会覆盖已有目录的 MySQL 一致性快照和部署密钥备份，记录三仓 commit。默认输出目录 `deploy/release/backups.local` 被 Git 忽略。该脚本明确不包括 Redis 队列和 AIOps 上传文档；完整恢复点还必须在暂停相关写入后保存对应数据卷。备份含资产密文、用户哈希和部署密钥，应存放在加密存储，限制访问。

发布模板将访问令牌有效期设为 30 分钟。沿用当前后端配置契约：`JWT_EXPIRATION` 的单位为分钟。

## 发布门槛

发布工具回归使用 `python3 -m unittest discover -s scripts/tests -v`，无需运行容器，覆盖仓库路径、启动预检与私有密钥的防覆盖行为。CI 与标签发布均执行该检查。

每次交付保存命令、退出码和日期，区分离线测试、隔离环境集成、真实外部依赖验收：

1. Go 全量测试与构建；Python 单测、依赖一致性和完整 lifespan；前端类型检查、单测与生产构建。
2. 新空库启动、管理员登录、普通用户越权拒绝、注销与过期会话拒绝。
3. 同源网关下静态页面、JWT API、AI 内部鉴权、SSE 与 WebSocket。
4. 统一浅色/暗色样式，表格、表单、空态、错误提示及窄屏可用性，以浏览器实际页面检查。
5. 真实集群只读查询、隔离资源变更与恢复；真实模型推理和知识检索；通知通道投递。
6. 备份恢复演练、升级回滚演练，以及曾被提交到 Git 的凭据撤销或轮换。

`release-smoke.py` 覆盖网关、登录与 AI 代理边界，不执行集群变更，也不把 AI 能力不可用当作全量功能验收通过。

## 依据

- [Docker Compose 健康依赖](https://docs.docker.com/compose/how-tos/startup-order/)：依赖容器启动不代表服务就绪。
- [Docker Compose secrets](https://docs.docker.com/compose/how-tos/use-secrets/)：生产密钥应由部署系统管理；本地 `.env` 适合隔离验收，受 Docker 主机管理员权限保护。

## 运行时版本

发布模板使用 MySQL 8.4.11 LTS 与 Redis 7.4.11。已有 MySQL 8.0 数据卷升级前必须备份并在副本演练，不能依赖直接降低镜像版本回滚。依据：[MySQL 支持公告](https://www.mysql.com/support/eol-notice.html)、[Redis 7.4 安全修复](https://redis.io/docs/latest/operate/oss_and_stack/stack-with-enterprise/release-notes/redisce/redisce-7.4-release-notes/)。

网络无法稳定访问 npm 官方源时，可在私有 `.env` 中设置 `NPM_REGISTRY=https://registry.npmmirror.com`。默认使用官方源，不关闭 TLS 或变更锁文件。

独立 MCP 服务复用 AIOps 镜像，只在 Compose 内网监听 9000，使用同一服务密钥；AIOps 通过 `MCP_SERVER_URL=http://mcp:9000` 连接。健康检查仅代表进程可用，具体工具仍需要各自的数据源配置。

## 定时脚本运行环境

后端发布镜像包含 Bash、Python3 和 Node.js，分别支持现有 shell、Python、JavaScript 脚本类型。运行命令具有后端容器的权限，应限制相关菜单/API 的角色授权。任务输出按持久化字段预算截断并标明，不将超限数据无限缓存在内存中。

`GET /health` 仅确认 HTTP 进程存活；`GET /ready` 在两秒总期限内检查数据库与 Redis，依赖不可用返回 503。就绪不包含模型、集群或外部通知能力，相关能力仍需分别验收。发布 smoke 的检查在 Python 优化模式下也生效。
