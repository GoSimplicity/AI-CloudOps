# AI-CloudOps 平台快速入门指南

## 🚀 快速开始

### 系统要求

#### 最低要求
- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+) 或 macOS 10.15+
- **Python 版本**: 3.11 或更高
- **内存**: 4GB RAM
- **存储**: 20GB 可用磁盘空间
- **网络**: 可访问互联网

#### 推荐配置
- **CPU**: 4 核或更多
- **内存**: 8GB RAM 或更多
- **存储**: 50GB SSD
- **Docker**: 20.10+ 版本
- **Kubernetes**: 1.19+ 版本（如需 K8s 功能）

### 一键启动

#### 使用 Docker Compose（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/GoSimplicity/AI-CloudOps.git
cd AI-CloudOps/python

# 2. 配置环境变量
cp env.example env.production
# 编辑 env.production 文件，设置必要的配置

# 3. 启动所有服务
docker-compose up -d --build

# 4. 等待服务启动完成（约 2-3 分钟）
docker-compose ps

# 5. 验证服务状态
curl http://localhost:8080/api/v1/health
```

#### 本地开发环境

```bash
# 1. 创建虚拟环境
python -m venv aiops-env
source aiops-env/bin/activate  # Linux/macOS
# 或 aiops-env\Scripts\activate  # Windows

# 2. 安装依赖
pip install -r requirements.txt

# 3. 配置环境变量
export ENV=development
export PROMETHEUS_HOST=127.0.0.1:9090
export LLM_PROVIDER=ollama

# 4. 启动应用
python app/main.py
```

### 配置管理

#### 环境变量配置（必需）

创建 `env.production` 文件：

```bash
# ==================== 环境配置 ====================
ENV=production

# ==================== LLM 配置 ====================
# OpenAI 兼容 API 配置
LLM_API_KEY=sk-your-api-key-here
LLM_BASE_URL=https://api.openai.com/v1

# 或者使用本地 Ollama
# LLM_PROVIDER=ollama
# OLLAMA_BASE_URL=http://127.0.0.1:11434

# ==================== 监控配置 ====================
PROMETHEUS_HOST=127.0.0.1:9090

# ==================== 通知配置 ====================
# 飞书 Webhook（可选）
FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook

# ==================== 搜索配置 ====================
# Tavily 搜索 API（可选，用于网络搜索增强）
TAVILY_API_KEY=tvly-your-api-key-here

# ==================== Kubernetes 配置 ====================
# K8s 配置文件路径（可选）
K8S_CONFIG_PATH=/path/to/kubeconfig
K8S_IN_CLUSTER=false
```

#### YAML 配置文件

系统会根据 `ENV` 环境变量自动选择配置文件：
- `development`: 使用 `config/config.yaml`
- `production`: 使用 `config/config.production.yaml`

### 验证安装

#### 1. 健康检查

```bash
# 系统整体健康状态
curl http://localhost:8080/api/v1/health

# 预期响应
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

#### 2. 功能测试

```bash
# 测试负载预测
curl http://localhost:8080/api/v1/predict

# 测试智能助手
curl -X POST http://localhost:8080/api/v1/assistant/session

# 测试根因分析（需要 Prometheus 数据）
curl -X POST http://localhost:8080/api/v1/rca \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-01T11:00:00Z",
    "metrics": ["up"]
  }'
```

#### 3. Web 界面

访问 API 文档界面：
- **Swagger UI**: http://localhost:8080/docs
- **ReDoc**: http://localhost:8080/redoc

## 📚 核心功能演示

### 1. 智能助手使用

#### 创建会话并提问

```bash
# 创建新会话
SESSION_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/assistant/session)
SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.session_id')

# 提问
curl -X POST http://localhost:8080/api/v1/assistant/query \
  -H "Content-Type: application/json" \
  -d "{
    \"question\": \"如何查看 Kubernetes Pod 的日志？\",
    \"session_id\": \"$SESSION_ID\"
  }"
```

#### 添加自定义知识

```bash
# 1. 将文档添加到知识库目录
echo "# 自定义运维指南
这是我们公司的 Kubernetes 运维最佳实践...
" > data/knowledge_base/custom_guide.md

# 2. 刷新知识库
curl -X POST http://localhost:8080/api/v1/assistant/refresh
```

### 2. 负载预测使用

#### 获取当前预测

```bash
# 基于当前系统状态预测
curl http://localhost:8080/api/v1/predict

# 自定义 QPS 预测
curl -X POST http://localhost:8080/api/v1/predict \
  -H "Content-Type: application/json" \
  -d '{
    "current_qps": 150.0,
    "include_features": true
  }'
```

#### 查看趋势预测

```bash
# 预测未来 24 小时负载
curl -X POST http://localhost:8080/api/v1/predict/trend \
  -H "Content-Type: application/json" \
  -d '{
    "hours_ahead": 24,
    "current_qps": 100.0
  }'
```

### 3. 根因分析使用

#### 执行 RCA 分析

