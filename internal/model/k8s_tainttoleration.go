package model

import "time"

// K8sTaintTolerationRequest 污点容忍请求
type K8sTaintTolerationRequest struct {
	ClusterID    int             `json:"cluster_id" binding:"required"`    // 集群ID，必填
	Namespace    string          `json:"namespace" binding:"required"`     // 命名空间，必填
	ResourceType string          `json:"resource_type" binding:"required"` // 资源类型，必填
	ResourceName string          `json:"resource_name" binding:"required"` // 资源名称，必填
	Tolerations  []K8sToleration `json:"tolerations"`                      // 容忍度列表
	NodeTaints   []K8sTaint      `json:"node_taints"`                      // 节点污点列表（用于验证）
	Operation    string          `json:"operation"`                        // 操作类型 (add, update, delete)
}

// K8sToleration 容忍度
type K8sToleration struct {
	Key               string `json:"key"`                // 键
	Operator          string `json:"operator"`           // 操作符 (Exists, Equal)
	Value             string `json:"value"`              // 值
	Effect            string `json:"effect"`             // 效果 (NoSchedule, PreferNoSchedule, NoExecute)
	TolerationSeconds *int64 `json:"toleration_seconds"` // 容忍时间（秒）
}

// K8sTaint 污点
type K8sTaint struct {
	Key    string `json:"key"`    // 键
	Value  string `json:"value"`  // 值
	Effect string `json:"effect"` // 效果 (NoSchedule, PreferNoSchedule, NoExecute)
}

// K8sTaintTolerationResponse 污点容忍响应
type K8sTaintTolerationResponse struct {
	ResourceType      string          `json:"resource_type"`      // 资源类型
	ResourceName      string          `json:"resource_name"`      // 资源名称
	Namespace         string          `json:"namespace"`          // 命名空间
	Tolerations       []K8sToleration `json:"tolerations"`        // 容忍度列表
	CompatibleNodes   []string        `json:"compatible_nodes"`   // 兼容的节点列表
	CreationTimestamp time.Time       `json:"creation_timestamp"` // 创建时间
}

// K8sTaintTolerationValidationRequest 污点容忍验证请求
type K8sTaintTolerationValidationRequest struct {
	ClusterID          int             `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace          string          `json:"namespace"`                     // 命名空间，可选
	Tolerations        []K8sToleration `json:"tolerations"`                   // 容忍度列表
	NodeName           string          `json:"node_name"`                     // 节点名称，可选
	CheckAllNodes      bool            `json:"check_all_nodes"`               // 是否检查所有节点
	SimulateScheduling bool            `json:"simulate_scheduling"`           // 是否模拟调度
}

// K8sTaintTolerationValidationResponse 污点容忍验证响应
type K8sTaintTolerationValidationResponse struct {
	Valid             bool      `json:"valid"`              // 是否有效
	CompatibleNodes   []string  `json:"compatible_nodes"`   // 兼容的节点列表
	IncompatibleNodes []string  `json:"incompatible_nodes"` // 不兼容的节点列表
	ValidationErrors  []string  `json:"validation_errors"`  // 验证错误
	Suggestions       []string  `json:"suggestions"`        // 建议
	SchedulingResult  string    `json:"scheduling_result"`  // 调度结果
	ValidationTime    time.Time `json:"validation_time"`    // 验证时间
}

// K8sNodeTaintRequest 节点污点管理请求
type K8sNodeTaintRequest struct {
	ClusterID int        `json:"cluster_id" binding:"required"` // 集群ID，必填
	NodeName  string     `json:"node_name" binding:"required"`  // 节点名称，必填
	Taints    []K8sTaint `json:"taints"`                        // 污点列表
	Operation string     `json:"operation"`                     // 操作类型 (add, update, delete)
}

// K8sNodeTaintResponse 节点污点管理响应
type K8sNodeTaintResponse struct {
	NodeName      string     `json:"node_name"`      // 节点名称
	Taints        []K8sTaint `json:"taints"`         // 污点列表
	AffectedPods  []string   `json:"affected_pods"`  // 受影响的 Pod 列表
	Operation     string     `json:"operation"`      // 操作类型
	OperationTime time.Time  `json:"operation_time"` // 操作时间
}

// K8sAffinityVisualizationRequest 亲和性可视化请求
type K8sAffinityVisualizationRequest struct {
	ClusterID         int    `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace         string `json:"namespace"`                     // 命名空间，可选
	ResourceType      string `json:"resource_type"`                 // 资源类型，可选
	ResourceName      string `json:"resource_name"`                 // 资源名称，可选
	VisualizationType string `json:"visualization_type"`            // 可视化类型 (node_affinity, pod_affinity, taint_toleration)
	IncludeDetails    bool   `json:"include_details"`               // 是否包含详细信息
}

