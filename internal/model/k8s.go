package model

import (
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	core "k8s.io/api/core/v1"
)

// K8sApp 表示面向运维的 Kubernetes 应用
type K8sApp struct {
	Model
	Name          string                 `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex:name_cluster;size:100;comment:应用名称"` // 应用名称
	K8sProjectID  uint                   `json:"k8sProjectId" gorm:"comment:关联的 Kubernetes 项目ID"`                                             // 关联的 Kubernetes 项目ID
	TreeNodeID    uint                   `json:"treeNodeId" gorm:"comment:关联的树节点ID"`                                                          // 关联的树节点ID
	UserID        uint                   `json:"userId" gorm:"comment:创建者用户ID"`                                                               // 创建者用户ID
	Cluster       string                 `json:"cluster" gorm:"uniqueIndex:name_cluster;size:100;comment:所属集群名称"`                             // 所属集群名称
	K8sInstances  []K8sInstance          `json:"k8sInstances" gorm:"foreignKey:K8sAppID;comment:关联的 Kubernetes 实例"`                           // 关联的 Kubernetes 实例
	ServiceType   string                 `json:"serviceType,omitempty" gorm:"comment:服务类型"`                                                   // 服务类型
	Namespace     string                 `json:"namespace,omitempty" gorm:"comment:Kubernetes 命名空间"`                                          // Kubernetes 命名空间
	ContainerCore `json:"containerCore"` // 容器核心配置

	// 前端使用字段
	TreeNodeObj    *TreeNode   `json:"treeNodeObj,omitempty" gorm:"-"`    // 树节点对象，不存储在数据库中
	ClusterObj     *K8sCluster `json:"clusterObj,omitempty" gorm:"-"`     // 集群对象，不存储在数据库中
	ProjectObj     *K8sProject `json:"projectObj,omitempty" gorm:"-"`     // 项目对象，不存储在数据库中
	CreateUserName string      `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，不存储在数据库中
	NodePath       string      `json:"nodePath,omitempty" gorm:"-"`       // 节点路径，不存储在数据库中
	K8sProjectName string      `json:"k8sProjectName,omitempty" gorm:"-"` // 项目名称，不存储在数据库中
	Key            string      `json:"key" gorm:"-"`                      // 前端表格使用的Key，不存储在数据库中
}

// ContainerCore 包含容器的核心配置
type ContainerCore struct {
	Envs          StringList `json:"envs,omitempty" gorm:"comment:环境变量组，格式 key=value"`         // 环境变量组
	Labels        StringList `json:"labels,omitempty" gorm:"comment:标签组，格式 key=value"`         // 标签组
	Commands      StringList `json:"commands,omitempty" gorm:"comment:启动命令组"`                  // 启动命令组
	Args          StringList `json:"args,omitempty" gorm:"comment:启动参数，空格分隔"`                  // 启动参数
	CpuRequest    string     `json:"cpuRequest,omitempty" gorm:"comment:CPU 请求量"`              // CPU 请求量
	CpuLimit      string     `json:"cpuLimit,omitempty" gorm:"comment:CPU 限制量"`                // CPU 限制量
	MemoryRequest string     `json:"memoryRequest,omitempty" gorm:"comment:内存请求量"`             // 内存请求量
	MemoryLimit   string     `json:"memoryLimit,omitempty" gorm:"comment:内存限制量"`               // 内存限制量
	VolumeJson    string     `json:"volumeJson,omitempty" gorm:"type:text;comment:卷和挂载配置JSON"` // 卷和挂载配置JSON
	PortJson      string     `json:"portJson,omitempty" gorm:"type:text;comment:容器和服务端口配置"`    // 容器和服务端口配置

	// 前端使用字段
	EnvsFront       []apiresponse.KeyValueItem `json:"envsFront,omitempty" gorm:"-"`       // 前端显示的环境变量，不存储在数据库中
	LabelsFront     []apiresponse.KeyValueItem `json:"labelsFront,omitempty" gorm:"-"`     // 前端显示的标签，不存储在数据库中
	CommandsFront   []apiresponse.KeyValueItem `json:"commandsFront,omitempty" gorm:"-"`   // 前端显示的命令，不存储在数据库中
	ArgsFront       []apiresponse.KeyValueItem `json:"argsFront,omitempty" gorm:"-"`       // 前端显示的参数，不存储在数据库中
	VolumeJsonFront []OneVolume                `json:"volumeJsonFront,omitempty" gorm:"-"` // 前端显示的卷配置，不存储在数据库中
	PortJsonFront   []core.ServicePort         `json:"portJsonFront,omitempty" gorm:"-"`   // 前端显示的端口配置，不存储在数据库中
}

// OneVolume 表示单个卷的配置
type OneVolume struct {
	Type         string `json:"type" gorm:"comment:卷类型，如 hostPath, configMap, emptyDir, pvc"`             // 卷类型
	Name         string `json:"name" gorm:"size:100;comment:卷名称"`                                         // 卷名称
	MountPath    string `json:"mountPath" gorm:"size:255;comment:挂载路径"`                                   // 挂载路径
	SubPath      string `json:"subPath,omitempty" gorm:"size:255;comment:子路径"`                            // 子路径（可选）
	PvcName      string `json:"pvcName,omitempty" gorm:"size:100;comment:PVC名称，当类型为 pvc 时使用"`             // PVC名称（可选）
	CmName       string `json:"cmName,omitempty" gorm:"size:100;comment:ConfigMap名称，当类型为 configMap 时使用"`  // ConfigMap名称（可选）
	HostPathPath string `json:"hostPathPath,omitempty" gorm:"size:255;comment:Host路径，当类型为 hostPath 时使用"`  // Host路径（可选）
	HostPathType string `json:"hostPathType,omitempty" gorm:"size:50;comment:Host路径类型，当类型为 hostPath 时使用"` // Host路径类型（可选）
}

// K8sCluster 表示 Kubernetes 集群的配置
type K8sCluster struct {
	Model
	Name                 string `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex;size:100;comment:集群名称"`     // 集群名称
	NameZh               string `json:"nameZh" binding:"required,min=1,max=500" gorm:"uniqueIndex;size:100;comment:集群中文名称"` // 集群中文名称
	UserID               uint   `json:"userId" gorm:"comment:创建者用户ID"`                                                      // 创建者用户ID
	CpuRequest           string `json:"cpuRequest,omitempty" gorm:"comment:CPU 请求量"`                                        // CPU 请求量
	CpuLimit             string `json:"cpuLimit,omitempty" gorm:"comment:CPU 限制量"`                                          // CPU 限制量
	MemoryRequest        string `json:"memoryRequest,omitempty" gorm:"comment:内存请求量"`                                       // 内存请求量
	MemoryLimit          string `json:"memoryLimit,omitempty" gorm:"comment:内存限制量"`                                         // 内存限制量
	Env                  string `json:"env,omitempty" gorm:"comment:集群环境，例如 prod, stage, dev, rc, press"`                   // 集群环境
	Version              string `json:"version,omitempty" gorm:"comment:集群版本"`                                              // 集群版本
	ApiServerAddr        string `json:"apiServerAddr,omitempty" gorm:"comment:API Server 地址"`                               // API Server 地址
	KubeConfigContent    string `json:"kubeConfigContent,omitempty" gorm:"type:text;comment:kubeConfig 内容"`                 // kubeConfig 内容
	ActionTimeoutSeconds int    `json:"actionTimeoutSeconds,omitempty" gorm:"comment:操作超时时间（秒）"`                            // 操作超时时间（秒）

	// 前端使用字段
	Key               string            `json:"key" gorm:"-"`                         // 前端表格使用的Key，不存储在数据库中
	CreateUserName    string            `json:"createUserName,omitempty" gorm:"-"`    // 创建者用户名，不存储在数据库中
	LastProbeSuccess  bool              `json:"lastProbeSuccess,omitempty" gorm:"-"`  // 最近一次探测是否成功，不存储在数据库中
	LastProbeErrorMsg string            `json:"lastProbeErrorMsg,omitempty" gorm:"-"` // 最近一次探测错误信息，不存储在数据库中
	LabelsFront       string            `json:"labelsFront,omitempty" gorm:"-"`       // 前端显示的标签字符串，不存储在数据库中
	AnnotationsFront  string            `json:"annotationsFront,omitempty" gorm:"-"`  // 前端显示的注解字符串，不存储在数据库中
	LabelsM           map[string]string `json:"labelsM,omitempty" gorm:"-"`           // 标签键值对映射，不存储在数据库中
	AnnotationsM      map[string]string `json:"annotationsM,omitempty" gorm:"-"`      // 注解键值对映射，不存储在数据库中
}

