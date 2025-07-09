import logging
import datetime
import numpy as np
import pandas as pd
from typing import Optional, Dict, Any, List

from app.core.prediction.model_loader import ModelLoader
from app.services.prometheus import PrometheusService
from app.utils.time_utils import TimeUtils
from app.config.settings import config
from app.constants import (
    LOW_QPS_THRESHOLD, QPS_CONFIDENCE_THRESHOLDS, HOUR_FACTORS, DAY_FACTORS,
    MAX_PREDICTION_HOURS, DEFAULT_PREDICTION_HOURS, PREDICTION_VARIATION_FACTOR
)
from app.utils.error_handlers import (
    ErrorHandler, ServiceError, ValidationError,
    validate_field_range, validate_field_type
)

logger = logging.getLogger("aiops.predictor")

class PredictionService:
    """负载预测服务，基于机器学习模型预测实例需求"""
    
    def __init__(self):
        self.prometheus = PrometheusService()
        self.model_loader = ModelLoader()
        self.model_loaded = False
        self.scaler_loaded = False
        self.error_handler = ErrorHandler(logger)
        self._initialize()
    
    def _initialize(self):
        """初始化预测服务"""
        try:
            success = self.model_loader.load_models()
            if success and self.model_loader.validate_model():
                self.model_loaded = True
                self.scaler_loaded = True
                logger.info("预测服务初始化成功")
            else:
                logger.error("预测服务初始化失败")
        except Exception as e:
            logger.error(f"预测服务初始化异常: {str(e)}")
    
    async def predict(
        self, 
        current_qps: Optional[float] = None, 
        timestamp: Optional[datetime.datetime] = None,
        include_features: bool = False
    ) -> Optional[Dict[str, Any]]:
        """执行实例数预测"""
        try:
            if not self.is_healthy():
                logger.error("预测服务不健康，无法执行预测")
                return None
            
            # 获取当前时间
            if timestamp is None:
                timestamp = datetime.datetime.now()
            
            # 获取当前QPS
            if current_qps is None:
                current_qps = await self._get_current_qps()
                logger.info(f"从Prometheus获取当前QPS: {current_qps}")
            
            # 验证QPS值，确保非负
            if current_qps < 0:
                logger.warning(f"QPS值异常: {current_qps}, 使用0")
                current_qps = 0
            
            # QPS为0或极低值时直接返回最小实例数
            if current_qps == 0 or current_qps < LOW_QPS_THRESHOLD:
                logger.info(f"当前QPS({current_qps})低于阈值{LOW_QPS_THRESHOLD}，返回最小实例数: {config.prediction.min_instances}")
                return {
                    "instances": config.prediction.min_instances,
                    "current_qps": round(current_qps, 2),
                    "timestamp": timestamp.isoformat(),
                    "confidence": 0.95,
                    "model_version": self.model_loader.model_metadata.get("version", "1.0"),
                    "prediction_type": "threshold_based"
                }
            
            # 提取时间特征
            time_features = TimeUtils.extract_time_features(timestamp)
            
            # 准备历史QPS数据
            qps_1h_ago = await self._get_historical_qps(timestamp - datetime.timedelta(hours=1))
            qps_1d_ago = await self._get_historical_qps(timestamp - datetime.timedelta(days=1))
            qps_1w_ago = await self._get_historical_qps(timestamp - datetime.timedelta(weeks=1))
            
            # 确保历史QPS数据有效，否则使用合理的估计值
            qps_1h_ago = qps_1h_ago if qps_1h_ago is not None else max(0, current_qps * 0.9)
            qps_1d_ago = qps_1d_ago if qps_1d_ago is not None else max(0, current_qps * 1.0)
            qps_1w_ago = qps_1w_ago if qps_1w_ago is not None else max(0, current_qps * 1.0)
            
            # 计算QPS变化率
            qps_change = (current_qps - qps_1h_ago) / max(1.0, qps_1h_ago)  # 避免除零
            
            # 计算平均QPS
            recent_qps_data = await self._get_recent_qps_data(timestamp, hours=6)
            if recent_qps_data and len(recent_qps_data) > 0:
                qps_avg_6h = sum(recent_qps_data) / len(recent_qps_data)
            else:
                qps_avg_6h = current_qps  # 如果无法获取历史数据，使用当前值
            
            # 构建特征向量 - 支持新增特征
            features_dict = {
                "QPS": [current_qps],
                "sin_time": [time_features['sin_time']],
                "cos_time": [time_features['cos_time']],
                "sin_day": [time_features['sin_day']],
                "cos_day": [time_features['cos_day']],
                "is_business_hour": [int(time_features['is_business_hour'])],
                "is_weekend": [int(time_features['is_weekend'])],
                "QPS_1h_ago": [qps_1h_ago],
                "QPS_1d_ago": [qps_1d_ago],
                "QPS_1w_ago": [qps_1w_ago],
                "QPS_change": [qps_change],
                "QPS_avg_6h": [qps_avg_6h]
            }
            
            # 创建DataFrame
            try:
                features = pd.DataFrame(features_dict)
                
                # 检查是否需要添加额外特征以匹配模型期望
                model_features = self.model_loader.model_metadata.get('features', [])
                for feature in model_features:
                    if feature not in features.columns:
                        logger.warning(f"模型期望特征 '{feature}' 不在当前特征集中，添加默认值0")
                        features[feature] = 0.0
                
                # 确保特征顺序匹配
                features = features[model_features]
                
            except Exception as e:
                logger.error(f"创建特征DataFrame失败: {str(e)}")
                return None
            
            # 标准化特征
            try:
                features_scaled = self.model_loader.scaler.transform(features)
            except Exception as e:
                logger.error(f"特征标准化失败: {str(e)}")
                return None
            
            # 执行预测
            try:
                prediction = self.model_loader.model.predict(features_scaled)[0]
            except Exception as e:
                logger.error(f"模型预测失败: {str(e)}")
                return None
            
            # 限制实例数范围并四舍五入（实例数应为整数）
            instances = int(np.clip(
                np.round(prediction), 
                config.prediction.min_instances, 
                config.prediction.max_instances
            ))
            
            # 计算置信度
            confidence = self._calculate_confidence(current_qps, time_features, prediction)
            
            logger.info(f"预测完成: QPS={current_qps:.2f}, 实例数={instances}, 置信度={confidence:.2f}")
            
            result = {
                "instances": instances,
                "current_qps": round(current_qps, 2),
                "timestamp": timestamp.isoformat(),
                "confidence": confidence,
                "model_version": self.model_loader.model_metadata.get("version", "1.0"),
                "prediction_type": "model_based"
            }
            
            # 包含特征信息
            if include_features:
                result["features"] = {
                    "qps": current_qps,
                    "sin_time": time_features['sin_time'],
                    "cos_time": time_features['cos_time'],
                    "hour": time_features['hour'],
                    "is_business_hour": time_features['is_business_hour'],
                    "is_weekend": time_features['is_weekend'],
                    "sin_day": features_dict["sin_day"][0],
                    "cos_day": features_dict["cos_day"][0],
                    "qps_1h_ago": qps_1h_ago,
                    "qps_1d_ago": qps_1d_ago,
                    "qps_1w_ago": qps_1w_ago,
                    "qps_change": qps_change,
                    "qps_avg_6h": qps_avg_6h
                }
            
            return result
            
        except Exception as e:
            logger.error(f"预测失败: {str(e)}")
            return None
    
    async def _get_current_qps(self) -> float:
        """从Prometheus获取当前QPS"""
        try:
            query = config.prediction.prometheus_query
            
            result = await self.prometheus.query_instant(query)
            
            if result and len(result) > 0:
                qps = float(result[0]['value'][1])
                logger.debug(f"从Prometheus获取QPS: {qps}")
                return max(0, qps)  # 确保非负
            else:
                logger.warning(f"未能从Prometheus获取QPS，使用默认值0: {query}")
                return 0.0
                
        except Exception as e:
            logger.error(f"获取QPS失败: {str(e)}")
            return 0.0
    
    async def _get_historical_qps(self, timestamp: datetime.datetime) -> Optional[float]:
        """获取指定时间点的历史QPS数据"""
        try:
            query = config.prediction.prometheus_query
            
            # 查询指定时间点的QPS
            result = await self.prometheus.query_instant(query, timestamp)
            
            if result and len(result) > 0:
                qps = float(result[0]['value'][1])
                logger.debug(f"从Prometheus获取历史QPS ({timestamp.isoformat()}): {qps}")
                return max(0, qps)  # 确保非负
            else:
                logger.warning(f"未能从Prometheus获取历史QPS ({timestamp.isoformat()})")
                return None
                
        except Exception as e:
            logger.error(f"获取历史QPS失败 ({timestamp.isoformat()}): {str(e)}")
            return None
    
    async def _get_recent_qps_data(self, end_time: datetime.datetime, hours: int = 6) -> List[float]:
        """获取最近几小时的QPS数据"""
        try:
            query = config.prediction.prometheus_query
            start_time = end_time - datetime.timedelta(hours=hours)
            
            # 使用范围查询获取一段时间内的QPS数据
            df = await self.prometheus.query_range(
                query=query,
                start_time=start_time,
                end_time=end_time,
                step="30m"  # 每30分钟一个数据点
            )
            
            if df is not None and not df.empty and 'value' in df.columns:
                # 转换为浮点数列表
                values = df['value'].tolist()
                logger.debug(f"获取到{len(values)}个历史QPS数据点")
                return [max(0, float(v)) for v in values]  # 确保所有值非负
            else:
                logger.warning(f"未能从Prometheus获取最近{hours}小时的QPS数据")
                return []
                
        except Exception as e:
            logger.error(f"获取最近QPS数据失败: {str(e)}")
            return []
    
    def _calculate_confidence(
        self, 
        qps: float, 
        time_features: dict, 
        prediction: float
    ) -> float:
        """计算预测置信度"""
        try:
            confidence_factors = []
            
            # 基于QPS值的稳定性
            if qps <= 100:
                qps_confidence = 0.9
            elif qps <= 500:
                qps_confidence = 0.8
            elif qps <= 1000:
                qps_confidence = 0.7
            else:
                qps_confidence = 0.6
            confidence_factors.append(qps_confidence)
            
            # 基于时间特征的稳定性
            hour = time_features.get('hour', 12)
            is_weekend = time_features.get('is_weekend', False)
            is_holiday = time_features.get('is_holiday', False)
            
            # 工作日/周末/节假日判断
            if is_holiday:
                time_confidence = 0.7  # 节假日预测较不稳定
            elif is_weekend:
                if 10 <= hour <= 20:  # 周末白天
                    time_confidence = 0.75
                else:  # 周末夜间
                    time_confidence = 0.85
            else:
                if time_features.get('is_business_hour', False):
                    time_confidence = 0.9  # 工作时间预测相对稳定
                elif 22 <= hour or hour <= 6:
                    time_confidence = 0.85  # 深夜时间比较稳定
                else:
                    time_confidence = 0.8  # 其他时间
            confidence_factors.append(time_confidence)
            
            # 基于模型元数据的稳定性
            model_age_days = self._get_model_age_days()
            if model_age_days <= 7:
                model_confidence = 0.95  # 新模型可信度高
            elif model_age_days <= 30:
                model_confidence = 0.85  # 较新模型可信度较高
            elif model_age_days <= 90:
                model_confidence = 0.75  # 中等年龄模型
            else:
                model_confidence = 0.65  # 旧模型可信度较低
            confidence_factors.append(model_confidence)
            
            # 计算综合置信度
            confidence = sum(confidence_factors) / len(confidence_factors)
            
            # 低流量场景下有更高的置信度（因为规则更简单）
            if qps < 5.0:
                confidence = max(confidence, 0.95)
            
            return round(confidence, 2)
        except Exception as e:
            logger.error(f"计算置信度失败: {str(e)}")
            return 0.8  # 默认中等置信度
    
    def _get_model_age_days(self) -> int:
        """获取模型的年龄（天数）"""
        try:
            created_at = self.model_loader.model_metadata.get("created_at")
            if not created_at:
                return 999  # 未知年龄，假设很旧
                
            created_date = datetime.datetime.fromisoformat(created_at)
            age_days = (datetime.datetime.now() - created_date).days
            return max(0, age_days)
        except:
            return 30  # 解析失败，返回默认值
    
    async def predict_trend(
        self, 
        hours_ahead: int = 24,
        current_qps: Optional[float] = None
    ) -> Optional[Dict[str, Any]]:
        """预测未来QPS趋势和实例数需求"""
        try:
            if not self.is_healthy():
                logger.error("预测服务不健康，无法执行趋势预测")
                return None
            
            # 限制预测时长
            hours_ahead = min(168, max(1, hours_ahead))  # 限制在1-168小时(一周)内
            
            # 获取当前QPS和时间
            now = datetime.datetime.now()
            if current_qps is None:
                current_qps = await self._get_current_qps()
            
            # 验证QPS值
            if current_qps < 0:
                logger.warning(f"QPS值异常: {current_qps}, 使用0")
                current_qps = 0
                
            # 获取历史QPS数据，用于预测趋势
            historical_data = await self._get_recent_qps_data(now, hours=24)
            
            # 预测每小时的QPS和实例数
            forecast = []
            predicted_qps = current_qps
            
            for hour in range(hours_ahead):
                # 预测时间点
                future_time = now + datetime.timedelta(hours=hour)
                
                # 提取时间特征
                time_features = TimeUtils.extract_time_features(future_time)
                
                # 应用简单的时间模式
                hour_factor = self._get_hour_factor(future_time.hour)
                day_factor = self._get_day_factor(future_time.weekday())
                
                if len(historical_data) > 0:
                    # 使用历史模式进行预测
                    base_qps = sum(historical_data) / len(historical_data)
                    time_pattern = hour_factor * day_factor
                    predicted_qps = base_qps * time_pattern
                    
                    # 添加一些随机波动（实际模型应该更复杂）
                    variation = 0.1  # 10%的波动
                    predicted_qps *= (1 + (np.random.random() - 0.5) * variation)
                else:
                    # 如果没有历史数据，使用当前QPS和时间模式
                    time_pattern = hour_factor * day_factor
                    predicted_qps = current_qps * time_pattern
                
                # 确保QPS非负
                predicted_qps = max(0, predicted_qps)
                
                # 基于预测的QPS计算实例数
                prediction_result = await self.predict(
                    current_qps=predicted_qps,
                    timestamp=future_time
                )
                
                instances = prediction_result.get('instances', config.prediction.min_instances) if prediction_result else config.prediction.min_instances
                
                forecast.append({
                    "timestamp": future_time.isoformat(),
                    "qps": round(predicted_qps, 2),
                    "instances": instances
                })
            
            return {
                "forecast": forecast,
                "current_qps": current_qps,
                "hours_ahead": hours_ahead,
                "timestamp": now.isoformat()
            }
            
        except Exception as e:
            logger.error(f"趋势预测失败: {str(e)}")
            return None
    
    def _get_hour_factor(self, hour: int) -> float:
        """根据小时获取QPS乘数"""
        return HOUR_FACTORS.get(hour, 0.5)  # 默认为0.5
    
    def _get_day_factor(self, day_of_week: int) -> float:
        """根据星期几获取QPS乘数"""
        return DAY_FACTORS.get(day_of_week, 1.0)  # 默认为1.0
    
    def is_healthy(self) -> bool:
        """检查预测服务健康状态"""
        return self.model_loaded and self.scaler_loaded
    
    def get_service_info(self) -> Dict[str, Any]:
        """获取服务信息"""
        model_info = self.model_loader.get_model_info()
        return {
            "healthy": self.is_healthy(),
            "model_loaded": self.model_loaded,
            "scaler_loaded": self.scaler_loaded,
            "model_info": model_info,
            "model_age_days": self._get_model_age_days()
        }
    
    async def reload_models(self) -> bool:
        """重新加载模型"""
        logger.info("重新加载预测模型...")
        try:
            success = self.model_loader.reload_models()
            if success:
                self.model_loaded = True
                self.scaler_loaded = True
                logger.info("模型重新加载成功")
            else:
                logger.error("模型重新加载失败")
            return success
        except Exception as e:
            logger.error(f"重新加载模型失败: {str(e)}")
            self.model_loaded = False
            self.scaler_loaded = False
            return False