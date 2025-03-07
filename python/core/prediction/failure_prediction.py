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
from typing import Dict, List, Any, Optional, Tuple, Union
from datetime import datetime, timedelta
import logging
import os
import joblib
from sklearn.ensemble import RandomForestClassifier, IsolationForest
from sklearn.preprocessing import StandardScaler
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report, confusion_matrix

from utils.logger import get_logger

logger = get_logger("failure_prediction")


class FailurePredictor:
    """故障预测器基类"""

    def __init__(self, model_dir: str = "./models/prediction"):
        """
        初始化故障预测器

        Args:
            model_dir: 模型保存目录
        """
        self.model_dir = model_dir
        os.makedirs(model_dir, exist_ok=True)
        self.model = None
        self.scaler = StandardScaler()
        self.feature_names = []

    def preprocess_data(
        self,
        metrics: List[Dict[str, Any]],
        logs: Optional[List[str]] = None,
        traces: Optional[List[Dict[str, Any]]] = None,
    ) -> pd.DataFrame:
        """
        预处理输入数据

        Args:
            metrics: 指标数据
            logs: 日志数据
            traces: 链路追踪数据

        Returns:
            处理后的DataFrame
        """
        # 处理指标数据
        metrics_df = pd.DataFrame(metrics)

        # 确保时间列是datetime类型
        if "timestamp" in metrics_df.columns:
            metrics_df["timestamp"] = pd.to_datetime(metrics_df["timestamp"])
            metrics_df = metrics_df.sort_values("timestamp")

        # 处理日志数据（如果有）
        if logs:
            # 简单示例：提取日志中的错误计数
            error_count = sum(
                1
                for log in logs
                if "error" in log.lower() or "exception" in log.lower()
            )
            warning_count = sum(
                1 for log in logs if "warning" in log.lower() or "warn" in log.lower()
            )

            metrics_df["error_count"] = error_count
            metrics_df["warning_count"] = warning_count

        # 处理链路追踪数据（如果有）
        if traces:
            # 简单示例：提取平均响应时间和错误率
            response_times = [
                trace.get("duration", 0) for trace in traces if "duration" in trace
            ]
            error_traces = [
                trace for trace in traces if trace.get("status", "") == "error"
            ]

            if response_times:
                metrics_df["avg_response_time"] = sum(response_times) / len(
                    response_times
                )
            else:
                metrics_df["avg_response_time"] = 0

            if traces:
                metrics_df["trace_error_rate"] = len(error_traces) / len(traces)
            else:
                metrics_df["trace_error_rate"] = 0

        # 处理缺失值
        metrics_df = metrics_df.interpolate(method="linear")

        return metrics_df


    def extract_features(self, df: pd.DataFrame) -> Tuple[np.ndarray, List[str]]:
        """
        特征提取

        Args:
            df: 预处理后的DataFrame

        Returns:
            特征矩阵和特征名称列表
        """
        # 移除非数值列和时间列
        numeric_cols = df.select_dtypes(include=["number"]).columns.tolist()
        if "timestamp" in numeric_cols:
            numeric_cols.remove("timestamp")

        # 创建一个新的DataFrame来存储所有特征
        feature_dfs = [df[numeric_cols].copy()]
        all_feature_names = numeric_cols.copy()

        # 添加时间特征（如果有timestamp列）
        if "timestamp" in df.columns:
            time_features = pd.DataFrame(
                {
                    "hour": df["timestamp"].dt.hour,
                    "day_of_week": df["timestamp"].dt.dayofweek,
                }
            )
            feature_dfs.append(time_features)
            all_feature_names.extend(["hour", "day_of_week"])

        # 添加统计特征
        if len(df) > 5:  # 确保有足够的数据点
            rolling_features = {}
            for col in numeric_cols:
                if col not in ["hour", "day_of_week"]:
                    # 计算滚动平均和标准差
                    rolling_features[f"{col}_rolling_mean"] = (
                        df[col].rolling(window=5, min_periods=1).mean()
                    )
                    rolling_features[f"{col}_rolling_std"] = (
                        df[col].rolling(window=5, min_periods=1).std().fillna(0)
                    )
                    all_feature_names.extend([f"{col}_rolling_mean", f"{col}_rolling_std"])

            if rolling_features:
                feature_dfs.append(pd.DataFrame(rolling_features))

        # 合并所有特征
        if len(feature_dfs) > 1:
            final_df = pd.concat(feature_dfs, axis=1)
        else:
            final_df = feature_dfs[0]

        # 标准化数值特征
        X = self.scaler.fit_transform(final_df)
        self.feature_names = all_feature_names

        return X, all_feature_names

    def train(
        self,
        metrics: List[Dict[str, Any]],
        labels: List[int],
        logs: Optional[List[str]] = None,
        traces: Optional[List[Dict[str, Any]]] = None,
    ) -> Dict[str, Any]:
        """
        训练故障预测模型

        Args:
            metrics: 指标数据
            labels: 故障标签 (0=正常, 1=故障)
            logs: 日志数据
            traces: 链路追踪数据

        Returns:
            训练结果信息
        """
        raise NotImplementedError("子类必须实现此方法")

    def predict(
        self,
        metrics: List[Dict[str, Any]],
        logs: Optional[List[str]] = None,
        traces: Optional[List[Dict[str, Any]]] = None,
    ) -> List[Dict[str, Any]]:
        """
        预测故障

        Args:
            metrics: 指标数据
            logs: 日志数据
            traces: 链路追踪数据

        Returns:
            预测结果列表
        """
        if self.model is None:
            raise ValueError("模型尚未训练")

        # 数据验证
        if not metrics:
            raise ValueError("指标数据不能为空")

        # 检查必要字段
        required_fields = ["timestamp"]  # 添加必要的字段
        for item in metrics:
            for field in required_fields:
                if field not in item:
                    raise ValueError(f"指标数据缺少必要字段: {field}")

        # 预处理数据
        df = self.preprocess_data(metrics, logs, traces)

        # 检查处理后的数据是否为空
        if df.empty:
            logger.warning("预处理后的数据为空，无法进行预测")
            return []

        # 提取特征 - 但不使用extract_features方法，而是直接构建与训练时相同的特征集
        # 创建一个新的DataFrame，只包含模型期望的特征
        feature_df = pd.DataFrame(index=range(len(df)))
        
        # 获取数值列
        numeric_cols = df.select_dtypes(include=["number"]).columns.tolist()
        if "timestamp" in numeric_cols:
            numeric_cols.remove("timestamp")
            
        # 添加基本特征
        for feat in numeric_cols:
            if feat in df.columns:
                feature_df[feat] = df[feat]
                
        # 添加时间特征
        if "timestamp" in df.columns:
            feature_df["hour"] = df["timestamp"].dt.hour
            feature_df["day_of_week"] = df["timestamp"].dt.dayofweek
            
        # 添加滚动特征
        if len(df) > 5:
            for col in numeric_cols:
                if col not in ["hour", "day_of_week"]:
                    feature_df[f"{col}_rolling_mean"] = df[col].rolling(window=5, min_periods=1).mean()
                    feature_df[f"{col}_rolling_std"] = df[col].rolling(window=5, min_periods=1).std().fillna(0)
        
        # 确保所有特征都存在
        for feat in self.feature_names:
            if feat not in feature_df.columns:
                logger.warning(f"缺失特征: {feat}，使用0填充")
                feature_df[feat] = 0
                
        # 确保特征顺序与训练时一致
        feature_df = feature_df[self.feature_names]
        
        # 使用标准化器转换特征
        X = self.scaler.transform(feature_df)
        
        # 预测
        y_pred = self.model.predict(X)
        y_prob = self.model.predict_proba(X)[:, 1]

        # 构建预测结果
        predictions = []
        for i, (pred, prob) in enumerate(zip(y_pred, y_prob)):
            timestamp = df["timestamp"].iloc[i] if "timestamp" in df.columns else None

            # 确定故障类型（简化示例）
            failure_type = "未知"
            if pred == 1:
                # 根据特征值确定可能的故障类型
                if "cpu_usage" in df.columns and df["cpu_usage"].iloc[i] > 90:
                    failure_type = "CPU资源不足"
                elif "memory_usage" in df.columns and df["memory_usage"].iloc[i] > 90:
                    failure_type = "内存资源不足"
                elif "error_count" in df.columns and df["error_count"].iloc[i] > 0:
                    failure_type = "应用错误"
                elif (
                    "trace_error_rate" in df.columns
                    and df["trace_error_rate"].iloc[i] > 0.1
                ):
                    failure_type = "服务调用异常"

            # 确定可能受影响的组件
            affected_components = []
            if pred == 1:
                # 简化示例：根据特征确定可能受影响的组件
                affected_components = ["应用服务器"]
                if (
                    "database_latency" in df.columns
                    and df["database_latency"].iloc[i] > 100
                ):
                    affected_components.append("数据库")
                if "network_errors" in df.columns and df["network_errors"].iloc[i] > 0:
                    affected_components.append("网络")

            # 建议的预防措施
            prevention_actions = []
            if pred == 1:
                if failure_type == "CPU资源不足":
                    prevention_actions = ["增加CPU资源", "优化应用性能", "启用自动伸缩"]
                elif failure_type == "内存资源不足":
                    prevention_actions = [
                        "增加内存资源",
                        "检查内存泄漏",
                        "优化内存使用",
                    ]
                elif failure_type == "应用错误":
                    prevention_actions = [
                        "检查应用日志",
                        "回滚最近的部署",
                        "修复代码错误",
                    ]
                elif failure_type == "服务调用异常":
                    prevention_actions = [
                        "检查依赖服务",
                        "实施断路器模式",
                        "增加重试机制",
                    ]

            # 预期发生时间（简化示例）
            expected_time = None
            if pred == 1 and timestamp is not None:
                # 简单假设：故障将在1小时内发生
                expected_time = (timestamp + timedelta(hours=1)).strftime(
                    "%Y-%m-%d %H:%M:%S"
                )

            predictions.append(
                {
                    "failure_predicted": bool(pred),
                    "probability": float(prob),
                    "failure_type": failure_type if pred == 1 else None,
                    "expected_time": expected_time,
                    "affected_components": affected_components,
                    "prevention_actions": prevention_actions,
                    "timestamp": (
                        timestamp.strftime("%Y-%m-%d %H:%M:%S")
                        if timestamp is not None
                        else None
                    ),
                }
            )

        return predictions


