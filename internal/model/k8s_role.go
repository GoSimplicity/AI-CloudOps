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

// GetRoleListReq 获取Role列表请求
type GetRoleListReq struct {
	ListReq
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
}

// GetRoleDetailsReq 获取Role详情请求
type GetRoleDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"Role名称"`
}

// GetRoleYamlReq 获取Role YAML请求
type GetRoleYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"Role名称"`
}

// CreateRoleReq 创建Role请求
type CreateRoleReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string            `json:"name" binding:"required" comment:"Role名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	Rules       []PolicyRule      `json:"rules" binding:"required" comment:"权限规则列表"`
}

// CreateRoleByYamlReq 通过YAML创建Role请求
type CreateRoleByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdateRoleReq 更新Role请求
type UpdateRoleReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string            `json:"name" binding:"required" comment:"Role名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	Rules       []PolicyRule      `json:"rules" binding:"required" comment:"权限规则列表"`
}

// UpdateRoleByYamlReq 通过YAML更新Role请求
type UpdateRoleByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"Role名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeleteRoleReq 删除Role请求
type DeleteRoleReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"Role名称"`
}

// K8sRole Role主model
type K8sRole struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	ClusterID         int               `json:"cluster_id"`
	UID               string            `json:"uid"`
	CreationTimestamp string            `json:"creation_timestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	Rules             []PolicyRule      `json:"rules"`
	ResourceVersion   string            `json:"resource_version"`
	Age               string            `json:"age"`
	RawRole           *rbacv1.Role      `json:"-"` // 原始Role对象，不序列化
}
