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

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
)

// WarningReasons 警告级别的事件原因
var WarningReasons = map[string]model.EventSeverity{
	"Failed":              model.EventSeverityHigh,
	"FailedMount":         model.EventSeverityHigh,
	"FailedScheduling":    model.EventSeverityHigh,
	"FailedSync":          model.EventSeverityMedium,
	"FailedValidation":    model.EventSeverityMedium,
	"Unhealthy":           model.EventSeverityHigh,
	"BackOff":             model.EventSeverityMedium,
	"DeadlineExceeded":    model.EventSeverityCritical,
	"Evicted":             model.EventSeverityHigh,
	"NodeNotReady":        model.EventSeverityCritical,
	"NodeNotSchedulable":  model.EventSeverityMedium,
	"OutOfDisk":           model.EventSeverityCritical,
	"OutOfMemory":         model.EventSeverityCritical,
	"FreeDiskSpaceFailed": model.EventSeverityHigh,
}

// NormalReasons 正常级别的事件原因
var NormalReasons = map[string]model.EventSeverity{
	"Scheduled":        model.EventSeverityLow,
	"Pulling":          model.EventSeverityLow,
	"Pulled":           model.EventSeverityLow,
	"Created":          model.EventSeverityLow,
	"Started":          model.EventSeverityLow,
	"Killing":          model.EventSeverityLow,
	"Preempting":       model.EventSeverityLow,
	"SuccessfulMount":  model.EventSeverityLow,
	"SuccessfulDelete": model.EventSeverityLow,
	"Sync":             model.EventSeverityLow,
}

// GetEventSeverity 根据事件类型和原因判断事件严重程度
func GetEventSeverity(eventType, reason string) model.EventSeverity {
	if eventType == "Warning" {
		if severity, exists := WarningReasons[reason]; exists {
			return severity
		}
		return model.EventSeverityMedium // 默认警告级别
	}

	if eventType == "Normal" {
		if severity, exists := NormalReasons[reason]; exists {
			return severity
		}
		return model.EventSeverityLow // 默认正常级别
	}

	return model.EventSeverityLow
}

// IsEventCritical 判断事件是否为关键事件
func IsEventCritical(eventType, reason string) bool {
	return GetEventSeverity(eventType, reason) == model.EventSeverityCritical
}

// FormatEventAge 格式化事件年龄显示
func FormatEventAge(timestamp time.Time) string {
	duration := time.Since(timestamp)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}

// FilterEventsByTimeRange 根据时间范围过滤事件
func FilterEventsByTimeRange(events []corev1.Event, startTime, endTime time.Time) []corev1.Event {
	var filteredEvents []corev1.Event

	for _, event := range events {
		eventTime := event.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = event.FirstTimestamp.Time
		}

		if !startTime.IsZero() && eventTime.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && eventTime.After(endTime) {
			continue
		}

		filteredEvents = append(filteredEvents, event)
	}

	return filteredEvents
}

