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

import "time"

type ContainerCore struct {
	Name       string            `json:"name,omitempty" gorm:"comment:容器名称"`                      // 容器名称
	CPU        string            `json:"cpu,omitempty" gorm:"comment:CPU 资源限制"`                   // CPU 资源限制(如 "100m", "0.5")
	Memory     string            `json:"memory,omitempty" gorm:"comment:内存资源限制"`                  // 内存资源限制(如 "512Mi", "2Gi")
	CPURequest string            `json:"cpu_request,omitempty" gorm:"comment:CPU 资源请求"`           // CPU 资源请求
	MemRequest string            `json:"mem_request,omitempty" gorm:"comment:内存资源请求"`             // 内存资源请求
	Command    []string          `json:"command,omitempty" gorm:"serializer:json;comment:容器启动命令"` // 容器启动命令
	Args       []string          `json:"args,omitempty" gorm:"serializer:json;comment:容器启动参数"`    // 容器启动参数
	Envs       map[string]string `json:"envs,omitempty" gorm:"serializer:json;comment:环境变量"`      // 环境变量
	PullPolicy string            `json:"pull_policy,omitempty" gorm:"comment:镜像拉取策略"`             // 镜像拉取策略
	Volumes    []Volume          `json:"volumes,omitempty" gorm:"serializer:json;comment:挂载卷"`    // 挂载卷
}

type OneEvent struct {
	Type      string `json:"type"`       // 事件类型，例如 "Normal", "Warning"
	Component string `json:"component"`  // 事件的组件来源，例如 "kubelet"
	Reason    string `json:"reason"`     // 事件的原因，例如 "NodeReady"
	Message   string `json:"message"`    // 事件的详细消息
	FirstTime string `json:"first_time"` // 事件第一次发生的时间，例如 "2024-04-27T10:00:00Z"
	LastTime  string `json:"last_time"`  // 事件最近一次发生的时间，例如 "2024-04-27T12:00:00Z"
	Object    string `json:"object"`     // 事件关联的对象信息，例如 "kind:Node name:node-1"
	Count     int    `json:"count"`      // 事件发生的次数
}

type NodeResources struct {
	NodeName string `json:"node_name"` // 节点名称
	CPU      string `json:"cpu"`       // CPU总量
	Memory   string `json:"memory"`    // 内存总量
	Storage  string `json:"storage"`   // 存储总量
	Pods     string `json:"pods"`      // Pod总量
	Status   string `json:"status"`    // 节点状态
	Ready    bool   `json:"ready"`     // 节点是否就绪
}

type Taint struct {
	Key    string `json:"key" binding:"required"`                                                // Taint 的键
	Value  string `json:"value,omitempty"`                                                       // Taint 的值
	Effect string `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"` // Taint 的效果，例如 "NoSchedule", "PreferNoSchedule", "NoExecute"
}

type ResourceRequirements struct {
	Requests K8sResourceList `json:"requests,omitempty" gorm:"type:text;serializer:json;comment:资源请求"` // 资源请求
	Limits   K8sResourceList `json:"limits,omitempty" gorm:"type:text;serializer:json;comment:资源限制"`   // 资源限制
}

type K8sResourceList struct {
	CPU    string `json:"cpu,omitempty" gorm:"size:50;comment:CPU 数量，例如 '500m', '2'"`     // CPU 数量，例如 "500m", "2"
	Memory string `json:"memory,omitempty" gorm:"size:50;comment:内存数量，例如 '1Gi', '512Mi'"` // 内存数量，例如 "1Gi", "512Mi"
}

type KeyValueItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BatchDeleteReq struct {
	IDs []int `json:"ids" binding:"required"`
}

type Volume struct {
	Name       string `json:"name"`                  // 卷名称
	Type       string `json:"type"`                  // 卷类型(ConfigMap, Secret, PVC, EmptyDir等)
	MountPath  string `json:"mount_path"`            // 挂载路径
	SubPath    string `json:"sub_path,omitempty"`    // 子路径
	ReadOnly   bool   `json:"read_only,omitempty"`   // 是否只读
	SourceName string `json:"source_name,omitempty"` // 源资源名称(如ConfigMap名称)
	Size       string `json:"size,omitempty"`        // 存储大小
}

// DaemonSetUpdateStrategy DaemonSet更新策略
type DaemonSetUpdateStrategy struct {
	Type          string                          `json:"type" gorm:"size:50;comment:更新策略类型"`              // 更新策略类型
	RollingUpdate *DaemonSetRollingUpdateStrategy `json:"rolling_update" gorm:"type:text;serializer:json"` // 滚动更新策略
}

