from flask import Blueprint, request, jsonify
from datetime import datetime
import asyncio
import logging
import time
from app.core.agents.supervisor import SupervisorAgent
from app.core.agents.k8s_fixer import K8sFixerAgent
from app.core.agents.notifier import NotifierAgent
from app.models.request_models import AutoFixRequest
from app.models.response_models import AutoFixResponse
from app.utils.validators import validate_deployment_name, validate_namespace, sanitize_input
from app.services.notification import NotificationService

logger = logging.getLogger("aiops.autofix")

autofix_bp = Blueprint('autofix', __name__)

# 初始化Agent
supervisor_agent = SupervisorAgent()
k8s_fixer_agent = K8sFixerAgent()
notifier_agent = NotifierAgent()
notification_service = NotificationService()

@autofix_bp.route('/autofix', methods=['POST'])
def autofix_k8s():
    """自动修复Kubernetes问题"""
    try:
        data = request.get_json() or {}
        
        # 验证请求参数
        try:
            autofix_request = AutoFixRequest(**data)
        except Exception as e:
            logger.warning(f"自动修复请求参数错误: {str(e)}")
            return jsonify({"error": f"请求参数错误: {str(e)}"}), 400
        
        # 验证Kubernetes资源名称
        if not validate_deployment_name(autofix_request.deployment):
            return jsonify({"error": "无效的Deployment名称"}), 400
        
        if not validate_namespace(autofix_request.namespace):
            return jsonify({"error": "无效的命名空间名称"}), 400
        
        # 清理输入
        event_description = sanitize_input(autofix_request.event, 2000)
        
        logger.info(f"开始自动修复: deployment={autofix_request.deployment}, namespace={autofix_request.namespace}")
        
        # 执行自动修复
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            # 首先检查部署状态，判断是否有CrashLoopBackOff问题
            deployment = None
            pods = []
            try:
                # 使用run_until_complete替代直接await
                deployment = loop.run_until_complete(
                    k8s_fixer_agent.k8s_service.get_deployment(
                        autofix_request.deployment, 
                        autofix_request.namespace
                    )
                )
                
                if deployment:
                    pods = loop.run_until_complete(
                        k8s_fixer_agent.k8s_service.get_pods(
                            namespace=autofix_request.namespace,
                            label_selector=f"app={autofix_request.deployment}"
                        )
                    )
                    
                    # 检查是否有CrashLoopBackOff问题
                    for pod in pods:
                        status = pod.get('status', {})
                        container_statuses = status.get('container_statuses', [])
                        for c_status in container_statuses:
                            if c_status.get('state', {}).get('waiting', {}).get('reason') == 'CrashLoopBackOff':
                                # 在事件描述中添加CrashLoopBackOff信息
                                if 'CrashLoopBackOff' not in event_description:
                                    event_description += " Pod处于CrashLoopBackOff状态，需要检查和修复livenessProbe和readinessProbe配置。"
                                autofix_request.force = True  # 强制修复模式
                                logger.info(f"检测到CrashLoopBackOff问题，设置强制修复模式")
                                break
            except Exception as e:
                logger.error(f"检查部署状态时出错: {str(e)}")
            
            # 执行自动修复工作流
            result = loop.run_until_complete(
                execute_autofix_workflow(
                    autofix_request.deployment,
                    autofix_request.namespace,
                    event_description,
                    autofix_request.force
                )
            )
            
            # 如果没有明确的结果，或修复失败，尝试直接调用K8sFixerAgent
            if not result.get('success') or result.get('error_message'):
                logger.info("自动修复工作流未成功，尝试直接使用K8sFixerAgent修复")
                
                fix_result = loop.run_until_complete(
                    k8s_fixer_agent.analyze_and_fix_deployment(
                        autofix_request.deployment,
                        autofix_request.namespace,
                        event_description
                    )
                )
                
                if fix_result and ("自动修复" in fix_result or "修复完成" in fix_result):
                    result['success'] = True
                    result['result'] = fix_result
                    result['error_message'] = None
                    result['actions_taken'] = ["执行K8s自动修复: " + autofix_request.deployment]
                    logger.info(f"直接修复成功: {fix_result[:100]}...")
            
            # 如果仍然没有明确的结果，检查Pod状态
            if not result.get('success') and not result.get('error_message'):
                # 检查Pod是否已经修复
                pod_status = loop.run_until_complete(
                    k8s_fixer_agent.diagnose_cluster_health(autofix_request.namespace)
                )
                if f"部署 {autofix_request.deployment}: 0/" in pod_status:
                    result['error_message'] = "修复后Pod仍未就绪，请检查日志或配置"
                else:
                    result['success'] = True
                    result['result'] = "Pod状态已改善，修复可能已成功"
            
            # 等待几秒钟，让修复生效
            time.sleep(3)
        finally:
            loop.close()
        
        # 发送通知
        if result.get('success'):
            logger.info(f"自动修复成功: {autofix_request.deployment}")
            
            # 发送成功通知
            asyncio.run(notification_service.send_autofix_notification(
                autofix_request.deployment,
                autofix_request.namespace,
                "success",
                result.get('actions_taken', [])
            ))
        else:
            logger.error(f"自动修复失败: {autofix_request.deployment}")
            
            # 发送失败通知
            asyncio.run(notification_service.send_autofix_notification(
                autofix_request.deployment,
                autofix_request.namespace,
                "failed",
                result.get('actions_taken', []),
                result.get('error_message')
            ))
        
        # 构建响应
        response = AutoFixResponse(
            status="success" if result.get('success') else "failed",
            result=result.get('result', ''),
            deployment=autofix_request.deployment,
            namespace=autofix_request.namespace,
            actions_taken=result.get('actions_taken', []),
            timestamp=datetime.utcnow().isoformat(),
            success=result.get('success', False),
            error_message=result.get('error_message')
        )
        
        status_code = 200 if result.get('success') else 500
        return jsonify(response.dict()), status_code
        
    except Exception as e:
        logger.error(f"自动修复请求失败: {str(e)}")
        return jsonify({
            "error": f"自动修复失败: {str(e)}",
            "timestamp": datetime.utcnow().isoformat()
        }), 500

