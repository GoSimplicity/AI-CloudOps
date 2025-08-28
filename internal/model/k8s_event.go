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

	corev1 "k8s.io/api/core/v1"
)

// K8sEventEntity Kubernetes Event数据库实体
type K8sEventEntity struct {
	Model
	Name              string                 `json:"name" gorm:"size:200;comment:事件名称"`                             // 事件名称
	Namespace         string                 `json:"namespace" gorm:"size:200;comment:所属命名空间"`                      // 所属命名空间
	ClusterID         int                    `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`               // 所属集群ID
	UID               string                 `json:"uid" gorm:"size:100;comment:事件UID"`                             // 事件UID
	Type              string                 `json:"type" gorm:"size:50;comment:事件类型"`                              // 事件类型
	Reason            string                 `json:"reason" gorm:"size:200;comment:事件原因"`                           // 事件原因
	Message           string                 `json:"message" gorm:"type:text;comment:事件消息"`                         // 事件消息
	Source            corev1.EventSource     `json:"source" gorm:"type:text;serializer:json;comment:事件来源"`          // 事件来源
	InvolvedObject    corev1.ObjectReference `json:"involved_object" gorm:"type:text;serializer:json;comment:相关对象"` // 相关对象
	FirstTimestamp    time.Time              `json:"first_timestamp" gorm:"comment:首次发生时间"`                         // 首次发生时间
	LastTimestamp     time.Time              `json:"last_timestamp" gorm:"comment:最后发生时间"`                          // 最后发生时间
	Count             int32                  `json:"count" gorm:"comment:发生次数"`                                     // 发生次数
	CreationTimestamp time.Time              `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`              // Kubernetes创建时间
	Age               string                 `json:"age" gorm:"-"`                                                  // 存在时间，前端计算使用
	Severity          string                 `json:"severity" gorm:"-"`                                             // 事件严重程度，前端计算使用
}

func (k *K8sEventEntity) TableName() string {
	return "cl_k8s_events"
}

// K8sEventListRequest Event列表查询请求
type K8sEventListReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace          string `json:"namespace" form:"namespace" comment:"命名空间"`                         // 命名空间
	LabelSelector      string `json:"label_selector" form:"label_selector" comment:"标签选择器"`              // 标签选择器
	FieldSelector      string `json:"field_selector" form:"field_selector" comment:"字段选择器"`              // 字段选择器
	EventType          string `json:"event_type" form:"event_type" comment:"事件类型过滤"`                     // 事件类型过滤 (Normal/Warning)
	Type               string `json:"type" form:"type" comment:"事件类型过滤"`                                 // 事件类型过滤
	Reason             string `json:"reason" form:"reason" comment:"事件原因过滤"`                             // 事件原因过滤
	Source             string `json:"source" form:"source" comment:"事件来源过滤"`                             // 事件来源过滤
	InvolvedObjectKind string `json:"involved_object_kind" form:"involved_object_kind" comment:"相关对象类型"` // 相关对象类型
	InvolvedObjectName string `json:"involved_object_name" form:"involved_object_name" comment:"相关对象名称"` // 相关对象名称
	Severity           string `json:"severity" form:"severity" comment:"严重程度过滤"`                         // 严重程度过滤
	LimitDays          int    `json:"limit_days" form:"limit_days" comment:"限制天数内的事件"`                   // 限制天数内的事件
	StartTime          string `json:"start_time" form:"start_time" comment:"开始时间"`                       // 开始时间
	EndTime            string `json:"end_time" form:"end_time" comment:"结束时间"`                           // 结束时间
	Page               int    `json:"page" form:"page" comment:"页码"`                                     // 页码
	PageSize           int    `json:"page_size" form:"page_size" comment:"每页大小"`                         // 每页大小
}

// K8sEventSearchRequest Event搜索请求
type K8sEventSearchReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" comment:"命名空间"`                     // 命名空间
	Keyword            string `json:"keyword" comment:"搜索关键词"`                      // 搜索关键词
	Type               string `json:"type" comment:"事件类型过滤"`                        // 事件类型过滤
	InvolvedObjectKind string `json:"involved_object_kind" comment:"相关对象类型"`        // 相关对象类型
	InvolvedObjectName string `json:"involved_object_name" comment:"相关对象名称"`        // 相关对象名称
	LimitDays          int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
	Page               int    `json:"page" comment:"页码"`                            // 页码
	PageSize           int    `json:"page_size" comment:"每页大小"`                     // 每页大小
}

// K8sEventStatisticsRequest Event统计请求
type K8sEventStatisticsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" comment:"命名空间"`                     // 命名空间
	LimitDays int    `json:"limit_days" comment:"统计天数"`                    // 统计天数
}

