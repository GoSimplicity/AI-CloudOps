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
	EventRestarted
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
type EventSeverity string

const (
	EventSeverityLow      EventSeverity = "Low"
	EventSeverityMedium   EventSeverity = "Medium"
	EventSeverityHigh     EventSeverity = "High"
	EventSeverityCritical EventSeverity = "Critical"
)

// K8sEvent k8s事件
type K8sEvent struct {
	Name               string            `json:"name"`                          // 名称
	Namespace          string            `json:"namespace"`                     // 命名空间
	UID                string            `json:"uid"`                           // UID
	ClusterID          int               `json:"cluster_id"`                    // 集群ID
	Type               string            `json:"type"`                          // Normal, Warning
	Reason             string            `json:"reason"`                        // 事件原因，如：BackOff, Pulled, Created
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

// GetEventListReq 获取事件列表请求
type GetEventListReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace          string `json:"namespace" form:"namespace" comment:"命名空间"`
	LabelSelector      string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector      string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
	EventType          string `json:"event_type" form:"event_type" comment:"事件类型：Normal,Warning"`
	Reason             string `json:"reason" form:"reason" comment:"事件原因"`
	Source             string `json:"source" form:"source" comment:"事件源组件"`
	InvolvedObjectKind string `json:"involved_object_kind" form:"involved_object_kind" comment:"涉及对象类型"`
	InvolvedObjectName string `json:"involved_object_name" form:"involved_object_name" comment:"涉及对象名称"`
	LimitDays          int    `json:"limit_days" form:"limit_days" comment:"限制天数"`
	Limit              int64  `json:"limit" form:"limit" comment:"限制结果数量"`
	Continue           string `json:"continue" form:"continue" comment:"分页续订令牌"`
}

// GetEventDetailReq 获取事件详情请求
type GetEventDetailReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"事件名称"`
}

// GetEventsByPodReq 获取Pod相关事件请求
type GetEventsByPodReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	PodName   string `json:"pod_name" binding:"required" comment:"Pod名称"`
}

// GetEventsByDeploymentReq 获取Deployment相关事件请求
type GetEventsByDeploymentReq struct {
	ClusterID      int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace      string `json:"namespace" binding:"required" comment:"命名空间"`
	DeploymentName string `json:"deployment_name" binding:"required" comment:"Deployment名称"`
}

// GetEventsByServiceReq 获取Service相关事件请求
type GetEventsByServiceReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	ServiceName string `json:"service_name" binding:"required" comment:"Service名称"`
}

// GetEventsByNodeReq 获取Node相关事件请求
type GetEventsByNodeReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" binding:"required" comment:"Node名称"`
}

// GetEventStatisticsReq 获取事件统计请求
type GetEventStatisticsReq struct {
	ClusterID int       `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string    `json:"namespace" form:"namespace" comment:"命名空间"`
	StartTime time.Time `json:"start_time" form:"start_time" comment:"开始时间"`
	EndTime   time.Time `json:"end_time" form:"end_time" comment:"结束时间"`
	GroupBy   string    `json:"group_by" form:"group_by" comment:"分组方式：type,reason,object,severity"`
}

// GetEventSummaryReq 获取事件汇总请求
type GetEventSummaryReq struct {
	ClusterID int       `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string    `json:"namespace" form:"namespace" comment:"命名空间"`
	StartTime time.Time `json:"start_time" form:"start_time" comment:"开始时间"`
	EndTime   time.Time `json:"end_time" form:"end_time" comment:"结束时间"`
}

// GetEventTimelineReq 获取事件时间线请求
type GetEventTimelineReq struct {
	ClusterID  int       `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace  string    `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ObjectKind string    `json:"object_kind" form:"object_kind" binding:"required" comment:"对象类型"`
	ObjectName string    `json:"object_name" form:"object_name" binding:"required" comment:"对象名称"`
	StartTime  time.Time `json:"start_time" form:"start_time" comment:"开始时间"`
	EndTime    time.Time `json:"end_time" form:"end_time" comment:"结束时间"`
}

// GetEventTrendsReq 获取事件趋势请求
type GetEventTrendsReq struct {
	ClusterID int       `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string    `json:"namespace" form:"namespace" comment:"命名空间"`
	StartTime time.Time `json:"start_time" form:"start_time" comment:"开始时间"`
	EndTime   time.Time `json:"end_time" form:"end_time" comment:"结束时间"`
	Interval  string    `json:"interval" form:"interval" comment:"时间间隔：1m,5m,15m,1h,1d"`
	EventType string    `json:"event_type" form:"event_type" comment:"事件类型：Normal,Warning"`
}

// GetEventGroupDataReq 获取事件分组数据请求
type GetEventGroupDataReq struct {
	ClusterID int       `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string    `json:"namespace" form:"namespace" comment:"命名空间"`
	GroupBy   string    `json:"group_by" form:"group_by" binding:"required" comment:"分组方式：type,reason,object,severity"`
	StartTime time.Time `json:"start_time" form:"start_time" comment:"开始时间"`
	EndTime   time.Time `json:"end_time" form:"end_time" comment:"结束时间"`
	Limit     int       `json:"limit" form:"limit" comment:"限制结果数量"`
}

// DeleteEventReq 删除事件请求
type DeleteEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"事件名称"`
}

// CleanupOldEventsReq 清理旧事件请求
type CleanupOldEventsReq struct {
	ClusterID  int       `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace  string    `json:"namespace" form:"namespace" comment:"命名空间"`
	BeforeTime time.Time `json:"before_time" form:"before_time" binding:"required" comment:"清理此时间之前的事件"`
	EventType  string    `json:"event_type" form:"event_type" comment:"事件类型：Normal,Warning"`
	DryRun     bool      `json:"dry_run" form:"dry_run" comment:"是否为试运行"`
}