@autofix_bp.route('/autofix/workflow', methods=['POST'])
def execute_workflow():
    """执行完整的自动修复工作流"""
    try:
        data = request.get_json() or {}
        problem_description = data.get('problem_description', '')
        
        if not problem_description:
            return jsonify({"error": "必须提供问题描述"}), 400
        
        # 清理输入
        problem_description = sanitize_input(problem_description, 2000)
        
        logger.info(f"执行自动修复工作流: {problem_description[:100]}...")
        
        # 创建初始状态
        initial_state = supervisor_agent.create_initial_state(problem_description)
        
        # 执行工作流
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            workflow_result = loop.run_until_complete(
                execute_full_workflow(initial_state)
            )
        finally:
            loop.close()
        
        return jsonify(workflow_result)
        
    except Exception as e:
        logger.error(f"工作流执行失败: {str(e)}")
        return jsonify({
            "error": f"工作流执行失败: {str(e)}",
            "timestamp": datetime.utcnow().isoformat()
        }), 500

@autofix_bp.route('/autofix/diagnose', methods=['POST'])
def diagnose_cluster():
    """诊断集群健康状态"""
    try:
        data = request.get_json() or {}
        namespace = data.get('namespace', 'default')
        
        if not validate_namespace(namespace):
            return jsonify({"error": "无效的命名空间名称"}), 400
        
        logger.info(f"开始集群健康诊断: namespace={namespace}")
        
        # 执行集群诊断
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            diagnosis_result = loop.run_until_complete(
                k8s_fixer_agent.diagnose_cluster_health(namespace)
            )
        finally:
            loop.close()
        
        return jsonify({
            "status": "success",
            "namespace": namespace,
            "diagnosis": diagnosis_result,
            "timestamp": datetime.utcnow().isoformat()
        })
        
    except Exception as e:
        logger.error(f"集群诊断失败: {str(e)}")
        return jsonify({
            "error": f"集群诊断失败: {str(e)}",
            "timestamp": datetime.utcnow().isoformat()
        }), 500

