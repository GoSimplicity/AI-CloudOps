"""
AI-CloudOps 应用常量定义
"""

# ==================== 时间相关常量 ====================
DEFAULT_TIMEOUT_SECONDS = 30
MAX_RETRIES = 3
CACHE_EXPIRY_SECONDS = 3600
HEALTH_CHECK_INTERVAL = 60

# ==================== LLM 服务常量 ====================
LLM_TIMEOUT_SECONDS = 30
LLM_MAX_RETRIES = 3
OPENAI_TEST_MAX_TOKENS = 5
LLM_CONFIDENCE_THRESHOLD = 0.1
LLM_TEMPERATURE_MIN = 0.0
LLM_TEMPERATURE_MAX = 2.0

# ==================== 负载预测常量 ====================
LOW_QPS_THRESHOLD = 5.0
QPS_CHANGE_DIVISOR = 1.0
MAX_PREDICTION_HOURS = 168  # 7 天
DEFAULT_PREDICTION_HOURS = 24
PREDICTION_VARIATION_FACTOR = 0.1  # 10% 波动

# QPS 置信度阈值
QPS_CONFIDENCE_THRESHOLDS = {
    'low': 100,
    'medium': 500,
    'high': 1000,
    'very_high': 2000
}

# 时间模式常量
HOUR_FACTORS = {
    0: 0.3, 1: 0.2, 2: 0.15, 3: 0.1, 4: 0.1, 5: 0.2,
    6: 0.4, 7: 0.6, 8: 0.8, 9: 0.9, 10: 1.0, 11: 0.95,
    12: 0.9, 13: 0.95, 14: 1.0, 15: 1.0, 16: 0.95, 17: 0.9,
    18: 0.8, 19: 0.7, 20: 0.6, 21: 0.5, 22: 0.4, 23: 0.3
}

DAY_FACTORS = {
    0: 0.95,  # 周一
    1: 1.0,   # 周二
    2: 1.05,  # 周三
    3: 1.05,  # 周四
    4: 0.95,  # 周五
    5: 0.7,   # 周六
    6: 0.6    # 周日
}

# ==================== RAG 助手常量 ====================
DEFAULT_TOP_K = 4
MAX_CONTEXT_LENGTH = 4000
SIMILARITY_THRESHOLD = 0.1
MAX_HISTORY_LENGTH = 20
HALLUCINATION_COVERAGE_THRESHOLD = 0.3
DEFAULT_MAX_CONTEXT_DOCS = 6
MIN_RELEVANCE_SCORE = 0.6

# 向量数据库常量
VECTOR_DB_COLLECTION_NAME = "aiops_knowledge"
EMBEDDING_BATCH_SIZE = 50
VECTOR_SEARCH_TIMEOUT = 30

# ==================== RCA 分析常量 ====================
RCA_ANOMALY_THRESHOLD = 0.65
RCA_CORRELATION_THRESHOLD = 0.6
RCA_MAX_CANDIDATES = 10
RCA_MIN_CONFIDENCE = 0.5
RCA_HISTORICAL_LOOKBACK_DAYS = 30

# 异常检测算法常量
Z_SCORE_THRESHOLD = 2.5
ISOLATION_FOREST_CONTAMINATION = 0.1
DBSCAN_EPS = 0.5
DBSCAN_MIN_SAMPLES = 5

# ==================== API 响应常量 ====================
API_DEFAULT_PAGE_SIZE = 20
API_MAX_PAGE_SIZE = 100
API_REQUEST_TIMEOUT = 30
API_RATE_LIMIT_REQUESTS = 100
API_RATE_LIMIT_WINDOW = 60  # 秒

# HTTP 状态码
HTTP_STATUS_OK = 200
HTTP_STATUS_CREATED = 201
HTTP_STATUS_BAD_REQUEST = 400
HTTP_STATUS_UNAUTHORIZED = 401
HTTP_STATUS_FORBIDDEN = 403
HTTP_STATUS_NOT_FOUND = 404
HTTP_STATUS_INTERNAL_ERROR = 500

# ==================== Kubernetes 自动修复常量 ====================
K8S_MAX_REPLICAS = 50
K8S_MIN_REPLICAS = 1
K8S_DEFAULT_REPLICAS = 3
K8S_SCALE_UP_THRESHOLD = 0.8
K8S_SCALE_DOWN_THRESHOLD = 0.3
K8S_COOLDOWN_PERIOD = 300  # 5 分钟

