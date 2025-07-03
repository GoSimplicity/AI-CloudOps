import pytest
import json
from datetime import datetime
from unittest.mock import patch, Mock
import numpy as np
import pandas as pd

class TestPredictionAPI:
    """测试预测API"""
    
    def test_predict_endpoint_get(self, client, mock_prediction_service):
        """测试GET预测接口"""
        response = client.get('/api/v1/predict')
        
        assert response.status_code == 200
        data = response.get_json()
        assert 'instances' in data
        assert 'current_qps' in data
        assert 'timestamp' in data
    
    def test_predict_endpoint_post(self, client, mock_prediction_service):
        """测试POST预测接口"""
        request_data = {
            "current_qps": 150.5,
            "timestamp": "2024-01-01T12:00:00Z"
        }
        
        response = client.post('/api/v1/predict',
                             data=json.dumps(request_data),
                             content_type='application/json')
        
        assert response.status_code == 200
        data = response.get_json()
        assert 1 <= data['instances'] <= 20  # 实例数应该在有效范围内
        assert data['current_qps'] == 150.5
    
    def test_predict_endpoint_invalid_qps(self, client):
        """测试无效QPS参数"""
        request_data = {
            "current_qps": -10.0  # 负数QPS
        }
        
        response = client.post('/api/v1/predict',
                             data=json.dumps(request_data),
                             content_type='application/json')
        
        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data
    
    def test_predict_health(self, client, mock_prediction_service):
        """测试预测服务健康检查"""
        response = client.get('/api/v1/predict/health')
        
        assert response.status_code == 200
        data = response.get_json()
        assert 'status' in data
        assert 'healthy' in data

class TestPredictionService:
    """测试预测服务"""
    
    @pytest.fixture
    def mock_model_loader(self):
        """模拟模型加载器"""
        with patch('app.core.prediction.model_loader.ModelLoader') as mock:
            mock_instance = Mock()
            mock_instance.load_models.return_value = True
            mock_instance.validate_model.return_value = True
            mock_instance.is_model_loaded.return_value = True
            mock_instance.model = Mock()
            mock_instance.scaler = Mock()
            mock_instance.model_metadata = {"version": "1.0"}
            
            # 模拟预测和标准化
            mock_instance.model.predict.return_value = np.array([5.2])
            mock_instance.scaler.transform.return_value = np.array([[0.5, 0.3, 0.8]])
            
            mock.return_value = mock_instance
            yield mock_instance
    
    def test_prediction_service_initialization(self, mock_model_loader, mock_prometheus_service):
        """测试预测服务初始化"""
        from app.core.prediction.predictor import PredictionService
        
        with patch('app.services.prometheus.PrometheusService', return_value=mock_prometheus_service):
            service = PredictionService()
            
            assert service.model_loaded == True
            assert service.scaler_loaded == True
    
    def test_prediction_with_custom_qps(self, mock_model_loader, mock_prometheus_service):
        """测试自定义QPS预测"""
        from app.core.prediction.predictor import PredictionService
        
        with patch('app.services.prometheus.PrometheusService', return_value=mock_prometheus_service):
            service = PredictionService()
            service.model_loader = mock_model_loader
            
            import asyncio
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            
            try:
                result = loop.run_until_complete(
                    service.predict(current_qps=200.0)
                )
                
                assert result is not None
                assert 'instances' in result
                assert 'current_qps' in result
                assert result['current_qps'] == 200.0
                assert 1 <= result['instances'] <= 20  # 在配置的范围内
            finally:
                loop.close()
    
    def test_prediction_with_prometheus_qps(self, mock_model_loader, mock_prometheus_service):
        """测试从Prometheus获取QPS进行预测"""
        from app.core.prediction.predictor import PredictionService
        
        # 配置Prometheus返回数据
        # 创建一个异步方法
        async def mock_query(*args, **kwargs):
            return [{'value': ['1234567890', '75.5']}]
        
        mock_prometheus_service.query_instant = mock_query
        
        with patch('app.services.prometheus.PrometheusService', return_value=mock_prometheus_service):
            service = PredictionService()
            service.prometheus = mock_prometheus_service
            service.model_loader = mock_model_loader
            
            import asyncio
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            
            try:
                result = loop.run_until_complete(
                    service.predict()  # 不提供QPS，应该从Prometheus获取
                )
                
                assert result is not None
                assert result['current_qps'] == 75.5
            finally:
                loop.close()
    
    def test_confidence_calculation(self, mock_model_loader, mock_prometheus_service):
        """测试置信度计算"""
        from app.core.prediction.predictor import PredictionService
        from app.utils.time_utils import TimeUtils
        
        with patch('app.services.prometheus.PrometheusService', return_value=mock_prometheus_service):
            service = PredictionService()
            service.model_loader = mock_model_loader
            
            # 测试不同QPS的置信度
            time_features = TimeUtils.extract_time_features(datetime.now())
            
            # 低QPS应该有高置信度
            low_qps_confidence = service._calculate_confidence(50.0, time_features, 3.0)
            assert 0.6 <= low_qps_confidence <= 1.0
            
            # 极高QPS应该有较低置信度
            high_qps_confidence = service._calculate_confidence(2000.0, time_features, 15.0)
            assert 0.3 <= high_qps_confidence <= 0.8
    
    def test_trend_prediction(self, mock_model_loader, mock_prometheus_service):
        """测试趋势预测"""
        from app.core.prediction.predictor import PredictionService
        
        with patch('app.services.prometheus.PrometheusService', return_value=mock_prometheus_service):
            service = PredictionService()
            service.model_loader = mock_model_loader
            
            import asyncio
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            
            try:
                result = loop.run_until_complete(
                    service.predict_trend(hours_ahead=6, current_qps=100.0)
                )
                
                assert result is not None
                assert 'trend_predictions' in result
                assert 'summary' in result
                assert len(result['trend_predictions']) == 6
                
                # 检查摘要统计
                summary = result['summary']
                assert 'max_instances' in summary
                assert 'min_instances' in summary
                assert 'avg_instances' in summary
            finally:
                loop.close()

