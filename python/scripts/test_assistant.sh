#!/bin/bash
# 测试小助手API的脚本

# 获取脚本所在目录的绝对路径
SCRIPT_DIR=$(cd $(dirname $0) && pwd)
ROOT_DIR=$(cd $SCRIPT_DIR/.. && pwd)

# 导入配置读取工具
source "$SCRIPT_DIR/config_reader.sh"

# 读取配置
read_config

# 设置API基础URL，默认从配置文件读取
DEFAULT_URL="http://${APP_HOST}:${APP_PORT}/api/v1/assistant"
BASE_URL="${1:-$DEFAULT_URL}"
HEADER="Content-Type: application/json"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}          智能小助手API测试脚本         ${NC}"
echo -e "${BLUE}=========================================${NC}"
echo -e "${YELLOW}API地址: ${BASE_URL}${NC}"
echo ""

# 1. 测试创建会话
echo -e "${YELLOW}1. 测试创建会话...${NC}"
SESSION_RESPONSE=$(curl -s -X POST "${BASE_URL}/session" -H "${HEADER}")
# 使用更可靠的方式提取session_id
SESSION_ID=$(echo $SESSION_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['session_id'])" 2>/dev/null)

if [ -z "$SESSION_ID" ]; then
  echo -e "${RED}创建会话失败!${NC}"
  echo $SESSION_RESPONSE
  exit 1
else
  echo -e "${GREEN}创建会话成功! 会话ID: $SESSION_ID${NC}"
fi
echo ""

# 2. 测试知识库刷新
echo -e "${YELLOW}2. 测试知识库刷新...${NC}"
REFRESH_RESPONSE=$(curl -s -X POST "${BASE_URL}/refresh" -H "${HEADER}")
REFRESH_CODE=$(echo $REFRESH_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$REFRESH_CODE" = "0" ]; then
  echo -e "${GREEN}知识库刷新成功!${NC}"
else
  echo -e "${YELLOW}知识库刷新失败或部分成功，继续测试...${NC}"
  echo $REFRESH_RESPONSE
fi
echo ""

# 3. 测试基本查询
echo -e "${YELLOW}3. 测试基本查询...${NC}"
QUERY_DATA='{"question":"AIOps平台是什么?","session_id":"'$SESSION_ID'"}'
echo -e "${BLUE}发送查询: $QUERY_DATA${NC}"

QUERY_RESPONSE=$(curl -s -X POST "${BASE_URL}/query" -H "${HEADER}" -d "$QUERY_DATA")
QUERY_CODE=$(echo $QUERY_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$QUERY_CODE" = "0" ]; then
  echo -e "${GREEN}查询成功!${NC}"
  # 提取并显示回答
  ANSWER=$(echo $QUERY_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['answer'])" 2>/dev/null)
  echo -e "${BLUE}回答:${NC} $ANSWER"
else
  echo -e "${RED}查询失败!${NC}"
  echo $QUERY_RESPONSE
fi
echo ""

# 4. 测试上下文查询
echo -e "${YELLOW}4. 测试上下文查询...${NC}"
CONTEXT_DATA='{"question":"它有哪些核心功能?","session_id":"'$SESSION_ID'"}'
echo -e "${BLUE}发送上下文查询: $CONTEXT_DATA${NC}"

CONTEXT_RESPONSE=$(curl -s -X POST "${BASE_URL}/query" -H "${HEADER}" -d "$CONTEXT_DATA")
CONTEXT_CODE=$(echo $CONTEXT_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$CONTEXT_CODE" = "0" ]; then
  echo -e "${GREEN}上下文查询成功!${NC}"
  # 提取并显示回答
  ANSWER=$(echo $CONTEXT_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['answer'])" 2>/dev/null)
  echo -e "${BLUE}回答:${NC} $ANSWER"
else
  echo -e "${RED}上下文查询失败!${NC}"
  echo $CONTEXT_RESPONSE
fi
echo ""

# 5. 测试网络搜索增强查询 (如果支持)
echo -e "${YELLOW}5. 测试网络搜索增强查询...${NC}"
WEB_DATA='{"question":"什么是人工智能运维?","use_web_search":true,"session_id":"'$SESSION_ID'"}'
echo -e "${BLUE}发送网络搜索查询: $WEB_DATA${NC}"

WEB_RESPONSE=$(curl -s -X POST "${BASE_URL}/query" -H "${HEADER}" -d "$WEB_DATA")
WEB_CODE=$(echo $WEB_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$WEB_CODE" = "0" ]; then
  echo -e "${GREEN}网络搜索查询成功!${NC}"
  # 提取并显示回答
  ANSWER=$(echo $WEB_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['answer'])" 2>/dev/null)
  echo -e "${BLUE}回答:${NC} $ANSWER"
else
  echo -e "${RED}网络搜索查询失败或未启用!${NC}"
  echo $WEB_RESPONSE
fi
echo ""

# 6. 测试清除缓存
echo -e "${YELLOW}6. 测试清除缓存...${NC}"
CACHE_RESPONSE=$(curl -s -X POST "${BASE_URL}/clear-cache" -H "${HEADER}")
CACHE_CODE=$(echo $CACHE_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['code'])" 2>/dev/null)

if [ "$CACHE_CODE" = "0" ]; then
  echo -e "${GREEN}缓存清除成功!${NC}"
else
  echo -e "${RED}缓存清除失败!${NC}"
  echo $CACHE_RESPONSE
fi
echo ""

echo -e "${GREEN}测试完成!${NC}" 