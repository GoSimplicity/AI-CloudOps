#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${RED}⚠️  WARNING: 自动监控功能已被禁用以防止循环生成问题${NC}"
echo -e "${YELLOW}💡 如需监控，请手动运行: bash scripts/swagger-auto-sync.sh watch${NC}"
echo -e "${GREEN}🔧 建议使用: make swagger (手动同步)${NC}"

exit 0