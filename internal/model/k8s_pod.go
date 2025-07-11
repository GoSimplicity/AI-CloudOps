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
	
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sPod 单个 Pod 的模型
type K8sPod struct {
	Model
	Name        string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:Pod 名称"`           // Pod 名称
	Namespace   string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:Pod 所属的命名空间"` // Pod 所属的命名空间
	Status      string            `json:"status" gorm:"comment:Pod 状态，例如 Running, Pending"`                               // Pod 状态，例如 "Running", "Pending"
	NodeName    string            `json:"node_name" gorm:"index;comment:Pod 所在节点名称"`                                      // Pod 所在节点名称
	Labels      map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:Pod 标签键值对"`                      // Pod 标签键值对
	Annotations map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:Pod 注解键值对"`                 // Pod 注解键值对
	Containers  []K8sPodContainer `json:"containers" gorm:"-"`                                                            // Pod 内的容器信息，前端使用
}

// K8sPodContainer Pod 中单个容器的模型
type K8sPodContainer struct {
	Name            string               `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:容器名称"`        // 容器名称
	Image           string               `json:"image" binding:"required" gorm:"size:500;comment:容器镜像"`                     // 容器镜像
	Command         StringList           `json:"command,omitempty" gorm:"type:text;serializer:json;comment:启动命令组"`          // 启动命令组
	Args            StringList           `json:"args,omitempty" gorm:"type:text;serializer:json;comment:启动参数，空格分隔"`         // 启动参数
	Envs            []K8sEnvVar          `json:"envs,omitempty" gorm:"type:text;serializer:json;comment:环境变量组"`             // 环境变量组
	Ports           []K8sContainerPort   `json:"ports,omitempty" gorm:"type:text;serializer:json;comment:容器端口配置"`           // 容器端口配置
	Resources       ResourceRequirements `json:"resources,omitempty" gorm:"type:text;serializer:json;comment:资源请求与限制"`      // 资源请求与限制
	VolumeMounts    []K8sVolumeMount     `json:"volume_mounts,omitempty" gorm:"type:text;serializer:json;comment:卷挂载配置"`    // 卷挂载配置
	LivenessProbe   *K8sProbe            `json:"liveness_probe,omitempty" gorm:"type:text;serializer:json;comment:存活探测配置"`  // 存活探测配置
	ReadinessProbe  *K8sProbe            `json:"readiness_probe,omitempty" gorm:"type:text;serializer:json;comment:就绪探测配置"` // 就绪探测配置
	ImagePullPolicy string               `json:"image_pull_policy,omitempty" gorm:"size:50;comment:镜像拉取策略"`                 // 镜像拉取策略，例如 "Always", "IfNotPresent", "Never"
}

// K8sEnvVar 环境变量的键值对
type K8sEnvVar struct {
	Name  string `json:"name" binding:"required" gorm:"size:100;comment:环境变量名称"` // 环境变量名称
	Value string `json:"value" gorm:"size:500;comment:环境变量值"`                    // 环境变量值
}

// K8sContainerPort 容器的端口配置
type K8sContainerPort struct {
	Name          string `json:"name,omitempty" gorm:"size:100;comment:端口名称"`            // 端口名称（可选）
	ContainerPort int    `json:"container_port" binding:"required" gorm:"comment:容器端口号"` // 容器端口号
	Protocol      string `json:"protocol,omitempty" gorm:"size:10;comment:协议类型"`         // 协议类型，例如 "TCP", "UDP"
}

// K8sVolumeMount 卷的挂载配置
type K8sVolumeMount struct {
	Name      string `json:"name" binding:"required" gorm:"size:100;comment:卷名称"`        // 卷名称，必填，长度限制为100字符
	MountPath string `json:"mount_path" binding:"required" gorm:"size:255;comment:挂载路径"` // 挂载路径，必填，长度限制为255字符
	ReadOnly  bool   `json:"read_only,omitempty" gorm:"comment:是否只读"`                    // 是否只读
	SubPath   string `json:"sub_path,omitempty" gorm:"size:255;comment:子路径"`             // 子路径（可选），长度限制为255字符
}

// K8sProbe 探测配置
type K8sProbe struct {
	HTTPGet *K8sHTTPGetAction `json:"http_get,omitempty" gorm:"type:text;serializer:json;comment:HTTP GET 探测配置"` // HTTP GET 探测
	// TCPSocket 和 Exec 探测也可以根据需要添加
	InitialDelaySeconds int `json:"initial_delay_seconds" gorm:"comment:探测初始延迟时间（秒）"` // 探测初始延迟时间
	PeriodSeconds       int `json:"period_seconds" gorm:"comment:探测间隔时间（秒）"`          // 探测间隔时间
	TimeoutSeconds      int `json:"timeout_seconds" gorm:"comment:探测超时时间（秒）"`         // 探测超时时间
	SuccessThreshold    int `json:"success_threshold" gorm:"comment:探测成功阈值"`          // 探测成功阈值
	FailureThreshold    int `json:"failure_threshold" gorm:"comment:探测失败阈值"`          // 探测失败阈值
}

// K8sHTTPGetAction HTTP GET 探测动作
type K8sHTTPGetAction struct {
	Path   string `json:"path" binding:"required" gorm:"size:255;comment:探测路径"` // 探测路径，必填，长度限制为255字符
	Port   int    `json:"port" binding:"required" gorm:"comment:探测端口号"`         // 探测端口号，必填
	Scheme string `json:"scheme,omitempty" gorm:"size:10;comment:协议类型"`         // 协议类型，例如 "HTTP", "HTTPS"，长度限制为10字符
}

// K8sPodRequest 创建 Pod 的请求结构
type K8sPodRequest struct {
	ClusterId int       `json:"cluster_id" binding:"required"` // 集群名称，必填
	Pod       *core.Pod `json:"pod"`                           // Pod 对象
}

// K8sDeploymentRequest Deployment 相关请求结构
type K8sDeploymentRequest struct {
	ClusterId       int                `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace       string             `json:"namespace" binding:"required"`  // 命名空间，必填
	DeploymentNames []string           `json:"deployment_names"`              // Deployment 名称，可选
	DeploymentYaml  *appsv1.Deployment `json:"deployment_yaml"`               // Deployment 对象, 可选
}

// K8sConfigMapRequest ConfigMap 相关请求结构
type K8sConfigMapRequest struct {
	ClusterId      int             `json:"cluster_id" binding:"required"` // 集群id，必填
	Namespace      string          `json:"namespace"`                     // 命名空间，可选, 删除用
	ConfigMapNames []string        `json:"config_map_names"`              // ConfigMap 名称，可选， 删除用
	ConfigMap      *core.ConfigMap `json:"config_map"`                    // ConfigMap 对象, 可选
}

// K8sServiceRequest Service 相关请求结构
type K8sServiceRequest struct {
	ClusterId    int           `json:"cluster_id" binding:"required"` // 集群id，必填
	Namespace    string        `json:"namespace"`                     // 命名空间，必填
	ServiceNames []string      `json:"service_names"`                 // Service 名称，可选
	ServiceYaml  *core.Service `json:"service_yaml"`                  // Service 对象, 可选
}

// K8sPodListResponse Pod 列表响应
type K8sPodListResponse struct {
	Pods       []K8sPod `json:"pods"`        // Pod 列表
	TotalCount int      `json:"total_count"` // 总数
}

// K8sStatefulSetRequest StatefulSet 相关请求结构
type K8sStatefulSetRequest struct {
	ClusterID        int                 `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace        string              `json:"namespace" binding:"required"`  // 命名空间，必填
	StatefulSetNames []string            `json:"statefulset_names"`             // StatefulSet 名称，可选
	StatefulSetYaml  *appsv1.StatefulSet `json:"statefulset_yaml"`              // StatefulSet 对象, 可选
}

