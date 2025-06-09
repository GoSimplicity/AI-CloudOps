#!/bin/bash

# AIOps平台启动脚本

set -e

# 配置
APP_NAME="AIOps Platform"
APP_VERSION="1.0.0"
DEFAULT_PORT=8080
DEFAULT_HOST="0.0.0.0"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 显示横幅
show_banner() {
    echo -e "${BLUE}"
    cat << 'EOF'
    ___    ____  ____        
   /   |  /  _/ / __ \____   _____
  / /| |  / /  / / / / __ \ / ___/
 / ___ |_/ /  / /_/ / /_/ (__  ) 
/_/  |_/___/  \____/ .___/____/  
                  /_/            
EOF
    echo -e "${NC}"
    echo -e "${GREEN}$APP_NAME v$APP_VERSION${NC}"
    echo "智能运维根因分析与自动修复系统"
    echo "================================"
}

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    # 检查Python
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装"
        exit 1
    fi
    
    # 检查pip包
    python3 -c "import flask, pandas, numpy, sklearn" 2>/dev/null || {
        log_error "Python依赖包未完整安装，请运行: pip install -r requirements.txt"
        exit 1
    }
    
    log_info "✅ 依赖检查通过"
}

# 检查配置文件
check_config() {
    log_info "检查配置文件..."
    
    if [ ! -f ".env" ]; then
        log_warn ".env 文件不存在，从模板创建"
        cp .env.example .env
    fi
    
    # 检查关键配置
    if [ ! -f "app/main.py" ]; then
        log_error "应用主文件不存在"
        exit 1
    fi
    
    log_info "✅ 配置检查通过"
}

# 检查端口
check_port() {
    local port=${1:-$DEFAULT_PORT}
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_warn "端口 $port 已被占用"
        
        # 提供选择
        echo "选项："
        echo "1) 终止占用进程"
        echo "2) 使用其他端口"
        echo "3) 退出"
        
        read -p "请选择 (1-3): " choice
        case $choice in
            1)
                log_info "正在终止占用端口 $port 的进程..."
                lsof -ti:$port | xargs kill -9
                log_info "✅ 进程已终止"
                ;;
            2)
                read -p "请输入新的端口号: " new_port
                if [[ $new_port =~ ^[0-9]+$ ]] && [ $new_port -ge 1024 ] && [ $new_port -le 65535 ]; then
                    export PORT=$new_port
                    log_info "使用端口: $new_port"
                else
                    log_error "无效的端口号"
                    exit 1
                fi
                ;;
            3)
                log_info "退出启动"
                exit 0
                ;;
            *)
                log_error "无效选择"
                exit 1
                ;;
        esac
    fi
}

# 创建必要目录
create_directories() {
    log_info "创建必要目录..."
    
    mkdir -p logs
    mkdir -p data/models
    mkdir -p data/sample
    
    log_info "✅ 目录创建完成"
}

# 设置环境变量
setup_environment() {
    log_info "设置环境变量..."
    
    # 加载.env文件
    if [ -f ".env" ]; then
        set -a
        source .env
        set +a
        log_info "✅ 已加载 .env 配置"
    fi
    
    # 设置默认值
    export PYTHONPATH="${PYTHONPATH:-.}"
    export FLASK_APP="${FLASK_APP:-app.main:app}"
    export HOST="${HOST:-$DEFAULT_HOST}"
    export PORT="${PORT:-$DEFAULT_PORT}"
    
    log_debug "PYTHONPATH: $PYTHONPATH"
    log_debug "FLASK_APP: $FLASK_APP"
    log_debug "HOST: $HOST"
    log_debug "PORT: $PORT"
}

# 健康检查
health_check() {
    local max_attempts=30
    local attempt=1
    local url="http://${HOST}:${PORT}/api/v1/health"
    
    log_info "等待服务启动..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$url" > /dev/null 2>&1; then
            log_info "✅ 服务健康检查通过"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    log_error "❌ 服务启动超时"
    return 1
}

# 显示服务信息
show_service_info() {
    log_info "服务信息："
    echo "  - 应用名称: $APP_NAME"
    echo "  - 版本: $APP_VERSION"
    echo "  - 地址: http://${HOST}:${PORT}"
    echo "  - 健康检查: http://${HOST}:${PORT}/api/v1/health"
    echo "  - API文档: http://${HOST}:${PORT}/"
    echo ""
    echo "可用的API端点："
    echo "  - GET  /api/v1/health        - 健康检查"
    echo "  - GET  /api/v1/predict       - 负载预测"
    echo "  - POST /api/v1/rca           - 根因分析"
    echo "  - POST /api/v1/autofix       - 自动修复"
    echo ""
}

