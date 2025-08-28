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
)

// K8sDaemonSetEntity Kubernetes DaemonSet数据库实体
type K8sDaemonSetEntity struct {
	Model
	Name                   string                 `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:DaemonSet名称"` // DaemonSet名称
	Namespace              string                 `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID              int                    `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID                    string                 `json:"uid" gorm:"size:100;comment:DaemonSet UID"`                                 // DaemonSet UID
	DesiredNumberScheduled int32                  `json:"desired_number_scheduled" gorm:"comment:期望调度数量"`                            // 期望调度数量
	CurrentNumberScheduled int32                  `json:"current_number_scheduled" gorm:"comment:当前调度数量"`                            // 当前调度数量
	NumberReady            int32                  `json:"number_ready" gorm:"comment:就绪数量"`                                          // 就绪数量
	NumberAvailable        int32                  `json:"number_available" gorm:"comment:可用数量"`                                      // 可用数量
	NumberUnavailable      int32                  `json:"number_unavailable" gorm:"comment:不可用数量"`                                   // 不可用数量
	UpdatedNumberScheduled int32                  `json:"updated_number_scheduled" gorm:"comment:更新调度数量"`                            // 更新调度数量
	NumberMisscheduled     int32                  `json:"number_misscheduled" gorm:"comment:错误调度数量"`                                 // 错误调度数量
	UpdateStrategy         string                 `json:"update_strategy" gorm:"size:50;comment:更新策略"`                               // 更新策略
	RevisionHistoryLimit   int32                  `json:"revision_history_limit" gorm:"comment:历史版本限制"`                              // 历史版本限制
	Selector               map[string]string      `json:"selector" gorm:"type:text;serializer:json;comment:选择器"`                     // 选择器
	PodTemplate            map[string]interface{} `json:"pod_template" gorm:"type:text;serializer:json;comment:Pod模板"`               // Pod模板
	Labels                 map[string]string      `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations            map[string]string      `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp      time.Time              `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age                    string                 `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	Status                 string                 `json:"status" gorm:"-"`                                                           // DaemonSet状态，前端计算使用
	Images                 []string               `json:"images" gorm:"-"`                                                           // 镜像列表，前端计算使用
}

func (k *K8sDaemonSetEntity) TableName() string {
	return "cl_k8s_daemonsets"
}

// K8sDaemonSetListRequest DaemonSet列表查询请求
type K8sDaemonSetListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sDaemonSetCreateRequest 创建DaemonSet请求
type K8sDaemonSetCreateReq struct {
	ClusterID            int                    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace            string                 `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name                 string                 `json:"name" binding:"required" comment:"DaemonSet名称"`   // DaemonSet名称，必填
	UpdateStrategy       string                 `json:"update_strategy" comment:"更新策略"`                  // 更新策略
	RevisionHistoryLimit *int32                 `json:"revision_history_limit" comment:"历史版本限制"`         // 历史版本限制
	Selector             map[string]string      `json:"selector" binding:"required" comment:"选择器"`       // 选择器，必填
	PodTemplate          map[string]interface{} `json:"pod_template" binding:"required" comment:"Pod模板"` // Pod模板，必填
	Labels               map[string]string      `json:"labels" comment:"标签"`                             // 标签
	Annotations          map[string]string      `json:"annotations" comment:"注解"`                        // 注解
	DaemonSetYaml        *appsv1.DaemonSet      `json:"daemonset_yaml" comment:"DaemonSet YAML对象"`       // DaemonSet YAML对象
}

// K8sDaemonSetUpdateRequest 更新DaemonSet请求
type K8sDaemonSetUpdateReq struct {
	ClusterID            int                    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace            string                 `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name                 string                 `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	UpdateStrategy       string                 `json:"update_strategy" comment:"更新策略"`                // 更新策略
	RevisionHistoryLimit *int32                 `json:"revision_history_limit" comment:"历史版本限制"`       // 历史版本限制
	Selector             map[string]string      `json:"selector" comment:"选择器"`                        // 选择器
	PodTemplate          map[string]interface{} `json:"pod_template" comment:"Pod模板"`                  // Pod模板
	Labels               map[string]string      `json:"labels" comment:"标签"`                           // 标签
	Annotations          map[string]string      `json:"annotations" comment:"注解"`                      // 注解
	DaemonSetYaml        *appsv1.DaemonSet      `json:"daemonset_yaml" comment:"DaemonSet YAML对象"`     // DaemonSet YAML对象
}

// K8sDaemonSetDeleteRequest 删除DaemonSet请求
type K8sDaemonSetDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`      // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                        // 是否强制删除
	OrphanDependents   bool   `json:"orphan_dependents" comment:"是否保留依赖资源"`          // 是否保留依赖资源
}

