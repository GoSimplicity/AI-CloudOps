import pytest
import asyncio
import os
import sys
import tempfile
from pathlib import Path
# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

@pytest.fixture
def app():
    """创建测试用的Flask应用"""
    from app.main import create_app
    
    app = create_app()
    app.config['TESTING'] = True
    app.config['DEBUG'] = False
    
    return app

@pytest.fixture
def client(app):
    """创建测试客户端"""
    return app.test_client()

@pytest.fixture
def prometheus_service():
    """获取Prometheus服务实例"""
    from app.services.prometheus import PrometheusService
    return PrometheusService()

@pytest.fixture
def k8s_service():
    """获取Kubernetes服务实例"""
    from app.services.kubernetes import KubernetesService
    return KubernetesService()

@pytest.fixture
def llm_service():
    """获取LLM服务实例"""
    from app.services.llm import LLMService
    return LLMService()

@pytest.fixture
def prediction_service():
    """获取预测服务实例"""
    from app.core.prediction.predictor import PredictionService
    return PredictionService()

@pytest.fixture
def sample_rca_request():
    """示例RCA请求数据"""
    return {
        "start_time": "2024-01-01T10:00:00Z",
        "end_time": "2024-01-01T11:00:00Z",
        "metrics": ["container_cpu_usage_seconds_total"]
    }

@pytest.fixture
def sample_autofix_request():
    """示例自动修复请求数据"""
    return {
        "deployment": "test-app",
        "namespace": "default",
        "event": "Pod启动失败"
    }

@pytest.fixture
def real_knowledge_base():
    """使用真实知识库目录"""
    from app.config.settings import config
    return config.rag.knowledge_base_path

@pytest.fixture
def sample_document():
    """示例知识库文档"""
    return """
# AIOps平台说明文档

## 简介

AIOps平台是一个智能运维系统，提供根因分析、自动修复和负载预测功能。

## 核心功能

1. 智能根因分析
2. Kubernetes自动修复
3. 基于机器学习的负载预测

## 系统架构

AIOps平台采用微服务架构，包括API网关、核心业务逻辑和服务层。

## 联系方式

如有问题请联系开发团队：support@example.com
"""

@pytest.fixture
def event_loop():
    """创建事件循环"""
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()

@pytest.fixture(autouse=True)
def setup_test_environment():
    """设置测试环境"""
    # 设置测试环境变量
    os.environ['DEBUG'] = 'true'
    os.environ['TESTING'] = 'true'
    os.environ['LOG_LEVEL'] = 'WARNING'  # 减少测试时的日志输出
    
    yield
    
    # 清理环境变量
    for key in ['DEBUG', 'TESTING', 'LOG_LEVEL']:
        os.environ.pop(key, None)