// K8sStatefulSetScaleRequest StatefulSet 扩缩容请求结构
type K8sStatefulSetScaleRequest struct {
	ClusterID       int    `json:"cluster_id" binding:"required"`       // 集群名称，必填
	Namespace       string `json:"namespace" binding:"required"`        // 命名空间，必填
	StatefulSetName string `json:"statefulset_name" binding:"required"` // StatefulSet 名称，必填
	Replicas        int32  `json:"replicas" binding:"required"`         // 副本数量，必填
}

// K8sStatefulSetStatus StatefulSet 状态响应
type K8sStatefulSetStatus struct {
	Name               string    `json:"name"`                // StatefulSet 名称
	Namespace          string    `json:"namespace"`           // 命名空间
	Replicas           int32     `json:"replicas"`            // 期望副本数
	ReadyReplicas      int32     `json:"ready_replicas"`      // 就绪副本数
	CurrentReplicas    int32     `json:"current_replicas"`    // 当前副本数
	UpdatedReplicas    int32     `json:"updated_replicas"`    // 已更新副本数
	AvailableReplicas  int32     `json:"available_replicas"`  // 可用副本数
	CurrentRevision    string    `json:"current_revision"`    // 当前修订版本
	UpdateRevision     string    `json:"update_revision"`     // 更新修订版本
	ObservedGeneration int64     `json:"observed_generation"` // 观察到的代数
	CreationTimestamp  time.Time `json:"creation_timestamp"`  // 创建时间
}

// K8sDaemonSetRequest DaemonSet 相关请求结构
type K8sDaemonSetRequest struct {
	ClusterID       int                 `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace       string              `json:"namespace" binding:"required"`  // 命名空间，必填
	DaemonSetNames  []string            `json:"daemonset_names"`               // DaemonSet 名称，可选
	DaemonSetYaml   *appsv1.DaemonSet   `json:"daemonset_yaml"`                // DaemonSet 对象, 可选
}

// K8sDaemonSetStatus DaemonSet 状态响应
type K8sDaemonSetStatus struct {
	Name                     string    `json:"name"`                       // DaemonSet 名称
	Namespace                string    `json:"namespace"`                  // 命名空间
	DesiredNumberScheduled   int32     `json:"desired_number_scheduled"`   // 期望调度的节点数
	CurrentNumberScheduled   int32     `json:"current_number_scheduled"`   // 当前调度的节点数
	NumberReady              int32     `json:"number_ready"`               // 就绪的Pod数
	UpdatedNumberScheduled   int32     `json:"updated_number_scheduled"`   // 已更新的调度数
	NumberAvailable          int32     `json:"number_available"`           // 可用的Pod数
	NumberUnavailable        int32     `json:"number_unavailable"`         // 不可用的Pod数
	NumberMisscheduled       int32     `json:"number_misscheduled"`        // 错误调度的Pod数
	ObservedGeneration       int64     `json:"observed_generation"`        // 观察到的代数
	CreationTimestamp        time.Time `json:"creation_timestamp"`         // 创建时间
}

// K8sJobRequest Job 相关请求结构
type K8sJobRequest struct {
	ClusterID int           `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace string        `json:"namespace" binding:"required"`  // 命名空间，必填
	JobNames  []string      `json:"job_names"`                     // Job 名称，可选
	JobYaml   *batchv1.Job  `json:"job_yaml"`                      // Job 对象, 可选
}

// K8sJobStatus Job 状态响应
type K8sJobStatus struct {
	Name                  string     `json:"name"`                    // Job 名称
	Namespace             string     `json:"namespace"`               // 命名空间
	Phase                 string     `json:"phase"`                   // Job 阶段 (Pending, Running, Succeeded, Failed)
	Active                int32      `json:"active"`                  // 活跃的Pod数
	Succeeded             int32      `json:"succeeded"`               // 成功的Pod数
	Failed                int32      `json:"failed"`                  // 失败的Pod数
	Completions           *int32     `json:"completions"`             // 期望完成数
	Parallelism           *int32     `json:"parallelism"`             // 并行度
	BackoffLimit          *int32     `json:"backoff_limit"`           // 重试限制
	ActiveDeadlineSeconds *int64     `json:"active_deadline_seconds"` // 活跃截止时间（秒）
	StartTime             *time.Time `json:"start_time"`              // 开始时间
	CompletionTime        *time.Time `json:"completion_time"`         // 完成时间
	CreationTimestamp     time.Time  `json:"creation_timestamp"`      // 创建时间
}

// K8sJobHistory Job 执行历史
type K8sJobHistory struct {
	Name              string     `json:"name"`               // Job 名称
	Namespace         string     `json:"namespace"`          // 命名空间
	Status            string     `json:"status"`             // 状态 (Pending, Running, Succeeded, Failed)
	Active            int32      `json:"active"`             // 活跃的Pod数
	Succeeded         int32      `json:"succeeded"`          // 成功的Pod数
	Failed            int32      `json:"failed"`             // 失败的Pod数
	StartTime         *time.Time `json:"start_time"`         // 开始时间
	CompletionTime    *time.Time `json:"completion_time"`    // 完成时间
	Duration          string     `json:"duration"`           // 执行时长
	CreationTimestamp time.Time  `json:"creation_timestamp"` // 创建时间
}

// K8sPVRequest PersistentVolume 相关请求结构
type K8sPVRequest struct {
	ClusterID  int                      `json:"cluster_id" binding:"required"` // 集群名称，必填
	PVNames    []string                 `json:"pv_names"`                      // PV 名称，可选
	PVYaml     *core.PersistentVolume   `json:"pv_yaml"`                       // PV 对象, 可选
}

// K8sPVStatus PersistentVolume 状态响应
type K8sPVStatus struct {
	Name               string                                 `json:"name"`                 // PV 名称
	Capacity           map[core.ResourceName]string           `json:"capacity"`             // 容量
	Phase              core.PersistentVolumePhase             `json:"phase"`                // 阶段 (Available, Bound, Released, Failed)
	ClaimRef           *core.ObjectReference                  `json:"claim_ref"`            // 绑定的 PVC 引用
	ReclaimPolicy      core.PersistentVolumeReclaimPolicy     `json:"reclaim_policy"`       // 回收策略
	StorageClass       string                                 `json:"storage_class"`        // 存储类
	VolumeMode         *core.PersistentVolumeMode             `json:"volume_mode"`          // 卷模式
	AccessModes        []core.PersistentVolumeAccessMode      `json:"access_modes"`         // 访问模式
	CreationTimestamp  time.Time                              `json:"creation_timestamp"`   // 创建时间
}

