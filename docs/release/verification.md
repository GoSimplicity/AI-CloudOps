# 交付验证记录

日期：2026-10-22，Asia/Shanghai。持续更新，当前还不是最终 1.0 验收结论。

## 已执行

| 范围 | 结果 | 边界 |
| --- | --- | --- |
| Compose 配置 | `config --quiet` 通过 | 配置可解析不代表镜像可启动 |
| MySQL、Redis、Prometheus | 隔离 Docker 环境实际启动，MySQL/Redis healthy | 新建数据卷，无生产数据 |
| 最终 Go 镜像 | Go 1.26.10 安全依赖版构建、启动及同源 smoke 通过 | 当前可达漏洞扫描为零，未宣称全部依赖无漏洞 |
| Go 主机进程 | 生产配置启动、空库迁移、随机管理员密码初始化通过 | 连接同一隔离数据库 |
| 登录与会话 | 登录、profile、刷新通过；注销后短/长令牌失效 | `release-smoke.py --api-only` |
| 业务列表 | 用户、角色、API、审计、定时任务、分类、表单、流程、模板、工单 10 个接口成功 | 新空库列表，非全量业务流 |
| 工单分类 | 创建、读取、更新、清理通过；客户端伪造操作者和 URL ID 均被拒绝覆盖 | 真实数据库写入，测试记录已清理 |
| Go → AIOps | JWT 代理注入服务密钥成功；匿名请求 401 | 模型缺少配置时就绪明确 503 |
| 最终 Python 镜像 | 依赖重建后纯 wheel 构建、生产启动通过 | 87 单测通过，锁定依赖审计零已知漏洞 |

## 追加实测

- 完整工单：分类、表单、发布流程、模板、草稿、提交、审批完成、评论、时间线及审计全部通过，测试对象已清理。
- 权限：只读角色 GET 成功、写操作 403；撤销立即生效；冻结和删除账号拒绝旧 access/refresh token。
- Kubernetes：独立 kind 1.33.1 集群，主机与 Docker 同源两条链路均通过 namespace/ConfigMap/Deployment/Service 创建查询、扩容、Pod 日志及实际 Service HTTP 内容。TLS 验证保留。只覆盖共享稳定 API，未验证所有 Kubernetes 版本。
- MySQL 从本次隔离 8.0 数据卷升级至 8.4.11，45 张表保留；Redis 升至 7.4.11。升级前已创建私有备份。
- 数据库恢复：快照在独立临时数据库恢复 45 张表，验证后仅删除该临时数据库。
- 前端：类型检查、245 个单测、完整生产构建通过；开发环境已走查业务路由和真实 Kubernetes 资源，最终 Docker 浏览器回归通过。

## 待完成门槛

- 最终镜像全栈启动、同源网关、浏览器全业务路由及明暗主题检查。
- SSH、真实模型与通知通道验收。
- 云账号按用户明确要求，以官方文档和 SDK 契约核对验收；无真实账号，未执行真实云调用，不将此列为交付阻断。
- 备份恢复与回滚演练；历史暴露凭据的撤销/轮换。

未配置的外部能力应返回明确不可用原因。此记录不将“未调用”或“使用测试替身”写为真实服务通过。

## 最终运行时回归

MySQL 8.4.11 / Redis 7.4.11 升级后，通过同源 `http://localhost:18080` 再次执行并全部通过：`release-smoke.py --crud`、`workorder_flow.py`、`account_access.py --rbac`，以及省略 Service 协议和类型的 `kubernetes_smoke.py --execute`。最后一轮 Kubernetes 测试 namespace 与集群登记已清理；此前为浏览器验收保留的专用记录不属于生产资源。

当前三仓 commit 与本机构建镜像 ID 保存在被忽略的 `deploy/release/manifest.local`。GitHub 最新状态查询受匿名 API 限流影响，不能把此前成功的 CI 当作最后提交的远端状态。

- 内部 MCP：实际 MCPSessionClient 列出 19 个工具，调用计算器得到 10，无密钥访问工具列表返回 401；未调用模型或外部基础设施。
- 云账号：按用户要求完成官方文档与已安装 SDK 契约核对，修复分页、ID 分批及误删风险，详见 `docs/aliyun-contract-review.md`；未连接真实云。

## 最终生产前端

最终源码 web `75f0f17` / backend `9badbe82` / AIOps `ae057fc`：同源 Docker 入口实际执行 2 个 Playwright 测试通过（46.7 秒），逐页断言 56 个业务路由实际路径，浏览器运行错误为零。登录、明暗、移动、主题刷新保持与真实 Kubernetes 四类资源页面通过；首页暗色背景已目视检查。仅允许明确未配置依赖的 `/api/v1/autofix/ready` 与 `/api/v1/rca/metrics` 返回 503，没有放行其他 500/502/503。

