#!/usr/bin/env python3
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

import os
import sys
import argparse
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
import random

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from core.prediction.resource_prediction import create_predictor
from core.prediction.failure_prediction import create_failure_predictor
from utils.logger import get_logger

logger = get_logger("setup_prediction")


def generate_sample_metrics(num_samples=100, with_anomalies=True):
  """生成样本指标数据用于测试"""
  now = datetime.now()
  metrics = []

  # 正常模式的基准值
  cpu_base = 50.0
  memory_base = 60.0
  disk_base = 40.0
  network_base = 200.0
  latency_base = 50.0

  # 生成样本数据
  for i in range(num_samples):
    timestamp = now - timedelta(minutes=num_samples - i)

    # 添加一些随机波动
    cpu_noise = np.random.normal(0, 5)
    memory_noise = np.random.normal(0, 3)
    disk_noise = np.random.normal(0, 2)
    network_noise = np.random.normal(0, 20)
    latency_noise = np.random.normal(0, 5)

    # 基本指标值
    cpu_usage = cpu_base + cpu_noise
    memory_usage = memory_base + memory_noise
    disk_usage = disk_base + disk_noise
    network_traffic = network_base + network_noise
    response_latency = latency_base + latency_noise

    # 添加一些异常值
    is_anomaly = False
    if with_anomalies and random.random() < 0.1:  # 10%的概率生成异常
      is_anomaly = True
      anomaly_type = random.choice(['cpu', 'memory', 'disk', 'network', 'latency'])

      if anomaly_type == 'cpu':
        cpu_usage += 30.0  # CPU突增
      elif anomaly_type == 'memory':
        memory_usage += 25.0  # 内存突增
      elif anomaly_type == 'disk':
        disk_usage += 40.0  # 磁盘使用突增
      elif anomaly_type == 'network':
        network_traffic += 150.0  # 网络流量突增
      elif anomaly_type == 'latency':
        response_latency += 100.0  # 响应延迟突增

    # 确保值在合理范围内
    cpu_usage = max(0, min(100, int(cpu_usage)))
    memory_usage = max(0, min(100, int(memory_usage)))
    disk_usage = max(0, min(100, int(disk_usage)))
    network_traffic = max(0, int(network_traffic))
    response_latency = max(0, int(response_latency))

    # 创建指标数据点
    metric = {
      "timestamp": timestamp,
      "cpu_usage": cpu_usage,
      "memory_usage": memory_usage,
      "disk_usage": disk_usage,
      "network_traffic": network_traffic,
      "response_latency": response_latency,
      "error_count": 1 if is_anomaly and random.random() < 0.3 else 0,
      "is_anomaly": 1 if is_anomaly else 0  # 用于监督学习的标签
    }

    metrics.append(metric)

  return metrics


def test_resource_prediction():
  """测试资源预测功能"""
  logger.info("开始测试资源预测功能...")

  # 生成测试数据
  metrics = generate_sample_metrics(num_samples=200, with_anomalies=False)

  # 创建时间序列预测器
  predictor = create_predictor(predictor_type="timeseries", model_type="arima")

  # 准备训练数据
  train_data = []
  for metric in metrics:
    train_data.append({
      "timestamp": metric["timestamp"],
      "value": metric["cpu_usage"],
      "metric_name": "cpu_usage"
    })

  # 训练模型
  logger.info("训练CPU使用率预测模型...")
  predictor.train(train_data)

  # 保存模型
  predictor.save_model("cpu_usage_predictor")

  # 预测未来值
  future_steps = 24  # 预测未来24个时间点
  predictions = predictor.predict(train_data[-30:], future_steps=future_steps)

  logger.info(f"预测结果: {predictions[:5]}...")

  # 创建ML预测器
  ml_predictor = create_predictor(predictor_type="ml", model_type="rf")

  # 准备ML训练数据
  ml_train_data = []
  for i in range(len(metrics) - 6):
    # 使用过去5个时间点预测下一个时间点
    features = {
      "timestamp": metrics[i + 5]["timestamp"],
      "target": metrics[i + 5]["cpu_usage"],
      "features": {
        "cpu_t-1": metrics[i + 4]["cpu_usage"],
        "cpu_t-2": metrics[i + 3]["cpu_usage"],
        "cpu_t-3": metrics[i + 2]["cpu_usage"],
        "cpu_t-4": metrics[i + 1]["cpu_usage"],
        "cpu_t-5": metrics[i]["cpu_usage"],
        "memory_t-1": metrics[i + 4]["memory_usage"],
        "disk_t-1": metrics[i + 4]["disk_usage"]
      }
    }
    ml_train_data.append(features)

  # 训练ML模型
  logger.info("训练ML资源预测模型...")
  ml_predictor.train(ml_train_data)

  # 保存ML模型
  ml_predictor.save_model("cpu_usage_ml_predictor")

  # 预测未来值
  ml_predictions = ml_predictor.predict(ml_train_data[-10:], future_steps=1)

  logger.info(f"ML预测结果: {ml_predictions}")

  logger.info("资源预测测试完成")


