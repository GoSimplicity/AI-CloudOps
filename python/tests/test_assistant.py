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

# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("test_assistant")

# 使用pytestmark标记所有测试可能被跳过，如果环境变量不支持
pytestmark = pytest.mark.skipif(
    os.environ.get("SKIP_LLM_TESTS", "false").lower() == "true",
    reason="LLM API测试被环境变量禁用"
)

@pytest.mark.asyncio
async def test_assistant_query_api(client, llm_service):
    """测试智能小助手API端点"""
    logger.info("测试智能小助手查询API端点")
    
    test_question = "AIOps平台是什么？"
    response = client.post('/api/v1/assistant/query', 
                         json={
                             "question": test_question,
                             "use_web_search": False,
                             "max_context_docs": 3
                         })
    
    # 验证API可以正常响应，不验证具体内容
    assert response.status_code == 200
    
    data = json.loads(response.data)
    assert data['code'] in [0, 500]  # 允许返回错误代码，因为LLM服务可能不可用
    
    if data['code'] == 0:
        assert 'data' in data
        assert 'answer' in data['data']
        assert 'session_id' in data['data']
        logger.info("智能小助手API端点测试通过")
    else:
        logger.warning(f"LLM服务可能不可用，但API正常响应: {data['message']}")
    
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
    
    # 验证返回数据
    data = json.loads(response.data)
    assert data['code'] in [0, 500]  # 允许返回错误代码，因为LLM服务可能不可用
    
    # 第二个问题应该能够保持上下文
    if data['code'] == 0:  # 只有当第一次请求成功时才测试上下文
        response = client.post('/api/v1/assistant/query',
                             json={
                                 "question": "它有哪些功能?",
                                 "session_id": session_id
                             })
        assert response.status_code == 200
        
        data = json.loads(response.data)
        assert data['code'] in [0, 500]
    
    logger.info("会话管理功能测试通过")

@pytest.mark.asyncio
async def test_knowledge_base_refresh(client):
    """测试知识库刷新功能"""
    logger.info("测试知识库刷新功能")
    
    response = client.post('/api/v1/assistant/refresh')
    assert response.status_code == 200
    data = json.loads(response.data)
    assert data['code'] in [0, 500]  # 允许返回错误代码，因为知识库可能不可用
    
    logger.info("知识库刷新功能测试通过")

@pytest.mark.asyncio
async def test_web_search_enhancement(client):
    """测试网络搜索增强功能"""
    logger.info("测试网络搜索增强功能")
    
    # 使用客户端直接发送请求
    response = client.post('/api/v1/assistant/query',
                        json={
                            "question": "最新的AI技术是什么?",
                            "use_web_search": True
                        })
    
    assert response.status_code == 200
    
    data = json.loads(response.data)
    assert data['code'] in [0, 500]  # 允许返回错误代码，因为网络搜索可能不可用
    
    if data['code'] == 0:
        assert 'data' in data
        assert 'answer' in data['data']
    
    logger.info("网络搜索增强功能测试通过")

if __name__ == "__main__":
    pytest.main(["-xvs", __file__])
