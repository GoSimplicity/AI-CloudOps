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
	"fmt"
	"io"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
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
			Content: "你是一个{role}。你需要用{style}规范的语气回答问题。你的目标是帮助使用AI-CloudOps开源项目的用户回答问题，同时提供一些有用的建议。",
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

	// 创建消息
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个" + message.Role + "。你需要用" + message.Style + "规范的语气回答问题。你的目标是帮助使用AI-CloudOps开源项目的用户回答问题，同时提供一些有用的建议。",
		},
		{
			Role:    schema.User,
			Content: message.Question,
		},
	}

	// 使用agent的流式响应
	streamResult, err := a.agent.Stream(ctx, messages)
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
			if ctx.Err() == context.DeadlineExceeded {
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
