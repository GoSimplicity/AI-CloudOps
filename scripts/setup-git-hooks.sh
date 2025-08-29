#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}[SETUP]${NC} 设置 Git Hooks 自动生成 Swagger 文档..."

# 检查是否在 Git 仓库中
if [ ! -d ".git" ]; then
    echo -e "${RED}[ERROR]${NC} 当前目录不是 Git 仓库"
    exit 1
fi

# 创建 Git hooks 目录
mkdir -p .git/hooks

# 复制 pre-commit hook
if [ -f ".githooks/pre-commit" ]; then
    cp .githooks/pre-commit .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    echo -e "${GREEN}[SUCCESS]${NC} pre-commit hook 已安装"
else
    echo -e "${RED}[ERROR]${NC} .githooks/pre-commit 文件不存在"
    exit 1
fi

# 检查 swag 工具
echo -e "${BLUE}[SETUP]${NC} 检查 swag 工具..."
if ! command -v swag &> /dev/null; then
    echo -e "${YELLOW}[WARNING]${NC} swag 工具未安装，正在安装..."
    go install github.com/swaggo/swag/cmd/swag@latest
    if command -v swag &> /dev/null; then
        echo -e "${GREEN}[SUCCESS]${NC} swag 工具安装成功"
    else
        echo -e "${RED}[ERROR]${NC} swag 工具安装失败"
        echo -e "${YELLOW}[INFO]${NC} 请手动安装: go install github.com/swaggo/swag/cmd/swag@latest"
    fi
else
    echo -e "${GREEN}[SUCCESS]${NC} swag 工具已安装"
fi

# 测试 hook
echo -e "${BLUE}[SETUP]${NC} 测试 pre-commit hook..."
if .git/hooks/pre-commit --help &>/dev/null || true; then
    echo -e "${GREEN}[SUCCESS]${NC} pre-commit hook 测试成功"
else
    echo -e "${YELLOW}[WARNING]${NC} pre-commit hook 可能存在问题，但已安装"
fi

echo ""
echo -e "${GREEN}[SUCCESS]${NC} 🎉 Git Hooks 设置完成！"
echo ""
echo -e "${BLUE}[INFO]${NC} 功能说明:"
echo "  ✅ 当修改 API 文件时，提交代码会自动生成 Swagger 文档"
echo "  ✅ 生成的文档会自动添加到暂存区"
echo "  ✅ 支持增量更新，只在有 API 变更时生成"
echo ""
echo -e "${BLUE}[INFO]${NC} 如需禁用自动生成："
echo "  git commit --no-verify -m 'your message'"
echo ""
echo -e "${BLUE}[INFO]${NC} 手动生成文档："
echo "  make swagger"
echo "  bash scripts/swagger-helper.sh generate"
