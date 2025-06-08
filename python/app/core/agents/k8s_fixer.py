import logging
import json
import yaml
from typing import Dict, Any, List, Optional
from langchain_core.tools import tool
from langchain_experimental.agents import create_pandas_dataframe_agent
from langchain_openai import ChatOpenAI
from app.services.kubernetes import KubernetesService
from app.services.llm import LLMService
from app.config.settings import config

logger = logging.getLogger("aiops.k8s_fixer")

class K8sFixerAgent:
    def __init__(self):
        self.k8s_service = KubernetesService()
        self.llm_service = LLMService()
        self.llm = ChatOpenAI(
            model=config.llm.model,
            api_key=config.llm.api_key,
            base_url=config.llm.base_url
        )
        logger.info("K8s修复Agent初始化完成")
    
    @tool
    async def analyze_and_fix_deployment(
        self, 
        deployment_name: str, 
        namespace: str, 
        error_description: str
    ) -> str:
        """分析并修复Kubernetes Deployment问题"""
        try:
            logger.info(f"开始分析和修复Deployment: {deployment_name}")
            
            # 获取当前Deployment配置
            deployment = await self.k8s_service.get_deployment(deployment_name, namespace)
            if not deployment:
                return f"无法获取Deployment {deployment_name} 的配置信息"
            
            # 获取相关事件
            events = await self.k8s_service.get_events(
                namespace=namespace,
                field_selector=f"involvedObject.name={deployment_name}"
            )
            
            # 获取Pod状态
            pods = await self.k8s_service.get_pods(
                namespace=namespace,
                label_selector=f"app={deployment_name}"
            )
            
            # 准备上下文信息
            context = {
                "deployment": deployment,
                "events": events[:10],  # 最近10个事件
                "pods": [self._extract_pod_info(pod) for pod in pods[:5]],  # 最近5个Pod
                "error_description": error_description
            }
            
            # 使用LLM分析问题
            analysis = await self.llm_service.analyze_k8s_problem(
                yaml.dump(deployment, default_flow_style=False),
                error_description,
                json.dumps(context, indent=2)
            )
            
            if not analysis:
                return "无法分析问题，建议手动检查"
            
            logger.info(f"问题分析完成: {analysis.get('analysis', 'N/A')}")
            
            # 执行修复操作
            fix_result = await self._execute_fix(
                deployment_name, 
                namespace, 
                analysis
            )
            
            return fix_result
            
        except Exception as e:
            logger.error(f"分析和修复Deployment失败: {str(e)}")
            return f"修复失败: {str(e)}"
    
    async def _execute_fix(
        self, 
        deployment_name: str, 
        namespace: str, 
        analysis: Dict[str, Any]
    ) -> str:
        """执行修复操作"""
        try:
            actions_taken = []
            patch = analysis.get('patch', {})
            risk_level = analysis.get('risk_level', 'medium')
            
            # 根据风险级别决定是否自动执行
            if risk_level == 'high':
                return f"风险级别过高({risk_level})，建议人工处理。分析结果：{analysis.get('solution', 'N/A')}"
            
            # 执行补丁修复
            if patch:
                success = await self.k8s_service.patch_deployment(
                    deployment_name, patch, namespace
                )
                if success:
                    actions_taken.append(f"应用配置补丁: {json.dumps(patch)}")
                else:
                    actions_taken.append("配置补丁应用失败")
            
            # 执行额外操作
            additional_actions = analysis.get('additional_actions', [])
            for action in additional_actions:
                action_result = await self._execute_additional_action(
                    deployment_name, namespace, action
                )
                actions_taken.append(action_result)
            
            # 验证修复结果
            verification_result = await self._verify_fix(deployment_name, namespace)
            
            result_summary = f"""
修复操作完成：
- 部署: {deployment_name}
- 命名空间: {namespace}
- 执行的操作: {'; '.join(actions_taken)}
- 验证结果: {verification_result}
- 分析结果: {analysis.get('analysis', 'N/A')}
- 解决方案: {analysis.get('solution', 'N/A')}
"""
            
            logger.info("修复操作完成")
            return result_summary
            
        except Exception as e:
            logger.error(f"执行修复操作失败: {str(e)}")
            return f"执行修复操作失败: {str(e)}"
    
    async def _execute_additional_action(
        self, 
        deployment_name: str, 
        namespace: str, 
        action: str
    ) -> str:
        """执行额外的修复操作"""
        try:
            action_lower = action.lower()
            
            if 'restart' in action_lower:
                success = await self.k8s_service.restart_deployment(deployment_name, namespace)
                return f"重启Deployment: {'成功' if success else '失败'}"
            
            elif 'scale' in action_lower:
                # 提取副本数（简单解析）
                import re
                replica_match = re.search(r'(\d+)', action)
                replicas = int(replica_match.group(1)) if replica_match else 2
                
                success = await self.k8s_service.scale_deployment(
                    deployment_name, replicas, namespace
                )
                return f"扩缩容到{replicas}副本: {'成功' if success else '失败'}"
            
            else:
                return f"不支持的操作: {action}"
                
        except Exception as e:
            return f"执行操作失败 {action}: {str(e)}"
    
    async def _verify_fix(self, deployment_name: str, namespace: str) -> str:
        """验证修复结果"""
        try:
            # 等待一段时间让修复生效
            import asyncio
            await asyncio.sleep(10)
            
            # 检查Deployment状态
            status = await self.k8s_service.get_deployment_status(deployment_name, namespace)
            if not status:
                return "无法获取部署状态"
            
            ready_replicas = status.get('ready_replicas', 0)
            replicas = status.get('replicas', 0)
            
            if ready_replicas >= replicas and replicas > 0:
                return f"验证成功: {ready_replicas}/{replicas} 副本就绪"
            else:
                return f"验证失败: 仅 {ready_replicas}/{replicas} 副本就绪"
                
        except Exception as e:
            return f"验证失败: {str(e)}"
    
    def _extract_pod_info(self, pod: Dict[str, Any]) -> Dict[str, Any]:
        """提取Pod关键信息"""
        try:
            metadata = pod.get('metadata', {})
            status = pod.get('status', {})
            
            return {
                'name': metadata.get('name', 'unknown'),
                'phase': status.get('phase', 'unknown'),
                'ready': self._is_pod_ready(status),
                'restart_count': self._get_restart_count(status),
                'creation_timestamp': metadata.get('creation_timestamp'),
                'conditions': status.get('conditions', [])[-3:]  # 最近3个条件
            }
        except Exception:
            return {'name': 'unknown', 'phase': 'unknown', 'ready': False}
    
    def _is_pod_ready(self, status: Dict[str, Any]) -> bool:
        """检查Pod是否就绪"""
        try:
            conditions = status.get('conditions', [])
            for condition in conditions:
                if condition.get('type') == 'Ready':
                    return condition.get('status') == 'True'
            return False
        except Exception:
            return False
    
    def _get_restart_count(self, status: Dict[str, Any]) -> int:
        """获取容器重启次数"""
        try:
            container_statuses = status.get('container_statuses', [])
            total_restarts = 0
            for container_status in container_statuses:
                total_restarts += container_status.get('restart_count', 0)
            return total_restarts
        except Exception:
            return 0
    
    async def diagnose_cluster_health(self, namespace: str = None) -> str:
        """诊断集群健康状态"""
        try:
            namespace = namespace or config.k8s.namespace
            
            # 获取节点信息（如果有权限）
            try:
                nodes_info = "节点信息需要集群级权限"
                # nodes = await self.k8s_service.get_nodes()
                # nodes_info = f"集群节点数: {len(nodes)}"
            except Exception:
                nodes_info = "无法获取节点信息"
            
            # 获取Pod状态统计
            pods = await self.k8s_service.get_pods(namespace)
            pod_phases = {}
            for pod in pods:
                phase = pod.get('status', {}).get('phase', 'Unknown')
                pod_phases[phase] = pod_phases.get(phase, 0) + 1
            
            # 获取最近事件
            events = await self.k8s_service.get_events(namespace, limit=20)
            warning_events = [e for e in events if e.get('type') == 'Warning']
            
            health_report = f"""
集群健康诊断报告:
- 命名空间: {namespace}
- {nodes_info}
- Pod状态统计: {pod_phases}
- 最近警告事件数: {len(warning_events)}
- 最近事件: {len(events)} 个

最近的警告事件:
"""
            
            for event in warning_events[:5]:
                event_time = event.get('last_timestamp', 'unknown')
                event_reason = event.get('reason', 'unknown')
                event_message = event.get('message', 'unknown')
                health_report += f"- [{event_time}] {event_reason}: {event_message}\n"
            
            return health_report
            
        except Exception as e:
            logger.error(f"集群健康诊断失败: {str(e)}")
            return f"集群健康诊断失败: {str(e)}"
    
    def get_available_tools(self) -> List[str]:
        """获取可用的修复工具"""
        return [
            "analyze_and_fix_deployment",
            "diagnose_cluster_health", 
            "restart_deployment",
            "scale_deployment",
            "check_pod_logs"
        ]