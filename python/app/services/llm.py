import logging
import json
import os
import re
from typing import Dict, Any, List, Optional, Union, Tuple
from openai import OpenAI
import ollama

from app.config.settings import config
from app.constants import (
    LLM_TIMEOUT_SECONDS, LLM_MAX_RETRIES, OPENAI_TEST_MAX_TOKENS,
    LLM_CONFIDENCE_THRESHOLD, LLM_TEMPERATURE_MIN, LLM_TEMPERATURE_MAX
)
from app.utils.error_handlers import (
    ErrorHandler, ServiceError, ValidationError, ExternalServiceError,
    retry_on_exception, validate_field_type, validate_field_range
)

logger = logging.getLogger("aiops.llm")

class LLMService:
    """LLM 服务管理类，支持 OpenAI 和 Ollama 提供商"""
    
    def __init__(self):
        """
        初始化LLM服务，支持OpenAI和Ollama
        系统会优先使用外部模型(OpenAI)，如果不可用则自动回退到本地模型(Ollama)
        """
        self.error_handler = ErrorHandler(logger)
        
        # 清理提供商字符串，移除可能的注释
        self.provider = config.llm.provider.split('#')[0].strip() if config.llm.provider else "openai"
        self.model = config.llm.effective_model
        self.temperature = self._validate_temperature(config.llm.temperature)
        self.max_tokens = config.llm.max_tokens
        
        # 初始化备用提供商和模型
        self.backup_provider = "ollama" if self.provider.lower() == "openai" else "openai"
        self.backup_model = config.llm.ollama_model if self.backup_provider == "ollama" else config.llm.model
        
        if self.provider.lower() == "openai":
            # 使用OpenAI兼容的API
            self.client = OpenAI(
                api_key=config.llm.effective_api_key,
                base_url=config.llm.effective_base_url
            )
            logger.info(f"LLM服务(OpenAI)初始化完成: {self.model}")
            
            # 预初始化备用Ollama客户端 - 无需设置host，使用环境变量
            try:
                # 使用环境变量设置Ollama主机地址
                os.environ["OLLAMA_HOST"] = config.llm.ollama_base_url.replace("/v1", "")
                logger.info(f"备用LLM服务(Ollama)初始化完成: {config.llm.ollama_model}, OLLAMA_HOST={os.environ.get('OLLAMA_HOST')}")
            except Exception as e:
                logger.warning(f"备用Ollama初始化失败: {str(e)}")
        
        elif self.provider.lower() == "ollama":
            # 使用Ollama客户端
            self.client = None  # Ollama使用独立的API调用
            # 使用环境变量设置Ollama主机地址
            os.environ["OLLAMA_HOST"] = config.llm.ollama_base_url.replace("/v1", "")
            logger.info(f"LLM服务(Ollama)初始化完成: {self.model}, OLLAMA_HOST={os.environ.get('OLLAMA_HOST')}")
            
            # 预初始化备用OpenAI客户端
            try:
                self.backup_client = OpenAI(
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url
                )
                logger.info(f"备用LLM服务(OpenAI)初始化完成: {config.llm.model}")
            except Exception as e:
                logger.warning(f"备用OpenAI初始化失败: {str(e)}")
        else:
            raise ValidationError(f"不支持的LLM提供商: {self.provider}")
    
    def _validate_temperature(self, temperature: float) -> float:
        """验证温度参数"""
        if not (LLM_TEMPERATURE_MIN <= temperature <= LLM_TEMPERATURE_MAX):
            logger.warning(f"温度参数 {temperature} 超出范围 [{LLM_TEMPERATURE_MIN}, {LLM_TEMPERATURE_MAX}]，使用默认值")
            return 0.7
        return temperature
    
    def _validate_generate_params(
        self, 
        messages: List[Dict[str, str]], 
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None
    ) -> Dict[str, Any]:
        """验证生成参数"""
        if not messages:
            raise ValidationError("消息列表不能为空")
        
        for i, msg in enumerate(messages):
            if not isinstance(msg, dict) or 'role' not in msg or 'content' not in msg:
                raise ValidationError(f"消息 {i} 格式无效，需要包含 role 和 content")
        
        effective_temp = temperature or self.temperature
        effective_max_tokens = max_tokens or self.max_tokens
        
        # 验证温度范围
        if temperature is not None:
            validate_field_range({'temperature': temperature}, 'temperature', 
                               LLM_TEMPERATURE_MIN, LLM_TEMPERATURE_MAX)
        
        return {
            'messages': messages,
            'temperature': effective_temp,
            'max_tokens': effective_max_tokens
        }
    
    async def generate_response(
        self, 
        messages: List[Dict[str, str]], 
        system_prompt: Optional[str] = None,
        response_format: Optional[Dict[str, str]] = None,
        temperature: Optional[float] = None,
        stream: bool = False,
        max_tokens: Optional[int] = None
    ) -> Union[str, Dict[str, Any]]:
        """生成LLM响应，支持多种模型和格式"""
        try:
            # 验证参数
            params = self._validate_generate_params(messages, temperature, max_tokens)
            
            # 添加系统提示
            if system_prompt:
                params['messages'] = [{"role": "system", "content": system_prompt}] + params['messages']
            
            logger.debug(f"LLM请求: {len(params['messages'])} 条消息, 模型: {self.model}")
            
            # 执行生成，带自动回退
            return await self._execute_generation_with_fallback(
                params['messages'], 
                response_format, 
                params['temperature'],
                params['max_tokens'],
                stream
            )
                
        except (ValidationError, ServiceError):
            raise
        except Exception as e:
            error_msg, _ = self.error_handler.log_and_return_error(e, "LLM响应生成")
            raise ServiceError(error_msg, "LLMService", "generate_response")
    
    @retry_on_exception(max_retries=LLM_MAX_RETRIES, delay=1.0, exceptions=(ExternalServiceError,))
    async def _execute_generation_with_fallback(
        self, 
        messages: List[Dict[str, str]], 
        response_format: Optional[Dict[str, str]], 
        temperature: float,
        max_tokens: int,
        stream: bool = False
    ) -> Union[str, Dict[str, Any]]:
        """执行生成，支持提供商自动回退"""
        
        # 尝试主要提供商
        try:
            if self.provider.lower() == "openai":
                return await self._call_openai_api(
                    messages, response_format, temperature, max_tokens, stream
                )
            elif self.provider.lower() == "ollama":
                return await self._call_ollama_api(
                    messages, temperature, max_tokens, stream
                )
        except Exception as e:
            logger.warning(f"{self.provider} API调用失败，尝试备用提供商: {str(e)}")
            
            # 尝试备用提供商
            try:
                if self.backup_provider.lower() == "ollama":
                    return await self._call_ollama_api(
                        messages, temperature, max_tokens, stream
                    )
                elif self.backup_provider.lower() == "openai":
                    return await self._call_openai_api(
                        messages, response_format, temperature, max_tokens, stream
                    )
            except Exception as backup_error:
                logger.error(f"备用提供商 {self.backup_provider} 也失败: {str(backup_error)}")
                raise ExternalServiceError(
                    f"所有LLM提供商均不可用: 主要({str(e)}), 备用({str(backup_error)})",
                    "LLM"
                )
        
        raise ServiceError("未知的LLM提供商配置", "LLMService")
    
    async def _call_openai_api(
        self, 
        messages: List[Dict[str, str]], 
        response_format: Optional[Dict[str, str]], 
        temperature: float,
        max_tokens: int,
        stream: bool = False
    ) -> Optional[str]:
        """调用OpenAI兼容API生成响应"""
        try:
            kwargs = {
                "model": config.llm.model,  # 确保使用正确的模型名称
                "messages": messages,
                "temperature": temperature,
                "max_tokens": max_tokens,
                "stream": stream
            }
            
            if response_format:
                kwargs["response_format"] = response_format
            
            client = getattr(self, "backup_client", None) if self.provider.lower() == "ollama" else self.client
            if not client:
                client = OpenAI(
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url
                )
            
            response = client.chat.completions.create(**kwargs)
            
            if stream:
                # 处理流式响应
                collected_chunks = []
                collected_content = []
                
                for chunk in response:
                    collected_chunks.append(chunk)
                    if chunk.choices and chunk.choices[0].delta.content:
                        collected_content.append(chunk.choices[0].delta.content)
                
                return "".join(collected_content)
            else:
                # 常规响应
                result = response.choices[0].message.content
                logger.debug(f"LLM响应长度: {len(result) if result else 0}")
                return result
        except Exception as e:
            logger.error(f"OpenAI API调用失败: {str(e)}")
            raise e
    
    async def _call_ollama_api(
        self, 
        messages: List[Dict[str, str]], 
        temperature: float,
        max_tokens: int,
        stream: bool = False
    ) -> Optional[str]:
        """调用Ollama API生成响应"""
        try:
            # 确保设置正确的Ollama host
            os.environ["OLLAMA_HOST"] = config.llm.ollama_base_url.replace("/v1", "")
            logger.debug(f"使用Ollama host: {os.environ.get('OLLAMA_HOST')}")
            
            # 将消息转换为Ollama格式
            if stream:
                # 流式处理
                response = ""
                for chunk in ollama.chat(
                    model=config.llm.ollama_model,
                    messages=[{"role": m["role"], "content": m["content"]} for m in messages],
                    stream=True,
                    options={
                        "temperature": temperature,
                        "num_predict": max_tokens
                    }
                ):
                    if "message" in chunk and "content" in chunk["message"]:
                        response += chunk["message"]["content"]
                        
                return response
            else:
                # 常规响应
                response = ollama.chat(
                    model=config.llm.ollama_model,
                    messages=[{"role": m["role"], "content": m["content"]} for m in messages],
                    options={
                        "temperature": temperature,
                        "num_predict": max_tokens
                    }
                )
                
                if "message" in response and "content" in response["message"]:
                    return response["message"]["content"]
                else:
                    logger.error("Ollama响应格式无效")
                    return None
        except Exception as e:
            logger.error(f"Ollama API调用失败: {str(e)}")
            raise e
    
    async def analyze_k8s_problem(
        self, 
        deployment_yaml: str, 
        error_event: str,
        additional_context: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """分析Kubernetes问题并提供修复建议"""
        system_prompt = """
你是一个Kubernetes专家，帮助用户分析和修复Kubernetes部署问题。
请根据提供的部署YAML和错误事件，识别问题并提出修复建议。
你的回答应该包含以下格式的JSON结构:
{
    "problem_summary": "简短的问题概述",
    "root_causes": ["根本原因1", "根本原因2"],
    "severity": "严重程度 (低/中/高/紧急)",
    "fixes": [
        {
            "description": "修复1的描述",
            "yaml_changes": "需要进行的YAML变更",
            "confidence": 0.9
        }
    ],
    "additional_notes": "任何额外的说明或建议"
}
请确保回答仅包含有效的JSON，不要添加额外解释。
"""

        try:
            # 准备消息
            context = f"""
部署YAML:
```yaml
{deployment_yaml}
```

错误事件:
```
{error_event}
```
"""
            if additional_context:
                context += f"\n额外上下文信息:\n```\n{additional_context}\n```"
                
            messages = [{"role": "user", "content": context}]
            
            # 调用LLM API
            response_format = {"type": "json_object"}
            response = await self.generate_response(
                messages=messages,
                system_prompt=system_prompt,
                response_format=response_format,
                temperature=0.1
            )
            
            if response:
                try:
                    # 提取JSON响应
                    return await self._extract_json_from_k8s_analysis(response, messages)
                except Exception as json_error:
                    logger.error(f"解析K8s分析JSON失败: {str(json_error)}")
                    # 尝试再次调用，但不指定JSON响应格式
                    alternative_response = await self.generate_response(
                        messages=messages,
                        system_prompt=system_prompt,
                        temperature=0.1
                    )
                    
                    if alternative_response:
                        return await self._extract_json_from_k8s_analysis(alternative_response, messages)
                    else:
                        logger.error("获取替代响应失败")
                        return self._create_default_analysis()
            else:
                logger.error("从LLM获取响应失败")
                return self._create_default_analysis()
                
        except Exception as e:
            logger.error(f"K8s问题分析失败: {str(e)}")
            return self._create_default_analysis()
    
    def _create_default_analysis(self) -> Dict[str, Any]:
        """创建默认的分析结果"""
        return {
            "problem_summary": "无法分析问题",
            "root_causes": ["分析过程中出现错误"],
            "severity": "未知",
            "fixes": [],
            "additional_notes": "请检查您的部署YAML和错误描述，并确保LLM服务正常运行。"
        }
    
    async def _extract_json_from_k8s_analysis(self, response: str, messages: List[Dict[str, str]]) -> Dict[str, Any]:
        """从LLM响应中提取JSON对象"""
        # 尝试直接解析
        try:
            return json.loads(response)
        except json.JSONDecodeError:
            logger.warning("直接解析JSON失败，尝试提取JSON部分")
            
        # 尝试从文本中提取JSON部分
        try:
            # 查找以 { 开头，以 } 结尾的部分
            json_match = re.search(r'(\{.*\})', response, re.DOTALL)
            if json_match:
                extracted_json = json_match.group(1)
                return json.loads(extracted_json)
        except (json.JSONDecodeError, AttributeError):
            logger.warning("从响应中提取JSON失败，尝试进行修复")
        
        # 尝试请求LLM修复JSON
        try:
            fix_prompt = """
上一条消息中的JSON格式有问题，请修复它。
返回一个有效的JSON对象，包含以下字段：
- problem_summary: 问题概述 (字符串)
- root_causes: 根本原因列表 (字符串数组)
- severity: 严重程度 (字符串: "低", "中", "高" 或 "紧急")
- fixes: 修复建议列表 (对象数组，每个对象包含 description, yaml_changes 和 confidence 字段)
- additional_notes: 额外说明 (字符串)

请确保返回的是一个有效的、格式正确的JSON对象，不要添加其他解释。
"""
            fix_messages = messages + [
                {"role": "assistant", "content": response},
                {"role": "user", "content": fix_prompt}
            ]
            
            fixed_response = await self.generate_response(
                messages=fix_messages,
                temperature=0.1,
                response_format={"type": "json_object"}
            )
            
            if fixed_response:
                return json.loads(fixed_response)
            else:
                logger.error("修复JSON响应失败")
                return self._create_default_analysis()
                
        except Exception as e:
            logger.error(f"修复JSON格式失败: {str(e)}")
            
            # 创建最基本的返回数据
            analysis = self._create_default_analysis()
            
            # 尝试从原始响应中提取有用信息
            if "问题概述" in response or "problem_summary" in response:
                analysis["problem_summary"] = "可能存在部署配置问题"
            
            if "修复" in response or "fix" in response:
                analysis["fixes"].append({
                    "description": "请查看原始响应中的修复建议",
                    "yaml_changes": "无法自动解析YAML变更",
                    "confidence": 0.5
                })
                
            return analysis
    
    async def generate_rca_summary(
        self, 
        anomalies: Dict[str, Any], 
        correlations: Dict[str, Any],
        candidates: List[Dict[str, Any]]
    ) -> Optional[str]:
        """生成根因分析总结"""
        system_prompt = """
你是一个专业的云平台监控和根因分析专家。
请根据提供的指标异常、相关性和候选根因，总结分析结果并提供清晰的根因说明。
提供具有洞察力的分析，着重于主要问题及其原因，并提供可能的解决方向。
使用简明专业的语言，注重实用性建议。
"""

        try:
            # 准备消息内容
            content = f"""
## 指标异常:
{json.dumps(anomalies, ensure_ascii=False, indent=2)}

## 相关性:
{json.dumps(correlations, ensure_ascii=False, indent=2)}

## 候选根因:
{json.dumps(candidates, ensure_ascii=False, indent=2)}

请生成一份专业的根因分析总结，并提出可能的解决方案。
"""
            messages = [{"role": "user", "content": content}]
            
            # 生成根因分析总结
            response = await self.generate_response(
                messages=messages,
                system_prompt=system_prompt,
                temperature=0.3
            )
            
            return response
            
        except Exception as e:
            logger.error(f"生成RCA总结失败: {str(e)}")
            return None
    
    async def generate_fix_explanation(
        self, 
        deployment: str, 
        actions_taken: List[str], 
        success: bool
    ) -> Optional[str]:
        """生成修复说明"""
        system_prompt = """
你是Kubernetes自动修复系统的解释器。
请根据提供的部署名称、已执行的操作和修复结果，提供一份简明清晰的修复说明。
内容应该简洁、专业，并对技术细节进行合理解释。
"""

        try:
            # 准备消息内容
            result = "成功" if success else "失败"
            content = f"""
部署: {deployment}
执行的操作:
{json.dumps(actions_taken, ensure_ascii=False, indent=2)}
修复结果: {result}

请生成一份简明的修复说明。
"""
            messages = [{"role": "user", "content": content}]
            
            # 生成修复说明
            response = await self.generate_response(
                messages=messages,
                system_prompt=system_prompt,
                temperature=0.3
            )
            
            return response
            
        except Exception as e:
            logger.error(f"生成修复说明失败: {str(e)}")
            return None
    
    def is_healthy(self) -> bool:
        """检查LLM服务是否健康"""
        try:
            logger.info("检查LLM服务健康状态")
            
            # 检查主要提供商健康状态
            provider_health = self._check_provider_health(self.provider)
            
            if provider_health:
                logger.info(f"LLM服务({self.provider})健康状态: 正常")
                return True
            
            # 如果主要提供商不健康，检查备用提供商
            logger.warning(f"LLM服务({self.provider})不可用，检查备用提供商({self.backup_provider})")
            backup_health = self._check_provider_health(self.backup_provider)
            
            if backup_health:
                logger.info(f"备用LLM服务({self.backup_provider})健康状态: 正常")
                return True
            
            logger.error("所有LLM服务均不可用")
            return False
            
        except Exception as e:
            logger.error(f"检查LLM服务健康状态时出错: {str(e)}")
            return False
    
    def _check_provider_health(self, provider: str) -> bool:
        """检查特定提供商的健康状态"""
        try:
            if provider.lower() == "openai":
                return self._check_openai_health()
            elif provider.lower() == "ollama":
                return self._check_ollama_health()
            else:
                logger.warning(f"不支持的LLM提供商: {provider}")
                return False
        except Exception as e:
            logger.error(f"检查{provider}健康状态失败: {str(e)}")
            return False
    
    def _check_openai_health(self) -> bool:
        """检查OpenAI服务健康状态"""
        try:
            # 尝试创建客户端并发送简单请求
            client = getattr(self, "backup_client", None) if self.provider.lower() == "ollama" else self.client
            if not client:
                client = OpenAI(
                    api_key=config.llm.api_key,
                    base_url=config.llm.base_url
                )
            
            # 发送简单请求以验证连接
            response = client.chat.completions.create(
                model=config.llm.model,
                messages=[{"role": "user", "content": "测试"}],
                max_tokens=5
            )
            
            # 检查是否有有效响应
            if response and hasattr(response, "choices") and len(response.choices) > 0:
                logger.debug("OpenAI健康检查通过")
                return True
            else:
                logger.warning("OpenAI服务响应无效")
                return False
                
        except Exception as e:
            logger.warning(f"OpenAI健康检查失败: {str(e)}")
            return False
    
    def _check_ollama_health(self) -> bool:
        """检查Ollama服务健康状态"""
        try:
            # 确保设置正确的Ollama host
            os.environ["OLLAMA_HOST"] = config.llm.ollama_base_url.replace("/v1", "")
            
            # 尝试获取模型列表以验证连接
            try:
                response = ollama.list()
                if response and "models" in response:
                    # 检查我们需要的模型是否可用
                    model_available = any(model["name"] == config.llm.ollama_model for model in response["models"])
                    if not model_available:
                        logger.warning(f"Ollama模型 {config.llm.ollama_model} 不可用")
                        return False
                    
                    logger.debug("Ollama健康检查通过")
                    return True
                else:
                    logger.warning("Ollama服务响应无效")
                    return False
            except Exception as e:
                logger.warning(f"获取Ollama模型列表失败: {str(e)}")
                
                # 尝试直接发送简单请求
                response = ollama.chat(
                    model=config.llm.ollama_model,
                    messages=[{"role": "user", "content": "测试"}]
                )
                
                if response and "message" in response:
                    logger.debug("Ollama单次请求测试通过")
                    return True
                else:
                    logger.warning("Ollama服务响应无效")
                    return False
                
        except Exception as e:
            logger.warning(f"Ollama健康检查失败: {str(e)}")
            return False
