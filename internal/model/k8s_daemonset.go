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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sDaemonSetStatus int8

const (
	K8sDaemonSetStatusRunning  K8sDaemonSetStatus = iota + 1 // 运行中
	K8sDaemonSetStatusError                                  // 异常
	K8sDaemonSetStatusUpdating                               // 更新中
)

type K8sDaemonSet struct {
	Model
	Name                   string               `json:"name" binding:"required,min=1,max=200"`      // DaemonSet名称
	Namespace              string               `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID              int                  `json:"cluster_id" gorm:"index;not null"`           // 所属集群ID
	UID                    string               `json:"uid" gorm:"size:100"`                        // DaemonSet UID
	DesiredNumberScheduled int32                `json:"desired_number_scheduled"`                   // 期望调度数量
	CurrentNumberScheduled int32                `json:"current_number_scheduled"`                   // 当前调度数量
	NumberReady            int32                `json:"number_ready"`                               // 就绪数量
	NumberAvailable        int32                `json:"number_available"`                           // 可用数量
	NumberUnavailable      int32                `json:"number_unavailable"`                         // 不可用数量
	UpdatedNumberScheduled int32                `json:"updated_number_scheduled"`                   // 更新调度数量
	NumberMisscheduled     int32                `json:"number_misscheduled"`                        // 错误调度数量
	UpdateStrategy         string               `json:"update_strategy"`                            // 更新策略
	RevisionHistoryLimit   int32                `json:"revision_history_limit"`                     // 历史版本限制
	Selector               map[string]string    `json:"selector"`                                   // 标签选择器
	Labels                 map[string]string    `json:"labels"`                                     // 标签
	Annotations            map[string]string    `json:"annotations"`                                // 注解
	Images                 []string             `json:"images"`                                     // 容器镜像列表
	Status                 K8sDaemonSetStatus   `json:"status"`                                     // DaemonSet状态
	Conditions             []DaemonSetCondition `json:"conditions"`                                 // DaemonSet条件
	CreatedAt              time.Time            `json:"created_at"`                                 // 创建时间
	UpdatedAt              time.Time            `json:"updated_at"`                                 // 更新时间
	RawDaemonSet           *appsv1.DaemonSet    `json:"-"`                                          // 原始 DaemonSet 对象，不序列化到 JSON
}

// DaemonSetCondition DaemonSet条件
type DaemonSetCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 条件状态
	LastUpdateTime     time.Time `json:"last_update_time"`     // 最后更新时间
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// DaemonSetSpec 创建/更新DaemonSet时的配置信息
type DaemonSetSpec struct {
	Selector             *metav1.LabelSelector           `json:"selector"`                         // 标签选择器
	Template             *corev1.PodTemplateSpec         `json:"template"`                         // Pod模板
	UpdateStrategy       *appsv1.DaemonSetUpdateStrategy `json:"update_strategy,omitempty"`        // 更新策略
	MinReadySeconds      *int32                          `json:"min_ready_seconds,omitempty"`      // 最小就绪时间
	RevisionHistoryLimit *int32                          `json:"revision_history_limit,omitempty"` // 历史版本限制
}

// K8sDaemonSetEvent DaemonSet相关事件
type K8sDaemonSetEvent struct {
	Type      string    `json:"type"`       // 事件类型
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Count     int32     `json:"count"`      // 事件计数
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Source    string    `json:"source"`     // 事件源
}

// K8sDaemonSetMetrics DaemonSet指标信息
type K8sDaemonSetMetrics struct {
	CPUUsage         float64   `json:"cpu_usage"`              // CPU使用率
	MemoryUsage      float64   `json:"memory_usage"`           // 内存使用率
	NetworkIn        float64   `json:"network_in"`             // 网络入流量（MB/s）
	NetworkOut       float64   `json:"network_out"`            // 网络出流量（MB/s）
	DiskUsage        float64   `json:"disk_usage"`             // 磁盘使用率
	NodesReady       int32     `json:"nodes_ready"`            // 就绪节点数
	NodesTotal       int32     `json:"nodes_total"`            // 总节点数
	RestartCount     int32     `json:"restart_count"`          // 重启次数
	AvailabilityRate float64   `json:"availability_rate"`      // 可用性
	LastUpdated      time.Time `json:"last_updated"`           // 最后更新时间
	MetricsAvailable bool      `json:"metrics_available"`      // 是否有详细指标数据（需要metrics-server）
	MetricsNote      string    `json:"metrics_note,omitempty"` // 指标说明信息
}

type K8sDaemonSetHistory struct {
	Revision int64     `json:"revision"` // 版本
	Date     time.Time `json:"date"`     // 日期
	Message  string    `json:"message"`  // 消息
}

// GetDaemonSetListReq 获取DaemonSet列表请求
type GetDaemonSetListReq struct {
	ListReq
	ClusterID int               `json:"cluster_id" form:"cluster_id"` // 集群ID
	Namespace string            `json:"namespace" form:"namespace"`   // 命名空间
	Status    string            `json:"status" form:"status"`         // DaemonSet状态
	Labels    map[string]string `json:"labels" form:"labels"`         // 标签
}

// GetDaemonSetDetailsReq 获取DaemonSet详情请求
type GetDaemonSetDetailsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // DaemonSet名称
}

// GetDaemonSetYamlReq 获取DaemonSet YAML请求
type GetDaemonSetYamlReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // DaemonSet名称
}

// CreateDaemonSetReq 创建DaemonSet请求
type CreateDaemonSetReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required"` // 集群ID
	Name        string            `json:"name" binding:"required"`       // DaemonSet名称
	Namespace   string            `json:"namespace" binding:"required"`  // 命名空间
	Images      []string          `json:"images" binding:"required"`     // 容器镜像列表
	Labels      map[string]string `json:"labels"`                        // 标签
	Annotations map[string]string `json:"annotations"`                   // 注解
	Spec        DaemonSetSpec     `json:"spec"`                          // DaemonSet规格
	YAML        string            `json:"yaml"`                          // YAML内容
}

