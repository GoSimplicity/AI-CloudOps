"""
MIT License

Copyright (c) 2024 Bamboo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
"""

import unittest
import os
import sys
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
import tempfile
import shutil

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from core.prediction.resource_prediction import create_predictor, TimeSeriesPredictor, MLPredictor
from core.prediction.failure_prediction import create_failure_predictor, SupervisedFailurePredictor, UnsupervisedFailurePredictor

class TestResourcePrediction(unittest.TestCase):
    """资源预测模块测试"""
    
    def setUp(self):
        """测试前准备"""
        # 创建临时目录用于保存模型
        self.test_dir = tempfile.mkdtemp()
        
        # 生成测试数据
        self.time_series_data = self._generate_time_series_data()
        self.ml_data = self._generate_ml_data()
    
    def tearDown(self):
        """测试后清理"""
        # 删除临时目录
        shutil.rmtree(self.test_dir)
    
    def _generate_time_series_data(self, n_samples=100):
        """生成时间序列测试数据"""
        now = datetime.now()
        data = []
        
        for i in range(n_samples):
            timestamp = now - timedelta(hours=n_samples-i)
            # 生成带有季节性和趋势的数据
            value = 50 + 10 * np.sin(i/12) + i/20 + np.random.normal(0, 2)
            
            data.append({
                "timestamp": timestamp,
                "value": value,
                "metric_name": "cpu_usage"
            })
        
        return data
    
    def _generate_ml_data(self, n_samples=100):
        """生成机器学习测试数据"""
        now = datetime.now()
        data = []
        
        # 生成基础时间序列
        base_series = [50 + 10 * np.sin(i/12) + i/20 + np.random.normal(0, 2) for i in range(n_samples+5)]
        
        for i in range(n_samples):
            timestamp = now - timedelta(hours=n_samples-i)
            
            # 使用过去5个点作为特征
            features = {
                f"cpu_t-{j+1}": base_series[i+5-j-1] for j in range(5)
            }
            
            # 添加其他特征
            features["memory_usage"] = 60 + 5 * np.sin(i/10) + np.random.normal(0, 3)
            features["disk_usage"] = 40 + i/50 + np.random.normal(0, 1)
            
            data.append({
                "timestamp": timestamp,
                "target": base_series[i+5],  # 当前值
                "features": features
            })
        
        return data
    
    def test_time_series_predictor_creation(self):
        """测试时间序列预测器创建"""
        predictor = create_predictor(predictor_type="timeseries", model_dir=self.test_dir)
        self.assertIsInstance(predictor, TimeSeriesPredictor)
    
    def test_ml_predictor_creation(self):
        """测试机器学习预测器创建"""
        predictor = create_predictor(predictor_type="ml", model_dir=self.test_dir)
        self.assertIsInstance(predictor, MLPredictor)
    
    def test_time_series_training(self):
        """测试时间序列预测器训练"""
        predictor = create_predictor(predictor_type="timeseries", model_dir=self.test_dir)
        
        # 训练模型
        train_result = predictor.train(self.time_series_data)
        
        # 验证训练结果
        self.assertIsNotNone(train_result)
        self.assertIn("model_type", train_result)
        self.assertIn("metrics", train_result)
    
    def test_time_series_prediction(self):
        """测试时间序列预测"""
        predictor = create_predictor(predictor_type="timeseries", model_dir=self.test_dir)
        
        # 训练模型
        predictor.train(self.time_series_data)
        
        # 预测未来值
        future_steps = 24
        predictions = predictor.predict(self.time_series_data[-30:], future_steps=future_steps)
        
        # 验证预测结果
        self.assertEqual(len(predictions), future_steps)
        for pred in predictions:
            self.assertIn("timestamp", pred)
            self.assertIn("value", pred)
            self.assertIn("metric_name", pred)
    
    def test_ml_training(self):
        """测试机器学习预测器训练"""
        predictor = create_predictor(predictor_type="ml", model_dir=self.test_dir)
        
        # 训练模型
        train_result = predictor.train(self.ml_data)
        
        # 验证训练结果
        self.assertIsNotNone(train_result)
        self.assertIn("model_type", train_result)
        self.assertIn("metrics", train_result)
    
    def test_ml_prediction(self):
        """测试机器学习预测"""
        predictor = create_predictor(predictor_type="ml", model_dir=self.test_dir)
        
        # 训练模型
        predictor.train(self.ml_data)
        
        # 预测未来值
        predictions = predictor.predict(self.ml_data[-10:], future_steps=1)
        
        # 验证预测结果
        self.assertGreater(len(predictions), 0)
        for pred in predictions:
            self.assertIn("timestamp", pred)
            self.assertIn("value", pred)
            self.assertIn("metric_name", pred)
    
    def test_model_save_load(self):
        """测试模型保存和加载"""
        predictor = create_predictor(predictor_type="timeseries", model_dir=self.test_dir)
        
        # 训练并保存模型
        predictor.train(self.time_series_data)
        model_path = predictor.save_model("test_model")
        
        # 创建新的预测器并加载模型
        new_predictor = create_predictor(predictor_type="timeseries", model_dir=self.test_dir)
        load_success = new_predictor.load_model("test_model")
        
        # 验证加载成功
        self.assertTrue(load_success)
        
        # 使用加载的模型进行预测
        predictions = new_predictor.predict(self.time_series_data[-30:], future_steps=5)
        
        # 验证预测结果
        self.assertEqual(len(predictions), 5)


