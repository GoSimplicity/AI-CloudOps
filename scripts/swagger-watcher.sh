#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# 配置文件
WATCH_DIRS="internal/*/api main.go"
EXCLUDE_PATTERNS="*_test.go *.swp *.tmp"
DEBOUNCE_DELAY=2  # 防抖延迟（秒）
LAST_BUILD_TIME=0

# 日志函数
log_info() {
    echo -e "${BLUE}[WATCHER]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[WATCHER]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WATCHER]${NC} $1"
}

log_error() {
    echo -e "${RED}[WATCHER]${NC} $1"
}

log_build() {
    echo -e "${PURPLE}[BUILD]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖工具..."
    
    # 检查 swag
    if ! command -v swag &> /dev/null; then
        log_error "swag 工具未安装，请运行: go install github.com/swaggo/swag/cmd/swag@latest"
        exit 1
    fi
    
    # 检查文件监控工具
    if command -v fswatch &> /dev/null; then
        WATCHER_TOOL="fswatch"
        log_success "使用 fswatch 作为文件监控工具"
    elif command -v inotifywait &> /dev/null; then
        WATCHER_TOOL="inotifywait"
        log_success "使用 inotifywait 作为文件监控工具"
    else
        log_warning "未找到文件监控工具，尝试安装 fswatch..."
        if command -v brew &> /dev/null; then
            brew install fswatch
            WATCHER_TOOL="fswatch"
            log_success "fswatch 安装成功"
        else
            log_error "请安装文件监控工具："
            echo "  macOS: brew install fswatch"
            echo "  Linux: apt-get install inotify-tools 或 yum install inotify-tools"
            exit 1
        fi
    fi
}

