#!/bin/bash

# Protocol Buffer 代码生成脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)
PROTO_DIR="${PROJECT_ROOT}/proto"
OUT_DIR="${PROJECT_ROOT}"

echo -e "${GREEN}开始生成 Protocol Buffer 代码...${NC}"

# 检查依赖
check_dependency() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}错误: $1 未安装${NC}"
        exit 1
    fi
}

echo -e "${YELLOW}检查依赖...${NC}"
check_dependency "protoc"
check_dependency "protoc-gen-go"
check_dependency "protoc-gen-go-grpc"

# 创建输出目录
mkdir -p "${OUT_DIR}/proto/aiops/v1"

# 生成 Go 代码
echo -e "${YELLOW}生成 Go 代码...${NC}"
protoc \
    --proto_path="${PROTO_DIR}" \
    --go_out="${OUT_DIR}" \
    --go_opt=paths=source_relative \
    --go-grpc_out="${OUT_DIR}" \
    --go-grpc_opt=paths=source_relative \
    "${PROTO_DIR}/aiops/v1/aiops_core.proto"

echo -e "${GREEN}✅ Protocol Buffer 代码生成完成!${NC}"

# 验证生成的文件
GENERATED_FILES=(
    "${OUT_DIR}/proto/aiops/v1/aiops_core.pb.go"
    "${OUT_DIR}/proto/aiops/v1/aiops_core_grpc.pb.go"
)

echo -e "${YELLOW}验证生成的文件...${NC}"
for file in "${GENERATED_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        echo -e "${GREEN}✅ $file${NC}"
    else
        echo -e "${RED}❌ $file - 文件未生成${NC}"
        exit 1
    fi
done

echo -e "${GREEN}🎉 所有文件生成成功!${NC}"
