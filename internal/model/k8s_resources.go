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

// import (
// 	"time"

// 	corev1 "k8s.io/api/core/v1"
// 	networkingv1 "k8s.io/api/networking/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// // ResourceQuota 资源配额信息
// type ResourceQuota struct {
// 	CpuLimit       string `json:"cpu_limit"`
// 	MemoryLimit    string `json:"memory_limit"`
// 	PodLimit       string `json:"pod_limit"`
// 	CpuRequest     string `json:"cpu_request"`
// 	MemoryRequest  string `json:"memory_request"`
// 	StorageLimit   string `json:"storage_limit"`
// 	ServicesLimit  string `json:"services_limit"`
// 	SecretsLimit   string `json:"secrets_limit"`
// 	ConfigMapLimit string `json:"configmap_limit"`
// }

// // ServicePort 服务端口信息 - 已在k8s_service.go中定义

// // K8sIngress Kubernetes Ingress响应信息
// type K8sIngress struct {
// 	Name              string              `json:"name"`
// 	UID               string              `json:"uid"`
// 	Namespace         string              `json:"namespace"`
// 	IngressClassName  string              `json:"ingress_class_name"`
// 	Rules             []IngressRule       `json:"rules"`
// 	TLS               []IngressTLS        `json:"tls,omitempty"`
// 	LoadBalancer      IngressLoadBalancer `json:"load_balancer"`
// 	Labels            map[string]string   `json:"labels"`
// 	Annotations       map[string]string   `json:"annotations"`
// 	CreationTimestamp time.Time           `json:"creation_timestamp"`
// 	Age               string              `json:"age"`
// 	Events            []K8sEvent          `json:"events,omitempty"`
// }

// // IngressServicePort Ingress服务端口信息
// type IngressServicePort struct {
// 	Name   string `json:"name,omitempty"`
// 	Number int32  `json:"number,omitempty"`
// }

// // IngressIngress Ingress入口信息
// type IngressIngress struct {
// 	IP       string               `json:"ip,omitempty"`
// 	Hostname string               `json:"hostname,omitempty"`
// 	Ports    []IngressIngressPort `json:"ports,omitempty"`
// }

// // IngressIngressPort Ingress入口端口信息
// type IngressIngressPort struct {
// 	Port     int32  `json:"port"`
// 	Protocol string `json:"protocol"`
// }

// // K8sPersistentVolume Kubernetes PersistentVolume响应信息
// type K8sPersistentVolume struct {
// 	Name              string                     `json:"name"`
// 	UID               string                     `json:"uid"`
// 	Capacity          string                     `json:"capacity"`
// 	AccessModes       []string                   `json:"access_modes"`
// 	ReclaimPolicy     string                     `json:"reclaim_policy"`
// 	Status            string                     `json:"status"`
// 	Claim             *PersistentVolumeClaimRef  `json:"claim,omitempty"`
// 	StorageClass      string                     `json:"storage_class"`
// 	VolumeSource      string                     `json:"volume_source"`
// 	NodeAffinity      *corev1.VolumeNodeAffinity `json:"node_affinity,omitempty"`
// 	MountOptions      []string                   `json:"mount_options,omitempty"`
// 	Labels            map[string]string          `json:"labels"`
// 	Annotations       map[string]string          `json:"annotations"`
// 	CreationTimestamp time.Time                  `json:"creation_timestamp"`
// 	Age               string                     `json:"age"`
// 	Events            []K8sEvent                 `json:"events,omitempty"`
// }

// // PersistentVolumeClaimRef PVC引用信息
// type PersistentVolumeClaimRef struct {
// 	Name      string `json:"name"`
// 	Namespace string `json:"namespace"`
// }

// // K8sPersistentVolumeClaim Kubernetes PersistentVolumeClaim响应信息
// type K8sPersistentVolumeClaim struct {
// 	Name              string            `json:"name"`
// 	UID               string            `json:"uid"`
// 	Namespace         string            `json:"namespace"`
// 	Status            string            `json:"status"`
// 	Volume            string            `json:"volume"`
// 	Capacity          string            `json:"capacity"`
// 	AccessModes       []string          `json:"access_modes"`
// 	StorageClass      string            `json:"storage_class"`
// 	VolumeMode        string            `json:"volume_mode"`
// 	Labels            map[string]string `json:"labels"`
// 	Annotations       map[string]string `json:"annotations"`
// 	CreationTimestamp time.Time         `json:"creation_timestamp"`
// 	Age               string            `json:"age"`
// 	Events            []K8sEvent        `json:"events,omitempty"`
// }

// // K8sConfigMap Kubernetes ConfigMap响应信息
// type K8sConfigMap struct {
// 	Name              string            `json:"name"`
// 	UID               string            `json:"uid"`
// 	Namespace         string            `json:"namespace"`
// 	Data              map[string]string `json:"data"`
// 	BinaryData        map[string][]byte `json:"binary_data,omitempty"`
// 	Labels            map[string]string `json:"labels"`
// 	Annotations       map[string]string `json:"annotations"`
// 	CreationTimestamp time.Time         `json:"creation_timestamp"`
// 	Age               string            `json:"age"`
// 	Events            []K8sEvent        `json:"events,omitempty"`
// }

