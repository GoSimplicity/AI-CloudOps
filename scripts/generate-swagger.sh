#!/bin/bash

# AI-CloudOps Swagger 文档生成脚本
# 设置环境变量并生成 Swagger 文档

set -e

echo "🚀 AI-CloudOps Swagger 文档生成器"
echo "=================================="

# 检查环境变量，如果没有设置则使用默认值
if [ -z "$SWAGGER_ENABLED" ]; then
    export SWAGGER_ENABLED=true
fi

if [ -z "$SWAGGER_VERBOSE" ]; then
    export SWAGGER_VERBOSE=true
fi

echo "⚙️  环境配置:"
echo "   SWAGGER_ENABLED=$SWAGGER_ENABLED"
echo "   SWAGGER_VERBOSE=$SWAGGER_VERBOSE"
echo ""

# 检查工具是否存在
if [ ! -f "tools/swagger-auto-gen/swagger-auto-gen" ]; then
    echo "🔧 构建生成工具..."
    cd tools/swagger-auto-gen && go build -o swagger-auto-gen . && cd ../..
fi

# 检查是否启用 Swagger 生成
if [ "$SWAGGER_ENABLED" = "false" ]; then
    echo "⏭️ Swagger文档生成已禁用，跳过生成过程"
    echo ""
    echo "💡 提示:"
    echo "   - 使用 'export SWAGGER_ENABLED=true' 重新启用"
    echo "   - 使用 'make swagger' 快速生成文档"
    exit 0
fi

# 生成文档
echo "📄 生成 Swagger 文档..."
./tools/swagger-auto-gen/swagger-auto-gen -root . -output ./docs -v

# 验证生成结果
echo ""
echo "✅ 验证生成结果..."
if [ -f "docs/swagger.json" ]; then
    echo "   ✅ swagger.json 已生成"
    echo "   📊 文件大小: $(du -h docs/swagger.json | cut -f1)"
else
    echo "   ❌ swagger.json 生成失败"
    exit 1
fi

if [ -f "docs/swagger.yaml" ]; then
    echo "   ✅ swagger.yaml 已生成"
    echo "   📊 文件大小: $(du -h docs/swagger.yaml | cut -f1)"
else
    echo "   ❌ swagger.yaml 生成失败"
    exit 1
fi

if [ -f "docs/docs.go" ]; then
    echo "   ✅ docs.go 已生成"
    echo "   📊 文件大小: $(du -h docs/docs.go | cut -f1)"
else
    echo "   ❌ docs.go 生成失败"
    exit 1
fi

echo ""
echo "🎉 Swagger 文档生成完成！"
echo "🌐 访问地址: http://localhost:8889/swagger/index.html"
echo ""
echo "💡 提示:"
echo "   - 使用 'go run main.go' 启动服务器"
echo "   - 使用 'make swagger' 快速生成文档"
echo "   - 使用 'go generate' 执行所有生成任务"
