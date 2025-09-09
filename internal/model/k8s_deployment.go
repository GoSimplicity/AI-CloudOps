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

// K8sDeploymentStatus Deployment状态枚举
type K8sDeploymentStatus int8

const (
	K8sDeploymentStatusRunning K8sDeploymentStatus = iota + 1 // 运行中
	K8sDeploymentStatusStopped                                // 停止
	K8sDeploymentStatusPaused                                 // 暂停
	K8sDeploymentStatusError                                  // 异常
)

// K8sDeployment Kubernetes Deployment数据库实体
type K8sDeployment struct {
	Model
	Name              string                `json:"name" binding:"required,min=1,max=200"`      // Deployment名称
	Namespace         string                `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID         int                   `json:"cluster_id" gorm:"index;not null"`           // 所属集群ID
	UID               string                `json:"uid" gorm:"size:100"`                        // Deployment UID
	Replicas          int32                 `json:"replicas"`                                   // 期望副本数
	ReadyReplicas     int32                 `json:"ready_replicas"`                             // 就绪副本数
	AvailableReplicas int32                 `json:"available_replicas"`                         // 可用副本数
	UpdatedReplicas   int32                 `json:"updated_replicas"`                           // 更新副本数
	Strategy          string                `json:"strategy"`                                   // 部署策略
	MaxUnavailable    string                `json:"max_unavailable"`                            // 最大不可用数量
	MaxSurge          string                `json:"max_surge"`                                  // 最大超出数量
	Selector          map[string]string     `json:"selector"`                                   // 标签选择器
	Labels            map[string]string     `json:"labels"`                                     // 标签
	Annotations       map[string]string     `json:"annotations"`                                // 注解
	Images            []string              `json:"images"`                                     // 容器镜像列表
	Status            K8sDeploymentStatus   `json:"status"`                                     // 部署状态
	Conditions        []DeploymentCondition `json:"conditions"`                                 // 部署条件
	CreatedAt         time.Time             `json:"created_at"`                                 // 创建时间
	UpdatedAt         time.Time             `json:"updated_at"`                                 // 更新时间
	RawDeployment     *appsv1.Deployment    `json:"-"`                                          // 原始 Deployment 对象，不序列化到 JSON
}

// DeploymentCondition Deployment条件
type DeploymentCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 条件状态
	LastUpdateTime     time.Time `json:"last_update_time"`     // 最后更新时间
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// DeploymentSpec 创建/更新Deployment时的配置信息
type DeploymentSpec struct {
	Replicas                *int32                     `json:"replicas"`                            // 副本数量
	Selector                *metav1.LabelSelector      `json:"selector"`                            // 标签选择器
	Template                *corev1.PodTemplateSpec    `json:"template"`                            // Pod模板
	Strategy                *appsv1.DeploymentStrategy `json:"strategy,omitempty"`                  // 部署策略
	MinReadySeconds         *int32                     `json:"min_ready_seconds,omitempty"`         // 最小就绪时间
	RevisionHistoryLimit    *int32                     `json:"revision_history_limit,omitempty"`    // 历史版本限制
	Paused                  *bool                      `json:"paused,omitempty"`                    // 是否暂停
	ProgressDeadlineSeconds *int32                     `json:"progress_deadline_seconds,omitempty"` // 进度截止时间
}

// K8sDeploymentEvent Deployment相关事件
type K8sDeploymentEvent struct {
	Type      string    `json:"type"`       // 事件类型
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Count     int32     `json:"count"`      // 事件计数
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Source    string    `json:"source"`     // 事件源
}

// K8sDeploymentMetrics Deployment指标信息

type K8sDeploymentHistory struct {
	Revision int64     `json:"revision"` // 版本
	Date     time.Time `json:"date"`     // 日期
	Message  string    `json:"message"`  // 消息
}

// GetDeploymentListReq 获取Deployment列表请求
type GetDeploymentListReq struct {
	ListReq
	ClusterID int               `json:"cluster_id" form:"cluster_id" comment:"集群ID"` // 集群ID
	Namespace string            `json:"namespace" form:"namespace" comment:"命名空间"`   // 命名空间
	Status    string            `json:"status" form:"status" comment:"Deployment状态"` // Deployment状态
	Labels    map[string]string `json:"labels" form:"labels" comment:"标签"`           // 标签
}

// GetDeploymentDetailsReq 获取Deployment详情请求
type GetDeploymentDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// GetDeploymentYamlReq 获取Deployment YAML请求
type GetDeploymentYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// CreateDeploymentReq 创建Deployment请求
type CreateDeploymentReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Name        string            `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Replicas    int32             `json:"replicas" binding:"required" comment:"副本数量"`     // 副本数量
	Images      []string          `json:"images" binding:"required" comment:"容器镜像列表"`     // 容器镜像列表
	Labels      map[string]string `json:"labels" comment:"标签"`                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                       // 注解
	Spec        DeploymentSpec    `json:"spec" comment:"Deployment规格"`                    // Deployment规格
}

// UpdateDeploymentReq 更新Deployment请求
type UpdateDeploymentReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Name        string            `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Replicas    int32             `json:"replicas" comment:"副本数量"`                        // 副本数量
	Images      []string          `json:"images" comment:"容器镜像列表"`                        // 容器镜像列表
	Labels      map[string]string `json:"labels" comment:"标签"`                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                       // 注解
	Spec        DeploymentSpec    `json:"spec" comment:"Deployment规格"`                    // Deployment规格
}

// CreateDeploymentByYamlReq 通过YAML创建Deployment请求
type CreateDeploymentByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`     // YAML内容
}

// UpdateDeploymentByYamlReq 通过YAML更新Deployment请求
type UpdateDeploymentByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`       // YAML内容
}

// DeleteDeploymentReq 删除Deployment请求
type DeleteDeploymentReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// RestartDeploymentReq 重启Deployment请求
type RestartDeploymentReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// ScaleDeploymentReq 伸缩Deployment请求
type ScaleDeploymentReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	Replicas  int32  `json:"replicas" binding:"required" comment:"副本数量"`     // 副本数量
}

// GetDeploymentMetricsReq 获取Deployment指标请求
type GetDeploymentMetricsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	StartTime string `json:"start_time" form:"start_time" comment:"开始时间"`    // 开始时间
	EndTime   string `json:"end_time" form:"end_time" comment:"结束时间"`        // 结束时间
	Step      string `json:"step" form:"step" comment:"查询步长"`                // 查询步长
}

// GetDeploymentEventsReq 获取Deployment事件请求
type GetDeploymentEventsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	EventType string `json:"event_type" form:"event_type" comment:"事件类型"`    // 事件类型
	Limit     int    `json:"limit" form:"limit" comment:"限制数量"`              // 限制数量
}

// GetDeploymentPodsReq 获取Deployment下的Pod列表请求
type GetDeploymentPodsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// GetDeploymentHistoryReq 获取Deployment版本历史请求
type GetDeploymentHistoryReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// RollbackDeploymentReq 回滚Deployment请求
type RollbackDeploymentReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
	Revision  int64  `json:"revision" binding:"required" comment:"回滚到的版本号"`  // 回滚到的版本号
}

// PauseDeploymentReq 暂停Deployment请求
type PauseDeploymentReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}

// ResumeDeploymentReq 恢复Deployment请求
type ResumeDeploymentReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称
}
