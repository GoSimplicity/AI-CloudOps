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

// K8sPersistentVolumeClaimEntity Kubernetes PersistentVolumeClaim数据库实体
type K8sPersistentVolumeClaimEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:PVC名称"`       // PVC名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:PVC UID"`                                       // PVC UID
	Status            string            `json:"status" gorm:"size:50;comment:PVC状态"`                                       // PVC状态
	Volume            string            `json:"volume" gorm:"size:200;comment:绑定的PV名称"`                                    // 绑定的PV名称
	Capacity          string            `json:"capacity" gorm:"size:50;comment:容量大小"`                                      // 容量大小
	AccessModes       []string          `json:"access_modes" gorm:"type:text;serializer:json;comment:访问模式"`                // 访问模式
	StorageClass      string            `json:"storage_class" gorm:"size:200;comment:存储类"`                                 // 存储类
	VolumeMode        string            `json:"volume_mode" gorm:"size:50;comment:存储模式"`                                   // 存储模式
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age               string            `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	UsedBy            []string          `json:"used_by" gorm:"-"`                                                          // 使用方Pod列表，前端使用
}

func (k *K8sPersistentVolumeClaimEntity) TableName() string {
	return "cl_k8s_persistent_volume_claims"
}

// K8sPersistentVolumeClaimListRequest PVC列表查询请求
type K8sPersistentVolumeClaimListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	StorageClass  string `json:"storage_class" form:"storage_class" comment:"存储类过滤"`             // 存储类过滤
	VolumeName    string `json:"volume_name" form:"volume_name" comment:"PV名称过滤"`                // PV名称过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sPersistentVolumeClaimCreateRequest 创建PVC请求
type K8sPersistentVolumeClaimCreateRequest struct {
	ClusterID    int                           `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace    string                        `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Name         string                        `json:"name" binding:"required" comment:"PVC名称"`        // PVC名称，必填
	AccessModes  []string                      `json:"access_modes" binding:"required" comment:"访问模式"` // 访问模式，必填
	Size         string                        `json:"size" binding:"required" comment:"存储大小"`         // 存储大小，必填
	StorageClass *string                       `json:"storage_class" comment:"存储类"`                    // 存储类
	VolumeMode   *string                       `json:"volume_mode" comment:"存储模式"`                     // 存储模式
	Selector     *LabelSelector                `json:"selector" comment:"标签选择器"`                       // 标签选择器
	Resources    ResourceRequirements          `json:"resources" comment:"资源需求"`                       // 资源需求
	Labels       map[string]string             `json:"labels" comment:"标签"`                            // 标签
	Annotations  map[string]string             `json:"annotations" comment:"注解"`                       // 注解
	PVCYaml      *corev1.PersistentVolumeClaim `json:"pvc_yaml" comment:"PVC YAML对象"`                  // PVC YAML对象
}

// K8sPersistentVolumeClaimUpdateRequest 更新PVC请求
type K8sPersistentVolumeClaimUpdateRequest struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name        string            `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	Size        string            `json:"size" comment:"存储大小"`                          // 存储大小
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
}

// K8sPersistentVolumeClaimDeleteRequest 删除PVC请求
type K8sPersistentVolumeClaimDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPersistentVolumeClaimBatchDeleteRequest 批量删除PVC请求
type K8sPersistentVolumeClaimBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"PVC名称列表"`   // PVC名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPersistentVolumeClaimEventRequest 获取PVC事件请求
type K8sPersistentVolumeClaimEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sPersistentVolumeClaimUsageRequest 获取PVC使用情况请求
type K8sPersistentVolumeClaimUsageRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
}

// K8sPersistentVolumeClaimExpandRequest PVC扩容请求
type K8sPersistentVolumeClaimExpandRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	NewSize   string `json:"new_size" binding:"required" comment:"新的存储大小"` // 新的存储大小，必填
}

// K8sPersistentVolumeClaimSnapshotRequest PVC快照请求
type K8sPersistentVolumeClaimSnapshotRequest struct {
	ClusterID    int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace    string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name         string `json:"name" binding:"required" comment:"PVC名称"`         // PVC名称，必填
	SnapshotName string `json:"snapshot_name" binding:"required" comment:"快照名称"` // 快照名称，必填
	Description  string `json:"description" comment:"快照描述"`                      // 快照描述
}
