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
  
  # 检查是否填写了必要的API密钥
  grep -q "your-api-key-here" "$ROOT_DIR/env.production" && {
    log "错误: 生产环境配置中包含默认API密钥，请先更新env.production文件"
    exit 1
  }
  
  # 检查Kubernetes配置
  if [[ ! -f "/var/run/secrets/kubernetes.io/serviceaccount/token" ]] && \
     grep -q "K8S_IN_CLUSTER=true" "$ROOT_DIR/env.production"; then
    log "警告: 配置了使用K8s集群内配置，但没有检测到ServiceAccount令牌"
    log "建议: 如果不在K8s集群中运行，请将K8S_IN_CLUSTER设置为false并提供配置文件路径"
  fi
  
  log "环境配置检查完成"
}

# 准备生产环境
prepare_environment() {
  log "准备生产环境..."
  
  # 复制生产环境配置
  cp "$ROOT_DIR/env.production" "$ROOT_DIR/.env"
  log "已加载生产环境配置"
  
  # 确保数据目录存在
  mkdir -p "$ROOT_DIR/data/vector_db"
  mkdir -p "$ROOT_DIR/data/models"
  
  log "环境准备完成"
}

# 运行预检查
run_prechecks() {
  log "运行服务预检查..."
  
  # 测试Prometheus连接
  PROMETHEUS_HOST=$(grep -E "^PROMETHEUS_HOST=" "$ROOT_DIR/env.production" | cut -d= -f2 | tr -d '"')
  if [[ -n "$PROMETHEUS_HOST" ]]; then
    log "测试Prometheus连接 ($PROMETHEUS_HOST)..."
    curl -s -o /dev/null "http://$PROMETHEUS_HOST/api/v1/status/config" || {
      log "警告: 无法连接到Prometheus服务"
    }
  fi
  
  # 检查模型文件
  MODEL_PATH=$(grep -E "^PREDICTION_MODEL_PATH=" "$ROOT_DIR/env.production" | cut -d= -f2 | tr -d '"')
  if [[ -n "$MODEL_PATH" ]] && [[ ! -f "$MODEL_PATH" ]]; then
    log "警告: 预测模型文件不存在: $MODEL_PATH"
    log "预测功能可能无法正常工作"
  fi
  
  log "预检查完成"
}

# 刷新知识库
refresh_knowledge_base() {
  log "刷新RAG知识库..."
  
  # 调用知识库刷新API
  python3 -c "
import requests
try:
    response = requests.post('http://localhost:8080/api/v1/assistant/refresh')
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
  export $(grep -v '^#' "$ROOT_DIR/.env" | xargs)
  
  # 启动应用
  python3 -m app.main >> "$LOG_FILE" 2>&1 &
  PID=$!
  
  echo $PID > "$ROOT_DIR/.pid"
  log "应用已启动 (PID: $PID)"
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