// // K8sSecret Kubernetes Secret响应信息
// type K8sSecret struct {
// 	Name              string            `json:"name"`
// 	UID               string            `json:"uid"`
// 	Namespace         string            `json:"namespace"`
// 	Type              string            `json:"type"`
// 	Data              map[string][]byte `json:"data"`
// 	StringData        map[string]string `json:"string_data,omitempty"`
// 	Labels            map[string]string `json:"labels"`
// 	Annotations       map[string]string `json:"annotations"`
// 	CreationTimestamp time.Time         `json:"creation_timestamp"`
// 	Age               string            `json:"age"`
// 	Events            []K8sEvent        `json:"events,omitempty"`
// }

// // K8sNetworkPolicy Kubernetes NetworkPolicy响应信息
// type K8sNetworkPolicy struct {
// 	Name              string                                  `json:"name"`
// 	UID               string                                  `json:"uid"`
// 	Namespace         string                                  `json:"namespace"`
// 	PodSelector       map[string]string                       `json:"pod_selector"`
// 	Ingress           []networkingv1.NetworkPolicyIngressRule `json:"ingress,omitempty"`
// 	Egress            []networkingv1.NetworkPolicyEgressRule  `json:"egress,omitempty"`
// 	PolicyTypes       []string                                `json:"policy_types"`
// 	Labels            map[string]string                       `json:"labels"`
// 	Annotations       map[string]string                       `json:"annotations"`
// 	CreationTimestamp time.Time                               `json:"creation_timestamp"`
// 	Age               string                                  `json:"age"`
// 	Events            []K8sEvent                              `json:"events,omitempty"`
// }

// // ==================== 通用结构体 ====================

// // ContainerPort 容器端口
// type ContainerPort struct {
// 	Name          string `json:"name" comment:"端口名称"`
// 	ContainerPort int32  `json:"container_port" comment:"容器端口"`
// 	Protocol      string `json:"protocol" comment:"端口协议"`
// }

// // EnvVar 环境变量
// type EnvVar struct {
// 	Name      string        `json:"name" comment:"环境变量名"`
// 	Value     string        `json:"value" comment:"环境变量值"`
// 	ValueFrom *EnvVarSource `json:"value_from,omitempty" comment:"环境变量来源"`
// }

// // EnvVarSource 环境变量来源
// type EnvVarSource struct {
// 	ConfigMapKeyRef *ConfigMapKeySelector `json:"config_map_key_ref,omitempty" comment:"ConfigMap引用"`
// 	SecretKeyRef    *SecretKeySelector    `json:"secret_key_ref,omitempty" comment:"Secret引用"`
// }

// // ConfigMapKeySelector ConfigMap键选择器
// type ConfigMapKeySelector struct {
// 	Name string `json:"name" comment:"ConfigMap名称"`
// 	Key  string `json:"key" comment:"键名"`
// }

// // SecretKeySelector Secret键选择器
// type SecretKeySelector struct {
// 	Name string `json:"name" comment:"Secret名称"`
// 	Key  string `json:"key" comment:"键名"`
// }

// // DeploymentStrategy 部署策略
// type DeploymentStrategy struct {
// 	Type          string                 `json:"type" comment:"部署策略类型"`
// 	RollingUpdate *RollingUpdateStrategy `json:"rolling_update,omitempty" comment:"滚动更新策略"`
// }

// // RollingUpdateStrategy 滚动更新策略
// type RollingUpdateStrategy struct {
// 	MaxUnavailable string `json:"max_unavailable" comment:"最大不可用数量"`
// 	MaxSurge       string `json:"max_surge" comment:"最大超出数量"`
// }

// // StatefulSetUpdateStrategy StatefulSet更新策略
// type StatefulSetUpdateStrategy struct {
// 	Type          string                            `json:"type" comment:"更新策略类型"`
// 	RollingUpdate *StatefulSetRollingUpdateStrategy `json:"rolling_update,omitempty" comment:"滚动更新策略"`
// }

// // StatefulSetRollingUpdateStrategy StatefulSet滚动更新策略
// type StatefulSetRollingUpdateStrategy struct {
// 	Partition      *int32 `json:"partition,omitempty" comment:"分区"`
// 	MaxUnavailable string `json:"max_unavailable,omitempty" comment:"最大不可用数量"`
// }

// // PersistentVolumeClaimTemplate 持久化存储声明模板
// type PersistentVolumeClaimTemplate struct {
// 	Name         string               `json:"name" comment:"PVC名称"`
// 	AccessModes  []string             `json:"access_modes" comment:"访问模式"`
// 	Size         string               `json:"size" comment:"存储大小"`
// 	StorageClass string               `json:"storage_class" comment:"存储类"`
// 	Resources    ResourceRequirements `json:"resources" comment:"资源需求"`
// 	Selector     *LabelSelector       `json:"selector,omitempty" comment:"标签选择器"`
// }

// // LabelSelector 标签选择器
// type LabelSelector struct {
// 	MatchLabels      map[string]string          `json:"match_labels,omitempty" comment:"匹配标签"`
// 	MatchExpressions []LabelSelectorRequirement `json:"match_expressions,omitempty" comment:"匹配表达式"`
// }

// // LabelSelectorRequirement 标签选择器需求
// type LabelSelectorRequirement struct {
// 	Key      string   `json:"key" comment:"键"`
// 	Operator string   `json:"operator" comment:"操作符"`
// 	Values   []string `json:"values,omitempty" comment:"值列表"`
// }

// // Toleration 容忍度
// type Toleration struct {
// 	Key               string `json:"key,omitempty" comment:"键"`
// 	Operator          string `json:"operator,omitempty" comment:"操作符"`
// 	Value             string `json:"value,omitempty" comment:"值"`
// 	Effect            string `json:"effect,omitempty" comment:"影响"`
// 	TolerationSeconds *int64 `json:"toleration_seconds,omitempty" comment:"容忍时间"`
// }

