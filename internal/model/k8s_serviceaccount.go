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

// ObjectReference Kubernetes 对象引用
type ObjectReference struct {
	Kind            string `json:"kind,omitempty"`             // 对象类型
	Namespace       string `json:"namespace,omitempty"`        // 命名空间
	Name            string `json:"name,omitempty"`             // 对象名称
	UID             string `json:"uid,omitempty"`              // 唯一标识符
	APIVersion      string `json:"api_version,omitempty"`      // API 版本
	ResourceVersion string `json:"resource_version,omitempty"` // 资源版本
	FieldPath       string `json:"field_path,omitempty"`       // 字段路径
}

// LocalObjectReference 本地对象引用
type LocalObjectReference struct {
	Name string `json:"name,omitempty"` // 对象名称
}

// K8sServiceAccount Kubernetes ServiceAccount 资源
type K8sServiceAccount struct {
	Name                     string                   `json:"name"`         // ServiceAccount 名称
	Namespace                string                   `json:"namespace"`    // 命名空间
	UID                      string                   `json:"uid"`          // 唯一标识符
	Labels                   StringList               `json:"labels,omitempty" gorm:"serializer:json;comment:标签"` // 标签
	Annotations              StringList               `json:"annotations,omitempty" gorm:"serializer:json;comment:注解"` // 注解
	Secrets                  []LocalObjectReference   `json:"secrets,omitempty" gorm:"serializer:json;comment:关联的 Secret"` // 关联的 Secret
	ImagePullSecrets         []LocalObjectReference   `json:"image_pull_secrets,omitempty" gorm:"serializer:json;comment:镜像拉取 Secret"` // 镜像拉取 Secret
	AutomountServiceAccountToken *bool                `json:"automount_service_account_token,omitempty"` // 是否自动挂载服务账户令牌
	CreatedAt                time.Time                `json:"created_at"`   // 创建时间
}

// CreateServiceAccountRequest 创建 ServiceAccount 请求
type CreateServiceAccountRequest struct {
	ClusterID                    int                    `json:"cluster_id" binding:"required"`   // 集群ID
	Name                         string                 `json:"name" binding:"required"`         // ServiceAccount 名称
	Namespace                    string                 `json:"namespace" binding:"required"`    // 命名空间
	Labels                       StringList             `json:"labels,omitempty"`                // 标签
	Annotations                  StringList             `json:"annotations,omitempty"`           // 注解
	Secrets                      []LocalObjectReference `json:"secrets,omitempty"`               // 关联的 Secret
	ImagePullSecrets             []LocalObjectReference `json:"image_pull_secrets,omitempty"`    // 镜像拉取 Secret
	AutomountServiceAccountToken *bool                  `json:"automount_service_account_token,omitempty"` // 是否自动挂载服务账户令牌
}

// UpdateServiceAccountRequest 更新 ServiceAccount 请求
type UpdateServiceAccountRequest struct {
	ClusterID                    int                    `json:"cluster_id" binding:"required"`   // 集群ID
	Name                         string                 `json:"name" binding:"required"`         // ServiceAccount 名称
	Namespace                    string                 `json:"namespace" binding:"required"`    // 命名空间
	Labels                       StringList             `json:"labels,omitempty"`                // 标签
	Annotations                  StringList             `json:"annotations,omitempty"`           // 注解
	Secrets                      []LocalObjectReference `json:"secrets,omitempty"`               // 关联的 Secret
	ImagePullSecrets             []LocalObjectReference `json:"image_pull_secrets,omitempty"`    // 镜像拉取 Secret
	AutomountServiceAccountToken *bool                  `json:"automount_service_account_token,omitempty"` // 是否自动挂载服务账户令牌
}

// DeleteServiceAccountRequest 删除 ServiceAccount 请求
type DeleteServiceAccountRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Name      string `json:"name" binding:"required"`       // ServiceAccount 名称
	Namespace string `json:"namespace" binding:"required"`  // 命名空间
}

// ServiceAccountTokenRequest 创建 ServiceAccount Token 请求
type ServiceAccountTokenRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required"`           // 集群ID
	ServiceAccountName string `json:"service_account_name" binding:"required"` // ServiceAccount 名称
	Namespace          string `json:"namespace" binding:"required"`            // 命名空间
	ExpirationSeconds  *int64 `json:"expiration_seconds,omitempty"`            // 过期时间（秒）
}

// ServiceAccountToken ServiceAccount Token 响应
type ServiceAccountToken struct {
	Token string `json:"token"` // Token 内容
}

// ServiceAccountPermissions ServiceAccount 权限信息
type ServiceAccountPermissions struct {
	ServiceAccountName string          `json:"service_account_name"` // ServiceAccount 名称
	Namespace          string          `json:"namespace"`            // 命名空间
	Roles              []K8sRole       `json:"roles,omitempty"`      // 绑定的 Role
	ClusterRoles       []K8sClusterRole `json:"cluster_roles,omitempty"` // 绑定的 ClusterRole
	RoleBindings       []K8sRoleBinding `json:"role_bindings,omitempty"` // 相关的 RoleBinding
	ClusterRoleBindings []K8sClusterRoleBinding `json:"cluster_role_bindings,omitempty"` // 相关的 ClusterRoleBinding
}

// BindRoleToServiceAccountRequest 绑定 Role 到 ServiceAccount 请求
type BindRoleToServiceAccountRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required"`           // 集群ID
	ServiceAccountName string `json:"service_account_name" binding:"required"` // ServiceAccount 名称
	Namespace          string `json:"namespace" binding:"required"`            // 命名空间
	RoleName           string `json:"role_name" binding:"required"`            // Role 名称
	RoleBindingName    string `json:"role_binding_name,omitempty"`             // RoleBinding 名称，可选
}

// BindClusterRoleToServiceAccountRequest 绑定 ClusterRole 到 ServiceAccount 请求
type BindClusterRoleToServiceAccountRequest struct {
	ClusterID               int    `json:"cluster_id" binding:"required"`                   // 集群ID
	ServiceAccountName      string `json:"service_account_name" binding:"required"`         // ServiceAccount 名称
	Namespace               string `json:"namespace" binding:"required"`                    // 命名空间
	ClusterRoleName         string `json:"cluster_role_name" binding:"required"`            // ClusterRole 名称
	ClusterRoleBindingName  string `json:"cluster_role_binding_name,omitempty"`             // ClusterRoleBinding 名称，可选
}

// UnbindRoleFromServiceAccountRequest 解绑 Role 从 ServiceAccount 请求
type UnbindRoleFromServiceAccountRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required"`           // 集群ID
	ServiceAccountName string `json:"service_account_name" binding:"required"` // ServiceAccount 名称
	Namespace          string `json:"namespace" binding:"required"`            // 命名空间
	RoleBindingName    string `json:"role_binding_name" binding:"required"`    // RoleBinding 名称
}

// UnbindClusterRoleFromServiceAccountRequest 解绑 ClusterRole 从 ServiceAccount 请求
type UnbindClusterRoleFromServiceAccountRequest struct {
	ClusterID               int    `json:"cluster_id" binding:"required"`               // 集群ID
	ServiceAccountName      string `json:"service_account_name" binding:"required"`     // ServiceAccount 名称
	ClusterRoleBindingName  string `json:"cluster_role_binding_name" binding:"required"` // ClusterRoleBinding 名称
}