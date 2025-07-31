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
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package model

import (
	"time"
)

// 发送状态常量
const (
	NotificationSendStatusPending   int8 = 1 // 待发送
	NotificationSendStatusSending   int8 = 2 // 发送中
	NotificationSendStatusSuccess   int8 = 3 // 发送成功
	NotificationSendStatusFailed    int8 = 4 // 发送失败
	NotificationSendStatusCancelled int8 = 5 // 已取消
)

// 通知状态常量
const (
	NotificationStatusDisabled int8 = 0 // 禁用
	NotificationStatusEnabled  int8 = 1 // 启用
	NotificationStatusSuspended int8 = 2 // 暂停
)

// 通知渠道类型常量
const (
	NotificationChannelEmail    = "email"    // 邮箱
	NotificationChannelSMS      = "sms"      // 短信
	NotificationChannelWebhook  = "webhook"  // Webhook
	NotificationChannelFeishu   = "feishu"   // 飞书
	NotificationChannelDingtalk = "dingtalk" // 钉钉
	NotificationChannelWechat   = "wechat"   // 企业微信
	NotificationChannelSlack    = "slack"    // Slack
	NotificationChannelTelegram = "telegram" // Telegram
	NotificationChannelBrowser  = "browser"  // 浏览器推送
	NotificationChannelApp      = "app"      // APP推送
)

// 触发事件类型常量
const (
	NotificationEventCreated     = "created"      // 工单创建
	NotificationEventSubmitted   = "submitted"    // 工单提交
	NotificationEventAssigned    = "assigned"     // 工单分配
	NotificationEventApproved    = "approved"     // 工单批准
	NotificationEventRejected    = "rejected"     // 工单拒绝
	NotificationEventTransferred = "transferred"  // 工单转交
	NotificationEventCompleted   = "completed"    // 工单完成
	NotificationEventCancelled   = "cancelled"    // 工单取消
	NotificationEventOverdue     = "overdue"      // 工单超时
	NotificationEventCommented   = "commented"    // 新增评论
	NotificationEventUpdated     = "updated"      // 工单更新
	NotificationEventSuspended   = "suspended"    // 工单暂停
	NotificationEventResumed     = "resumed"      // 工单恢复
	NotificationEventReopened    = "reopened"     // 工单重开
	NotificationEventReminder    = "reminder"     // 定时提醒
)

// 触发类型常量
const (
	NotificationTriggerImmediate = "immediate" // 立即发送
	NotificationTriggerScheduled = "scheduled" // 定时发送
	NotificationTriggerCondition = "condition" // 条件触发
	NotificationTriggerManual    = "manual"    // 手动发送
)

// 接收人类型常量
const (
	NotificationRecipientCreator  = "creator"   // 创建人
	NotificationRecipientAssignee = "assignee"  // 处理人
	NotificationRecipientManager  = "manager"   // 管理员
	NotificationRecipientCustom   = "custom"    // 自定义用户
	NotificationRecipientRole     = "role"      // 角色
	NotificationRecipientDept     = "dept"      // 部门
	NotificationRecipientGroup    = "group"     // 用户组
)

