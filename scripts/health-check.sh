#!/bin/bash

# AI-CloudOps 健康检查脚本 - 完整链路监控

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# 配置
BACKEND_URL="http://localhost:8889"
GATEWAY_URL="http://localhost:80"
AI_GRPC_URL="localhost:9000"
MYSQL_HOST="localhost"
MYSQL_PORT="3306"
REDIS_HOST="localhost"
REDIS_PORT="36379"
PROMETHEUS_URL="http://localhost:9090"

# 服务健康状态
declare -A SERVICE_STATUS

echo -e "${BLUE}🏥 AI-CloudOps 健康检查开始${NC}"
echo "========================================"

# 函数：检查单个服务
check_service() {
    local name="$1"
    local check_command="$2"
    local description="$3"
    
    echo -e "${YELLOW}🔍 检查 $name...${NC}"
    
    if eval "$check_command" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $name: 健康 - $description${NC}"
        SERVICE_STATUS["$name"]="healthy"
        return 0
    else
        echo -e "${RED}❌ $name: 不健康 - $description${NC}"
        SERVICE_STATUS["$name"]="unhealthy"
        return 1
    fi
}

# 函数：检查端口
check_port() {
    local host="$1"
    local port="$2"
    
    if command -v nc >/dev/null 2>&1; then
        nc -z "$host" "$port"
    else
        # 使用telnet作为备选
        timeout 3 bash -c "echo >/dev/tcp/$host/$port" 2>/dev/null
    fi
}

# 函数：检查HTTP服务
check_http() {
    local url="$1"
    curl -s -f "$url" >/dev/null
}

# 函数：获取服务详细状态
get_service_details() {
    echo ""
    echo -e "${BLUE}📊 详细服务状态报告${NC}"
    echo "========================================"
    
    # Go后端服务详情
    echo -e "${PURPLE}🔧 Go Backend Service${NC}"
    if curl -s "$BACKEND_URL/" | jq . 2>/dev/null; then
        echo -e "${GREEN}  状态: 运行正常${NC}"
    else
        echo -e "${RED}  状态: 响应异常${NC}"
    fi
    
    # AI服务详情
    echo -e "${PURPLE}🤖 AI Service${NC}"
    if curl -s "$BACKEND_URL/api/v1/health" | jq . 2>/dev/null; then
        echo -e "${GREEN}  状态: AI服务健康${NC}"
    else
        echo -e "${RED}  状态: AI服务异常${NC}"
    fi
    
    # gRPC服务详情
    echo -e "${PURPLE}🔌 gRPC Service${NC}"
    if check_port "localhost" "9000"; then
        echo -e "${GREEN}  状态: gRPC端口开放${NC}"
    else
        echo -e "${RED}  状态: gRPC端口关闭${NC}"
    fi
    
    # 网关详情
    echo -e "${PURPLE}🌐 API Gateway${NC}"
    if check_port "localhost" "80"; then
        echo -e "${GREEN}  状态: 网关端口开放${NC}"
        if curl -s "$GATEWAY_URL/" >/dev/null; then
            echo -e "${GREEN}  路由: 正常工作${NC}"
        else
            echo -e "${RED}  路由: 响应异常${NC}"
        fi
    else
        echo -e "${RED}  状态: 网关端口关闭${NC}"
    fi
}

# 函数：生成健康检查报告
generate_report() {
    local healthy_count=0
    local total_count=0
    
    echo ""
    echo -e "${BLUE}📋 健康检查总结报告${NC}"
    echo "========================================"
    
    for service in "${!SERVICE_STATUS[@]}"; do
        status="${SERVICE_STATUS[$service]}"
        total_count=$((total_count + 1))
        
        if [ "$status" == "healthy" ]; then
            healthy_count=$((healthy_count + 1))
            echo -e "${GREEN}✅ $service${NC}"
        else
            echo -e "${RED}❌ $service${NC}"
        fi
    done
    
    echo ""
    echo -e "${BLUE}总体健康度: $healthy_count/$total_count${NC}"
    
    if [ $healthy_count -eq $total_count ]; then
        echo -e "${GREEN}🎉 所有服务运行正常!${NC}"
        return 0
    elif [ $healthy_count -gt $((total_count / 2)) ]; then
        echo -e "${YELLOW}⚠️  大部分服务正常，但有部分服务需要注意${NC}"
        return 1
    else
        echo -e "${RED}🚨 多个关键服务异常，需要紧急处理!${NC}"
        return 2
    fi
}

# 主要健康检查流程
main() {
    echo -e "${BLUE}🔍 开始全面健康检查...${NC}"
    echo ""
    
    # 1. 基础设施层检查
    echo -e "${PURPLE}🏗️  基础设施层检查${NC}"
    echo "--------------------------------"
    
    check_service "MySQL数据库" \
        "check_port '$MYSQL_HOST' '$MYSQL_PORT'" \
        "数据持久化存储"
    
    check_service "Redis缓存" \
        "check_port '$REDIS_HOST' '$REDIS_PORT'" \
        "缓存和会话存储"
    
    check_service "Prometheus监控" \
        "check_http '$PROMETHEUS_URL'" \
        "指标收集和监控"
    
    echo ""
    
    # 2. 应用服务层检查  
    echo -e "${PURPLE}🚀 应用服务层检查${NC}"
    echo "--------------------------------"
    
    check_service "Go后端服务" \
        "check_http '$BACKEND_URL/'" \
        "主要业务逻辑API"
    
    check_service "AI健康检查端点" \
        "check_http '$BACKEND_URL/api/v1/health'" \
        "AI服务健康状态"
    
    check_service "gRPC服务端口" \
        "check_port 'localhost' '9000'" \
        "AI服务gRPC通信"
    
    echo ""
    
    # 3. 网关层检查
    echo -e "${PURPLE}🌐 网关层检查${NC}" 
    echo "--------------------------------"
    
    check_service "API网关" \
        "check_port 'localhost' '80'" \
        "统一API入口"
    
    check_service "网关路由" \
        "check_http '$GATEWAY_URL/'" \
        "请求路由和负载均衡"
    
    echo ""
    
    # 4. 端到端连通性检查
    echo -e "${PURPLE}🔄 端到端连通性检查${NC}"
    echo "--------------------------------"
    
    check_service "后端->AI服务连通" \
        "curl -s -f '$BACKEND_URL/api/v1/health'" \
        "后端调用AI服务"
    
    check_service "网关->后端连通" \
        "curl -s -f '$GATEWAY_URL/api/v1/health'" \
        "网关代理后端服务"
    
    # 获取详细信息
    get_service_details
    
    # 生成报告
    local exit_code
    generate_report
    exit_code=$?
    
    echo ""
    echo -e "${BLUE}💡 建议操作:${NC}"
    if [ $exit_code -eq 0 ]; then
        echo "• 系统运行良好，建议定期执行健康检查"
        echo "• 可以进行功能测试和性能优化"
    elif [ $exit_code -eq 1 ]; then  
        echo "• 检查异常服务的日志文件"
        echo "• 确认网络连接和端口配置"
        echo "• 重启异常服务"
    else
        echo "• 立即检查系统资源使用情况"
        echo "• 查看所有服务日志排查问题"
        echo "• 考虑重新部署整个系统"
    fi
    
    return $exit_code
}

# 检查依赖工具
echo -e "${YELLOW}🔧 检查依赖工具...${NC}"
if ! command -v curl >/dev/null 2>&1; then
    echo -e "${RED}❌ curl 未安装，请先安装 curl${NC}"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  建议安装 jq 以获得更好的JSON输出格式${NC}"
fi

# 运行健康检查
main "$@"
