# 智能助手 (RAG) 使用指南

## 概述

AI-CloudOps 智能助手是一个基于检索增强生成 (RAG) 技术的智能问答系统，能够理解自然语言问题并从知识库中检索相关信息，提供准确、有用的回答。

## 🧠 核心技术

### 1. 检索增强生成 (RAG)
- **文档索引**: 将知识库文档转换为向量表示
- **语义检索**: 基于问题语义检索最相关的文档片段
- **上下文生成**: 结合检索到的内容生成准确回答
- **事实验证**: 减少幻觉，提高回答准确性

### 2. 向量数据库
- **向量化**: 使用 OpenAI 或 Ollama 的嵌入模型
- **存储**: 基于 ChromaDB 的向量数据库
- **检索**: 余弦相似度搜索
- **更新**: 支持增量更新和全量重建

### 3. 大语言模型集成
- **OpenAI 兼容**: 支持 GPT-4、GPT-3.5 等模型
- **Ollama 本地**: 支持 Qwen、Llama 等开源模型
- **自动切换**: 主备模式，自动故障切换
- **流式输出**: 支持实时响应流

## 🚀 快速开始

### 1. 创建会话

```bash
# 创建新的对话会话
curl -X POST http://localhost:8080/api/v1/assistant/session

# 响应示例
{
  "session_id": "session_20240101_120000",
  "created_at": "2024-01-01T12:00:00Z",
  "status": "active"
}
```

### 2. 提问查询

```bash
# 发送问题查询
curl -X POST http://localhost:8080/api/v1/assistant/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "如何优化Kubernetes集群的资源使用？",
    "session_id": "session_20240101_120000",
    "use_web_search": true,
    "max_context_docs": 4
  }'
```

### 3. 流式查询

```bash
# WebSocket 连接进行流式对话
ws://localhost:8080/api/v1/assistant/stream?session_id=session_20240101_120000
```

## 📖 知识库管理

### 1. 支持的文档格式

- **Markdown**: .md 文件
- **PDF**: .pdf 文件
- **纯文本**: .txt 文件
- **CSV**: .csv 文件
- **JSON**: .json 文件
- **HTML**: .html 文件

### 2. 添加知识

```bash
# 1. 将文档添加到知识库目录
cp your_document.md data/knowledge_base/

# 2. 刷新知识库
curl -X POST http://localhost:8080/api/v1/assistant/refresh

# 3. 验证添加结果
curl -X GET http://localhost:8080/api/v1/assistant/health
```

### 3. 文档结构建议

```markdown
# 主标题
## 二级标题
### 三级标题

- 要点1
- 要点2
- 要点3

### 代码示例
```bash
kubectl get pods
```

### 表格信息
| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| timeout | 超时时间 | 30s |
```

## 🔧 高级功能

### 1. 网络搜索增强

开启网络搜索获取最新信息：

```bash
curl -X POST http://localhost:8080/api/v1/assistant/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Kubernetes 1.29 有什么新特性？",
    "session_id": "session_id",
    "use_web_search": true,
    "web_search_num_results": 5
  }'
```

### 2. 上下文控制

控制检索的文档数量和相关性：

```bash
{
  "question": "如何监控 Pod 性能？",
  "session_id": "session_id",
  "max_context_docs": 6,
  "min_relevance_score": 0.7,
  "enable_context_compression": true
}
```

### 3. 会话记忆

系统自动维护会话上下文：

```bash
# 第一个问题
POST /api/v1/assistant/query
{
  "question": "什么是 Kubernetes？",
  "session_id": "session_id"
}

# 后续问题会基于上下文
POST /api/v1/assistant/query
{
  "question": "它有什么优势？",  # 系统理解"它"指的是 Kubernetes
  "session_id": "session_id"
}
```

## 🎯 使用场景

### 1. 运维知识查询

```bash
# 查询运维最佳实践
{
  "question": "Kubernetes 生产环境部署有什么注意事项？",
  "use_web_search": false,
  "max_context_docs": 5
}
```

### 2. 故障排除指导

```bash
# 获取故障排除指导
{
  "question": "Pod 一直处于 Pending 状态，如何排查？",
  "session_id": "troubleshooting_session",
  "max_context_docs": 4
}
```

### 3. 配置参考

```bash
# 获取配置示例
{
  "question": "给我一个 Nginx Deployment 的 YAML 配置示例",
  "session_id": "config_session"
}
```

