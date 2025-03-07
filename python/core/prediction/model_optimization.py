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
import numpy as np
import pandas as pd
from typing import List, Dict, Any, Optional, Tuple, Union
from datetime import datetime
import joblib
from sklearn.model_selection import GridSearchCV, RandomizedSearchCV, TimeSeriesSplit
from sklearn.ensemble import RandomForestRegressor, GradientBoostingRegressor
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error, mean_absolute_error, r2_score
import optuna
from statsmodels.tsa.arima.model import ARIMA
from statsmodels.tsa.statespace.sarimax import SARIMAX
from prophet import Prophet

from utils.logger import get_logger

logger = get_logger("model_optimization")


def create_optimizer(optimizer_type: str, model_dir: str = "./models/prediction") -> 'ModelOptimizer':
    """
    创建模型优化器工厂函数
    
    Args:
        optimizer_type: 优化器类型，可选值为 "time_series", "ml", "failure"
        model_dir: 模型保存目录
        
    Returns:
        对应类型的模型优化器实例
    """
    if optimizer_type == "time_series":
        return TimeSeriesModelOptimizer(model_dir)
    elif optimizer_type == "ml":
        return MLModelOptimizer(model_dir)
    elif optimizer_type == "failure":
        return FailurePredictionModelOptimizer(model_dir)
    else:
        raise ValueError(f"不支持的优化器类型: {optimizer_type}")


class ModelOptimizer:
    """模型优化器基类"""

    def __init__(self, model_dir: str = "./models/prediction"):
        """
        初始化模型优化器

        Args:
            model_dir: 模型保存目录
        """
        self.model_dir = model_dir
        os.makedirs(model_dir, exist_ok=True)

    def optimize(self, data: List[Dict[str, Any]], **kwargs) -> Dict[str, Any]:
        """
        优化模型

        Args:
            data: 训练数据
            **kwargs: 其他参数

        Returns:
            优化结果
        """
        raise NotImplementedError("子类必须实现此方法")

    def save_best_model(self, model: Any, model_name: str) -> str:
        """
        保存最佳模型

        Args:
            model: 模型对象
            model_name: 模型名称

        Returns:
            模型保存路径
        """
        model_path = os.path.join(self.model_dir, f"{model_name}.pkl")
        joblib.dump(model, model_path)
        logger.info(f"最佳模型已保存到 {model_path}")
        return model_path


