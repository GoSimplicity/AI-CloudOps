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

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

// K8sCluster Kubernetes 集群的配置
type K8sCluster struct {
	Model
	Name                string     `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:集群名称"`      // 集群名称
	NameZh              string     `json:"name_zh" binding:"required,min=1,max=500" gorm:"uniqueIndex;size:100;comment:集群中文名称"` // 集群中文名称
	UserID              int        `json:"user_id" gorm:"comment:创建者用户ID"`                                                      // 创建者用户ID
	CpuRequest          string     `json:"cpu_request,omitempty" gorm:"comment:CPU 请求量"`                                        // CPU 请求量
	CpuLimit            string     `json:"cpu_limit,omitempty" gorm:"comment:CPU 限制量"`                                          // CPU 限制量
	MemoryRequest       string     `json:"memory_request,omitempty" gorm:"comment:内存请求量"`                                       // 内存请求量
	MemoryLimit         string     `json:"memory_limit,omitempty" gorm:"comment:内存限制量"`                                         // 内存限制量
	RestrictedNameSpace StringList `json:"restricted_name_space" gorm:"comment:资源限制命名空间"`                                       // 资源限制命名空间

	Env                  string `json:"env,omitempty" gorm:"comment:集群环境，例如 prod, stage, dev, rc, press"`     // 集群环境
	Version              string `json:"version,omitempty" gorm:"comment:集群版本"`                                // 集群版本
	ApiServerAddr        string `json:"api_server_addr,omitempty" gorm:"comment:API Server 地址"`               // API Server 地址
	KubeConfigContent    string `json:"kube_config_content,omitempty" gorm:"type:text;comment:kubeConfig 内容"` // kubeConfig 内容
	ActionTimeoutSeconds int    `json:"action_timeout_seconds,omitempty" gorm:"comment:操作超时时间（秒）"`            // 操作超时时间（秒）

	// 前端使用字段
	CreateUserName    string            `json:"create_username,omitempty" gorm:"-"`      // 创建者用户名
	LastProbeSuccess  bool              `json:"last_probe_success,omitempty" gorm:"-"`   // 最近一次探测是否成功
	LastProbeErrorMsg string            `json:"last_probe_error_msg,omitempty" gorm:"-"` // 最近一次探测错误信息
	LabelsFront       string            `json:"labels_front,omitempty" gorm:"-"`         // 前端显示的标签字符串
	AnnotationsFront  string            `json:"annotations_front,omitempty" gorm:"-"`    // 前端显示的注解字符串
	LabelsMap         map[string]string `json:"labels_map,omitempty" gorm:"-"`           // 标签键值对映射
	AnnotationsMap    map[string]string `json:"annotations_map,omitempty" gorm:"-"`      // 注解键值对映射
}

