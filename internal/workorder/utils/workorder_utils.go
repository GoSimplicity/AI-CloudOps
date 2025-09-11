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

package utils

import "github.com/GoSimplicity/AI-CloudOps/internal/model"

// GetInstanceStatusName 获取状态名称
func GetInstanceStatusName(status int8) string {
	switch status {
	case model.InstanceStatusDraft:
		return "草稿"
	case model.InstanceStatusPending:
		return "待审批"
	case model.InstanceStatusProcessing:
		return "处理中"
	case model.InstanceStatusCompleted:
		return "已完成"
	case model.InstanceStatusRejected:
		return "已拒绝"
	case model.InstanceStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}

// GetEventTypeName 获取事件类型友好名称
func GetEventTypeName(eventType string) string {
	switch eventType {
	case model.EventTypeInstanceCreated:
		return "工单创建"
	case model.EventTypeInstanceSubmitted:
		return "工单提交"
	case model.EventTypeInstanceAssigned:
		return "工单指派"
	case model.EventTypeInstanceApproved:
		return "工单审批通过"
	case model.EventTypeInstanceRejected:
		return "工单拒绝"
	case model.EventTypeInstanceCompleted:
		return "工单完成"
	case model.EventTypeInstanceCancelled:
		return "工单取消"
	case model.EventTypeInstanceUpdated:
		return "工单更新"
	case model.EventTypeInstanceCommented:
		return "工单评论"
	default:
		return "未知事件"
	}
}

// GetNotificationChannelName 获取通知渠道友好名称
func GetNotificationChannelName(channel string) string {
	switch channel {
	case model.NotificationChannelEmail:
		return "邮件"
	case model.NotificationChannelFeishu:
		return "飞书"
	case model.NotificationChannelSMS:
		return "短信"
	case model.NotificationChannelWebhook:
		return "Webhook"
	default:
		return "未知渠道"
	}
}

// GetRecipientTypeName 获取接收者类型友好名称
func GetRecipientTypeName(recipientType string) string {
	switch recipientType {
	case model.RecipientTypeCreator:
		return "工单创建人"
	case model.RecipientTypeAssignee:
		return "工单处理人"
	case model.RecipientTypeUser:
		return "指定用户"
	case model.RecipientTypeRole:
		return "角色用户"
	case model.RecipientTypeDept:
		return "部门用户"
	case model.RecipientTypeCustom:
		return "自定义用户"
	default:
		return "未知类型"
	}
}

// GetAllEventTypes 获取所有事件类型
func GetAllEventTypes() []string {
	return []string{
		model.EventTypeInstanceCreated,
		model.EventTypeInstanceSubmitted,
		model.EventTypeInstanceAssigned,
		model.EventTypeInstanceApproved,
		model.EventTypeInstanceRejected,
		model.EventTypeInstanceCompleted,
		model.EventTypeInstanceCancelled,
		model.EventTypeInstanceUpdated,
		model.EventTypeInstanceCommented,
	}
}

// GetAllNotificationChannels 获取所有通知渠道
func GetAllNotificationChannels() []string {
	return []string{
		model.NotificationChannelEmail,
		model.NotificationChannelFeishu,
		model.NotificationChannelSMS,
		model.NotificationChannelWebhook,
	}
}

// GetAllRecipientTypes 获取所有接收者类型
func GetAllRecipientTypes() []string {
	return []string{
		model.RecipientTypeCreator,
		model.RecipientTypeAssignee,
		model.RecipientTypeUser,
		model.RecipientTypeRole,
		model.RecipientTypeDept,
		model.RecipientTypeCustom,
	}
}