// // IngressServicePortRequest Ingress服务端口请求
// type IngressServicePortRequest struct {
// 	Name   string `json:"name,omitempty" comment:"端口名"`
// 	Number int32  `json:"number,omitempty" comment:"端口号"`
// }

// // PersistentVolumeSourceRequest PV存储源请求
// type PersistentVolumeSourceRequest struct {
// 	HostPath *HostPathVolumeSourceRequest `json:"host_path,omitempty" comment:"主机路径"`
// 	NFS      *NFSVolumeSourceRequest      `json:"nfs,omitempty" comment:"NFS"`
// 	CSI      *CSIVolumeSourceRequest      `json:"csi,omitempty" comment:"CSI"`
// }

// // HostPathVolumeSourceRequest 主机路径存储源请求
// type HostPathVolumeSourceRequest struct {
// 	Path string  `json:"path" comment:"主机路径"`
// 	Type *string `json:"type,omitempty" comment:"路径类型"`
// }

// // NFSVolumeSourceRequest NFS存储源请求
// type NFSVolumeSourceRequest struct {
// 	Server   string `json:"server" comment:"NFS服务器"`
// 	Path     string `json:"path" comment:"NFS路径"`
// 	ReadOnly bool   `json:"read_only,omitempty" comment:"是否只读"`
// }

// // CSIVolumeSourceRequest CSI存储源请求
// type CSIVolumeSourceRequest struct {
// 	Driver           string            `json:"driver" comment:"CSI驱动"`
// 	VolumeHandle     string            `json:"volume_handle" comment:"卷句柄"`
// 	ReadOnly         bool              `json:"read_only,omitempty" comment:"是否只读"`
// 	VolumeAttributes map[string]string `json:"volume_attributes,omitempty" comment:"卷属性"`
// }

// // VolumeNodeAffinityRequest 卷节点亲和性请求
// type VolumeNodeAffinityRequest struct {
// 	Required *NodeSelectorRequest `json:"required,omitempty" comment:"必须满足的节点选择器"`
// }

// // NodeSelectorRequest 节点选择器请求
// type NodeSelectorRequest struct {
// 	NodeSelectorTerms []NodeSelectorTermRequest `json:"node_selector_terms" comment:"节点选择器条件"`
// }

// // NodeSelectorTermRequest 节点选择器条件请求
// type NodeSelectorTermRequest struct {
// 	MatchExpressions []NodeSelectorRequirementRequest `json:"match_expressions,omitempty" comment:"匹配表达式"`
// 	MatchFields      []NodeSelectorRequirementRequest `json:"match_fields,omitempty" comment:"匹配字段"`
// }

// // NodeSelectorRequirementRequest 节点选择器需求请求
// type NodeSelectorRequirementRequest struct {
// 	Key      string   `json:"key" comment:"键"`
// 	Operator string   `json:"operator" comment:"操作符"`
// 	Values   []string `json:"values,omitempty" comment:"值列表"`
// }

// // PortForwardPort 端口转发端口配置 - 已在k8s_service.go中定义

// // K8sGetResourceReq 获取单个k8s资源请求
// type K8sGetResourceReq struct {
// 	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace    string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
// 	ResourceName string `json:"resource_name" form:"resource_name" uri:"resource_name" binding:"required" comment:"资源名称"`
// }

// // K8sGetResourceListReq 获取k8s资源列表请求
// type K8sGetResourceListReq struct {
// 	ClusterID     int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`
// 	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
// 	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
// 	Limit         int64  `json:"limit" form:"limit" comment:"限制结果数量"`
// 	Continue      string `json:"continue" form:"continue" comment:"分页续订令牌"`
// }

// // K8sDeleteResourceReq 删除k8s资源请求
// type K8sDeleteResourceReq struct {
// 	ClusterID          int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace          string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
// 	ResourceName       string `json:"resource_name" form:"resource_name" binding:"required" comment:"资源名称"`
// 	GracePeriodSeconds *int64 `json:"grace_period_seconds" form:"grace_period_seconds" comment:"优雅删除时间"`
// 	Force              bool   `json:"force" form:"force" comment:"是否强制删除"`
// }

// // K8sGetResourceYamlReq 获取k8s资源YAML请求
// type K8sGetResourceYamlReq struct {
// 	ClusterID    int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace    string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
// 	ResourceName string `json:"resource_name" form:"resource_name" uri:"resource_name" binding:"required" comment:"资源名称"`
// }

// // K8sBaseReq K8s资源操作的基础请求结构
// type K8sBaseReq struct {
// 	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
// }

// // K8sResourceIdentifierReq K8s资源标识请求结构
// type K8sResourceIdentifierReq struct {
// 	K8sBaseReq
// 	ResourceName string `json:"resource_name" binding:"required" comment:"资源名称"`
// }

// // K8sListReq K8s资源列表查询请求结构
// type K8sListReq struct {
// 	ClusterID     int    `json:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace     string `json:"namespace" comment:"命名空间，为空则查询所有"`
// 	LabelSelector string `json:"label_selector" comment:"标签选择器"`
// 	FieldSelector string `json:"field_selector" comment:"字段选择器"`
// 	Limit         int64  `json:"limit" comment:"限制结果数量"`
// 	Continue      string `json:"continue" comment:"分页续订令牌"`
// }

