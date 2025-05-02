# AI+CloudOps 前端

**AI+CloudOps-web** 是 **AI+CloudOps** 平台的用户界面部分，旨在通过用户友好的界面提升运维效率和智能化水平。本项目专注于前端开发，与独立的后端服务无缝集成。

## 目录

- [AI+CloudOps 前端](#AICloudOps-前端)
  - [目录](#目录)
  - [项目介绍](#项目介绍)
  - [快速开始](#快速开始)
    - [克隆项目](#克隆项目)
    - [安装依赖](#安装依赖)
    - [配置环境变量](#配置环境变量)
    - [运行前端项目](#运行前端项目)
  - [项目结构](#项目结构)
  - [构建与部署](#构建与部署)
  - [许可证](#许可证)
  - [联系方式](#联系方式)
  - [致谢](#致谢)
  - [贡献指南](#贡献指南)

## 项目介绍

**AI+CloudOps-web** 是 **AI+CloudOps** 平台的用户界面部分，提供以下核心功能：

1. **用户与权限管理**：通过直观的界面管理用户、角色及权限，确保系统的安全和可控性。
2. **服务树与 CMDB**：可视化展示服务树结构和配置管理数据库（CMDB），实现运维资源的全面管理。
3. **工单系统**：支持工单的创建、分配、处理和追踪，提高问题解决效率。
4. **实时监控与告警**：集成 Prometheus，实时监控系统性能，结合 AI 技术进行异常预警和自动化响应。
5. **Kubernetes 管理**：提供 Kubernetes 集群的管理与监控界面，简化云端资源操作，并集成 AI 进行自动化优化和资源调度。

## 快速开始

### 克隆项目

首先，将前端项目克隆到本地：

```bash
git clone https://github.com/GoSimplicity/AI-CloudOps-Frontend.git
```

### 安装依赖

进入前端项目目录并安装依赖：

```bash
cd AI-CloudOps-Frontend/web
# 推荐使用 Node.js 21 版本
pnpm install
```

> **注意**：如果尚未安装 `pnpm`，可以通过以下命令进行安装：

```bash
npm install -g pnpm
```

### 配置环境变量

在运行前端项目之前，您需要配置后端 API 的地址。请按照以下步骤操作：

1. 在项目根目录创建 `.env` 文件（如果尚未存在）：

   ```bash
   touch .env
   ```

2. 在 `.env` 文件中添加以下内容，设置后端 API 的基地址：

   ```env
   VITE_API_BASE_URL=http://localhost:8000/api
   ```

   > **说明**：
   >
   > - `VITE_API_BASE_URL`：后端 API 的基础 URL，根据实际情况进行修改。
   > - 如果使用不同的环境（如开发、生产），可以创建对应的环境文件，例如 `.env.development` 和 `.env.production`。

### 运行前端项目

启动前端开发服务器：

```bash
pnpm run dev
```

打开浏览器访问 [http://localhost:3000](http://localhost:3000)（默认端口），即可查看运行中的前端应用。

> **提示**：确保后端服务已启动，并且前端配置的 API 地址正确指向后端。

## 项目结构

```plaintext
AI-CloudOps-Frontend/
│
├── LICENSE               # 许可证文件
├── README.md             # 项目说明文档
├── package.json          # 前端依赖和脚本
├── pnpm-lock.yaml        # pnpm 依赖锁定文件
├── .env                  # 环境变量配置
│
├── public/               # 公共资源文件
├── app/                  # 源代码目录
├── scripts/              # 各种脚本文件
└── vite.config.js        # Vite 配置文件
```

## 构建与部署

### 构建生产版本

在准备部署时，您需要构建生产版本：

```bash
pnpm run build
```

构建完成后，生成的静态文件将位于 `dist/` 目录中。

### 部署

将 `dist/` 目录中的文件部署到您的静态资源服务器或 CDN。例如，可以使用以下方法之一：

- **使用 Vercel 部署**：

  ```bash
  pnpm install -g vercel
  vercel
  ```

- **使用 Netlify 部署**：

  将 `dist/` 目录连接到 Netlify 进行自动部署。

- **手动部署**：

  将 `dist/` 目录中的文件上传到您的服务器，并配置服务器以提供静态文件。

> **注意**：确保部署后的前端应用能够正确访问后端 API。您可能需要在生产环境中更新 `.env` 文件中的 `VITE_API_BASE_URL`。

## 许可证

本项目使用 [MIT 许可证](./LICENSE)，详情请查看 LICENSE 文件。

## 联系方式

如果有任何问题或建议，欢迎通过以下方式联系我：

- Email: [wzijian62@gmail.com](mailto:wzijian62@gmail.com)
- 微信：GoSimplicity（加我后可邀请进微信群交流）

## 致谢

感谢所有为本项目贡献代码、文档和建议的人！**AI+CloudOps 前端** 的发展离不开社区的支持和贡献。

## 贡献指南

欢迎向 **AI+CloudOps 前端** 项目贡献代码！请按照以下步骤进行：

1. **Fork 本仓库**  
   点击右上角的 Fork 按钮，将仓库 Fork 到您的 GitHub 账户。

2. **创建分支**  
   为您的功能或修复创建一个新的分支：

   ```bash
   git checkout -b feature/您的功能名称
   ```

3. **提交更改**  
   进行代码更改后，提交您的更改：

   ```bash
   git commit -m "描述您的更改"
   ```

4. **推送到分支**  
   将您的分支推送到 GitHub：

   ```bash
   git push origin feature/您的功能名称
   ```

5. **创建 Pull Request**  
   在 GitHub 上创建一个 Pull Request，描述您的更改和改进。

请确保您的代码遵循项目的代码规范，并通过所有测试。

---

欢迎使用 **AI+CloudOps** 云原生运维平台，期待您的参与和贡献！
