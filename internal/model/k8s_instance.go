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

// K8sInstance 扩展模型
type K8sInstance struct {
	Model
	Name          string `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:实例名称"` // 实例名称
	UserID        int    `json:"user_id" gorm:"comment:创建者用户ID"`                                     // 创建者用户ID
	ClusterID     int    `json:"cluster_id" gorm:"comment:所属集群ID"`                                   // 所属集群ID
	ContainerCore `json:"containerCore" gorm:"embedded"`
	Image         string `json:"image,omitempty" gorm:"comment:镜像"`                        // 镜像
	Replicas      int    `json:"replicas,omitempty" gorm:"default:1;comment:副本数量"`         // 副本数量
	K8sAppID      int    `json:"k8s_app_id" gorm:"index;comment:关联的 Kubernetes 应用ID"`      // 关联的 Kubernetes 应用ID，修正字段名称为k8s_app_id
	Namespace     string `json:"namespace,omitempty" gorm:"index;comment:Kubernetes 命名空间"` // 命名空间

	Type              string            `json:"type,omitempty" gorm:"default:Deployment;comment:实例类型"`        // 实例类型(Deployment/StatefulSet/DaemonSet)
	Status            string            `json:"status,omitempty" gorm:"-"`                                    // 运行状态(运行时查询，不存储)
	AvailableReplicas int               `json:"available_replicas,omitempty" gorm:"-"`                        // 可用副本数(运行时查询)
	Strategy          string            `json:"strategy,omitempty" gorm:"default:RollingUpdate;comment:部署策略"` // 部署策略(RollingUpdate/Recreate)
	ServiceName       string            `json:"service_name,omitempty" gorm:"comment:关联服务名称"`                 // 关联的服务名称
	Labels            map[string]string `json:"labels,omitempty" gorm:"serializer:json;comment:标签"`           // K8s标签
	Annotations       map[string]string `json:"annotations,omitempty" gorm:"serializer:json;comment:注解"`      // K8s注解

	// 健康检查
	LivenessProbe  *Probe `json:"liveness_probe,omitempty" gorm:"serializer:json;comment:存活探针"`  // 存活探针
	ReadinessProbe *Probe `json:"readiness_probe,omitempty" gorm:"serializer:json;comment:就绪探针"` // 就绪探针
	StartupProbe   *Probe `json:"startup_probe,omitempty" gorm:"serializer:json;comment:启动探针"`   // 启动探针

	// 高级配置
	NodeSelector map[string]string `json:"node_selector,omitempty" gorm:"serializer:json;comment:节点选择器"` // 节点选择器
	Affinity     *Affinity         `json:"affinity,omitempty" gorm:"serializer:json;comment:亲和性配置"`      // 亲和性配置
	Tolerations  []Toleration      `json:"tolerations,omitempty" gorm:"serializer:json;comment:容忍配置"`    // 容忍配置

	// 关联信息
	CreatedByUser string `json:"created_by_user,omitempty" gorm:"-"` // 创建者用户名(关联查询)
	ClusterName   string `json:"cluster_name,omitempty" gorm:"-"`    // 集群名称(关联查询)
	AppName       string `json:"app_name,omitempty" gorm:"-"`        // 应用名称(关联查询)
}

// Probe 探针配置
type Probe struct {
	Type                string   `json:"type"`                            // 探针类型(http/tcp/exec)
	Path                string   `json:"path,omitempty"`                  // HTTP路径
	Port                int      `json:"port,omitempty"`                  // 端口
	Command             []string `json:"command,omitempty"`               // 执行命令
	InitialDelaySeconds int      `json:"initial_delay_seconds,omitempty"` // 初始延迟秒数
	TimeoutSeconds      int      `json:"timeout_seconds,omitempty"`       // 超时秒数
	PeriodSeconds       int      `json:"period_seconds,omitempty"`        // 检测周期
	SuccessThreshold    int      `json:"success_threshold,omitempty"`     // 成功阈值
	FailureThreshold    int      `json:"failure_threshold,omitempty"`     // 失败阈值
}