// // K8sYamlApplyReq K8s YAML应用请求结构
// type K8sYamlApplyReq struct {
// 	K8sBaseReq
// 	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
// 	DryRun      bool   `json:"dry_run" comment:"是否为试运行"`
// }

// // ==================== Service层兼容结构体 ====================

// // ConfigMapCreateReq 创建ConfigMap请求
// type ConfigMapCreateReq struct {
// 	K8sBaseReq
// 	Name        string            `json:"name" binding:"required" comment:"ConfigMap名称"`
// 	Data        map[string]string `json:"data" comment:"字符串数据"`
// 	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`
// 	Labels      map[string]string `json:"labels" comment:"标签"`
// 	Annotations map[string]string `json:"annotations" comment:"注解"`
// }

// // ConfigMapUpdateReq 更新ConfigMap请求
// type ConfigMapUpdateReq struct {
// 	K8sResourceIdentifierReq
// 	Data        map[string]string `json:"data" comment:"字符串数据"`
// 	BinaryData  map[string][]byte `json:"binary_data" comment:"二进制数据"`
// 	Labels      map[string]string `json:"labels" comment:"标签"`
// 	Annotations map[string]string `json:"annotations" comment:"注解"`
// }

// // SecretCreateReq 创建Secret请求
// type SecretCreateReq struct {
// 	K8sBaseReq
// 	Name        string            `json:"name" binding:"required" comment:"Secret名称"`
// 	Type        string            `json:"type" comment:"Secret类型"`
// 	Data        map[string][]byte `json:"data" comment:"加密数据"`
// 	StringData  map[string]string `json:"string_data" comment:"明文数据"`
// 	Labels      map[string]string `json:"labels" comment:"标签"`
// 	Annotations map[string]string `json:"annotations" comment:"注解"`
// }

// // SecretUpdateReq 更新Secret请求
// type SecretUpdateReq struct {
// 	K8sResourceIdentifierReq
// 	Data        map[string][]byte `json:"data" comment:"加密数据"`
// 	StringData  map[string]string `json:"string_data" comment:"明文数据"`
// 	Labels      map[string]string `json:"labels" comment:"标签"`
// 	Annotations map[string]string `json:"annotations" comment:"注解"`
// }

// // StatefulSetCreateReq 创建StatefulSet请求
// type StatefulSetCreateReq struct {
// 	K8sBaseReq
// 	Name                 string                          `json:"name" binding:"required" comment:"StatefulSet名称"`
// 	Replicas             int32                           `json:"replicas" comment:"副本数量"`
// 	ServiceName          string                          `json:"service_name" binding:"required" comment:"服务名称"`
// 	Image                string                          `json:"image" binding:"required" comment:"镜像地址"`
// 	Ports                []ContainerPort                 `json:"ports" comment:"容器端口"`
// 	Env                  []EnvVar                        `json:"env" comment:"环境变量"`
// 	Resources            ResourceRequirements            `json:"resources" comment:"资源限制"`
// 	VolumeClaimTemplates []PersistentVolumeClaimTemplate `json:"volume_claim_templates" comment:"存储卷声明模板"`
// 	Labels               map[string]string               `json:"labels" comment:"标签"`
// 	Annotations          map[string]string               `json:"annotations" comment:"注解"`
// 	UpdateStrategy       StatefulSetUpdateStrategy       `json:"update_strategy" comment:"更新策略"`
// }

// // StatefulSetUpdateReq 更新StatefulSet请求
// type StatefulSetUpdateReq struct {
// 	K8sResourceIdentifierReq
// 	Replicas             *int32                          `json:"replicas" comment:"副本数量"`
// 	Image                string                          `json:"image" comment:"镜像地址"`
// 	Ports                []ContainerPort                 `json:"ports" comment:"容器端口"`
// 	Env                  []EnvVar                        `json:"env" comment:"环境变量"`
// 	Resources            ResourceRequirements            `json:"resources" comment:"资源限制"`
// 	VolumeClaimTemplates []PersistentVolumeClaimTemplate `json:"volume_claim_templates" comment:"存储卷声明模板"`
// 	Labels               map[string]string               `json:"labels" comment:"标签"`
// 	Annotations          map[string]string               `json:"annotations" comment:"注解"`
// 	UpdateStrategy       StatefulSetUpdateStrategy       `json:"update_strategy" comment:"更新策略"`
// }

// // StatefulSetScaleReq StatefulSet扩缩容请求
// type StatefulSetScaleReq struct {
// 	K8sResourceIdentifierReq
// 	Replicas int32 `json:"replicas" binding:"required,min=0" comment:"副本数量"`
// }

// // PodLogReq Pod日志查询请求
// type PodLogReq struct {
// 	K8sResourceIdentifierReq
// 	Container    string `json:"container" comment:"容器名称"`
// 	Follow       bool   `json:"follow" comment:"是否持续跟踪"`
// 	Previous     bool   `json:"previous" comment:"是否获取前一个容器的日志"`
// 	SinceSeconds *int64 `json:"since_seconds" comment:"获取多少秒内的日志"`
// 	SinceTime    string `json:"since_time" comment:"从指定时间开始获取日志"`
// 	Timestamps   bool   `json:"timestamps" comment:"是否显示时间戳"`
// 	TailLines    *int64 `json:"tail_lines" comment:"获取最后几行日志"`
// 	LimitBytes   *int64 `json:"limit_bytes" comment:"限制日志字节数"`
// }

