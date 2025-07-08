#!/usr/bin/env python
"""
测试智能小助手API端点
"""

import os
import sys
import asyncio
import requests
import logging
from dotenv import load_dotenv

# 添加项目根目录到Python路径
current_path = os.path.dirname(os.path.abspath(__file__))
project_path = os.path.dirname(current_path)
sys.path.append(project_path)

# 加载环境变量
load_dotenv(os.path.join(project_path, ".env"))

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("api_test_assistant")

# API基础URL
API_BASE_URL = "http://127.0.0.1:8080"

def test_assistant_query(question, session_id=None, use_web_search=False):
    """测试智能小助手查询端点"""
    url = f"{API_BASE_URL}/api/v1/assistant/query"
    payload = {
        "question": question,
        "chat_history": [],
        "use_web_search": use_web_search,
        "max_context_docs": 4
    }
    
    # 如果提供了会话ID，添加到请求中
    if session_id:
        payload["session_id"] = session_id
    
    try:
        logger.info(f"发送查询请求: {question}" + (f", 会话ID: {session_id}" if session_id else ""))
        response = requests.post(url, json=payload, timeout=60)  # 增加超时时间以支持网络搜索
        
        if response.status_code == 200:
            result = response.json()
            logger.info(f"状态码: {response.status_code}")
            
            # 输出结果
            print("\n" + "="*50)
            print("问题:", question)
            print("-"*50)
            print("回答:", result["data"]["answer"])
            print("-"*50)
            print("相关性分数:", result["data"]["relevance_score"])
            
            # 显示会话ID
            session_id = result["data"].get("session_id")
            if session_id:
                print(f"会话ID: {session_id}")
            
            # 显示源文档
            source_docs = result["data"].get("source_documents", [])
            if source_docs:
                print(f"\n找到 {len(source_docs)} 个相关文档:")
                for idx, doc in enumerate(source_docs[:3]):  # 只显示前3个
                    source = doc.get("source", "未知来源")
                    is_web = doc.get("is_web_result", False)
                    print(f"  [{idx+1}] {'[网络] ' if is_web else ''}{source}")
            
            # 显示后续问题
            follow_up_questions = result["data"]["follow_up_questions"]
            if follow_up_questions:
                print("\n您可能还想问:")
                for i, q in enumerate(follow_up_questions):
                    print(f"  {i+1}. {q}")
            
            print("="*50)
            return True, result
        else:
            logger.error(f"API调用失败，状态码: {response.status_code}")
            logger.error(f"错误信息: {response.text}")
            return False, None
    
    except requests.exceptions.RequestException as e:
        logger.error(f"请求异常: {str(e)}")
        return False, None
    except Exception as e:
        logger.error(f"发生未知错误: {str(e)}")
        return False, None

def test_refresh_knowledge_base():
    """测试知识库刷新端点"""
    url = f"{API_BASE_URL}/api/v1/assistant/refresh"
    
    try:
        logger.info("发送知识库刷新请求")
        response = requests.post(url, timeout=30)
        
        if response.status_code == 200:
            result = response.json()
            logger.info(f"状态码: {response.status_code}")
            logger.info(f"结果: {result}")
            return True, result
        else:
            logger.error(f"API调用失败，状态码: {response.status_code}")
            logger.error(f"错误信息: {response.text}")
            return False, None
    
    except requests.exceptions.RequestException as e:
        logger.error(f"请求异常: {str(e)}")
        return False, None
    except Exception as e:
        logger.error(f"发生未知错误: {str(e)}")
        return False, None
        
def test_create_session():
    """测试创建会话端点"""
    url = f"{API_BASE_URL}/api/v1/assistant/session"
    
    try:
        logger.info("发送创建会话请求")
        response = requests.post(url, timeout=30)
        
        if response.status_code == 200:
            result = response.json()
            logger.info(f"状态码: {response.status_code}")
            session_id = result["data"]["session_id"]
            logger.info(f"创建的会话ID: {session_id}")
            return True, session_id
        else:
            logger.error(f"API调用失败，状态码: {response.status_code}")
            logger.error(f"错误信息: {response.text}")
            return False, None
    
    except requests.exceptions.RequestException as e:
        logger.error(f"请求异常: {str(e)}")
        return False, None
    except Exception as e:
        logger.error(f"发生未知错误: {str(e)}")
        return False, None
        
def test_conversation_flow():
    """测试对话流程"""
    logger.info("\n--- 对话流程测试 ---")
    
    # 1. 创建会话
    success, session_id = test_create_session()
    if not success or not session_id:
        logger.error("创建会话失败，无法进行对话测试")
        return False
        
    # 2. 进行第一轮对话
    logger.info("\n第一轮对话:")
    success1, _ = test_assistant_query("AIOps平台是什么？", session_id)
    if not success1:
        logger.error("第一轮对话失败")
        return False
        
    # 3. 进行第二轮对话（基于上下文）
    logger.info("\n第二轮对话:")
    success2, _ = test_assistant_query("它有哪些核心功能？", session_id)
    if not success2:
        logger.error("第二轮对话失败")
        return False
        
    # 4. 使用网络搜索
    logger.info("\n第三轮对话(使用网络搜索):")
    success3, _ = test_assistant_query("AIops和DevOps有什么区别？", session_id, use_web_search=True)
    
    return True

