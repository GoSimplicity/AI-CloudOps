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

// K8sConfigMapEntity Kubernetes ConfigMap数据库实体
type K8sConfigMapEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:ConfigMap名称"` // ConfigMap名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:ConfigMap UID"`                                 // ConfigMap UID
	Data              map[string]string `json:"data" gorm:"type:text;serializer:json;comment:字符串数据"`                       // 字符串数据
	BinaryData        map[string][]byte `json:"binary_data" gorm:"type:text;serializer:json;comment:二进制数据"`                // 二进制数据
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age               string            `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	DataCount         int               `json:"data_count" gorm:"-"`                                                       // 数据条目数量，前端计算使用
	Size              string            `json:"size" gorm:"-"`                                                             // 数据大小，前端计算使用
}

func (k *K8sConfigMapEntity) TableName() string {
	return "cl_k8s_configmaps"
}

// ====================== 补充缺失的ConfigMap请求结构体 ======================

// GetConfigMapListReq 获取ConfigMap列表请求
type GetConfigMapListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	Name          string `json:"name" form:"name" comment:"ConfigMap名称过滤"`                       // ConfigMap名称过滤
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	DataKey       string `json:"data_key" form:"data_key" comment:"数据键过滤"`                       // 数据键过滤
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sConfigMapCreateRequest 创建ConfigMap请求
type K8sConfigMapCreateReq struct {
	ClusterID     int               `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace     string            `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name          string            `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
	Data          map[string]string `json:"data" comment:"字符串数据"`                          // 字符串数据
	BinaryData    map[string][]byte `json:"binary_data" comment:"二进制数据"`                   // 二进制数据
	Labels        map[string]string `json:"labels" comment:"标签"`                           // 标签
	Annotations   map[string]string `json:"annotations" comment:"注解"`                      // 注解
	ConfigMapYaml *corev1.ConfigMap `json:"configmap_yaml" comment:"ConfigMap YAML对象"`     // ConfigMap YAML对象
}

// K8sConfigMapUpdateRequest 更新ConfigMap请求
type K8sConfigMapUpdateReq struct {
	ClusterID     int               `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace     string            `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name          string            `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
	Data          map[string]string `json:"data" comment:"字符串数据"`                          // 字符串数据
	BinaryData    map[string][]byte `json:"binary_data" comment:"二进制数据"`                   // 二进制数据
	Labels        map[string]string `json:"labels" comment:"标签"`                           // 标签
	Annotations   map[string]string `json:"annotations" comment:"注解"`                      // 注解
	ConfigMapYaml *corev1.ConfigMap `json:"configmap_yaml" comment:"ConfigMap YAML对象"`     // ConfigMap YAML对象
}