// // PodExecReq Pod执行命令请求
// type PodExecReq struct {
// 	K8sResourceIdentifierReq
// 	Container string   `json:"container" comment:"容器名称"`
// 	Command   []string `json:"command" binding:"required" comment:"执行的命令"`
// 	Stdin     bool     `json:"stdin" comment:"是否启用标准输入"`
// 	Stdout    bool     `json:"stdout" comment:"是否启用标准输出"`
// 	Stderr    bool     `json:"stderr" comment:"是否启用标准错误"`
// 	TTY       bool     `json:"tty" comment:"是否分配TTY"`
// }

// // PodPortForwardReq Pod端口转发请求
// type PodPortForwardReq struct {
// 	K8sResourceIdentifierReq
// 	Ports []PortForwardPort `json:"ports" binding:"required" comment:"端口转发配置"`
// }

// type PortForwardPort struct {
// 	LocalPort  int `json:"local_port" binding:"required" comment:"本地端口"`
// 	RemotePort int `json:"remote_port" binding:"required" comment:"远程端口"`
// }

// // PodContainersReq 获取Pod容器列表请求
// type PodContainersReq struct {
// 	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
// 	PodName   string `json:"pod_name" form:"pod_name" uri:"pod_name" binding:"required" comment:"Pod名称"`
// }

// // PodsByNodeReq 根据节点获取Pod列表请求
// type PodsByNodeReq struct {
// 	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	NodeName  string `json:"node_name" form:"node_name" binding:"required" comment:"节点名称"`
// }

// // ==================== 响应结构体 ====================

// // K8sPodResponse Kubernetes Pod响应信息
// type K8sPodResponse struct {
// 	Name              string                  `json:"name"`
// 	UID               string                  `json:"uid"`
// 	Namespace         string                  `json:"namespace"`
// 	Status            string                  `json:"status"`
// 	Phase             string                  `json:"phase"`
// 	NodeName          string                  `json:"node_name"`
// 	PodIP             string                  `json:"pod_ip"`
// 	HostIP            string                  `json:"host_ip"`
// 	RestartCount      int32                   `json:"restart_count"`
// 	Age               string                  `json:"age"`
// 	Labels            map[string]string       `json:"labels"`
// 	Annotations       map[string]string       `json:"annotations"`
// 	OwnerReferences   []metav1.OwnerReference `json:"owner_references"`
// 	CreationTimestamp time.Time               `json:"creation_timestamp"`
// 	Containers        []ContainerInfo         `json:"containers"`
// 	Events            []K8sEvent              `json:"events,omitempty"`
// }

// // ContainerInfo 容器信息
// type ContainerInfo struct {
// 	Name         string                 `json:"name"`
// 	Image        string                 `json:"image"`
// 	Status       string                 `json:"status"`
// 	Ready        bool                   `json:"ready"`
// 	RestartCount int32                  `json:"restart_count"`
// 	Resources    ContainerResources     `json:"resources"`
// 	Ports        []corev1.ContainerPort `json:"ports,omitempty"`
// 	Env          []corev1.EnvVar        `json:"env,omitempty"`
// 	VolumeMounts []corev1.VolumeMount   `json:"volume_mounts,omitempty"`
// }

// // ContainerResources 容器资源信息
// type ContainerResources struct {
// 	CpuRequest    string `json:"cpu_request"`
// 	CpuLimit      string `json:"cpu_limit"`
// 	MemoryRequest string `json:"memory_request"`
// 	MemoryLimit   string `json:"memory_limit"`
// 	CpuUsage      string `json:"cpu_usage,omitempty"`
// 	MemoryUsage   string `json:"memory_usage,omitempty"`
// }

// // ==================== YAML任务和模板请求结构体 ====================

// // YamlTaskCreateReq 创建YAML任务请求
// type YamlTaskCreateReq struct {
// 	Name       string     `json:"name" binding:"required,min=1,max=255" comment:"YAML任务名称"`
// 	UserID     int        `json:"user_id" comment:"创建者用户ID"`
// 	TemplateID int        `json:"template_id" comment:"关联的模板ID"`
// 	ClusterId  int        `json:"cluster_id" comment:"集群ID"`
// 	Variables  StringList `json:"variables" comment:"yaml变量，格式k=v,k=v"`
// }

// // YamlTaskUpdateReq 更新YAML任务请求
// type YamlTaskUpdateReq struct {
// 	ID         int        `json:"id" binding:"required" comment:"任务ID"`
// 	Name       string     `json:"name" binding:"required,min=1,max=255" comment:"YAML任务名称"`
// 	UserID     int        `json:"user_id" comment:"创建者用户ID"`
// 	TemplateID int        `json:"template_id" comment:"关联的模板ID"`
// 	ClusterId  int        `json:"cluster_id" comment:"集群ID"`
// 	Variables  StringList `json:"variables" comment:"yaml变量，格式k=v,k=v"`
// }

// // YamlTemplateCreateReq 创建YAML模板请求
// type YamlTemplateCreateReq struct {
// 	Name      string `json:"name" binding:"required,min=1,max=50" comment:"模板名称"`
// 	UserID    int    `json:"user_id" comment:"创建者用户ID"`
// 	Content   string `json:"content" binding:"required" comment:"yaml模板内容"`
// 	ClusterId int    `json:"cluster_id" comment:"对应集群ID"`
// }

// // ==================== Resource相关请求结构体 ====================