// WorkorderNotification 工单通知配置实体
type WorkorderNotification struct {
	Model
	Name            string     `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:通知配置名称"`
	Description     string     `json:"description" gorm:"column:description;type:varchar(1000);comment:通知配置描述"`
	ProcessID       *int       `json:"process_id" gorm:"column:process_id;index;comment:关联流程ID"`
	TemplateID      *int       `json:"template_id" gorm:"column:template_id;index;comment:关联模板ID"`
	CategoryID      *int       `json:"category_id" gorm:"column:category_id;index;comment:关联分类ID"`
	EventTypes      StringList `json:"event_types" gorm:"column:event_types;not null;comment:触发事件类型"`
	TriggerType     string     `json:"trigger_type" gorm:"column:trigger_type;type:varchar(20);not null;default:'immediate';comment:触发类型"`
	TriggerCondition JSONMap   `json:"trigger_condition" gorm:"column:trigger_condition;type:json;comment:触发条件"`
	Channels        StringList `json:"channels" gorm:"column:channels;not null;comment:通知渠道"`
	RecipientTypes  StringList `json:"recipient_types" gorm:"column:recipient_types;not null;comment:接收人类型"`
	RecipientUsers  StringList `json:"recipient_users" gorm:"column:recipient_users;comment:自定义接收人用户ID"`
	RecipientRoles  StringList `json:"recipient_roles" gorm:"column:recipient_roles;comment:接收人角色ID"`
	RecipientDepts  StringList `json:"recipient_depts" gorm:"column:recipient_depts;comment:接收人部门ID"`
	MessageTemplate string     `json:"message_template" gorm:"column:message_template;type:text;not null;comment:消息模板"`
	SubjectTemplate string     `json:"subject_template" gorm:"column:subject_template;type:varchar(500);comment:主题模板"`
	ScheduledTime   *time.Time `json:"scheduled_time" gorm:"column:scheduled_time;comment:定时发送时间"`
	RepeatInterval  *int       `json:"repeat_interval" gorm:"column:repeat_interval;comment:重复间隔(分钟)"`
	MaxRetries      int        `json:"max_retries" gorm:"column:max_retries;not null;default:3;comment:最大重试次数"`
	RetryInterval   int        `json:"retry_interval" gorm:"column:retry_interval;not null;default:5;comment:重试间隔(分钟)"`
	Status          int8       `json:"status" gorm:"column:status;not null;default:1;index;comment:状态"`
	Priority        int8       `json:"priority" gorm:"column:priority;not null;default:2;comment:优先级：1-低，2-普通，3-高，4-紧急"`
	CreatorID       int        `json:"creator_id" gorm:"column:creator_id;not null;index;comment:创建人ID"`
	CreatorName     string     `json:"creator_name" gorm:"-"`
	IsDefault       bool       `json:"is_default" gorm:"column:is_default;not null;default:false;comment:是否默认配置"`
	Settings        JSONMap    `json:"settings" gorm:"column:settings;type:json;comment:通知设置"`

	// 关联信息（不存储到数据库）
	ProcessName  string `json:"process_name,omitempty" gorm:"-"`
	TemplateName string `json:"template_name,omitempty" gorm:"-"`
	CategoryName string `json:"category_name,omitempty" gorm:"-"`
}

// TableName 指定工单通知配置表名
func (WorkorderNotification) TableName() string {
	return "cl_workorder_notification"
}

// WorkorderNotificationLog 工单通知发送记录实体
type WorkorderNotificationLog struct {
	Model
	NotificationID int        `json:"notification_id" gorm:"column:notification_id;not null;index;index:idx_notification_status,priority:1;comment:通知配置ID"`
	InstanceID     *int       `json:"instance_id" gorm:"column:instance_id;index;comment:工单实例ID"`
	EventType      string     `json:"event_type" gorm:"column:event_type;type:varchar(50);not null;index;comment:触发事件类型"`
	Channel        string     `json:"channel" gorm:"column:channel;type:varchar(20);not null;index;comment:发送渠道"`
	RecipientType  string     `json:"recipient_type" gorm:"column:recipient_type;type:varchar(20);not null;comment:接收人类型"`
	RecipientID    string     `json:"recipient_id" gorm:"column:recipient_id;type:varchar(100);not null;index;comment:接收人ID"`
	RecipientName  string     `json:"recipient_name" gorm:"column:recipient_name;type:varchar(200);comment:接收人名称"`
	RecipientAddr  string     `json:"recipient_addr" gorm:"column:recipient_addr;type:varchar(500);not null;comment:接收人地址"`
	Subject        string     `json:"subject" gorm:"column:subject;type:varchar(500);comment:消息主题"`
	Content        string     `json:"content" gorm:"column:content;type:text;not null;comment:发送内容"`
	Status         int8       `json:"status" gorm:"column:status;not null;index;index:idx_notification_status,priority:2;comment:发送状态"`
	ErrorMessage   string     `json:"error_message" gorm:"column:error_message;type:text;comment:错误信息"`
	ResponseData   JSONMap    `json:"response_data" gorm:"column:response_data;type:json;comment:响应数据"`
	SendAt         time.Time  `json:"send_at" gorm:"column:send_at;not null;comment:发送时间"`
	DeliveredAt    *time.Time `json:"delivered_at" gorm:"column:delivered_at;comment:送达时间"`
	ReadAt         *time.Time `json:"read_at" gorm:"column:read_at;comment:阅读时间"`
	Cost           *float64   `json:"cost" gorm:"column:cost;comment:发送成本"`
	RetryCount     int        `json:"retry_count" gorm:"column:retry_count;not null;default:0;comment:重试次数"`
	NextRetryAt    *time.Time `json:"next_retry_at" gorm:"column:next_retry_at;comment:下次重试时间"`
	SenderID       int        `json:"sender_id" gorm:"column:sender_id;not null;comment:发送人ID"`
	SenderName     string     `json:"sender_name" gorm:"-"`
	ExtendedData   JSONMap    `json:"extended_data" gorm:"column:extended_data;type:json;comment:扩展数据"`

	// 关联信息（不存储到数据库）
	NotificationName string `json:"notification_name,omitempty" gorm:"-"`
	InstanceTitle    string `json:"instance_title,omitempty" gorm:"-"`
}

