#!/usr/bin/env bash
# 配置本仓库使用 .githooks/ 下的 Git 钩子，并检查 gitleaks 是否已安装。
# 用法: scripts/setup-git-hooks.sh
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1" >&2; }

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    log_error "当前目录不是 Git 仓库: $PROJECT_ROOT"
    exit 1
fi

chmod +x .githooks/* 2>/dev/null || true

# 在 linked worktree 中写入 worktree 级配置，避免修改主仓库/其他 worktree 共享的 .git/config。
git_dir="$(cd "$(git rev-parse --git-dir)" && pwd)"
common_dir="$(cd "$(git rev-parse --git-common-dir)" && pwd)"
if [ "$git_dir" != "$common_dir" ]; then
    if [ "$(git config --type=bool --get extensions.worktreeConfig 2>/dev/null || echo false)" != "true" ]; then
        log_error "当前是 linked worktree，且未启用 extensions.worktreeConfig。"
        log_error "为避免改动共享配置，请先执行: git config extensions.worktreeConfig true  后重试。"
        exit 1
    fi
    git config --worktree core.hooksPath .githooks
    log_info "已设置 (worktree 级) core.hooksPath=.githooks"
else
    git config core.hooksPath .githooks
    log_info "已设置 core.hooksPath=.githooks"
fi

if command -v gitleaks >/dev/null 2>&1; then
    log_success "gitleaks 已安装: $(gitleaks version)"
else
    log_error "未找到 gitleaks。pre-commit 钩子会因此阻止提交。"
    log_error "请从 https://github.com/gitleaks/gitleaks/releases 下载官方二进制，校验 sha256 后放入 PATH（如 ~/.local/bin）。"
    exit 1
fi

if [ -f package.json ] && grep -q '"husky"' package.json; then
    log_warning "本仓库依赖 husky：执行 pnpm install 时 husky 可能把 core.hooksPath 改回 .husky/_，之后请重新运行本脚本。"
fi

log_success "Git hooks 设置完成（提交前将运行 gitleaks 密钥扫描）。"
