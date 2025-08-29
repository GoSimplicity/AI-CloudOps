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

// K8sCluster Kubernetes 集群的配置
type K8sCluster struct {
	Model
	Name                 string     `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:集群名称"`      // 集群名称
	NameZh               string     `json:"name_zh" binding:"required,min=1,max=500" gorm:"size:100;comment:集群中文名称"` // 集群中文名称
	UserID               int        `json:"user_id" gorm:"comment:创建者用户ID"`                                          // 创建者用户ID
	CpuRequest           string     `json:"cpu_request,omitempty" gorm:"comment:CPU 请求量"`                            // CPU 请求量
	CpuLimit             string     `json:"cpu_limit,omitempty" gorm:"comment:CPU 限制量"`                              // CPU 限制量
	MemoryRequest        string     `json:"memory_request,omitempty" gorm:"comment:内存请求量"`                           // 内存请求量
	MemoryLimit          string     `json:"memory_limit,omitempty" gorm:"comment:内存限制量"`                             // 内存限制量
	RestrictedNameSpace  StringList `json:"restricted_name_space" gorm:"comment:资源限制命名空间"`                           // 资源限制命名空间
	Status               string     `json:"status" gorm:"comment:集群状态"`                                              // 集群状态
	Env                  string     `json:"env,omitempty" gorm:"comment:集群环境，例如 prod, stage, dev, rc, press"`        // 集群环境
	Version              string     `json:"version,omitempty" gorm:"comment:集群版本"`                                   // 集群版本
	ApiServerAddr        string     `json:"api_server_addr,omitempty" gorm:"comment:API Server 地址"`                  // API Server 地址
	KubeConfigContent    string     `json:"kube_config_content,omitempty" gorm:"type:text;comment:kubeConfig 内容"`    // kubeConfig 内容
	ActionTimeoutSeconds int        `json:"action_timeout_seconds,omitempty" gorm:"comment:操作超时时间（秒）"`               // 操作超时时间（秒）
}

func (k8sCluster *K8sCluster) TableName() string {
	return "cl_k8s_clusters"
}

// ClusterNamespaces 表示一个集群及其命名空间列表
type ClusterNamespaces struct {
	ClusterName string      `json:"cluster_name"` // 集群名称
	ClusterId   int         `json:"cluster_id"`   // 集群ID
	Namespaces  []Namespace `json:"namespaces"`   // 命名空间列表
}

// Namespace 命名空间响应结构体
type Namespace struct {
	Name         string    `json:"name"`                  // 命名空间名称
	UID          string    `json:"uid"`                   // 命名空间唯一标识符
	Status       string    `json:"status"`                // 命名空间状态，例如 Active
	CreationTime time.Time `json:"creation_time"`         // 创建时间
	Labels       []string  `json:"labels,omitempty"`      // 命名空间标签
	Annotations  []string  `json:"annotations,omitempty"` // 命名空间注解
}

// CreateNamespaceReq 创建新的命名空间请求结构体
type CreateNamespaceReq struct {
	ClusterId   int      `json:"cluster_id" binding:"required"`
	Name        string   `json:"namespace" binding:"required"`
	Labels      []string `json:"labels,omitempty"`      // 命名空间标签
	Annotations []string `json:"annotations,omitempty"` // 命名空间注解
}

// UpdateNamespaceReq 更新命名空间请求结构体
type UpdateNamespaceReq struct {
	ClusterId   int      `json:"cluster_id" binding:"required"`
	Name        string   `json:"namespace" binding:"required"`
	Labels      []string `json:"labels,omitempty"`      // 命名空间标签
	Annotations []string `json:"annotations,omitempty"` // 命名空间注解
}

// K8sClusterNodesReq 定义集群节点请求的基础结构
type K8sClusterNodesReq struct {
	ClusterId int    `json:"cluster_id" binding:"required"` // 集群id，必填
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称列表，必填
}

// ClusterListReq 获取集群列表请求
type ClusterListReq struct {
}

// ClusterCreateReq 创建集群请求
type ClusterCreateReq struct {
	K8sCluster
}

// ClusterUpdateReq 更新集群请求
type ClusterUpdateReq struct {
	K8sCluster
}

