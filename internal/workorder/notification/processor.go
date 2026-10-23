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
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type Processor struct {
	manager *Manager
	logger  *zap.Logger
}

func NewProcessor(manager *Manager, logger *zap.Logger) *Processor {
	return &Processor{
		manager: manager,
		logger:  logger,
	}
}

func (p *Processor) RegisterTasks(mux *asynq.ServeMux) {
	mux.HandleFunc(model.TaskTypeSendNotification, p.HandleSendNotification)
	mux.HandleFunc(model.TaskTypeBatchSendNotification, p.HandleBatchSendNotification)
	mux.HandleFunc(model.TaskTypeScheduledNotification, p.HandleScheduledNotification)
	mux.HandleFunc(model.TaskTypeRetryFailedNotification, p.HandleRetryFailedNotification)
}

func (p *Processor) HandleSendNotification(ctx context.Context, task *asynq.Task) error {
	request, err := decodeSendNotificationTask(task)
	if err != nil {
		return fmt.Errorf("通知任务无效: %w", err)
	}
	payload := SendNotificationPayload{Request: request}

	p.logger.Info("开始处理通知发送任务",
		zap.String("task_type", task.Type()),
		zap.String("message_id", payload.Request.MessageID))

	// 发送通知
	response, err := p.manager.sendNotification(ctx, payload.Request, true)
	if err != nil {
		p.logger.Error("通知发送失败",
			zap.String("task_type", task.Type()),
			zap.String("message_id", payload.Request.MessageID))
		return err
	}

	p.logger.Info("通知发送成功",
		zap.String("task_type", task.Type()),
		zap.String("message_id", payload.Request.MessageID),
		zap.String("status", response.Status))

	return nil
}

func (p *Processor) HandleBatchSendNotification(ctx context.Context, task *asynq.Task) error {
	var payload BatchSendNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		p.logger.Error("反序列化批量通知任务失败",
			zap.String("task_id", task.Type()))
		return fmt.Errorf("反序列化任务失败: %w", err)
	}

	p.logger.Info("开始处理批量通知发送任务",
		zap.String("task_id", task.Type()),
		zap.Int("batch_size", len(payload.Requests)))

	// 批量发送通知
	responses := make([]*SendResponse, 0, len(payload.Requests))
	var firstErr error
	for _, request := range payload.Requests {
		response, err := p.manager.sendNotification(ctx, request, true)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		responses = append(responses, response)
	}

	successCount := 0
	failedCount := len(payload.Requests) - len(responses)
	for _, response := range responses {
		if response.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	p.logger.Info("批量通知发送完成",
		zap.String("task_id", task.Type()),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount))

	if firstErr != nil {
		// 整批重放会重复发送已成功的收件人，明确失败并禁止任务级重放
		return fmt.Errorf("批量通知部分发送失败: %w: %w", firstErr, asynq.SkipRetry)
	}

	return nil
}

func (p *Processor) HandleScheduledNotification(ctx context.Context, task *asynq.Task) error {
	var payload ScheduledNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		p.logger.Error("反序列化定时通知任务失败",
			zap.String("task_id", task.Type()))
		return fmt.Errorf("反序列化任务失败: %w", err)
	}

	if payload.Request == nil {
		return fmt.Errorf("通知任务缺少request")
	}

	p.logger.Info("开始处理定时通知任务",
		zap.String("task_id", task.Type()),
		zap.String("schedule_type", payload.ScheduleType),
		zap.Time("scheduled_at", payload.ScheduledAt))

	if time.Now().Before(payload.ScheduledAt) {
		p.logger.Warn("定时通知尚未到达执行时间",
			zap.String("task_id", task.Type()),
			zap.Time("scheduled_at", payload.ScheduledAt))
		return fmt.Errorf("尚未到达执行时间")
	}

	// 发送通知
	response, err := p.manager.sendNotification(ctx, payload.Request, true)
	if err != nil {
		p.logger.Error("定时通知发送失败",
			zap.String("task_id", task.Type()))
		return err
	}

	p.logger.Info("定时通知发送成功",
		zap.String("task_id", task.Type()),
		zap.String("status", response.Status))

	return nil
}

