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
	corev1 "k8s.io/api/core/v1"
)

// GetPVCListReq 获取PVC列表请求
type GetPVCListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
	Status        string `json:"status" form:"status" comment:"PVC状态过滤"`
	StorageClass  string `json:"storage_class" form:"storage_class" comment:"存储类过滤"`
	AccessMode    string `json:"access_mode" form:"access_mode" comment:"访问模式过滤"`
	VolumeName    string `json:"volume_name" form:"volume_name" comment:"PV名称过滤"`
	Page          int    `json:"page" form:"page" comment:"页码"`
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`
}

// GetPVCDetailsReq 获取PVC详情请求
type GetPVCDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"PVC名称"`
}

// GetPVCYamlReq 获取PVC YAML请求
type GetPVCYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"PVC名称"`
}

// CreatePVCReq 创建PVC请求
type CreatePVCReq struct {
	ClusterID      int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace      string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name           string            `json:"name" binding:"required" comment:"PVC名称"`
	RequestStorage string            `json:"request_storage" binding:"required" comment:"请求存储"`
	AccessModes    []string          `json:"access_modes" binding:"required" comment:"访问模式"`
	StorageClass   string            `json:"storage_class" comment:"存储类"`
	VolumeMode     string            `json:"volume_mode" comment:"卷模式"`
	VolumeName     string            `json:"volume_name" comment:"指定PV名称"`
	Selector       map[string]string `json:"selector" comment:"选择器"`
	Labels         map[string]string `json:"labels" comment:"标签"`
	Annotations    map[string]string `json:"annotations" comment:"注解"`
}

// CreatePVCByYamlReq 通过YAML创建PVC请求
type CreatePVCByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdatePVCReq 更新PVC请求
type UpdatePVCReq struct {
	ClusterID      int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace      string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name           string            `json:"name" binding:"required" comment:"PVC名称"`
	RequestStorage string            `json:"request_storage" comment:"请求存储"`
	AccessModes    []string          `json:"access_modes" comment:"访问模式"`
	StorageClass   string            `json:"storage_class" comment:"存储类"`
	VolumeMode     string            `json:"volume_mode" comment:"卷模式"`
	VolumeName     string            `json:"volume_name" comment:"指定PV名称"`
	Selector       map[string]string `json:"selector" comment:"选择器"`
	Labels         map[string]string `json:"labels" comment:"标签"`
	Annotations    map[string]string `json:"annotations" comment:"注解"`
}

// UpdatePVCByYamlReq 通过YAML更新PVC请求
type UpdatePVCByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"PVC名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeletePVCReq 删除PVC请求
type DeletePVCReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`
	Name               string `json:"name" binding:"required" comment:"PVC名称"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`
	Force              bool   `json:"force" comment:"是否强制删除"`
}

// ExpandPVCReq 扩容PVC请求
type ExpandPVCReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"PVC名称"`
	NewCapacity string `json:"new_capacity" binding:"required" comment:"新容量"`
}

// GetPVCPodsReq 获取使用PVC的Pod列表请求
type GetPVCPodsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"PVC名称"`
}

// K8sPVC PersistentVolumeClaim主model
type K8sPVC struct {
	Name              string                        `json:"name"`
	Namespace         string                        `json:"namespace"`
	ClusterID         int                           `json:"cluster_id"`
	UID               string                        `json:"uid"`
	CreationTimestamp string                        `json:"creation_timestamp"`
	Labels            map[string]string             `json:"labels"`
	Annotations       map[string]string             `json:"annotations"`
	Capacity          string                        `json:"capacity"`
	RequestStorage    string                        `json:"request_storage"`
	AccessModes       []string                      `json:"access_modes"`
	StorageClass      string                        `json:"storage_class"`
	VolumeMode        string                        `json:"volume_mode"`
	Status            string                        `json:"status"`
	VolumeName        string                        `json:"volume_name"`
	Selector          map[string]string             `json:"selector"`
	ResourceVersion   string                        `json:"resource_version"`
	Age               string                        `json:"age"`
	RawPVC            *corev1.PersistentVolumeClaim `json:"-"` // 原始PVC对象，不序列化
}
