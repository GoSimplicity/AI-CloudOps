from flask import Blueprint, jsonify
from datetime import datetime
import time
import psutil
import logging
from app.services.prometheus import PrometheusService
from app.services.kubernetes import KubernetesService
from app.services.llm import LLMService
from app.services.notification import NotificationService
from app.core.prediction.predictor import PredictionService
from app.models.response_models import APIResponse

logger = logging.getLogger("aiops.health")

health_bp = Blueprint('health', __name__)

# 应用启动时间
start_time = time.time()

@health_bp.route('/health', methods=['GET'])
def health_check():
    """系统健康检查"""
    try:
        # 基本健康状态
        current_time = datetime.utcnow()
        uptime = time.time() - start_time
        
        # 检查各组件状态
        components_status = check_components_health()
        
        # 系统资源状态
        system_status = get_system_status()
        
        # 判断整体健康状态
        is_healthy = all(components_status.values())
        
        health_data = {
            "status": "healthy" if is_healthy else "unhealthy",
            "timestamp": current_time.isoformat(),
            "uptime": round(uptime, 2),
            "version": "1.0.0",
            "components": components_status,
            "system": system_status
        }
        
        return jsonify(APIResponse(
            code=0,
            message="健康检查完成",
            data=health_data
        ).dict())
        
    except Exception as e:
        logger.error(f"健康检查失败: {str(e)}")
        return jsonify(APIResponse(
            code=500,
            message=f"健康检查失败: {str(e)}",
            data={"timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

@health_bp.route('/health/components', methods=['GET'])
def components_health():
    """详细的组件健康检查"""
    try:
        components_detail = {}
        
        # Prometheus服务
        prometheus_service = PrometheusService()
        prometheus_healthy = prometheus_service.is_healthy()
        components_detail["prometheus"] = {
            "healthy": prometheus_healthy,
            "url": prometheus_service.base_url,
            "timeout": prometheus_service.timeout
        }
        
        # Kubernetes服务
        try:
            k8s_service = KubernetesService()
            k8s_healthy = k8s_service.is_healthy()
            components_detail["kubernetes"] = {
                "healthy": k8s_healthy,
                "in_cluster": k8s_service.k8s_config.in_cluster if hasattr(k8s_service, 'k8s_config') else False
            }
        except Exception as e:
            components_detail["kubernetes"] = {
                "healthy": False,
                "error": str(e)
            }
        
        # LLM服务
        try:
            llm_service = LLMService()
            llm_healthy = llm_service.is_healthy()
            components_detail["llm"] = {
                "healthy": llm_healthy,
                "model": llm_service.model,
                "base_url": llm_service.client.base_url
            }
        except Exception as e:
            components_detail["llm"] = {
                "healthy": False,
                "error": str(e)
            }
        
        # 通知服务
        try:
            notification_service = NotificationService()
            notification_healthy = notification_service.is_healthy()
            components_detail["notification"] = {
                "healthy": notification_healthy,
                "enabled": notification_service.enabled
            }
        except Exception as e:
            components_detail["notification"] = {
                "healthy": False,
                "error": str(e)
            }
        
        # 预测服务
        try:
            prediction_service = PredictionService()
            prediction_healthy = prediction_service.is_healthy()
            components_detail["prediction"] = {
                "healthy": prediction_healthy,
                "model_loaded": prediction_service.model_loaded,
                "scaler_loaded": prediction_service.scaler_loaded
            }
        except Exception as e:
            components_detail["prediction"] = {
                "healthy": False,
                "error": str(e)
            }
        
        return jsonify(APIResponse(
            code=0,
            message="组件健康检查完成",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "components": components_detail
            }
        ).dict())
        
    except Exception as e:
        logger.error(f"组件健康检查失败: {str(e)}")
        return jsonify(APIResponse(
            code=500,
            message=f"组件健康检查失败: {str(e)}",
            data={"timestamp": datetime.utcnow().isoformat()}
        ).dict()), 500

@health_bp.route('/health/metrics', methods=['GET'])
def health_metrics():
    """健康指标"""
    try:
        # 系统指标
        cpu_percent = psutil.cpu_percent(interval=1)
        memory = psutil.virtual_memory()
        disk = psutil.disk_usage('/')
        
        # 网络指标
        network = psutil.net_io_counters()
        
        # 进程指标
        process = psutil.Process()
        process_memory = process.memory_info()
        
        metrics = {
            "timestamp": datetime.utcnow().isoformat(),
            "system": {
                "cpu_percent": cpu_percent,
                "memory_percent": memory.percent,
                "memory_available": memory.available,
                "memory_total": memory.total,
                "disk_percent": (disk.used / disk.total) * 100,
                "disk_free": disk.free,
                "disk_total": disk.total
            },
            "network": {
                "bytes_sent": network.bytes_sent,
                "bytes_recv": network.bytes_recv,
                "packets_sent": network.packets_sent,
                "packets_recv": network.packets_recv
            },
            "process": {
                "memory_rss": process_memory.rss,
                "memory_vms": process_memory.vms,
                "cpu_percent": process.cpu_percent(),
                "num_threads": process.num_threads(),
                "create_time": process.create_time()
            },
            "uptime": time.time() - start_time
        }
        
        return jsonify(APIResponse(
            code=0,
            message="健康指标获取成功",
            data=metrics
        ).dict())
        
    except Exception as e:
        logger.error(f"获取健康指标失败: {str(e)}")
        return jsonify(APIResponse(
            code=500,
            message=f"获取健康指标失败: {str(e)}",
            data={"timestamp": datetime.datetime.utcnow().isoformat()}
        ).dict()), 500

@health_bp.route('/health/ready', methods=['GET'])
def readiness_probe():
    """就绪性探针"""
    try:
        # 检查关键组件是否就绪
        components_status = check_components_health()
        
        # 至少需要基本组件就绪
        required_components = ["prometheus", "prediction"]
        ready = all(components_status.get(comp, False) for comp in required_components)
        
        if ready:
            return jsonify(APIResponse(
                code=0,
                message="服务就绪",
                data={
                    "status": "ready",
                    "timestamp": datetime.utcnow().isoformat()
                }
            ).dict())
        else:
            return jsonify(APIResponse(
                code=503,
                message="服务未就绪",
                data={
                    "status": "not ready",
                    "timestamp": datetime.utcnow().isoformat(),
                    "components": components_status
                }
            ).dict()), 503
            
    except Exception as e:
        logger.error(f"就绪性检查失败: {str(e)}")
        return jsonify(APIResponse(
            code=500,
            message=f"就绪性检查失败: {str(e)}",
            data={
                "status": "error",
                "timestamp": datetime.utcnow().isoformat()
            }
        ).dict()), 500

@health_bp.route('/health/live', methods=['GET'])
def liveness_probe():
    """存活性探针"""
    try:
        # 简单的存活性检查
        return jsonify(APIResponse(
            code=0,
            message="服务存活",
            data={
                "status": "alive",
                "timestamp": datetime.utcnow().isoformat(),
                "uptime": time.time() - start_time
            }
        ).dict())
        
    except Exception as e:
        logger.error(f"存活性检查失败: {str(e)}")
        return jsonify(APIResponse(
            code=500,
            message=f"存活性检查失败: {str(e)}",
            data={
                "status": "error",
                "timestamp": datetime.utcnow().isoformat()
            }
        ).dict()), 500

def check_components_health():
    """检查各组件健康状态"""
    components_status = {}
    
    # Prometheus
    try:
        prometheus_service = PrometheusService()
        components_status["prometheus"] = prometheus_service.is_healthy()
    except Exception:
        components_status["prometheus"] = False
    
    # Kubernetes
    try:
        k8s_service = KubernetesService()
        components_status["kubernetes"] = k8s_service.is_healthy()
    except Exception:
        components_status["kubernetes"] = False
    
    # LLM
    try:
        llm_service = LLMService()
        components_status["llm"] = llm_service.is_healthy()
    except Exception:
        components_status["llm"] = False
    
    # 通知服务
    try:
        notification_service = NotificationService()
        components_status["notification"] = notification_service.is_healthy()
    except Exception:
        components_status["notification"] = False
    
    # 预测服务
    try:
        prediction_service = PredictionService()
        components_status["prediction"] = prediction_service.is_healthy()
    except Exception:
        components_status["prediction"] = False
    
    return components_status

def get_system_status():
    """获取系统资源状态"""
    cpu_percent = psutil.cpu_percent(interval=1)
    memory = psutil.virtual_memory()
    disk = psutil.disk_usage('/')
    
    system_status = {
        "cpu_usage_percent": cpu_percent,
        "memory_usage_percent": memory.percent,
        "disk_usage_percent": (disk.used / disk.total) * 100,
        "memory_available_mb": memory.available / (1024 * 1024),
        "disk_free_gb": disk.free / (1024 * 1024 * 1024)
    }
    
    return system_status