// K8sAffinityVisualizationResponse 亲和性可视化响应
type K8sAffinityVisualizationResponse struct {
	ClusterID     int                    `json:"cluster_id"`     // 集群ID
	Namespace     string                 `json:"namespace"`      // 命名空间
	Visualization map[string]interface{} `json:"visualization"`  // 可视化数据
	GeneratedTime time.Time              `json:"generated_time"` // 生成时间
}

// K8sNodeRelationship 节点关系
type K8sNodeRelationship struct {
	SourceNode       string            `json:"source_node"`       // 源节点
	TargetNode       string            `json:"target_node"`       // 目标节点
	RelationshipType string            `json:"relationship_type"` // 关系类型
	Labels           map[string]string `json:"labels"`            // 标签
	Taints           []K8sTaint        `json:"taints"`            // 污点
	Strength         float64           `json:"strength"`          // 关系强度
}

// K8sPodRelationship Pod 关系
type K8sPodRelationship struct {
	SourcePod        string            `json:"source_pod"`        // 源 Pod
	TargetPod        string            `json:"target_pod"`        // 目标 Pod
	RelationshipType string            `json:"relationship_type"` // 关系类型 (affinity, anti-affinity)
	TopologyKey      string            `json:"topology_key"`      // 拓扑键
	Labels           map[string]string `json:"labels"`            // 标签
	Weight           int32             `json:"weight"`            // 权重
	Namespace        string            `json:"namespace"`         // 命名空间
}

