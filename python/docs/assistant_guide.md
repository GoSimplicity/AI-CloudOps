# AI-CloudOps 智能小助手使用指南

## 1. 简介

智能小助手是 AIOps 平台的一个核心模块，提供基于检索增强生成（RAG）的问答能力。它可以回答关于系统架构、功能特点、操作指南等问题，同时支持网络搜索增强功能，以提供更全面、准确的回答。

本指南将帮助您了解如何使用智能小助手，以及如何扩展其知识库和功能。

## 2. 主要功能

智能小助手提供以下核心功能：

- **知识库问答**：基于向量检索的本地知识库问答
- **上下文会话**：支持多轮对话，保持上下文连贯性
- **网络搜索增强**：可选的网络搜索功能，增强回答的广度和时效性
- **推荐后续问题**：智能推荐相关的后续问题，引导用户深入了解
- **相关度评分**：提供回答相关度的评分，帮助用户判断回答质量
- **源文档引用**：展示回答依据的源文档，增加回答的可信度

## 3. 使用方法

### 3.1 WebUI 使用

1. 启动 AIOps 平台后，访问 Web 界面
2. 在左侧导航栏选择"智能小助手"
3. 在输入框中输入您的问题，点击发送或按回车
4. 查看智能小助手的回答及相关推荐

### 3.2 API 调用

智能小助手提供了 REST API 接口，可以方便地集成到其他系统中：

```bash
# 创建新会话
curl -X POST http://localhost:8080/api/v1/assistant/session

# 发送问题（不使用会话）
curl -X POST http://localhost:8080/api/v1/assistant/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "AIOps平台有哪些核心功能？",
    "use_web_search": false
  }'

# 发送问题（使用会话）
curl -X POST http://localhost:8080/api/v1/assistant/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "它的根因分析功能如何工作？",
    "session_id": "会话ID",
    "use_web_search": true
  }'

# 刷新知识库
curl -X POST http://localhost:8080/api/v1/assistant/refresh
```

### 3.3 WebSocket 流式接口

对于需要实时响应的场景，智能小助手还提供了 WebSocket 流式接口：

```javascript
// 前端示例代码
const ws = new WebSocket("ws://localhost:8080/api/v1/assistant/stream");

// 发送问题
ws.send(
  JSON.stringify({
    question: "AIOps平台是什么？",
    session_id: "可选会话ID",
    use_web_search: false,
  })
);

// 接收流式回答
ws.onmessage = function (event) {
  const data = JSON.parse(event.data);
  if (data.type === "content") {
    // 处理内容块
    console.log(data.content);
  } else if (data.type === "end") {
    // 处理回答结束
    console.log("回答完成", data.metadata);
  }
};
```

## 4. 知识库扩展

智能小助手的知识库位于`data/knowledge_base`目录下，您可以通过以下方式扩展知识库：

### 4.1 添加 Markdown 文档

1. 创建 Markdown 格式的知识文档（.md 文件）
2. 将文档放入`data/knowledge_base`目录
3. 调用刷新知识库 API 或重启服务

### 4.2 通过 API 动态添加知识

```bash
curl -X POST http://localhost:8080/api/v1/assistant/add-document \
  -H "Content-Type: application/json" \
  -d '{
    "content": "AIOps平台是一个智能运维平台，专注于...",
    "metadata": {
      "source": "用户手册",
      "author": "运维团队"
    }
  }'
```

## 5. 配置说明

智能小助手的配置位于`.env`文件或环境变量中，主要配置项包括：

| 配置项                   | 说明              | 默认值              |
| ------------------------ | ----------------- | ------------------- |
| RAG_VECTOR_DB_PATH       | 向量数据库路径    | data/vector_db      |
| RAG_KNOWLEDGE_BASE_PATH  | 知识库路径        | data/knowledge_base |
| RAG_CHUNK_SIZE           | 文档分块大小      | 1000                |
| RAG_TOP_K                | 检索文档数量      | 4                   |
| RAG_SIMILARITY_THRESHOLD | 相似度阈值        | 0.7                 |
| TAVILY_API_KEY           | 网络搜索 API 密钥 | -                   |

## 6. 最佳实践

1. **明确提问**：提供具体、明确的问题，可以获得更准确的回答
2. **利用会话功能**：多轮对话时使用同一会话 ID，保持上下文连贯性
3. **合理使用网络搜索**：对于知识库中可能没有的最新信息，开启网络搜索功能
4. **查看相关度评分**：根据相关度评分判断回答的质量和可信度
5. **关注源文档**：查看回答的来源文档，了解信息的出处

## 7. 故障排查

| 问题         | 可能原因           | 解决方案                 |
| ------------ | ------------------ | ------------------------ |
| 回答不准确   | 知识库缺少相关信息 | 扩充知识库或开启网络搜索 |
| 相关度评分低 | 问题超出知识范围   | 调整问题或使用网络搜索   |
| 响应缓慢     | LLM 服务负载高     | 检查 LLM 服务状态或配置  |
| 会话失效     | 会话超时或服务重启 | 创建新会话并重新提问     |

## 8. 常见问题

**Q: 如何判断回答的质量？**
A: 可以通过相关度评分和源文档引用来判断，分数越高，回答越可靠。

**Q: 智能小助手支持哪些语言？**
A: 智能小助手主要支持中文交流，但也能理解简单的英文问题。

**Q: 如何提高回答的准确性？**
A: 提供明确的问题，扩充专业领域的知识库，根据场景适当开启网络搜索。

## 9. 联系与支持

如有任何问题或需求，请联系 AIOps 平台管理团队：

- 邮箱：13664854532@163.com
- 问题追踪：https://github.com/GoSimplicity/AI-CloudOps/issues