// K8sCronjob 表示 Kubernetes 定时任务的配置
type K8sCronjob struct {
	Model
	Name         string     `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex:name_k8s_project_id;size:100;comment:定时任务名称"` // 定时任务名称
	Cluster      string     `json:"cluster,omitempty" gorm:"size:100;comment:所属集群"`                                                       // 所属集群
	TreeNodeID   uint       `json:"treeNodeId" gorm:"comment:关联的树节点ID"`                                                                   // 关联的树节点ID
	UserID       uint       `json:"userId" gorm:"comment:创建者用户ID"`                                                                        // 创建者用户ID
	K8sProjectID uint       `json:"k8sProjectId" gorm:"uniqueIndex:name_k8s_project_id;comment:关联的 Kubernetes 项目ID"`                      // 关联的 Kubernetes 项目ID
	Namespace    string     `json:"namespace,omitempty" gorm:"comment:命名空间"`                                                              // 命名空间
	Schedule     string     `json:"schedule,omitempty" gorm:"comment:调度表达式"`                                                              // 调度表达式
	Image        string     `json:"image,omitempty" gorm:"comment:镜像"`                                                                    // 镜像
	Commands     StringList `json:"commands,omitempty" gorm:"comment:启动命令组"`                                                              // 启动命令组
	Args         StringList `json:"args,omitempty" gorm:"comment:启动参数，空格分隔"`                                                              // 启动参数

	// 前端使用字段
	CommandsFront       []apiresponse.KeyValueItem `json:"commandsFront,omitempty" gorm:"-"`       // 前端显示的命令，不存储在数据库中
	ArgsFront           []apiresponse.KeyValueItem `json:"argsFront,omitempty" gorm:"-"`           // 前端显示的参数，不存储在数据库中
	LastScheduleTime    string                     `json:"lastScheduleTime,omitempty" gorm:"-"`    // 最近一次调度时间，不存储在数据库中
	LastSchedulePodName string                     `json:"lastSchedulePodName,omitempty" gorm:"-"` // 最近一次调度的 Pod 名称，不存储在数据库中
	CreateUserName      string                     `json:"createUserName,omitempty" gorm:"-"`      // 创建者用户名，不存储在数据库中
	NodePath            string                     `json:"nodePath,omitempty" gorm:"-"`            // 节点路径，不存储在数据库中
	Key                 string                     `json:"key" gorm:"-"`                           // 前端表格使用的Key，不存储在数据库中
	TreeNodeObj         *TreeNode                  `json:"treeNodeObj,omitempty" gorm:"-"`         // 树节点对象，不存储在数据库中
	ClusterObj          *K8sCluster                `json:"clusterObj,omitempty" gorm:"-"`          // 集群对象，不存储在数据库中
	ProjectObj          *K8sProject                `json:"projectObj,omitempty" gorm:"-"`          // 项目对象，不存储在数据库中
	K8sProjectName      string                     `json:"k8sProjectName,omitempty" gorm:"-"`      // 项目名称，不存储在数据库中
}