// K8sConfigMapDeleteRequest 删除ConfigMap请求
type K8sConfigMapDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`      // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                        // 是否强制删除
}

// ====================== ConfigMap响应实体 ======================

// ConfigMapEntity ConfigMap响应实体
type ConfigMapEntity struct {
	Name        string            `json:"name"`        // ConfigMap名称
	Namespace   string            `json:"namespace"`   // 命名空间
	UID         string            `json:"uid"`         // ConfigMap UID
	Labels      map[string]string `json:"labels"`      // 标签
	Annotations map[string]string `json:"annotations"` // 注解
	Data        map[string]string `json:"data"`        // 字符串数据
	BinaryData  map[string][]byte `json:"binary_data"` // 二进制数据
	DataCount   int               `json:"data_count"`  // 数据条目数量
	Size        string            `json:"size"`        // 数据大小
	Immutable   bool              `json:"immutable"`   // 是否不可变
	Age         string            `json:"age"`         // 存在时间
	CreatedAt   string            `json:"created_at"`  // 创建时间
}

// ConfigMapListResponse ConfigMap列表响应
type ConfigMapListResponse struct {
	Items      []ConfigMapEntity `json:"items"`       // ConfigMap列表
	TotalCount int               `json:"total_count"` // 总数
}

// ConfigMapDetailResponse ConfigMap详情响应
type ConfigMapDetailResponse struct {
	ConfigMap ConfigMapEntity        `json:"config_map"` // ConfigMap信息
	YAML      string                 `json:"yaml"`       // YAML内容
	Events    []ConfigMapEventEntity `json:"events"`     // 事件列表
	Usage     ConfigMapUsageEntity   `json:"usage"`      // 使用情况
}

// ConfigMapEventEntity ConfigMap事件实体
type ConfigMapEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// ConfigMapUsageEntity ConfigMap使用情况实体
type ConfigMapUsageEntity struct {
	UsedByPods         []ConfigMapPodUsageEntity         `json:"used_by_pods"`         // 被Pod使用
	UsedByDeployments  []ConfigMapDeploymentUsageEntity  `json:"used_by_deployments"`  // 被Deployment使用
	UsedByStatefulSets []ConfigMapStatefulSetUsageEntity `json:"used_by_statefulsets"` // 被StatefulSet使用
	UsedByDaemonSets   []ConfigMapDaemonSetUsageEntity   `json:"used_by_daemonsets"`   // 被DaemonSet使用
	UsedByJobs         []ConfigMapJobUsageEntity         `json:"used_by_jobs"`         // 被Job使用
}

// ConfigMapPodUsageEntity Pod使用ConfigMap实体
type ConfigMapPodUsageEntity struct {
	PodName       string   `json:"pod_name"`       // Pod名称
	Namespace     string   `json:"namespace"`      // 命名空间
	UsageType     string   `json:"usage_type"`     // 使用类型(volume/env)
	MountPath     string   `json:"mount_path"`     // 挂载路径
	Keys          []string `json:"keys"`           // 使用的键
	ContainerName string   `json:"container_name"` // 容器名称
}

// ConfigMapDeploymentUsageEntity Deployment使用ConfigMap实体
type ConfigMapDeploymentUsageEntity struct {
	DeploymentName string   `json:"deployment_name"` // Deployment名称
	Namespace      string   `json:"namespace"`       // 命名空间
	UsageType      string   `json:"usage_type"`      // 使用类型
	MountPath      string   `json:"mount_path"`      // 挂载路径
	Keys           []string `json:"keys"`            // 使用的键
	ContainerName  string   `json:"container_name"`  // 容器名称
}

// ConfigMapStatefulSetUsageEntity StatefulSet使用ConfigMap实体
type ConfigMapStatefulSetUsageEntity struct {
	StatefulSetName string   `json:"statefulset_name"` // StatefulSet名称
	Namespace       string   `json:"namespace"`        // 命名空间
	UsageType       string   `json:"usage_type"`       // 使用类型
	MountPath       string   `json:"mount_path"`       // 挂载路径
	Keys            []string `json:"keys"`             // 使用的键
	ContainerName   string   `json:"container_name"`   // 容器名称
}

// ConfigMapDaemonSetUsageEntity DaemonSet使用ConfigMap实体
type ConfigMapDaemonSetUsageEntity struct {
	DaemonSetName string   `json:"daemonset_name"` // DaemonSet名称
	Namespace     string   `json:"namespace"`      // 命名空间
	UsageType     string   `json:"usage_type"`     // 使用类型
	MountPath     string   `json:"mount_path"`     // 挂载路径
	Keys          []string `json:"keys"`           // 使用的键
	ContainerName string   `json:"container_name"` // 容器名称
}

// ConfigMapJobUsageEntity Job使用ConfigMap实体
type ConfigMapJobUsageEntity struct {
	JobName       string   `json:"job_name"`       // Job名称
	Namespace     string   `json:"namespace"`      // 命名空间
	UsageType     string   `json:"usage_type"`     // 使用类型
	MountPath     string   `json:"mount_path"`     // 挂载路径
	Keys          []string `json:"keys"`           // 使用的键
	ContainerName string   `json:"container_name"` // 容器名称
}

// ConfigMapDataResponse ConfigMap数据响应
type ConfigMapDataResponse struct {
	Name       string            `json:"name"`        // ConfigMap名称
	Namespace  string            `json:"namespace"`   // 命名空间
	Data       map[string]string `json:"data"`        // 字符串数据
	BinaryData map[string][]byte `json:"binary_data"` // 二进制数据
	DataCount  int               `json:"data_count"`  // 数据条目数量
	Size       string            `json:"size"`        // 数据大小
}

// ConfigMapBackupResponse ConfigMap备份响应
type ConfigMapBackupResponse struct {
	BackupName     string   `json:"backup_name"`     // 备份名称
	ClusterID      int      `json:"cluster_id"`      // 集群ID
	Namespace      string   `json:"namespace"`       // 命名空间
	ConfigMapNames []string `json:"configmap_names"` // ConfigMap名称列表
	BackupPath     string   `json:"backup_path"`     // 备份路径
	Size           string   `json:"size"`            // 备份大小
	Status         string   `json:"status"`          // 备份状态
	Message        string   `json:"message"`         // 备份消息
	CreatedAt      string   `json:"created_at"`      // 创建时间
}
