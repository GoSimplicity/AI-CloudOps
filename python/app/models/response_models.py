from datetime import datetime
from typing import List, Dict, Any, Optional, Generic, TypeVar, Union
from pydantic import BaseModel

T = TypeVar('T')

class APIResponse(BaseModel, Generic[T]):
    """统一API响应格式"""
    code: int = 0
    message: str = ""
    data: Optional[T] = None

class AnomalyInfo(BaseModel):
    count: int
    first_occurrence: str
    last_occurrence: str
    max_score: float
    avg_score: float
    detection_methods: Dict[str, Any]

class RootCauseCandidate(BaseModel):
    metric: str
    confidence: float
    first_occurrence: str
    anomaly_count: int
    related_metrics: List[tuple]
    description: Optional[str] = None

class RCAResponse(BaseModel):
    status: str
    anomalies: Dict[str, AnomalyInfo]
    correlations: Dict[str, List[tuple]]
    root_cause_candidates: List[RootCauseCandidate]
    analysis_time: str
    time_range: Dict[str, str]
    metrics_analyzed: List[str]
    summary: Optional[str] = None

class PredictionResponse(BaseModel):
    instances: int
    current_qps: float
    timestamp: str
    confidence: Optional[float] = None
    model_version: Optional[str] = None
    prediction_type: Optional[str] = None
    features: Optional[Dict[str, float]] = None

class AutoFixResponse(BaseModel):
    status: str
    result: str
    deployment: str
    namespace: str
    actions_taken: List[str]
    timestamp: str
    success: bool
    error_message: Optional[str] = None

class HealthResponse(BaseModel):
    status: str
    components: Dict[str, bool]
    timestamp: str
    version: Optional[str] = None
    uptime: Optional[float] = None