```bash
# 分析最近 1 小时的异常
END_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
START_TIME=$(date -u -d "1 hour ago" +"%Y-%m-%dT%H:%M:%SZ")

curl -X POST http://localhost:8080/api/v1/rca \
  -H "Content-Type: application/json" \
  -d "{
    \"start_time\": \"$START_TIME\",
    \"end_time\": \"$END_TIME\",
    \"metrics\": [
      \"container_cpu_usage_seconds_total\",
      \"container_memory_usage_bytes\"
    ]
  }"
```

### 4. K8s 自动修复使用

#### 诊断集群状态

```bash
# 诊断默认命名空间
curl -X POST http://localhost:8080/api/v1/autofix/diagnose \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default"
  }'
```

#### 自动修复部署

```bash
# 修复有问题的部署
curl -X POST http://localhost:8080/api/v1/autofix \
  -H "Content-Type: application/json" \
  -d '{
    "deployment": "nginx-deployment",
    "namespace": "default",
    "event": "Pod启动失败，健康检查配置错误",
    "force": false
  }'
```

## 🔧 常见配置场景

### 1. 纯本地环境（Ollama）

```bash
# 安装 Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# 下载模型
ollama pull qwen2.5:3b

# 配置环境变量
export LLM_PROVIDER=ollama
export OLLAMA_BASE_URL=http://127.0.0.1:11434
export LLM_MODEL=qwen2.5:3b
```

### 2. 云端 API（OpenAI 兼容）

```bash
# 配置环境变量
export LLM_PROVIDER=openai
export LLM_API_KEY=sk-your-api-key
export LLM_BASE_URL=https://api.openai.com/v1
export LLM_MODEL=gpt-3.5-turbo
```

### 3. 生产环境监控集成

```bash
# Prometheus 配置
export PROMETHEUS_HOST=prometheus.monitoring.svc.cluster.local:9090

# Kubernetes 配置
export K8S_IN_CLUSTER=true  # 如果在 K8s 集群内运行
export K8S_CONFIG_PATH=/etc/kubernetes/admin.conf  # 如果在集群外
```

### 4. 通知系统配置

```bash
# 飞书机器人
export FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/xxx

# 启用通知
export NOTIFICATION_ENABLED=true
```

## 🔍 故障排除

### 常见问题解决

#### 1. LLM 服务连接失败

```bash
# 检查 LLM 服务状态
curl http://localhost:8080/api/v1/health

# 测试 Ollama 连接
curl http://127.0.0.1:11434/api/tags

# 测试 OpenAI API
curl -H "Authorization: Bearer $LLM_API_KEY" \
     -H "Content-Type: application/json" \
     "$LLM_BASE_URL/models"
```

#### 2. Prometheus 连接问题

```bash
# 检查 Prometheus 连接
curl http://127.0.0.1:9090/-/healthy

# 测试查询
curl "http://127.0.0.1:9090/api/v1/query?query=up"
```

#### 3. 知识库加载失败

```bash
# 检查知识库目录
ls -la data/knowledge_base/

# 手动刷新知识库
curl -X POST http://localhost:8080/api/v1/assistant/refresh

# 检查向量数据库
ls -la data/vector_db/
```

#### 4. 容器启动失败

```bash
# 查看容器日志
docker-compose logs aiops-backend

# 检查资源使用
docker stats

# 重启服务
docker-compose restart aiops-backend
```

### 日志分析

#### 应用日志位置

```bash
# 容器环境
docker-compose logs -f aiops-backend

# 本地环境
tail -f logs/app.log

# 按模块查看日志
grep "aiops.assistant" logs/app.log
grep "aiops.rca" logs/app.log
grep "aiops.predictor" logs/app.log
```

#### 调试模式

```bash
# 启用调试日志
export LOG_LEVEL=DEBUG

# 重启应用
python app/main.py
```

## 📖 下一步

### 1. 深入了解功能

- **智能助手**: 阅读 [intelligent_assistant_guide.md](intelligent_assistant_guide.md)
- **负载预测**: 阅读 [load_prediction_guide.md](load_prediction_guide.md)
- **根因分析**: 阅读 [rca_analysis_guide.md](rca_analysis_guide.md)
- **K8s 修复**: 阅读 [k8s_autofix_guide.md](k8s_autofix_guide.md)

### 2. 生产环境部署

- 阅读完整部署指南
- 配置监控和告警
- 设置备份和恢复
- 制定运维流程

### 3. 定制开发

- 查看 API 文档
- 了解扩展机制
- 开发自定义 Agent
- 集成现有系统

### 4. 社区参与

- 提交问题和建议
- 分享使用经验
- 贡献代码和文档
- 参与技术讨论

## 📞 获取帮助

### 技术支持

- **文档**: 查看完整技术文档
- **API 文档**: http://localhost:8080/docs
- **GitHub Issues**: https://github.com/GoSimplicity/AI-CloudOps/issues
- **邮件支持**: 13664854532@163.com

### 学习资源

- **示例代码**: 查看 `examples/` 目录
- **测试用例**: 查看 `tests/` 目录
- **配置示例**: 查看 `config/` 目录
- **脚本工具**: 查看 `scripts/` 目录

### 社区交流

- **项目主页**: https://github.com/GoSimplicity/AI-CloudOps
- **技术博客**: 关注项目更新和技术分享
- **用户群组**: 加入用户交流群

---

*欢迎使用 AI-CloudOps！如果您在使用过程中遇到任何问题，请随时联系我们。*