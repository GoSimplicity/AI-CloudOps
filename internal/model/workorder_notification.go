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

// 事件类型
const (
	EventTypeInstanceCreated   = "instance_created"   // 工单创建
	EventTypeInstanceSubmitted = "instance_submitted" // 工单提交
	EventTypeInstanceAssigned  = "instance_assigned"  // 工单指派
	EventTypeInstanceApproved  = "instance_approved"  // 工单审批通过
	EventTypeInstanceRejected  = "instance_rejected"  // 工单拒绝
	EventTypeInstanceCompleted = "instance_completed" // 工单完成
	EventTypeInstanceCancelled = "instance_cancelled" // 工单取消
	EventTypeInstanceUpdated   = "instance_updated"   // 工单更新
	EventTypeInstanceCommented = "instance_commented" // 工单评论
	EventTypeInstanceDeleted   = "instance_deleted"   // 工单删除
	EventTypeInstanceReturned  = "instance_returned"  // 工单退回
)

// 通知状态常量
const (
	NotificationStatusPending int8 = 1 // 待发送
	NotificationStatusSending int8 = 2 // 发送中
	NotificationStatusSuccess int8 = 3 // 发送成功
	NotificationStatusFailed  int8 = 4 // 发送失败
)

// 通知渠道
const (
	NotificationChannelEmail   = "email"   // 邮件通知
	NotificationChannelFeishu  = "feishu"  // 飞书通知
	NotificationChannelSMS     = "sms"     // 短信通知
	NotificationChannelWebhook = "webhook" // Webhook通知
)

// 接收人类型
const (
	RecipientTypeCreator  = "creator"  // 工单创建人
	RecipientTypeAssignee = "assignee" // 工单处理人
	RecipientTypeUser     = "user"     // 指定用户
	RecipientTypeRole     = "role"     // 角色用户
	RecipientTypeDept     = "dept"     // 部门用户
	RecipientTypeCustom   = "custom"   // 自定义用户
)

// 通知状态常量
const (
	NotificationStatusEnabled  int8 = 1 // 启用
	NotificationStatusDisabled int8 = 2 // 禁用
)

// 优先级
const (
	NotificationPriorityHigh   int8 = 1 // 高优先级
	NotificationPriorityMedium int8 = 2 // 中优先级
	NotificationPriorityLow    int8 = 3 // 低优先级
)

// 发送状态
const (
	NotificationSendStatusPending   int8 = 1 // 待发送
	NotificationSendStatusSending   int8 = 2 // 发送中
	NotificationSendStatusSuccess   int8 = 3 // 发送成功
	NotificationSendStatusFailed    int8 = 4 // 发送失败
	NotificationSendStatusCancelled int8 = 5 // 已取消
)

// 队列状态
const (
	NotificationQueueStatusPending    int8 = 1 // 待处理
	NotificationQueueStatusProcessing int8 = 2 // 处理中
	NotificationQueueStatusSuccess    int8 = 3 // 处理成功
	NotificationQueueStatusFailed     int8 = 4 // 处理失败
)

// 触发类型
const (
	TriggerTypeImmediate   = "immediate"   // 立即触发
	TriggerTypeDelayed     = "delayed"     // 延迟触发
	TriggerTypeScheduled   = "scheduled"   // 定时触发
	TriggerTypeConditional = "conditional" // 条件触发
)

// 任务类型
const (
	TaskTypeSendNotification        = "notification:send"
	TaskTypeBatchSendNotification   = "notification:batch_send"
	TaskTypeScheduledNotification   = "notification:scheduled"
	TaskTypeRetryFailedNotification = "notification:retry_failed"
)

// 流程动作
const (
	FlowActionUpdate  = "update"  // 更新
	FlowActionComment = "comment" // 评论
)

// 默认配置
const (
	IsDefaultYes int8 = 1 // 是
	IsDefaultNo  int8 = 2 // 否
)

