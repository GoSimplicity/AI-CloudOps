#!/usr/bin/env python3
"""在隔离 AIOps 容器内验证 MCP 鉴权与确定性工具，不调用模型或基础设施变更"""

import asyncio
import os

import aiohttp

from app.mcp.mcp_client import MCPSessionClient


async def main():
    endpoint = os.environ.get("MCP_SERVER_URL", "http://mcp:9000").rstrip("/")
    client = MCPSessionClient(endpoint)
    if not await client.health_check():
        raise RuntimeError("MCP health failed")
    tools = await client.get_available_tools()
    if not tools:
        raise RuntimeError("MCP tool list is empty")
    result = await client.execute_tool("calculate", {"expression": "2 + 3 * 4"})
    if result.get("result") != 14:
        raise RuntimeError("MCP calculator result mismatch")
    async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=10)) as session:
        async with session.get(endpoint + "/tools") as response:
            if response.status != 401:
                raise RuntimeError("MCP anonymous tool access was not rejected")
    print("PASS internal MCP health, authenticated tool listing, calculator and anonymous 401")


if __name__ == "__main__":
    asyncio.run(main())