# 生成 Swagger 文档
generate_swagger() {
    local current_time=$(date +%s)
    
    # 防抖：如果距离上次构建时间太短，则跳过
    if [ $((current_time - LAST_BUILD_TIME)) -lt $DEBOUNCE_DELAY ]; then
        log_warning "构建请求过于频繁，跳过..."
        return
    fi
    
    LAST_BUILD_TIME=$current_time
    
    log_build "检测到文件变化，正在自动重新生成 Swagger 文档（无需手动注释）..."
    
    # 清理旧文档
    rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
    
    # 构建自动生成工具（如果需要）
    if [ -d "tools/swagger-auto-gen" ] && [ ! -f "bin/swagger-auto-gen" ]; then
        log_info "构建自动生成工具..."
        cd tools/swagger-auto-gen && go build -o ../../bin/swagger-auto-gen . && cd "$PROJECT_ROOT"
    fi
    
    # 生成新文档
    local success=false
    if [ -f "bin/swagger-auto-gen" ]; then
        # 使用自动生成工具
        if ./bin/swagger-auto-gen -root . -output ./docs 2>/dev/null; then
            success=true
        else
            log_warning "自动生成失败，回退到传统方式..."
        fi
    fi
    
    if [ "$success" = false ]; then
        # 使用传统方式
        if swag init --output ./docs --parseDependency --parseInternal --exclude ./internal/*/service --dir ./ --generalInfo main.go 2>/dev/null; then
            success=true
        fi
    fi
    
    if [ "$success" = true ]; then
        # 获取统计信息
        local api_count=0
        local file_size="未知"
        
        if [ -f "docs/swagger.json" ]; then
            api_count=$(grep -c '"paths"' docs/swagger.json 2>/dev/null || echo "0")
            file_size=$(du -h docs/swagger.json | cut -f1)
        fi
        
        local timestamp=$(date "+%H:%M:%S")
        log_success "✅ [$timestamp] 文档生成成功 (API数量: $api_count, 大小: $file_size)"
    else
        log_error "❌ 文档生成失败"
    fi
}

# 使用 fswatch 监控（macOS）
watch_with_fswatch() {
    log_info "使用 fswatch 开始监控文件变化..."
    
    # 构建监控路径
    local watch_paths=""
    for dir in $WATCH_DIRS; do
        if [ -e "$dir" ]; then
            watch_paths="$watch_paths $dir"
        fi
    done
    
    if [ -z "$watch_paths" ]; then
        log_error "没有找到需要监控的目录"
        exit 1
    fi
    
    log_info "监控路径: $watch_paths"
    log_info "排除模式: $EXCLUDE_PATTERNS"
    
    # 初始生成一次
    generate_swagger
    
    # 开始监控
    fswatch -r --exclude='.*\.git.*' --exclude='.*_test\.go$' --exclude='.*\.swp$' --exclude='.*\.tmp$' --exclude='docs/.*' $watch_paths | while read file; do
        # 检查文件是否是 Go 文件或 main.go
        if [[ "$file" =~ \.go$ ]] && [[ ! "$file" =~ _test\.go$ ]]; then
            echo -e "${CYAN}[CHANGE]${NC} $file"
            generate_swagger &
        fi
    done
}

# 使用 inotifywait 监控（Linux）
watch_with_inotifywait() {
    log_info "使用 inotifywait 开始监控文件变化..."
    
    # 构建监控路径
    local watch_paths=""
    for dir in $WATCH_DIRS; do
        if [ -d "$dir" ]; then
            watch_paths="$watch_paths $dir"
        elif [ -f "$dir" ]; then
            watch_paths="$watch_paths $(dirname $dir)"
        fi
    done
    
    if [ -z "$watch_paths" ]; then
        log_error "没有找到需要监控的目录"
        exit 1
    fi
    
    log_info "监控路径: $watch_paths"
    
    # 初始生成一次
    generate_swagger
    
    # 开始监控
    inotifywait -m -r -e modify,create,delete --format '%w%f' --exclude='.*\.git.*|.*_test\.go$|.*\.swp$|.*\.tmp$|docs/.*' $watch_paths 2>/dev/null | while read file; do
        if [[ "$file" =~ \.go$ ]]; then
            echo -e "${CYAN}[CHANGE]${NC} $file"
            generate_swagger &
        fi
    done
}

# 信号处理
cleanup() {
    log_info "停止文件监控..."
    jobs -p | xargs -r kill 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT SIGTERM

# 显示帮助信息
show_help() {
    echo "Swagger 文档自动监控工具"
    echo ""
    echo "用法:"
    echo "  bash scripts/swagger-watcher.sh [options]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示帮助信息"
    echo "  -d, --delay N  设置防抖延迟（默认: ${DEBOUNCE_DELAY}秒）"
    echo ""
    echo "功能:"
    echo "  - 监控 API 相关文件变化"
    echo "  - 自动重新生成 Swagger 文档"
    echo "  - 支持防抖延迟，避免频繁构建"
    echo "  - 跨平台支持 (macOS/Linux)"
    echo ""
    echo "快捷键:"
    echo "  Ctrl+C  停止监控"
    echo ""
    echo "示例:"
    echo "  bash scripts/swagger-watcher.sh         # 开始监控"
    echo "  bash scripts/swagger-watcher.sh -d 5    # 使用5秒防抖延迟"
}

# 主函数
main() {
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -d|--delay)
                if [[ -n $2 ]] && [[ $2 =~ ^[0-9]+$ ]]; then
                    DEBOUNCE_DELAY=$2
                    shift 2
                else
                    log_error "延迟参数必须是数字"
                    exit 1
                fi
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                     Swagger 文档监控工具                        ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    
    # 检查依赖
    check_dependencies
    
    log_info "配置信息:"
    echo "  - 监控目录: $WATCH_DIRS"
    echo "  - 防抖延迟: ${DEBOUNCE_DELAY}秒"
    echo "  - 监控工具: $WATCHER_TOOL"
    echo ""
    
    log_info "开始监控... (按 Ctrl+C 停止)"
    echo ""
    
    # 根据工具类型选择监控方式
    case $WATCHER_TOOL in
        "fswatch")
            watch_with_fswatch
            ;;
        "inotifywait")
            watch_with_inotifywait
            ;;
        *)
            log_error "不支持的监控工具: $WATCHER_TOOL"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
