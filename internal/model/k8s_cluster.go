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

// ENV映射
type Env int8

const (
	EnvProd  Env = iota + 1 // 生产环境
	EnvDev                  // 开发环境
	EnvStage                // 预发环境
	EnvRc                   // 测试环境
	EnvPress                // 灰度环境
)

// Status 集群状态
type Status int8

const (
	StatusRunning Status = iota + 1 // 运行中
	StatusStopped                   // 停止
	StatusError                     // 异常
)

// K8sCluster Kubernetes 集群的配置
type K8sCluster struct {
	Model
	Name                 string                  `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:集群名称"`        // 集群名称
	CpuRequest           string                  `json:"cpu_request,omitempty" gorm:"comment:CPU 请求量 (m)"`                          // CPU 请求量
	CpuLimit             string                  `json:"cpu_limit,omitempty" gorm:"comment:CPU 限制量 (m)"`                            // CPU 限制量
	MemoryRequest        string                  `json:"memory_request,omitempty" gorm:"comment:内存请求量 (Mi)"`                        // 内存请求量
	MemoryLimit          string                  `json:"memory_limit,omitempty" gorm:"comment:内存限制量 (Mi)"`                          // 内存限制量
	RestrictNamespace    StringList              `json:"restrict_namespace" gorm:"comment:资源限制命名空间"`                                // 资源限制命名空间
	Status               Status                  `json:"status" gorm:"comment:集群状态 (1:Running, 2:Stopped, 3:Error)"`                // 集群状态
	Env                  Env                     `json:"env,omitempty" gorm:"comment:集群环境 (1:Prod, 2:Dev, 3:Stage, 4:Rc, 5:Press)"` // 集群环境
	Version              string                  `json:"version,omitempty" gorm:"comment:集群版本"`                                     // 集群版本
	ApiServerAddr        string                  `json:"api_server_addr,omitempty" gorm:"comment:API Server 地址"`                    // API Server 地址
	KubeConfigContent    string                  `json:"kube_config_content,omitempty" gorm:"type:text;comment:kubeConfig 内容"`      // kubeConfig 内容
	ActionTimeoutSeconds int                     `json:"action_timeout_seconds,omitempty" gorm:"comment:操作超时时间（秒）"`                 // 操作超时时间（秒）
	CreateUserName       string                  `json:"create_user_name,omitempty" gorm:"comment:创建者用户名"`                          // 创建者用户名
	CreateUserID         int                     `json:"create_user_id,omitempty" gorm:"comment:创建者用户ID"`                           // 创建者用户ID
	Tags                 KeyValueList            `json:"tags,omitempty" gorm:"type:text;serializer:json;comment:标签"`                // 标签
	ComponentStatus      []ComponentHealthStatus `json:"component_status" gorm:"-"`                                                 // 组件状态
	ClusterStats         ClusterStats            `json:"cluster_stats" gorm:"-"`                                                    // 集群统计信息
}

func (k8sCluster *K8sCluster) TableName() string {
	return "cl_k8s_clusters"
}

// CreateClusterReq 创建集群请求
type CreateClusterReq struct {
	Name                 string       `json:"name" binding:"required,min=1,max=200"` // 集群名称
	CpuRequest           string       `json:"cpu_request,omitempty"`                 // CPU 请求量
	CpuLimit             string       `json:"cpu_limit,omitempty"`                   // CPU 限制量
	MemoryRequest        string       `json:"memory_request,omitempty"`              // 内存请求量
	MemoryLimit          string       `json:"memory_limit,omitempty"`                // 内存限制量
	RestrictNamespace    StringList   `json:"restrict_namespace"`                    // 资源限制命名空间
	Status               Status       `json:"status"`                                // 集群状态
	Env                  Env          `json:"env,omitempty"`                         // 集群环境
	Version              string       `json:"version,omitempty"`                     // 集群版本
	ApiServerAddr        string       `json:"api_server_addr,omitempty"`             // API Server 地址
	KubeConfigContent    string       `json:"kube_config_content,omitempty"`         // kubeConfig 内容
	ActionTimeoutSeconds int          `json:"action_timeout_seconds,omitempty"`      // 操作超时时间（秒）
	CreateUserName       string       `json:"create_user_name,omitempty"`            // 创建者用户名
	CreateUserID         int          `json:"create_user_id,omitempty"`              // 创建者用户ID
	Tags                 KeyValueList `json:"tags,omitempty"`                        // 标签
}

// UpdateClusterReq 更新集群请求
type UpdateClusterReq struct {
	ID                   int          `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
	Name                 string       `json:"name" binding:"required,min=1,max=200"` // 集群名称
	CpuRequest           string       `json:"cpu_request,omitempty"`                 // CPU 请求量
	CpuLimit             string       `json:"cpu_limit,omitempty"`                   // CPU 限制量
	MemoryRequest        string       `json:"memory_request,omitempty"`              // 内存请求量
	MemoryLimit          string       `json:"memory_limit,omitempty"`                // 内存限制量
	RestrictNamespace    StringList   `json:"restrict_namespace"`                    // 资源限制命名空间
	Status               Status       `json:"status"`                                // 集群状态
	Env                  Env          `json:"env,omitempty"`                         // 集群环境
	Version              string       `json:"version,omitempty"`                     // 集群版本
	ApiServerAddr        string       `json:"api_server_addr,omitempty"`             // API Server 地址
	KubeConfigContent    string       `json:"kube_config_content,omitempty"`         // kubeConfig 内容
	ActionTimeoutSeconds int          `json:"action_timeout_seconds,omitempty"`      // 操作超时时间（秒）
	Tags                 KeyValueList `json:"tags,omitempty"`                        // 标签
}

// DeleteClusterReq 删除集群请求
type DeleteClusterReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// RefreshClusterReq 刷新集群请求
type RefreshClusterReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// CheckClusterHealthReq 检查集群健康请求
type CheckClusterHealthReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// GetClusterStatsReq 获取集群统计请求
type GetClusterStatsReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// GetClusterReq 获取单个集群请求
type GetClusterReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// ListClustersReq 获取集群列表请求
type ListClustersReq struct {
	ListReq
	Status string `json:"status" form:"status"` // 集群状态过滤
	Env    string `json:"env" form:"env"`       // 环境过滤
}

// RefreshClusterStatusReq 刷新集群状态请求
type RefreshClusterStatusReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required"`
}

