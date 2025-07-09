# AI-CloudOps 智能运维平台完整指南

## 📖 项目概述

AI-CloudOps AI 模块是一个基于人工智能的智能运维系统，专注于 Kubernetes 环境的根因分析和自动化修复。系统集成了多种 AI 技术，包括异常检测、相关性分析、大语言模型和多 Agent 协作，为运维团队提供智能化的问题诊断和解决方案。

## ✨ 核心功能特性

### 🔍 智能根因分析 (RCA)
- **多维度异常检测**: 基于 Z-Score、IQR、孤立森林、DBSCAN 等多种算法
- **相关性分析**: 发现指标间的关联关系，识别潜在的因果链
- **时间序列分析**: 支持时间窗口分析和趋势检测
- **AI 智能总结**: 基于 LLM 生成人类可读的分析报告

### 🤖 多 Agent 自动修复系统
- **Supervisor Agent**: 协调整体修复流程，管理决策链
- **K8s Fixer Agent**: 专门处理 Kubernetes 相关问题
- **Researcher Agent**: 搜索解决方案和最佳实践
- **Coder Agent**: 执行数据分析和计算任务
- **Notifier Agent**: 发送通知和告警

### 📊 智能负载预测
- **机器学习预测模型**: 基于历史数据预测未来实例需求
- **时间特征工程**: 考虑时间周期性和业务模式
- **置信度评估**: 提供预测结果的可信度分析
- **自动扩缩容**: 与 Kubernetes HPA 集成实现智能扩缩容

### 🧠 智能助手 (RAG)
- **检索增强生成**: 从知识库中检索相关内容回答问题
- **上下文感知**: 支持会话记忆和多轮对话
- **网络搜索增强**: 可连接互联网获取最新信息
- **多格式文档处理**: 支持 Markdown、PDF、CSV 等多种格式
- **反幻觉机制**: 通过验证减少虚假信息生成

### 🔔 智能通知系统
- **飞书集成**: 支持富文本卡片消息和机器人推送
- **分级告警**: 根据严重程度发送不同级别的通知
- **人工干预识别**: 自动识别需要人工介入的场景
- **智能值班推荐**: 基于问题类型推荐合适的值班人员

## 🏗 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway                          │
├─────────────────────────────────────────────────────────────┤
│ Health API │ Predict API │ RCA API │ AutoFix API │ Assistant API │
├─────────────────────────────────────────────────────────────┤
│                    Core Business Logic                      │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
│ │   RCA       │ │ Prediction  │ │    Multi-Agent          │ │
│ │   Engine    │ │ Service     │ │    System               │ │
│ └─────────────┘ └─────────────┘ └─────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                     Service Layer                           │
│ Prometheus │ Kubernetes │ LLM │ Notification │ Vector Store │
├─────────────────────────────────────────────────────────────┤
│                    Infrastructure                           │
│ Prometheus │ Grafana │ Ollama │ Redis │ Chroma DB │ K8s     │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求
- Python 3.11+
- Docker & Docker Compose
- Kubernetes 集群（可选）
- 大语言模型服务（Ollama 本地服务或 OpenAI API）
- 至少 4GB RAM

### 1. 项目部署

```bash
# 克隆项目
git clone <repository-url>
cd AI-CloudOps-backend/python

# 配置环境变量
cp env.example env.production
# 编辑 env.production 文件，配置相关参数

# 使用 Docker Compose 启动
docker-compose up -d --build

# 验证服务
curl http://localhost:8080/api/v1/health
```

### 2. 配置说明

系统采用双层配置机制：

#### 环境变量配置 (敏感信息)
```bash
# 环境选择
ENV=production

# LLM 配置
LLM_API_KEY=sk-xxx
LLM_BASE_URL=https://api.url

# 通知配置
FEISHU_WEBHOOK=https://webhook.url

# 搜索API
TAVILY_API_KEY=key-xxx
```

#### YAML 配置文件 (非敏感配置)
- 开发环境：`config/config.yaml`
- 生产环境：`config/config.production.yaml`

### 3. 本地开发

```bash
# 创建虚拟环境
python -m venv venv
source venv/bin/activate

# 安装依赖
pip install -r requirements.txt

# 启动开发服务器
python app/main.py
```

## 📚 API 使用指南

### 健康检查 API

```bash
# 系统健康检查
GET /api/v1/health

# 响应示例
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "services": {
    "prometheus": "healthy",
    "llm": "healthy",
    "vector_store": "healthy"
  }
}
```

### 智能根因分析 API

```bash
# 执行根因分析
POST /api/v1/rca
{
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-01T11:00:00Z",
  "metrics": ["container_cpu_usage_seconds_total", "container_memory_usage_bytes"]
}

# 响应示例
{
  "analysis_id": "rca_20240101_120000",
  "anomalies": [
    {
      "metric": "container_cpu_usage_seconds_total",
      "severity": "高",
      "score": 0.85,
      "timestamp": "2024-01-01T10:30:00Z"
    }
  ],
  "root_causes": [
    {
      "description": "CPU 使用率异常增高",
      "confidence": 0.92,
      "related_metrics": ["container_cpu_usage_seconds_total"]
    }
  ],
  "summary": "系统在 10:30 左右出现 CPU 使用率异常，可能由于突发流量导致..."
}
```

