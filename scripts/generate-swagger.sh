#!/bin/bash
# Swagger文档自动生成脚本

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$PROJECT_ROOT/docs"
TOOL_PATH="$PROJECT_ROOT/tools/swagger-auto-gen/swagger-auto-gen"

echo "🚀 开始生成 Swagger 文档..."
echo "📁 项目根目录: $PROJECT_ROOT"
echo "📄 输出目录: $DOCS_DIR"

# 检查工具是否存在
if [ ! -f "$TOOL_PATH" ]; then
    echo "⚠️  自动生成工具不存在，正在编译..."
    cd "$PROJECT_ROOT/tools/swagger-auto-gen"
    go build -o swagger-auto-gen main.go
    echo "✅ 工具编译完成"
fi

# 清理旧文档
echo "🧹 清理旧文档..."
rm -f "$DOCS_DIR"/swagger.json "$DOCS_DIR"/swagger.yaml "$DOCS_DIR"/docs.go

# 运行自动生成工具
echo "🔧 运行自动生成工具..."
cd "$PROJECT_ROOT"
"$TOOL_PATH" -root . -output ./docs -v

echo ""
echo "✅ Swagger 文档生成完成！"
echo "📊 文档统计:"
wc -l docs/swagger.json docs/swagger.yaml docs/docs.go | tail -1

echo ""
echo "📄 生成的文件:"
echo "  - docs/swagger.json  (JSON格式API文档)"
echo "  - docs/swagger.yaml  (YAML格式API文档)"
echo "  - docs/docs.go       (Go代码集成文档)"

echo ""
echo "🌐 访问 Swagger UI: http://localhost:8889/swagger/index.html"
