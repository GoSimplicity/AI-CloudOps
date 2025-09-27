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

// K8sConfigMap Kubernetes ConfigMap
type K8sConfigMap struct {
	Name         string            `json:"name" binding:"required,min=1,max=200"`      // ConfigMap名称
	Namespace    string            `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID    int               `json:"cluster_id" gorm:"index;not null"`           // 所属集群ID
	UID          string            `json:"uid" gorm:"size:100"`                        // ConfigMap UID
	Data         map[string]string `json:"data"`                                       // 字符串数据
	BinaryData   map[string][]byte `json:"binary_data"`                                // 二进制数据
	Labels       map[string]string `json:"labels"`                                     // 标签
	Annotations  map[string]string `json:"annotations"`                                // 注解
	Immutable    bool              `json:"immutable"`                                  // 是否不可变
	DataCount    int               `json:"data_count"`                                 // 数据条目数量
	Size         string            `json:"size"`                                       // 数据大小
	CreatedAt    time.Time         `json:"created_at"`                                 // 创建时间
	UpdatedAt    time.Time         `json:"updated_at"`                                 // 更新时间
	Age          string            `json:"age"`                                        // 存在时间，前端计算使用
	RawConfigMap *corev1.ConfigMap `json:"-"`                                          // 原始 ConfigMap 对象，不序列化到 JSON
}

// GetConfigMapListReq 获取ConfigMap列表请求
type GetConfigMapListReq struct {
	ListReq
	ClusterID int               `json:"cluster_id" form:"cluster_id" comment:"集群ID"` // 集群ID
	Namespace string            `json:"namespace" form:"namespace" comment:"命名空间"`   // 命名空间
	Labels    map[string]string `json:"labels" form:"labels" comment:"标签"`           // 标签
}

// GetConfigMapDetailsReq 获取ConfigMap详情请求
type GetConfigMapDetailsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"ConfigMap名称"`      // ConfigMap名称
}

// GetConfigMapYamlReq 获取ConfigMap YAML请求
type GetConfigMapYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"ConfigMap名称"`      // ConfigMap名称
}

// CreateConfigMapReq 创建ConfigMap请求
type CreateConfigMapReq struct {
	ClusterID   int               `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name        string            `json:"name" form:"name" binding:"required" comment:"ConfigMap名称"`      // ConfigMap名称
	Namespace   string            `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Data        map[string]string `json:"data" comment:"字符串数据"`                                           // 字符串数据
	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`                                    // 二进制数据
	Labels      map[string]string `json:"labels" comment:"标签"`                                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                                       // 注解
	Immutable   bool              `json:"immutable" comment:"是否不可变"`                                      // 是否不可变
}

// UpdateConfigMapReq 更新ConfigMap请求
type UpdateConfigMapReq struct {
	ClusterID   int               `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name        string            `json:"name" form:"name" binding:"required" comment:"ConfigMap名称"`      // ConfigMap名称
	Namespace   string            `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Data        map[string]string `json:"data" comment:"字符串数据"`                                           // 字符串数据
	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`                                    // 二进制数据
	Labels      map[string]string `json:"labels" comment:"标签"`                                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                                       // 注解
}

// CreateConfigMapByYamlReq 通过YAML创建ConfigMap请求
type CreateConfigMapByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// UpdateConfigMapByYamlReq 通过YAML更新ConfigMap请求
type UpdateConfigMapByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`                    // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"ConfigMap名称"`      // ConfigMap名称
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// DeleteConfigMapReq 删除ConfigMap请求
type DeleteConfigMapReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" binding:"required" comment:"ConfigMap名称"`                  // ConfigMap名称
}