// UpdateDaemonSetReq 更新DaemonSet请求
type UpdateDaemonSetReq struct {
	ClusterID   int               `json:"cluster_id"`  // 集群ID
	Name        string            `json:"name"`        // DaemonSet名称
	Namespace   string            `json:"namespace"`   // 命名空间
	Images      []string          `json:"images"`      // 容器镜像列表
	Labels      map[string]string `json:"labels"`      // 标签
	Annotations map[string]string `json:"annotations"` // 注解
	Spec        DaemonSetSpec     `json:"spec"`        // DaemonSet规格
	YAML        string            `json:"yaml"`        // YAML内容
}

// DeleteDaemonSetReq 删除DaemonSet请求
type DeleteDaemonSetReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // DaemonSet名称
}

// RestartDaemonSetReq 重启DaemonSet请求
type RestartDaemonSetReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // DaemonSet名称
}

// GetDaemonSetMetricsReq 获取DaemonSet指标请求
type GetDaemonSetMetricsReq struct {
	ClusterID int    `json:"cluster_id"`                   // 集群ID
	Namespace string `json:"namespace"`                    // 命名空间
	Name      string `json:"name"`                         // DaemonSet名称
	StartTime string `json:"start_time" form:"start_time"` // 开始时间
	EndTime   string `json:"end_time" form:"end_time"`     // 结束时间
	Step      string `json:"step" form:"step"`             // 查询步长
}

// GetDaemonSetEventsReq 获取DaemonSet事件请求
type GetDaemonSetEventsReq struct {
	ClusterID int    `json:"cluster_id"`                   // 集群ID
	Namespace string `json:"namespace"`                    // 命名空间
	Name      string `json:"name"`                         // DaemonSet名称
	EventType string `json:"event_type" form:"event_type"` // 事件类型
	Limit     int    `json:"limit" form:"limit"`           // 限制数量
}

// GetDaemonSetPodsReq 获取DaemonSet下的Pod列表请求
type GetDaemonSetPodsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // DaemonSet名称
}

// GetDaemonSetHistoryReq 获取DaemonSet版本历史请求
type GetDaemonSetHistoryReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // DaemonSet名称
}

// RollbackDaemonSetReq 回滚DaemonSet请求
type RollbackDaemonSetReq struct {
	ClusterID int    `json:"cluster_id"`                  // 集群ID
	Namespace string `json:"namespace"`                   // 命名空间
	Name      string `json:"name"`                        // DaemonSet名称
	Revision  int64  `json:"revision" binding:"required"` // 回滚到的版本号
}