class TimeSeriesModelOptimizer(ModelOptimizer):
    """时间序列模型优化器"""

    def optimize(self, data: List[Dict[str, Any]], **kwargs) -> Dict[str, Any]:
        """
        优化时间序列模型

        Args:
            data: 训练数据，格式为 [{"timestamp": datetime, "value": float, "metric_name": str}, ...]
            **kwargs: 其他参数
                - model_types: 要尝试的模型类型列表，默认为 ["arima", "sarima", "prophet"]
                - cv_folds: 交叉验证折数，默认为 3
                - metric: 评估指标，默认为 "rmse"

        Returns:
            优化结果
        """
        model_types = kwargs.get("model_types", ["arima", "sarima", "prophet"])
        cv_folds = kwargs.get("cv_folds", 3)
        metric = kwargs.get("metric", "rmse")

        # 准备数据
        df = pd.DataFrame(data)
        df["timestamp"] = pd.to_datetime(df["timestamp"])
        df = df.sort_values("timestamp")

        # 设置时间索引
        ts_df = df.set_index("timestamp")["value"]

        # 分割数据为训练集和验证集
        train_size = int(len(ts_df) * 0.8)
        train_data = ts_df[:train_size]
        valid_data = ts_df[train_size:]

        best_model = None
        best_model_type = None
        best_params = None
        best_score = float("inf") if metric in ["rmse", "mae"] else float("-inf")

        # 尝试不同的模型类型
        for model_type in model_types:
            logger.info(f"优化 {model_type} 模型...")

            if model_type == "arima":
                best_model_result = self._optimize_arima(train_data, valid_data, metric)
            elif model_type == "sarima":
                best_model_result = self._optimize_sarima(
                    train_data, valid_data, metric
                )
            elif model_type == "prophet":
                best_model_result = self._optimize_prophet(
                    train_data, valid_data, metric
                )
            else:
                logger.warning(f"不支持的模型类型: {model_type}")
                continue

            model, params, score = best_model_result

            # 更新最佳模型
            if (metric in ["rmse", "mae"] and score < best_score) or (
                metric == "r2" and score > best_score
            ):
                best_model = model
                best_model_type = model_type
                best_params = params
                best_score = score

        # 使用全部数据重新训练最佳模型
        if best_model_type == "arima":
            final_model = ARIMA(ts_df, order=best_params["order"]).fit()
        elif best_model_type == "sarima":
            final_model = SARIMAX(
                ts_df,
                order=best_params["order"],
                seasonal_order=best_params["seasonal_order"],
            ).fit(disp=False)
        elif best_model_type == "prophet":
            prophet_df = pd.DataFrame({"ds": ts_df.index, "y": ts_df.values})
            final_model = Prophet(**best_params)
            final_model.fit(prophet_df)

        # 保存最佳模型
        model_name = (
            f"optimized_{best_model_type}_{kwargs.get('metric_name', 'metric')}"
        )
        model_path = self.save_best_model(final_model, model_name)

        return {
            "model_type": best_model_type,
            "model_path": model_path,
            "best_params": best_params,
            "best_score": best_score,
            "metric": metric,
        }

    def _optimize_arima(self, train_data, valid_data, metric):
        """优化ARIMA模型"""

        def objective(trial):
            # 定义超参数搜索空间
            p = trial.suggest_int("p", 0, 5)
            d = trial.suggest_int("d", 0, 2)
            q = trial.suggest_int("q", 0, 5)

            try:
                # 训练模型
                model = ARIMA(train_data, order=(p, d, q)).fit(disp=False)

                # 预测验证集
                predictions = model.forecast(steps=len(valid_data))

                # 计算评估指标
                if metric == "rmse":
                    score = np.sqrt(mean_squared_error(valid_data, predictions))
                elif metric == "mae":
                    score = mean_absolute_error(valid_data, predictions)
                elif metric == "r2":
                    score = r2_score(valid_data, predictions)

                return score if metric != "r2" else -score  # Optuna默认最小化目标
            except:
                # 如果模型训练失败，返回一个很差的分数
                return float("inf") if metric != "r2" else float("-inf")

        # 创建Optuna研究
        study = optuna.create_study(direction="minimize")
        study.optimize(objective, n_trials=20)

        # 获取最佳参数
        best_params = study.best_params
        best_order = (best_params["p"], best_params["d"], best_params["q"])

        # 使用最佳参数训练模型
        best_model = ARIMA(train_data, order=best_order).fit(disp=False)

        # 预测验证集
        predictions = best_model.forecast(steps=len(valid_data))

        # 计算最终评估指标
        if metric == "rmse":
            best_score = np.sqrt(mean_squared_error(valid_data, predictions))
        elif metric == "mae":
            best_score = mean_absolute_error(valid_data, predictions)
        elif metric == "r2":
            best_score = r2_score(valid_data, predictions)

        return best_model, {"order": best_order}, best_score

    def _optimize_sarima(self, train_data, valid_data, metric):
        """优化SARIMA模型"""

        def objective(trial):
            # 定义超参数搜索空间
            p = trial.suggest_int("p", 0, 3)
            d = trial.suggest_int("d", 0, 2)
            q = trial.suggest_int("q", 0, 3)
            P = trial.suggest_int("P", 0, 2)
            D = trial.suggest_int("D", 0, 1)
            Q = trial.suggest_int("Q", 0, 2)
            s = trial.suggest_int("s", 4, 12)  # 季节性周期

            try:
                # 训练模型
                model = SARIMAX(
                    train_data, order=(p, d, q), seasonal_order=(P, D, Q, s)
                ).fit(disp=False)

                # 预测验证集
                predictions = model.forecast(steps=len(valid_data))

                # 计算评估指标
                if metric == "rmse":
                    score = np.sqrt(mean_squared_error(valid_data, predictions))
                elif metric == "mae":
                    score = mean_absolute_error(valid_data, predictions)
                elif metric == "r2":
                    score = r2_score(valid_data, predictions)

                return score if metric != "r2" else -score  # Optuna默认最小化目标
            except:
                # 如果模型训练失败，返回一个很差的分数
                return float("inf") if metric != "r2" else float("-inf")

        # 创建Optuna研究
        study = optuna.create_study(direction="minimize")
        study.optimize(objective, n_trials=15)

        # 获取最佳参数
        best_params = study.best_params
        best_order = (best_params["p"], best_params["d"], best_params["q"])
        best_seasonal_order = (
            best_params["P"],
            best_params["D"],
            best_params["Q"],
            best_params["s"],
        )

        # 使用最佳参数训练模型
        best_model = SARIMAX(
            train_data, order=best_order, seasonal_order=best_seasonal_order
        ).fit(disp=False)

        # 预测验证集
        predictions = best_model.forecast(steps=len(valid_data))

        # 计算最终评估指标
        if metric == "rmse":
            best_score = np.sqrt(mean_squared_error(valid_data, predictions))
        elif metric == "mae":
            best_score = mean_absolute_error(valid_data, predictions)
        elif metric == "r2":
            best_score = r2_score(valid_data, predictions)

        return (
            best_model,
            {"order": best_order, "seasonal_order": best_seasonal_order},
            best_score,
        )

    def _optimize_prophet(self, train_data, valid_data, metric):
        """优化Prophet模型"""
        # 准备Prophet格式数据
        train_prophet = pd.DataFrame({"ds": train_data.index, "y": train_data.values})

        valid_prophet = pd.DataFrame({"ds": valid_data.index, "y": valid_data.values})

        def objective(trial):
            # 定义超参数搜索空间
            changepoint_prior_scale = trial.suggest_float(
                "changepoint_prior_scale", 0.001, 0.5, log=True
            )
            seasonality_prior_scale = trial.suggest_float(
                "seasonality_prior_scale", 0.01, 10, log=True
            )
            holidays_prior_scale = trial.suggest_float(
                "holidays_prior_scale", 0.01, 10, log=True
            )
            seasonality_mode = trial.suggest_categorical(
                "seasonality_mode", ["additive", "multiplicative"]
            )

            try:
                # 训练模型
                model = Prophet(
                    changepoint_prior_scale=changepoint_prior_scale,
                    seasonality_prior_scale=seasonality_prior_scale,
                    holidays_prior_scale=holidays_prior_scale,
                    seasonality_mode=seasonality_mode,
                )
                model.fit(train_prophet)

                # 预测验证集
                future = model.make_future_dataframe(periods=len(valid_data), freq="D")
                forecast = model.predict(future)
                predictions = forecast.tail(len(valid_data))["yhat"].values

                # 计算评估指标
                if metric == "rmse":
                    score = np.sqrt(mean_squared_error(valid_data.values, predictions))
                elif metric == "mae":
                    score = mean_absolute_error(valid_data.values, predictions)
                elif metric == "r2":
                    score = r2_score(valid_data.values, predictions)

                return score if metric != "r2" else -score  # Optuna默认最小化目标
            except:
                # 如果模型训练失败，返回一个很差的分数
                return float("inf") if metric != "r2" else float("-inf")

        # 创建Optuna研究
        study = optuna.create_study(direction="minimize")
        study.optimize(objective, n_trials=15)

        # 获取最佳参数
        best_params = {
            "changepoint_prior_scale": study.best_params["changepoint_prior_scale"],
            "seasonality_prior_scale": study.best_params["seasonality_prior_scale"],
            "holidays_prior_scale": study.best_params["holidays_prior_scale"],
            "seasonality_mode": study.best_params["seasonality_mode"],
        }

        # 使用最佳参数训练模型
        best_model = Prophet(**best_params)
        best_model.fit(train_prophet)

        # 预测验证集
        future = best_model.make_future_dataframe(periods=len(valid_data), freq="D")
        forecast = best_model.predict(future)
        predictions = forecast.tail(len(valid_data))["yhat"].values

        # 计算最终评估指标
        if metric == "rmse":
            best_score = np.sqrt(mean_squared_error(valid_data.values, predictions))
        elif metric == "mae":
            best_score = mean_absolute_error(valid_data.values, predictions)
        elif metric == "r2":
            best_score = r2_score(valid_data.values, predictions)

        return best_model, best_params, best_score


