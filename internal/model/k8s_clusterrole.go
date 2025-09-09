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

// GetClusterRoleListReq 获取ClusterRole列表请求
type GetClusterRoleListReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Keyword   string `json:"keyword" form:"keyword" comment:"关键字搜索"`
	Page      int    `json:"page" form:"page" comment:"页码"`
	PageSize  int    `json:"page_size" form:"page_size" comment:"页面大小"`
}

// GetClusterRoleDetailsReq 获取ClusterRole详情请求
type GetClusterRoleDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"ClusterRole名称"`
}

// GetClusterRoleYamlReq 获取ClusterRole YAML请求
type GetClusterRoleYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"ClusterRole名称"`
}

// CreateClusterRoleReq 创建ClusterRole请求
type CreateClusterRoleReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"ClusterRole名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	Rules       []PolicyRule      `json:"rules" binding:"required" comment:"权限规则列表"`
}

// CreateClusterRoleByYamlReq 通过YAML创建ClusterRole请求
type CreateClusterRoleByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdateClusterRoleReq 更新ClusterRole请求
type UpdateClusterRoleReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string            `json:"name" binding:"required" comment:"ClusterRole名称"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
	Rules       []PolicyRule      `json:"rules" binding:"required" comment:"权限规则列表"`
}

// UpdateClusterRoleByYamlReq 通过YAML更新ClusterRole请求
type UpdateClusterRoleByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name        string `json:"name" binding:"required" comment:"ClusterRole名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeleteClusterRoleReq 删除ClusterRole请求
type DeleteClusterRoleReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Name      string `json:"name" binding:"required" comment:"ClusterRole名称"`
}

// K8sClusterRole ClusterRole主model
type K8sClusterRole struct {
	Name              string              `json:"name"`
	ClusterID         int                 `json:"cluster_id"`
	UID               string              `json:"uid"`
	CreationTimestamp string              `json:"creation_timestamp"`
	Labels            map[string]string   `json:"labels"`
	Annotations       map[string]string   `json:"annotations"`
	Rules             []PolicyRule        `json:"rules"`
	ResourceVersion   string              `json:"resource_version"`
	Age               string              `json:"age"`
	RawClusterRole    *rbacv1.ClusterRole `json:"-"` // 原始ClusterRole对象，不序列化
}
