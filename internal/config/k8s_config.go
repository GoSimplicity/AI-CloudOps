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

package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"sync"
)

// K8sConfig K8s相关配置
type K8sConfig struct {
	ResourceDefaults   ResourceDefaults   `yaml:"resource_defaults" json:"resource_defaults"`
	TaintDefaults      TaintDefaults      `yaml:"taint_defaults" json:"taint_defaults"`
	TimeDefaults       TimeDefaults       `yaml:"time_defaults" json:"time_defaults"`
	ValidationRules    ValidationRules    `yaml:"validation_rules" json:"validation_rules"`
	TolerationSettings TolerationSettings `yaml:"toleration_settings" json:"toleration_settings"`
	EffectManagement   EffectManagement   `yaml:"effect_management" json:"effect_management"`
}

// ResourceDefaults 资源默认配置
type ResourceDefaults struct {
	CPU              string `yaml:"cpu" json:"cpu"`
	Memory           string `yaml:"memory" json:"memory"`
	CPURequest       string `yaml:"cpu_request" json:"cpu_request"`
	MemoryRequest    string `yaml:"memory_request" json:"memory_request"`
	PVCSize          string `yaml:"pvc_size" json:"pvc_size"`
	QuotaName        string `yaml:"quota_name" json:"quota_name"`
	LimitRangeName   string `yaml:"limit_range_name" json:"limit_range_name"`
	EphemeralStorage string `yaml:"ephemeral_storage" json:"ephemeral_storage"`
	MockCPUUsage     string `yaml:"mock_cpu_usage" json:"mock_cpu_usage"`
	MockMemoryUsage  string `yaml:"mock_memory_usage" json:"mock_memory_usage"`
	DefaultNamespace string `yaml:"default_namespace" json:"default_namespace"`
}

// TaintDefaults 污点默认配置
type TaintDefaults struct {
	DefaultTolerationTime int64    `yaml:"default_toleration_time" json:"default_toleration_time"`
	MaxTolerationTime     int64    `yaml:"max_toleration_time" json:"max_toleration_time"`
	MinTolerationTime     int64    `yaml:"min_toleration_time" json:"min_toleration_time"`
	EffectPriority        []string `yaml:"effect_priority" json:"effect_priority"`
	DefaultOperator       string   `yaml:"default_operator" json:"default_operator"`
	GracePeriod           int64    `yaml:"grace_period" json:"grace_period"`
}

// TimeDefaults 时间默认配置
type TimeDefaults struct {
	JobTTLAfterFinished       int32 `yaml:"job_ttl_after_finished" json:"job_ttl_after_finished"`
	JobActiveDeadlineSeconds  int64 `yaml:"job_active_deadline_seconds" json:"job_active_deadline_seconds"`
	JobBackoffLimit           int32 `yaml:"job_backoff_limit" json:"job_backoff_limit"`
	CronJobStartingDeadline   int64 `yaml:"cronjob_starting_deadline" json:"cronjob_starting_deadline"`
	CronJobSuccessHistory     int32 `yaml:"cronjob_success_history" json:"cronjob_success_history"`
	CronJobFailedHistory      int32 `yaml:"cronjob_failed_history" json:"cronjob_failed_history"`
	DeploymentRevisionHistory int32 `yaml:"deployment_revision_history" json:"deployment_revision_history"`
	RestartGracePeriod        int64 `yaml:"restart_grace_period" json:"restart_grace_period"`
	EvictionGracePeriod       int64 `yaml:"eviction_grace_period" json:"eviction_grace_period"`
}

// ValidationRules 验证规则配置
type ValidationRules struct {
	EnableStrictValidation   bool     `yaml:"enable_strict_validation" json:"enable_strict_validation"`
	AllowedTaintKeys         []string `yaml:"allowed_taint_keys" json:"allowed_taint_keys"`
	AllowedTaintEffects      []string `yaml:"allowed_taint_effects" json:"allowed_taint_effects"`
	AllowedOperators         []string `yaml:"allowed_operators" json:"allowed_operators"`
	MaxTolerationCount       int      `yaml:"max_toleration_count" json:"max_toleration_count"`
	MaxTaintCount            int      `yaml:"max_taint_count" json:"max_taint_count"`
	RequiredTaintKeys        []string `yaml:"required_taint_keys" json:"required_taint_keys"`
	ForbiddenTaintKeys       []string `yaml:"forbidden_taint_keys" json:"forbidden_taint_keys"`
	MaxTolerationTimeSeconds int64    `yaml:"max_toleration_time_seconds" json:"max_toleration_time_seconds"`
}

