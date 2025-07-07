from datetime import datetime, timedelta, timezone
from typing import List, Optional, Dict
from pydantic import BaseModel, Field, validator
from app.config.settings import config

class RCARequest(BaseModel):
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    metrics: Optional[List[str]] = None
    time_range_minutes: Optional[int] = Field(None, ge=1, le=config.rca.max_time_range)
    
    @validator('start_time', 'end_time', pre=True, allow_reuse=True)
    def parse_datetime(cls, v):
        if isinstance(v, str):
            try:
                return datetime.fromisoformat(v.replace('Z', '+00:00'))
            except ValueError:
                return datetime.fromisoformat(v)
        return v
    
    def __init__(self, **data):
        super().__init__(**data)
        
        # 如果没有提供时间范围，使用默认值
        if not self.start_time or not self.end_time:
            tz = timezone.utc
            now = datetime.now(tz)
            if self.time_range_minutes:
                self.end_time = now
                self.start_time = self.end_time - timedelta(minutes=self.time_range_minutes)
            else:
                self.end_time = now
                self.start_time = self.end_time - timedelta(minutes=config.rca.default_time_range)
        
        # 如果没有提供指标，使用默认指标
        if not self.metrics:
            self.metrics = config.rca.default_metrics

class AutoFixRequest(BaseModel):
    deployment: str = Field(..., min_length=1)
    namespace: str = Field(default="default", min_length=1)
    event: str = Field(..., min_length=1)
    force: bool = Field(default=False)
    auto_restart: bool = Field(default=True)

class PredictionRequest(BaseModel):
    current_qps: Optional[float] = None
    timestamp: Optional[datetime] = None
    include_confidence: bool = Field(default=True)
    
    @validator('current_qps', allow_reuse=True)
    def validate_qps(cls, v):
        if v is not None and v < 0:
            raise ValueError("QPS不能为负数")
        return v

class AssistantRequest(BaseModel):
    """智能小助手请求模型"""
    question: str = Field(..., min_length=1, description="用户提问")
    chat_history: Optional[List[Dict[str, str]]] = Field(default=None, description="对话历史记录")
    use_web_search: bool = Field(default=False, description="是否使用网络搜索增强回答")
    max_context_docs: int = Field(default=4, ge=1, le=10, description="最大上下文文档数量")
    session_id: Optional[str] = Field(default=None, description="会话ID，为空则创建新会话")