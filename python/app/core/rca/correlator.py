import logging
import numpy as np
import pandas as pd
from typing import Dict, List, Tuple, Optional
from scipy.stats import pearsonr, spearmanr
from app.models.data_models import CorrelationResult
from app.config.settings import config

logger = logging.getLogger("aiops.correlator")

class CorrelationAnalyzer:
    def __init__(self, correlation_threshold: float = None):
        self.correlation_threshold = correlation_threshold or config.rca.correlation_threshold
        logger.info(f"相关性分析器初始化完成, 阈值: {self.correlation_threshold}")
    
    async def analyze_correlations(
        self, 
        metrics_data: Dict[str, pd.DataFrame]
    ) -> Dict[str, List[Tuple[str, float]]]:
        """分析指标间的相关性"""
        try:
            if len(metrics_data) < 2:
                logger.warning("指标数量少于2个，无法进行相关性分析")
                return {}
            
            # 准备数据
            combined_df = self._prepare_correlation_data(metrics_data)
            if combined_df.empty:
                logger.warning("准备相关性分析数据失败")
                return {}
            
            logger.info(f"准备了 {len(combined_df.columns)} 个指标进行相关性分析")
            
            # 计算相关性矩阵
            correlation_matrix = self._calculate_correlation_matrix(combined_df)
            
            # 提取显著相关性
            significant_correlations = self._extract_significant_correlations(
                correlation_matrix
            )
            
            logger.info(f"发现 {len(significant_correlations)} 组显著相关性")
            return significant_correlations
            
        except Exception as e:
            logger.error(f"相关性分析失败: {str(e)}")
            return {}
    
    def _prepare_correlation_data(
        self, 
        metrics_data: Dict[str, pd.DataFrame]
    ) -> pd.DataFrame:
        """准备相关性分析数据"""
        try:
            series_list = []
            
            for metric_name, df in metrics_data.items():
                if 'value' in df.columns and not df.empty:
                    # 清理数据
                    clean_series = df['value'].dropna()
                    if len(clean_series) > 5:  # 确保有足够的数据点
                        clean_series.name = metric_name
                        series_list.append(clean_series)
            
            if not series_list:
                return pd.DataFrame()
            
            # 合并时间序列，使用外连接
            combined_df = pd.concat(series_list, axis=1, join='outer')
            
            # 重采样到统一时间间隔
            if not combined_df.empty:
                combined_df = combined_df.resample('1T').mean()
            
            # 只保留有足够数据的行
            min_valid_points = max(3, len(combined_df.columns) // 2)
            combined_df = combined_df.dropna(thresh=min_valid_points)
            
            # 移除方差为0的列
            for col in combined_df.columns:
                if combined_df[col].var() == 0:
                    combined_df = combined_df.drop(columns=[col])
                    logger.warning(f"移除方差为0的指标: {col}")
            
            logger.info(f"相关性分析数据准备完成: {combined_df.shape}")
            return combined_df
            
        except Exception as e:
            logger.error(f"准备相关性数据失败: {str(e)}")
            return pd.DataFrame()
    
    def _calculate_correlation_matrix(self, df: pd.DataFrame) -> pd.DataFrame:
        """计算相关性矩阵"""
        try:
            # 使用Pearson相关系数
            correlation_matrix = df.corr(method='pearson')
            
            # 处理NaN值
            correlation_matrix = correlation_matrix.fillna(0)
            
            logger.debug(f"相关性矩阵计算完成: {correlation_matrix.shape}")
            return correlation_matrix
            
        except Exception as e:
            logger.error(f"计算相关性矩阵失败: {str(e)}")
            return pd.DataFrame()
    
    def _extract_significant_correlations(
        self, 
        correlation_matrix: pd.DataFrame
    ) -> Dict[str, List[Tuple[str, float]]]:
        """提取显著相关性"""
        significant_correlations = {}
        
        try:
            for metric in correlation_matrix.columns:
                correlations = []
                
                for other_metric in correlation_matrix.columns:
                    if metric != other_metric:
                        corr_value = correlation_matrix.loc[metric, other_metric]
                        
                        # 检查是否显著相关
                        if abs(corr_value) >= self.correlation_threshold and not np.isnan(corr_value):
                            correlations.append((other_metric, round(corr_value, 3)))
                
                if correlations:
                    # 按相关性强度排序
                    correlations.sort(key=lambda x: abs(x[1]), reverse=True)
                    significant_correlations[metric] = correlations[:5]  # 只保留前5个
            
            return significant_correlations
            
        except Exception as e:
            logger.error(f"提取显著相关性失败: {str(e)}")
            return {}
    
    async def calculate_cross_correlation(
        self, 
        series1: pd.Series, 
        series2: pd.Series, 
        max_lags: int = 10
    ) -> Dict[int, float]:
        """计算交叉相关性（考虑时间滞后）"""
        try:
            # 确保两个序列长度相同
            min_length = min(len(series1), len(series2))
            if min_length < max_lags * 2:
                max_lags = min_length // 2
            
            series1 = series1.iloc[-min_length:]
            series2 = series2.iloc[-min_length:]
            
            cross_correlations = {}
            
            for lag in range(-max_lags, max_lags + 1):
                try:
                    if lag < 0:
                        # series1滞后
                        s1 = series1.iloc[-lag:]
                        s2 = series2.iloc[:lag] if lag != 0 else series2
                    elif lag > 0:
                        # series2滞后
                        s1 = series1.iloc[:-lag] if lag != 0 else series1
                        s2 = series2.iloc[lag:]
                    else:
                        # 无滞后
                        s1 = series1
                        s2 = series2
                    
                    if len(s1) > 3 and len(s2) > 3 and len(s1) == len(s2):
                        corr, p_value = pearsonr(s1, s2)
                        if not np.isnan(corr) and p_value < 0.05:  # 显著性检验
                            cross_correlations[lag] = round(corr, 3)
                except Exception:
                    continue
            
            return cross_correlations
            
        except Exception as e:
            logger.error(f"计算交叉相关性失败: {str(e)}")
            return {}
    
    async def detect_causal_relationships(
        self, 
        metrics_data: Dict[str, pd.DataFrame],
        max_lag: int = 5
    ) -> Dict[str, List[str]]:
        """检测因果关系（基于时间滞后的简化Granger因果检验）"""
        try:
            causal_relationships = {}
            
            # 准备数据
            combined_df = self._prepare_correlation_data(metrics_data)
            if combined_df.shape[1] < 2:
                return {}
            
            metrics = list(combined_df.columns)
            
            for i, metric1 in enumerate(metrics):
                potential_causes = []
                
                for j, metric2 in enumerate(metrics):
                    if i != j:
                        # 计算metric2对metric1的因果关系
                        if self._test_granger_causality(
                            combined_df[metric1], 
                            combined_df[metric2], 
                            max_lag
                        ):
                            potential_causes.append(metric2)
                
                if potential_causes:
                    causal_relationships[metric1] = potential_causes
            
            logger.info(f"检测到 {len(causal_relationships)} 组潜在因果关系")
            return causal_relationships
            
        except Exception as e:
            logger.error(f"检测因果关系失败: {str(e)}")
            return {}
    
    def _test_granger_causality(
        self, 
        target: pd.Series, 
        predictor: pd.Series, 
        max_lag: int
    ) -> bool:
        """简化的Granger因果检验"""
        try:
            # 清理数据
            df = pd.DataFrame({'target': target, 'predictor': predictor}).dropna()
            if len(df) < max_lag * 3:
                return False
            
            # 简单的滞后相关性检验
            significant_lags = 0
            for lag in range(1, max_lag + 1):
                if len(df) > lag:
                    lagged_predictor = df['predictor'].shift(lag)
                    current_target = df['target']
                    
                    # 移除NaN值
                    valid_data = pd.DataFrame({
                        'target': current_target,
                        'predictor': lagged_predictor
                    }).dropna()
                    
                    if len(valid_data) > 10:
                        corr, p_value = pearsonr(
                            valid_data['target'], 
                            valid_data['predictor']
                        )
                        
                        if abs(corr) > 0.3 and p_value < 0.05:
                            significant_lags += 1
            
            # 如果有多个显著滞后，认为存在因果关系
            return significant_lags >= 2
            
        except Exception:
            return False
    
    async def calculate_partial_correlations(
        self,
        metrics_data: Dict[str, pd.DataFrame]
    ) -> Dict[str, Dict[str, float]]:
        """计算偏相关系数"""
        try:
            combined_df = self._prepare_correlation_data(metrics_data)
            if combined_df.shape[1] < 3:
                return {}
            
            partial_correlations = {}
            metrics = list(combined_df.columns)
            
            for i, metric1 in enumerate(metrics):
                partial_correlations[metric1] = {}
                
                for j, metric2 in enumerate(metrics):
                    if i != j:
                        # 计算控制其他变量后的偏相关
                        control_vars = [m for m in metrics if m != metric1 and m != metric2]
                        
                        if control_vars:
                            partial_corr = self._calculate_partial_correlation(
                                combined_df, metric1, metric2, control_vars
                            )
                            
                            if not np.isnan(partial_corr) and abs(partial_corr) > 0.3:
                                partial_correlations[metric1][metric2] = round(partial_corr, 3)
            
            return partial_correlations
            
        except Exception as e:
            logger.error(f"计算偏相关系数失败: {str(e)}")
            return {}
    
    def _calculate_partial_correlation(
        self,
        df: pd.DataFrame,
        var1: str,
        var2: str,
        control_vars: List[str]
    ) -> float:
        """计算偏相关系数"""
        try:
            from sklearn.linear_model import LinearRegression
            
            # 准备数据
            clean_df = df[[var1, var2] + control_vars].dropna()
            if len(clean_df) < 10:
                return np.nan
            
            # 对var1回归控制变量
            reg1 = LinearRegression()
            reg1.fit(clean_df[control_vars], clean_df[var1])
            residual1 = clean_df[var1] - reg1.predict(clean_df[control_vars])
            
            # 对var2回归控制变量
            reg2 = LinearRegression()
            reg2.fit(clean_df[control_vars], clean_df[var2])
            residual2 = clean_df[var2] - reg2.predict(clean_df[control_vars])
            
            # 计算残差相关系数
            corr, _ = pearsonr(residual1, residual2)
            return corr
            
        except Exception:
            return np.nan