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



// K8sDeploymentEntity Kubernetes Deployment数据库实体
type K8sDeploymentEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:Deployment名称"` // Deployment名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"`  // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                            // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:Deployment UID"`                                 // Deployment UID
	Replicas          int32             `json:"replicas" gorm:"comment:期望副本数"`                                              // 期望副本数
	ReadyReplicas     int32             `json:"ready_replicas" gorm:"comment:就绪副本数"`                                        // 就绪副本数
	AvailableReplicas int32             `json:"available_replicas" gorm:"comment:可用副本数"`                                    // 可用副本数
	UpdatedReplicas   int32             `json:"updated_replicas" gorm:"comment:更新副本数"`                                      // 更新副本数
	Strategy          string            `json:"strategy" gorm:"size:50;comment:部署策略"`                                       // 部署策略
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                         // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                    // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                           // Kubernetes创建时间
	Images            []string          `json:"images" gorm:"type:text;serializer:json;comment:容器镜像列表"`                     // 容器镜像列表
	Age               string            `json:"age" gorm:"-"`                                                               // 存在时间，前端计算使用
	Status            string            `json:"status" gorm:"-"`                                                            // 部署状态，前端计算使用
}

func (k *K8sDeploymentEntity) TableName() string {
	return "cl_k8s_deployments"
}

// K8sDeploymentListRequest Deployment列表查询请求
type K8sDeploymentListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sDeploymentCreateRequest 创建Deployment请求
type K8sDeploymentCreateReq struct {
	ClusterID      int                  `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace      string               `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name           string               `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
	Replicas       int32                `json:"replicas" comment:"副本数量"`                        // 副本数量
	Image          string               `json:"image" binding:"required" comment:"镜像地址"`        // 镜像地址，必填
	Ports          []ContainerPort      `json:"ports" comment:"容器端口"`                           // 容器端口
	Env            []EnvVar             `json:"env" comment:"环境变量"`                             // 环境变量
	Resources      ResourceRequirements `json:"resources" comment:"资源限制"`                       // 资源限制
	Labels         map[string]string    `json:"labels" comment:"标签"`                            // 标签
	Annotations    map[string]string    `json:"annotations" comment:"注解"`                       // 注解
	Strategy       DeploymentStrategy   `json:"strategy" comment:"部署策略"`                        // 部署策略
	DeploymentYaml *appsv1.Deployment   `json:"deployment_yaml" comment:"Deployment YAML对象"`    // Deployment YAML对象
}

// K8sDeploymentUpdateRequest 更新Deployment请求
type K8sDeploymentUpdateReq struct {
	ClusterID      int                  `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace      string               `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name           string               `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
	Replicas       *int32               `json:"replicas" comment:"副本数量"`                        // 副本数量
	Image          string               `json:"image" comment:"镜像地址"`                           // 镜像地址
	Ports          []ContainerPort      `json:"ports" comment:"容器端口"`                           // 容器端口
	Env            []EnvVar             `json:"env" comment:"环境变量"`                             // 环境变量
	Resources      ResourceRequirements `json:"resources" comment:"资源限制"`                       // 资源限制
	Labels         map[string]string    `json:"labels" comment:"标签"`                            // 标签
	Annotations    map[string]string    `json:"annotations" comment:"注解"`                       // 注解
	Strategy       DeploymentStrategy   `json:"strategy" comment:"部署策略"`                        // 部署策略
	DeploymentYaml *appsv1.Deployment   `json:"deployment_yaml" comment:"Deployment YAML对象"`    // Deployment YAML对象
}

// K8sDeploymentDeleteRequest 删除Deployment请求
type K8sDeploymentDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`       // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                         // 是否强制删除
}

// K8sDeploymentBatchDeleteRequest 批量删除Deployment请求
type K8sDeploymentBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`      // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`       // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"Deployment名称列表"` // Deployment名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`          // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                            // 是否强制删除
}

// K8sDeploymentScaleRequest Deployment扩缩容请求
type K8sDeploymentScaleReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Deployment名称"`   // Deployment名称，必填
	Replicas  int32  `json:"replicas" binding:"required,min=0" comment:"副本数量"` // 副本数量，必填且大等于0
}

// K8sDeploymentRestartRequest 重启Deployment请求
type K8sDeploymentRestartReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
}

// K8sDeploymentBatchRestartRequest 批量重启Deployment请求
type K8sDeploymentBatchRestartReq struct {
	ClusterID int      `json:"cluster_id" binding:"required" comment:"集群ID"`      // 集群ID，必填
	Namespace string   `json:"namespace" binding:"required" comment:"命名空间"`       // 命名空间，必填
	Names     []string `json:"names" binding:"required" comment:"Deployment名称列表"` // Deployment名称列表，必填
}

// K8sDeploymentRollbackRequest Deployment回滚请求
type K8sDeploymentRollbackReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
	Revision  int64  `json:"revision" comment:"回滚到的版本，不指定则回滚到前一版本"`          // 回滚到的版本
}

// K8sDeploymentHistoryRequest 获取Deployment历史版本请求
type K8sDeploymentHistoryReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
}

// K8sDeploymentEventRequest 获取Deployment事件请求
type K8sDeploymentEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Deployment名称"` // Deployment名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                  // 限制天数内的事件
}

// ====================== Deployment响应实体 ======================

