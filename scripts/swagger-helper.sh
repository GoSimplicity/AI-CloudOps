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

# 检查依赖
check_dependencies() {
    log_info "检查依赖工具..."
    
    if ! command -v swag &> /dev/null; then
        log_error "swag 工具未安装，请运行: go install github.com/swaggo/swag/cmd/swag@latest"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go 工具未安装"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 生成 Swagger 文档
generate_docs() {
    log_info "生成 Swagger API 文档..."
    
    # 清理旧文档
    rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
    
    # 生成新文档
    swag init --output ./docs --parseDependency --parseInternal --exclude ./internal/*/service --dir ./ --generalInfo main.go
    
    if [ -f "docs/swagger.json" ]; then
        local file_size=$(du -h docs/swagger.json | cut -f1)
        log_success "文档生成成功！文件大小: $file_size"
        log_info "访问地址: http://localhost:8080/swagger/index.html"
    else
        log_error "文档生成失败！"
        exit 1
    fi
}

# 检查 Swagger 注解完整性
check_annotations() {
    log_info "检查 Swagger 注解完整性..."
    
    # 统计 API 函数
    local total_funcs=$(find internal/*/api -name "*.go" -exec grep -l "func.*Handler" {} \; | xargs grep -c "func.*Handler.*" | awk -F: '{sum += $2} END {print sum}')
    local swagger_funcs=$(find internal/*/api -name "*.go" -exec grep -c "@Summary" {} \; | awk '{sum += $1} END {print sum}')
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 Swagger 注解统计报告"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "总 API 处理函数数量: $total_funcs"
    echo "包含 @Summary 注解的函数: $swagger_funcs"
    
    if [ "$swagger_funcs" -lt "$total_funcs" ]; then
        local missing=$((total_funcs - swagger_funcs))
        log_warning "发现 $missing 个函数缺少 Swagger 注解"
        
        echo ""
        echo "缺少注解的文件:"
        find internal/*/api -name "*.go" -exec grep -L "@Summary" {} \; 2>/dev/null || true
        
        echo ""
        log_info "建议运行 'bash scripts/swagger-helper.sh fix' 来自动修复"
    else
        log_success "所有 API 函数都包含 Swagger 注解！"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# 验证生成的文档
validate_docs() {
    log_info "验证生成的文档..."
    
    # 检查文件是否存在
    if [ ! -f "docs/swagger.json" ]; then
        log_error "swagger.json 不存在，请先生成文档"
        return 1
    fi
    
    if [ ! -f "docs/swagger.yaml" ]; then
        log_error "swagger.yaml 不存在，请先生成文档"
        return 1
    fi
    
    # 统计 API 数量
    local api_count=$(grep -o '"paths"' docs/swagger.json | wc -l)
    local method_count=$(grep -o '"get"\|"post"\|"put"\|"delete"\|"patch"' docs/swagger.json | wc -l)
    
    log_success "文档验证通过"
    echo "  - swagger.json: $(du -h docs/swagger.json | cut -f1)"
    echo "  - swagger.yaml: $(du -h docs/swagger.yaml | cut -f1)"
    echo "  - API 路径数量: $api_count"
    echo "  - HTTP 方法数量: $method_count"
}

# 启动服务器并打开 Swagger UI
serve_docs() {
    log_info "打开 Swagger UI..."
    
    # 检查服务器是否运行
    if ! pgrep -f "AI-CloudOps" > /dev/null; then
        log_warning "服务器未运行，请先启动服务器:"
        echo "  make dev    # 使用 Docker"
        echo "  go run main.go    # 直接运行"
        return 1
    fi
    
    local swagger_url="http://localhost:8080/swagger/index.html"
    log_info "Swagger UI 地址: $swagger_url"
    
    # 根据操作系统打开浏览器
    if command -v open > /dev/null; then
        open "$swagger_url"
    elif command -v xdg-open > /dev/null; then
        xdg-open "$swagger_url"
    else
        log_info "请手动在浏览器中打开: $swagger_url"
    fi
}

# 修复缺失的 Swagger 注解 (基础版本)
fix_annotations() {
    log_info "自动修复缺失的 Swagger 注解..."
    
    # 查找缺少注解的文件
    local files_to_fix=$(find internal/*/api -name "*.go" -exec grep -L "@Summary" {} \;)
    
    if [ -z "$files_to_fix" ]; then
        log_success "没有发现缺少注解的文件"
        return 0
    fi
    
    log_warning "以下文件需要添加 Swagger 注解:"
    echo "$files_to_fix"
    echo ""
    
    log_info "请手动为这些文件添加 Swagger 注解。"
    log_info "参考格式:"
    echo "// @Summary 功能描述"
    echo "// @Description 详细描述"
    echo "// @Tags 模块标签"
    echo "// @Accept json"
    echo "// @Produce json"
    echo "// @Param param_name param_location param_type required \"描述\""
    echo "// @Success 200 {object} utils.ApiResponse \"成功\""
    echo "// @Failure 400 {object} utils.ApiResponse \"参数错误\""
    echo "// @Security BearerAuth"
    echo "// @Router /api/path [method]"
}

# 显示帮助信息
show_help() {
    echo "AI-CloudOps Swagger 文档助手"
    echo ""
    echo "用法:"
    echo "  bash scripts/swagger-helper.sh <command>"
    echo ""
    echo "命令:"
    echo "  generate    生成 Swagger API 文档"
    echo "  check       检查 Swagger 注解完整性"
    echo "  validate    验证生成的文档"
    echo "  serve       启动服务器并打开 Swagger UI"
    echo "  fix         修复缺失的 Swagger 注解"
    echo "  all         执行完整的工作流 (generate + validate + check)"
    echo "  help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  bash scripts/swagger-helper.sh generate"
    echo "  bash scripts/swagger-helper.sh all"
}

# 主函数
main() {
    local command="${1:-help}"
    
    case "$command" in
        "generate")
            check_dependencies
            generate_docs
            ;;
        "check")
            check_annotations
            ;;
        "validate")
            validate_docs
            ;;
        "serve")
            serve_docs
            ;;
        "fix")
            fix_annotations
            ;;
        "all")
            check_dependencies
            generate_docs
            validate_docs
            check_annotations
            log_success "🎉 Swagger 文档工作流完成！"
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