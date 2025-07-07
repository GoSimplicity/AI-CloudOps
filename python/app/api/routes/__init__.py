"""
路由模块
"""

import logging
from flask import Blueprint

logger = logging.getLogger("aiops.routes")

api_v1 = Blueprint('api_v1', __name__, url_prefix='/api/v1')

try:
    from .health import health_bp
    api_v1.register_blueprint(health_bp)
    logger.info("已注册健康检查路由")
except Exception as e:
    logger.warning(f"注册健康检查路由失败: {str(e)}")

try:
    from .predict import predict_bp
    api_v1.register_blueprint(predict_bp)
    logger.info("已注册预测路由")
except Exception as e:
    logger.warning(f"注册预测路由失败: {str(e)}")

try:
    from .rca import rca_bp
    api_v1.register_blueprint(rca_bp)
    logger.info("已注册根因分析路由")
except Exception as e:
    logger.warning(f"注册根因分析路由失败: {str(e)}")

try:
    from .autofix import autofix_bp
    api_v1.register_blueprint(autofix_bp)
    logger.info("已注册自动修复路由")
except Exception as e:
    logger.warning(f"注册自动修复路由失败: {str(e)}")

try:
    from .assistant import assistant_bp
    api_v1.register_blueprint(assistant_bp, url_prefix='/assistant')
    logger.info("已注册智能助手路由")
except Exception as e:
    logger.warning(f"注册智能助手路由失败: {str(e)}")

def register_routes(app):
    """注册所有路由"""
    
    app.register_blueprint(api_v1)
    
    # 根路径重定向到健康检查
    @app.route('/')
    def index():
        return {
            "service": "AIOps Platform",
            "version": "1.0.0",
            "status": "running",
            "endpoints": {
                "health": "/api/v1/health",
                "prediction": "/api/v1/predict",
                "rca": "/api/v1/rca",
                "autofix": "/api/v1/autofix",
                "assistant": "/api/v1/assistant"
            }
        }