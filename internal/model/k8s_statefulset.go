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

type K8sStatefulSetStatus int8

const (
	K8sStatefulSetStatusRunning  K8sStatefulSetStatus = iota + 1 // 运行中
	K8sStatefulSetStatusStopped                                  // 停止
	K8sStatefulSetStatusUpdating                                 // 更新中
	K8sStatefulSetStatusError                                    // 异常
)

type K8sStatefulSet struct {
	Name                 string                 `json:"name" binding:"required,min=1,max=200"`      // StatefulSet名称
	Namespace            string                 `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID            int                    `json:"cluster_id" gorm:"index;not null"`           // 所属集群ID
	UID                  string                 `json:"uid" gorm:"size:100"`                        // StatefulSet UID
	Replicas             int32                  `json:"replicas"`                                   // 期望副本数
	ReadyReplicas        int32                  `json:"ready_replicas"`                             // 就绪副本数
	CurrentReplicas      int32                  `json:"current_replicas"`                           // 当前副本数
	UpdatedReplicas      int32                  `json:"updated_replicas"`                           // 更新副本数
	ServiceName          string                 `json:"service_name"`                               // 服务名称
	UpdateStrategy       string                 `json:"update_strategy"`                            // 更新策略
	RevisionHistoryLimit int32                  `json:"revision_history_limit"`                     // 历史版本限制
	PodManagementPolicy  string                 `json:"pod_management_policy"`                      // Pod管理策略
	Selector             map[string]string      `json:"selector"`                                   // 选择器
	Labels               map[string]string      `json:"labels"`                                     // 标签
	Annotations          map[string]string      `json:"annotations"`                                // 注解
	Images               []string               `json:"images"`                                     // 容器镜像列表
	Status               K8sStatefulSetStatus   `json:"status"`                                     // StatefulSet状态
	Conditions           []StatefulSetCondition `json:"conditions"`                                 // StatefulSet条件
	CreatedAt            time.Time              `json:"created_at"`                                 // 创建时间
	UpdatedAt            time.Time              `json:"updated_at"`                                 // 更新时间
	RawStatefulSet       *appsv1.StatefulSet    `json:"-"`                                          // 原始 StatefulSet 对象，不序列化到 JSON
}

// StatefulSetCondition StatefulSet条件
type StatefulSetCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 条件状态
	LastUpdateTime     time.Time `json:"last_update_time"`     // 最后更新时间
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// StatefulSetSpec 创建/更新StatefulSet时的配置信息
type StatefulSetSpec struct {
	Replicas             *int32                            `json:"replicas"`                         // 副本数量
	Selector             *metav1.LabelSelector             `json:"selector"`                         // 标签选择器
	Template             *corev1.PodTemplateSpec           `json:"template"`                         // Pod模板
	VolumeClaimTemplates []corev1.PersistentVolumeClaim    `json:"volume_claim_templates,omitempty"` // 卷声明模板
	ServiceName          string                            `json:"service_name"`                     // 服务名称
	PodManagementPolicy  *appsv1.PodManagementPolicyType   `json:"pod_management_policy,omitempty"`  // Pod管理策略
	UpdateStrategy       *appsv1.StatefulSetUpdateStrategy `json:"update_strategy,omitempty"`        // 更新策略
	RevisionHistoryLimit *int32                            `json:"revision_history_limit,omitempty"` // 历史版本限制
	MinReadySeconds      *int32                            `json:"min_ready_seconds,omitempty"`      // 最小就绪时间
}

// K8sStatefulSetHistory StatefulSet版本历史
type K8sStatefulSetHistory struct {
	Revision int64     `json:"revision"` // 版本
	Date     time.Time `json:"date"`     // 日期
	Message  string    `json:"message"`  // 消息
}

// GetStatefulSetListReq 获取StatefulSet列表请求
type GetStatefulSetListReq struct {
	ListReq
	ClusterID   int               `json:"cluster_id" form:"cluster_id"`     // 集群ID
	Namespace   string            `json:"namespace" form:"namespace"`       // 命名空间
	Status      string            `json:"status" form:"status"`             // StatefulSet状态
	ServiceName string            `json:"service_name" form:"service_name"` // 服务名称
	Labels      map[string]string `json:"labels" form:"labels"`             // 标签
}

// GetStatefulSetDetailsReq 获取StatefulSet详情请求
type GetStatefulSetDetailsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // StatefulSet名称
}

// GetStatefulSetYamlReq 获取StatefulSet YAML请求
type GetStatefulSetYamlReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // StatefulSet名称
}

// CreateStatefulSetReq 创建StatefulSet请求
type CreateStatefulSetReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required"`   // 集群ID
	Name        string            `json:"name" binding:"required"`         // StatefulSet名称
	Namespace   string            `json:"namespace" binding:"required"`    // 命名空间
	Replicas    int32             `json:"replicas" binding:"required"`     // 副本数量
	ServiceName string            `json:"service_name" binding:"required"` // 服务名称
	Images      []string          `json:"images" binding:"required"`       // 容器镜像列表
	Labels      map[string]string `json:"labels"`                          // 标签
	Annotations map[string]string `json:"annotations"`                     // 注解
	Spec        StatefulSetSpec   `json:"spec"`                            // StatefulSet规格
	YAML        string            `json:"yaml"`                            // YAML内容
}