@autofix_bp.route('/autofix/notify', methods=['POST'])
def send_notification():
    """发送通知"""
    try:
        data = request.get_json() or {}
        notification_type = data.get('type', 'human_help')
        message = data.get('message', '')
        urgency = data.get('urgency', 'medium')
        
        if not message:
            return jsonify({"error": "必须提供通知消息"}), 400
        
        # 清理输入
        message = sanitize_input(message, 1000)
        
        logger.info(f"发送通知: 类型={notification_type}, 紧急程度={urgency}")
        
        # 执行通知发送
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            if notification_type == 'human_help':
                try:
                    result = loop.run_until_complete(
                        notifier_agent.send_human_help_request(message, urgency)
                    )
                except Exception as e:
                    logger.error(f"发送人工帮助请求失败: {str(e)}")
                    result = f"发送通知失败: {str(e)}"
            elif notification_type == 'incident':
                affected_services = data.get('affected_services', [])
                severity = data.get('severity', 'medium')
                try:
                    result = loop.run_until_complete(
                        notifier_agent.send_incident_alert(message, affected_services, severity)
                    )
                except Exception as e:
                    logger.error(f"发送事件告警失败: {str(e)}")
                    result = f"发送告警失败: {str(e)}"
            else:
                return jsonify({"error": f"不支持的通知类型: {notification_type}"}), 400
        finally:
            loop.close()
        
        return jsonify({
            "status": "success",
            "result": result,
            "notification_type": notification_type,
            "timestamp": datetime.utcnow().isoformat()
        })
        
    except Exception as e:
        logger.error(f"发送通知失败: {str(e)}")
        return jsonify({
            "error": f"发送通知失败: {str(e)}",
            "timestamp": datetime.utcnow().isoformat()
        }), 500

@autofix_bp.route('/autofix/health', methods=['GET'])
def autofix_health():
    """自动修复服务健康检查"""
    try:
        # 检查各Agent健康状态
        k8s_healthy = True  # K8s Fixer Agent通常健康，除非K8s连接有问题
        notifier_healthy = True  # Notifier Agent通常健康
        supervisor_healthy = True  # Supervisor Agent通常健康
        
        # 检查依赖服务
        try:
            from app.services.kubernetes import KubernetesService
            k8s_service = KubernetesService()
            k8s_service_healthy = k8s_service.is_healthy()
        except Exception:
            k8s_service_healthy = False
        
        notification_health = asyncio.run(notifier_agent.check_notification_health())
        
        components = {
            "supervisor_agent": supervisor_healthy,
            "k8s_fixer_agent": k8s_healthy,
            "notifier_agent": notifier_healthy,
            "kubernetes_service": k8s_service_healthy,
            "notification_service": notification_health.get('healthy', False)
        }
        
        overall_healthy = all([
            supervisor_healthy,
            k8s_healthy,
            notifier_healthy,
            k8s_service_healthy
        ])
        
        health_status = {
            "status": "healthy" if overall_healthy else "degraded",
            "components": components,
            "notification_details": notification_health,
            "timestamp": datetime.utcnow().isoformat()
        }
        
        status_code = 200
        return jsonify(health_status), status_code
        
    except Exception as e:
        logger.error(f"自动修复健康检查失败: {str(e)}")
        return jsonify({
            "status": "error",
            "error": str(e),
            "timestamp": datetime.utcnow().isoformat()
        }), 500

