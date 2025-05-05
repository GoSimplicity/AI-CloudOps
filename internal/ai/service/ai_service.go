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
	"os"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/cloudwego/eino-ext/components/model/openai"
	enioModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type AIService interface {
	SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error)
	StreamChatMessage(ctx context.Context, message model.ChatMessage, responseChan chan<- model.StreamResponse) error
}

type aiService struct {
	logger *zap.Logger
}

func NewAIService(logger *zap.Logger) AIService {
	return &aiService{
		logger: logger,
	}
}

// SendChatMessage 发送聊天消息
func (a *aiService) SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error) {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 创建 OpenAI 聊天模型
	cm := a.createOpenAIChatModel(ctx)
	if cm == nil {
		return nil, fmt.Errorf("创建聊天模型失败")
	}

	// 根据消息内容创建模板消息
	messages := a.createMessagesFromRequest(message)

	// 获取流式响应
	streamResult := a.stream(ctx, cm, messages)
	if streamResult == nil {
		return nil, fmt.Errorf("获取流式响应失败")
	}

	// 处理流式响应
	content, err := a.reportStream(streamResult)
	if err != nil {
		return nil, err
	}

	// 组装响应
	return &model.ChatCompletionResponse{
		Answer: content,
	}, nil
}

// StreamChatMessage 流式发送聊天消息，结果通过channel返回
func (a *aiService) StreamChatMessage(ctx context.Context, message model.ChatMessage, responseChan chan<- model.StreamResponse) error {
	defer close(responseChan)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 创建 OpenAI 聊天模型
	cm := a.createOpenAIChatModel(ctx)
	if cm == nil {
		return fmt.Errorf("创建聊天模型失败")
	}

	// 根据消息内容创建模板消息
	messages := a.createMessagesFromRequest(message)

	// 获取流式响应
	streamResult := a.stream(ctx, cm, messages)
	if streamResult == nil {
		return fmt.Errorf("获取流式响应失败")
	}

	defer streamResult.Close()

	// 逐个处理流式响应并发送到channel
	for {
		message, err := streamResult.Recv()
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
			Content: message.Content,
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

func (a *aiService) stream(ctx context.Context, llm enioModel.ChatModel, in []*schema.Message) *schema.StreamReader[*schema.Message] {
	result, err := llm.Stream(ctx, in)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			a.logger.Error("llm生成超时", zap.Error(err))
		} else {
			a.logger.Error("llm生成失败", zap.Error(err))
		}
		return nil
	}
	return result
}

func (a *aiService) createOpenAIChatModel(ctx context.Context) enioModel.ChatModel {
	key := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   modelName,
		APIKey:  key,
		BaseURL: baseURL,
		Timeout: 25 * time.Second, // 设置单次请求超时时间
	})
	if err != nil {
		a.logger.Error("创建openai聊天模型失败", zap.Error(err))
		return nil
	}
	return chatModel
}

func (a *aiService) createMessagesFromRequest(message model.ChatMessage) []*schema.Message {
	template := a.createTemplate()

	// 使用模板生成消息
	messages, err := template.Format(context.Background(), map[string]any{
		"role":         message.Role,
		"style":        message.Style,
		"question":     message.Question,
		"chat_history": convertToSchemaMessages(message.ChatHistory),
	})
	if err != nil {
		a.logger.Error("生成消息失败", zap.Error(err))
		return nil
	}
	return messages
}

// 将自定义聊天历史转换为schema中的消息格式
func convertToSchemaMessages(history []model.HistoryMessage) []*schema.Message {
	var result []*schema.Message
	for _, msg := range history {
		if msg.Role == "user" {
			result = append(result, schema.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			result = append(result, schema.AssistantMessage(msg.Content, nil))
		}
	}
	return result
}

func (a *aiService) createTemplate() prompt.ChatTemplate {
	// 创建模板，使用 FString 格式
	return prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage("你是一个{role}。你需要用{style}规范的语气回答问题。你的目标是帮助使用AI-CloudOps开源项目的用户回答问题，同时提供一些有用的建议。"),

		// 插入需要的对话历史（新对话的话这里不填）
		schema.MessagesPlaceholder("chat_history", true),

		// 用户消息模板
		schema.UserMessage("问题: {question}"),
	)
}

func (a *aiService) reportStream(sr *schema.StreamReader[*schema.Message]) (string, error) {
	if sr == nil {
		return "", fmt.Errorf("流式响应为空")
	}

	defer sr.Close()

	var content string
	for {
		message, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		content += message.Content
	}

	return content, nil
}
