package model

import (
	core "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"time"
)

// K8sStorageClassRequest StorageClass 相关请求结构
type K8sStorageClassRequest struct {
	ClusterID         int                     `json:"cluster_id" binding:"required"` // 集群名称，必填
	StorageClassNames []string                `json:"storage_class_names"`           // StorageClass 名称，可选
	StorageClassYaml  *storagev1.StorageClass `json:"storage_class_yaml"`            // StorageClass 对象, 可选
}

// K8sStorageClassStatus StorageClass 状态响应
type K8sStorageClassStatus struct {
	Name                 string                              `json:"name"`                   // StorageClass 名称
	Provisioner          string                              `json:"provisioner"`            // 存储提供者
	Parameters           map[string]string                   `json:"parameters"`             // 存储参数
	ReclaimPolicy        *core.PersistentVolumeReclaimPolicy `json:"reclaim_policy"`         // 回收策略
	VolumeBindingMode    *storagev1.VolumeBindingMode        `json:"volume_binding_mode"`    // 卷绑定模式
	AllowVolumeExpansion *bool                               `json:"allow_volume_expansion"` // 是否允许卷扩展
	CreationTimestamp    time.Time                           `json:"creation_timestamp"`     // 创建时间
}
