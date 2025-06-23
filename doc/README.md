# AI-CloudOps 项目文档

## 项目概述

AI-CloudOps 是一个基于人工智能的云运维管理平台，支持多云资源管理、自动化运维、智能监控等功能。

**最后更新**: 2025年6月23日

## 最新进展

### 🎯 华为云提供商实现完成 (2025-06-23)

华为云提供商（`HuaweiProviderImpl`）已全面实现并优化完成：

#### ✅ 核心功能实现
- **ECS实例管理**: 完整的生命周期管理（创建、删除、启动、停止、重启）
- **VPC网络管理**: VPC和子网的创建、删除、查询
- **安全组管理**: 安全组的创建、删除、查询
- **磁盘管理**: 磁盘的创建、删除、挂载、卸载
- **资源同步**: 并发同步多种资源类型
- **区域发现**: 动态区域发现和智能探测

#### ✅ 优化成果
- **彻底移除硬编码**: 所有静态数据改为动态获取
- **SDK优先策略**: 优先通过华为云SDK动态获取数据
- **三层fallback机制**: SDK → 配置 → 静态兜底
- **并发处理**: 支持并发资源同步和区域探测
- **缓存优化**: 6小时缓存，提高性能
- **错误处理**: 完善的错误处理和降级机制

#### ✅ 测试验证
- **编译测试**: ✅ 成功
- **单元测试**: ✅ 9个测试全部通过
- **竞态检测**: ✅ 无并发安全问题
- **代码质量**: ✅ go vet检查通过

## 项目结构

```
AI-CloudOps/
├── cmd/                    # 命令行工具
├── config/                 # 配置文件
├── deploy/                 # 部署配置
├── dify/                   # Dify AI平台集成
├── doc/                    # 项目文档
├── internal/               # 内部包
│   ├── ai/                # AI相关功能
│   ├── k8s/               # Kubernetes管理
│   ├── prometheus/        # 监控系统
│   ├── tree/              # 多云资源管理
│   │   └── provider/      # 云提供商实现
│   │       ├── aliyun_provider.go     # 阿里云提供商
│   │       ├── huawei_provider.go     # 华为云提供商 ✅
│   │       ├── huawei_provider_test.go # 华为云测试 ✅
│   │       └── factory.go             # 工厂模式
│   ├── user/              # 用户管理
│   └── workorder/         # 工单系统
├── pkg/                   # 公共包
│   ├── aliyun/            # 阿里云SDK
│   ├── huawei/            # 华为云SDK
│   └── utils/             # 工具函数
├── python/                # Python服务
├── test/                  # 测试文件
└── ui/                    # 前端界面
```

## 核心功能

### 1. 多云资源管理

#### 支持的云提供商
- ✅ **阿里云**: 完整的资源管理支持
- ✅ **华为云**: 完整的资源管理支持（新增）

#### 资源类型支持
- **ECS实例**: 创建、删除、启动、停止、重启
- **VPC网络**: VPC和子网管理
- **安全组**: 安全策略管理
- **磁盘**: 存储管理
- **负载均衡**: 负载均衡器管理
- **数据库**: 数据库实例管理

### 2. 智能监控

#### Prometheus集成
- **指标收集**: 自动收集云资源指标
- **告警规则**: 智能告警规则管理
- **事件处理**: 告警事件处理
- **值班管理**: 值班人员管理

#### 监控功能
- **资源监控**: 实时监控云资源状态
- **性能分析**: 性能指标分析
- **容量规划**: 资源容量规划
- **成本优化**: 成本分析和优化建议

### 3. AI辅助运维

#### Dify AI平台集成
- **智能问答**: 基于AI的运维问答
- **自动化建议**: 智能运维建议
- **故障诊断**: AI辅助故障诊断
- **知识库**: 运维知识库管理

#### AI功能
- **预测分析**: 资源使用预测
- **异常检测**: 智能异常检测
- **优化建议**: 性能优化建议
- **自动化脚本**: 智能脚本生成

### 4. 工单系统

#### 工单管理
- **工单创建**: 支持多种工单类型
- **流程管理**: 可配置的工单流程
- **状态跟踪**: 实时状态跟踪
- **统计分析**: 工单统计分析

#### 表单设计
- **动态表单**: 可配置的表单设计
- **字段验证**: 智能字段验证
- **附件管理**: 文件附件管理
- **审批流程**: 灵活的审批流程

## 技术架构

### 1. 后端架构

#### 技术栈
- **Go 1.24.3**: 主要开发语言
- **Gin**: Web框架
- **GORM**: ORM框架
- **Redis**: 缓存和会话管理
- **MySQL**: 主数据库
- **Prometheus**: 监控系统

#### 架构特点
- **微服务架构**: 模块化设计
- **依赖注入**: 松耦合设计
- **中间件**: 可扩展的中间件系统
- **配置管理**: 灵活的配置管理

### 2. 前端架构

#### 技术栈
- **React**: 前端框架
- **TypeScript**: 类型安全
- **Ant Design**: UI组件库
- **Vite**: 构建工具
- **Tailwind CSS**: 样式框架

#### 架构特点
- **组件化**: 可复用组件设计
- **状态管理**: 集中状态管理
- **路由管理**: 动态路由管理
- **主题系统**: 可配置主题

### 3. 部署架构

