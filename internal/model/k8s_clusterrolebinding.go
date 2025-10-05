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

// K8sClusterRoleBinding ClusterRoleBinding主model
type K8sClusterRoleBinding struct {
	Name                  string                     `json:"name"`
	ClusterID             int                        `json:"cluster_id"`
	UID                   string                     `json:"uid"`
	CreatedAt             string                     `json:"created_at"`
	Labels                map[string]string          `json:"labels"`
	Annotations           map[string]string          `json:"annotations"`
	RoleRef               RoleRef                    `json:"role_ref"`
	Subjects              []Subject                  `json:"subjects"`
	ResourceVersion       string                     `json:"resource_version"`
	Age                   string                     `json:"age"`
	RawClusterRoleBinding *rbacv1.ClusterRoleBinding `json:"-"` // 原始ClusterRoleBinding对象，不序列化
}

// GetClusterRoleBindingListReq 获取ClusterRoleBinding列表请求
type GetClusterRoleBindingListReq struct {
	ListReq
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
}

// GetClusterRoleBindingDetailsReq 获取ClusterRoleBinding详情请求
type GetClusterRoleBindingDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
}

// GetClusterRoleBindingYamlReq 获取ClusterRoleBinding YAML请求
type GetClusterRoleBindingYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
}

// CreateClusterRoleBindingReq 创建ClusterRoleBinding请求
type CreateClusterRoleBindingReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	RoleRef     RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects    []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// CreateClusterRoleBindingByYamlReq 通过YAML创建ClusterRoleBinding请求
type CreateClusterRoleBindingByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdateClusterRoleBindingReq 更新ClusterRoleBinding请求
type UpdateClusterRoleBindingReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	RoleRef     RoleRef           `json:"role_ref" binding:"required" comment:"角色引用"`
	Subjects    []Subject         `json:"subjects" binding:"required" comment:"主体列表"`
}

// UpdateClusterRoleBindingByYamlReq 通过YAML更新ClusterRoleBinding请求
type UpdateClusterRoleBindingByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeleteClusterRoleBindingReq 删除ClusterRoleBinding请求
type DeleteClusterRoleBindingReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"ClusterRoleBinding名称"`
}
