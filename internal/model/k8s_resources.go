/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package model

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sNamespace Kubernetes命名空间响应信息
type K8sNamespace struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Status            string            `json:"status"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	ResourceQuota     *ResourceQuota    `json:"resource_quota,omitempty"`
}

// ResourceQuota 资源配额信息
type ResourceQuota struct {
	CpuLimit       string `json:"cpu_limit"`
	MemoryLimit    string `json:"memory_limit"`
	PodLimit       string `json:"pod_limit"`
	CpuRequest     string `json:"cpu_request"`
	MemoryRequest  string `json:"memory_request"`
	StorageLimit   string `json:"storage_limit"`
	ServicesLimit  string `json:"services_limit"`
	SecretsLimit   string `json:"secrets_limit"`
	ConfigMapLimit string `json:"configmap_limit"`
}

// K8sDeployment Kubernetes Deployment响应信息
type K8sDeployment struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"ready_replicas"`
	AvailableReplicas int32             `json:"available_replicas"`
	UpdatedReplicas   int32             `json:"updated_replicas"`
	Strategy          string            `json:"strategy"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Images            []string          `json:"images"`
	Age               string            `json:"age"`
	Events            []K8sEvent        `json:"events,omitempty"`
}

// K8sStatefulSet Kubernetes StatefulSet响应信息
type K8sStatefulSet struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"ready_replicas"`
	CurrentReplicas   int32             `json:"current_replicas"`
	UpdatedReplicas   int32             `json:"updated_replicas"`
	ServiceName       string            `json:"service_name"`
	UpdateStrategy    string            `json:"update_strategy"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Images            []string          `json:"images"`
	Age               string            `json:"age"`
	Events            []K8sEvent        `json:"events,omitempty"`
}

// K8sDaemonSet Kubernetes DaemonSet响应信息
type K8sDaemonSet struct {
	Name               string            `json:"name"`
	UID                string            `json:"uid"`
	Namespace          string            `json:"namespace"`
	DesiredNumber      int32             `json:"desired_number"`
	CurrentNumber      int32             `json:"current_number"`
	ReadyNumber        int32             `json:"ready_number"`
	UpdatedNumber      int32             `json:"updated_number"`
	AvailableNumber    int32             `json:"available_number"`
	MisscheduledNumber int32             `json:"misscheduled_number"`
	UpdateStrategy     string            `json:"update_strategy"`
	Labels             map[string]string `json:"labels"`
	Annotations        map[string]string `json:"annotations"`
	CreationTimestamp  time.Time         `json:"creation_timestamp"`
	Images             []string          `json:"images"`
	Age                string            `json:"age"`
	Events             []K8sEvent        `json:"events,omitempty"`
}

// K8sService Kubernetes Service响应信息
type K8sService struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	Type              string            `json:"type"`
	ClusterIP         string            `json:"cluster_ip"`
	ExternalIPs       []string          `json:"external_ips,omitempty"`
	LoadBalancerIP    string            `json:"load_balancer_ip,omitempty"`
	Ports             []ServicePort     `json:"ports"`
	Selector          map[string]string `json:"selector"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Age               string            `json:"age"`
	Events            []K8sEvent        `json:"events,omitempty"`
}

// ServicePort 服务端口信息
type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
	Protocol   string `json:"protocol"`
}

// K8sIngress Kubernetes Ingress响应信息
type K8sIngress struct {
	Name              string              `json:"name"`
	UID               string              `json:"uid"`
	Namespace         string              `json:"namespace"`
	IngressClassName  string              `json:"ingress_class_name"`
	Rules             []IngressRule       `json:"rules"`
	TLS               []IngressTLS        `json:"tls,omitempty"`
	LoadBalancer      IngressLoadBalancer `json:"load_balancer"`
	Labels            map[string]string   `json:"labels"`
	Annotations       map[string]string   `json:"annotations"`
	CreationTimestamp time.Time           `json:"creation_timestamp"`
	Age               string              `json:"age"`
	Events            []K8sEvent          `json:"events,omitempty"`
}

// IngressRule Ingress规则信息
type IngressRule struct {
	Host  string        `json:"host"`
	Paths []IngressPath `json:"paths"`
}