// Volume 卷配置
type Volume struct {
	Name       string `json:"name"`                  // 卷名称
	Type       string `json:"type"`                  // 卷类型(ConfigMap, Secret, PVC, EmptyDir等)
	MountPath  string `json:"mount_path"`            // 挂载路径
	SubPath    string `json:"sub_path,omitempty"`    // 子路径
	ReadOnly   bool   `json:"read_only,omitempty"`   // 是否只读
	SourceName string `json:"source_name,omitempty"` // 源资源名称(如ConfigMap名称)
	Size       string `json:"size,omitempty"`       // 存储大小
}

// Affinity 亲和性配置
type Affinity struct {
	NodeAffinity    []AffinityRule `json:"node_affinity,omitempty"`     // 节点亲和性
	PodAffinity     []AffinityRule `json:"pod_affinity,omitempty"`      // Pod亲和性
	PodAntiAffinity []AffinityRule `json:"pod_anti_affinity,omitempty"` // Pod反亲和性
}

// AffinityRule 亲和性规则
type AffinityRule struct {
	Key      string   `json:"key"`              // 标签键
	Operator string   `json:"operator"`         // 操作符(In, NotIn, Exists等)
	Values   []string `json:"values,omitempty"` // 标签值列表
	Weight   int      `json:"weight,omitempty"` // 权重(1-100)
}

// Toleration 容忍配置
type Toleration struct {
	Key      string `json:"key,omitempty"`      // 键
	Operator string `json:"operator,omitempty"` // 操作符(Equal, Exists)
	Value    string `json:"value,omitempty"`    // 值
	Effect   string `json:"effect,omitempty"`   // 影响(NoSchedule, PreferNoSchedule, NoExecute)
}

// CreateK8sInstanceReq 创建K8s实例请求
type CreateK8sInstanceReq struct {
	Name           string            `json:"name" binding:"required,min=1,max=200"` // 实例名称
	UserID         int               `json:"user_id" binding:"required"`            // 创建者用户ID
	ClusterID      int               `json:"cluster_id" binding:"required"`         // 所属集群ID
	ContainerCore  ContainerCore     `json:"containerCore" binding:"required"`      // 容器配置
	Image          string            `json:"image" binding:"required"`              // 镜像
	Replicas       int               `json:"replicas,omitempty"`                    // 副本数量
	K8sAppID       int               `json:"k8s_app_id" binding:"required"`         // 关联的 Kubernetes 应用ID，修正字段名称
	Namespace      string            `json:"namespace,omitempty"`                   // 命名空间
	Type           string            `json:"type,omitempty"`                        // 实例类型(Deployment/StatefulSet/DaemonSet)
	Strategy       string            `json:"strategy,omitempty"`                    // 部署策略
	ServiceName    string            `json:"service_name,omitempty"`                // 关联的服务名称
	Labels         map[string]string `json:"labels,omitempty"`                      // K8s标签
	Annotations    map[string]string `json:"annotations,omitempty"`                 // K8s注解
	LivenessProbe  *Probe            `json:"liveness_probe,omitempty"`              // 存活探针
	ReadinessProbe *Probe            `json:"readiness_probe,omitempty"`             // 就绪探针
	StartupProbe   *Probe            `json:"startup_probe,omitempty"`               // 启动探针
	NodeSelector   map[string]string `json:"node_selector,omitempty"`               // 节点选择器
	Affinity       *Affinity         `json:"affinity,omitempty"`                    // 亲和性配置
	Tolerations    []Toleration      `json:"tolerations,omitempty"`                 // 容忍配置
	Volumes        []Volume          `json:"volumes,omitempty"`                     // 卷配置，添加缺失的卷配置字段
}



// CreateK8sInstanceResp 创建K8s实例响应
type CreateK8sInstanceResp struct {
}