class MLModelOptimizer(ModelOptimizer):
    """机器学习模型优化器"""

    def optimize(self, data: List[Dict[str, Any]], **kwargs) -> Dict[str, Any]:
        """
        优化机器学习模型

        Args:
            data: 训练数据，格式为 [{"timestamp": datetime, "target": float, "features": Dict[str, float]}, ...]
            **kwargs: 其他参数
                - model_types: 要尝试的模型类型列表，默认为 ["rf", "gbdt", "linear"]
                - cv_folds: 交叉验证折数，默认为 5
                - metric: 评估指标，默认为 "rmse"
                - search_method: 超参数搜索方法，默认为 "random"

        Returns:
            优化结果
        """
        model_types = kwargs.get("model_types", ["rf", "gbdt", "linear"])
        cv_folds = kwargs.get("cv_folds", 5)
        metric = kwargs.get("metric", "rmse")
        search_method = kwargs.get("search_method", "random")

        # 准备数据
        X, y, timestamps = self._prepare_data(data)

        # 创建时间序列交叉验证
        tscv = TimeSeriesSplit(n_splits=cv_folds)

        best_model = None
        best_model_type = None
        best_params = None
        best_score = float("inf") if metric in ["rmse", "mae"] else float("-inf")

        # 尝试不同的模型类型
        for model_type in model_types:
            logger.info(f"优化 {model_type} 模型...")

            if model_type == "rf":
                model_class = RandomForestRegressor
                param_grid = {
                    "n_estimators": [50, 100, 200],
                    "max_depth": [None, 10, 20, 30],
                    "min_samples_split": [2, 5, 10],
                    "min_samples_leaf": [1, 2, 4],
                }
            elif model_type == "gbdt":
                model_class = GradientBoostingRegressor
                param_grid = {
                    "n_estimators": [50, 100, 200],
                    "learning_rate": [0.01, 0.05, 0.1],
                    "max_depth": [3, 5, 7],
                    "subsample": [0.8, 1.0],
                }
            elif model_type == "linear":
                model_class = LinearRegression
                param_grid = {}  # 线性回归没有超参数
            else:
                logger.warning(f"不支持的模型类型: {model_type}")
                continue

            # 创建基础模型
            base_model = model_class()

            # 如果没有超参数，直接训练模型
            if not param_grid:
                model = base_model
                model.fit(X, y)

                # 计算交叉验证分数
                cv_scores = []
                for train_idx, test_idx in tscv.split(X):
                    X_train, X_test = X[train_idx], X[test_idx]
                    y_train, y_test = y[train_idx], y[test_idx]

                    model.fit(X_train, y_train)
                    y_pred = model.predict(X_test)

                    if metric == "rmse":
                        score = np.sqrt(mean_squared_error(y_test, y_pred))
                    elif metric == "mae":
                        score = mean_absolute_error(y_test, y_pred)
                    elif metric == "r2":
                        score = r2_score(y_test, y_pred)

                    cv_scores.append(score)

                avg_score = np.mean(cv_scores)

                if (metric in ["rmse", "mae"] and avg_score < best_score) or (
                    metric == "r2" and avg_score > best_score
                ):
                    best_model = model
                    best_model_type = model_type
                    best_params = {}
                    best_score = avg_score
            else:
                # 使用网格搜索或随机搜索优化超参数
                if search_method == "grid":
                    search = GridSearchCV(
                        base_model,
                        param_grid,
                        cv=tscv,
                        scoring=self._get_scoring_metric(metric),
                        n_jobs=-1,
                    )
                else:  # 随机搜索
                    search = RandomizedSearchCV(
                        base_model,
                        param_distributions=param_grid,
                        n_iter=20,
                        cv=tscv,
                        scoring=self._get_scoring_metric(metric),
                        n_jobs=-1,
                    )

                # 执行搜索
                search.fit(X, y)

                # 获取最佳模型和分数
                model = search.best_estimator_
                params = search.best_params_
                score = (
                    -search.best_score_
                    if metric in ["rmse", "mae"]
                    else search.best_score_
                )

                if (metric in ["rmse", "mae"] and score < best_score) or (
                    metric == "r2" and score > best_score
                ):
                    best_model = model
                    best_model_type = model_type
                    best_params = params
                    best_score = score

        # 保存最佳模型
        model_name = (
            f"optimized_{best_model_type}_{kwargs.get('metric_name', 'metric')}"
        )
        model_path = self.save_best_model(best_model, model_name)

        return {
            "model_type": best_model_type,
            "model_path": model_path,
            "best_params": best_params,
            "best_score": best_score,
            "metric": metric,
        }

    def _prepare_data(self, data):
        """准备机器学习数据"""
        X_list = []
        y_list = []
        timestamps = []

        for item in data:
            features = item["features"]
            target = item["target"]
            timestamp = item["timestamp"]

            # 将特征转换为数组
            feature_values = list(features.values())

            X_list.append(feature_values)
            y_list.append(target)
            timestamps.append(timestamp)

        return np.array(X_list), np.array(y_list), timestamps

    def _get_scoring_metric(self, metric):
        """获取评分指标"""
        if metric == "rmse":
            return "neg_root_mean_squared_error"
        elif metric == "mae":
            return "neg_mean_absolute_error"
        elif metric == "r2":
            return "r2"
        else:
            return "neg_root_mean_squared_error"  # 默认使用RMSE


