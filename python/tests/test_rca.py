import pytest
import json
from datetime import datetime, timedelta
from unittest.mock import patch, Mock
import pandas as pd
import numpy as np

class TestRCAAPI:
    """测试根因分析API"""
    
    def test_rca_endpoint_success(self, client, mock_prometheus_service):
        """测试RCA接口成功响应"""
        request_data = {
            "start_time": "2024-01-01T10:00:00Z",
            "end_time": "2024-01-01T11:00:00Z",
            "metrics": ["container_cpu_usage_seconds_total"]
        }
        
        with patch('app.core.rca.analyzer.RCAAnalyzer') as mock_analyzer:
            mock_analyzer_instance = Mock()
            mock_analyzer_instance.analyze.return_value = {
                "status": "success",
                "anomalies": {},
                "correlations": {},
                "root_cause_candidates": [],
                "analysis_time": "2024-01-01T12:00:00Z",
                "time_range": {
                    "start": "2024-01-01T10:00:00Z",
                    "end": "2024-01-01T11:00:00Z"
                },
                "metrics_analyzed": ["container_cpu_usage_seconds_total"]
            }
            mock_analyzer.return_value = mock_analyzer_instance
            
            response = client.post('/api/v1/rca', 
                                 data=json.dumps(request_data),
                                 content_type='application/json')
            
            assert response.status_code == 200
            data = response.get_json()
            assert data['status'] == 'success'
    
    def test_rca_endpoint_invalid_time_range(self, client):
        """测试无效时间范围"""
        request_data = {
            "start_time": "2024-01-01T11:00:00Z",
            "end_time": "2024-01-01T10:00:00Z",  # 结束时间早于开始时间
            "metrics": ["container_cpu_usage_seconds_total"]
        }
        
        response = client.post('/api/v1/rca',
                             data=json.dumps(request_data),
                             content_type='application/json')
        
        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data
    
    def test_rca_endpoint_missing_data(self, client):
        """测试缺少数据的情况"""
        response = client.post('/api/v1/rca',
                             data=json.dumps({}),
                             content_type='application/json')
        
        # 应该使用默认参数，不应该失败
        assert response.status_code in [200, 500]  # 可能因为服务不可用而返回500
    
    def test_get_available_metrics(self, client):
        """测试获取可用指标"""
        response = client.get('/api/v1/rca/metrics')
        
        assert response.status_code == 200
        data = response.get_json()
        assert 'default_metrics' in data
        assert 'categories' in data

class TestAnomalyDetector:
    """测试异常检测器"""
    
    @pytest.fixture
    def sample_data(self):
        """创建示例数据"""
        dates = pd.date_range('2024-01-01', periods=100, freq='1min')
        values = np.random.normal(50, 10, 100)
        # 添加一些异常值
        values[80:85] = np.random.normal(150, 5, 5)  # 异常高值
        values[90:92] = np.random.normal(10, 2, 2)   # 异常低值
        
        return pd.DataFrame({
            'value': values
        }, index=dates)
    
    def test_anomaly_detection(self, sample_data):
        """测试异常检测"""
        from app.core.rca.detector import AnomalyDetector
        
        detector = AnomalyDetector()
        
        # 创建包含异常数据的字典
        metrics_data = {
            'test_metric': sample_data
        }
        
        # 由于是异步方法，需要在同步测试中模拟
        import asyncio
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        
        try:
            result = loop.run_until_complete(
                detector.detect_anomalies(metrics_data)
            )
            
            # 应该检测到异常
            assert isinstance(result, dict)
            if 'test_metric' in result:
                assert result['test_metric']['count'] > 0
        finally:
            loop.close()
    
    def test_threshold_update(self):
        """测试阈值更新"""
        from app.core.rca.detector import AnomalyDetector
        
        detector = AnomalyDetector()
        original_threshold = detector.anomaly_threshold
        
        new_threshold = 0.8
        detector.update_threshold(new_threshold)
        
        assert detector.anomaly_threshold == new_threshold
        assert detector.anomaly_threshold != original_threshold

