from datetime import datetime, timedelta
from typing import Tuple, Optional
import pandas as pd
import numpy as np

class TimeUtils:
    """时间相关的工具函数"""
    
    @staticmethod
    def extract_time_features(timestamp: datetime) -> dict:
        """提取时间特征"""
        # 将时间转换为分钟
        minutes = timestamp.hour * 60 + timestamp.minute
        
        # 计算周期性特征
        sin_time = np.sin(2 * np.pi * minutes / 1440)  # 1440分钟 = 24小时
        cos_time = np.cos(2 * np.pi * minutes / 1440)
        
        return {
            'sin_time': sin_time,
            'cos_time': cos_time,
            'hour': timestamp.hour,
            'day_of_week': timestamp.weekday(),
            'minute': timestamp.minute,
            'is_weekend': timestamp.weekday() >= 5,
            'is_business_hour': 9 <= timestamp.hour <= 17
        }
    
    @staticmethod
    def validate_time_range(start_time: datetime, end_time: datetime, max_range_minutes: int = 1440) -> bool:
        """验证时间范围"""
        if start_time >= end_time:
            return False
        
        time_diff = (end_time - start_time).total_seconds() / 60
        if time_diff > max_range_minutes:
            return False
        
        # 检查是否是未来时间
        now = datetime.utcnow()
        if start_time > now or end_time > now:
            return False
        
        return True
    
    @staticmethod
    def resample_dataframe(df: pd.DataFrame, freq: str = '1T') -> pd.DataFrame:
        """重采样时间序列数据"""
        if df.empty:
            return df
        
        # 确保索引是时间类型
        if not isinstance(df.index, pd.DatetimeIndex):
            return df
        
        # 重采样并前向填充
        return df.resample(freq).mean().fillna(method='ffill')
    
    @staticmethod
    def get_time_windows(start_time: datetime, end_time: datetime, window_size_minutes: int = 5) -> list:
        """获取时间窗口列表"""
        windows = []
        current = start_time
        window_delta = timedelta(minutes=window_size_minutes)
        
        while current < end_time:
            window_end = min(current + window_delta, end_time)
            windows.append((current, window_end))
            current = window_end
        
        return windows
    
    @staticmethod
    def format_duration(seconds: float) -> str:
        """格式化持续时间"""
        if seconds < 60:
            return f"{seconds:.1f}秒"
        elif seconds < 3600:
            return f"{seconds/60:.1f}分钟"
        else:
            return f"{seconds/3600:.1f}小时"