// K8sDaemonSetBatchDeleteRequest 批量删除DaemonSet请求
type K8sDaemonSetBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"DaemonSet名称列表"` // DaemonSet名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`         // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                           // 是否强制删除
	OrphanDependents   bool     `json:"orphan_dependents" comment:"是否保留依赖资源"`             // 是否保留依赖资源
}

// K8sDaemonSetRestartRequest 重启DaemonSet请求
type K8sDaemonSetRestartReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
}

// K8sDaemonSetRollbackRequest 回滚DaemonSet请求
type K8sDaemonSetRollbackReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	Revision  int64  `json:"revision" binding:"required" comment:"回滚版本"`    // 回滚版本，必填
}

// K8sDaemonSetEventRequest 获取DaemonSet事件请求
type K8sDaemonSetEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                 // 限制天数内的事件
}

// K8sDaemonSetMetricsRequest 获取DaemonSet指标请求
type K8sDaemonSetMetricsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	TimeRange string `json:"time_range" comment:"时间范围"`                     // 时间范围
}

// K8sDaemonSetHistoryRequest 获取DaemonSet历史版本请求
type K8sDaemonSetHistoryReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
}

// K8sDaemonSetBatchRestartReq 批量重启DaemonSet请求
type K8sDaemonSetBatchRestartReq struct {
	ClusterID int      `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace string   `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Names     []string `json:"names" binding:"required" comment:"DaemonSet名称列表"` // DaemonSet名称列表，必填
}

