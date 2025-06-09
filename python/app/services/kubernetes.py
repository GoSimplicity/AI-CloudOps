import logging
import json
import yaml
from typing import Dict, Any, Optional, List
from datetime import datetime
from kubernetes import client, config as k8s_config
from kubernetes.client.rest import ApiException
from app.config.settings import config

logger = logging.getLogger("aiops.kubernetes")

class KubernetesService:
    def __init__(self):
        self._load_config()
        self.apps_v1 = client.AppsV1Api()
        self.core_v1 = client.CoreV1Api()
        logger.info("Kubernetes服务初始化完成")
    
    def _load_config(self):
        """加载Kubernetes配置"""
        try:
            if config.k8s.in_cluster:
                k8s_config.load_incluster_config()
                logger.info("使用集群内K8s配置")
            else:
                k8s_config.load_kube_config(config_file=config.k8s.config_path)
                logger.info("使用本地K8s配置")
        except Exception as e:
            logger.error(f"无法加载K8s配置: {str(e)}")
            raise
    
    async def get_deployment(self, name: str, namespace: str = None) -> Optional[Dict]:
        """获取Deployment信息"""
        try:
            namespace = namespace or config.k8s.namespace
            deployment = self.apps_v1.read_namespaced_deployment(
                name=name, namespace=namespace
            )
            
            deployment_dict = deployment.to_dict()
            # 清理敏感信息
            if 'metadata' in deployment_dict:
                metadata = deployment_dict['metadata']
                for key in ['managed_fields', 'resource_version', 'uid', 'self_link']:
                    metadata.pop(key, None)
            
            logger.info(f"获取Deployment成功: {name}")
            return deployment_dict
            
        except ApiException as e:
            logger.error(f"获取Deployment失败: {str(e)}")
            return None
        except Exception as e:
            logger.error(f"获取Deployment异常: {str(e)}")
            return None
    
    async def patch_deployment(
        self, 
        name: str, 
        patch: Dict[str, Any], 
        namespace: str = None
    ) -> bool:
        """更新Deployment"""
        try:
            namespace = namespace or config.k8s.namespace
            
            logger.info(f"更新Deployment: {name}, patch: {json.dumps(patch, indent=2)}")
            
            self.apps_v1.patch_namespaced_deployment(
                name=name,
                namespace=namespace,
                body=patch
            )
            
            logger.info(f"成功更新Deployment {name}")
            return True
            
        except ApiException as e:
            logger.error(f"更新Deployment失败: {str(e)}")
            return False
        except Exception as e:
            logger.error(f"更新Deployment异常: {str(e)}")
            return False
    
    async def get_pods(self, namespace: str = None, label_selector: str = None) -> List[Dict]:
        """获取Pod列表"""
        try:
            namespace = namespace or config.k8s.namespace
            pods = self.core_v1.list_namespaced_pod(
                namespace=namespace,
                label_selector=label_selector
            )
            
            pod_list = []
            for pod in pods.items:
                pod_dict = pod.to_dict()
                # 清理不必要的字段
                if 'metadata' in pod_dict:
                    metadata = pod_dict['metadata']
                    for key in ['managed_fields', 'resource_version', 'uid']:
                        metadata.pop(key, None)
                pod_list.append(pod_dict)
            
            logger.info(f"获取到 {len(pod_list)} 个Pod")
            return pod_list
            
        except ApiException as e:
            logger.error(f"获取Pod列表失败: {str(e)}")
            return []
        except Exception as e:
            logger.error(f"获取Pod列表异常: {str(e)}")
            return []
    
    async def get_events(
        self, 
        namespace: str = None, 
        field_selector: str = None,
        limit: int = 100
    ) -> List[Dict]:
        """获取事件列表"""
        try:
            namespace = namespace or config.k8s.namespace
            events = self.core_v1.list_namespaced_event(
                namespace=namespace,
                field_selector=field_selector,
                limit=limit
            )
            
            event_list = []
            for event in events.items:
                event_dict = event.to_dict()
                # 清理不必要的字段
                if 'metadata' in event_dict:
                    metadata = event_dict['metadata']
                    for key in ['managed_fields', 'resource_version', 'uid']:
                        metadata.pop(key, None)
                event_list.append(event_dict)
            
            logger.info(f"获取到 {len(event_list)} 个事件")
            return event_list
            
        except ApiException as e:
            logger.error(f"获取事件列表失败: {str(e)}")
            return []
        except Exception as e:
            logger.error(f"获取事件列表异常: {str(e)}")
            return []
    
    async def restart_deployment(self, name: str, namespace: str = None) -> bool:
        """重启Deployment"""
        try:
            namespace = namespace or config.k8s.namespace
            
            # 添加重启注解
            patch = {
                "spec": {
                    "template": {
                        "metadata": {
                            "annotations": {
                                "kubectl.kubernetes.io/restartedAt": 
                                    datetime.utcnow().isoformat()
                            }
                        }
                    }
                }
            }
            
            result = await self.patch_deployment(name, patch, namespace)
            if result:
                logger.info(f"成功重启Deployment: {name}")
            
            return result
            
        except Exception as e:
            logger.error(f"重启Deployment失败: {str(e)}")
            return False
    
    async def scale_deployment(self, name: str, replicas: int, namespace: str = None) -> bool:
        """扩缩容Deployment"""
        try:
            namespace = namespace or config.k8s.namespace
            
            patch = {
                "spec": {
                    "replicas": replicas
                }
            }
            
            result = await self.patch_deployment(name, patch, namespace)
            if result:
                logger.info(f"成功扩缩容Deployment {name} 到 {replicas} 副本")
            
            return result
            
        except Exception as e:
            logger.error(f"扩缩容Deployment失败: {str(e)}")
            return False
    
    def is_healthy(self) -> bool:
        """检查Kubernetes连接健康状态"""
        try:
            self.core_v1.get_api_versions()
            logger.debug("Kubernetes连接健康")
            return True
        except Exception as e:
            logger.error(f"Kubernetes连接异常: {str(e)}")
            return False
    
    async def get_deployment_status(self, name: str, namespace: str = None) -> Optional[Dict[str, Any]]:
        """获取Deployment状态详情"""
        try:
            deployment = await self.get_deployment(name, namespace)
            if not deployment:
                return None
            
            status = deployment.get('status', {})
            spec = deployment.get('spec', {})
            
            return {
                'name': name,
                'namespace': namespace or config.k8s.namespace,
                'replicas': spec.get('replicas', 0),
                'ready_replicas': status.get('ready_replicas', 0),
                'available_replicas': status.get('available_replicas', 0),
                'updated_replicas': status.get('updated_replicas', 0),
                'conditions': status.get('conditions', []),
                'strategy': spec.get('strategy', {}),
                'creation_timestamp': deployment.get('metadata', {}).get('creation_timestamp')
            }
            
        except Exception as e:
            logger.error(f"获取Deployment状态失败: {str(e)}")
            return None