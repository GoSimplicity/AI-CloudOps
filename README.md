# AI-CloudOps

> **[English](README.en.md) | 中文**

一个基于人工智能的云原生运维管理平台，集成 Kubernetes 管理、监控告警、智能分析等功能。

[![Go](https://img.shields.io/badge/Go-1.26.10-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.0+-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.25+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Stars](https://img.shields.io/github/stars/GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/stargazers)

![AI-CloudOps](https://socialify.git.ci/GoSimplicity/AI-CloudOps/image?description=1&font=Inter&forks=1&issues=1&name=1&pattern=Solid&pulls=1&stargazers=1&theme=Dark)

## 1.0 交付入口

[验收记录](docs/release/verification.md) · [固定源码与镜像清单](docs/release/release-candidate.json) · [部署与升级](docs/release/README.md) · [阶段变更](docs/release/CHANGELOG.md)

三仓放在同级目录。1.0 的云资源支持阿里 ECS，工单自动通知支持即时触发；其它提供商和高级触发明确关闭。工作区使用 API 级 RBAC，模型、集群、SSH 和通知须配置各自依赖，具体实测与边界见验收记录。

## 项目介绍

AI-CloudOps 是一个现代化的云原生运维管理平台，旨在通过人工智能技术提升运维效率和系统稳定性。

### 核心特性

- **智能运维**: 基于机器学习的异常检测、故障预测和自动修复
- **Kubernetes 管理**: 完整的集群管理、应用部署和监控功能
- **监控告警**: 基于 Prometheus 的全方位监控和智能告警
- **权限管理**: 共享运维工作区内基于 RBAC 的接口权限控制
- **工单系统**: 完整的运维工单流程管理
- **资源管理**: CMDB 资产管理和服务拓扑可视化

### 技术架构

- **后端**: Go + Gin + GORM + Redis + MySQL
- **前端**: Vue 3 + TypeScript + Ant Design Vue
- **AI模块**: Python + FastAPI + scikit-learn
- **基础设施**: Kubernetes + Prometheus + Grafana

### 项目组成

本项目由三个主要仓库组成：

- **[AI-CloudOps](https://github.com/GoSimplicity/AI-CloudOps)** - 核心后端服务
- **[AI-CloudOps-web](https://github.com/GoSimplicity/AI-CloudOps-web)** - 前端界面
- **[AI-CloudOps-aiops](https://github.com/GoSimplicity/AI-CloudOps-aiops)** - AI 智能分析模块

## 功能模块

### 智能运维 (AIOps)

- 智能监控：基于机器学习的监控数据分析
- 异常检测：自动识别系统异常和性能问题  
- 故障预测：预测性维护和问题预警
- 自动修复：基于规则和 AI 的智能修复
- 根因分析：快速定位故障根本原因

### 权限管理 (RBAC)

- 工作区边界：当前版本面向单个组织的共享运维工作区，业务资源按接口权限授权；创建者字段不构成租户或资源行隔离
- 角色管理：灵活的角色权限配置
- 细粒度权限：操作级别的权限控制
- 审计日志：完整的操作审计记录

AI 助手会话另按已登录用户隔离，内部服务身份拥有独立会话空间。需要互不信任的多组织使用时，应为各组织配置独立部署、数据库、缓存及凭据；当前版本不提供同一部署中的多组织强隔离。监控配置详情权限包含抓取认证配置，授予此权限等同授予该工作区监控凭据读取能力，不能作为普通业务只读权限下发。

### 资源管理 (CMDB)

- 服务树：可视化的资源拓扑结构
- 资产发现：自动化资源发现和管理
- 依赖关系：服务依赖图谱管理
- 标签管理：灵活的资源分类标记

### 工单系统

- 工单模板：可定制的工单类型
- 流程引擎：自动化审批工作流
- SLA 管理：服务等级协议监控
- 协作工具：内置沟通和协作功能

### 监控告警

- 多维监控：全方位系统性能监控
- 实时告警：毫秒级告警响应
- 智能阈值：AI 动态阈值调整
- 告警收敛：智能告警聚合和降噪

### Kubernetes 管理

- 集群管理：多集群统一管理
- 应用部署：可视化应用部署和更新
- 资源监控：Pod、Node、Service 监控
- 配置管理：ConfigMap 和 Secret 管理

## 在线演示

公开演示入口不作为本次发布验收依据。自建环境使用启动时生成的独立管理员密码，不共享演示账号或固定默认密码，详见[部署与验收](docs/release/README.md)。

## 核心贡献者

感谢以下开发者对项目的重要贡献：

- **[GoSimplicity](https://github.com/GoSimplicity)** - 项目发起人和核心维护者
- **[Penge666](https://github.com/Penge666)** - 资深开发者
- **[shixiaocaia](https://github.com/shixiaocaia)** - 核心贡献者
- **[daihao4371](https://github.com/daihao4371)** - 功能开发者

[![Contributors](https://contrib.rocks/image?repo=GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/graphs/contributors)

## 快速开始

三个仓库必须同级放置，目录名区分大小写。以下为已实际验证的 Docker Compose 入口，详细配置、备份与验收见 [1.0 部署指南](docs/release/README.md)。

### 获取代码与启动

需要 Git、Docker 和 Docker Compose v2，以及用于生成私有配置的 Python 3。镜像内使用项目固定的 Go 1.26.10、Node 22 与 Python 3.11，无需为容器部署在宿主机安装这些语言运行时。

```sh
mkdir -p ~/project
cd ~/project
git clone https://github.com/GoSimplicity/AI-CloudOps.git
git clone https://github.com/GoSimplicity/AI-CloudOps-web.git
git clone https://github.com/GoSimplicity/AI-CloudOps-aiops.git
cd AI-CloudOps
bash scripts/release-start.sh
```

入口为 <http://localhost:18080>。脚本检查同级仓库、生成私有配置、构建并等待服务健康。首次管理员为 `admin`，密码来自权限为 `0600`、已被 Git 忽略的 `deploy/release/.env` 中 `BOOTSTRAP_ADMIN_PASSWORD`。已有数据库不会重置管理员。

```sh
python3 scripts/release-smoke.py --crud
docker compose -f deploy/release/compose.yaml ps
```

`--crud` 会创建并清理唯一命名的测试分类，保留真实审计记录，只在隔离验收环境执行。模型、Kubernetes、SSH 和通知分别配置与验收；基础健康检查成功不代表这些能力全部可用。

### 源码开发

本机后端使用 `go.mod` 指定的工具链。所有环境（包括开发环境）启动时都会校验 JWT 密钥，且不提供放宽开关：
`JWT_KEY1`、`JWT_KEY2` 各自至少32字符、不同字符至少10个、首尾不能有空白且不能含控制字符（不会自动去除），
两者不得相同（忽略大小写和首尾空白比较），也不能使用或包含仓库示例值、历史默认值（`env.example` 中的占位符会被拒绝，大小写变化同样拒绝）。
报错信息只给出变量名和原因，不会输出密钥内容。

**第一次启动必做**：分别生成两把独立密钥写入 `.env`，否则后端拒绝启动：

```sh
openssl rand -base64 48   # 作为 JWT_KEY1
openssl rand -base64 48   # 再运行一次，作为 JWT_KEY2
```


```sh
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go build .
```

前端安装、开发端口、类型检查与构建以 [前端 README](https://github.com/GoSimplicity/AI-CloudOps-web#readme) 为准；AIOps 的环境与验证方式见 [生产配置说明](https://github.com/GoSimplicity/AI-CloudOps-aiops/blob/main/docs/production.md)。开发进程连接容器依赖时，可使用部署指南中的只绑定回环地址的 `local-ports.yaml`，不要直接公开数据库端口。

## 生产部署

默认编排只在回环地址发布 `18080`。对外部署应配置 TLS 反向代理与实际 `PUBLIC_ORIGIN`，并按部署指南设置备份、密钥、数据源与资源预算。不要使用未经验收的历史 Compose 示例替代 `deploy/release/compose.yaml`。

- [部署、可选 SSH 信任配置与回滚边界](docs/release/README.md)
- [分阶段交付计划与测试矩阵](docs/release/plan-1.0.md)
- [本轮及历史验证记录](docs/release/verification.md)
- [逐阶段变更日志](docs/release/CHANGELOG.md)

当前交付仍在持续验证，完成发布门槛前不把某次单测或镜像构建标记为最终 1.0 生产验收。

## 项目架构

### 后端结构

```text
AI-CloudOps/
├── cmd/                    # 命令行工具
├── config/                 # 配置文件
├── internal/               # 核心业务逻辑
│   ├── middleware/         # 中间件
│   ├── model/              # 数据模型
│   ├── k8s/                # Kubernetes 管理
│   ├── user/               # 用户管理
│   ├── prometheus/         # 监控模块
│   ├── workorder/          # 工单系统
│   ├── tree/               # 服务树 CMDB
│   └── system/             # 系统管理
├── pkg/                    # 公共包
├── docs/                   # API 文档
└── deploy/                 # 部署配置
```

### 前端结构

```text
AI-CloudOps-web/
├── apps/web-antd/          # 主应用
│   ├── src/
│   │   ├── api/            # API 接口
│   │   ├── components/     # 组件
│   │   ├── views/          # 页面
│   │   ├── router/         # 路由
│   │   └── store/          # 状态管理
│   └── dist/               # 构建产物
└── packages/               # 共享包
```

### AI 模块结构

```text
AI-CloudOps-aiops/
├── app/                    # 应用代码
│   ├── api/                # API 路由
│   ├── core/               # 核心逻辑
│   ├── models/             # 数据模型
│   └── services/           # 业务服务
├── data/                   # 数据存储
├── config/                 # 配置文件
└── tests/                  # 测试用例
```

## 贡献指南

### 贡献流程

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```text
<type>(<scope>): <description>

[optional body]
[optional footer]
```

**提交类型**:

- `feat`: 新功能
- `fix`: Bug 修复  
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 代码重构
- `test`: 测试相关
- `ci`: CI/CD 相关

### 代码规范

- 遵循项目现有代码风格
- 添加必要的注释和文档
- 确保测试覆盖率
- 遵循最佳实践

## 开源许可

本项目基于 [MIT License](LICENSE) 开源。

## 联系我们

- **邮箱**: <bamboocloudops@gmail.com>
- **微信**: GoSimplicity (添加请备注 "AI-CloudOps")
- **GitHub**: [提交 Issue](https://github.com/GoSimplicity/AI-CloudOps/issues)

### 微信交流群

![微信群二维码](![image](https://github.com/user-attachments/assets/c6112b5d-0333-4f3f-8359-f9f2b1916b72))

## 致谢

感谢所有为项目做出贡献的开发者和用户，以及以下开源项目：

- [Go](https://golang.org/) - 高性能后端语言
- [Vue.js](https://vuejs.org/) - 渐进式前端框架  
- [Kubernetes](https://kubernetes.io/) - 容器编排平台
- [Prometheus](https://prometheus.io/) - 监控系统
- [Ant Design Vue](https://antdv.com/) - 企业级 UI 组件库

---

**如果觉得项目对您有帮助，请给我们一个 Star ⭐**

[![Star History Chart](https://api.star-history.com/svg?repos=GoSimplicity/AI-CloudOps&type=Date)](https://star-history.com/#GoSimplicity/AI-CloudOps&Date)

### 生产启动安全要求

生产环境设置 `ENV=production`、`MOCK_ENABLED=false`，并分别注入独立的
`JWT_KEY1`、`JWT_KEY2`、`TREE_PASSWORD_ENCRYPTION_KEY`（至少32字符，不能使用仓库示例值）。
其中 JWT 两把密钥的长度、熵（不同字符至少10个）、首尾空白/控制字符、互不相同和示例值检查在所有环境都执行，不仅限生产环境。
首次空数据库启动使用 `BOOTSTRAP_ADMIN_PASSWORD`（至少12字符）创建管理员，成功后移除该变量；已有用户不会被重置。
`SERVER_ALLOWED_ORIGINS` 是逗号分隔的精确 Origin 白名单，留空仅支持同源浏览器访问。
`JWT_EXPIRATION` 是访问令牌有效期，单位是**分钟**（不是秒）；默认值和发布配置均为 `30`（30 分钟）。
请勿按秒填写 `3600` 之类的值，否则令牌会有效 60 小时。访问令牌过期后，前端会用刷新令牌（有效期 7 天）自动续期。
`jwt.issuer`（可用 `JWT_ISSUER` 覆盖）不能为空，否则后端启动失败。
**升级提示**：升级到要求签发者（iss）的版本后，之前签发的、不带 iss 的旧刷新令牌全部失效，所有用户需要重新登录一次；之后正常续期。

定时任务的命令行（类型 2）/脚本（类型 4）任务由 `cron.allow_local_exec`（环境变量 `CRON_ALLOW_LOCAL_EXEC`）控制，默认关闭。
开关在后端启动时读取，修改后需重启后端。关闭开关只会阻止之后的创建、启用、调度和执行，
**不会终止已经在运行的命令/脚本进程**（包括它们派生的后台进程）；关闭后请在后端主机上确认并手动结束残留进程。
自定义环境变量（`environment` 字段）只对命令行任务（类型 2）生效，也只对这一类在保存和执行时校验；HTTP、SSH、脚本任务不使用该字段，不会因其中内容被拦截（SSH 任务使用独立的 `ssh_environment`）。
命令行任务的环境变量规则：
- 变量名和值都为空的行（前端点“添加”后未填写的空行）在保存前丢弃，执行时跳过，不会报错；
- 变量名为空但值非空，或变量名包含 `=`、NUL 字符时拒绝保存和执行；
- 以下变量名不允许设置（不区分大小写）：前缀 `LD_*`、`DYLD_*`、`BASH_FUNC_*`、`MALLOC_*`，以及 `BASH_ENV`、`ENV`、`IFS`、`SHELLOPTS`、`BASHOPTS`、`PS4`、`GCONV_PATH`、`GLIBC_TUNABLES`、`LOCPATH`；
- 报错只包含变量名，不包含变量值。数据库中已有的不合规命令行任务会在执行时被拒绝，需要编辑后重新保存。

AI 请求通过 Go 的 `/api/v1/*` 网关接受 JWT 和角色权限检查，然后向
`AIOPS_BASE_URL` 注入 `AIOPS_API_KEY`。该密钥必须与 Python 服务一致（生产至少32字符），不得进入前端配置。
非管理员需在角色 API 权限中授权相应 `/api/v1/` 路径；服务账号不再自动绕过角色权限。
`GET /health` 提供进程存活检查，不代表数据库、Redis或外部云服务均已就绪。

生产 SSH 终端必须将经运维人员核验的 OpenSSH `known_hosts` 文件挂载到服务容器，
设置 `SSH_KNOWN_HOSTS` 为该文件路径；缺失或主机密钥不匹配时拒绝连接。
`TREE_PASSWORD_ENCRYPTION_KEY` 必须恰好32字节。WebSocket 使用同源或
`SERVER_ALLOWED_ORIGINS` 精确白名单校验；审计仅保存操作元数据，不保存请求/响应正文中的凭据。

通知配置支持创建人、处理人、指定用户、角色成员，并自动去重。角色仅解析启用角色中的
启用用户；部门/自定义接收人尚缺组织或地址配置，返回明确错误，不会假报发送成功。
测试通知必须填写实际接收地址；渠道/接收人发送失败会返回错误。

安全维护基线使用 Go 1.26.10、Prometheus 0.311.3 和 Kubernetes Go SDK 0.35.3。
开发环境设置 `GOTOOLCHAIN=auto` 可按 `go.mod` 下载所需 Go 工具链。
CI 在每次提交运行 `govulncheck`；只有显式推送 `v*` 标签才生成发行附件，
附件通过测试、网关冒烟、静态检查和可达漏洞检查，并提供 SHA256 校验和。

每次受保护请求及刷新令牌均查询当前账号：冻结、删除、注销或用户名变更后，旧令牌立即
被拒绝；账号类型以数据库当前值为准，角色权限仍逐请求查询。数据库不可用时拒绝认证。
内置 `admin` 为保留的管理账号；撤销其管理权限应冻结或删除该账号，而不是只移除角色。

账号由管理员分配：`POST /api/user/signup` 保留原有请求格式，但现在必须携带有效的
`admin` 会话；匿名请求返回401，普通/服务账号返回403。新用户默认普通账号，不能通过
请求体自行获得服务账号权限。登录和刷新令牌仍为公开入口，刷新时校验会话和账号状态。

权限目录在每次启动时从已注册的 Gin 路由幂等初始化，与 `MOCK_ENABLED` 无关。
已有相同方法/路径的自定义名称和描述不会被覆盖，也不会自动向任何角色授予权限。
管理员可以创建只读角色并选择具体 GET 接口；含 `:id` 的目录项按注册路由匹配，
不会因为路径参数展开而失效。账号权限与完整工单流转的真实 HTTP 验收脚本位于
`scripts/integration/`，凭据仅从 `BOOTSTRAP_ADMIN_PASSWORD` 环境变量读取。
