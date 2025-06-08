import logging
import json
from typing import Dict, Any, List, Optional
from openai import OpenAI
from app.config.settings import config

logger = logging.getLogger("aiops.llm")

class LLMService:
    def __init__(self):
        self.client = OpenAI(
            api_key=config.llm.api_key,
            base_url=config.llm.base_url
        )
        self.model = config.llm.model
        self.temperature = config.llm.temperature
        self.max_tokens = config.llm.max_tokens
        logger.info(f"LLM服务初始化完成: {self.model}")
    
    async def generate_response(
        self, 
        messages: List[Dict[str, str]], 
        system_prompt: Optional[str] = None,
        response_format: Optional[Dict[str, str]] = None,
        temperature: Optional[float] = None
    ) -> Optional[str]:
        """生成LLM响应"""
        try:
            if system_prompt:
                messages = [{"role": "system", "content": system_prompt}] + messages
            
            kwargs = {
                "model": self.model,
                "messages": messages,
                "temperature": temperature or self.temperature,
                "max_tokens": self.max_tokens
            }
            
            if response_format:
                kwargs["response_format"] = response_format
            
            logger.debug(f"LLM请求: {len(messages)} 条消息")
            response = self.client.chat.completions.create(**kwargs)
            
            result = response.choices[0].message.content
            logger.debug(f"LLM响应长度: {len(result) if result else 0}")
            
            return result
            
        except Exception as e:
            logger.error(f"LLM生成响应失败: {str(e)}")
            return None
    
    async def analyze_k8s_problem(
        self, 
        deployment_yaml: str, 
        error_event: str,
        additional_context: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """分析Kubernetes问题并生成修复方案"""
        try:
            system_prompt = """你是一个Kubernetes专家，专门分析和修复Kubernetes部署问题。
请分析给定的错误信息和部署配置，生成适当的修复方案。
返回格式必须是JSON，包含以下字段：
{
  "analysis": "问题分析描述",
  "root_cause": "根本原因",
  "solution": "解决方案描述",
  "patch": {具体的kubectl patch JSON内容},
  "risk_level": "low/medium/high",
  "additional_actions": ["额外建议的操作"],
  "confidence": 0.8
}"""
            
            context_info = f"\n\n额外上下文：{additional_context}" if additional_context else ""
            
            user_message = f"""
错误事件：{error_event}

当前Deployment YAML：
{deployment_yaml}

{context_info}

请分析问题并提供修复方案。
"""
            
            messages = [{"role": "user", "content": user_message}]
            
            response = await self.generate_response(
                messages=messages,
                system_prompt=system_prompt,
                response_format={"type": "json_object"}
            )
            
            if response:
                try:
                    analysis_result = json.loads(response)
                    logger.info("成功生成K8s问题分析")
                    return analysis_result
                except json.JSONDecodeError as e:
                    logger.error(f"解析LLM JSON响应失败: {str(e)}")
                    return None
            
            return None
            
        except Exception as e:
            logger.error(f"分析K8s问题失败: {str(e)}")
            return None
    
    async def generate_rca_summary(
        self, 
        anomalies: Dict[str, Any], 
        correlations: Dict[str, Any],
        candidates: List[Dict[str, Any]]
    ) -> Optional[str]:
        """生成根因分析摘要"""
        try:
            system_prompt = """你是一个AIOps专家，专门分析系统异常和根因。
基于提供的异常检测结果、相关性分析和候选根因，生成一个清晰、准确的根因分析摘要。
摘要应该：
1. 总结发现的主要异常
2. 解释可能的根本原因
3. 提供具体的建议
4. 使用专业但易懂的语言"""
            
            user_message = f"""
异常检测结果：
{json.dumps(anomalies, indent=2, ensure_ascii=False)}

相关性分析：
{json.dumps(correlations, indent=2, ensure_ascii=False)}

根因候选：
{json.dumps(candidates, indent=2, ensure_ascii=False)}

请生成一个专业的根因分析摘要报告，包含问题总结、根因分析和建议措施。
"""
            
            messages = [{"role": "user", "content": user_message}]
            
            summary = await self.generate_response(
                messages=messages,
                system_prompt=system_prompt
            )
            
            if summary:
                logger.info("成功生成根因分析摘要")
            
            return summary
            
        except Exception as e:
            logger.error(f"生成根因分析摘要失败: {str(e)}")
            return None
    
    async def generate_fix_explanation(
        self, 
        deployment: str, 
        actions_taken: List[str], 
        success: bool
    ) -> Optional[str]:
        """生成修复操作解释"""
        try:
            status = "成功" if success else "失败"
            system_prompt = f"""你是一个运维专家，请解释Kubernetes部署修复操作的结果。
请用专业但易懂的语言解释修复过程和结果。"""
            
            user_message = f"""
部署名称：{deployment}
修复状态：{status}
执行的操作：
{chr(10).join(f'- {action}' for action in actions_taken)}

请解释这次修复操作的内容和影响。
"""
            
            messages = [{"role": "user", "content": user_message}]
            
            explanation = await self.generate_response(
                messages=messages,
                system_prompt=system_prompt
            )
            
            return explanation
            
        except Exception as e:
            logger.error(f"生成修复解释失败: {str(e)}")
            return None
    
    def is_healthy(self) -> bool:
        """检查LLM服务健康状态"""
        try:
            # 简单的健康检查
            test_messages = [{"role": "user", "content": "健康检查"}]
            response = self.client.chat.completions.create(
                model=self.model,
                messages=test_messages,
                max_tokens=10,
                temperature=0
            )
            
            is_healthy = response.choices[0].message.content is not None
            logger.debug(f"LLM健康状态: {is_healthy}")
            return is_healthy
            
        except Exception as e:
            logger.error(f"LLM健康检查失败: {str(e)}")
            return False