// 工单通知配置
type WorkorderNotification struct {
	Model
	Name             string     `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:通知配置名称"`
	Description      string     `json:"description" gorm:"column:description;type:varchar(1000);comment:通知配置描述"`
	ProcessID        *int       `json:"process_id" gorm:"column:process_id;index;comment:关联流程ID"`
	TemplateID       *int       `json:"template_id" gorm:"column:template_id;index;comment:关联模板ID"`
	CategoryID       *int       `json:"category_id" gorm:"column:category_id;index;comment:关联分类ID"`
	EventTypes       StringList `json:"event_types" gorm:"column:event_types;type:text;not null;comment:触发事件类型"`
	TriggerType      string     `json:"trigger_type" gorm:"column:trigger_type;type:varchar(20);not null;default:'immediate';comment:触发类型"`
	TriggerCondition JSONMap    `json:"trigger_condition" gorm:"column:trigger_condition;type:json;comment:触发条件"`
	Channels         StringList `json:"channels" gorm:"column:channels;type:text;not null;comment:通知渠道"`
	RecipientTypes   StringList `json:"recipient_types" gorm:"column:recipient_types;type:text;not null;comment:接收人类型"`
	RecipientUsers   StringList `json:"recipient_users" gorm:"column:recipient_users;type:text;comment:自定义接收人用户ID"`
	RecipientRoles   StringList `json:"recipient_roles" gorm:"column:recipient_roles;type:text;comment:接收人角色ID"`
	RecipientDepts   StringList `json:"recipient_depts" gorm:"column:recipient_depts;type:text;comment:接收人部门ID"`
	MessageTemplate  string     `json:"message_template" gorm:"column:message_template;type:text;not null;comment:消息模板"`
	SubjectTemplate  string     `json:"subject_template" gorm:"column:subject_template;type:varchar(500);comment:主题模板"`
	ScheduledTime    *time.Time `json:"scheduled_time" gorm:"column:scheduled_time;comment:定时发送时间"`
	RepeatInterval   *int       `json:"repeat_interval" gorm:"column:repeat_interval;comment:重复间隔(分钟)"`
	MaxRetries       int        `json:"max_retries" gorm:"column:max_retries;not null;default:3;comment:最大重试次数"`
	RetryInterval    int        `json:"retry_interval" gorm:"column:retry_interval;not null;default:5;comment:重试间隔(分钟)"`
	Status           int8       `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-启用，2-禁用"`
	Priority         int8       `json:"priority" gorm:"column:priority;not null;default:2;comment:优先级：1-高，2-中，3-低"`
	OperatorID       int        `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	IsDefault        int8       `json:"is_default" gorm:"column:is_default;not null;default:2;comment:是否默认配置：1-是，2-否"`
	Settings         JSONMap    `json:"settings" gorm:"column:settings;type:json;comment:通知设置"`
}

func (WorkorderNotification) TableName() string {
	return "cl_workorder_notification"
}

type WorkorderNotificationChannel struct {
	Channels StringList `json:"channels"`
}

// CreateWorkorderNotificationReq 创建通知配置
type CreateWorkorderNotificationReq struct {
	Name             string     `json:"name" binding:"required"`
	Description      string     `json:"description"`
	ProcessID        *int       `json:"process_id"`
	TemplateID       *int       `json:"template_id"`
	CategoryID       *int       `json:"category_id"`
	EventTypes       StringList `json:"event_types" binding:"required"`
	TriggerType      string     `json:"trigger_type" binding:"required"`
	TriggerCondition JSONMap    `json:"trigger_condition"`
	Channels         StringList `json:"channels" binding:"required"`
	RecipientTypes   StringList `json:"recipient_types" binding:"required"`
	RecipientUsers   StringList `json:"recipient_users"`
	RecipientRoles   StringList `json:"recipient_roles"`
	RecipientDepts   StringList `json:"recipient_depts"`
	MessageTemplate  string     `json:"message_template" binding:"required"`
	SubjectTemplate  string     `json:"subject_template"`
	ScheduledTime    *time.Time `json:"scheduled_time"`
	RepeatInterval   *int       `json:"repeat_interval"`
	MaxRetries       int        `json:"max_retries"`
	RetryInterval    int        `json:"retry_interval"`
	Status           int8       `json:"status"`
	Priority         int8       `json:"priority"`
	IsDefault        int8       `json:"is_default" binding:"omitempty,oneof=1 2"`
	Settings         JSONMap    `json:"settings"`
	UserID           int        `json:"-"` // 由中间件注入
}

// UpdateWorkorderNotificationReq 更新通知配置
type UpdateWorkorderNotificationReq struct {
	ID               int        `json:"id" binding:"required"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	ProcessID        *int       `json:"process_id"`
	TemplateID       *int       `json:"template_id"`
	CategoryID       *int       `json:"category_id"`
	EventTypes       StringList `json:"event_types"`
	TriggerType      string     `json:"trigger_type"`
	TriggerCondition JSONMap    `json:"trigger_condition"`
	Channels         StringList `json:"channels"`
	RecipientTypes   StringList `json:"recipient_types"`
	RecipientUsers   StringList `json:"recipient_users"`
	RecipientRoles   StringList `json:"recipient_roles"`
	RecipientDepts   StringList `json:"recipient_depts"`
	MessageTemplate  string     `json:"message_template"`
	SubjectTemplate  string     `json:"subject_template"`
	ScheduledTime    *time.Time `json:"scheduled_time"`
	RepeatInterval   *int       `json:"repeat_interval"`
	MaxRetries       int        `json:"max_retries"`
	RetryInterval    int        `json:"retry_interval"`
	Status           int8       `json:"status"`
	Priority         int8       `json:"priority"`
	IsDefault        int8       `json:"is_default" binding:"omitempty,oneof=1 2"`
	Settings         JSONMap    `json:"settings"`
}

