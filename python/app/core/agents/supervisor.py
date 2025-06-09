import logging
from typing import Dict, Any, List, Optional
from langchain_core.messages import HumanMessage, BaseMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_openai import ChatOpenAI
from pydantic import BaseModel
from typing_extensions import Literal
from app.config.settings import config
from app.models.data_models import AgentState

logger = logging.getLogger("aiops.supervisor")

class RouteResponse(BaseModel):
    next: Literal["Researcher", "Coder", "K8sFixer", "Notifier", "FINISH"]
    reasoning: Optional[str] = None

class SupervisorAgent:
    def __init__(self):
        self.llm = ChatOpenAI(
            model=config.llm.model,
            api_key=config.llm.api_key,
            base_url=config.llm.base_url,
            temperature=config.llm.temperature
        )
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

        self.prompt = ChatPromptTemplate.from_messages([
            ("system", system_prompt),
            MessagesPlaceholder(variable_name="messages"),
            ("system", """
基于上面的对话历史，决定下一步行动：
- 如果问题需要更多信息，选择 Researcher
- 如果需要代码分析，选择 Coder
- 如果是K8s问题需要修复，选择 K8sFixer  
- 如果需要通知或人工介入，选择 Notifier
- 如果问题已解决，选择 FINISH

从以下选项中选择: {options}
同时简要说明选择理由。
            """)
        ]).partial(options=str(["Researcher", "Coder", "K8sFixer", "Notifier", "FINISH"]))
    
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
            
            # 构建消息历史
            messages = []
            for msg in state.messages[-10:]:  # 只保留最近10条消息
                if isinstance(msg, dict):
                    content = msg.get('content', str(msg))
                    role = msg.get('role', 'user')
                    messages.append(HumanMessage(content=content))
                elif isinstance(msg, BaseMessage):
                    messages.append(msg)
                else:
                    messages.append(HumanMessage(content=str(msg)))
            
            # 调用LLM进行路由决策
            chain = self.prompt | self.llm.with_structured_output(RouteResponse)
            response = chain.invoke({"messages": messages})
            
            logger.info(f"Supervisor决策: {response.next}, 理由: {response.reasoning}")
            
            return {
                "next": response.next,
                "reasoning": response.reasoning or "",
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
            
            messages = [HumanMessage(content=analysis_prompt.format(problem=problem_description))]
            
            response = await self.llm.ainvoke(messages)
            
            try:
                import json
                analysis = json.loads(response.content)
                return analysis
            except json.JSONDecodeError:
                return {
                    "problem_type": "unknown",
                    "components": ["kubernetes"],
                    "severity": "medium",
                    "suggested_approach": response.content,
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
            "total_iterations": state.iteration_count,
            "agents_used": list(agents_used),
            "actions_taken": actions_taken,
            "final_state": state.current_step,
            "context": state.context
        }