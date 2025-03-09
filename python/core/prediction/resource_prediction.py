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
from typing import Dict, List, Any, Tuple
from datetime import datetime, timedelta
from statsmodels.tsa.arima.model import ARIMA
from statsmodels.tsa.holtwinters import ExponentialSmoothing
from sklearn.ensemble import RandomForestRegressor
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import mean_squared_error, mean_absolute_error, r2_score
import joblib
import os

from utils.logger import get_logger

logger = get_logger("resource_prediction")


class ResourcePredictor:
    """资源预测器基类"""

    def __init__(self, model_dir: str = "./models/prediction"):
        """
        初始化资源预测器

        Args:
            model_dir: 模型保存目录
        """
        self.model_dir = model_dir
        os.makedirs(model_dir, exist_ok=True)
        self.model = None
        self.scaler = StandardScaler()

    def preprocess_data(self, data: List[Dict[str, Any]]) -> pd.DataFrame:
        """
        预处理输入数据

        Args:
            data: 原始数据列表，每个元素是一个字典

        Returns:
            处理后的DataFrame
        """
        df = pd.DataFrame(data)

        # 确保时间列是datetime类型
        if "timestamp" in df.columns:
            df["timestamp"] = pd.to_datetime(df["timestamp"])
            df = df.sort_values("timestamp")
            df = df.set_index("timestamp")

        # 处理缺失值
        df = df.interpolate(method="linear")

        return df

    def extract_features(self, df: pd.DataFrame) -> Tuple[np.ndarray, List[str]]:
        """
        特征提取

        Args:
            df: 预处理后的DataFrame

        Returns:
            特征矩阵和特征名称列表
        """
        # 移除非数值列
        numeric_cols = df.select_dtypes(include=["number"]).columns.tolist()

        # 添加时间特征
        if isinstance(df.index, pd.DatetimeIndex):
            df["hour"] = df.index.hour
            df["day_of_week"] = df.index.dayofweek
            df["day_of_month"] = df.index.day
            df["month"] = df.index.month

            numeric_cols.extend(["hour", "day_of_week", "day_of_month", "month"])

        # 标准化数值特征
        X = self.scaler.fit_transform(df[numeric_cols])

        return X, numeric_cols

    def train(self, data: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        训练预测模型

        Args:
            data: 训练数据

        Returns:
            训练结果信息
        """
        raise NotImplementedError("子类必须实现此方法")

    def predict(self, data: List[Dict[str, Any]], future_steps: int = 1) -> List[Dict[str, Any]]:
        """
        预测未来资源使用情况

        Args:
            data: 历史数据
            future_steps: 预测未来的步数

        Returns:
            预测结果列表
        """
        raise NotImplementedError("子类必须实现此方法")

    def save_model(self, model_name: str) -> str:
        """
        保存模型

        Args:
            model_name: 模型名称

        Returns:
            模型保存路径
        """
        if self.model is None:
            raise ValueError("模型尚未训练")

        model_path = os.path.join(self.model_dir, f"{model_name}.joblib")
        scaler_path = os.path.join(self.model_dir, f"{model_name}_scaler.joblib")

        joblib.dump(self.model, model_path)
        joblib.dump(self.scaler, scaler_path)

        logger.info(f"模型已保存到 {model_path}")
        return model_path

    def load_model(self, model_name: str) -> bool:
        """
        加载模型

        Args:
            model_name: 模型名称

        Returns:
            是否成功加载
        """
        model_path = os.path.join(self.model_dir, f"{model_name}.joblib")
        scaler_path = os.path.join(self.model_dir, f"{model_name}_scaler.joblib")

        if not os.path.exists(model_path) or not os.path.exists(scaler_path):
            logger.warning(f"模型文件不存在: {model_path}")
            return False

        try:
            self.model = joblib.load(model_path)
            self.scaler = joblib.load(scaler_path)
            logger.info(f"模型已从 {model_path} 加载")
            return True
        except Exception as e:
            logger.error(f"加载模型失败: {str(e)}")
            return False


class TimeSeriesPredictor(ResourcePredictor):
    """时间序列预测器"""

    def __init__(self, model_dir: str, model_type: str = "arima", **kwargs):
        """
        初始化时间序列预测器

        Args:
            model_dir: 模型保存目录
            model_type: 模型类型 (arima, ets)
        """
        super().__init__(model_dir)
        self.model_dir = model_dir
        self.model_type = model_type
        self.model = None
        self.metric_name = "cpu_usage"  # 默认指标名称
        self.model_params = {}

    def train(self, data: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        训练时间序列预测模型

        Args:
            data: 训练数据

        Returns:
            训练结果信息
        """
        df = self.preprocess_data(data)

        # 确保有值列
        if "value" not in df.columns:
            raise ValueError("数据中缺少值列: value")

        # 获取目标时间序列
        ts = df["value"]

        if self.model_type == "arima":
            # 自动确定ARIMA参数
            try:
                from pmdarima import auto_arima

                auto_model = auto_arima(
                    ts,
                    seasonal=True,
                    m=24,
                    suppress_warnings=True,
                    error_action="ignore",
                )
                order = auto_model.order
                seasonal_order = auto_model.seasonal_order

                self.model_params = {"order": order, "seasonal_order": seasonal_order}

                # 训练ARIMA模型
                self.model = ARIMA(ts, order=order, seasonal_order=seasonal_order)
                self.model = self.model.fit()

            except ImportError:
                # 如果没有pmdarima，使用默认参数
                self.model = ARIMA(ts, order=(5, 1, 0))
                self.model = self.model.fit()
                self.model_params = {"order": (5, 1, 0)}

        elif self.model_type == "ets":
            # 指数平滑模型
            self.model = ExponentialSmoothing(ts, seasonal="add", seasonal_periods=24)
            self.model = self.model.fit()

        else:
            raise ValueError(f"不支持的模型类型: {self.model_type}")

        logger.info(
            f"{self.model_type.upper()} 模型训练完成，参数: {self.model_params}"
        )

        # 计算简单的评估指标
        if len(ts) > 10:
            train_size = int(len(ts) * 0.8)
            train_data = ts[:train_size]
            test_data = ts[train_size:]
            
            if self.model_type == "arima":
                predictions = self.model.forecast(steps=len(test_data))
            else:  # ets
                predictions = self.model.predict(start=train_size, end=len(ts)-1)
                
            metrics = {
                "rmse": np.sqrt(np.mean((test_data.values - predictions)**2)),
                "mae": np.mean(np.abs(test_data.values - predictions)),
                "r2": 1 - np.sum((test_data.values - predictions)**2) / np.sum((test_data.values - np.mean(test_data.values))**2)
            }
        else:
            metrics = {"rmse": 0.0, "mae": 0.0, "r2": 0.0}
            
        # 如果数据中有metric_name字段，则使用它
        if "metric_name" in df.columns and not df["metric_name"].empty:
            self.metric_name = df["metric_name"].iloc[0]

        return {
            "status": "success", 
            "model_type": self.model_type, 
            "params": self.model_params,
            "metrics": metrics  # 添加评估指标
        }

    def predict(self, data: List[Dict[str, Any]], future_steps: int = 1) -> List[Dict[str, Any]]:
        """
        预测未来资源使用情况

        Args:
            data: 历史数据
            future_steps: 预测未来的步数

        Returns:
            预测结果列表
        """
        if self.model is None:
            raise ValueError("模型尚未训练")

        df = self.preprocess_data(data)

        # 获取最后一个时间戳
        last_timestamp = df.index[-1]

        # 生成预测时间点
        future_timestamps = [
            last_timestamp + timedelta(hours=i + 1) for i in range(future_steps)
        ]

        # 进行预测
        if self.model_type == "arima":
            forecast_results = self.model.forecast(steps=future_steps)
        elif self.model_type == "ets":
            forecast_results = self.model.predict(steps=future_steps)
        else:
            raise ValueError(f"不支持的模型类型: {self.model_type}")

        # 构建预测结果
        predictions = []
        for i, (ts, value) in enumerate(zip(future_timestamps, forecast_results)):
            predictions.append(
                {
                    "timestamp": ts.strftime("%Y-%m-%d %H:%M:%S"),
                    "value": float(value),
                    "metric_name": self.metric_name,
                    "confidence_lower": float(value * 0.9),  # 简化的置信区间
                    "confidence_upper": float(value * 1.1),
                }
            )

        return predictions


class MLPredictor(ResourcePredictor):
    """机器学习预测器"""

    def __init__(self, model_dir: str = "./models/prediction", model_type: str = "rf"):
        """
        初始化机器学习预测器

        Args:
            model_dir: 模型保存目录
            model_type: 模型类型 (rf=随机森林, etc)
        """
        super().__init__(model_dir)
        self.model_type = model_type
        self.feature_names = []
        self.metric_name = "cpu_usage"  # 默认指标名称

    def train(self, data: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        训练机器学习预测模型

        Args:
            data: 训练数据，包含特征和目标值

        Returns:
            训练结果信息
        """
        # 提取特征和目标
        features = []
        targets = []

        for item in data:
            if "features" not in item or "target" not in item:
                raise ValueError("数据格式错误，需要包含'features'和'target'字段")

            features.append(item["features"])
            targets.append(item["target"])
            
            # 如果有metric_name字段，则使用它
            if "metric_name" in item:
                self.metric_name = item["metric_name"]

        # 转换为DataFrame
        X_df = pd.DataFrame(features)
        y = np.array(targets)

        # 标准化特征
        X = self.scaler.fit_transform(X_df)
        self.feature_names = X_df.columns.tolist()

        # 训练模型
        if self.model_type == "rf":
            self.model = RandomForestRegressor(n_estimators=100, random_state=42)
            self.model.fit(X, y)
        else:
            raise ValueError(f"不支持的模型类型: {self.model_type}")

        logger.info(f"{self.model_type.upper()} 模型训练完成")
        
        # 计算训练集上的评估指标
        y_pred = self.model.predict(X)
        metrics = {
            "rmse": np.sqrt(mean_squared_error(y, y_pred)),
            "mae": mean_absolute_error(y, y_pred),
            "r2": r2_score(y, y_pred)
        }

        return {
            "status": "success", 
            "model_type": self.model_type, 
            "feature_count": len(self.feature_names),
            "metrics": metrics  # 添加评估指标
        }

    def predict(self, data: List[Dict[str, Any]], future_steps: int = 1) -> List[Dict[str, Any]]:
        """
        预测未来资源使用情况

        Args:
            data: 历史数据，包含特征
            future_steps: 预测未来的步数

        Returns:
            预测结果列表
        """
        if self.model is None:
            raise ValueError("模型尚未训练")

        # 提取最近的数据点用于预测
        recent_data = data[-future_steps:] if len(data) >= future_steps else data

        predictions = []

        for item in recent_data:
            if "features" not in item:
                raise ValueError("数据格式错误，需要包含'features'字段")

            # 提取特征
            features_df = pd.DataFrame([item["features"]])

            # 确保所有特征都存在
            for feat in self.feature_names:
                if feat not in features_df.columns:
                    features_df[feat] = 0

            # 按照训练时的特征顺序排列
            features_df = features_df[self.feature_names]

            # 标准化特征
            X_pred = self.scaler.transform(features_df)

            # 预测
            pred_value = float(self.model.predict(X_pred)[0])

            # 添加预测结果
            timestamp = item.get("timestamp", datetime.now().strftime("%Y-%m-%d %H:%M:%S"))

            predictions.append({
                "timestamp": timestamp,
                "value": pred_value,
                "metric_name": self.metric_name,  # 添加指标名称
                "confidence_lower": pred_value * 0.9,  # 简化的置信区间
                "confidence_upper": pred_value * 1.1,
            })

        return predictions


def create_predictor(predictor_type: str = "timeseries", **kwargs) -> ResourcePredictor:
    """
    创建预测器工厂函数

    Args:
        predictor_type: 预测器类型 (timeseries, ml)
        **kwargs: 其他参数

    Returns:
        预测器实例
    """
    if predictor_type == "timeseries":
        model_type = kwargs.get("model_type", "arima")
        return TimeSeriesPredictor(
            model_dir=kwargs.get("model_dir", "./models/prediction"),
            model_type=model_type,
        )
    elif predictor_type == "ml":
        model_type = kwargs.get("model_type", "rf")
        return MLPredictor(
            model_dir=kwargs.get("model_dir", "./models/prediction"),
            model_type=model_type,
        )
    else:
        raise ValueError(f"不支持的预测器类型: {predictor_type}")