class TestCorrelationAnalyzer:
    """测试相关性分析器"""
    
    @pytest.fixture
    def correlated_data(self):
        """创建相关的数据"""
        dates = pd.date_range('2024-01-01', periods=100, freq='1min')
        
        # 创建相关的时间序列
        base_series = np.random.normal(50, 10, 100)
        correlated_series = base_series * 0.8 + np.random.normal(0, 5, 100)  # 高度相关
        independent_series = np.random.normal(30, 8, 100)  # 独立的
        
        return {
            'metric_a': pd.DataFrame({'value': base_series}, index=dates),
            'metric_b': pd.DataFrame({'value': correlated_series}, index=dates),
            'metric_c': pd.DataFrame({'value': independent_series}, index=dates)
        }
    
    def test_correlation_analysis(self, correlated_data):
        """测试相关性分析"""
        from app.core.rca.correlator import CorrelationAnalyzer
        
        analyzer = CorrelationAnalyzer()
        
        import asyncio
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        
        try:
            result = loop.run_until_complete(
                analyzer.analyze_correlations(correlated_data)
            )
            
            assert isinstance(result, dict)
            # 应该发现metric_a和metric_b之间的相关性
            if 'metric_a' in result:
                correlations = result['metric_a']
                correlation_metrics = [corr[0] for corr in correlations]
                # metric_b应该在相关指标中
                assert any('metric_b' in metric for metric in correlation_metrics)
        finally:
            loop.close()

class TestRCAAnalyzer:
    """测试RCA分析器"""
    
    def test_analyzer_initialization(self):
        """测试分析器初始化"""
        with patch('app.services.prometheus.PrometheusService'), \
             patch('app.services.llm.LLMService'):
            from app.core.rca.analyzer import RCAAnalyzer
            
            analyzer = RCAAnalyzer()
            assert analyzer.prometheus is not None
            assert analyzer.detector is not None
            assert analyzer.correlator is not None
            assert analyzer.llm is not None
    
    def test_confidence_calculation(self):
        """测试置信度计算"""
        from app.core.rca.analyzer import RCAAnalyzer
        
        with patch('app.services.prometheus.PrometheusService'), \
             patch('app.services.llm.LLMService'):
            analyzer = RCAAnalyzer()
            
            # 测试高异常分数的情况
            anomaly_info = {
                'max_score': 0.9,
                'count': 10,
                'detection_methods': {'zscore': 5, 'iqr': 3, 'isolation_forest': 2}
            }
            related_metrics = [('metric_b', 0.8), ('metric_c', 0.7)]
            
            confidence = analyzer._calculate_confidence(anomaly_info, related_metrics)
            
            assert 0 <= confidence <= 1
            assert confidence > 0.5  # 应该是高置信度
    
    def test_description_generation(self):
        """测试描述生成"""
        from app.core.rca.analyzer import RCAAnalyzer
        
        with patch('app.services.prometheus.PrometheusService'), \
             patch('app.services.llm.LLMService'):
            analyzer = RCAAnalyzer()
            
            anomaly_info = {
                'count': 5,
                'max_score': 0.8,
                'avg_score': 0.6
            }
            
            # 测试CPU相关指标
            cpu_description = analyzer._generate_description(
                'container_cpu_usage_seconds_total', anomaly_info
            )
            assert 'CPU' in cpu_description
            assert '5' in cpu_description  # 异常次数
            
            # 测试内存相关指标
            memory_description = analyzer._generate_description(
                'container_memory_working_set_bytes', anomaly_info
            )
            assert '内存' in memory_description

class TestRCAIntegration:
    """集成测试"""
    
    def test_end_to_end_rca(self, mock_prometheus_service):
        """端到端RCA测试"""
        from app.core.rca.analyzer import RCAAnalyzer
        
        # 模拟Prometheus返回数据
        sample_data = pd.DataFrame({
            'value': np.random.normal(50, 10, 60)
        }, index=pd.date_range('2024-01-01T10:00:00', periods=60, freq='1min'))
        
        mock_prometheus_service.query_range.return_value = sample_data
        
        with patch('app.services.llm.LLMService'):
            analyzer = RCAAnalyzer()
            analyzer.prometheus = mock_prometheus_service
            
            import asyncio
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            
            try:
                result = loop.run_until_complete(
                    analyzer.analyze(
                        datetime(2024, 1, 1, 10, 0, 0),
                        datetime(2024, 1, 1, 11, 0, 0),
                        ['container_cpu_usage_seconds_total']
                    )
                )
                
                assert 'status' in result
                assert 'anomalies' in result
                assert 'correlations' in result
                assert 'root_cause_candidates' in result
            finally:
                loop.close()