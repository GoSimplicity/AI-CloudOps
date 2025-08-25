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

// K8sPersistentVolumeEntity Kubernetes PersistentVolume数据库实体
type K8sPersistentVolumeEntity struct {
	Model
	Name              string                     `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:PV名称"` // PV名称
	ClusterID         int                        `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                    // 所属集群ID
	UID               string                     `json:"uid" gorm:"size:100;comment:PV UID"`                                 // PV UID
	Capacity          string                     `json:"capacity" gorm:"size:50;comment:容量大小"`                               // 容量大小
	AccessModes       []string                   `json:"access_modes" gorm:"type:text;serializer:json;comment:访问模式"`         // 访问模式
	ReclaimPolicy     string                     `json:"reclaim_policy" gorm:"size:50;comment:回收策略"`                         // 回收策略
	Status            string                     `json:"status" gorm:"size:50;comment:PV状态"`                                 // PV状态
	StorageClass      string                     `json:"storage_class" gorm:"size:200;comment:存储类"`                          // 存储类
	VolumeSource      string                     `json:"volume_source" gorm:"size:100;comment:存储源类型"`                        // 存储源类型
	NodeAffinity      *corev1.VolumeNodeAffinity `json:"node_affinity" gorm:"type:text;serializer:json;comment:节点亲和性"`       // 节点亲和性
	MountOptions      []string                   `json:"mount_options" gorm:"type:text;serializer:json;comment:挂载选项"`        // 挂载选项
	Labels            map[string]string          `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                 // 标签
	Annotations       map[string]string          `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`            // 注解
	CreationTimestamp time.Time                  `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                   // Kubernetes创建时间
	Age               string                     `json:"age" gorm:"-"`                                                       // 存在时间，前端计算使用
	Claim             *PersistentVolumeClaimRef  `json:"claim" gorm:"type:text;serializer:json;comment:绑定的PVC"`              // 绑定的PVC
}

func (k *K8sPersistentVolumeEntity) TableName() string {
	return "cl_k8s_persistent_volumes"
}

// K8sPersistentVolumeListRequest PV列表查询请求
type K8sPersistentVolumeListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	StorageClass  string `json:"storage_class" form:"storage_class" comment:"存储类过滤"`             // 存储类过滤
	ReclaimPolicy string `json:"reclaim_policy" form:"reclaim_policy" comment:"回收策略过滤"`          // 回收策略过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sPersistentVolumeCreateRequest 创建PV请求
type K8sPersistentVolumeCreateRequest struct {
	ClusterID     int                           `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Name          string                        `json:"name" binding:"required" comment:"PV名称"`         // PV名称，必填
	Capacity      string                        `json:"capacity" binding:"required" comment:"容量大小"`     // 容量大小，必填
	AccessModes   []string                      `json:"access_modes" binding:"required" comment:"访问模式"` // 访问模式，必填
	ReclaimPolicy string                        `json:"reclaim_policy" comment:"回收策略"`                  // 回收策略
	StorageClass  string                        `json:"storage_class" comment:"存储类"`                    // 存储类
	VolumeSource  PersistentVolumeSourceRequest `json:"volume_source" binding:"required" comment:"存储源"` // 存储源，必填
	NodeAffinity  *VolumeNodeAffinityRequest    `json:"node_affinity" comment:"节点亲和性"`                  // 节点亲和性
	MountOptions  []string                      `json:"mount_options" comment:"挂载选项"`                   // 挂载选项
	Labels        map[string]string             `json:"labels" comment:"标签"`                            // 标签
	Annotations   map[string]string             `json:"annotations" comment:"注解"`                       // 注解
	PVYaml        *corev1.PersistentVolume      `json:"pv_yaml" comment:"PV YAML对象"`                    // PV YAML对象
}

// K8sPersistentVolumeUpdateRequest 更新PV请求
type K8sPersistentVolumeUpdateRequest struct {
	ClusterID     int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name          string            `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
	ReclaimPolicy string            `json:"reclaim_policy" comment:"回收策略"`                // 回收策略
	MountOptions  []string          `json:"mount_options" comment:"挂载选项"`                 // 挂载选项
	Labels        map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations   map[string]string `json:"annotations" comment:"注解"`                     // 注解
}

// K8sPersistentVolumeDeleteRequest 删除PV请求
type K8sPersistentVolumeDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name               string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPersistentVolumeBatchDeleteRequest 批量删除PV请求
type K8sPersistentVolumeBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Names              []string `json:"names" binding:"required" comment:"PV名称列表"`    // PV名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPersistentVolumeEventRequest 获取PV事件请求
type K8sPersistentVolumeEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sPersistentVolumeClaimRequest 获取PV绑定的PVC信息请求
type K8sPersistentVolumeClaimRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
}

// K8sPersistentVolumeReleaseRequest 释放PV请求
type K8sPersistentVolumeReleaseRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
}