// UpdateK8sInstanceReq 更新K8s实例请求
type UpdateK8sInstanceReq struct {
	ID            int           `json:"id" binding:"required"`                 // 实例ID
	Name          string        `json:"name" binding:"required,min=1,max=200"` // 实例名称
	UserID        int           `json:"user_id" binding:"required"`            // 创建者用户ID
	ClusterID     int           `json:"cluster_id" binding:"required"`         // 所属集群ID
	ContainerCore ContainerCore `json:"containerCore" binding:"required"`      // 容器配置
	Image         string        `json:"image" binding:"required"`              // 镜像
	Replicas      int           `json:"replicas,omitempty"`                    // 副本数量
	K8sAppID      int           `json:"k8s_appId" binding:"required"`          // 关联的 Kubernetes 应用ID
	Namespace     string        `json:"namespace,omitempty"`                   // 命名空间

	Type           string            `json:"type,omitempty"`            // 实例类型(Deployment/StatefulSet/DaemonSet)
	Strategy       string            `json:"strategy,omitempty"`        // 部署策略
	ServiceName    string            `json:"service_name,omitempty"`    // 关联的服务名称
	Labels         map[string]string `json:"labels,omitempty"`          // K8s标签
	Annotations    map[string]string `json:"annotations,omitempty"`     // K8s注解
	LivenessProbe  *Probe            `json:"liveness_probe,omitempty"`  // 存活探针
	ReadinessProbe *Probe            `json:"readiness_probe,omitempty"` // 就绪探针
	StartupProbe   *Probe            `json:"startup_probe,omitempty"`   // 启动探针
	NodeSelector   map[string]string `json:"node_selector,omitempty"`   // 节点选择器
	Affinity       *Affinity         `json:"affinity,omitempty"`        // 亲和性配置
	Tolerations    []Toleration      `json:"tolerations,omitempty"`     // 容忍配置
}

// UpdateK8sInstanceResp 更新K8s实例响应
type UpdateK8sInstanceResp struct {
	InstanceID int `json:"instance_id"` // 实例ID
}

type BatchDeleteK8sInstanceReq struct {
	Instances []K8sInstance `json:"instances" binding:"required,min=1"` // 实例列表
}

type BatchDeleteK8sInstanceResp struct {
	DeletedCount int `json:"deleted_count"` // 成功删除的实例数量
}

type BatchRestartK8sInstanceReq struct {
	Instances []K8sInstance `json:"instances" binding:"required,min=1"` // 实例列表
}

type BatchRestartK8sInstanceResp struct {
	RestartedCount int `json:"restarted_count"` // 成功重启的实例数量
}

type GetK8sInstanceListReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"`                                // 集群ID
	Namespace string `json:"namespace,omitempty" form:"namespace"`                         // 命名空间过滤
	AppID     int    `json:"app_id,omitempty" form:"app_id"`                               // 应用ID过滤
	Name      string `json:"name,omitempty" form:"name"`                                   // 名称过滤（模糊查询）
	Page      int    `json:"page,omitempty" form:"page" binding:"min=1"`                   // 分页页码
	PageSize  int    `json:"page_size,omitempty" form:"page_size" binding:"min=1,max=100"` // 分页大小
	Type      string `json:"type,omitempty" form:"type"`                                   // 实例类型过滤(Deployment/StatefulSet/DaemonSet/Job/CronJob)
}

type GetK8sInstanceListResp struct {
	Total int           `json:"total"` // 总记录数
	Items []K8sInstance `json:"items"` // 实例列表
}

type GetK8sInstanceByAppReq struct {
	AppID     int    `json:"app_id" binding:"required"`     // 应用ID
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace,omitempty"`           // 命名空间
}

type GetK8sInstanceByAppResp struct {
	Total int           `json:"total"` // 总记录数
	Items []interface{} `json:"items"` // 实例运行状态列表
}

type GetK8sInstanceReq struct {
	Name      string `json:"name,omitempty"`                // 实例名称
	Namespace string `json:"namespace,omitempty"`           // 命名空间
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Type      string `json:"type,omitempty"`                // 实例类型(Deployment/StatefulSet/DaemonSet/Job/CronJob)
}

type GetK8sInstanceResp struct {
	Item interface{} `json:"item"` // Kubernetes 实例信息
}
