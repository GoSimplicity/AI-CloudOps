package model

import "time"

// K8sNodeAffinityRequest 节点亲和性请求
type K8sNodeAffinityRequest struct {
	ClusterID         int                          `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace         string                       `json:"namespace" binding:"required"`     // 命名空间，必填
	ResourceType      string                       `json:"resource_type" binding:"required"` // 资源类型，必填
	ResourceName      string                       `json:"resource_name" binding:"required"` // 资源名称，必填
	RequiredAffinity  []K8sNodeSelectorTerm        `json:"required_affinity"`                // 硬亲和性规则
	PreferredAffinity []K8sPreferredSchedulingTerm `json:"preferred_affinity"`               // 软亲和性规则
	NodeSelector      map[string]string            `json:"node_selector"`                    // 节点选择器
	Operation         string                       `json:"operation"`                        // 操作类型 (add, update, delete)
}

// K8sNodeSelectorTerm 节点选择器条件
type K8sNodeSelectorTerm struct {
	MatchExpressions []K8sNodeSelectorRequirement `json:"match_expressions"` // 匹配表达式
	MatchFields      []K8sNodeSelectorRequirement `json:"match_fields"`      // 匹配字段
}

// K8sNodeSelectorRequirement 节点选择器要求
type K8sNodeSelectorRequirement struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符 (In, NotIn, Exists, DoesNotExist, Gt, Lt)
	Values   []string `json:"values"`   // 值列表
}

// K8sPreferredSchedulingTerm 优先调度条件
type K8sPreferredSchedulingTerm struct {
	Weight     int32               `json:"weight"`     // 权重 (1-100)
	Preference K8sNodeSelectorTerm `json:"preference"` // 偏好条件
}

// K8sNodeAffinityResponse 节点亲和性响应
type K8sNodeAffinityResponse struct {
	ResourceType      string                       `json:"resource_type"`      // 资源类型
	ResourceName      string                       `json:"resource_name"`      // 资源名称
	Namespace         string                       `json:"namespace"`          // 命名空间
	RequiredAffinity  []K8sNodeSelectorTerm        `json:"required_affinity"`  // 硬亲和性规则
	PreferredAffinity []K8sPreferredSchedulingTerm `json:"preferred_affinity"` // 软亲和性规则
	NodeSelector      map[string]string            `json:"node_selector"`      // 节点选择器
	CreationTimestamp time.Time                    `json:"creation_timestamp"` // 创建时间
}

// K8sNodeAffinityValidationRequest 节点亲和性验证请求
type K8sNodeAffinityValidationRequest struct {
	ClusterID          int                          `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace          string                       `json:"namespace"`                     // 命名空间，可选
	RequiredAffinity   []K8sNodeSelectorTerm        `json:"required_affinity"`             // 硬亲和性规则
	PreferredAffinity  []K8sPreferredSchedulingTerm `json:"preferred_affinity"`            // 软亲和性规则
	NodeSelector       map[string]string            `json:"node_selector"`                 // 节点选择器
	SimulateScheduling bool                         `json:"simulate_scheduling"`           // 是否模拟调度
}

// K8sNodeAffinityValidationResponse 节点亲和性验证响应
type K8sNodeAffinityValidationResponse struct {
	Valid            bool      `json:"valid"`             // 是否有效
	MatchingNodes    []string  `json:"matching_nodes"`    // 匹配的节点列表
	ValidationErrors []string  `json:"validation_errors"` // 验证错误
	Suggestions      []string  `json:"suggestions"`       // 建议
	SchedulingResult string    `json:"scheduling_result"` // 调度结果
	ValidationTime   time.Time `json:"validation_time"`   // 验证时间
}