func (p *Processor) HandleRetryFailedNotification(ctx context.Context, task *asynq.Task) error {
	var payload RetryFailedNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		p.logger.Error("反序列化重试通知任务失败",
			zap.String("task_id", task.Type()))
		return fmt.Errorf("反序列化任务失败: %w", err)
	}

	if payload.Request == nil {
		return fmt.Errorf("通知任务缺少request")
	}

	p.logger.Info("开始处理重试通知任务",
		zap.String("task_id", task.Type()),
		zap.String("original_message_id", payload.OriginalMessageID),
		zap.Int("retry_count", payload.RetryCount))

	// 发送通知
	response, err := p.manager.sendNotification(ctx, payload.Request, true)
	if err != nil {
		p.logger.Error("重试通知发送失败",
			zap.String("task_id", task.Type()),
			zap.String("original_message_id", payload.OriginalMessageID),
			zap.Int("retry_count", payload.RetryCount))
		return err
	}

	p.logger.Info("重试通知发送成功",
		zap.String("task_id", task.Type()),
		zap.String("original_message_id", payload.OriginalMessageID),
		zap.String("status", response.Status))

	return nil
}

type SendNotificationPayload struct {
	Request   *SendRequest           `json:"request"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type BatchSendNotificationPayload struct {
	Requests  []*SendRequest         `json:"requests"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type ScheduledNotificationPayload struct {
	Request      *SendRequest           `json:"request"`
	ScheduledAt  time.Time              `json:"scheduled_at"`
	ScheduleType string                 `json:"schedule_type"` // once, daily, weekly, monthly
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

type RetryFailedNotificationPayload struct {
	Request           *SendRequest           `json:"request"`
	OriginalMessageID string                 `json:"original_message_id"`
	RetryCount        int                    `json:"retry_count"`
	LastError         string                 `json:"last_error"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
}

func CreateSendNotificationTask(request *SendRequest, metadata map[string]interface{}) (*asynq.Task, error) {
	payload := SendNotificationPayload{
		Request:   request,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	return asynq.NewTask(model.TaskTypeSendNotification, data), nil
}

func CreateBatchSendNotificationTask(requests []*SendRequest, metadata map[string]interface{}) (*asynq.Task, error) {
	payload := BatchSendNotificationPayload{
		Requests:  requests,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	return asynq.NewTask(model.TaskTypeBatchSendNotification, data), nil
}

func CreateScheduledNotificationTask(request *SendRequest, scheduledAt time.Time, scheduleType string, metadata map[string]interface{}) (*asynq.Task, error) {
	payload := ScheduledNotificationPayload{
		Request:      request,
		ScheduledAt:  scheduledAt,
		ScheduleType: scheduleType,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	return asynq.NewTask(model.TaskTypeScheduledNotification, data), nil
}

func CreateRetryFailedNotificationTask(request *SendRequest, originalMessageID string, retryCount int, lastError string, metadata map[string]interface{}) (*asynq.Task, error) {
	payload := RetryFailedNotificationPayload{
		Request:           request,
		OriginalMessageID: originalMessageID,
		RetryCount:        retryCount,
		LastError:         lastError,
		Metadata:          metadata,
		CreatedAt:         time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	return asynq.NewTask(model.TaskTypeRetryFailedNotification, data), nil
}

// decodeSendNotificationTask 兼容两个已存在的任务生产格式，拒绝空 request
func decodeSendNotificationTask(task *asynq.Task) (*SendRequest, error) {
	if task == nil {
		return nil, fmt.Errorf("通知任务不能为空")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(task.Payload(), &fields); err != nil {
		return nil, fmt.Errorf("通知任务JSON无效: %w", err)
	}
	data := task.Payload()
	if raw, ok := fields["request"]; ok {
		data = raw
	}
	var request *SendRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("通知请求JSON无效: %w", err)
	}
	if request == nil {
		return nil, fmt.Errorf("通知任务缺少request")
	}
	return request, nil
}
