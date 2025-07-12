package model

import (
	networkingv1 "k8s.io/api/networking/v1"
	"time"
)

// K8sIngressRequest Ingress 相关请求结构
type K8sIngressRequest struct {
	ClusterID    int                   `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace    string                `json:"namespace" binding:"required"`  // 命名空间，必填
	IngressNames []string              `json:"ingress_names"`                 // Ingress 名称，可选
	IngressYaml  *networkingv1.Ingress `json:"ingress_yaml"`                  // Ingress 对象, 可选
}

// K8sIngressStatus Ingress 状态响应
type K8sIngressStatus struct {
	Name              string                                 `json:"name"`               // Ingress 名称
	Namespace         string                                 `json:"namespace"`          // 命名空间
	IngressClass      *string                                `json:"ingress_class"`      // Ingress 类
	Rules             []networkingv1.IngressRule             `json:"rules"`              // Ingress 规则
	TLS               []networkingv1.IngressTLS              `json:"tls"`                // TLS 配置
	Hosts             []string                               `json:"hosts"`              // 主机列表
	Paths             []string                               `json:"paths"`              // 路径列表
	LoadBalancer      networkingv1.IngressLoadBalancerStatus `json:"load_balancer"`      // 负载均衡器状态
	CreationTimestamp time.Time                              `json:"creation_timestamp"` // 创建时间
}
