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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// serializeRequest 将发送请求序列化为JSON
func serializeRequest(request *SendRequest) []byte {
	data, _ := json.Marshal(request)
	return data
}

// deserializeRequest 将JSON反序列化为发送请求
func deserializeRequest(data []byte) (*SendRequest, error) {
	var request SendRequest
	err := json.Unmarshal(data, &request)
	return &request, err
}

// isValidEmail 检查邮箱地址格式是否有效
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// 简单的邮箱验证正则
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

// FormatPriority 将优先级数字转为文本表示
func FormatPriority(priority int8) string {
	switch priority {
	case 1:
		return "高"
	case 2:
		return "中"
	case 3:
		return "低"
	default:
		return "中"
	}
}

// FormatPriorityIcon 获取优先级对应的图标
func FormatPriorityIcon(priority int8) string {
	switch priority {
	case 1:
		return "🔴"
	case 2:
		return "🟡"
	case 3:
		return "🟢"
	default:
		return "🟡"
	}
}

// GetEventTypeText 获取事件类型的文本描述
func GetEventTypeText(eventType string) string {
	eventMap := map[string]string{
		"created":   "工单创建",
		"updated":   "工单更新",
		"approved":  "工单审批通过",
		"rejected":  "工单审批拒绝",
		"completed": "工单完成",
		"closed":    "工单关闭",
		"cancelled": "工单取消",
		"assigned":  "工单分配",
		"commented": "工单评论",
		"escalated": "工单升级",
		"due_soon":  "工单即将到期",
		"overdue":   "工单已逾期",
	}

	if text, exists := eventMap[eventType]; exists {
		return text
	}
	return eventType
}

// GetEventTypeIcon 获取事件类型对应的图标
func GetEventTypeIcon(eventType string) string {
	eventIcons := map[string]string{
		"created":   "📝",
		"updated":   "🔄",
		"approved":  "✅",
		"rejected":  "❌",
		"completed": "🎉",
		"closed":    "🔒",
		"cancelled": "🚫",
		"assigned":  "👤",
		"commented": "💬",
		"escalated": "⚡",
		"due_soon":  "⏰",
		"overdue":   "🚨",
		"test":      "🧪",
	}

	// 兼容中文事件类型
	chineseEventIcons := map[string]string{
		"工单创建": "📝",
		"工单提交": "📤",
		"工单指派": "👤",
		"工单审批": "✅",
		"工单拒绝": "❌",
		"工单完成": "🎉",
		"工单关闭": "🔒",
	}

	if icon, exists := eventIcons[eventType]; exists {
		return icon
	}
	if icon, exists := chineseEventIcons[eventType]; exists {
		return icon
	}
	return "📋" // 默认图标
}

// RenderTemplate 渲染模板内容
func RenderTemplate(content string, request *SendRequest) (string, error) {
	if content == "" {
		return content, nil
	}

	// 构建模板变量映射
	variables := buildTemplateVariables(request)

	// 使用字符串替换方式，支持多种格式的模板变量
	return replaceTemplateVariables(content, variables), nil
}

// buildTemplateVariables 构建模板变量映射
func buildTemplateVariables(request *SendRequest) map[string]string {
	variables := make(map[string]string)

	// ===== 核心业务变量 =====
	variables["subject"] = safeString(request.Subject)
	variables["content"] = safeString(request.Content)
	variables["recipient_name"] = safeString(request.RecipientName)
	variables["recipient_addr"] = safeString(request.RecipientAddr)

	// ===== 工单业务变量 =====
	if request.InstanceID != nil {
		variables["workorder_id"] = fmt.Sprintf("%d", *request.InstanceID)
		variables["serial_number"] = fmt.Sprintf("WO-%d", *request.InstanceID)
	} else {
		variables["workorder_id"] = ""
		variables["serial_number"] = "系统通知"
	}

	// ===== 优先级相关变量 =====
	variables["priority_level"] = fmt.Sprintf("%d", int(request.Priority))
	variables["priority_text"] = FormatPriority(request.Priority)
	variables["priority_icon"] = FormatPriorityIcon(request.Priority)

	// ===== 事件类型变量 =====
	variables["event_type"] = GetEventTypeText(request.EventType)
	variables["event_type_text"] = GetEventTypeText(request.EventType)
	variables["event_type_icon"] = GetEventTypeIcon(request.EventType)

	// ===== 时间相关变量 =====
	currentTime := time.Now()
	variables["notification_time"] = currentTime.Format("2006-01-02 15:04:05")
	variables["notification_date"] = currentTime.Format("2006-01-02")
	variables["notification_year"] = currentTime.Format("2006")
	variables["notification_month"] = currentTime.Format("01")
	variables["notification_day"] = currentTime.Format("02")

	// ===== 企业信息变量 =====
	variables["company_name"] = "AI-CloudOps"
	variables["platform_name"] = "智能运维管理平台"
	variables["department"] = "技术运维部"
	variables["service_hotline"] = "400-000-0000"
	variables["copyright"] = "Copyright © 2024 AI-CloudOps. All rights reserved."

	// ===== 从Templates中获取业务变量 =====
	if request.Templates != nil {
		for key, value := range request.Templates {
			variables[key] = value
		}
	}

	// ===== 从元数据中获取扩展变量 =====
	if request.Metadata != nil {
		for key, value := range request.Metadata {
			if str, ok := value.(string); ok {
				variables[key] = str
			} else {
				variables[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	return variables
}

// replaceTemplateVariables 在模板中替换变量占位符
func replaceTemplateVariables(template string, variables map[string]string) string {
	if template == "" {
		return ""
	}

	result := template

	// 支持多种格式的模板变量替换
	for key, value := range variables {
		// 替换 ${变量名} 格式（需要最先替换，避免与其他格式冲突）
		result = strings.ReplaceAll(result, "${"+key+"}", value)
		// 替换 {{变量名}} 格式
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
		// 替换 {{ 变量名 }} 格式（带空格）
		result = strings.ReplaceAll(result, "{{ "+key+" }}", value)
		// 替换 {变量名} 格式
		result = strings.ReplaceAll(result, "{"+key+"}", value)
		// 替换 { 变量名 } 格式（带空格）
		result = strings.ReplaceAll(result, "{ "+key+" }", value)
	}

	return result
}

// safeString 安全处理字符串
func safeString(s string) string {
	if s == "" {
		return ""
	}
	return s
}
