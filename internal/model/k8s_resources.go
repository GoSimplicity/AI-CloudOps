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

// ServicePort 服务端口信息 - 已在k8s_service.go中定义

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

// IngressServicePort Ingress服务端口信息
type IngressServicePort struct {
	Name   string `json:"name,omitempty"`
	Number int32  `json:"number,omitempty"`
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

// IngressServicePortRequest Ingress服务端口请求
type IngressServicePortRequest struct {
	Name   string `json:"name,omitempty" comment:"端口名"`
	Number int32  `json:"number,omitempty" comment:"端口号"`
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

// PortForwardPort 端口转发端口配置 - 已在k8s_service.go中定义

// K8sGetResourceReq 获取单个k8s资源请求
type K8sGetResourceReq struct {
	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace    string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ResourceName string `json:"resource_name" form:"resource_name" uri:"resource_name" binding:"required" comment:"资源名称"`
}

// K8sGetResourceListReq 获取k8s资源列表请求
type K8sGetResourceListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
	Limit         int64  `json:"limit" form:"limit" comment:"限制结果数量"`
	Continue      string `json:"continue" form:"continue" comment:"分页续订令牌"`
}

// K8sDeleteResourceReq 删除k8s资源请求
type K8sDeleteResourceReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace          string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ResourceName       string `json:"resource_name" form:"resource_name" binding:"required" comment:"资源名称"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds" form:"grace_period_seconds" comment:"优雅删除时间"`
	Force              bool   `json:"force" form:"force" comment:"是否强制删除"`
}

// K8sGetResourceYamlReq 获取k8s资源YAML请求
type K8sGetResourceYamlReq struct {
	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace    string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	ResourceName string `json:"resource_name" form:"resource_name" uri:"resource_name" binding:"required" comment:"资源名称"`
}

// K8sBaseReq K8s资源操作的基础请求结构
type K8sBaseReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
}

// K8sResourceIdentifierReq K8s资源标识请求结构
type K8sResourceIdentifierReq struct {
	K8sBaseReq
	ResourceName string `json:"resource_name" binding:"required" comment:"资源名称"`
}

// K8sListReq K8s资源列表查询请求结构
type K8sListReq struct {
	ClusterID     int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace     string `json:"namespace" comment:"命名空间，为空则查询所有"`
	LabelSelector string `json:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" comment:"字段选择器"`
	Limit         int64  `json:"limit" comment:"限制结果数量"`
	Continue      string `json:"continue" comment:"分页续订令牌"`
}

// K8sBatchDeleteReq K8s资源批量删除请求结构
type K8sBatchDeleteReq struct {
	K8sBaseReq
	ResourceNames []string `json:"resource_names" binding:"required" comment:"资源名称列表"`
}

// K8sBatchOperationReq K8s资源批量操作请求结构
type K8sBatchOperationReq struct {
	K8sBaseReq
	ResourceNames []string `json:"resource_names" binding:"required" comment:"资源名称列表"`
	Operation     string   `json:"operation" binding:"required,oneof=restart scale delete" comment:"操作类型：restart|scale|delete"`
	Parameters    any      `json:"parameters" comment:"操作参数"`
}

// K8sYamlApplyReq K8s YAML应用请求结构
type K8sYamlApplyReq struct {
	K8sBaseReq
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
	DryRun      bool   `json:"dry_run" comment:"是否为试运行"`
}

// ==================== Service层兼容结构体 ====================

