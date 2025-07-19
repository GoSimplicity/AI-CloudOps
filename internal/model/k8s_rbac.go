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

// PolicyRule RBAC 策略规则
type PolicyRule struct {
	Verbs           []string `json:"verbs" gorm:"serializer:json;comment:允许的操作动词"`                      // 允许的操作动词，如 get, list, create, update, delete
	APIGroups       []string `json:"api_groups" gorm:"serializer:json;comment:API 组"`                   // API 组，如 "", "apps", "extensions"
	Resources       []string `json:"resources" gorm:"serializer:json;comment:资源类型"`                     // 资源类型，如 pods, services, deployments
	ResourceNames   []string `json:"resource_names,omitempty" gorm:"serializer:json;comment:特定资源名称"`    // 特定资源名称
	NonResourceURLs []string `json:"non_resource_urls,omitempty" gorm:"serializer:json;comment:非资源URL"` // 非资源URL，如 "/healthz"
}

// Subject RBAC 主体
type Subject struct {
	Kind      string `json:"kind" binding:"required,oneof=User Group ServiceAccount"` // 主体类型：User, Group, ServiceAccount
	APIGroup  string `json:"api_group,omitempty"`                                     // API 组，User和Group为"rbac.authorization.k8s.io"，ServiceAccount为""
	Name      string `json:"name" binding:"required"`                                 // 主体名称
	Namespace string `json:"namespace,omitempty"`                                     // ServiceAccount 所在的命名空间
}

// RoleRef RBAC 角色引用
type RoleRef struct {
	APIGroup string `json:"api_group" binding:"required"`                   // API 组，通常为 "rbac.authorization.k8s.io"
	Kind     string `json:"kind" binding:"required,oneof=Role ClusterRole"` // 角色类型：Role 或 ClusterRole
	Name     string `json:"name" binding:"required"`                        // 角色名称
}

// K8sRole Kubernetes Role 资源
type K8sRole struct {
	Name        string       `json:"name"`                                                    // Role 名称
	Namespace   string       `json:"namespace"`                                               // 命名空间
	UID         string       `json:"uid"`                                                     // 唯一标识符
	Labels      StringList   `json:"labels,omitempty" gorm:"serializer:json;comment:标签"`      // 标签
	Annotations StringList   `json:"annotations,omitempty" gorm:"serializer:json;comment:注解"` // 注解
	Rules       []PolicyRule `json:"rules" gorm:"serializer:json;comment:策略规则"`               // 策略规则
	CreatedAt   time.Time    `json:"created_at"`                                              // 创建时间
}

// K8sClusterRole Kubernetes ClusterRole 资源
type K8sClusterRole struct {
	Name        string       `json:"name"`                                                    // ClusterRole 名称
	UID         string       `json:"uid"`                                                     // 唯一标识符
	Labels      StringList   `json:"labels,omitempty" gorm:"serializer:json;comment:标签"`      // 标签
	Annotations StringList   `json:"annotations,omitempty" gorm:"serializer:json;comment:注解"` // 注解
	Rules       []PolicyRule `json:"rules" gorm:"serializer:json;comment:策略规则"`               // 策略规则
	CreatedAt   time.Time    `json:"created_at"`                                              // 创建时间
}

// K8sRoleBinding Kubernetes RoleBinding 资源
type K8sRoleBinding struct {
	Name        string     `json:"name"`                                                    // RoleBinding 名称
	Namespace   string     `json:"namespace"`                                               // 命名空间
	UID         string     `json:"uid"`                                                     // 唯一标识符
	Labels      StringList `json:"labels,omitempty" gorm:"serializer:json;comment:标签"`      // 标签
	Annotations StringList `json:"annotations,omitempty" gorm:"serializer:json;comment:注解"` // 注解
	Subjects    []Subject  `json:"subjects" gorm:"serializer:json;comment:绑定的主体"`           // 绑定的主体
	RoleRef     RoleRef    `json:"role_ref" gorm:"serializer:json;comment:角色引用"`            // 角色引用
	CreatedAt   time.Time  `json:"created_at"`                                              // 创建时间
}

// K8sClusterRoleBinding Kubernetes ClusterRoleBinding 资源
type K8sClusterRoleBinding struct {
	Name        string     `json:"name"`                                                    // ClusterRoleBinding 名称
	UID         string     `json:"uid"`                                                     // 唯一标识符
	Labels      StringList `json:"labels,omitempty" gorm:"serializer:json;comment:标签"`      // 标签
	Annotations StringList `json:"annotations,omitempty" gorm:"serializer:json;comment:注解"` // 注解
	Subjects    []Subject  `json:"subjects" gorm:"serializer:json;comment:绑定的主体"`           // 绑定的主体
	RoleRef     RoleRef    `json:"role_ref" gorm:"serializer:json;comment:角色引用"`            // 角色引用
	CreatedAt   time.Time  `json:"created_at"`                                              // 创建时间
}

// CreateK8sRoleRequest 创建 Kubernetes Role 请求
type CreateK8sRoleRequest struct {
	ClusterID   int          `json:"cluster_id" binding:"required"` // 集群ID
	Name        string       `json:"name" binding:"required"`       // Role 名称
	Namespace   string       `json:"namespace" binding:"required"`  // 命名空间
	Labels      StringList   `json:"labels,omitempty"`              // 标签
	Annotations StringList   `json:"annotations,omitempty"`         // 注解
	Rules       []PolicyRule `json:"rules" binding:"required"`      // 策略规则
}