// K8sInstance 表示 Kubernetes 实例的配置
type K8sInstance struct {
	Model
	Name          string                 `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex:name_k8s_app_id;size:100;comment:实例名称"` // 实例名称
	UserID        uint                   `json:"userId" gorm:"comment:创建者用户ID"`                                                                  // 创建者用户ID
	Cluster       string                 `json:"cluster,omitempty" gorm:"size:100;comment:所属集群"`                                                 // 所属集群
	ContainerCore `json:"containerCore"` // 容器核心配置
	Image         string                 `json:"image,omitempty" gorm:"comment:镜像"`                                       // 镜像
	Replicas      int                    `json:"replicas,omitempty" gorm:"comment:副本数量"`                                  // 副本数量
	K8sAppID      uint                   `json:"k8sAppId" gorm:"uniqueIndex:name_k8s_app_id;comment:关联的 Kubernetes 应用ID"` // 关联的 Kubernetes 应用ID

	// 前端使用字段
	K8sAppName     string      `json:"k8sAppName,omitempty" gorm:"-"`     // 应用名称，不存储在数据库中
	CreateUserName string      `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，不存储在数据库中
	NodePath       string      `json:"nodePath,omitempty" gorm:"-"`       // 节点路径，不存储在数据库中
	Key            string      `json:"key" gorm:"-"`                      // 前端表格使用的Key，不存储在数据库中
	Namespace      string      `json:"namespace,omitempty" gorm:"-"`      // 命名空间，不存储在数据库中
	K8sAppObj      *K8sApp     `json:"k8sAppObj,omitempty" gorm:"-"`      // 应用对象，不存储在数据库中
	ClusterObj     *K8sCluster `json:"clusterObj,omitempty" gorm:"-"`     // 集群对象，不存储在数据库中
	ReadyStatus    string      `json:"readyStatus,omitempty" gorm:"-"`    // 就绪状态，不存储在数据库中
}

// K8sProject 表示 Kubernetes 项目的配置
type K8sProject struct {
	Model
	Name       string   `json:"name" binding:"required,min=1,max=200" gorm:"uniqueIndex:name_cluster;size:100;comment:项目名称"` // 项目名称
	NameZh     string   `json:"nameZh" binding:"required,min=1,max=500" gorm:"uniqueIndex;size:100;comment:项目中文名称"`          // 项目中文名称
	Cluster    string   `json:"cluster" gorm:"uniqueIndex:name_cluster;size:100;comment:所属集群名称"`                             // 所属集群名称
	TreeNodeID uint     `json:"treeNodeId" gorm:"comment:关联的树节点ID"`                                                          // 关联的树节点ID
	UserID     uint     `json:"userId" gorm:"comment:创建者用户ID"`                                                               // 创建者用户ID
	K8sApps    []K8sApp `json:"k8sApps,omitempty" gorm:"foreignKey:K8sProjectID;comment:关联的 Kubernetes 应用"`                  // 关联的 Kubernetes 应用

	// 前端使用字段
	CreateUserName string    `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，不存储在数据库中
	NodePath       string    `json:"nodePath,omitempty" gorm:"-"`       // 节点路径，不存储在数据库中
	Key            string    `json:"key" gorm:"-"`                      // 前端表格使用的Key，不存储在数据库中
	TreeNodeObj    *TreeNode `json:"treeNodeObj,omitempty" gorm:"-"`    // 树节点对象，不存储在数据库中
}