// ClusterDeleteReq 删除集群请求
type ClusterDeleteReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// ClusterRefreshReq 刷新集群状态请求
type ClusterRefreshReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// ClusterGetReq 获取单个集群请求
type ClusterGetReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// Resource 命名空间中的资源响应结构体
type Resource struct {
	Type         string    `json:"type"`          // 资源类型，例如 Pod, Service, Deployment
	Name         string    `json:"name"`          // 资源名称
	Namespace    string    `json:"namespace"`     // 所属命名空间
	Status       string    `json:"status"`        // 资源状态，例如 Running, Pending
	CreationTime time.Time `json:"creation_time"` // 创建时间
}

// Event 命名空间事件响应结构体
type Event struct {
	Reason         string           `json:"reason"`          // 事件原因
	Message        string           `json:"message"`         // 事件消息
	Type           string           `json:"type"`            // 事件类型，例如 Normal, Warning
	FirstTimestamp time.Time        `json:"first_timestamp"` // 第一次发生时间
	LastTimestamp  time.Time        `json:"last_timestamp"`  // 最后一次发生时间
	Count          int32            `json:"count"`           // 事件发生次数
	Source         core.EventSource `json:"source"`          // 事件来源
}

// ClusterEntity 集群响应实体
type ClusterEntity struct {
	ID                   int      `json:"id"`                     // 集群ID
	Name                 string   `json:"name"`                   // 集群名称
	NameZh               string   `json:"name_zh"`                // 集群中文名称
	UserID               int      `json:"user_id"`                // 创建者用户ID
	CpuRequest           string   `json:"cpu_request"`            // CPU请求量
	CpuLimit             string   `json:"cpu_limit"`              // CPU限制量
	MemoryRequest        string   `json:"memory_request"`         // 内存请求量
	MemoryLimit          string   `json:"memory_limit"`           // 内存限制量
	RestrictedNameSpace  []string `json:"restricted_name_space"`  // 限制的命名空间
	Status               string   `json:"status"`                 // 集群状态
	Env                  string   `json:"env"`                    // 集群环境
	Version              string   `json:"version"`                // 集群版本
	ApiServerAddr        string   `json:"api_server_addr"`        // API服务器地址
	ActionTimeoutSeconds int      `json:"action_timeout_seconds"` // 操作超时时间
	CreatedAt            string   `json:"created_at"`             // 创建时间
	UpdatedAt            string   `json:"updated_at"`             // 更新时间
}

// CreateClusterReq 创建集群请求
type CreateClusterReq struct {
	Name                 string   `json:"name" binding:"required,min=1,max=200"`    // 集群名称
	NameZh               string   `json:"name_zh" binding:"required,min=1,max=500"` // 集群中文名称
	CpuRequest           string   `json:"cpu_request"`                              // CPU请求量
	CpuLimit             string   `json:"cpu_limit"`                                // CPU限制量
	MemoryRequest        string   `json:"memory_request"`                           // 内存请求量
	MemoryLimit          string   `json:"memory_limit"`                             // 内存限制量
	RestrictedNameSpace  []string `json:"restricted_name_space"`                    // 限制的命名空间
	Env                  string   `json:"env"`                                      // 集群环境
	KubeConfigContent    string   `json:"kube_config_content" binding:"required"`   // kubeConfig内容
	ActionTimeoutSeconds int      `json:"action_timeout_seconds"`                   // 操作超时时间
}

// UpdateClusterReq 更新集群请求
type UpdateClusterReq struct {
	ID                   int      `json:"id" binding:"required,gt=0"`               // 集群ID
	Name                 string   `json:"name" binding:"required,min=1,max=200"`    // 集群名称
	NameZh               string   `json:"name_zh" binding:"required,min=1,max=500"` // 集群中文名称
	CpuRequest           string   `json:"cpu_request"`                              // CPU请求量
	CpuLimit             string   `json:"cpu_limit"`                                // CPU限制量
	MemoryRequest        string   `json:"memory_request"`                           // 内存请求量
	MemoryLimit          string   `json:"memory_limit"`                             // 内存限制量
	RestrictedNameSpace  []string `json:"restricted_name_space"`                    // 限制的命名空间
	Env                  string   `json:"env"`                                      // 集群环境
	KubeConfigContent    string   `json:"kube_config_content"`                      // kubeConfig内容
	ActionTimeoutSeconds int      `json:"action_timeout_seconds"`                   // 操作超时时间
}

