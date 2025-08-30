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

# 创建 pre-commit hook
create_pre_commit_hook() {
    local hooks_dir=".git/hooks"
    local pre_commit_file="$hooks_dir/pre-commit"
    
    log_info "创建 pre-commit git hook..."
    
    # 确保 hooks 目录存在
    mkdir -p "$hooks_dir"
    
    # 创建 pre-commit hook
    cat > "$pre_commit_file" << 'EOF'
#!/bin/bash

# AI-CloudOps Swagger 文档自动同步 Pre-commit Hook

set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "🔍 检查 Swagger 文档同步状态..."

# 检查是否有 swagger 相关文件的变更
if git diff --cached --name-only | grep -E "\.(go)$|swagger\.(json|yaml)$" > /dev/null; then
    echo "📄 检测到代码或文档变更，检查同步状态..."
    
    # 运行同步检查
    if [ -f "scripts/swagger-auto-sync.sh" ]; then
        # 验证当前同步状态
        if ! bash scripts/swagger-auto-sync.sh verify > /dev/null 2>&1; then
            echo "⚠️  检测到文档可能未同步，正在自动同步..."
            
            # 自动同步
            if bash scripts/swagger-auto-sync.sh sync; then
                echo "✅ 文档同步完成"
                
                # 将更新的 docs.go 加入到 staging area
                if [ -f "docs/docs.go" ]; then
                    git add docs/docs.go
                    echo "📝 已将更新的 docs.go 加入提交"
                fi
            else
                echo "❌ 文档同步失败，请手动修复后重新提交"
                echo "建议运行: bash scripts/swagger-auto-sync.sh force"
                exit 1
            fi
        else
            echo "✅ 文档同步状态正常"
        fi
    else
        echo "⚠️  未找到自动同步脚本，跳过同步检查"
    fi
else
    echo "📄 未检测到相关文件变更，跳过文档同步检查"
fi

echo "🚀 Pre-commit 检查完成"
EOF
    
    # 给 hook 添加执行权限
    chmod +x "$pre_commit_file"
    
    log_success "✅ Pre-commit hook 创建成功"
}

# 创建 post-merge hook
create_post_merge_hook() {
    local hooks_dir=".git/hooks"
    local post_merge_file="$hooks_dir/post-merge"
    
    log_info "创建 post-merge git hook..."
    
    # 创建 post-merge hook
    cat > "$post_merge_file" << 'EOF'
#!/bin/bash

# AI-CloudOps Swagger 文档自动同步 Post-merge Hook

set -e

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "$PROJECT_ROOT"

echo "🔄 合并后检查 Swagger 文档同步状态..."

# 检查是否有 swagger 相关文件的变更
if git diff HEAD~1 --name-only | grep -E "\.(go)$|swagger\.(json|yaml)$" > /dev/null; then
    echo "📄 检测到代码或文档变更，检查同步状态..."
    
    # 运行同步检查
    if [ -f "scripts/swagger-auto-sync.sh" ]; then
        if ! bash scripts/swagger-auto-sync.sh verify > /dev/null 2>&1; then
            echo "⚠️  检测到文档可能未同步，建议运行同步："
            echo "  bash scripts/swagger-auto-sync.sh sync"
            echo "  或者运行: make swagger"
        else
            echo "✅ 文档同步状态正常"
        fi
    fi
else
    echo "📄 未检测到相关文件变更"
fi

echo "🎉 Post-merge 检查完成"
EOF
    
    # 给 hook 添加执行权限
    chmod +x "$post_merge_file"
    
    log_success "✅ Post-merge hook 创建成功"
}

# 创建 swagger 同步提醒的 wrapper
create_swagger_watcher() {
    local watcher_file="scripts/swagger-watcher.sh"
    
    log_info "创建 Swagger 文档监控脚本..."
    
    cat > "$watcher_file" << 'EOF'
#!/bin/bash

# Swagger 文档开发时自动监控脚本

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🔍 启动 Swagger 文档开发监控..."
echo "📁 监控目录: $PROJECT_ROOT"
echo "🎯 监控文件: *.go, swagger.json, swagger.yaml"
echo "⏹️  按 Ctrl+C 退出"

# 启动自动同步监控
if [ -f "scripts/swagger-auto-sync.sh" ]; then
    bash scripts/swagger-auto-sync.sh watch
else
    echo "❌ 未找到 swagger-auto-sync.sh 脚本"
    exit 1
fi
EOF
    
    # 给脚本添加执行权限
    chmod +x "$watcher_file"
    
    log_success "✅ Swagger 监控脚本创建成功"
}

# 检查 git 仓库
check_git_repo() {
    if [ ! -d ".git" ]; then
        log_error "当前目录不是 Git 仓库"
        exit 1
    fi
}

# 备份现有的 hooks
backup_existing_hooks() {
    local hooks_dir=".git/hooks"
    local backup_dir=".git/hooks.backup.$(date +%Y%m%d_%H%M%S)"
    
    if [ -f "$hooks_dir/pre-commit" ] || [ -f "$hooks_dir/post-merge" ]; then
        log_info "备份现有的 Git hooks..."
        mkdir -p "$backup_dir"
        
        [ -f "$hooks_dir/pre-commit" ] && cp "$hooks_dir/pre-commit" "$backup_dir/"
        [ -f "$hooks_dir/post-merge" ] && cp "$hooks_dir/post-merge" "$backup_dir/"
        
        log_success "现有 hooks 已备份到: $backup_dir"
    fi
}

# 显示设置结果
show_setup_result() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🎉 Git Hooks 设置完成！"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "已设置的 Hooks:"
    echo "  ✅ pre-commit   - 提交前自动检查并同步文档"
    echo "  ✅ post-merge   - 合并后提醒检查文档同步状态"
    echo ""
    echo "可用的开发工具:"
    echo "  📜 scripts/swagger-auto-sync.sh  - 手动同步工具"
    echo "  🔍 scripts/swagger-watcher.sh    - 开发时监控工具"
    echo ""
    echo "使用方法:"
    echo "  # 手动同步文档"
    echo "  bash scripts/swagger-auto-sync.sh sync"
    echo ""
    echo "  # 开发时自动监控（推荐）"
    echo "  make swagger-watch"
    echo "  # 或者"
    echo "  bash scripts/swagger-watcher.sh"
    echo ""
    echo "  # 验证同步状态"
    echo "  bash scripts/swagger-auto-sync.sh verify"
    echo ""
    echo "  # 强制重新同步"
    echo "  bash scripts/swagger-auto-sync.sh force"
    echo ""
    echo "Makefile 命令:"
    echo "  make swagger      - 生成文档并自动同步"
    echo "  make swagger-watch - 启动文档监控"
    echo "  make swagger-all  - 完整的文档生成流程"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# 主函数
main() {
    log_info "🚀 开始设置 AI-CloudOps Swagger 自动同步..."
    
    check_git_repo
    backup_existing_hooks
    create_pre_commit_hook
    create_post_merge_hook
    create_swagger_watcher
    
    show_setup_result
}

# 执行主函数
main "$@"