// DaemonSetRollingUpdateStrategy DaemonSet滚动更新策略
type DaemonSetRollingUpdateStrategy struct {
	MaxUnavailable *int32 `json:"max_unavailable" gorm:"comment:最大不可用数量"` // 最大不可用数量
	MaxSurge       *int32 `json:"max_surge" gorm:"comment:最大超出数量"`        // 最大超出数量
}

// K8sTLSTestResult TLS测试结果
type K8sTLSTestResult struct {
	Host       string    `json:"host"`        // 主机名
	Valid      bool      `json:"valid"`       // 证书是否有效
	Issuer     string    `json:"issuer"`      // 证书颁发者
	Subject    string    `json:"subject"`     // 证书主题
	ExpiryDate time.Time `json:"expiry_date"` // 证书过期时间
	ErrorMsg   string    `json:"error_msg"`   // 错误信息
}

// K8sBackendHealth 后端健康状态
type K8sBackendHealth struct {
	ServiceName string `json:"service_name"` // 服务名称
	ServicePort int    `json:"service_port"` // 服务端口
	Healthy     bool   `json:"healthy"`      // 是否健康
	Message     string `json:"message"`      // 状态信息
}

// EventSource 事件源
type EventSource struct {
	Component string `json:"component" gorm:"size:100;comment:组件名称"` // 组件名称
	Host      string `json:"host" gorm:"size:200;comment:主机名"`       // 主机名
}

// EventInvolvedObject 事件涉及的对象
type EventInvolvedObject struct {
	Kind       string `json:"kind" gorm:"size:50;comment:对象类型"`          // 对象类型
	Name       string `json:"name" gorm:"size:200;comment:对象名称"`         // 对象名称
	Namespace  string `json:"namespace" gorm:"size:100;comment:命名空间"`    // 命名空间
	UID        string `json:"uid" gorm:"size:100;comment:对象UID"`         // 对象UID
	APIVersion string `json:"api_version" gorm:"size:100;comment:API版本"` // API版本
}

// K8sEventStatistics 事件统计
type K8sEventStatistics struct {
	TotalEvents   int                `json:"total_events"`   // 总事件数
	WarningEvents int                `json:"warning_events"` // 警告事件数
	NormalEvents  int                `json:"normal_events"`  // 正常事件数
	TopReasons    []EventReasonCount `json:"top_reasons"`    // 主要原因统计
	TopSources    []EventSourceCount `json:"top_sources"`    // 主要来源统计
}

// EventReasonCount 事件原因统计
type EventReasonCount struct {
	Reason string `json:"reason"` // 原因
	Count  int    `json:"count"`  // 数量
}

// EventSourceCount 事件来源统计
type EventSourceCount struct {
	Source string `json:"source"` // 来源
	Count  int    `json:"count"`  // 数量
}

// K8sEventTimelineItem 事件时间线项
type K8sEventTimelineItem struct {
	Timestamp time.Time `json:"timestamp"` // 时间戳
	Type      string    `json:"type"`      // 事件类型
	Reason    string    `json:"reason"`    // 原因
	Message   string    `json:"message"`   // 消息
	Object    string    `json:"object"`    // 对象
}

// K8sEventCleanupResult 事件清理结果
type K8sEventCleanupResult struct {
	CleanedCount int      `json:"cleaned_count"` // 清理数量
	ErrorCount   int      `json:"error_count"`   // 错误数量
	Errors       []string `json:"errors"`        // 错误列表
}

// K8sPVUsageInfo PV使用信息
type K8sPVUsageInfo struct {
	Total     string  `json:"total"`      // 总容量
	Used      string  `json:"used"`       // 已使用
	Available string  `json:"available"`  // 可用
	UsageRate float64 `json:"usage_rate"` // 使用率
}

// K8sPVCUsageInfo PVC使用信息
type K8sPVCUsageInfo struct {
	Total     string  `json:"total"`      // 总容量
	Used      string  `json:"used"`       // 已使用
	Available string  `json:"available"`  // 可用
	UsageRate float64 `json:"usage_rate"` // 使用率
}

// ====================== 通用历史版本结构体 ======================

// K8sResourceHistory K8s资源历史版本信息
type K8sResourceHistory struct {
	Revision      int64             `json:"revision"`       // 版本号
	ChangeTime    time.Time         `json:"change_time"`    // 变更时间
	ChangeCause   string            `json:"change_cause"`   // 变更原因
	Status        string            `json:"status"`         // 状态
	Annotations   map[string]string `json:"annotations"`    // 注解
	Labels        map[string]string `json:"labels"`         // 标签
	ResourceType  string            `json:"resource_type"`  // 资源类型 (deployment, daemonset, statefulset等)
	ResourceName  string            `json:"resource_name"`  // 资源名称
	Namespace     string            `json:"namespace"`      // 命名空间
	ConfigChanges string            `json:"config_changes"` // 配置变更内容
}
