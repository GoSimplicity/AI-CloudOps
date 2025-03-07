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

import numpy as np
import pandas as pd
from typing import Dict, List, Union, Optional, Callable, Tuple
from abc import ABC, abstractmethod
import logging

logger = logging.getLogger(__name__)

class FeatureExtractor(ABC):
    """特征提取器的抽象基类"""
    
    @abstractmethod
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'FeatureExtractor':
        """根据输入数据拟合特征提取器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的特征提取器实例
        """
        pass
    
    @abstractmethod
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """提取特征
        
        Args:
            data: 需要提取特征的输入数据
            
        Returns:
            提取的特征
        """
        pass
    
    def fit_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """拟合并提取特征
        
        Args:
            data: 输入数据
            
        Returns:
            提取的特征
        """
        return self.fit(data).transform(data)


class StatisticalFeatureExtractor(FeatureExtractor):
    """统计特征提取器，提取数值数据的统计特征"""
    
    def __init__(self, features: List[str] = None, window_size: Optional[int] = None):
        """
        Args:
            features: 要提取的统计特征列表，默认为["mean", "std", "min", "max", "median"]
            window_size: 滑动窗口大小，如果为None则使用全部数据
        """
        self.features = features or ["mean", "std", "min", "max", "median"]
        self.window_size = window_size
        self._feature_functions = {
            "mean": np.mean,
            "std": np.std,
            "min": np.min,
            "max": np.max,
            "median": np.median,
            "sum": np.sum,
            "var": np.var,
            "kurtosis": lambda x: pd.Series(x).kurtosis(),
            "skew": lambda x: pd.Series(x).skew(),
            "range": lambda x: np.max(x) - np.min(x),
            "iqr": lambda x: np.percentile(x, 75) - np.percentile(x, 25)
        }
        self._validate_features()
    
    def _validate_features(self):
        """验证请求的特征是否都可用"""
        invalid_features = [f for f in self.features if f not in self._feature_functions]
        if invalid_features:
            raise ValueError(f"不支持的统计特征: {invalid_features}. 支持的特征有: {list(self._feature_functions.keys())}")
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'StatisticalFeatureExtractor':
        """对于统计特征提取器，不需要特殊的拟合步骤"""
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> pd.DataFrame:
        """提取统计特征
        
        Args:
            data: 输入数据
            
        Returns:
            包含提取统计特征的DataFrame
        """
        if isinstance(data, np.ndarray):
            data = pd.DataFrame(data)
        
        results = {}
        
        # 处理每一列
        for column in data.columns:
            if not pd.api.types.is_numeric_dtype(data[column]):
                logger.warning(f"列 '{column}' 不是数值类型，跳过统计特征提取")
                continue
            
            column_data = data[column].values
            
            if self.window_size is None:
                # 使用全部数据计算统计特征
                for feature in self.features:
                    feature_name = f"{column}_{feature}"
                    try:
                        results[feature_name] = self._feature_functions[feature](column_data)
                    except Exception as e:
                        logger.error(f"计算特征 {feature} 时出错: {str(e)}")
                        results[feature_name] = np.nan
            else:
                # 使用滑动窗口计算统计特征
                for feature in self.features:
                    feature_name = f"{column}_{feature}"
                    feature_values = []
                    func = self._feature_functions[feature]
                    
                    for i in range(len(column_data)):
                        start = max(0, i - self.window_size + 1)
                        window_data = column_data[start:i+1]
                        try:
                            value = func(window_data) if len(window_data) > 0 else np.nan
                            feature_values.append(value)
                        except Exception as e:
                            logger.error(f"计算特征 {feature} 时出错: {str(e)}")
                            feature_values.append(np.nan)
                    
                    results[feature_name] = feature_values
        
        return pd.DataFrame(results)


class TimeSeriesFeatureExtractor(FeatureExtractor):
    """时间序列特征提取器"""
    
    def __init__(self, 
                lag_features: List[int] = None, 
                rolling_windows: List[int] = None,
                diff_orders: List[int] = None,
                seasonal_periods: List[int] = None):
        """
        Args:
            lag_features: 滞后特征的周期列表，如[1, 2, 3]表示添加t-1, t-2, t-3时刻的值
            rolling_windows: 滚动窗口大小列表，用于计算滚动统计量
            diff_orders: 差分阶数列表，如[1, 2]表示添加一阶和二阶差分
            seasonal_periods: 季节性周期列表，用于季节性差分
        """
        self.lag_features = lag_features or []
        self.rolling_windows = rolling_windows or []
        self.diff_orders = diff_orders or []
        self.seasonal_periods = seasonal_periods or []
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'TimeSeriesFeatureExtractor':
        """对于时间序列特征提取器，不需要特殊的拟合步骤"""
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> pd.DataFrame:
        """提取时间序列特征
        
        Args:
            data: 输入数据
            
        Returns:
            包含提取时间序列特征的DataFrame
        """
        if isinstance(data, np.ndarray):
            if data.ndim == 1:
                data = pd.DataFrame(data, columns=["value"])
            else:
                data = pd.DataFrame(data)
        
        result = data.copy()
        
        # 处理每一列
        for column in data.columns:
            if not pd.api.types.is_numeric_dtype(data[column]):
                logger.warning(f"列 '{column}' 不是数值类型，跳过时间序列特征提取")
                continue
            
            # 添加滞后特征
            for lag in self.lag_features:
                result[f"{column}_lag_{lag}"] = data[column].shift(lag)
            
            # 添加滚动窗口特征
            for window in self.rolling_windows:
                result[f"{column}_rolling_mean_{window}"] = data[column].rolling(window=window).mean()
                result[f"{column}_rolling_std_{window}"] = data[column].rolling(window=window).std()
            
            # 添加差分特征
            for order in self.diff_orders:
                result[f"{column}_diff_{order}"] = data[column].diff(order)
            
            # 添加季节性差分特征
            for period in self.seasonal_periods:
                result[f"{column}_seasonal_diff_{period}"] = data[column].diff(period)
        
        return result


class FeatureEngineeringPipeline:
    """特征工程流水线，串联多个特征提取器"""
    
    def __init__(self, extractors: List[FeatureExtractor]):
        """
        Args:
            extractors: 特征提取器列表
        """
        self.extractors = extractors
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'FeatureEngineeringPipeline':
        """拟合所有特征提取器
        
        Args:
            data: 输入数据
            
        Returns:
            self: 返回拟合后的流水线实例
        """
        for extractor in self.extractors:
            extractor.fit(data)
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> pd.DataFrame:
        """应用所有特征提取器
        
        Args:
            data: 输入数据
            
        Returns:
            包含所有提取特征的DataFrame
        """
        if isinstance(data, np.ndarray):
            data = pd.DataFrame(data)
        
        result = data.copy()
        
        for extractor in self.extractors:
            features = extractor.transform(data)
            if isinstance(features, pd.DataFrame):
                # 避免列名冲突
                for col in features.columns:
                    if col in result.columns:
                        features.rename(columns={col: f"{col}_derived"}, inplace=True)
                result = pd.concat([result, features], axis=1)
            else:
                logger.warning(f"提取器 {type(extractor).__name__} 未返回DataFrame，跳过合并")
        
        return result
    
    def fit_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> pd.DataFrame:
        """拟合并应用所有特征提取器
        
        Args:
            data: 输入数据
            
        Returns:
            包含所有提取特征的DataFrame
        """
        return self.fit(data).transform(data)