# 启动服务
start_service() {
    log_info "启动 $APP_NAME..."
    
    # 后台启动选项
    if [ "$1" = "--daemon" ] || [ "$1" = "-d" ]; then
        log_info "以守护进程模式启动..."
        nohup python3 app/main.py > logs/app.log 2>&1 &
        local pid=$!
        echo $pid > logs/app.pid
        log_info "✅ 服务已启动，PID: $pid"
        
        # 健康检查
        if health_check; then
            show_service_info
            log_info "日志文件: logs/app.log"
            log_info "停止服务: kill $pid 或运行 scripts/stop.sh"
        else
            log_error "服务启动失败，请查看日志: logs/app.log"
            exit 1
        fi
    else
        log_info "以前台模式启动..."
        show_service_info
        log_info "按 Ctrl+C 停止服务"
        echo ""
        
        # 前台启动
        python3 app/main.py
    fi
}

# 停止服务
stop_service() {
    if [ -f "logs/app.pid" ]; then
        local pid=$(cat logs/app.pid)
        if kill -0 $pid 2>/dev/null; then
            log_info "停止服务 (PID: $pid)..."
            kill $pid
            rm -f logs/app.pid
            log_info "✅ 服务已停止"
        else
            log_warn "服务进程不存在"
            rm -f logs/app.pid
        fi
    else
        log_warn "未找到PID文件"
    fi
}

# 重启服务
restart_service() {
    stop_service
    sleep 2
    start_service --daemon
}

# 显示状态
show_status() {
    if [ -f "logs/app.pid" ]; then
        local pid=$(cat logs/app.pid)
        if kill -0 $pid 2>/dev/null; then
            log_info "服务正在运行 (PID: $pid)"
            
            # 尝试健康检查
            local url="http://${HOST:-$DEFAULT_HOST}:${PORT:-$DEFAULT_PORT}/api/v1/health"
            if curl -f -s "$url" > /dev/null 2>&1; then
                log_info "✅ 服务健康"
            else
                log_warn "⚠️  服务可能异常"
            fi
        else
            log_warn "服务未运行（PID文件存在但进程不存在）"
            rm -f logs/app.pid
        fi
    else
        log_info "服务未运行"
    fi
}

# 显示帮助
show_help() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  start, -s          启动服务（前台模式）"
    echo "  start -d, --daemon 启动服务（守护进程模式）"
    echo "  stop               停止服务"
    echo "  restart            重启服务"
    echo "  status             显示服务状态"
    echo "  health             检查服务健康状态"
    echo "  logs               显示服务日志"
    echo "  help, -h, --help   显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 start           # 前台启动"
    echo "  $0 start -d        # 后台启动"
    echo "  $0 restart         # 重启服务"
    echo ""
}

# 显示日志
show_logs() {
    if [ -f "logs/app.log" ]; then
        tail -f logs/app.log
    else
        log_warn "日志文件不存在"
    fi
}

# 检查健康状态
check_health() {
    local url="http://${HOST:-$DEFAULT_HOST}:${PORT:-$DEFAULT_PORT}/api/v1/health"
    
    log_info "检查服务健康状态..."
    
    if command -v curl &> /dev/null; then
        if curl -f -s "$url"; then
            echo ""
            log_info "✅ 服务健康"
        else
            log_error "❌ 服务异常或未启动"
            exit 1
        fi
    else
        log_warn "curl 未安装，无法进行健康检查"
    fi
}

# 主函数
main() {
    # 显示横幅
    show_banner
    
    # 解析命令行参数
    case "${1:-start}" in
        "start"|"-s")
            check_dependencies
            check_config
            setup_environment
            check_port
            create_directories
            start_service "$2"
            ;;
        "stop")
            stop_service
            ;;
        "restart")
            restart_service
            ;;
        "status")
            setup_environment
            show_status
            ;;
        "health")
            setup_environment
            check_health
            ;;
        "logs")
            show_logs
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 处理中断信号
trap 'log_info "收到中断信号，正在退出..."; exit 130' INT TERM

# 运行主函数
main "$@"