// TolerationSettings 容忍度设置
type TolerationSettings struct {
	EnableAutoToleration bool                 `yaml:"enable_auto_toleration" json:"enable_auto_toleration"`
	DefaultTolerations   []DefaultToleration  `yaml:"default_tolerations" json:"default_tolerations"`
	TimeScalingEnabled   bool                 `yaml:"time_scaling_enabled" json:"time_scaling_enabled"`
	ConditionalTimeouts  []ConditionalTimeout `yaml:"conditional_timeouts" json:"conditional_timeouts"`
	PreferredScheduling  bool                 `yaml:"preferred_scheduling" json:"preferred_scheduling"`
}

// DefaultToleration 默认容忍度
type DefaultToleration struct {
	Key               string   `yaml:"key" json:"key"`
	Operator          string   `yaml:"operator" json:"operator"`
	Value             string   `yaml:"value" json:"value"`
	Effect            string   `yaml:"effect" json:"effect"`
	TolerationSeconds *int64   `yaml:"toleration_seconds" json:"toleration_seconds"`
	Priority          int      `yaml:"priority" json:"priority"`
	ApplyToNamespaces []string `yaml:"apply_to_namespaces" json:"apply_to_namespaces"`
}

// ConditionalTimeout 条件超时配置
type ConditionalTimeout struct {
	Condition      string   `yaml:"condition" json:"condition"`
	TimeoutValue   int64    `yaml:"timeout_value" json:"timeout_value"`
	Priority       int      `yaml:"priority" json:"priority"`
	ApplyToEffects []string `yaml:"apply_to_effects" json:"apply_to_effects"`
	Description    string   `yaml:"description" json:"description"`
}

// EffectManagement 效果管理配置
type EffectManagement struct {
	NoSchedule       EffectConfig `yaml:"no_schedule" json:"no_schedule"`
	PreferNoSchedule EffectConfig `yaml:"prefer_no_schedule" json:"prefer_no_schedule"`
	NoExecute        EffectConfig `yaml:"no_execute" json:"no_execute"`
}

// EffectConfig 效果配置
type EffectConfig struct {
	Enabled             bool   `yaml:"enabled" json:"enabled"`
	DefaultWeight       int32  `yaml:"default_weight" json:"default_weight"`
	GracefulHandling    bool   `yaml:"graceful_handling" json:"graceful_handling"`
	MaxEvictionRate     string `yaml:"max_eviction_rate" json:"max_eviction_rate"`
	EvictionTimeout     int64  `yaml:"eviction_timeout" json:"eviction_timeout"`
	RescheduleAttempts  int    `yaml:"reschedule_attempts" json:"reschedule_attempts"`
	MonitoringEnabled   bool   `yaml:"monitoring_enabled" json:"monitoring_enabled"`
	NotificationEnabled bool   `yaml:"notification_enabled" json:"notification_enabled"`
}

var (
	globalConfig *K8sConfig
	configMutex  sync.RWMutex
)

// GetK8sConfig 获取K8s配置
func GetK8sConfig() *K8sConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if globalConfig == nil {
		return getDefaultConfig()
	}
	return globalConfig
}

