package model

import (
	core "k8s.io/api/core/v1"
	"time"
)

// K8sConfigMapVersionRequest ConfigMap 版本管理请求
type K8sConfigMapVersionRequest struct {
	ClusterID     int             `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string          `json:"namespace" binding:"required"`      // 命名空间
	ConfigMapName string          `json:"configmap_name" binding:"required"` // ConfigMap 名称
	Version       string          `json:"version"`                           // 版本号
	Description   string          `json:"description"`                       // 版本描述
	ConfigMap     *core.ConfigMap `json:"config_map"`                        // ConfigMap 对象
}

// K8sConfigMapVersion ConfigMap 版本信息
type K8sConfigMapVersion struct {
	Version           string          `json:"version"`            // 版本号
	Description       string          `json:"description"`        // 版本描述
	ConfigMap         *core.ConfigMap `json:"config_map"`         // ConfigMap 对象
	CreationTimestamp time.Time       `json:"creation_timestamp"` // 创建时间
	Author            string          `json:"author"`             // 创建者
}

// K8sConfigMapHotReloadRequest ConfigMap 热更新请求
type K8sConfigMapHotReloadRequest struct {
	ClusterID      int               `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace      string            `json:"namespace" binding:"required"`      // 命名空间
	ConfigMapName  string            `json:"configmap_name" binding:"required"` // ConfigMap 名称
	ReloadType     string            `json:"reload_type"`                       // 重载类型 (pods, deployments, all)
	TargetSelector map[string]string `json:"target_selector"`                   // 目标选择器
}

// K8sConfigMapRollbackRequest ConfigMap 回滚请求
type K8sConfigMapRollbackRequest struct {
	ClusterID     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	ConfigMapName string `json:"configmap_name" binding:"required"` // ConfigMap 名称
	TargetVersion string `json:"target_version" binding:"required"` // 目标版本
}
