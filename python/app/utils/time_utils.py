from datetime import datetime, timedelta, timezone
from typing import Tuple, Optional
import pandas as pd
import numpy as np
import calendar

class TimeUtils:
    """时间相关的工具函数"""
    
    # 中国主要法定节假日（简化版，实际应该使用完整的节假日API或数据）
    HOLIDAYS = {
        # 元旦
        "0101": True, "0102": True, "0103": True,
        # 春节 (假设2024年日期，实际应根据农历确定)
        "0210": True, "0211": True, "0212": True, "0213": True, 
        "0214": True, "0215": True, "0216": True, "0217": True,
        # 清明节
        "0404": True, "0405": True, "0406": True,
        # 劳动节
        "0501": True, "0502": True, "0503": True, "0504": True, "0505": True,
        # 端午节
        "0610": True, "0611": True, "0612": True,
        # 中秋节
        "0917": True, "0918": True, "0919": True,
        # 国庆节
        "1001": True, "1002": True, "1003": True, "1004": True, 
        "1005": True, "1006": True, "1007": True
    }
    
    @staticmethod
    def extract_time_features(timestamp: datetime) -> dict:
        """提取时间特征，包括周期性特征、工作/非工作时间等"""
        # 将时间转换为分钟
        minutes = timestamp.hour * 60 + timestamp.minute
        
        # 计算时间周期性特征
        sin_time = np.sin(2 * np.pi * minutes / 1440)  # 1440分钟 = 24小时
        cos_time = np.cos(2 * np.pi * minutes / 1440)
        
        # 周几特征 (0是周一，6是周日)
        day_of_week = timestamp.weekday()
        
        # 判断是否是周末
        is_weekend = day_of_week >= 5
        
        # 判断是否是工作时间 (工作日9点到17点)
        is_business_hour = (9 <= timestamp.hour <= 17) and not is_weekend
        
        # 判断是否是节假日
        date_key = timestamp.strftime("%m%d")
        is_holiday = TimeUtils.HOLIDAYS.get(date_key, False)
        
        # 获取月份信息和日期信息
        month = timestamp.month
        day = timestamp.day
        
        # 计算月份周期性特征
        sin_month = np.sin(2 * np.pi * month / 12)
        cos_month = np.cos(2 * np.pi * month / 12)
        
        # 判断是否是月初/月末
        is_month_start = day == 1
        days_in_month = calendar.monthrange(timestamp.year, timestamp.month)[1]
        is_month_end = day == days_in_month
        
        # 返回所有特征
        return {
            'sin_time': sin_time,
            'cos_time': cos_time,
            'hour': timestamp.hour,
            'minute': timestamp.minute,
            'day_of_week': day_of_week,
            'sin_day': np.sin(2 * np.pi * day_of_week / 7),
            'cos_day': np.cos(2 * np.pi * day_of_week / 7),
            'is_weekend': is_weekend,
            'is_business_hour': is_business_hour,
            'is_holiday': is_holiday,
            'month': month,
            'day': day,
            'sin_month': sin_month,
            'cos_month': cos_month,
            'is_month_start': is_month_start,
            'is_month_end': is_month_end
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
        now = datetime.now(timezone.utc)
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