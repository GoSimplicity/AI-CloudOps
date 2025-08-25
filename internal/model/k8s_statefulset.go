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

// K8sStatefulSetEntity Kubernetes StatefulSet数据库实体
type K8sStatefulSetEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:StatefulSet名称"` // StatefulSet名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"`   // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                             // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:StatefulSet UID"`                                 // StatefulSet UID
	Replicas          int32             `json:"replicas" gorm:"comment:期望副本数"`                                               // 期望副本数
	ReadyReplicas     int32             `json:"ready_replicas" gorm:"comment:就绪副本数"`                                         // 就绪副本数
	CurrentReplicas   int32             `json:"current_replicas" gorm:"comment:当前副本数"`                                       // 当前副本数
	UpdatedReplicas   int32             `json:"updated_replicas" gorm:"comment:更新副本数"`                                       // 更新副本数
	ServiceName       string            `json:"service_name" gorm:"size:200;comment:关联服务名称"`                                 // 关联服务名称
	UpdateStrategy    string            `json:"update_strategy" gorm:"size:50;comment:更新策略"`                                 // 更新策略
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                          // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                     // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                            // Kubernetes创建时间
	Images            []string          `json:"images" gorm:"type:text;serializer:json;comment:容器镜像列表"`                      // 容器镜像列表
	Age               string            `json:"age" gorm:"-"`                                                                // 存在时间，前端计算使用
	Status            string            `json:"status" gorm:"-"`                                                             // StatefulSet状态，前端计算使用
}

func (k *K8sStatefulSetEntity) TableName() string {
	return "cl_k8s_statefulsets"
}

// K8sStatefulSetListRequest StatefulSet列表查询请求
type K8sStatefulSetListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sStatefulSetCreateRequest 创建StatefulSet请求
type K8sStatefulSetCreateRequest struct {
	ClusterID            int                             `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace            string                          `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name                 string                          `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	Replicas             int32                           `json:"replicas" comment:"副本数量"`                         // 副本数量
	ServiceName          string                          `json:"service_name" binding:"required" comment:"服务名称"`  // 服务名称，必填
	Image                string                          `json:"image" binding:"required" comment:"镜像地址"`         // 镜像地址，必填
	Ports                []ContainerPort                 `json:"ports" comment:"容器端口"`                            // 容器端口
	Env                  []EnvVar                        `json:"env" comment:"环境变量"`                              // 环境变量
	Resources            ResourceRequirements            `json:"resources" comment:"资源限制"`                        // 资源限制
	VolumeClaimTemplates []PersistentVolumeClaimTemplate `json:"volume_claim_templates" comment:"存储卷声明模板"`        // 存储卷声明模板
	Labels               map[string]string               `json:"labels" comment:"标签"`                             // 标签
	Annotations          map[string]string               `json:"annotations" comment:"注解"`                        // 注解
	UpdateStrategy       StatefulSetUpdateStrategy       `json:"update_strategy" comment:"更新策略"`                  // 更新策略
	StatefulSetYaml      *appsv1.StatefulSet             `json:"statefulset_yaml" comment:"StatefulSet YAML对象"`   // StatefulSet YAML对象
}

// K8sStatefulSetUpdateRequest 更新StatefulSet请求
type K8sStatefulSetUpdateRequest struct {
	ClusterID            int                             `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace            string                          `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name                 string                          `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	Replicas             *int32                          `json:"replicas" comment:"副本数量"`                         // 副本数量
	Image                string                          `json:"image" comment:"镜像地址"`                            // 镜像地址
	Ports                []ContainerPort                 `json:"ports" comment:"容器端口"`                            // 容器端口
	Env                  []EnvVar                        `json:"env" comment:"环境变量"`                              // 环境变量
	Resources            ResourceRequirements            `json:"resources" comment:"资源限制"`                        // 资源限制
	VolumeClaimTemplates []PersistentVolumeClaimTemplate `json:"volume_claim_templates" comment:"存储卷声明模板"`        // 存储卷声明模板
	Labels               map[string]string               `json:"labels" comment:"标签"`                             // 标签
	Annotations          map[string]string               `json:"annotations" comment:"注解"`                        // 注解
	UpdateStrategy       StatefulSetUpdateStrategy       `json:"update_strategy" comment:"更新策略"`                  // 更新策略
	StatefulSetYaml      *appsv1.StatefulSet             `json:"statefulset_yaml" comment:"StatefulSet YAML对象"`   // StatefulSet YAML对象
}

// K8sStatefulSetDeleteRequest 删除StatefulSet请求
type K8sStatefulSetDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`        // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                          // 是否强制删除
}

// K8sStatefulSetBatchDeleteRequest 批量删除StatefulSet请求
type K8sStatefulSetBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`       // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`        // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"StatefulSet名称列表"` // StatefulSet名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`           // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                             // 是否强制删除
}

// K8sStatefulSetScaleRequest StatefulSet扩缩容请求
type K8sStatefulSetScaleRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"`  // StatefulSet名称，必填
	Replicas  int32  `json:"replicas" binding:"required,min=0" comment:"副本数量"` // 副本数量，必填且大等于0
}

// K8sStatefulSetRestartRequest 重启StatefulSet请求
type K8sStatefulSetRestartRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
}

// K8sStatefulSetBatchRestartRequest 批量重启StatefulSet请求
type K8sStatefulSetBatchRestartRequest struct {
	ClusterID int      `json:"cluster_id" binding:"required" comment:"集群ID"`       // 集群ID，必填
	Namespace string   `json:"namespace" binding:"required" comment:"命名空间"`        // 命名空间，必填
	Names     []string `json:"names" binding:"required" comment:"StatefulSet名称列表"` // StatefulSet名称列表，必填
}

// K8sStatefulSetHistoryRequest 获取StatefulSet历史版本请求
type K8sStatefulSetHistoryRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
}

// K8sStatefulSetEventRequest 获取StatefulSet事件请求
type K8sStatefulSetEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                   // 限制天数内的事件
}
