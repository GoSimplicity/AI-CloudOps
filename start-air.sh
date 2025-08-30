#!/bin/bash

# Air 启动脚本 - 确保环境变量正确设置
echo "🚀 启动 Air 开发服务器"
echo "========================"

# 设置默认值
export SWAGGER_ENABLED=${SWAGGER_ENABLED:-false}
export SWAGGER_VERBOSE=${SWAGGER_VERBOSE:-false}

echo "环境变量设置:"
echo "  SWAGGER_ENABLED=$SWAGGER_ENABLED"
echo "  SWAGGER_VERBOSE=$SWAGGER_VERBOSE"

echo ""
echo "💡 使用说明:"
echo "  - 设置 SWAGGER_ENABLED=false 禁用 Swagger 生成"
echo "  - 设置 SWAGGER_ENABLED=true 启用 Swagger 生成"
echo "  - 例如: SWAGGER_ENABLED=false ./start-air.sh"
echo ""

# 启动 Air
echo "🎯 启动 Air..."
air
