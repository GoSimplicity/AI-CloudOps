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

from core.prediction.model_optimization import create_optimizer, TimeSeriesModelOptimizer, MLModelOptimizer, FailurePredictionModelOptimizer

class TestModelOptimization(unittest.TestCase):
    """模型优化器测试"""
    
    def setUp(self):
        """测试前准备"""
        # 创建临时目录用于保存模型
        self.test_dir = tempfile.mkdtemp()
        
        # 生成测试数据
        self.time_series_data = self._generate_time_series_data()
        self.ml_data = self._generate_ml_data()
        self.failure_data, self.failure_labels = self._generate_failure_data()
    
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
    
    def _generate_failure_data(self, n_samples=200):
        """生成故障预测测试数据"""
        now = datetime.now()
        data = []
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
            
            data.append(metric)
            labels.append(1 if is_anomaly else 0)
        
        return data, labels
    
    def test_optimizer_creation(self):
        """测试优化器创建"""
        # 创建时间序列优化器
        ts_optimizer = create_optimizer(optimizer_type="timeseries", model_dir=self.test_dir)
        self.assertIsInstance(ts_optimizer, TimeSeriesModelOptimizer)
        
        # 创建机器学习优化器
        ml_optimizer = create_optimizer(optimizer_type="ml", model_dir=self.test_dir)
        self.assertIsInstance(ml_optimizer, MLModelOptimizer)
        
        # 创建故障预测优化器
        failure_optimizer = create_optimizer(optimizer_type="failure", model_dir=self.test_dir)
        self.assertIsInstance(failure_optimizer, FailurePredictionModelOptimizer)
        
        # 测试无效的优化器类型
        with self.assertRaises(ValueError):
            create_optimizer(optimizer_type="invalid_type")
    
    def test_time_series_optimization(self):
        """测试时间序列优化"""
        # 创建时间序列优化器
        optimizer = create_optimizer(optimizer_type="timeseries", model_dir=self.test_dir)
        
        # 优化模型（使用较少的试验次数加速测试）
        result = optimizer.optimize(
            self.time_series_data, 
            model_types=["arima"],  # 仅使用ARIMA模型加速测试
            metric_name="cpu_usage"
        )
        
        # 验证优化结果
        self.assertIsNotNone(result)
        self.assertIn("model_type", result)
        self.assertIn("model_path", result)
        self.assertIn("best_params", result)
        self.assertIn("best_score", result)
        
        # 验证模型文件存在
        self.assertTrue(os.path.exists(result["model_path"]))
    
    def test_ml_optimization(self):
        """测试机器学习优化"""
        # 创建机器学习优化器
        optimizer = create_optimizer(optimizer_type="ml", model_dir=self.test_dir)
        
        # 优化模型（使用较少的模型类型加速测试）
        result = optimizer.optimize(
            self.ml_data, 
            model_types=["linear"],  # 仅使用线性回归加速测试
            metric_name="cpu_usage"
        )
        
        # 验证优化结果
        self.assertIsNotNone(result)
        self.assertIn("model_type", result)
        self.assertIn("model_path", result)
        self.assertIn("best_params", result)
        self.assertIn("best_score", result)
        
        # 验证模型文件存在
        self.assertTrue(os.path.exists(result["model_path"]))
    
    def test_supervised_failure_optimization(self):
        """测试监督学习故障预测优化"""
        # 创建故障预测优化器
        optimizer = create_optimizer(optimizer_type="failure", model_dir=self.test_dir)
        
        # 优化模型（使用较少的模型类型加速测试）
        result = optimizer.optimize(
            self.failure_data, 
            labels=self.failure_labels,
            predictor_type="supervised",
            model_types=["logistic"],  # 仅使用逻辑回归加速测试
        )
        
        # 验证优化结果
        self.assertIsNotNone(result)
        self.assertIn("model_type", result)
        self.assertIn("model_path", result)
        self.assertIn("best_params", result)
        self.assertIn("best_score", result)
        self.assertIn("metrics", result)
        
        # 验证模型文件存在
        self.assertTrue(os.path.exists(result["model_path"]))
    
    def test_unsupervised_failure_optimization(self):
        """测试无监督学习故障预测优化"""
        # 创建故障预测优化器
        optimizer = create_optimizer(optimizer_type="failure", model_dir=self.test_dir)
        
        # 优化模型（使用较少的模型类型加速测试）
        result = optimizer.optimize(
            self.failure_data,
            predictor_type="unsupervised",
            model_types=["iforest"],  # 仅使用隔离森林加速测试
        )
        
        # 验证优化结果
        self.assertIsNotNone(result)
        self.assertIn("model_type", result)
        self.assertIn("model_path", result)
        self.assertIn("best_params", result)
        self.assertIn("best_score", result)
        
        # 验证模型文件存在
        self.assertTrue(os.path.exists(result["model_path"]))


if __name__ == "__main__":
    unittest.main()