// K8sEventExportRequest Event导出请求
type K8sEventExportReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" comment:"命名空间"`                     // 命名空间
	Type               string `json:"type" comment:"事件类型过滤"`                        // 事件类型过滤
	Reason             string `json:"reason" comment:"事件原因过滤"`                      // 事件原因过滤
	InvolvedObjectKind string `json:"involved_object_kind" comment:"相关对象类型"`        // 相关对象类型
	InvolvedObjectName string `json:"involved_object_name" comment:"相关对象名称"`        // 相关对象名称
	StartTime          string `json:"start_time" comment:"开始时间"`                    // 开始时间
	EndTime            string `json:"end_time" comment:"结束时间"`                      // 结束时间
	Format             string `json:"format" comment:"导出格式(excel,csv,json)"`        // 导出格式
}

// K8sEventCleanupRequest Event清理请求
type K8sEventCleanupReq struct {
	ClusterID  int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace  string `json:"namespace" comment:"命名空间"`                     // 命名空间
	DaysToKeep int    `json:"days_to_keep" comment:"保留天数"`                  // 保留天数，删除指定天数之前的事件
}

// K8sEventAlertRequest Event告警规则请求
type K8sEventAlertReq struct {
	ClusterID    int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace    string   `json:"namespace" comment:"命名空间"`                     // 命名空间
	EventTypes   []string `json:"event_types" comment:"监控的事件类型"`                // 监控的事件类型
	EventReasons []string `json:"event_reasons" comment:"监控的事件原因"`              // 监控的事件原因
	AlertChannel string   `json:"alert_channel" comment:"告警渠道"`                 // 告警渠道
	Enabled      bool     `json:"enabled" comment:"是否启用告警"`                     // 是否启用告警
}

// K8sEventByObjectRequest 根据对象查询事件请求
type K8sEventByObjectReq struct {
	ClusterID  int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace  string `json:"namespace" comment:"命名空间"`                      // 命名空间
	ObjectName string `json:"object_name" binding:"required" comment:"对象名称"` // 对象名称，必填
	ObjectKind string `json:"object_kind" binding:"required" comment:"对象类型"` // 对象类型，必填
	ObjectUID  string `json:"object_uid" comment:"对象UID"`                    // 对象UID，可选
	LimitDays  int    `json:"limit_days" comment:"限制天数内的事件"`                 // 限制天数内的事件
}

// K8sEventTimelineRequest 事件时间线查询请求
type K8sEventTimelineReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" comment:"命名空间"`                     // 命名空间
	InvolvedObjectKind string `json:"involved_object_kind" comment:"相关对象类型"`        // 相关对象类型
	InvolvedObjectName string `json:"involved_object_name" comment:"相关对象名称"`        // 相关对象名称
	StartTime          string `json:"start_time" comment:"开始时间"`                    // 开始时间
	EndTime            string `json:"end_time" comment:"结束时间"`                      // 结束时间
	Granularity        string `json:"granularity" comment:"时间粒度(hour/day)"`         // 时间粒度
}

// ====================== Event响应实体 ======================

// EventEntity Event响应实体
type EventEntity struct {
	Name           string            `json:"name"`            // 事件名称
	Namespace      string            `json:"namespace"`       // 命名空间
	UID            string            `json:"uid"`             // 事件UID
	Type           string            `json:"type"`            // 事件类型(Normal/Warning)
	Reason         string            `json:"reason"`          // 事件原因
	Message        string            `json:"message"`         // 事件消息
	Source         EventSourceEntity `json:"source"`          // 事件来源
	InvolvedObject EventObjectEntity `json:"involved_object"` // 相关对象
	FirstTime      string            `json:"first_time"`      // 首次发生时间
	LastTime       string            `json:"last_time"`       // 最后发生时间
	Count          int32             `json:"count"`           // 发生次数
	Severity       string            `json:"severity"`        // 事件严重程度
	Age            string            `json:"age"`             // 存在时间
	CreatedAt      string            `json:"created_at"`      // 创建时间
}

// EventSourceEntity 事件来源实体
type EventSourceEntity struct {
	Component string `json:"component"` // 组件名称
	Host      string `json:"host"`      // 主机名
}

// EventObjectEntity 事件相关对象实体
type EventObjectEntity struct {
	Kind            string `json:"kind"`             // 对象类型
	Name            string `json:"name"`             // 对象名称
	Namespace       string `json:"namespace"`        // 命名空间
	UID             string `json:"uid"`              // 对象UID
	APIVersion      string `json:"api_version"`      // API版本
	ResourceVersion string `json:"resource_version"` // 资源版本
	FieldPath       string `json:"field_path"`       // 字段路径
}

// EventListResponse Event列表响应
type EventListResponse struct {
	Items      []EventEntity `json:"items"`       // Event列表
	TotalCount int           `json:"total_count"` // 总数
}

// EventDetailResponse Event详情响应
type EventDetailResponse struct {
	Event         EventEntity         `json:"event"`          // Event信息
	RelatedEvents []EventEntity       `json:"related_events"` // 相关事件
	Timeline      EventTimelineEntity `json:"timeline"`       // 事件时间线
}

// EventTimelineEntity 事件时间线实体
type EventTimelineEntity struct {
	StartTime  string                     `json:"start_time"`  // 开始时间
	EndTime    string                     `json:"end_time"`    // 结束时间
	TimePoints []EventTimelinePointEntity `json:"time_points"` // 时间点列表
}

