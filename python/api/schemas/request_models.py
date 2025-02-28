"""
MIT License

Copyright (c) 2024 Bamboo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

description: 请求模型
"""

from pydantic import BaseModel, Field
from typing import Dict, List, Any, Optional, Union
from datetime import datetime
from enum import Enum

# 通用模型
class TimeRange(BaseModel):
    start: str = Field(..., description="开始时间，ISO 8601格式")
    end: str = Field(..., description="结束时间，ISO 8601格式")

class ServiceIdentifier(BaseModel):
    service_name: str = Field(..., description="服务名称")
    environment: Optional[str] = Field(None, description="环境（如prod, dev, test）")
    cluster: Optional[str] = Field(None, description="集群名称")

# 异常检测模型
class MetricDataPoint(BaseModel):
    timestamp: Union[int, str] = Field(..., description="时间戳")
    value: float = Field(..., description="指标值")
    tags: Optional[Dict[str, str]] = Field(None, description="标签")

class LogEntry(BaseModel):
    timestamp: Union[int, str] = Field(..., description="时间戳")
    message: str = Field(..., description="日志内容")
    level: Optional[str] = Field(None, description="日志级别")
    source: Optional[str] = Field(None, description="日志来源")

class TraceSpan(BaseModel):
    trace_id: str = Field(..., description="链路ID")
    span_id: str = Field(..., description="跨度ID")
    parent_span_id: Optional[str] = Field(None, description="父跨度ID")
    operation_name: str = Field(..., description="操作名称")
    start_time: Union[int, str] = Field(..., description="开始时间")
    end_time: Union[int, str] = Field(..., description="结束时间")
    tags: Optional[Dict[str, str]] = Field(None, description="标签")

class AnomalyType(str, Enum):
    SPIKE = "spike"
    DROP = "drop"
    TREND = "trend"
    PATTERN = "pattern"
    OUTLIER = "outlier"

class AnomalyDetectionResult(BaseModel):
    timestamp: Union[int, str] = Field(..., description="异常发现时间")
    anomaly_type: AnomalyType = Field(..., description="异常类型")
    metric_name: Optional[str] = Field(None, description="异常的指标名称")
    score: float = Field(..., description="异常分数，越高越异常")
    description: str = Field(..., description="异常描述")
    affected_services: List[str] = Field(default_factory=list, description="受影响的服务")
    related_anomalies: List[str] = Field(default_factory=list, description="相关的其他异常")

# 预测模型
class ResourceType(str, Enum):
    CPU = "cpu"
    MEMORY = "memory"
    DISK = "disk"
    NETWORK = "network"
    LATENCY = "latency"

class ResourceUsagePrediction(BaseModel):
    timestamp: Union[int, str] = Field(..., description="预测时间点")
    resource_type: ResourceType = Field(..., description="资源类型")
    predicted_value: float = Field(..., description="预测值")
    lower_bound: Optional[float] = Field(None, description="下界")
    upper_bound: Optional[float] = Field(None, description="上界")
    confidence: float = Field(..., description="置信度")

class FailurePrediction(BaseModel):
    service_name: str = Field(..., description="服务名称")
    failure_type: str = Field(..., description="故障类型")
    probability: float = Field(..., description="故障概率")
    expected_time: Optional[str] = Field(None, description="预期发生时间")
    affected_components: List[str] = Field(default_factory=list, description="可能受影响的组件")
    prevention_actions: List[str] = Field(default_factory=list, description="建议的预防措施")

# 根因分析模型
class RootCauseAnalysisRequest(BaseModel):
    incident_id: str = Field(..., description="事件ID")
    affected_services: List[str] = Field(..., description="受影响的服务")
    symptoms: List[str] = Field(..., description="症状描述")
    time_range: TimeRange = Field(..., description="时间范围")
    
class RootCauseAnalysisResult(BaseModel):
    incident_id: str = Field(..., description="事件ID")
    root_causes: List[Dict[str, Any]] = Field(..., description="根因列表")
    confidence: float = Field(..., description="置信度")
    related_evidence: List[Dict[str, Any]] = Field(..., description="相关证据")
    recommendation: str = Field(..., description="建议措施")