"""
路由模块
"""

from flask import Blueprint
from .health import health_bp
from .predict import predict_bp
from .rca import rca_bp
from .autofix import autofix_bp

def register_routes(app):
    """注册所有路由"""
    
    # 创建API蓝图
    api_v1 = Blueprint('api_v1', __name__, url_prefix='/api/v1')
    
    # 注册子蓝图
    api_v1.register_blueprint(health_bp)
    api_v1.register_blueprint(predict_bp)
    api_v1.register_blueprint(rca_bp)
    api_v1.register_blueprint(autofix_bp)
    
    # 注册到应用
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
                "autofix": "/api/v1/autofix"
            }
        }