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
from typing import Dict, List, Union, Optional, Tuple
from abc import ABC, abstractmethod
import pickle
import os
import logging
import json

logger = logging.getLogger(__name__)

class Normalizer(ABC):
    """归一化器的抽象基类"""
    
    @abstractmethod
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'Normalizer':
        """根据输入数据拟合归一化器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的归一化器实例
        """
        pass
    
    @abstractmethod
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """归一化数据
        
        Args:
            data: 需要归一化的输入数据
            
        Returns:
            归一化后的数据
        """
        pass
    
    @abstractmethod
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反归一化数据
        
        Args:
            data: 需要反归一化的数据
            
        Returns:
            反归一化后的数据
        """
        pass
    
    def fit_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """拟合并归一化数据
        
        Args:
            data: 输入数据
            
        Returns:
            归一化后的数据
        """
        return self.fit(data).transform(data)
    
    def save(self, path: str) -> None:
        """保存归一化器到文件
        
        Args:
            path: 保存路径
        """
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, 'wb') as f:
            pickle.dump(self, f)
        logger.info(f"归一化器已保存到 {path}")
    
    @classmethod
    def load(cls, path: str) -> 'Normalizer':
        """从文件加载归一化器
        
        Args:
            path: 加载路径
            
        Returns:
            加载的归一化器实例
        """
        with open(path, 'rb') as f:
            normalizer = pickle.load(f)
        logger.info(f"已从 {path} 加载归一化器")
        return normalizer


class MinMaxNormalizer(Normalizer):
    """最小-最大归一化器，将数据缩放到指定范围内"""
    
    def __init__(self, feature_range: Tuple[float, float] = (0, 1), copy: bool = True):
        """
        Args:
            feature_range: 归一化的目标范围
            copy: 是否复制输入数据
        """
        self.feature_range = feature_range
        self.copy = copy
        self.min_ = None
        self.scale_ = None
        self.data_min_ = None
        self.data_max_ = None
        self.data_range_ = None
        self.n_samples_seen_ = 0
        self.column_names_ = None
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'MinMaxNormalizer':
        """根据输入数据拟合最小-最大归一化器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的归一化器实例
        """
        if isinstance(data, pd.DataFrame):
            self.column_names_ = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
            self.column_names_ = [f"feature_{i}" for i in range(X.shape[1])]
        
        self.data_min_ = np.nanmin(X, axis=0)
        self.data_max_ = np.nanmax(X, axis=0)
        self.data_range_ = self.data_max_ - self.data_min_
        
        # 处理数据范围为0的特征
        self.data_range_[self.data_range_ == 0.0] = 1.0
        
        # 计算缩放参数
        self.scale_ = (self.feature_range[1] - self.feature_range[0]) / self.data_range_
        self.min_ = self.feature_range[0] - self.data_min_ * self.scale_
        
        self.n_samples_seen_ = X.shape[0]
        
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """归一化数据
        
        Args:
            data: 需要归一化的输入数据
            
        Returns:
            归一化后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("归一化器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 应用缩放
        X = X * self.scale_ + self.min_
        
        if was_dataframe:
            return pd.DataFrame(X, columns=column_names, index=data.index)
        return X
    
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反归一化数据
        
        Args:
            data: 需要反归一化的数据
            
        Returns:
            反归一化后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("归一化器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 反向缩放
        X = (X - self.min_) / self.scale_
        
        if was_dataframe:
            return pd.DataFrame(X, columns=column_names, index=data.index)
        return X


class StandardScaler(Normalizer):
    """标准化缩放器，将数据转换为均值为0，标准差为1的分布"""
    
    def __init__(self, copy: bool = True, with_mean: bool = True, with_std: bool = True):
        """
        Args:
            copy: 是否复制输入数据
            with_mean: 是否减去均值
            with_std: 是否除以标准差
        """
        self.copy = copy
        self.with_mean = with_mean
        self.with_std = with_std
        self.mean_ = None
        self.var_ = None
        self.scale_ = None
        self.n_samples_seen_ = 0
        self.column_names_ = None
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'StandardScaler':
        """根据输入数据拟合标准化缩放器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的归一化器实例
        """
        if isinstance(data, pd.DataFrame):
            self.column_names_ = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
            self.column_names_ = [f"feature_{i}" for i in range(X.shape[1])]
        
        # 计算均值和方差
        self.mean_ = np.nanmean(X, axis=0) if self.with_mean else np.zeros(X.shape[1])
        self.var_ = np.nanvar(X, axis=0) if self.with_std else np.ones(X.shape[1])
        
        # 处理方差为0的特征
        self.var_[self.var_ == 0.0] = 1.0
        
        self.scale_ = np.sqrt(self.var_)
        self.n_samples_seen_ = X.shape[0]
        
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """标准化数据
        
        Args:
            data: 需要标准化的输入数据
            
        Returns:
            标准化后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("标准化器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 应用标准化
        if self.with_mean:
            X = X - self.mean_
        if self.with_std:
            X = X / self.scale_
        
        if was_dataframe:
            return pd.DataFrame(X, columns=column_names, index=data.index)
        return X
    
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反标准化数据
        
        Args:
            data: 需要反标准化的数据
            
        Returns:
            反标准化后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("标准化器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 反向标准化
        if self.with_std:
            X = X * self.scale_
        if self.with_mean:
            X = X + self.mean_
        
        if was_dataframe:
            return pd.DataFrame(X, columns=column_names, index=data.index)
        return X


class RobustScaler(Normalizer):
    """鲁棒归一化器，使用中位数和四分位距离进行缩放，对异常值不敏感"""
    
    def __init__(self, copy: bool = True, with_centering: bool = True, with_scaling: bool = True,
                 quantile_range: Tuple[float, float] = (25.0, 75.0)):
        """
        Args:
            copy: 是否复制输入数据
            with_centering: 是否减去中位数
            with_scaling: 是否除以四分位距离
            quantile_range: 四分位范围
        """
        self.copy = copy
        self.with_centering = with_centering
        self.with_scaling = with_scaling
        self.quantile_range = quantile_range
        self.center_ = None
        self.scale_ = None
        self.n_samples_seen_ = 0
        self.column_names_ = None
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'RobustScaler':
        """根据输入数据拟合鲁棒归一化器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的归一化器实例
        """
        if isinstance(data, pd.DataFrame):
            self.column_names_ = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
            self.column_names_ = [f"feature_{i}" for i in range(X.shape[1])]
        
        # 计算中位数和四分位距离
        q_min, q_max = self.quantile_range
        self.center_ = np.nanmedian(X, axis=0) if self.with_centering else np.zeros(X.shape[1])
        
        if self.with_scaling:
            quantiles = np.nanpercentile(X, [q_min, q_max], axis=0)
            self.scale_ = quantiles[1] - quantiles[0]
            # 处理尺度为0的特征
            self.scale_[self.scale_ == 0.0] = 1.0
        else:
            self.scale_ = np.ones(X.shape[1])
        
        self.n_samples_seen_ = X.shape[0]
        
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """归一化数据
        
        Args:
            data: 需要归一化的输入数据
            
        Returns:
            归一化后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("归一化器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 应用鲁棒缩放
        if self.with_centering:
            X = X - self.center_
        if self.with_scaling:
            X = X / self.scale_
        
        if was_dataframe:
            return pd.DataFrame(X, columns=column_names, index=data.index)
        return X
    
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反归一化数据
        
        Args:
            data: 需要反归一化的数据
            
        Returns:
            反归一化后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("归一化器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 反向鲁棒缩放
        if self.with_scaling:
            X = X * self.scale_
        if self.with_centering:
            X = X + self.center_
        
        if was_dataframe:
            return pd.DataFrame(X, columns=column_names, index=data.index)
        return X


class LogTransformer(Normalizer):
    """对数变换归一化器"""
    
    def __init__(self, copy: bool = True, base: float = np.e, offset: float = 1.0):
        """
        Args:
            copy: 是否复制输入数据
            base: 对数的底数
            offset: 对数变换前添加的偏移量，避免对0或负数取对数
        """
        self.copy = copy
        self.base = base
        self.offset = offset
        self.n_samples_seen_ = 0
        self.column_names_ = None
        self.min_values_ = None
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'LogTransformer':
        """根据输入数据拟合对数变换归一化器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的归一化器实例
        """
        if isinstance(data, pd.DataFrame):
            self.column_names_ = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
            self.column_names_ = [f"feature_{i}" for i in range(X.shape[1])]
        
        # 记录最小值，确保所有值在变换后都是正的
        self.min_values_ = np.nanmin(X, axis=0)
        self.n_samples_seen_ = X.shape[0]
        
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """对数变换数据
        
        Args:
            data: 需要变换的输入数据
            
        Returns:
            变换后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("变换器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 确保所有值都是正的
        offset = np.maximum(0, -self.min_values_) + self.offset
        
        # 应用对数变换
        X_transformed = np.log(X + offset) / np.log(self.base)
        
        if was_dataframe:
            return pd.DataFrame(X_transformed, columns=column_names, index=data.index)
        return X_transformed
    
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反向对数变换数据
        
        Args:
            data: 需要反变换的数据
            
        Returns:
            反变换后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("变换器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 确保所有值都是正的
        offset = np.maximum(0, -self.min_values_) + self.offset
        
        # 应用反向对数变换
        X_inverse = np.power(self.base, X) - offset
        
        if was_dataframe:
            return pd.DataFrame(X_inverse, columns=column_names, index=data.index)
        return X_inverse


class PowerTransformer(Normalizer):
    """幂变换归一化器"""
    
    def __init__(self, copy: bool = True, power: float = 0.5, offset: float = 0.0):
        """
        Args:
            copy: 是否复制输入数据
            power: 幂次，例如0.5表示平方根变换
            offset: 变换前添加的偏移量，避免对负数进行幂变换
        """
        self.copy = copy
        self.power = power
        self.offset = offset
        self.n_samples_seen_ = 0
        self.column_names_ = None
        self.min_values_ = None
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'PowerTransformer':
        """根据输入数据拟合幂变换归一化器
        
        Args:
            data: 用于拟合的输入数据
            
        Returns:
            self: 返回拟合后的归一化器实例
        """
        if isinstance(data, pd.DataFrame):
            self.column_names_ = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
            self.column_names_ = [f"feature_{i}" for i in range(X.shape[1])]
        
        # 记录最小值，确保所有值在变换后都是非负的
        self.min_values_ = np.nanmin(X, axis=0)
        self.n_samples_seen_ = X.shape[0]
        
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """幂变换数据
        
        Args:
            data: 需要变换的输入数据
            
        Returns:
            变换后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("变换器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 确保所有值都是非负的
        offset = np.maximum(0, -self.min_values_) + self.offset
        
        # 应用幂变换
        X_transformed = np.power(X + offset, self.power)
        
        if was_dataframe:
            return pd.DataFrame(X_transformed, columns=column_names, index=data.index)
        return X_transformed
    
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反向幂变换数据
        
        Args:
            data: 需要反变换的数据
            
        Returns:
            反变换后的数据
        """
        if self.n_samples_seen_ == 0:
            raise ValueError("变换器尚未拟合数据")
        
        was_dataframe = False
        if isinstance(data, pd.DataFrame):
            was_dataframe = True
            column_names = data.columns.tolist()
            X = data.values
        else:
            X = data
            if X.ndim == 1:
                X = X.reshape(-1, 1)
        
        if self.copy:
            X = X.copy()
        
        # 确保所有值都是非负的
        offset = np.maximum(0, -self.min_values_) + self.offset
        
        # 应用反向幂变换
        X_inverse = np.power(X, 1.0 / self.power) - offset
        
        if was_dataframe:
            return pd.DataFrame(X_inverse, columns=column_names, index=data.index)
        return X_inverse


class NormalizationPipeline:
    """归一化流水线，串联多个归一化器"""
    
    def __init__(self, normalizers: List[Normalizer]):
        """
        Args:
            normalizers: 归一化器列表
        """
        self.normalizers = normalizers
    
    def fit(self, data: Union[pd.DataFrame, np.ndarray]) -> 'NormalizationPipeline':
        """拟合所有归一化器
        
        Args:
            data: 输入数据
            
        Returns:
            self: 返回拟合后的流水线实例
        """
        X = data
        for normalizer in self.normalizers:
            normalizer.fit(X)
            X = normalizer.transform(X)
        return self
    
    def transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """应用所有归一化器
        
        Args:
            data: 输入数据
            
        Returns:
            归一化后的数据
        """
        X = data
        for normalizer in self.normalizers:
            X = normalizer.transform(X)
        return X
    
    def inverse_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """反向应用所有归一化器
        
        Args:
            data: 输入数据
            
        Returns:
            反归一化后的数据
        """
        X = data
        for normalizer in reversed(self.normalizers):
            X = normalizer.inverse_transform(X)
        return X
    
    def fit_transform(self, data: Union[pd.DataFrame, np.ndarray]) -> Union[pd.DataFrame, np.ndarray]:
        """拟合并应用所有归一化器
        
        Args:
            data: 输入数据
            
        Returns:
            归一化后的数据
        """
        return self.fit(data).transform(data)
    
    def save(self, path: str) -> None:
        """保存归一化流水线到文件
        
        Args:
            path: 保存路径
        """
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, 'wb') as f:
            pickle.dump(self, f)
        logger.info(f"归一化流水线已保存到 {path}")
    
    @classmethod
    def load(cls, path: str) -> 'NormalizationPipeline':
        """从文件加载归一化流水线
        
        Args:
            path: 加载路径
            
        Returns:
            加载的归一化流水线实例
        """
        with open(path, 'rb') as f:
            pipeline = pickle.load(f)
        logger.info(f"已从 {path} 加载归一化流水线")
        return pipeline