// K8sPVCRequest PersistentVolumeClaim 相关请求结构
type K8sPVCRequest struct {
	ClusterID int                           `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace string                        `json:"namespace" binding:"required"`  // 命名空间，必填
	PVCNames  []string                      `json:"pvc_names"`                     // PVC 名称，可选
	PVCYaml   *core.PersistentVolumeClaim   `json:"pvc_yaml"`                      // PVC 对象, 可选
}

// K8sPVCStatus PersistentVolumeClaim 状态响应
type K8sPVCStatus struct {
	Name               string                                 `json:"name"`                 // PVC 名称
	Namespace          string                                 `json:"namespace"`            // 命名空间
	Phase              core.PersistentVolumeClaimPhase        `json:"phase"`                // 阶段 (Pending, Bound, Lost)
	VolumeName         string                                 `json:"volume_name"`          // 绑定的 PV 名称
	Capacity           map[core.ResourceName]string           `json:"capacity"`             // 分配的容量
	RequestedStorage   string                                 `json:"requested_storage"`    // 请求的存储容量
	StorageClass       *string                                `json:"storage_class"`        // 存储类
	VolumeMode         *core.PersistentVolumeMode             `json:"volume_mode"`          // 卷模式
	AccessModes        []core.PersistentVolumeAccessMode      `json:"access_modes"`         // 访问模式
	CreationTimestamp  time.Time                              `json:"creation_timestamp"`   // 创建时间
}

// K8sStorageClassRequest StorageClass 相关请求结构
type K8sStorageClassRequest struct {
	ClusterID          int                       `json:"cluster_id" binding:"required"` // 集群名称，必填
	StorageClassNames  []string                  `json:"storage_class_names"`           // StorageClass 名称，可选
	StorageClassYaml   *storagev1.StorageClass   `json:"storage_class_yaml"`            // StorageClass 对象, 可选
}

// K8sStorageClassStatus StorageClass 状态响应
type K8sStorageClassStatus struct {
	Name                 string                           `json:"name"`                   // StorageClass 名称
	Provisioner          string                           `json:"provisioner"`            // 存储提供者
	Parameters           map[string]string                `json:"parameters"`             // 存储参数
	ReclaimPolicy        *core.PersistentVolumeReclaimPolicy `json:"reclaim_policy"`     // 回收策略
	VolumeBindingMode    *storagev1.VolumeBindingMode     `json:"volume_binding_mode"`    // 卷绑定模式
	AllowVolumeExpansion *bool                            `json:"allow_volume_expansion"` // 是否允许卷扩展
	CreationTimestamp    time.Time                        `json:"creation_timestamp"`     // 创建时间
}

// K8sEndpointRequest Endpoint 相关请求结构
type K8sEndpointRequest struct {
	ClusterID     int            `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace     string         `json:"namespace" binding:"required"`  // 命名空间，必填
	EndpointNames []string       `json:"endpoint_names"`               // Endpoint 名称，可选
	EndpointYaml  *core.Endpoints `json:"endpoint_yaml"`                // Endpoint 对象, 可选
}

// K8sEndpointStatus Endpoint 状态响应
type K8sEndpointStatus struct {
	Name              string                 `json:"name"`               // Endpoint 名称
	Namespace         string                 `json:"namespace"`          // 命名空间
	Subsets           []core.EndpointSubset  `json:"subsets"`            // 端点子集
	Addresses         []string               `json:"addresses"`          // 端点地址列表
	Ports             []core.EndpointPort    `json:"ports"`              // 端点端口列表
	ServiceName       string                 `json:"service_name"`       // 关联的服务名称
	HealthyEndpoints  int                    `json:"healthy_endpoints"`  // 健康端点数量
	UnhealthyEndpoints int                   `json:"unhealthy_endpoints"` // 不健康端点数量
	CreationTimestamp time.Time              `json:"creation_timestamp"` // 创建时间
}

// K8sIngressRequest Ingress 相关请求结构
type K8sIngressRequest struct {
	ClusterID     int                        `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace     string                     `json:"namespace" binding:"required"`  // 命名空间，必填
	IngressNames  []string                   `json:"ingress_names"`                 // Ingress 名称，可选
	IngressYaml   *networkingv1.Ingress      `json:"ingress_yaml"`                  // Ingress 对象, 可选
}

// K8sIngressStatus Ingress 状态响应
type K8sIngressStatus struct {
	Name              string                            `json:"name"`               // Ingress 名称
	Namespace         string                            `json:"namespace"`          // 命名空间
	IngressClass      *string                           `json:"ingress_class"`      // Ingress 类
	Rules             []networkingv1.IngressRule        `json:"rules"`              // Ingress 规则
	TLS               []networkingv1.IngressTLS         `json:"tls"`                // TLS 配置
	Hosts             []string                          `json:"hosts"`              // 主机列表
	Paths             []string                          `json:"paths"`              // 路径列表
	LoadBalancer      networkingv1.IngressLoadBalancerStatus `json:"load_balancer"`     // 负载均衡器状态
	CreationTimestamp time.Time                         `json:"creation_timestamp"` // 创建时间
}

// K8sNetworkPolicyRequest NetworkPolicy 相关请求结构
type K8sNetworkPolicyRequest struct {
	ClusterID           int                              `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace           string                           `json:"namespace" binding:"required"`  // 命名空间，必填
	NetworkPolicyNames  []string                         `json:"network_policy_names"`          // NetworkPolicy 名称，可选
	NetworkPolicyYaml   *networkingv1.NetworkPolicy      `json:"network_policy_yaml"`           // NetworkPolicy 对象, 可选
}

// K8sNetworkPolicyStatus NetworkPolicy 状态响应
type K8sNetworkPolicyStatus struct {
	Name              string                                `json:"name"`               // NetworkPolicy 名称
	Namespace         string                                `json:"namespace"`          // 命名空间
	PodSelector       *metav1.LabelSelector          `json:"pod_selector"`       // Pod 选择器
	PolicyTypes       []networkingv1.PolicyType             `json:"policy_types"`       // 策略类型 (Ingress/Egress)
	Ingress           []networkingv1.NetworkPolicyIngressRule `json:"ingress"`           // 入站规则
	Egress            []networkingv1.NetworkPolicyEgressRule  `json:"egress"`            // 出站规则
	CreationTimestamp time.Time                             `json:"creation_timestamp"` // 创建时间
}

