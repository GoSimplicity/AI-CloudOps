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
	Name               string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:DaemonSet名称"` // DaemonSet名称
	Namespace          string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID          int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID                string            `json:"uid" gorm:"size:100;comment:DaemonSet UID"`                                 // DaemonSet UID
	DesiredNumber      int32             `json:"desired_number" gorm:"comment:期望数量"`                                        // 期望数量
	CurrentNumber      int32             `json:"current_number" gorm:"comment:当前数量"`                                        // 当前数量
	ReadyNumber        int32             `json:"ready_number" gorm:"comment:就绪数量"`                                          // 就绪数量
	UpdatedNumber      int32             `json:"updated_number" gorm:"comment:更新数量"`                                        // 更新数量
	AvailableNumber    int32             `json:"available_number" gorm:"comment:可用数量"`                                      // 可用数量
	MisscheduledNumber int32             `json:"misscheduled_number" gorm:"comment:错误调度数量"`                                 // 错误调度数量
	UpdateStrategy     string            `json:"update_strategy" gorm:"size:50;comment:更新策略"`                               // 更新策略
	Labels             map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations        map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp  time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Images             []string          `json:"images" gorm:"type:text;serializer:json;comment:容器镜像列表"`                    // 容器镜像列表
	Age                string            `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	Status             string            `json:"status" gorm:"-"`                                                           // DaemonSet状态，前端计算使用
}

func (k *K8sDaemonSetEntity) TableName() string {
	return "cl_k8s_daemonsets"
}

// K8sDaemonSetListRequest DaemonSet列表查询请求
type K8sDaemonSetListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	NodeName      string `json:"node_name" form:"node_name" comment:"节点名称过滤"`                    // 节点名称过滤
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sDaemonSetCreateRequest 创建DaemonSet请求
type K8sDaemonSetCreateRequest struct {
	ClusterID      int                     `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace      string                  `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name           string                  `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	Image          string                  `json:"image" binding:"required" comment:"镜像地址"`       // 镜像地址，必填
	Ports          []ContainerPort         `json:"ports" comment:"容器端口"`                          // 容器端口
	Env            []EnvVar                `json:"env" comment:"环境变量"`                            // 环境变量
	Resources      ResourceRequirements    `json:"resources" comment:"资源限制"`                      // 资源限制
	NodeSelector   map[string]string       `json:"node_selector" comment:"节点选择器"`                 // 节点选择器
	Tolerations    []Toleration            `json:"tolerations" comment:"容忍度"`                     // 容忍度
	Labels         map[string]string       `json:"labels" comment:"标签"`                           // 标签
	Annotations    map[string]string       `json:"annotations" comment:"注解"`                      // 注解
	UpdateStrategy DaemonSetUpdateStrategy `json:"update_strategy" comment:"更新策略"`                // 更新策略
	DaemonSetYaml  *appsv1.DaemonSet       `json:"daemonset_yaml" comment:"DaemonSet YAML对象"`     // DaemonSet YAML对象
}

// K8sDaemonSetUpdateRequest 更新DaemonSet请求
type K8sDaemonSetUpdateRequest struct {
	ClusterID      int                     `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace      string                  `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name           string                  `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	Image          string                  `json:"image" comment:"镜像地址"`                          // 镜像地址
	Ports          []ContainerPort         `json:"ports" comment:"容器端口"`                          // 容器端口
	Env            []EnvVar                `json:"env" comment:"环境变量"`                            // 环境变量
	Resources      ResourceRequirements    `json:"resources" comment:"资源限制"`                      // 资源限制
	NodeSelector   map[string]string       `json:"node_selector" comment:"节点选择器"`                 // 节点选择器
	Tolerations    []Toleration            `json:"tolerations" comment:"容忍度"`                     // 容忍度
	Labels         map[string]string       `json:"labels" comment:"标签"`                           // 标签
	Annotations    map[string]string       `json:"annotations" comment:"注解"`                      // 注解
	UpdateStrategy DaemonSetUpdateStrategy `json:"update_strategy" comment:"更新策略"`                // 更新策略
	DaemonSetYaml  *appsv1.DaemonSet       `json:"daemonset_yaml" comment:"DaemonSet YAML对象"`     // DaemonSet YAML对象
}

// K8sDaemonSetDeleteRequest 删除DaemonSet请求
type K8sDaemonSetDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`      // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                        // 是否强制删除
}

// K8sDaemonSetBatchDeleteRequest 批量删除DaemonSet请求
type K8sDaemonSetBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"DaemonSet名称列表"` // DaemonSet名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`         // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                           // 是否强制删除
}

// K8sDaemonSetRestartRequest 重启DaemonSet请求
type K8sDaemonSetRestartRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
}

// K8sDaemonSetBatchRestartRequest 批量重启DaemonSet请求
type K8sDaemonSetBatchRestartRequest struct {
	ClusterID int      `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace string   `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Names     []string `json:"names" binding:"required" comment:"DaemonSet名称列表"` // DaemonSet名称列表，必填
}

// K8sDaemonSetHistoryRequest 获取DaemonSet历史版本请求
type K8sDaemonSetHistoryRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
}

// K8sDaemonSetEventRequest 获取DaemonSet事件请求
type K8sDaemonSetEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                 // 限制天数内的事件
}

// K8sDaemonSetNodePodsRequest 获取DaemonSet在指定节点的Pod请求
type K8sDaemonSetNodePodsRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"DaemonSet名称"` // DaemonSet名称，必填
	NodeName  string `json:"node_name" binding:"required" comment:"节点名称"`   // 节点名称，必填
}
