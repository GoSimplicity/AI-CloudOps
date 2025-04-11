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

// K8sApp 面向运维的 Kubernetes 应用
type K8sApp struct {
	Model
	Name          string                 `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:应用名称"` // 应用名称
	K8sProjectID  int                    `json:"k8s_project_id" gorm:"comment:关联的 Kubernetes 项目ID"`                  // 关联的 Kubernetes 项目ID
	TreeNodeID    int                    `json:"tree_node_id" gorm:"comment:关联的树节点ID"`                               // 关联的树节点ID
	UserID        int                    `json:"user_id" gorm:"comment:创建者用户ID"`                                     // 创建者用户ID
	Cluster       string                 `json:"cluster" gorm:"size:100;comment:所属集群名称"`                             // 所属集群名称
	K8sInstances  []K8sInstance          `json:"k8s_instances" gorm:"foreignKey:K8sAppID;comment:关联的 Kubernetes 实例"` // 关联的 Kubernetes 实例
	ServiceType   string                 `json:"service_type,omitempty" gorm:"comment:服务类型"`                         // 服务类型
	Namespace     string                 `json:"namespace,omitempty" gorm:"comment:Kubernetes 命名空间"`                 // Kubernetes 命名空间
	ContainerCore `json:"containerCore"` // 容器核心配置
}

// CreateK8sAppRequest 创建 Kubernetes 应用的请求
type CreateK8sAppRequest struct {
	Name          string                 `json:"name" binding:"required,min=1,max=200"` // 应用名称
	K8sProjectID  int64                  `json:"k8s_project_id" binding:"required"`     // 关联的 Kubernetes 项目ID
	UserID        int                    `json:"user_id" binding:"required"`            // 创建者用户ID
	Cluster       string                 `json:"cluster" binding:"required"`            // 所属集群名称
	ServiceType   string                 `json:"service_type,omitempty"`                // 服务类型
	Namespace     string                 `json:"namespace" binding:"required"`          // Kubernetes 命名空间
	ContainerCore `json:"containerCore"` // 容器核心配置
}

// UpdateK8sAppRequest 更新 Kubernetes 应用的请求
type UpdateK8sAppRequest struct {
	ID            int64                  `json:"id" binding:"required"`
	Name          string                 `json:"name" binding:"required,min=1,max=200"` // 应用名称
	K8sProjectID  int64                  `json:"k8s_project_id" binding:"required"`     // 关联的 Kubernetes 项目ID
	Cluster       string                 `json:"cluster" binding:"required"`            // 所属集群名称
	ServiceType   string                 `json:"service_type,omitempty"`                // 服务类型
	Namespace     string                 `json:"namespace" binding:"required"`          // Kubernetes 命名空间
	ContainerCore `json:"containerCore"` // 容器核心配置
}

// GetK8sAppListRequest 获取 Kubernetes 应用列表的请求
type GetK8sAppListRequest struct {
	ProjectID   int64  `json:"project_id,omitempty"`   // 项目ID过滤
	ClusterName string `json:"cluster_name,omitempty"` // 集群名称过滤
	Namespace   string `json:"namespace,omitempty"`    // 命名空间过滤
	Name        string `json:"name,omitempty"`         // 名称过滤（模糊查询）
	Page        int    `json:"page,omitempty"`         // 分页页码
	PageSize    int    `json:"page_size,omitempty"`    // 分页大小
}