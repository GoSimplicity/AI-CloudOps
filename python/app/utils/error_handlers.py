"""
统一的错误处理工具
提供标准化的错误处理、日志记录和响应格式
"""

import logging
import traceback
from typing import Any, Dict, Optional, Tuple, Union, Type, List
from functools import wraps
from datetime import datetime
import asyncio

from app.constants import (
    HTTP_STATUS_BAD_REQUEST, HTTP_STATUS_INTERNAL_ERROR,
    HTTP_STATUS_NOT_FOUND, HTTP_STATUS_UNAUTHORIZED,
    ERROR_MESSAGES, SUCCESS_MESSAGES
)


class AICloudOpsError(Exception):
    """AI-CloudOps 基础异常类"""
    
    def __init__(self, message: str, error_code: str = "UNKNOWN", details: Optional[Dict[str, Any]] = None):
        self.message = message
        self.error_code = error_code
        self.details = details or {}
        self.timestamp = datetime.now()
        super().__init__(self.message)


class ValidationError(AICloudOpsError):
    """输入验证错误"""
    
    def __init__(self, message: str, field: Optional[str] = None, value: Optional[Any] = None):
        details = {}
        if field:
            details['field'] = field
        if value is not None:
            details['invalid_value'] = str(value)
        super().__init__(message, "VALIDATION_ERROR", details)


class ServiceError(AICloudOpsError):
    """服务层错误"""
    
    def __init__(self, message: str, service: str, operation: Optional[str] = None):
        details = {'service': service}
        if operation:
            details['operation'] = operation
        super().__init__(message, "SERVICE_ERROR", details)


class ConfigurationError(AICloudOpsError):
    """配置错误"""
    
    def __init__(self, message: str, config_key: Optional[str] = None):
        details = {}
        if config_key:
            details['config_key'] = config_key
        super().__init__(message, "CONFIGURATION_ERROR", details)


class ExternalServiceError(AICloudOpsError):
    """外部服务错误"""
    
    def __init__(self, message: str, service: str, status_code: Optional[int] = None):
        details = {'external_service': service}
        if status_code:
            details['status_code'] = status_code
        super().__init__(message, "EXTERNAL_SERVICE_ERROR", details)


class ErrorHandler:
    """统一错误处理器"""
    
    def __init__(self, logger: Optional[logging.Logger] = None):
        self.logger = logger or logging.getLogger(__name__)
    
    def log_and_return_error(
        self, 
        error: Exception, 
        context: str,
        include_traceback: bool = True
    ) -> Tuple[str, Dict[str, Any]]:
        """记录错误并返回格式化的错误信息"""
        
        error_id = f"error_{datetime.now().strftime('%Y%m%d_%H%M%S_%f')}"
        
        # 构建错误详情
        error_details = {
            'error_id': error_id,
            'error_type': type(error).__name__,
            'context': context,
            'timestamp': datetime.now().isoformat()
        }
        
        # 如果是自定义异常，添加额外信息
        if isinstance(error, AICloudOpsError):
            error_details.update({
                'error_code': error.error_code,
                'details': error.details
            })
        
        # 记录日志
        log_message = f"[{error_id}] {context}: {str(error)}"
        
        if include_traceback:
            self.logger.error(log_message, exc_info=True)
            error_details['traceback'] = traceback.format_exc()
        else:
            self.logger.error(log_message)
        
        return str(error), error_details
    
    def handle_validation_error(self, error: Exception, context: str = "") -> Tuple[Dict[str, Any], int]:
        """处理验证错误"""
        message, details = self.log_and_return_error(error, f"Validation error: {context}", False)
        
        return {
            'code': HTTP_STATUS_BAD_REQUEST,
            'message': ERROR_MESSAGES.get('invalid_input', message),
            'data': {},
            'error_details': details
        }, HTTP_STATUS_BAD_REQUEST
    
    def handle_service_error(self, error: Exception, context: str = "") -> Tuple[Dict[str, Any], int]:
        """处理服务错误"""
        message, details = self.log_and_return_error(error, f"Service error: {context}")
        
        return {
            'code': HTTP_STATUS_INTERNAL_ERROR,
            'message': ERROR_MESSAGES.get('internal_error', message),
            'data': {},
            'error_details': details
        }, HTTP_STATUS_INTERNAL_ERROR
    
    def handle_not_found_error(self, resource: str, identifier: str = "") -> Tuple[Dict[str, Any], int]:
        """处理资源未找到错误"""
        message = f"{resource} not found"
        if identifier:
            message += f": {identifier}"
        
        self.logger.warning(message)
        
        return {
            'code': HTTP_STATUS_NOT_FOUND,
            'message': ERROR_MESSAGES.get('not_found', message),
            'data': {},
            'error_details': {
                'resource': resource,
                'identifier': identifier,
                'timestamp': datetime.now().isoformat()
            }
        }, HTTP_STATUS_NOT_FOUND