// K8sSecretRequest Secret 相关请求结构
type K8sSecretRequest struct {
	ClusterID   int           `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace   string        `json:"namespace" binding:"required"`  // 命名空间，必填
	SecretNames []string      `json:"secret_names"`                  // Secret 名称，可选
	SecretYaml  *core.Secret  `json:"secret_yaml"`                   // Secret 对象, 可选
}

// K8sSecretStatus Secret 状态响应
type K8sSecretStatus struct {
	Name              string            `json:"name"`               // Secret 名称
	Namespace         string            `json:"namespace"`          // 命名空间
	Type              core.SecretType   `json:"type"`               // Secret 类型
	DataKeys          []string          `json:"data_keys"`          // 数据键列表（不包含敏感值）
	DataSize          int               `json:"data_size"`          // 数据总大小
	Immutable         *bool             `json:"immutable"`          // 是否不可变
	CreationTimestamp time.Time         `json:"creation_timestamp"` // 创建时间
}

// K8sSecretEncryptionRequest Secret 加密请求
type K8sSecretEncryptionRequest struct {
	ClusterID int                       `json:"cluster_id" binding:"required"` // 集群ID
	Namespace string                    `json:"namespace" binding:"required"`  // 命名空间
	Name      string                    `json:"name" binding:"required"`       // Secret 名称
	Type      core.SecretType           `json:"type"`                          // Secret 类型
	Data      map[string]string         `json:"data"`                          // 明文数据
	StringData map[string]string        `json:"string_data"`                   // 字符串数据
	Immutable *bool                     `json:"immutable"`                     // 是否不可变
}

// K8sConfigMapVersionRequest ConfigMap 版本管理请求
type K8sConfigMapVersionRequest struct {
	ClusterID     int                         `json:"cluster_id" binding:"required"` // 集群ID
	Namespace     string                      `json:"namespace" binding:"required"`  // 命名空间
	ConfigMapName string                      `json:"configmap_name" binding:"required"` // ConfigMap 名称
	Version       string                      `json:"version"`                       // 版本号
	Description   string                      `json:"description"`                   // 版本描述
	ConfigMap     *core.ConfigMap             `json:"config_map"`                    // ConfigMap 对象
}

// K8sConfigMapVersion ConfigMap 版本信息
type K8sConfigMapVersion struct {
	Version           string                      `json:"version"`            // 版本号
	Description       string                      `json:"description"`        // 版本描述
	ConfigMap         *core.ConfigMap             `json:"config_map"`         // ConfigMap 对象
	CreationTimestamp time.Time                   `json:"creation_timestamp"` // 创建时间
	Author            string                      `json:"author"`             // 创建者
}

// K8sConfigMapHotReloadRequest ConfigMap 热更新请求
type K8sConfigMapHotReloadRequest struct {
	ClusterID     int                         `json:"cluster_id" binding:"required"` // 集群ID
	Namespace     string                      `json:"namespace" binding:"required"`  // 命名空间
	ConfigMapName string                      `json:"configmap_name" binding:"required"` // ConfigMap 名称
	ReloadType    string                      `json:"reload_type"`                   // 重载类型 (pods, deployments, all)
	TargetSelector map[string]string          `json:"target_selector"`               // 目标选择器
}

// K8sConfigMapRollbackRequest ConfigMap 回滚请求
type K8sConfigMapRollbackRequest struct {
	ClusterID     int                         `json:"cluster_id" binding:"required"` // 集群ID
	Namespace     string                      `json:"namespace" binding:"required"`  // 命名空间
	ConfigMapName string                      `json:"configmap_name" binding:"required"` // ConfigMap 名称
	TargetVersion string                      `json:"target_version" binding:"required"` // 目标版本
}

// K8sResourceQuotaRequest ResourceQuota 相关请求结构
type K8sResourceQuotaRequest struct {
	ClusterID            int                    `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace            string                 `json:"namespace" binding:"required"`  // 命名空间，必填
	ResourceQuotaNames   []string               `json:"resource_quota_names"`          // ResourceQuota 名称列表，批量删除用
	ResourceQuotaYaml    *core.ResourceQuota    `json:"resource_quota_yaml"`           // ResourceQuota 对象，可选
}

// K8sResourceQuotaUsage ResourceQuota 使用情况响应
type K8sResourceQuotaUsage struct {
	Name              string                 `json:"name"`               // ResourceQuota 名称
	Namespace         string                 `json:"namespace"`          // 命名空间
	Hard              map[string]string      `json:"hard"`               // 资源配额限制
	Used              map[string]string      `json:"used"`               // 当前使用量
	UsagePercentage   map[string]float64     `json:"usage_percentage"`   // 使用率百分比
	CreationTimestamp time.Time              `json:"creation_timestamp"` // 创建时间
}

// K8sResourceQuotaStatus ResourceQuota 状态响应
type K8sResourceQuotaStatus struct {
	Name              string                 `json:"name"`               // ResourceQuota 名称
	Namespace         string                 `json:"namespace"`          // 命名空间
	Hard              map[string]string      `json:"hard"`               // 资源配额限制
	Used              map[string]string      `json:"used"`               // 当前使用量
	Scopes            []string               `json:"scopes"`             // 资源范围
	CreationTimestamp time.Time              `json:"creation_timestamp"` // 创建时间
}