// DeploymentEntity Deployment响应实体
type DeploymentEntity struct {
	Name                string                      `json:"name"`                 // Deployment名称
	Namespace           string                      `json:"namespace"`            // 命名空间
	UID                 string                      `json:"uid"`                  // Deployment UID
	Labels              map[string]string           `json:"labels"`               // 标签
	Annotations         map[string]string           `json:"annotations"`          // 注解
	Replicas            int32                       `json:"replicas"`             // 期望副本数
	ReadyReplicas       int32                       `json:"ready_replicas"`       // 就绪副本数
	AvailableReplicas   int32                       `json:"available_replicas"`   // 可用副本数
	UpdatedReplicas     int32                       `json:"updated_replicas"`     // 更新副本数
	UnavailableReplicas int32                       `json:"unavailable_replicas"` // 不可用副本数
	Strategy            DeploymentStrategyEntity    `json:"strategy"`             // 部署策略
	Conditions          []DeploymentConditionEntity `json:"conditions"`           // 部署条件
	Selector            map[string]string           `json:"selector"`             // 选择器
	Images              []string                    `json:"images"`               // 镜像列表
	Status              string                      `json:"status"`               // 部署状态
	Age                 string                      `json:"age"`                  // 存在时间
	CreatedAt           string                      `json:"created_at"`           // 创建时间
}

// DeploymentStrategyEntity 部署策略实体
type DeploymentStrategyEntity struct {
	Type          string                         `json:"type"`           // 策略类型
	RollingUpdate *DeploymentRollingUpdateEntity `json:"rolling_update"` // 滚动更新配置
}

// DeploymentRollingUpdateEntity 滚动更新配置
type DeploymentRollingUpdateEntity struct {
	MaxUnavailable string `json:"max_unavailable"` // 最大不可用数量
	MaxSurge       string `json:"max_surge"`       // 最大超出数量
}

// DeploymentConditionEntity 部署条件
type DeploymentConditionEntity struct {
	Type               string `json:"type"`                 // 条件类型
	Status             string `json:"status"`               // 条件状态
	LastUpdateTime     string `json:"last_update_time"`     // 最后更新时间
	LastTransitionTime string `json:"last_transition_time"` // 最后转换时间
	Reason             string `json:"reason"`               // 原因
	Message            string `json:"message"`              // 消息
}

// DeploymentListResponse Deployment列表响应
type DeploymentListResponse struct {
	Items      []DeploymentEntity `json:"items"`       // Deployment列表
	TotalCount int                `json:"total_count"` // 总数
}

// DeploymentDetailResponse Deployment详情响应
type DeploymentDetailResponse struct {
	Deployment  DeploymentEntity          `json:"deployment"`   // Deployment信息
	YAML        string                    `json:"yaml"`         // YAML内容
	Events      []DeploymentEventEntity   `json:"events"`       // 事件列表
	Pods        []PodEntity               `json:"pods"`         // 关联Pod列表
	ReplicaSets []ReplicaSetEntity        `json:"replica_sets"` // 关联ReplicaSet列表
	History     []DeploymentHistoryEntity `json:"history"`      // 部署历史
}

// DeploymentEventEntity Deployment事件实体
type DeploymentEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// ReplicaSetEntity ReplicaSet实体
type ReplicaSetEntity struct {
	Name              string            `json:"name"`               // ReplicaSet名称
	Namespace         string            `json:"namespace"`          // 命名空间
	UID               string            `json:"uid"`                // ReplicaSet UID
	Labels            map[string]string `json:"labels"`             // 标签
	Annotations       map[string]string `json:"annotations"`        // 注解
	Replicas          int32             `json:"replicas"`           // 期望副本数
	ReadyReplicas     int32             `json:"ready_replicas"`     // 就绪副本数
	AvailableReplicas int32             `json:"available_replicas"` // 可用副本数
	Images            []string          `json:"images"`             // 镜像列表
	Status            string            `json:"status"`             // 状态
	Age               string            `json:"age"`                // 存在时间
	CreatedAt         string            `json:"created_at"`         // 创建时间
}

// DeploymentHistoryEntity 部署历史实体
type DeploymentHistoryEntity struct {
	Revision   int64    `json:"revision"`    // 版本号
	ChangeInfo string   `json:"change_info"` // 变更信息
	CreatedAt  string   `json:"created_at"`  // 创建时间
	Images     []string `json:"images"`      // 镜像列表
	Replicas   int32    `json:"replicas"`    // 副本数
}

// DeploymentScaleResponse 扩缩容响应
type DeploymentScaleResponse struct {
	Name             string `json:"name"`              // Deployment名称
	Namespace        string `json:"namespace"`         // 命名空间
	PreviousReplicas int32  `json:"previous_replicas"` // 之前副本数
	CurrentReplicas  int32  `json:"current_replicas"`  // 当前副本数
	Status           string `json:"status"`            // 扩缩容状态
}

// DeploymentRollbackResponse 回滚响应
type DeploymentRollbackResponse struct {
	Name             string `json:"name"`              // Deployment名称
	Namespace        string `json:"namespace"`         // 命名空间
	PreviousRevision int64  `json:"previous_revision"` // 之前版本
	CurrentRevision  int64  `json:"current_revision"`  // 当前版本
	Status           string `json:"status"`            // 回滚状态
}
