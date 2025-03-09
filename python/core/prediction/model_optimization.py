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
from typing import List, Dict, Any, Optional
import joblib
from sklearn.model_selection import GridSearchCV, RandomizedSearchCV, TimeSeriesSplit
from sklearn.ensemble import RandomForestRegressor, GradientBoostingRegressor
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error, mean_absolute_error, r2_score
from statsmodels.tsa.arima.model import ARIMA
from statsmodels.tsa.statespace.sarimax import SARIMAX
from prophet import Prophet
from typing import List, Dict, Any, Optional
import numpy as np
import optuna
from sklearn.metrics import r2_score, mean_squared_error
from core.prediction.resource_prediction import create_predictor
from utils.logger import get_logger

logger = get_logger("model_optimization")


def create_optimizer(optimizer_type: str = "supervised", **kwargs) -> "ModelOptimizer":
    """
    创建模型优化器工厂函数

    Args:
        optimizer_type: 优化器类型 (supervised, unsupervised, timeseries, ml, failure)
        **kwargs: 其他参数

    Returns:
        优化器实例
    """
    if optimizer_type == "supervised":
        return SupervisedModelOptimizer(
            model_dir=kwargs.get("model_dir", "./models/optimized"),
            n_trials=kwargs.get("n_trials", 20),
            cv=kwargs.get("cv", 5),
        )
    elif optimizer_type == "unsupervised":
        return UnsupervisedModelOptimizer(
            model_dir=kwargs.get("model_dir", "./models/optimized"),
            n_trials=kwargs.get("n_trials", 20),
        )
    elif optimizer_type == "timeseries":
        # 添加对时间序列优化器的支持
        return TimeSeriesModelOptimizer(
            model_dir=kwargs.get("model_dir", "./models/optimized"),
            n_trials=kwargs.get("n_trials", 20),
        )
    elif optimizer_type == "ml":
        return MLModelOptimizer(
            model_dir=kwargs.get("model_dir", "./models/optimized"),
        )
    elif optimizer_type == "failure":
        return FailurePredictionModelOptimizer(
            model_dir=kwargs.get("model_dir", "./models/optimized"),
        )
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

    def __init__(self, model_dir: str = "./models/optimized", n_trials: int = 20):
        """
        初始化时间序列模型优化器

        Args:
            model_dir: 优化后模型保存目录
            n_trials: 优化尝试次数
        """
        super().__init__(model_dir)
        self.n_trials = n_trials

    def optimize(self, data: List[Dict[str, Any]], model_types: List[str] = None, metric_name: str = "cpu_usage") -> Dict[str, Any]:
        """
        优化时间序列预测模型

        Args:
            data: 训练数据
            model_types: 要优化的模型类型列表，默认为 ["arima", "ets"]
            metric_name: 目标指标名称

        Returns:
            优化结果
        """
        if model_types is None:
            model_types = ["arima", "ets"]

        logger.info(f"优化时间序列模型，目标指标: {metric_name}...")

        # 提取时间序列数据
        ts_data = []
        for item in data:
            ts_data.append({
                "timestamp": item["timestamp"],
                "value": item["value"],
                "metric_name": item.get("metric_name", metric_name)
            })

        # 创建优化研究
        study = optuna.create_study(direction="maximize")

        # 定义优化目标函数
        def objective(trial):
            # 定义超参数搜索空间
            model_type = trial.suggest_categorical("model_type", model_types)

            if model_type == "arima":
                # ARIMA参数
                p = trial.suggest_int("p", 0, 5)
                d = trial.suggest_int("d", 0, 2)
                q = trial.suggest_int("q", 0, 5)
                seasonal_p = trial.suggest_int("seasonal_p", 0, 2)
                seasonal_d = trial.suggest_int("seasonal_d", 0, 1)
                seasonal_q = trial.suggest_int("seasonal_q", 0, 2)
                seasonal_m = trial.suggest_int("seasonal_m", 12, 24)

                # 创建预测器
                predictor = create_predictor(
                    predictor_type="timeseries",
                    model_dir=self.model_dir,
                    model_type="arima"
                )

                # 设置模型参数
                predictor.model_params = {
                    "order": (p, d, q),
                    "seasonal_order": (seasonal_p, seasonal_d, seasonal_q, seasonal_m)
                }

            else:  # ets
                # 指数平滑参数
                seasonal = trial.suggest_categorical("seasonal", ["add", "mul", None])
                seasonal_periods = trial.suggest_int("seasonal_periods", 12, 24)

                # 创建预测器
                predictor = create_predictor(
                    predictor_type="timeseries",
                    model_dir=self.model_dir,
                    model_type="ets"
                )

                # 设置模型参数
                predictor.model_params = {
                    "seasonal": seasonal,
                    "seasonal_periods": seasonal_periods
                }

            # 分割数据
            train_size = int(len(ts_data) * 0.8)
            train_data = ts_data[:train_size]
            test_data = ts_data[train_size:]

            if len(train_data) < 10 or len(test_data) < 5:
                return 0.0  # 数据不足

            try:
                # 训练模型
                predictor.train(train_data)

                # 预测
                predictions = predictor.predict(train_data[-10:], future_steps=len(test_data))

                # 计算评分 (使用R²或负RMSE)
                actual = [item["value"] for item in test_data]
                predicted = [pred["value"] for pred in predictions]

                if len(actual) != len(predicted):
                    return 0.0

                # 计算R²分数
                r2 = r2_score(actual, predicted)

                # 计算RMSE
                rmse = np.sqrt(mean_squared_error(actual, predicted))

                # 返回评分 (R² 或 -RMSE)
                return r2 if not np.isnan(r2) else -rmse

            except Exception as e:
                logger.warning(f"优化过程中出错: {str(e)}")
                return 0.0

        # 运行优化
        study.optimize(objective, n_trials=self.n_trials)

        # 获取最佳参数
        best_params = study.best_params
        best_value = study.best_value

        # 使用最佳参数创建和训练模型
        model_type = best_params.get("model_type", "arima")

        predictor = create_predictor(
            predictor_type="timeseries",
            model_dir=self.model_dir,
            model_type=model_type
        )

        if model_type == "arima":
            predictor.model_params = {
                "order": (
                    best_params.get("p", 1),
                    best_params.get("d", 1),
                    best_params.get("q", 0)
                ),
                "seasonal_order": (
                    best_params.get("seasonal_p", 0),
                    best_params.get("seasonal_d", 0),
                    best_params.get("seasonal_q", 0),
                    best_params.get("seasonal_m", 12)
                )
            }
        else:  # ets
            predictor.model_params = {
                "seasonal": best_params.get("seasonal", "add"),
                "seasonal_periods": best_params.get("seasonal_periods", 12)
            }

        # 训练最终模型
        predictor.train(ts_data)

        # 保存模型
        model_path = predictor.save_model(f"optimized_{model_type}_{metric_name}")

        logger.info(f"最佳模型已保存到 {model_path}")

        return {
            "model_type": model_type,
            "best_params": best_params,
            "best_score": best_value,
            "model_path": model_path
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

class SupervisedModelOptimizer(ModelOptimizer):
    """监督学习模型优化器"""

    def __init__(self, model_dir: str = "./models/optimized", n_trials: int = 20, cv: int = 5):
        """
        初始化监督学习模型优化器

        Args:
            model_dir: 模型保存目录
            n_trials: 优化尝试次数
            cv: 交叉验证折数
        """
        super().__init__(model_dir)
        self.n_trials = n_trials
        self.cv = cv

    def optimize(self, data: List[Dict[str, Any]], **kwargs) -> Dict[str, Any]:
        """
        优化监督学习模型

        Args:
            data: 训练数据
            **kwargs: 其他参数，包括:
                - target_col: 目标列名
                - feature_cols: 特征列名列表
                - model_type: 模型类型 (rf, gbdt, linear等)
                - metric: 评估指标 (rmse, mae, r2等)

        Returns:
            优化结果
        """
        # 提取目标和特征
        target_col = kwargs.get("target_col", "target")
        feature_cols = kwargs.get("feature_cols", None)
        model_type = kwargs.get("model_type", "rf")
        metric = kwargs.get("metric", "rmse")

        # 准备数据
        X, y = self._prepare_data(data, target_col, feature_cols)

        # 创建模型和参数网格
        if model_type == "rf":
            model = RandomForestRegressor()
            param_grid = {
                "n_estimators": [50, 100, 200],
                "max_depth": [None, 10, 20, 30],
                "min_samples_split": [2, 5, 10]
            }
        elif model_type == "gbdt":
            model = GradientBoostingRegressor()
            param_grid = {
                "n_estimators": [50, 100, 200],
                "learning_rate": [0.01, 0.1, 0.3],
                "max_depth": [3, 5, 7]
            }
        elif model_type == "linear":
            model = LinearRegression()
            param_grid = {}
        else:
            raise ValueError(f"不支持的模型类型: {model_type}")

        # 创建优化研究
        study = optuna.create_study()

        def objective(trial):
            params = {}
            for param_name, param_values in param_grid.items():
                if isinstance(param_values[0], int):
                    params[param_name] = trial.suggest_int(param_name, min(param_values), max(param_values))
                elif isinstance(param_values[0], float):
                    params[param_name] = trial.suggest_float(param_name, min(param_values), max(param_values))
                else:
                    params[param_name] = trial.suggest_categorical(param_name, param_values)

            model.set_params(**params)
            cv_scores = []

            tscv = TimeSeriesSplit(n_splits=self.cv)
            for train_idx, val_idx in tscv.split(X):
                X_train, X_val = X[train_idx], X[val_idx]
                y_train, y_val = y[train_idx], y[val_idx]

                model.fit(X_train, y_train)
                y_pred = model.predict(X_val)

                if metric == "rmse":
                    score = np.sqrt(mean_squared_error(y_val, y_pred))
                elif metric == "mae":
                    score = mean_absolute_error(y_val, y_pred)
                else:  # r2
                    score = r2_score(y_val, y_pred)

                cv_scores.append(score)

            return np.mean(cv_scores)

        # 执行优化
        study.optimize(objective, n_trials=self.n_trials)

        # 使用最佳参数训练最终模型
        best_params = study.best_params
        model.set_params(**best_params)
        model.fit(X, y)

        # 保存模型
        model_path = self.save_best_model(model, f"optimized_{model_type}")

        return {
            "model_type": model_type,
            "best_params": best_params,
            "best_score": study.best_value,
            "model_path": model_path
        }

    def _prepare_data(self, data: List[Dict[str, Any]], labels: Optional[List[int]] = None):
        """
        准备机器学习数据

        Args:
            data: 训练数据
            labels: 可选的标签数据，用于监督学习

        Returns:
            特征矩阵、目标向量和时间戳列表
        """
        X_list = []
        y_list = []
        timestamps = []

        for item in data:
            features = item["features"]
            target = item.get("target", None)
            timestamp = item["timestamp"]

            # 将特征转换为数组
            feature_values = list(features.values())

            X_list.append(feature_values)
            if target is not None:
                y_list.append(target)
            timestamps.append(timestamp)

        if labels is not None:
            return np.array(X_list), np.array(labels), timestamps
        elif y_list:
            return np.array(X_list), np.array(y_list), timestamps
        else:
            return np.array(X_list), None, timestamps


class UnsupervisedModelOptimizer(ModelOptimizer):
    """无监督学习模型优化器"""

    def __init__(self, model_dir: str = "./models/optimized", n_trials: int = 20):
        """
        初始化无监督学习模型优化器

        Args:
            model_dir: 模型保存目录
            n_trials: 优化尝试次数
        """
        super().__init__(model_dir)
        self.n_trials = n_trials

    def optimize(self, data: List[Dict[str, Any]], **kwargs) -> Dict[str, Any]:
        """
        优化无监督学习模型

        Args:
            data: 训练数据
            **kwargs: 其他参数，包括:
                - feature_cols: 特征列名列表
                - model_type: 模型类型 (kmeans, dbscan等)
                - metric: 评估指标 (silhouette, calinski_harabasz等)

        Returns:
            优化结果
        """
        from sklearn.cluster import KMeans, DBSCAN
        from sklearn.metrics import silhouette_score, calinski_harabasz_score

        feature_cols = kwargs.get("feature_cols", None)
        model_type = kwargs.get("model_type", "kmeans")
        metric = kwargs.get("metric", "silhouette")

        # 准备数据
        X = self._prepare_data(data, feature_cols)

        # 创建优化研究
        study = optuna.create_study(direction="maximize")

        def objective(trial):
            if model_type == "kmeans":
                n_clusters = trial.suggest_int("n_clusters", 2, 10)
                model = KMeans(n_clusters=n_clusters, random_state=42)
            elif model_type == "dbscan":
                eps = trial.suggest_float("eps", 0.1, 1.0)
                min_samples = trial.suggest_int("min_samples", 2, 10)
                model = DBSCAN(eps=eps, min_samples=min_samples)
            else:
                raise ValueError(f"不支持的模型类型: {model_type}")

            # 训练模型
            labels = model.fit_predict(X)

            # 计算评估指标
            if len(np.unique(labels)) < 2:
                return float("-inf")

            if metric == "silhouette":
                score = silhouette_score(X, labels)
            else:  # calinski_harabasz
                score = calinski_harabasz_score(X, labels)

            return score

        # 执行优化
        study.optimize(objective, n_trials=self.n_trials)

        # 使用最佳参数训练最终模型
        best_params = study.best_params
        if model_type == "kmeans":
            model = KMeans(n_clusters=best_params["n_clusters"], random_state=42)
        else:  # dbscan
            model = DBSCAN(eps=best_params["eps"], min_samples=best_params["min_samples"])

        model.fit(X)

        # 保存模型
        model_path = self.save_best_model(model, f"optimized_{model_type}")

        return {
            "model_type": model_type,
            "best_params": best_params,
            "best_score": study.best_value,
            "model_path": model_path
        }

    def _prepare_data(self, data: List[Dict[str, Any]], feature_cols: Optional[List[str]] = None) -> np.ndarray:
        """
        准备训练数据

        Args:
            data: 原始数据
            feature_cols: 特征列名列表

        Returns:
            特征矩阵
        """
        if feature_cols is None:
            feature_cols = list(data[0].keys())

        X = np.array([[item[col] for col in feature_cols] for item in data])
        return X
