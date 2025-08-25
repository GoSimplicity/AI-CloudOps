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
type K8sEventListRequest struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace          string `json:"namespace" form:"namespace" comment:"命名空间"`                         // 命名空间
	Type               string `json:"type" form:"type" comment:"事件类型过滤"`                                 // 事件类型过滤
	Reason             string `json:"reason" form:"reason" comment:"事件原因过滤"`                             // 事件原因过滤
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
type K8sEventSearchRequest struct {
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
type K8sEventStatisticsRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" comment:"命名空间"`                     // 命名空间
	LimitDays int    `json:"limit_days" comment:"统计天数"`                    // 统计天数
}

// K8sEventExportRequest Event导出请求
type K8sEventExportRequest struct {
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
type K8sEventCleanupRequest struct {
	ClusterID  int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace  string `json:"namespace" comment:"命名空间"`                     // 命名空间
	DaysToKeep int    `json:"days_to_keep" comment:"保留天数"`                  // 保留天数，删除指定天数之前的事件
}

// K8sEventAlertRequest Event告警规则请求
type K8sEventAlertRequest struct {
	ClusterID    int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace    string   `json:"namespace" comment:"命名空间"`                     // 命名空间
	EventTypes   []string `json:"event_types" comment:"监控的事件类型"`                // 监控的事件类型
	EventReasons []string `json:"event_reasons" comment:"监控的事件原因"`              // 监控的事件原因
	AlertChannel string   `json:"alert_channel" comment:"告警渠道"`                 // 告警渠道
	Enabled      bool     `json:"enabled" comment:"是否启用告警"`                     // 是否启用告警
}
