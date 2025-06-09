"""
配置模块 - 管理所有配置信息
"""

from .settings import config
from .logging import setup_logging

__all__ = ["config", "setup_logging"]