// K8sLimitRangeRequest LimitRange 相关请求结构
type K8sLimitRangeRequest struct {
	ClusterID         int                    `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace         string                 `json:"namespace" binding:"required"`  // 命名空间，必填
	LimitRangeNames   []string               `json:"limit_range_names"`             // LimitRange 名称列表，批量删除用
	LimitRangeYaml    *core.LimitRange       `json:"limit_range_yaml"`              // LimitRange 对象，可选
}

// K8sLimitRangeStatus LimitRange 状态响应
type K8sLimitRangeStatus struct {
	Name              string                 `json:"name"`               // LimitRange 名称
	Namespace         string                 `json:"namespace"`          // 命名空间
	Limits            []core.LimitRangeItem  `json:"limits"`             // 限制项列表
	CreationTimestamp time.Time              `json:"creation_timestamp"` // 创建时间
}

// K8sLabelRequest 标签管理相关请求结构
type K8sLabelRequest struct {
	ClusterID      int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace      string                 `json:"namespace"`                          // 命名空间，可选
	ResourceType   string                 `json:"resource_type" binding:"required"`   // 资源类型，必填
	ResourceName   string                 `json:"resource_name"`                      // 资源名称，可选
	Labels         map[string]string      `json:"labels"`                             // 标签键值对
	Annotations    map[string]string      `json:"annotations"`                        // 注解键值对
	LabelSelector  map[string]string      `json:"label_selector"`                     // 标签选择器
	Operation      string                 `json:"operation"`                          // 操作类型 (add, update, delete)
	ResourceNames  []string               `json:"resource_names"`                     // 批量操作的资源名称列表
}

// K8sLabelResponse 标签管理响应结构
type K8sLabelResponse struct {
	ResourceType      string                 `json:"resource_type"`       // 资源类型
	ResourceName      string                 `json:"resource_name"`       // 资源名称
	Namespace         string                 `json:"namespace"`           // 命名空间
	Labels            map[string]string      `json:"labels"`              // 标签键值对
	Annotations       map[string]string      `json:"annotations"`         // 注解键值对
	CreationTimestamp time.Time              `json:"creation_timestamp"`  // 创建时间
}

// K8sLabelSelectorRequest 标签选择器查询请求
type K8sLabelSelectorRequest struct {
	ClusterID     int                    `json:"cluster_id" binding:"required"`   // 集群ID，必填
	Namespace     string                 `json:"namespace"`                       // 命名空间，可选
	ResourceType  string                 `json:"resource_type" binding:"required"` // 资源类型，必填
	LabelSelector map[string]string      `json:"label_selector"`                  // 标签选择器
	FieldSelector string                 `json:"field_selector"`                  // 字段选择器
	Limit         int                    `json:"limit"`                           // 限制数量
}

// K8sLabelPolicyRequest 标签策略请求
type K8sLabelPolicyRequest struct {
	ClusterID     int                    `json:"cluster_id" binding:"required"`   // 集群ID，必填
	Namespace     string                 `json:"namespace"`                       // 命名空间，可选
	PolicyName    string                 `json:"policy_name" binding:"required"`  // 策略名称，必填
	PolicyType    string                 `json:"policy_type"`                     // 策略类型 (required, forbidden, preferred)
	ResourceType  string                 `json:"resource_type"`                   // 资源类型
	LabelRules    []K8sLabelRule         `json:"label_rules"`                     // 标签规则
	Enabled       bool                   `json:"enabled"`                         // 是否启用
	Description   string                 `json:"description"`                     // 策略描述
}

// K8sLabelRule 标签规则
type K8sLabelRule struct {
	Key         string   `json:"key"`         // 标签键
	Values      []string `json:"values"`      // 标签值列表
	Operator    string   `json:"operator"`    // 操作符 (In, NotIn, Exists, DoesNotExist)
	Required    bool     `json:"required"`    // 是否必需
	Description string   `json:"description"` // 规则描述
}

// K8sLabelComplianceRequest 标签合规性检查请求
type K8sLabelComplianceRequest struct {
	ClusterID    int                    `json:"cluster_id" binding:"required"`   // 集群ID，必填
	Namespace    string                 `json:"namespace"`                       // 命名空间，可选
	ResourceType string                 `json:"resource_type"`                   // 资源类型，可选
	PolicyName   string                 `json:"policy_name"`                     // 策略名称，可选
	CheckAll     bool                   `json:"check_all"`                       // 是否检查所有资源
}

// K8sLabelComplianceResponse 标签合规性检查响应
type K8sLabelComplianceResponse struct {
	ResourceType      string                 `json:"resource_type"`       // 资源类型
	ResourceName      string                 `json:"resource_name"`       // 资源名称
	Namespace         string                 `json:"namespace"`           // 命名空间
	PolicyName        string                 `json:"policy_name"`         // 策略名称
	Compliant         bool                   `json:"compliant"`           // 是否合规
	ViolationReason   string                 `json:"violation_reason"`    // 违规原因
	MissingLabels     []string               `json:"missing_labels"`      // 缺失的标签
	ExtraLabels       []string               `json:"extra_labels"`        // 多余的标签
	CheckTime         time.Time              `json:"check_time"`          // 检查时间
}

// K8sLabelBatchRequest 批量标签操作请求
type K8sLabelBatchRequest struct {
	ClusterID     int                    `json:"cluster_id" binding:"required"`   // 集群ID，必填
	Namespace     string                 `json:"namespace"`                       // 命名空间，可选
	ResourceType  string                 `json:"resource_type" binding:"required"` // 资源类型，必填
	ResourceNames []string               `json:"resource_names"`                  // 资源名称列表
	Operation     string                 `json:"operation" binding:"required"`    // 操作类型 (add, update, delete)
	Labels        map[string]string      `json:"labels"`                          // 标签键值对
	LabelSelector map[string]string      `json:"label_selector"`                  // 标签选择器（用于批量选择）
}

// K8sLabelHistoryRequest 标签历史记录请求
type K8sLabelHistoryRequest struct {
	ClusterID    int                    `json:"cluster_id" binding:"required"`   // 集群ID，必填
	Namespace    string                 `json:"namespace"`                       // 命名空间，可选
	ResourceType string                 `json:"resource_type"`                   // 资源类型，可选
	ResourceName string                 `json:"resource_name"`                   // 资源名称，可选
	StartTime    *time.Time             `json:"start_time"`                      // 开始时间
	EndTime      *time.Time             `json:"end_time"`                        // 结束时间
	Limit        int                    `json:"limit"`                           // 限制数量
}

// K8sLabelHistoryResponse 标签历史记录响应
type K8sLabelHistoryResponse struct {
	ID               int                    `json:"id"`                 // 记录ID
	ClusterID        int                    `json:"cluster_id"`         // 集群ID
	Namespace        string                 `json:"namespace"`          // 命名空间
	ResourceType     string                 `json:"resource_type"`      // 资源类型
	ResourceName     string                 `json:"resource_name"`      // 资源名称
	Operation        string                 `json:"operation"`          // 操作类型
	OldLabels        map[string]string      `json:"old_labels"`         // 原标签
	NewLabels        map[string]string      `json:"new_labels"`         // 新标签
	ChangedBy        string                 `json:"changed_by"`         // 修改者
	ChangeTime       time.Time              `json:"change_time"`        // 修改时间
	ChangeReason     string                 `json:"change_reason"`      // 修改原因
}

// K8sNodeAffinityRequest 节点亲和性请求
type K8sNodeAffinityRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace" binding:"required"`       // 命名空间，必填
	ResourceType           string                 `json:"resource_type" binding:"required"`   // 资源类型，必填
	ResourceName           string                 `json:"resource_name" binding:"required"`   // 资源名称，必填
	RequiredAffinity       []K8sNodeSelectorTerm  `json:"required_affinity"`                  // 硬亲和性规则
	PreferredAffinity      []K8sPreferredSchedulingTerm `json:"preferred_affinity"`        // 软亲和性规则
	NodeSelector           map[string]string      `json:"node_selector"`                      // 节点选择器
	Operation              string                 `json:"operation"`                          // 操作类型 (add, update, delete)
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
	Weight     int32                `json:"weight"`     // 权重 (1-100)
	Preference K8sNodeSelectorTerm  `json:"preference"` // 偏好条件
}

// K8sNodeAffinityResponse 节点亲和性响应
type K8sNodeAffinityResponse struct {
	ResourceType      string                       `json:"resource_type"`       // 资源类型
	ResourceName      string                       `json:"resource_name"`       // 资源名称
	Namespace         string                       `json:"namespace"`           // 命名空间
	RequiredAffinity  []K8sNodeSelectorTerm        `json:"required_affinity"`   // 硬亲和性规则
	PreferredAffinity []K8sPreferredSchedulingTerm `json:"preferred_affinity"`  // 软亲和性规则
	NodeSelector      map[string]string            `json:"node_selector"`       // 节点选择器
	CreationTimestamp time.Time                    `json:"creation_timestamp"`  // 创建时间
}

// K8sNodeAffinityValidationRequest 节点亲和性验证请求
type K8sNodeAffinityValidationRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace"`                          // 命名空间，可选
	RequiredAffinity       []K8sNodeSelectorTerm  `json:"required_affinity"`                  // 硬亲和性规则
	PreferredAffinity      []K8sPreferredSchedulingTerm `json:"preferred_affinity"`        // 软亲和性规则
	NodeSelector           map[string]string      `json:"node_selector"`                      // 节点选择器
	SimulateScheduling     bool                   `json:"simulate_scheduling"`               // 是否模拟调度
}

// K8sNodeAffinityValidationResponse 节点亲和性验证响应
type K8sNodeAffinityValidationResponse struct {
	Valid              bool                   `json:"valid"`                // 是否有效
	MatchingNodes      []string               `json:"matching_nodes"`       // 匹配的节点列表
	ValidationErrors   []string               `json:"validation_errors"`    // 验证错误
	Suggestions        []string               `json:"suggestions"`          // 建议
	SchedulingResult   string                 `json:"scheduling_result"`    // 调度结果
	ValidationTime     time.Time              `json:"validation_time"`      // 验证时间
}

// K8sPodAffinityRequest Pod 亲和性请求
type K8sPodAffinityRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace" binding:"required"`       // 命名空间，必填
	ResourceType           string                 `json:"resource_type" binding:"required"`   // 资源类型，必填
	ResourceName           string                 `json:"resource_name" binding:"required"`   // 资源名称，必填
	PodAffinity            []K8sPodAffinityTerm   `json:"pod_affinity"`                       // Pod 亲和性
	PodAntiAffinity        []K8sPodAffinityTerm   `json:"pod_anti_affinity"`                 // Pod 反亲和性
	TopologyKey            string                 `json:"topology_key"`                       // 拓扑键
	Operation              string                 `json:"operation"`                          // 操作类型 (add, update, delete)
}

// K8sPodAffinityTerm Pod 亲和性条件
type K8sPodAffinityTerm struct {
	LabelSelector      K8sLabelSelector       `json:"label_selector"`       // 标签选择器
	Namespaces         []string               `json:"namespaces"`           // 命名空间列表
	TopologyKey        string                 `json:"topology_key"`         // 拓扑键
	NamespaceSelector  *K8sLabelSelector      `json:"namespace_selector"`   // 命名空间选择器
	Weight             int32                  `json:"weight,omitempty"`     // 权重（仅用于软亲和性）
}

