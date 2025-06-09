from datetime import datetime
from typing import List, Dict, Any, Optional
from dataclasses import dataclass
import pandas as pd

@dataclass
class MetricData:
    name: str
    values: pd.DataFrame
    labels: Dict[str, str]
    unit: Optional[str] = None
    source: Optional[str] = None

@dataclass
class AnomalyResult:
    metric: str
    anomaly_points: List[datetime]
    anomaly_scores: List[float]
    detection_methods: Dict[str, Any]
    severity: str = "medium"  # low, medium, high
    
@dataclass
class CorrelationResult:
    metric_pairs: List[tuple]
    correlation_matrix: pd.DataFrame
    significant_correlations: Dict[str, List[tuple]]
    method: str = "pearson"

@dataclass
class AgentState:
    messages: List[Dict[str, Any]]
    current_step: str
    context: Dict[str, Any]
    next_action: Optional[str] = None
    iteration_count: int = 0
    max_iterations: int = 10

@dataclass
class PredictionFeatures:
    qps: float
    sin_time: float
    cos_time: float
    hour: int
    day_of_week: int
    timestamp: datetime