// K8sNode Kubernetes 节点
type K8sNode struct {
	Name              string               `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:节点名称"` // 节点名称
	ClusterID         int                  `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                                // 所属集群ID
	Status            string               `json:"status" gorm:"comment:节点状态，例如 Ready, NotReady, SchedulingDisabled"`              // 节点状态
	ScheduleEnable    bool                 `json:"schedule_enable" gorm:"comment:节点是否可调度"`                                         // 节点是否可调度
	Roles             []string             `json:"roles" gorm:"type:text;serializer:json;comment:节点角色，例如 master, worker"`          // 节点角色
	Age               string               `json:"age" gorm:"comment:节点存在时间，例如 5d"`                                                // 节点存在时间
	IP                string               `json:"ip" gorm:"comment:节点内部IP"`                                                       // 节点内部IP
	PodNum            int                  `json:"pod_num" gorm:"comment:节点上的 Pod 数量"`                                             // 节点上的 Pod 数量
	CpuRequestInfo    string               `json:"cpu_request_info" gorm:"comment:CPU 请求信息，例如 500m/2"`                             // CPU 请求信息
	CpuLimitInfo      string               `json:"cpu_limit_info" gorm:"comment:CPU 限制信息，例如 1/2"`                                  // CPU 限制信息
	CpuUsageInfo      string               `json:"cpu_usage_info" gorm:"comment:CPU 使用信息，例如 300m/2 (15%)"`                         // CPU 使用信息
	MemoryRequestInfo string               `json:"memory_request_info" gorm:"comment:内存请求信息，例如 1Gi/8Gi"`                           // 内存请求信息
	MemoryLimitInfo   string               `json:"memory_limit_info" gorm:"comment:内存限制信息，例如 2Gi/8Gi"`                             // 内存限制信息
	MemoryUsageInfo   string               `json:"memory_usage_info" gorm:"comment:内存使用信息，例如 1.5Gi/8Gi (18.75%)"`                  // 内存使用信息
	PodNumInfo        string               `json:"pod_num_info" gorm:"comment:Pod 数量信息，例如 10/50 (20%)"`                            // Pod 数量信息
	CpuCores          string               `json:"cpu_cores" gorm:"comment:CPU 核心信息，例如 2/4"`                                       // CPU 核心信息
	MemGibs           string               `json:"mem_gibs" gorm:"comment:内存信息，例如 8Gi/16Gi"`                                       // 内存信息
	EphemeralStorage  string               `json:"ephemeral_storage" gorm:"comment:临时存储信息，例如 100Gi/200Gi"`                         // 临时存储信息
	KubeletVersion    string               `json:"kubelet_version" gorm:"comment:Kubelet 版本"`                                      // Kubelet 版本
	CriVersion        string               `json:"cri_version" gorm:"comment:容器运行时接口版本"`                                           // 容器运行时接口版本
	OsVersion         string               `json:"os_version" gorm:"comment:操作系统版本"`                                               // 操作系统版本
	KernelVersion     string               `json:"kernel_version" gorm:"comment:内核版本"`                                             // 内核版本
	Labels            []string             `json:"labels" gorm:"type:text;serializer:json;comment:节点标签列表"`                         // 节点标签列表
	LabelsFront       string               `json:"labels_front" gorm:"-"`                                                          // 前端显示的标签字符串，格式为多行 key=value
	TaintsFront       string               `json:"taints_front" gorm:"-"`                                                          // 前端显示的 Taints 字符串，格式为多行 key=value:Effect
	LabelPairs        map[string]string    `json:"label_pairs" gorm:"-"`                                                           // 标签键值对映射
	Annotation        map[string]string    `json:"annotation" gorm:"type:text;serializer:json;comment:注解键值对映射"`                    // 注解键值对映射
	Conditions        []core.NodeCondition `json:"conditions" gorm:"-"`                                                            // 节点条件列表
	Taints            []core.Taint         `json:"taints" gorm:"-"`                                                                // 节点 Taints 列表
	Events            []OneEvent           `json:"events" gorm:"-"`                                                                // 节点相关事件列表，包含最近的事件信息
	CreatedAt         time.Time            `json:"created_at" gorm:"comment:创建时间"`                                                 // 创建时间
	UpdatedAt         time.Time            `json:"updated_at" gorm:"comment:更新时间"`                                                 // 更新时间
}

// K8sApp 面向运维的 Kubernetes 应用
type K8sApp struct {
	Model
	Name         string        `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:应用名称"` // 应用名称
	K8sProjectID int           `json:"k8s_project_id" gorm:"comment:关联的 Kubernetes 项目ID"`                              // 关联的 Kubernetes 项目ID
	TreeNodeID   int           `json:"tree_node_id" gorm:"comment:关联的树节点ID"`                                           // 关联的树节点ID
	UserID       int           `json:"user_id" gorm:"comment:创建者用户ID"`                                                 // 创建者用户ID
	Cluster      string        `json:"cluster" gorm:"uniqueIndex;size:100;comment:所属集群名称"`                             // 所属集群名称
	K8sInstances []K8sInstance `json:"k8s_instances" gorm:"foreignKey:K8sAppID;comment:关联的 Kubernetes 实例"`             // 关联的 Kubernetes 实例
	ServiceType  string        `json:"service_type,omitempty" gorm:"comment:服务类型"`                                     // 服务类型
	Namespace    string        `json:"namespace,omitempty" gorm:"comment:Kubernetes 命名空间"`                             // Kubernetes 命名空间

	ContainerCore `json:"containerCore"` // 容器核心配置

	// 前端使用字段
	TreeNodeObj    *TreeNode   `json:"tree_node_obj,omitempty" gorm:"-"`    // 树节点对象
	ClusterObj     *K8sCluster `json:"cluster_obj,omitempty" gorm:"-"`      // 集群对象
	ProjectObj     *K8sProject `json:"project_obj,omitempty" gorm:"-"`      // 项目对象
	CreateUserName string      `json:"create_username,omitempty" gorm:"-"`  // 创建者用户名
	NodePath       string      `json:"node_path,omitempty" gorm:"-"`        // 节点路径
	K8sProjectName string      `json:"k8s_project_name,omitempty" gorm:"-"` // 项目名称
}

