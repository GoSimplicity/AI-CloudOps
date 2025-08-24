# AI+CloudOps: AI 驱动的云原生运维平台

<p align="center">
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8.svg?style=flat-square&logo=go" alt="Go Version"></a>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="License"></a>
    <a href="https://github.com/GoSimplicity/AI-CloudOps/stargazers"><img src="https://img.shields.io/github/stars/GoSimplicity/AI-CloudOps?style=flat-square&logo=github" alt="GitHub Stars"></a>
    <a href="https://github.com/GoSimplicity/AI-CloudOps/network"><img src="https://img.shields.io/github/forks/GoSimplicity/AI-CloudOps?style=flat-square&logo=github" alt="GitHub Forks"></a>
    <a href="https://github.com/GoSimplicity/AI-CloudOps/issues"><img src="https://img.shields.io/github/issues/GoSimplicity/AI-CloudOps?style=flat-square&logo=github" alt="GitHub Issues"></a>
</p>

![AI-CloudOps](https://socialify.git.ci/GoSimplicity/AI-CloudOps/image?description=1&font=Inter&forks=1&issues=1&name=1&pattern=Solid&pulls=1&stargazers=1&theme=Dark)

---

## 📖 项目介绍 (Introduction)

**AI+CloudOps** 是一个专为企业设计的、由 AI 驱动的云原生运维管理平台。我们的目标是融合人工智能技术与云原生实践，显著提升运维工作的效率、自动化和智能化水平。

- **后端仓库**: [GoSimplicity/AI-CloudOps](https://github.com/GoSimplicity/AI-CloudOps)
- **前端仓库**: [GoSimplicity/AI-CloudOps-web](https://github.com/GoSimplicity/AI-CloudOps-web)
- **AIOps 模块**: [GoSimplicity/AI-CloudOps-aiops](https://github.com/GoSimplicity/AI-CloudOps-aiops)

## ✨ 主要功能 (Features)

- **智能 AIOps**: 通过机器学习分析监控数据和日志，提供智能告警、故障预测及根因分析。
- **多维度权限管理**: 精细化的用户、角色、权限控制，保障系统和资源安全。
- **可视化 CMDB**: 以服务树的形式直观展示和管理所有运维资源。
- **高效工单系统**: 全生命周期追踪工单，从创建、分配到解决，流程清晰，提升协作效率。
- **深度集成 Prometheus**: 实时监控系统性能，并结合 AI 实现异常的智能预警和自动化响应。
- **一体化 Kubernetes 管理**: 简化 K8s 集群的日常管理和监控，利用 AI 实现自动化资源调度和优化。

## 预览地址

- 项目地址: [http://68.64.177.180](http://68.64.177.180)
- 账号：demo
- 密码：Demo@2025

## 📸 项目演示 (Screenshots)

|              登录页              |             API 管理             |
| :------------------------------: | :------------------------------: |
|      ![登录页](image/1.png)      |     ![API管理](image/2.png)      |
|           **表单设计**           |           **流程管理**           |
|     ![表单设计](image/3.png)     |     ![流程管理](image/4.png)     |
|        **服务树节点概览**        |           **根因分析**           |
|  ![服务树节点概览](image/5.png)  |    ![根因分析](image/10.png)     |
|       **k8s 故障自动修复**       |       **k8s 故障自动修复**       |
| ![k8s故障自动修复](image/12.png) | ![k8s故障自动修复](image/13.png) |

## 🚀 快速开始 (Quick Start)

### 1. 环境准备 (Prerequisites)

请确保您的开发环境中已安装以下软件：

- Go `1.21+`
- Node.js `21.x`
- pnpm `latest`
- Docker & Docker Compose
- Python `3.11.x`

### 2. 克隆项目 (Clone Repositories)

您需要分别克隆后端和前端项目：

```bash
# 克隆后端项目
git clone https://github.com/GoSimplicity/AI-CloudOps.git

# 克隆前端项目
git clone https://github.com/GoSimplicity/AI-CloudOps-web.git

# 克隆 AIOps 项目
git clone https://github.com/GoSimplicity/AI-CloudOps-aiops.git
```

### 3. 开发模式 (Development Mode)

**步骤一：启动依赖服务**

```bash
# 进入后端项目目录
cd AI-CloudOps

# 使用 Docker Compose 启动 MySQL, Redis 等中间件
docker-compose -f docker-compose-env.yaml up -d

# 复制并配置环境变量
cp env.example .env
```

**步骤二：启动前端服务**

```bash
# 进入前端项目目录
cd ../AI-CloudOps-web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm run dev
```

> 默认访问地址: `http://localhost:3000`

**步骤三：启动后端服务**

```bash
# 回到后端项目目录
cd ../AI-CloudOps

# 安装 Go 依赖
go mod tidy

# 启动后端主服务
go run main.go
```

> 默认服务地址: `http://localhost:8000`

**步骤四：启动 AIOps 服务 (可选)**

```bash
# 进入 AIOps 项目目录
cd ../AI-CloudOps-aiops

# 配置环境变量
cp env.example .env

# 安装依赖
pip install -r requirements.txt

# 训练模型 (如果需要)
cd data/ && python machine-learning.py && cd ..

# 启动服务
python app/main.py
```

### 4. 生产模式 (Production Mode)

**步骤一：构建前端静态资源**

```bash
# 进入前端项目目录
cd AI-CloudOps-web

# 安装依赖并构建
pnpm install
pnpm run build
```

构建产物位于 `dist/` 目录，请将其部署到 Nginx 或其他 Web 服务器。

**步骤二：构建并运行后端服务**

```bash
# 回到后端项目目录
cd AI-CloudOps

# 构建二进制文件
go build -o bin/ai-cloudops main.go

# 运行生产服务
./bin/ai-cloudops
```

**步骤三 (推荐)：使用 Docker Compose 部署**

我们强烈推荐使用 Docker Compose 来部署整个应用，这能简化流程并保证环境一致性。

```bash
# 在 AI-CloudOps 项目根目录
# 确保您的 docker-compose.yaml 已配置好前端镜像和后端服务

# 启动所有服务
docker-compose up -d
```

## 🏗️ 项目结构 (Project Structure)

### 后端 (AI-CloudOps)

```text
AI-CloudOps/
├── cmd/                  # 可执行程序的主入口
├── config/               # 配置文件目录
├── deploy/               # 部署相关文件 (K8s, Docker)
├── internal/             # 内部模块与业务逻辑
├── main.go               # 主程序入口
├── Makefile              # 项目构建和管理文件
└── go.mod                # Go 模块依赖
```

### 前端 (AI-CloudOps-web)

```text
AI-CloudOps-web/
├── apps/
│   └── web-antd/         # 基于 Ant Design 的主应用
├── packages/             # 共享组件和工具库 (monorepo)
├── package.json          # Node.js 依赖
├── pnpm-workspace.yaml   # pnpm workspace 配置
└── turbo.json            # Turborepo 配置
```

### AIOps (AI-CloudOps-aiops)

```text
AI-CloudOps-aiops/
├── app/                  # 主应用代码
├── config/               # 配置文件
├── data/                 # 数据和模型训练脚本
├── deploy/               # 部署相关文件
├── requirements.txt      # Python 依赖
└── Dockerfile            # Docker 构建文件
```

## 🤝 贡献指南 (Contributing)

我们非常欢迎来自社区的任何贡献！无论是提交 Bug、建议新功能，还是直接贡献代码。

1.  **Fork** 本仓库
2.  创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3.  提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4.  推送到分支 (`git push origin feature/AmazingFeature`)
5.  发起一个 **Pull Request**

## 📄 许可证 (License)

本项目基于 [MIT License](./LICENSE) 开源。

## 📞 联系我们 (Contact)

- **Email**: [bamboocloudops@gmail.com](mailto:bamboocloudops@gmail.com)
- **微信 (WeChat)**: `GoSimplicity` (添加时请备注 "AI-CloudOps"，我会邀请您加入交流群)
- ![image](https://github.com/user-attachments/assets/75c84edc-7a12-4ce0-bbce-8ccbbc84a83e)



## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=GoSimplicity/AI-CloudOps&type=Date)](https://star-history.com/#GoSimplicity/AI-CloudOps&Date)

## 🙏 致谢 (Acknowledgements)

感谢所有为 AI-CloudOps 做出贡献的开发者和用户。正是因为你们，这个项目才能不断进步。
