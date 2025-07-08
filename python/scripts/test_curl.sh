#!/bin/bash

# 简单的curl测试脚本，用于测试AI小助手API
# 作者: AI-CloudOps 团队

API_URL="http://localhost:8080/api/v1/assistant"
BASE_URL="http://localhost:8080/api/v1"

echo "===== 测试AI小助手API ====="

echo -e "\n1. 测试健康检查接口"
curl -s ${BASE_URL}/health | python -m json.tool

echo -e "\n2. 创建会话"
SESSION_RESPONSE=$(curl -s -X POST ${API_URL}/session)
echo $SESSION_RESPONSE | python -m json.tool
SESSION_ID=$(echo $SESSION_RESPONSE | python -c "import sys, json; try: print(json.load(sys.stdin)['data']['session_id']); except: print('')" 2>/dev/null)
echo "获取到会话ID: $SESSION_ID"

echo -e "\n3. 测试查询"
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "{\"question\":\"什么是AIOps?\",\"session_id\":\"$SESSION_ID\"}" \
  ${API_URL}/query | python -m json.tool

echo -e "\n4. 测试刷新知识库"
curl -s -X POST ${API_URL}/refresh | python -m json.tool

echo -e "\n5. 测试添加文档"
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"这是一个测试文档，用于测试AI小助手的知识库功能。\",\"metadata\":{\"source\":\"测试\",\"author\":\"测试脚本\"}}" \
  ${API_URL}/add-document | python -m json.tool

echo -e "\n===== 测试完成 ====="
