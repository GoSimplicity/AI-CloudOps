package model

import (
	appsv1 "k8s.io/api/apps/v1"
	"time"
)

// K8sStatefulSetRequest StatefulSet 相关请求结构
type K8sStatefulSetRequest struct {
	ClusterID        int                 `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace        string              `json:"namespace" binding:"required"`  // 命名空间，必填
	StatefulSetNames []string            `json:"statefulset_names"`             // StatefulSet 名称，可选
	StatefulSetYaml  *appsv1.StatefulSet `json:"statefulset_yaml"`              // StatefulSet 对象, 可选
}

// K8sStatefulSetScaleRequest StatefulSet 扩缩容请求结构
type K8sStatefulSetScaleRequest struct {
	ClusterID       int    `json:"cluster_id" binding:"required"`       // 集群名称，必填
	Namespace       string `json:"namespace" binding:"required"`        // 命名空间，必填
	StatefulSetName string `json:"statefulset_name" binding:"required"` // StatefulSet 名称，必填
	Replicas        int32  `json:"replicas" binding:"required"`         // 副本数量，必填
}

// K8sStatefulSetStatus StatefulSet 状态响应
type K8sStatefulSetStatus struct {
	Name               string    `json:"name"`                // StatefulSet 名称
	Namespace          string    `json:"namespace"`           // 命名空间
	Replicas           int32     `json:"replicas"`            // 期望副本数
	ReadyReplicas      int32     `json:"ready_replicas"`      // 就绪副本数
	CurrentReplicas    int32     `json:"current_replicas"`    // 当前副本数
	UpdatedReplicas    int32     `json:"updated_replicas"`    // 已更新副本数
	AvailableReplicas  int32     `json:"available_replicas"`  // 可用副本数
	CurrentRevision    string    `json:"current_revision"`    // 当前修订版本
	UpdateRevision     string    `json:"update_revision"`     // 更新修订版本
	ObservedGeneration int64     `json:"observed_generation"` // 观察到的代数
	CreationTimestamp  time.Time `json:"creation_timestamp"`  // 创建时间
}
