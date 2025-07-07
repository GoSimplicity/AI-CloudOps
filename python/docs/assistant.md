# AIOps 智能小助手功能

## 1. 功能概述

AIOps智能小助手是一个基于RAG（检索增强生成）技术的问答系统，可以帮助用户快速获取知识库中的信息，并提供准确的回答。主要特点：

- 强大的信息检索能力，能够准确找到与用户问题相关的文档
- 支持对话记忆，可以进行连续的上下文对话
- 支持网络搜索增强，获取实时外部信息
- 提供后续问题建议，引导用户深入了解
- 同时支持HTTP接口和WebSocket流式接口

## 2. 技术实现

### 2.1 核心组件

- **AssistantAgent**: 智能小助手核心代理类，实现了知识检索、问答生成等功能
- **RAG技术栈**: 使用LangChain框架实现检索增强生成
- **向量数据库**: 使用ChromaDB存储文档向量
- **嵌入模型**: 支持OpenAI和Ollama的嵌入模型
- **大语言模型**: 支持OpenAI和Ollama的语言模型
- **API接口**: 提供HTTP和WebSocket接口

### 2.2 主要优化

1. **文档相关性评估优化**:
   - 使用更精确的提示模板指导LLM评估文档相关性
   - 实现了严格的文档过滤机制
   - 优化了文档截取逻辑，处理长文档

2. **检索质量优化**:
   - 问题重写机制，提高检索准确性
   - 上下文与问题结合，处理多轮对话

3. **流式响应**:
   - 实现WebSocket接口，支持流式反馈
   - 提供中间状态更新，提升用户体验

## 3. 使用指南

### 3.1 启动服务

```bash
# 克隆代码
git clone <repository-url>
cd AI-CloudOps-backend

# 安装依赖
cd python
pip install -r requirements.txt

# 设置环境变量
cp env.example .env
# 编辑.env文件，设置必要的API密钥

# 启动应用（使用提供的启动脚本）
./scripts/start.sh

# 或者手动设置PYTHONPATH并启动
export PYTHONPATH=$PYTHONPATH:$(pwd)/..
python app/main.py
```

### 3.2 API接口

#### HTTP接口

- 查询: `POST /api/v1/assistant/query`
- 创建会话: `POST /api/v1/assistant/session`
- 刷新知识库: `POST /api/v1/assistant/refresh`
- 添加文档: `POST /api/v1/assistant/add-document`

详细接口文档请参考 [docs/assistant_guide.md](docs/assistant_guide.md)

#### WebSocket接口

连接URL: `ws://localhost:8080/api/v1/assistant/stream`

客户端发送格式:
```json
{
  "question": "AIOps平台有哪些功能？",
  "session_id": "uuid",
  "use_web_search": false,
  "max_context_docs": 4
}
```

服务端返回多条流式消息，包括处理状态、文档发现和最终回答等。

### 3.3 知识库管理

系统会自动加载`data/knowledge_base`目录下的所有`.md`和`.txt`文档。

更新知识库:
1. 在该目录中添加或修改文档
2. 调用API刷新知识库: `POST /api/v1/assistant/refresh`

## 4. 测试工具

项目提供了几个测试脚本:

- `scripts/api_test_assistant.py`: 测试HTTP API接口
- `scripts/test_websocket_assistant.py`: 测试WebSocket流式接口
- `scripts/test_knowledge_load.py`: 测试知识库加载和检索功能
- `scripts/interactive_assistant.py`: 交互式命令行测试
- `scripts/test_all.py`: 全面测试所有功能

运行测试:
```bash
# 全面测试
python scripts/test_all.py

# 单独测试
python scripts/test_knowledge_load.py
python scripts/api_test_assistant.py
python scripts/test_websocket_assistant.py

# 交互式测试
python scripts/interactive_assistant.py
```

## 5. 文档

- [使用指南](docs/assistant_guide.md): 详细使用方法和接口说明
- [效果评估](docs/assistant_evaluation.md): 性能评估和优化建议
- [示例知识库](data/knowledge_base/): 包含示例文档

## 6. 常见问题和解决方案

### 6.1 模块导入问题
如果遇到"No module named 'app'"错误，请确保正确设置PYTHONPATH：
```bash
export PYTHONPATH=$PYTHONPATH:$(pwd)/..
```
或使用提供的`scripts/start.sh`脚本启动。

### 6.2 依赖问题
如果缺少依赖，请安装完整的requirements.txt：
```bash
pip install -r requirements.txt
```

### 6.3 知识库加载问题
如果知识库文档未正确加载：
1. 确认文档存放在`data/knowledge_base`目录下
2. 确认文件格式为`.md`或`.txt`
3. 调用`/api/v1/assistant/refresh`刷新知识库
4. 检查日志文件获取详细错误信息

### 6.4 连接WebSocket错误
如果无法连接WebSocket：
1. 确认服务已正常启动
2. 检查连接URL是否正确（ws://localhost:8080/api/v1/assistant/stream）
3. 检查发送的消息格式是否正确

## 7. 后续改进方向

- 支持更多文档格式(PDF, HTML等)
- 实现异步并行检索，提高响应速度
- 添加文档自动更新机制
- 实现多语言知识库支持
- 增强流式响应功能，提供更细粒度的反馈
- 添加更多单元测试和集成测试
- 改进缓存机制，提高响应速度
