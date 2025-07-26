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

	core "k8s.io/api/core/v1"
)

// K8sCluster Kubernetes 集群的配置
type K8sCluster struct {
	Model
	Name                 string     `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:集群名称"`      // 集群名称
	NameZh               string     `json:"name_zh" binding:"required,min=1,max=500" gorm:"size:100;comment:集群中文名称"` // 集群中文名称
	UserID               int        `json:"user_id" gorm:"comment:创建者用户ID"`                                          // 创建者用户ID
	CpuRequest           string     `json:"cpu_request,omitempty" gorm:"comment:CPU 请求量"`                            // CPU 请求量
	CpuLimit             string     `json:"cpu_limit,omitempty" gorm:"comment:CPU 限制量"`                              // CPU 限制量
	MemoryRequest        string     `json:"memory_request,omitempty" gorm:"comment:内存请求量"`                           // 内存请求量
	MemoryLimit          string     `json:"memory_limit,omitempty" gorm:"comment:内存限制量"`                             // 内存限制量
	RestrictedNameSpace  StringList `json:"restricted_name_space" gorm:"comment:资源限制命名空间"`                           // 资源限制命名空间
	Status               string     `json:"status" gorm:"comment:集群状态"`                                              // 集群状态
	Env                  string     `json:"env,omitempty" gorm:"comment:集群环境，例如 prod, stage, dev, rc, press"`        // 集群环境
	Version              string     `json:"version,omitempty" gorm:"comment:集群版本"`                                   // 集群版本
	ApiServerAddr        string     `json:"api_server_addr,omitempty" gorm:"comment:API Server 地址"`                  // API Server 地址
	KubeConfigContent    string     `json:"kube_config_content,omitempty" gorm:"type:text;comment:kubeConfig 内容"`    // kubeConfig 内容
	ActionTimeoutSeconds int        `json:"action_timeout_seconds,omitempty" gorm:"comment:操作超时时间（秒）"`               // 操作超时时间（秒）
}

func (k8sCluster *K8sCluster) TableName() string {
	return "cl_k8s_clusters"
}

// ClusterNamespaces 表示一个集群及其命名空间列表
type ClusterNamespaces struct {
	ClusterName string      `json:"cluster_name"` // 集群名称
	ClusterId   int         `json:"cluster_id"`   // 集群ID
	Namespaces  []Namespace `json:"namespaces"`   // 命名空间列表
}

// Namespace 命名空间响应结构体
type Namespace struct {
	Name         string    `json:"name"`                  // 命名空间名称
	UID          string    `json:"uid"`                   // 命名空间唯一标识符
	Status       string    `json:"status"`                // 命名空间状态，例如 Active
	CreationTime time.Time `json:"creation_time"`         // 创建时间
	Labels       []string  `json:"labels,omitempty"`      // 命名空间标签
	Annotations  []string  `json:"annotations,omitempty"` // 命名空间注解
}

// CreateNamespaceRequest 创建新的命名空间请求结构体
type CreateNamespaceRequest struct {
	ClusterId   int    `json:"cluster_id" binding:"required"`
	Name        string   `json:"namespace" binding:"required"`
	Labels      []string `json:"labels,omitempty"`      // 命名空间标签
	Annotations []string `json:"annotations,omitempty"` // 命名空间注解
}

// UpdateNamespaceRequest 更新命名空间请求结构体
type UpdateNamespaceRequest struct {
	ClusterId   int    `json:"cluster_id" binding:"required"`
	Name        string   `json:"namespace" binding:"required"`
	Labels      []string `json:"labels,omitempty"`      // 命名空间标签
	Annotations []string `json:"annotations,omitempty"` // 命名空间注解
}

// K8sClusterNodesRequest 定义集群节点请求的基础结构
type K8sClusterNodesRequest struct {
	ClusterId int    `json:"cluster_id" binding:"required"` // 集群id，必填
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称列表，必填
}

// Resource 命名空间中的资源响应结构体
type Resource struct {
	Type         string    `json:"type"`          // 资源类型，例如 Pod, Service, Deployment
	Name         string    `json:"name"`          // 资源名称
	Namespace    string    `json:"namespace"`     // 所属命名空间
	Status       string    `json:"status"`        // 资源状态，例如 Running, Pending
	CreationTime time.Time `json:"creation_time"` // 创建时间
}

// Event 命名空间事件响应结构体
type Event struct {
	Reason         string           `json:"reason"`          // 事件原因
	Message        string           `json:"message"`         // 事件消息
	Type           string           `json:"type"`            // 事件类型，例如 Normal, Warning
	FirstTimestamp time.Time        `json:"first_timestamp"` // 第一次发生时间
	LastTimestamp  time.Time        `json:"last_timestamp"`  // 最后一次发生时间
	Count          int32            `json:"count"`           // 事件发生次数
	Source         core.EventSource `json:"source"`          // 事件来源
}