// LoadK8sConfig 加载K8s配置
func LoadK8sConfig(configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	config := getDefaultConfig()

	// 如果配置文件存在，则加载
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("读取配置文件失败: %w", err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return fmt.Errorf("解析配置文件失败: %w", err)
			}
		}
	}

	// 从环境变量覆盖配置
	overrideFromEnv(config)

	// 验证配置
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	globalConfig = config
	return nil
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *K8sConfig {
	return &K8sConfig{
		ResourceDefaults: ResourceDefaults{
			CPU:              "100m",
			Memory:           "128Mi",
			CPURequest:       "10m",
			MemoryRequest:    "64Mi",
			PVCSize:          "1Gi",
			QuotaName:        "compute-quota",
			LimitRangeName:   "resource-limits",
			EphemeralStorage: "1Gi",
			MockCPUUsage:     "100m",
			MockMemoryUsage:  "100Mi",
			DefaultNamespace: "default",
		},
		TaintDefaults: TaintDefaults{
			DefaultTolerationTime: 300,
			MaxTolerationTime:     3600,
			MinTolerationTime:     10,
			EffectPriority:        []string{"NoExecute", "NoSchedule", "PreferNoSchedule"},
			DefaultOperator:       "Equal",
			GracePeriod:           30,
		},
		TimeDefaults: TimeDefaults{
			JobTTLAfterFinished:       3600,
			JobActiveDeadlineSeconds:  86400,
			JobBackoffLimit:           6,
			CronJobStartingDeadline:   60,
			CronJobSuccessHistory:     3,
			CronJobFailedHistory:      1,
			DeploymentRevisionHistory: 10,
			RestartGracePeriod:        30,
			EvictionGracePeriod:       30,
		},
		ValidationRules: ValidationRules{
			EnableStrictValidation:   false,
			AllowedTaintEffects:      []string{"NoSchedule", "PreferNoSchedule", "NoExecute"},
			AllowedOperators:         []string{"Equal", "Exists"},
			MaxTolerationCount:       20,
			MaxTaintCount:            10,
			MaxTolerationTimeSeconds: 7200,
		},
		TolerationSettings: TolerationSettings{
			EnableAutoToleration: false,
			TimeScalingEnabled:   false,
			PreferredScheduling:  true,
		},
		EffectManagement: EffectManagement{
			NoSchedule: EffectConfig{
				Enabled:             true,
				GracefulHandling:    true,
				MonitoringEnabled:   true,
				NotificationEnabled: false,
			},
			PreferNoSchedule: EffectConfig{
				Enabled:             true,
				DefaultWeight:       100,
				MonitoringEnabled:   true,
				NotificationEnabled: false,
			},
			NoExecute: EffectConfig{
				Enabled:             true,
				GracefulHandling:    true,
				MaxEvictionRate:     "25%",
				EvictionTimeout:     300,
				RescheduleAttempts:  3,
				MonitoringEnabled:   true,
				NotificationEnabled: true,
			},
		},
	}
}

// overrideFromEnv 从环境变量覆盖配置
func overrideFromEnv(config *K8sConfig) {
	// 资源默认值
	if val := os.Getenv("K8S_DEFAULT_CPU"); val != "" {
		config.ResourceDefaults.CPU = val
	}
	if val := os.Getenv("K8S_DEFAULT_MEMORY"); val != "" {
		config.ResourceDefaults.Memory = val
	}
	if val := os.Getenv("K8S_DEFAULT_CPU_REQUEST"); val != "" {
		config.ResourceDefaults.CPURequest = val
	}
	if val := os.Getenv("K8S_DEFAULT_MEMORY_REQUEST"); val != "" {
		config.ResourceDefaults.MemoryRequest = val
	}
	if val := os.Getenv("K8S_DEFAULT_PVC_SIZE"); val != "" {
		config.ResourceDefaults.PVCSize = val
	}
	if val := os.Getenv("K8S_QUOTA_NAME"); val != "" {
		config.ResourceDefaults.QuotaName = val
	}
	if val := os.Getenv("K8S_LIMIT_RANGE_NAME"); val != "" {
		config.ResourceDefaults.LimitRangeName = val
	}
	if val := os.Getenv("K8S_DEFAULT_NAMESPACE"); val != "" {
		config.ResourceDefaults.DefaultNamespace = val
	}
	if val := os.Getenv("K8S_EPHEMERAL_STORAGE"); val != "" {
		config.ResourceDefaults.EphemeralStorage = val
	}
	if val := os.Getenv("K8S_MOCK_CPU_USAGE"); val != "" {
		config.ResourceDefaults.MockCPUUsage = val
	}
	if val := os.Getenv("K8S_MOCK_MEMORY_USAGE"); val != "" {
		config.ResourceDefaults.MockMemoryUsage = val
	}

	// 污点默认值
	if val := os.Getenv("K8S_DEFAULT_TOLERATION_TIME"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.TaintDefaults.DefaultTolerationTime = intVal
		}
	}
	if val := os.Getenv("K8S_MAX_TOLERATION_TIME"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.TaintDefaults.MaxTolerationTime = intVal
		}
	}
	if val := os.Getenv("K8S_MIN_TOLERATION_TIME"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.TaintDefaults.MinTolerationTime = intVal
		}
	}
	if val := os.Getenv("K8S_DEFAULT_OPERATOR"); val != "" {
		config.TaintDefaults.DefaultOperator = val
	}
	if val := os.Getenv("K8S_GRACE_PERIOD"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.TaintDefaults.GracePeriod = intVal
		}
	}

	// 时间默认值
	if val := os.Getenv("K8S_JOB_TTL_AFTER_FINISHED"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 32); err == nil {
			config.TimeDefaults.JobTTLAfterFinished = int32(intVal)
		}
	}
	if val := os.Getenv("K8S_JOB_ACTIVE_DEADLINE"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.TimeDefaults.JobActiveDeadlineSeconds = intVal
		}
	}
	if val := os.Getenv("K8S_JOB_BACKOFF_LIMIT"); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 32); err == nil {
			config.TimeDefaults.JobBackoffLimit = int32(intVal)
		}
	}

	// 验证规则
	if val := os.Getenv("K8S_ENABLE_STRICT_VALIDATION"); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			config.ValidationRules.EnableStrictValidation = boolVal
		}
	}
	if val := os.Getenv("K8S_MAX_TOLERATION_COUNT"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			config.ValidationRules.MaxTolerationCount = intVal
		}
	}
	if val := os.Getenv("K8S_MAX_TAINT_COUNT"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			config.ValidationRules.MaxTaintCount = intVal
		}
	}
}

