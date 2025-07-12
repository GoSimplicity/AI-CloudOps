package model

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// K8sNetworkPolicyRequest NetworkPolicy 相关请求结构
type K8sNetworkPolicyRequest struct {
	ClusterID          int                         `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace          string                      `json:"namespace" binding:"required"`  // 命名空间，必填
	NetworkPolicyNames []string                    `json:"network_policy_names"`          // NetworkPolicy 名称，可选
	NetworkPolicyYaml  *networkingv1.NetworkPolicy `json:"network_policy_yaml"`           // NetworkPolicy 对象, 可选
}

// K8sNetworkPolicyStatus NetworkPolicy 状态响应
type K8sNetworkPolicyStatus struct {
	Name              string                                  `json:"name"`               // NetworkPolicy 名称
	Namespace         string                                  `json:"namespace"`          // 命名空间
	PodSelector       *metav1.LabelSelector                   `json:"pod_selector"`       // Pod 选择器
	PolicyTypes       []networkingv1.PolicyType               `json:"policy_types"`       // 策略类型 (Ingress/Egress)
	Ingress           []networkingv1.NetworkPolicyIngressRule `json:"ingress"`            // 入站规则
	Egress            []networkingv1.NetworkPolicyEgressRule  `json:"egress"`             // 出站规则
	CreationTimestamp time.Time                               `json:"creation_timestamp"` // 创建时间
}
