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

// GetPVListReq 获取PV列表请求
type GetPVListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
	Status        string `json:"status" form:"status" comment:"PV状态过滤"`
	StorageClass  string `json:"storage_class" form:"storage_class" comment:"存储类过滤"`
	AccessMode    string `json:"access_mode" form:"access_mode" comment:"访问模式过滤"`
	VolumeType    string `json:"volume_type" form:"volume_type" comment:"卷类型过滤"`
	Page          int    `json:"page" form:"page" comment:"页码"`
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`
}

// GetPVDetailsReq 获取PV详情请求
type GetPVDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"PV名称"`
}

// GetPVYamlReq 获取PV YAML请求
type GetPVYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"PV名称"`
}

// CreatePVReq 创建PV请求
type CreatePVReq struct {
	ClusterID     int                    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name          string                 `json:"name" binding:"required" comment:"PV名称"`
	Capacity      string                 `json:"capacity" binding:"required" comment:"存储容量"`
	AccessModes   []string               `json:"access_modes" binding:"required" comment:"访问模式"`
	ReclaimPolicy string                 `json:"reclaim_policy" comment:"回收策略"`
	StorageClass  string                 `json:"storage_class" comment:"存储类"`
	VolumeMode    string                 `json:"volume_mode" comment:"卷模式"`
	VolumeSource  map[string]interface{} `json:"volume_source" binding:"required" comment:"卷源配置"`
	NodeAffinity  map[string]interface{} `json:"node_affinity" comment:"节点亲和性"`
	Labels        map[string]string      `json:"labels" comment:"标签"`
	Annotations   map[string]string      `json:"annotations" comment:"注解"`
}

// CreatePVByYamlReq 通过YAML创建PV请求
type CreatePVByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdatePVReq 更新PV请求
type UpdatePVReq struct {
	ClusterID     int                    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name          string                 `json:"name" binding:"required" comment:"PV名称"`
	Capacity      string                 `json:"capacity" comment:"存储容量"`
	AccessModes   []string               `json:"access_modes" comment:"访问模式"`
	ReclaimPolicy string                 `json:"reclaim_policy" comment:"回收策略"`
	StorageClass  string                 `json:"storage_class" comment:"存储类"`
	VolumeMode    string                 `json:"volume_mode" comment:"卷模式"`
	VolumeSource  map[string]interface{} `json:"volume_source" comment:"卷源配置"`
	NodeAffinity  map[string]interface{} `json:"node_affinity" comment:"节点亲和性"`
	Labels        map[string]string      `json:"labels" comment:"标签"`
	Annotations   map[string]string      `json:"annotations" comment:"注解"`
}

// UpdatePVByYamlReq 通过YAML更新PV请求
type UpdatePVByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string `json:"name" binding:"required" comment:"PV名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeletePVReq 删除PV请求
type DeletePVReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name               string `json:"name" binding:"required" comment:"PV名称"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`
	Force              bool   `json:"force" comment:"是否强制删除"`
}

// ReclaimPVReq 回收PV请求
type ReclaimPVReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"PV名称"`
}

// K8sPV PersistentVolume主model
type K8sPV struct {
	Name              string                   `json:"name"`
	ClusterID         int                      `json:"cluster_id"`
	UID               string                   `json:"uid"`
	CreationTimestamp string                   `json:"creation_timestamp"`
	Labels            map[string]string        `json:"labels"`
	Annotations       map[string]string        `json:"annotations"`
	Capacity          string                   `json:"capacity"`
	AccessModes       []string                 `json:"access_modes"`
	ReclaimPolicy     string                   `json:"reclaim_policy"`
	StorageClass      string                   `json:"storage_class"`
	VolumeMode        string                   `json:"volume_mode"`
	Status            string                   `json:"status"`
	ClaimRef          map[string]string        `json:"claim_ref"`
	VolumeSource      map[string]interface{}   `json:"volume_source"`
	NodeAffinity      map[string]interface{}   `json:"node_affinity"`
	ResourceVersion   string                   `json:"resource_version"`
	Age               string                   `json:"age"`
	RawPV             *corev1.PersistentVolume `json:"-"` // 原始PV对象，不序列化
}
