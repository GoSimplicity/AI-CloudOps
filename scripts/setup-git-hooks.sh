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

# 检查 git 仓库
check_git_repo() {
    if [ ! -d ".git" ]; then
        log_error "当前目录不是 Git 仓库"
        exit 1
    fi
}

# 显示设置结果
show_setup_result() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🎉 Git Hooks 设置完成！"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📝 注意："
    echo "  Swagger 自动生成功能已禁用"
    echo "  请使用以下命令手动生成文档："
    echo ""
    echo "  make swagger          - 生成 Swagger 文档"
    echo "  make swagger-manual   - 使用传统方式生成"
    echo "  make swagger-validate - 验证生成的文档"
    echo "  make swagger-clean    - 清理生成的文档"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# 主函数
main() {
    log_info "🚀 设置 AI-CloudOps Git Hooks..."
    
    check_git_repo
    
    log_warning "Swagger 自动生成功能已被禁用"
    log_info "文档生成已改为纯手动方式"
    
    show_setup_result
}

# 执行主函数
main "$@"
