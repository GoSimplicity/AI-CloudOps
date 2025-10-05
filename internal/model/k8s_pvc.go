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

// K8sPVCStatus PVC状态枚举
type K8sPVCStatus int8

const (
	K8sPVCStatusPending     K8sPVCStatus = iota + 1 // 等待中
	K8sPVCStatusBound                               // 已绑定
	K8sPVCStatusLost                                // 丢失
	K8sPVCStatusTerminating                         // 终止中
	K8sPVCStatusUnknown                             // 未知
)

// K8sPVC Kubernetes PersistentVolumeClaim
type K8sPVC struct {
	Name            string                        `json:"name" binding:"required,min=1,max=200"`      // PVC名称
	Namespace       string                        `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID       int                           `json:"cluster_id" gorm:"index;not null"`           // 所属集群ID
	UID             string                        `json:"uid" gorm:"size:100"`                        // PVC UID
	Capacity        string                        `json:"capacity"`                                   // 实际容量
	RequestStorage  string                        `json:"request_storage"`                            // 请求存储
	AccessModes     []string                      `json:"access_modes"`                               // 访问模式
	StorageClass    string                        `json:"storage_class"`                              // 存储类
	VolumeMode      string                        `json:"volume_mode"`                                // 卷模式
	Status          K8sPVCStatus                  `json:"status"`                                     // PVC状态
	VolumeName      string                        `json:"volume_name"`                                // 绑定的PV名称
	Selector        map[string]string             `json:"selector"`                                   // 选择器
	Labels          map[string]string             `json:"labels"`                                     // 标签
	Annotations     map[string]string             `json:"annotations"`                                // 注解
	ResourceVersion string                        `json:"resource_version"`                           // 资源版本
	CreatedAt       time.Time                     `json:"created_at"`                                 // 创建时间
	Age             string                        `json:"age"`                                        // 年龄
	RawPVC          *corev1.PersistentVolumeClaim `json:"-"`                                          // 原始PVC对象，不序列化到JSON
}

// PVCCondition PVC条件
type PVCCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 条件状态
	LastUpdateTime     time.Time `json:"last_update_time"`     // 最后更新时间
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// PVCSpec 创建/更新PVC时的配置信息
type PVCSpec struct {
	RequestStorage string            `json:"request_storage"` // 请求存储
	AccessModes    []string          `json:"access_modes"`    // 访问模式
	StorageClass   string            `json:"storage_class"`   // 存储类
	VolumeMode     string            `json:"volume_mode"`     // 卷模式
	VolumeName     string            `json:"volume_name"`     // 指定PV名称
	Selector       map[string]string `json:"selector"`        // 选择器
}

// GetPVCListReq 获取PVC列表请求
type GetPVCListReq struct {
	ListReq
	ClusterID  int               `json:"cluster_id" form:"cluster_id" comment:"集群ID"`   // 集群ID
	Namespace  string            `json:"namespace" form:"namespace" comment:"命名空间"`     // 命名空间
	Status     K8sPVCStatus      `json:"status" form:"status" comment:"PVC状态"`          // PVC状态 (0表示不过滤，1-5对应具体状态)
	Labels     map[string]string `json:"labels" form:"labels" comment:"标签"`             // 标签
	AccessMode string            `json:"access_mode" form:"access_mode" comment:"访问模式"` // 访问模式
}

// GetPVCDetailsReq 获取PVC详情请求
type GetPVCDetailsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
}

// GetPVCYamlReq 获取PVC YAML请求
type GetPVCYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
}

// CreatePVCReq 创建PVC请求
type CreatePVCReq struct {
	ClusterID   int               `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace   string            `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name        string            `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
	Labels      map[string]string `json:"labels" comment:"标签"`                                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                                       // 注解
	Spec        PVCSpec           `json:"spec" comment:"PVC规格"`                                           // PVC规格
}

// UpdatePVCReq 更新PVC请求
type UpdatePVCReq struct {
	ClusterID   int               `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace   string            `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name        string            `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
	Labels      map[string]string `json:"labels" comment:"标签"`                                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                                       // 注解
	Spec        PVCSpec           `json:"spec" comment:"PVC规格"`                                           // PVC规格
}

// CreatePVCByYamlReq 通过YAML创建PVC请求
type CreatePVCByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// UpdatePVCByYamlReq 通过YAML更新PVC请求
type UpdatePVCByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`                    // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// DeletePVCReq 删除PVC请求
type DeletePVCReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" binding:"required" comment:"PVC名称"`                        // PVC名称
}

// ExpandPVCReq 扩容PVC请求
type ExpandPVCReq struct {
	ClusterID   int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace   string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name        string `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
	NewCapacity string `json:"new_capacity" binding:"required" comment:"新容量"`                  // 新容量
}

// GetPVCPodsReq 获取使用PVC的Pod列表请求
type GetPVCPodsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"PVC名称"`            // PVC名称
}
