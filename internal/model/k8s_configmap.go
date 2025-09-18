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

// K8sConfigMap K8s ConfigMap模型
type K8sConfigMap struct {
	Name              string            `json:"name"`               // ConfigMap名称
	Namespace         string            `json:"namespace"`          // 所属命名空间
	ClusterID         int               `json:"cluster_id"`         // 所属集群ID
	UID               string            `json:"uid"`                // ConfigMap UID
	Data              map[string]string `json:"data"`               // 字符串数据
	BinaryData        map[string][]byte `json:"binary_data"`        // 二进制数据
	Labels            map[string]string `json:"labels"`             // 标签
	Annotations       map[string]string `json:"annotations"`        // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp"` // Kubernetes创建时间
	Age               string            `json:"age"`                // 存在时间，前端计算使用
	DataCount         int               `json:"data_count"`         // 数据条目数量，前端计算使用
	Size              string            `json:"size"`               // 数据大小，前端计算使用
}

// ListConfigMapsReq 获取ConfigMap列表请求
type ListConfigMapsReq struct {
	ListReq
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	DataKey       string `json:"data_key" form:"data_key" comment:"数据键过滤"`                       // 数据键过滤
}

// GetConfigMapReq 获取单个ConfigMap详情请求
type GetConfigMapReq struct {
	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required"`
	Namespace    string `json:"namespace" form:"namespace" uri:"namespace" binding:"required"`
	ResourceName string `json:"resource_name" form:"resource_name" uri:"name" binding:"required"`
}

// CreateConfigMapReq 创建ConfigMap请求
type CreateConfigMapReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace   string            `json:"namespace" binding:"required"`  // 命名空间，必填
	Name        string            `json:"name" binding:"required"`       // ConfigMap名称，必填
	Data        map[string]string `json:"data"`                          // 字符串数据
	BinaryData  map[string][]byte `json:"binary_data"`                   // 二进制数据
	Labels      map[string]string `json:"labels"`                        // 标签
	Annotations map[string]string `json:"annotations"`                   // 注解
	Immutable   bool              `json:"immutable"`                     // 是否不可变
}

// UpdateConfigMapReq 更新ConfigMap请求
type UpdateConfigMapReq struct {
	ClusterID    int               `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace    string            `json:"namespace" binding:"required"`     // 命名空间，必填
	ResourceName string            `json:"resource_name" binding:"required"` // ConfigMap名称，必填
	Data         map[string]string `json:"data"`                             // 字符串数据
	BinaryData   map[string][]byte `json:"binary_data"`                      // 二进制数据
	Labels       map[string]string `json:"labels"`                           // 标签
	Annotations  map[string]string `json:"annotations"`                      // 注解
	Immutable    bool              `json:"immutable"`                        // 是否不可变
}

// DeleteConfigMapReq 删除ConfigMap请求
type DeleteConfigMapReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required"`
	Namespace          string `json:"namespace" form:"namespace" uri:"namespace" binding:"required"`
	ResourceName       string `json:"resource_name" form:"resource_name" uri:"name" binding:"required"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds"` // 优雅删除时间（秒）
	Force              bool   `json:"force"`                // 是否强制删除
}

// GetConfigMapYAMLReq 获取ConfigMap YAML请求
type GetConfigMapYAMLReq struct {
	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required"`
	Namespace    string `json:"namespace" form:"namespace" uri:"namespace" binding:"required"`
	ResourceName string `json:"resource_name" form:"resource_name" uri:"name" binding:"required"`
}

// 删除冗余的ConfigMap YAML专用请求结构，统一使用通用YAML请求
