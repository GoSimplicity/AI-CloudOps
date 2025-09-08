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
)

// K8sPod Kubernetes Pod模型
type K8sPod struct {
	ClusterID         int64      `json:"cluster_id"`         // 集群ID
	Name              string     `json:"name"`               // Pod名称
	Namespace         string     `json:"namespace"`          // 所属命名空间
	UID               string     `json:"uid"`                // Pod UID
	Labels            string     `json:"labels"`             // 标签(JSON字符串)
	Annotations       string     `json:"annotations"`        // 注解(JSON字符串)
	Status            string     `json:"status"`             // Pod状态
	Phase             string     `json:"phase"`              // Pod阶段
	NodeName          string     `json:"node_name"`          // 所在节点
	PodIP             string     `json:"pod_ip"`             // Pod IP地址
	HostIP            string     `json:"host_ip"`            // 宿主机IP地址
	QosClass          string     `json:"qos_class"`          // QoS等级
	RestartCount      int32      `json:"restart_count"`      // 重启次数
	Ready             string     `json:"ready"`              // 就绪状态(如"1/1")
	ServiceAccount    string     `json:"service_account"`    // 服务账户
	RestartPolicy     string     `json:"restart_policy"`     // 重启策略
	DNSPolicy         string     `json:"dns_policy"`         // DNS策略
	Conditions        string     `json:"conditions"`         // Pod条件(JSON字符串)
	Containers        string     `json:"containers"`         // 容器列表(JSON字符串)
	InitContainers    string     `json:"init_containers"`    // 初始化容器列表(JSON字符串)
	Volumes           string     `json:"volumes"`            // 卷列表(JSON字符串)
	CreationTimestamp time.Time  `json:"creation_timestamp"` // 创建时间
	StartTime         *time.Time `json:"start_time"`         // 启动时间
	DeletionTimestamp *time.Time `json:"deletion_timestamp"` // 删除时间戳
	OwnerReferences   string     `json:"owner_references"`   // 所有者引用(JSON字符串)
	ResourceVersion   string     `json:"resource_version"`   // 资源版本
	Generation        int64      `json:"generation"`         // 生成版本号
	Spec              string     `json:"spec"`               // Pod规格(JSON字符串)
}

// PodContainer Pod容器信息
type PodContainer struct {
	Name            string                  `json:"name"`              // 容器名称
	Image           string                  `json:"image"`             // 容器镜像
	Command         []string                `json:"command"`           // 启动命令
	Args            []string                `json:"args"`              // 启动参数
	Envs            []PodEnvVar             `json:"envs"`              // 环境变量
	Ports           []PodContainerPort      `json:"ports"`             // 容器端口
	Resources       PodResourceRequirements `json:"resources"`         // 资源要求
	VolumeMounts    []PodVolumeMount        `json:"volume_mounts"`     // 卷挂载
	LivenessProbe   *PodProbe               `json:"liveness_probe"`    // 存活探测
	ReadinessProbe  *PodProbe               `json:"readiness_probe"`   // 就绪探测
	ImagePullPolicy string                  `json:"image_pull_policy"` // 镜像拉取策略
	Ready           bool                    `json:"ready"`             // 是否就绪
	RestartCount    int32                   `json:"restart_count"`     // 重启次数
	State           PodContainerState       `json:"state"`             // 容器状态
}

// PodEnvVar 环境变量
type PodEnvVar struct {
	Name  string `json:"name"`  // 环境变量名称
	Value string `json:"value"` // 环境变量值
}

// PodContainerPort 容器端口
type PodContainerPort struct {
	Name          string `json:"name"`           // 端口名称
	ContainerPort int32  `json:"container_port"` // 容器端口号
	Protocol      string `json:"protocol"`       // 协议类型
}

// PodResourceRequirements 资源要求
type PodResourceRequirements struct {
	Requests PodResourceList `json:"requests"` // 资源请求
	Limits   PodResourceList `json:"limits"`   // 资源限制
}

// PodResourceList 资源列表
type PodResourceList struct {
	CPU    string `json:"cpu"`    // CPU数量
	Memory string `json:"memory"` // 内存数量
}

// PodVolumeMount 卷挂载
type PodVolumeMount struct {
	Name      string `json:"name"`       // 卷名称
	MountPath string `json:"mount_path"` // 挂载路径
	ReadOnly  bool   `json:"read_only"`  // 是否只读
	SubPath   string `json:"sub_path"`   // 子路径
}

// PodProbe 探测配置
type PodProbe struct {
	HTTPGet             *PodHTTPGetAction `json:"http_get"`              // HTTP GET探测
	InitialDelaySeconds int32             `json:"initial_delay_seconds"` // 初始延迟时间
	PeriodSeconds       int32             `json:"period_seconds"`        // 探测间隔时间
	TimeoutSeconds      int32             `json:"timeout_seconds"`       // 探测超时时间
	SuccessThreshold    int32             `json:"success_threshold"`     // 成功阈值
	FailureThreshold    int32             `json:"failure_threshold"`     // 失败阈值
}

