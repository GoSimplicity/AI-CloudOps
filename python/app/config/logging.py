import logging
import sys
from typing import Optional
from flask import Flask
from app.config.settings import config

def setup_logging(app: Optional[Flask] = None) -> None:
    """设置日志配置"""
    
    # 日志格式
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    
    # 控制台处理器
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setFormatter(formatter)
    console_handler.setLevel(getattr(logging, config.log_level.upper()))
    
    # 根日志器配置
    root_logger = logging.getLogger()
    root_logger.setLevel(getattr(logging, config.log_level.upper()))
    
    # 清除已有的处理器
    for handler in root_logger.handlers[:]:
        root_logger.removeHandler(handler)
    
    root_logger.addHandler(console_handler)
    
    # Flask应用日志配置
    if app:
        app.logger.setLevel(getattr(logging, config.log_level.upper()))
        for handler in app.logger.handlers[:]:
            app.logger.removeHandler(handler)
        app.logger.addHandler(console_handler)
    
    # 设置第三方库日志级别
    logging.getLogger('urllib3').setLevel(logging.WARNING)
    logging.getLogger('requests').setLevel(logging.WARNING)
    logging.getLogger('kubernetes').setLevel(logging.WARNING)
    logging.getLogger('openai').setLevel(logging.WARNING)
    
    # 设置应用日志器
    app_logger = logging.getLogger('aiops')
    app_logger.setLevel(getattr(logging, config.log_level.upper()))