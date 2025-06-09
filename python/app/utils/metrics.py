import pandas as pd
import numpy as np
from typing import List, Dict, Any, Optional

class MetricsUtils:
    """指标相关的工具函数"""
    
    @staticmethod
    def normalize_metric_name(metric_name: str) -> str:
        """标准化指标名称"""
        # 移除特殊字符，转换为小写
        normalized = metric_name.lower().replace('-', '_').replace(':', '_')
        return normalized
    
    @staticmethod
    def calculate_percentiles(data: pd.Series, percentiles: List[float] = None) -> Dict[str, float]:
        """计算数据的百分位数"""
        if percentiles is None:
            percentiles = [50, 75, 90, 95, 99]
        
        result = {}
        for p in percentiles:
            result[f'p{p}'] = data.quantile(p/100)
        
        return result
    
    @staticmethod
    def detect_outliers_iqr(data: pd.Series, multiplier: float = 1.5) -> pd.Series:
        """使用IQR方法检测异常值"""
        q1 = data.quantile(0.25)
        q3 = data.quantile(0.75)
        iqr = q3 - q1
        
        lower_bound = q1 - multiplier * iqr
        upper_bound = q3 + multiplier * iqr
        
        return (data < lower_bound) | (data > upper_bound)
    
    @staticmethod
    def calculate_moving_average(data: pd.Series, window: int = 5) -> pd.Series:
        """计算移动平均"""
        return data.rolling(window=window, min_periods=1).mean()
    
    @staticmethod
    def calculate_rate_of_change(data: pd.Series, periods: int = 1) -> pd.Series:
        """计算变化率"""
        return data.pct_change(periods=periods)
    
    @staticmethod
    def aggregate_metrics(metrics_data: Dict[str, pd.DataFrame], method: str = 'mean') -> pd.DataFrame:
        """聚合多个指标数据"""
        if not metrics_data:
            return pd.DataFrame()
        
        aggregated = {}
        
        for metric_name, df in metrics_data.items():
            if 'value' in df.columns:
                if method == 'mean':
                    aggregated[metric_name] = df['value'].resample('1T').mean()
                elif method == 'sum':
                    aggregated[metric_name] = df['value'].resample('1T').sum()
                elif method == 'max':
                    aggregated[metric_name] = df['value'].resample('1T').max()
                elif method == 'min':
                    aggregated[metric_name] = df['value'].resample('1T').min()
        
        return pd.DataFrame(aggregated)
    
    @staticmethod
    def calculate_metric_health_score(data: pd.Series, baseline_mean: float = None, baseline_std: float = None) -> float:
        """计算指标健康分数 (0-1)"""
        if data.empty:
            return 0.0
        
        # 如果没有提供基线，使用历史数据
        if baseline_mean is None:
            baseline_mean = data.mean()
        if baseline_std is None:
            baseline_std = data.std()
        
        if baseline_std == 0:
            return 1.0 if abs(data.iloc[-1] - baseline_mean) < 0.01 else 0.5
        
        # 计算最新值与基线的偏差
        latest_value = data.iloc[-1]
        z_score = abs(latest_value - baseline_mean) / baseline_std
        
        # 转换为健康分数
        health_score = max(0, 1 - (z_score / 3))  # 3个标准差外为0分
        
        return min(1.0, health_score)