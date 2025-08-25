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

// K8sConfigMapListRequest ConfigMap列表查询请求
type K8sConfigMapListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	DataKey       string `json:"data_key" form:"data_key" comment:"数据键过滤"`                       // 数据键过滤
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sConfigMapCreateRequest 创建ConfigMap请求
type K8sConfigMapCreateRequest struct {
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
type K8sConfigMapUpdateRequest struct {
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
type K8sConfigMapDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`      // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                        // 是否强制删除
}

// K8sConfigMapBatchDeleteRequest 批量删除ConfigMap请求
type K8sConfigMapBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"ConfigMap名称列表"` // ConfigMap名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`         // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                           // 是否强制删除
}

// K8sConfigMapDataRequest 获取ConfigMap数据请求
type K8sConfigMapDataRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
	Key       string `json:"key" comment:"数据键，为空则获取所有"`                     // 数据键，为空则获取所有
}

// K8sConfigMapEventRequest 获取ConfigMap事件请求
type K8sConfigMapEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                 // 限制天数内的事件
}

// K8sConfigMapUsageRequest 获取ConfigMap使用情况请求
type K8sConfigMapUsageRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"ConfigMap名称"` // ConfigMap名称，必填
}

// K8sConfigMapBackupRequest 备份ConfigMap请求
type K8sConfigMapBackupRequest struct {
	ClusterID   int      `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace   string   `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	Names       []string `json:"names" binding:"required" comment:"ConfigMap名称列表"` // ConfigMap名称列表，必填
	BackupName  string   `json:"backup_name" binding:"required" comment:"备份名称"`    // 备份名称，必填
	Description string   `json:"description" comment:"备份描述"`                       // 备份描述
}
