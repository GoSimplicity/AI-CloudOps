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
)



// K8sStatefulSetEntity Kubernetes StatefulSet数据库实体
type K8sStatefulSetEntity struct {
	Model
	Name                 string                   `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:StatefulSet名称"` // StatefulSet名称
	Namespace            string                   `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"`   // 所属命名空间
	ClusterID            int                      `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                             // 所属集群ID
	UID                  string                   `json:"uid" gorm:"size:100;comment:StatefulSet UID"`                                 // StatefulSet UID
	Replicas             int32                    `json:"replicas" gorm:"comment:期望副本数"`                                               // 期望副本数
	ReadyReplicas        int32                    `json:"ready_replicas" gorm:"comment:就绪副本数"`                                         // 就绪副本数
	CurrentReplicas      int32                    `json:"current_replicas" gorm:"comment:当前副本数"`                                       // 当前副本数
	UpdatedReplicas      int32                    `json:"updated_replicas" gorm:"comment:更新副本数"`                                       // 更新副本数
	ServiceName          string                   `json:"service_name" gorm:"size:200;comment:服务名称"`                                   // 服务名称
	UpdateStrategy       string                   `json:"update_strategy" gorm:"size:50;comment:更新策略"`                                 // 更新策略
	RevisionHistoryLimit int32                    `json:"revision_history_limit" gorm:"comment:历史版本限制"`                                // 历史版本限制
	PodManagementPolicy  string                   `json:"pod_management_policy" gorm:"size:50;comment:Pod管理策略"`                        // Pod管理策略
	Selector             map[string]string        `json:"selector" gorm:"type:text;serializer:json;comment:选择器"`                       // 选择器
	PodTemplate          map[string]interface{}   `json:"pod_template" gorm:"type:text;serializer:json;comment:Pod模板"`                 // Pod模板
	VolumeClaimTemplates []map[string]interface{} `json:"volume_claim_templates" gorm:"type:text;serializer:json;comment:卷声明模板"`       // 卷声明模板
	Labels               map[string]string        `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                          // 标签
	Annotations          map[string]string        `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                     // 注解
	CreationTimestamp    time.Time                `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                            // Kubernetes创建时间
	Age                  string                   `json:"age" gorm:"-"`                                                                // 存在时间，前端计算使用
	Status               string                   `json:"status" gorm:"-"`                                                             // StatefulSet状态，前端计算使用
	Images               []string                 `json:"images" gorm:"-"`                                                             // 镜像列表，前端计算使用
}

func (k *K8sStatefulSetEntity) TableName() string {
	return "cl_k8s_statefulsets"
}

// K8sStatefulSetListRequest StatefulSet列表查询请求
type K8sStatefulSetListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	ServiceName   string `json:"service_name" form:"service_name" comment:"服务名称过滤"`              // 服务名称过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sStatefulSetCreateRequest 创建StatefulSet请求
type K8sStatefulSetCreateReq struct {
	ClusterID            int                      `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace            string                   `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name                 string                   `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	Replicas             int32                    `json:"replicas" binding:"required" comment:"副本数"`       // 副本数，必填
	ServiceName          string                   `json:"service_name" binding:"required" comment:"服务名称"`  // 服务名称，必填
	UpdateStrategy       string                   `json:"update_strategy" comment:"更新策略"`                  // 更新策略
	RevisionHistoryLimit *int32                   `json:"revision_history_limit" comment:"历史版本限制"`         // 历史版本限制
	PodManagementPolicy  string                   `json:"pod_management_policy" comment:"Pod管理策略"`         // Pod管理策略
	Selector             map[string]string        `json:"selector" binding:"required" comment:"选择器"`       // 选择器，必填
	PodTemplate          map[string]interface{}   `json:"pod_template" binding:"required" comment:"Pod模板"` // Pod模板，必填
	VolumeClaimTemplates []map[string]interface{} `json:"volume_claim_templates" comment:"卷声明模板"`          // 卷声明模板
	Labels               map[string]string        `json:"labels" comment:"标签"`                             // 标签
	Annotations          map[string]string        `json:"annotations" comment:"注解"`                        // 注解
	StatefulSetYaml      *appsv1.StatefulSet      `json:"statefulset_yaml" comment:"StatefulSet YAML对象"`   // StatefulSet YAML对象
}

// K8sStatefulSetUpdateRequest 更新StatefulSet请求
type K8sStatefulSetUpdateReq struct {
	ClusterID            int                      `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace            string                   `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name                 string                   `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	Replicas             *int32                   `json:"replicas" comment:"副本数"`                          // 副本数
	ServiceName          string                   `json:"service_name" comment:"服务名称"`                     // 服务名称
	UpdateStrategy       string                   `json:"update_strategy" comment:"更新策略"`                  // 更新策略
	RevisionHistoryLimit *int32                   `json:"revision_history_limit" comment:"历史版本限制"`         // 历史版本限制
	PodManagementPolicy  string                   `json:"pod_management_policy" comment:"Pod管理策略"`         // Pod管理策略
	Selector             map[string]string        `json:"selector" comment:"选择器"`                          // 选择器
	PodTemplate          map[string]interface{}   `json:"pod_template" comment:"Pod模板"`                    // Pod模板
	VolumeClaimTemplates []map[string]interface{} `json:"volume_claim_templates" comment:"卷声明模板"`          // 卷声明模板
	Labels               map[string]string        `json:"labels" comment:"标签"`                             // 标签
	Annotations          map[string]string        `json:"annotations" comment:"注解"`                        // 注解
	StatefulSetYaml      *appsv1.StatefulSet      `json:"statefulset_yaml" comment:"StatefulSet YAML对象"`   // StatefulSet YAML对象
}

