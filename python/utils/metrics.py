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

description: 指标收集器
"""

import time
from functools import wraps
from typing import Dict, List, Callable, Any
from utils.logger import get_logger

logger = get_logger("metrics")

class MetricsCollector:
    _metrics = {}

    @classmethod
    def record_time(cls, func_name: str, duration: float):
        """记录函数执行时间"""
        if func_name not in cls._metrics:
            cls._metrics[func_name] = {"count": 0, "total_time": 0, "avg_time": 0}
        
        cls._metrics[func_name]["count"] += 1
        cls._metrics[func_name]["total_time"] += duration
        cls._metrics[func_name]["avg_time"] = cls._metrics[func_name]["total_time"] / cls._metrics[func_name]["count"]
    
    @classmethod
    def get_metrics(cls) -> Dict[str, Dict[str, float]]:
        """获取所有指标"""
        return cls._metrics.copy()
    
    @classmethod
    def reset_metrics(cls):
        """重置所有指标"""
        cls._metrics.clear()

def timing_decorator(func):
    """函数执行时间装饰器"""
    @wraps(func)
    def wrapper(*args, **kwargs):
        start_time = time.time()
        result = func(*args, **kwargs)
        end_time = time.time()
        duration = end_time - start_time
        MetricsCollector.record_time(func.__name__, duration)
        logger.debug(f"Function {func.__name__} took {duration:.4f} seconds to execute")
        return result
    return wrapper

class APIMetrics:
    """API调用指标收集器"""
    _api_metrics = {}

    @classmethod
    def record_api_call(cls, endpoint: str, status_code: int, duration: float):
        """记录API调用"""
        if endpoint not in cls._api_metrics:
            cls._api_metrics[endpoint] = {"calls": 0, "errors": 0, "total_time": 0, "avg_time": 0}
        
        cls._api_metrics[endpoint]["calls"] += 1
        cls._api_metrics[endpoint]["total_time"] += duration
        cls._api_metrics[endpoint]["avg_time"] = cls._api_metrics[endpoint]["total_time"] / cls._api_metrics[endpoint]["calls"]
        
        if status_code >= 400:
            cls._api_metrics[endpoint]["errors"] += 1
    
    @classmethod
    def get_api_metrics(cls) -> Dict[str, Dict[str, float]]:
        """获取API指标"""
        return cls._api_metrics.copy()