#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否需要同步
check_sync_needed() {
    local swagger_json="docs/swagger.json"
    local docs_go="docs/docs.go"
    
    if [ ! -f "$swagger_json" ]; then
        log_warning "swagger.json 不存在，需要生成文档"
        return 0
    fi
    
    if [ ! -f "$docs_go" ]; then
        log_warning "docs.go 不存在，需要同步"
        return 0
    fi
    
    # 检查文件修改时间
    local swagger_time=$(stat -f "%m" "$swagger_json" 2>/dev/null || echo "0")
    local docs_time=$(stat -f "%m" "$docs_go" 2>/dev/null || echo "0")
    
    if [ "$swagger_time" -gt "$docs_time" ]; then
        log_info "swagger.json 比 docs.go 更新，需要同步"
        return 0
    fi
    
    # 检查内容是否一致
    if ! grep -q "swagger.json" "$docs_go" > /dev/null 2>&1; then
        log_warning "docs.go 内容可能过期，需要同步"
        return 0
    fi
    
    return 1
}

# 自动同步文档
auto_sync() {
    log_info "🔄 开始自动同步 Swagger 文档..."
    
    if check_sync_needed; then
        log_info "执行同步操作..."
        
        # 运行 go generate
        if go generate ./...; then
            log_success "✅ 文档同步完成！"
            
            # 验证同步结果
            if [ -f "docs/docs.go" ] && [ -f "docs/swagger.json" ]; then
                local docs_size=$(du -h docs/docs.go | cut -f1)
                local swagger_size=$(du -h docs/swagger.json | cut -f1)
                log_info "docs.go 大小: $docs_size"
                log_info "swagger.json 大小: $swagger_size"
            fi
        else
            log_error "❌ 同步失败！"
            return 1
        fi
    else
        log_success "✅ 文档已是最新，无需同步"
    fi
}

# 监控模式 - 文件变化时自动同步
watch_mode() {
    log_info "🔍 启动文件监控模式..."
    log_info "监控目录: docs/"
    log_info "按 Ctrl+C 退出监控"
    
    # 检查是否有 fswatch 工具
    if command -v fswatch > /dev/null; then
        log_info "使用 fswatch 监控文件变化..."
        fswatch -o docs/swagger.json | while read f; do
            log_info "检测到 swagger.json 文件变化，开始同步..."
            auto_sync
        done
    elif command -v inotifywait > /dev/null; then
        log_info "使用 inotifywait 监控文件变化..."
        while true; do
            inotifywait -e modify docs/swagger.json 2>/dev/null && {
                log_info "检测到 swagger.json 文件变化，开始同步..."
                auto_sync
            }
        done
    else
        log_warning "未找到文件监控工具，使用轮询模式（每5秒检查一次）..."
        local last_check=0
        echo "⚠️  监控模式已被禁用以防止循环生成问题"
        echo "💡 请手动运行同步: bash scripts/swagger-auto-sync.sh sync"
        echo "🔧 或使用: make swagger"
        return 0
    fi
}

# 强制同步
force_sync() {
    log_info "🔄 强制同步 Swagger 文档..."
    
    # 重新生成 swagger 文档
    log_info "重新生成 swagger 文档..."
    if make swagger > /dev/null 2>&1; then
        log_success "✅ 强制同步完成！"
    else
        log_error "❌ 强制同步失败！"
        return 1
    fi
}

# 验证同步状态
verify_sync() {
    log_info "🔍 验证文档同步状态..."
    
    local swagger_json="docs/swagger.json"
    local docs_go="docs/docs.go"
    
    if [ ! -f "$swagger_json" ]; then
        log_error "❌ swagger.json 不存在"
        return 1
    fi
    
    if [ ! -f "$docs_go" ]; then
        log_error "❌ docs.go 不存在"
        return 1
    fi
    
    # 检查修改时间
    local swagger_time=$(stat -f "%m" "$swagger_json" 2>/dev/null || echo "0")
    local docs_time=$(stat -f "%m" "$docs_go" 2>/dev/null || echo "0")
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 文档同步状态报告"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "swagger.json 修改时间: $(date -r "$swagger_time" 2>/dev/null || echo "未知")"
    echo "docs.go 修改时间: $(date -r "$docs_time" 2>/dev/null || echo "未知")"
    echo "swagger.json 大小: $(du -h "$swagger_json" | cut -f1)"
    echo "docs.go 大小: $(du -h "$docs_go" | cut -f1)"
    
    if [ "$swagger_time" -le "$docs_time" ]; then
        log_success "✅ 文档同步状态正常"
    else
        log_warning "⚠️ docs.go 可能需要更新"
        echo "建议运行: bash scripts/swagger-auto-sync.sh sync"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# 安装文件监控工具
install_tools() {
    log_info "🔧 安装文件监控工具..."
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if ! command -v fswatch > /dev/null; then
            if command -v brew > /dev/null; then
                log_info "使用 Homebrew 安装 fswatch..."
                brew install fswatch
            else
                log_warning "请安装 Homebrew 或手动安装 fswatch"
            fi
        else
            log_success "fswatch 已安装"
        fi
    elif [[ "$OSTYPE" == "linux"* ]]; then
        # Linux
        if ! command -v inotifywait > /dev/null; then
            if command -v apt-get > /dev/null; then
                log_info "使用 apt 安装 inotify-tools..."
                sudo apt-get update && sudo apt-get install -y inotify-tools
            elif command -v yum > /dev/null; then
                log_info "使用 yum 安装 inotify-tools..."
                sudo yum install -y inotify-tools
            else
                log_warning "请手动安装 inotify-tools"
            fi
        else
            log_success "inotify-tools 已安装"
        fi
    fi
}

# 显示帮助信息
show_help() {
    echo "Swagger 文档自动同步工具"
    echo ""
    echo "用法:"
    echo "  bash scripts/swagger-auto-sync.sh <command>"
    echo ""
    echo "命令:"
    echo "  sync          自动检查并同步文档"
    echo "  watch         监控文件变化并自动同步"
    echo "  force         强制重新生成并同步"
    echo "  verify        验证当前同步状态"
    echo "  install       安装文件监控工具"
    echo "  help          显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  bash scripts/swagger-auto-sync.sh sync"
    echo "  bash scripts/swagger-auto-sync.sh watch"
    echo ""
    echo "集成到开发流程:"
    echo "  # 在 .bashrc 或 .zshrc 中添加别名"
    echo "  alias swagger-sync='bash scripts/swagger-auto-sync.sh sync'"
    echo "  alias swagger-watch='bash scripts/swagger-auto-sync.sh watch'"
}

# 主函数
main() {
    local command="${1:-help}"
    
    case "$command" in
        "sync")
            auto_sync
            ;;
        "watch")
            watch_mode
            ;;
        "force")
            force_sync
            ;;
        "verify")
            verify_sync
            ;;
        "install")
            install_tools
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