# Pod 健康检查常量
DEFAULT_INITIAL_DELAY_SECONDS = 30
DEFAULT_PERIOD_SECONDS = 10
DEFAULT_TIMEOUT_SECONDS = 5
DEFAULT_FAILURE_THRESHOLD = 3
DEFAULT_SUCCESS_THRESHOLD = 1

# 资源配置常量
DEFAULT_CPU_REQUEST = "100m"
DEFAULT_MEMORY_REQUEST = "128Mi"
DEFAULT_CPU_LIMIT = "500m"
DEFAULT_MEMORY_LIMIT = "512Mi"

# ==================== 监控和告警常量 ====================
PROMETHEUS_QUERY_TIMEOUT = 30
PROMETHEUS_MAX_POINTS = 11000
PROMETHEUS_DEFAULT_STEP = "1m"

# 健康检查所需组件
REQUIRED_HEALTH_COMPONENTS = [
    "prometheus",
    "llm",
    "vector_store",
    "prediction"
]

# ==================== 日志常量 ====================
LOG_FORMAT = '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
LOG_MAX_BYTES = 10 * 1024 * 1024  # 10MB
LOG_BACKUP_COUNT = 5

# 日志级别
LOG_LEVELS = {
    'DEBUG': 10,
    'INFO': 20,
    'WARNING': 30,
    'ERROR': 40,
    'CRITICAL': 50
}

# ==================== 通知系统常量 ====================
NOTIFICATION_RETRY_ATTEMPTS = 3
NOTIFICATION_RETRY_DELAY = 5  # 秒
NOTIFICATION_TIMEOUT = 10  # 秒

# 通知严重程度
NOTIFICATION_SEVERITY = {
    'low': '低',
    'medium': '中',
    'high': '高',
    'critical': '紧急'
}

# ==================== 文件和路径常量 ====================
DEFAULT_KNOWLEDGE_BASE_PATH = "data/knowledge_base"
DEFAULT_VECTOR_DB_PATH = "data/vector_db"
DEFAULT_MODELS_PATH = "data/models"
DEFAULT_LOGS_PATH = "logs"
DEFAULT_CONFIG_PATH = "config"

# 支持的文档格式
SUPPORTED_DOC_FORMATS = [
    '.md', '.txt', '.pdf', '.csv', '.json', '.html', '.xml'
]

# ==================== 性能和限制常量 ====================
MAX_CONCURRENT_REQUESTS = 100
MAX_MEMORY_USAGE_MB = 1024
MAX_FILE_SIZE_MB = 100
MAX_BATCH_SIZE = 1000

# 缓存配置
CACHE_DEFAULT_TTL = 3600  # 1 小时
CACHE_MAX_SIZE = 1000
CACHE_EVICTION_POLICY = "LRU"

# ==================== 安全常量 ====================
MAX_LOGIN_ATTEMPTS = 5
SESSION_TIMEOUT = 3600  # 1 小时
PASSWORD_MIN_LENGTH = 8
TOKEN_EXPIRY_HOURS = 24

# API 密钥长度限制
MIN_API_KEY_LENGTH = 32
MAX_API_KEY_LENGTH = 256

# ==================== 模型和算法常量 ====================
MODEL_VERSION = "1.0"
MODEL_RETRAIN_INTERVAL_DAYS = 7
MODEL_ACCURACY_THRESHOLD = 0.8
MODEL_CONFIDENCE_THRESHOLD = 0.7

# 特征工程常量
TIME_WINDOW_MINUTES = 60
FEATURE_WINDOW_HOURS = 24
MAX_FEATURE_COUNT = 50

# ==================== 环境和部署常量 ====================
ENVIRONMENTS = ['development', 'staging', 'production']
DEFAULT_ENVIRONMENT = 'development'

# 资源配置建议
RESOURCE_REQUIREMENTS = {
    'small': {'cpu': '2', 'memory': '4Gi'},
    'medium': {'cpu': '4', 'memory': '8Gi'},
    'large': {'cpu': '8', 'memory': '16Gi'}
}

# ==================== 错误消息常量 ====================
ERROR_MESSAGES = {
    'invalid_input': '输入参数无效',
    'service_unavailable': '服务暂时不可用',
    'timeout': '请求超时',
    'not_found': '请求的资源未找到',
    'unauthorized': '未授权访问',
    'rate_limited': '请求频率超限',
    'internal_error': '内部服务错误'
}

# 成功消息
SUCCESS_MESSAGES = {
    'operation_completed': '操作成功完成',
    'data_updated': '数据更新成功',
    'analysis_finished': '分析完成',
    'model_trained': '模型训练完成'
}