// IngressPath Ingress路径信息
type IngressPath struct {
	Path        string             `json:"path"`
	PathType    string             `json:"path_type"`
	ServiceName string             `json:"service_name"`
	ServicePort IngressServicePort `json:"service_port"`
}

// IngressServicePort Ingress服务端口信息
type IngressServicePort struct {
	Name   string `json:"name,omitempty"`
	Number int32  `json:"number,omitempty"`
}

// IngressTLS Ingress TLS信息
type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secret_name"`
}

// IngressLoadBalancer Ingress负载均衡器信息
type IngressLoadBalancer struct {
	Ingress []IngressIngress `json:"ingress,omitempty"`
}

// IngressIngress Ingress入口信息
type IngressIngress struct {
	IP       string               `json:"ip,omitempty"`
	Hostname string               `json:"hostname,omitempty"`
	Ports    []IngressIngressPort `json:"ports,omitempty"`
}

// IngressIngressPort Ingress入口端口信息
type IngressIngressPort struct {
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

// K8sPersistentVolume Kubernetes PersistentVolume响应信息
type K8sPersistentVolume struct {
	Name              string                     `json:"name"`
	UID               string                     `json:"uid"`
	Capacity          string                     `json:"capacity"`
	AccessModes       []string                   `json:"access_modes"`
	ReclaimPolicy     string                     `json:"reclaim_policy"`
	Status            string                     `json:"status"`
	Claim             *PersistentVolumeClaimRef  `json:"claim,omitempty"`
	StorageClass      string                     `json:"storage_class"`
	VolumeSource      string                     `json:"volume_source"`
	NodeAffinity      *corev1.VolumeNodeAffinity `json:"node_affinity,omitempty"`
	MountOptions      []string                   `json:"mount_options,omitempty"`
	Labels            map[string]string          `json:"labels"`
	Annotations       map[string]string          `json:"annotations"`
	CreationTimestamp time.Time                  `json:"creation_timestamp"`
	Age               string                     `json:"age"`
	Events            []K8sEvent                 `json:"events,omitempty"`
}

// PersistentVolumeClaimRef PVC引用信息
type PersistentVolumeClaimRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// K8sPersistentVolumeClaim Kubernetes PersistentVolumeClaim响应信息
type K8sPersistentVolumeClaim struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	Status            string            `json:"status"`
	Volume            string            `json:"volume"`
	Capacity          string            `json:"capacity"`
	AccessModes       []string          `json:"access_modes"`
	StorageClass      string            `json:"storage_class"`
	VolumeMode        string            `json:"volume_mode"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Age               string            `json:"age"`
	Events            []K8sEvent        `json:"events,omitempty"`
}

// K8sConfigMap Kubernetes ConfigMap响应信息
type K8sConfigMap struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	Data              map[string]string `json:"data"`
	BinaryData        map[string][]byte `json:"binary_data,omitempty"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Age               string            `json:"age"`
	Events            []K8sEvent        `json:"events,omitempty"`
}

