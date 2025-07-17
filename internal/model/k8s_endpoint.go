package model

import (
	core "k8s.io/api/core/v1"
	"time"
)

// K8sEndpointRequest Endpoint 相关请求结构
type K8sEndpointRequest struct {
	ClusterID     int             `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace     string          `json:"namespace" binding:"required"`  // 命名空间，必填
	EndpointNames []string        `json:"endpoint_names"`                // Endpoint 名称，可选
	EndpointYaml  *core.Endpoints `json:"endpoint_yaml"`                 // Endpoint 对象, 可选
}

// K8sEndpointStatus Endpoint 状态响应
type K8sEndpointStatus struct {
	Name               string                `json:"name"`                // Endpoint 名称
	Namespace          string                `json:"namespace"`           // 命名空间
	Subsets            []core.EndpointSubset `json:"subsets"`             // 端点子集
	Addresses          []string              `json:"addresses"`           // 端点地址列表
	Ports              []core.EndpointPort   `json:"ports"`               // 端点端口列表
	ServiceName        string                `json:"service_name"`        // 关联的服务名称
	HealthyEndpoints   int                   `json:"healthy_endpoints"`   // 健康端点数量
	UnhealthyEndpoints int                   `json:"unhealthy_endpoints"` // 不健康端点数量
	CreationTimestamp  time.Time             `json:"creation_timestamp"`  // 创建时间
}