class FailurePredictionModelOptimizer(ModelOptimizer):
    """故障预测模型优化器"""

    def optimize(
        self, data: List[Dict[str, Any]], labels: Optional[List[int]] = None, **kwargs
    ) -> Dict[str, Any]:
        """
        优化故障预测模型

        Args:
            data: 训练数据
            labels: 标签数据，用于监督学习
            **kwargs: 其他参数
                - predictor_type: 预测器类型，默认为 "supervised"
                - model_types: 要尝试的模型类型列表
                - cv_folds: 交叉验证折数，默认为 5
                - metric: 评估指标，默认为 "f1"

        Returns:
            优化结果
        """
        predictor_type = kwargs.get("predictor_type", "supervised")

        if predictor_type == "supervised":
            if labels is None:
                raise ValueError("监督学习需要提供标签数据")
            return self._optimize_supervised(data, labels, **kwargs)
        else:
            return self._optimize_unsupervised(data, **kwargs)

    def _optimize_supervised(self, data, labels, **kwargs):
        """优化监督学习故障预测模型"""
        from sklearn.ensemble import RandomForestClassifier, GradientBoostingClassifier
        from sklearn.linear_model import LogisticRegression
        from sklearn.svm import SVC
        from sklearn.metrics import (
            f1_score,
            precision_score,
            recall_score,
            accuracy_score,
        )

        model_types = kwargs.get("model_types", ["rf", "gbdt", "logistic", "svm"])
        cv_folds = kwargs.get("cv_folds", 5)
        metric = kwargs.get("metric", "f1")

        # 准备数据
        X, y, timestamps = self._prepare_data(data, labels)

        # 创建时间序列交叉验证
        tscv = TimeSeriesSplit(n_splits=cv_folds)

        best_model = None
        best_model_type = None
        best_params = None
        best_score = float("-inf")

        # 尝试不同的模型类型
        for model_type in model_types:
            logger.info(f"优化 {model_type} 故障预测模型...")

            if model_type == "rf":
                model_class = RandomForestClassifier
                param_grid = {
                    "n_estimators": [50, 100, 200],
                    "max_depth": [None, 10, 20, 30],
                    "min_samples_split": [2, 5, 10],
                    "class_weight": ["balanced", "balanced_subsample", None],
                }
            elif model_type == "gbdt":
                model_class = GradientBoostingClassifier
                param_grid = {
                    "n_estimators": [50, 100, 200],
                    "learning_rate": [0.01, 0.05, 0.1],
                    "max_depth": [3, 5, 7],
                    "subsample": [0.8, 1.0],
                }
            elif model_type == "logistic":
                model_class = LogisticRegression
                param_grid = {
                    "C": [0.1, 1.0, 10.0],
                    "penalty": ["l1", "l2"],
                    "solver": ["liblinear", "saga"],
                    "class_weight": ["balanced", None],
                }
            elif model_type == "svm":
                model_class = SVC
                param_grid = {
                    "C": [0.1, 1.0, 10.0],
                    "kernel": ["linear", "rbf"],
                    "gamma": ["scale", "auto"],
                    "probability": [True],
                    "class_weight": ["balanced", None],
                }
            else:
                logger.warning(f"不支持的模型类型: {model_type}")
                continue

            # 使用随机搜索优化超参数
            search = RandomizedSearchCV(
                model_class(),
                param_distributions=param_grid,
                n_iter=20,
                cv=tscv,
                scoring=self._get_classification_scoring_metric(metric),
                n_jobs=-1,
            )

            # 执行搜索
            search.fit(X, y)

            # 获取最佳模型和分数
            model = search.best_estimator_
            params = search.best_params_
            score = search.best_score_

            if score > best_score:
                best_model = model
                best_model_type = model_type
                best_params = params
                best_score = score

        # 保存最佳模型
        model_name = f"optimized_{best_model_type}_failure_predictor"
        model_path = self.save_best_model(best_model, model_name)

        # 计算最终评估指标
        y_pred = best_model.predict(X)
        metrics = {
            "accuracy": accuracy_score(y, y_pred),
            "precision": precision_score(y, y_pred, average="weighted"),
            "recall": recall_score(y, y_pred, average="weighted"),
            "f1": f1_score(y, y_pred, average="weighted"),
        }

        return {
            "model_type": best_model_type,
            "model_path": model_path,
            "best_params": best_params,
            "best_score": best_score,
            "metrics": metrics,
            "metric": metric,
        }

    def _optimize_unsupervised(self, data, **kwargs):
        """优化无监督学习故障预测模型"""
        from sklearn.ensemble import IsolationForest
        from sklearn.neighbors import LocalOutlierFactor
        from sklearn.cluster import DBSCAN
        from sklearn.metrics import silhouette_score

        model_types = kwargs.get("model_types", ["iforest", "lof", "dbscan"])

        # 准备数据
        X, _, timestamps = self._prepare_data(data)

        best_model = None
        best_model_type = None
        best_params = None
        best_score = float("-inf")

        # 尝试不同的模型类型
        for model_type in model_types:
            logger.info(f"优化 {model_type} 无监督故障预测模型...")

            if model_type == "iforest":

                def objective(trial):
                    n_estimators = trial.suggest_int("n_estimators", 50, 200)
                    contamination = trial.suggest_float("contamination", 0.01, 0.1)
                    max_samples = trial.suggest_float("max_samples", 0.5, 1.0)

                    model = IsolationForest(
                        n_estimators=n_estimators,
                        contamination=contamination,
                        max_samples=max_samples,
                        random_state=42,
                    )

                    # 训练模型
                    model.fit(X)

                    # 预测异常分数
                    scores = -model.score_samples(X)

                    # 使用异常分数的分布特性作为评估指标
                    # 好的异常检测模型应该有较大的异常分数方差
                    score_variance = np.var(scores)

                    return score_variance

                # 创建Optuna研究
                study = optuna.create_study(direction="maximize")
                study.optimize(objective, n_trials=20)

                # 获取最佳参数
                best_params_model = study.best_params

                # 使用最佳参数创建模型
                model = IsolationForest(
                    n_estimators=best_params_model["n_estimators"],
                    contamination=best_params_model["contamination"],
                    max_samples=best_params_model["max_samples"],
                    random_state=42,
                )

                # 训练模型
                model.fit(X)

                # 计算评分
                score = study.best_value

                if score > best_score:
                    best_model = model
                    best_model_type = model_type
                    best_params = best_params_model
                    best_score = score

            elif model_type == "lof":

                def objective(trial):
                    n_neighbors = trial.suggest_int("n_neighbors", 5, 50)
                    contamination = trial.suggest_float("contamination", 0.01, 0.1)

                    model = LocalOutlierFactor(
                        n_neighbors=n_neighbors,
                        contamination=contamination,
                        novelty=True,
                    )

                    # 训练模型
                    model.fit(X)

                    # 预测异常分数
                    scores = -model.score_samples(X)

                    # 使用异常分数的分布特性作为评估指标
                    score_variance = np.var(scores)

                    return score_variance

                # 创建Optuna研究
                study = optuna.create_study(direction="maximize")
                study.optimize(objective, n_trials=20)

                # 获取最佳参数
                best_params_model = study.best_params

                # 使用最佳参数创建模型
                model = LocalOutlierFactor(
                    n_neighbors=best_params_model["n_neighbors"],
                    contamination=best_params_model["contamination"],
                    novelty=True,
                )

                # 训练模型
                model.fit(X)

                # 计算评分
                score = study.best_value

                if score > best_score:
                    best_model = model
                    best_model_type = model_type
                    best_params = best_params_model
                    best_score = score

            elif model_type == "dbscan":

                def objective(trial):
                    eps = trial.suggest_float("eps", 0.1, 5.0)
                    min_samples = trial.suggest_int("min_samples", 2, 20)

                    model = DBSCAN(eps=eps, min_samples=min_samples)

                    # 训练模型
                    labels = model.fit_predict(X)

                    # 计算轮廓系数，如果只有一个簇，则返回一个较低的分数
                    n_clusters = len(set(labels)) - (1 if -1 in labels else 0)

                    if n_clusters <= 1:
                        return -1.0

                    try:
                        score = silhouette_score(X, labels)
                        return score
                    except:
                        return -1.0

                # 创建Optuna研究
                study = optuna.create_study(direction="maximize")
                study.optimize(objective, n_trials=20)

                # 获取最佳参数
                best_params_model = study.best_params

                # 使用最佳参数创建模型
                model = DBSCAN(
                    eps=best_params_model["eps"],
                    min_samples=best_params_model["min_samples"],
                )

                # 训练模型
                model.fit(X)

                # 计算评分
                score = study.best_value

                if score > best_score:
                    best_model = model
                    best_model_type = model_type
                    best_params = best_params_model
                    best_score = score

        # 保存最佳模型
        model_name = f"optimized_{best_model_type}_unsupervised_failure_predictor"
        model_path = self.save_best_model(best_model, model_name)

        return {
            "model_type": best_model_type,
            "model_path": model_path,
            "best_params": best_params,
            "best_score": best_score,
            "metric": (
                "variance" if best_model_type in ["iforest", "lof"] else "silhouette"
            ),
        }

    def _prepare_data(self, data, labels=None):
        """准备故障预测数据"""
        X_list = []
        timestamps = []

        for item in data:
            # 提取所有数值特征
            features = []
            for key, value in item.items():
                if key != "timestamp" and isinstance(value, (int, float)):
                    features.append(value)

            X_list.append(features)
            timestamps.append(item["timestamp"])

        X = np.array(X_list)

        if labels is not None:
            y = np.array(labels)
            return X, y, timestamps
        else:
            return X, None, timestamps

    def _get_classification_scoring_metric(self, metric):
        """获取分类评分指标"""
        if metric == "f1":
            return "f1_weighted"
        elif metric == "precision":
            return "precision_weighted"
        elif metric == "recall":
            return "recall_weighted"
        elif metric == "accuracy":
            return "accuracy"
        else:
            return "f1_weighted"  # 默认使用F1分数