// K8sPodAffinityRequest Pod 亲和性请求
type K8sPodAffinityRequest struct {
	ClusterID       int                  `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace       string               `json:"namespace" binding:"required"`     // 命名空间，必填
	ResourceType    string               `json:"resource_type" binding:"required"` // 资源类型，必填
	ResourceName    string               `json:"resource_name" binding:"required"` // 资源名称，必填
	PodAffinity     []K8sPodAffinityTerm `json:"pod_affinity"`                     // Pod 亲和性
	PodAntiAffinity []K8sPodAffinityTerm `json:"pod_anti_affinity"`                // Pod 反亲和性
	TopologyKey     string               `json:"topology_key"`                     // 拓扑键
	Operation       string               `json:"operation"`                        // 操作类型 (add, update, delete)
}

// K8sPodAffinityTerm Pod 亲和性条件
type K8sPodAffinityTerm struct {
	LabelSelector     K8sLabelSelector  `json:"label_selector"`     // 标签选择器
	Namespaces        []string          `json:"namespaces"`         // 命名空间列表
	TopologyKey       string            `json:"topology_key"`       // 拓扑键
	NamespaceSelector *K8sLabelSelector `json:"namespace_selector"` // 命名空间选择器
	Weight            int32             `json:"weight,omitempty"`   // 权重（仅用于软亲和性）
}

// K8sPodAntiAffinityTerm Pod 反亲和性条件
type K8sPodAntiAffinityTerm struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []K8sPodAffinityTermSpec     `json:"required_during_scheduling_ignored_during_execution"`  // 硬反亲和性
	PreferredDuringSchedulingIgnoredDuringExecution []K8sWeightedPodAffinityTerm `json:"preferred_during_scheduling_ignored_during_execution"` // 软反亲和性
}

// K8sPodAffinityTermSpec Pod 亲和性条件规格
type K8sPodAffinityTermSpec struct {
	LabelSelector     *K8sLabelSelector `json:"label_selector"`     // 标签选择器
	Namespaces        []string          `json:"namespaces"`         // 命名空间列表
	TopologyKey       string            `json:"topology_key"`       // 拓扑键
	NamespaceSelector *K8sLabelSelector `json:"namespace_selector"` // 命名空间选择器
}

// K8sLabelSelector 标签选择器
type K8sLabelSelector struct {
	MatchLabels      map[string]string             `json:"match_labels"`      // 匹配标签
	MatchExpressions []K8sLabelSelectorRequirement `json:"match_expressions"` // 匹配表达式
}

// K8sLabelSelectorRequirement 标签选择器要求
type K8sLabelSelectorRequirement struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符 (In, NotIn, Exists, DoesNotExist)
	Values   []string `json:"values"`   // 值列表
}

// K8sWeightedPodAffinityTerm 带权重的 Pod 亲和性条件
type K8sWeightedPodAffinityTerm struct {
	Weight          int32                  `json:"weight"`            // 权重 (1-100)
	PodAffinityTerm K8sPodAffinityTermSpec `json:"pod_affinity_term"` // Pod 亲和性条件
}

// K8sPodAffinityResponse Pod 亲和性响应
type K8sPodAffinityResponse struct {
	ResourceType      string               `json:"resource_type"`      // 资源类型
	ResourceName      string               `json:"resource_name"`      // 资源名称
	Namespace         string               `json:"namespace"`          // 命名空间
	PodAffinity       []K8sPodAffinityTerm `json:"pod_affinity"`       // Pod 亲和性
	PodAntiAffinity   []K8sPodAffinityTerm `json:"pod_anti_affinity"`  // Pod 反亲和性
	TopologyKey       string               `json:"topology_key"`       // 拓扑键
	TopologyDomains   []string             `json:"topology_domains"`   // 拓扑域列表
	CreationTimestamp time.Time            `json:"creation_timestamp"` // 创建时间
}

// K8sPodAffinityValidationRequest Pod 亲和性验证请求
type K8sPodAffinityValidationRequest struct {
	ClusterID          int                     `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace          string                  `json:"namespace"`                     // 命名空间，可选
	PodAffinity        *K8sPodAffinityTerm     `json:"pod_affinity"`                  // Pod 亲和性
	PodAntiAffinity    *K8sPodAntiAffinityTerm `json:"pod_anti_affinity"`             // Pod 反亲和性
	SimulateScheduling bool                    `json:"simulate_scheduling"`           // 是否模拟调度
}

// K8sPodAffinityValidationResponse Pod 亲和性验证响应
type K8sPodAffinityValidationResponse struct {
	Valid            bool      `json:"valid"`             // 是否有效
	MatchingPods     []string  `json:"matching_pods"`     // 匹配的 Pod 列表
	ValidationErrors []string  `json:"validation_errors"` // 验证错误
	Suggestions      []string  `json:"suggestions"`       // 建议
	SchedulingResult string    `json:"scheduling_result"` // 调度结果
	ValidationTime   time.Time `json:"validation_time"`   // 验证时间
}
