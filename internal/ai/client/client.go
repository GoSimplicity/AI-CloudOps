package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type AIClient interface {
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

type aiClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	l          *zap.Logger
}

func NewAIClient(l *zap.Logger) AIClient {
	BaseURL := viper.GetString("ai.base_url")
	if BaseURL == "" {
		BaseURL = "http://localhost/v1"
	}

	apiKey := viper.GetString("ai.api_key")
	if apiKey == "" {
		l.Error("未配置AI API密钥")
	}

	return &aiClient{
		BaseURL:    BaseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
		l:          l,
	}
}

// 统一的错误处理
func (a *aiClient) handleAPIError(resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API错误: %d - %s", resp.StatusCode, string(bodyBytes))
}

// SendChatMessage 发送聊天消息
func (a *aiClient) SendChatMessage(ctx context.Context, message model.ChatMessage) (*model.ChatCompletionResponse, error) {
	url := fmt.Sprintf("%s/chat-messages", a.BaseURL)
	a.l.Info("发送聊天消息到", zap.String("url", url))

	jsonData, err := json.Marshal(message)
	if err != nil {
		a.l.Error("序列化消息失败", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("API错误: %d - %s", resp.StatusCode, string(bodyBytes))
		a.l.Error("API错误",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(bodyBytes)))
		return nil, errors.New(errMsg)
	}

	var result model.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		a.l.Error("解析响应失败", zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// SendStreamingChatMessage 发送流式聊天消息
func (a *aiClient) SendStreamingChatMessage(ctx context.Context, message model.ChatMessage, callback func(model.ChunkChatCompletionResponse) error) error {
	// 确保使用流式模式
	message.ResponseMode = "streaming"

	url := fmt.Sprintf("%s/chat-messages", a.BaseURL)
	a.l.Info("发送流式聊天消息到", zap.String("url", url))

	jsonData, err := json.Marshal(message)
	if err != nil {
		a.l.Error("序列化消息失败", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return a.handleAPIError(resp)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			a.l.Error("读取流数据错误", zap.Error(err))
			return fmt.Errorf("读取流数据错误: %w", err)
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk model.ChunkChatCompletionResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			a.l.Error("解析流数据失败", zap.Error(err), zap.String("data", data))
			return err
		}

		if err := callback(chunk); err != nil {
			a.l.Error("处理回调失败", zap.Error(err))
			return err
		}

		if chunk.Event == "message_end" || chunk.Event == "error" {
			a.l.Debug("流式消息结束", zap.String("event", chunk.Event))
			break
		}
	}

	return nil
}

// UploadFile 上传文件
func (a *aiClient) UploadFile(ctx context.Context, filePath string, user string) (*model.FileUploadResponse, error) {
	url := fmt.Sprintf("%s/files/upload", a.BaseURL)
	a.l.Info("上传文件", zap.String("url", url), zap.String("filePath", filePath), zap.String("user", user))

	file, err := os.Open(filePath)
	if err != nil {
		a.l.Error("打开文件失败", zap.Error(err), zap.String("filePath", filePath))
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		a.l.Error("创建表单文件失败", zap.Error(err))
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		a.l.Error("复制文件内容失败", zap.Error(err))
		return nil, err
	}

	err = writer.WriteField("user", user)
	if err != nil {
		a.l.Error("写入表单字段失败", zap.Error(err))
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		a.l.Error("关闭表单写入器失败", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return nil, a.handleAPIError(resp)
	}

	var result model.FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		a.l.Error("解析响应失败", zap.Error(err))
		return nil, err
	}

	a.l.Info("文件上传成功", zap.String("fileID", result.ID))
	return &result, nil
}

// StopResponse 停止响应
func (a *aiClient) StopResponse(ctx context.Context, taskID string, user string) error {
	url := fmt.Sprintf("%s/chat-messages/%s/stop", a.BaseURL, taskID)
	a.l.Info("停止响应", zap.String("url", url), zap.String("taskID", taskID), zap.String("user", user))

	data := map[string]string{
		"user": user,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		a.l.Error("序列化数据失败", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return a.handleAPIError(resp)
	}

	a.l.Info("成功停止响应", zap.String("taskID", taskID))
	return nil
}

// SendFeedback 发送消息反馈
func (a *aiClient) SendFeedback(ctx context.Context, messageID string, rating string, user string, content string) error {
	url := fmt.Sprintf("%s/messages/%s/feedbacks", a.BaseURL, messageID)
	a.l.Info("发送反馈",
		zap.String("url", url),
		zap.String("messageID", messageID),
		zap.String("rating", rating),
		zap.String("user", user))

	data := map[string]string{
		"rating":  rating,
		"user":    user,
		"content": content,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		a.l.Error("序列化数据失败", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return a.handleAPIError(resp)
	}

	a.l.Info("反馈发送成功", zap.String("messageID", messageID))
	return nil
}

// GetSuggestedQuestions 获取下一轮建议问题列表
func (a *aiClient) GetSuggestedQuestions(ctx context.Context, messageID string, user string) ([]string, error) {
	url := fmt.Sprintf("%s/messages/%s/suggested?user=%s", a.BaseURL, messageID, user)
	a.l.Info("获取建议问题", zap.String("url", url), zap.String("messageID", messageID), zap.String("user", user))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return nil, a.handleAPIError(resp)
	}

	var result struct {
		Result string   `json:"result"`
		Data   []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		a.l.Error("解析响应失败", zap.Error(err))
		return nil, err
	}

	a.l.Info("成功获取建议问题", zap.Int("count", len(result.Data)))
	return result.Data, nil
}

// GetMessages 获取会话历史消息
func (a *aiClient) GetMessages(ctx context.Context, conversationID string, user string, firstID string, limit int) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/messages?conversation_id=%s&user=%s", a.BaseURL, conversationID, user)

	if firstID != "" {
		url += fmt.Sprintf("&first_id=%s", firstID)
	}

	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	a.l.Info("获取会话历史消息",
		zap.String("url", url),
		zap.String("conversationID", conversationID),
		zap.String("user", user),
		zap.String("firstID", firstID),
		zap.Int("limit", limit))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return nil, a.handleAPIError(resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		a.l.Error("解析响应失败", zap.Error(err))
		return nil, err
	}

	a.l.Info("成功获取会话历史消息", zap.String("conversationID", conversationID))
	return result, nil
}

// GetConversations 获取会话列表
func (a *aiClient) GetConversations(ctx context.Context, user string, lastID string, limit int, sortBy string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/conversations?user=%s", a.BaseURL, user)

	if lastID != "" {
		url += fmt.Sprintf("&last_id=%s", lastID)
	}

	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	if sortBy != "" {
		url += fmt.Sprintf("&sort_by=%s", sortBy)
	}

	a.l.Info("获取会话列表",
		zap.String("url", url),
		zap.String("user", user),
		zap.String("lastID", lastID),
		zap.Int("limit", limit),
		zap.String("sortBy", sortBy))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return nil, a.handleAPIError(resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		a.l.Error("解析响应失败", zap.Error(err))
		return nil, err
	}

	a.l.Info("成功获取会话列表", zap.String("user", user))
	return result, nil
}

// DeleteConversation 删除会话
func (a *aiClient) DeleteConversation(ctx context.Context, conversationID string, user string) error {
	url := fmt.Sprintf("%s/conversations/%s", a.BaseURL, conversationID)
	a.l.Info("删除会话", zap.String("url", url), zap.String("conversationID", conversationID), zap.String("user", user))

	data := map[string]string{
		"user": user,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		a.l.Error("序列化数据失败", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return a.handleAPIError(resp)
	}

	a.l.Info("成功删除会话", zap.String("conversationID", conversationID))
	return nil
}

// RenameConversation 会话重命名
func (a *aiClient) RenameConversation(ctx context.Context, conversationID string, name string, autoGenerate bool, user string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/conversations/%s/name", a.BaseURL, conversationID)
	a.l.Info("重命名会话",
		zap.String("url", url),
		zap.String("conversationID", conversationID),
		zap.String("name", name),
		zap.Bool("autoGenerate", autoGenerate),
		zap.String("user", user))

	data := map[string]interface{}{
		"user":          user,
		"auto_generate": autoGenerate,
	}

	if name != "" {
		data["name"] = name
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		a.l.Error("序列化数据失败", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		a.l.Error("创建请求失败", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.APIKey))

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.l.Error("发送请求失败", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.l.Error("API返回非200状态码", zap.Int("status_code", resp.StatusCode))
		return nil, a.handleAPIError(resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		a.l.Error("解析响应失败", zap.Error(err))
		return nil, err
	}

	a.l.Info("成功重命名会话", zap.String("conversationID", conversationID))
	return result, nil
}