def test_failure_prediction():
  """测试故障预测功能"""
  logger.info("开始测试故障预测功能...")

  # 生成测试数据
  metrics = generate_sample_metrics(num_samples=300, with_anomalies=True)
  
  # 将指标数据转换为DataFrame
  df = pd.DataFrame(metrics)
  
  # 提取标签
  labels = df["is_anomaly"].tolist()

  # 创建监督学习故障预测器
  predictor = create_failure_predictor(predictor_type="supervised", model_type="rf")

  # 训练模型 - 使用DataFrame的副本避免性能警告
  logger.info("训练监督学习故障预测模型...")
  # 使用to_dict('records')前先创建副本以避免DataFrame碎片化警告
  df_copy = df.copy()
  train_result = predictor.train(df_copy.to_dict('records'), labels)

  logger.info(f"训练结果: {train_result}")

  # 保存模型
  predictor.save_model("supervised_failure_predictor")

  # 生成测试数据
  test_metrics = generate_sample_metrics(num_samples=50, with_anomalies=True)
  test_df = pd.DataFrame(test_metrics)
  test_df_copy = test_df.copy()  # 创建副本避免碎片化

  # 预测故障
  predictions = predictor.predict(test_df_copy.to_dict('records'))

  logger.info(f"监督学习预测结果示例: {predictions[:2]}")

  # 创建无监督学习故障预测器
  unsupervised_predictor = create_failure_predictor(predictor_type="unsupervised",
                                                    model_type="iforest")

  # 训练模型 - 使用DataFrame的副本避免性能警告
  logger.info("训练无监督学习故障预测模型...")
  df_copy_unsupervised = df.copy()  # 再次创建副本
  unsupervised_train_result = unsupervised_predictor.train(df_copy_unsupervised.to_dict('records'))

  logger.info(f"无监督学习训练结果: {unsupervised_train_result}")

  # 保存模型
  unsupervised_predictor.save_model("unsupervised_failure_predictor")

  # 预测故障 - 使用DataFrame的副本
  test_df_copy_unsupervised = test_df.copy()
  unsupervised_predictions = unsupervised_predictor.predict(test_df_copy_unsupervised.to_dict('records'))

  logger.info(f"无监督学习预测结果示例: {unsupervised_predictions[:2]}")

  logger.info("故障预测测试完成")


def main():
  parser = argparse.ArgumentParser(description="设置和测试预测模块")
  parser.add_argument("--test-resource", action="store_true", help="测试资源预测")
  parser.add_argument("--test-failure", action="store_true", help="测试故障预测")
  parser.add_argument("--test-all", action="store_true", help="测试所有预测功能")

  args = parser.parse_args()

  if args.test_all or (not args.test_resource and not args.test_failure):
    test_resource_prediction()
    test_failure_prediction()
  else:
    if args.test_resource:
      test_resource_prediction()
    if args.test_failure:
      test_failure_prediction()

  logger.info("预测模块测试完成")


if __name__ == "__main__":
  main()
