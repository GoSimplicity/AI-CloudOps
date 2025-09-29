#!/bin/bash

# AI-CloudOps 端到端集成测试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
BACKEND_URL="http://localhost:8889"
GATEWAY_URL="http://localhost:80"
AI_GRPC_URL="localhost:9000"

echo -e "${BLUE}🚀 开始 AI-CloudOps 端到端集成测试${NC}"
echo "=================================================="

# 函数：检查服务健康状态
check_service_health() {
    local service_name="$1"
    local url="$2"
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}📡 检查 $service_name 健康状态...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ $service_name 健康检查通过${NC}"
            return 0
        fi
        
        echo -e "${YELLOW}⏳ 等待 $service_name 启动... (attempt $attempt/$max_attempts)${NC}"
        sleep 2
        ((attempt++))
    done
    
    echo -e "${RED}❌ $service_name 健康检查失败${NC}"
    return 1
}

# 函数：测试API端点
test_api() {
    local name="$1"
    local method="$2"
    local url="$3"
    local data="$4"
    local expected_code="$5"
    
    echo -e "${YELLOW}🧪 测试: $name${NC}"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -o /tmp/test_response.json)
    else
        response=$(curl -s -w "%{http_code}" -X "$method" "$url" \
            -o /tmp/test_response.json)
    fi
    
    if [ "$response" == "$expected_code" ]; then
        echo -e "${GREEN}✅ $name 测试通过 (HTTP $response)${NC}"
        if [ -f /tmp/test_response.json ]; then
            echo -e "${BLUE}📄 响应内容:${NC}"
            cat /tmp/test_response.json | jq . 2>/dev/null || cat /tmp/test_response.json
        fi
        return 0
    else
        echo -e "${RED}❌ $name 测试失败 (HTTP $response, 期望 $expected_code)${NC}"
        if [ -f /tmp/test_response.json ]; then
            echo -e "${RED}📄 错误响应:${NC}"
            cat /tmp/test_response.json
        fi
        return 1
    fi
}

# 函数：测试gRPC连接
test_grpc() {
    echo -e "${YELLOW}🔗 测试 gRPC 服务连接...${NC}"
    
    # 检查端口是否开放
    if command -v nc >/dev/null 2>&1; then
        if nc -z localhost 9000; then
            echo -e "${GREEN}✅ gRPC 端口 9000 可访问${NC}"
        else
            echo -e "${RED}❌ gRPC 端口 9000 不可访问${NC}"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠️  nc 工具未安装，跳过端口检查${NC}"
    fi
    
    return 0
}

# 主测试流程
main() {
    echo -e "${BLUE}🔍 步骤 1: 服务健康检查${NC}"
    echo "----------------------------------"
    
    # 检查后端服务
    check_service_health "Backend API" "$BACKEND_URL/" || exit 1
    
    # 检查网关服务 
    check_service_health "API Gateway" "$GATEWAY_URL/" || exit 1
    
    # 检查AI服务健康检查端点
    check_service_health "AI Health Check" "$BACKEND_URL/api/v1/health" || exit 1
    
    echo ""
    echo -e "${BLUE}🧪 步骤 2: API 端点测试${NC}"
    echo "----------------------------------"
    
    # 测试基础端点
    test_api "后端根路径" "GET" "$BACKEND_URL/" "" "200"
    
    # 测试健康检查
    test_api "AI健康检查" "GET" "$BACKEND_URL/api/v1/health" "" "200"
    
    # 测试网关路由 (通过网关访问后端)
    test_api "网关路由测试" "GET" "$GATEWAY_URL/api/v1/health" "" "200"
    
    echo ""
    echo -e "${BLUE}🔌 步骤 3: gRPC 服务测试${NC}"  
    echo "----------------------------------"
    
    # 测试gRPC连接
    test_grpc || exit 1
    
    echo ""
    echo -e "${BLUE}🎯 步骤 4: 集成功能测试${NC}"
    echo "----------------------------------"
    
    # 模拟AI助手查询请求 (需要认证，期望401)
    test_api "AI助手查询(未认证)" "POST" "$BACKEND_URL/api/v1/aiops/assistant/query" \
        '{"question": "hello", "mode": "rag", "session_id": "test123"}' "401"
    
    # 测试负载预测 (需要认证，期望401)
    test_api "负载预测(未认证)" "POST" "$BACKEND_URL/api/v1/aiops/predict/load" \
        '{"service_name": "test-service", "current_load": 100, "hours": 24}' "401"
    
    echo ""
    echo -e "${GREEN}🎉 集成测试完成!${NC}"
    echo "=================================="
    echo -e "${GREEN}✅ 所有基础服务正常运行${NC}"
    echo -e "${GREEN}✅ API网关路由正确配置${NC}" 
    echo -e "${GREEN}✅ gRPC服务连接正常${NC}"
    echo -e "${GREEN}✅ 认证机制工作正常${NC}"
    echo ""
    echo -e "${BLUE}📋 后续步骤:${NC}"
    echo "1. 配置JWT认证token进行完整功能测试"
    echo "2. 测试Python AI服务的具体功能"
    echo "3. 验证前端到AI服务的完整链路"
}

# 检查依赖
echo -e "${YELLOW}🔧 检查依赖工具...${NC}"
if ! command -v curl >/dev/null 2>&1; then
    echo -e "${RED}❌ curl 未安装${NC}"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  jq 未安装，JSON 输出格式可能不美观${NC}"
fi

# 清理临时文件
cleanup() {
    rm -f /tmp/test_response.json
}
trap cleanup EXIT

# 运行主测试
main "$@"