class TestFailurePrediction(unittest.TestCase):
    """故障预测模块测试"""
    
    def setUp(self):
        """测试前准备"""
        # 创建临时目录用于保存模型
        self.test_dir = tempfile.mkdtemp()
        
        # 生成测试数据
        self.metrics, self.labels = self._generate_test_data()
        self.logs = self._generate_test_logs()
        self.traces = self._generate_test_traces()
    
    def tearDown(self):
        """测试后清理"""
        # 删除临时目录
        shutil.rmtree(self.test_dir)
    
    def _generate_test_data(self, n_samples=200):
        """生成测试指标数据和标签"""
        now = datetime.now()
        metrics = []
        labels = []
        
        for i in range(n_samples):
            timestamp = now - timedelta(minutes=n_samples-i)
            
            # 基础指标值
            cpu_usage = 50 + 10 * np.sin(i/20) + np.random.normal(0, 5)
            memory_usage = 60 + 5 * np.sin(i/15) + np.random.normal(0, 3)
            disk_usage = 40 + i/100 + np.random.normal(0, 2)
            network_traffic = 200 + 50 * np.sin(i/30) + np.random.normal(0, 20)
            response_latency = 50 + np.random.normal(0, 5)
            
            # 每10个样本生成一个异常点
            is_anomaly = False
            if i % 20 == 0:
                is_anomaly = True
                anomaly_type = i % 5
                
                if anomaly_type == 0:
                    cpu_usage += 30  # CPU突增
                elif anomaly_type == 1:
                    memory_usage += 25  # 内存突增
                elif anomaly_type == 2:
                    disk_usage += 40  # 磁盘使用突增
                elif anomaly_type == 3:
                    network_traffic += 150  # 网络流量突增
                elif anomaly_type == 4:
                    response_latency += 100  # 响应延迟突增
            
            # 确保值在合理范围内
            cpu_usage = max(0, min(100, cpu_usage))
            memory_usage = max(0, min(100, memory_usage))
            disk_usage = max(0, min(100, disk_usage))
            network_traffic = max(0, network_traffic)
            response_latency = max(0, response_latency)
            
            # 创建指标数据点
            metric = {
                "timestamp": timestamp,
                "cpu_usage": cpu_usage,
                "memory_usage": memory_usage,
                "disk_usage": disk_usage,
                "network_traffic": network_traffic,
                "response_latency": response_latency,
                "error_count": 1 if is_anomaly and np.random.random() < 0.3 else 0
            }
            
            metrics.append(metric)
            labels.append(1 if is_anomaly else 0)
        
        return metrics, labels
    
    def _generate_test_logs(self, n_samples=50):
        """生成测试日志数据"""
        logs = []
        
        error_messages = [
            "Exception in thread main java.lang.NullPointerException",
            "Error: Connection refused",
            "Failed to connect to database",
            "Out of memory error",
            "Timeout waiting for response"
        ]
        
        warning_messages = [
            "Warning: High CPU usage detected",
            "Warning: Memory usage above threshold",
            "Warning: Slow database query",
            "Warning: Network latency increased",
            "Warning: Disk space running low"
        ]
        
        info_messages = [
            "Service started successfully",
            "Request processed in 50ms",
            "Database connection established",
            "User authentication successful",
            "Cache refreshed"
        ]
        
        for i in range(n_samples):
            if i % 10 == 0:
                # 生成错误日志
                log = f"[ERROR] {datetime.now().isoformat()} {np.random.choice(error_messages)}"
            elif i % 5 == 0:
                # 生成警告日志
                log = f"[WARN] {datetime.now().isoformat()} {np.random.choice(warning_messages)}"
            else:
                # 生成信息日志
                log = f"[INFO] {datetime.now().isoformat()} {np.random.choice(info_messages)}"
            
            logs.append(log)
        
        return logs
    
    def _generate_test_traces(self, n_samples=30):
        """生成测试链路追踪数据"""
        traces = []
        
        services = ["web-server", "api-gateway", "auth-service", "database", "cache"]
        
        for i in range(n_samples):
            trace_id = f"trace-{i}"
            
            # 创建主调用
            main_span = {
                "trace_id": trace_id,
                "span_id": f"span-{i}-0",
                "service": "web-server",
                "operation": "GET /api/v1/data",
                "start_time": datetime.now() - timedelta(minutes=i),
                "duration": 100 + np.random.normal(0, 20),
                "status": "error" if i % 10 == 0 else "success"
            }
            
            traces.append(main_span)
            
            # 添加子调用
            for j in range(1, np.random.randint(2, 5)):
                child_span = {
                    "trace_id": trace_id,
                    "span_id": f"span-{i}-{j}",
                    "parent_id": f"span-{i}-0",
                    "service": np.random.choice(services),
                    "operation": f"internal-operation-{j}",
                    "start_time": main_span["start_time"] + timedelta(milliseconds=10*j),
                    "duration": 50 + np.random.normal(0, 10),
                    "status": "error" if i % 10 == 0 and j == 1 else "success"
                }
                
                traces.append(child_span)
        
        return traces
    
    def test_supervised_predictor_creation(self):
        """测试监督学习故障预测器创建"""
        predictor = create_failure_predictor(predictor_type="supervised", model_dir=self.test_dir)
        self.assertIsInstance(predictor, SupervisedFailurePredictor)
    
    def test_unsupervised_predictor_creation(self):
        """测试无监督学习故障预测器创建"""
        predictor = create_failure_predictor(predictor_type="unsupervised", model_dir=self.test_dir)
        self.assertIsInstance(predictor, UnsupervisedFailurePredictor)
    
    def test_supervised_training(self):
        """测试监督学习故障预测器训练"""
        predictor = create_failure_predictor(predictor_type="supervised", model_dir=self.test_dir)
        
        # 训练模型
        train_result = predictor.train(self.metrics, self.labels, logs=self.logs, traces=self.traces)
        
        # 验证训练结果
        self.assertIsNotNone(train_result)
        self.assertIn("accuracy", train_result)
        self.assertIn("precision", train_result)
        self.assertIn("recall", train_result)
        self.assertIn("f1", train_result)
    
    def test_supervised_prediction(self):
        """测试监督学习故障预测"""
        predictor = create_failure_predictor(predictor_type="supervised", model_dir=self.test_dir)
        
        # 训练模型
        predictor.train(self.metrics, self.labels, logs=self.logs, traces=self.traces)
        
        # 预测故障
        test_metrics = self.metrics[:20]  # 使用部分数据进行测试
        predictions = predictor.predict(test_metrics, logs=self.logs[:10], traces=self.traces[:5])
        
        # 验证预测结果
        self.assertEqual(len(predictions), len(test_metrics))
        for pred in predictions:
            self.assertIn("failure_predicted", pred)
            self.assertIn("probability", pred)
            self.assertIn("timestamp", pred)
            
            # 如果预测有故障，应该有相关信息
            if pred["failure_predicted"]:
                self.assertIn("failure_type", pred)
                self.assertIn("expected_time", pred)
                self.assertIn("affected_components", pred)
                self.assertIn("prevention_actions", pred)
    
    def test_unsupervised_training(self):
        """测试无监督学习故障预测器训练"""
        predictor = create_failure_predictor(predictor_type="unsupervised", model_dir=self.test_dir)
        
        # 训练模型
        train_result = predictor.train(self.metrics, logs=self.logs, traces=self.traces)
        
        # 验证训练结果
        self.assertIsNotNone(train_result)
        self.assertIn("model_type", train_result)
    
    def test_unsupervised_prediction(self):
        """测试无监督学习故障预测"""
        predictor = create_failure_predictor(predictor_type="unsupervised", model_dir=self.test_dir)
        
        # 训练模型
        predictor.train(self.metrics, logs=self.logs, traces=self.traces)
        
        # 预测故障
        test_metrics = self.metrics[:20]  # 使用部分数据进行测试
        predictions = predictor.predict(test_metrics, logs=self.logs[:10], traces=self.traces[:5])
        
        # 验证预测结果
        self.assertEqual(len(predictions), len(test_metrics))
        for pred in predictions:
            self.assertIn("failure_predicted", pred)
            self.assertIn("probability", pred)
            self.assertIn("timestamp", pred)
            self.assertIn("anomaly_score", pred)
            
            # 如果预测有故障，应该有相关信息
            if pred["failure_predicted"]:
                self.assertIn("failure_type", pred)
                self.assertIn("expected_time", pred)
                self.assertIn("affected_components", pred)
                self.assertIn("prevention_actions", pred)
    
    def test_failure_model_save_load(self):
        """测试故障预测模型保存和加载"""
        predictor = create_failure_predictor(predictor_type="supervised", model_dir=self.test_dir)
        
        # 训练并保存模型
        predictor.train(self.metrics, self.labels, logs=self.logs, traces=self.traces)
        model_path = predictor.save_model("test_failure_model")
        
        # 创建新的预测器并加载模型
        new_predictor = create_failure_predictor(predictor_type="supervised", model_dir=self.test_dir)
        load_success = new_predictor.load_model("test_failure_model")
        
        # 验证加载成功
        self.assertTrue(load_success)
        
        # 使用加载的模型进行预测
        predictions = new_predictor.predict(self.metrics[:5])
        
        # 验证预测结果
        self.assertEqual(len(predictions), 5)


if __name__ == "__main__":
    unittest.main()