class TestModelLoader:
    """测试模型加载器"""
    
    def test_model_validation(self):
        """测试模型验证"""
        from app.core.prediction.model_loader import ModelLoader
        
        loader = ModelLoader()
        
        # 模拟加载的模型和标准化器
        with patch('joblib.load') as mock_load:
            mock_model = Mock()
            mock_scaler = Mock()
            
            mock_model.predict.return_value = np.array([5.0])
            mock_scaler.transform.return_value = np.array([[0.1, 0.2, 0.3]])
            
            mock_load.side_effect = [mock_model, mock_scaler]
            
            # 模拟文件存在
            with patch('os.path.exists', return_value=True):
                success = loader.load_models()
                assert success == True
                
                # 测试验证
                validation_result = loader.validate_model()
                assert validation_result == True
    
    def test_model_loading_failure(self):
        """测试模型加载失败"""
        from app.core.prediction.model_loader import ModelLoader
        
        loader = ModelLoader()
        
        # 模拟文件不存在
        with patch('os.path.exists', return_value=False):
            success = loader.load_models()
            assert success == False
            assert loader.is_model_loaded() == False

class TestTimeUtils:
    """测试时间工具"""
    
    def test_time_feature_extraction(self):
        """测试时间特征提取"""
        from app.utils.time_utils import TimeUtils
        
        # 测试特定时间
        test_time = datetime(2024, 1, 1, 14, 30, 0)  # 周一下午2:30
        features = TimeUtils.extract_time_features(test_time)
        
        assert 'sin_time' in features
        assert 'cos_time' in features
        assert 'hour' in features
        assert 'day_of_week' in features
        assert 'is_business_hour' in features
        assert 'is_weekend' in features
        
        assert features['hour'] == 14
        assert features['day_of_week'] == 0  # 周一
        assert features['is_business_hour'] == True  # 工作时间
        assert features['is_weekend'] == False  # 工作日
        
        # 验证三角函数特征在合理范围内
        assert -1 <= features['sin_time'] <= 1
        assert -1 <= features['cos_time'] <= 1
    
    def test_time_range_validation(self):
        """测试时间范围验证"""
        from app.utils.time_utils import TimeUtils
        from datetime import timedelta
        
        now = datetime.utcnow()
        past = now - timedelta(hours=1)
        future = now + timedelta(hours=1)
        
        # 有效的时间范围
        assert TimeUtils.validate_time_range(past, now) == True
        
        # 无效的时间范围（开始时间晚于结束时间）
        assert TimeUtils.validate_time_range(now, past) == False
        
        # 无效的时间范围（未来时间）
        assert TimeUtils.validate_time_range(now, future) == False

class TestPredictionIntegration:
    """预测功能集成测试"""
    
    def test_full_prediction_workflow(self, client):
        """完整预测工作流测试"""
        with patch('app.core.prediction.predictor.PredictionService') as mock_service:
            mock_instance = Mock()
            mock_instance.is_healthy.return_value = True
            mock_instance.predict.return_value = {
                'instances': 8,
                'current_qps': 250.0,
                'timestamp': '2024-01-01T12:00:00Z',
                'confidence': 0.78,
                'model_version': '1.0',
                'features': {
                    'qps': 250.0,
                    'sin_time': 0.5,
                    'cos_time': 0.866,
                    'hour': 12,
                    'is_business_hour': True,
                    'is_weekend': False
                }
            }
            mock_service.return_value = mock_instance
            
            # 测试带有特征信息的预测请求
            request_data = {
                "current_qps": 250.0,
                "include_confidence": True
            }
            
            response = client.post('/api/v1/predict',
                                 data=json.dumps(request_data),
                                 content_type='application/json')
            
            assert response.status_code == 200
            data = response.get_json()
            
                        # 验证返回的所有字段
            expected_fields = ['instances', 'current_qps', 'timestamp', 'confidence', 'model_version']
            for field in expected_fields:
                assert field in data
                
            assert 1 <= data['instances'] <= 20  # 实例数应该在有效范围内
            assert data['current_qps'] == 250.0
            assert 0 <= data['confidence'] <= 1.0  # 信心值应该在0到1之间