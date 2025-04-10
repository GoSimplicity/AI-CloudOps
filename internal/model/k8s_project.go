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

// K8sProject Kubernetes 项目的配置
type K8sProject struct {
	Model
	Name       string   `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:项目名称"`          // 项目名称
	NameZh     string   `json:"name_zh" binding:"required,min=1,max=500" gorm:"size:100;comment:项目中文名称"`     // 项目中文名称
	Cluster    string   `json:"cluster" gorm:"size:100;comment:所属集群名称"`                                      // 所属集群名称
	TreeNodeID int      `json:"tree_node_id" gorm:"comment:关联的树节点ID"`                                        // 关联的树节点ID
	UserID     int      `json:"user_id" gorm:"comment:创建者用户ID"`                                              // 创建者用户ID
	K8sApps    []K8sApp `json:"k8s_apps,omitempty" gorm:"foreignKey:K8sProjectID;comment:关联的 Kubernetes 应用"` // 关联的 Kubernetes 应用
}

// CreateK8sProjectRequest 创建 Kubernetes 项目的请求
type CreateK8sProjectRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=200"`    // 项目名称
	NameZh     string `json:"name_zh" binding:"required,min=1,max=500"` // 项目中文名称
	Cluster    string `json:"cluster" binding:"required"`               // 所属集群名称
	TreeNodeID int    `json:"tree_node_id,omitempty"`                   // 关联的树节点ID（可选）
	UserID     int    `json:"user_id" binding:"required"`               // 创建者用户ID
}

// UpdateK8sProjectRequest 更新 Kubernetes 项目的请求
type UpdateK8sProjectRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Name       string `json:"name" binding:"required,min=1,max=200"`    // 项目名称
	NameZh     string `json:"name_zh" binding:"required,min=1,max=500"` // 项目中文名称
	Cluster    string `json:"cluster" binding:"required"`               // 所属集群名称
	TreeNodeID int    `json:"tree_node_id,omitempty"`                   // 关联的树节点ID（可选）
}

// GetK8sProjectListRequest 获取 Kubernetes 项目列表的请求
type GetK8sProjectListRequest struct {
	ClusterName string `json:"cluster_name,omitempty"` // 集群名称过滤
	Name        string `json:"name,omitempty"`         // 名称过滤（模糊查询）
	TreeNodeID  int    `json:"tree_node_id,omitempty"` // 树节点ID过滤
	Page        int    `json:"page,omitempty"`         // 分页页码
	PageSize    int    `json:"page_size,omitempty"`    // 分页大小
}