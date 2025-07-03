from flask import Blueprint, request, jsonify
from datetime import datetime
import asyncio
import logging
import time
from app.core.agents.supervisor import SupervisorAgent
from app.core.agents.k8s_fixer import K8sFixerAgent
from app.core.agents.notifier import NotifierAgent
from app.models.request_models import AutoFixRequest
from app.models.response_models import AutoFixResponse, APIResponse
from app.utils.validators import validate_deployment_name, validate_namespace, sanitize_input
from app.services.notification import NotificationService
from app.config.settings import config

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
            return jsonify(APIResponse(code=400, message=f"请求参数错误: {str(e)}", data={}).dict()), 400
        
        # 验证Kubernetes资源名称
        if not validate_deployment_name(autofix_request.deployment):
            return jsonify(APIResponse(code=400, message="无效的Deployment名称", data={}).dict()), 400
        
        if not validate_namespace(autofix_request.namespace):
            return jsonify(APIResponse(code=400, message="无效的命名空间名称", data={}).dict()), 400
        
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
        
        message = "自动修复成功" if result.get('success') else f"自动修复失败: {result.get('error_message', '')}"
        code = 0 if result.get('success') else 500
        
        return jsonify(APIResponse(
            code=code,
            message=message,
            data=response.dict()
        ).dict())
        
    except Exception as e:
        logger.error(f"自动修复请求失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"自动修复失败: {str(e)}", 
            data={"timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

@autofix_bp.route('/autofix/workflow', methods=['POST'])
def execute_workflow():
    """执行完整的自动修复工作流"""
    try:
        data = request.get_json() or {}
        problem_description = data.get('problem_description', '')
        
        if not problem_description:
            return jsonify(APIResponse(code=400, message="必须提供问题描述", data={}).dict()), 400
        
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
        
        return jsonify(APIResponse(code=0, message="工作流执行完成", data=workflow_result).dict())
        
    except Exception as e:
        logger.error(f"工作流执行失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"工作流执行失败: {str(e)}", 
            data={"timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

@autofix_bp.route('/autofix/diagnose', methods=['POST'])
def diagnose_cluster():
    """诊断集群健康状态"""
    try:
        data = request.get_json() or {}
        namespace = data.get('namespace', 'default')
        
        if not validate_namespace(namespace):
            return jsonify(APIResponse(code=400, message="无效的命名空间名称", data={}).dict()), 400
        
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
        
        return jsonify(APIResponse(
            code=0,
            message="集群诊断完成",
            data={
                "status": "success",
                "namespace": namespace,
                "diagnosis": diagnosis_result,
                "timestamp": datetime.utcnow().isoformat()
            }
        ).dict())
        
    except Exception as e:
        logger.error(f"集群诊断失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"集群诊断失败: {str(e)}", 
            data={"timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

@autofix_bp.route('/autofix/notify', methods=['POST'])
def send_notification():
    """发送通知"""
    try:
        data = request.get_json() or {}
        
        # 验证必要参数
        if not data.get('message'):
            return jsonify(APIResponse(code=400, message="必须提供通知消息", data={}).dict()), 400
        
        notification_type = data.get('type', 'info')
        title = data.get('title', '系统通知')
        message = data.get('message')
        
        logger.info(f"发送通知: {title}, 类型={notification_type}")
        
        # 发送通知
        asyncio.run(notification_service.send_notification(
            title=title,
            message=message,
            notification_type=notification_type
        ))
        
        return jsonify(APIResponse(
            code=0,
            message="通知发送成功",
            data={
                "success": True,
                "type": notification_type,
                "timestamp": datetime.utcnow().isoformat()
            }
        ).dict())
        
    except Exception as e:
        logger.error(f"发送通知失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"发送通知失败: {str(e)}", 
            data={"timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

@autofix_bp.route('/autofix/health', methods=['GET'])
def autofix_health():
    """自动修复服务健康检查"""
    try:
        # 检查各组件状态
        k8s_healthy = k8s_fixer_agent.k8s_service.is_healthy()
        llm_healthy = k8s_fixer_agent.llm_service.is_healthy()
        notification_healthy = notification_service.is_healthy()
        
        # 总体状态
        overall_status = k8s_healthy  # 只要k8s正常，其他组件可以通过备用服务实现
        
        # 确定LLM状态描述
        llm_status = "primary" if llm_healthy else "backup"
        if not llm_healthy:
            try:
                # 尝试检查备用服务
                if config.llm.provider.lower() == "openai":
                    # 检查Ollama
                    import ollama
                    # 使用环境变量设置Ollama主机
                    os.environ["OLLAMA_HOST"] = config.llm.ollama_base_url.replace("/v1", "")
                    models = ollama.list().get('models', [])
                    backup_healthy = any(model.get('name', '').startswith(config.llm.ollama_model.split(':')[0]) for model in models)
                    if backup_healthy:
                        llm_status = "backup"
                        logger.info("主要LLM服务不可用，但备用Ollama服务正常")
                    else:
                        llm_status = "unavailable"
                else:
                    # 检查OpenAI
                    from openai import OpenAI
                    client = OpenAI(api_key=config.llm.api_key, base_url=config.llm.base_url)
                    response = client.chat.completions.create(
                        model=config.llm.model,
                        messages=[{"role": "user", "content": "hi"}],
                        max_tokens=5
                    )
                    backup_healthy = response.choices[0].message.content is not None
                    if backup_healthy:
                        llm_status = "backup"
                        logger.info("主要LLM服务不可用，但备用OpenAI服务正常")
                    else:
                        llm_status = "unavailable"
            except Exception as e:
                logger.warning(f"备用LLM服务检查失败: {str(e)}")
                llm_status = "unavailable"
        
        health_status = {
            "status": "healthy" if overall_status else "degraded",
            "components": {
                "kubernetes": k8s_healthy,
                "llm": llm_status,  # 现在返回状态描述而不是布尔值
                "notification": notification_healthy,
                "supervisor": True  # 纯计算模块，通常健康
            },
            "timestamp": datetime.utcnow().isoformat()
        }
        
        return jsonify(APIResponse(
            code=0,
            message="自动修复服务健康检查完成",
            data=health_status
        ).dict())
        
    except Exception as e:
        logger.error(f"自动修复健康检查失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"健康检查失败: {str(e)}", 
            data={"status": "error", "timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

async def execute_autofix_workflow(deployment: str, namespace: str, event: str, force: bool = False):
    """执行自动修复工作流"""
    try:
        # 创建工作流初始状态
        problem_description = f"Kubernetes部署 {deployment} 在命名空间 {namespace} 中出现问题：{event}"
        
        # 创建初始状态，同时添加部署特定信息到context中
        initial_state = supervisor_agent.create_initial_state(problem_description)
        
        # 更新初始上下文，而不是直接对AgentState赋值
        # 使用dataclasses方法创建更新的上下文
        from dataclasses import asdict, replace
        
        # 获取当前上下文的副本
        context = dict(initial_state.context)
        
        # 更新上下文
        context['deployment'] = deployment
        context['namespace'] = namespace
        context['force'] = force
        
        # 使用replace创建新的AgentState实例
        initial_state = replace(initial_state, context=context)
        
        logger.info(f"创建自动修复工作流初始状态: deployment={deployment}, namespace={namespace}, force={force}")
        
        # 执行步骤1: 分析和诊断
        state = await execute_agent_action('researcher', initial_state)
        
        # 步骤2: 自动修复
        state = await execute_agent_action('k8s_fixer', state)
        
        # 处理结果
        result = {
            'success': getattr(state, 'success', False),
            'result': getattr(state, 'result', ''),
            'actions_taken': getattr(state, 'actions_taken', []),
            'error_message': getattr(state, 'error', None)
        }
        
        return result
    
    except Exception as e:
        logger.error(f"执行自动修复工作流失败: {str(e)}")
        return {
            'success': False,
            'error_message': f"执行修复工作流失败: {str(e)}"
        }

async def execute_full_workflow(initial_state):
    """执行完整工作流"""
    try:
        # 不需要将AgentState对象转换为可序列化的字典
        state = initial_state
        
        # 步骤1: 研究与分析
        logger.info("步骤1: 执行研究与分析")
        state = await execute_agent_action('researcher', state)
        
        # 步骤2: K8s修复
        logger.info("步骤2: 执行Kubernetes修复")
        state = await execute_agent_action('k8s_fixer', state)
        
        # 步骤3: 通知
        logger.info("步骤3: 发送通知")
        state = await execute_agent_action('notifier', state)
        
        # 步骤4: 总结
        logger.info("步骤4: 由监督者总结")
        state = await supervisor_agent.process_agent_state(state)
        
        # 从AgentState中获取结果
        from dataclasses import asdict
        # 获取当前上下文
        context = dict(state.context)
        
        # 生成最终结果
        result = {
            'success': context.get('success', False),
            'result': context.get('result', ''),
            'actions_taken': context.get('actions_taken', []),
            'summary': context.get('summary', ''),
            'steps': context.get('steps', []),
            'error': context.get('error')
        }
        
        return result
    
    except Exception as e:
        logger.error(f"执行完整工作流失败: {str(e)}")
        return {
            'success': False,
            'error': f"执行完整工作流失败: {str(e)}"
        }

async def execute_agent_action(agent_name: str, state):
    """执行特定智能体动作"""
    try:
        logger.info(f"执行智能体: {agent_name}")
        
        if agent_name == 'researcher':
            # 研究者智能体
            from app.core.agents.researcher import ResearcherAgent
            agent = ResearcherAgent()
            updated_state = await agent.process_agent_state(state)
            
        elif agent_name == 'k8s_fixer':
            # K8s修复智能体
            updated_state = await k8s_fixer_agent.process_agent_state(state)
            
        elif agent_name == 'notifier':
            # 通知智能体
            updated_state = await notifier_agent.process_agent_state(state)
            
        else:
            logger.warning(f"未知的智能体: {agent_name}")
            updated_state = state
            # 使用dataclasses更新error属性
            from dataclasses import replace
            context = dict(updated_state.context)
            context['error'] = f"未知的智能体: {agent_name}"
            updated_state = replace(updated_state, context=context)
        
        return updated_state
        
    except Exception as e:
        logger.error(f"执行智能体 {agent_name} 失败: {str(e)}")
        # 使用dataclasses更新error属性
        from dataclasses import replace
        context = dict(state.context)
        context['error'] = f"执行智能体 {agent_name} 失败: {str(e)}"
        return replace(state, context=context)