// ComponentHealthStatus 组件健康状态
type ComponentHealthStatus struct {
	Name      string `json:"name"`      // 组件名称
	Status    string `json:"status"`    // 状态: healthy, unhealthy
	Message   string `json:"message"`   // 状态信息
	Timestamp string `json:"timestamp"` // 时间戳
}

// ClusterStats 集群统计信息
type ClusterStats struct {
	ClusterID      int            `json:"cluster_id"`       // 集群ID
	ClusterName    string         `json:"cluster_name"`     // 集群名称
	LastUpdateTime string         `json:"last_update_time"` // 最后更新时间
	NodeStats      NodeStats      `json:"node_stats"`       // 节点统计
	PodStats       PodStats       `json:"pod_stats"`        // Pod统计
	NamespaceStats NamespaceStats `json:"namespace_stats"`  // 命名空间统计
	WorkloadStats  WorkloadStats  `json:"workload_stats"`   // 工作负载统计
	ResourceStats  ResourceStats  `json:"resource_stats"`   // 资源统计
	StorageStats   StorageStats   `json:"storage_stats"`    // 存储统计
	NetworkStats   NetworkStats   `json:"network_stats"`    // 网络统计
	EventStats     EventStats     `json:"event_stats"`      // 事件统计
}

// NodeStats 节点统计
type NodeStats struct {
	TotalNodes    int `json:"total_nodes"`     // 总节点数
	ReadyNodes    int `json:"ready_nodes"`     // 就绪节点数
	NotReadyNodes int `json:"not_ready_nodes"` // 未就绪节点数
	MasterNodes   int `json:"master_nodes"`    // 主节点数
	WorkerNodes   int `json:"worker_nodes"`    // 工作节点数
}

// PodStats Pod统计
type PodStats struct {
	TotalPods     int `json:"total_pods"`     // 总Pod数
	RunningPods   int `json:"running_pods"`   // 运行中Pod数
	PendingPods   int `json:"pending_pods"`   // 等待中Pod数
	SucceededPods int `json:"succeeded_pods"` // 成功Pod数
	FailedPods    int `json:"failed_pods"`    // 失败Pod数
	UnknownPods   int `json:"unknown_pods"`   // 未知状态Pod数
}

// NamespaceStats 命名空间统计
type NamespaceStats struct {
	TotalNamespaces  int      `json:"total_namespaces"`  // 总命名空间数
	ActiveNamespaces int      `json:"active_namespaces"` // 活跃命名空间数
	SystemNamespaces int      `json:"system_namespaces"` // 系统命名空间数
	UserNamespaces   int      `json:"user_namespaces"`   // 用户命名空间数
	TopNamespaces    []string `json:"top_namespaces"`    // 资源使用较多的命名空间
}

// WorkloadStats 工作负载统计
type WorkloadStats struct {
	Deployments  int `json:"deployments"`  // Deployment数量
	StatefulSets int `json:"statefulsets"` // StatefulSet数量
	DaemonSets   int `json:"daemonsets"`   // DaemonSet数量
	Jobs         int `json:"jobs"`         // Job数量
	CronJobs     int `json:"cronjobs"`     // CronJob数量
	Services     int `json:"services"`     // Service数量
	ConfigMaps   int `json:"configmaps"`   // ConfigMap数量
	Secrets      int `json:"secrets"`      // Secret数量
	Ingresses    int `json:"ingresses"`    // Ingress数量
}

// ResourceStats 资源统计
type ResourceStats struct {
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

// StorageStats 存储统计
type StorageStats struct {
	TotalPV        int    `json:"total_pv"`        // 总PV数量
	BoundPV        int    `json:"bound_pv"`        // 已绑定PV数量
	AvailablePV    int    `json:"available_pv"`    // 可用PV数量
	TotalPVC       int    `json:"total_pvc"`       // 总PVC数量
	BoundPVC       int    `json:"bound_pvc"`       // 已绑定PVC数量
	PendingPVC     int    `json:"pending_pvc"`     // 等待中PVC数量
	StorageClasses int    `json:"storage_classes"` // 存储类数量
	TotalCapacity  string `json:"total_capacity"`  // 总容量
}

// NetworkStats 网络统计
type NetworkStats struct {
	Services        int `json:"services"`         // Service数量
	Endpoints       int `json:"endpoints"`        // Endpoint数量
	Ingresses       int `json:"ingresses"`        // Ingress数量
	NetworkPolicies int `json:"network_policies"` // NetworkPolicy数量
}

// EventStats 事件统计
type EventStats struct {
	TotalEvents   int `json:"total_events"`   // 总事件数
	WarningEvents int `json:"warning_events"` // 警告事件数
	NormalEvents  int `json:"normal_events"`  // 正常事件数
	RecentEvents  int `json:"recent_events"`  // 最近事件数（1小时内）
}