// PodHTTPGetAction HTTP GET探测动作
type PodHTTPGetAction struct {
	Path   string `json:"path"`   // 探测路径
	Port   int32  `json:"port"`   // 探测端口
	Scheme string `json:"scheme"` // 协议类型
}

// PodContainerState 容器状态
type PodContainerState struct {
	Waiting    *PodContainerStateWaiting    `json:"waiting"`    // 等待状态
	Running    *PodContainerStateRunning    `json:"running"`    // 运行状态
	Terminated *PodContainerStateTerminated `json:"terminated"` // 终止状态
}

// PodContainerStateWaiting 容器等待状态
type PodContainerStateWaiting struct {
	Reason  string `json:"reason"`  // 等待原因
	Message string `json:"message"` // 等待消息
}

// PodContainerStateRunning 容器运行状态
type PodContainerStateRunning struct {
	StartedAt time.Time `json:"started_at"` // 开始时间
}

// PodContainerStateTerminated 容器终止状态
type PodContainerStateTerminated struct {
	ExitCode    int32     `json:"exit_code"`    // 退出码
	Signal      int32     `json:"signal"`       // 信号
	Reason      string    `json:"reason"`       // 终止原因
	Message     string    `json:"message"`      // 终止消息
	StartedAt   time.Time `json:"started_at"`   // 开始时间
	FinishedAt  time.Time `json:"finished_at"`  // 结束时间
	ContainerID string    `json:"container_id"` // 容器ID
}

// PodCondition Pod条件
type PodCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 条件状态
	LastProbeTime      time.Time `json:"last_probe_time"`      // 最后探测时间
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// GetPodListReq 获取Pod列表请求
type GetPodListReq struct {
	ListReq
	ClusterID int    `json:"cluster_id" form:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" form:"namespace"`                                       // 命名空间
	Status    string `json:"status" form:"status"`                                             // Pod状态
}

// GetPodDetailsReq 获取Pod详情请求
type GetPodDetailsReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	Name      string `json:"name" uri:"name" binding:"required"`             // Pod名称
}

// GetPodYamlReq 获取Pod YAML请求
type GetPodYamlReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	Name      string `json:"name" uri:"name" binding:"required"`             // Pod名称
}

// CreatePodReq 创建Pod请求
type CreatePodReq struct {
	ClusterID      int                  `json:"cluster_id" binding:"required"` // 集群ID
	Name           string               `json:"name" binding:"required"`       // Pod名称
	Namespace      string               `json:"namespace" binding:"required"`  // 命名空间
	Labels         map[string]string    `json:"labels"`                        // 标签
	Annotations    map[string]string    `json:"annotations"`                   // 注解
	Containers     []CreatePodContainer `json:"containers" binding:"required"` // 容器列表
	InitContainers []CreatePodContainer `json:"init_containers"`               // 初始化容器列表
	RestartPolicy  string               `json:"restart_policy"`                // 重启策略
	NodeSelector   map[string]string    `json:"node_selector"`                 // 节点选择器
	Tolerations    []corev1.Toleration  `json:"tolerations"`                   // 容忍度
	Affinity       *corev1.Affinity     `json:"affinity"`                      // 亲和性
	Volumes        []corev1.Volume      `json:"volumes"`                       // 卷
	HostNetwork    bool                 `json:"host_network"`                  // 是否使用主机网络
	HostPID        bool                 `json:"host_pid"`                      // 是否使用主机PID
	DNSPolicy      string               `json:"dns_policy"`                    // DNS策略
	ServiceAccount string               `json:"service_account"`               // 服务账户
}

// CreatePodContainer 创建Pod容器配置
type CreatePodContainer struct {
	Name            string                  `json:"name" binding:"required"`  // 容器名称
	Image           string                  `json:"image" binding:"required"` // 容器镜像
	Command         []string                `json:"command"`                  // 启动命令
	Args            []string                `json:"args"`                     // 启动参数
	Envs            []PodEnvVar             `json:"envs"`                     // 环境变量
	Ports           []PodContainerPort      `json:"ports"`                    // 容器端口
	Resources       PodResourceRequirements `json:"resources"`                // 资源要求
	VolumeMounts    []PodVolumeMount        `json:"volume_mounts"`            // 卷挂载
	LivenessProbe   *PodProbe               `json:"liveness_probe"`           // 存活探测
	ReadinessProbe  *PodProbe               `json:"readiness_probe"`          // 就绪探测
	ImagePullPolicy string                  `json:"image_pull_policy"`        // 镜像拉取策略
	WorkingDir      string                  `json:"working_dir"`              // 工作目录
	SecurityContext *corev1.SecurityContext `json:"security_context"`         // 安全上下文
}

