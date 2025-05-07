/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package di

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func InitAgent() *react.Agent {
	ctx := context.Background()
	baseUrl := os.Getenv("OPENAI_BASE_URL")
	apiKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseUrl,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
		Model:   modelName,
		// Options: &api.Options{
		// 	Temperature: 0.7,
		// 	NumPredict:  100,
		// },
	})
	if err != nil {
		fmt.Printf("初始化OpenAI模型失败: %v\n", err)
		return nil
	}

	var allMcpTools []tool.BaseTool

	// 初始化MCP客户端
	cliMcp, err := client.NewSSEMCPClient(os.Getenv("MCP_URL"))
	if err != nil {
		fmt.Printf("初始化MCP客户端失败: %v\n", err)
		return nil
	}

	// 启动MCP客户端
	err = cliMcp.Start(ctx)
	if err != nil {
		fmt.Printf("启动MCP客户端失败: %v\n", err)
		return nil
	}

	// 初始化MCP请求
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "AI-CloudOps",
		Version: "1.0.0",
	}

	// 初始化MCP请求
	_, err = cliMcp.Initialize(ctx, initRequest)
	if err != nil {
		fmt.Printf("初始化MCP请求失败: %v\n", err)
		return nil
	}

	// 获取MCP工具
	mcpTools, err := einomcp.GetTools(ctx, &einomcp.Config{
		Cli: cliMcp,
	})
	if err != nil {
		fmt.Printf("获取MCP工具失败: %v\n", err)
		return nil
	}
	allMcpTools = append(allMcpTools, mcpTools...)
	fmt.Printf("成功加载MCP工具，获取到 %d 个工具\n", len(mcpTools))

	if len(allMcpTools) == 0 {
		fmt.Println("没有加载到任何MCP工具，返回nil")
		return nil
	}

	fmt.Printf("总共加载了 %d 个MCP工具\n", len(allMcpTools))

	// 自定义工具调用检查器，检查所有流式输出中是否包含工具调用
	toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		defer sr.Close()
		for {
			msg, err := sr.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// 流结束
					break
				}
				return false, err
			}

			if len(msg.ToolCalls) > 0 {
				return true, nil
			}
		}
		return false, nil
	}

	// 创建React Agent
	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: allMcpTools,
		},
		StreamToolCallChecker: toolCallChecker,
	})
	if err != nil {
		fmt.Printf("创建React Agent失败: %v\n", err)
		return nil
	}

	// 确保返回的Agent不为nil
	if reactAgent == nil {
		fmt.Println("警告: 创建的React Agent为nil")
	} else {
		fmt.Println("成功创建React Agent")
	}

	return reactAgent
}