// CreateStatefulSetByYamlReq 通过YAML创建StatefulSet请求
type CreateStatefulSetByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
}

// UpdateStatefulSetReq 更新StatefulSet请求
type UpdateStatefulSetReq struct {
	ClusterID   int               `json:"cluster_id"`   // 集群ID
	Name        string            `json:"name"`         // StatefulSet名称
	Namespace   string            `json:"namespace"`    // 命名空间
	Replicas    int32             `json:"replicas"`     // 副本数量
	ServiceName string            `json:"service_name"` // 服务名称
	Images      []string          `json:"images"`       // 容器镜像列表
	Labels      map[string]string `json:"labels"`       // 标签
	Annotations map[string]string `json:"annotations"`  // 注解
	Spec        StatefulSetSpec   `json:"spec"`         // StatefulSet规格
	YAML        string            `json:"yaml"`         // YAML内容
}

// UpdateStatefulSetByYamlReq 通过YAML更新StatefulSet请求
type UpdateStatefulSetByYamlReq struct {
	ClusterID int    `json:"cluster_id"`              // 集群ID
	Namespace string `json:"namespace"`               // 命名空间
	Name      string `json:"name"`                    // StatefulSet名称
	YAML      string `json:"yaml" binding:"required"` // YAML内容
}

// DeleteStatefulSetReq 删除StatefulSet请求
type DeleteStatefulSetReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // StatefulSet名称
}

// RestartStatefulSetReq 重启StatefulSet请求
type RestartStatefulSetReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // StatefulSet名称
}

// ScaleStatefulSetReq 伸缩StatefulSet请求
type ScaleStatefulSetReq struct {
	ClusterID int    `json:"cluster_id"`                  // 集群ID
	Namespace string `json:"namespace"`                   // 命名空间
	Name      string `json:"name"`                        // StatefulSet名称
	Replicas  int32  `json:"replicas" binding:"required"` // 副本数量
}

// GetStatefulSetPodsReq 获取StatefulSet下的Pod列表请求
type GetStatefulSetPodsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // StatefulSet名称
}

// GetStatefulSetHistoryReq 获取StatefulSet版本历史请求
type GetStatefulSetHistoryReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // StatefulSet名称
}

// RollbackStatefulSetReq 回滚StatefulSet请求
type RollbackStatefulSetReq struct {
	ClusterID int    `json:"cluster_id"`                  // 集群ID
	Namespace string `json:"namespace"`                   // 命名空间
	Name      string `json:"name"`                        // StatefulSet名称
	Revision  int64  `json:"revision" binding:"required"` // 回滚到的版本号
}