// DeleteWorkorderNotificationReq 删除通知配置
type DeleteWorkorderNotificationReq struct {
	ID int `json:"id" binding:"required"`
}

// ListWorkorderNotificationReq 通知配置列表
type ListWorkorderNotificationReq struct {
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"page_size" form:"page_size"`
	Name       string `json:"name" form:"name"`
	ProcessID  *int   `json:"process_id" form:"process_id"`
	TemplateID *int   `json:"template_id" form:"template_id"`
	CategoryID *int   `json:"category_id" form:"category_id"`
	Status     *int8  `json:"status" form:"status"`
	IsDefault  *int8  `json:"is_default" form:"is_default" binding:"omitempty,oneof=1 2"`
}

// DetailWorkorderNotificationReq 通知配置详情
type DetailWorkorderNotificationReq struct {
	ID int `json:"id" binding:"required"`
}

// 工单通知发送记录
type WorkorderNotificationLog struct {
	Model
	NotificationID int        `json:"notification_id" gorm:"not null;index;comment:通知配置ID"`
	InstanceID     *int       `json:"instance_id" gorm:"index;comment:工单实例ID"`
	EventType      string     `json:"event_type" gorm:"type:varchar(50);not null;index;comment:触发事件类型"`
	Channel        string     `json:"channel" gorm:"type:varchar(20);not null;index;comment:发送渠道"`
	RecipientType  string     `json:"recipient_type" gorm:"type:varchar(20);not null;comment:接收人类型"`
	RecipientID    string     `json:"recipient_id" gorm:"type:varchar(100);not null;index;comment:接收人ID"`
	RecipientName  string     `json:"recipient_name" gorm:"type:varchar(200);comment:接收人名称"`
	RecipientAddr  string     `json:"recipient_addr" gorm:"type:varchar(500);not null;comment:接收人地址"`
	Subject        string     `json:"subject" gorm:"type:varchar(500);comment:消息主题"`
	Content        string     `json:"content" gorm:"type:text;not null;comment:发送内容"`
	Status         int8       `json:"status" gorm:"not null;index;comment:发送状态：1-待发送，2-发送中，3-发送成功，4-发送失败"`
	ErrorMessage   string     `json:"error_message" gorm:"type:text;comment:错误信息"`
	ResponseData   JSONMap    `json:"response_data" gorm:"type:json;comment:响应数据"`
	SendAt         time.Time  `json:"send_at" gorm:"not null;comment:发送时间"`
	DeliveredAt    *time.Time `json:"delivered_at" gorm:"comment:送达时间"`
	ReadAt         *time.Time `json:"read_at" gorm:"comment:阅读时间"`
	Cost           *float64   `json:"cost" gorm:"comment:发送成本"`
	RetryCount     int        `json:"retry_count" gorm:"not null;default:0;comment:重试次数"`
	NextRetryAt    *time.Time `json:"next_retry_at" gorm:"comment:下次重试时间"`
	SenderID       int        `json:"sender_id" gorm:"not null;comment:发送人ID"`
	ExtendedData   JSONMap    `json:"extended_data" gorm:"type:json;comment:扩展数据"`
}

func (WorkorderNotificationLog) TableName() string {
	return "cl_workorder_notification_log"
}

// ListWorkorderNotificationLogReq 工单通知发送记录列表
type ListWorkorderNotificationLogReq struct {
	Page           int    `json:"page" form:"page"`
	PageSize       int    `json:"page_size" form:"page_size"`
	NotificationID *int   `json:"notification_id" form:"notification_id"`
	InstanceID     *int   `json:"instance_id" form:"instance_id"`
	EventType      string `json:"event_type" form:"event_type"`
	Channel        string `json:"channel" form:"channel"`
	RecipientType  string `json:"recipient_type" form:"recipient_type"`
	RecipientID    string `json:"recipient_id" form:"recipient_id"`
	Status         *int8  `json:"status" form:"status"`
}

// 工单通知队列
type WorkorderNotificationQueue struct {
	Model
	NotificationID int        `json:"notification_id" gorm:"not null;index;comment:通知配置ID"`
	InstanceID     *int       `json:"instance_id" gorm:"index;comment:工单实例ID"`
	EventType      string     `json:"event_type" gorm:"type:varchar(50);not null;index;comment:触发事件类型"`
	Channel        string     `json:"channel" gorm:"type:varchar(20);not null;comment:发送渠道"`
	RecipientType  string     `json:"recipient_type" gorm:"type:varchar(20);not null;comment:接收人类型"`
	RecipientID    string     `json:"recipient_id" gorm:"type:varchar(100);not null;comment:接收人ID"`
	RecipientAddr  string     `json:"recipient_addr" gorm:"type:varchar(500);not null;comment:接收人地址"`
	Subject        string     `json:"subject" gorm:"type:varchar(500);comment:消息主题"`
	Content        string     `json:"content" gorm:"type:text;not null;comment:发送内容"`
	Priority       int8       `json:"priority" gorm:"not null;default:2;index;comment:优先级：1-高，2-中，3-低"`
	Status         int8       `json:"status" gorm:"not null;default:1;index;comment:状态：1-待处理，2-处理中，3-处理成功，4-处理失败"`
	ScheduledAt    time.Time  `json:"scheduled_at" gorm:"not null;index;comment:计划发送时间"`
	ProcessedAt    *time.Time `json:"processed_at" gorm:"comment:处理时间"`
	RetryCount     int        `json:"retry_count" gorm:"not null;default:0;comment:重试次数"`
	NextRetryAt    *time.Time `json:"next_retry_at" gorm:"index;comment:下次重试时间"`
	ErrorMessage   string     `json:"error_message" gorm:"type:text;comment:错误信息"`
	ExtendedData   JSONMap    `json:"extended_data" gorm:"type:json;comment:扩展数据"`
}

func (WorkorderNotificationQueue) TableName() string {
	return "cl_workorder_notification_queue"
}

// TestSendWorkorderNotificationReq 测试发送工单通知
type TestSendWorkorderNotificationReq struct {
	NotificationID int    `json:"notification_id" binding:"required"`
	Recipient      string `json:"recipient"` // 可选，如果不提供则使用默认测试地址
}

// ListWorkorderNotificationQueueReq 工单通知队列列表
type ListWorkorderNotificationQueueReq struct {
	Page           int    `json:"page" form:"page"`
	PageSize       int    `json:"page_size" form:"page_size"`
	NotificationID *int   `json:"notification_id" form:"notification_id"`
	InstanceID     *int   `json:"instance_id" form:"instance_id"`
	EventType      string `json:"event_type" form:"event_type"`
	Channel        string `json:"channel" form:"channel"`
	RecipientType  string `json:"recipient_type" form:"recipient_type"`
	RecipientID    string `json:"recipient_id" form:"recipient_id"`
	Status         *int8  `json:"status" form:"status"`
	Priority       *int8  `json:"priority" form:"priority"`
}

// ManualSendNotificationReq 手动发送通知请求
type ManualSendNotificationReq struct {
	Channels  []string `json:"channels" binding:"required"`  // 通知渠道列表
	Recipient string   `json:"recipient" binding:"required"` // 接收人地址
	Subject   string   `json:"subject" binding:"required"`   // 通知主题
	Content   string   `json:"content" binding:"required"`   // 通知内容
}