// TableName 指定工单通知发送记录表名
func (WorkorderNotificationLog) TableName() string {
	return "cl_workorder_notification_log"
}

// CreateWorkorderNotificationReq 创建工单通知配置请求
type CreateWorkorderNotificationReq struct {
	Name             string                 `json:"name" binding:"required,min=1,max=200"`
	Description      string                 `json:"description" binding:"omitempty,max=1000"`
	ProcessID        *int                   `json:"process_id" binding:"omitempty,min=1"`
	TemplateID       *int                   `json:"template_id" binding:"omitempty,min=1"`
	CategoryID       *int                   `json:"category_id" binding:"omitempty,min=1"`
	EventTypes       []string               `json:"event_types" binding:"required,min=1"`
	TriggerType      string                 `json:"trigger_type" binding:"required,oneof=immediate scheduled condition manual"`
	TriggerCondition map[string]interface{} `json:"trigger_condition"`
	Channels         []string               `json:"channels" binding:"required,min=1"`
	RecipientTypes   []string               `json:"recipient_types" binding:"required,min=1"`
	RecipientUsers   []string               `json:"recipient_users"`
	RecipientRoles   []string               `json:"recipient_roles"`
	RecipientDepts   []string               `json:"recipient_depts"`
	MessageTemplate  string                 `json:"message_template" binding:"required,min=1"`
	SubjectTemplate  string                 `json:"subject_template" binding:"omitempty,max=500"`
	ScheduledTime    *time.Time             `json:"scheduled_time"`
	RepeatInterval   *int                   `json:"repeat_interval" binding:"omitempty,min=1"`
	MaxRetries       int                    `json:"max_retries" binding:"omitempty,min=0,max=10"`
	RetryInterval    int                    `json:"retry_interval" binding:"omitempty,min=1,max=1440"`
	Priority         int8                   `json:"priority" binding:"omitempty,oneof=1 2 3 4"`
	IsDefault        bool                   `json:"is_default"`
	Settings         map[string]interface{} `json:"settings"`
}

// UpdateWorkorderNotificationReq 更新工单通知配置请求
type UpdateWorkorderNotificationReq struct {
	ID               int                    `json:"id" binding:"required,min=1"`
	Name             string                 `json:"name" binding:"required,min=1,max=200"`
	Description      string                 `json:"description" binding:"omitempty,max=1000"`
	ProcessID        *int                   `json:"process_id" binding:"omitempty,min=1"`
	TemplateID       *int                   `json:"template_id" binding:"omitempty,min=1"`
	CategoryID       *int                   `json:"category_id" binding:"omitempty,min=1"`
	EventTypes       []string               `json:"event_types" binding:"required,min=1"`
	TriggerType      string                 `json:"trigger_type" binding:"required,oneof=immediate scheduled condition manual"`
	TriggerCondition map[string]interface{} `json:"trigger_condition"`
	Channels         []string               `json:"channels" binding:"required,min=1"`
	RecipientTypes   []string               `json:"recipient_types" binding:"required,min=1"`
	RecipientUsers   []string               `json:"recipient_users"`
	RecipientRoles   []string               `json:"recipient_roles"`
	RecipientDepts   []string               `json:"recipient_depts"`
	MessageTemplate  string                 `json:"message_template" binding:"required,min=1"`
	SubjectTemplate  string                 `json:"subject_template" binding:"omitempty,max=500"`
	ScheduledTime    *time.Time             `json:"scheduled_time"`
	RepeatInterval   *int                   `json:"repeat_interval" binding:"omitempty,min=1"`
	MaxRetries       int                    `json:"max_retries" binding:"omitempty,min=0,max=10"`
	RetryInterval    int                    `json:"retry_interval" binding:"omitempty,min=1,max=1440"`
	Status           int8                   `json:"status" binding:"required,oneof=0 1 2"`
	Priority         int8                   `json:"priority" binding:"omitempty,oneof=1 2 3 4"`
	IsDefault        bool                   `json:"is_default"`
	Settings         map[string]interface{} `json:"settings"`
}

