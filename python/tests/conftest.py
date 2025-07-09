import pytest
import asyncio
import os
import sys
import tempfile
import yaml
from pathlib import Path
# 添加项目路径到sys.path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

@pytest.fixture(scope="session", autouse=True)
def setup_test_config():
    """设置测试配置文件"""
    # 设置环境为测试环境
    os.environ['ENV'] = 'test'
    
    # 创建测试配置文件
    config_dir = Path(__file__).parent.parent / "config"
    config_dir.mkdir(exist_ok=True)
    
    test_config_path = config_dir / "config.test.yaml"
    if not test_config_path.exists():
        # 基于默认配置创建测试配置
        default_config_path = config_dir / "config.yaml"
        if default_config_path.exists():
            with open(default_config_path, 'r', encoding='utf-8') as f:
                config_data = yaml.safe_load(f)
                
            # 修改配置适应测试环境
            config_data['app']['debug'] = True
            config_data['app']['log_level'] = 'WARNING'
            config_data['testing'] = {'skip_llm_tests': True}  # 默认跳过LLM测试
            
            with open(test_config_path, 'w', encoding='utf-8') as f:
                yaml.dump(config_data, f, allow_unicode=True)
    
    yield
    
    # 清理环境变量
    os.environ.pop('ENV', None)

@pytest.fixture
def app():
    """创建测试用的Flask应用"""
    from app.main import create_app
    
    app = create_app()
    app.config['TESTING'] = True
    app.config['DEBUG'] = True
    
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

@pytest.fixture
def mock_env_vars():
    """模拟环境变量"""
    old_vars = {}
    
    def _set_vars(**kwargs):
        for key, value in kwargs.items():
            old_vars[key] = os.environ.get(key)
            os.environ[key] = value
            
        return old_vars
    
    yield _set_vars
    
    # 恢复原始环境变量
    for key, value in old_vars.items():
        if value is None:
            os.environ.pop(key, None)
        else:
            os.environ[key] = value