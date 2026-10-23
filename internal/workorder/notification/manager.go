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

type NotificationConfig interface {
	GetEmail() EmailConfig
	GetFeishu() FeishuConfig
}

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

type notificationQueue interface {
	EnqueueContext(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)
	Close() error
}

type Manager struct {
	channels    map[string]NotificationChannel
	queueClient notificationQueue
	logger      *zap.Logger
	config      NotificationConfig
	mu          sync.RWMutex
}

func NewManager(config NotificationConfig, queueClient *asynq.Client, logger *zap.Logger) (*Manager, error) {
	manager := &Manager{
		channels: make(map[string]NotificationChannel),
		logger:   logger,
		config:   config,
	}

	if queueClient != nil {
		manager.queueClient = queueClient
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

// SendNotification 同步发送通知消息
func (m *Manager) SendNotification(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	return m.sendNotification(ctx, request, false)
}

// 队列负责重试时每次执行只交付一次，避免两层重试相乘
func (m *Manager) sendNotification(ctx context.Context, request *SendRequest, singleAttempt bool) (*SendResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("通知请求不能为空")
	}
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}
	failure := func(status string, err error) (*SendResponse, error) {
		return &SendResponse{MessageID: request.MessageID, Status: status, ErrorMessage: err.Error(), SendTime: time.Now()}, err
	}
	if err := ctx.Err(); err != nil {
		return failure("cancelled", err)
	}
	channel, err := m.requestChannel(request)
	if err != nil {
		return failure("failed", err)
	}
	maxRetries, interval, err := requestRetryPolicy(request, channel)
	if err != nil {
		return failure("failed", err)
	}
	if singleAttempt {
		maxRetries = 0
	}
	normalized, err := NormalizeAddress(channel.GetName(), request.RecipientAddr)
	if err != nil {
		return failure("failed", err)
	}
	request.RecipientAddr = normalized
	if !channel.IsEnabled() {
		return failure("failed", fmt.Errorf("通知渠道 %s 未启用", channel.GetName()))
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return failure("cancelled", ctx.Err())
			case <-timer.C:
			}
		}
		if err := ctx.Err(); err != nil {
			return failure("cancelled", err)
		}
		response, sendErr := channel.Send(ctx, request)
		lastErr = sendErr
		if lastErr == nil && (response == nil || !response.Success) {
			lastErr = fmt.Errorf("通知渠道未确认发送成功")
		}
		if lastErr == nil {
			response.MessageID = request.MessageID
			m.logger.Info("通知发送成功", zap.String("channel", channel.GetName()), zap.String("message_id", request.MessageID), zap.Int("attempts", attempt+1))
			return response, nil
		}
		if err := ctx.Err(); err != nil {
			return failure("cancelled", err)
		}
		m.logger.Warn("发送通知失败", zap.String("channel", channel.GetName()), zap.String("message_id", request.MessageID), zap.Int("attempt", attempt+1))
	}
	response, _ := failure("failed", fmt.Errorf("发送失败，已尝试 %d 次: %w", maxRetries+1, lastErr))
	return response, lastErr
}

func (m *Manager) requestChannel(request *SendRequest) (NotificationChannel, error) {
	route := request.Channel
	if route == "" {
		route = request.RecipientType
	}
	return m.getChannel(route)
}

func requestRetryPolicy(request *SendRequest, channel NotificationChannel) (int, time.Duration, error) {
	retries, interval := channel.GetMaxRetries(), channel.GetRetryInterval()
	if request.MaxRetries != nil {
		retries = *request.MaxRetries
	}
	if retries < 0 || retries > 100 {
		return 0, 0, fmt.Errorf("通知额外重试次数必须在0至100之间")
	}
	if request.RetryIntervalMinutes != nil {
		minutes := *request.RetryIntervalMinutes
		// 先验证再转换，避免持久化配置的乘法溢出变成负间隔或短间隔
		if minutes < 0 || uint64(minutes) > uint64((1<<63-1)/time.Minute) {
			return 0, 0, fmt.Errorf("通知重试间隔超出有效分钟范围")
		}
		interval = time.Duration(minutes) * time.Minute
	}
	if interval < 0 {
		return 0, 0, fmt.Errorf("通知重试间隔不能为负数")
	}
	return retries, interval, nil
}

// SendNotificationAsync 异步发送通知消息
func (m *Manager) SendNotificationAsync(ctx context.Context, request *SendRequest, delay time.Duration) error {
	if request == nil {
		return fmt.Errorf("通知请求不能为空")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay < 0 {
		return fmt.Errorf("通知排队延迟不能为负数")
	}
	if request.MessageID == "" {
		request.MessageID = uuid.New().String()
	}
	channel, err := m.requestChannel(request)
	if err != nil {
		return err
	}
	maxRetries, _, err := requestRetryPolicy(request, channel)
	if err != nil {
		return err
	}
	if !channel.IsEnabled() {
		return fmt.Errorf("通知渠道 %s 未启用", channel.GetName())
	}
	normalized, err := NormalizeAddress(channel.GetName(), request.RecipientAddr)
	if err != nil {
		return err
	}
	request.RecipientAddr = normalized
	// 使用已发布的 request envelope；worker 同时接受早期直接序列化请求的格式
	task, err := CreateSendNotificationTask(request, nil)
	if err != nil {
		return err
	}
	if m.queueClient == nil {
		return fmt.Errorf("通知队列未配置")
	}
	info, err := m.queueClient.EnqueueContext(ctx, task,
		asynq.ProcessIn(delay), asynq.TaskID(request.MessageID), asynq.MaxRetry(maxRetries))
	if err != nil {
		m.logger.Warn("通知任务入队失败", zap.String("message_id", request.MessageID))
		return err
	}
	m.logger.Info("通知任务已入队", zap.String("message_id", request.MessageID), zap.String("queue", info.Queue))
	return nil
}

// RetryDelay 供通知任务的 Asynq Server 使用；其他任务保持 Asynq 默认退避
func (m *Manager) RetryDelay(n int, err error, task *asynq.Task) time.Duration {
	if task != nil && task.Type() == model.TaskTypeSendNotification {
		request, decodeErr := decodeSendNotificationTask(task)
		if decodeErr == nil {
			channel, channelErr := m.requestChannel(request)
			if channelErr == nil {
				_, interval, policyErr := requestRetryPolicy(request, channel)
				if policyErr == nil {
					return interval
				}
			}
		}
	}
	return asynq.DefaultRetryDelayFunc(n, err, task)
}

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
			if err != nil && resp == nil {
				messageID := ""
				if req != nil {
					messageID = req.MessageID
				}
				resp = &SendResponse{
					Success:      false,
					MessageID:    messageID,
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

func (m *Manager) ValidateChannelConfig(channelName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channel, exists := m.channels[channelName]
	if !exists {
		return fmt.Errorf("通知渠道 %s 不存在", channelName)
	}

	return channel.Validate()
}

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
		return nil, fmt.Errorf("未知通知渠道: %s", recipientType)
	}

	channel, exists := m.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("通知渠道 %s 不可用", channelName)
	}

	return channel, nil
}

func (m *Manager) ProcessNotificationTask(ctx context.Context, task *asynq.Task) error {
	request, err := decodeSendNotificationTask(task)
	if err != nil {
		m.logger.Error("反序列化通知任务失败", zap.Error(err))
		return err
	}

	_, err = m.sendNotification(ctx, request, true)
	return err
}

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

func (m *Manager) Close() error {
	if m.queueClient != nil {
		return m.queueClient.Close()
	}
	return nil
}
