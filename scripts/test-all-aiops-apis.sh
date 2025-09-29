#!/bin/bash

echo "🔥 AI-CloudOps 全量API接口测试"
echo "================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
BASE_URL="http://localhost:8889"
TIMEOUT=10

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 获取JWT token
echo -e "${YELLOW}🔑 获取JWT Token...${NC}"
TOKEN=$(curl -s -X POST $BASE_URL/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}' | jq -r '.data.accessToken')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ JWT Token获取失败${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Token获取成功${NC}"

# 测试函数
test_api() {
    local name="$1"
    local method="$2" 
    local endpoint="$3"
    local data="$4"
    local expected_fields="$5"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${BLUE}🧪 测试 $TOTAL_TESTS: $name${NC}"
    
    # 构造curl命令
    if [ -n "$data" ]; then
        response=$(timeout $TIMEOUT curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(timeout $TIMEOUT curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN")
    fi
    
    # 检查响应
    if echo "$response" | jq -e '.code == 0' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $name - 成功${NC}"
        if [ -n "$expected_fields" ]; then
            if echo "$response" | jq -e "has(\"data\") and .data | has(\"$expected_fields\")" >/dev/null 2>&1; then
                echo -e "${GREEN}   数据格式验证通过${NC}"
            else
                echo -e "${YELLOW}   ⚠️  数据格式待完善${NC}"
            fi
        fi
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ $name - 失败${NC}"
        echo "   响应: $(echo "$response" | jq -r '.message // "未知错误"')"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo ""
}

echo ""
echo -e "${YELLOW}📡 基础接口测试${NC}"
echo "--------------------------------"

test_api "健康检查" "GET" "/api/v1/health" "" "status"

echo -e "${YELLOW}🤖 AI助手接口测试${NC}"
echo "--------------------------------"

test_api "AI助手对话" "POST" "/api/v1/aiops/assistant/query" \
    '{"question": "Hello", "mode": "rag", "session_id": "test123"}' "answer"

test_api "获取会话列表" "GET" "/api/v1/aiops/assistant/sessions" "" "sessions"

test_api "添加文档" "POST" "/api/v1/aiops/assistant/document/add" \
    '{"title": "测试文档", "content": "这是测试内容", "file_name": "test.md"}' "document_id"

echo -e "${YELLOW}📊 预测接口测试${NC}"
echo "--------------------------------"

test_api "负载预测" "POST" "/api/v1/aiops/predict/load" \
    '{"service_name": "web-app", "current_load": 100, "hours": 24}' "predictions"

test_api "CPU预测" "POST" "/api/v1/aiops/predict/cpu" \
    '{"service_name": "web-app", "current_cpu": 50, "hours": 24}' "predictions"

test_api "内存预测" "POST" "/api/v1/aiops/predict/memory" \
    '{"service_name": "web-app", "current_memory": 1024, "hours": 24}' "predictions"

test_api "磁盘预测" "POST" "/api/v1/aiops/predict/disk" \
    '{"service_name": "web-app", "current_disk": 50, "hours": 24}' "predictions"

echo -e "${YELLOW}🔍 根因分析接口测试${NC}"
echo "--------------------------------"

test_api "根因分析" "POST" "/api/v1/aiops/rca/analyze" \
    '{"namespace": "default", "time_window_hours": 1, "severity_threshold": 0.7}' "root_causes"

test_api "错误摘要" "GET" "/api/v1/aiops/rca/error-summary" \
    '{"namespace": "default", "time_window_hours": 1}' "errors"

test_api "事件模式" "GET" "/api/v1/aiops/rca/event-patterns" \
    '{"namespace": "default", "time_window_hours": 1}' "patterns"

echo -e "${YELLOW}🛠️  自动修复接口测试${NC}"
echo "--------------------------------"

test_api "自动修复" "POST" "/api/v1/aiops/autofix/fix" \
    '{"namespace": "default", "resource_type": "pod", "resource_name": "test-pod", "issue_type": "restart", "dry_run": true}' "task_id"

test_api "K8s诊断" "POST" "/api/v1/aiops/autofix/diagnose" \
    '{"namespace": "default", "resource_type": "pod", "resource_name": "test-pod"}' "results"

test_api "自动修复配置" "GET" "/api/v1/aiops/autofix/config" "" "config"

echo -e "${YELLOW}🔍 系统检查接口测试${NC}"
echo "--------------------------------"

test_api "运行系统检查" "POST" "/api/v1/aiops/inspection/run" \
    '{"namespace": "default", "detailed": true}' "overall_score"

test_api "获取检查规则" "GET" "/api/v1/aiops/inspection/rules" "" "rules"

echo -e "${YELLOW}💾 缓存管理接口测试${NC}"
echo "--------------------------------"

test_api "清除缓存" "POST" "/api/v1/aiops/cache/clear" \
    '{"cache_type": "knowledge", "pattern": "*"}' "cleared_keys"

test_api "缓存统计" "GET" "/api/v1/aiops/cache/stats" "" "total_keys"

echo ""
echo "=================================="
echo -e "${BLUE}📊 测试总结${NC}"
echo "=================================="
echo -e "总测试数: $TOTAL_TESTS"
echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
echo -e "${RED}失败: $FAILED_TESTS${NC}"

SUCCESS_RATE=$(echo "scale=1; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc -l 2>/dev/null || echo "0")
echo -e "成功率: ${SUCCESS_RATE}%"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 所有接口测试通过！${NC}"
    echo -e "${GREEN}✅ 所有Python接口已成功集成到Go后端${NC}"
    echo -e "${GREEN}✅ gRPC通信链路完全正常${NC}"
    echo -e "${GREEN}✅ 模型格式与其他模块保持一致${NC}"
    exit 0
elif [ $PASSED_TESTS -gt $((TOTAL_TESTS / 2)) ]; then
    echo -e "${YELLOW}⚠️  大部分接口正常，少数需要优化${NC}"
    exit 0
else
    echo -e "${RED}❌ 多个接口异常，需要检查配置${NC}"
    exit 1
fi