// // ResourceOverviewReq 资源概览请求
// type ResourceOverviewReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // ResourceStatisticsReq 资源统计请求
// type ResourceStatisticsReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // ResourceDistributionReq 资源分布请求
// type ResourceDistributionReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // ResourceTrendReq 资源趋势请求
// type ResourceTrendReq struct {
// 	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// 	Period    string `json:"period" form:"period" comment:"时间周期: 1h, 6h, 24h, 7d, 30d"`
// }

// // ResourceUtilizationReq 资源利用率请求
// type ResourceUtilizationReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // ResourceHealthReq 资源健康请求
// type ResourceHealthReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // WorkloadDistributionReq 工作负载分布请求
// type WorkloadDistributionReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // NamespaceResourcesReq 命名空间资源请求
// type NamespaceResourcesReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // StorageOverviewReq 存储概览请求
// type StorageOverviewReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // NetworkOverviewReq 网络概览请求
// type NetworkOverviewReq struct {
// 	ClusterID int `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
// }

// // CompareClusterResourcesReq 对比集群资源请求
// type CompareClusterResourcesReq struct {
// 	ClusterIDs []int `json:"cluster_ids" binding:"required,min=2,max=10" comment:"要对比的集群ID列表"`
// }

// // ==================== Resource相关响应结构体 ====================

// // ResourceOverview 资源概览响应
// type ResourceOverview struct {
// 	ClusterID       int              `json:"cluster_id"`
// 	ClusterName     string           `json:"cluster_name"`
// 	Status          string           `json:"status"`
// 	Version         string           `json:"version"`
// 	NodeSummary     NodeSummary      `json:"node_summary"`
// 	PodSummary      PodSummary       `json:"pod_summary"`
// 	ResourceSummary ResourceSummary  `json:"resource_summary"`
// 	HealthStatus    ClusterHealth    `json:"health_status"`
// 	TopNamespaces   []NamespaceUsage `json:"top_namespaces"`
// 	RecentEvents    []EventSummary   `json:"recent_events"`
// }

// // NodeSummary 节点汇总
// type NodeSummary struct {
// 	Total      int     `json:"total"`
// 	Ready      int     `json:"ready"`
// 	NotReady   int     `json:"not_ready"`
// 	Master     int     `json:"master"`
// 	Worker     int     `json:"worker"`
// 	HealthRate float64 `json:"health_rate"`
// }

// // PodSummary Pod汇总
// type PodSummary struct {
// 	Total      int     `json:"total"`
// 	Running    int     `json:"running"`
// 	Pending    int     `json:"pending"`
// 	Failed     int     `json:"failed"`
// 	Succeeded  int     `json:"succeeded"`
// 	HealthRate float64 `json:"health_rate"`
// }

// // ResourceSummary 资源汇总
// type ResourceSummary struct {
// 	CPU     ResourceUsage `json:"cpu"`
// 	Memory  ResourceUsage `json:"memory"`
// 	Storage ResourceUsage `json:"storage"`
// }

// // ResourceUsage 资源使用情况
// type ResourceUsage struct {
// 	Total       string  `json:"total"`
// 	Used        string  `json:"used"`
// 	Available   string  `json:"available"`
// 	Utilization float64 `json:"utilization"`
// }

// // ClusterHealth 集群健康状态
// type ClusterHealth struct {
// 	OverallStatus string                  `json:"overall_status"`
// 	Score         int                     `json:"score"`
// 	Components    []ComponentHealthStatus `json:"components"`
// 	Issues        []string                `json:"issues"`
// }

// // NamespaceUsage 命名空间使用情况
// type NamespaceUsage struct {
// 	Name     string `json:"name"`
// 	PodCount int    `json:"pod_count"`
// 	CPUUsage string `json:"cpu_usage"`
// 	MemUsage string `json:"memory_usage"`
// 	IsSystem bool   `json:"is_system"`
// 	Status   string `json:"status"`
// }

// // ResourceDistribution 资源分布响应
// type ResourceDistribution struct {
// 	ClusterID          int                     `json:"cluster_id"`
// 	NodeDistribution   []NodeResourceDistrib   `json:"node_distribution"`
// 	NSDistribution     []NSResourceDistrib     `json:"namespace_distribution"`
// 	WorkloadDistrib    WorkloadDistribution    `json:"workload_distribution"`
// 	ResourceAllocation ResourceAllocationChart `json:"resource_allocation"`
// }

// // NodeResourceDistrib 节点资源分布
// type NodeResourceDistrib struct {
// 	NodeName   string  `json:"node_name"`
// 	Role       string  `json:"role"`
// 	CPU        string  `json:"cpu"`
// 	Memory     string  `json:"memory"`
// 	Storage    string  `json:"storage"`
// 	PodCount   int     `json:"pod_count"`
// 	CPUUtil    float64 `json:"cpu_utilization"`
// 	MemoryUtil float64 `json:"memory_utilization"`
// 	Status     string  `json:"status"`
// }

// // NSResourceDistrib 命名空间资源分布
// type NSResourceDistrib struct {
// 	Namespace  string  `json:"namespace"`
// 	PodCount   int     `json:"pod_count"`
// 	CPURequest string  `json:"cpu_request"`
// 	CPULimit   string  `json:"cpu_limit"`
// 	MemRequest string  `json:"memory_request"`
// 	MemLimit   string  `json:"memory_limit"`
// 	CPUUtil    float64 `json:"cpu_utilization"`
// 	MemoryUtil float64 `json:"memory_utilization"`
// 	IsSystem   bool    `json:"is_system"`
// }