// DeleteWorkorderNotificationReq 删除工单通知配置请求
type DeleteWorkorderNotificationReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderNotificationReq 获取工单通知配置详情请求
type DetailWorkorderNotificationReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderNotificationReq 工单通知配置列表请求
type ListWorkorderNotificationReq struct {
	ListReq
	ProcessID   *int    `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
	TemplateID  *int    `json:"template_id" form:"template_id" binding:"omitempty,min=1"`
	CategoryID  *int    `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	Status      *int8   `json:"status" form:"status" binding:"omitempty,oneof=0 1 2"`
	TriggerType *string `json:"trigger_type" form:"trigger_type" binding:"omitempty,oneof=immediate scheduled condition manual"`
	Channel     *string `json:"channel" form:"channel"`
	EventType   *string `json:"event_type" form:"event_type"`
	IsDefault   *bool   `json:"is_default" form:"is_default"`
}

// EnableWorkorderNotificationReq 启用工单通知配置请求
type EnableWorkorderNotificationReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// DisableWorkorderNotificationReq 禁用工单通知配置请求
type DisableWorkorderNotificationReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// TestWorkorderNotificationReq 测试工单通知配置请求
type TestWorkorderNotificationReq struct {
	ID         int                    `json:"id" binding:"required,min=1"`
	InstanceID *int                   `json:"instance_id" binding:"omitempty,min=1"`
	EventType  string                 `json:"event_type" binding:"required"`
	TestData   map[string]interface{} `json:"test_data"`
}

// CloneWorkorderNotificationReq 克隆工单通知配置请求
type CloneWorkorderNotificationReq struct {
	ID   int    `json:"id" binding:"required,min=1"`
	Name string `json:"name" binding:"required,min=1,max=200"`
}

// BatchUpdateNotificationStatusReq 批量更新通知配置状态请求
type BatchUpdateNotificationStatusReq struct {
	IDs    []int `json:"ids" binding:"required,min=1,dive,min=1"`
	Status int8  `json:"status" binding:"required,oneof=0 1 2"`
}

// SetDefaultNotificationReq 设置默认通知配置请求
type SetDefaultNotificationReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// ListWorkorderNotificationLogReq 工单通知发送记录列表请求
type ListWorkorderNotificationLogReq struct {
	ListReq
	NotificationID *int       `json:"notification_id" form:"notification_id" binding:"omitempty,min=1"`
	InstanceID     *int       `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	EventType      *string    `json:"event_type" form:"event_type"`
	Channel        *string    `json:"channel" form:"channel"`
	Status         *int8    `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5"`
	RecipientType  *string    `json:"recipient_type" form:"recipient_type"`
	StartDate      *time.Time `json:"start_date" form:"start_date"`
	EndDate        *time.Time `json:"end_date" form:"end_date"`
}

// GetNotificationLogDetailReq 获取通知发送记录详情请求
type GetNotificationLogDetailReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ResendNotificationReq 重新发送通知请求
type ResendNotificationReq struct {
	LogID int `json:"log_id" binding:"required,min=1"`
}

// BatchResendNotificationReq 批量重新发送通知请求
type BatchResendNotificationReq struct {
	LogIDs []int `json:"log_ids" binding:"required,min=1,dive,min=1"`
}

// GetNotificationStatisticsReq 获取通知统计请求
type GetNotificationStatisticsReq struct {
	NotificationID *int       `json:"notification_id" form:"notification_id" binding:"omitempty,min=1"`
	StartDate      *time.Time `json:"start_date" form:"start_date"`
	EndDate        *time.Time `json:"end_date" form:"end_date"`
	GroupBy        string     `json:"group_by" form:"group_by" binding:"omitempty,oneof=channel event_type status day week month"`
}