// FilterEventsByType 根据事件类型过滤事件
func FilterEventsByType(events []corev1.Event, eventType string) []corev1.Event {
	if eventType == "" {
		return events
	}

	var filteredEvents []corev1.Event
	for _, event := range events {
		if event.Type == eventType {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents
}

// FilterEventsByReason 根据事件原因过滤事件
func FilterEventsByReason(events []corev1.Event, reason string) []corev1.Event {
	if reason == "" {
		return events
	}

	var filteredEvents []corev1.Event
	for _, event := range events {
		if event.Reason == reason {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents
}

// FilterEventsByObject 根据涉及对象过滤事件
func FilterEventsByObject(events []corev1.Event, objectKind, objectName string) []corev1.Event {
	var filteredEvents []corev1.Event

	for _, event := range events {
		if objectKind != "" && event.InvolvedObject.Kind != objectKind {
			continue
		}
		if objectName != "" && event.InvolvedObject.Name != objectName {
			continue
		}

		filteredEvents = append(filteredEvents, event)
	}

	return filteredEvents
}

// GroupEventsByType 按事件类型分组
func GroupEventsByType(events []corev1.Event) map[string][]corev1.Event {
	groups := make(map[string][]corev1.Event)

	for _, event := range events {
		groups[event.Type] = append(groups[event.Type], event)
	}

	return groups
}

// GroupEventsByReason 按事件原因分组
func GroupEventsByReason(events []corev1.Event) map[string][]corev1.Event {
	groups := make(map[string][]corev1.Event)

	for _, event := range events {
		groups[event.Reason] = append(groups[event.Reason], event)
	}

	return groups
}

// GroupEventsByObject 按涉及对象分组
func GroupEventsByObject(events []corev1.Event) map[string][]corev1.Event {
	groups := make(map[string][]corev1.Event)

	for _, event := range events {
		key := fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)
		groups[key] = append(groups[key], event)
	}

	return groups
}

// ConvertEventToK8sEvent 将 corev1.Event 转换为 model.K8sEvent
func ConvertEventToK8sEvent(event corev1.Event) model.K8sEvent {
	return model.K8sEvent{
		Name:           event.Name,
		Namespace:      event.Namespace,
		UID:            string(event.UID),
		Message:        event.Message,
		FirstTimestamp: event.FirstTimestamp.Time,
		LastTimestamp:  event.LastTimestamp.Time,
		Count:          int64(event.Count),
		InvolvedObject: model.InvolvedObject{
			Kind:       event.InvolvedObject.Kind,
			Name:       event.InvolvedObject.Name,
			Namespace:  event.InvolvedObject.Namespace,
			UID:        string(event.InvolvedObject.UID),
			APIVersion: event.InvolvedObject.APIVersion,
			FieldPath:  event.InvolvedObject.FieldPath,
		},
		Source: model.EventSource{
			Component: event.Source.Component,
			Host:      event.Source.Host,
		},
	}
}

// GetEventSummary 获取事件摘要信息
func GetEventSummary(events []corev1.Event) map[string]interface{} {
	summary := make(map[string]interface{})

	totalEvents := len(events)
	normalEvents := 0
	warningEvents := 0
	criticalEvents := 0

	reasonCounts := make(map[string]int)
	objectCounts := make(map[string]int)

	for _, event := range events {
		// 统计事件类型
		if event.Type == "Normal" {
			normalEvents++
		} else if event.Type == "Warning" {
			warningEvents++
		}

		// 统计关键事件
		if IsEventCritical(event.Type, event.Reason) {
			criticalEvents++
		}

		// 统计原因
		reasonCounts[event.Reason]++

		// 统计对象
		objectKey := fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)
		objectCounts[objectKey]++
	}

	summary["total_events"] = totalEvents
	summary["normal_events"] = normalEvents
	summary["warning_events"] = warningEvents
	summary["critical_events"] = criticalEvents
	summary["reason_counts"] = reasonCounts
	summary["object_counts"] = objectCounts

	return summary
}

// ParseEventMessage 解析事件消息，提取关键信息
func ParseEventMessage(message string) map[string]string {
	info := make(map[string]string)
	info["message"] = message

	// 提取常见的错误信息
	if strings.Contains(message, "ErrImagePull") {
		info["error_type"] = "ImagePull"
		info["severity"] = "High"
	} else if strings.Contains(message, "ErrImageNeverPull") {
		info["error_type"] = "ImageNeverPull"
		info["severity"] = "High"
	} else if strings.Contains(message, "InvalidImageName") {
		info["error_type"] = "InvalidImageName"
		info["severity"] = "High"
	} else if strings.Contains(message, "CrashLoopBackOff") {
		info["error_type"] = "CrashLoopBackOff"
		info["severity"] = "Critical"
	} else if strings.Contains(message, "OOMKilled") {
		info["error_type"] = "OutOfMemory"
		info["severity"] = "Critical"
	} else if strings.Contains(message, "Insufficient") {
		info["error_type"] = "InsufficientResources"
		info["severity"] = "High"
	}

	return info
}

// SortEventsByTime 按时间排序事件（最新的在前）
func SortEventsByTime(events []corev1.Event) []corev1.Event {
	// 使用副本避免修改原切片
	sortedEvents := make([]corev1.Event, len(events))
	copy(sortedEvents, events)

	for i := 0; i < len(sortedEvents)-1; i++ {
		for j := i + 1; j < len(sortedEvents); j++ {
			timeI := sortedEvents[i].LastTimestamp.Time
			if timeI.IsZero() {
				timeI = sortedEvents[i].FirstTimestamp.Time
			}

			timeJ := sortedEvents[j].LastTimestamp.Time
			if timeJ.IsZero() {
				timeJ = sortedEvents[j].FirstTimestamp.Time
			}

			if timeI.Before(timeJ) {
				sortedEvents[i], sortedEvents[j] = sortedEvents[j], sortedEvents[i]
			}
		}
	}

	return sortedEvents
}