// K8sPodAntiAffinityTerm Pod 反亲和性条件
type K8sPodAntiAffinityTerm struct {
	RequiredDuringSchedulingIgnoredDuringExecution     []K8sPodAffinityTermSpec          `json:"required_during_scheduling_ignored_during_execution"`     // 硬反亲和性
	PreferredDuringSchedulingIgnoredDuringExecution    []K8sWeightedPodAffinityTerm      `json:"preferred_during_scheduling_ignored_during_execution"`    // 软反亲和性
}

// K8sPodAffinityTermSpec Pod 亲和性条件规格
type K8sPodAffinityTermSpec struct {
	LabelSelector      *K8sLabelSelector      `json:"label_selector"`       // 标签选择器
	Namespaces         []string               `json:"namespaces"`           // 命名空间列表
	TopologyKey        string                 `json:"topology_key"`         // 拓扑键
	NamespaceSelector  *K8sLabelSelector      `json:"namespace_selector"`   // 命名空间选择器
}

// K8sLabelSelector 标签选择器
type K8sLabelSelector struct {
	MatchLabels      map[string]string                `json:"match_labels"`       // 匹配标签
	MatchExpressions []K8sLabelSelectorRequirement    `json:"match_expressions"`  // 匹配表达式
}

// K8sLabelSelectorRequirement 标签选择器要求
type K8sLabelSelectorRequirement struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符 (In, NotIn, Exists, DoesNotExist)
	Values   []string `json:"values"`   // 值列表
}

// K8sWeightedPodAffinityTerm 带权重的 Pod 亲和性条件
type K8sWeightedPodAffinityTerm struct {
	Weight          int32                    `json:"weight"`           // 权重 (1-100)
	PodAffinityTerm K8sPodAffinityTermSpec   `json:"pod_affinity_term"` // Pod 亲和性条件
}

// K8sPodAffinityResponse Pod 亲和性响应
type K8sPodAffinityResponse struct {
	ResourceType      string                   `json:"resource_type"`       // 资源类型
	ResourceName      string                   `json:"resource_name"`       // 资源名称
	Namespace         string                   `json:"namespace"`           // 命名空间
	PodAffinity       []K8sPodAffinityTerm     `json:"pod_affinity"`        // Pod 亲和性
	PodAntiAffinity   []K8sPodAffinityTerm     `json:"pod_anti_affinity"`   // Pod 反亲和性
	TopologyKey       string                   `json:"topology_key"`        // 拓扑键
	TopologyDomains   []string                 `json:"topology_domains"`    // 拓扑域列表
	CreationTimestamp time.Time                `json:"creation_timestamp"`  // 创建时间
}

// K8sPodAffinityValidationRequest Pod 亲和性验证请求
type K8sPodAffinityValidationRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace"`                          // 命名空间，可选
	PodAffinity            *K8sPodAffinityTerm    `json:"pod_affinity"`                       // Pod 亲和性
	PodAntiAffinity        *K8sPodAntiAffinityTerm `json:"pod_anti_affinity"`                 // Pod 反亲和性
	SimulateScheduling     bool                   `json:"simulate_scheduling"`               // 是否模拟调度
}

// K8sPodAffinityValidationResponse Pod 亲和性验证响应
type K8sPodAffinityValidationResponse struct {
	Valid              bool                   `json:"valid"`                // 是否有效
	MatchingPods       []string               `json:"matching_pods"`        // 匹配的 Pod 列表
	ValidationErrors   []string               `json:"validation_errors"`    // 验证错误
	Suggestions        []string               `json:"suggestions"`          // 建议
	SchedulingResult   string                 `json:"scheduling_result"`    // 调度结果
	ValidationTime     time.Time              `json:"validation_time"`      // 验证时间
}

// K8sTaintTolerationRequest 污点容忍请求
type K8sTaintTolerationRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace" binding:"required"`       // 命名空间，必填
	ResourceType           string                 `json:"resource_type" binding:"required"`   // 资源类型，必填
	ResourceName           string                 `json:"resource_name" binding:"required"`   // 资源名称，必填
	Tolerations            []K8sToleration        `json:"tolerations"`                        // 容忍度列表
	NodeTaints             []K8sTaint             `json:"node_taints"`                        // 节点污点列表（用于验证）
	Operation              string                 `json:"operation"`                          // 操作类型 (add, update, delete)
}

// K8sToleration 容忍度
type K8sToleration struct {
	Key               string  `json:"key"`                // 键
	Operator          string  `json:"operator"`           // 操作符 (Exists, Equal)
	Value             string  `json:"value"`              // 值
	Effect            string  `json:"effect"`             // 效果 (NoSchedule, PreferNoSchedule, NoExecute)
	TolerationSeconds *int64  `json:"toleration_seconds"` // 容忍时间（秒）
}

// K8sTaint 污点
type K8sTaint struct {
	Key    string `json:"key"`    // 键
	Value  string `json:"value"`  // 值
	Effect string `json:"effect"` // 效果 (NoSchedule, PreferNoSchedule, NoExecute)
}

// K8sTaintTolerationResponse 污点容忍响应
type K8sTaintTolerationResponse struct {
	ResourceType      string                 `json:"resource_type"`       // 资源类型
	ResourceName      string                 `json:"resource_name"`       // 资源名称
	Namespace         string                 `json:"namespace"`           // 命名空间
	Tolerations       []K8sToleration        `json:"tolerations"`         // 容忍度列表
	CompatibleNodes   []string               `json:"compatible_nodes"`    // 兼容的节点列表
	CreationTimestamp time.Time              `json:"creation_timestamp"`  // 创建时间
}

// K8sTaintTolerationValidationRequest 污点容忍验证请求
type K8sTaintTolerationValidationRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace"`                          // 命名空间，可选
	Tolerations            []K8sToleration        `json:"tolerations"`                        // 容忍度列表
	NodeName               string                 `json:"node_name"`                          // 节点名称，可选
	CheckAllNodes          bool                   `json:"check_all_nodes"`                    // 是否检查所有节点
	SimulateScheduling     bool                   `json:"simulate_scheduling"`               // 是否模拟调度
}

// K8sTaintTolerationValidationResponse 污点容忍验证响应
type K8sTaintTolerationValidationResponse struct {
	Valid              bool                   `json:"valid"`                // 是否有效
	CompatibleNodes    []string               `json:"compatible_nodes"`     // 兼容的节点列表
	IncompatibleNodes  []string               `json:"incompatible_nodes"`   // 不兼容的节点列表
	ValidationErrors   []string               `json:"validation_errors"`    // 验证错误
	Suggestions        []string               `json:"suggestions"`          // 建议
	SchedulingResult   string                 `json:"scheduling_result"`    // 调度结果
	ValidationTime     time.Time              `json:"validation_time"`      // 验证时间
}

// K8sNodeTaintRequest 节点污点管理请求
type K8sNodeTaintRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	NodeName               string                 `json:"node_name" binding:"required"`       // 节点名称，必填
	Taints                 []K8sTaint             `json:"taints"`                             // 污点列表
	Operation              string                 `json:"operation"`                          // 操作类型 (add, update, delete)
}

