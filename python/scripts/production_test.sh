#!/bin/bash
# ===================================
# AIOps平台生产环境测试运行脚本
# 用于在生产环境中执行完整测试流程
# ===================================

set -e

# 获取脚本所在目录的绝对路径
SCRIPT_DIR=$(cd $(dirname $0) && pwd)
ROOT_DIR=$(cd $SCRIPT_DIR/.. && pwd)
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_DIR="$ROOT_DIR/logs"
REPORT_DIR="$ROOT_DIR/docs"
LOG_FILE="$LOG_DIR/production_test_$TIMESTAMP.log"

# 导入配置读取工具
source "$SCRIPT_DIR/config_reader.sh"

# 确保日志目录存在
mkdir -p $LOG_DIR
mkdir -p $REPORT_DIR

echo "=========================================================="
echo " AIOps平台生产环境测试 - $(date)"
echo "=========================================================="
echo "项目根目录: $ROOT_DIR"
echo "测试日志: $LOG_FILE"

# 创建日志函数
log() {
  local message="[$(date +"%Y-%m-%d %H:%M:%S")] $1"
  echo "$message" | tee -a "$LOG_FILE"
}

# 确认当前环境是否已准备好进行测试
confirm_test_environment() {
  log "确认测试环境..."
  
  # 检查 python 可用性
  python3 --version > /dev/null 2>&1 || {
    log "错误: 找不到 Python3"
    exit 1
  }
  
  # 检查 pytest 是否已安装
  python3 -c "import pytest" > /dev/null 2>&1 || {
    log "错误: 找不到 pytest，请先安装依赖"
    exit 1
  }
  
  # 确保配置文件存在
  if [[ ! -f "$ROOT_DIR/config/config.production.yaml" ]]; then
    log "错误: 找不到生产环境配置文件 (config/config.production.yaml)"
    exit 1
  fi
  
  # 设置生产环境配置文件路径
  export CONFIG_FILE="$ROOT_DIR/config/config.production.yaml"
  
  # 读取配置
  read_config
  
  # 检查是否能访问 Prometheus
  if [[ -z "$PROMETHEUS_HOST" ]]; then
    log "警告: 无法从配置文件获取Prometheus主机地址"
  else
    log "测试Prometheus连接 ($PROMETHEUS_HOST)..."
    curl -s -o /dev/null "http://$PROMETHEUS_HOST/api/v1/status/config" || {
      log "警告: 无法连接到Prometheus服务"
    }
  fi
  
  # 检查是否能访问Kubernetes
  kubectl version --client > /dev/null 2>&1 || {
    log "警告: kubectl未正确安装或配置"
  }
  
  log "环境检查完成"
}

# 设置生产测试环境变量
setup_test_env() {
  log "设置生产测试环境变量..."
  
  # 设置测试环境
  export ENV=test
  
  # 创建测试配置文件，基于生产环境配置
  mkdir -p "$ROOT_DIR/config"
  if [[ ! -f "$ROOT_DIR/config/config.test.yaml" ]]; then
    cp "$ROOT_DIR/config/config.production.yaml" "$ROOT_DIR/config/config.test.yaml"
    
    # 修改测试配置
    # 使用Python和PyYAML修改配置文件
    python3 -c "
import yaml
import os

test_config_file = '$ROOT_DIR/config/config.test.yaml'
with open(test_config_file, 'r') as f:
    config = yaml.safe_load(f)

# 设置测试特定配置
if 'testing' not in config:
    config['testing'] = {}
config['testing']['skip_llm_tests'] = False

# 确保真实服务
config['app']['debug'] = True
config['app']['log_level'] = 'INFO'

with open(test_config_file, 'w') as f:
    yaml.dump(config, f)
"
    
    log "已创建测试配置文件: config/config.test.yaml"
  fi
  
  # 设置测试环境配置文件并读取
  export CONFIG_FILE="$ROOT_DIR/config/config.test.yaml"
  read_config
  
  log "环境变量设置完成"
  log "应用主机: $APP_HOST"
  log "应用端口: $APP_PORT"
  log "Prometheus主机: $PROMETHEUS_HOST"
}

# 运行单元测试
run_unit_tests() {
  log "运行单元测试..."
  
  cd $ROOT_DIR
  python3 -m pytest tests/test_health.py -v
  
  log "单元测试完成"
}

# 运行集成测试
run_integration_tests() {
  log "运行集成测试..."
  
  cd $ROOT_DIR
  # 根据需要选择要运行的集成测试
  python3 -m pytest tests/test_rca.py tests/test_prediction.py tests/test_autofix.py -v
  
  log "集成测试完成"
}

# 运行端到端测试
run_e2e_tests() {
  log "运行端到端测试..."
  
  cd $ROOT_DIR
  python3 -m pytest tests/test_assistant.py tests/test_knowledge_load.py tests/test_websocket_assistant.py -v
  
  log "端到端测试完成"
}

# 运行全面的测试并生成报告
run_all_tests_with_report() {
  log "运行所有测试并生成报告..."
  
  cd $ROOT_DIR
  python3 scripts/run_tests.py
  
  log "全面测试完成，报告已生成"
}

# 主函数
main() {
  confirm_test_environment
  setup_test_env
  
  # 根据参数选择要运行的测试类型
  case "$1" in
    "unit")
      run_unit_tests
      ;;
    "integration")
      run_integration_tests
      ;;
    "e2e")
      run_e2e_tests
      ;;
    "all"|"")
      log "开始全面测试流程..."
      run_all_tests_with_report
      ;;
    *)
      log "未知的测试类型: $1"
      echo "用法: $0 [unit|integration|e2e|all]"
      exit 1
      ;;
  esac
  
  log "测试流程完成！报告位置: $REPORT_DIR/latest_test_report.md"
}

# 运行主函数
main "$@"