// K8sTolerationConfigRequest 容忍度配置请求
type K8sTolerationConfigRequest struct {
	ClusterID          int                   `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace          string                `json:"namespace"`                     // 命名空间，可选
	ResourceType       string                `json:"resource_type"`                 // 资源类型，可选
	ResourceName       string                `json:"resource_name"`                 // 资源名称，可选
	TolerationTemplate K8sTolerationTemplate `json:"toleration_template"`           // 容忍度模板
	ApplyToExisting    bool                  `json:"apply_to_existing"`             // 是否应用到现有资源
	AutoUpdate         bool                  `json:"auto_update"`                   // 是否自动更新
	Description        string                `json:"description"`                   // 配置描述
}

// K8sTolerationTemplate 容忍度模板
type K8sTolerationTemplate struct {
	Name                  string              `json:"name"`                    // 模板名称
	Tolerations           []K8sTolerationSpec `json:"tolerations"`             // 容忍度规格列表
	DefaultTolerationTime *int64              `json:"default_toleration_time"` // 默认容忍时间
	EffectPriority        []string            `json:"effect_priority"`         // 效果优先级
	AutoCleanup           bool                `json:"auto_cleanup"`            // 自动清理
	Tags                  map[string]string   `json:"tags"`                    // 标签
}

// K8sTolerationSpec 增强的容忍度规格
type K8sTolerationSpec struct {
	Key               string                `json:"key"`                // 键
	Operator          string                `json:"operator"`           // 操作符 (Exists, Equal)
	Value             string                `json:"value"`              // 值
	Effect            string                `json:"effect"`             // 效果 (NoSchedule, PreferNoSchedule, NoExecute)
	TolerationSeconds *int64                `json:"toleration_seconds"` // 容忍时间（秒）
	Priority          int                   `json:"priority"`           // 优先级
	Conditions        []TolerationCondition `json:"conditions"`         // 容忍条件
	Metadata          map[string]string     `json:"metadata"`           // 元数据
}

// TolerationCondition 容忍条件
type TolerationCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 状态
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// K8sTaintEffectManagementRequest 污点效果管理请求
type K8sTaintEffectManagementRequest struct {
	ClusterID         int                  `json:"cluster_id" binding:"required"` // 集群ID，必填
	NodeName          string               `json:"node_name"`                     // 节点名称，可选
	NodeSelector      map[string]string    `json:"node_selector"`                 // 节点选择器
	TaintEffectConfig K8sTaintEffectConfig `json:"taint_effect_config"`           // 污点效果配置
	BatchOperation    bool                 `json:"batch_operation"`               // 批量操作
	GracePeriod       *int64               `json:"grace_period"`                  // 优雅期限
	ForceEviction     bool                 `json:"force_eviction"`                // 强制驱逐
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
	Enabled            bool               `json:"enabled"`             // 是否启用
	ExceptionPods      []string           `json:"exception_pods"`      // 例外Pod列表
	GracefulHandling   bool               `json:"graceful_handling"`   // 优雅处理
	NotificationConfig NotificationConfig `json:"notification_config"` // 通知配置
}

// PreferNoScheduleConfig PreferNoSchedule效果配置
type PreferNoScheduleConfig struct {
	Enabled           bool   `json:"enabled"`            // 是否启用
	PreferenceWeight  int32  `json:"preference_weight"`  // 偏好权重
	FallbackStrategy  string `json:"fallback_strategy"`  // 回退策略
	MonitoringEnabled bool   `json:"monitoring_enabled"` // 监控启用
}

// NoExecuteConfig NoExecute效果配置
type NoExecuteConfig struct {
	Enabled          bool           `json:"enabled"`           // 是否启用
	EvictionTimeout  *int64         `json:"eviction_timeout"`  // 驱逐超时
	GracefulEviction bool           `json:"graceful_eviction"` // 优雅驱逐
	EvictionPolicy   EvictionPolicy `json:"eviction_policy"`   // 驱逐策略
	RetryConfig      RetryConfig    `json:"retry_config"`      // 重试配置
}

// EffectTransition 效果转换配置
type EffectTransition struct {
	AllowTransition bool             `json:"allow_transition"` // 允许转换
	TransitionRules []TransitionRule `json:"transition_rules"` // 转换规则
	TransitionDelay *int64           `json:"transition_delay"` // 转换延迟
}

// EvictionPolicy 驱逐策略
type EvictionPolicy struct {
	Strategy            string `json:"strategy"`              // 策略 (immediate, graceful, delayed)
	MaxEvictionRate     string `json:"max_eviction_rate"`     // 最大驱逐率
	PodDisruptionBudget string `json:"pod_disruption_budget"` // Pod中断预算
	RescheduleAttempts  int    `json:"reschedule_attempts"`   // 重调度尝试次数
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int      `json:"max_retries"`      // 最大重试次数
	RetryInterval   *int64   `json:"retry_interval"`   // 重试间隔
	BackoffStrategy string   `json:"backoff_strategy"` // 退避策略
	RetryConditions []string `json:"retry_conditions"` // 重试条件
}

// TransitionRule 转换规则
type TransitionRule struct {
	FromEffect string `json:"from_effect"` // 源效果
	ToEffect   string `json:"to_effect"`   // 目标效果
	Condition  string `json:"condition"`   // 条件
	AutoApply  bool   `json:"auto_apply"`  // 自动应用
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled  bool     `json:"enabled"`  // 是否启用
	Channels []string `json:"channels"` // 通知渠道
	Template string   `json:"template"` // 通知模板
	Severity string   `json:"severity"` // 严重程度
}

// K8sTolerationTimeRequest 容忍时间设置请求
type K8sTolerationTimeRequest struct {
	ClusterID        int                  `json:"cluster_id" binding:"required"` // 集群ID，必填
	Namespace        string               `json:"namespace"`                     // 命名空间，可选
	ResourceType     string               `json:"resource_type"`                 // 资源类型，可选
	ResourceName     string               `json:"resource_name"`                 // 资源名称，可选
	TimeConfig       TolerationTimeConfig `json:"time_config"`                   // 时间配置
	GlobalSettings   bool                 `json:"global_settings"`               // 全局设置
	OverrideExisting bool                 `json:"override_existing"`             // 覆盖现有
}

// TolerationTimeConfig 容忍时间配置
type TolerationTimeConfig struct {
	DefaultTolerationTime *int64               `json:"default_toleration_time"` // 默认容忍时间
	MaxTolerationTime     *int64               `json:"max_toleration_time"`     // 最大容忍时间
	MinTolerationTime     *int64               `json:"min_toleration_time"`     // 最小容忍时间
	TimeScalingPolicy     TimeScalingPolicy    `json:"time_scaling_policy"`     // 时间缩放策略
	ConditionalTimeouts   []ConditionalTimeout `json:"conditional_timeouts"`    // 条件超时
	TimeZoneHandling      string               `json:"timezone_handling"`       // 时区处理
}

// TimeScalingPolicy 时间缩放策略
type TimeScalingPolicy struct {
	PolicyType        string   `json:"policy_type"`        // 策略类型 (fixed, linear, exponential)
	ScalingFactor     float64  `json:"scaling_factor"`     // 缩放因子
	BaseTime          *int64   `json:"base_time"`          // 基础时间
	MaxScaledTime     *int64   `json:"max_scaled_time"`    // 最大缩放时间
	ScalingConditions []string `json:"scaling_conditions"` // 缩放条件
}

// ConditionalTimeout 条件超时
type ConditionalTimeout struct {
	Condition      string   `json:"condition"`        // 条件
	TimeoutValue   *int64   `json:"timeout_value"`    // 超时值
	Priority       int      `json:"priority"`         // 优先级
	ApplyToEffects []string `json:"apply_to_effects"` // 应用到效果
}

// K8sTaintEffectManagementResponse 污点效果管理响应
type K8sTaintEffectManagementResponse struct {
	NodeName        string            `json:"node_name"`        // 节点名称
	AffectedPods    []PodEvictionInfo `json:"affected_pods"`    // 受影响的Pod信息
	EffectChanges   []EffectChange    `json:"effect_changes"`   // 效果变化
	EvictionSummary EvictionSummary   `json:"eviction_summary"` // 驱逐摘要
	OperationTime   time.Time         `json:"operation_time"`   // 操作时间
	Status          string            `json:"status"`           // 状态
	Warnings        []string          `json:"warnings"`         // 警告
}

// PodEvictionInfo Pod驱逐信息
type PodEvictionInfo struct {
	PodName            string     `json:"pod_name"`            // Pod名称
	Namespace          string     `json:"namespace"`           // 命名空间
	EvictionReason     string     `json:"eviction_reason"`     // 驱逐原因
	EvictionTime       *time.Time `json:"eviction_time"`       // 驱逐时间
	RescheduleAttempts int        `json:"reschedule_attempts"` // 重调度尝试
	NewNodeName        string     `json:"new_node_name"`       // 新节点名称
	Status             string     `json:"status"`              // 状态
}

// EffectChange 效果变化
type EffectChange struct {
	TaintKey     string    `json:"taint_key"`     // 污点键
	OldEffect    string    `json:"old_effect"`    // 旧效果
	NewEffect    string    `json:"new_effect"`    // 新效果
	ChangeReason string    `json:"change_reason"` // 变化原因
	ChangeTime   time.Time `json:"change_time"`   // 变化时间
}

// EvictionSummary 驱逐摘要
type EvictionSummary struct {
	TotalPods           int     `json:"total_pods"`            // 总Pod数
	EvictedPods         int     `json:"evicted_pods"`          // 已驱逐Pod数
	FailedEvictions     int     `json:"failed_evictions"`      // 失败驱逐数
	PendingEvictions    int     `json:"pending_evictions"`     // 待驱逐数
	RescheduledPods     int     `json:"rescheduled_pods"`      // 重调度Pod数
	AverageEvictionTime float64 `json:"average_eviction_time"` // 平均驱逐时间
}

// K8sTolerationTimeResponse 容忍时间设置响应
type K8sTolerationTimeResponse struct {
	ResourceType      string                 `json:"resource_type"`      // 资源类型
	ResourceName      string                 `json:"resource_name"`      // 资源名称
	Namespace         string                 `json:"namespace"`          // 命名空间
	AppliedTimeouts   []AppliedTimeout       `json:"applied_timeouts"`   // 应用的超时
	ValidationResults []TimeValidationResult `json:"validation_results"` // 验证结果
	CreationTimestamp time.Time              `json:"creation_timestamp"` // 创建时间
	Status            string                 `json:"status"`             // 状态
}

// AppliedTimeout 应用的超时
type AppliedTimeout struct {
	TaintKey         string `json:"taint_key"`         // 污点键
	Effect           string `json:"effect"`            // 效果
	TimeoutValue     *int64 `json:"timeout_value"`     // 超时值
	AppliedCondition string `json:"applied_condition"` // 应用条件
	Source           string `json:"source"`            // 来源
}

// TimeValidationResult 时间验证结果
type TimeValidationResult struct {
	TaintKey           string    `json:"taint_key"`           // 污点键
	IsValid            bool      `json:"is_valid"`            // 是否有效
	ValidationMessage  string    `json:"validation_message"`  // 验证消息
	RecommendedTimeout *int64    `json:"recommended_timeout"` // 推荐超时
	ValidationTime     time.Time `json:"validation_time"`     // 验证时间
}
