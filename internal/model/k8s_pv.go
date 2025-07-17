package model

import (
	core "k8s.io/api/core/v1"
	"time"
)

// K8sPVRequest PersistentVolume 相关请求结构
type K8sPVRequest struct {
	ClusterID int                    `json:"cluster_id" binding:"required"` // 集群名称，必填
	PVNames   []string               `json:"pv_names"`                      // PV 名称，可选
	PVYaml    *core.PersistentVolume `json:"pv_yaml"`                       // PV 对象, 可选
}

// K8sPVStatus PersistentVolume 状态响应
type K8sPVStatus struct {
	Name              string                             `json:"name"`               // PV 名称
	Capacity          map[core.ResourceName]string       `json:"capacity"`           // 容量
	Phase             core.PersistentVolumePhase         `json:"phase"`              // 阶段 (Available, Bound, Released, Failed)
	ClaimRef          *core.ObjectReference              `json:"claim_ref"`          // 绑定的 PVC 引用
	ReclaimPolicy     core.PersistentVolumeReclaimPolicy `json:"reclaim_policy"`     // 回收策略
	StorageClass      string                             `json:"storage_class"`      // 存储类
	VolumeMode        *core.PersistentVolumeMode         `json:"volume_mode"`        // 卷模式
	AccessModes       []core.PersistentVolumeAccessMode  `json:"access_modes"`       // 访问模式
	CreationTimestamp time.Time                          `json:"creation_timestamp"` // 创建时间
}