class SupervisedFailurePredictor(FailurePredictor):
    """监督学习故障预测器"""

    def __init__(self, model_dir: str = "./models/prediction", model_type: str = "rf"):
        """
        初始化监督学习故障预测器

        Args:
            model_dir: 模型保存目录
            model_type: 模型类型 (rf=随机森林)
        """
        super().__init__(model_dir)
        self.model_type = model_type

    def save_model(self, model_name: str) -> str:
        """
        保存模型

        Args:
            model_name: 模型名称

        Returns:
            模型保存路径
        """
        if self.model is None:
            raise ValueError("模型尚未训练，无法保存")
            
        # 创建模型目录
        os.makedirs(self.model_dir, exist_ok=True)
        
        # 保存模型
        model_path = os.path.join(self.model_dir, f"{model_name}.joblib")
        
        # 保存模型元数据
        metadata = {
            "model_name": model_name,
            "model_type": self.__class__.__name__,
            "feature_names": self.feature_names,
            "feature_count": len(self.feature_names),
            "created_at": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "version": "1.0.0",
        }
        
        metadata_path = os.path.join(self.model_dir, f"{model_name}_metadata.json")
        with open(metadata_path, "w") as f:
            import json
            json.dump(metadata, f, indent=2)
        
        # 保存模型和标准化器
        model_data = {
            "model": self.model,
            "scaler": self.scaler,
            "feature_names": self.feature_names,
            "metadata": metadata
        }
        
        joblib.dump(model_data, model_path)
        logger.info(f"模型已保存到: {model_path}")
        
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
        
        if not os.path.exists(model_path):
            logger.error(f"模型文件不存在: {model_path}")
            return False
            
        try:
            model_data = joblib.load(model_path)
            
            self.model = model_data["model"]
            self.scaler = model_data["scaler"]
            self.feature_names = model_data["feature_names"]
            
            logger.info(f"模型已从 {model_path} 加载")
            return True
        except Exception as e:
            logger.error(f"加载模型失败: {str(e)}")
            return False

    def train(
        self,
        metrics: List[Dict[str, Any]],
        labels: List[int],
        logs: Optional[List[str]] = None,
        traces: Optional[List[Dict[str, Any]]] = None,
    ) -> Dict[str, Any]:
        """
        训练监督学习故障预测模型

        Args:
            metrics: 指标数据
            labels: 故障标签 (0=正常, 1=故障)
            logs: 日志数据
            traces: 链路追踪数据

        Returns:
            训练结果信息
        """
        # 预处理数据
        df = self.preprocess_data(metrics, logs, traces)

        # 确保标签长度匹配
        if len(labels) != len(df):
            raise ValueError(
                f"标签数量 ({len(labels)}) 与数据点数量 ({len(df)}) 不匹配"
            )

        # 特征提取
        X, feature_names = self.extract_features(df)
        y = np.array(labels)

        # 划分训练集和测试集
        X_train, X_test, y_train, y_test = train_test_split(
            X, y, test_size=0.2, random_state=42
        )

        # 训练模型
        if self.model_type == "rf":
            self.model = RandomForestClassifier(
                n_estimators=100, max_depth=10, random_state=42
            )
        else:
            raise ValueError(f"不支持的模型类型: {self.model_type}")

        self.model.fit(X_train, y_train)

        # 评估模型
        y_pred = self.model.predict(X_test)

        # 计算评估指标
        from sklearn.metrics import (
            accuracy_score,
            precision_score,
            recall_score,
            f1_score,
        )

        results = {
            "model_type": self.model_type,
            "accuracy": accuracy_score(y_test, y_pred),
            "precision": precision_score(y_test, y_pred, zero_division=0),
            "recall": recall_score(y_test, y_pred, zero_division=0),
            "f1": f1_score(y_test, y_pred, zero_division=0),
        }

        # 保存模型
        model_path = os.path.join(
            self.model_dir, f"supervised_{self.model_type}.joblib"
        )
        scaler_path = os.path.join(
            self.model_dir, f"supervised_{self.model_type}_scaler.joblib"
        )

        joblib.dump(self.model, model_path)
        joblib.dump(self.scaler, scaler_path)

        logger.info(f"{self.model_type.upper()} 模型训练完成")
        logger.info(f"评估结果: {results}")

        return results