// K8sCronjob Kubernetes 定时任务的配置
type K8sCronjob struct {
	Model
	Name         string     `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:定时任务名称"` // 定时任务名称
	Cluster      string     `json:"cluster,omitempty" gorm:"size:100;comment:所属集群"`                                   // 所属集群
	TreeNodeID   int        `json:"tree_node_id" gorm:"comment:关联的树节点ID"`                                             // 关联的树节点ID
	UserID       int        `json:"user_id" gorm:"comment:创建者用户ID"`                                                   // 创建者用户ID
	K8sProjectID int        `json:"k8s_project_id" gorm:"uniqueIndex;comment:关联的 Kubernetes 项目ID"`                    // 关联的 Kubernetes 项目ID
	Namespace    string     `json:"namespace,omitempty" gorm:"comment:命名空间"`                                          // 命名空间
	Schedule     string     `json:"schedule,omitempty" gorm:"comment:调度表达式"`                                          // 调度表达式
	Image        string     `json:"image,omitempty" gorm:"comment:镜像"`                                                // 镜像
	Commands     StringList `json:"commands,omitempty" gorm:"comment:启动命令组"`                                          // 启动命令组
	Args         StringList `json:"args,omitempty" gorm:"comment:启动参数，空格分隔"`                                          // 启动参数

	// 前端使用字段
	CommandsFront       []apiresponse.KeyValueItem `json:"commands_front,omitempty" gorm:"-"`         // 前端显示的命令
	ArgsFront           []apiresponse.KeyValueItem `json:"args_front,omitempty" gorm:"-"`             // 前端显示的参数
	LastScheduleTime    string                     `json:"last_schedule_time,omitempty" gorm:"-"`     // 最近一次调度时间
	LastSchedulePodName string                     `json:"last_schedule_pod_name,omitempty" gorm:"-"` // 最近一次调度的 Pod 名称
	CreateUserName      string                     `json:"create_username,omitempty" gorm:"-"`        // 创建者用户名
	NodePath            string                     `json:"node_path,omitempty" gorm:"-"`              // 节点路径
	Key                 string                     `json:"key" gorm:"-"`                              // 前端表格使用的Key
	TreeNodeObj         *TreeNode                  `json:"tree_node_obj,omitempty" gorm:"-"`          // 树节点对象
	ClusterObj          *K8sCluster                `json:"cluster_obj,omitempty" gorm:"-"`            // 集群对象
	ProjectObj          *K8sProject                `json:"project_obj,omitempty" gorm:"-"`            // 项目对象
	K8sProjectName      string                     `json:"k8s_project_name,omitempty" gorm:"-"`       // 项目名称
}

// K8sInstance Kubernetes 实例的配置
type K8sInstance struct {
	Model
	Name          string                 `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:实例名称"` // 实例名称
	UserID        int                    `json:"user_id" gorm:"comment:创建者用户ID"`                                                 // 创建者用户ID
	Cluster       string                 `json:"cluster,omitempty" gorm:"size:100;comment:所属集群"`                                 // 所属集群
	ContainerCore `json:"containerCore"` // 容器核心配置
	Image         string                 `json:"image,omitempty" gorm:"comment:镜像"`                        // 镜像
	Replicas      int                    `json:"replicas,omitempty" gorm:"comment:副本数量"`                   // 副本数量
	K8sAppID      int                    `json:"k8s_appId" gorm:"uniqueIndex;comment:关联的 Kubernetes 应用ID"` // 关联的 Kubernetes 应用ID

	// 前端使用字段
	K8sAppName     string      `json:"k8s_app_name,omitempty" gorm:"-"`    // 应用名称
	CreateUserName string      `json:"create_username,omitempty" gorm:"-"` // 创建者用户名
	NodePath       string      `json:"node_path,omitempty" gorm:"-"`       // 节点路径
	Key            string      `json:"key" gorm:"-"`                       // 前端表格使用的Key
	Namespace      string      `json:"namespace,omitempty" gorm:"-"`       // 命名空间
	K8sAppObj      *K8sApp     `json:"k8s_app_obj,omitempty" gorm:"-"`     // 应用对象
	ClusterObj     *K8sCluster `json:"cluster_obj,omitempty" gorm:"-"`     // 集群对象
	ReadyStatus    string      `json:"ready_status,omitempty" gorm:"-"`    // 就绪状态
}

