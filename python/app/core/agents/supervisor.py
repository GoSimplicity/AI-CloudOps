import logging
from typing import Dict, Any, List, Optional
from langchain_core.messages import HumanMessage, BaseMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from pydantic import BaseModel
from typing_extensions import Literal
from app.config.settings import config
from app.services.llm import LLMService
from app.models.data_models import AgentState

logger = logging.getLogger("aiops.supervisor")

class RouteResponse(BaseModel):
    next: Literal["Researcher", "Coder", "K8sFixer", "Notifier", "FINISH"]
    reasoning: Optional[str] = None

class SupervisorAgent:
    def __init__(self):
        # 使用我们自己的LLM服务
        self.llm_service = LLMService()
        self.members = ["Researcher", "Coder", "K8sFixer", "Notifier"]
        self._setup_prompt()
        logger.info("Supervisor Agent初始化完成")
    
    def _setup_prompt(self):
        """设置提示词模板"""
        system_prompt = """你是一个AIOps系统的主管，负责协调以下工作人员来解决Kubernetes相关问题：

工作人员及其职责：
1. Researcher: 负责网络搜索和信息收集，获取相关技术文档和解决方案
2. Coder: 负责执行Python代码，进行数据分析和计算任务
3. K8sFixer: 负责分析和修复Kubernetes部署问题，执行自动化修复操作
4. Notifier: 负责发送通知和警报，联系相关人员

你的任务是：
1. 分析当前问题和上下文
2. 决定下一步应该让哪个工作人员行动
3. 当问题解决完成时返回FINISH

决策原则：
- 如果需要搜索技术信息或最佳实践，选择Researcher
- 如果需要数据分析或复杂计算，选择Coder  
- 如果是Kubernetes部署问题需要修复，选择K8sFixer
- 如果需要发送通知或寻求人工帮助，选择Notifier
- 如果问题已解决或无法继续处理，选择FINISH

请根据当前对话内容和问题状态，决定下一个行动者。"""

        self.prompt_template = system_prompt + """
基于上面的对话历史，决定下一步行动：
- 如果问题需要更多信息，选择 Researcher
- 如果需要代码分析，选择 Coder
- 如果是K8s问题需要修复，选择 K8sFixer  
- 如果需要通知或人工介入，选择 Notifier
- 如果问题已解决，选择 FINISH

从以下选项中选择: ["Researcher", "Coder", "K8sFixer", "Notifier", "FINISH"]
同时简要说明选择理由。
"""
    
    async def route_next_action(self, state: AgentState) -> Dict[str, Any]:
        """决定下一个执行的Agent"""
        try:
            # 检查迭代次数限制
            if state.iteration_count >= state.max_iterations:
                logger.warning(f"达到最大迭代次数限制: {state.max_iterations}")
                return {
                    "next": "FINISH",
                    "reasoning": "达到最大迭代次数限制"
                }
            
            # 构建消息历史文本
            message_history = ""
            for msg in state.messages[-10:]:  # 只保留最近10条消息
                if isinstance(msg, dict):
                    role = msg.get('role', 'user')
                    content = msg.get('content', str(msg))
                    message_history += f"\n{role}: {content}\n"
                elif isinstance(msg, BaseMessage):
                    message_history += f"\n{msg.type}: {msg.content}\n"
                else:
                    message_history += f"\nuser: {str(msg)}\n"
            
            # 构建完整提示词
            full_prompt = f"{self.prompt_template}\n\n对话历史:\n{message_history}"
            
            # 调用LLM服务进行路由决策
            messages = [{"role": "user", "content": full_prompt}]
            response_text = await self.llm_service.generate_response(messages)
            
            if not response_text:
                logger.error("LLM响应为空")
                return {
                    "next": "FINISH",
                    "reasoning": "LLM服务未返回有效响应"
                }
            
            # 解析响应，提取next和reasoning
            next_agent = None
            reasoning = None
            
            # 尝试找出决策结果
            response_text = response_text.strip()
            
            # 如果返回的是JSON格式
            if response_text.startswith("{") and response_text.endswith("}"):
                try:
                    import json
                    parsed = json.loads(response_text)
                    next_agent = parsed.get("next")
                    reasoning = parsed.get("reasoning")
                except:
                    pass
            
            # 如果不是JSON或JSON解析失败，尝试直接从文本中提取
            if not next_agent:
                for member in self.members + ["FINISH"]:
                    if f"next: {member}" in response_text or f'"next": "{member}"' in response_text or f"选择 {member}" in response_text or f"选择: {member}" in response_text:
                        next_agent = member
                        break
                    elif member in response_text:
                        # 如果直接找到了成员名，检查上下文是否表明选择了它
                        surrounding = response_text.split(member)[0][-20:] + response_text.split(member)[1][:20]
                        if "选择" in surrounding or "next" in surrounding:
                            next_agent = member
                            break
            
            # 提取推理
            if not reasoning and "理由" in response_text:
                reasoning_parts = response_text.split("理由")
                if len(reasoning_parts) > 1:
                    reasoning = reasoning_parts[1].strip(": ").strip()
            
            # 如果无法确定，默认为FINISH
            if not next_agent:
                logger.warning(f"无法从响应中确定下一个Agent: {response_text}")
                next_agent = "FINISH"
                reasoning = "无法确定下一步行动"
            
            logger.info(f"Supervisor决策: {next_agent}, 理由: {reasoning}")
            
            return {
                "next": next_agent,
                "reasoning": reasoning or "",
                "iteration_count": state.iteration_count + 1
            }
            
        except Exception as e:
            logger.error(f"Supervisor路由决策失败: {str(e)}")
            return {
                "next": "FINISH",
                "reasoning": f"决策失败: {str(e)}"
            }
    
    async def analyze_problem_context(self, problem_description: str) -> Dict[str, Any]:
        """分析问题上下文"""
        try:
            analysis_prompt = """分析以下问题，提供结构化的问题分析：

问题描述：{problem}

请分析：
1. 问题类型（性能、错误、配置等）
2. 涉及的组件
3. 严重程度
4. 建议的处理方式
5. 需要的工作人员类型

以JSON格式返回分析结果。"""
            
            messages = [{"role": "user", "content": analysis_prompt.format(problem=problem_description)}]
            
            response = await self.llm_service.generate_response(messages)
            
            try:
                import json
                analysis = json.loads(response)
                return analysis
            except json.JSONDecodeError:
                # 尝试从响应中提取JSON
                if "```json" in response:
                    json_part = response.split("```json")[1].split("```")[0]
                    try:
                        analysis = json.loads(json_part)
                        return analysis
                    except:
                        pass
                        
                return {
                    "problem_type": "unknown",
                    "components": ["kubernetes"],
                    "severity": "medium",
                    "suggested_approach": response,
                    "required_agents": ["K8sFixer"]
                }
                
        except Exception as e:
            logger.error(f"问题上下文分析失败: {str(e)}")
            return {
                "problem_type": "unknown",
                "severity": "medium",
                "suggested_approach": "需要进一步分析",
                "required_agents": ["K8sFixer"]
            }
    
    def create_initial_state(self, problem_description: str) -> AgentState:
        """创建初始状态"""
        return AgentState(
            messages=[{
                "role": "user", 
                "content": problem_description,
                "timestamp": "now"
            }],
            current_step="analyzing",
            context={
                "problem": problem_description,
                "start_time": "now"
            },
            iteration_count=0,
            max_iterations=10
        )
    
    def should_continue(self, state: AgentState) -> bool:
        """判断是否应该继续处理"""
        if state.iteration_count >= state.max_iterations:
            return False
        
        if state.next_action == "FINISH":
            return False
        
        # 检查是否有错误循环
        recent_actions = [msg.get('agent') for msg in state.messages[-5:] if isinstance(msg, dict)]
        if len(set(recent_actions)) <= 1 and len(recent_actions) >= 3:
            logger.warning("检测到可能的无限循环，停止处理")
            return False
        
        return True
    
    def get_workflow_summary(self, state: AgentState) -> Dict[str, Any]:
        """获取工作流总结"""
        agents_used = set()
        actions_taken = []
        
        for msg in state.messages:
            if isinstance(msg, dict):
                agent = msg.get('agent')
                if agent:
                    agents_used.add(agent)
                    action = msg.get('action', 'unknown action')
                    actions_taken.append(f"{agent}: {action}")
        
        return {
            'agents_used': list(agents_used),
            'actions_taken': actions_taken,
            'iterations': state.iteration_count,
            'final_step': state.current_step
        }
    
    async def process_agent_state(self, state: AgentState) -> AgentState:
        """处理状态并生成工作流总结
        
        Args:
            state: 当前Agent状态
            
        Returns:
            更新后的状态
        """
        try:
            from dataclasses import replace
            
            # 获取工作流总结
            summary = self.get_workflow_summary(state)
            
            # 获取当前上下文
            context = dict(state.context)
            
            # 添加总结信息
            context['summary'] = f"工作流完成，共使用了 {len(summary['agents_used'])} 个智能体，执行了 {state.iteration_count} 次迭代。"
            context['workflow_summary'] = summary
            context['workflow_completed'] = True
            
            # 根据是否存在错误来设置成功状态
            if 'error' not in context:
                context['success'] = True
                
            # 生成最终总结
            final_status = "成功" if context.get('success', False) else "失败"
            context['result'] = f"自动修复工作流已{final_status}完成"
            
            # 返回更新后的状态
            return replace(state, context=context)
            
        except Exception as e:
            logger.error(f"生成工作流总结失败: {str(e)}")
            # 获取当前上下文
            context = dict(state.context)
            context['error'] = f"生成工作流总结失败: {str(e)}"
            return replace(state, context=context)