// WorkorderNotificationStatistics 工单通知统计
type WorkorderNotificationStatistics struct {
	TotalCount        int64   `json:"total_count"`         // 总通知数
	EnabledCount      int64   `json:"enabled_count"`       // 启用数量
	DisabledCount     int64   `json:"disabled_count"`      // 禁用数量
	SuspendedCount    int64   `json:"suspended_count"`     // 暂停数量
	DefaultCount      int64   `json:"default_count"`       // 默认配置数量
	TotalSent         int64   `json:"total_sent"`          // 总发送次数
	TotalSuccess      int64   `json:"total_success"`       // 总成功次数
	TotalFailed       int64   `json:"total_failed"`        // 总失败次数
	SuccessRate       float64 `json:"success_rate"`        // 成功率
	AvgResponseTime   float64 `json:"avg_response_time"`   // 平均响应时间(毫秒)
	AvgDeliveryTime   float64 `json:"avg_delivery_time"`   // 平均送达时间(毫秒)
	TotalCost         float64 `json:"total_cost"`          // 总成本
	ChannelStats      []NotificationChannelStats `json:"channel_stats"` // 渠道统计
	EventStats        []NotificationEventStats   `json:"event_stats"`   // 事件统计
}

// NotificationChannelStats 通知渠道统计
type NotificationChannelStats struct {
	Channel     string  `json:"channel"`      // 渠道
	Count       int64   `json:"count"`        // 数量
	SuccessRate float64 `json:"success_rate"` // 成功率
	AvgCost     float64 `json:"avg_cost"`     // 平均成本
}

// NotificationEventStats 通知事件统计
type NotificationEventStats struct {
	EventType   string  `json:"event_type"`   // 事件类型
	Count       int64   `json:"count"`        // 数量
	SuccessRate float64 `json:"success_rate"` // 成功率
}

// WorkorderNotificationChart 工单通知图表数据
type WorkorderNotificationChart struct {
	Date         string `json:"date"`          // 日期
	Channel      string `json:"channel"`       // 渠道
	EventType    string `json:"event_type"`    // 事件类型
	Status       string `json:"status"`        // 状态
	Count        int64  `json:"count"`         // 数量
	SuccessCount int64  `json:"success_count"` // 成功数量
	FailedCount  int64  `json:"failed_count"`  // 失败数量
}

// WorkorderNotificationQueue 工单通知队列实体
type WorkorderNotificationQueue struct {
	Model
	NotificationID int                    `json:"notification_id" gorm:"column:notification_id;not null;index;comment:通知配置ID"`
	InstanceID     *int                   `json:"instance_id" gorm:"column:instance_id;index;comment:工单实例ID"`
	EventType      string                 `json:"event_type" gorm:"column:event_type;type:varchar(50);not null;index;comment:触发事件类型"`
	Channel        string                 `json:"channel" gorm:"column:channel;type:varchar(20);not null;comment:发送渠道"`
	RecipientType  string                 `json:"recipient_type" gorm:"column:recipient_type;type:varchar(20);not null;comment:接收人类型"`
	RecipientID    string                 `json:"recipient_id" gorm:"column:recipient_id;type:varchar(100);not null;comment:接收人ID"`
	RecipientAddr  string                 `json:"recipient_addr" gorm:"column:recipient_addr;type:varchar(500);not null;comment:接收人地址"`
	Subject        string                 `json:"subject" gorm:"column:subject;type:varchar(500);comment:消息主题"`
	Content        string                 `json:"content" gorm:"column:content;type:text;not null;comment:发送内容"`
	Priority       int8                   `json:"priority" gorm:"column:priority;not null;default:2;index;comment:优先级"`
	Status         int8                   `json:"status" gorm:"column:status;not null;default:1;index;comment:状态"`
	ScheduledAt    time.Time              `json:"scheduled_at" gorm:"column:scheduled_at;not null;index;comment:计划发送时间"`
	ProcessedAt    *time.Time             `json:"processed_at" gorm:"column:processed_at;comment:处理时间"`
	RetryCount     int                    `json:"retry_count" gorm:"column:retry_count;not null;default:0;comment:重试次数"`
	NextRetryAt    *time.Time             `json:"next_retry_at" gorm:"column:next_retry_at;index;comment:下次重试时间"`
	ErrorMessage   string                 `json:"error_message" gorm:"column:error_message;type:text;comment:错误信息"`
	ExtendedData   JSONMap                `json:"extended_data" gorm:"column:extended_data;type:json;comment:扩展数据"`
}

// TableName 指定工单通知队列表名
func (WorkorderNotificationQueue) TableName() string {
	return "cl_workorder_notification_queue"
}
