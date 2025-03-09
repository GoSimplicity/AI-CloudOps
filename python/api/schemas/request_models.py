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

from pydantic import BaseModel, Field, validator
from typing import List, Dict, Any, Optional, Union
from datetime import datetime

# 预测模块请求和响应模型

class ResourcePredictionRequest(BaseModel):
    """资源预测请求模型"""
    predictor_type: str = Field("timeseries", description="预测器类型，可选值为 timeseries, ml")
    model_dir: str = Field("./models/prediction", description="模型目录")
    model_name: Optional[str] = Field(None, description="模型名称，用于加载或保存模型")
    train_data: Optional[List[Dict[str, Any]]] = Field(None, description="训练数据")
    predict_data: List[Dict[str, Any]] = Field(..., description="预测数据")
    future_steps: int = Field(12, description="预测未来的时间点数量")
    save_model: bool = Field(False, description="是否保存模型")


class ResourcePredictionResponse(BaseModel):
    """资源预测响应模型"""
    predictions: List[Dict[str, Any]] = Field(..., description="预测结果")
    model_path: Optional[str] = Field(None, description="模型保存路径")
    status: str = Field(..., description="状态")
    message: str = Field(..., description="消息")


class FailurePredictionRequest(BaseModel):
    """故障预测请求模型"""
    predictor_type: str = Field("supervised", description="预测器类型，可选值为 supervised, unsupervised")
    model_dir: str = Field("./models/prediction", description="模型目录")
    model_name: Optional[str] = Field(None, description="模型名称，用于加载或保存模型")
    train_data: Optional[List[Dict[str, Any]]] = Field(None, description="训练数据")
    labels: Optional[List[int]] = Field(None, description="标签数据，用于监督学习")
    logs: Optional[List[Dict[str, Any]]] = Field(None, description="日志数据")
    traces: Optional[List[Dict[str, Any]]] = Field(None, description="链路追踪数据")
    predict_data: List[Dict[str, Any]] = Field(..., description="预测数据")
    predict_logs: Optional[List[Dict[str, Any]]] = Field(None, description="预测用的日志数据")
    predict_traces: Optional[List[Dict[str, Any]]] = Field(None, description="预测用的链路追踪数据")
    save_model: bool = Field(False, description="是否保存模型")


class FailurePredictionResponse(BaseModel):
    """故障预测响应模型"""
    predictions: List[Dict[str, Any]] = Field(..., description="预测结果")
    model_path: Optional[str] = Field(None, description="模型保存路径")
    status: str = Field(..., description="状态")
    message: str = Field(..., description="消息")


class ModelOptimizationRequest(BaseModel):
    """模型优化请求模型"""
    optimizer_type: str = Field(..., description="优化器类型，可选值为 timeseries, ml, failure")
    predictor_type: Optional[str] = Field(None, description="预测器类型，用于故障预测")
    model_dir: str = Field("./models/prediction", description="模型目录")
    data: List[Dict[str, Any]] = Field(..., description="训练数据")
    labels: Optional[List[int]] = Field(None, description="标签数据，用于监督学习")
    model_types: Optional[List[str]] = Field(None, description="要尝试的模型类型列表")
    cv_folds: Optional[int] = Field(5, description="交叉验证折数")
    metric: Optional[str] = Field(None, description="评估指标")
    metric_name: Optional[str] = Field(None, description="指标名称")


class ModelOptimizationResponse(BaseModel):
    """模型优化响应模型"""
    model_type: str = Field(..., description="最佳模型类型")
    model_path: str = Field(..., description="模型保存路径")
    best_params: Dict[str, Any] = Field(..., description="最佳参数")
    best_score: float = Field(..., description="最佳分数")
    status: str = Field(..., description="状态")
    message: str = Field(..., description="消息")

# 异常检测请求模型
class AnomalyDetectionRequest(BaseModel):
    """异常检测请求模型"""
    
    metrics: List[Dict[str, Any]] = Field(..., description="指标数据")
    logs: Optional[List[str]] = Field(None, description="日志数据")
    traces: Optional[List[Dict[str, Any]]] = Field(None, description="链路追踪数据")
    detector_type: str = Field("statistical", description="检测器类型: statistical, ml, 或 ensemble")
    
    @validator('metrics')
    def validate_metrics(cls, v):
        """验证指标数据格式"""
        if not v:
            raise ValueError("指标数据不能为空")
        
        # 检查指标数据格式
        for item in v:
            if 'timestamp' not in item:
                raise ValueError("每个指标数据点必须包含 timestamp 字段")
            if 'value' not in item and not any(k for k in item.keys() if k not in ['timestamp']):
                raise ValueError("每个指标数据点必须包含至少一个指标值")
        
        return v

# 根因分析请求模型
class RootCauseAnalysisRequest(BaseModel):
    """根因分析请求模型"""
    
    metrics: List[Dict[str, Any]] = Field(..., description="指标数据")
    logs: Optional[List[str]] = Field(None, description="日志数据")
    traces: Optional[List[Dict[str, Any]]] = Field(None, description="链路追踪数据")
    anomaly_time: datetime = Field(..., description="异常发生时间")
    service_name: str = Field(..., description="服务名称")
    analysis_type: str = Field("causal_graph", description="分析类型: causal_graph 或 fault_localization")
    
    @validator('metrics')
    def validate_metrics(cls, v):
        """验证指标数据格式"""
        if not v:
            raise ValueError("指标数据不能为空")
        
        return v