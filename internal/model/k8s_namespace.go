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
)

// K8sNamespaceEntity Kubernetes 命名空间数据库实体
type K8sNamespaceEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:命名空间名称"`   // 命名空间名称
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                        // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:命名空间UID"`                                    // 命名空间UID
	Status            string            `json:"status" gorm:"size:50;comment:命名空间状态"`                                   // 命名空间状态
	Phase             string            `json:"phase" gorm:"size:50;comment:命名空间阶段"`                                    // 命名空间阶段
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                     // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                       // Kubernetes创建时间
	ResourceQuota     *ResourceQuota    `json:"resource_quota,omitempty" gorm:"type:text;serializer:json;comment:资源配额"` // 资源配额
	Age               string            `json:"age" gorm:"-"`                                                           // 存在时间，前端计算使用
}

func (k *K8sNamespaceEntity) TableName() string {
	return "cl_k8s_namespaces"
}

// K8sNamespaceListRequest 命名空间列表查询请求
type K8sNamespaceListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sNamespaceCreateRequest 创建命名空间请求
type K8sNamespaceCreateRequest struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`           // 集群ID，必填
	Name        string            `json:"name" binding:"required,min=1,max=200" comment:"命名空间名称"` // 命名空间名称，必填
	Labels      map[string]string `json:"labels" comment:"标签"`                                    // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                               // 注解
}

// K8sNamespaceUpdateRequest 更新命名空间请求
type K8sNamespaceUpdateRequest struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name        string            `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
}

// K8sNamespaceDeleteRequest 删除命名空间请求
type K8sNamespaceDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name               string `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sNamespaceBatchDeleteRequest 批量删除命名空间请求
type K8sNamespaceBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Names              []string `json:"names" binding:"required" comment:"命名空间名称列表"`  // 命名空间名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sNamespaceResourceRequest 获取命名空间资源请求
type K8sNamespaceResourceRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
}

// K8sNamespaceQuotaRequest 设置命名空间资源配额请求
type K8sNamespaceQuotaRequest struct {
	ClusterID     int            `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Name          string         `json:"name" binding:"required" comment:"命名空间名称"`         // 命名空间名称，必填
	ResourceQuota *ResourceQuota `json:"resource_quota" binding:"required" comment:"资源配额"` // 资源配额，必填
}

// K8sNamespaceEventRequest 获取命名空间事件请求
type K8sNamespaceEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}
