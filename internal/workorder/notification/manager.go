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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// NotificationConfig 通知配置
type NotificationConfig interface {
	GetEmail() EmailConfig
	GetFeishu() FeishuConfig
}

// EmailConfig 邮箱配置
type EmailConfig interface {
	IsEnabled() bool
	GetMaxRetries() int
	GetRetryInterval() time.Duration
	GetTimeout() time.Duration
	GetChannelName() string
	Validate() error
	GetSMTPHost() string
	GetSMTPPort() int
	GetUsername() string
	GetPassword() string
	GetFromName() string
	GetUseTLS() bool
}

// FeishuConfig 飞书配置
type FeishuConfig interface {
	IsEnabled() bool
	GetMaxRetries() int
	GetRetryInterval() time.Duration
	GetTimeout() time.Duration
	GetChannelName() string
	Validate() error
	GetAppID() string
	GetAppSecret() string
	GetWebhookURL() string
	GetPrivateMessageAPI() string
	GetTenantAccessTokenAPI() string
}

// Manager 通知管理器
type Manager struct {
	channels    map[string]NotificationChannel
	queueClient *asynq.Client
	logger      *zap.Logger
	config      NotificationConfig
	mu          sync.RWMutex
}

// NewManager 创建管理器
func NewManager(config NotificationConfig, queueClient *asynq.Client, logger *zap.Logger) (*Manager, error) {
	manager := &Manager{
		channels:    make(map[string]NotificationChannel),
		queueClient: queueClient,
		logger:      logger,
		config:      config,
	}

	// 邮箱渠道
	emailConfig := config.GetEmail()
	if emailConfig != nil && emailConfig.IsEnabled() {
		if err := emailConfig.Validate(); err != nil {
			logger.Warn("邮箱配置验证失败", zap.Error(err))
		} else {
			emailChannel := NewEmailChannel(emailConfig, logger)
			manager.channels["email"] = emailChannel
			logger.Info("邮箱通知渠道已启用")
		}
	}

	// 飞书渠道
	feishuConfig := config.GetFeishu()
	if feishuConfig != nil && feishuConfig.IsEnabled() {
		if err := feishuConfig.Validate(); err != nil {
			logger.Warn("飞书配置验证失败", zap.Error(err))
		} else {
			feishuChannel := NewFeishuChannel(feishuConfig, logger)
			manager.channels["feishu"] = feishuChannel
			logger.Info("飞书通知渠道已启用")
		}
	}

	if len(manager.channels) == 0 {
		logger.Warn("没有可用的通知渠道")
	}

	return manager, nil
}

// SendNotification 发送通知
func (m *Manager) SendNotification(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	// 生成ID
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}

	// 获取渠道
	channel, err := m.getChannel(request.RecipientType)
	if err != nil {
		return &SendResponse{
			Success:      false,
			MessageID:    request.MessageID,
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("获取通知渠道失败: %v", err),
			SendTime:     time.Now(),
		}, err
	}

	// 检查启用状态
	if !channel.IsEnabled() {
		err := fmt.Errorf("通知渠道 %s 未启用", channel.GetName())
		return &SendResponse{
			Success:      false,
			MessageID:    request.MessageID,
			Status:       "failed",
			ErrorMessage: err.Error(),
			SendTime:     time.Now(),
		}, err
	}

	// 添加重试机制
	maxRetries := channel.GetMaxRetries()
	retryInterval := channel.GetRetryInterval()

	var lastErr error
	var response *SendResponse

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			m.logger.Info("重试发送通知",
				zap.String("channel", channel.GetName()),
				zap.String("message_id", request.MessageID),
				zap.Int("attempt", attempt),
				zap.Duration("retry_interval", retryInterval))

			// 等待重试间隔
			select {
			case <-ctx.Done():
				return &SendResponse{
					Success:      false,
					MessageID:    request.MessageID,
					Status:       "cancelled",
					ErrorMessage: "context cancelled",
					SendTime:     time.Now(),
				}, ctx.Err()
			case <-time.After(retryInterval):
				// 继续重试
			}
		}

		// 发送
		response, lastErr = channel.Send(ctx, request)
		if lastErr == nil {
			// 发送成功
			m.logger.Info("通知发送成功",
				zap.String("channel", channel.GetName()),
				zap.String("message_id", request.MessageID),
				zap.String("recipient", request.RecipientAddr),
				zap.Int("attempts", attempt+1))
			return response, nil
		}

		// 记录错误
		m.logger.Error("发送通知失败",
			zap.String("channel", channel.GetName()),
			zap.String("message_id", request.MessageID),
			zap.String("recipient", request.RecipientAddr),
			zap.Int("attempt", attempt+1),
			zap.Error(lastErr))

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries {
			break
		}
	}

	// 所有重试都失败了
	return &SendResponse{
		Success:      false,
		MessageID:    request.MessageID,
		Status:       "failed",
		ErrorMessage: fmt.Sprintf("发送失败，已重试 %d 次: %v", maxRetries+1, lastErr),
		SendTime:     time.Now(),
	}, lastErr
}