def test_add_document(content, metadata=None):
    """测试添加文档端点"""
    url = f"{API_BASE_URL}/api/v1/assistant/add-document"
    
    payload = {
        "content": content,
        "metadata": metadata or {}
    }
    
    try:
        logger.info("发送添加文档请求")
        response = requests.post(url, json=payload, timeout=30)
        
        if response.status_code == 200:
            result = response.json()
            logger.info(f"状态码: {response.status_code}")
            logger.info(f"结果: {result}")
            return True, result
        else:
            logger.error(f"API调用失败，状态码: {response.status_code}")
            logger.error(f"错误信息: {response.text}")
            return False, None
    
    except requests.exceptions.RequestException as e:
        logger.error(f"请求异常: {str(e)}")
        return False, None
    except Exception as e:
        logger.error(f"发生未知错误: {str(e)}")
        return False, None

def test_clear_cache():
    """测试清除缓存端点"""
    url = f"{API_BASE_URL}/api/v1/assistant/clear-cache"
    
    try:
        logger.info("发送清除缓存请求")
        response = requests.post(url, timeout=10)
        
        if response.status_code == 200:
            result = response.json()
            logger.info(f"状态码: {response.status_code}")
            logger.info(f"结果: {result}")
            return True, result
        else:
            logger.error(f"API调用失败，状态码: {response.status_code}")
            logger.error(f"错误信息: {response.text}")
            return False, None
    
    except requests.exceptions.RequestException as e:
        logger.error(f"请求异常: {str(e)}")
        return False, None
    except Exception as e:
        logger.error(f"发生未知错误: {str(e)}")
        return False, None

async def main():
    """主函数"""
    try:
        # 准备一个简单的知识库文档
        new_document = """
# AIOps云运维专家系统
        
云运维专家系统是一个集成了自动化、智能化运维功能的系统，可以帮助企业有效管理云资源。
        
## 功能特点

1. **资源监控**: 实时监控云资源使用情况
2. **成本分析**: 提供详细的成本分析和优化建议
3. **自动伸缩**: 根据负载情况自动调整资源配置
4. **异常检测**: 智能检测系统异常并发出警报
        
## 联系方式

如有问题，请联系我们的技术支持团队：
        
- 技术支持: support@aiops.com, 电话: 400-123-4567
- 销售咨询: sales@aiops.com, 电话: 400-123-4568
        """
        
        # 测试流程
        logger.info("开始测试智能小助手API")
        
        # 首先清除缓存
        logger.info("\n--- 初始步骤: 清除缓存 ---")
        test_clear_cache()
        
        # 第1步：测试查询 - 预期失败或返回通用回答
        logger.info("\n--- 步骤1: 测试初始查询 ---")
        success, _ = test_assistant_query("AIOps云运维专家系统是什么？")
        
        # 第2步：添加文档
        logger.info("\n--- 步骤2: 添加知识库文档 ---")
        document_success, _ = test_add_document(new_document, {"source": "api_test"})
        if not document_success:
            logger.error("文档添加失败，跳过后续测试")
            return
        
        # 第3步：刷新知识库
        logger.info("\n--- 步骤3: 刷新知识库 ---")
        refresh_success, _ = test_refresh_knowledge_base()
        if not refresh_success:
            logger.error("知识库刷新失败，跳过后续测试")
            return
            
        # 再次清除缓存
        logger.info("再次清除缓存...")
        test_clear_cache()
        
        # 添加等待时间确保向量数据库完全初始化
        logger.info("等待向量数据库初始化...")
        await asyncio.sleep(2)  # 等待2秒让向量数据库初始化完成
        
        # 第4步：再次测试查询 - 预期成功
        logger.info("\n--- 步骤4: 测试添加文档后的查询 ---")
        success4, result4 = test_assistant_query("AIOps云运维专家系统是什么？")
        
        # 输出更详细的调试信息
        if success4 and "提供的文档中没有这些信息" in result4["data"]["answer"]:
            logger.warning("仍然无法检索到相关信息，尝试清除缓存并重新发送查询...")
            # 再次刷新知识库
            refresh_success, _ = test_refresh_knowledge_base()
            await asyncio.sleep(2)  # 再等待2秒
            success4, result4 = test_assistant_query("AIOps云运维专家系统是什么？")
        
        # 第5步：测试联系方式查询
        logger.info("\n--- 步骤5: 测试联系方式查询 ---")
        success5, result5 = test_assistant_query("技术支持的联系方式是什么？")
        
        # 输出更详细的调试信息
        if success5 and "提供的文档中没有这些信息" in result5["data"]["answer"]:
            logger.warning("联系方式查询仍然无法检索到相关信息...")
        
        # 第6步：创建会话测试
        logger.info("\n--- 步骤6: 测试创建会话 ---")
        session_success, session_id = test_create_session()
        if not session_success:
            logger.error("创建会话失败，跳过会话测试")
        else:
            # 第7步：使用会话ID测试查询
            logger.info("\n--- 步骤7: 使用会话ID测试查询 ---")
            test_assistant_query("AIOps平台有哪些核心功能？", session_id)
            
            # 第8步：测试上下文相关问题
            logger.info("\n--- 步骤8: 测试上下文相关问题 ---")
            test_assistant_query("这些功能如何帮助企业提高效率？", session_id)
            
        # 第9步：测试完整对话流程
        logger.info("\n--- 步骤9: 测试完整对话流程 ---")
        test_conversation_flow()
        
        # 第10步：测试带网络搜索的查询
        logger.info("\n--- 步骤10: 测试带网络搜索的查询 ---")
        test_assistant_query("什么是 MLOps?", use_web_search=True)
        
        logger.info("\n智能小助手API测试完成")
        
    except Exception as e:
        logger.error(f"测试过程中出错: {str(e)}")
        import traceback
        logger.error(traceback.format_exc())

if __name__ == "__main__":
    asyncio.run(main())
