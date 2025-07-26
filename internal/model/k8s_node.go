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

// K8sNode Kubernetes 节点
type K8sNode struct {
	Name              string               `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:节点名称"`    // 节点名称
	ClusterID         int                  `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                       // 所属集群ID
	Status            string               `json:"status" gorm:"comment:节点状态，例如 Ready, NotReady, SchedulingDisabled"`     // 节点状态
	ScheduleEnable    bool                 `json:"schedule_enable" gorm:"comment:节点是否可调度"`                                // 节点是否可调度
	Roles             []string             `json:"roles" gorm:"type:text;serializer:json;comment:节点角色，例如 master, worker"` // 节点角色
	Age               string               `json:"age" gorm:"comment:节点存在时间，例如 5d"`                                       // 节点存在时间
	IP                string               `json:"ip" gorm:"comment:节点内部IP"`                                              // 节点内部IP
	PodNum            int                  `json:"pod_num" gorm:"comment:节点上的 Pod 数量"`                                    // 节点上的 Pod 数量
	CpuRequestInfo    string               `json:"cpu_request_info" gorm:"comment:CPU 请求信息，例如 500m/2"`                    // CPU 请求信息
	CpuLimitInfo      string               `json:"cpu_limit_info" gorm:"comment:CPU 限制信息，例如 1/2"`                         // CPU 限制信息
	CpuUsageInfo      string               `json:"cpu_usage_info" gorm:"comment:CPU 使用信息，例如 300m/2 (15%)"`                // CPU 使用信息
	MemoryRequestInfo string               `json:"memory_request_info" gorm:"comment:内存请求信息，例如 1Gi/8Gi"`                  // 内存请求信息
	MemoryLimitInfo   string               `json:"memory_limit_info" gorm:"comment:内存限制信息，例如 2Gi/8Gi"`                    // 内存限制信息
	MemoryUsageInfo   string               `json:"memory_usage_info" gorm:"comment:内存使用信息，例如 1.5Gi/8Gi (18.75%)"`         // 内存使用信息
	PodNumInfo        string               `json:"pod_num_info" gorm:"comment:Pod 数量信息，例如 10/50 (20%)"`                   // Pod 数量信息
	CpuCores          string               `json:"cpu_cores" gorm:"comment:CPU 核心信息，例如 2/4"`                              // CPU 核心信息
	MemGibs           string               `json:"mem_gibs" gorm:"comment:内存信息，例如 8Gi/16Gi"`                              // 内存信息
	EphemeralStorage  string               `json:"ephemeral_storage" gorm:"comment:临时存储信息，例如 100Gi/200Gi"`                // 临时存储信息
	KubeletVersion    string               `json:"kubelet_version" gorm:"comment:Kubelet 版本"`                             // Kubelet 版本
	CriVersion        string               `json:"cri_version" gorm:"comment:容器运行时接口版本"`                                  // 容器运行时接口版本
	OsVersion         string               `json:"os_version" gorm:"comment:操作系统版本"`                                      // 操作系统版本
	KernelVersion     string               `json:"kernel_version" gorm:"comment:内核版本"`                                    // 内核版本
	Labels            []string             `json:"labels" gorm:"type:text;serializer:json;comment:节点标签列表"`                // 节点标签列表
	LabelsFront       string               `json:"labels_front" gorm:"-"`                                                 // 前端显示的标签字符串，格式为多行 key=value
	TaintsFront       string               `json:"taints_front" gorm:"-"`                                                 // 前端显示的 Taints 字符串，格式为多行 key=value:Effect
	LabelPairs        map[string]string    `json:"label_pairs" gorm:"-"`                                                  // 标签键值对映射
	Annotation        map[string]string    `json:"annotation" gorm:"type:text;serializer:json;comment:注解键值对映射"`           // 注解键值对映射
	Conditions        []core.NodeCondition `json:"conditions" gorm:"-"`                                                   // 节点条件列表
	Taints            []core.Taint         `json:"taints" gorm:"-"`                                                       // 节点 Taints 列表
	Events            []OneEvent           `json:"events" gorm:"-"`                                                       // 节点相关事件列表，包含最近的事件信息
	CreatedAt         time.Time            `json:"created_at" gorm:"comment:创建时间"`                                        // 创建时间
	UpdatedAt         time.Time            `json:"updated_at" gorm:"comment:更新时间"`                                        // 更新时间
}

func (K8sNode) TableName() string {
	return "cl_k8s_nodes"
}

// LabelK8sNodesRequest 定义为节点添加标签的请求结构
type LabelK8sNodesRequest struct {
	*K8sClusterNodesRequest
	ModType string   `json:"mod_type" binding:"required,oneof=add del"` // 操作类型，必填，值为 "add" 或 "del"
	Labels  []string `json:"labels" binding:"required"`                 // 标签键值对，必填
}

// TaintK8sNodesRequest 定义为节点添加或删除 Taint 的请求结构
type TaintK8sNodesRequest struct {
	*K8sClusterNodesRequest
	ModType   string `json:"mod_type"`             // 操作类型，值为 "add" 或 "del"
	TaintYaml string `json:"taint_yaml,omitempty"` // 可选的 Taint YAML 字符串，用于验证或其他用途
}

// ScheduleK8sNodesRequest 定义调度节点的请求结构
type ScheduleK8sNodesRequest struct {
	*K8sClusterNodesRequest
	ScheduleEnable bool `json:"schedule_enable"`
}
