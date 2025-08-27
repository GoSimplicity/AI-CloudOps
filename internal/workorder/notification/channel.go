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
	"time"
)

// NotificationChannel 通知渠道接口
type NotificationChannel interface {
	// GetName 获取渠道名称
	GetName() string
	
	// Send 发送通知
	Send(ctx context.Context, request *SendRequest) (*SendResponse, error)
	
	// Validate 验证配置
	Validate() error
	
	// IsEnabled 是否启用
	IsEnabled() bool
	
	// GetMaxRetries 获取最大重试次数
	GetMaxRetries() int
	
	// GetRetryInterval 获取重试间隔
	GetRetryInterval() time.Duration
}

// SendRequest 发送请求
type SendRequest struct {
	// 基础信息
	MessageID     string            `json:"message_id"`     // 消息ID
	Subject       string            `json:"subject"`        // 主题
	Content       string            `json:"content"`        // 内容
	Priority      int8              `json:"priority"`       // 优先级 1-高 2-中 3-低
	
	// 接收人信息
	RecipientType string            `json:"recipient_type"` // 接收人类型
	RecipientID   string            `json:"recipient_id"`   // 接收人ID
	RecipientAddr string            `json:"recipient_addr"` // 接收人地址(邮箱/手机号等)
	RecipientName string            `json:"recipient_name"` // 接收人名称
	
	// 工单相关
	InstanceID     *int              `json:"instance_id,omitempty"`     // 工单实例ID
	EventType      string            `json:"event_type"`                // 事件类型
	
	// 扩展数据
	Metadata      map[string]interface{} `json:"metadata,omitempty"`      // 元数据
	Templates     map[string]string      `json:"templates,omitempty"`     // 模板变量
	Attachments   []Attachment           `json:"attachments,omitempty"`   // 附件
}

// SendResponse 发送响应
type SendResponse struct {
	Success      bool                   `json:"success"`       // 是否成功
	MessageID    string                 `json:"message_id"`    // 消息ID
	ExternalID   string                 `json:"external_id"`   // 外部系统消息ID
	Status       string                 `json:"status"`        // 状态
	ErrorMessage string                 `json:"error_message"` // 错误信息
	Cost         *float64               `json:"cost,omitempty"` // 发送成本
	SendTime     time.Time              `json:"send_time"`     // 发送时间
	ResponseData map[string]interface{} `json:"response_data,omitempty"` // 响应数据
}

// Attachment 附件
type Attachment struct {
	Name        string `json:"name"`         // 附件名称
	Content     []byte `json:"content"`      // 附件内容
	ContentType string `json:"content_type"` // 内容类型
	Size        int64  `json:"size"`         // 大小
}

// ChannelConfig 渠道配置接口
type ChannelConfig interface {
	GetChannelName() string
	Validate() error
}

// BaseChannelConfig 基础渠道配置
type BaseChannelConfig struct {
	Enabled       bool          `json:"enabled" yaml:"enabled"`             // 是否启用
	MaxRetries    int           `json:"max_retries" yaml:"max_retries"`     // 最大重试次数
	RetryInterval time.Duration `json:"retry_interval" yaml:"retry_interval"` // 重试间隔
	Timeout       time.Duration `json:"timeout" yaml:"timeout"`             // 超时时间
}

// GetMaxRetries 获取最大重试次数
func (c *BaseChannelConfig) GetMaxRetries() int {
	if c.MaxRetries <= 0 {
		return 3 // 默认重试3次
	}
	return c.MaxRetries
}

// GetRetryInterval 获取重试间隔
func (c *BaseChannelConfig) GetRetryInterval() time.Duration {
	if c.RetryInterval <= 0 {
		return 5 * time.Minute // 默认5分钟
	}
	return c.RetryInterval
}

// GetTimeout 获取超时时间
func (c *BaseChannelConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 30 * time.Second // 默认30秒
	}
	return c.Timeout
}

// IsEnabled 是否启用
func (c *BaseChannelConfig) IsEnabled() bool {
	return c.Enabled
}