// CreateClusterRoleRequest 创建 ClusterRole 请求
type CreateClusterRoleRequest struct {
	ClusterID   int          `json:"cluster_id" binding:"required"` // 集群ID
	Name        string       `json:"name" binding:"required"`       // ClusterRole 名称
	Labels      StringList   `json:"labels,omitempty"`              // 标签
	Annotations StringList   `json:"annotations,omitempty"`         // 注解
	Rules       []PolicyRule `json:"rules" binding:"required"`      // 策略规则
}

// CreateRoleBindingRequest 创建 RoleBinding 请求
type CreateRoleBindingRequest struct {
	ClusterID   int        `json:"cluster_id" binding:"required"` // 集群ID
	Name        string     `json:"name" binding:"required"`       // RoleBinding 名称
	Namespace   string     `json:"namespace" binding:"required"`  // 命名空间
	Labels      StringList `json:"labels,omitempty"`              // 标签
	Annotations StringList `json:"annotations,omitempty"`         // 注解
	Subjects    []Subject  `json:"subjects" binding:"required"`   // 绑定的主体
	RoleRef     RoleRef    `json:"role_ref" binding:"required"`   // 角色引用
}

// CreateClusterRoleBindingRequest 创建 ClusterRoleBinding 请求
type CreateClusterRoleBindingRequest struct {
	ClusterID   int        `json:"cluster_id" binding:"required"` // 集群ID
	Name        string     `json:"name" binding:"required"`       // ClusterRoleBinding 名称
	Labels      StringList `json:"labels,omitempty"`              // 标签
	Annotations StringList `json:"annotations,omitempty"`         // 注解
	Subjects    []Subject  `json:"subjects" binding:"required"`   // 绑定的主体
	RoleRef     RoleRef    `json:"role_ref" binding:"required"`   // 角色引用
}

// UpdateK8sRoleRequest 更新 Kubernetes Role 请求
type UpdateK8sRoleRequest struct {
	ClusterID   int          `json:"cluster_id" binding:"required"` // 集群ID
	Name        string       `json:"name" binding:"required"`       // Role 名称
	Namespace   string       `json:"namespace" binding:"required"`  // 命名空间
	Labels      StringList   `json:"labels,omitempty"`              // 标签
	Annotations StringList   `json:"annotations,omitempty"`         // 注解
	Rules       []PolicyRule `json:"rules" binding:"required"`      // 策略规则
}

// UpdateClusterRoleRequest 更新 ClusterRole 请求
type UpdateClusterRoleRequest struct {
	ClusterID   int          `json:"cluster_id" binding:"required"` // 集群ID
	Name        string       `json:"name" binding:"required"`       // ClusterRole 名称
	Labels      StringList   `json:"labels,omitempty"`              // 标签
	Annotations StringList   `json:"annotations,omitempty"`         // 注解
	Rules       []PolicyRule `json:"rules" binding:"required"`      // 策略规则
}

// UpdateRoleBindingRequest 更新 RoleBinding 请求
type UpdateRoleBindingRequest struct {
	ClusterID   int        `json:"cluster_id" binding:"required"` // 集群ID
	Name        string     `json:"name" binding:"required"`       // RoleBinding 名称
	Namespace   string     `json:"namespace" binding:"required"`  // 命名空间
	Labels      StringList `json:"labels,omitempty"`              // 标签
	Annotations StringList `json:"annotations,omitempty"`         // 注解
	Subjects    []Subject  `json:"subjects" binding:"required"`   // 绑定的主体
	RoleRef     RoleRef    `json:"role_ref" binding:"required"`   // 角色引用
}

// UpdateClusterRoleBindingRequest 更新 ClusterRoleBinding 请求
type UpdateClusterRoleBindingRequest struct {
	ClusterID   int        `json:"cluster_id" binding:"required"` // 集群ID
	Name        string     `json:"name" binding:"required"`       // ClusterRoleBinding 名称
	Labels      StringList `json:"labels,omitempty"`              // 标签
	Annotations StringList `json:"annotations,omitempty"`         // 注解
	Subjects    []Subject  `json:"subjects" binding:"required"`   // 绑定的主体
	RoleRef     RoleRef    `json:"role_ref" binding:"required"`   // 角色引用
}

// DeleteK8sRoleRequest 删除 Kubernetes Role 请求
type DeleteK8sRoleRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Name      string `json:"name" binding:"required"`       // Role 名称
	Namespace string `json:"namespace" binding:"required"`  // 命名空间
}

// DeleteClusterRoleRequest 删除 ClusterRole 请求
type DeleteClusterRoleRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Name      string `json:"name" binding:"required"`       // ClusterRole 名称
}

// DeleteRoleBindingRequest 删除 RoleBinding 请求
type DeleteRoleBindingRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Name      string `json:"name" binding:"required"`       // RoleBinding 名称
	Namespace string `json:"namespace" binding:"required"`  // 命名空间
}

// DeleteClusterRoleBindingRequest 删除 ClusterRoleBinding 请求
type DeleteClusterRoleBindingRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Name      string `json:"name" binding:"required"`       // ClusterRoleBinding 名称
}
