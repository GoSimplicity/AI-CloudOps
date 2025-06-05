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

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type AIService interface {
	SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error)
	StreamChatMessage(ctx context.Context, message model.ChatMessage, responseChan chan<- model.StreamResponse) error
}

type aiService struct {
	logger *zap.Logger
	agent  *react.Agent
}

func NewAIService(logger *zap.Logger, agent *react.Agent) AIService {
	return &aiService{
		logger: logger,
		agent:  agent,
	}
}

// SendChatMessage 发送聊天消息
func (a *aiService) SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()

	if a.agent == nil {
		return nil, fmt.Errorf("agent未初始化")
	}

	// 使用非流式响应
	result, err := a.agent.Generate(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个AI-CloudOps项目专家。你需要用严谨规范的语气回答问题。你的目标是帮助使用AI-CloudOps开源项目的用户回答问题，同时提供一些有用的建议。",
		},
		{
			Role:    schema.User,
			Content: message.Question,
		},
	})
	if err != nil {
		a.logger.Error("获取响应失败", zap.Error(err))
		return nil, fmt.Errorf("获取响应失败: %v", err)
	}

	// 组装响应
	return &model.ChatCompletionResponse{
		Answer: result.Content,
	}, nil
}

// StreamChatMessage 流式发送聊天消息，结果通过channel返回
func (a *aiService) StreamChatMessage(ctx context.Context, message model.ChatMessage, responseChan chan<- model.StreamResponse) error {
	defer close(responseChan)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()

	if a.agent == nil {
		responseChan <- model.StreamResponse{
			Error: "agent未初始化",
		}
		return fmt.Errorf("agent未初始化")
	}

	// 创建模版
	template := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个{role}，专注于云计算和DevOps领域的专家。你需要用{style}规范的语气回答问题，保持简洁和友好。你的目标是帮助使用AI-CloudOps开源项目的用户回答问题，同时提供一些有用的建议和最佳实践。请基于事实回答，如果不确定，请明确表示。你可以解释复杂的技术概念，提供代码示例，并引导用户解决云环境中的常见问题。记住，你的回答应该既有教育意义又有实用价值，帮助用户更好地理解和使用AI-CloudOps。请使用Markdown格式输出，包括适当的标题、列表、代码块和强调，使回答更加结构化和易于阅读。"),
		schema.MessagesPlaceholder("history_key", false), // 消息占位符
		&schema.Message{
			Role:    schema.User,
			Content: "我现在在使用AI-CloudOps平台，请为我下面的问题提供帮助: {question}",
		},
	)

	// 获取历史记录
	histories := make([]*schema.Message, 0, len(message.ChatHistory))
	for i := range message.ChatHistory {
		histories = append(histories, &schema.Message{
			Role:    schema.RoleType(message.ChatHistory[i].Role),
			Content: message.ChatHistory[i].Content,
		})
	}

	// 准备变量
	variables := map[string]any{
		"role":        "AI-CloudOps项目专家",
		"style":       "严谨、专业",
		"history_key": histories,
		"question":    message.Question,
	}

	// 格式化模板
	messages, err := template.Format(ctx, variables)
	if err != nil {
		a.logger.Error("格式化模板发生错误", zap.Error(err))
		responseChan <- model.StreamResponse{
			Error: fmt.Sprintf("格式化模板失败: %v", err),
		}
		return fmt.Errorf("格式化模板失败: %v", err)
	}

	// 使用agent的流式响应
	streamResult, err := a.agent.Stream(context.Background(), messages, agent.WithComposeOptions(compose.WithCallbacks(&LoggerCallback{})))
	if err != nil {
		a.logger.Error("获取流式响应失败", zap.Error(err))
		responseChan <- model.StreamResponse{
			Error: fmt.Sprintf("获取流式响应失败: %v", err),
		}
		return fmt.Errorf("获取流式响应失败: %v", err)
	}

	defer streamResult.Close()

	// 逐个处理流式响应并发送到channel
	for {
		chunk, err := streamResult.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				responseChan <- model.StreamResponse{
					Error: "请求超时",
				}
				return fmt.Errorf("请求超时")
			}
			responseChan <- model.StreamResponse{
				Error: err.Error(),
			}
			return err
		}
		responseChan <- model.StreamResponse{
			Content: chunk.Content,
			Done:    false,
		}
	}

	// 发送完成信号
	responseChan <- model.StreamResponse{
		Content: "",
		Done:    true,
	}

	return nil
}

type LoggerCallback struct {
	callbacks.HandlerBuilder
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	fmt.Println("==================")
	inputStr, _ := json.MarshalIndent(input, "", "  ")
	fmt.Printf("[OnStart] %s\n", string(inputStr))
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return ctx
}
