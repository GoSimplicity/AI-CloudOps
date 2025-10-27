# AI-CloudOps

> **[English](README.en.md) | 中文**

一个基于人工智能的云原生运维管理平台，集成 Kubernetes 管理、监控告警、智能分析等功能。

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.0+-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.25+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Stars](https://img.shields.io/github/stars/GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/stargazers)

![AI-CloudOps](https://socialify.git.ci/GoSimplicity/AI-CloudOps/image?description=1&font=Inter&forks=1&issues=1&name=1&pattern=Solid&pulls=1&stargazers=1&theme=Dark)

## 项目介绍

AI-CloudOps 是一个现代化的云原生运维管理平台，旨在通过人工智能技术提升运维效率和系统稳定性。

### 核心特性

- **智能运维**: 基于机器学习的异常检测、故障预测和自动修复
- **Kubernetes 管理**: 完整的集群管理、应用部署和监控功能
- **监控告警**: 基于 Prometheus 的全方位监控和智能告警
- **权限管理**: 基于 RBAC 的多租户权限控制系统
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

- 多租户架构：支持多组织隔离
- 角色管理：灵活的角色权限配置
- 细粒度权限：操作级别的权限控制
- 审计日志：完整的操作审计记录

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

**演示地址**: <http://68.64.177.180>

**登录信息**:

- 用户名: `demo`
- 密码: `Demo@2025`

*注意：演示环境仅供测试使用，请勿上传敏感信息*

## 核心贡献者

感谢以下开发者对项目的重要贡献：

- **[GoSimplicity](https://github.com/GoSimplicity)** - 项目发起人和核心维护者
- **[Penge666](https://github.com/Penge666)** - 资深开发者
- **[shixiaocaia](https://github.com/shixiaocaia)** - 核心贡献者
- **[daihao4371](https://github.com/daihao4371)** - 功能开发者

[![Contributors](https://contrib.rocks/image?repo=GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/graphs/contributors)

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 21.x
- pnpm latest
- Docker & Docker Compose
- Python 3.11+ (AI 模块)

### 环境检查

```bash
go version          # >= 1.21
node --version      # >= 21.0
pnpm --version      # latest
docker --version    # latest
python3 --version   # >= 3.11
```

### 获取代码

```bash
# 克隆后端项目
git clone https://github.com/GoSimplicity/AI-CloudOps.git
cd AI-CloudOps

# 克隆前端项目
git clone https://github.com/GoSimplicity/AI-CloudOps-web.git

# 克隆 AI 模块项目
git clone https://github.com/GoSimplicity/AI-CloudOps-aiops.git
```

### 开发环境启动

#### 1. 启动基础服务

```bash
cd AI-CloudOps
# 启动数据库和中间件
docker-compose -f docker-compose-env.yaml up -d

# 配置环境变量
cp env.example .env

# 检查服务状态
docker-compose -f docker-compose-env.yaml ps
```

#### 2. 启动前端服务

```bash
cd AI-CloudOps-web
# 安装依赖
pnpm install

# 启动开发服务器
pnpm run dev
```

前端将在 [http://localhost:3000](http://localhost:3000) 启动

#### 3. 启动后端服务

```bash
cd AI-CloudOps
# 安装依赖
go mod tidy

# 启动后端服务
go run main.go
```

后端服务地址：

- API 服务: [http://localhost:8000](http://localhost:8000)
- Swagger 文档: [http://localhost:8000/swagger](http://localhost:8000/swagger)

#### 4. 启动 AI 服务 (可选)

```bash
cd AI-CloudOps-aiops
# 配置环境变量
cp env.example .env

# 安装依赖
pip install -r requirements.txt

# 训练初始模型
cd data/ && python machine-learning.py && cd ..

# 启动 AI 服务
python app/main.py
```

AI 服务地址: [http://localhost:8001](http://localhost:8001)

## 生产部署

### Docker Compose 部署 (推荐)

```bash
cd AI-CloudOps
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### Kubernetes 部署

```bash
cd deploy/kubernetes/
# 配置环境变量
cp config.example config

# 部署到集群
kubectl apply -f .

# 查看部署状态
kubectl get pods,svc,ingress -l app=ai-cloudops
```

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