// SendNotificationAsync 异步发送
func (m *Manager) SendNotificationAsync(ctx context.Context, request *SendRequest, delay time.Duration) error {
	// 生成ID
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}

	// 创建任务
	task := asynq.NewTask("notification:send", serializeRequest(request))

	// 设置选项
	opts := []asynq.Option{
		asynq.ProcessIn(delay),
		asynq.TaskID(request.MessageID),
	}

	// 获取重试次数
	if channel, err := m.getChannel(request.RecipientType); err == nil {
		opts = append(opts, asynq.MaxRetry(channel.GetMaxRetries()))
	}

	// 入队
	info, err := m.queueClient.Enqueue(task, opts...)
	if err != nil {
		m.logger.Error("通知任务入队失败",
			zap.String("message_id", request.MessageID),
			zap.Error(err))
		return err
	}

	m.logger.Info("通知任务已入队",
		zap.String("message_id", request.MessageID),
		zap.String("queue", info.Queue),
		zap.Time("next_process_at", info.NextProcessAt))

	return nil
}

// GetAvailableChannels 获取可用渠道
func (m *Manager) GetAvailableChannels() []string {
	var channels []string
	for name, channel := range m.channels {
		if channel.IsEnabled() {
			channels = append(channels, name)
		}
	}
	return channels
}

// BatchSendNotification 批量发送
func (m *Manager) BatchSendNotification(ctx context.Context, requests []*SendRequest) ([]*SendResponse, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	responses := make([]*SendResponse, len(requests))
	var wg sync.WaitGroup

	// 并发发送
	for i, request := range requests {
		wg.Add(1)
		go func(index int, req *SendRequest) {
			defer wg.Done()

			resp, err := m.SendNotification(ctx, req)
			if err != nil {
				resp = &SendResponse{
					Success:      false,
					MessageID:    req.MessageID,
					Status:       "failed",
					ErrorMessage: err.Error(),
					SendTime:     time.Now(),
				}
			}
			responses[index] = resp
		}(i, request)
	}

	wg.Wait()
	return responses, nil
}

// ValidateChannelConfig 验证配置
func (m *Manager) ValidateChannelConfig(channelName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channel, exists := m.channels[channelName]
	if !exists {
		return fmt.Errorf("通知渠道 %s 不存在", channelName)
	}

	return channel.Validate()
}

// ReloadChannel 重新加载
func (m *Manager) ReloadChannel(channelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch channelName {
	case model.NotificationChannelEmail:
		emailConfig := m.config.GetEmail()
		if emailConfig != nil && emailConfig.IsEnabled() {
			if err := emailConfig.Validate(); err != nil {
				return fmt.Errorf("邮箱配置验证失败: %w", err)
			}
			m.channels[model.NotificationChannelEmail] = NewEmailChannel(emailConfig, m.logger)
			m.logger.Info("邮箱通知渠道已重新加载")
		}
	case model.NotificationChannelFeishu:
		feishuConfig := m.config.GetFeishu()
		if feishuConfig != nil && feishuConfig.IsEnabled() {
			if err := feishuConfig.Validate(); err != nil {
				return fmt.Errorf("飞书配置验证失败: %w", err)
			}
			m.channels[model.NotificationChannelFeishu] = NewFeishuChannel(feishuConfig, m.logger)
			m.logger.Info("飞书通知渠道已重新加载")
		}
	default:
		return fmt.Errorf("不支持的通知渠道: %s", channelName)
	}

	return nil
}

// getChannel 获取渠道
func (m *Manager) getChannel(recipientType string) (NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 推断渠道
	var channelName string
	switch recipientType {
	case "email", "user_email":
		channelName = model.NotificationChannelEmail
	case "feishu", "feishu_user", "feishu_group":
		channelName = model.NotificationChannelFeishu
	default:
		// 从地址推断
		channelName = m.inferChannelFromAddress(recipientType)
	}

	channel, exists := m.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("通知渠道 %s 不可用", channelName)
	}

	return channel, nil
}

// inferChannelFromAddress 推断渠道
func (m *Manager) inferChannelFromAddress(address string) string {
	// 邮箱
	if isValidEmail(address) {
		return model.NotificationChannelEmail
	}

	// 飞书ID
	if len(address) > 0 && (address[0] == 'o' || address[0] == 'u') {
		return model.NotificationChannelFeishu
	}

	// 默认邮箱
	return model.NotificationChannelEmail
}

// ProcessNotificationTask 处理任务
func (m *Manager) ProcessNotificationTask(ctx context.Context, task *asynq.Task) error {
	request, err := deserializeRequest(task.Payload())
	if err != nil {
		m.logger.Error("反序列化通知任务失败", zap.Error(err))
		return err
	}

	_, err = m.SendNotification(ctx, request)
	return err
}

// GetChannelStats 获取统计
func (m *Manager) GetChannelStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, channel := range m.channels {
		stats[name] = map[string]interface{}{
			"enabled":        channel.IsEnabled(),
			"max_retries":    channel.GetMaxRetries(),
			"retry_interval": channel.GetRetryInterval().String(),
		}
	}
	return stats
}

// Close 关闭
func (m *Manager) Close() error {
	if m.queueClient != nil {
		return m.queueClient.Close()
	}
	return nil
}
