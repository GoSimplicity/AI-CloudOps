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
	rbacv1 "k8s.io/api/rbac/v1"
)

// GetRoleBindingListReq 获取RoleBinding列表请求
type GetRoleBindingListReq struct {
	ListReq
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
}

// GetRoleBindingDetailsReq 获取RoleBinding详情请求
type GetRoleBindingDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"RoleBinding名称"`
}

// GetRoleBindingYamlReq 获取RoleBinding YAML请求
type GetRoleBindingYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"RoleBinding名称"`
}

// CreateRoleBindingReq 创建RoleBinding请求
type CreateRoleBindingReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string            `json:"name" binding:"required" comment:"RoleBinding名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	RoleRef     RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects    []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// CreateRoleBindingByYamlReq 通过YAML创建RoleBinding请求
type CreateRoleBindingByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdateRoleBindingReq 更新RoleBinding请求
type UpdateRoleBindingReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string            `json:"name" binding:"required" comment:"RoleBinding名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	RoleRef     RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects    []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// UpdateRoleBindingByYamlReq 通过YAML更新RoleBinding请求
type UpdateRoleBindingByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"RoleBinding名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeleteRoleBindingReq 删除RoleBinding请求
type DeleteRoleBindingReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"RoleBinding名称"`
}

// K8sRoleBinding RoleBinding主model
type K8sRoleBinding struct {
	Name            string              `json:"name"`
	Namespace       string              `json:"namespace"`
	ClusterID       int                 `json:"cluster_id"`
	UID             string              `json:"uid"`
	CreatedAt       string              `json:"created_at"`
	Labels          map[string]string   `json:"labels"`
	Annotations     map[string]string   `json:"annotations"`
	RoleRef         RoleRef             `json:"role_ref"`
	Subjects        []Subject           `json:"subjects"`
	ResourceVersion string              `json:"resource_version"`
	Age             string              `json:"age"`
	RawRoleBinding  *rbacv1.RoleBinding `json:"-"` // 原始RoleBinding对象，不序列化
}