// ConfigMapCreateReq 创建ConfigMap请求
type ConfigMapCreateReq struct {
	K8sBaseReq
	Name        string            `json:"name" binding:"required" comment:"ConfigMap名称"`
	Data        map[string]string `json:"data" comment:"字符串数据"`
	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// ConfigMapUpdateReq 更新ConfigMap请求
type ConfigMapUpdateReq struct {
	K8sResourceIdentifierReq
	Data        map[string]string `json:"data" comment:"字符串数据"`
	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// SecretCreateReq 创建Secret请求
type SecretCreateReq struct {
	K8sBaseReq
	Name        string            `json:"name" binding:"required" comment:"Secret名称"`
	Type        string            `json:"type" comment:"Secret类型"`
	Data        map[string][]byte `json:"data" comment:"加密数据"`
	StringData  map[string]string `json:"string_data" comment:"明文数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// SecretUpdateReq 更新Secret请求
type SecretUpdateReq struct {
	K8sResourceIdentifierReq
	Data        map[string][]byte `json:"data" comment:"加密数据"`
	StringData  map[string]string `json:"string_data" comment:"明文数据"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// DeploymentBatchDeleteReq 批量删除Deployment请求
type DeploymentBatchDeleteReq struct {
	K8sBaseReq
	DeploymentNames []string `json:"deployment_names" binding:"required" comment:"Deployment名称列表"`
}

// DeploymentBatchRestartReq 批量重启Deployment请求
type DeploymentBatchRestartReq struct {
	K8sBaseReq
	DeploymentNames []string `json:"deployment_names" binding:"required" comment:"Deployment名称列表"`
}

// StatefulSetCreateReq 创建StatefulSet请求
type StatefulSetCreateReq struct {
	K8sBaseReq
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

// StatefulSetUpdateReq 更新StatefulSet请求
type StatefulSetUpdateReq struct {
	K8sResourceIdentifierReq
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

// StatefulSetScaleReq StatefulSet扩缩容请求
type StatefulSetScaleReq struct {
	K8sResourceIdentifierReq
	Replicas int32 `json:"replicas" binding:"required,min=0" comment:"副本数量"`
}

// PodLogReq Pod日志查询请求
type PodLogReq struct {
	K8sResourceIdentifierReq
	Container    string `json:"container" comment:"容器名称"`
	Follow       bool   `json:"follow" comment:"是否持续跟踪"`
	Previous     bool   `json:"previous" comment:"是否获取前一个容器的日志"`
	SinceSeconds *int64 `json:"since_seconds" comment:"获取多少秒内的日志"`
	SinceTime    string `json:"since_time" comment:"从指定时间开始获取日志"`
	Timestamps   bool   `json:"timestamps" comment:"是否显示时间戳"`
	TailLines    *int64 `json:"tail_lines" comment:"获取最后几行日志"`
	LimitBytes   *int64 `json:"limit_bytes" comment:"限制日志字节数"`
}

// PodExecReq Pod执行命令请求
type PodExecReq struct {
	K8sResourceIdentifierReq
	Container string   `json:"container" comment:"容器名称"`
	Command   []string `json:"command" binding:"required" comment:"执行的命令"`
	Stdin     bool     `json:"stdin" comment:"是否启用标准输入"`
	Stdout    bool     `json:"stdout" comment:"是否启用标准输出"`
	Stderr    bool     `json:"stderr" comment:"是否启用标准错误"`
	TTY       bool     `json:"tty" comment:"是否分配TTY"`
}

// PodPortForwardReq Pod端口转发请求
type PodPortForwardReq struct {
	K8sResourceIdentifierReq
	Ports []PortForwardPort `json:"ports" binding:"required" comment:"端口转发配置"`
}

type PortForwardPort struct {
	LocalPort  int `json:"local_port" binding:"required" comment:"本地端口"`
	RemotePort int `json:"remote_port" binding:"required" comment:"远程端口"`
}

// PodContainersReq 获取Pod容器列表请求
type PodContainersReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	PodName   string `json:"pod_name" form:"pod_name" uri:"pod_name" binding:"required" comment:"Pod名称"`
}

// PodsByNodeReq 根据节点获取Pod列表请求
type PodsByNodeReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	NodeName  string `json:"node_name" form:"node_name" binding:"required" comment:"节点名称"`
}

// DeploymentRestartReq 重启Deployment请求
type DeploymentRestartReq struct {
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

// ==================== YAML任务和模板请求结构体 ====================

// YamlTaskCreateReq 创建YAML任务请求
type YamlTaskCreateReq struct {
	Name       string     `json:"name" binding:"required,min=1,max=255" comment:"YAML任务名称"`
	UserID     int        `json:"user_id" comment:"创建者用户ID"`
	TemplateID int        `json:"template_id" comment:"关联的模板ID"`
	ClusterId  int        `json:"cluster_id" comment:"集群ID"`
	Variables  StringList `json:"variables" comment:"yaml变量，格式k=v,k=v"`
}

// YamlTaskUpdateReq 更新YAML任务请求
type YamlTaskUpdateReq struct {
	ID         int        `json:"id" binding:"required" comment:"任务ID"`
	Name       string     `json:"name" binding:"required,min=1,max=255" comment:"YAML任务名称"`
	UserID     int        `json:"user_id" comment:"创建者用户ID"`
	TemplateID int        `json:"template_id" comment:"关联的模板ID"`
	ClusterId  int        `json:"cluster_id" comment:"集群ID"`
	Variables  StringList `json:"variables" comment:"yaml变量，格式k=v,k=v"`
}

// YamlTaskApplyReq 应用YAML任务请求
type YamlTaskApplyReq struct {
	ID int `json:"id" binding:"required" comment:"任务ID"`
}

// YamlTaskDeleteReq 删除YAML任务请求
type YamlTaskDeleteReq struct {
	ID int `json:"id" binding:"required" comment:"任务ID"`
}

// YamlTemplateCreateReq 创建YAML模板请求
type YamlTemplateCreateReq struct {
	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`
	UserID    int    `json:"user_id" comment:"创建者用户ID"`
	Content   string `json:"content" binding:"required" comment:"yaml模板内容"`
	ClusterId int    `json:"cluster_id" comment:"对应集群ID"`
}

// YamlTemplateUpdateReq 更新YAML模板请求
type YamlTemplateUpdateReq struct {
	ID        int    `json:"id" binding:"required" comment:"模板ID"`
	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`
	UserID    int    `json:"user_id" comment:"创建者用户ID"`
	Content   string `json:"content" binding:"required" comment:"yaml模板内容"`
	ClusterId int    `json:"cluster_id" comment:"对应集群ID"`
}

// YamlTemplateDeleteReq 删除YAML模板请求
type YamlTemplateDeleteReq struct {
	ID        int `json:"id" binding:"required" comment:"模板ID"`
	ClusterId int `json:"cluster_id" binding:"required" comment:"集群ID"`
}

// YamlTemplateCheckReq 检查YAML模板请求
type YamlTemplateCheckReq struct {
	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`
	Content   string `json:"content" binding:"required" comment:"yaml模板内容"`
	ClusterId int    `json:"cluster_id" comment:"对应集群ID"`
}

// YamlTemplateGetReq 获取YAML模板详情请求
type YamlTemplateGetReq struct {
	ID        int `json:"id" binding:"required" comment:"模板ID"`
	ClusterId int `json:"cluster_id" binding:"required" comment:"集群ID"`
}