// K8sStatefulSetDeleteRequest 删除StatefulSet请求
type K8sStatefulSetDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`        // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                          // 是否强制删除
	OrphanDependents   bool   `json:"orphan_dependents" comment:"是否保留依赖资源"`            // 是否保留依赖资源
}

// K8sStatefulSetBatchDeleteRequest 批量删除StatefulSet请求
type K8sStatefulSetBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`       // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`        // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"StatefulSet名称列表"` // StatefulSet名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`           // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                             // 是否强制删除
	OrphanDependents   bool     `json:"orphan_dependents" comment:"是否保留依赖资源"`               // 是否保留依赖资源
}

// K8sStatefulSetScaleRequest 扩缩容StatefulSet请求
type K8sStatefulSetScaleReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	Replicas  int32  `json:"replicas" binding:"required" comment:"副本数"`       // 副本数，必填
}

// K8sStatefulSetRestartRequest 重启StatefulSet请求
type K8sStatefulSetRestartReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
}

// K8sStatefulSetEventRequest 获取StatefulSet事件请求
type K8sStatefulSetEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                   // 限制天数内的事件
}

// K8sStatefulSetMetricsRequest 获取StatefulSet指标请求
type K8sStatefulSetMetricsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"StatefulSet名称"` // StatefulSet名称，必填
	TimeRange string `json:"time_range" comment:"时间范围"`                       // 时间范围
}

// StatefulSetEntity StatefulSet响应实体
type StatefulSetEntity struct {
	Name                 string                                 `json:"name"`                   // StatefulSet名称
	Namespace            string                                 `json:"namespace"`              // 命名空间
	UID                  string                                 `json:"uid"`                    // StatefulSet UID
	Labels               map[string]string                      `json:"labels"`                 // 标签
	Annotations          map[string]string                      `json:"annotations"`            // 注解
	Replicas             int32                                  `json:"replicas"`               // 期望副本数
	ReadyReplicas        int32                                  `json:"ready_replicas"`         // 就绪副本数
	CurrentReplicas      int32                                  `json:"current_replicas"`       // 当前副本数
	UpdatedReplicas      int32                                  `json:"updated_replicas"`       // 更新副本数
	ServiceName          string                                 `json:"service_name"`           // 服务名称
	UpdateStrategy       StatefulSetUpdateStrategyEntity        `json:"update_strategy"`        // 更新策略
	RevisionHistoryLimit int32                                  `json:"revision_history_limit"` // 历史版本限制
	PodManagementPolicy  string                                 `json:"pod_management_policy"`  // Pod管理策略
	Selector             StatefulSetSelectorEntity              `json:"selector"`               // 选择器
	PodTemplate          StatefulSetPodTemplateEntity           `json:"pod_template"`           // Pod模板
	VolumeClaimTemplates []StatefulSetVolumeClaimTemplateEntity `json:"volume_claim_templates"` // 卷声明模板
	Status               string                                 `json:"status"`                 // StatefulSet状态
	Images               []string                               `json:"images"`                 // 镜像列表
	Age                  string                                 `json:"age"`                    // 存在时间
	CreatedAt            string                                 `json:"created_at"`             // 创建时间
}

// StatefulSetUpdateStrategyEntity StatefulSet更新策略实体
type StatefulSetUpdateStrategyEntity struct {
	Type          string                                 `json:"type"`           // 更新类型
	RollingUpdate StatefulSetRollingUpdateStrategyEntity `json:"rolling_update"` // 滚动更新策略
}

// StatefulSetRollingUpdateStrategyEntity StatefulSet滚动更新策略实体
type StatefulSetRollingUpdateStrategyEntity struct {
	Partition      *int32 `json:"partition"`       // 分区
	MaxUnavailable *int32 `json:"max_unavailable"` // 最大不可用
}

// StatefulSetSelectorEntity StatefulSet选择器实体
type StatefulSetSelectorEntity struct {
	MatchLabels      map[string]string                      `json:"match_labels"`      // 标签匹配
	MatchExpressions []StatefulSetSelectorRequirementEntity `json:"match_expressions"` // 表达式匹配
}

// StatefulSetSelectorRequirementEntity StatefulSet选择器要求实体
type StatefulSetSelectorRequirementEntity struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符
	Values   []string `json:"values"`   // 值列表
}

// StatefulSetPodTemplateEntity StatefulSet Pod模板实体
type StatefulSetPodTemplateEntity struct {
	Labels      map[string]string        `json:"labels"`      // 标签
	Annotations map[string]string        `json:"annotations"` // 注解
	Spec        StatefulSetPodSpecEntity `json:"spec"`        // Pod规格
}

// StatefulSetPodSpecEntity StatefulSet Pod规格实体
type StatefulSetPodSpecEntity struct {
	Containers                    []StatefulSetContainerEntity        `json:"containers"`                       // 容器列表
	InitContainers                []StatefulSetContainerEntity        `json:"init_containers"`                  // 初始化容器列表
	RestartPolicy                 string                              `json:"restart_policy"`                   // 重启策略
	TerminationGracePeriodSeconds *int64                              `json:"termination_grace_period_seconds"` // 终止宽限期
	DNSPolicy                     string                              `json:"dns_policy"`                       // DNS策略
	ServiceAccountName            string                              `json:"service_account_name"`             // 服务账户名称
	SecurityContext               StatefulSetPodSecurityContextEntity `json:"security_context"`                 // 安全上下文
	Volumes                       []StatefulSetVolumeEntity           `json:"volumes"`                          // 卷列表
	NodeSelector                  map[string]string                   `json:"node_selector"`                    // 节点选择器
	Tolerations                   []StatefulSetTolerationEntity       `json:"tolerations"`                      // 容忍度
	Affinity                      StatefulSetAffinityEntity           `json:"affinity"`                         // 亲和性
}

// StatefulSetContainerEntity StatefulSet容器实体
type StatefulSetContainerEntity struct {
	Name            string                                `json:"name"`              // 容器名称
	Image           string                                `json:"image"`             // 镜像
	ImagePullPolicy string                                `json:"image_pull_policy"` // 镜像拉取策略
	Ports           []StatefulSetContainerPortEntity      `json:"ports"`             // 端口列表
	Env             []StatefulSetEnvVarEntity             `json:"env"`               // 环境变量
	Resources       StatefulSetResourceRequirementsEntity `json:"resources"`         // 资源要求
	VolumeMounts    []StatefulSetVolumeMountEntity        `json:"volume_mounts"`     // 卷挂载
	LivenessProbe   StatefulSetProbeEntity                `json:"liveness_probe"`    // 存活探针
	ReadinessProbe  StatefulSetProbeEntity                `json:"readiness_probe"`   // 就绪探针
	StartupProbe    StatefulSetProbeEntity                `json:"startup_probe"`     // 启动探针
	SecurityContext StatefulSetSecurityContextEntity      `json:"security_context"`  // 安全上下文
	Command         []string                              `json:"command"`           // 命令
	Args            []string                              `json:"args"`              // 参数
}

// StatefulSetContainerPortEntity StatefulSet容器端口实体
type StatefulSetContainerPortEntity struct {
	Name          string `json:"name"`           // 端口名称
	ContainerPort int32  `json:"container_port"` // 容器端口
	Protocol      string `json:"protocol"`       // 协议
	HostIP        string `json:"host_ip"`        // 主机IP
	HostPort      int32  `json:"host_port"`      // 主机端口
}

// StatefulSetEnvVarEntity StatefulSet环境变量实体
type StatefulSetEnvVarEntity struct {
	Name      string                        `json:"name"`       // 变量名
	Value     string                        `json:"value"`      // 变量值
	ValueFrom StatefulSetEnvVarSourceEntity `json:"value_from"` // 变量来源
}

// StatefulSetEnvVarSourceEntity StatefulSet环境变量来源实体
type StatefulSetEnvVarSourceEntity struct {
	FieldRef         StatefulSetObjectFieldSelectorEntity   `json:"field_ref"`          // 字段引用
	ResourceFieldRef StatefulSetResourceFieldSelectorEntity `json:"resource_field_ref"` // 资源字段引用
	ConfigMapKeyRef  StatefulSetConfigMapKeySelectorEntity  `json:"config_map_key_ref"` // ConfigMap键引用
	SecretKeyRef     StatefulSetSecretKeySelectorEntity     `json:"secret_key_ref"`     // Secret键引用
}

// StatefulSetObjectFieldSelectorEntity StatefulSet对象字段选择器实体
type StatefulSetObjectFieldSelectorEntity struct {
	APIVersion string `json:"api_version"` // API版本
	FieldPath  string `json:"field_path"`  // 字段路径
}

// StatefulSetResourceFieldSelectorEntity StatefulSet资源字段选择器实体
type StatefulSetResourceFieldSelectorEntity struct {
	ContainerName string `json:"container_name"` // 容器名称
	Resource      string `json:"resource"`       // 资源
	Divisor       string `json:"divisor"`        // 除数
}

// StatefulSetConfigMapKeySelectorEntity StatefulSet ConfigMap键选择器实体
type StatefulSetConfigMapKeySelectorEntity struct {
	Name     string `json:"name"`     // ConfigMap名称
	Key      string `json:"key"`      // 键
	Optional *bool  `json:"optional"` // 是否可选
}

// StatefulSetSecretKeySelectorEntity StatefulSet Secret键选择器实体
type StatefulSetSecretKeySelectorEntity struct {
	Name     string `json:"name"`     // Secret名称
	Key      string `json:"key"`      // 键
	Optional *bool  `json:"optional"` // 是否可选
}

// StatefulSetResourceRequirementsEntity StatefulSet资源要求实体
type StatefulSetResourceRequirementsEntity struct {
	Limits   map[string]string `json:"limits"`   // 资源限制
	Requests map[string]string `json:"requests"` // 资源请求
}

// StatefulSetVolumeMountEntity StatefulSet卷挂载实体
type StatefulSetVolumeMountEntity struct {
	Name             string `json:"name"`              // 卷名称
	MountPath        string `json:"mount_path"`        // 挂载路径
	SubPath          string `json:"sub_path"`          // 子路径
	MountPropagation string `json:"mount_propagation"` // 挂载传播
	ReadOnly         bool   `json:"read_only"`         // 是否只读
}

// StatefulSetProbeEntity StatefulSet探针实体
type StatefulSetProbeEntity struct {
	HTTPGet             StatefulSetHTTPGetActionEntity   `json:"http_get"`              // HTTP GET
	Exec                StatefulSetExecActionEntity      `json:"exec"`                  // 执行命令
	TCPSocket           StatefulSetTCPSocketActionEntity `json:"tcp_socket"`            // TCP套接字
	InitialDelaySeconds int32                            `json:"initial_delay_seconds"` // 初始延迟
	TimeoutSeconds      int32                            `json:"timeout_seconds"`       // 超时时间
	PeriodSeconds       int32                            `json:"period_seconds"`        // 周期
	SuccessThreshold    int32                            `json:"success_threshold"`     // 成功阈值
	FailureThreshold    int32                            `json:"failure_threshold"`     // 失败阈值
}

// StatefulSetHTTPGetActionEntity StatefulSet HTTP GET动作实体
type StatefulSetHTTPGetActionEntity struct {
	Path        string                        `json:"path"`         // 路径
	Port        int32                         `json:"port"`         // 端口
	Host        string                        `json:"host"`         // 主机
	Scheme      string                        `json:"scheme"`       // 协议
	HTTPHeaders []StatefulSetHTTPHeaderEntity `json:"http_headers"` // HTTP头
}

// StatefulSetHTTPHeaderEntity StatefulSet HTTP头实体
type StatefulSetHTTPHeaderEntity struct {
	Name  string `json:"name"`  // 头名称
	Value string `json:"value"` // 头值
}

// StatefulSetExecActionEntity StatefulSet执行命令动作实体
type StatefulSetExecActionEntity struct {
	Command []string `json:"command"` // 命令
}

// StatefulSetTCPSocketActionEntity StatefulSet TCP套接字动作实体
type StatefulSetTCPSocketActionEntity struct {
	Port int32  `json:"port"` // 端口
	Host string `json:"host"` // 主机
}

// StatefulSetSecurityContextEntity StatefulSet安全上下文实体
type StatefulSetSecurityContextEntity struct {
	RunAsUser                *int64                        `json:"run_as_user"`                // 运行用户ID
	RunAsGroup               *int64                        `json:"run_as_group"`               // 运行组ID
	RunAsNonRoot             *bool                         `json:"run_as_non_root"`            // 是否以非root运行
	ReadOnlyRootFilesystem   *bool                         `json:"read_only_root_filesystem"`  // 根文件系统是否只读
	AllowPrivilegeEscalation *bool                         `json:"allow_privilege_escalation"` // 是否允许特权升级
	Privileged               *bool                         `json:"privileged"`                 // 是否特权模式
	Capabilities             StatefulSetCapabilitiesEntity `json:"capabilities"`               // 能力
}

// StatefulSetCapabilitiesEntity StatefulSet能力实体
type StatefulSetCapabilitiesEntity struct {
	Add  []string `json:"add"`  // 添加的能力
	Drop []string `json:"drop"` // 删除的能力
}

// StatefulSetPodSecurityContextEntity StatefulSet Pod安全上下文实体
type StatefulSetPodSecurityContextEntity struct {
	SELinuxOptions      StatefulSetSELinuxOptionsEntity                `json:"selinux_options"`        // SELinux选项
	WindowsOptions      StatefulSetWindowsSecurityContextOptionsEntity `json:"windows_options"`        // Windows选项
	RunAsUser           *int64                                         `json:"run_as_user"`            // 运行用户ID
	RunAsGroup          *int64                                         `json:"run_as_group"`           // 运行组ID
	RunAsNonRoot        *bool                                          `json:"run_as_non_root"`        // 是否以非root运行
	SupplementalGroups  []int64                                        `json:"supplemental_groups"`    // 补充组
	FSGroup             *int64                                         `json:"fs_group"`               // 文件系统组
	Sysctls             []StatefulSetSysctlEntity                      `json:"sysctls"`                // 系统控制
	FSGroupChangePolicy string                                         `json:"fs_group_change_policy"` // 文件系统组变更策略
	SeccompProfile      StatefulSetSeccompProfileEntity                `json:"seccomp_profile"`        // Seccomp配置
}

// StatefulSetSELinuxOptionsEntity StatefulSet SELinux选项实体
type StatefulSetSELinuxOptionsEntity struct {
	User  string `json:"user"`  // 用户
	Role  string `json:"role"`  // 角色
	Type  string `json:"type"`  // 类型
	Level string `json:"level"` // 级别
}

// StatefulSetWindowsSecurityContextOptionsEntity StatefulSet Windows安全上下文选项实体
type StatefulSetWindowsSecurityContextOptionsEntity struct {
	GMSACredentialSpecName string `json:"gmsa_credential_spec_name"` // GMSA凭据规格名称
	GMSACredentialSpec     string `json:"gmsa_credential_spec"`      // GMSA凭据规格
	RunAsUserName          string `json:"run_as_user_name"`          // 运行用户名
	HostProcess            *bool  `json:"host_process"`              // 是否主机进程
}

// StatefulSetSysctlEntity StatefulSet系统控制实体
type StatefulSetSysctlEntity struct {
	Name  string `json:"name"`  // 名称
	Value string `json:"value"` // 值
}

// StatefulSetSeccompProfileEntity StatefulSet Seccomp配置实体
type StatefulSetSeccompProfileEntity struct {
	Type             string `json:"type"`              // 类型
	LocalhostProfile string `json:"localhost_profile"` // 本地配置文件
}

// StatefulSetVolumeEntity StatefulSet卷实体
type StatefulSetVolumeEntity struct {
	Name                  string                                             `json:"name"`                    // 卷名称
	HostPath              StatefulSetHostPathVolumeSourceEntity              `json:"host_path"`               // 主机路径
	EmptyDir              StatefulSetEmptyDirVolumeSourceEntity              `json:"empty_dir"`               // 空目录
	Secret                StatefulSetSecretVolumeSourceEntity                `json:"secret"`                  // Secret
	ConfigMap             StatefulSetConfigMapVolumeSourceEntity             `json:"config_map"`              // ConfigMap
	PersistentVolumeClaim StatefulSetPersistentVolumeClaimVolumeSourceEntity `json:"persistent_volume_claim"` // PVC
}

// StatefulSetHostPathVolumeSourceEntity StatefulSet主机路径卷源实体
type StatefulSetHostPathVolumeSourceEntity struct {
	Path string `json:"path"` // 路径
	Type string `json:"type"` // 类型
}

// StatefulSetEmptyDirVolumeSourceEntity StatefulSet空目录卷源实体
type StatefulSetEmptyDirVolumeSourceEntity struct {
	Medium    string `json:"medium"`     // 介质
	SizeLimit string `json:"size_limit"` // 大小限制
}

// StatefulSetSecretVolumeSourceEntity StatefulSet Secret卷源实体
type StatefulSetSecretVolumeSourceEntity struct {
	SecretName  string                       `json:"secret_name"`  // Secret名称
	Items       []StatefulSetKeyToPathEntity `json:"items"`        // 项目列表
	DefaultMode *int32                       `json:"default_mode"` // 默认模式
	Optional    *bool                        `json:"optional"`     // 是否可选
}

// StatefulSetConfigMapVolumeSourceEntity StatefulSet ConfigMap卷源实体
type StatefulSetConfigMapVolumeSourceEntity struct {
	Name        string                       `json:"name"`         // ConfigMap名称
	Items       []StatefulSetKeyToPathEntity `json:"items"`        // 项目列表
	DefaultMode *int32                       `json:"default_mode"` // 默认模式
	Optional    *bool                        `json:"optional"`     // 是否可选
}

// StatefulSetPersistentVolumeClaimVolumeSourceEntity StatefulSet PVC卷源实体
type StatefulSetPersistentVolumeClaimVolumeSourceEntity struct {
	ClaimName string `json:"claim_name"` // PVC名称
	ReadOnly  bool   `json:"read_only"`  // 是否只读
}

// StatefulSetKeyToPathEntity StatefulSet键到路径实体
type StatefulSetKeyToPathEntity struct {
	Key  string `json:"key"`  // 键
	Path string `json:"path"` // 路径
	Mode *int32 `json:"mode"` // 模式
}

// StatefulSetTolerationEntity StatefulSet容忍度实体
type StatefulSetTolerationEntity struct {
	Key               string `json:"key"`                // 键
	Operator          string `json:"operator"`           // 操作符
	Value             string `json:"value"`              // 值
	Effect            string `json:"effect"`             // 效果
	TolerationSeconds *int64 `json:"toleration_seconds"` // 容忍时间
}

// StatefulSetAffinityEntity StatefulSet亲和性实体
type StatefulSetAffinityEntity struct {
	NodeAffinity    StatefulSetNodeAffinityEntity    `json:"node_affinity"`     // 节点亲和性
	PodAffinity     StatefulSetPodAffinityEntity     `json:"pod_affinity"`      // Pod亲和性
	PodAntiAffinity StatefulSetPodAntiAffinityEntity `json:"pod_anti_affinity"` // Pod反亲和性
}

// StatefulSetNodeAffinityEntity StatefulSet节点亲和性实体
type StatefulSetNodeAffinityEntity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  StatefulSetNodeSelectorEntity              `json:"required_during_scheduling_ignored_during_execution"`  // 调度时必须满足
	PreferredDuringSchedulingIgnoredDuringExecution []StatefulSetPreferredSchedulingTermEntity `json:"preferred_during_scheduling_ignored_during_execution"` // 调度时优先满足
}

// StatefulSetNodeSelectorEntity StatefulSet节点选择器实体
type StatefulSetNodeSelectorEntity struct {
	NodeSelectorTerms []StatefulSetNodeSelectorTermEntity `json:"node_selector_terms"` // 节点选择器条件
}

// StatefulSetNodeSelectorTermEntity StatefulSet节点选择器条件实体
type StatefulSetNodeSelectorTermEntity struct {
	MatchExpressions []StatefulSetNodeSelectorRequirementEntity `json:"match_expressions"` // 匹配表达式
	MatchFields      []StatefulSetNodeSelectorRequirementEntity `json:"match_fields"`      // 匹配字段
}

// StatefulSetNodeSelectorRequirementEntity StatefulSet节点选择器要求实体
type StatefulSetNodeSelectorRequirementEntity struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符
	Values   []string `json:"values"`   // 值列表
}

// StatefulSetPreferredSchedulingTermEntity StatefulSet优先调度条件实体
type StatefulSetPreferredSchedulingTermEntity struct {
	Weight     int32                             `json:"weight"`     // 权重
	Preference StatefulSetNodeSelectorTermEntity `json:"preference"` // 偏好
}

// StatefulSetPodAffinityEntity StatefulSet Pod亲和性实体
type StatefulSetPodAffinityEntity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []StatefulSetPodAffinityTermEntity         `json:"required_during_scheduling_ignored_during_execution"`  // 调度时必须满足
	PreferredDuringSchedulingIgnoredDuringExecution []StatefulSetWeightedPodAffinityTermEntity `json:"preferred_during_scheduling_ignored_during_execution"` // 调度时优先满足
}

// StatefulSetPodAntiAffinityEntity StatefulSet Pod反亲和性实体
type StatefulSetPodAntiAffinityEntity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []StatefulSetPodAffinityTermEntity         `json:"required_during_scheduling_ignored_during_execution"`  // 调度时必须满足
	PreferredDuringSchedulingIgnoredDuringExecution []StatefulSetWeightedPodAffinityTermEntity `json:"preferred_during_scheduling_ignored_during_execution"` // 调度时优先满足
}

// StatefulSetPodAffinityTermEntity StatefulSet Pod亲和性条件实体
type StatefulSetPodAffinityTermEntity struct {
	LabelSelector     StatefulSetSelectorEntity `json:"label_selector"`     // 标签选择器
	NamespaceSelector StatefulSetSelectorEntity `json:"namespace_selector"` // 命名空间选择器
	Namespaces        []string                  `json:"namespaces"`         // 命名空间列表
	TopologyKey       string                    `json:"topology_key"`       // 拓扑键
}

// StatefulSetWeightedPodAffinityTermEntity StatefulSet加权Pod亲和性条件实体
type StatefulSetWeightedPodAffinityTermEntity struct {
	Weight          int32                            `json:"weight"`            // 权重
	PodAffinityTerm StatefulSetPodAffinityTermEntity `json:"pod_affinity_term"` // Pod亲和性条件
}

// StatefulSetVolumeClaimTemplateEntity StatefulSet卷声明模板实体
type StatefulSetVolumeClaimTemplateEntity struct {
	Metadata StatefulSetVolumeClaimTemplateMetadataEntity `json:"metadata"` // 元数据
	Spec     StatefulSetVolumeClaimTemplateSpecEntity     `json:"spec"`     // 规格
}

// StatefulSetVolumeClaimTemplateMetadataEntity StatefulSet卷声明模板元数据实体
type StatefulSetVolumeClaimTemplateMetadataEntity struct {
	Name        string            `json:"name"`        // 名称
	Labels      map[string]string `json:"labels"`      // 标签
	Annotations map[string]string `json:"annotations"` // 注解
}

// StatefulSetVolumeClaimTemplateSpecEntity StatefulSet卷声明模板规格实体
type StatefulSetVolumeClaimTemplateSpecEntity struct {
	AccessModes      []string                                   `json:"access_modes"`       // 访问模式
	Selector         StatefulSetSelectorEntity                  `json:"selector"`           // 选择器
	Resources        StatefulSetResourceRequirementsEntity      `json:"resources"`          // 资源要求
	VolumeName       string                                     `json:"volume_name"`        // 卷名称
	StorageClassName string                                     `json:"storage_class_name"` // 存储类名称
	VolumeMode       string                                     `json:"volume_mode"`        // 卷模式
	DataSource       StatefulSetTypedLocalObjectReferenceEntity `json:"data_source"`        // 数据源
}

// StatefulSetTypedLocalObjectReferenceEntity StatefulSet类型化本地对象引用实体
type StatefulSetTypedLocalObjectReferenceEntity struct {
	APIGroup string `json:"api_group"` // API组
	Kind     string `json:"kind"`      // 类型
	Name     string `json:"name"`      // 名称
}

// StatefulSetListResponse StatefulSet列表响应
type StatefulSetListResponse struct {
	Items      []StatefulSetEntity `json:"items"`       // StatefulSet列表
	TotalCount int                 `json:"total_count"` // 总数
}

// StatefulSetDetailResponse StatefulSet详情响应
type StatefulSetDetailResponse struct {
	StatefulSet StatefulSetEntity        `json:"statefulset"` // StatefulSet信息
	YAML        string                   `json:"yaml"`        // YAML内容
	Events      []StatefulSetEventEntity `json:"events"`      // 事件列表
	Pods        []StatefulSetPodEntity   `json:"pods"`        // Pod列表
	Metrics     StatefulSetMetricsEntity `json:"metrics"`     // 指标信息
	Service     StatefulSetServiceEntity `json:"service"`     // 关联服务
}

// StatefulSetEventEntity StatefulSet事件实体
type StatefulSetEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// StatefulSetPodEntity StatefulSet Pod实体
type StatefulSetPodEntity struct {
	Name      string `json:"name"`      // Pod名称
	Ready     string `json:"ready"`     // 就绪状态
	Status    string `json:"status"`    // Pod状态
	Restarts  int32  `json:"restarts"`  // 重启次数
	Age       string `json:"age"`       // 存在时间
	IP        string `json:"ip"`        // Pod IP
	Node      string `json:"node"`      // 节点名称
	Nominated string `json:"nominated"` // 提名节点
	Readiness string `json:"readiness"` // 就绪状态
}

// StatefulSetMetricsEntity StatefulSet指标实体
type StatefulSetMetricsEntity struct {
	CPUUsage     float64 `json:"cpu_usage"`     // CPU使用量
	MemoryUsage  int64   `json:"memory_usage"`  // 内存使用量
	NetworkRx    int64   `json:"network_rx"`    // 网络接收
	NetworkTx    int64   `json:"network_tx"`    // 网络发送
	StorageUsage int64   `json:"storage_usage"` // 存储使用量
}

// StatefulSetServiceEntity StatefulSet服务实体
type StatefulSetServiceEntity struct {
	Name        string                         `json:"name"`         // 服务名称
	Type        string                         `json:"type"`         // 服务类型
	ClusterIP   string                         `json:"cluster_ip"`   // 集群IP
	ExternalIPs []string                       `json:"external_ips"` // 外部IP
	Ports       []StatefulSetServicePortEntity `json:"ports"`        // 端口列表
	Selector    map[string]string              `json:"selector"`     // 选择器
}

// StatefulSetServicePortEntity StatefulSet服务端口实体
type StatefulSetServicePortEntity struct {
	Name       string `json:"name"`        // 端口名称
	Protocol   string `json:"protocol"`    // 协议
	Port       int32  `json:"port"`        // 端口
	TargetPort string `json:"target_port"` // 目标端口
	NodePort   int32  `json:"node_port"`   // 节点端口
}

// StatefulSetScaleResponse StatefulSet扩缩容响应
type StatefulSetScaleResponse struct {
	Name        string `json:"name"`         // StatefulSet名称
	Namespace   string `json:"namespace"`    // 命名空间
	OldReplicas int32  `json:"old_replicas"` // 原副本数
	NewReplicas int32  `json:"new_replicas"` // 新副本数
	Status      string `json:"status"`       // 扩缩容状态
	Message     string `json:"message"`      // 扩缩容消息
	StartTime   string `json:"start_time"`   // 开始时间
	EndTime     string `json:"end_time"`     // 结束时间
}

// StatefulSetRestartResponse StatefulSet重启响应
type StatefulSetRestartResponse struct {
	Name          string   `json:"name"`           // StatefulSet名称
	Namespace     string   `json:"namespace"`      // 命名空间
	Status        string   `json:"status"`         // 重启状态
	Message       string   `json:"message"`        // 重启消息
	RestartedPods []string `json:"restarted_pods"` // 重启的Pod列表
	StartTime     string   `json:"start_time"`     // 开始时间
	EndTime       string   `json:"end_time"`       // 结束时间
}