class UnsupervisedFailurePredictor(FailurePredictor):
    """无监督学习故障预测器"""

    def __init__(
        self, model_dir: str = "./models/prediction", model_type: str = "iforest"
    ):
        """
        初始化无监督学习故障预测器

        Args:
            model_dir: 模型保存目录
            model_type: 模型类型 (iforest=隔离森林)
        """
        super().__init__(model_dir)
        self.model_type = model_type
        self.threshold = -0.5  # 默认异常分数阈值

    # 添加save_model方法
    def save_model(self, model_name: str) -> str:
        """
        保存模型

        Args:
            model_name: 模型名称

        Returns:
            模型保存路径
        """
        if self.model is None:
            raise ValueError("模型尚未训练，无法保存")
            
        # 创建模型目录
        os.makedirs(self.model_dir, exist_ok=True)
        
        # 保存模型
        model_path = os.path.join(self.model_dir, f"{model_name}.joblib")
        
        # 保存模型元数据
        metadata = {
            "model_name": model_name,
            "model_type": self.__class__.__name__,
            "feature_names": self.feature_names,
            "feature_count": len(self.feature_names),
            "created_at": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "version": "1.0.0",
        }
        
        metadata_path = os.path.join(self.model_dir, f"{model_name}_metadata.json")
        with open(metadata_path, "w") as f:
            import json
            json.dump(metadata, f, indent=2)
        
        # 保存模型和标准化器
        model_data = {
            "model": self.model,
            "scaler": self.scaler,
            "feature_names": self.feature_names,
            "metadata": metadata
        }
        
        joblib.dump(model_data, model_path)
        logger.info(f"模型已保存到: {model_path}")
        
        return model_path
        
    # 添加load_model方法
    def load_model(self, model_name: str) -> bool:
        """
        加载模型

        Args:
            model_name: 模型名称

        Returns:
            是否成功加载
        """
        model_path = os.path.join(self.model_dir, f"{model_name}.joblib")
        
        if not os.path.exists(model_path):
            logger.error(f"模型文件不存在: {model_path}")
            return False
            
        try:
            model_data = joblib.load(model_path)
            
            self.model = model_data["model"]
            self.scaler = model_data["scaler"]
            self.feature_names = model_data["feature_names"]
            
            logger.info(f"模型已从 {model_path} 加载")
            return True
        except Exception as e:
            logger.error(f"加载模型失败: {str(e)}")
            return False

    def train(
        self,
        metrics: List[Dict[str, Any]],
        labels: Optional[List[int]] = None,  # 无监督学习不需要标签，但保持接口一致
        logs: Optional[List[str]] = None,
        traces: Optional[List[Dict[str, Any]]] = None,
    ) -> Dict[str, Any]:
        """
        训练无监督学习故障预测模型

        Args:
            metrics: 指标数据
            labels: 故障标签（可选，用于评估）
            logs: 日志数据
            traces: 链路追踪数据

        Returns:
            训练结果信息
        """
        # 预处理数据
        df = self.preprocess_data(metrics, logs, traces)

        # 特征提取
        X, feature_names = self.extract_features(df)

        # 训练模型
        if self.model_type == "iforest":
            self.model = IsolationForest(
                n_estimators=100,
                contamination=0.05,  # 假设5%的数据是异常的
                random_state=42,
            )
        else:
            raise ValueError(f"不支持的模型类型: {self.model_type}")

        self.model.fit(X)

        # 评估模型（如果有标签）
        results = {"model_type": self.model_type}
        if labels is not None and len(labels) == len(X):
            # 预测异常分数
            scores = self.model.decision_function(X)
            # 预测结果（-1表示异常，1表示正常）
            predictions = self.model.predict(X)
            # 转换为二进制标签（0表示正常，1表示异常）
            binary_preds = [1 if p == -1 else 0 for p in predictions]

            # 计算评估指标
            from sklearn.metrics import (
                accuracy_score,
                precision_score,
                recall_score,
                f1_score,
            )

            results["accuracy"] = accuracy_score(labels, binary_preds)
            results["precision"] = precision_score(
                labels, binary_preds, zero_division=0
            )
            results["recall"] = recall_score(labels, binary_preds, zero_division=0)
            results["f1"] = f1_score(labels, binary_preds, zero_division=0)

        logger.info(f"{self.model_type.upper()} 模型训练完成")
        if "accuracy" in results:
            logger.info(f"评估结果: {results}")

        return results

    def predict(
        self,
        metrics: List[Dict[str, Any]],
        logs: Optional[List[str]] = None,
        traces: Optional[List[Dict[str, Any]]] = None,
    ) -> List[Dict[str, Any]]:
        """
        预测故障

        Args:
            metrics: 指标数据
            logs: 日志数据
            traces: 链路追踪数据

        Returns:
            预测结果列表
        """
        if self.model is None:
            raise ValueError("模型尚未训练")

        # 预处理数据
        df = self.preprocess_data(metrics, logs, traces)

        # 创建一个新的DataFrame，只包含模型期望的特征
        feature_df = pd.DataFrame(index=range(len(df)))
        
        # 获取数值列
        numeric_cols = df.select_dtypes(include=["number"]).columns.tolist()
        if "timestamp" in numeric_cols:
            numeric_cols.remove("timestamp")
            
        # 添加基本特征
        for feat in numeric_cols:
            if feat in df.columns:
                feature_df[feat] = df[feat]
                
        # 添加时间特征
        if "timestamp" in df.columns:
            feature_df["hour"] = df["timestamp"].dt.hour
            feature_df["day_of_week"] = df["timestamp"].dt.dayofweek
            
        # 添加滚动特征
        if len(df) > 5:
            for col in numeric_cols:
                if col not in ["hour", "day_of_week"]:
                    feature_df[f"{col}_rolling_mean"] = df[col].rolling(window=5, min_periods=1).mean()
                    feature_df[f"{col}_rolling_std"] = df[col].rolling(window=5, min_periods=1).std().fillna(0)
        
        # 确保所有特征都存在
        for feat in self.feature_names:
            if feat not in feature_df.columns:
                logger.warning(f"缺失特征: {feat}，使用0填充")
                feature_df[feat] = 0
                
        # 确保特征顺序与训练时一致
        feature_df = feature_df[self.feature_names]
        
        # 使用标准化器转换特征
        X = self.scaler.transform(feature_df)

        # 预测异常分数
        scores = self.model.decision_function(X)
        # 预测结果（-1表示异常，1表示正常）
        predictions = self.model.predict(X)

        # 构建预测结果
        results = []
        for i, (pred, score) in enumerate(zip(predictions, scores)):
            # 转换为二进制标签（0表示正常，1表示异常）
            is_anomaly = pred == -1

            # 计算异常概率（简化转换）
            # 将分数转换为0-1之间的概率值
            probability = 1.0 / (1.0 + np.exp(min(5, max(-5, score))))

            timestamp = df["timestamp"].iloc[i] if "timestamp" in df.columns else None

            # 确定故障类型（简化示例）
            failure_type = "未知"
            if is_anomaly:
                # 查找贡献最大的特征
                feature_contributions = {}
                for j, feature in enumerate(self.feature_names):
                    if j < X.shape[1]:
                        # 简化的特征贡献度计算
                        contribution = abs(X[i, j])
                        feature_contributions[feature] = contribution

                # 找出贡献最大的特征
                if feature_contributions:
                    top_feature = max(
                        feature_contributions.items(), key=lambda x: x[1]
                    )[0]

                    if "cpu" in top_feature.lower():
                        failure_type = "CPU异常"
                    elif "memory" in top_feature.lower():
                        failure_type = "内存异常"
                    elif "disk" in top_feature.lower():
                        failure_type = "磁盘异常"
                    elif "network" in top_feature.lower():
                        failure_type = "网络异常"
                    elif "error" in top_feature.lower():
                        failure_type = "错误率异常"
                    elif (
                        "latency" in top_feature.lower()
                        or "response" in top_feature.lower()
                    ):
                        failure_type = "延迟异常"

            # 确定可能受影响的组件和预防措施（与监督学习方法类似）
            affected_components = []
            prevention_actions = []
            if is_anomaly:
                # 简化示例
                affected_components = ["应用服务器"]

                if failure_type == "CPU异常":
                    prevention_actions = ["检查CPU使用情况", "优化应用性能"]
                elif failure_type == "内存异常":
                    prevention_actions = ["检查内存使用情况", "排查内存泄漏"]
                elif failure_type == "磁盘异常":
                    prevention_actions = ["检查磁盘空间", "清理不必要的文件"]
                elif failure_type == "网络异常":
                    prevention_actions = ["检查网络连接", "排查网络瓶颈"]
                elif failure_type == "错误率异常":
                    prevention_actions = ["检查应用日志", "排查错误原因"]
                elif failure_type == "延迟异常":
                    prevention_actions = ["检查系统负载", "优化性能瓶颈"]
                else:
                    prevention_actions = ["进行系统全面检查", "监控关键指标"]

            # 预期发生时间
            expected_time = None
            if is_anomaly and timestamp is not None:
                # 简单假设：故障将在概率*10小时内发生
                hours = max(1, min(24, int(10 * probability)))
                expected_time = (timestamp + timedelta(hours=hours)).strftime(
                    "%Y-%m-%d %H:%M:%S"
                )

            results.append(
                {
                    "failure_predicted": is_anomaly,
                    "probability": float(probability),
                    "failure_type": failure_type if is_anomaly else None,
                    "expected_time": expected_time,
                    "affected_components": affected_components,
                    "prevention_actions": prevention_actions,
                    "timestamp": (
                        timestamp.strftime("%Y-%m-%d %H:%M:%S")
                        if timestamp is not None
                        else None
                    ),
                    "anomaly_score": float(score),
                }
            )

        return results


def create_failure_predictor(
    predictor_type: str = "supervised", **kwargs
) -> FailurePredictor:
    """
    创建故障预测器工厂函数

    Args:
        predictor_type: 预测器类型 (supervised, unsupervised)
        **kwargs: 其他参数

    Returns:
        故障预测器实例
    """
    if predictor_type == "supervised":
        model_type = kwargs.get("model_type", "rf")
        return SupervisedFailurePredictor(
            model_dir=kwargs.get("model_dir", "./models/prediction"),
            model_type=model_type,
        )
    elif predictor_type == "unsupervised":
        model_type = kwargs.get("model_type", "iforest")
        return UnsupervisedFailurePredictor(
            model_dir=kwargs.get("model_dir", "./models/prediction"),
            model_type=model_type,
        )
    else:
        raise ValueError(f"不支持的预测器类型: {predictor_type}")