// CreatePodByYamlReq 通过YAML创建Pod请求
type CreatePodByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
}

// UpdatePodReq 更新Pod请求
type UpdatePodReq struct {
	ClusterID   int               `json:"cluster_id"`  // 集群ID
	Name        string            `json:"name"`        // Pod名称
	Namespace   string            `json:"namespace"`   // 命名空间
	Labels      map[string]string `json:"labels"`      // 标签
	Annotations map[string]string `json:"annotations"` // 注解
}

// UpdatePodByYamlReq 通过YAML更新Pod请求
type UpdatePodByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" binding:"required"`  // 命名空间
	Name      string `json:"name" binding:"required"`       // Pod名称
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
}

// DeletePodReq 删除Pod请求
type DeletePodReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required"` // 集群ID
	Namespace          string `json:"namespace" binding:"required"`  // 命名空间
	Name               string `json:"name" binding:"required"`       // Pod名称
	GracePeriodSeconds *int64 `json:"grace_period_seconds"`          // 优雅删除时间（秒）
	Force              bool   `json:"force"`                         // 是否强制删除
}

// BatchDeletePodsReq 批量删除Pod请求
type BatchDeletePodsReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required"` // 集群ID
	Namespace          string   `json:"namespace" binding:"required"`  // 命名空间
	Names              []string `json:"names" binding:"required"`      // Pod名称列表
	GracePeriodSeconds *int64   `json:"grace_period_seconds"`          // 优雅删除时间（秒）
	Force              bool     `json:"force"`                         // 是否强制删除
}

// GetPodsByNodeReq 根据节点获取Pod列表请求
type GetPodsByNodeReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	NodeName  string `json:"node_name" uri:"node_name" binding:"required"`   // 节点名称
}

// GetPodContainersReq 获取Pod容器列表请求
type GetPodContainersReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	PodName   string `json:"pod_name" uri:"pod_name" binding:"required"`     // Pod名称
}

// GetPodLogsReq Pod日志查询请求
type GetPodLogsReq struct {
	ClusterID    int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace    string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	PodName      string `json:"pod_name" uri:"pod_name" binding:"required"`     // Pod名称
	Container    string `json:"container" uri:"container" binding:"required"`   // 容器名称
	Follow       bool   `json:"follow"`                                         // 是否持续跟踪
	Previous     bool   `json:"previous"`                                       // 是否获取前一个容器的日志
	SinceSeconds *int64 `json:"since_seconds"`                                  // 获取多少秒内的日志
	SinceTime    string `json:"since_time"`                                     // 从指定时间开始获取日志
	Timestamps   bool   `json:"timestamps"`                                     // 是否显示时间戳
	TailLines    *int64 `json:"tail_lines"`                                     // 获取最后几行日志
	LimitBytes   *int64 `json:"limit_bytes"`                                    // 限制日志字节数
}

// PodExecReq Pod执行命令请求
type PodExecReq struct {
	ClusterID int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	PodName   string `json:"pod_name" uri:"pod_name" binding:"required"`     // Pod名称
	Container string `json:"container" uri:"container" binding:"required"`   // 容器名称
	Shell     string `json:"shell" form:"shell"`                             // shell类型
}

// PodPortForwardReq Pod端口转发请求
type PodPortForwardReq struct {
	ClusterID int                  `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace string               `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	PodName   string               `json:"pod_name" uri:"pod_name" binding:"required"`     // Pod名称
	Ports     []PodPortForwardPort `json:"ports" binding:"required"`                       // 端口转发配置
}

// PodPortForwardPort 端口转发端口配置
type PodPortForwardPort struct {
	LocalPort  int `json:"local_port" binding:"required"`  // 本地端口
	RemotePort int `json:"remote_port" binding:"required"` // 远程端口
}

// PodFileUploadReq Pod文件上传请求
type PodFileUploadReq struct {
	ClusterID     int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace     string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	PodName       string `json:"pod_name" uri:"pod_name" binding:"required"`     // Pod名称
	ContainerName string `json:"container" uri:"container" binding:"required"`   // 容器名称
	FilePath      string `json:"file_path" uri:"file_path" binding:"required"`   // 文件路径
}

// PodFileDownloadReq Pod文件下载请求
type PodFileDownloadReq struct {
	ClusterID     int    `json:"cluster_id" uri:"cluster_id" binding:"required"` // 集群ID
	Namespace     string `json:"namespace" uri:"namespace" binding:"required"`   // 命名空间
	PodName       string `json:"pod_name" uri:"pod_name" binding:"required"`     // Pod名称
	ContainerName string `json:"container" uri:"container" binding:"required"`   // 容器名称
	FilePath      string `json:"file_path" uri:"file_path" binding:"required"`   // 文件路径
}