// K8sSecret Kubernetes Secret响应信息
type K8sSecret struct {
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Namespace         string            `json:"namespace"`
	Type              string            `json:"type"`
	Data              map[string][]byte `json:"data"`
	StringData        map[string]string `json:"string_data,omitempty"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp time.Time         `json:"creation_timestamp"`
	Age               string            `json:"age"`
	Events            []K8sEvent        `json:"events,omitempty"`
}

// K8sEvent Kubernetes事件响应信息
type K8sEvent struct {
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	Type              string                 `json:"type"`
	Reason            string                 `json:"reason"`
	Message           string                 `json:"message"`
	Source            corev1.EventSource     `json:"source"`
	InvolvedObject    corev1.ObjectReference `json:"involved_object"`
	FirstTimestamp    time.Time              `json:"first_timestamp"`
	LastTimestamp     time.Time              `json:"last_timestamp"`
	Count             int32                  `json:"count"`
	CreationTimestamp time.Time              `json:"creation_timestamp"`
}

// K8sNetworkPolicy Kubernetes NetworkPolicy响应信息
type K8sNetworkPolicy struct {
	Name              string                                  `json:"name"`
	UID               string                                  `json:"uid"`
	Namespace         string                                  `json:"namespace"`
	PodSelector       map[string]string                       `json:"pod_selector"`
	Ingress           []networkingv1.NetworkPolicyIngressRule `json:"ingress,omitempty"`
	Egress            []networkingv1.NetworkPolicyEgressRule  `json:"egress,omitempty"`
	PolicyTypes       []string                                `json:"policy_types"`
	Labels            map[string]string                       `json:"labels"`
	Annotations       map[string]string                       `json:"annotations"`
	CreationTimestamp time.Time                               `json:"creation_timestamp"`
	Age               string                                  `json:"age"`
	Events            []K8sEvent                              `json:"events,omitempty"`
}

// ==================== 通用结构体 ====================

// ContainerPort 容器端口
type ContainerPort struct {
	Name          string `json:"name" comment:"端口名称"`
	ContainerPort int32  `json:"container_port" comment:"容器端口"`
	Protocol      string `json:"protocol" comment:"端口协议"`
}

// EnvVar 环境变量
type EnvVar struct {
	Name      string        `json:"name" comment:"环境变量名"`
	Value     string        `json:"value" comment:"环境变量值"`
	ValueFrom *EnvVarSource `json:"value_from,omitempty" comment:"环境变量来源"`
}

// EnvVarSource 环境变量来源
type EnvVarSource struct {
	ConfigMapKeyRef *ConfigMapKeySelector `json:"config_map_key_ref,omitempty" comment:"ConfigMap引用"`
	SecretKeyRef    *SecretKeySelector    `json:"secret_key_ref,omitempty" comment:"Secret引用"`
}

// ConfigMapKeySelector ConfigMap键选择器
type ConfigMapKeySelector struct {
	Name string `json:"name" comment:"ConfigMap名称"`
	Key  string `json:"key" comment:"键名"`
}

// SecretKeySelector Secret键选择器
type SecretKeySelector struct {
	Name string `json:"name" comment:"Secret名称"`
	Key  string `json:"key" comment:"键名"`
}

// DeploymentStrategy 部署策略
type DeploymentStrategy struct {
	Type          string                 `json:"type" comment:"部署策略类型"`
	RollingUpdate *RollingUpdateStrategy `json:"rolling_update,omitempty" comment:"滚动更新策略"`
}

// RollingUpdateStrategy 滚动更新策略
type RollingUpdateStrategy struct {
	MaxUnavailable string `json:"max_unavailable" comment:"最大不可用数量"`
	MaxSurge       string `json:"max_surge" comment:"最大超出数量"`
}

// StatefulSetUpdateStrategy StatefulSet更新策略
type StatefulSetUpdateStrategy struct {
	Type          string                            `json:"type" comment:"更新策略类型"`
	RollingUpdate *StatefulSetRollingUpdateStrategy `json:"rolling_update,omitempty" comment:"滚动更新策略"`
}

// StatefulSetRollingUpdateStrategy StatefulSet滚动更新策略
type StatefulSetRollingUpdateStrategy struct {
	Partition      *int32 `json:"partition,omitempty" comment:"分区"`
	MaxUnavailable string `json:"max_unavailable,omitempty" comment:"最大不可用数量"`
}

// DaemonSetUpdateStrategy DaemonSet更新策略
type DaemonSetUpdateStrategy struct {
	Type          string                          `json:"type" comment:"更新策略类型"`
	RollingUpdate *DaemonSetRollingUpdateStrategy `json:"rolling_update,omitempty" comment:"滚动更新策略"`
}

// DaemonSetRollingUpdateStrategy DaemonSet滚动更新策略
type DaemonSetRollingUpdateStrategy struct {
	MaxUnavailable string `json:"max_unavailable,omitempty" comment:"最大不可用数量"`
	MaxSurge       string `json:"max_surge,omitempty" comment:"最大超出数量"`
}

// PersistentVolumeClaimTemplate 持久化存储声明模板
type PersistentVolumeClaimTemplate struct {
	Name         string               `json:"name" comment:"PVC名称"`
	AccessModes  []string             `json:"access_modes" comment:"访问模式"`
	Size         string               `json:"size" comment:"存储大小"`
	StorageClass string               `json:"storage_class" comment:"存储类"`
	Resources    ResourceRequirements `json:"resources" comment:"资源需求"`
	Selector     *LabelSelector       `json:"selector,omitempty" comment:"标签选择器"`
}

// LabelSelector 标签选择器
type LabelSelector struct {
	MatchLabels      map[string]string          `json:"match_labels,omitempty" comment:"匹配标签"`
	MatchExpressions []LabelSelectorRequirement `json:"match_expressions,omitempty" comment:"匹配表达式"`
}

// LabelSelectorRequirement 标签选择器需求
type LabelSelectorRequirement struct {
	Key      string   `json:"key" comment:"键"`
	Operator string   `json:"operator" comment:"操作符"`
	Values   []string `json:"values,omitempty" comment:"值列表"`
}

// Toleration 容忍度
type Toleration struct {
	Key               string `json:"key,omitempty" comment:"键"`
	Operator          string `json:"operator,omitempty" comment:"操作符"`
	Value             string `json:"value,omitempty" comment:"值"`
	Effect            string `json:"effect,omitempty" comment:"影响"`
	TolerationSeconds *int64 `json:"toleration_seconds,omitempty" comment:"容忍时间"`
}

// IngressRuleRequest Ingress规则请求
type IngressRuleRequest struct {
	Host  string               `json:"host" comment:"主机名"`
	Paths []IngressPathRequest `json:"paths" comment:"路径规则"`
}

// IngressPathRequest Ingress路径请求
type IngressPathRequest struct {
	Path        string                    `json:"path" comment:"路径"`
	PathType    string                    `json:"path_type" comment:"路径类型"`
	ServiceName string                    `json:"service_name" comment:"后端服务名"`
	ServicePort IngressServicePortRequest `json:"service_port" comment:"后端服务端口"`
}

// IngressServicePortRequest Ingress服务端口请求
type IngressServicePortRequest struct {
	Name   string `json:"name,omitempty" comment:"端口名"`
	Number int32  `json:"number,omitempty" comment:"端口号"`
}

// IngressTLSRequest Ingress TLS请求
type IngressTLSRequest struct {
	Hosts      []string `json:"hosts" comment:"TLS主机列表"`
	SecretName string   `json:"secret_name" comment:"TLS密钥名称"`
}

// PersistentVolumeSourceRequest PV存储源请求
type PersistentVolumeSourceRequest struct {
	HostPath *HostPathVolumeSourceRequest `json:"host_path,omitempty" comment:"主机路径"`
	NFS      *NFSVolumeSourceRequest      `json:"nfs,omitempty" comment:"NFS"`
	CSI      *CSIVolumeSourceRequest      `json:"csi,omitempty" comment:"CSI"`
}

// HostPathVolumeSourceRequest 主机路径存储源请求
type HostPathVolumeSourceRequest struct {
	Path string  `json:"path" comment:"主机路径"`
	Type *string `json:"type,omitempty" comment:"路径类型"`
}

// NFSVolumeSourceRequest NFS存储源请求
type NFSVolumeSourceRequest struct {
	Server   string `json:"server" comment:"NFS服务器"`
	Path     string `json:"path" comment:"NFS路径"`
	ReadOnly bool   `json:"read_only,omitempty" comment:"是否只读"`
}

// CSIVolumeSourceRequest CSI存储源请求
type CSIVolumeSourceRequest struct {
	Driver           string            `json:"driver" comment:"CSI驱动"`
	VolumeHandle     string            `json:"volume_handle" comment:"卷句柄"`
	ReadOnly         bool              `json:"read_only,omitempty" comment:"是否只读"`
	VolumeAttributes map[string]string `json:"volume_attributes,omitempty" comment:"卷属性"`
}

// VolumeNodeAffinityRequest 卷节点亲和性请求
type VolumeNodeAffinityRequest struct {
	Required *NodeSelectorRequest `json:"required,omitempty" comment:"必须满足的节点选择器"`
}

// NodeSelectorRequest 节点选择器请求
type NodeSelectorRequest struct {
	NodeSelectorTerms []NodeSelectorTermRequest `json:"node_selector_terms" comment:"节点选择器条件"`
}

// NodeSelectorTermRequest 节点选择器条件请求
type NodeSelectorTermRequest struct {
	MatchExpressions []NodeSelectorRequirementRequest `json:"match_expressions,omitempty" comment:"匹配表达式"`
	MatchFields      []NodeSelectorRequirementRequest `json:"match_fields,omitempty" comment:"匹配字段"`
}

// NodeSelectorRequirementRequest 节点选择器需求请求
type NodeSelectorRequirementRequest struct {
	Key      string   `json:"key" comment:"键"`
	Operator string   `json:"operator" comment:"操作符"`
	Values   []string `json:"values,omitempty" comment:"值列表"`
}

// PortForwardPort 端口转发端口配置
type PortForwardPort struct {
	LocalPort  int `json:"local_port" binding:"required" comment:"本地端口"`
	RemotePort int `json:"remote_port" binding:"required" comment:"远程端口"`
}

// ==================== 通用工具方法 ====================

// ToMetaV1ListOptions 将K8sGetResourceListRequest转换为metav1.ListOptions
func (r *K8sGetResourceListRequest) ToMetaV1ListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: r.LabelSelector,
		FieldSelector: r.FieldSelector,
		Limit:         r.Limit,
		Continue:      r.Continue,
	}
}

// ToMetaV1ListOptions 将K8sListRequest转换为metav1.ListOptions
func (r *K8sListRequest) ToMetaV1ListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: r.LabelSelector,
		FieldSelector: r.FieldSelector,
		Limit:         r.Limit,
		Continue:      r.Continue,
	}
}

// ==================== 通用资源操作请求结构体 ====================

// K8sGetResourceRequest 获取单个k8s资源请求
type K8sGetResourceRequest struct {
	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace    string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ResourceName string `json:"resource_name" form:"resource_name" uri:"resource_name" binding:"required" comment:"资源名称"`
}

// K8sGetResourceListRequest 获取k8s资源列表请求
type K8sGetResourceListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
	Limit         int64  `json:"limit" form:"limit" comment:"限制结果数量"`
	Continue      string `json:"continue" form:"continue" comment:"分页续订令牌"`
}

// K8sDeleteResourceRequest 删除k8s资源请求
type K8sDeleteResourceRequest struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace          string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ResourceName       string `json:"resource_name" form:"resource_name" binding:"required" comment:"资源名称"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds" form:"grace_period_seconds" comment:"优雅删除时间"`
	Force              bool   `json:"force" form:"force" comment:"是否强制删除"`
}

