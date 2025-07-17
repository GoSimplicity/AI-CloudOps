package model

import (
	appsv1 "k8s.io/api/apps/v1"
	"time"
)

// K8sDaemonSetRequest DaemonSet 相关请求结构
type K8sDaemonSetRequest struct {
	ClusterID      int               `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace      string            `json:"namespace" binding:"required"`  // 命名空间，必填
	DaemonSetNames []string          `json:"daemonset_names"`               // DaemonSet 名称，可选
	DaemonSetYaml  *appsv1.DaemonSet `json:"daemonset_yaml"`                // DaemonSet 对象, 可选
}

// K8sDaemonSetStatus DaemonSet 状态响应
type K8sDaemonSetStatus struct {
	Name                   string    `json:"name"`                     // DaemonSet 名称
	Namespace              string    `json:"namespace"`                // 命名空间
	DesiredNumberScheduled int32     `json:"desired_number_scheduled"` // 期望调度的节点数
	CurrentNumberScheduled int32     `json:"current_number_scheduled"` // 当前调度的节点数
	NumberReady            int32     `json:"number_ready"`             // 就绪的Pod数
	UpdatedNumberScheduled int32     `json:"updated_number_scheduled"` // 已更新的调度数
	NumberAvailable        int32     `json:"number_available"`         // 可用的Pod数
	NumberUnavailable      int32     `json:"number_unavailable"`       // 不可用的Pod数
	NumberMisscheduled     int32     `json:"number_misscheduled"`      // 错误调度的Pod数
	ObservedGeneration     int64     `json:"observed_generation"`      // 观察到的代数
	CreationTimestamp      time.Time `json:"creation_timestamp"`       // 创建时间
}
