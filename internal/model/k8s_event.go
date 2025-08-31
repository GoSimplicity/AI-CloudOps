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

// EventType 事件类型
type EventType int8

const (
	EventTypeNormal EventType = iota + 1
	EventTypeWarning
)

// EventReason 事件原因
type EventReason int8

const (
	EventReasonBackOff EventReason = iota + 1
	EventReasonPulled
	EventReasonCreated
	EventReasonDeleted
	EventReasonUpdated
	EventReasonRestarted
	EventReasonStarted
	EventReasonStopped
	EventReasonFailed
	EventReasonSucceeded
	EventReasonUnknown
	EventReasonWarning
	EventReasonError
	EventReasonFatal
	EventReasonPanic
	EventReasonTimeout
	EventReasonCancelled
	EventReasonInterrupted
	EventReasonAborted
	EventReasonIgnored
	EventReasonOther
)

// EventSeverity 事件严重程度
type EventSeverity int8

const (
	EventSeverityLow EventSeverity = iota + 1
	EventSeverityMedium
	EventSeverityHigh
	EventSeverityCritical
)

// K8sEvent k8s事件
type K8sEvent struct {
	Name               string            `json:"name"`                          // 名称
	Namespace          string            `json:"namespace"`                     // 命名空间
	UID                string            `json:"uid"`                           // UID
	ClusterID          int               `json:"cluster_id"`                    // 集群ID
	Type               EventType         `json:"type"`                          // Normal, Warning
	Reason             EventReason       `json:"reason"`                        // 事件原因，如：BackOff, Pulled, Created
	Message            string            `json:"message"`                       // 详细消息
	Severity           EventSeverity     `json:"severity"`                      // 严重程度：low, medium, high, critical
	FirstTimestamp     time.Time         `json:"first_timestamp"`               // 首次发生时间
	LastTimestamp      time.Time         `json:"last_timestamp"`                // 最后发生时间
	Count              int64             `json:"count"`                         // 事件发生次数
	InvolvedObject     InvolvedObject    `json:"involved_object"`               // 涉及对象
	Source             EventSource       `json:"source"`                        // 事件源
	Action             string            `json:"action,omitempty"`              // 执行的动作
	ReportingComponent string            `json:"reporting_component,omitempty"` // 报告组件
	ReportingInstance  string            `json:"reporting_instance,omitempty"`  // 报告实例
	Labels             map[string]string `json:"labels,omitempty"`              // 标签
	Annotations        map[string]string `json:"annotations,omitempty"`         // 注解
}

// InvolvedObject 事件涉及的K8s对象
type InvolvedObject struct {
	Kind       string `json:"kind"`                 // Pod, Service, Deployment等
	Name       string `json:"name"`                 // 对象名称
	Namespace  string `json:"namespace"`            // 命名空间
	UID        string `json:"uid"`                  // 对象UID
	APIVersion string `json:"api_version"`          // API版本
	FieldPath  string `json:"field_path,omitempty"` // 如：spec.containers{nginx}
}

// EventSource 事件源信息
type EventSource struct {
	Component string `json:"component"` // kubelet, controller-manager等
	Host      string `json:"host"`      // 节点名称
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"` // 开始时间
	End   time.Time `json:"end"`   // 结束时间
}

// CountItem 计数项
type CountItem struct {
	Name       string  `json:"name"`       // 名称
	Count      int64   `json:"count"`      // 计数
	Percentage float64 `json:"percentage"` // 百分比
}

// EventSummary 事件汇总
type EventSummary struct {
	TotalEvents   int64            `json:"total_events"`   // 总事件数
	UniqueEvents  int64            `json:"unique_events"`  // 唯一事件数
	WarningEvents int64            `json:"warning_events"` // 警告事件数
	NormalEvents  int64            `json:"normal_events"`  // 正常事件数
	Distribution  map[string]int64 `json:"distribution"`   // 按severity分布
	TopReasons    []CountItem      `json:"top_reasons"`    // 热门原因
	TopObjects    []CountItem      `json:"top_objects"`    // 热门对象
}

// EventGroupData 分组数据
type EventGroupData struct {
	Group  string     `json:"group"`            // 分组名称
	Count  int64      `json:"count"`            // 计数
	Events []K8sEvent `json:"events,omitempty"` // 可选：包含该组的事件样本
}

// EventTrend 事件趋势
type EventTrend struct {
	Timestamp time.Time `json:"timestamp"`      // 时间戳
	Count     int64     `json:"count"`          // 计数
	Type      string    `json:"type,omitempty"` // 类型
}

// EventStatistics 事件统计
type EventStatistics struct {
	TimeRange TimeRange        `json:"time_range"`       // 时间范围
	Summary   EventSummary     `json:"summary"`          // 汇总信息
	GroupData []EventGroupData `json:"group_data"`       // 分组数据
	Trends    []EventTrend     `json:"trends,omitempty"` // 趋势数据
}

// EventTimelineItem 时间线项
type EventTimelineItem struct {
	Timestamp time.Time `json:"timestamp"` // 时间戳
	Type      string    `json:"type"`      // 类型
	Reason    string    `json:"reason"`    // 原因
	Message   string    `json:"message"`   // 消息
	Count     int64     `json:"count"`     // 计数
}

// EventTimeline 事件时间线
type EventTimeline struct {
	Object   InvolvedObject      `json:"object"`   // 涉及对象
	Timeline []EventTimelineItem `json:"timeline"` // 时间线
}

// EventListReq 事件列表查询请求 - 通用查询参数
type EventListReq struct {
	ListReq
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" form:"namespace"`
}

// EventStatisticsReq 事件统计请求
type EventStatisticsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace"`
}

// EventCleanupReq 事件清理请求
type EventCleanupReq struct {
	ClusterID  int        `json:"cluster_id" binding:"required"`
	Namespace  string     `json:"namespace"`                             // 空则清理所有namespace
	BeforeTime *time.Time `json:"before_time"`                           // 清理此时间之前的事件
	DaysToKeep int        `json:"days_to_keep"`                          // 保留最近N天，与before_time二选一
	DryRun     int8       `json:"dry_run" binding:"required,oneof= 1 2"` // 模拟运行，不实际删除
}
