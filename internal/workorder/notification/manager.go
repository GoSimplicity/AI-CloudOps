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

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// Manager 通知管理器
type Manager struct {
	channels    map[string]NotificationChannel
	queueClient *asynq.Client
	logger      *zap.Logger
	config      *NotificationConfig
	mu          sync.RWMutex
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Email  *EmailConfig  `json:"email" yaml:"email"`
	Feishu *FeishuConfig `json:"feishu" yaml:"feishu"`
}

// NewManager 创建通知管理器
func NewManager(config *NotificationConfig, queueClient *asynq.Client, logger *zap.Logger) (*Manager, error) {
	manager := &Manager{
		channels:    make(map[string]NotificationChannel),
		queueClient: queueClient,
		logger:      logger,
		config:      config,
	}

	// 初始化邮箱渠道
	if config.Email != nil && config.Email.IsEnabled() {
		if err := config.Email.Validate(); err != nil {
			logger.Warn("邮箱配置验证失败", zap.Error(err))
		} else {
			emailChannel := NewEmailChannel(config.Email, logger)
			manager.channels["email"] = emailChannel
			logger.Info("邮箱通知渠道已启用")
		}
	}

	// 初始化飞书渠道
	if config.Feishu != nil && config.Feishu.IsEnabled() {
		if err := config.Feishu.Validate(); err != nil {
			logger.Warn("飞书配置验证失败", zap.Error(err))
		} else {
			feishuChannel := NewFeishuChannel(config.Feishu, logger)
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
	// 生成消息ID
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}

	// 获取渠道
	channel, err := m.getChannel(request.RecipientType)
	if err != nil {
		return nil, err
	}

	// 检查渠道是否启用
	if !channel.IsEnabled() {
		return nil, fmt.Errorf("通知渠道 %s 未启用", channel.GetName())
	}

	// 发送通知
	response, err := channel.Send(ctx, request)
	if err != nil {
		m.logger.Error("发送通知失败",
			zap.String("channel", channel.GetName()),
			zap.String("message_id", request.MessageID),
			zap.String("recipient", request.RecipientAddr),
			zap.Error(err))
		return response, err
	}

	m.logger.Info("通知发送成功",
		zap.String("channel", channel.GetName()),
		zap.String("message_id", request.MessageID),
		zap.String("recipient", request.RecipientAddr))

	return response, nil
}

// SendNotificationAsync 异步发送通知
func (m *Manager) SendNotificationAsync(ctx context.Context, request *SendRequest, delay time.Duration) error {
	// 生成消息ID
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}

	// 创建异步任务
	task := asynq.NewTask("notification:send", serializeRequest(request))

	// 设置任务选项
	opts := []asynq.Option{
		asynq.ProcessIn(delay),
		asynq.TaskID(request.MessageID),
	}

	// 获取渠道配置重试次数
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

// GetAvailableChannels 获取可用的通知渠道
func (m *Manager) GetAvailableChannels() []string {
	var channels []string
	for name, channel := range m.channels {
		if channel.IsEnabled() {
			channels = append(channels, name)
		}
	}
	return channels
}

// BatchSendNotification 批量发送通知
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

// ValidateChannelConfig 验证渠道配置
func (m *Manager) ValidateChannelConfig(channelName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channel, exists := m.channels[channelName]
	if !exists {
		return fmt.Errorf("通知渠道 %s 不存在", channelName)
	}

	return channel.Validate()
}

// ReloadChannel 重新加载渠道配置
func (m *Manager) ReloadChannel(channelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch channelName {
	case "email":
		if m.config.Email != nil && m.config.Email.IsEnabled() {
			if err := m.config.Email.Validate(); err != nil {
				return fmt.Errorf("邮箱配置验证失败: %w", err)
			}
			m.channels["email"] = NewEmailChannel(m.config.Email, m.logger)
			m.logger.Info("邮箱通知渠道已重新加载")
		}
	case "feishu":
		if m.config.Feishu != nil && m.config.Feishu.IsEnabled() {
			if err := m.config.Feishu.Validate(); err != nil {
				return fmt.Errorf("飞书配置验证失败: %w", err)
			}
			m.channels["feishu"] = NewFeishuChannel(m.config.Feishu, m.logger)
			m.logger.Info("飞书通知渠道已重新加载")
		}
	default:
		return fmt.Errorf("不支持的通知渠道: %s", channelName)
	}

	return nil
}

// getChannel 根据接收人类型获取对应的通知渠道
func (m *Manager) getChannel(recipientType string) (NotificationChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 根据接收人类型推断渠道
	var channelName string
	switch recipientType {
	case "email", "user_email":
		channelName = "email"
	case "feishu", "feishu_user", "feishu_group":
		channelName = "feishu"
	default:
		// 如果类型不明确，尝试从接收人地址推断
		channelName = m.inferChannelFromAddress(recipientType)
	}

	channel, exists := m.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("通知渠道 %s 不可用", channelName)
	}

	return channel, nil
}

// inferChannelFromAddress 从接收人地址推断渠道类型
func (m *Manager) inferChannelFromAddress(address string) string {
	// 邮箱地址
	if isValidEmail(address) {
		return "email"
	}

	// 飞书用户ID或群组ID
	if len(address) > 0 && (address[0] == 'o' || address[0] == 'u') {
		return "feishu"
	}

	// 默认返回邮箱
	return "email"
}

// ProcessNotificationTask 处理通知任务（用于队列消费者）
func (m *Manager) ProcessNotificationTask(ctx context.Context, task *asynq.Task) error {
	request, err := deserializeRequest(task.Payload())
	if err != nil {
		m.logger.Error("反序列化通知任务失败", zap.Error(err))
		return err
	}

	_, err = m.SendNotification(ctx, request)
	return err
}

// GetChannelStats 获取渠道统计信息
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

// Close 关闭管理器
func (m *Manager) Close() error {
	if m.queueClient != nil {
		return m.queueClient.Close()
	}
	return nil
}
