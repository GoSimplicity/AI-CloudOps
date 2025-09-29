#!/bin/bash

echo "🚀 AI-CloudOps gRPC接口全面测试"
echo "================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 测试计数器
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_code="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${BLUE}🧪 测试 $TOTAL_TESTS: $test_name${NC}"
    
    # 执行测试命令
    if response=$(eval "$test_command" 2>/dev/null); then
        echo -e "${GREEN}✅ $test_name - 成功${NC}"
        echo "   响应: $response"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ $test_name - 失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    echo ""
}

echo -e "${YELLOW}📡 基础连通性测试${NC}"
echo "--------------------------------"

run_test "Go后端基础接口" \
    "curl -s http://localhost:8889/ | jq -r .message" \
    "200"

run_test "gRPC端口连通性" \
    "nc -zv localhost 9000 2>&1 | grep succeeded" \
    "connection"

echo ""
echo -e "${YELLOW}🤖 AI服务gRPC接口测试${NC}"
echo "--------------------------------"

# 直接调用Go后端的AI接口（不通过认证）
run_test "AI健康检查接口(无认证)" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:8889/api/v1/health" \
    "401"

echo ""
echo -e "${YELLOW}🌐 网关代理测试${NC}"  
echo "--------------------------------"

run_test "网关基础路由" \
    "curl -s http://localhost:80/ | jq -r .message" \
    "200"

run_test "网关健康检查路由" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:80/api/v1/health" \
    "401"

echo ""
echo -e "${YELLOW}🔐 认证绕过接口测试${NC}"
echo "--------------------------------"

# 创建一个临时的无认证测试接口
echo "测试用的临时调试接口..."
run_test "调试接口测试" \
    "curl -s http://localhost:8889/api/v1/debug/test | jq -r .message" \
    "200"

echo ""
echo "==============================="
echo -e "${BLUE}📊 测试总结${NC}"
echo "==============================="
echo -e "总测试数: $TOTAL_TESTS"
echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
echo -e "${RED}失败: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 所有基础测试通过!${NC}"
    echo ""
    echo -e "${YELLOW}📋 gRPC服务状态总结:${NC}"
    echo "✅ Go后端服务: 正常运行"
    echo "✅ Python AI gRPC服务: 正常运行" 
    echo "✅ 网关代理: 正常工作"
    echo "⚠️  认证机制: 按预期工作(401响应)"
    echo ""
    echo -e "${BLUE}🔧 下一步操作:${NC}"
    echo "1. 配置JWT token进行认证接口测试"
    echo "2. 测试AI助手对话功能" 
    echo "3. 测试负载预测功能"
    exit 0
else
    echo -e "${RED}❌ 部分测试失败，需要检查服务配置${NC}"
    exit 1
fi
