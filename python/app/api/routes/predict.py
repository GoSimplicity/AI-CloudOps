from flask import Blueprint, request, jsonify
import datetime
import logging
import asyncio
from app.core.prediction.predictor import PredictionService
from app.models.request_models import PredictionRequest
from app.models.response_models import PredictionResponse, APIResponse
from app.utils.validators import validate_qps

logger = logging.getLogger("aiops.predict")

predict_bp = Blueprint('predict', __name__)

# 初始化预测服务
prediction_service = PredictionService()

@predict_bp.route('/predict', methods=['GET', 'POST'])
def predict_instances():
    """预测实例数接口"""
    try:
        # 处理请求参数
        if request.method == 'POST':
            data = request.get_json() or {}
            try:
                predict_request = PredictionRequest(**data)
            except Exception as e:
                return jsonify(APIResponse(code=400, message=f"请求参数错误: {str(e)}", data={}).dict()), 400
        else:
            # GET请求使用默认参数
            predict_request = PredictionRequest()
        
        # 验证QPS参数
        if predict_request.current_qps is not None:
            if not validate_qps(predict_request.current_qps):
                return jsonify(APIResponse(code=400, message="QPS参数无效", data={}).dict()), 400
        
        logger.info(f"收到预测请求: QPS={predict_request.current_qps}, 时间={predict_request.timestamp}")
        
        # 执行预测
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            result = loop.run_until_complete(
                prediction_service.predict(
                    current_qps=predict_request.current_qps,
                    timestamp=predict_request.timestamp,
                    include_features=predict_request.include_confidence
                )
            )
        finally:
            loop.close()
        
        if result is None:
            return jsonify(APIResponse(code=500, message="预测失败，模型未加载或服务异常", data={}).dict()), 500
        
        # 构建响应
        response = PredictionResponse(
            instances=result['instances'],
            current_qps=result['current_qps'],
            timestamp=result['timestamp'],
            confidence=result.get('confidence') if predict_request.include_confidence else None,
            model_version=result.get('model_version'),
            prediction_type=result.get('prediction_type', 'model_based'),
            features=result.get('features')
        )
        
        logger.info(f"预测完成: 实例数={response.instances}, QPS={response.current_qps}, 置信度={response.confidence}")
        
        return jsonify(APIResponse(code=0, message="预测成功", data=response.dict()).dict())
        
    except Exception as e:
        logger.error(f"预测请求失败: {str(e)}")
        return jsonify(APIResponse(code=500, message=f"预测失败: {str(e)}", data={}).dict()), 500

@predict_bp.route('/predict/trend', methods=['GET', 'POST'])
def predict_trend():
    """预测未来趋势"""
    try:
        if request.method == 'POST':
            data = request.get_json() or {}
            hours_ahead = data.get('hours_ahead', 24)
            current_qps = data.get('current_qps')
        else:
            # 处理GET请求的查询参数
            hours_ahead = request.args.get('hours', type=int, default=24)
            current_qps = request.args.get('qps', type=float)
        
        # 验证参数
        if not isinstance(hours_ahead, int) or hours_ahead < 1 or hours_ahead > 168:  # 最多一周
            return jsonify(APIResponse(code=400, message="hours_ahead参数必须在1-168之间", data={}).dict()), 400
        
        if current_qps is not None and not validate_qps(current_qps):
            return jsonify(APIResponse(code=400, message="QPS参数无效", data={}).dict()), 400
        
        logger.info(f"收到趋势预测请求: 未来{hours_ahead}小时, QPS={current_qps}")
        
        # 执行趋势预测
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            result = loop.run_until_complete(
                prediction_service.predict_trend(
                    hours_ahead=hours_ahead,
                    current_qps=current_qps
                )
            )
        finally:
            loop.close()
        
        if result is None:
            return jsonify(APIResponse(code=500, message="趋势预测失败", data={}).dict()), 500
        
        logger.info(f"趋势预测完成: {len(result.get('forecast', []))} 个预测点")
        
        return jsonify(APIResponse(code=0, message="趋势预测成功", data=result).dict())
        
    except Exception as e:
        logger.error(f"趋势预测失败: {str(e)}")
        return jsonify(APIResponse(code=500, message=f"趋势预测失败: {str(e)}", data={}).dict()), 500

@predict_bp.route('/predict/models/reload', methods=['POST'])
def reload_models():
    """重新加载预测模型"""
    try:
        logger.info("收到模型重新加载请求")
        
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            success = loop.run_until_complete(prediction_service.reload_models())
        finally:
            loop.close()
        
        if success:
            logger.info("模型重新加载成功")
            return jsonify(APIResponse(
                code=0, 
                message="模型重新加载成功", 
                data={
                    "timestamp": datetime.datetime.utcnow().isoformat(),
                    "model_info": prediction_service.get_service_info()
                }
            ).dict())
        else:
            logger.error("模型重新加载失败")
            return jsonify(APIResponse(
                code=500, 
                message="模型重新加载失败", 
                data={"timestamp": datetime.datetime.utcnow().isoformat()}
            ).dict()), 500
        
    except Exception as e:
        logger.error(f"模型重新加载异常: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"模型重新加载异常: {str(e)}", 
            data={"timestamp": datetime.datetime.utcnow().isoformat()}
        ).dict()), 500

@predict_bp.route('/predict/info', methods=['GET'])
def prediction_info():
    """获取预测服务信息"""
    try:
        service_info = prediction_service.get_service_info()
        return jsonify(APIResponse(
            code=0, 
            message="获取预测服务信息成功", 
            data={
                "timestamp": datetime.datetime.utcnow().isoformat(),
                "service_info": service_info
            }
        ).dict())
        
    except Exception as e:
        logger.error(f"获取预测服务信息失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"获取服务信息失败: {str(e)}", 
            data={"timestamp": datetime.datetime.utcnow().isoformat()}
        ).dict()), 500

@predict_bp.route('/predict/health', methods=['GET'])
def predict_health():
    """预测服务健康检查"""
    try:
        is_healthy = prediction_service.is_healthy()
        service_info = prediction_service.get_service_info()
        
        health_status = {
            "status": "healthy" if is_healthy else "unhealthy",
            "healthy": is_healthy,
            "model_loaded": prediction_service.model_loaded,
            "scaler_loaded": prediction_service.scaler_loaded,
            "timestamp": datetime.datetime.utcnow().isoformat(),
            "details": service_info
        }
        
        return jsonify(APIResponse(
            code=0, 
            message="健康检查完成", 
            data=health_status
        ).dict())
        
    except Exception as e:
        logger.error(f"预测健康检查失败: {str(e)}")
        return jsonify(APIResponse(
            code=500, 
            message=f"健康检查失败: {str(e)}", 
            data={
                "status": "error",
                "healthy": False,
                "timestamp": datetime.datetime.utcnow().isoformat()
            }
        ).dict()), 500