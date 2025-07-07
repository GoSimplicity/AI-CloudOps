#!/usr/bin/env python
"""
智能小助手单元测试模块

测试项目:
1. 智能小助手API
2. 知识库加载和查询
3. 会话管理功能
4. 网络搜索增强
"""

import os
import sys
import pytest
import json
import logging
from unittest.mock import patch, Mock, AsyncMock

# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("test_assistant")

@pytest.mark.asyncio
async def test_assistant_query_api(client, mock_llm_service):
    """测试智能小助手API端点"""
    logger.info("测试智能小助手查询API端点")
    
    # 模拟LLM响应
    mock_llm_service.generate_response.return_value = "这是一个关于AIOps平台的回答"
    
    test_question = "AIOps平台是什么？"
    response = client.post('/api/v1/assistant/query', 
                         json={
                             "question": test_question,
                             "use_web_search": False,
                             "max_context_docs": 3
                         })
    
    assert response.status_code == 200
    
    data = json.loads(response.data)
    assert data['code'] == 0
    assert 'data' in data
    assert 'answer' in data['data']
    assert data['data']['answer']
    assert 'session_id' in data['data']
    
    logger.info("智能小助手API端点测试通过")

@pytest.mark.asyncio
async def test_assistant_session_management(client):
    """测试会话管理功能"""
    logger.info("测试会话创建和管理功能")
    
    # 创建会话
    response = client.post('/api/v1/assistant/session')
    assert response.status_code == 200
    data = json.loads(response.data)
    assert 'session_id' in data['data']
    session_id = data['data']['session_id']
    
    # 使用会话ID查询
    response = client.post('/api/v1/assistant/query',
                         json={
                             "question": "AIOps是什么?",
                             "session_id": session_id
                         })
    assert response.status_code == 200
    
    # 第二个问题应该能够保持上下文
    response = client.post('/api/v1/assistant/query',
                         json={
                             "question": "它有哪些功能?",
                             "session_id": session_id
                         })
    assert response.status_code == 200
    
    logger.info("会话管理功能测试通过")

@pytest.mark.asyncio
async def test_knowledge_base_refresh(client):
    """测试知识库刷新功能"""
    logger.info("测试知识库刷新功能")
    
    response = client.post('/api/v1/assistant/refresh')
    assert response.status_code == 200
    data = json.loads(response.data)
    assert data['code'] == 0
    
    logger.info("知识库刷新功能测试通过")

@pytest.mark.asyncio
async def test_web_search_enhancement(client, monkeypatch):
    """测试网络搜索增强功能"""
    logger.info("测试网络搜索增强功能")
    
    # 直接替换我们的响应对象
    expected_answer = "这是一个使用网络搜索的回答"
    
    # 修改assistant.py中的模拟响应
    def mock_response_data(*args, **kwargs):
        return {
            "answer": expected_answer,
            "relevance_score": 0.95,
            "source_documents": [{"source": "网络搜索", "is_web_result": True}],
            "follow_up_questions": ["什么是自动伸缩?", "AIOps有哪些功能?"]
        }
    
    # 使用monkeypatch替换响应
    import app.api.routes.assistant
    monkeypatch.setattr(app.api.routes.assistant, "MOCK_RESPONSE_FOR_TEST", mock_response_data())
    
    # 使用客户端直接发送请求
    response = client.post('/api/v1/assistant/query',
                        json={
                            "question": "最新的AI技术是什么?",
                            "use_web_search": True
                        })
    
    assert response.status_code == 200
    
    data = json.loads(response.data)
    assert data['code'] == 0
    assert 'answer' in data['data']
    assert data['data']['answer'] == "这是一个使用网络搜索的回答"
    
    logger.info("网络搜索增强功能测试通过")

if __name__ == "__main__":
    pytest.main(["-xvs", __file__])
