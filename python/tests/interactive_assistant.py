#!/usr/bin/env python
"""
交互式智能小助手测试工具
"""

import os
import sys
import asyncio
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
logger = logging.getLogger("interactive_assistant")

class InteractiveAssistant:
    """交互式智能小助手"""
    
    def __init__(self):
        """初始化交互式助手"""
        from app.core.agents.assistant import AssistantAgent
        self.agent = AssistantAgent()
        self.session_id = None
        logger.info("智能小助手初始化完成")
    
    async def start(self):
        """启动交互式会话"""
        print("\n" + "="*50)
        print("交互式智能小助手")
        print("输入 'exit' 或 'quit' 退出")
        print("输入 'refresh' 刷新知识库")
        print("输入 'new' 开始新会话")
        print("输入 'clear' 清除当前会话历史")
        print("输入 'web on/off' 开启/关闭网络搜索")
        print("输入 'help' 查看帮助")
        print("="*50 + "\n")
        
        # 创建初始会话
        self.session_id = self.agent.create_session()
        print(f"已创建新会话，ID: {self.session_id[:8]}...")
        
        # 网络搜索开关
        use_web_search = False
        
        # 循环处理用户输入
        while True:
            try:
                # 获取用户输入
                question = input("\n问题: ")
                
                # 处理特殊命令
                if question.lower() in ['exit', 'quit']:
                    print("感谢使用，再见！")
                    break
                elif question.lower() == 'refresh':
                    print("刷新知识库...")
                    if self.agent.refresh_knowledge_base():
                        print("知识库刷新成功！")
                    else:
                        print("知识库刷新失败！")
                    continue
                elif question.lower() == 'new':
                    self.session_id = self.agent.create_session()
                    print(f"已创建新会话，ID: {self.session_id[:8]}...")
                    continue
                elif question.lower() == 'clear':
                    if self.agent.clear_session_history(self.session_id):
                        print("会话历史已清除")
                    else:
                        print("清除会话历史失败")
                    continue
                elif question.lower() == 'web on':
                    use_web_search = True
                    print("已开启网络搜索")
                    continue
                elif question.lower() == 'web off':
                    use_web_search = False
                    print("已关闭网络搜索")
                    continue
                elif question.lower() == 'help':
                    print("\n" + "="*50)
                    print("帮助信息:")
                    print("exit/quit - 退出程序")
                    print("refresh - 刷新知识库")
                    print("new - 开始新会话")
                    print("clear - 清除当前会话历史")
                    print("web on/off - 开启/关闭网络搜索")
                    print("help - 显示此帮助信息")
                    print("="*50)
                    continue
                elif not question.strip():
                    continue
                
                # 获取回答
                print(f"思考中... (会话ID: {self.session_id[:8]}...)")
                result = await self.agent.get_answer(
                    question=question,
                    session_id=self.session_id,
                    use_web_search=use_web_search
                )
                
                # 显示回答
                print("\n回答:\n" + "="*50)
                print(result["answer"])
                print("="*50)
                
                # 显示相关性分数
                print(f"相关性分数: {result['relevance_score']}")
                
                # 显示源文档信息
                if result['source_documents']:
                    print(f"找到 {len(result['source_documents'])} 个相关文档:")
                    for idx, doc in enumerate(result['source_documents'][:3]):  # 只显示前3个
                        source = doc.get('source', '未知来源')
                        is_web = doc.get('is_web_result', False)
                        print(f"  [{idx+1}] {'[网络] ' if is_web else ''}{source}")
                
                # 显示后续问题建议
                if result['follow_up_questions']:
                    print("\n您可能还想问:")
                    for i, q in enumerate(result['follow_up_questions']):
                        print(f"  {i+1}. {q}")
                
            except KeyboardInterrupt:
                print("\n程序被中断，退出中...")
                break
            except Exception as e:
                print(f"发生错误: {str(e)}")

async def main():
    """主函数"""
    try:
        assistant = InteractiveAssistant()
        await assistant.start()
    except Exception as e:
        logger.error(f"程序出错: {str(e)}")
        import traceback
        logger.error(traceback.format_exc())

if __name__ == "__main__":
    print("启动交互式智能小助手...")
    asyncio.run(main()) 