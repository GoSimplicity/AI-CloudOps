import os
import yaml
from dataclasses import dataclass, field
from typing import List, Optional, Dict, Any
from dotenv import load_dotenv
from pathlib import Path

# 定义项目根目录
ROOT_DIR = Path(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

# 加载环境变量
load_dotenv()

# 确定运行环境
ENV = os.getenv("ENV", "development")

# 加载配置文件
def load_config() -> Dict[str, Any]:
    """加载配置文件，优先使用环境对应的配置，如果不存在则使用默认配置"""
    config_file = ROOT_DIR / "config" / f"config{'.' + ENV if ENV != 'development' else ''}.yaml"
    default_config_file = ROOT_DIR / "config" / "config.yaml"
    
    try:
        if config_file.exists():
            with open(config_file, "r", encoding="utf-8") as f:
                return yaml.safe_load(f)
        elif default_config_file.exists():
            with open(default_config_file, "r", encoding="utf-8") as f:
                return yaml.safe_load(f)
        else:
            print(f"警告: 未找到配置文件 {config_file} 或 {default_config_file}，将使用环境变量默认值")
            return {}
    except Exception as e:
        print(f"加载配置文件出错: {e}")
        return {}

# 加载配置
CONFIG = load_config()

def get_env_or_config(env_key, config_path, default=None, transform=None):
    """从环境变量或配置文件获取值，支持类型转换"""
    parts = config_path.split('.')
    config_value = CONFIG
    for part in parts:
        config_value = config_value.get(part, {}) if isinstance(config_value, dict) else {}
    
    value = os.getenv(env_key) or config_value or default
    
    if transform and value is not None:
        if transform == bool and isinstance(value, str):
            return value.lower() == "true"
        return transform(value)
    return value

@dataclass
class PrometheusConfig:
    host: str = get_env_or_config("PROMETHEUS_HOST", "prometheus.host", "127.0.0.1:9090")
    timeout: int = get_env_or_config("PROMETHEUS_TIMEOUT", "prometheus.timeout", 30, int)
    
    @property
    def url(self) -> str:
        return f"http://{self.host}"

@dataclass
class LLMConfig:
    provider: str = (get_env_or_config("LLM_PROVIDER", "llm.provider", "openai")).split('#')[0].strip()
    model: str = get_env_or_config("LLM_MODEL", "llm.model", "Qwen/Qwen3-14B")
    task_model: str = get_env_or_config("LLM_TASK_MODEL", "llm.task_model", "Qwen/Qwen2.5-14B-Instruct")
    api_key: str = get_env_or_config("LLM_API_KEY", "llm.api_key", "sk-xxx")
    base_url: str = get_env_or_config("LLM_BASE_URL", "llm.base_url", "https://api.siliconflow.cn/v1")
    temperature: float = get_env_or_config("LLM_TEMPERATURE", "llm.temperature", 0.7, float)
    max_tokens: int = get_env_or_config("LLM_MAX_TOKENS", "llm.max_tokens", 2048, int)
    
    # Ollama 配置
    ollama_model: str = get_env_or_config("OLLAMA_MODEL", "llm.ollama_model", "qwen2.5:3b")
    ollama_base_url: str = get_env_or_config("OLLAMA_BASE_URL", "llm.ollama_base_url", "http://127.0.0.1:11434/v1")
    
    @property
    def effective_model(self) -> str:
        """根据提供商返回有效的模型名称"""
        if self.provider.lower() == "ollama":
            return self.ollama_model.split('#')[0].strip() if self.ollama_model else ""
        return self.model.split('#')[0].strip() if self.model else ""
    
    @property
    def effective_base_url(self) -> str:
        """根据提供商返回有效的基础URL"""
        if self.provider.lower() == "ollama":
            return self.ollama_base_url.split('#')[0].strip() if self.ollama_base_url else ""
        return self.base_url.split('#')[0].strip() if self.base_url else ""
    
    @property
    def effective_api_key(self) -> str:
        """根据提供商返回有效的API密钥"""
        return "ollama" if self.provider.lower() == "ollama" else self.api_key

@dataclass
class K8sConfig:
    in_cluster: bool = get_env_or_config("K8S_IN_CLUSTER", "kubernetes.in_cluster", False, bool)
    config_path: Optional[str] = get_env_or_config("K8S_CONFIG_PATH", "kubernetes.config_path") or str(ROOT_DIR / "deploy/kubernetes/config")
    namespace: str = get_env_or_config("K8S_NAMESPACE", "kubernetes.namespace", "default")

@dataclass
class RCAConfig:
    default_time_range: int = get_env_or_config("RCA_DEFAULT_TIME_RANGE", "rca.default_time_range", 30, int)
    max_time_range: int = get_env_or_config("RCA_MAX_TIME_RANGE", "rca.max_time_range", 1440, int)
    anomaly_threshold: float = get_env_or_config("RCA_ANOMALY_THRESHOLD", "rca.anomaly_threshold", 0.65, float)
    correlation_threshold: float = get_env_or_config("RCA_CORRELATION_THRESHOLD", "rca.correlation_threshold", 0.7, float)
    
    default_metrics: List[str] = field(default_factory=lambda: CONFIG.get("rca", {}).get("default_metrics", [
        'container_cpu_usage_seconds_total',
        'container_memory_working_set_bytes',
        'kube_pod_container_status_restarts_total',
        'kube_pod_status_phase',
        'node_cpu_seconds_total',
        'node_memory_MemFree_bytes',
        'kubelet_http_requests_duration_seconds_count',
        'kubelet_http_requests_duration_seconds_sum'
    ]))

@dataclass
class PredictionConfig:
    model_path: str = get_env_or_config("PREDICTION_MODEL_PATH", "prediction.model_path", "data/models/time_qps_auto_scaling_model.pkl")
    scaler_path: str = get_env_or_config("PREDICTION_SCALER_PATH", "prediction.scaler_path", "data/models/time_qps_auto_scaling_scaler.pkl")
    max_instances: int = get_env_or_config("PREDICTION_MAX_INSTANCES", "prediction.max_instances", 20, int)
    min_instances: int = get_env_or_config("PREDICTION_MIN_INSTANCES", "prediction.min_instances", 1, int)
    prometheus_query: str = get_env_or_config("PREDICTION_PROMETHEUS_QUERY", "prediction.prometheus_query", 
        'rate(nginx_ingress_controller_nginx_process_requests_total{service="ingress-nginx-controller-metrics"}[10m])')

@dataclass
class NotificationConfig:
    feishu_webhook: str = get_env_or_config("FEISHU_WEBHOOK", "notification.feishu_webhook", "")
    enabled: bool = get_env_or_config("NOTIFICATION_ENABLED", "notification.enabled", True, bool)

@dataclass
class TavilyConfig:
    api_key: str = get_env_or_config("TAVILY_API_KEY", "tavily.api_key", "")
    max_results: int = get_env_or_config("TAVILY_MAX_RESULTS", "tavily.max_results", 5, int)

@dataclass
class RAGConfig:
    """RAG智能小助手配置"""
    vector_db_path: str = get_env_or_config("RAG_VECTOR_DB_PATH", "rag.vector_db_path", "data/vector_db")
    collection_name: str = get_env_or_config("RAG_COLLECTION_NAME", "rag.collection_name", "aiops-assistant")
    knowledge_base_path: str = get_env_or_config("RAG_KNOWLEDGE_BASE_PATH", "rag.knowledge_base_path", "data/knowledge_base")
    chunk_size: int = get_env_or_config("RAG_CHUNK_SIZE", "rag.chunk_size", 1000, int)
    chunk_overlap: int = get_env_or_config("RAG_CHUNK_OVERLAP", "rag.chunk_overlap", 200, int)
    top_k: int = get_env_or_config("RAG_TOP_K", "rag.top_k", 4, int)
    similarity_threshold: float = get_env_or_config("RAG_SIMILARITY_THRESHOLD", "rag.similarity_threshold", 0.7, float)
    openai_embedding_model: str = get_env_or_config("RAG_OPENAI_EMBEDDING_MODEL", "rag.openai_embedding_model", "Pro/BAAI/bge-m3")
    ollama_embedding_model: str = get_env_or_config("RAG_OLLAMA_EMBEDDING_MODEL", "rag.ollama_embedding_model", "nomic-embed-text")
    max_context_length: int = get_env_or_config("RAG_MAX_CONTEXT_LENGTH", "rag.max_context_length", 4000, int)
    temperature: float = get_env_or_config("RAG_TEMPERATURE", "rag.temperature", 0.1, float)

    @property
    def effective_embedding_model(self) -> str:
        """根据LLM提供商返回有效的嵌入模型"""
        llm_provider = get_env_or_config("LLM_PROVIDER", "llm.provider", "openai").lower()
        return self.ollama_embedding_model if llm_provider == "ollama" else self.openai_embedding_model

@dataclass
class AppConfig:
    debug: bool = get_env_or_config("DEBUG", "app.debug", False, bool)
    host: str = get_env_or_config("HOST", "app.host", "0.0.0.0")
    port: int = get_env_or_config("PORT", "app.port", 8080, int)
    log_level: str = get_env_or_config("LOG_LEVEL", "app.log_level", "INFO")

    prometheus: PrometheusConfig = field(default_factory=PrometheusConfig)
    llm: LLMConfig = field(default_factory=LLMConfig)
    k8s: K8sConfig = field(default_factory=K8sConfig)
    rca: RCAConfig = field(default_factory=RCAConfig)
    prediction: PredictionConfig = field(default_factory=PredictionConfig)
    notification: NotificationConfig = field(default_factory=NotificationConfig)
    tavily: TavilyConfig = field(default_factory=TavilyConfig)
    rag: RAGConfig = field(default_factory=RAGConfig)

# 全局配置实例
config = AppConfig()