// K8sYamlTask 表示 Kubernetes YAML 任务的配置
type K8sYamlTask struct {
	Model
	Name        string     `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:YAML 任务名称"` // YAML 任务名称
	UserID      uint       `json:"userId" gorm:"comment:创建者用户ID"`                                                      // 创建者用户ID
	TemplateID  uint       `json:"templateId" gorm:"comment:关联的模板ID"`                                                  // 关联的模板ID
	ClusterName string     `json:"clusterName,omitempty" gorm:"comment:集群名称"`                                          // 集群名称
	Variables   StringList `json:"variables,omitempty" gorm:"type:text;comment:yaml 变量，格式 k=v,k=v"`                    // YAML 变量
	Status      string     `json:"status,omitempty" gorm:"comment:当前状态"`                                               // 当前状态
	ApplyResult string     `json:"applyResult,omitempty" gorm:"comment:apply 后的返回数据"`                                  // apply 结果

	// 前端使用字段
	Key            string `json:"key" gorm:"-"`                      // 前端表格使用的Key，不存储在数据库中
	VariablesFront string `json:"variablesFront,omitempty" gorm:"-"` // 前端显示的变量，不存储在数据库中
	YamlString     string `json:"yamlString,omitempty" gorm:"-"`     // YAML 字符串，不存储在数据库中
	TemplateName   string `json:"templateName,omitempty" gorm:"-"`   // 模板名称，不存储在数据库中
	CreateUserName string `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，不存储在数据库中
}

// K8sYamlTemplate 表示 Kubernetes YAML 模板的配置
type K8sYamlTemplate struct {
	Model
	Name    string `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:模板名称"` // 模板名称
	UserID  uint   `json:"userId" gorm:"comment:创建者用户ID"`                                                 // 创建者用户ID
	Content string `json:"content,omitempty" gorm:"type:text;comment:yaml 模板内容"`                          // YAML 模板内容

	// 前端使用字段
	Key            string `json:"key" gorm:"-"`                      // 前端表格使用的Key，不存储在数据库中
	CreateUserName string `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，不存储在数据库中
}
