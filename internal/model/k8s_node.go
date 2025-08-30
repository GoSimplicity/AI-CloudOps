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

// LabelK8sNodesReq 定义为节点添加标签的请求结构
type LabelK8sNodesReq struct {
	NodeName  string   `json:"node_name" binding:"required"`              // 节点名称，必填
	ClusterID int      `json:"cluster_id" binding:"required"`             // 集群ID，必填
	ModType   string   `json:"mod_type" binding:"required,oneof=add del"` // 操作类型，必填，值为 "add" 或 "del"
	Labels    []string `json:"labels" binding:"required"`                 // 标签键值对，必填
}

// TaintK8sNodesReq 定义为节点添加或删除 Taint 的请求结构
type TaintK8sNodesReq struct {
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称，必填
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID，必填
	ModType   string `json:"mod_type"`                      // 操作类型，值为 "add" 或 "del"
	TaintYaml string `json:"taint_yaml,omitempty"`          // 可选的 Taint YAML 字符串，用于验证或其他用途
}

// ScheduleK8sNodesReq 定义调度节点的请求结构
type ScheduleK8sNodesReq struct {
	NodeName       string `json:"node_name" binding:"required"`  // 节点名称，必填
	ClusterID      int    `json:"cluster_id" binding:"required"` // 集群ID，必填
	ScheduleEnable bool   `json:"schedule_enable"`
}

// NodeListReq 获取节点列表请求
type NodeListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
}

// NodeGetReq 获取单个节点请求
type NodeGetReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" form:"node_name" uri:"node_name" binding:"required" comment:"节点名称"`
}

// NodeCordonReq 封锁节点请求
type NodeCordonReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" binding:"required" comment:"节点名称"`
}

// NodeUncordonReq 解封节点请求
type NodeUncordonReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" binding:"required" comment:"节点名称"`
}

// NodeDrainReq 排空节点请求
type NodeDrainReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName           string `json:"node_name" binding:"required" comment:"节点名称"`
	Force              bool   `json:"force" comment:"是否强制排空"`
	DeleteLocalData    bool   `json:"delete_local_data" comment:"是否删除本地数据"`
	IgnoreDaemonsets   bool   `json:"ignore_daemonsets" comment:"是否忽略DaemonSet"`
	GracePeriodSeconds int    `json:"grace_period_seconds" comment:"优雅关闭时间"`
	Timeout            int    `json:"timeout" comment:"超时时间"`
}

type NodeResourcesReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" binding:"required" comment:"节点名称"`
}

type NodeEventsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" binding:"required" comment:"节点名称"`
}

// K8sClusterNodesReq 定义集群节点操作的请求结构
type K8sClusterNodesReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" binding:"required" comment:"节点名称"`
}

// ====================== Node响应实体 ======================

// NodeEntity Node响应实体
type NodeEntity struct {
	Name                    string                `json:"name"`                      // 节点名称
	UID                     string                `json:"uid"`                       // 节点UID
	Labels                  map[string]string     `json:"labels"`                    // 标签
	Annotations             map[string]string     `json:"annotations"`               // 注解
	Status                  string                `json:"status"`                    // 节点状态
	ScheduleEnable          bool                  `json:"schedule_enable"`           // 是否可调度
	Roles                   []string              `json:"roles"`                     // 节点角色
	Age                     string                `json:"age"`                       // 存在时间
	InternalIP              string                `json:"internal_ip"`               // 内部IP
	ExternalIP              string                `json:"external_ip"`               // 外部IP
	Hostname                string                `json:"hostname"`                  // 主机名
	KubeletVersion          string                `json:"kubelet_version"`           // Kubelet版本
	KubeProxyVersion        string                `json:"kube_proxy_version"`        // KubeProxy版本
	ContainerRuntimeVersion string                `json:"container_runtime_version"` // 容器运行时版本
	OperatingSystem         string                `json:"operating_system"`          // 操作系统
	Architecture            string                `json:"architecture"`              // 架构
	KernelVersion           string                `json:"kernel_version"`            // 内核版本
	OSImage                 string                `json:"os_image"`                  // 操作系统镜像
	Conditions              []NodeConditionEntity `json:"conditions"`                // 节点条件
	Taints                  []NodeTaintEntity     `json:"taints"`                    // 节点污点
	Resources               NodeResourcesEntity   `json:"resources"`                 // 资源信息
	PodCIDR                 string                `json:"pod_cidr"`                  // Pod CIDR
	PodCIDRs                []string              `json:"pod_cidrs"`                 // Pod CIDR列表
	ProviderID              string                `json:"provider_id"`               // 提供商ID
	CreatedAt               string                `json:"created_at"`                // 创建时间
}

// NodeConditionEntity 节点条件实体
type NodeConditionEntity struct {
	Type               string `json:"type"`                 // 条件类型
	Status             string `json:"status"`               // 条件状态
	LastHeartbeatTime  string `json:"last_heartbeat_time"`  // 最后心跳时间
	LastTransitionTime string `json:"last_transition_time"` // 最后转换时间
	Reason             string `json:"reason"`               // 原因
	Message            string `json:"message"`              // 消息
}

// NodeTaintEntity 节点污点实体
type NodeTaintEntity struct {
	Key       string `json:"key"`        // 键
	Value     string `json:"value"`      // 值
	Effect    string `json:"effect"`     // 效果
	TimeAdded string `json:"time_added"` // 添加时间
}