// validateConfig 验证配置
func validateConfig(config *K8sConfig) error {
	// 验证资源默认值
	if config.ResourceDefaults.CPU == "" {
		return fmt.Errorf("CPU默认值不能为空")
	}
	if config.ResourceDefaults.Memory == "" {
		return fmt.Errorf("内存默认值不能为空")
	}

	// 验证污点时间配置
	if config.TaintDefaults.DefaultTolerationTime <= 0 {
		return fmt.Errorf("默认容忍时间必须大于0")
	}
	if config.TaintDefaults.MaxTolerationTime < config.TaintDefaults.MinTolerationTime {
		return fmt.Errorf("最大容忍时间不能小于最小容忍时间")
	}

	// 验证效果优先级
	validEffects := map[string]bool{
		"NoSchedule":       true,
		"PreferNoSchedule": true,
		"NoExecute":        true,
	}
	for _, effect := range config.TaintDefaults.EffectPriority {
		if !validEffects[effect] {
			return fmt.Errorf("无效的污点效果: %s", effect)
		}
	}

	// 验证验证规则
	if config.ValidationRules.MaxTolerationCount <= 0 {
		return fmt.Errorf("最大容忍度数量必须大于0")
	}
	if config.ValidationRules.MaxTaintCount <= 0 {
		return fmt.Errorf("最大污点数量必须大于0")
	}

	return nil
}

// UpdateConfig 更新配置
func UpdateConfig(updateFunc func(*K8sConfig)) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig == nil {
		globalConfig = getDefaultConfig()
	}

	updateFunc(globalConfig)

	// 重新验证配置
	return validateConfig(globalConfig)
}

// SaveConfig 保存配置到文件
func SaveConfig(configPath string) error {
	configMutex.RLock()
	config := globalConfig
	configMutex.RUnlock()

	if config == nil {
		return fmt.Errorf("配置未初始化")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetResourceDefault 获取资源默认值
func GetResourceDefault(resourceType string) string {
	config := GetK8sConfig()
	switch resourceType {
	case "cpu":
		return config.ResourceDefaults.CPU
	case "memory":
		return config.ResourceDefaults.Memory
	case "cpu_request":
		return config.ResourceDefaults.CPURequest
	case "memory_request":
		return config.ResourceDefaults.MemoryRequest
	case "pvc_size":
		return config.ResourceDefaults.PVCSize
	case "ephemeral_storage":
		return config.ResourceDefaults.EphemeralStorage
	case "mock_cpu_usage":
		return config.ResourceDefaults.MockCPUUsage
	case "mock_memory_usage":
		return config.ResourceDefaults.MockMemoryUsage
	case "default_namespace":
		return config.ResourceDefaults.DefaultNamespace
	default:
		return ""
	}
}

// GetTaintDefault 获取污点默认配置
func GetTaintDefault() *TaintDefaults {
	return &GetK8sConfig().TaintDefaults
}

// GetTimeDefault 获取时间默认配置
func GetTimeDefault() *TimeDefaults {
	return &GetK8sConfig().TimeDefaults
}

// GetValidationRules 获取验证规则
func GetValidationRules() *ValidationRules {
	return &GetK8sConfig().ValidationRules
}

// GetTolerationSettings 获取容忍度设置
func GetTolerationSettings() *TolerationSettings {
	return &GetK8sConfig().TolerationSettings
}

// GetEffectManagement 获取效果管理配置
func GetEffectManagement() *EffectManagement {
	return &GetK8sConfig().EffectManagement
}
