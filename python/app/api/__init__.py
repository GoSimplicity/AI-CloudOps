"""
API模块 - 处理HTTP请求和响应
"""

from .routes import register_routes
from .middleware import register_middleware

__all__ = ["register_routes", "register_middleware"]