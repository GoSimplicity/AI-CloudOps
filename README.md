# CloudOps  
云原生运维平台

## 目录
- [CloudOps](#cloudops)
  - [目录](#目录)
  - [项目介绍](#项目介绍)
  - [快速开始](#快速开始)
    - [克隆项目](#克隆项目)
    - [运行后端项目](#运行后端项目)
    - [运行前端项目](#运行前端项目)
  - [项目结构](#项目结构)
  - [许可证](#许可证)
  - [联系方式](#联系方式)
  - [致谢](#致谢)

## 项目介绍
CloudOps 是一个面向企业的云原生运维管理平台，旨在提供高效、可扩展的企业级运维解决方案。平台主要分为以下五个核心模块：
1. **用户与权限**：管理用户、角色及权限，确保系统的安全和可控性。
2. **服务树与 CMDB**：提供可视化的服务树结构和配置管理数据库（CMDB）。
3. **工单系统**：支持工单的创建、分配、处理和追踪，提高问题解决效率。
4. **Prometheus 集成**：实时监控系统性能并预警异常情况。
5. **Kubernetes 管理**：支持 Kubernetes 集群的管理与监控，简化云端资源操作。

## 快速开始

### 克隆项目
首先，将项目克隆到本地：
```bash
git clone https://github.com/GoSimplicity/CloudOps.git
```

### 运行后端项目
进入项目目录并安装依赖：
```bash
cd CloudOps
go mod tidy
```
启动后端服务：
```bash
go run cmd/cloudops/main.go
```

### 运行前端项目
进入前端目录并安装依赖：
```bash
cd web
pnpm install
```
启动前端项目：
```bash
pnpm run dev
```

## 项目结构
```text
CloudOps/
│
├── LICENSE               # 许可证文件
├── README.md             # 项目说明文档
├── Makefile              # 项目构建和管理文件
├── go.mod                # Go 模块依赖文件
├── go.sum                # Go 依赖校验文件
│
├── config/               # 配置文件目录
├── doc/                  # 项目文档目录
├── pkg/                  # 公共库和工具包
├── web/                  # 前端项目目录
│
├── cmd/                  # 可执行程序的主入口
├── deploy/               # 部署相关文件
├── internal/             # 内部模块与业务逻辑
└── scripts/              # 各种脚本文件
```

## 许可证
本项目使用 [MIT 许可证](./LICENSE)，详情请查看 LICENSE 文件。

## 联系方式
如果有任何问题或建议，欢迎通过以下方式联系我：
- Email: [wzijian62@gmail.com](mailto:wzijian62@gmail.com)
- 微信：GoSimplicity（加我后可邀请进微信群交流）

## 致谢
感谢所有为本项目贡献代码、文档和建议的人！CloudOps 的发展离不开社区的支持和贡献。

---

欢迎加入 CloudOps 云原生运维平台，期待你的参与和贡献！