证据：前端 `test-results/results.json`、本机 `/tmp/cloudops-web-release-e2e.log` 和父目录 `cloudops-production-*.png`。前端类型检查、245 项单测、生产构建通过。

功能边界仍需如实保留：目前仅实现阿里云 ECS；其余云厂商、部门和自定义通知接收人不能宣称已实现。真实 SSH、模型和外部通知投递尚未验收。历史泄漏的集群凭据需由对应管理员轮换。

## 重启恢复验收

OrbStack 已恢复原 2048 MiB 内存额度；临时 MySQL/Redis 调试端口已移除，最终仅发布回环地址 18080。8 个 Compose 服务全部 healthy，kind 中两个验收 Pod 均恢复 Running/Ready。

恢复检查发现 Nginx 静态上游 IP 缓存导致 502，已按 Nginx 1.28 支持的 `resolver`、共享 `zone` 与 `server ... resolve` 修复。实测强制重建 backend/web 后，网关无需重启即可恢复；完整 smoke/分类持久化再次通过。依据：https://nginx.org/en/docs/http/ngx_http_upstream_module.html#server

## 2026-10-23 新工作区基线

以下为本轮 `~/project` 三仓克隆后的验证，前文为历史记录，不能代替本轮运行结果。

- 修复发布 Compose、启动与备份脚本中的 AIOps 仓库大小写，统一为 GitHub 实际目录名 `AI-CloudOps-aiops`，避免 Linux 下路径失效
- `codex-python -m unittest discover -s scripts/tests -v`：4 项通过，覆盖密钥权限/保密/防覆盖、缺失仓库拒绝和正确目录启动编排
- Docker 启动编排使用命令替身；本项未启动全栈，不能据此宣称运行时验收通过
- `bash -n scripts/release-start.sh`、`docker compose ... config --quiet` 及三个实际构建目录存在检查均通过
- 发布工具测试纳入分支 CI 和标签发布门槛

### 本轮监控采集回归

`GOTOOLCHAIN=auto go test ./internal/prometheus/cache` 通过。测试包含本机临时 TLS 服务的证书拒绝行为和上游 Kubernetes SD YAML 解析；没有连接真实集群或 Prometheus。当前固定 Go 1.26.6 的本机环境需使用 `GOTOOLCHAIN=auto` 执行项目指定 1.26.10。

### 本轮定时调度回归

`GOTOOLCHAIN=auto go test -race ./internal/cron/scheduler` 通过。覆盖重复重载保持单条目、变更替换、查询失败保留、删除精确注销及幂等移除。底层使用真实 Asynq 内存条目，不启动 Redis 连接或实际定时执行。

### 本轮隔离容器与 MCP

- MySQL 8.4.11、Redis 7.4.11、Prometheus 和 AIOps/MCP/前端容器实际启动为 healthy
- `docker compose -f deploy/release/compose.yaml exec -T aiops python - < scripts/integration/mcp_smoke.py` 通过：真实内网鉴权列工具、计算器结果 14、匿名工具访问 401
- 尚未配置真实模型或外部通知，不将 MCP 算术调用等同于 LLM 或 Kubernetes 工具全部验收
- 后端镜像重建遇到 Alpine 索引瞬时 I/O 失败，独立验证索引恢复可达后已重试；最终运行时结果另记

### 本轮全栈隔离验收

- 八个服务均 healthy，入口 `http://127.0.0.1:18080`
- `release-smoke.py --crud` 全通过：健康、数据库/Redis 就绪、HTML、匿名拒绝、登录、10 项业务列表、分类持久化与操作者保护、令牌刷新及注销撤销
- `account_access.py --rbac` 全通过：账号创建、普通用户限制、只读权限、立即撤权、冻结/删除撤销旧令牌，测试账号与角色清理成功
- `workorder_flow.py` 全通过：分类/表单/流程/模板/草稿/提交/审批完成、评论、流转与审计，创建对象清理成功
- 所选 `Qwen/Qwen3-30B-A3B-Instruct-2507` 经真实硅基流动请求返回预期 calculate 工具调用，单次 186 token；此项尚不等同项目内助手全链路通过
- 发布环境 `.env` 已确认被 Git 忽略，权限 0600；验证命令不输出密钥

### 新连接池与空库启动

最新后端镜像在独立临时 Compose 项目中实际启动 MySQL、Redis 与后端，新数据库完成迁移、管理员初始化和 `/ready`；测试结束删除的仅为该随机命名项目及其自有数据卷。主验收环境数据未受影响。

真实知识库新增后的查询未命中新文档，原生 MCP 也曾返回不确定工具却 code=0，均按失败记录并继续修复。RAG 测试专用文件已删除，正确的 `/api/v1/assistant/refresh` 返回 HTTP200/code0 完成索引刷新；最初清理尝试使用错误路由404，随后已纠正，不把该次404当成功。
