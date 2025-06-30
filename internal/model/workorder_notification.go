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

package model

import (
	"time"
)

// 通知状态常量
const (
	NotificationStatusDisabled int8 = 0 // 禁用
	NotificationStatusEnabled  int8 = 1 // 启用
)

// 通知渠道类型
const (
	NotificationChannelFeishu   = "feishu"   // 飞书
	NotificationChannelEmail    = "email"    // 邮箱
	NotificationChannelDingtalk = "dingtalk" // 钉钉
	NotificationChannelWechat   = "wechat"   // 企业微信
)

// 触发类型
const (
	NotificationTriggerManual    = "manual"    // 手动发送
	NotificationTriggerImmediate = "immediate" // 表单发布后立即发送
	NotificationTriggerScheduled = "scheduled" // 定时发送
)

// Notification 通知配置模型
type Notification struct {
	Model
	FormID          int        `json:"form_id" gorm:"column:form_id;not null;index;comment:关联表单ID"`
	Channels        StringList `json:"channels" gorm:"column:channels;not null;comment:通知渠道"`
	Recipients      StringList `json:"recipients" gorm:"column:recipients;not null;comment:接收人"`
	MessageTemplate string     `json:"message_template" gorm:"column:message_template;type:text;not null;comment:消息模板"`
	TriggerType     string     `json:"trigger_type" gorm:"column:trigger_type;not null;default:manual;comment:触发类型：manual-手动,immediate-立即,scheduled-定时"`
	ScheduledTime   *time.Time `json:"scheduled_time" gorm:"column:scheduled_time;comment:定时发送时间"`
	Status          int8       `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用,1-启用"`
	SentCount       int        `json:"sent_count" gorm:"column:sent_count;not null;default:0;comment:已发送次数"`
	LastSent        *time.Time `json:"last_sent" gorm:"column:last_sent;comment:最后发送时间"`
	FormUrl         string     `json:"form_url" gorm:"column:form_url;comment:表单链接"`
	CreatorID       int        `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`

	CreatorName string `json:"creator_name" gorm:"-"`
	FormName    string `json:"form_name" gorm:"-"`
}

// TableName 指定通知配置表名
func (Notification) TableName() string {
	return "workorder_notification"
}

// NotificationLog 通知发送记录
type NotificationLog struct {
	Model
	NotificationID int    `json:"notification_id" gorm:"column:notification_id;not null;index;comment:通知配置ID"`
	Channel        string `json:"channel" gorm:"column:channel;not null;comment:发送渠道"`
	Recipient      string `json:"recipient" gorm:"column:recipient;not null;comment:接收人"`
	Status         string `json:"status" gorm:"column:status;not null;comment:发送状态：success-成功,failed-失败"`
	Error          string `json:"error" gorm:"column:error;type:text;comment:错误信息"`
	Content        string `json:"content" gorm:"column:content;type:text;comment:发送内容"`
	SenderID       int    `json:"sender_id" gorm:"column:sender_id;not null;comment:发送人ID"`
	SenderName     string `json:"sender_name" gorm:"-"`
}

// TableName 指定通知发送记录表名
func (NotificationLog) TableName() string {
	return "workorder_notification_log"
}

// CreateNotificationReq 创建通知配置请求
type CreateNotificationReq struct {
	FormID          int        `json:"form_id" binding:"required"`                                       // 关联表单ID
	UserID          int        `json:"user_id" binding:"required"`                                       // 用户ID
	Channels        []string   `json:"channels" binding:"required,min=1"`                                // 通知渠道
	Recipients      []string   `json:"recipients" binding:"required,min=1"`                              // 接收人
	MessageTemplate string     `json:"message_template" binding:"required"`                              // 消息模板
	TriggerType     string     `json:"trigger_type" binding:"required,oneof=manual immediate scheduled"` // 触发类型
	ScheduledTime   *time.Time `json:"scheduled_time"`                                                   // 定时发送时间
	FormUrl         string     `json:"form_url"`                                                         // 表单链接
}

// UpdateNotificationReq 更新通知配置请求
type UpdateNotificationReq struct {
	ID              int        `json:"id" binding:"required"`                                            // 通知配置ID
	FormID          int        `json:"form_id" binding:"required"`                                       // 关联表单ID
	Channels        []string   `json:"channels" binding:"required,min=1"`                                // 通知渠道
	Recipients      []string   `json:"recipients" binding:"required,min=1"`                              // 接收人
	MessageTemplate string     `json:"message_template" binding:"required"`                              // 消息模板
	TriggerType     string     `json:"trigger_type" binding:"required,oneof=manual immediate scheduled"` // 触发类型
	ScheduledTime   *time.Time `json:"scheduled_time"`                                                   // 定时发送时间
	Status          int8       `json:"status" binding:"omitempty,oneof=0 1"`                             // 状态
	FormUrl         string     `json:"form_url"`                                                         // 表单链接
}

type DeleteNotificationReq struct {
	ID int `json:"id" binding:"required"` // 通知配置ID
}

// ListNotificationReq 查询通知配置列表请求
type ListNotificationReq struct {
	ListReq
	Channel *string `json:"channel" form:"channel"` // 通知渠道
	Status  *int8   `json:"status" form:"status"`   // 状态
	FormID  *int    `json:"form_id" form:"form_id"` // 表单ID
}

// DetailNotificationReq 获取通知配置详情请求
type DetailNotificationReq struct {
	ID int `json:"id" form:"id" binding:"required"` // 通知配置ID
}

// UpdateStatusReq 更新通知配置状态请求
type UpdateStatusReq struct {
	ID     int  `json:"id" binding:"required"`               // 通知配置ID
	Status int8 `json:"status" binding:"required,oneof=0 1"` // 状态
}

// TestSendNotificationReq 测试发送通知请求
type TestSendNotificationReq struct {
	NotificationID int `json:"notification_id" binding:"required"` // 通知配置ID
}

// DuplicateNotificationReq 复制通知配置请求
type DuplicateNotificationReq struct {
	SourceID int  `json:"source_id" binding:"required"` // 源通知配置ID
	Rename   bool `json:"rename"`                       // 是否重命名
}

// ListSendLogReq 查询发送记录请求
type ListSendLogReq struct {
	ListReq
	NotificationID int     `json:"notification_id" form:"notification_id" binding:"required"` // 通知配置ID
	Channel        *string `json:"channel" form:"channel"`                                    // 通知渠道
	Status         *string `json:"status" form:"status"`                                      // 发送状态
}

// NotificationStats 通知统计数据
type NotificationStats struct {
	Enabled   int `json:"enabled"`    // 启用状态数量
	Disabled  int `json:"disabled"`   // 禁用状态数量
	TodaySent int `json:"today_sent"` // 今日发送数量
}
