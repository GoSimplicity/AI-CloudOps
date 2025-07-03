import logging
import joblib
import os
from typing import Optional, Tuple
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
from app.config.settings import config

logger = logging.getLogger("aiops.model_loader")

class ModelLoader:
    def __init__(self):
        self.model = None
        self.scaler = None
        self.model_metadata = {}
        logger.info("模型加载器初始化完成")
    
    def load_models(self) -> bool:
        """加载预测模型和标准化器"""
        try:
            # 获取模型文件的绝对路径
            model_path = os.path.abspath(config.prediction.model_path)
            scaler_path = os.path.abspath(config.prediction.scaler_path)
            
            # 检查模型文件是否存在
            if not os.path.exists(model_path):
                logger.error(f"模型文件不存在: {model_path}")
                return False
            
            if not os.path.exists(scaler_path):
                logger.error(f"标准化器文件不存在: {scaler_path}")
                return False
            
            # 加载模型
            self.model = joblib.load(model_path)
            logger.info(f"成功加载预测模型: {model_path}")
            
            # 加载标准化器
            self.scaler = joblib.load(scaler_path)
            logger.info(f"成功加载数据标准化器: {scaler_path}")
            
            # 加载模型元数据
            self._load_model_metadata()
            
            return True
            
        except Exception as e:
            logger.error(f"加载模型失败: {str(e)}")
            self.model = None
            self.scaler = None
            return False
    
    def _load_model_metadata(self):
        """加载模型元数据"""
        try:
            metadata_path = config.prediction.model_path.replace('.pkl', '_metadata.json')
            metadata_path = os.path.abspath(metadata_path)
            
            if os.path.exists(metadata_path):
                import json
                with open(metadata_path, 'r') as f:
                    self.model_metadata = json.load(f)
                logger.info("成功加载模型元数据")
            else:
                # 默认元数据
                self.model_metadata = {
                    "version": "1.0",
                    "created_at": datetime.now().isoformat(),
                    "features": ["QPS", "sin_time", "cos_time"],
                    "target": "instances",
                    "algorithm": "unknown"
                }
                logger.warning("未找到模型元数据，使用默认值")
        except Exception as e:
            logger.error(f"加载模型元数据失败: {str(e)}")
            self.model_metadata = {}
    
    def is_model_loaded(self) -> bool:
        """检查模型是否已加载"""
        return self.model is not None and self.scaler is not None
    
    def get_model_info(self) -> dict:
        """获取模型信息"""
        return {
            "loaded": self.is_model_loaded(),
            "metadata": self.model_metadata,
            "model_path": config.prediction.model_path,
            "scaler_path": config.prediction.scaler_path
        }
    
    def validate_model(self) -> bool:
        """验证模型有效性"""
        if not self.is_model_loaded():
            return False
        
        try:
            # 获取模型所需的特征列表
            model_features = self.model_metadata.get('features', [])
            
            # 创建测试数据 - 包含模型训练时使用的所有特征
            test_features_dict = {
                "QPS": [10.0],
                "sin_time": [0.5],
                "cos_time": [0.8]
            }
            
            # 为新版本模型添加额外特征
            extra_features = {
                "sin_day": [0.7],
                "cos_day": [0.7],
                "is_business_hour": [1],
                "is_weekend": [0],
                "is_holiday": [0],
                "QPS_1h_ago": [9.0],
                "QPS_1d_ago": [11.0],
                "QPS_1w_ago": [10.5],
                "QPS_change": [0.1],
                "QPS_avg_6h": [9.8]
            }
            
            # 根据模型所需特征添加额外特征
            for feature in model_features:
                if feature not in test_features_dict and feature in extra_features:
                    test_features_dict[feature] = extra_features[feature]
            
            test_features = pd.DataFrame(test_features_dict)
            
            # 测试标准化
            scaled_features = self.scaler.transform(test_features)
            
            # 测试预测
            prediction = self.model.predict(scaled_features)
            
            # 验证预测结果
            if len(prediction) != 1 or not isinstance(prediction[0], (int, float, np.integer, np.floating)):
                logger.error("模型预测结果格式错误")
                return False
            
            if prediction[0] < 0 or prediction[0] > 1000:  # 合理范围检查
                logger.warning(f"模型预测结果可能异常: {prediction[0]}")
            
            logger.info("模型验证通过")
            return True
            
        except Exception as e:
            logger.error(f"模型验证失败: {str(e)}")
            return False
    
    def reload_models(self) -> bool:
        """重新加载模型"""
        logger.info("重新加载模型...")
        self.model = None
        self.scaler = None
        self.model_metadata = {}
        return self.load_models()
    
    def save_model_metadata(self, metadata: dict):
        """保存模型元数据"""
        try:
            metadata_path = config.prediction.model_path.replace('.pkl', '_metadata.json')
            import json
            with open(metadata_path, 'w') as f:
                json.dump(metadata, f, indent=2)
            logger.info("模型元数据保存成功")
        except Exception as e:
            logger.error(f"保存模型元数据失败: {str(e)}")