#### 容器化部署
- **Docker**: 容器化部署
- **Kubernetes**: 容器编排
- **Helm**: 包管理工具
- **Nginx**: 反向代理

#### 监控和日志
- **Prometheus**: 指标监控
- **Grafana**: 可视化面板
- **ELK Stack**: 日志管理
- **Jaeger**: 链路追踪

## 快速开始

### 1. 环境要求

```bash
# Go版本要求
go version >= 1.24.3

# Node.js版本要求
node version >= 18.0.0

# Docker版本要求
docker version >= 20.10.0
```

### 2. 安装依赖

```bash
# 后端依赖
go mod download

# 前端依赖
cd ui && npm install
```

### 3. 配置环境

```bash
# 复制配置文件
cp config/config.example.yaml config/config.yaml

# 配置数据库
cp deploy/init.sql.example deploy/init.sql
```

### 4. 启动服务

```bash
# 启动后端服务
go run cmd/main.go

# 启动前端服务
cd ui && npm run dev
```

## 配置说明

### 1. 云提供商配置

#### 阿里云配置
```yaml
aliyun:
  access_key_id: your_access_key_id
  access_key_secret: your_access_key_secret
  region: cn-hangzhou
```

#### 华为云配置
```yaml
huawei:
  access_key_id: your_access_key_id
  access_key_secret: your_access_key_secret
  region: cn-north-4
  discovery:
    enable_auto_discovery: true
    cache_duration: 6h
```

### 2. 数据库配置

```yaml
database:
  host: localhost
  port: 3306
  username: root
  password: password
  database: ai_cloudops
```

### 3. Redis配置

```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  database: 0
```

## 开发指南

### 1. 代码规范

#### Go代码规范
- 遵循Go官方代码规范
- 使用gofmt格式化代码
- 添加必要的注释和文档
- 编写单元测试

#### TypeScript代码规范
- 使用ESLint和Prettier
- 遵循TypeScript最佳实践
- 添加类型注解
- 编写组件测试

### 2. 提交规范

```bash
# 提交格式
<type>(<scope>): <subject>

# 示例
feat(provider): 添加华为云提供商支持
fix(monitor): 修复监控数据查询问题
docs(readme): 更新项目文档
```

### 3. 测试规范

```bash
# 运行单元测试
go test ./...

# 运行集成测试
go test -tags=integration ./...

# 运行前端测试
cd ui && npm test
```

## 部署指南

### 1. Docker部署

```bash
# 构建镜像
docker build -t ai-cloudops .

# 运行容器
docker run -d -p 8080:8080 ai-cloudops
```

### 2. Kubernetes部署

```bash
# 应用配置
kubectl apply -f deploy/k8s.yaml

# 查看状态
kubectl get pods -l app=ai-cloudops
```

### 3. Helm部署

```bash
# 安装Chart
helm install ai-cloudops ./deploy/helm/

# 升级Chart
helm upgrade ai-cloudops ./deploy/helm/
```

## 监控和运维

### 1. 监控指标

#### 系统指标
- CPU使用率
- 内存使用率
- 磁盘使用率
- 网络流量

#### 应用指标
- 请求响应时间
- 错误率
- 并发连接数
- 数据库连接数

### 2. 告警规则

#### 系统告警
- CPU使用率 > 80%
- 内存使用率 > 85%
- 磁盘使用率 > 90%
- 服务不可用

#### 业务告警
- API响应时间 > 5s
- 错误率 > 5%
- 数据库连接失败
- 云资源异常

### 3. 日志管理

#### 日志级别
- DEBUG: 调试信息
- INFO: 一般信息
- WARN: 警告信息
- ERROR: 错误信息

#### 日志格式
```json
{
  "timestamp": "2025-06-23T10:30:00Z",
  "level": "INFO",
  "service": "ai-cloudops",
  "message": "服务启动成功",
  "trace_id": "abc123"
}
```

## 贡献指南

### 1. 开发流程

1. **Fork项目**: Fork到自己的仓库
2. **创建分支**: 创建功能分支
3. **开发功能**: 实现新功能
4. **编写测试**: 添加单元测试
5. **提交代码**: 提交到分支
6. **创建PR**: 创建Pull Request

### 2. 代码审查

- 所有代码必须经过审查
- 确保测试覆盖率
- 遵循代码规范
- 添加必要文档

### 3. 发布流程

1. **版本规划**: 确定版本计划
2. **功能开发**: 完成功能开发
3. **测试验证**: 全面测试验证
4. **文档更新**: 更新相关文档
5. **发布版本**: 发布新版本

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](LICENSE) 文件。

## 联系方式

- **项目地址**: [GitHub Repository](https://github.com/your-org/ai-cloudops)
- **问题反馈**: [Issues](https://github.com/your-org/ai-cloudops/issues)
- **讨论交流**: [Discussions](https://github.com/your-org/ai-cloudops/discussions)

## 更新日志

### v1.0.0 (2025-06-23)
- ✅ 完成华为云提供商实现
- ✅ 移除所有硬编码，实现完全动态化
- ✅ 优化性能和错误处理
- ✅ 完善测试覆盖
- ✅ 更新项目文档

### v0.9.0 (2025-06-20)
- 🚧 华为云提供商开发中
- ✅ 阿里云提供商完成
- ✅ 基础框架搭建完成
- ✅ 监控系统集成完成

---

**最后更新**: 2025年6月23日  
**维护者**: AI-CloudOps Team 