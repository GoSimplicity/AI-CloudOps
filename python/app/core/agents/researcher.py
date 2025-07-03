import logging
from typing import Dict, Any, List, Optional
from langchain_core.tools import tool
from langchain_community.tools.tavily_search import TavilySearchResults
from app.config.settings import config
from app.services.llm import LLMService

logger = logging.getLogger("aiops.researcher")

class ResearcherAgent:
    def __init__(self):
        # 使用我们自己的LLM服务
        self.llm_service = LLMService()
        
        # 初始化搜索工具
        try:
            if config.tavily.api_key:
                self.search_tool = TavilySearchResults(
                    max_results=config.tavily.max_results,
                    api_key=config.tavily.api_key
                )
                self.search_enabled = True
                logger.info("Researcher Agent初始化完成（支持网络搜索）")
            else:
                self.search_tool = None
                self.search_enabled = False
                logger.info("Researcher Agent初始化完成（无网络搜索）")
        except Exception as e:
            logger.warning(f"搜索工具初始化失败: {str(e)}")
            self.search_tool = None
            self.search_enabled = False
    
    @tool
    async def search_kubernetes_solutions(self, query: str) -> str:
        """搜索Kubernetes问题解决方案"""
        try:
            if not self.search_enabled:
                return await self._provide_local_knowledge(query)
            
            # 构建专业的搜索查询
            k8s_query = f"Kubernetes {query} solution troubleshooting best practices"
            
            search_results = self.search_tool.run(k8s_query)
            
            if not search_results:
                return await self._provide_local_knowledge(query)
            
            # 整理搜索结果
            formatted_results = "找到以下Kubernetes解决方案:\n\n"
            
            for i, result in enumerate(search_results[:3], 1):
                title = result.get('title', 'Unknown')
                content = result.get('content', 'No content')
                url = result.get('url', 'No URL')
                
                formatted_results += f"{i}. **{title}**\n"
                formatted_results += f"   内容: {content[:200]}...\n"
                formatted_results += f"   来源: {url}\n\n"
            
            # 使用LLM总结和分析搜索结果
            summary = await self._summarize_search_results(query, search_results)
            
            return formatted_results + "\n" + summary
            
        except Exception as e:
            logger.error(f"搜索Kubernetes解决方案失败: {str(e)}")
            return await self._provide_local_knowledge(query)
    
    async def _summarize_search_results(self, query: str, results: List[Dict]) -> str:
        """总结搜索结果"""
        try:
            context = "\n".join([
                f"标题: {r.get('title', '')}\n内容: {r.get('content', '')}"
                for r in results[:3]
            ])
            
            prompt = f"""
基于以下搜索结果，为Kubernetes问题"{query}"提供专业的解决方案总结：

搜索结果：
{context}

请提供：
1. 问题的常见原因
2. 推荐的解决步骤
3. 最佳实践建议
4. 需要注意的事项

用简洁专业的语言回答。
"""
            
            messages = [{"role": "user", "content": prompt}]
            response = await self.llm_service.generate_response(messages)
            
            return f"\n**AI总结:**\n{response}"
            
        except Exception as e:
            logger.error(f"总结搜索结果失败: {str(e)}")
            return "\n**AI总结:** 无法生成总结"
    
    async def _provide_local_knowledge(self, query: str) -> str:
        """提供本地知识库的解决方案"""
        try:
            # 基于查询关键词提供本地知识
            query_lower = query.lower()
            
            if any(keyword in query_lower for keyword in ['cpu', 'memory', 'resource']):
                return self._get_resource_troubleshooting_guide()
            elif any(keyword in query_lower for keyword in ['pod', 'container', 'restart']):
                return self._get_pod_troubleshooting_guide()
            elif any(keyword in query_lower for keyword in ['network', 'connection', 'service']):
                return self._get_network_troubleshooting_guide()
            elif any(keyword in query_lower for keyword in ['storage', 'volume', 'pvc']):
                return self._get_storage_troubleshooting_guide()
            else:
                return self._get_general_troubleshooting_guide()
                
        except Exception as e:
            logger.error(f"提供本地知识失败: {str(e)}")
            return "无法提供相关的故障排除指南"
    
    def _get_resource_troubleshooting_guide(self) -> str:
        """资源问题故障排除指南"""
        return """
**Kubernetes资源问题故障排除指南:**

**常见CPU/内存问题:**
1. 检查资源限制和请求设置
2. 查看Pod的资源使用情况
3. 检查节点资源可用性
4. 考虑水平扩容或垂直扩容

**解决步骤:**
1. `kubectl describe pod <pod-name>` - 查看Pod详情
2. `kubectl top pod <pod-name>` - 查看实时资源使用
3. `kubectl get nodes -o wide` - 检查节点状态
4. 调整resources.requests和resources.limits

**最佳实践:**
- 设置合理的资源请求和限制
- 使用HPA进行自动扩容
- 监控资源使用趋势
- 定期检查资源配额
"""
    
    def _get_pod_troubleshooting_guide(self) -> str:
        """Pod问题故障排除指南"""
        return """
**Kubernetes Pod问题故障排除指南:**

**常见Pod问题:**
1. CrashLoopBackOff - 容器启动后崩溃
2. ImagePullBackOff - 镜像拉取失败  
3. Pending - Pod无法调度
4. Error - Pod运行错误

**诊断步骤:**
1. `kubectl describe pod <pod-name>` - 查看事件和状态
2. `kubectl logs <pod-name>` - 查看容器日志
3. `kubectl get events` - 查看集群事件
4. 检查镜像名称和标签是否正确

**常见解决方案:**
- 检查镜像拉取策略和凭据
- 验证容器启动命令和参数
- 检查健康检查配置
- 确认资源限制设置合理
"""
    
    def _get_network_troubleshooting_guide(self) -> str:
        """网络问题故障排除指南"""
        return """
**Kubernetes网络问题故障排除指南:**

**常见网络问题:**
1. Service无法访问
2. Pod间通信失败
3. Ingress配置错误
4. DNS解析问题

**诊断步骤:**
1. 检查Service和Endpoints
2. 验证网络策略配置
3. 测试Pod间连通性
4. 检查DNS配置

**解决方案:**
- 验证标签选择器匹配
- 检查端口配置
- 确认网络插件正常工作
- 验证Ingress控制器状态
"""
    
    def _get_storage_troubleshooting_guide(self) -> str:
        """存储问题故障排除指南"""
        return """
**Kubernetes存储问题故障排除指南:**

**常见存储问题:**
1. PVC Pending状态
2. 卷挂载失败
3. 存储空间不足
4. 持久卷访问权限问题

**诊断步骤:**
1. 检查StorageClass配置
2. 验证PV和PVC状态
3. 查看存储驱动程序日志
4. 检查节点存储空间

**解决方案:**
- 确认StorageClass支持动态配置
- 检查访问模式兼容性
- 验证存储后端可用性
- 调整存储容量需求
"""
    
    def _get_general_troubleshooting_guide(self) -> str:
        """通用故障排除指南"""
        return """
**Kubernetes通用故障排除指南:**

**基本诊断命令:**
1. `kubectl get pods -o wide` - 查看Pod状态
2. `kubectl describe <resource> <name>` - 查看资源详情
3. `kubectl logs <pod-name>` - 查看日志
4. `kubectl get events --sort-by=.metadata.creationTimestamp` - 查看事件

**系统检查清单:**
- 集群节点状态
- 系统资源使用情况
- 网络连通性
- 存储可用性
- 配置正确性

**常见解决方案:**
- 重启相关Pod或服务
- 检查和修复配置文件
- 调整资源分配
- 更新镜像版本
- 检查权限设置
"""
    
    @tool
    async def search_error_solutions(self, error_message: str) -> str:
        """搜索特定错误消息的解决方案"""
        try:
            # 清理错误消息，提取关键信息
            cleaned_error = self._clean_error_message(error_message)
            
            if self.search_enabled:
                query = f"Kubernetes error '{cleaned_error}' solution fix"
                search_results = self.search_tool.run(query)
                
                if search_results:
                    return await self._format_error_solutions(cleaned_error, search_results)
            
            # 使用本地错误知识库
            return self._get_local_error_solution(cleaned_error)
            
        except Exception as e:
            logger.error(f"搜索错误解决方案失败: {str(e)}")
            return f"无法搜索错误解决方案: {str(e)}"
    
    def _clean_error_message(self, error_message: str) -> str:
        """清理错误消息，提取关键部分"""
        # 移除时间戳、Pod名称等变量部分
        import re
        
        # 移除时间戳
        cleaned = re.sub(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[Z\+\-\d:]*', '', error_message)
        
        # 移除Pod名称（通常包含随机字符）
        cleaned = re.sub(r'[a-z0-9]+-[a-z0-9]{5}', '<pod-name>', cleaned)
        
        # 移除UUID
        cleaned = re.sub(r'[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}', '<uuid>', cleaned)
        
        # 保留前100个字符的关键信息
        return cleaned.strip()[:100]
    
    async def _format_error_solutions(self, error: str, search_results: List[Dict]) -> str:
        """格式化错误解决方案"""
        try:
            formatted = f"针对错误 '{error}' 的解决方案:\n\n"
            
            for i, result in enumerate(search_results[:2], 1):
                title = result.get('title', 'Unknown')
                content = result.get('content', 'No content')
                
                formatted += f"{i}. **{title}**\n"
                formatted += f"   {content[:150]}...\n\n"
            
            return formatted
            
        except Exception as e:
            return f"格式化解决方案失败: {str(e)}"
    
    def _get_local_error_solution(self, error: str) -> str:
        """从本地知识库获取错误解决方案"""
        error_lower = error.lower()
        
        if 'crashloopbackoff' in error_lower:
            return """
**CrashLoopBackOff错误解决方案:**
1. 检查容器启动命令和参数
2. 查看容器日志: kubectl logs <pod-name>
3. 检查健康检查配置
4. 验证镜像是否正确
5. 检查资源限制设置
"""
        elif 'imagepullbackoff' in error_lower:
            return """
**ImagePullBackOff错误解决方案:**
1. 检查镜像名称和标签
2. 验证镜像仓库访问权限
3. 检查镜像拉取策略
4. 确认网络连接正常
5. 验证Secret配置
"""
        elif 'pending' in error_lower:
            return """
**Pod Pending状态解决方案:**
1. 检查节点资源可用性
2. 验证节点选择器和亲和性规则
3. 检查污点和容忍度设置
4. 确认PVC绑定状态
5. 查看调度器日志
"""
        else:
            return f"""
**通用错误处理建议:**
1. 查看详细错误信息: kubectl describe pod <pod-name>
2. 检查相关日志
3. 验证配置文件
4. 检查资源和权限
5. 重启相关组件

错误信息: {error}
"""
    
    def get_available_tools(self) -> List[str]:
        """获取可用的研究工具"""
        tools = [
            "search_kubernetes_solutions",
            "search_error_solutions"
        ]
        
        if self.search_enabled:
            tools.append("web_search_enabled")
        else:
            tools.append("local_knowledge_only")
        
        return tools
        
    async def process_agent_state(self, state) -> Any:
        """处理Agent状态，支持工作流处理
        
        Args:
            state: 工作流状态对象 (AgentState)
            
        Returns:
            更新后的AgentState对象
        """
        try:
            from dataclasses import replace
            
            # 获取状态上下文信息（确保是字典副本）
            context = dict(state.context) if state.context else {}
            
            # 获取问题信息
            problem = context.get('problem', '')
            
            # 如果没有问题描述，无法进行研究
            if not problem:
                logger.warning("没有问题描述，无法进行研究")
                context['error'] = "没有问题描述，无法进行研究"
                return replace(state, context=context)
            
            logger.info(f"Researcher开始研究问题: {problem[:100]}...")
            
            try:
                # 搜索解决方案
                solution = None
                if any(kw in problem.lower() for kw in ["kubernetes", "k8s", "容器", "部署"]):
                    logger.info("检测到Kubernetes相关问题，使用专业搜索")
                    solution = await self.search_kubernetes_solutions(problem)
                else:
                    # 提取可能的错误信息
                    error_parts = [part for part in problem.split() if any(err in part.lower() for err in ["error", "fail", "exception", "crash", "错误", "失败", "异常"])]
                    error_message = " ".join(error_parts) if error_parts else problem
                    logger.info(f"提取错误信息: {error_message[:50]}...")
                    solution = await self.search_error_solutions(error_message)
                
                # 确保有结果
                if not solution:
                    logger.warning("搜索没有返回结果，使用本地知识")
                    solution = await self._provide_local_knowledge(problem)
                    
                logger.info(f"搜索结果长度: {len(solution) if solution else 0}字符")
            except Exception as search_e:
                logger.error(f"搜索过程发生错误: {str(search_e)}")
                solution = f"搜索过程发生错误: {str(search_e)}\n\n以下是基于本地知识的建议:\n" + await self._provide_local_knowledge(problem)
            
            # 更新上下文
            context['research_result'] = solution
            
            # 添加操作记录
            if 'actions_taken' not in context or not isinstance(context['actions_taken'], list):
                context['actions_taken'] = []
            context['actions_taken'].append(f"Researcher搜索解决方案: {problem[:50]}...")
            
            # 返回更新后的状态
            return replace(state, context=context)
            
        except Exception as e:
            logger.error(f"Researcher处理状态失败: {str(e)}")
            # 确保context是一个字典
            try:
                context = dict(state.context) if state.context else {}
            except:
                context = {}
                
            context['error'] = f"Researcher处理失败: {str(e)}"
            from dataclasses import replace
            return replace(state, context=context)