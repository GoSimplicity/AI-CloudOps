# AI-CloudOps AI Module - 智能运维根因分析与自动修复系统

[![Python Version](https://img.shields.io/badge/python-3.11+-blue.svg)](https://python.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-enabled-blue.svg)](Dockerfile)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.104.1-green.svg)](https://fastapi.tiangolo.com/)
[![LangChain](https://img.shields.io/badge/LangChain-0.1.0-orange.svg)](https://langchain.com/)

## 📖 项目简介

AI-CloudOps AI 模块是一个基于人工智能的智能运维系统，专注于 Kubernetes 环境的根因分析和自动化修复。系统集成了多种 AI 技术，包括异常检测、相关性分析、大语言模型和多 Agent 协作，为运维团队提供智能化的问题诊断和解决方案。该系统通过 RAG（检索增强生成）技术提供智能问答功能，结合多 Agent 协作实现自动化运维。

## ✨ 核心功能

### 🔍 智能根因分析

- **多维度异常检测**: 基于 Z-Score、IQR、孤立森林、DBSCAN 等多种算法
- **相关性分析**: 发现指标间的关联关系，识别潜在的因果链
- **时间序列分析**: 支持时间窗口分析和趋势检测
- **AI 智能总结**: 基于 LLM 生成人类可读的分析报告

### 🤖 多 Agent 自动修复

- **Supervisor Agent**: 协调整体修复流程
- **K8s Fixer Agent**: 专门处理 Kubernetes 问题
- **Researcher Agent**: 搜索解决方案和最佳实践
- **Coder Agent**: 执行数据分析和计算任务
- **Notifier Agent**: 发送通知和告警

### 📊 负载预测

- **基于机器学习的预测模型**: 预测未来实例需求
- **时间特征工程**: 考虑时间周期性和业务模式
- **置信度评估**: 提供预测结果的可信度

### 🔔 智能通知

- **飞书集成**: 支持富文本卡片消息
- **分级告警**: 根据严重程度发送不同级别的通知
- **人工干预**: 自动识别需要人工介入的场景

### 🧠 智能助手

- **基于 RAG 的知识检索**: 从知识库中检索相关内容回答问题
- **上下文感知**: 支持会话记忆和多轮对话
- **网络搜索增强**: 可连接互联网获取最新信息
- **文档处理**: 支持多种格式（Markdown、PDF、CSV 等）
- **反幻觉机制**: 通过验证减少虚假信息生成

## 🏗 系统架构

```bash
┌─────────────────────────────────────────────────────────────┐
│ API Gateway │
├─────────────────────────────────────────────────────────────┤
│ Health API │ Predict API │ RCA API │ AutoFix API │ Assistant API │
├─────────────────────────────────────────────────────────────┤
│ Core Business Logic │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
│ │ RCA │ │ Prediction │ │ Multi-Agent │ │
│ │ Engine │ │ Service │ │ System │ │
│ └─────────────┘ └─────────────┘ └─────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ Service Layer │
│ Prometheus │ Kubernetes │ LLM │ Notification │ Vector Store │
├─────────────────────────────────────────────────────────────┤
│ Infrastructure │
│ Prometheus │ Grafana │ Ollama │ Redis │ Chroma DB │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求

- Python 3.11+
- Docker & Docker Compose
- Kubernetes 集群（可选）
- 大语言模型服务（Ollama 本地服务或 OpenAI API）
- 至少 4GB RAM

### 1. 克隆项目

```bash
git clone <repository-url>
cd AI-CloudOps-backend/python
```

### 2. 配置环境变量和配置文件

```bash
# 复制环境变量示例文件
cp env.example env.production
# 编辑 env.production 文件，配置相关参数
```

系统采用双层配置机制：

- **环境变量**：只存储敏感信息（API 密钥、Webhook 等）和环境选择
- **YAML 配置文件**：存储其他所有配置项，分为开发环境和生产环境两个配置文件

环境变量配置：

```
# 环境配置
ENV=production  # 设置使用的配置文件：development或production

# 敏感信息 - API密钥
LLM_API_KEY=sk-xxx  # LLM API密钥
LLM_BASE_URL=https://api.url  # API基础URL

# 通知配置
FEISHU_WEBHOOK=https://webhook.url

# Tavily搜索API密钥
TAVILY_API_KEY=key-xxx
```

配置文件位置：

- 开发环境：`config/config.yaml`
- 生产环境：`config/config.production.yaml`

### 3. 本地开发环境

```bash
# 创建虚拟环境
python -m venv venv
source venv/bin/activate  # Linux/Mac
# 或
venv\Scripts\activate  # Windows

# 安装依赖
pip install -r requirements.txt

# 启动开发服务器
python app/main.py
```

### 4. 使用 Docker Compose 启动

```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f aiops-backend
```

### 5. 验证服务

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 查看API文档
curl http://localhost:8080/docs
```

## 📚 典型使用场景

### 场景一：Kubernetes Pod 无法启动问题自动修复

当集群中的 Pod 频繁重启或无法正常启动时，系统可以：

1. 自动分析 Pod 事件和日志
2. 识别关键问题（如探针配置错误、资源限制问题）
3. 生成修复方案并自动应用
4. 验证修复结果并通知运维人员

示例命令：

```bash
curl -X POST http://localhost:8080/api/v1/autofix \
  -H "Content-Type: application/json" \
  -d '{
    "deployment": "nginx-deployment",
    "namespace": "default",
    "event": "Pod启动失败，CrashLoopBackOff状态",
    "force": true
  }'
```

### 场景二：使用智能助手进行知识查询

运维人员可以通过智能助手查询特定问题或最佳实践：

1. 创建新的会话
2. 提交关于 K8s 集群优化的问题
3. 系统从知识库中检索相关文档并生成回答
4. 支持多轮对话和后续问题

示例命令：

```bash
# 创建会话
curl -X POST http://localhost:8080/api/v1/assistant/session

# 查询问题
curl -X POST http://localhost:8080/api/v1/assistant/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "如何优化Kubernetes中的资源配额设置?",
    "session_id": "session-123",
    "use_web_search": true
  }'
```

### 场景三：负载预测与自动扩缩容

系统可以基于历史数据预测未来负载并提前调整资源：

1. 分析历史 QPS 和资源使用趋势
2. 预测未来时间窗口的需求
3. 根据预测结果自动调整副本数
4. 通过 Kubernetes HPA 控制器实现自动扩缩容

## 📋 API 文档

### 健康检查

```bash
GET /api/v1/health
```

### 负载预测

```bash
# 获取当前预测
GET /api/v1/predict

# 自定义预测
POST /api/v1/predict
{
  "current_qps": 100.5,
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 根因分析

```bash
POST /api/v1/rca
{
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-01T11:00:00Z",
  "metrics": ["container_cpu_usage_seconds_total"]
}
```

### 自动修复

```bash
POST /api/v1/autofix
{
  "deployment": "my-app",
  "namespace": "default",
  "event": "Pod启动失败"
}
```

### 智能助手

```bash
# 创建新会话
POST /api/v1/assistant/session

# 查询问题
POST /api/v1/assistant/query
{
  "question": "如何分析Kubernetes集群中的资源使用情况?",
  "use_web_search": true,
  "session_id": "session-123",
  "max_context_docs": 4
}

# 刷新知识库
POST /api/v1/assistant/refresh
```

## 🛠 开发指南

### 项目结构

```bash
python/
├── app/                    # 应用核心代码
│   ├── api/               # API路由和中间件
│   │   ├── middleware/    # 中间件（CORS, 错误处理等）
│   │   └── routes/        # API路由模块
│   ├── config/            # 配置管理
│   ├── core/              # 核心业务逻辑
│   │   ├── agents/        # 多Agent系统实现
│   │   ├── prediction/    # 负载预测模块
│   │   └── rca/           # 根因分析模块
│   ├── models/            # 数据模型
│   ├── services/          # 外部服务集成
│   ├── utils/             # 工具函数
│   └── main.py            # 应用入口
├── data/                  # 数据文件
│   ├── knowledge_base/    # 智能助手知识库
│   ├── models/            # 机器学习模型
│   └── sample/            # 样例数据
├── deploy/                # 部署配置
│   ├── kubernetes/        # K8s部署配置
│   └── prometheus/        # Prometheus配置
├── docs/                  # 文档
├── tests/                 # 测试代码
├── scripts/               # 脚本文件
└── docker-compose.yml     # Docker编排文件
```

### 扩展知识库

要向智能助手添加新知识：

1. 将文档（Markdown、PDF 等）添加到 `data/knowledge_base/` 目录
2. 刷新知识库：
   ```bash
   curl -X POST http://localhost:8080/api/v1/assistant/refresh
   ```

### 添加新 Agent

1. 在 `app/core/agents/` 目录下创建新的 Agent 类
2. 实现必要的接口方法
3. 在 SupervisorAgent 中注册新 Agent

### 运行测试

```bash
# 运行所有测试
python tests/run_tests.py

# 运行特定测试
pytest tests/test_health.py
pytest tests/test_assistant.py

# 跳过依赖LLM的测试（当LLM服务不可用时）
SKIP_LLM_TESTS=true pytest tests/

# 生成覆盖率报告
pytest --cov=app tests/
```

### 生产环境部署

生产环境部署推荐使用：

```bash
# 使用生产环境配置
cp env.example env.production
# 编辑配置文件...

# 启动生产服务
./scripts/start_production.sh
```

## 🔧 配置说明

### 主要配置项

| 配置项                   | 说明                | 默认值              |
| ------------------------ | ------------------- | ------------------- |
| PROMETHEUS_HOST          | Prometheus 服务地址 | 127.0.0.1:9090      |
| LLM_PROVIDER             | LLM 提供商          | openai              |
| LLM_MODEL                | 使用的 LLM 模型     | qwen2.5:3b          |
| LLM_API_KEY              | API 密钥            | sk-xxx              |
| RCA_ANOMALY_THRESHOLD    | 异常检测阈值        | 0.65                |
| PREDICTION_MAX_INSTANCES | 最大实例数          | 20                  |
| NOTIFICATION_ENABLED     | 是否启用通知        | true                |
| RAG_VECTOR_DB_PATH       | 向量数据库路径      | data/vector_db      |
| RAG_KNOWLEDGE_BASE_PATH  | 知识库路径          | data/knowledge_base |

### LLM 模型配置

系统支持多种 LLM 配置方式：

1. **Ollama 本地模型（推荐开发环境）**：

   - 设置 `LLM_PROVIDER=ollama`
   - 配置 `OLLAMA_MODEL` 和 `OLLAMA_BASE_URL`

2. **OpenAI 兼容 API**：

   - 设置 `LLM_PROVIDER=openai`
   - 配置 `LLM_MODEL`、`LLM_API_KEY` 和 `LLM_BASE_URL`

3. **自动故障切换**：
   - 系统会在主要提供商不可用时自动切换到备用提供商

## 📋 贡献指南

我们欢迎社区贡献，无论是报告问题、提交功能请求还是直接提交代码。

### 提交问题或建议

1. 确保您的问题未被报告过
2. 使用清晰的标题和详细描述
3. 包括复现步骤、预期行为和实际行为
4. 附上相关日志和截图

### 代码贡献流程

1. Fork 项目仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

### 编码规范

- 遵循 PEP 8 风格指南
- 为新功能编写单元测试
- 保持代码简洁明了
- 添加适当的文档注释

## 📞 联系我们

- **项目负责人**: Bamboo
- **邮箱**: [13664854532@163.com](mailto:13664854532@163.com)
- **项目主页**: [https://github.com/GoSimplicity/AI-CloudOps]

## 📄 许可证

本项目采用 MIT 许可证 - 详见 LICENSE 文件

## K8s 故障自动修复模块

K8s 故障自动修复模块是一个用于自动诊断和修复 Kubernetes 集群中常见问题的子系统。该模块可以检测和修复以下常见问题：

- Pod 健康检查（readinessProbe 和 livenessProbe）配置不当
- 资源请求和限制设置不合理
- 探针路径和探针频率配置错误
- CrashLoopBackOff 故障的分析和修复
- 集群中部署的其他常见错误

### 架构

自动修复模块由以下组件组成：

1. **K8sFixerAgent**: 核心修复逻辑，负责分析和修复 Kubernetes 相关问题
2. **SupervisorAgent**: 工作流协调器，管理修复过程中的决策链和修复步骤
3. **NotifierAgent**: 在关键修复事件中发送通知
4. **KubernetesService**: K8s API 的抽象层，提供集群操作功能

### 使用方法

#### 自动修复 API

```bash
# 修复特定部署
curl -X POST http://localhost:8080/api/v1/autofix \
  -H "Content-Type: application/json" \
  -d '{
    "deployment": "nginx-problematic",
    "namespace": "default",
    "event": "Pod启动失败，原因是readinessProbe探针路径(/health)不存在，探针频率过高，且资源请求过高",
    "force": true
  }'
```

#### 诊断集群健康状态

```bash
# 诊断特定命名空间
curl -X POST http://localhost:8080/api/v1/autofix/diagnose \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default"
  }'
```

### 典型使用场景

1. **自动修复健康检查问题**：

   - 检测并修复错误的探针路径
   - 调整不合理的探针频率和失败阈值
   - 添加缺失的 readinessProbe 或修复 livenessProbe

2. **资源优化**：

   - 识别并修复资源请求过高的配置
   - 平衡 CPU 和内存限制与请求

3. **多因素问题修复**：
   - 处理由多个因素引起的 CrashLoopBackOff 问题
   - 组合应用多项修复

### 样例修复流程

1. 发现 Pod 无法启动，处于 CrashLoopBackOff 状态
2. 调用自动修复 API
3. 系统分析 Pod 状态、事件和配置
4. 识别出 livenessProbe 配置不当：路径错误、频率过高
5. 自动应用修复配置
6. 验证 Pod 状态恢复正常

### 配置和调试

K8s 自动修复模块需要正确配置以下项：

- Kubernetes 配置文件路径: `K8S_CONFIG_PATH`环境变量
- 集群连接方式: `K8S_IN_CLUSTER`环境变量

#### API 端点

自动修复模块提供以下 API 端点：

- `POST /api/v1/autofix`: 自动修复指定的 deployment

  ```json
  {
    "deployment": "nginx-deployment",
    "namespace": "default",
    "event": "Pod启动失败，原因是健康检查配置不当",
    "force": true
  }
  ```

- `GET /api/v1/autofix/health`: 检查自动修复服务健康状态

- `POST /api/v1/autofix/diagnose`: 诊断集群健康状态

  ```json
  {
    "namespace": "default"
  }
  ```

- `POST /api/v1/autofix/workflow`: 执行完整的自动修复工作流

  ```json
  {
    "problem_description": "Kubernetes集群中的nginx-deployment出现了Pod无法启动的问题"
  }
  ```

- `POST /api/v1/autofix/notify`: 发送通知
  ```json
  {
    "type": "human_help",
    "message": "需要人工协助处理Kubernetes集群中的问题",
    "urgency": "medium"
  }
  ```

### 测试

项目包含一个测试脚本`tests/test-autofix.py`，可以用于测试自动修复功能。测试脚本会检查健康状态、执行诊断、尝试修复正常和异常部署，并验证结果。

#### 前提条件

1. 有一个可用的 Kubernetes 集群
2. 集群中已部署示例应用（正常的 nginx-deployment 和有问题的 nginx-problematic）
3. K8s 配置文件位于`deploy/kubernetes/config`

#### 运行测试

```bash
# 设置环境变量
export KUBECONFIG=deploy/kubernetes/config
export PYTHONPATH=$(pwd)

# 启动应用
python app/main.py

# 在另一个终端运行测试
python tests/test-autofix.py
```

## 🔄 更新日志

### v1.0.0 (2025-07-8)

- 初始版本发布
- 实现核心功能：智能根因分析、多 Agent 自动修复、负载预测
- 添加智能助手模块

### v1.1.0 (计划中)

- 增强智能助手功能，支持更多文档格式
- 改进多 Agent 协作机制
- 添加自动化测试覆盖

### 常见问题排查

#### 无法连接到 Kubernetes

确保：

- K8s 配置文件正确且包含访问权限
- 设置了正确的环境变量：`KUBECONFIG`和`K8S_CONFIG_PATH`

#### LLM 服务连接失败

检查：

- 网络连接是否正常
- API 密钥是否正确
- 尝试切换到备用模型提供商

#### 修复不生效

可能的原因：

- 集群资源不足，无法满足新 Pod 的请求
- 权限问题，无法修改 deployment
- 网络问题，无法访问 K8s API 服务器

可以尝试减小资源请求：

```bash
kubectl patch deployment <problematic-deployment> -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container-name>","resources":{"requests":{"memory":"32Mi","cpu":"50m"},"limits":{"memory":"64Mi","cpu":"100m"}},"readinessProbe":{"httpGet":{"path":"/"},"periodSeconds":10,"failureThreshold":3}}]}}}}'
```

## 配置管理

AIOps 平台使用两种配置机制：

1. **YAML 配置文件** - 存放在 `config/` 目录下，包含所有非敏感配置

   - `config.yaml` - 默认配置（开发环境）
   - `config.production.yaml` - 生产环境配置
   - 可根据需要创建其他环境配置文件，如 `config.test.yaml`

2. **环境变量** - 存放在 `.env` 或环境变量中，仅包含敏感数据（API 密钥、密码等）
   - `env.example` - 示例环境变量文件（模板）
   - `env.production` - 生产环境敏感数据

### 配置优先级

系统加载配置的优先级顺序为：

1. 环境变量（最高优先级）
2. 环境特定 YAML 配置文件（如 `config.production.yaml`）
3. 默认 YAML 配置文件（`config.yaml`）
4. 代码中的默认值（最低优先级）

### 使用方法

通过设置 `ENV` 环境变量来指定使用的配置：

```bash
# 开发环境（默认）
ENV=development ./scripts/start.sh

# 生产环境
ENV=production ./scripts/start.sh

# 或使用生产专用脚本
./scripts/start_production.sh
```

详细配置说明请参考 [配置指南](config/README.md)。
