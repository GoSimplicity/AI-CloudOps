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
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

// 飞书ID类型枚举
const (
	FeishuIDTypeOpenID  = "open_id"
	FeishuIDTypeUserID  = "user_id"
	FeishuIDTypeChatID  = "chat_id"
	FeishuIDTypeEmail   = "email"
	FeishuIDTypeUnionID = "union_id"
)

var (
	// 飞书Chat ID模式 (以oc_开头)
	chatIDPattern = regexp.MustCompile(`^oc_[a-zA-Z0-9]+$`)
	// 飞书Open ID模式 (以ou_开头)
	openIDPattern = regexp.MustCompile(`^ou_[a-zA-Z0-9]+$`)
	// 飞书Union ID模式 (以on_开头)
	unionIDPattern = regexp.MustCompile(`^on_[a-zA-Z0-9]+$`)
	// 邮箱模式
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type FeishuChannel struct {
	config      FeishuConfig
	logger      *zap.Logger
	httpClient  *http.Client
	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewFeishuChannel(config FeishuConfig, logger *zap.Logger) *FeishuChannel {
	return &FeishuChannel{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: config.GetTimeout(),
		},
	}
}

func (f *FeishuChannel) GetName() string {
	return model.NotificationChannelFeishu
}

// Send 发送飞书消息到指定接收人
func (f *FeishuChannel) Send(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	startTime := time.Now()

	if request.RecipientAddr == "" {
		return f.createErrorResponse(request.MessageID, "recipient address is empty",
			fmt.Errorf("飞书收件人地址不能为空"), startTime), fmt.Errorf("飞书收件人地址不能为空")
	}

	// Both chat_id and user identifiers use the authenticated message API.
	// A chat ID is not a robot-webhook token.
	return f.sendPrivateMessage(ctx, request, startTime)
}

// determineRecipientType 识别接收人ID类型
func (f *FeishuChannel) determineRecipientType(recipientAddr string) (string, error) {
	f.logger.Debug("确定收件人ID类型",
		zap.String("recipient_addr", recipientAddr),
		zap.String("recipient_length", fmt.Sprintf("%d", len(recipientAddr))))

	switch {
	case chatIDPattern.MatchString(recipientAddr):
		f.logger.Debug("匹配到群聊ID", zap.String("type", FeishuIDTypeChatID))
		return FeishuIDTypeChatID, nil
	case openIDPattern.MatchString(recipientAddr):
		f.logger.Debug("匹配到开放ID", zap.String("type", FeishuIDTypeOpenID))
		return FeishuIDTypeOpenID, nil
	case unionIDPattern.MatchString(recipientAddr):
		f.logger.Debug("匹配到联合ID", zap.String("type", FeishuIDTypeUnionID))
		return FeishuIDTypeUnionID, nil
	case emailPattern.MatchString(recipientAddr):
		f.logger.Debug("匹配到邮箱", zap.String("type", FeishuIDTypeEmail))
		return FeishuIDTypeEmail, nil
	default:
		// 如果都不匹配，默认为用户ID
		f.logger.Debug("默认为用户ID", zap.String("type", FeishuIDTypeUserID))
		return FeishuIDTypeUserID, nil
	}
}

// sendPrivateMessage 发送私聊消息
func (f *FeishuChannel) sendPrivateMessage(ctx context.Context, request *SendRequest, startTime time.Time) (*SendResponse, error) {
	if err := f.ensureAccessToken(ctx); err != nil {
		return f.createErrorResponse(request.MessageID, "get access token failed", err, startTime), err
	}

	// 确定收件人ID类型
	recipientType, err := f.determineRecipientType(request.RecipientAddr)
	if err != nil {
		return f.createErrorResponse(request.MessageID, "invalid recipient format", err, startTime), err
	}

	// 验证recipientType不为空
	if recipientType == "" {
		err := fmt.Errorf("recipient type is empty")
		f.logger.Error("收件人类型为空",
			zap.String("recipient_addr", request.RecipientAddr))
		return f.createErrorResponse(request.MessageID, "recipient type is empty", err, startTime), err
	}

	message := f.buildPrivateMessageContent(request, recipientType)

	// 构建带查询参数的URL
	apiURL := fmt.Sprintf("%s?receive_id_type=%s", f.config.GetPrivateMessageAPI(), recipientType)

	// 添加详细调试日志
	jsonData, marshalErr := json.Marshal(message)
	if marshalErr != nil {
		f.logger.Error("序列化消息失败", zap.Error(marshalErr))
		return f.createErrorResponse(request.MessageID, "marshal message failed", marshalErr, startTime), marshalErr
	}

	f.logger.Debug("飞书私聊消息请求详情",
		zap.String("recipient", request.RecipientAddr),
		zap.String("recipient_type", recipientType),
		zap.String("api_url", apiURL),
		zap.String("message_json", string(jsonData)),
		zap.Any("message_struct", message))

	if receive_id, ok := message["receive_id"].(string); !ok || receive_id == "" {
		err := fmt.Errorf("receive_id is missing or empty")
		f.logger.Error("receive_id字段缺失", zap.Any("message", message))
		return f.createErrorResponse(request.MessageID, "receive_id is missing", err, startTime), err
	}

	return f.sendHTTPRequest(ctx, apiURL, message, request.MessageID, startTime, true)
}

// sendHTTPRequest 发送HTTP请求并处理响应
func (f *FeishuChannel) sendHTTPRequest(ctx context.Context, url string, message map[string]interface{},
	messageID string, startTime time.Time, needAuth bool) (*SendResponse, error) {

	jsonData, err := json.Marshal(message)
	if err != nil {
		return f.createErrorResponse(messageID, "marshal message failed", err, startTime), err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return f.createErrorResponse(messageID, "create request failed", err, startTime), err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if needAuth {
		f.tokenMu.Lock()
		req.Header.Set("Authorization", "Bearer "+f.accessToken)
		f.tokenMu.Unlock()
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		f.logger.Error("发送飞书消息失败",
			zap.String("url", url),
			zap.Bool("need_auth", needAuth),
			zap.Error(err))
		return f.createErrorResponse(messageID, "send request failed", err, startTime), err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return f.createErrorResponse(messageID, "read response failed", err, startTime), err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return f.createErrorResponse(messageID, "parse response failed", err, startTime), err
	}

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		errorMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		f.logger.Error("飞书API返回错误状态码",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)),
			zap.String("url", url))
		return f.createErrorResponse(messageID, errorMsg, errors.New(errorMsg), startTime), errors.New(errorMsg)
	}

	if code, ok := response["code"].(float64); !ok || code != 0 {
		errorMsg := fmt.Sprintf("Feishu API error (code: %.0f): %v", code, response["msg"])
		f.logger.Error("飞书API返回业务错误",
			zap.Float64("error_code", code),
			zap.Any("error_msg", response["msg"]),
			zap.Any("error_detail", response["error"]),
			zap.String("url", url))

		return f.createErrorResponse(messageID, errorMsg, errors.New(errorMsg), startTime), errors.New(errorMsg)
	}

	msgType := "群消息"
	if needAuth {
		msgType = "私聊消息"
	}

	f.logger.Info("飞书消息发送成功",
		zap.String("message_type", msgType),
		zap.String("message_id", messageID),
		zap.Duration("duration", time.Since(startTime)))

	// 获取外部消息ID
	var externalID string
	if data, ok := response["data"].(map[string]interface{}); ok {
		if msgID, ok := data["message_id"].(string); ok {
			externalID = msgID
		}
	}

	return &SendResponse{
		Success:      true,
		MessageID:    messageID,
		ExternalID:   externalID,
		Status:       "sent",
		SendTime:     startTime,
		ResponseData: response,
	}, nil
}

