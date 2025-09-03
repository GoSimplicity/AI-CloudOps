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

// NodeStatus 节点状态枚举
type NodeStatus int8

const (
	NodeStatusReady              NodeStatus = iota + 1 // 就绪
	NodeStatusNotReady                                 // 未就绪
	NodeStatusSchedulingDisabled                       // 调度禁用
	NodeStatusUnknown                                  // 未知
	NodeStatusError                                    // 异常
)

// K8sNode Kubernetes 节点
type K8sNode struct {
	Name             string               `json:"name"`                                         // 节点名称
	ClusterID        int                  `json:"cluster_id"`                                   // 所属集群ID
	Status           NodeStatus           `json:"status"`                                       // 节点状态
	Schedulable      int8                 `json:"schedulable" binding:"required,oneof=1 2"`     // 节点是否可调度
	Roles            []string             `json:"roles" gorm:"type:text;serializer:json"`       // 节点角色，例如 master, worker
	Age              string               `json:"age"`                                          // 节点存在时间，例如 5d
	InternalIP       string               `json:"internal_ip"`                                  // 节点内部IP
	ExternalIP       string               `json:"external_ip"`                                  // 节点外部IP（如果有）
	HostName         string               `json:"hostname"`                                     // 主机名
	CPU              NodeResource         `json:"cpu"`                                          // CPU 资源信息
	Memory           NodeResource         `json:"memory"`                                       // 内存资源信息
	Storage          NodeResource         `json:"storage"`                                      // 存储资源信息
	Pods             NodeResource         `json:"pods"`                                         // Pod 资源信息
	EphemeralStorage NodeResource         `json:"ephemeral_storage"`                            // 临时存储信息
	KubeletVersion   string               `json:"kubelet_version"`                              // Kubelet 版本
	KubeProxyVersion string               `json:"kube_proxy_version"`                           // KubeProxy 版本
	ContainerRuntime string               `json:"container_runtime"`                            // 容器运行时
	OperatingSystem  string               `json:"operating_system"`                             // 操作系统
	Architecture     string               `json:"architecture"`                                 // 系统架构
	KernelVersion    string               `json:"kernel_version"`                               // 内核版本
	OSImage          string               `json:"os_image"`                                     // 操作系统镜像
	Labels           map[string]string    `json:"labels" gorm:"type:text;serializer:json"`      // 节点标签
	Annotations      map[string]string    `json:"annotations" gorm:"type:text;serializer:json"` // 节点注解
	Conditions       []core.NodeCondition `json:"conditions" gorm:"type:text;serializer:json"`  // 节点条件
	Taints           []core.Taint         `json:"taints" gorm:"type:text;serializer:json"`      // 节点污点
	Events           []NodeEvent          `json:"events" gorm:"type:text;serializer:json"`      // 节点相关事件
	CreatedAt        time.Time            `json:"created_at"`                                   // 创建时间
	UpdatedAt        time.Time            `json:"updated_at"`                                   // 更新时间
	RawNode          *core.Node           `json:"-"`                                            // 原始 Node 对象，不序列化到 JSON
}

// NodeResource 资源信息结构
type NodeResource struct {
	Used     string  `json:"used"`     // 已使用量
	Total    string  `json:"total"`    // 总量
	Percent  float64 `json:"percent"`  // 使用百分比
	Requests string  `json:"requests"` // 请求量
	Limits   string  `json:"limits"`   // 限制量
}

// NodeEvent 节点事件
type NodeEvent struct {
	Type           string    `json:"type"`            // 事件类型 (Normal, Warning)
	Reason         string    `json:"reason"`          // 事件原因
	Message        string    `json:"message"`         // 事件消息
	Component      string    `json:"component"`       // 事件来源组件
	Host           string    `json:"host"`            // 主机
	FirstTimestamp time.Time `json:"first_timestamp"` // 首次发生时间
	LastTimestamp  time.Time `json:"last_timestamp"`  // 最后发生时间
	Count          int32     `json:"count"`           // 发生次数
}

