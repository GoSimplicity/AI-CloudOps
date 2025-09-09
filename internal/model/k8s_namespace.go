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

// K8sNamespace Kubernetes 命名空间
type K8sNamespace struct {
	ClusterID   int          `json:"cluster_id"`                             // 所属集群ID
	Name        string       `json:"name" binding:"required,min=1,max=200" ` // 命名空间名称
	UID         string       `json:"uid"`                                    // 命名空间UID
	Status      string       `json:"status" `                                // 命名空间状态
	Phase       string       `json:"phase" `                                 // 命名空间阶段
	Labels      KeyValueList `json:"labels" `                                // 标签
	Annotations KeyValueList `json:"annotations" `                           // 注解
}

// K8sNamespaceListReq 命名空间列表查询请求
type K8sNamespaceListReq struct {
	ListReq
	ClusterID     int          `json:"cluster_id" form:"cluster_id" binding:"required"` // 集群ID，必填
	Status        string       `json:"status" form:"status"`                            // 状态过滤
	Labels        KeyValueList `json:"labels" form:"labels"`                            // 标签
	LabelSelector string       `json:"label_selector" form:"label_selector"`            // 标签选择器
	Search        string       `json:"search" form:"search"`                            // 搜索关键字（用于过滤命名空间名称）
}

// K8sNamespaceCreateReq 创建命名空间请求
type K8sNamespaceCreateReq struct {
	ClusterID   int          `json:"cluster_id" form:"cluster_id" binding:"required"` // 集群ID，必填
	Name        string       `json:"name" binding:"required,min=1,max=200"`           // 命名空间名称，必填
	Labels      KeyValueList `json:"labels"`                                          // 标签
	Annotations KeyValueList `json:"annotations"`                                     // 注解
}

// K8sNamespaceUpdateReq 更新命名空间请求
type K8sNamespaceUpdateReq struct {
	ClusterID   int          `json:"cluster_id" form:"cluster_id" binding:"required"` // 集群ID，必填
	Name        string       `json:"name" binding:"required"`                         // 命名空间名称，必填
	Labels      KeyValueList `json:"labels"`                                          // 标签
	Annotations KeyValueList `json:"annotations"`                                     // 注解
}

// K8sNamespaceDeleteReq 删除命名空间请求
type K8sNamespaceDeleteReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" binding:"required"` // 集群ID，必填
	Name               string `json:"name" binding:"required"`                         // 命名空间名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds"`                            // 优雅删除时间（秒）
	Force              int8   `json:"force" binding:"required,oneof= 1 2"`             // 是否强制删除
}

// K8sNamespaceGetDetailsReq 获取命名空间详情请求
type K8sNamespaceGetDetailsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required"` // 集群ID，必填
	Name      string `json:"name" binding:"required"`                         // 命名空间名称，必填
}