### 4. 最佳实践咨询

```bash
# 询问最佳实践
{
  "question": "如何设计 Kubernetes 集群的监控体系？",
  "use_web_search": true,
  "max_context_docs": 6
}
```

## 🛡️ 安全和隐私

### 1. 数据隐私

- **本地处理**: 敏感数据不发送到外部服务
- **会话隔离**: 不同会话之间数据隔离
- **定期清理**: 自动清理过期会话数据

### 2. 访问控制

```yaml
# 配置访问控制
assistant:
  security:
    enable_auth: true
    max_sessions_per_user: 5
    session_timeout: 3600
    rate_limit: 100
```

### 3. 内容过滤

- **敏感信息过滤**: 自动过滤 API 密钥、密码等
- **恶意内容检测**: 检测和阻止恶意提问
- **输出审查**: 回答内容安全审查

## 📊 性能优化

### 1. 向量数据库优化

```yaml
# ChromaDB 配置优化
vector_db:
  collection_name: "aiops_knowledge"
  embedding_function: "openai"
  similarity_search:
    k: 10
    score_threshold: 0.6
  index_params:
    ef_construction: 200
    m: 16
```

### 2. 缓存策略

- **查询缓存**: 缓存常见问题的答案
- **文档缓存**: 缓存处理过的文档
- **向量缓存**: 缓存计算过的向量

### 3. 并发控制

```yaml
assistant:
  performance:
    max_concurrent_queries: 10
    query_timeout: 30
    embedding_batch_size: 100
    max_context_length: 4000
```

## 🔍 监控和调试

### 1. 健康检查

```bash
# 检查助手健康状态
curl -X GET http://localhost:8080/api/v1/assistant/health

# 响应示例
{
  "status": "healthy",
  "vector_store": "connected",
  "llm_service": "available",
  "knowledge_base": {
    "documents": 25,
    "last_updated": "2024-01-01T12:00:00Z"
  }
}
```

### 2. 性能指标

- **查询延迟**: 平均查询响应时间
- **检索准确率**: 检索到相关文档的比例
- **用户满意度**: 用户对回答质量的评分

### 3. 日志分析

```bash
# 查看查询日志
grep "assistant.query" logs/app.log

# 查看检索日志
grep "vector_store" logs/app.log

# 查看错误日志
grep "ERROR" logs/app.log | grep "assistant"
```

## 🎨 定制化配置

### 1. 提示词模板

```python
# 自定义系统提示词
SYSTEM_PROMPT = """
你是一个专业的 Kubernetes 运维专家。
请基于提供的文档回答用户的问题。
如果文档中没有相关信息，请明确说明。
回答要准确、简洁、实用。
"""
```

### 2. 检索策略

```yaml
retrieval:
  strategy: "hybrid"  # semantic, keyword, hybrid
  rerank: true
  max_docs: 10
  min_score: 0.7
  diversity_penalty: 0.1
```

### 3. 生成参数

```yaml
generation:
  temperature: 0.1
  max_tokens: 1000
  top_p: 0.9
  frequency_penalty: 0.0
  presence_penalty: 0.0
```

## 💡 最佳实践

### 1. 问题设计

- **具体明确**: 避免模糊的问题
- **上下文完整**: 提供必要的背景信息
- **分步骤**: 复杂问题可以分解为多个步骤

### 2. 知识库维护

- **定期更新**: 及时更新过期信息
- **结构化**: 使用清晰的标题和分类
- **示例丰富**: 提供充足的代码示例

### 3. 性能优化

- **批量查询**: 避免频繁的单次查询
- **缓存利用**: 充分利用查询缓存
- **会话管理**: 适时清理无用会话

## 🔮 未来规划

### 1. 多模态支持

- **图像理解**: 支持图表和架构图分析
- **代码生成**: 自动生成配置文件
- **语音交互**: 支持语音问答

### 2. 智能推荐

- **相关问题**: 推荐相关的后续问题
- **最佳实践**: 主动推荐最佳实践
- **个性化**: 基于用户历史的个性化推荐

### 3. 集成增强

- **Slack 集成**: 支持 Slack Bot
- **API 扩展**: 提供更多 API 接口
- **插件系统**: 支持第三方插件

---

*智能助手是 AI-CloudOps 的核心组件之一，持续优化中。如有问题或建议，请通过 GitHub Issues 反馈。*