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

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FeishuConfig 飞书配置
type FeishuConfig struct {
	BaseChannelConfig    `yaml:",inline"`
	AppID                string `json:"app_id" yaml:"app_id"`
	AppSecret            string `json:"app_secret" yaml:"app_secret"`
	WebhookURL           string `json:"webhook_url" yaml:"webhook_url"`
	PrivateMessageAPI    string `json:"private_message_api" yaml:"private_message_api"`
	TenantAccessTokenAPI string `json:"tenant_access_token_api" yaml:"tenant_access_token_api"`
}

// GetChannelName 获取渠道名称
func (c *FeishuConfig) GetChannelName() string {
	return "feishu"
}

// Validate 验证配置
func (c *FeishuConfig) Validate() error {
	if c.AppID == "" {
		return fmt.Errorf("app_id is required")
	}
	if c.AppSecret == "" {
		return fmt.Errorf("app_secret is required")
	}
	if c.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}
	if c.PrivateMessageAPI == "" {
		return fmt.Errorf("private_message_api is required")
	}
	if c.TenantAccessTokenAPI == "" {
		return fmt.Errorf("tenant_access_token_api is required")
	}
	return nil
}

// FeishuChannel 飞书通知渠道
type FeishuChannel struct {
	config      *FeishuConfig
	logger      *zap.Logger
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
}

// NewFeishuChannel 创建飞书通知渠道
func NewFeishuChannel(config *FeishuConfig, logger *zap.Logger) *FeishuChannel {
	return &FeishuChannel{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: config.GetTimeout(),
		},
	}
}

// GetName 获取渠道名称
func (f *FeishuChannel) GetName() string {
	return "feishu"
}

// Send 发送飞书消息
func (f *FeishuChannel) Send(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	startTime := time.Now()

	// 判断是群组消息还是私聊消息
	if strings.HasPrefix(request.RecipientAddr, "oc_") {
		// 群组webhook消息
		return f.sendGroupMessage(ctx, request, startTime)
	} else {
		// 私聊消息
		return f.sendPrivateMessage(ctx, request, startTime)
	}
}

// sendGroupMessage 发送群组消息（webhook）
func (f *FeishuChannel) sendGroupMessage(ctx context.Context, request *SendRequest, startTime time.Time) (*SendResponse, error) {
	// 构建webhook URL
	webhookURL := f.config.WebhookURL + request.RecipientAddr

	// 构建消息内容
	message := f.buildGroupMessage(request)

	// 发送请求
	jsonData, _ := json.Marshal(message)
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return f.createErrorResponse(request.MessageID, "create request failed", err, startTime), err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		f.logger.Error("发送飞书群组消息失败",
			zap.String("webhook_url", webhookURL),
			zap.Error(err))
		return f.createErrorResponse(request.MessageID, "send request failed", err, startTime), err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return f.createErrorResponse(request.MessageID, "read response failed", err, startTime), err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return f.createErrorResponse(request.MessageID, "parse response failed", err, startTime), err
	}

	// 检查响应状态
	if resp.StatusCode != 200 {
		errorMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return f.createErrorResponse(request.MessageID, errorMsg, fmt.Errorf(errorMsg), startTime), fmt.Errorf(errorMsg)
	}

	// 检查飞书响应码
	if code, ok := response["code"].(float64); ok && code != 0 {
		errorMsg := fmt.Sprintf("Feishu error code: %v", response["msg"])
		return f.createErrorResponse(request.MessageID, errorMsg, fmt.Errorf(errorMsg), startTime), fmt.Errorf(errorMsg)
	}

	f.logger.Info("飞书群组消息发送成功",
		zap.String("recipient", request.RecipientAddr),
		zap.Duration("duration", time.Since(startTime)))

	return &SendResponse{
		Success:      true,
		MessageID:    request.MessageID,
		Status:       "sent",
		SendTime:     startTime,
		ResponseData: response,
	}, nil
}

// sendPrivateMessage 发送私聊消息
func (f *FeishuChannel) sendPrivateMessage(ctx context.Context, request *SendRequest, startTime time.Time) (*SendResponse, error) {
	// 获取访问令牌
	if err := f.ensureAccessToken(ctx); err != nil {
		return f.createErrorResponse(request.MessageID, "get access token failed", err, startTime), err
	}

	// 构建私聊消息
	message := f.buildPrivateMessage(request)

	// 发送请求
	jsonData, _ := json.Marshal(message)
	req, err := http.NewRequestWithContext(ctx, "POST", f.config.PrivateMessageAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return f.createErrorResponse(request.MessageID, "create request failed", err, startTime), err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.accessToken)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		f.logger.Error("发送飞书私聊消息失败",
			zap.String("api", f.config.PrivateMessageAPI),
			zap.Error(err))
		return f.createErrorResponse(request.MessageID, "send request failed", err, startTime), err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return f.createErrorResponse(request.MessageID, "read response failed", err, startTime), err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return f.createErrorResponse(request.MessageID, "parse response failed", err, startTime), err
	}

	// 检查响应状态
	if resp.StatusCode != 200 {
		errorMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return f.createErrorResponse(request.MessageID, errorMsg, fmt.Errorf(errorMsg), startTime), fmt.Errorf(errorMsg)
	}

	// 检查飞书响应码
	if code, ok := response["code"].(float64); ok && code != 0 {
		errorMsg := fmt.Sprintf("Feishu error code: %v", response["msg"])
		return f.createErrorResponse(request.MessageID, errorMsg, fmt.Errorf(errorMsg), startTime), fmt.Errorf(errorMsg)
	}

	f.logger.Info("飞书私聊消息发送成功",
		zap.String("recipient", request.RecipientAddr),
		zap.Duration("duration", time.Since(startTime)))

	// 获取消息ID
	var externalID string
	if data, ok := response["data"].(map[string]interface{}); ok {
		if msgID, ok := data["message_id"].(string); ok {
			externalID = msgID
		}
	}

	return &SendResponse{
		Success:      true,
		MessageID:    request.MessageID,
		ExternalID:   externalID,
		Status:       "sent",
		SendTime:     startTime,
		ResponseData: response,
	}, nil
}

