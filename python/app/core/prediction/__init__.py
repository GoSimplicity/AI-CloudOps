"""
预测模块
"""

from .predictor import PredictionService
from .model_loader import ModelLoader

__all__ = ["PredictionService", "ModelLoader"]