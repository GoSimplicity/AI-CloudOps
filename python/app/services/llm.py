import logging
import json
import os
import requests
from typing import Dict, Any, List, Optional, Union, Tuple
from openai import OpenAI
import ollama
from app.config.settings import config

logger = logging.getLogger("aiops.llm")

class LLMService:
    def __init__(self):
        """
        初始化LLM服务，支持OpenAI和Ollama
        系统会优先使用外部模型(OpenAI)，如果不可用则自动回退到本地模型(Ollama)
        """
        self.provider = config.llm.provider
        self.model = config.llm.effective_model
        self.temperature = config.llm.temperature
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
            raise ValueError(f"不支持的LLM提供商: {self.provider}")
    
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
            if system_prompt:
                messages = [{"role": "system", "content": system_prompt}] + messages
            
            effective_temp = temperature or self.temperature
            effective_max_tokens = max_tokens or self.max_tokens
            
            logger.debug(f"LLM请求: {len(messages)} 条消息, 模型: {self.model}")
            
            # 根据提供商选择不同的调用方式
            if self.provider.lower() == "openai":
                try:
                    return await self._call_openai_api(
                        messages, 
                        response_format, 
                        effective_temp,
                        effective_max_tokens,
                        stream
                    )
                except Exception as e:
                    logger.error(f"OpenAI API调用失败，尝试使用备用Ollama模型: {str(e)}")
                    return await self._call_ollama_api(
                        messages, 
                        effective_temp,
                        effective_max_tokens,
                        stream
                    )
            elif self.provider.lower() == "ollama":
                try:
                    return await self._call_ollama_api(
                        messages, 
                        effective_temp,
                        effective_max_tokens,
                        stream
                    )
                except Exception as e:
                    logger.error(f"Ollama API调用失败，尝试使用备用OpenAI模型: {str(e)}")
                    return await self._call_openai_api(
                        messages, 
                        response_format, 
                        effective_temp,
                        effective_max_tokens,
                        stream
                    )
            else:
                raise ValueError(f"不支持的LLM提供商: {self.provider}")
                
        except Exception as e:
            logger.error(f"LLM生成响应失败: {str(e)}")
            # 尝试使用备用方式生成响应
            try:
                if self.provider.lower() == "openai":
                    logger.info("尝试使用备用Ollama模型生成响应")
                    result = await self._call_ollama_api(messages, effective_temp, effective_max_tokens)
                    return result
                elif self.provider.lower() == "ollama":
                    logger.info("尝试使用备用OpenAI模型生成响应")
                    result = await self._call_openai_api(
                        messages, 
                        response_format, 
                        effective_temp,
                        effective_max_tokens
                    )
                    return result
                else:
                    logger.error("无法生成LLM响应")
                    return None
            except Exception as backup_error:
                logger.error(f"备用模型生成响应失败: {str(backup_error)}")
                return None
    
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
                    model=config.llm.ollama_model,  # 确保使用正确的模型名称
                    messages=messages,
                    stream=True,
                    options={
                        "temperature": temperature,
                        "num_predict": max_tokens
                    }
                ):
                    if chunk.get('message', {}).get('content'):
                        response += chunk['message']['content']
                return response
            else:
                # 常规响应
                response = ollama.chat(
                    model=config.llm.ollama_model,
                    messages=messages,
                    options={
                        "temperature": temperature,
                        "num_predict": max_tokens
                    }
                )
                result = response.get('message', {}).get('content', '')
                logger.debug(f"LLM响应长度: {len(result) if result else 0}")
                return result
        except Exception as e:
            logger.error(f"Ollama API调用失败: {str(e)}")
            raise e
    
    async def analyze_k8s_problem(
        self, 
        deployment_yaml: str, 
        error_event: str,
        additional_context: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """分析Kubernetes问题并生成修复方案，增强错误处理和模型回退功能"""
        try:
            system_prompt = """你是一个Kubernetes专家，专门分析和修复Kubernetes部署问题。
请分析给定的错误信息和部署配置，生成适当的修复方案。
返回格式必须是JSON，包含以下字段：
{
  "analysis": "问题分析描述",
  "root_cause": "根本原因",
  "solution": "解决方案描述",
  "action": "需要执行的操作类型，例如：修改资源限制、修改健康检查、重启部署等"
}

对于不同类型的问题，可能还需要额外的字段：
1. 对于资源配置问题：
{
  "requests": {
    "cpu": "100m",  // CPU请求值
    "memory": "128Mi"  // 内存请求值
  },
  "limits": {
    "cpu": "200m",  // CPU限制值
    "memory": "256Mi"  // 内存限制值
  }
}

2. 对于健康检查问题：
{
  "httpGet": {
    "path": "/",  // 正确的检查路径
    "port": 80  // 正确的端口
  },
  "periodSeconds": 10,  // 检查间隔
  "failureThreshold": 3  // 失败阈值
}

严格遵守以下规则:
1. 仅返回JSON格式，不要包含任何其他文本或解释
2. 不要使用注释，包括JSON中的//注释
3. 确保所有字符串使用双引号，而不是单引号
4. 确保所有键名都有双引号"""

            # 准备用户提示
            user_prompt = f"""Kubernetes部署YAML:
```yaml
{deployment_yaml}
```

错误描述: {error_event}

"""
            
            if additional_context:
                user_prompt += f"\n额外上下文信息:\n{additional_context}\n"
            
            user_prompt += "\n请分析问题原因并给出解决方案，仅返回有效的标准JSON格式。不要包含解释、注释或其他文本。"
            
            messages = [{"role": "user", "content": user_prompt}]
            
            # 设置响应格式为JSON (对OpenAI有效)
            response_format = {"type": "json_object"} if self.provider.lower() == "openai" else None
            
            # 记录当前使用的模型提供商
            current_provider = self.provider
            logger.info(f"尝试使用 {current_provider} 分析K8s问题")
            
            try:
                # 第一次尝试生成响应
                response = await self.generate_response(
                    messages=messages,
                    system_prompt=system_prompt,
                    response_format=response_format,
                    temperature=0.2
                )
                
                if not response:
                    logger.error(f"{current_provider} 生成K8s问题分析失败：空响应")
                    # 如果主模型返回为空，尝试使用备用模型重试
                    raise ValueError("空响应")
                    
            except Exception as e:
                logger.warning(f"使用 {current_provider} 分析失败: {str(e)}，尝试备用方法")
                
                # 设置更简单的系统提示，可能对一些模型更友好
                simplified_system_prompt = """你是Kubernetes专家。分析给定的部署问题并返回JSON解决方案。
必须包含这些字段: "analysis", "root_cause", "solution", "action"。
只返回JSON，不要有任何其他文本。"""
                
                try:
                    # 尝试使用简化提示
                    response = await self.generate_response(
                        messages=messages,
                        system_prompt=simplified_system_prompt,
                        temperature=0.1
                    )
                    
                    if not response:
                        logger.error("备用方法也失败，返回默认分析结果")
                        return self._create_default_analysis()
                        
                except Exception as backup_e:
                    logger.error(f"所有LLM尝试均失败: {str(backup_e)}")
                    return self._create_default_analysis()
            
            # 尝试解析JSON
            return await self._extract_json_from_k8s_analysis(response, messages)
            
        except Exception as e:
            logger.error(f"分析K8s问题失败: {str(e)}")
            return self._create_default_analysis()
    
    def _create_default_analysis(self) -> Dict[str, Any]:
        """创建默认的问题分析结果"""
        return {
            "analysis": "无法获取有效的分析结果，可能是模型服务不可用",
            "root_cause": "可能是探针配置问题或资源限制不合理",
            "solution": "建议检查YAML配置，特别是健康检查、资源限制部分",
            "action": "需要手动分析"
        }
    
    async def _extract_json_from_k8s_analysis(self, response: str, messages: List[Dict[str, str]]) -> Dict[str, Any]:
        """从K8s分析响应中提取JSON，包含多种提取方法"""
        try:
            # 直接尝试解析完整响应
            analysis_result = json.loads(response)
            logger.info("成功直接解析K8s问题分析JSON")
            return analysis_result
        except json.JSONDecodeError as e:
            logger.error(f"解析LLM JSON响应失败: {str(e)}")
            
            # 尝试提取代码块中的JSON
            if "```json" in response:
                try:
                    json_content = response.split("```json")[1].split("```")[0].strip()
                    analysis_result = json.loads(json_content)
                    logger.info("成功从代码块提取JSON分析结果")
                    return analysis_result
                except (json.JSONDecodeError, IndexError) as json_e:
                    logger.error(f"提取和解析JSON失败: {str(json_e)}")
            
            # 检查任意代码块
            if "```" in response:
                try:
                    parts = response.split("```")
                    for part in parts:
                        if part.strip() and "{" in part and "}" in part:
                            # 清理可能的语言标记
                            clean_part = re.sub(r'^[a-z]+\n', '', part.strip())
                            analysis_result = json.loads(clean_part)
                            logger.info("成功从代码块提取JSON分析结果")
                            return analysis_result
                except (json.JSONDecodeError, IndexError) as json_e:
                    logger.error(f"尝试从代码块提取JSON失败: {str(json_e)}")
            
            # 尝试修复常见JSON格式错误
            try:
                # 替换单引号为双引号
                fixed_json = response.replace("'", '"')
                # 修复没有引号的键
                import re
                fixed_json = re.sub(r'(\s*?)(\w+)(\s*?):', r'\1"\2"\3:', fixed_json)
                # 删除注释
                fixed_json = re.sub(r'//.*?\n', '\n', fixed_json)
                # 尝试使用修复后的文本
                analysis_result = json.loads(fixed_json)
                logger.info("成功通过修复格式解析JSON")
                return analysis_result
            except json.JSONDecodeError:
                logger.error("尝试修复JSON格式失败")
            
            # 尝试查找大括号之间的内容
            try:
                start_idx = response.find('{')
                end_idx = response.rfind('}')
                if start_idx >= 0 and end_idx > start_idx:
                    json_str = response[start_idx:end_idx+1]
                    # 替换单引号为双引号
                    json_str = json_str.replace("'", '"')
                    # 修复没有引号的键
                    json_str = re.sub(r'(\s*?)(\w+)(\s*?):', r'\1"\2"\3:', json_str)
                    # 删除注释
                    json_str = re.sub(r'//.*?\n', '\n', json_str)
                    
                    analysis_result = json.loads(json_str)
                    logger.info("成功从文本中提取JSON对象")
                    return analysis_result
            except (json.JSONDecodeError, ValueError) as e:
                logger.error(f"尝试提取JSON对象失败: {str(e)}")
            
            # 所有直接解析方法失败，尝试重新请求模型
            logger.info("尝试引导模型重新生成纯JSON")
            clarification_msg = "你的响应包含无效的JSON格式。请只返回有效的JSON对象，不要包含任何注释、说明或代码块标记。"
            messages.append({"role": "assistant", "content": response})
            messages.append({"role": "user", "content": clarification_msg})
            
            try:
                # 重新请求，使用更严格的系统提示
                retry_response = await self.generate_response(
                    messages=messages,
                    system_prompt="只返回有效的JSON格式，不要有任何额外文本或代码块标记。确保所有键名和字符串值使用双引号。",
                    response_format={"type": "json_object"} if self.provider.lower() == "openai" else None,
                    temperature=0.1
                )
                
                # 尝试直接解析重试响应
                analysis_result = json.loads(retry_response)
                logger.info("成功通过重新请求生成有效JSON")
                return analysis_result
            except (json.JSONDecodeError, Exception) as retry_e:
                logger.error(f"重新请求获取JSON失败: {str(retry_e)}")
                
                # 尝试最后的正则表达式提取
                try:
                    import re
                    json_pattern = r'\{(?:[^{}]|(?R))*\}'
                    matches = re.findall(r'\{.*?\}', retry_response, re.DOTALL)
                    if matches:
                        # 按长度排序，通常更长的匹配更可能是完整的JSON
                        matches.sort(key=len, reverse=True)
                        for potential_json in matches:
                            try:
                                # 替换单引号并清理
                                cleaned = potential_json.replace("'", '"')
                                analysis_result = json.loads(cleaned)
                                logger.info("成功通过正则表达式提取JSON")
                                return analysis_result
                            except:
                                continue
                except Exception as regex_e:
                    logger.error(f"正则表达式提取JSON失败: {str(regex_e)}")
            
            # 所有方法都失败，返回默认分析结果
            logger.error("所有JSON解析方法均失败，返回默认分析结果")
            return self._create_default_analysis()
            
        except Exception as e:
            logger.error(f"提取JSON过程中出错: {str(e)}")
            return self._create_default_analysis()
    
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
        """
        检查LLM服务健康状态
        按照优先级顺序检查: 1. 主要模型(OpenAI) 2. 备用模型(Ollama)
        如果任一服务正常，返回True
        使用更健壮的API检查方式替代直接模型调用
        """
        try:
            # 记录开始检查
            logger.debug(f"开始检查LLM服务健康状态: 主要提供商={self.provider}")
            
            # 检查主要提供商
            primary_healthy = self._check_provider_health(self.provider)
            
            # 如果主要提供商健康，直接返回True
            if primary_healthy:
                logger.debug(f"主要LLM服务({self.provider})健康")
                return True
                
            # 如果主要提供商不健康，检查备用提供商
            backup_provider = "ollama" if self.provider.lower() == "openai" else "openai"
            logger.info(f"主要LLM服务({self.provider})不健康，检查备用服务({backup_provider})")
            
            backup_healthy = self._check_provider_health(backup_provider)
            
            if backup_healthy:
                logger.info(f"备用LLM服务({backup_provider})健康，可用于故障转移")
                return True
            else:
                logger.error(f"主要和备用LLM服务均不健康，服务可能不可用")
                return False
                
        except Exception as e:
            logger.error(f"LLM健康检查过程发生异常: {str(e)}")
            return False
    
    def _check_provider_health(self, provider: str) -> bool:
        """检查特定提供商的健康状态"""
        try:
            if provider.lower() == "openai":
                return self._check_openai_health()
            elif provider.lower() == "ollama":
                return self._check_ollama_health()
            else:
                logger.warning(f"未知的LLM提供商: {provider}")
                return False
        except Exception as e:
            logger.error(f"检查{provider}健康状态时出错: {str(e)}")
            return False
    
    def _check_openai_health(self) -> bool:
        """检查OpenAI服务健康状态"""
        try:
            # 构建轻量级请求，只请求models列表而不实际调用模型
            headers = {
                "Authorization": f"Bearer {config.llm.api_key}",
                "Content-Type": "application/json"
            }
            
            # 尝试请求模型列表，这是一个轻量级API调用
            url = f"{config.llm.base_url.rstrip('/')}/models"
            logger.debug(f"检查OpenAI健康，URL: {url}")
            
            response = requests.get(
                url,
                headers=headers,
                timeout=3  # 使用较短的超时时间
            )
            
            if response.status_code < 400:
                # 检查是否包含预期的响应结构
                data = response.json()
                if "data" in data and isinstance(data["data"], list):
                    logger.debug("OpenAI服务健康")
                    return True
                else:
                    logger.warning("OpenAI服务响应格式异常")
                    return False
            else:
                logger.warning(f"OpenAI API响应错误状态码: {response.status_code}")
                return False
                
        except requests.exceptions.Timeout:
            logger.warning("OpenAI API请求超时")
            return False
        except requests.exceptions.ConnectionError:
            logger.warning("OpenAI API连接错误")
            return False
        except Exception as e:
            logger.warning(f"OpenAI健康检查出现异常: {str(e)}")
            return False
    
    def _check_ollama_health(self) -> bool:
        """检查Ollama服务健康状态"""
        try:
            # 设置Ollama主机地址
            ollama_host = config.llm.ollama_base_url.replace("/v1", "")
            os.environ["OLLAMA_HOST"] = ollama_host
            
            # 首先尝试使用API列出可用模型
            api_url = f"{ollama_host}/api/tags"
            logger.debug(f"检查Ollama健康，URL: {api_url}")
            
            response = requests.get(api_url, timeout=3)
            
            if response.status_code == 200:
                # 确认响应包含预期结构
                data = response.json()
                if "models" in data and isinstance(data["models"], list):
                    # 检查是否包含我们需要的模型
                    model_found = False
                    for model in data["models"]:
                        if model.get("name") == config.llm.ollama_model:
                            model_found = True
                            break
                    
                    if model_found:
                        logger.debug(f"Ollama服务健康，找到模型: {config.llm.ollama_model}")
                    else:
                        logger.debug("Ollama服务健康，但未找到指定模型")
                    
                    return True
                else:
                    logger.warning("Ollama API响应格式异常")
                    return False
            else:
                logger.warning(f"Ollama API响应错误状态码: {response.status_code}")
                return False
                
        except requests.exceptions.Timeout:
            logger.warning("Ollama API请求超时")
            return False
        except requests.exceptions.ConnectionError:
            logger.warning("Ollama API连接错误")
            return False
        except Exception as e:
            logger.warning(f"Ollama健康检查出现异常: {str(e)}")
            return False