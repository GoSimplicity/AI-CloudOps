#!/bin/bash
# ===================================
# AIOps平台生产环境启动脚本
# ===================================

set -e

# 获取脚本所在目录的绝对路径
SCRIPT_DIR=$(cd $(dirname $0) && pwd)
ROOT_DIR=$(cd $SCRIPT_DIR/.. && pwd)
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_DIR="$ROOT_DIR/logs"
LOG_FILE="$LOG_DIR/production_$TIMESTAMP.log"

# 导入配置读取工具
source "$SCRIPT_DIR/config_reader.sh"

# 确保日志目录存在
mkdir -p $LOG_DIR

echo "=========================================================="
echo " AIOps平台生产环境启动 - $(date)"
echo "=========================================================="
echo "项目根目录: $ROOT_DIR"
echo "日志文件: $LOG_FILE"

# 创建日志函数
log() {
  local message="[$(date +"%Y-%m-%d %H:%M:%S")] $1"
  echo "$message" | tee -a "$LOG_FILE"
}

# 检查环境配置
check_environment() {
  log "检查环境配置..."
  
  # 确保生产环境配置文件存在
  if [[ ! -f "$ROOT_DIR/env.production" ]]; then
    log "错误: 找不到生产环境配置文件 (env.production)"
    exit 1
  fi
  
  # 确保YAML配置文件存在
  if [[ ! -f "$ROOT_DIR/config/config.production.yaml" ]]; then
    log "错误: 找不到生产环境YAML配置文件 (config/config.production.yaml)"
    exit 1
  fi
  
  # 检查是否填写了必要的API密钥
  grep -q "your-api-key-here" "$ROOT_DIR/env.production" && {
    log "错误: 生产环境配置中包含默认API密钥，请先更新env.production文件"
    exit 1
  }
  
  # 检查Kubernetes配置
  if [[ -n "$(grep -q "kubernetes:" "$ROOT_DIR/config/config.production.yaml" && grep -q "in_cluster: true" "$ROOT_DIR/config/config.production.yaml")" ]] && \
     [[ ! -f "/var/run/secrets/kubernetes.io/serviceaccount/token" ]]; then
    log "警告: 配置了使用K8s集群内配置，但没有检测到ServiceAccount令牌"
    log "建议: 如果不在K8s集群中运行，请将kubernetes.in_cluster设置为false并提供配置文件路径"
  fi
  
  log "环境配置检查完成"
}

# 准备生产环境
prepare_environment() {
  log "准备生产环境..."
  
  # 设置环境变量
  export ENV=production
  export CONFIG_FILE="$ROOT_DIR/config/config.production.yaml"
  log "已设置环境变量: ENV=production"
  
  # 复制生产环境配置
  cp "$ROOT_DIR/env.production" "$ROOT_DIR/.env"
  log "已加载生产环境配置（敏感数据）"
  
  # 确保数据目录存在
  mkdir -p "$ROOT_DIR/data/vector_db"
  mkdir -p "$ROOT_DIR/data/models"
  
  # 读取配置文件
  read_config "$CONFIG_FILE"
  
  log "环境准备完成"
}

# 运行预检查
run_prechecks() {
  log "运行服务预检查..."
  
  # 使用配置中的Prometheus主机
  log "测试Prometheus连接 ($PROMETHEUS_HOST)..."
  if [[ -n "$PROMETHEUS_HOST" ]]; then
    curl -s -o /dev/null "http://$PROMETHEUS_HOST/api/v1/status/config" || {
      log "警告: 无法连接到Prometheus服务"
    }
  fi
  
  # 从YAML获取模型路径
  MODEL_PATH=$(grep -A 6 "prediction:" "$ROOT_DIR/config/config.production.yaml" | grep "model_path:" | awk -F': ' '{print $2}')
  
  # 检查模型文件
  if [[ -n "$MODEL_PATH" ]] && [[ ! -f "$ROOT_DIR/$MODEL_PATH" ]]; then
    log "警告: 预测模型文件不存在: $MODEL_PATH"
    log "预测功能可能无法正常工作"
  fi
  
  log "预检查完成"
}

# 刷新知识库
refresh_knowledge_base() {
  log "刷新RAG知识库..."
  
  # 调用知识库刷新API，使用配置文件中的主机和端口
  python3 -c "
import requests
try:
    response = requests.post('http://${APP_HOST}:${APP_PORT}/api/v1/assistant/refresh')
    print(f'知识库刷新状态: {response.status_code}')
except Exception as e:
    print(f'知识库刷新失败: {str(e)}')
" >> "$LOG_FILE" 2>&1 &
  
  log "知识库刷新进程已启动"
}

# 启动应用
start_application() {
  log "启动AIOps平台应用..."
  
  cd "$ROOT_DIR"
  
  # 使用生产环境配置
  export ENV=production
  
  # 加载敏感环境变量
  export $(grep -v '^#' "$ROOT_DIR/.env" | xargs)
  
  # 启动应用
  python3 -m app.main >> "$LOG_FILE" 2>&1 &
  PID=$!
  
  echo $PID > "$ROOT_DIR/.pid"
  log "应用已启动 (PID: $PID)"
  log "应用地址: http://${APP_HOST}:${APP_PORT}"
  log "日志文件: $LOG_FILE"
  
  # 等待应用启动
  log "等待应用启动..."
  sleep 10
  
  # 检查应用是否成功启动
  if ps -p $PID > /dev/null; then
    log "应用成功启动!"
    
    # 启动后刷新知识库
    refresh_knowledge_base
  else
    log "错误: 应用启动失败，请查看日志文件"
    exit 1
  fi
}

# 主函数
main() {
  check_environment
  prepare_environment
  run_prechecks
  start_application
  
  log "AIOps平台启动流程完成"
  echo ""
  echo "AIOps平台已成功启动! 可以通过以下命令查看日志:"
  echo "tail -f $LOG_FILE"
}

# 运行主函数
main "$@"