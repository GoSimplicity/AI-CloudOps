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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ResourceRequirements 资源要求
type ResourceRequirements struct {
	Requests K8sResourceList `json:"requests,omitempty" gorm:"type:text;serializer:json;comment:资源请求"` // 资源请求
	Limits   K8sResourceList `json:"limits,omitempty" gorm:"type:text;serializer:json;comment:资源限制"`   // 资源限制
}

// K8sResourceList K8s资源列表
type K8sResourceList struct {
	CPU    string `json:"cpu,omitempty" gorm:"size:50;comment:CPU 数量，例如 '500m', '2'"`     // CPU 数量，例如 "500m", "2"
	Memory string `json:"memory,omitempty" gorm:"size:50;comment:内存数量，例如 '1Gi', '512Mi'"` // 内存数量，例如 "1Gi", "512Mi"
}

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

func (k *K8sPod) TableName() string {
	return "cl_k8s_pods"
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

// K8sPodReq 创建 Pod 的请求结构
type K8sPodReq struct {
	ClusterId int         `json:"cluster_id" binding:"required"` // 集群名称，必填
	Pod       *corev1.Pod `json:"pod"`                           // Pod 对象
}

// K8sDeploymentReq Deployment 相关请求结构
type K8sDeploymentReq struct {
	ClusterId       int                `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace       string             `json:"namespace" binding:"required"`  // 命名空间，必填
	DeploymentNames []string           `json:"deployment_names"`              // Deployment 名称，可选
	DeploymentYaml  *appsv1.Deployment `json:"deployment_yaml"`               // Deployment 对象, 可选
}

// K8sConfigMapReq ConfigMap 相关请求结构
type K8sConfigMapReq struct {
	ClusterId      int               `json:"cluster_id" binding:"required"` // 集群id，必填
	Namespace      string            `json:"namespace"`                     // 命名空间，可选, 删除用
	ConfigMapNames []string          `json:"config_map_names"`              // ConfigMap 名称，可选， 删除用
	ConfigMap      *corev1.ConfigMap `json:"config_map"`                    // ConfigMap 对象, 可选
}

// K8sServiceReq Service 相关请求结构
type K8sServiceReq struct {
	ClusterId    int             `json:"cluster_id" binding:"required"` // 集群id，必填
	Namespace    string          `json:"namespace"`                     // 命名空间，必填
	ServiceNames []string        `json:"service_names"`                 // Service 名称，可选
	ServiceYaml  *corev1.Service `json:"service_yaml"`                  // Service 对象, 可选
}

// PodListReq 获取Pod列表请求
type PodListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`
	Limit         int64  `json:"limit" form:"limit" comment:"限制结果数量"`
}

// PodGetReq 获取单个Pod请求
type PodGetReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	PodName   string `json:"pod_name" form:"pod_name" uri:"pod_name" binding:"required" comment:"Pod名称"`
}

// PodCreateReq 创建Pod请求
type PodCreateReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string            `json:"name" binding:"required" comment:"Pod名称"`
	YAML        string            `json:"yaml" binding:"required" comment:"YAML配置"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// PodUpdateReq 更新Pod请求
type PodUpdateReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string            `json:"name" binding:"required" comment:"Pod名称"`
	YAML        string            `json:"yaml" binding:"required" comment:"YAML配置"`
	Labels      map[string]string `json:"labels" comment:"标签"`
	Annotations map[string]string `json:"annotations" comment:"注解"`
}

// PodDeleteReq 删除Pod请求
type PodDeleteReq struct {
	ClusterID          int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace          string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`
	PodName            string `json:"pod_name" form:"pod_name" binding:"required" comment:"Pod名称"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间"`
	Force              bool   `json:"force" comment:"是否强制删除"`
}

// PodEntity Pod响应实体
type PodEntity struct {
	Name        string            `json:"name"`        // Pod名称
	Namespace   string            `json:"namespace"`   // 命名空间
	UID         string            `json:"uid"`         // Pod UID
	NodeName    string            `json:"node_name"`   // 所在节点
	Status      string            `json:"status"`      // Pod状态
	PodIP       string            `json:"pod_ip"`      // Pod IP
	HostIP      string            `json:"host_ip"`     // 宿主机IP
	QosClass    string            `json:"qos_class"`   // QoS等级
	Restarts    int32             `json:"restarts"`    // 重启次数
	Age         string            `json:"age"`         // 存在时间
	Ready       string            `json:"ready"`       // 就绪状态
	Labels      map[string]string `json:"labels"`      // 标签
	Annotations map[string]string `json:"annotations"` // 注解
	CreatedAt   string            `json:"created_at"`  // 创建时间
}

// PodListResponse Pod列表响应
type PodListResponse struct {
	Items      []PodEntity `json:"items"`       // Pod列表
	TotalCount int         `json:"total_count"` // 总数
}

// K8sPodListResponse Pod 列表响应 (保持向后兼容)
type K8sPodListResponse struct {
	Pods       []K8sPod `json:"pods"`        // Pod 列表
	TotalCount int      `json:"total_count"` // 总数
}
