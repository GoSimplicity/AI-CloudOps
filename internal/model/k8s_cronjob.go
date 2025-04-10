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
	batchv1 "k8s.io/api/batch/v1"
)

// K8sCronjob Kubernetes 定时任务的配置
type K8sCronjob struct {
	Model
	Name         string     `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:定时任务名称"` // 定时任务名称
	Cluster      string     `json:"cluster,omitempty" gorm:"size:100;comment:所属集群"`                       // 所属集群
	TreeNodeID   int        `json:"tree_node_id" gorm:"comment:关联的树节点ID"`                                 // 关联的树节点ID
	UserID       int        `json:"user_id" gorm:"comment:创建者用户ID"`                                       // 创建者用户ID
	K8sProjectID int        `json:"k8s_project_id" gorm:"comment:关联的 Kubernetes 项目ID"`                    // 关联的 Kubernetes 项目ID
	Namespace    string     `json:"namespace,omitempty" gorm:"comment:命名空间"`                              // 命名空间
	Schedule     string     `json:"schedule,omitempty" gorm:"comment:调度表达式"`                              // 调度表达式
	Image        string     `json:"image,omitempty" gorm:"comment:镜像"`                                    // 镜像
	Commands     StringList `json:"commands,omitempty" gorm:"comment:启动命令组"`                              // 启动命令组
	Args         StringList `json:"args,omitempty" gorm:"comment:启动参数，空格分隔"`                              // 启动参数
}

// CreateK8sCronjobRequest 创建 CronJob 的请求
type CreateK8sCronjobRequest struct {
	Name         string     `json:"name" binding:"required,min=1,max=200"` // 定时任务名称
	Cluster      string     `json:"cluster" binding:"required"`            // 所属集群
	TreeNodeID   int        `json:"tree_node_id,omitempty"`                // 关联的树节点ID（可选）
	UserID       int        `json:"user_id" binding:"required"`            // 创建者用户ID
	K8sProjectID int        `json:"k8s_project_id" binding:"required"`     // 关联的 Kubernetes 项目ID
	Namespace    string     `json:"namespace" binding:"required"`          // 命名空间
	Schedule     string     `json:"schedule" binding:"required"`           // 调度表达式
	Image        string     `json:"image" binding:"required"`              // 镜像
	Commands     StringList `json:"commands,omitempty"`                    // 启动命令组
	Args         StringList `json:"args,omitempty"`                        // 启动参数
}

// UpdateK8sCronjobRequest 更新 CronJob 的请求
type UpdateK8sCronjobRequest struct {
	ID           int64      `json:"id" binding:"required"`
	Name         string     `json:"name" binding:"required,min=1,max=200"` // 定时任务名称
	Cluster      string     `json:"cluster" binding:"required"`            // 所属集群
	TreeNodeID   int        `json:"tree_node_id,omitempty"`                // 关联的树节点ID（可选）
	K8sProjectID int        `json:"k8s_project_id" binding:"required"`     // 关联的 Kubernetes 项目ID
	Namespace    string     `json:"namespace" binding:"required"`          // 命名空间
	Schedule     string     `json:"schedule" binding:"required"`           // 调度表达式
	Image        string     `json:"image" binding:"required"`              // 镜像
	Commands     StringList `json:"commands,omitempty"`                    // 启动命令组
	Args         StringList `json:"args,omitempty"`                        // 启动参数
}

// GetK8sCronjobListRequest 获取 CronJob 列表的请求
type GetK8sCronjobListRequest struct {
	ProjectID   int64  `json:"project_id,omitempty"`   // 项目ID过滤
	ClusterName string `json:"cluster_name,omitempty"` // 集群名称过滤
	Namespace   string `json:"namespace,omitempty"`    // 命名空间过滤
	Name        string `json:"name,omitempty"`         // 名称过滤（模糊查询）
	Page        int    `json:"page,omitempty"`         // 分页页码
	PageSize    int    `json:"page_size,omitempty"`    // 分页大小
}

// BatchDeleteK8sCronjobRequest 批量删除 CronJob 的请求
type BatchDeleteK8sCronjobRequest struct {
	CronjobIDs []int64 `json:"cronjob_ids" binding:"required"`
}

// K8sCronjobRequest CronJob 相关请求结构
type K8sCronjobRequest struct {
	ClusterId    int              `json:"cluster_id" binding:"required"` // 集群id，必填
	Namespace    string           `json:"namespace" binding:"required"`  // 命名空间，必填
	CronjobNames []string         `json:"cronjob_names,omitempty"`       // CronJob 名称，可选
	CronjobYaml  *batchv1.CronJob `json:"cronjob_yaml,omitempty"`        // CronJob 对象, 可选
}

// K8sCronjobPodResponse CronJob Pod 响应
type K8sCronjobPodResponse struct {
	Pod     *K8sPod `json:"pod"`      // Pod 信息
	LogData string  `json:"log_data"` // 日志数据
}