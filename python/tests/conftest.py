import pytest
import asyncio
import os
import sys
from unittest.mock import Mock, patch

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
def mock_prometheus_service():
    """模拟Prometheus服务"""
    with patch('app.services.prometheus.PrometheusService') as mock:
        mock_instance = Mock()
        mock_instance.is_healthy.return_value = True
        mock_instance.query_range.return_value = None
        mock_instance.query_instant.return_value = [{'value': ['1234567890', '10.5']}]
        mock.return_value = mock_instance
        yield mock_instance

@pytest.fixture
def mock_k8s_service():
    """模拟Kubernetes服务"""
    with patch('app.services.kubernetes.KubernetesService') as mock:
        mock_instance = Mock()
        mock_instance.is_healthy.return_value = True
        mock_instance.get_deployment.return_value = {
            'metadata': {'name': 'test-deployment'},
            'spec': {'replicas': 3},
            'status': {'ready_replicas': 3}
        }
        mock.return_value = mock_instance
        yield mock_instance

@pytest.fixture
def mock_llm_service():
    """模拟LLM服务"""
    with patch('app.services.llm.LLMService') as mock:
        mock_instance = Mock()
        mock_instance.is_healthy.return_value = True
        mock_instance.generate_response.return_value = "测试响应"
        mock.return_value = mock_instance
        yield mock_instance

@pytest.fixture
def mock_prediction_service():
    """模拟预测服务"""
    with patch('app.core.prediction.predictor.PredictionService') as mock:
        mock_instance = Mock()
        mock_instance.is_healthy.return_value = True
        mock_instance.predict.return_value = {
            'instances': 5,
            'current_qps': 100.0,
            'timestamp': '2024-01-01T12:00:00',
            'confidence': 0.85,
            'model_version': '1.0'
        }
        mock.return_value = mock_instance
        yield mock_instance

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