func (f *FeishuChannel) ensureAccessToken(ctx context.Context) error {
	f.tokenMu.Lock()
	defer f.tokenMu.Unlock()
	if f.accessToken != "" && time.Now().Before(f.tokenExpiry) {
		return nil
	}

	f.logger.Debug("获取飞书访问令牌", zap.String("api_url", f.config.GetTenantAccessTokenAPI()))

	tokenReq := map[string]string{
		"app_id":     f.config.GetAppID(),
		"app_secret": f.config.GetAppSecret(),
	}

	jsonData, err := json.Marshal(tokenReq)
	if err != nil {
		return fmt.Errorf("marshal token request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.config.GetTenantAccessTokenAPI(), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create token request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

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

	if code, ok := tokenResp["code"].(float64); !ok || code != 0 || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get access token error (code: %.0f): %v", code, tokenResp["msg"])
	}

	if token, ok := tokenResp["tenant_access_token"].(string); ok && token != "" {
		f.accessToken = token
		// 设置过期时间（提前5分钟过期以避免边界情况）
		f.tokenExpiry = time.Now().Add(90*time.Minute - 5*time.Minute)

		f.logger.Debug("飞书访问令牌获取成功",
			zap.Time("expires_at", f.tokenExpiry))

		return nil
	}

	return fmt.Errorf("invalid token response: missing tenant_access_token")
}

func (f *FeishuChannel) getPriorityConfig(priority int) (icon, text, color, templateColor string) {
	icon = FormatPriorityIcon(int8(priority))
	text = FormatPriority(int8(priority))

	switch priority {
	case 1: // 高优先级
		color, templateColor = "red", "red"
	case 3: // 低优先级
		color, templateColor = "green", "green"
	default: // 中等优先级
		color, templateColor = "orange", "blue"
	}
	return
}

func (f *FeishuChannel) buildGroupMessage(request *SendRequest) map[string]interface{} {
	priorityIcon, priorityText, _, templateColor := f.getPriorityConfig(int(request.Priority))
	eventText := GetEventTypeText(request.EventType)

	headerTitle := fmt.Sprintf("⚡ AI-CloudOps | %s", eventText)

	workorderNumber := "系统通知"
	if request.InstanceID != nil {
		workorderNumber = fmt.Sprintf("WO-%d", *request.InstanceID)
	}

	// 构建简洁商务化卡片内容元素
	elements := []map[string]interface{}{
		// 核心信息区域 - 紧凑布局
		{
			"tag": "div",
			"fields": []map[string]interface{}{
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**工单编号**\n`%s`", workorderNumber),
					},
				},
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**优先级**\n%s %s", priorityIcon, priorityText),
					},
				},
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**时间**\n%s", time.Now().Format("01-02 15:04")),
					},
				},
			},
		},

		// 通知内容区域 - 简洁呈现
		{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**📋 通知内容**\n\n%s", f.renderContent(request)),
			},
		},

		{
			"tag": "note",
			"elements": []map[string]interface{}{
				{
					"tag":     "plain_text",
					"content": "AI-CloudOps 运维管理平台发送 | 技术支持：400-000-0000",
				},
			},
		},
	}

	// 如果有工单ID，添加专业操作按钮
	if request.InstanceID != nil {
		actionButtons := map[string]interface{}{
			"tag": "action",
			"actions": []map[string]interface{}{
				{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "立即查看",
					},
					"type": "primary",
					"url":  fmt.Sprintf("#/workorder/instance/detail/%d", *request.InstanceID),
				},
				{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "管理平台",
					},
					"type": "default",
					"url":  "#/dashboard",
				},
			},
		}
		elements = append(elements, actionButtons)
	}

	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"elements": elements,
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": headerTitle,
				},
				"template": templateColor,
			},
			"config": map[string]interface{}{
				"wide_screen_mode": true,
				"enable_forward":   true,
			},
		},
	}
}

