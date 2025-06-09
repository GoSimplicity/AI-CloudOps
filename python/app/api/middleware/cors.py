from flask_cors import CORS
import logging

logger = logging.getLogger("aiops.cors")

def setup_cors(app):
    """设置CORS中间件"""
    try:
        # 配置CORS
        CORS(app, 
             resources={
                 r"/api/*": {
                     "origins": "*",
                     "methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
                     "allow_headers": ["Content-Type", "Authorization", "X-Requested-With"],
                     "supports_credentials": True
                 }
             })
        
        @app.after_request
        def after_request(response):
            # 添加额外的CORS头
            response.headers.add('Access-Control-Allow-Origin', '*')
            response.headers.add('Access-Control-Allow-Headers', 'Content-Type,Authorization,X-Requested-With')
            response.headers.add('Access-Control-Allow-Methods', 'GET,PUT,POST,DELETE,OPTIONS')
            response.headers.add('Access-Control-Allow-Credentials', 'true')
            return response
        
        logger.info("CORS中间件设置完成")
        
    except Exception as e:
        logger.error(f"CORS中间件设置失败: {str(e)}")