// EventTimelinePointEntity 事件时间线点实体
type EventTimelinePointEntity struct {
	Time      string `json:"time"`       // 时间
	Count     int    `json:"count"`      // 事件数量
	EventType string `json:"event_type"` // 事件类型
}

// EventStatisticsResponse Event统计响应
type EventStatisticsResponse struct {
	TotalEvents       int                        `json:"total_events"`        // 总事件数
	NormalEvents      int                        `json:"normal_events"`       // 正常事件数
	WarningEvents     int                        `json:"warning_events"`      // 警告事件数
	EventsByType      []EventTypeStatEntity      `json:"events_by_type"`      // 按类型统计
	EventsByReason    []EventReasonStatEntity    `json:"events_by_reason"`    // 按原因统计
	EventsByNamespace []EventNamespaceStatEntity `json:"events_by_namespace"` // 按命名空间统计
	EventsByObject    []EventObjectStatEntity    `json:"events_by_object"`    // 按对象统计
	TrendData         []EventTrendDataEntity     `json:"trend_data"`          // 趋势数据
}

// EventTypeStatEntity 事件类型统计实体
type EventTypeStatEntity struct {
	Type  string `json:"type"`  // 事件类型
	Count int    `json:"count"` // 数量
}

// EventReasonStatEntity 事件原因统计实体
type EventReasonStatEntity struct {
	Reason string `json:"reason"` // 事件原因
	Count  int    `json:"count"`  // 数量
}

// EventNamespaceStatEntity 事件命名空间统计实体
type EventNamespaceStatEntity struct {
	Namespace string `json:"namespace"` // 命名空间
	Count     int    `json:"count"`     // 数量
}

// EventObjectStatEntity 事件对象统计实体
type EventObjectStatEntity struct {
	ObjectKind string `json:"object_kind"` // 对象类型
	ObjectName string `json:"object_name"` // 对象名称
	Namespace  string `json:"namespace"`   // 命名空间
	Count      int    `json:"count"`       // 数量
}

// EventTrendDataEntity 事件趋势数据实体
type EventTrendDataEntity struct {
	Time         string `json:"time"`          // 时间
	NormalCount  int    `json:"normal_count"`  // 正常事件数
	WarningCount int    `json:"warning_count"` // 警告事件数
}

// EventSearchResponse Event搜索响应
type EventSearchResponse struct {
	Items      []EventEntity `json:"items"`       // 搜索结果
	TotalCount int           `json:"total_count"` // 总数
	Keywords   []string      `json:"keywords"`    // 搜索关键词
	SearchTime string        `json:"search_time"` // 搜索时间
}

// EventExportResponse Event导出响应
type EventExportResponse struct {
	FileName    string `json:"file_name"`    // 文件名
	FilePath    string `json:"file_path"`    // 文件路径
	FileSize    string `json:"file_size"`    // 文件大小
	Format      string `json:"format"`       // 导出格式
	EventCount  int    `json:"event_count"`  // 事件数量
	ExportTime  string `json:"export_time"`  // 导出时间
	DownloadUrl string `json:"download_url"` // 下载链接
}

// EventCleanupResponse Event清理响应
type EventCleanupResponse struct {
	ClusterID      int    `json:"cluster_id"`      // 集群ID
	Namespace      string `json:"namespace"`       // 命名空间
	DaysToKeep     int    `json:"days_to_keep"`    // 保留天数
	DeletedCount   int    `json:"deleted_count"`   // 删除数量
	RemainingCount int    `json:"remaining_count"` // 剩余数量
	CleanupTime    string `json:"cleanup_time"`    // 清理时间
	Status         string `json:"status"`          // 清理状态
	Message        string `json:"message"`         // 清理消息
}

// EventAlertResponse Event告警响应
type EventAlertResponse struct {
	ClusterID     int      `json:"cluster_id"`      // 集群ID
	Namespace     string   `json:"namespace"`       // 命名空间
	EventTypes    []string `json:"event_types"`     // 监控的事件类型
	EventReasons  []string `json:"event_reasons"`   // 监控的事件原因
	AlertChannel  string   `json:"alert_channel"`   // 告警渠道
	Enabled       bool     `json:"enabled"`         // 是否启用告警
	LastAlertTime string   `json:"last_alert_time"` // 最后告警时间
	AlertCount    int      `json:"alert_count"`     // 告警次数
	Status        string   `json:"status"`          // 告警状态
}

// EventByObjectResponse 根据对象查询事件响应
type EventByObjectResponse struct {
	ObjectName   string        `json:"object_name"`   // 对象名称
	ObjectKind   string        `json:"object_kind"`   // 对象类型
	ObjectUID    string        `json:"object_uid"`    // 对象UID
	Namespace    string        `json:"namespace"`     // 命名空间
	Events       []EventEntity `json:"events"`        // 事件列表
	TotalCount   int           `json:"total_count"`   // 总数
	WarningCount int           `json:"warning_count"` // 警告数量
	NormalCount  int           `json:"normal_count"`  // 正常数量
}