// ensureAccessToken 确保访问令牌有效
func (f *FeishuChannel) ensureAccessToken(ctx context.Context) error {
	// 检查token是否过期
	if f.accessToken != "" && time.Now().Before(f.tokenExpiry) {
		return nil
	}

	// 获取新的访问令牌
	tokenReq := map[string]string{
		"app_id":     f.config.AppID,
		"app_secret": f.config.AppSecret,
	}

	jsonData, _ := json.Marshal(tokenReq)
	req, err := http.NewRequestWithContext(ctx, "POST", f.config.TenantAccessTokenAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create token request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("get access token failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read token response failed: %w", err)
	}

	var tokenResp map[string]interface{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parse token response failed: %w", err)
	}

	// 检查响应
	if code, ok := tokenResp["code"].(float64); ok && code != 0 {
		return fmt.Errorf("get access token error: %v", tokenResp["msg"])
	}

	if token, ok := tokenResp["tenant_access_token"].(string); ok {
		f.accessToken = token
		// 设置过期时间（通常为2小时，这里设置为1.5小时确保安全）
		f.tokenExpiry = time.Now().Add(90 * time.Minute)
		return nil
	}

	return fmt.Errorf("invalid token response")
}

// buildGroupMessage 构建群组消息
func (f *FeishuChannel) buildGroupMessage(request *SendRequest) map[string]interface{} {
	// 获取优先级标识
	priorityIcon := "🔔"
	priorityText := "中等"
	if request.Priority == 1 {
		priorityIcon = "🔴"
		priorityText = "高"
	} else if request.Priority == 3 {
		priorityIcon = "🟢"
		priorityText = "低"
	}

	// 构建富文本内容
	elements := []map[string]interface{}{
		{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("%s **工单系统通知**", priorityIcon),
			},
		},
		{
			"tag": "hr",
		},
		{
			"tag": "div",
			"fields": []map[string]interface{}{
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**接收人：**\n%s", request.RecipientName),
					},
				},
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**优先级：**\n%s", priorityText),
					},
				},
			},
		},
	}

	// 添加工单信息
	if request.InstanceID != nil {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"fields": []map[string]interface{}{
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**工单ID：**\n#%d", *request.InstanceID),
					},
				},
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**事件类型：**\n%s", request.EventType),
					},
				},
			},
		})
	}

	// 添加消息内容
	elements = append(elements, map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"tag":     "lark_md",
			"content": fmt.Sprintf("**消息内容：**\n%s", request.Content),
		},
	})

	// 添加时间戳
	elements = append(elements, map[string]interface{}{
		"tag": "note",
		"elements": []map[string]interface{}{
			{
				"tag":     "lark_md",
				"content": fmt.Sprintf("发送时间：%s", time.Now().Format("2006-01-02 15:04:05")),
			},
		},
	})

	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"elements": elements,
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "lark_md",
					"content": request.Subject,
				},
				"template": "blue",
			},
		},
	}
}

// buildPrivateMessage 构建私聊消息
func (f *FeishuChannel) buildPrivateMessage(request *SendRequest) map[string]interface{} {
	// 构建文本消息
	content := fmt.Sprintf("🔔 **工单系统通知**\n\n")
	content += fmt.Sprintf("**接收人：** %s\n", request.RecipientName)
	content += fmt.Sprintf("**事件类型：** %s\n", request.EventType)

	if request.InstanceID != nil {
		content += fmt.Sprintf("**工单ID：** #%d\n", *request.InstanceID)
	}

	content += fmt.Sprintf("\n**消息内容：**\n%s\n", request.Content)
	content += fmt.Sprintf("\n---\n*发送时间：%s*", time.Now().Format("2006-01-02 15:04:05"))

	return map[string]interface{}{
		"receive_id":      request.RecipientAddr,
		"receive_id_type": "user_id",
		"msg_type":        "text",
		"content":         fmt.Sprintf(`{"text":"%s"}`, strings.ReplaceAll(content, "\"", "\\\"")),
	}
}

// createErrorResponse 创建错误响应
func (f *FeishuChannel) createErrorResponse(messageID, errorMsg string, err error, startTime time.Time) *SendResponse {
	return &SendResponse{
		Success:      false,
		MessageID:    messageID,
		Status:       "failed",
		ErrorMessage: errorMsg,
		SendTime:     startTime,
		ResponseData: map[string]interface{}{
			"error": err.Error(),
		},
	}
}

// Validate 验证配置
func (f *FeishuChannel) Validate() error {
	return f.config.Validate()
}

// IsEnabled 是否启用
func (f *FeishuChannel) IsEnabled() bool {
	return f.config.IsEnabled()
}

// GetMaxRetries 获取最大重试次数
func (f *FeishuChannel) GetMaxRetries() int {
	return f.config.GetMaxRetries()
}

// GetRetryInterval 获取重试间隔
func (f *FeishuChannel) GetRetryInterval() time.Duration {
	return f.config.GetRetryInterval()
}
