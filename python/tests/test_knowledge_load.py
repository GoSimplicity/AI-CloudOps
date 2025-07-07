#!/usr/bin/env python
"""
知识库加载测试模块

测试项目:
1. 知识库文件加载
2. 向量数据库初始化和查询
3. 文档分块处理
4. 嵌入模型功能
"""

import os
import sys
import pytest
import logging
from pathlib import Path
import shutil
import tempfile

# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("test_knowledge_load")

@pytest.fixture(scope="module")
def temp_knowledge_base():
    """创建临时知识库目录"""
    temp_dir = tempfile.mkdtemp()
    yield temp_dir
    shutil.rmtree(temp_dir)

@pytest.fixture(scope="module")
def temp_vector_db():
    """创建临时向量数据库目录"""
    temp_dir = tempfile.mkdtemp()
    yield temp_dir
    shutil.rmtree(temp_dir)

@pytest.fixture
def sample_document():
    """创建示例文档内容"""
    return """
# AIOps平台说明文档

## 简介

AIOps平台是一个智能运维系统，提供根因分析、自动修复和负载预测功能。

## 核心功能

1. 智能根因分析
2. Kubernetes自动修复
3. 基于机器学习的负载预测

## 系统架构

AIOps平台采用微服务架构，包括API网关、核心业务逻辑和服务层。

## 联系方式

如有问题请联系开发团队：support@example.com
"""

def test_document_loader(temp_knowledge_base, sample_document):
    """测试文档加载器功能"""
    logger.info("测试文档加载器功能")
    
    # 导入文档加载器
    from app.core.agents.assistant import DocumentLoader
    
    # 创建测试文档
    doc_path = os.path.join(temp_knowledge_base, "test_doc.md")
    with open(doc_path, "w", encoding="utf-8") as f:
        f.write(sample_document)
    
    # 初始化加载器
    loader = DocumentLoader(knowledge_base_path=temp_knowledge_base)
    
    # 测试加载文档
    documents = loader.load()
    
    # 验证结果
    assert len(documents) > 0, "文档加载失败"
    assert any("AIOps平台" in doc.page_content for doc in documents), "文档内容加载错误"
    
    logger.info(f"成功加载 {len(documents)} 个文档")

def test_document_splitting():
    """测试文档分块功能"""
    logger.info("测试文档分块功能")
    
    # 导入文档处理工具
    from app.core.agents.assistant import DocumentSplitter
    from langchain_core.documents import Document
    
    # 创建测试文档
    test_doc = Document(
        page_content="这是第一段内容。\n\n这是第二段内容。\n\n这是第三段内容，比较长，包含了更多的信息。",
        metadata={"source": "测试文档"}
    )
    
    # 测试不同的分块大小
    for chunk_size in [10, 20, 50]:
        splitter = DocumentSplitter(chunk_size=chunk_size, chunk_overlap=5)
        chunks = splitter.split_documents([test_doc])
        
        # 验证结果
        assert len(chunks) > 0, f"使用块大小 {chunk_size} 的分块失败"
        logger.info(f"块大小 {chunk_size}: 生成了 {len(chunks)} 个块")

def test_vector_database(temp_vector_db, sample_document):
    """测试向量数据库功能"""
    logger.info("测试向量数据库功能")
    
    try:
        from app.core.agents.assistant import VectorDatabaseManager
        from langchain_core.documents import Document
        
        # 创建测试文档
        test_doc = Document(
            page_content=sample_document,
            metadata={"source": "测试文档"}
        )
        
        # 初始化向量数据库管理器
        db_manager = VectorDatabaseManager(
            vector_db_path=temp_vector_db,
            collection_name="test_collection"
        )
        
        # 测试添加文档
        db_manager.add_documents([test_doc])
        
        # 测试相似度搜索
        results = db_manager.similarity_search("AIOps平台的核心功能是什么？", k=2)
        
        # 验证结果
        assert len(results) > 0, "向量搜索失败"
        assert any("核心功能" in doc.page_content for doc in results), "搜索结果相关性不足"
        
        logger.info(f"成功获取 {len(results)} 个相关文档片段")
        
    except ImportError as e:
        logger.warning(f"无法导入向量数据库组件: {str(e)}")
        pytest.skip("向量数据库组件不可用")
    except Exception as e:
        logger.error(f"向量数据库测试失败: {str(e)}")
        raise

def test_knowledge_base_integration(temp_knowledge_base, temp_vector_db):
    """测试知识库集成功能"""
    logger.info("测试知识库集成功能")
    
    try:
        from app.config.settings import config
        import os
        
        # 临时修改配置
        original_kb_path = config.rag.knowledge_base_path
        original_vdb_path = config.rag.vector_db_path
        
        os.environ["RAG_KNOWLEDGE_BASE_PATH"] = temp_knowledge_base
        os.environ["RAG_VECTOR_DB_PATH"] = temp_vector_db
        
        # 重新加载配置
        from app.core.agents.assistant import AssistantAgent
        
        # 创建示例文档
        doc_path = os.path.join(temp_knowledge_base, "integration_test.md")
        with open(doc_path, "w", encoding="utf-8") as f:
            f.write("# AIOps集成测试\n\nAIOps平台集成测试文档，用于验证知识库功能。")
        
        # 初始化助手
        agent = AssistantAgent()
        
        # 刷新知识库
        result = agent.refresh_knowledge_base()
        assert result, "知识库刷新失败"
        
        logger.info("知识库集成测试通过")
        
        # 恢复环境变量
        if original_kb_path:
            os.environ["RAG_KNOWLEDGE_BASE_PATH"] = original_kb_path
        else:
            os.environ.pop("RAG_KNOWLEDGE_BASE_PATH", None)
            
        if original_vdb_path:
            os.environ["RAG_VECTOR_DB_PATH"] = original_vdb_path
        else:
            os.environ.pop("RAG_VECTOR_DB_PATH", None)
            
    except ImportError as e:
        logger.warning(f"无法导入助手组件: {str(e)}")
        pytest.skip("助手组件不可用")
    except Exception as e:
        logger.error(f"知识库集成测试失败: {str(e)}")
        raise

if __name__ == "__main__":
    pytest.main(["-xvs", __file__])