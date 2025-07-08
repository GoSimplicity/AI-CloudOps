"""
AIOps平台主应用入口
"""

import os
import sys
import logging
import time
from flask import Flask

# 添加项目根目录到系统路径
current_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if current_dir not in sys.path:
    sys.path.insert(0, current_dir)
from app.config.settings import config
from app.config.logging import setup_logging
from app.api.routes import register_routes
from app.api.middleware import register_middleware

# 记录启动时间
start_time = time.time()

def create_app():
    """创建Flask应用实例"""
    app = Flask(__name__)
    
    # 设置日志
    setup_logging(app)
    
    # 获取应用日志器
    logger = logging.getLogger("aiops")
    logger.info("=" * 50)
    logger.info("AIOps平台启动中...")
    logger.info(f"调试模式: {config.debug}")
    logger.info(f"日志级别: {config.log_level}")
    logger.info("=" * 50)
    
    # 注册中间件
    try:
        register_middleware(app)
        logger.info("中间件注册完成")
    except Exception as e:
        logger.error(f"中间件注册失败: {str(e)}")
        logger.warning("将继续启动，但部分中间件功能可能不可用")
    
    # 注册路由
    try:
        register_routes(app)
        logger.info("路由注册完成")
    except Exception as e:
        logger.error(f"路由注册失败: {str(e)}")
        logger.warning("将继续启动，但部分路由功能可能不可用")
        
    # 初始化WebSocket
    try:
        from app.api.routes.assistant import init_websocket
        init_websocket(app)
        logger.info("WebSocket初始化完成")
    except ImportError as e:
        logger.warning(f"WebSocket模块导入失败: {str(e)}")
        logger.warning("将继续启动，但WebSocket功能不可用")
    except Exception as e:
        logger.error(f"WebSocket初始化失败: {str(e)}")
        logger.warning("将继续启动，但WebSocket功能不可用")
    
    # 定义启动信息函数
    def log_startup_info():
        startup_time = time.time() - start_time
        logger.info(f"AIOps平台启动完成，耗时: {startup_time:.2f}秒")
        logger.info(f"服务地址: http://{config.host}:{config.port}")
        logger.info("可用的API端点:")
        logger.info("  - GET  /api/v1/health        - 健康检查")
        logger.info("  - GET  /api/v1/predict       - 负载预测")
        logger.info("  - POST /api/v1/rca           - 根因分析")
        logger.info("  - POST /api/v1/autofix       - 自动修复")
        logger.info("  - POST /api/v1/assistant/query - 智能小助手")
        logger.info("  - WS   /api/v1/assistant/stream - 流式智能小助手")
    
    # 替代 before_first_request 的解决方案
    app_started = False
    
    @app.before_request
    def _log_startup_wrapper():
        nonlocal app_started
        if not app_started:
            log_startup_info()
            app_started = True
    
    # 添加关闭处理
    @app.teardown_appcontext
    def cleanup(error):
        if error:
            logger = logging.getLogger("aiops")
            logger.error(f"应用上下文清理时发生错误: {str(error)}")
    
    return app

# 创建应用实例
app = create_app()

if __name__ == "__main__":
    logger = logging.getLogger("aiops")
    
    try:
        logger.info(f"在 {config.host}:{config.port} 启动Flask服务器")
        app.run(
            host=config.host,
            port=config.port,
            debug=config.debug,
            threaded=True
        )
    except KeyboardInterrupt:
        logger.info("收到中断信号，正在关闭服务...")
    except Exception as e:
        logger.error(f"服务启动失败: {str(e)}")
        raise
    finally:
        total_time = time.time() - start_time
        logger.info(f"AIOps平台运行总时长: {total_time:.2f}秒")
        logger.info("AIOps平台已关闭")