// DeleteClusterReq 删除集群请求
type DeleteClusterReq struct {
	ID int `json:"id" binding:"required,gt=0"` // 集群ID
}

// GetClusterReq 获取集群请求
type GetClusterReq struct {
	ID int `json:"id" binding:"required,gt=0"` // 集群ID
}

// ListClustersReq 获取集群列表请求
type ListClustersReq struct {
	ListReq
	Status string `json:"status" form:"status"` // 集群状态过滤
	Env    string `json:"env" form:"env"`       // 环境过滤
}

// BatchDeleteClustersReq 批量删除集群请求
type BatchDeleteClustersReq struct {
	IDs []int `json:"ids" binding:"required,min=1"` // 集群ID列表
}

// RefreshClusterReq 刷新集群请求
type RefreshClusterReq struct {
	ID int `json:"id" binding:"required,gt=0"` // 集群ID
}

// ClusterHealthResponse 集群健康检查响应
type ClusterHealthResponse struct {
	ClusterID       int                     `json:"cluster_id"`       // 集群ID
	ClusterName     string                  `json:"cluster_name"`     // 集群名称
	Status          string                  `json:"status"`           // 健康状态: healthy, unhealthy, unknown
	Connected       bool                    `json:"connected"`        // 是否连接成功
	Version         string                  `json:"version"`          // K8s版本
	ApiServerAddr   string                  `json:"api_server_addr"`  // API Server地址
	NodeCount       int                     `json:"node_count"`       // 节点数量
	NamespaceCount  int                     `json:"namespace_count"`  // 命名空间数量
	LastCheckTime   string                  `json:"last_check_time"`  // 最后检查时间
	ResponseTime    string                  `json:"response_time"`    // 响应时间
	ErrorMessage    string                  `json:"error_message"`    // 错误信息
	ComponentStatus []ComponentHealthStatus `json:"component_status"` // 组件状态
	ResourceSummary ClusterResourceSummary  `json:"resource_summary"` // 资源概览
}

// ComponentHealthStatus 组件健康状态
type ComponentHealthStatus struct {
	Name      string `json:"name"`      // 组件名称
	Status    string `json:"status"`    // 状态: healthy, unhealthy
	Message   string `json:"message"`   // 状态信息
	Timestamp string `json:"timestamp"` // 时间戳
}

// ClusterResourceSummary 集群资源概览
type ClusterResourceSummary struct {
	TotalCPU    string `json:"total_cpu"`    // 总CPU
	TotalMemory string `json:"total_memory"` // 总内存
	UsedCPU     string `json:"used_cpu"`     // 已使用CPU
	UsedMemory  string `json:"used_memory"`  // 已使用内存
	TotalPods   int    `json:"total_pods"`   // 总Pod数量
	RunningPods int    `json:"running_pods"` // 运行中Pod数量
	PendingPods int    `json:"pending_pods"` // 等待中Pod数量
	FailedPods  int    `json:"failed_pods"`  // 失败Pod数量
}

// ClusterStatsResponse 集群统计信息响应
type ClusterStatsResponse struct {
	ClusterID      int                `json:"cluster_id"`       // 集群ID
	ClusterName    string             `json:"cluster_name"`     // 集群名称
	NodeStats      NodeStatsInfo      `json:"node_stats"`       // 节点统计
	PodStats       PodStatsInfo       `json:"pod_stats"`        // Pod统计
	NamespaceStats NamespaceStatsInfo `json:"namespace_stats"`  // 命名空间统计
	WorkloadStats  WorkloadStatsInfo  `json:"workload_stats"`   // 工作负载统计
	ResourceStats  ResourceStatsInfo  `json:"resource_stats"`   // 资源统计
	StorageStats   StorageStatsInfo   `json:"storage_stats"`    // 存储统计
	NetworkStats   NetworkStatsInfo   `json:"network_stats"`    // 网络统计
	EventStats     EventStatsInfo     `json:"event_stats"`      // 事件统计
	LastUpdateTime string             `json:"last_update_time"` // 最后更新时间
}

