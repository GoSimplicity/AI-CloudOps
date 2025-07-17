package model

import (
	core "k8s.io/api/core/v1"
	"time"
)

// K8sPVCRequest PersistentVolumeClaim 相关请求结构
type K8sPVCRequest struct {
	ClusterID int                         `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace string                      `json:"namespace" binding:"required"`  // 命名空间，必填
	PVCNames  []string                    `json:"pvc_names"`                     // PVC 名称，可选
	PVCYaml   *core.PersistentVolumeClaim `json:"pvc_yaml"`                      // PVC 对象, 可选
}

// K8sPVCStatus PersistentVolumeClaim 状态响应
type K8sPVCStatus struct {
	Name              string                            `json:"name"`               // PVC 名称
	Namespace         string                            `json:"namespace"`          // 命名空间
	Phase             core.PersistentVolumeClaimPhase   `json:"phase"`              // 阶段 (Pending, Bound, Lost)
	VolumeName        string                            `json:"volume_name"`        // 绑定的 PV 名称
	Capacity          map[core.ResourceName]string      `json:"capacity"`           // 分配的容量
	RequestedStorage  string                            `json:"requested_storage"`  // 请求的存储容量
	StorageClass      *string                           `json:"storage_class"`      // 存储类
	VolumeMode        *core.PersistentVolumeMode        `json:"volume_mode"`        // 卷模式
	AccessModes       []core.PersistentVolumeAccessMode `json:"access_modes"`       // 访问模式
	CreationTimestamp time.Time                         `json:"creation_timestamp"` // 创建时间
}
