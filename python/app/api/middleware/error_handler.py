import logging
import traceback
from flask import jsonify, request
from datetime import datetime
from app.models.response_models import APIResponse

logger = logging.getLogger("aiops.error_handler")

def setup_error_handlers(app):
    """设置错误处理器"""
    
    @app.errorhandler(400)
    def bad_request(error):
        """处理400错误"""
        logger.warning(f"Bad request: {request.url} - {str(error)}")
        return jsonify(APIResponse(
            code=400,
            message=str(error.description) if hasattr(error, 'description') else "请求参数错误",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 400
    
    @app.errorhandler(401)
    def unauthorized(error):
        """处理401错误"""
        logger.warning(f"Unauthorized access: {request.url}")
        return jsonify(APIResponse(
            code=401,
            message="未授权访问",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 401
    
    @app.errorhandler(403)
    def forbidden(error):
        """处理403错误"""
        logger.warning(f"Forbidden access: {request.url}")
        return jsonify(APIResponse(
            code=403,
            message="访问被禁止",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 403
    
    @app.errorhandler(404)
    def not_found(error):
        """处理404错误"""
        logger.warning(f"Not found: {request.url}")
        return jsonify(APIResponse(
            code=404,
            message=f"请求的资源 {request.path} 不存在",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 404
    
    @app.errorhandler(405)
    def method_not_allowed(error):
        """处理405错误"""
        logger.warning(f"Method not allowed: {request.method} {request.url}")
        return jsonify(APIResponse(
            code=405,
            message=f"方法 {request.method} 不被允许用于 {request.path}",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path,
                "method": request.method
            }
        ).dict()), 405
    
    @app.errorhandler(422)
    def unprocessable_entity(error):
        """处理422错误"""
        logger.warning(f"Unprocessable entity: {request.url} - {str(error)}")
        return jsonify(APIResponse(
            code=422,
            message="请求格式正确但语义错误",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 422
    
    @app.errorhandler(429)
    def rate_limit_exceeded(error):
        """处理429错误"""
        logger.warning(f"Rate limit exceeded: {request.url}")
        return jsonify(APIResponse(
            code=429,
            message="请求过于频繁，请稍后再试",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 429
    
    @app.errorhandler(500)
    def internal_server_error(error):
        """处理500错误"""
        error_id = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        logger.error(f"Internal server error [{error_id}]: {request.url}")
        logger.error(f"Error details: {str(error)}")
        logger.error(f"Traceback: {traceback.format_exc()}")
        
        return jsonify(APIResponse(
            code=500,
            message="服务器遇到意外错误",
            data={
                "error_id": error_id,
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 500
    
    @app.errorhandler(502)
    def bad_gateway(error):
        """处理502错误"""
        logger.error(f"Bad gateway: {request.url}")
        return jsonify(APIResponse(
            code=502,
            message="上游服务器返回无效响应",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 502
    
    @app.errorhandler(503)
    def service_unavailable(error):
        """处理503错误"""
        logger.error(f"Service unavailable: {request.url}")
        return jsonify(APIResponse(
            code=503,
            message="服务暂时不可用，请稍后重试",
            data={
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }
        ).dict()), 503
    
    @app.errorhandler(Exception)
    def handle_unexpected_error(error):
        """处理未预期的错误"""
        error_id = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        logger.error(f"Unexpected error [{error_id}]: {request.url}")
        logger.error(f"Error type: {type(error).__name__}")
        logger.error(f"Error message: {str(error)}")
        logger.error(f"Traceback: {traceback.format_exc()}")
        
        # 在开发模式下返回详细错误信息
        from app.config.settings import config
        if config.debug:
            return jsonify(APIResponse(
                code=500,
                message=f"意外错误（调试模式）: {str(error)}",
                data={
                    "type": type(error).__name__,
                    "error_id": error_id,
                    "timestamp": datetime.utcnow().isoformat(),
                    "path": request.path,
                    "traceback": traceback.format_exc().split('\n')
                }
            ).dict()), 500
        else:
            return jsonify(APIResponse(
                code=500,
                message="服务器遇到意外错误，请联系管理员",
                data={
                    "error_id": error_id,
                    "timestamp": datetime.utcnow().isoformat(),
                    "path": request.path
                }
            ).dict()), 500
    
    logger.info("错误处理器设置完成")