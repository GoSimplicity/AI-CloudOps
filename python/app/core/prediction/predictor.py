import logging
import datetime
import numpy as np
import pandas as pd
from typing import Optional, Dict, Any
from app.core.prediction.model_loader import ModelLoader
from app.services.prometheus import PrometheusService
from app.utils.time_utils import TimeUtils
from app.config.settings import config

logger = logging.getLogger("aiops.predictor")

class PredictionService:
    def __init__(self):
        self.prometheus = PrometheusService()
        self.model_loader = ModelLoader()
        self.model_loaded = False
        self.scaler_loaded = False
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
            
            # 验证QPS值
            if current_qps < 0:
                logger.warning(f"QPS值异常: {current_qps}, 使用0")
                current_qps = 0
            
            # 提取时间特征
            time_features = TimeUtils.extract_time_features(timestamp)
            
            # 构建特征向量
            features = pd.DataFrame({
                "QPS": [current_qps],
                "sin_time": [time_features['sin_time']],
                "cos_time": [time_features['cos_time']]
            })
            
            # 标准化特征
            features_scaled = self.model_loader.scaler.transform(features)
            
            # 执行预测
            prediction = self.model_loader.model.predict(features_scaled)[0]
            
            # 限制实例数范围
            instances = int(np.clip(
                prediction, 
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
                "model_version": self.model_loader.model_metadata.get("version", "1.0")
            }
            
            # 包含特征信息
            if include_features:
                result["features"] = {
                    "qps": current_qps,
                    "sin_time": time_features['sin_time'],
                    "cos_time": time_features['cos_time'],
                    "hour": time_features['hour'],
                    "is_business_hour": time_features['is_business_hour'],
                    "is_weekend": time_features['is_weekend']
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
                logger.warning("未能从Prometheus获取QPS，使用默认值0")
                return 0.0
                
        except Exception as e:
            logger.error(f"获取QPS失败: {str(e)}")
            return 0.0
    
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
            if time_features.get('is_business_hour', False):
                time_confidence = 0.8  # 工作时间预测相对稳定
            elif 22 <= hour or hour <= 6:
                time_confidence = 0.9  # 深夜时间比较稳定
            else:
                time_confidence = 0.7
            confidence_factors.append(time_confidence)
            
            # 基于预测值的合理性
            if config.prediction.min_instances <= prediction <= config.prediction.max_instances:
                pred_confidence = 0.9
            else:
                pred_confidence = 0.5  # 预测值超出合理范围
            confidence_factors.append(pred_confidence)
            
            # 基于模型元数据
            model_age_days = self._get_model_age_days()
            if model_age_days <= 30:
                model_confidence = 0.9
            elif model_age_days <= 90:
                model_confidence = 0.8
            else:
                model_confidence = 0.6  # 模型较旧
            confidence_factors.append(model_confidence)
            
            # 计算综合置信度
            final_confidence = np.mean(confidence_factors)
            
            return round(final_confidence, 2)
            
        except Exception as e:
            logger.error(f"计算置信度失败: {str(e)}")
            return 0.5
    
    def _get_model_age_days(self) -> int:
        """获取模型年龄（天数）"""
        try:
            created_at = self.model_loader.model_metadata.get("created_at")
            if created_at:
                created_date = datetime.datetime.fromisoformat(created_at.replace('Z', '+00:00'))
                age = (datetime.datetime.now() - created_date).days
                return age
            return 365  # 默认认为模型较旧
        except Exception:
            return 365
    
    async def predict_trend(
        self, 
        hours_ahead: int = 24,
        current_qps: Optional[float] = None
    ) -> Optional[Dict[str, Any]]:
        """预测未来趋势"""
        try:
            if not self.is_healthy():
                return None
            
            if current_qps is None:
                current_qps = await self._get_current_qps()
            
            now = datetime.datetime.now()
            predictions = []
            
            for hour in range(hours_ahead):
                future_time = now + datetime.timedelta(hours=hour)
                
                # 简单的QPS趋势模拟（实际应用中可能需要更复杂的模型）
                time_factor = np.sin(2 * np.pi * hour / 24)  # 24小时周期
                projected_qps = max(0, current_qps * (1 + 0.1 * time_factor))
                
                prediction_result = await self.predict(
                    current_qps=projected_qps,
                    timestamp=future_time
                )
                
                if prediction_result:
                    predictions.append({
                        "timestamp": future_time.isoformat(),
                        "projected_qps": round(projected_qps, 2),
                        "predicted_instances": prediction_result["instances"],
                        "confidence": prediction_result["confidence"]
                    })
            
            if predictions:
                return {
                    "trend_predictions": predictions,
                    "summary": {
                        "max_instances": max(p["predicted_instances"] for p in predictions),
                        "min_instances": min(p["predicted_instances"] for p in predictions),
                        "avg_instances": round(np.mean([p["predicted_instances"] for p in predictions]), 1),
                        "avg_confidence": round(np.mean([p["confidence"] for p in predictions]), 2)
                    }
                }
            
            return None
            
        except Exception as e:
            logger.error(f"趋势预测失败: {str(e)}")
            return None
    
    def is_healthy(self) -> bool:
        """检查预测服务健康状态"""
        return self.model_loaded and self.scaler_loaded and self.model_loader.is_model_loaded()
    
    def get_service_info(self) -> Dict[str, Any]:
        """获取服务信息"""
        return {
            "healthy": self.is_healthy(),
            "model_loaded": self.model_loaded,
            "scaler_loaded": self.scaler_loaded,
            "model_info": self.model_loader.get_model_info(),
            "config": {
                "min_instances": config.prediction.min_instances,
                "max_instances": config.prediction.max_instances,
                "prometheus_query": config.prediction.prometheus_query
            }
        }
    
    async def reload_models(self) -> bool:
        """重新加载模型"""
        try:
            logger.info("重新加载预测模型...")
            success = self.model_loader.reload_models()
            if success:
                self.model_loaded = True
                self.scaler_loaded = True
                logger.info("模型重新加载成功")
            else:
                self.model_loaded = False
                self.scaler_loaded = False
                logger.error("模型重新加载失败")
            return success
        except Exception as e:
            logger.error(f"重新加载模型异常: {str(e)}")
            return False