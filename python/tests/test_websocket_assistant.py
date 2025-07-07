#!/usr/bin/env python
"""
WebSocket智能小助手测试模块

测试项目:
1. WebSocket连接建立
2. 消息发送
3. 流式响应接收
4. 会话管理
"""

import os
import sys
import pytest
import json
import asyncio
import websockets
import logging
from pathlib import Path

# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("test_websocket_assistant")

# WebSocket连接配置
WS_URL = "ws://localhost:8080/api/v1/assistant/stream"
TIMEOUT = 30  # 连接超时时间(秒)

@pytest.mark.asyncio
async def test_websocket_connection():
    """测试WebSocket连接建立"""
    logger.info("测试WebSocket连接建立")
    
    try:
        async with websockets.connect(WS_URL, timeout=TIMEOUT) as websocket:
            logger.info("WebSocket连接建立成功")
            assert websocket.open
    except Exception as e:
        logger.error(f"WebSocket连接建立失败: {str(e)}")
        pytest.fail(f"WebSocket连接测试失败: {str(e)}")

@pytest.mark.asyncio
async def test_websocket_simple_query():
    """测试简单查询"""
    logger.info("测试WebSocket简单查询")
    
    try:
        async with websockets.connect(WS_URL, timeout=TIMEOUT) as websocket:
            # 发送查询
            query = {
                "question": "AIOps平台是什么？",
                "use_web_search": False
            }
            await websocket.send(json.dumps(query))
            
            # 接收响应
            content_received = False
            metadata_received = False
            full_response = ""
            
            # 设置超时
            start_time = asyncio.get_event_loop().time()
            
            while True:
                if asyncio.get_event_loop().time() - start_time > TIMEOUT:
                    pytest.fail("WebSocket响应超时")
                
                try:
                    # 设置接收超时
                    response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                    data = json.loads(response)
                    
                    if data.get("type") == "content":
                        content_received = True
                        full_response += data.get("content", "")
                        logger.info(f"收到内容片段: {data.get('content')[:30]}...")
                    
                    elif data.get("type") == "end":
                        metadata_received = True
                        logger.info(f"收到元数据: {data}")
                        break
                except asyncio.TimeoutError:
                    logger.warning("等待响应超时，可能流式传输已结束")
                    break
            
            # 验证响应
            assert content_received, "未接收到内容数据"
            assert metadata_received, "未接收到元数据"
            assert len(full_response) > 0, "接收到的响应内容为空"
            
            logger.info(f"完整响应长度: {len(full_response)} 字符")
            logger.info("WebSocket简单查询测试通过")
            
    except Exception as e:
        logger.error(f"WebSocket查询测试失败: {str(e)}")
        pytest.fail(f"WebSocket查询测试失败: {str(e)}")

@pytest.mark.asyncio
async def test_websocket_session_management():
    """测试WebSocket会话管理"""
    logger.info("测试WebSocket会话管理")
    
    try:
        session_id = None
        
        # 第一个连接 - 获取会话ID
        async with websockets.connect(WS_URL, timeout=TIMEOUT) as websocket1:
            # 发送第一个查询
            query1 = {
                "question": "AIOps平台有哪些功能？",
                "use_web_search": False
            }
            await websocket1.send(json.dumps(query1))
            
            # 接收响应并获取会话ID
            while True:
                response = await asyncio.wait_for(websocket1.recv(), timeout=10.0)
                data = json.loads(response)
                
                if data.get("type") == "end":
                    session_id = data.get("metadata", {}).get("session_id")
                    logger.info(f"获取到会话ID: {session_id}")
                    assert session_id, "未获取到有效的会话ID"
                    break
        
        # 第二个连接 - 使用同一会话ID
        async with websockets.connect(WS_URL, timeout=TIMEOUT) as websocket2:
            # 发送相关的后续问题
            query2 = {
                "question": "智能根因分析功能具体是什么？",
                "session_id": session_id,
                "use_web_search": False
            }
            await websocket2.send(json.dumps(query2))
            
            # 接收响应
            content_received = False
            end_received = False
            
            while True:
                try:
                    response = await asyncio.wait_for(websocket2.recv(), timeout=5.0)
                    data = json.loads(response)
                    
                    if data.get("type") == "content":
                        content_received = True
                    
                    elif data.get("type") == "end":
                        end_received = True
                        # 验证返回的会话ID与发送的一致
                        returned_session_id = data.get("metadata", {}).get("session_id")
                        assert returned_session_id == session_id, "返回的会话ID与发送的不一致"
                        break
                except asyncio.TimeoutError:
                    break
            
            # 验证响应
            assert content_received, "未接收到内容数据"
            assert end_received, "未接收到结束标记"
            
            logger.info("WebSocket会话管理测试通过")
            
    except Exception as e:
        logger.error(f"WebSocket会话管理测试失败: {str(e)}")
        pytest.fail(f"WebSocket会话管理测试失败: {str(e)}")

if __name__ == "__main__":
    pytest.main(["-xvs", __file__])