def error_handler(
    logger: Optional[logging.Logger] = None,
    return_exceptions: bool = False,
    default_return_value: Any = None
):
    """错误处理装饰器"""
    
    def decorator(func):
        @wraps(func)
        async def async_wrapper(*args, **kwargs):
            handler = ErrorHandler(logger)
            try:
                return await func(*args, **kwargs)
            except Exception as e:
                if return_exceptions:
                    return default_return_value
                error_msg, details = handler.log_and_return_error(e, f"Function {func.__name__}")
                raise ServiceError(error_msg, func.__name__)
        
        @wraps(func)
        def sync_wrapper(*args, **kwargs):
            handler = ErrorHandler(logger)
            try:
                return func(*args, **kwargs)
            except Exception as e:
                if return_exceptions:
                    return default_return_value
                error_msg, details = handler.log_and_return_error(e, f"Function {func.__name__}")
                raise ServiceError(error_msg, func.__name__)
        
        return async_wrapper if asyncio.iscoroutinefunction(func) else sync_wrapper
    
    return decorator


def retry_on_exception(
    max_retries: int = 3,
    delay: float = 1.0,
    backoff_factor: float = 2.0,
    exceptions: Tuple[Type[Exception], ...] = (Exception,),
    logger: Optional[logging.Logger] = None
):
    """重试装饰器"""
    
    def decorator(func):
        @wraps(func)
        async def async_wrapper(*args, **kwargs):
            _logger = logger or logging.getLogger(func.__module__)
            last_exception = None
            
            for attempt in range(max_retries + 1):
                try:
                    return await func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    if attempt == max_retries:
                        _logger.error(f"Function {func.__name__} failed after {max_retries} retries: {str(e)}")
                        raise
                    
                    retry_delay = delay * (backoff_factor ** attempt)
                    _logger.warning(f"Function {func.__name__} failed (attempt {attempt + 1}/{max_retries + 1}), retrying in {retry_delay}s: {str(e)}")
                    await asyncio.sleep(retry_delay)
            
            raise last_exception
        
        @wraps(func)
        def sync_wrapper(*args, **kwargs):
            import time
            _logger = logger or logging.getLogger(func.__module__)
            last_exception = None
            
            for attempt in range(max_retries + 1):
                try:
                    return func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    if attempt == max_retries:
                        _logger.error(f"Function {func.__name__} failed after {max_retries} retries: {str(e)}")
                        raise
                    
                    retry_delay = delay * (backoff_factor ** attempt)
                    _logger.warning(f"Function {func.__name__} failed (attempt {attempt + 1}/{max_retries + 1}), retrying in {retry_delay}s: {str(e)}")
                    time.sleep(retry_delay)
            
            raise last_exception
        
        return async_wrapper if asyncio.iscoroutinefunction(func) else sync_wrapper
    
    return decorator


def validate_required_fields(data: Dict[str, Any], required_fields: List[str]) -> None:
    """验证必需字段"""
    missing_fields = []
    
    for field in required_fields:
        if field not in data or data[field] is None:
            missing_fields.append(field)
        elif isinstance(data[field], str) and not data[field].strip():
            missing_fields.append(field)
    
    if missing_fields:
        raise ValidationError(
            f"Missing required fields: {', '.join(missing_fields)}",
            field=missing_fields[0] if len(missing_fields) == 1 else None
        )


def validate_field_type(data: Dict[str, Any], field: str, expected_type: Type, required: bool = True) -> None:
    """验证字段类型"""
    if field not in data:
        if required:
            raise ValidationError(f"Missing required field: {field}", field=field)
        return
    
    value = data[field]
    if value is None and not required:
        return
    
    if not isinstance(value, expected_type):
        raise ValidationError(
            f"Field '{field}' must be of type {expected_type.__name__}, got {type(value).__name__}",
            field=field,
            value=value
        )


def validate_field_range(
    data: Dict[str, Any], 
    field: str, 
    min_value: Optional[Union[int, float]] = None,
    max_value: Optional[Union[int, float]] = None
) -> None:
    """验证字段范围"""
    if field not in data:
        return
    
    value = data[field]
    if value is None:
        return
    
    if min_value is not None and value < min_value:
        raise ValidationError(
            f"Field '{field}' must be >= {min_value}, got {value}",
            field=field,
            value=value
        )
    
    if max_value is not None and value > max_value:
        raise ValidationError(
            f"Field '{field}' must be <= {max_value}, got {value}",
            field=field,
            value=value
        )


def safe_cast(value: Any, target_type: Type, default: Any = None) -> Any:
    """安全类型转换"""
    try:
        if target_type == bool and isinstance(value, str):
            return value.lower() in ('true', '1', 'yes', 'on')
        return target_type(value)
    except (ValueError, TypeError):
        return default


class ContextualLogger:
    """带上下文的日志记录器"""
    
    def __init__(self, logger: logging.Logger, context: Dict[str, Any]):
        self.logger = logger
        self.context = context
    
    def _format_message(self, message: str) -> str:
        """格式化消息，添加上下文信息"""
        context_str = " | ".join([f"{k}={v}" for k, v in self.context.items()])
        return f"[{context_str}] {message}"
    
    def debug(self, message: str, **kwargs):
        self.logger.debug(self._format_message(message), **kwargs)
    
    def info(self, message: str, **kwargs):
        self.logger.info(self._format_message(message), **kwargs)
    
    def warning(self, message: str, **kwargs):
        self.logger.warning(self._format_message(message), **kwargs)
    
    def error(self, message: str, **kwargs):
        self.logger.error(self._format_message(message), **kwargs)
    
    def critical(self, message: str, **kwargs):
        self.logger.critical(self._format_message(message), **kwargs)


def create_contextual_logger(logger: logging.Logger, **context) -> ContextualLogger:
    """创建带上下文的日志记录器"""
    return ContextualLogger(logger, context)