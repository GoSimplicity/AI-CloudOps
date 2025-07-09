#!/bin/bash
# =============================================
# 配置文件读取工具脚本
# 用于从YAML配置文件中读取配置项
# =============================================

# 获取配置文件路径
CONFIG_FILE="${CONFIG_FILE:-config/config.yaml}"
if [ "$ENV" = "production" ]; then
    CONFIG_FILE="config/config.production.yaml"
fi

# 读取YAML配置的辅助函数
get_yaml_value() {
    local file=$1
    local path=$2
    python3 -c "
import yaml
import sys
try:
    with open('$file', 'r') as f:
        data = yaml.safe_load(f)
    path_parts = '$path'.split('.')
    result = data
    for part in path_parts:
        result = result[part]
    print(result)
except Exception as e:
    print('', file=sys.stderr)
    exit(1)
"
}

# 获取应用配置
get_app_host() {
    get_yaml_value "$CONFIG_FILE" "app.host"
}

get_app_port() {
    get_yaml_value "$CONFIG_FILE" "app.port"
}

get_app_debug() {
    get_yaml_value "$CONFIG_FILE" "app.debug"
}

# 获取Prometheus配置
get_prometheus_host() {
    get_yaml_value "$CONFIG_FILE" "prometheus.host"
}

get_prometheus_timeout() {
    get_yaml_value "$CONFIG_FILE" "prometheus.timeout"
}

# 获取LLM配置
get_llm_provider() {
    get_yaml_value "$CONFIG_FILE" "llm.provider"
}

get_llm_model() {
    get_yaml_value "$CONFIG_FILE" "llm.model"
}

get_ollama_url() {
    get_yaml_value "$CONFIG_FILE" "llm.ollama_base_url"
}

# 主要配置读取函数
read_config() {
    local config_path="${1:-$CONFIG_FILE}"
    echo "正在读取配置: $config_path"
    
    # 导出配置变量
    export APP_HOST=$(get_app_host "$config_path")
    export APP_PORT=$(get_app_port "$config_path")
    export APP_DEBUG=$(get_app_debug "$config_path")
    export PROMETHEUS_HOST=$(get_prometheus_host "$config_path")
    export PROMETHEUS_TIMEOUT=$(get_prometheus_timeout "$config_path")
    export LLM_PROVIDER=$(get_llm_provider "$config_path")
    export LLM_MODEL=$(get_llm_model "$config_path")
    export OLLAMA_URL=$(get_ollama_url "$config_path")
    
    # 打印配置，如果是调试模式
    if [ "$APP_DEBUG" = "true" ] || [ "$DEBUG" = "true" ]; then
        echo "已读取配置:"
        echo "APP_HOST: $APP_HOST"
        echo "APP_PORT: $APP_PORT"
        echo "PROMETHEUS_HOST: $PROMETHEUS_HOST"
        echo "LLM_PROVIDER: $LLM_PROVIDER"
        echo "LLM_MODEL: $LLM_MODEL"
    fi
}

# 如果直接运行此脚本，则读取并显示配置
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    read_config "$@"
fi 