// // WorkloadDistribution 工作负载分布
// type WorkloadDistribution struct {
// 	Deployments     int                `json:"deployments"`
// 	StatefulSets    int                `json:"statefulsets"`
// 	DaemonSets      int                `json:"daemonsets"`
// 	Jobs            int                `json:"jobs"`
// 	CronJobs        int                `json:"cronjobs"`
// 	Services        int                `json:"services"`
// 	ConfigMaps      int                `json:"configmaps"`
// 	Secrets         int                `json:"secrets"`
// 	Ingresses       int                `json:"ingresses"`
// 	WorkloadsByNS   []NSWorkloadCount  `json:"workloads_by_namespace"`
// 	ResourcesByType []WorkloadResource `json:"resources_by_type"`
// }

// // NSWorkloadCount 命名空间工作负载数量
// type NSWorkloadCount struct {
// 	Namespace string         `json:"namespace"`
// 	Count     int            `json:"count"`
// 	Types     map[string]int `json:"types"`
// }

// // WorkloadResource 工作负载资源使用
// type WorkloadResource struct {
// 	Type       string `json:"type"`
// 	Count      int    `json:"count"`
// 	CPURequest string `json:"cpu_request"`
// 	CPULimit   string `json:"cpu_limit"`
// 	MemRequest string `json:"memory_request"`
// 	MemLimit   string `json:"memory_limit"`
// }

// // ResourceAllocationChart 资源分配图表数据
// type ResourceAllocationChart struct {
// 	CPUChart    PieChartData `json:"cpu_chart"`
// 	MemoryChart PieChartData `json:"memory_chart"`
// 	PodChart    PieChartData `json:"pod_chart"`
// }

// // PieChartData 饼图数据
// type PieChartData struct {
// 	Labels []string  `json:"labels"`
// 	Values []float64 `json:"values"`
// 	Colors []string  `json:"colors"`
// }

// // ResourceTrend 资源趋势响应
// type ResourceTrend struct {
// 	ClusterID   int               `json:"cluster_id"`
// 	Period      string            `json:"period"`
// 	TimeRange   TimeRange         `json:"time_range"`
// 	CPUTrend    TrendData         `json:"cpu_trend"`
// 	MemoryTrend TrendData         `json:"memory_trend"`
// 	PodTrend    TrendData         `json:"pod_trend"`
// 	NodeTrend   TrendData         `json:"node_trend"`
// 	Predictions []ResourcePredict `json:"predictions"`
// }

// // TrendData 趋势数据
// type TrendData struct {
// 	Timestamps []string  `json:"timestamps"`
// 	Values     []float64 `json:"values"`
// 	Unit       string    `json:"unit"`
// 	Max        float64   `json:"max"`
// 	Min        float64   `json:"min"`
// 	Avg        float64   `json:"avg"`
// }

// // ResourcePredict 资源预测
// type ResourcePredict struct {
// 	Resource    string  `json:"resource"`
// 	PredictDays int     `json:"predict_days"`
// 	Tendency    string  `json:"tendency"`
// 	Confidence  float64 `json:"confidence"`
// 	Value       float64 `json:"value"`
// 	Suggestion  string  `json:"suggestion"`
// }

// // ResourceUtilization 资源利用率响应
// type ResourceUtilization struct {
// 	ClusterID       int                 `json:"cluster_id"`
// 	OverallUtil     UtilizationSummary  `json:"overall_utilization"`
// 	NodeUtils       []NodeUtilization   `json:"node_utilizations"`
// 	NSUtils         []NSUtilization     `json:"namespace_utilizations"`
// 	UtilChart       UtilizationChart    `json:"utilization_chart"`
// 	Recommendations []UtilizationAdvice `json:"recommendations"`
// }

// // UtilizationSummary 利用率汇总
// type UtilizationSummary struct {
// 	CPU     float64 `json:"cpu"`
// 	Memory  float64 `json:"memory"`
// 	Storage float64 `json:"storage"`
// 	Network float64 `json:"network"`
// 	Overall float64 `json:"overall"`
// }

// // NodeUtilization 节点利用率
// type NodeUtilization struct {
// 	NodeName   string  `json:"node_name"`
// 	CPU        float64 `json:"cpu"`
// 	Memory     float64 `json:"memory"`
// 	Storage    float64 `json:"storage"`
// 	PodCount   int     `json:"pod_count"`
// 	Status     string  `json:"status"`
// 	Efficiency string  `json:"efficiency"`
// }

// // NSUtilization 命名空间利用率
// type NSUtilization struct {
// 	Namespace string  `json:"namespace"`
// 	CPU       float64 `json:"cpu"`
// 	Memory    float64 `json:"memory"`
// 	PodCount  int     `json:"pod_count"`
// 	IsSystem  bool    `json:"is_system"`
// }

// // UtilizationChart 利用率图表
// type UtilizationChart struct {
// 	HeatmapData [][]float64 `json:"heatmap_data"`
// 	XLabels     []string    `json:"x_labels"`
// 	YLabels     []string    `json:"y_labels"`
// }

// // UtilizationAdvice 利用率建议
// type UtilizationAdvice struct {
// 	Type        string  `json:"type"`
// 	Priority    string  `json:"priority"`
// 	Resource    string  `json:"resource"`
// 	Target      string  `json:"target"`
// 	Current     float64 `json:"current"`
// 	Suggested   float64 `json:"suggested"`
// 	Description string  `json:"description"`
// 	Impact      string  `json:"impact"`
// }

