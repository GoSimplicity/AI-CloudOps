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

import requests
import pandas as pd
import numpy as np
import logging
import time
from typing import Dict, List, Union, Optional, Tuple
from datetime import datetime, timedelta
from abc import ABC, abstractmethod

logger = logging.getLogger(__name__)

class PrometheusCollector:
    """Prometheus 指标收集器，用于从 Prometheus 查询和收集时序数据"""
    
    def __init__(self, base_url: str, timeout: int = 30):
        """
        Args:
            base_url: Prometheus 服务器的基础 URL
            timeout: 请求超时时间（秒）
        """
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        logger.info(f"初始化 Prometheus 收集器，连接到 {base_url}")
    
    def query(self, query: str, time: Optional[datetime] = None) -> pd.DataFrame:
        """执行 PromQL 即时查询
        
        Args:
            query: PromQL 查询语句
            time: 查询的时间点，默认为当前时间
            
        Returns:
            包含查询结果的 DataFrame
        """
        endpoint = f"{self.base_url}/api/v1/query"
        
        params = {"query": query}
        if time is not None:
            params["time"] = time.timestamp()
        
        try:
            response = requests.get(endpoint, params=params, timeout=self.timeout)
            response.raise_for_status()
            result = response.json()
            
            if result["status"] != "success":
                logger.error(f"Prometheus 查询失败: {result.get('error', '未知错误')}")
                return pd.DataFrame()
            
            return self._parse_query_result(result)
        
        except requests.exceptions.RequestException as e:
            logger.error(f"Prometheus 查询请求异常: {str(e)}")
            return pd.DataFrame()
    
    def query_range(self, query: str, start_time: datetime, end_time: datetime, 
                   step: Union[str, timedelta] = "1m") -> pd.DataFrame:
        """执行 PromQL 范围查询
        
        Args:
            query: PromQL 查询语句
            start_time: 开始时间
            end_time: 结束时间
            step: 步长，可以是字符串（如 "1m"）或 timedelta 对象
            
        Returns:
            包含查询结果的 DataFrame
        """
        endpoint = f"{self.base_url}/api/v1/query_range"
        
        # 转换 step 为字符串格式
        if isinstance(step, timedelta):
            step_seconds = step.total_seconds()
            if step_seconds < 60:
                step = f"{int(step_seconds)}s"
            else:
                step = f"{int(step_seconds // 60)}m"
        
        params = {
            "query": query,
            "start": start_time.timestamp(),
            "end": end_time.timestamp(),
            "step": step
        }
        
        try:
            response = requests.get(endpoint, params=params, timeout=self.timeout)
            response.raise_for_status()
            result = response.json()
            
            if result["status"] != "success":
                logger.error(f"Prometheus 范围查询失败: {result.get('error', '未知错误')}")
                return pd.DataFrame()
            
            return self._parse_range_result(result)
        
        except requests.exceptions.RequestException as e:
            logger.error(f"Prometheus 范围查询请求异常: {str(e)}")
            return pd.DataFrame()
    
    def _parse_query_result(self, result: Dict) -> pd.DataFrame:
        """解析即时查询结果
        
        Args:
            result: Prometheus API 返回的结果
            
        Returns:
            解析后的 DataFrame
        """
        data = result.get("data", {})
        result_type = data.get("resultType", "")
        
        if result_type == "vector":
            results = []
            for item in data.get("result", []):
                metric = item.get("metric", {})
                value = item.get("value", [None, "NaN"])
                
                # 提取标签和值
                row = {k: v for k, v in metric.items()}
                row["value"] = float(value[1]) if value[1] != "NaN" else np.nan
                row["timestamp"] = datetime.fromtimestamp(value[0])
                
                results.append(row)
            
            if not results:
                return pd.DataFrame()
            
            df = pd.DataFrame(results)
            return df
        
        elif result_type == "matrix":
            # 矩阵结果通常来自范围查询，但有时即时查询也会返回
            return self._parse_range_result(result)
        
        elif result_type == "scalar":
            value = data.get("result", [None, "NaN"])
            df = pd.DataFrame({
                "value": [float(value[1]) if value[1] != "NaN" else np.nan],
                "timestamp": [datetime.fromtimestamp(value[0])]
            })
            return df
        
        else:
            logger.warning(f"不支持的结果类型: {result_type}")
            return pd.DataFrame()
    
    def _parse_range_result(self, result: Dict) -> pd.DataFrame:
        """解析范围查询结果
        
        Args:
            result: Prometheus API 返回的结果
            
        Returns:
            解析后的 DataFrame，时间序列数据以宽格式返回
        """
        data = result.get("data", {})
        result_type = data.get("resultType", "")
        
        if result_type != "matrix":
            logger.warning(f"范围查询预期 'matrix' 类型结果，但收到 '{result_type}'")
            return pd.DataFrame()
        
        # 收集所有时间戳
        all_timestamps = set()
        series_data = []
        
        for item in data.get("result", []):
            metric = item.get("metric", {})
            values = item.get("values", [])
            
            # 创建序列标识符
            metric_name = metric.get("__name__", "unknown")
            labels = "_".join([f"{k}={v}" for k, v in sorted(metric.items()) if k != "__name__"])
            series_id = f"{metric_name}_{labels}" if labels else metric_name
            
            # 收集时间戳和值
            timestamps_values = {}
            for ts, val in values:
                timestamp = datetime.fromtimestamp(ts)
                all_timestamps.add(timestamp)
                timestamps_values[timestamp] = float(val) if val != "NaN" else np.nan
            
            series_data.append((series_id, timestamps_values, metric))
        
        if not series_data:
            return pd.DataFrame()
        
        # 创建排序的时间戳列表
        sorted_timestamps = sorted(all_timestamps)
        
        # 创建结果 DataFrame
        result_df = pd.DataFrame(index=sorted_timestamps)
        result_df.index.name = "timestamp"
        
        # 添加每个序列的数据
        for series_id, timestamps_values, metric in series_data:
            series_values = [timestamps_values.get(ts, np.nan) for ts in sorted_timestamps]
            result_df[series_id] = series_values
            
            # 添加元数据列
            for key, value in metric.items():
                meta_col = f"{series_id}_meta_{key}"
                result_df[meta_col] = value
        
        return result_df
    
    def get_metric_names(self) -> List[str]:
        """获取所有可用的指标名称
        
        Returns:
            指标名称列表
        """
        endpoint = f"{self.base_url}/api/v1/label/__name__/values"
        
        try:
            response = requests.get(endpoint, timeout=self.timeout)
            response.raise_for_status()
            result = response.json()
            
            if result["status"] != "success":
                logger.error(f"获取指标名称失败: {result.get('error', '未知错误')}")
                return []
            
            return result.get("data", [])
        
        except requests.exceptions.RequestException as e:
            logger.error(f"获取指标名称请求异常: {str(e)}")
            return []
    
    def get_label_values(self, label: str) -> List[str]:
        """获取指定标签的所有可能值
        
        Args:
            label: 标签名称
            
        Returns:
            标签值列表
        """
        endpoint = f"{self.base_url}/api/v1/label/{label}/values"
        
        try:
            response = requests.get(endpoint, timeout=self.timeout)
            response.raise_for_status()
            result = response.json()
            
            if result["status"] != "success":
                logger.error(f"获取标签值失败: {result.get('error', '未知错误')}")
                return []
            
            return result.get("data", [])
        
        except requests.exceptions.RequestException as e:
            logger.error(f"获取标签值请求异常: {str(e)}")
            return []
    
    def get_metric_metadata(self, metric: Optional[str] = None) -> Dict:
        """获取指标的元数据
        
        Args:
            metric: 指标名称，如果为 None 则获取所有指标的元数据
            
        Returns:
            指标元数据字典
        """
        endpoint = f"{self.base_url}/api/v1/metadata"
        
        params = {}
        if metric is not None:
            params["metric"] = metric
        
        try:
            response = requests.get(endpoint, params=params, timeout=self.timeout)
            response.raise_for_status()
            result = response.json()
            
            if result["status"] != "success":
                logger.error(f"获取指标元数据失败: {result.get('error', '未知错误')}")
                return {}
            
            return result.get("data", {})
        
        except requests.exceptions.RequestException as e:
            logger.error(f"获取指标元数据请求异常: {str(e)}")
            return {}
    
    def check_connection(self) -> bool:
        """检查与 Prometheus 服务器的连接
        
        Returns:
            连接是否成功
        """
        try:
            response = requests.get(f"{self.base_url}/-/healthy", timeout=self.timeout)
            return response.status_code == 200
        except requests.exceptions.RequestException:
            return False