// K8sDaemonSetNodePodsReq 获取DaemonSet在指定节点上的Pod请求
type K8sDaemonSetNodePodsReq struct {
	ClusterID     int    `json:"cluster_id" binding:"required" comment:"集群ID"`            // 集群ID，必填
	Namespace     string `json:"namespace" binding:"required" comment:"命名空间"`             // 命名空间，必填
	DaemonSetName string `json:"daemonset_name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	NodeName      string `json:"node_name" binding:"required" comment:"节点名称"`             // 节点名称，必填
}

// ====================== DaemonSet响应实体 ======================

// DaemonSetEntity DaemonSet响应实体
type DaemonSetEntity struct {
	Name                   string            `json:"name"`                     // DaemonSet名称
	Namespace              string            `json:"namespace"`                // 命名空间
	UID                    string            `json:"uid"`                      // DaemonSet UID
	Labels                 map[string]string `json:"labels"`                   // 标签
	Annotations            map[string]string `json:"annotations"`              // 注解
	DesiredNumberScheduled int32             `json:"desired_number_scheduled"` // 期望调度数量
	CurrentNumberScheduled int32             `json:"current_number_scheduled"` // 当前调度数量
	NumberReady            int32             `json:"number_ready"`             // 就绪数量
	NumberAvailable        int32             `json:"number_available"`         // 可用数量
	NumberUnavailable      int32             `json:"number_unavailable"`       // 不可用数量
	UpdatedNumberScheduled int32             `json:"updated_number_scheduled"` // 更新调度数量
	NumberMisscheduled     int32             `json:"number_misscheduled"`      // 错误调度数量
	UpdateStrategy         string            `json:"update_strategy"`          // 更新策略
	RevisionHistoryLimit   int32             `json:"revision_history_limit"`   // 历史版本限制
	Status                 string            `json:"status"`                   // DaemonSet状态
	Images                 []string          `json:"images"`                   // 镜像列表
	Age                    string            `json:"age"`                      // 存在时间
	CreatedAt              string            `json:"created_at"`               // 创建时间
}

// DaemonSetListResponse DaemonSet列表响应
type DaemonSetListResponse struct {
	Items      []DaemonSetEntity `json:"items"`       // DaemonSet列表
	TotalCount int               `json:"total_count"` // 总数
}

// DaemonSetDetailResponse DaemonSet详情响应
type DaemonSetDetailResponse struct {
	DaemonSet DaemonSetEntity          `json:"daemonset"` // DaemonSet信息
	YAML      string                   `json:"yaml"`      // YAML内容
	Events    []DaemonSetEventEntity   `json:"events"`    // 事件列表
	Pods      []DaemonSetPodEntity     `json:"pods"`      // Pod列表
	Metrics   DaemonSetMetricsEntity   `json:"metrics"`   // 指标信息
	History   []DaemonSetHistoryEntity `json:"history"`   // 历史版本
}

// DaemonSetEventEntity DaemonSet事件实体
type DaemonSetEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// DaemonSetPodEntity DaemonSet Pod实体
type DaemonSetPodEntity struct {
	Name      string `json:"name"`      // Pod名称
	Ready     string `json:"ready"`     // 就绪状态
	Status    string `json:"status"`    // Pod状态
	Restarts  int32  `json:"restarts"`  // 重启次数
	Age       string `json:"age"`       // 存在时间
	IP        string `json:"ip"`        // Pod IP
	Node      string `json:"node"`      // 节点名称
	Nominated string `json:"nominated"` // 提名节点
	Readiness string `json:"readiness"` // 就绪状态
}

// DaemonSetMetricsEntity DaemonSet指标实体
type DaemonSetMetricsEntity struct {
	CPUUsage     float64 `json:"cpu_usage"`     // CPU使用量
	MemoryUsage  int64   `json:"memory_usage"`  // 内存使用量
	NetworkRx    int64   `json:"network_rx"`    // 网络接收
	NetworkTx    int64   `json:"network_tx"`    // 网络发送
	StorageUsage int64   `json:"storage_usage"` // 存储使用量
}

// DaemonSetHistoryEntity DaemonSet历史版本实体
type DaemonSetHistoryEntity struct {
	Revision     int64  `json:"revision"`      // 版本号
	ChangeReason string `json:"change_reason"` // 变更原因
	CreatedAt    string `json:"created_at"`    // 创建时间
	Image        string `json:"image"`         // 镜像
	Current      bool   `json:"current"`       // 是否当前版本
}

// DaemonSetRestartResponse DaemonSet重启响应
type DaemonSetRestartResponse struct {
	Name          string   `json:"name"`           // DaemonSet名称
	Namespace     string   `json:"namespace"`      // 命名空间
	Status        string   `json:"status"`         // 重启状态
	Message       string   `json:"message"`        // 重启消息
	RestartedPods []string `json:"restarted_pods"` // 重启的Pod列表
	StartTime     string   `json:"start_time"`     // 开始时间
	EndTime       string   `json:"end_time"`       // 结束时间
}

// DaemonSetRollbackResponse DaemonSet回滚响应
type DaemonSetRollbackResponse struct {
	Name         string `json:"name"`          // DaemonSet名称
	Namespace    string `json:"namespace"`     // 命名空间
	FromRevision int64  `json:"from_revision"` // 源版本
	ToRevision   int64  `json:"to_revision"`   // 目标版本
	Status       string `json:"status"`        // 回滚状态
	Message      string `json:"message"`       // 回滚消息
	StartTime    string `json:"start_time"`    // 开始时间
	EndTime      string `json:"end_time"`      // 结束时间
}
