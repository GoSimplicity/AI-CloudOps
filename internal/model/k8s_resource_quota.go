package model

import (
	core "k8s.io/api/core/v1"
	"time"
)

// K8sResourceQuotaRequest ResourceQuota 相关请求结构
type K8sResourceQuotaRequest struct {
	ClusterID          int                 `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace          string              `json:"namespace" binding:"required"`  // 命名空间，必填
	ResourceQuotaNames []string            `json:"resource_quota_names"`          // ResourceQuota 名称列表，批量删除用
	ResourceQuotaYaml  *core.ResourceQuota `json:"resource_quota_yaml"`           // ResourceQuota 对象，可选
}

// K8sResourceQuotaUsage ResourceQuota 使用情况响应
type K8sResourceQuotaUsage struct {
	Name              string             `json:"name"`               // ResourceQuota 名称
	Namespace         string             `json:"namespace"`          // 命名空间
	Hard              map[string]string  `json:"hard"`               // 资源配额限制
	Used              map[string]string  `json:"used"`               // 当前使用量
	UsagePercentage   map[string]float64 `json:"usage_percentage"`   // 使用率百分比
	CreationTimestamp time.Time          `json:"creation_timestamp"` // 创建时间
}

// K8sResourceQuotaStatus ResourceQuota 状态响应
type K8sResourceQuotaStatus struct {
	Name              string            `json:"name"`               // ResourceQuota 名称
	Namespace         string            `json:"namespace"`          // 命名空间
	Hard              map[string]string `json:"hard"`               // 资源配额限制
	Used              map[string]string `json:"used"`               // 当前使用量
	Scopes            []string          `json:"scopes"`             // 资源范围
	CreationTimestamp time.Time         `json:"creation_timestamp"` // 创建时间
}

// K8sLimitRangeRequest LimitRange 相关请求结构
type K8sLimitRangeRequest struct {
	ClusterID       int              `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace       string           `json:"namespace" binding:"required"`  // 命名空间，必填
	LimitRangeNames []string         `json:"limit_range_names"`             // LimitRange 名称列表，批量删除用
	LimitRangeYaml  *core.LimitRange `json:"limit_range_yaml"`              // LimitRange 对象，可选
}

// K8sLimitRangeStatus LimitRange 状态响应
type K8sLimitRangeStatus struct {
	Name              string                `json:"name"`               // LimitRange 名称
	Namespace         string                `json:"namespace"`          // 命名空间
	Limits            []core.LimitRangeItem `json:"limits"`             // 限制项列表
	CreationTimestamp time.Time             `json:"creation_timestamp"` // 创建时间
}