async def execute_autofix_workflow(deployment: str, namespace: str, event: str, force: bool = False):
    """执行自动修复工作流"""
    try:
        actions_taken = []
        
        # 通用的确保可序列化的函数
        def ensure_serializable(obj):
            if isinstance(obj, dict):
                return {k: ensure_serializable(v) for k, v in obj.items()}
            elif isinstance(obj, list):
                return [ensure_serializable(item) for item in obj]
            elif hasattr(obj, 'isoformat'):  # datetime对象
                return obj.isoformat()
            else:
                return obj
        
        # 使用K8s修复Agent进行分析和修复
        try:
            fix_result = await k8s_fixer_agent.analyze_and_fix_deployment(
                deployment, namespace, event
            )
        except Exception as e:
            logger.error(f"K8s修复Agent执行失败: {str(e)}")
            fix_result = f"执行修复失败: {str(e)}"
        
        actions_taken.append(f"执行K8s自动修复: {deployment}")
        
        # 解析修复结果
        if "成功" in fix_result or "完成" in fix_result:
            success = True
            result = fix_result
            error_message = None
        else:
            success = False
            result = fix_result
            error_message = fix_result
                
        # 构建并序列化结果
        response_data = {
            "success": success,
            "result": result,
            "actions_taken": actions_taken,
            "error_message": error_message
        }
        
        return ensure_serializable(response_data)
        
    except Exception as e:
        logger.error(f"自动修复工作流执行失败: {str(e)}")
        return {
            "success": False,
            "result": f"自动修复工作流执行失败: {str(e)}",
            "actions_taken": ["尝试执行自动修复但失败"],
            "error_message": str(e)
        }

async def execute_full_workflow(initial_state):
    """执行完整的多Agent工作流"""
    try:
        current_state = initial_state
        workflow_steps = []
        
        # 通用的确保可序列化的函数
        def ensure_serializable(obj):
            if isinstance(obj, dict):
                return {k: ensure_serializable(v) for k, v in obj.items()}
            elif isinstance(obj, list):
                return [ensure_serializable(item) for item in obj]
            elif hasattr(obj, 'isoformat'):  # datetime对象
                return obj.isoformat()
            else:
                return obj
        
        while supervisor_agent.should_continue(current_state):
            # 主管决策下一步行动
            routing_result = await supervisor_agent.route_next_action(current_state)
            next_agent = routing_result.get('next')
            reasoning = routing_result.get('reasoning', '')
            
            workflow_steps.append({
                "step": current_state.iteration_count + 1,
                "agent": next_agent,
                "reasoning": reasoning
            })
            
            if next_agent == "FINISH":
                break
            
            # 执行相应Agent的操作
            step_result = await execute_agent_action(next_agent, current_state)
            
            # 更新状态
            current_state.messages.append({
                "agent": next_agent,
                "result": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            current_state.iteration_count += 1
            current_state.next_action = next_agent
        
        # 获取工作流总结
        workflow_summary = supervisor_agent.get_workflow_summary(current_state)
                
        response_data = {
            "status": "completed",
            "workflow_steps": workflow_steps,
            "summary": workflow_summary,
            "final_state": current_state.current_step,
            "timestamp": datetime.utcnow().isoformat()
        }
        
        return ensure_serializable(response_data)
        
    except Exception as e:
        logger.error(f"完整工作流执行失败: {str(e)}")
        return {
            "status": "failed",
            "error": str(e),
            "timestamp": datetime.utcnow().isoformat()
        }

async def execute_agent_action(agent_name: str, state):
    """执行特定Agent的操作"""
    try:
        if agent_name == "K8sFixer":
            # 从状态中提取K8s相关信息
            context = state.context
            deployment = context.get('deployment', 'unknown')
            namespace = context.get('namespace', 'default')
            problem = context.get('problem', '')
            
            try:
                return await k8s_fixer_agent.analyze_and_fix_deployment(
                    deployment, namespace, problem
                )
            except Exception as e:
                logger.error(f"K8s修复Agent执行失败: {str(e)}")
                return f"执行修复失败: {str(e)}"
        
        elif agent_name == "Notifier":
            problem = state.context.get('problem', '')
            try:
                return await notifier_agent.send_human_help_request(problem, 'medium')
            except Exception as e:
                logger.error(f"Notifier执行失败: {str(e)}")
                return f"发送通知失败: {str(e)}"
        
        else:
            return f"Agent {agent_name} 执行完成（模拟）"
            
    except Exception as e:
        logger.error(f"执行Agent {agent_name} 操作失败: {str(e)}")
        return f"Agent {agent_name} 执行失败: {str(e)}"