func (f *FeishuChannel) buildPrivateMessageContent(request *SendRequest, recipientType string) map[string]interface{} {
	f.logger.Debug("构建商务化私聊消息内容",
		zap.String("recipient_addr", request.RecipientAddr),
		zap.String("recipient_type", recipientType))

	priorityIcon, priorityText, _, templateColor := f.getPriorityConfig(int(request.Priority))
	eventText := GetEventTypeText(request.EventType)

	headerTitle := fmt.Sprintf("⚡ AI-CloudOps | %s", eventText)

	workorderNumber := "系统通知"
	if request.InstanceID != nil {
		workorderNumber = fmt.Sprintf("WO-%d", *request.InstanceID)
	}

	// 构建简洁商务化卡片内容元素
	elements := []map[string]interface{}{
		// 核心信息区域 - 紧凑布局
		{
			"tag": "div",
			"fields": []map[string]interface{}{
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**工单编号**\n`%s`", workorderNumber),
					},
				},
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**优先级**\n%s %s", priorityIcon, priorityText),
					},
				},
				{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**时间**\n%s", time.Now().Format("01-02 15:04")),
					},
				},
			},
		},

		// 通知内容区域 - 简洁呈现
		{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**📋 通知内容**\n\n%s", f.renderContent(request)),
			},
		},

		{
			"tag": "note",
			"elements": []map[string]interface{}{
				{
					"tag":     "plain_text",
					"content": "AI-CloudOps 运维管理平台发送 | 技术支持：400-000-0000",
				},
			},
		},
	}

	// 如果有工单ID，添加专业操作按钮
	if request.InstanceID != nil {
		actionButtons := map[string]interface{}{
			"tag": "action",
			"actions": []map[string]interface{}{
				{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "立即查看",
					},
					"type": "primary",
					"url":  fmt.Sprintf("#/workorder/instance/detail/%d", *request.InstanceID),
				},
				{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "管理平台",
					},
					"type": "default",
					"url":  "#/dashboard",
				},
			},
		}
		elements = append(elements, actionButtons)
	}

	// 构建卡片内容（注意：这里直接是卡片内容，不包含外层的card字段）
	cardContent := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
			"enable_forward":   true,
		},
		"elements": elements,
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": headerTitle,
			},
			"template": templateColor,
		},
	}

	// 序列化卡片内容为JSON字符串
	contentBytes, err := json.Marshal(cardContent)
	if err != nil {
		f.logger.Error("序列化卡片内容失败", zap.Error(err))
		// 提供一个简单的fallback内容
		contentBytes = []byte(`{"text":"消息内容序列化失败"}`)
	}

	finalMessage := map[string]interface{}{
		"receive_id": request.RecipientAddr,
		"msg_type":   "interactive",
		"content":    string(contentBytes), // content字段的值是卡片的JSON字符串
	}

	f.logger.Debug("私聊消息构建完成",
		zap.String("receive_id", request.RecipientAddr),
		zap.String("msg_type", "interactive"),
		zap.Int("content_length", len(string(contentBytes))))

	return finalMessage
}

func (f *FeishuChannel) renderContent(request *SendRequest) string {
	renderedContent, err := RenderTemplate(request.Content, request)
	if err != nil {
		return request.Content // 渲染失败时使用原始内容
	}
	return renderedContent
}

func (f *FeishuChannel) getDisplayName(name string) string {
	if name == "" {
		return "系统用户"
	}
	return name
}

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

func (f *FeishuChannel) Validate() error {
	return f.config.Validate()
}

func (f *FeishuChannel) IsEnabled() bool {
	return f.config.IsEnabled()
}

func (f *FeishuChannel) GetMaxRetries() int {
	return f.config.GetMaxRetries()
}

func (f *FeishuChannel) GetRetryInterval() time.Duration {
	return f.config.GetRetryInterval()
}