### 负载预测 API

```bash
# 获取当前预测
GET /api/v1/predict

# 自定义预测
POST /api/v1/predict
{
  "current_qps": 100.5,
  "timestamp": "2024-01-01T12:00:00Z",
  "include_features": true
}

# 响应示例
{
  "instances": 5,
  "current_qps": 100.5,
  "timestamp": "2024-01-01T12:00:00Z",
  "confidence": 0.87,
  "model_version": "1.0",
  "prediction_type": "model_based",
  "features": {
    "qps": 100.5,
    "hour": 12,
    "is_business_hour": true,
    "is_weekend": false
  }
}
```

### 自动修复 API

```bash
# 自动修复部署问题
POST /api/v1/autofix
{
  "deployment": "nginx-deployment",
  "namespace": "default",
  "event": "Pod启动失败，CrashLoopBackOff状态",
  "force": true
}

# 诊断集群健康状态
POST /api/v1/autofix/diagnose
{
  "namespace": "default"
}

# 执行完整修复工作流
POST /api/v1/autofix/workflow
{
  "problem_description": "Kubernetes集群中的nginx-deployment出现了Pod无法启动的问题"
}
```

### 智能助手 API

```bash
# 创建新会话
POST /api/v1/assistant/session
# 响应: {"session_id": "session_123", "created_at": "2024-01-01T12:00:00Z"}

# 查询问题
POST /api/v1/assistant/query
{
  "question": "如何优化Kubernetes中的资源配额设置?",
  "session_id": "session_123",
  "use_web_search": true,
  "max_context_docs": 4
}

# 刷新知识库
POST /api/v1/assistant/refresh
```

## 🛠 开发指南

### 项目结构

```
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
├── tests/                 # 测试代码
└── scripts/               # 脚本文件
```

### 扩展知识库

向智能助手添加新知识：

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

# 跳过依赖LLM的测试
SKIP_LLM_TESTS=true pytest tests/

# 生成覆盖率报告
pytest --cov=app tests/
```

## 📊 监控与运维

### 性能指标

系统的关键性能指标：

- **响应时间**: < 100ms (API 响应)
- **准确率**: > 95% (根因分析)
- **召回率**: > 90% (异常检测)
- **可用性**: 99.99% (系统可用性)

### 日志管理

```bash
# 查看应用日志
tail -f logs/app.log

# 查看容器日志
docker logs -f aiops-backend

# 查看特定模块日志
grep "aiops.assistant" logs/app.log
```

### 常见问题排查

#### 1. 无法连接到 Kubernetes
- 检查 K8s 配置文件是否正确
- 验证 `KUBECONFIG` 环境变量
- 确认集群访问权限

#### 2. LLM 服务连接失败
- 检查网络连接
- 验证 API 密钥是否正确
- 尝试切换到备用模型提供商

#### 3. 预测结果异常
- 检查 Prometheus 连接状态
- 验证历史数据是否充足
- 重新加载预测模型

## 🔧 高级配置

### LLM 模型配置

支持多种 LLM 配置方式：

1. **Ollama 本地模型**（推荐开发环境）
2. **OpenAI 兼容 API**（推荐生产环境）
3. **自动故障切换**（主备模式）

### 向量数据库配置

```yaml
rag:
  vector_db:
    type: "chroma"
    path: "data/vector_db"
    collection_name: "aiops_knowledge"
  embeddings:
    provider: "openai"  # 或 "ollama"
    model: "text-embedding-ada-002"
```

### 通知系统配置

```yaml
notification:
  feishu:
    webhook_url: "${FEISHU_WEBHOOK}"
    enabled: true
    card_template: "rich"
  email:
    enabled: false
    smtp_server: "smtp.example.com"
```

## 🔄 系统维护

### 定期维护任务

1. **数据库备份**
   ```bash
   # 备份向量数据库
   cp -r data/vector_db data/backup/vector_db_$(date +%Y%m%d)
   ```

2. **模型更新**
   ```bash
   # 重新训练预测模型
   python scripts/train_model.py
   ```

3. **知识库同步**
   ```bash
   # 同步最新文档
   curl -X POST http://localhost:8080/api/v1/assistant/refresh
   ```

### 容量规划

根据使用规模调整资源配置：

- **小规模**（< 100个Pod）: 2核4GB
- **中规模**（100-1000个Pod）: 4核8GB
- **大规模**（> 1000个Pod）: 8核16GB

## 📞 技术支持

### 联系方式

- **项目负责人**: Bamboo
- **邮箱**: 13664854532@163.com
- **项目主页**: https://github.com/GoSimplicity/AI-CloudOps

### 社区支持

- **文档**: 参考项目 Wiki
- **问题反馈**: 通过 GitHub Issues
- **功能请求**: 通过 GitHub Discussions

---

*本文档最后更新时间: 2024-01-01*