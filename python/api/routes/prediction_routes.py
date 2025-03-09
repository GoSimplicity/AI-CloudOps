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

from fastapi import APIRouter, HTTPException, Depends, Query, Body
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
import json

from core.prediction.resource_prediction import create_predictor
from core.prediction.failure_prediction import create_failure_predictor
from core.prediction.model_optimization import create_optimizer
from data.collectors.prometheus_collector import PrometheusCollector
from utils.logger import get_logger
from api.schemas.request_models import (
    ResourcePredictionRequest,
    ResourcePredictionResponse,
    FailurePredictionRequest,
    FailurePredictionResponse,
    ModelOptimizationRequest,
    ModelOptimizationResponse
)

logger = get_logger("prediction_routes")

router = APIRouter(prefix="/prediction", tags=["prediction"])

# 创建Prometheus收集器实例
prometheus_collector = PrometheusCollector()

@router.post("/resource", response_model=ResourcePredictionResponse)
async def predict_resource(request: ResourcePredictionRequest):
    """
    预测资源使用情况
    
    Args:
        request: 资源预测请求
        
    Returns:
        资源预测结果
    """
    try:
        # 创建预测器
        predictor = create_predictor(
            predictor_type=request.predictor_type,
            model_dir=request.model_dir
        )
        
        # 如果提供了模型名称，则加载模型
        if request.model_name:
            load_success = predictor.load_model(request.model_name)
            if not load_success:
                raise HTTPException(status_code=404, detail=f"模型 {request.model_name} 不存在")
        
        # 如果提供了训练数据，则训练模型
        if request.train_data:
            train_result = predictor.train(request.train_data)
            logger.info(f"模型训练结果: {train_result}")
        
        # 预测未来资源使用
        predictions = predictor.predict(
            request.predict_data,
            future_steps=request.future_steps
        )
        
        # 如果需要保存模型
        model_path = None
        if request.save_model and request.model_name:
            model_path = predictor.save_model(request.model_name)
        
        return ResourcePredictionResponse(
            predictions=predictions,
            model_path=model_path,
            status="success",
            message="资源预测成功"
        )
    
    except Exception as e:
        logger.error(f"资源预测失败: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"资源预测失败: {str(e)}")


@router.post("/failure", response_model=FailurePredictionResponse)
async def predict_failure(request: FailurePredictionRequest):
    """
    预测系统故障
    
    Args:
        request: 故障预测请求
        
    Returns:
        故障预测结果
    """
    try:
        # 记录请求信息
        logger.info(f"收到故障预测请求: predictor_type={request.predictor_type}, model_name={request.model_name}")
        
        # 创建预测器
        predictor = create_failure_predictor(
            predictor_type=request.predictor_type,
            model_dir=request.model_dir
        )
        
        # 如果提供了模型名称，则加载模型
        if request.model_name:
            logger.info(f"尝试加载模型: {request.model_name}")
            load_success = predictor.load_model(request.model_name)
            if not load_success:
                logger.error(f"模型 {request.model_name} 不存在")
                raise HTTPException(status_code=404, detail=f"模型 {request.model_name} 不存在")
            logger.info(f"模型 {request.model_name} 加载成功")
        
        # 如果提供了训练数据，则训练模型
        if request.train_data:
            logger.info(f"开始训练模型，数据点数量: {len(request.train_data)}")
            train_result = predictor.train(
                request.train_data,
                labels=request.labels,
                logs=request.logs,
                traces=request.traces
            )
            logger.info(f"模型训练结果: {train_result}")
        
        # 预测故障
        logger.info(f"开始预测故障，数据点数量: {len(request.predict_data)}")
        predictions = predictor.predict(
            request.predict_data,
            logs=request.predict_logs,
            traces=request.predict_traces
        )
        logger.info(f"故障预测完成，预测结果数量: {len(predictions)}")
        
        # 如果需要保存模型
        model_path = None
        if request.save_model and request.model_name:
            logger.info(f"保存模型: {request.model_name}")
            model_path = predictor.save_model(request.model_name)
        
        return FailurePredictionResponse(
            predictions=predictions,
            model_path=model_path,
            status="success",
            message="故障预测成功"
        )
    
    except Exception as e:
        logger.error(f"故障预测失败: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"故障预测失败: {str(e)}")


@router.post("/optimize", response_model=ModelOptimizationResponse)
async def optimize_model(request: ModelOptimizationRequest):
    """
    优化预测模型
    
    Args:
        request: 模型优化请求
        
    Returns:
        模型优化结果
    """
    try:
        # 创建优化器
        optimizer = create_optimizer(
            optimizer_type=request.optimizer_type,
            model_dir=request.model_dir
        )
        
        # 执行模型优化
        if request.optimizer_type == "failure" and request.predictor_type == "supervised":
            # 监督学习故障预测需要标签
            if not request.labels:
                raise HTTPException(status_code=400, detail="监督学习故障预测需要提供标签数据")
            
            result = optimizer.optimize(
                request.data,
                labels=request.labels,
                predictor_type=request.predictor_type,
                model_types=request.model_types,
                cv_folds=request.cv_folds,
                metric=request.metric
            )
        else:
            # 其他类型的优化
            result = optimizer.optimize(
                request.data,
                model_types=request.model_types,
                cv_folds=request.cv_folds,
                metric=request.metric,
                metric_name=request.metric_name
            )
        
        return ModelOptimizationResponse(
            model_type=result["model_type"],
            model_path=result["model_path"],
            best_params=result["best_params"],
            best_score=result["best_score"],
            status="success",
            message="模型优化成功"
        )
    
    except Exception as e:
        logger.error(f"模型优化失败: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"模型优化失败: {str(e)}")


@router.get("/metrics")
async def get_metrics(
    metric_names: List[str] = Query(..., description="指标名称列表"),
    start_time: datetime = Query(..., description="开始时间"),
    end_time: datetime = Query(..., description="结束时间"),
    step: str = Query("5m", description="时间步长")
):
    """
    从Prometheus获取指标数据
    
    Args:
        metric_names: 指标名称列表
        start_time: 开始时间
        end_time: 结束时间
        step: 时间步长
        
    Returns:
        指标数据
    """
    try:
        result = {}
        
        for metric_name in metric_names:
            metrics = prometheus_collector.collect_metrics(
                metric_name=metric_name,
                start_time=start_time,
                end_time=end_time,
                step=step
            )
            
            result[metric_name] = metrics
        
        return {
            "metrics": result,
            "status": "success",
            "message": "指标数据获取成功"
        }
    
    except Exception as e:
        logger.error(f"指标数据获取失败: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"指标数据获取失败: {str(e)}")


@router.post("/resource/batch")
async def batch_predict_resource(
    metric_names: List[str] = Query(..., description="指标名称列表"),
    start_time: datetime = Query(..., description="开始时间"),
    end_time: datetime = Query(..., description="结束时间"),
    step: str = Query("5m", description="时间步长"),
    future_steps: int = Query(12, description="预测未来的时间点数量"),
    predictor_type: str = Query("timeseries", description="预测器类型")
):
    """
    批量预测多个资源指标
    
    Args:
        metric_names: 指标名称列表
        start_time: 开始时间
        end_time: 结束时间
        step: 时间步长
        future_steps: 预测未来的时间点数量
        predictor_type: 预测器类型
        
    Returns:
        批量预测结果
    """
    try:
        result = {}
        
        for metric_name in metric_names:
            # 从Prometheus获取指标数据
            metrics = prometheus_collector.collect_metrics(
                metric_name=metric_name,
                start_time=start_time,
                end_time=end_time,
                step=step
            )
            
            # 转换为预测器需要的格式
            processed_data = []
            for metric in metrics:
                processed_data.append({
                    "timestamp": datetime.fromtimestamp(metric["timestamp"]),
                    "value": float(metric["value"]),
                    "metric_name": metric_name
                })
            
            # 创建预测器
            predictor = create_predictor(predictor_type=predictor_type)
            
            # 训练模型
            predictor.train(processed_data)
            
            # 预测未来值
            predictions = predictor.predict(processed_data[-30:], future_steps=future_steps)
            
            result[metric_name] = predictions
        
        return {
            "predictions": result,
            "status": "success",
            "message": "批量资源预测成功"
        }
    
    except Exception as e:
        logger.error(f"批量资源预测失败: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"批量资源预测失败: {str(e)}")