// // ResourceHealth 资源健康响应
// type ResourceHealth struct {
// 	ClusterID        int                `json:"cluster_id"`
// 	OverallHealth    HealthScore        `json:"overall_health"`
// 	ComponentHealth  []ComponentHealth  `json:"component_health"`
// 	ResourceIssues   []ResourceIssue    `json:"resource_issues"`
// 	HealthTrend      []HealthTrendPoint `json:"health_trend"`
// 	ActionableAlerts []ActionableAlert  `json:"actionable_alerts"`
// }

// // HealthScore 健康评分
// type HealthScore struct {
// 	Score       int      `json:"score"`
// 	Level       string   `json:"level"`
// 	Description string   `json:"description"`
// 	Factors     []string `json:"factors"`
// }

// // ComponentHealth 组件健康
// type ComponentHealth struct {
// 	Component string   `json:"component"`
// 	Status    string   `json:"status"`
// 	Score     int      `json:"score"`
// 	Issues    int      `json:"issues"`
// 	LastCheck string   `json:"last_check"`
// 	Details   []string `json:"details"`
// }

// // ResourceIssue 资源问题
// type ResourceIssue struct {
// 	Type        string `json:"type"`
// 	Severity    string `json:"severity"`
// 	Resource    string `json:"resource"`
// 	Namespace   string `json:"namespace"`
// 	Description string `json:"description"`
// 	Since       string `json:"since"`
// 	Suggestion  string `json:"suggestion"`
// }

// // HealthTrendPoint 健康趋势点
// type HealthTrendPoint struct {
// 	Timestamp string `json:"timestamp"`
// 	Score     int    `json:"score"`
// 	Issues    int    `json:"issues"`
// }

// // ActionableAlert 可操作警报
// type ActionableAlert struct {
// 	ID          string   `json:"id"`
// 	Title       string   `json:"title"`
// 	Description string   `json:"description"`
// 	Severity    string   `json:"severity"`
// 	Actions     []string `json:"actions"`
// 	CreatedAt   string   `json:"created_at"`
// }

// // AllClustersSummary 所有集群汇总
// type AllClustersSummary struct {
// 	TotalClusters      int                     `json:"total_clusters"`
// 	HealthyClusters    int                     `json:"healthy_clusters"`
// 	UnhealthyClusters  int                     `json:"unhealthy_clusters"`
// 	TotalResources     GlobalResourceSummary   `json:"total_resources"`
// 	ClustersOverview   []ClusterBriefSummary   `json:"clusters_overview"`
// 	ResourceComparison ResourceComparisonChart `json:"resource_comparison"`
// 	AlertsSummary      GlobalAlertsSummary     `json:"alerts_summary"`
// }

// // GlobalResourceSummary 全局资源汇总
// type GlobalResourceSummary struct {
// 	TotalNodes   int     `json:"total_nodes"`
// 	TotalPods    int     `json:"total_pods"`
// 	TotalCPU     string  `json:"total_cpu"`
// 	TotalMemory  string  `json:"total_memory"`
// 	TotalStorage string  `json:"total_storage"`
// 	AvgCPUUtil   float64 `json:"avg_cpu_utilization"`
// 	AvgMemUtil   float64 `json:"avg_memory_utilization"`
// }

// // ClusterBriefSummary 集群简要汇总
// type ClusterBriefSummary struct {
// 	ClusterID   int     `json:"cluster_id"`
// 	ClusterName string  `json:"cluster_name"`
// 	Status      string  `json:"status"`
// 	HealthScore int     `json:"health_score"`
// 	NodeCount   int     `json:"node_count"`
// 	PodCount    int     `json:"pod_count"`
// 	CPUUtil     float64 `json:"cpu_utilization"`
// 	MemoryUtil  float64 `json:"memory_utilization"`
// 	Issues      int     `json:"issues"`
// }

// // ResourceComparisonChart 资源对比图表
// type ResourceComparisonChart struct {
// 	ClusterNames []string            `json:"cluster_names"`
// 	CPUData      []float64           `json:"cpu_data"`
// 	MemoryData   []float64           `json:"memory_data"`
// 	PodData      []float64           `json:"pod_data"`
// 	Detailed     []ClusterComparison `json:"detailed"`
// }

// // ClusterComparison 集群对比
// type ClusterComparison struct {
// 	ClusterName string  `json:"cluster_name"`
// 	CPU         string  `json:"cpu"`
// 	Memory      string  `json:"memory"`
// 	Nodes       int     `json:"nodes"`
// 	Pods        int     `json:"pods"`
// 	Efficiency  float64 `json:"efficiency"`
// }

// // GlobalAlertsSummary 全局警报汇总
// type GlobalAlertsSummary struct {
// 	TotalAlerts     int            `json:"total_alerts"`
// 	CriticalAlerts  int            `json:"critical_alerts"`
// 	WarningAlerts   int            `json:"warning_alerts"`
// 	InfoAlerts      int            `json:"info_alerts"`
// 	AlertsByCluster map[string]int `json:"alerts_by_cluster"`
// 	TopAlerts       []GlobalAlert  `json:"top_alerts"`
// }

// // GlobalAlert 全局警报
// type GlobalAlert struct {
// 	ClusterName string `json:"cluster_name"`
// 	Type        string `json:"type"`
// 	Message     string `json:"message"`
// 	Severity    string `json:"severity"`
// 	Count       int    `json:"count"`
// 	FirstSeen   string `json:"first_seen"`
// 	LastSeen    string `json:"last_seen"`
// }