// K8sGetResourceYamlRequest 获取k8s资源YAML请求
type K8sGetResourceYamlRequest struct {
	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace    string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ResourceName string `json:"resource_name" form:"resource_name" uri:"resource_name" binding:"required" comment:"资源名称"`
}

// K8sBaseRequest K8s资源操作的基础请求结构
type K8sBaseRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
}

// K8sResourceIdentifier K8s资源标识请求结构
type K8sResourceIdentifier struct {
	K8sBaseRequest
	ResourceName string `json:"resource_name" binding:"required" comment:"资源名称"`
}

// K8sListRequest K8s资源列表查询请求结构
type K8sListRequest struct {
	ClusterID     int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace     string `json:"namespace" comment:"命名空间，为空则查询所有"`
	LabelSelector string `json:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" comment:"字段选择器"`
	Limit         int64  `json:"limit" comment:"限制结果数量"`
	Continue      string `json:"continue" comment:"分页续订令牌"`
}

// K8sBatchDeleteRequest K8s资源批量删除请求结构
type K8sBatchDeleteRequest struct {
	K8sBaseRequest
	ResourceNames []string `json:"resource_names" binding:"required" comment:"资源名称列表"`
}

// K8sBatchOperationRequest K8s资源批量操作请求结构
type K8sBatchOperationRequest struct {
	K8sBaseRequest
	ResourceNames []string `json:"resource_names" binding:"required" comment:"资源名称列表"`
	Operation     string   `json:"operation" binding:"required,oneof=restart scale delete" comment:"操作类型：restart|scale|delete"`
	Parameters    any      `json:"parameters" comment:"操作参数"`
}

