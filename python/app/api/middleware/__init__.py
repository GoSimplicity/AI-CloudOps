"""
中间件模块
"""

from .cors import setup_cors
from .error_handler import setup_error_handlers

def register_middleware(app):
    """注册所有中间件"""
    setup_cors(app)
    setup_error_handlers(app)