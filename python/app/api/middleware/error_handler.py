import logging
import traceback
from flask import jsonify, request
from datetime import datetime

logger = logging.getLogger("aiops.error_handler")

def setup_error_handlers(app):
    """设置错误处理器"""
    
    @app.errorhandler(400)
    def bad_request(error):
        """处理400错误"""
        logger.warning(f"Bad request: {request.url} - {str(error)}")
        return jsonify({
            "error": "请求参数错误",
            "message": str(error.description) if hasattr(error, 'description') else "Bad Request",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 400
    
    @app.errorhandler(401)
    def unauthorized(error):
        """处理401错误"""
        logger.warning(f"Unauthorized access: {request.url}")
        return jsonify({
            "error": "未授权访问",
            "message": "需要有效的身份验证",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 401
    
    @app.errorhandler(403)
    def forbidden(error):
        """处理403错误"""
        logger.warning(f"Forbidden access: {request.url}")
        return jsonify({
            "error": "访问被禁止",
            "message": "没有访问权限",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 403
    
    @app.errorhandler(404)
    def not_found(error):
        """处理404错误"""
        logger.warning(f"Not found: {request.url}")
        return jsonify({
            "error": "资源未找到",
            "message": f"请求的资源 {request.path} 不存在",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 404
    
    @app.errorhandler(405)
    def method_not_allowed(error):
        """处理405错误"""
        logger.warning(f"Method not allowed: {request.method} {request.url}")
        return jsonify({
            "error": "方法不被允许",
            "message": f"方法 {request.method} 不被允许用于 {request.path}",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path,
            "method": request.method
        }), 405
    
    @app.errorhandler(422)
    def unprocessable_entity(error):
        """处理422错误"""
        logger.warning(f"Unprocessable entity: {request.url} - {str(error)}")
        return jsonify({
            "error": "无法处理的实体",
            "message": "请求格式正确但语义错误",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 422
    
    @app.errorhandler(429)
    def rate_limit_exceeded(error):
        """处理429错误"""
        logger.warning(f"Rate limit exceeded: {request.url}")
        return jsonify({
            "error": "请求频率超限",
            "message": "请求过于频繁，请稍后再试",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 429
    
    @app.errorhandler(500)
    def internal_server_error(error):
        """处理500错误"""
        error_id = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        logger.error(f"Internal server error [{error_id}]: {request.url}")
        logger.error(f"Error details: {str(error)}")
        logger.error(f"Traceback: {traceback.format_exc()}")
        
        return jsonify({
            "error": "内部服务器错误",
            "message": "服务器遇到意外错误",
            "error_id": error_id,
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 500
    
    @app.errorhandler(502)
    def bad_gateway(error):
        """处理502错误"""
        logger.error(f"Bad gateway: {request.url}")
        return jsonify({
            "error": "网关错误",
            "message": "上游服务器返回无效响应",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 502
    
    @app.errorhandler(503)
    def service_unavailable(error):
        """处理503错误"""
        logger.error(f"Service unavailable: {request.url}")
        return jsonify({
            "error": "服务不可用",
            "message": "服务暂时不可用，请稍后重试",
            "timestamp": datetime.utcnow().isoformat(),
            "path": request.path
        }), 503
    
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
            return jsonify({
                "error": "意外错误（调试模式）",
                "message": str(error),
                "type": type(error).__name__,
                "error_id": error_id,
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path,
                "traceback": traceback.format_exc().split('\n')
            }), 500
        else:
            return jsonify({
                "error": "意外错误",
                "message": "服务器遇到意外错误，请联系管理员",
                "error_id": error_id,
                "timestamp": datetime.utcnow().isoformat(),
                "path": request.path
            }), 500
    
    logger.info("错误处理器设置完成")