// K8sProject Kubernetes 项目的配置
type K8sProject struct {
	Model
	Name       string   `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:项目名称"`      // 项目名称
	NameZh     string   `json:"name_zh" binding:"required,min=1,max=500" gorm:"uniqueIndex;size:100;comment:项目中文名称"` // 项目中文名称
	Cluster    string   `json:"cluster" gorm:"uniqueIndex;size:100;comment:所属集群名称"`                                  // 所属集群名称
	TreeNodeID int      `json:"tree_node_id" gorm:"comment:关联的树节点ID"`                                                // 关联的树节点ID
	UserID     int      `json:"user_id" gorm:"comment:创建者用户ID"`                                                      // 创建者用户ID
	K8sApps    []K8sApp `json:"k8s_apps,omitempty" gorm:"foreignKey:K8sProjectID;comment:关联的 Kubernetes 应用"`         // 关联的 Kubernetes 应用

	// 前端使用字段
	CreateUserName string    `json:"create_username,omitempty" gorm:"-"` // 创建者用户名
	NodePath       string    `json:"node_path,omitempty" gorm:"-"`       // 节点路径
	Key            string    `json:"key" gorm:"-"`                       // 前端表格使用的Key
	TreeNodeObj    *TreeNode `json:"tree_node_obj,omitempty" gorm:"-"`   // 树节点对象
}

// K8sYamlTask Kubernetes YAML 任务的配置
type K8sYamlTask struct {
	Model
	Name        string     `json:"name" gorm:"type:varchar(255);uniqueIndex;comment:YAML 任务名称"`     // YAML 任务名称
	UserID      int        `json:"user_id" gorm:"comment:创建者用户ID"`                                  // 创建者用户ID
	TemplateID  int        `json:"template_id" gorm:"comment:关联的模板ID"`                              // 关联的模板ID
	ClusterId   int        `json:"cluster_id,omitempty" gorm:"comment:集群名称"`                        // 集群名称
	Variables   StringList `json:"variables,omitempty" gorm:"type:text;comment:yaml 变量，格式 k=v,k=v"` // YAML 变量
	Status      string     `json:"status,omitempty" gorm:"comment:当前状态"`                            // 当前状态
	ApplyResult string     `json:"apply_result,omitempty" gorm:"comment:apply 后的返回数据"`              // apply 结果

	// 前端使用字段
	Key            string `json:"key" gorm:"-"`                       // 前端表格使用的Key
	VariablesFront string `json:"variables_front,omitempty" gorm:"-"` // 前端显示的变量
	YamlString     string `json:"yaml_string,omitempty" gorm:"-"`     // YAML 字符串
	TemplateName   string `json:"template_name,omitempty" gorm:"-"`   // 模板名称
	CreateUserName string `json:"create_username,omitempty" gorm:"-"` // 创建者用户名
}

