from datetime import datetime
from typing import List, Optional
import re

def validate_time_range(start_time: datetime, end_time: datetime, max_range_minutes: int = 1440) -> bool:
    """验证时间范围"""
    from app.utils.time_utils import TimeUtils
    return TimeUtils.validate_time_range(start_time, end_time, max_range_minutes)

def validate_metric_name(metric_name: str) -> bool:
    """验证指标名称格式"""
    if not metric_name or not isinstance(metric_name, str):
        return False
    
    # 指标名称应该符合Prometheus命名规范
    pattern = r'^[a-zA-Z_:][a-zA-Z0-9_:]*$'
    return bool(re.match(pattern, metric_name))

def validate_deployment_name(deployment_name: str) -> bool:
    """验证Kubernetes Deployment名称"""
    if not deployment_name or not isinstance(deployment_name, str):
        return False
    
    # Kubernetes资源名称规范
    pattern = r'^[a-z0-9]([-a-z0-9]*[a-z0-9])?$'
    return bool(re.match(pattern, deployment_name)) and len(deployment_name) <= 253

def validate_namespace(namespace: str) -> bool:
    """验证Kubernetes命名空间"""
    if not namespace or not isinstance(namespace, str):
        return False
    
    # Kubernetes命名空间规范
    pattern = r'^[a-z0-9]([-a-z0-9]*[a-z0-9])?$'
    return bool(re.match(pattern, namespace)) and len(namespace) <= 63

def validate_qps(qps: float) -> bool:
    """验证QPS值"""
    return isinstance(qps, (int, float)) and qps >= 0

def validate_confidence(confidence: float) -> bool:
    """验证置信度值"""
    return isinstance(confidence, (int, float)) and 0 <= confidence <= 1

def validate_metric_list(metrics: List[str]) -> bool:
    """验证指标列表"""
    if not metrics or not isinstance(metrics, list):
        return False
    
    return all(validate_metric_name(metric) for metric in metrics)

def sanitize_input(text: str, max_length: int = 1000) -> str:
    """清理输入文本"""
    if not isinstance(text, str):
        return ""
    
    # 移除危险字符
    sanitized = re.sub(r'[<>&"\'`]', '', text)
    
    # 限制长度
    return sanitized[:max_length]