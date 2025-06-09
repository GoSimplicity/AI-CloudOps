from datetime import datetime, timedelta
from typing import List, Optional
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
            if self.time_range_minutes:
                self.end_time = datetime.utcnow()
                self.start_time = self.end_time - timedelta(minutes=self.time_range_minutes)
            else:
                self.end_time = datetime.utcnow()
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