// NodeTaintEntity 节点污点实体
type NodeTaintEntity struct {
	Key    string `json:"key" binding:"required"`    // 污点键
	Value  string `json:"value"`                     // 污点值
	Effect string `json:"effect" binding:"required"` // 污点效果: NoSchedule, PreferNoSchedule, NoExecute
}

// NodeMetrics 节点指标信息

// GetNodeListReq 获取节点列表请求
type GetNodeListReq struct {
	ListReq
	ClusterID     int          `json:"cluster_id" binding:"required"` // 集群ID
	Status        []NodeStatus `json:"status"`                        // 状态过滤
	LabelSelector string       `json:"label_selector"`                // 标签选择器
}

// GetNodeDetailReq 获取节点详情请求
type GetNodeDetailReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称
}

// AddLabelNodesReq 添加节点标签请求
type AddLabelNodesReq struct {
	ClusterID int               `json:"cluster_id" binding:"required"`          // 集群ID
	NodeName  string            `json:"node_name" binding:"required"`           // 节点名称
	Labels    map[string]string `json:"labels" binding:"required"`              // 要添加的标签
	Overwrite int8              `json:"overwrite" binding:"required,oneof=1 2"` // 是否覆盖已存在的标签
}

// DeleteLabelNodesReq 删除节点标签请求
type DeleteLabelNodesReq struct {
	ClusterID int      `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string   `json:"node_name" binding:"required"`  // 节点名称
	LabelKeys []string `json:"label_keys" binding:"required"` // 要删除的标签键
}

// GetNodeResourceReq 获取节点资源请求
type GetNodeResourceReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name"`                     // 节点名称列表（可选，为空则获取所有节点）
}

// GetNodeEventsReq 获取节点事件请求
type GetNodeEventsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称
	Limit     int    `json:"limit"`                         // 事件数量限制
}

// DrainNodeReq 驱逐节点请求
type DrainNodeReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required"`                   // 集群ID
	NodeName           string `json:"node_name" binding:"required"`                    // 节点名称
	Force              int8   `json:"force" binding:"required,oneof=1 2"`              // 是否强制驱逐
	IgnoreDaemonSets   int8   `json:"ignore_daemon_sets" binding:"required,oneof=1 2"` // 是否忽略DaemonSet
	DeleteLocalData    int8   `json:"delete_local_data" binding:"required,oneof=1 2"`  // 是否删除本地数据
	GracePeriodSeconds int    `json:"grace_period_seconds"`                            // 优雅关闭时间(秒)
	TimeoutSeconds     int    `json:"timeout_seconds"`                                 // 超时时间(秒)
}

// NodeCordonReq 禁止节点调度请求
type NodeCordonReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称
}

// NodeUncordonReq 解除节点调度限制请求
type NodeUncordonReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称
}

// GetNodeTaintsReq 获取节点污点请求
type GetNodeTaintsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称
}

// AddNodeTaintsReq 添加节点污点请求
type AddNodeTaintsReq struct {
	ClusterID int               `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string            `json:"node_name" binding:"required"`  // 节点名称
	Taints    []NodeTaintEntity `json:"taints" binding:"required"`     // 要添加的污点
}

// DeleteNodeTaintsReq 删除节点污点请求
type DeleteNodeTaintsReq struct {
	ClusterID int      `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string   `json:"node_name" binding:"required"`  // 节点名称
	TaintKeys []string `json:"taint_keys" binding:"required"` // 要删除的污点键
}

// CheckTaintYamlReq 检查污点YAML配置请求
type CheckTaintYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称
	YamlData  string `json:"yaml_data" binding:"required"`  // YAML数据
}

// SwitchNodeScheduleReq 切换节点调度状态请求
type SwitchNodeScheduleReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"`       // 集群ID
	NodeName  string `json:"node_name" binding:"required"`        // 节点名称
	Enable    int8   `json:"enable" binding:"required,oneof=1 2"` // 是否启用调度
}

// GetNodeMetricsReq 获取节点指标请求
type GetNodeMetricsReq struct {
	ClusterID int      `json:"cluster_id" binding:"required"` // 集群ID
	NodeNames []string `json:"node_names"`                    // 节点名称列表（可选）
}
