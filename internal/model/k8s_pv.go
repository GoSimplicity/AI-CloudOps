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

// K8sPVStatus PV状态枚举
type K8sPVStatus int8

const (
	K8sPVStatusAvailable K8sPVStatus = iota + 1 // 可用
	K8sPVStatusBound                            // 已绑定
	K8sPVStatusReleased                         // 已释放
	K8sPVStatusFailed                           // 失败
	K8sPVStatusUnknown                          // 未知
)

// K8sPV Kubernetes PersistentVolume
type K8sPV struct {
	Name            string                   `json:"name" binding:"required,min=1,max=200"` // PV名称
	ClusterID       int                      `json:"cluster_id" gorm:"index;not null"`      // 所属集群ID
	UID             string                   `json:"uid" gorm:"size:100"`                   // PV UID
	Capacity        string                   `json:"capacity"`                              // 存储容量
	AccessModes     []string                 `json:"access_modes"`                          // 访问模式
	ReclaimPolicy   string                   `json:"reclaim_policy"`                        // 回收策略
	StorageClass    string                   `json:"storage_class"`                         // 存储类
	VolumeMode      string                   `json:"volume_mode"`                           // 卷模式
	Status          K8sPVStatus              `json:"status"`                                // PV状态
	ClaimRef        map[string]string        `json:"claim_ref"`                             // 绑定的PVC信息
	VolumeSource    map[string]interface{}   `json:"volume_source"`                         // 卷源配置
	NodeAffinity    map[string]interface{}   `json:"node_affinity"`                         // 节点亲和性
	Labels          map[string]string        `json:"labels"`                                // 标签
	Annotations     map[string]string        `json:"annotations"`                           // 注解
	ResourceVersion string                   `json:"resource_version"`                      // 资源版本
	CreatedAt       time.Time                `json:"created_at"`                            // 创建时间
	Age             string                   `json:"age"`                                   // 存活时长
	RawPV           *corev1.PersistentVolume `json:"-"`                                     // 原始PV对象，不序列化
}

// GetPVListReq 获取PV列表请求
type GetPVListReq struct {
	ListReq
	ClusterID  int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Status     string `json:"status" form:"status" comment:"PV状态过滤"`                          // PV状态过滤
	AccessMode string `json:"access_mode" form:"access_mode" comment:"访问模式过滤"`                // 访问模式过滤
	VolumeType string `json:"volume_type" form:"volume_type" comment:"卷类型过滤"`                 // 卷类型过滤
}

// GetPVDetailsReq 获取PV详情请求
type GetPVDetailsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name      string `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
}

// GetPVYamlReq 获取PV YAML请求
type GetPVYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name      string `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
}

// CreatePVReq 创建PV请求
type CreatePVReq struct {
	ClusterID     int                    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name          string                 `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
	Capacity      string                 `json:"capacity" binding:"required" comment:"存储容量"`                     // 存储容量
	AccessModes   []string               `json:"access_modes" binding:"required" comment:"访问模式"`                 // 访问模式
	ReclaimPolicy string                 `json:"reclaim_policy" comment:"回收策略"`                                  // 回收策略
	StorageClass  string                 `json:"storage_class" comment:"存储类"`                                    // 存储类
	VolumeMode    string                 `json:"volume_mode" comment:"卷模式"`                                      // 卷模式
	VolumeSource  map[string]interface{} `json:"volume_source" binding:"required" comment:"卷源配置"`                // 卷源配置
	NodeAffinity  map[string]interface{} `json:"node_affinity" comment:"节点亲和性"`                                  // 节点亲和性
	Labels        map[string]string      `json:"labels" comment:"标签"`                                            // 标签
	Annotations   map[string]string      `json:"annotations" comment:"注解"`                                       // 注解
}

// CreatePVByYamlReq 通过YAML创建PV请求
type CreatePVByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// UpdatePVReq 更新PV请求
type UpdatePVReq struct {
	ClusterID     int                    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name          string                 `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
	Capacity      string                 `json:"capacity" comment:"存储容量"`                                        // 存储容量
	AccessModes   []string               `json:"access_modes" comment:"访问模式"`                                    // 访问模式
	ReclaimPolicy string                 `json:"reclaim_policy" comment:"回收策略"`                                  // 回收策略
	StorageClass  string                 `json:"storage_class" comment:"存储类"`                                    // 存储类
	VolumeMode    string                 `json:"volume_mode" comment:"卷模式"`                                      // 卷模式
	VolumeSource  map[string]interface{} `json:"volume_source" comment:"卷源配置"`                                   // 卷源配置
	NodeAffinity  map[string]interface{} `json:"node_affinity" comment:"节点亲和性"`                                  // 节点亲和性
	Labels        map[string]string      `json:"labels" comment:"标签"`                                            // 标签
	Annotations   map[string]string      `json:"annotations" comment:"注解"`                                       // 注解
}

// UpdatePVByYamlReq 通过YAML更新PV请求
type UpdatePVByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name      string `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// DeletePVReq 删除PV请求
type DeletePVReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name               string `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`                       // 优雅删除时间（秒）
	Force              bool   `json:"force" comment:"是否强制删除"`                                         // 是否强制删除
}

// ReclaimPVReq 回收PV请求
type ReclaimPVReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name      string `json:"name" form:"name" binding:"required" comment:"PV名称"`             // PV名称
}
