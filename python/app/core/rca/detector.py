import logging
import numpy as np
import pandas as pd
from typing import Dict, List, Tuple, Optional
from sklearn.ensemble import IsolationForest
from sklearn.cluster import DBSCAN
from statsmodels.tsa.stattools import adfuller
from scipy import stats
from app.models.data_models import AnomalyResult

logger = logging.getLogger("aiops.detector")

class AnomalyDetector:
    def __init__(self, anomaly_threshold: float = 0.65):
        self.anomaly_threshold = anomaly_threshold
        logger.info(f"异常检测器初始化完成, 阈值: {anomaly_threshold}")
    
    async def detect_anomalies(self, metrics_data: Dict[str, pd.DataFrame]) -> Dict[str, Dict]:
        """检测所有指标的异常"""
        anomalies = {}
        
        logger.info(f"开始检测 {len(metrics_data)} 个指标的异常")
        
        for metric_name, df in metrics_data.items():
            try:
                if df.empty or 'value' not in df.columns:
                    logger.warning(f"指标 {metric_name} 数据为空或缺少value列")
                    continue
                
                # 清理数据
                df_clean = df.dropna(subset=['value'])
                if len(df_clean) < 5:  # 数据点太少
                    logger.warning(f"指标 {metric_name} 数据点过少: {len(df_clean)}")
                    continue
                
                anomaly_result = await self._detect_metric_anomalies(df_clean, metric_name)
                
                if anomaly_result and anomaly_result.anomaly_points:
                    anomalies[metric_name] = {
                        'count': len(anomaly_result.anomaly_points),
                        'first_occurrence': anomaly_result.anomaly_points[0].isoformat(),
                        'last_occurrence': anomaly_result.anomaly_points[-1].isoformat(),
                        'max_score': max(anomaly_result.anomaly_scores),
                        'avg_score': np.mean(anomaly_result.anomaly_scores),
                        'detection_methods': anomaly_result.detection_methods
                    }
                    logger.info(f"指标 {metric_name} 检测到 {len(anomaly_result.anomaly_points)} 个异常点")
                    
            except Exception as e:
                logger.error(f"检测指标 {metric_name} 异常失败: {str(e)}")
                continue
        
        logger.info(f"异常检测完成, 发现 {len(anomalies)} 个异常指标")
        return anomalies
    
    async def _detect_metric_anomalies(self, df: pd.DataFrame, metric_name: str) -> Optional[AnomalyResult]:
        """检测单个指标的异常"""
        try:
            values = df['value'].values
            
            # 1. Z-Score异常检测
            zscore_anomalies = self._zscore_detection(df['value'])
            
            # 2. IQR异常检测
            iqr_anomalies = self._iqr_detection(df['value'])
            
            # 3. 孤立森林异常检测
            isolation_anomalies = self._isolation_forest_detection(df['value'])
            
            # 4. DBSCAN聚类异常检测
            dbscan_anomalies = self._dbscan_detection(df['value'])
            
            # 5. 时间序列平稳性检测
            stationarity_score = self._stationarity_detection(df['value'])
            
            # 6. 移动平均偏差检测
            moving_avg_anomalies = self._moving_average_detection(df['value'])
            
            # 综合异常评分
            anomaly_scores = self._calculate_composite_score(
                zscore_anomalies, iqr_anomalies, isolation_anomalies, 
                dbscan_anomalies, moving_avg_anomalies, stationarity_score
            )
            
            # 识别异常点
            anomaly_mask = anomaly_scores > self.anomaly_threshold
            anomaly_indices = df.index[anomaly_mask]
            
            if len(anomaly_indices) == 0:
                return None
            
            # 计算严重程度
            max_score = np.max(anomaly_scores[anomaly_mask])
            severity = "high" if max_score > 0.8 else "medium" if max_score > 0.6 else "low"
            
            return AnomalyResult(
                metric=metric_name,
                anomaly_points=anomaly_indices.tolist(),
                anomaly_scores=anomaly_scores[anomaly_mask].tolist(),
                detection_methods={
                    'zscore': int(zscore_anomalies[anomaly_mask].sum()),
                    'iqr': int(iqr_anomalies[anomaly_mask].sum()),
                    'isolation_forest': int(isolation_anomalies[anomaly_mask].sum()),
                    'dbscan': int(dbscan_anomalies[anomaly_mask].sum()),
                    'moving_average': int(moving_avg_anomalies[anomaly_mask].sum()),
                    'stationarity_score': float(stationarity_score)
                },
                severity=severity
            )
            
        except Exception as e:
            logger.error(f"检测指标 {metric_name} 异常失败: {str(e)}")
            return None
    
    def _zscore_detection(self, series: pd.Series, threshold: float = 3.0) -> np.ndarray:
        """Z-Score异常检测"""
        try:
            z_scores = np.abs(stats.zscore(series))
            return (z_scores > threshold).astype(int)
        except Exception:
            return np.zeros(len(series), dtype=int)
    
    def _iqr_detection(self, series: pd.Series, multiplier: float = 1.5) -> np.ndarray:
        """IQR异常检测"""
        try:
            q1 = series.quantile(0.25)
            q3 = series.quantile(0.75)
            iqr = q3 - q1
            
            if iqr == 0:
                return np.zeros(len(series), dtype=int)
            
            lower_bound = q1 - multiplier * iqr
            upper_bound = q3 + multiplier * iqr
            return ((series < lower_bound) | (series > upper_bound)).astype(int)
        except Exception:
            return np.zeros(len(series), dtype=int)
    
    def _isolation_forest_detection(
        self, 
        series: pd.Series, 
        contamination: float = 0.1
    ) -> np.ndarray:
        """孤立森林异常检测"""
        try:
            if len(series) < 10:
                return np.zeros(len(series), dtype=int)
            
            # 动态调整contamination
            contamination = min(contamination, 0.5)
            
            iso_forest = IsolationForest(
                contamination=contamination, 
                random_state=42,
                n_estimators=100
            )
            anomalies = iso_forest.fit_predict(series.values.reshape(-1, 1))
            return (anomalies == -1).astype(int)
        except Exception:
            return np.zeros(len(series), dtype=int)
    
    def _dbscan_detection(
        self, 
        series: pd.Series, 
        eps: float = 0.5, 
        min_samples: int = 5
    ) -> np.ndarray:
        """DBSCAN聚类异常检测"""
        try:
            if len(series) < min_samples * 2:
                return np.zeros(len(series), dtype=int)
            
            # 标准化数据
            series_std = series.std()
            if series_std == 0:
                return np.zeros(len(series), dtype=int)
            
            normalized = (series - series.mean()) / series_std
            
            # 动态调整参数
            eps = max(0.1, min(eps, 2.0))
            min_samples = max(3, min(min_samples, len(series) // 4))
            
            dbscan = DBSCAN(eps=eps, min_samples=min_samples)
            clusters = dbscan.fit_predict(normalized.values.reshape(-1, 1))
            return (clusters == -1).astype(int)
        except Exception:
            return np.zeros(len(series), dtype=int)
    
    def _moving_average_detection(
        self, 
        series: pd.Series, 
        window: int = 10, 
        threshold: float = 2.0
    ) -> np.ndarray:
        """移动平均偏差检测"""
        try:
            if len(series) < window:
                return np.zeros(len(series), dtype=int)
            
            # 计算移动平均和标准差
            moving_avg = series.rolling(window=window, min_periods=1).mean()
            moving_std = series.rolling(window=window, min_periods=1).std()
            
            # 计算偏差
            deviations = np.abs(series - moving_avg)
            
            # 避免除零
            moving_std = moving_std.replace(0, np.nan).fillna(moving_std.mean())
            
            # 标准化偏差
            normalized_deviations = deviations / moving_std
            
            return (normalized_deviations > threshold).astype(int)
        except Exception:
            return np.zeros(len(series), dtype=int)
    
    def _stationarity_detection(self, series: pd.Series) -> float:
        """时间序列平稳性检测"""
        try:
            # ADF测试
            clean_series = series.dropna()
            if len(clean_series) < 10:
                return 0.0
            
            result = adfuller(clean_series)
            p_value = result[1]
            
            # p值越大，越不平稳，异常分数越高
            return min(p_value * 2, 1.0)  # 限制在0-1之间
        except Exception:
            return 0.0
    
    def _calculate_composite_score(
        self, 
        zscore: np.ndarray,
        iqr: np.ndarray,
        isolation: np.ndarray,
        dbscan: np.ndarray,
        moving_avg: np.ndarray,
        stationarity: float
    ) -> np.ndarray:
        """计算综合异常评分"""
        # 权重配置
        weights = {
            'zscore': 0.20,
            'iqr': 0.20,
            'isolation': 0.25,
            'dbscan': 0.15,
            'moving_avg': 0.15,
            'stationarity': 0.05
        }
        
        # 计算综合评分
        composite_score = (
            zscore * weights['zscore'] +
            iqr * weights['iqr'] +
            isolation * weights['isolation'] +
            dbscan * weights['dbscan'] +
            moving_avg * weights['moving_avg'] +
            stationarity * weights['stationarity']
        )
        
        return composite_score
    
    def update_threshold(self, new_threshold: float):
        """动态更新异常检测阈值"""
        if 0 < new_threshold <= 1:
            self.anomaly_threshold = new_threshold
            logger.info(f"异常检测阈值更新为: {new_threshold}")
        else:
            logger.warning(f"无效的阈值: {new_threshold}, 保持当前值: {self.anomaly_threshold}")