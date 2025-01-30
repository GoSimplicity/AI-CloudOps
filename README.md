# AI+CloudOps

AI 驱动的云原生运维平台

## 目录

- [AI+CloudOps](#AICloudOps)
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

AI+CloudOps 是一个面向企业的 AI 驱动云原生运维管理平台，旨在通过人工智能技术提升运维效率和智能化水平。平台包含以下核心模块：

1. **AIOps 模块**：通过机器学习和 AI 技术，分析系统日志、监控数据，提供智能告警、故障预测和根因分析。
2. **用户与权限**：管理用户、角色及权限，确保系统的安全和可控性。
3. **服务树与 CMDB**：提供可视化的服务树结构和配置管理数据库（CMDB），实现运维资源的全面管理。
4. **工单系统**：支持工单的创建、分配、处理和追踪，提高问题解决效率。
5. **Prometheus 集成**：实时监控系统性能，结合 AI 技术，进行异常预警和自动化响应。
6. **Kubernetes 管理**：支持 Kubernetes 集群的管理与监控，简化云端资源操作，集成 AI 进行自动化优化和资源调度。

## 项目演示

![image 1](image/1.png)
![image 2](image/2.png)
![image 3](image/3.png)
![image 4](image/4.png)
![image 5](image/5.png)
![image 6](image/6.png)
![image 7](image/7.png)
![image 8](image/8.png)

## 快速开始

### 克隆项目

首先，将项目克隆到本地：

```bash
git clone https://github.com/GoSimplicity/AI-CloudOps.git
```

### 运行后端项目

进入项目目录并安装依赖：

```bash
go mod tidy
```

启动后端服务：

```bash
go run cmd/cloudops/main.go
```

### 运行前端项目

前端项目地址：<https://github.com/GoSimplicity/AI-CloudOps-web>

```bash
# clone前端项目
git clone https://github.com/GoSimplicity/AI-CloudOps-web.git
```

进入前端目录并安装依赖：

```bash
# 进入项目根目录
cd AI-CloudOps-web
# 推荐使用 node21 版本
pnpm install
```

启动前端项目：

```bash
pnpm run dev
```

## 项目结构

```text
AI-CloudOps/
│
├── LICENSE               # 许可证文件
├── README.md             # 项目说明文档
├── Makefile              # 项目构建和管理文件
├── go.mod                # Go 模块依赖文件
├── go.sum                # Go 依赖校验文件
├── config/               # 配置文件目录
├── doc/                  # 项目文档目录
├── pkg/                  # 公共库和工具包
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

感谢所有为本项目贡献代码、文档和建议的人！AI+CloudOps 的发展离不开社区的支持和贡献。

---

欢迎加入 AI+CloudOps 云原生运维平台，期待你的参与和贡献！