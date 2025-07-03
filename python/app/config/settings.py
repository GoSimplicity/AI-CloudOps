import os
from dataclasses import dataclass, field
from typing import List, Optional
from dotenv import load_dotenv

# 加载环境变量
load_dotenv()

@dataclass
class PrometheusConfig:
    host: str = os.getenv("PROMETHEUS_HOST", "127.0.0.1:9090")
    timeout: int = int(os.getenv("PROMETHEUS_TIMEOUT", "30"))
    
    @property
    def url(self) -> str:
        return f"http://{self.host}"

@dataclass
class LLMConfig:
    provider: str = os.getenv("LLM_PROVIDER", "openai")  # 'openai' 或 'ollama'
    model: str = os.getenv("LLM_MODEL", "Qwen/Qwen2.5-14B-Instruct")
    api_key: str = os.getenv("LLM_API_KEY", "sk-xrykvuqngkhbsmdtmvhzsupjafandfyhcdbcqojlyvrftttq")
    base_url: str = os.getenv("LLM_BASE_URL", "https://api.siliconflow.cn/v1")
    temperature: float = float(os.getenv("LLM_TEMPERATURE", "0.7"))
    max_tokens: int = int(os.getenv("LLM_MAX_TOKENS", "2048"))
    
    # Ollama 配置
    ollama_model: str = os.getenv("OLLAMA_MODEL", "qwen2.5:3b")
    ollama_base_url: str = os.getenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434/v1")
    
    @property
    def effective_model(self) -> str:
        """根据提供商返回有效的模型名称"""
        if self.provider.lower() == "ollama":
            return self.ollama_model
        return self.model
    
    @property
    def effective_base_url(self) -> str:
        """根据提供商返回有效的基础URL"""
        if self.provider.lower() == "ollama":
            return self.ollama_base_url
        return self.base_url
    
    @property
    def effective_api_key(self) -> str:
        """根据提供商返回有效的API密钥"""
        if self.provider.lower() == "ollama":
            return "ollama"
        return self.api_key

@dataclass
class K8sConfig:
    in_cluster: bool = os.getenv("K8S_IN_CLUSTER", "false").lower() == "true"
    config_path: Optional[str] = os.getenv("K8S_CONFIG_PATH") or os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), "deploy/kubernetes/config")
    namespace: str = os.getenv("K8S_NAMESPACE", "default")

@dataclass
class RCAConfig:
    default_time_range: int = int(os.getenv("RCA_DEFAULT_TIME_RANGE", "30"))
    max_time_range: int = int(os.getenv("RCA_MAX_TIME_RANGE", "1440"))
    anomaly_threshold: float = float(os.getenv("RCA_ANOMALY_THRESHOLD", "0.65"))
    correlation_threshold: float = float(os.getenv("RCA_CORRELATION_THRESHOLD", "0.7"))
    
    default_metrics: List[str] = field(default_factory=lambda: [
        'container_cpu_usage_seconds_total',
        'container_memory_working_set_bytes',
        'kube_pod_container_status_restarts_total',
        'kube_pod_status_phase',
        'node_cpu_seconds_total',
        'node_memory_MemFree_bytes',
        'kubelet_http_requests_duration_seconds_count',
        'kubelet_http_requests_duration_seconds_sum'
    ])

@dataclass
class PredictionConfig:
    model_path: str = os.getenv("PREDICTION_MODEL_PATH", "data/models/time_qps_auto_scaling_model.pkl")
    scaler_path: str = os.getenv("PREDICTION_SCALER_PATH", "data/models/time_qps_auto_scaling_scaler.pkl")
    max_instances: int = int(os.getenv("PREDICTION_MAX_INSTANCES", "20"))
    min_instances: int = int(os.getenv("PREDICTION_MIN_INSTANCES", "1"))
    prometheus_query: str = os.getenv(
        "PREDICTION_PROMETHEUS_QUERY",
        'rate(nginx_ingress_controller_nginx_process_requests_total{service="ingress-nginx-controller-metrics"}[10m])'
    )

@dataclass
class NotificationConfig:
    feishu_webhook: str = os.getenv("FEISHU_WEBHOOK", "")
    enabled: bool = os.getenv("NOTIFICATION_ENABLED", "true").lower() == "true"

@dataclass
class TavilyConfig:
    api_key: str = os.getenv("TAVILY_API_KEY", "")
    max_results: int = int(os.getenv("TAVILY_MAX_RESULTS", "5"))

@dataclass
class AppConfig:
    debug: bool = os.getenv("DEBUG", "false").lower() == "true"
    host: str = os.getenv("HOST", "0.0.0.0")
    port: int = int(os.getenv("PORT", "8080"))
    log_level: str = os.getenv("LOG_LEVEL", "INFO")
    
    prometheus: PrometheusConfig = field(default_factory=PrometheusConfig)
    llm: LLMConfig = field(default_factory=LLMConfig)
    k8s: K8sConfig = field(default_factory=K8sConfig)
    rca: RCAConfig = field(default_factory=RCAConfig)
    prediction: PredictionConfig = field(default_factory=PredictionConfig)
    notification: NotificationConfig = field(default_factory=NotificationConfig)
    tavily: TavilyConfig = field(default_factory=TavilyConfig)

# 全局配置实例
config = AppConfig()