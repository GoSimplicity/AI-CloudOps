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
)

// serializeRequest 序列化发送请求
func serializeRequest(request *SendRequest) []byte {
	data, _ := json.Marshal(request)
	return data
}

// deserializeRequest 反序列化发送请求
func deserializeRequest(data []byte) (*SendRequest, error) {
	var request SendRequest
	err := json.Unmarshal(data, &request)
	return &request, err
}

// isValidEmail 验证邮箱地址格式
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// 简单的邮箱验证正则
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

// isValidFeishuID 验证飞书ID格式
func isValidFeishuID(id string) bool {
	if id == "" {
		return false
	}

	// 飞书用户ID通常以 "ou_" 开头，群组ID以 "oc_" 开头
	return strings.HasPrefix(id, "ou_") || strings.HasPrefix(id, "oc_") || strings.HasPrefix(id, "cli_")
}

// FormatPriority 格式化优先级
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

// FormatPriorityIcon 格式化优先级图标
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

// GetEventTypeText 获取事件类型文本
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

// SanitizeContent 清理内容，防止注入
func SanitizeContent(content string) string {
	// 移除或转义潜在的危险字符
	content = strings.ReplaceAll(content, "<script", "&lt;script")
	content = strings.ReplaceAll(content, "</script>", "&lt;/script&gt;")
	content = strings.ReplaceAll(content, "javascript:", "")
	content = strings.ReplaceAll(content, "vbscript:", "")
	content = strings.ReplaceAll(content, "onload=", "")
	content = strings.ReplaceAll(content, "onerror=", "")

	return content
}

// TruncateText 截断文本
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	if maxLength <= 3 {
		return "..."
	}

	return text[:maxLength-3] + "..."
}

// ExtractMentions 提取@提及
func ExtractMentions(content string) []string {
	// 匹配 @username 格式
	re := regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	var mentions []string
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}

	return mentions
}

// IsURL 检查是否为URL
func IsURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

// GenerateCallbackURL 生成回调URL
func GenerateCallbackURL(baseURL string, instanceID int, action string) string {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL + "workorder/" + action + "?id=" + string(rune(instanceID))
}

// MaskSensitiveData 掩码敏感数据
func MaskSensitiveData(data string) string {
	if len(data) <= 4 {
		return "****"
	}

	// 保留前2位和后2位
	return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
}

// ValidateRecipientFormat 验证接收人格式
func ValidateRecipientFormat(recipientType, recipientAddr string) error {
	switch recipientType {
	case "email", "user_email":
		if !isValidEmail(recipientAddr) {
			return fmt.Errorf("invalid email format: %s", recipientAddr)
		}
	case "feishu", "feishu_user", "feishu_group":
		if !isValidFeishuID(recipientAddr) {
			return fmt.Errorf("invalid feishu ID format: %s", recipientAddr)
		}
	}
	return nil
}

// BuildNotificationContext 构建通知上下文
func BuildNotificationContext(request *SendRequest) map[string]interface{} {
	context := map[string]interface{}{
		"message_id":     request.MessageID,
		"recipient_type": request.RecipientType,
		"recipient_id":   request.RecipientID,
		"recipient_name": request.RecipientName,
		"event_type":     request.EventType,
		"priority":       request.Priority,
		"priority_text":  FormatPriority(request.Priority),
		"priority_icon":  FormatPriorityIcon(request.Priority),
		"event_text":     GetEventTypeText(request.EventType),
	}

	if request.InstanceID != nil {
		context["instance_id"] = *request.InstanceID
	}

	// 合并元数据
	for key, value := range request.Metadata {
		context[key] = value
	}

	return context
}
