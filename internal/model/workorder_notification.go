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
	Status           int8       `json:"status" gorm:"column:status;not null;default:1;index;comment:状态"`
	Priority         int8       `json:"priority" gorm:"column:priority;not null;default:2;comment:优先级"`
	OperatorID       int        `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	IsDefault        bool       `json:"is_default" gorm:"column:is_default;not null;default:false;comment:是否默认配置"`
	Settings         JSONMap    `json:"settings" gorm:"column:settings;type:json;comment:通知设置"`
}

func (WorkorderNotification) TableName() string {
	return "cl_workorder_notification"
}

// CreateWorkorderNotificationReq 创建工单通知配置
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
	IsDefault        bool       `json:"is_default"`
	Settings         JSONMap    `json:"settings"`
	UserID           int        `json:"-"` // 由中间件注入
}

// UpdateWorkorderNotificationReq 更新工单通知配置
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
	IsDefault        bool       `json:"is_default"`
	Settings         JSONMap    `json:"settings"`
}

// DeleteWorkorderNotificationReq 删除工单通知配置
type DeleteWorkorderNotificationReq struct {
	ID int `json:"id" binding:"required"`
}

// ListWorkorderNotificationReq 工单通知配置列表
type ListWorkorderNotificationReq struct {
	Page        int    `json:"page" form:"page"`
	PageSize    int    `json:"page_size" form:"page_size"`
	Name        string `json:"name" form:"name"`
	ProcessID   *int   `json:"process_id" form:"process_id"`
	TemplateID  *int   `json:"template_id" form:"template_id"`
	CategoryID  *int   `json:"category_id" form:"category_id"`
	Status      *int8  `json:"status" form:"status"`
	IsDefault   *bool  `json:"is_default" form:"is_default"`
}

// DetailWorkorderNotificationReq 工单通知配置详情
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
	Status         int8       `json:"status" gorm:"not null;index;comment:发送状态"`
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
	Priority       int8       `json:"priority" gorm:"not null;default:2;index;comment:优先级"`
	Status         int8       `json:"status" gorm:"not null;default:1;index;comment:状态"`
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
	Recipient      string `json:"recipient" binding:"required"`
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
