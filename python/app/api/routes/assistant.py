"""
智能小助手API路由
"""

import asyncio
import logging
import json
import sys
from datetime import datetime
from flask import Blueprint, request, jsonify

# 创建日志器
logger = logging.getLogger("aiops.api.assistant")

# 创建蓝图
assistant_bp = Blueprint('assistant', __name__, url_prefix='')

# 尝试导入Flask-Sock
try:
    from flask_sock import Sock
    # 创建WebSocket对象
    sock = Sock()
    WEBSOCKET_AVAILABLE = True
except ImportError:
    logger.warning("Flask-Sock模块未安装，WebSocket功能将不可用")
    WEBSOCKET_AVAILABLE = False
    sock = None

# 创建助手代理全局实例
_assistant_agent = None

def get_assistant_agent():
    """获取助手代理单例实例"""
    global _assistant_agent
    if _assistant_agent is None:
        try:
            logger.info("初始化智能小助手代理...")
            from app.core.agents.assistant import AssistantAgent
            _assistant_agent = AssistantAgent()
        except Exception as e:
            logger.error(f"初始化智能小助手代理失败: {str(e)}")
            return None
    return _assistant_agent

def init_websocket(app):
    """初始化WebSocket"""
    if WEBSOCKET_AVAILABLE and sock is not None:
        sock.init_app(app)
        logger.info("已初始化WebSocket服务")
    else:
        logger.warning("WebSocket功能不可用，相关接口将不能使用")

def safe_async_run(coroutine):
    """安全地运行异步函数，处理不同环境下的运行方式"""
    try:
        # 检查当前是否在事件循环中
        loop = asyncio.get_event_loop()
        if loop.is_running():
            # 在有事件循环的环境中，尝试等待协程
            return loop.run_until_complete(coroutine)
        else:
            # 没有事件循环，使用asyncio.run
            return asyncio.run(coroutine)
    except Exception as e:
        logger.error(f"执行异步函数失败: {str(e)}")
        raise e


@assistant_bp.route('/query', methods=['POST'])
def assistant_query():
    """智能小助手查询API - 同步包装异步函数"""
    try:
        data = request.json
        question = data.get('question', '')
        use_web_search = data.get('use_web_search', False)
        session_id = data.get('session_id')
        max_context_docs = data.get('max_context_docs', 4)
        
        if not question:
            return jsonify({
                'code': 400,
                'message': '问题不能为空',
                'data': {}
            }), 400
        
        agent = get_assistant_agent()
        if not agent:
            return jsonify({
                'code': 500,
                'message': '智能小助手服务未正确初始化',
                'data': {}
            }), 500
        
        # 调用助手代理获取回答
        try:
            result = safe_async_run(agent.get_answer(
                question=question,
                use_web_search=use_web_search,
                session_id=session_id,
                max_context_docs=max_context_docs
            ))
        except Exception as e:
            logger.error(f"获取回答失败: {str(e)}")
            return jsonify({
                'code': 500,
                'message': f'获取回答时出错: {str(e)}',
                'data': {}
            }), 500
        
        # 生成会话ID（如果不存在）
        if not session_id:
            session_id = agent.create_session()
        
        return jsonify({
            'code': 0,
            'message': '查询成功',
            'data': {
                'answer': result['answer'],
                'session_id': session_id,
                'relevance_score': result.get('relevance_score'),
                'recall_rate': result.get('recall_rate', 0.0),
                'sources': result.get('source_documents', []),
                'follow_up_questions': result.get('follow_up_questions', []),
                'timestamp': datetime.now().isoformat()
            }
        })
    except Exception as e:
        logger.error(f"查询处理失败: {str(e)}")
        return jsonify({
            'code': 500,
            'message': f'处理查询时出错: {str(e)}',
            'data': {}
        }), 500


@assistant_bp.route('/session', methods=['POST'])
def create_session():
    """创建新会话 - 同步包装异步函数"""
    try:
        agent = get_assistant_agent()
        if not agent:
            return jsonify({
                'code': 500,
                'message': '智能小助手服务未正确初始化',
                'data': {}
            }), 500
        
        session_id = agent.create_session()
        
        return jsonify({
            'code': 0,
            'message': '会话创建成功',
            'data': {
                'session_id': session_id,
                'timestamp': datetime.now().isoformat()
            }
        })
    except Exception as e:
        logger.error(f"创建会话失败: {str(e)}")
        return jsonify({
            'code': 500,
            'message': f'创建会话时出错: {str(e)}',
            'data': {}
        }), 500


@assistant_bp.route('/refresh', methods=['POST'])
def refresh_knowledge_base():
    """刷新知识库 - 同步包装异步函数"""
    try:
        agent = get_assistant_agent()
        if not agent:
            return jsonify({
                'code': 500,
                'message': '智能小助手服务未正确初始化',
                'data': {}
            }), 500
        
        try:
            result = safe_async_run(agent.refresh_knowledge_base())
        except Exception as e:
            logger.error(f"刷新知识库失败: {str(e)}")
            return jsonify({
                'code': 500,
                'message': f'刷新知识库时出错: {str(e)}',
                'data': {}
            }), 500
        
        return jsonify({
            'code': 0,
            'message': '知识库刷新成功',
            'data': {
                'documents_count': result.get('documents_count', 0),
                'timestamp': datetime.now().isoformat()
            }
        })
    except Exception as e:
        logger.error(f"刷新知识库失败: {str(e)}")
        return jsonify({
            'code': 500,
            'message': f'刷新知识库时出错: {str(e)}',
            'data': {}
        }), 500