// K8sYamlTemplate Kubernetes YAML 模板的配置
type K8sYamlTemplate struct {
	Model
	Name      string `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:模板名称"` // 模板名称
	UserID    int    `json:"user_id" gorm:"comment:创建者用户ID"`                                                // 创建者用户ID
	Content   string `json:"content,omitempty" gorm:"type:text;comment:yaml 模板内容"`                          // YAML 模板内容
	ClusterId int    `json:"cluster_id,omitempty" gorm:"comment:对应集群id"`
	// 前端使用字段
	Key            string `json:"key" gorm:"-"`                       // 前端表格使用的Key
	CreateUserName string `json:"create_username,omitempty" gorm:"-"` // 创建者用户名
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

// ContainerCore 包含容器的核心配置
type ContainerCore struct {
	Envs          StringList `json:"envs,omitempty" gorm:"comment:环境变量组，格式 key=value"`          // 环境变量组
	Labels        StringList `json:"labels,omitempty" gorm:"comment:标签组，格式 key=value"`          // 标签组
	Commands      StringList `json:"commands,omitempty" gorm:"comment:启动命令组"`                   // 启动命令组
	Args          StringList `json:"args,omitempty" gorm:"comment:启动参数，空格分隔"`                   // 启动参数
	CpuRequest    string     `json:"cpu_request,omitempty" gorm:"comment:CPU 请求量"`              // CPU 请求量
	CpuLimit      string     `json:"cpu_limit,omitempty" gorm:"comment:CPU 限制量"`                // CPU 限制量
	MemoryRequest string     `json:"memory_request,omitempty" gorm:"comment:内存请求量"`             // 内存请求量
	MemoryLimit   string     `json:"memory_limit,omitempty" gorm:"comment:内存限制量"`               // 内存限制量
	VolumeJson    string     `json:"volume_json,omitempty" gorm:"type:text;comment:卷和挂载配置JSON"` // 卷和挂载配置JSON
	PortJson      string     `json:"port_json,omitempty" gorm:"type:text;comment:容器和服务端口配置"`    // 容器和服务端口配置

	// 前端使用字段
	EnvsFront       []apiresponse.KeyValueItem `json:"envs_front,omitempty" gorm:"-"`        // 前端显示的环境变量
	LabelsFront     []apiresponse.KeyValueItem `json:"labels_front,omitempty" gorm:"-"`      // 前端显示的标签
	CommandsFront   []apiresponse.KeyValueItem `json:"commands_front,omitempty" gorm:"-"`    // 前端显示的命令
	ArgsFront       []apiresponse.KeyValueItem `json:"args_front,omitempty" gorm:"-"`        // 前端显示的参数
	VolumeJsonFront []K8sOneVolume             `json:"volume_json_front,omitempty" gorm:"-"` // 前端显示的卷配置
	PortJsonFront   []core.ServicePort         `json:"port_json_front,omitempty" gorm:"-"`   // 前端显示的端口配置
}

// K8sOneVolume 单个卷的配置
type K8sOneVolume struct {
	Type         string `json:"type" gorm:"comment:卷类型，如 hostPath, configMap, emptyDir, pvc"`               // 卷类型
	Name         string `json:"name" gorm:"size:100;comment:卷名称"`                                           // 卷名称
	MountPath    string `json:"mount_path" gorm:"size:255;comment:挂载路径"`                                    // 挂载路径
	SubPath      string `json:"sub_path,omitempty" gorm:"size:255;comment:子路径"`                             // 子路径（可选）
	PvcName      string `json:"pvc_name,omitempty" gorm:"size:100;comment:PVC名称，当类型为 pvc 时使用"`              // PVC名称（可选）
	CmName       string `json:"cm_name,omitempty" gorm:"size:100;comment:ConfigMap名称，当类型为 configMap 时使用"`   // ConfigMap名称（可选）
	HostPath     string `json:"host_path,omitempty" gorm:"size:255;comment:Host路径，当类型为 hostPath 时使用"`       // Host路径（可选）
	HostPathType string `json:"host_path_type,omitempty" gorm:"size:50;comment:Host路径类型，当类型为 hostPath 时使用"` // Host路径类型（可选）
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

// K8sClusterNodesRequest 定义集群节点请求的基础结构
type K8sClusterNodesRequest struct {
	ClusterId int    `json:"cluster_id" binding:"required"` // 集群id，必填
	NodeName  string `json:"node_name" binding:"required"`  // 节点名称列表，必填
}

// ResourceRequirements 资源的请求与限制
type ResourceRequirements struct {
	Requests K8sResourceList `json:"requests,omitempty" gorm:"type:text;serializer:json;comment:资源请求"` // 资源请求
	Limits   K8sResourceList `json:"limits,omitempty" gorm:"type:text;serializer:json;comment:资源限制"`   // 资源限制
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

// K8sResourceList 资源的具体数量
type K8sResourceList struct {
	CPU    string `json:"cpu,omitempty" gorm:"size:50;comment:CPU 数量，例如 '500m', '2'"`     // CPU 数量，例如 "500m", "2"
	Memory string `json:"memory,omitempty" gorm:"size:50;comment:内存数量，例如 '1Gi', '512Mi'"` // 内存数量，例如 "1Gi", "512Mi"
}

// LabelK8sNodesRequest 定义为节点添加标签的请求结构
type LabelK8sNodesRequest struct {
	*K8sClusterNodesRequest
	ModType string   `json:"mod_type" binding:"required,oneof=add del"` // 操作类型，必填，值为 "add" 或 "del"
	Labels  []string `json:"labels" binding:"required"`                 // 标签键值对，必填
}

// TaintK8sNodesRequest 定义为节点添加或删除 Taint 的请求结构
type TaintK8sNodesRequest struct {
	*K8sClusterNodesRequest
	ModType   string `json:"mod_type"`             // 操作类型，值为 "add" 或 "del"
	TaintYaml string `json:"taint_yaml,omitempty"` // 可选的 Taint YAML 字符串，用于验证或其他用途
}

// OneEvent 单个事件的模型
type OneEvent struct {
	Type      string `json:"type"`       // 事件类型，例如 "Normal", "Warning"
	Component string `json:"component"`  // 事件的组件来源，例如 "kubelet"
	Reason    string `json:"reason"`     // 事件的原因，例如 "NodeReady"
	Message   string `json:"message"`    // 事件的详细消息
	FirstTime string `json:"first_time"` // 事件第一次发生的时间，例如 "2024-04-27T10:00:00Z"
	LastTime  string `json:"last_time"`  // 事件最近一次发生的时间，例如 "2024-04-27T12:00:00Z"
	Object    string `json:"object"`     // 事件关联的对象信息，例如 "kind:Node name:node-1"
	Count     int    `json:"count"`      // 事件发生的次数
}

// Taint 定义 Taint 的模型
type Taint struct {
	Key    string `json:"key" binding:"required"`                                                // Taint 的键
	Value  string `json:"value,omitempty"`                                                       // Taint 的值
	Effect string `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"` // Taint 的效果，例如 "NoSchedule", "PreferNoSchedule", "NoExecute"
}

// ScheduleK8sNodesRequest 定义调度节点的请求结构
type ScheduleK8sNodesRequest struct {
	*K8sClusterNodesRequest
	ScheduleEnable bool `json:"schedule_enable"`
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

type BatchDeleteReq struct {
	IDs []int `json:"ids" binding:"required"`
}

// CreateNamespaceRequest 创建新的命名空间请求结构体
type CreateNamespaceRequest struct {
	ClusterId   int      `json:"cluster_id" binding:"required"`
	Name        string   `json:"namespace" binding:"required"`
	Labels      []string `json:"labels,omitempty"`      // 命名空间标签
	Annotations []string `json:"annotations,omitempty"` // 命名空间注解
}

// UpdateNamespaceRequest 更新命名空间请求结构体
type UpdateNamespaceRequest struct {
	ClusterId   int      `json:"cluster_id" binding:"required"`
	Name        string   `json:"namespace" binding:"required"`
	Labels      []string `json:"labels,omitempty"`      // 命名空间标签
	Annotations []string `json:"annotations,omitempty"` // 命名空间注解
}

// Resource 命名空间中的资源响应结构体
type Resource struct {
	Type         string    `json:"type"`          // 资源类型，例如 Pod, Service, Deployment
	Name         string    `json:"name"`          // 资源名称
	Namespace    string    `json:"namespace"`     // 所属命名空间
	Status       string    `json:"status"`        // 资源状态，例如 Running, Pending
	CreationTime time.Time `json:"creation_time"` // 创建时间
}

// Event 命名空间事件响应结构体
type Event struct {
	Reason         string           `json:"reason"`          // 事件原因
	Message        string           `json:"message"`         // 事件消息
	Type           string           `json:"type"`            // 事件类型，例如 Normal, Warning
	FirstTimestamp time.Time        `json:"first_timestamp"` // 第一次发生时间
	LastTimestamp  time.Time        `json:"last_timestamp"`  // 最后一次发生时间
	Count          int32            `json:"count"`           // 事件发生次数
	Source         core.EventSource `json:"source"`          // 事件来源
}

// Namespace 命名空间响应结构体
type Namespace struct {
	Name         string    `json:"name"`                  // 命名空间名称
	UID          string    `json:"uid"`                   // 命名空间唯一标识符
	Status       string    `json:"status"`                // 命名空间状态，例如 Active
	CreationTime time.Time `json:"creation_time"`         // 创建时间
	Labels       []string  `json:"labels,omitempty"`      // 命名空间标签
	Annotations  []string  `json:"annotations,omitempty"` // 命名空间注解
}

// ClusterNamespaces 表示一个集群及其命名空间列表
type ClusterNamespaces struct {
	ClusterName string      `json:"cluster_name"` // 集群名称
	ClusterId   int         `json:"cluster_id"`   // 集群ID
	Namespaces  []Namespace `json:"namespaces"`   // 命名空间列表
}
