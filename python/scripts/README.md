# RAG系统测试指南

我将为您创建一篇关于如何测试整个RAG系统的README.md问答文档。这个文档将指导用户如何测试系统，最终目标是成功运行rag_demo.py。

```markdown:/Users/wangzijian/golangProject/CloudOps/python/README.md
# RAG系统测试指南

本文档提供了关于如何测试和验证RAG（检索增强生成）系统的完整指南。成功的测试标准是能够顺利运行`rag_demo.py`脚本并获得正确的问答响应。

## 目录

1. [系统要求](#系统要求)
2. [环境设置](#环境设置)
3. [测试步骤](#测试步骤)
4. [常见问题](#常见问题)
5. [故障排除](#故障排除)

## 系统要求

在开始测试之前，请确保您的系统满足以下要求：

- Python 3.8或更高版本
- 至少4GB可用内存
- 如果使用Ollama作为LLM提供者，需要安装Ollama并确保服务正在运行
- 如果使用OpenAI作为LLM提供者，需要有效的API密钥

## 环境设置

### 1. 安装依赖

首先，运行设置脚本来安装所有必要的依赖：

```bash
python scripts/setup_rag.py
```

此脚本将：
- 安装所需的Python包
- 创建必要的目录结构
- 设置环境变量
- 创建示例文档

### 2. 配置环境变量

设置脚本会引导您配置环境变量，或者您可以手动设置以下关键环境变量：

- `LLM_PROVIDER`: 选择`ollama`或`openai` (当前版本仅支持ollama)
- `LLM_MODEL`: 模型名称（例如`deepseek-r1:8b`或其他）
- `OPENAI_API_KEY`: 如果使用OpenAI，需要设置API密钥(当前版本仅支持ollama)
- `OLLAMA_HOST`: Ollama服务地址（默认为`http://127.0.0.1:11434`）
- `EMBEDDING_MODEL`: 嵌入模型名称 推荐使用 nomic-embed-text:latest

可以通过创建`.env`文件或在终端中导出这些变量：

```bash
export LLM_PROVIDER=ollama
export LLM_MODEL=deepseek-r1:8b
export OLLAMA_HOST=http://127.0.0.1:11434
```

## 测试步骤

### 1. 诊断系统

首先运行诊断脚本，检查系统是否正确配置：

```bash
python scripts/diagnose_rag.py
```

此脚本将检查：
- 系统信息
- 依赖包状态
- 环境变量
- Ollama服务（如果使用）
- OpenAI API（如果使用）
- 向量存储
- 知识库
- 简单的LLM测试

### 2. 加载知识库

加载您的文档到知识库：

```bash
python scripts/load_knowledge.py --docs-dir ./knowledge_docs
```

参数说明：
- `--docs-dir`: 知识文档目录（默认为`./knowledge_docs`）
- `--persist-dir`: 向量存储持久化目录（默认为`./data/storage/vector_store`）
- `--embedding-model`: 嵌入向量模型名称
- `--embedding-provider`: 嵌入向量提供者（`ollama`或`openai`）
- `--reload`: 强制重新加载文档

### 3. 运行RAG演示

最后，运行RAG演示脚本测试系统：

```bash
python scripts/rag_demo.py --question "v先生的手机号是多少？"
```

参数说明：
- `--provider`: LLM提供者（`ollama`或`openai`）
- `--model`: LLM模型名称
- `--docs-dir`: 知识文档目录
- `--persist-dir`: 向量存储持久化目录
- `--question`: 要提问的问题
- `--reload`: 强制重新加载文档
- `--embedding-provider`: 嵌入向量提供者
- `--embedding-model`: 嵌入向量模型名称

## 常见问题

### Q: 如何确认RAG系统工作正常？
A: 成功运行`rag_demo.py`并获得合理的回答表明系统工作正常。回答应该基于您的知识库内容。

### Q: 我的文档没有被正确加载，怎么办？
A: 使用`--reload`参数重新加载文档：
```bash
python scripts/load_knowledge.py --reload
```

### Q: 如何添加新文档到知识库？
A: 将新文档放入`knowledge_docs`目录，然后运行`load_knowledge.py`脚本重新加载。

### Q: 支持哪些文档格式？
A: 系统支持`.txt`、`.md`、`.pdf`、`.csv`等格式的文档。

## 故障排除

如果您遇到问题，请尝试以下步骤：

1. **检查依赖**：确保所有必要的依赖已正确安装
   ```bash
   python scripts/setup_rag.py --skip-env --skip-dirs
   ```

2. **检查Ollama服务**：如果使用Ollama，确保服务正在运行
   ```bash
   ollama serve
   ```

3. **检查模型可用性**：确保您指定的模型已下载
   ```bash
   ollama list
   ```

4. **清空向量存储**：如果怀疑向量存储有问题，尝试重新加载
   ```bash
   python scripts/rag_demo.py --reload
   ```

5. **查看详细日志**：运行脚本时添加详细日志
   ```bash
   python -m logging scripts/rag_demo.py
   ```

如果问题仍然存在，请查看诊断报告（`rag_diagnosis_report.json`）以获取更多信息。
```