// K8sYamlApplyRequest K8s YAML应用请求结构
type K8sYamlApplyRequest struct {
	K8sBaseRequest
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
	DryRun      bool   `json:"dry_run" comment:"是否为试运行"`
}

// ==================== Service层兼容结构体 ====================

// ConfigMapCreateRequest 创建ConfigMap请求（兼容）
type ConfigMapCreateRequest struct {
	K8sBaseRequest
	Name        string            `json:"name" binding:"required" comment:"ConfigMap名称"`
	Data        map[string]string `json:"data" comment:"字符串数据"`
	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// ConfigMapUpdateRequest 更新ConfigMap请求（兼容）
type ConfigMapUpdateRequest struct {
	K8sResourceIdentifier
	Data        map[string]string `json:"data" comment:"字符串数据"`
	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// SecretCreateRequest 创建Secret请求（兼容）
type SecretCreateRequest struct {
	K8sBaseRequest
	Name        string            `json:"name" binding:"required" comment:"Secret名称"`
	Type        string            `json:"type" comment:"Secret类型"`
	Data        map[string][]byte `json:"data" comment:"加密数据"`
	StringData  map[string]string `json:"string_data" comment:"明文数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// SecretUpdateRequest 更新Secret请求（兼容）
type SecretUpdateRequest struct {
	K8sResourceIdentifier
	Data        map[string][]byte `json:"data" comment:"加密数据"`
	StringData  map[string]string `json:"string_data" comment:"明文数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// DeploymentBatchDeleteRequest 批量删除Deployment请求（兼容）
type DeploymentBatchDeleteRequest struct {
	K8sBaseRequest
	DeploymentNames []string `json:"deployment_names" binding:"required" comment:"Deployment名称列表"`
}

// DeploymentBatchRestartRequest 批量重启Deployment请求（兼容）
type DeploymentBatchRestartRequest struct {
	K8sBaseRequest
	DeploymentNames []string `json:"deployment_names" binding:"required" comment:"Deployment名称列表"`
}

// StatefulSetCreateRequest 创建StatefulSet请求（兼容）
type StatefulSetCreateRequest struct {
	K8sBaseRequest
	Name                 string                          `json:"name" binding:"required" comment:"StatefulSet名称"`
	Replicas             int32                           `json:"replicas" comment:"副本数量"`
	ServiceName          string                          `json:"service_name" binding:"required" comment:"服务名称"`
	Image                string                          `json:"image" binding:"required" comment:"镜像地址"`
	Ports                []ContainerPort                 `json:"ports" comment:"容器端口"`
	Env                  []EnvVar                        `json:"env" comment:"环境变量"`
	Resources            ResourceRequirements            `json:"resources" comment:"资源限制"`
	VolumeClaimTemplates []PersistentVolumeClaimTemplate `json:"volume_claim_templates" comment:"存储卷声明模板"`
	Labels               map[string]string               `json:"labels" comment:"标签"`
	Annotations          map[string]string               `json:"annotations" comment:"注解"`
	UpdateStrategy       StatefulSetUpdateStrategy       `json:"update_strategy" comment:"更新策略"`
}

// StatefulSetUpdateRequest 更新StatefulSet请求（兼容）
type StatefulSetUpdateRequest struct {
	K8sResourceIdentifier
	Replicas             *int32                          `json:"replicas" comment:"副本数量"`
	Image                string                          `json:"image" comment:"镜像地址"`
	Ports                []ContainerPort                 `json:"ports" comment:"容器端口"`
	Env                  []EnvVar                        `json:"env" comment:"环境变量"`
	Resources            ResourceRequirements            `json:"resources" comment:"资源限制"`
	VolumeClaimTemplates []PersistentVolumeClaimTemplate `json:"volume_claim_templates" comment:"存储卷声明模板"`
	Labels               map[string]string               `json:"labels" comment:"标签"`
	Annotations          map[string]string               `json:"annotations" comment:"注解"`
	UpdateStrategy       StatefulSetUpdateStrategy       `json:"update_strategy" comment:"更新策略"`
}

// StatefulSetScaleRequest StatefulSet扩缩容请求（兼容）
type StatefulSetScaleRequest struct {
	K8sResourceIdentifier
	Replicas int32 `json:"replicas" binding:"required,min=0" comment:"副本数量"`
}

// ==================== Pod专用请求结构体 ====================

// PodLogRequest Pod日志查询请求
type PodLogRequest struct {
	K8sResourceIdentifier
	Container    string `json:"container" comment:"容器名称"`
	Follow       bool   `json:"follow" comment:"是否持续跟踪"`
	Previous     bool   `json:"previous" comment:"是否获取前一个容器的日志"`
	SinceSeconds *int64 `json:"since_seconds" comment:"获取多少秒内的日志"`
	SinceTime    string `json:"since_time" comment:"从指定时间开始获取日志"`
	Timestamps   bool   `json:"timestamps" comment:"是否显示时间戳"`
	TailLines    *int64 `json:"tail_lines" comment:"获取最后几行日志"`
	LimitBytes   *int64 `json:"limit_bytes" comment:"限制日志字节数"`
}

// PodExecRequest Pod执行命令请求
type PodExecRequest struct {
	K8sResourceIdentifier
	Container string   `json:"container" comment:"容器名称"`
	Command   []string `json:"command" binding:"required" comment:"执行的命令"`
	Stdin     bool     `json:"stdin" comment:"是否启用标准输入"`
	Stdout    bool     `json:"stdout" comment:"是否启用标准输出"`
	Stderr    bool     `json:"stderr" comment:"是否启用标准错误"`
	TTY       bool     `json:"tty" comment:"是否分配TTY"`
}

// PodPortForwardRequest Pod端口转发请求
type PodPortForwardRequest struct {
	K8sResourceIdentifier
	Ports []PortForwardPort `json:"ports" binding:"required" comment:"端口转发配置"`
}

// PodContainersRequest 获取Pod容器列表请求
type PodContainersRequest struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	PodName   string `json:"pod_name" form:"pod_name" uri:"pod_name" binding:"required" comment:"Pod名称"`
}

// PodsByNodeRequest 根据节点获取Pod列表请求
type PodsByNodeRequest struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" form:"node_name" binding:"required" comment:"节点名称"`
}

// DeploymentRestartRequest 重启Deployment请求
type DeploymentRestartRequest struct {
	ClusterID      int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace      string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	DeploymentName string `json:"deployment_name" form:"deployment_name" binding:"required" comment:"Deployment名称"`
}

// ==================== 响应结构体 ====================

// K8sPodResponse Kubernetes Pod响应信息
type K8sPodResponse struct {
	Name              string                  `json:"name"`
	UID               string                  `json:"uid"`
	Namespace         string                  `json:"namespace"`
	Status            string                  `json:"status"`
	Phase             string                  `json:"phase"`
	NodeName          string                  `json:"node_name"`
	PodIP             string                  `json:"pod_ip"`
	HostIP            string                  `json:"host_ip"`
	RestartCount      int32                   `json:"restart_count"`
	Age               string                  `json:"age"`
	Labels            map[string]string       `json:"labels"`
	Annotations       map[string]string       `json:"annotations"`
	OwnerReferences   []metav1.OwnerReference `json:"owner_references"`
	CreationTimestamp time.Time               `json:"creation_timestamp"`
	Containers        []ContainerInfo         `json:"containers"`
	Events            []K8sEvent              `json:"events,omitempty"`
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Status       string                 `json:"status"`
	Ready        bool                   `json:"ready"`
	RestartCount int32                  `json:"restart_count"`
	Resources    ContainerResources     `json:"resources"`
	Ports        []corev1.ContainerPort `json:"ports,omitempty"`
	Env          []corev1.EnvVar        `json:"env,omitempty"`
	VolumeMounts []corev1.VolumeMount   `json:"volume_mounts,omitempty"`
}

// ContainerResources 容器资源信息
type ContainerResources struct {
	CpuRequest    string `json:"cpu_request"`
	CpuLimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
	CpuUsage      string `json:"cpu_usage,omitempty"`
	MemoryUsage   string `json:"memory_usage,omitempty"`
}