// NodeStatsInfo 节点统计信息
type NodeStatsInfo struct {
	TotalNodes    int `json:"total_nodes"`     // 总节点数
	ReadyNodes    int `json:"ready_nodes"`     // 就绪节点数
	NotReadyNodes int `json:"not_ready_nodes"` // 未就绪节点数
	MasterNodes   int `json:"master_nodes"`    // 主节点数
	WorkerNodes   int `json:"worker_nodes"`    // 工作节点数
}

// PodStatsInfo Pod统计信息
type PodStatsInfo struct {
	TotalPods     int `json:"total_pods"`     // 总Pod数
	RunningPods   int `json:"running_pods"`   // 运行中Pod数
	PendingPods   int `json:"pending_pods"`   // 等待中Pod数
	SucceededPods int `json:"succeeded_pods"` // 成功Pod数
	FailedPods    int `json:"failed_pods"`    // 失败Pod数
	UnknownPods   int `json:"unknown_pods"`   // 未知状态Pod数
}

// NamespaceStatsInfo 命名空间统计信息
type NamespaceStatsInfo struct {
	TotalNamespaces  int      `json:"total_namespaces"`  // 总命名空间数
	ActiveNamespaces int      `json:"active_namespaces"` // 活跃命名空间数
	SystemNamespaces int      `json:"system_namespaces"` // 系统命名空间数
	UserNamespaces   int      `json:"user_namespaces"`   // 用户命名空间数
	TopNamespaces    []string `json:"top_namespaces"`    // 资源使用量前几的命名空间
}

// WorkloadStatsInfo 工作负载统计信息
type WorkloadStatsInfo struct {
	Deployments  int `json:"deployments"`  // Deployment数量
	StatefulSets int `json:"statefulsets"` // StatefulSet数量
	DaemonSets   int `json:"daemonsets"`   // DaemonSet数量
	Jobs         int `json:"jobs"`         // Job数量
	CronJobs     int `json:"cronjobs"`     // CronJob数量
	Services     int `json:"services"`     // Service数量
	Ingresses    int `json:"ingresses"`    // Ingress数量
	ConfigMaps   int `json:"configmaps"`   // ConfigMap数量
	Secrets      int `json:"secrets"`      // Secret数量
}

// ResourceStatsInfo 资源统计信息
type ResourceStatsInfo struct {
	TotalCPU           string  `json:"total_cpu"`           // 总CPU
	TotalMemory        string  `json:"total_memory"`        // 总内存
	TotalStorage       string  `json:"total_storage"`       // 总存储
	UsedCPU            string  `json:"used_cpu"`            // 已使用CPU
	UsedMemory         string  `json:"used_memory"`         // 已使用内存
	UsedStorage        string  `json:"used_storage"`        // 已使用存储
	CPUUtilization     float64 `json:"cpu_utilization"`     // CPU使用率
	MemoryUtilization  float64 `json:"memory_utilization"`  // 内存使用率
	StorageUtilization float64 `json:"storage_utilization"` // 存储使用率
}

// StorageStatsInfo 存储统计信息
type StorageStatsInfo struct {
	TotalPV        int    `json:"total_pv"`        // 总PV数量
	BoundPV        int    `json:"bound_pv"`        // 已绑定PV数量
	AvailablePV    int    `json:"available_pv"`    // 可用PV数量
	TotalPVC       int    `json:"total_pvc"`       // 总PVC数量
	BoundPVC       int    `json:"bound_pvc"`       // 已绑定PVC数量
	PendingPVC     int    `json:"pending_pvc"`     // 等待中PVC数量
	StorageClasses int    `json:"storage_classes"` // 存储类数量
	TotalCapacity  string `json:"total_capacity"`  // 总容量
}

// NetworkStatsInfo 网络统计信息
type NetworkStatsInfo struct {
	Services        int `json:"services"`         // Service数量
	Endpoints       int `json:"endpoints"`        // Endpoint数量
	Ingresses       int `json:"ingresses"`        // Ingress数量
	NetworkPolicies int `json:"network_policies"` // 网络策略数量
}

// EventStatsInfo 事件统计信息
type EventStatsInfo struct {
	TotalEvents   int `json:"total_events"`   // 总事件数
	WarningEvents int `json:"warning_events"` // 警告事件数
	NormalEvents  int `json:"normal_events"`  // 正常事件数
	RecentEvents  int `json:"recent_events"`  // 最近1小时事件数
}