// NodeResourcesEntity 节点资源实体
type NodeResourcesEntity struct {
	Capacity    NodeResourceMapEntity `json:"capacity"`    // 资源容量
	Allocatable NodeResourceMapEntity `json:"allocatable"` // 可分配资源
	Usage       NodeResourceMapEntity `json:"usage"`       // 资源使用量
	Requests    NodeResourceMapEntity `json:"requests"`    // 资源请求量
	Limits      NodeResourceMapEntity `json:"limits"`      // 资源限制量
}

// NodeResourceMapEntity 节点资源映射实体
type NodeResourceMapEntity struct {
	CPU              string            `json:"cpu"`               // CPU
	Memory           string            `json:"memory"`            // 内存
	Storage          string            `json:"storage"`           // 存储
	EphemeralStorage string            `json:"ephemeral_storage"` // 临时存储
	Pods             string            `json:"pods"`              // Pod数量
	HugePagesSize    map[string]string `json:"hugepages_size"`    // 大页内存
}

// NodeListResponse Node列表响应
type NodeListResponse struct {
	Items      []NodeEntity `json:"items"`       // Node列表
	TotalCount int          `json:"total_count"` // 总数
}

// NodeDetailResponse Node详情响应
type NodeDetailResponse struct {
	Node    NodeEntity        `json:"node"`    // Node信息
	YAML    string            `json:"yaml"`    // YAML内容
	Events  []NodeEventEntity `json:"events"`  // 事件列表
	Pods    []PodEntity       `json:"pods"`    // 节点上的Pod列表
	Metrics NodeMetricsEntity `json:"metrics"` // 节点指标
}

// NodeEventEntity Node事件实体
type NodeEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// NodeMetricsEntity 节点指标实体
type NodeMetricsEntity struct {
	CPU        NodeResourceMetricsEntity   `json:"cpu"`        // CPU指标
	Memory     NodeResourceMetricsEntity   `json:"memory"`     // 内存指标
	Storage    NodeResourceMetricsEntity   `json:"storage"`    // 存储指标
	Network    NodeNetworkMetricsEntity    `json:"network"`    // 网络指标
	Filesystem NodeFilesystemMetricsEntity `json:"filesystem"` // 文件系统指标
	Timestamp  string                      `json:"timestamp"`  // 指标时间戳
}

// NodeResourceMetricsEntity 节点资源指标
type NodeResourceMetricsEntity struct {
	Usage      string  `json:"usage"`      // 使用量
	Capacity   string  `json:"capacity"`   // 容量
	Available  string  `json:"available"`  // 可用量
	Percentage float64 `json:"percentage"` // 使用百分比
}

// NodeNetworkMetricsEntity 节点网络指标
type NodeNetworkMetricsEntity struct {
	RxBytes   int64 `json:"rx_bytes"`   // 接收字节数
	TxBytes   int64 `json:"tx_bytes"`   // 发送字节数
	RxPackets int64 `json:"rx_packets"` // 接收包数
	TxPackets int64 `json:"tx_packets"` // 发送包数
	RxErrors  int64 `json:"rx_errors"`  // 接收错误数
	TxErrors  int64 `json:"tx_errors"`  // 发送错误数
}

// NodeFilesystemMetricsEntity 节点文件系统指标
type NodeFilesystemMetricsEntity struct {
	AvailableBytes int64   `json:"available_bytes"` // 可用字节数
	CapacityBytes  int64   `json:"capacity_bytes"`  // 容量字节数
	UsedBytes      int64   `json:"used_bytes"`      // 已用字节数
	UsagePercent   float64 `json:"usage_percent"`   // 使用百分比
}

// NodeCordonResponse 封锁节点响应
type NodeCordonResponse struct {
	NodeName string `json:"node_name"` // 节点名称
	Status   string `json:"status"`    // 操作状态
	Message  string `json:"message"`   // 操作消息
}

// NodeUncordonResponse 解封节点响应
type NodeUncordonResponse struct {
	NodeName string `json:"node_name"` // 节点名称
	Status   string `json:"status"`    // 操作状态
	Message  string `json:"message"`   // 操作消息
}

// NodeDrainResponse 排空节点响应
type NodeDrainResponse struct {
	NodeName    string   `json:"node_name"`    // 节点名称
	DrainedPods []string `json:"drained_pods"` // 被排空的Pod列表
	SkippedPods []string `json:"skipped_pods"` // 跳过的Pod列表
	Status      string   `json:"status"`       // 排空状态
	Message     string   `json:"message"`      // 排空消息
	Duration    string   `json:"duration"`     // 排空耗时
}

// NodeLabelResponse 节点标签操作响应
type NodeLabelResponse struct {
	NodeName      string            `json:"node_name"`      // 节点名称
	Operation     string            `json:"operation"`      // 操作类型(add/remove)
	Labels        map[string]string `json:"labels"`         // 操作的标签
	CurrentLabels map[string]string `json:"current_labels"` // 当前标签
	Status        string            `json:"status"`         // 操作状态
	Message       string            `json:"message"`        // 操作消息
}

// NodeTaintResponse 节点污点操作响应
type NodeTaintResponse struct {
	NodeName      string            `json:"node_name"`      // 节点名称
	Operation     string            `json:"operation"`      // 操作类型(add/remove)
	Taints        []NodeTaintEntity `json:"taints"`         // 操作的污点
	CurrentTaints []NodeTaintEntity `json:"current_taints"` // 当前污点
	Status        string            `json:"status"`         // 操作状态
	Message       string            `json:"message"`        // 操作消息
}
