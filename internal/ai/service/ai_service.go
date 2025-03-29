package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/ai/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type AIService interface {
	SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error)
	SendStreamingChatMessage(ctx context.Context, message model.ChatMessage, callback func(model.ChunkChatCompletionResponse) error) error
	UploadFile(ctx context.Context, filePath string, user string) (*model.FileUploadResponse, error)
	StopResponse(ctx context.Context, taskID string, user string) error
	SendFeedback(ctx context.Context, messageID string, rating string, user string, content string) error
	GetSuggestedQuestions(ctx context.Context, messageID string, user string) ([]string, error)
	GetMessages(ctx context.Context, conversationID string, user string, firstID string, limit int) (map[string]interface{}, error)
	GetConversations(ctx context.Context, user string, lastID string, limit int, sortBy string) (map[string]interface{}, error)
	DeleteConversation(ctx context.Context, conversationID string, user string) error
	RenameConversation(ctx context.Context, conversationID string, name string, autoGenerate bool, user string) (map[string]interface{}, error)
}

type aiService struct {
	client client.AIClient
}

func NewAIService(client client.AIClient) AIService {
	return &aiService{
		client: client,
	}
}

func (s *aiService) SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error) {
	return s.client.SendChatMessage(ctx, message)
}

func (s *aiService) SendStreamingChatMessage(ctx context.Context, message model.ChatMessage, callback func(model.ChunkChatCompletionResponse) error) error {
	return s.client.SendStreamingChatMessage(ctx, message, callback)
}

func (s *aiService) UploadFile(ctx context.Context, filePath string, user string) (*model.FileUploadResponse, error) {
	return s.client.UploadFile(ctx, filePath, user)
}

func (s *aiService) StopResponse(ctx context.Context, taskID string, user string) error {
	return s.client.StopResponse(ctx, taskID, user)
}

func (s *aiService) SendFeedback(ctx context.Context, messageID string, rating string, user string, content string) error {
	return s.client.SendFeedback(ctx, messageID, rating, user, content)
}

func (s *aiService) GetSuggestedQuestions(ctx context.Context, messageID string, user string) ([]string, error) {
	return s.client.GetSuggestedQuestions(ctx, messageID, user)
}

func (s *aiService) GetMessages(ctx context.Context, conversationID string, user string, firstID string, limit int) (map[string]interface{}, error) {
	return s.client.GetMessages(ctx, conversationID, user, firstID, limit)
}

func (s *aiService) GetConversations(ctx context.Context, user string, lastID string, limit int, sortBy string) (map[string]interface{}, error) {
	return s.client.GetConversations(ctx, user, lastID, limit, sortBy)
}

func (s *aiService) DeleteConversation(ctx context.Context, conversationID string, user string) error {
	return s.client.DeleteConversation(ctx, conversationID, user)
}

func (s *aiService) RenameConversation(ctx context.Context, conversationID string, name string, autoGenerate bool, user string) (map[string]interface{}, error) {
	return s.client.RenameConversation(ctx, conversationID, name, autoGenerate, user)
}