// K8sNodeTaintResponse 节点污点管理响应
type K8sNodeTaintResponse struct {
	NodeName           string                 `json:"node_name"`           // 节点名称
	Taints             []K8sTaint             `json:"taints"`              // 污点列表
	AffectedPods       []string               `json:"affected_pods"`       // 受影响的 Pod 列表
	Operation          string                 `json:"operation"`           // 操作类型
	OperationTime      time.Time              `json:"operation_time"`      // 操作时间
}

// K8sAffinityVisualizationRequest 亲和性可视化请求
type K8sAffinityVisualizationRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace"`                          // 命名空间，可选
	ResourceType           string                 `json:"resource_type"`                      // 资源类型，可选
	ResourceName           string                 `json:"resource_name"`                      // 资源名称，可选
	VisualizationType      string                 `json:"visualization_type"`                 // 可视化类型 (node_affinity, pod_affinity, taint_toleration)
	IncludeDetails         bool                   `json:"include_details"`                    // 是否包含详细信息
}

// K8sAffinityVisualizationResponse 亲和性可视化响应
type K8sAffinityVisualizationResponse struct {
	ClusterID              int                    `json:"cluster_id"`             // 集群ID
	Namespace              string                 `json:"namespace"`              // 命名空间
	Visualization          map[string]interface{} `json:"visualization"`          // 可视化数据
	GeneratedTime          time.Time              `json:"generated_time"`         // 生成时间
}

// K8sNodeRelationship 节点关系
type K8sNodeRelationship struct {
	SourceNode        string            `json:"source_node"`         // 源节点
	TargetNode        string            `json:"target_node"`         // 目标节点
	RelationshipType  string            `json:"relationship_type"`   // 关系类型
	Labels            map[string]string `json:"labels"`              // 标签
	Taints            []K8sTaint        `json:"taints"`              // 污点
	Strength          float64           `json:"strength"`            // 关系强度
}

// K8sPodRelationship Pod 关系
type K8sPodRelationship struct {
	SourcePod         string            `json:"source_pod"`          // 源 Pod
	TargetPod         string            `json:"target_pod"`          // 目标 Pod
	RelationshipType  string            `json:"relationship_type"`   // 关系类型 (affinity, anti-affinity)
	TopologyKey       string            `json:"topology_key"`        // 拓扑键
	Labels            map[string]string `json:"labels"`              // 标签
	Weight            int32             `json:"weight"`              // 权重
	Namespace         string            `json:"namespace"`           // 命名空间
}

// K8sTolerationConfigRequest 容忍度配置请求
type K8sTolerationConfigRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace"`                          // 命名空间，可选
	ResourceType           string                 `json:"resource_type"`                      // 资源类型，可选
	ResourceName           string                 `json:"resource_name"`                      // 资源名称，可选
	TolerationTemplate     K8sTolerationTemplate  `json:"toleration_template"`                // 容忍度模板
	ApplyToExisting        bool                   `json:"apply_to_existing"`                  // 是否应用到现有资源
	AutoUpdate             bool                   `json:"auto_update"`                        // 是否自动更新
	Description            string                 `json:"description"`                        // 配置描述
}

// K8sTolerationTemplate 容忍度模板
type K8sTolerationTemplate struct {
	Name                   string                 `json:"name"`                    // 模板名称
	Tolerations            []K8sTolerationSpec    `json:"tolerations"`             // 容忍度规格列表
	DefaultTolerationTime  *int64                 `json:"default_toleration_time"` // 默认容忍时间
	EffectPriority         []string               `json:"effect_priority"`         // 效果优先级
	AutoCleanup            bool                   `json:"auto_cleanup"`            // 自动清理
	Tags                   map[string]string      `json:"tags"`                    // 标签
}

// K8sTolerationSpec 增强的容忍度规格
type K8sTolerationSpec struct {
	Key               string               `json:"key"`                // 键
	Operator          string               `json:"operator"`           // 操作符 (Exists, Equal)
	Value             string               `json:"value"`              // 值
	Effect            string               `json:"effect"`             // 效果 (NoSchedule, PreferNoSchedule, NoExecute)
	TolerationSeconds *int64               `json:"toleration_seconds"` // 容忍时间（秒）
	Priority          int                  `json:"priority"`           // 优先级
	Conditions        []TolerationCondition `json:"conditions"`        // 容忍条件
	Metadata          map[string]string    `json:"metadata"`           // 元数据
}

// TolerationCondition 容忍条件
type TolerationCondition struct {
	Type               string    `json:"type"`                // 条件类型
	Status             string    `json:"status"`              // 状态
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`              // 原因
	Message            string    `json:"message"`             // 消息
}

// K8sTaintEffectManagementRequest 污点效果管理请求
type K8sTaintEffectManagementRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	NodeName               string                 `json:"node_name"`                          // 节点名称，可选
	NodeSelector           map[string]string      `json:"node_selector"`                      // 节点选择器
	TaintEffectConfig      K8sTaintEffectConfig   `json:"taint_effect_config"`                // 污点效果配置
	BatchOperation         bool                   `json:"batch_operation"`                    // 批量操作
	GracePeriod            *int64                 `json:"grace_period"`                       // 优雅期限
	ForceEviction          bool                   `json:"force_eviction"`                     // 强制驱逐
}

// K8sTaintEffectConfig 污点效果配置
type K8sTaintEffectConfig struct {
	NoScheduleConfig       NoScheduleConfig       `json:"no_schedule_config"`        // NoSchedule配置
	PreferNoScheduleConfig PreferNoScheduleConfig `json:"prefer_no_schedule_config"` // PreferNoSchedule配置
	NoExecuteConfig        NoExecuteConfig        `json:"no_execute_config"`         // NoExecute配置
	EffectTransition       EffectTransition       `json:"effect_transition"`         // 效果转换
}

// NoScheduleConfig NoSchedule效果配置
type NoScheduleConfig struct {
	Enabled                bool                   `json:"enabled"`                 // 是否启用
	ExceptionPods          []string               `json:"exception_pods"`          // 例外Pod列表
	GracefulHandling       bool                   `json:"graceful_handling"`       // 优雅处理
	NotificationConfig     NotificationConfig     `json:"notification_config"`     // 通知配置
}

// PreferNoScheduleConfig PreferNoSchedule效果配置
type PreferNoScheduleConfig struct {
	Enabled                bool                   `json:"enabled"`                 // 是否启用
	PreferenceWeight       int32                  `json:"preference_weight"`       // 偏好权重
	FallbackStrategy       string                 `json:"fallback_strategy"`       // 回退策略
	MonitoringEnabled      bool                   `json:"monitoring_enabled"`      // 监控启用
}

// NoExecuteConfig NoExecute效果配置
type NoExecuteConfig struct {
	Enabled                bool                   `json:"enabled"`                 // 是否启用
	EvictionTimeout        *int64                 `json:"eviction_timeout"`        // 驱逐超时
	GracefulEviction       bool                   `json:"graceful_eviction"`       // 优雅驱逐
	EvictionPolicy         EvictionPolicy         `json:"eviction_policy"`         // 驱逐策略
	RetryConfig            RetryConfig            `json:"retry_config"`            // 重试配置
}

// EffectTransition 效果转换配置
type EffectTransition struct {
	AllowTransition        bool                   `json:"allow_transition"`        // 允许转换
	TransitionRules        []TransitionRule       `json:"transition_rules"`        // 转换规则
	TransitionDelay        *int64                 `json:"transition_delay"`        // 转换延迟
}

// EvictionPolicy 驱逐策略
type EvictionPolicy struct {
	Strategy               string                 `json:"strategy"`                // 策略 (immediate, graceful, delayed)
	MaxEvictionRate        string                 `json:"max_eviction_rate"`       // 最大驱逐率
	PodDisruptionBudget    string                 `json:"pod_disruption_budget"`   // Pod中断预算
	RescheduleAttempts     int                    `json:"reschedule_attempts"`     // 重调度尝试次数
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries             int                    `json:"max_retries"`             // 最大重试次数
	RetryInterval          *int64                 `json:"retry_interval"`          // 重试间隔
	BackoffStrategy        string                 `json:"backoff_strategy"`        // 退避策略
	RetryConditions        []string               `json:"retry_conditions"`        // 重试条件
}

// TransitionRule 转换规则
type TransitionRule struct {
	FromEffect             string                 `json:"from_effect"`             // 源效果
	ToEffect               string                 `json:"to_effect"`               // 目标效果
	Condition              string                 `json:"condition"`               // 条件
	AutoApply              bool                   `json:"auto_apply"`              // 自动应用
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled                bool                   `json:"enabled"`                 // 是否启用
	Channels               []string               `json:"channels"`                // 通知渠道
	Template               string                 `json:"template"`                // 通知模板
	Severity               string                 `json:"severity"`                // 严重程度
}

// K8sTolerationTimeRequest 容忍时间设置请求
type K8sTolerationTimeRequest struct {
	ClusterID              int                    `json:"cluster_id" binding:"required"`      // 集群ID，必填
	Namespace              string                 `json:"namespace"`                          // 命名空间，可选
	ResourceType           string                 `json:"resource_type"`                      // 资源类型，可选
	ResourceName           string                 `json:"resource_name"`                      // 资源名称，可选
	TimeConfig             TolerationTimeConfig   `json:"time_config"`                        // 时间配置
	GlobalSettings         bool                   `json:"global_settings"`                    // 全局设置
	OverrideExisting       bool                   `json:"override_existing"`                  // 覆盖现有
}

// TolerationTimeConfig 容忍时间配置
type TolerationTimeConfig struct {
	DefaultTolerationTime  *int64                 `json:"default_toleration_time"`  // 默认容忍时间
	MaxTolerationTime      *int64                 `json:"max_toleration_time"`      // 最大容忍时间
	MinTolerationTime      *int64                 `json:"min_toleration_time"`      // 最小容忍时间
	TimeScalingPolicy      TimeScalingPolicy      `json:"time_scaling_policy"`      // 时间缩放策略
	ConditionalTimeouts    []ConditionalTimeout   `json:"conditional_timeouts"`     // 条件超时
	TimeZoneHandling       string                 `json:"timezone_handling"`        // 时区处理
}

// TimeScalingPolicy 时间缩放策略
type TimeScalingPolicy struct {
	PolicyType             string                 `json:"policy_type"`             // 策略类型 (fixed, linear, exponential)
	ScalingFactor          float64                `json:"scaling_factor"`          // 缩放因子
	BaseTime               *int64                 `json:"base_time"`               // 基础时间
	MaxScaledTime          *int64                 `json:"max_scaled_time"`         // 最大缩放时间
	ScalingConditions      []string               `json:"scaling_conditions"`      // 缩放条件
}

// ConditionalTimeout 条件超时
type ConditionalTimeout struct {
	Condition              string                 `json:"condition"`               // 条件
	TimeoutValue           *int64                 `json:"timeout_value"`           // 超时值
	Priority               int                    `json:"priority"`                // 优先级
	ApplyToEffects         []string               `json:"apply_to_effects"`        // 应用到效果
}

// K8sTaintEffectManagementResponse 污点效果管理响应
type K8sTaintEffectManagementResponse struct {
	NodeName               string                 `json:"node_name"`               // 节点名称
	AffectedPods           []PodEvictionInfo      `json:"affected_pods"`           // 受影响的Pod信息
	EffectChanges          []EffectChange         `json:"effect_changes"`          // 效果变化
	EvictionSummary        EvictionSummary        `json:"eviction_summary"`        // 驱逐摘要
	OperationTime          time.Time              `json:"operation_time"`          // 操作时间
	Status                 string                 `json:"status"`                  // 状态
	Warnings               []string               `json:"warnings"`                // 警告
}

// PodEvictionInfo Pod驱逐信息
type PodEvictionInfo struct {
	PodName                string                 `json:"pod_name"`                // Pod名称
	Namespace              string                 `json:"namespace"`               // 命名空间
	EvictionReason         string                 `json:"eviction_reason"`         // 驱逐原因
	EvictionTime           *time.Time             `json:"eviction_time"`           // 驱逐时间
	RescheduleAttempts     int                    `json:"reschedule_attempts"`     // 重调度尝试
	NewNodeName            string                 `json:"new_node_name"`           // 新节点名称
	Status                 string                 `json:"status"`                  // 状态
}

// EffectChange 效果变化
type EffectChange struct {
	TaintKey               string                 `json:"taint_key"`               // 污点键
	OldEffect              string                 `json:"old_effect"`              // 旧效果
	NewEffect              string                 `json:"new_effect"`              // 新效果
	ChangeReason           string                 `json:"change_reason"`           // 变化原因
	ChangeTime             time.Time              `json:"change_time"`             // 变化时间
}

// EvictionSummary 驱逐摘要
type EvictionSummary struct {
	TotalPods              int                    `json:"total_pods"`              // 总Pod数
	EvictedPods            int                    `json:"evicted_pods"`            // 已驱逐Pod数
	FailedEvictions        int                    `json:"failed_evictions"`        // 失败驱逐数
	PendingEvictions       int                    `json:"pending_evictions"`       // 待驱逐数
	RescheduledPods        int                    `json:"rescheduled_pods"`        // 重调度Pod数
	AverageEvictionTime    float64                `json:"average_eviction_time"`   // 平均驱逐时间
}

// K8sTolerationTimeResponse 容忍时间设置响应
type K8sTolerationTimeResponse struct {
	ResourceType           string                 `json:"resource_type"`           // 资源类型
	ResourceName           string                 `json:"resource_name"`           // 资源名称
	Namespace              string                 `json:"namespace"`               // 命名空间
	AppliedTimeouts        []AppliedTimeout       `json:"applied_timeouts"`        // 应用的超时
	ValidationResults      []TimeValidationResult `json:"validation_results"`      // 验证结果
	CreationTimestamp      time.Time              `json:"creation_timestamp"`      // 创建时间
	Status                 string                 `json:"status"`                  // 状态
}

// AppliedTimeout 应用的超时
type AppliedTimeout struct {
	TaintKey               string                 `json:"taint_key"`               // 污点键
	Effect                 string                 `json:"effect"`                  // 效果
	TimeoutValue           *int64                 `json:"timeout_value"`           // 超时值
	AppliedCondition       string                 `json:"applied_condition"`       // 应用条件
	Source                 string                 `json:"source"`                  // 来源
}

// TimeValidationResult 时间验证结果
type TimeValidationResult struct {
	TaintKey               string                 `json:"taint_key"`               // 污点键
	IsValid                bool                   `json:"is_valid"`                // 是否有效
	ValidationMessage      string                 `json:"validation_message"`      // 验证消息
	RecommendedTimeout     *int64                 `json:"recommended_timeout"`     // 推荐超时
	ValidationTime         time.Time              `json:"validation_time"`         // 验证时间
}
