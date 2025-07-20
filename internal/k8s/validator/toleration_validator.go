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

package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ValidationResult 验证结果
type ValidationResult struct {
	Valid       bool     `json:"valid"`
	Errors      []string `json:"errors"`
	Warnings    []string `json:"warnings"`
	Suggestions []string `json:"suggestions"`
}

// TolerationValidator 容忍度验证器
type TolerationValidator struct {
	config *config.K8sConfig
}

// NewTolerationValidator 创建容忍度验证器
func NewTolerationValidator() *TolerationValidator {
	return &TolerationValidator{
		config: config.GetK8sConfig(),
	}
}

// ValidateToleration 验证单个容忍度
func (v *TolerationValidator) ValidateToleration(toleration *model.K8sToleration) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 验证Key
	if err := v.validateTaintKey(toleration.Key); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("污点键验证失败: %v", err))
	}

	// 验证Operator
	if err := v.validateOperator(toleration.Operator); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("操作符验证失败: %v", err))
	}

	// 验证Effect
	if err := v.validateEffect(toleration.Effect); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("效果验证失败: %v", err))
	}

	// 验证Value（当Operator为Equal时）
	if toleration.Operator == "Equal" && toleration.Value == "" {
		result.Warnings = append(result.Warnings, "当操作符为Equal时，建议设置Value值")
	}

	// 验证TolerationSeconds
	if toleration.TolerationSeconds != nil {
		if err := v.validateTolerationSeconds(*toleration.TolerationSeconds); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("容忍时间验证失败: %v", err))
		}
	} else if toleration.Effect == "NoExecute" {
		result.Warnings = append(result.Warnings, "NoExecute效果建议设置容忍时间")
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("建议设置容忍时间为%d秒", v.config.TaintDefaults.DefaultTolerationTime))
	}

	return result
}

// ValidateTolerationsRequest 验证容忍度请求
func (v *TolerationValidator) ValidateTolerationsRequest(req *model.K8sTaintTolerationRequest) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 验证必填字段
	if req.ClusterID <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "集群ID必须大于0")
	}

	if req.Namespace == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "命名空间不能为空")
	}

	if req.ResourceType == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "资源类型不能为空")
	}

	if req.ResourceName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "资源名称不能为空")
	}

	// 验证容忍度数量
	if len(req.Tolerations) > v.config.ValidationRules.MaxTolerationCount {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("容忍度数量不能超过%d个", v.config.ValidationRules.MaxTolerationCount))
	}

	// 验证每个容忍度
	duplicateKeys := make(map[string]int)
	for i, toleration := range req.Tolerations {
		tolerationResult := v.ValidateToleration(&toleration)
		if !tolerationResult.Valid {
			result.Valid = false
			for _, err := range tolerationResult.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("容忍度[%d]: %s", i, err))
			}
		}
		result.Warnings = append(result.Warnings, tolerationResult.Warnings...)
		result.Suggestions = append(result.Suggestions, tolerationResult.Suggestions...)

		// 检查重复的容忍度
		key := fmt.Sprintf("%s:%s:%s", toleration.Key, toleration.Value, toleration.Effect)
		duplicateKeys[key]++
	}

	// 检查重复项
	for key, count := range duplicateKeys {
		if count > 1 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("发现重复的容忍度配置: %s", key))
		}
	}

	return result
}

// ValidateTaintRequest 验证污点请求
func (v *TolerationValidator) ValidateTaintRequest(req *model.K8sNodeTaintRequest) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 验证必填字段
	if req.ClusterID <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "集群ID必须大于0")
	}

	if req.NodeName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "节点名称不能为空")
	}

	// 验证污点数量
	if len(req.Taints) > v.config.ValidationRules.MaxTaintCount {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("污点数量不能超过%d个", v.config.ValidationRules.MaxTaintCount))
	}

	// 验证每个污点
	duplicateKeys := make(map[string]int)
	for i, taint := range req.Taints {
		taintResult := v.ValidateTaint(&taint)
		if !taintResult.Valid {
			result.Valid = false
			for _, err := range taintResult.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("污点[%d]: %s", i, err))
			}
		}
		result.Warnings = append(result.Warnings, taintResult.Warnings...)
		result.Suggestions = append(result.Suggestions, taintResult.Suggestions...)

		// 检查重复的污点
		key := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, taint.Effect)
		duplicateKeys[key]++
	}

	// 检查重复项
	for key, count := range duplicateKeys {
		if count > 1 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("发现重复的污点配置: %s", key))
		}
	}

	return result
}

// ValidateTaint 验证单个污点
func (v *TolerationValidator) ValidateTaint(taint *model.K8sTaint) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 验证Key
	if err := v.validateTaintKey(taint.Key); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("污点键验证失败: %v", err))
	}

	// 验证Effect
	if err := v.validateEffect(taint.Effect); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("效果验证失败: %v", err))
	}

	// 验证Value格式
	if taint.Value != "" {
		if err := v.validateTaintValue(taint.Value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("污点值验证失败: %v", err))
		}
	}

	return result
}

// ValidateTolerationTimeConfig 验证容忍时间配置
func (v *TolerationValidator) ValidateTolerationTimeConfig(config *model.TolerationTimeConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 验证默认容忍时间
	if config.DefaultTolerationTime != nil {
		if *config.DefaultTolerationTime <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "默认容忍时间必须大于0")
		}
		if *config.DefaultTolerationTime > v.config.ValidationRules.MaxTolerationTimeSeconds {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("默认容忍时间不能超过%d秒", v.config.ValidationRules.MaxTolerationTimeSeconds))
		}
	}

	// 验证最大容忍时间
	if config.MaxTolerationTime != nil {
		if *config.MaxTolerationTime <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "最大容忍时间必须大于0")
		}
		if *config.MaxTolerationTime > v.config.ValidationRules.MaxTolerationTimeSeconds {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("最大容忍时间不能超过%d秒", v.config.ValidationRules.MaxTolerationTimeSeconds))
		}
	}

	// 验证最小容忍时间
	if config.MinTolerationTime != nil {
		if *config.MinTolerationTime <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "最小容忍时间必须大于0")
		}
	}

	// 验证时间范围
	if config.MaxTolerationTime != nil && config.MinTolerationTime != nil {
		if *config.MaxTolerationTime < *config.MinTolerationTime {
			result.Valid = false
			result.Errors = append(result.Errors, "最大容忍时间不能小于最小容忍时间")
		}
	}

	// 验证默认时间在范围内
	if config.DefaultTolerationTime != nil {
		if config.MinTolerationTime != nil && *config.DefaultTolerationTime < *config.MinTolerationTime {
			result.Valid = false
			result.Errors = append(result.Errors, "默认容忍时间不能小于最小容忍时间")
		}
		if config.MaxTolerationTime != nil && *config.DefaultTolerationTime > *config.MaxTolerationTime {
			result.Valid = false
			result.Errors = append(result.Errors, "默认容忍时间不能大于最大容忍时间")
		}
	}

	// 验证时间缩放策略
	if config.TimeScalingPolicy.PolicyType != "" {
		scalingResult := v.validateTimeScalingPolicy(&config.TimeScalingPolicy)
		if !scalingResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, scalingResult.Errors...)
		}
		result.Warnings = append(result.Warnings, scalingResult.Warnings...)
	}

	// 验证条件超时
	for i, timeout := range config.ConditionalTimeouts {
		timeoutResult := v.validateConditionalTimeout(&timeout)
		if !timeoutResult.Valid {
			result.Valid = false
			for _, err := range timeoutResult.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("条件超时[%d]: %s", i, err))
			}
		}
	}

	return result
}

// ValidateResourceValue 验证资源值
func (v *TolerationValidator) ValidateResourceValue(value, resourceType string) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	if value == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("资源值为空，将使用默认值: %s", config.GetResourceDefault(resourceType)))
		return result
	}

	// 验证资源值格式
	_, err := resource.ParseQuantity(value)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("无效的资源值格式: %v", err))
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("建议使用标准格式，如: %s", config.GetResourceDefault(resourceType)))
	}

	return result
}

// validateTaintKey 验证污点键
func (v *TolerationValidator) validateTaintKey(key string) error {
	if key == "" {
		return fmt.Errorf("污点键不能为空")
	}

	// 检查严格验证模式
	if v.config.ValidationRules.EnableStrictValidation {
		// 检查允许的键
		if len(v.config.ValidationRules.AllowedTaintKeys) > 0 {
			allowed := false
			for _, allowedKey := range v.config.ValidationRules.AllowedTaintKeys {
				if key == allowedKey {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("污点键 '%s' 不在允许的列表中", key)
			}
		}

		// 检查禁止的键
		for _, forbiddenKey := range v.config.ValidationRules.ForbiddenTaintKeys {
			if key == forbiddenKey {
				return fmt.Errorf("污点键 '%s' 在禁止的列表中", key)
			}
		}
	}

	// 验证Kubernetes标准格式
	if err := v.validateKubernetesLabelKey(key); err != nil {
		return fmt.Errorf("污点键格式无效: %v", err)
	}

	return nil
}

// validateOperator 验证操作符
func (v *TolerationValidator) validateOperator(operator string) error {
	if operator == "" {
		return fmt.Errorf("操作符不能为空")
	}

	validOperators := v.config.ValidationRules.AllowedOperators
	if len(validOperators) == 0 {
		validOperators = []string{"Equal", "Exists"}
	}

	for _, validOp := range validOperators {
		if operator == validOp {
			return nil
		}
	}

	return fmt.Errorf("无效的操作符 '%s'，支持的操作符: %v", operator, validOperators)
}

// validateEffect 验证效果
func (v *TolerationValidator) validateEffect(effect string) error {
	if effect == "" {
		return fmt.Errorf("效果不能为空")
	}

	validEffects := v.config.ValidationRules.AllowedTaintEffects
	if len(validEffects) == 0 {
		validEffects = []string{"NoSchedule", "PreferNoSchedule", "NoExecute"}
	}

	for _, validEffect := range validEffects {
		if effect == validEffect {
			return nil
		}
	}

	return fmt.Errorf("无效的效果 '%s'，支持的效果: %v", effect, validEffects)
}

// validateTolerationSeconds 验证容忍时间
func (v *TolerationValidator) validateTolerationSeconds(seconds int64) error {
	if seconds <= 0 {
		return fmt.Errorf("容忍时间必须大于0")
	}

	if seconds > v.config.ValidationRules.MaxTolerationTimeSeconds {
		return fmt.Errorf("容忍时间不能超过%d秒", v.config.ValidationRules.MaxTolerationTimeSeconds)
	}

	if seconds < v.config.TaintDefaults.MinTolerationTime {
		return fmt.Errorf("容忍时间不能小于%d秒", v.config.TaintDefaults.MinTolerationTime)
	}

	return nil
}

// validateTaintValue 验证污点值
func (v *TolerationValidator) validateTaintValue(value string) error {
	// Kubernetes标签值的验证规则
	if len(value) > 63 {
		return fmt.Errorf("污点值长度不能超过63个字符")
	}

	// 允许空值
	if value == "" {
		return nil
	}

	// 验证字符集
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)
	if !validPattern.MatchString(value) {
		return fmt.Errorf("污点值包含无效字符，只允许字母、数字、点、破折号和下划线")
	}

	return nil
}

// validateKubernetesLabelKey 验证Kubernetes标签键格式
func (v *TolerationValidator) validateKubernetesLabelKey(key string) error {
	if len(key) > 253 {
		return fmt.Errorf("键长度不能超过253个字符")
	}

	// 检查是否包含域名前缀
	parts := strings.Split(key, "/")
	if len(parts) > 2 {
		return fmt.Errorf("键格式无效，最多只能包含一个'/'分隔符")
	}

	// 验证名称部分
	name := parts[len(parts)-1]
	if name == "" {
		return fmt.Errorf("键的名称部分不能为空")
	}

	if len(name) > 63 {
		return fmt.Errorf("键的名称部分长度不能超过63个字符")
	}

	// 验证名称格式
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)
	if !namePattern.MatchString(name) {
		return fmt.Errorf("键的名称部分格式无效")
	}

	// 验证域名前缀（如果存在）
	if len(parts) == 2 {
		prefix := parts[0]
		if prefix == "" {
			return fmt.Errorf("域名前缀不能为空")
		}

		domainPattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
		if !domainPattern.MatchString(prefix) {
			return fmt.Errorf("域名前缀格式无效")
		}
	}

	return nil
}

// validateTimeScalingPolicy 验证时间缩放策略
func (v *TolerationValidator) validateTimeScalingPolicy(policy *model.TimeScalingPolicy) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	validPolicyTypes := []string{"fixed", "linear", "exponential"}
	validType := false
	for _, validPolicyType := range validPolicyTypes {
		if policy.PolicyType == validPolicyType {
			validType = true
			break
		}
	}
	if !validType {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("无效的策略类型 '%s'，支持的类型: %v", policy.PolicyType, validPolicyTypes))
	}

	if policy.ScalingFactor <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "缩放因子必须大于0")
	}

	if policy.BaseTime != nil && *policy.BaseTime <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "基础时间必须大于0")
	}

	return result
}

// validateConditionalTimeout 验证条件超时
func (v *TolerationValidator) validateConditionalTimeout(timeout *model.ConditionalTimeout) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	if timeout.Condition == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "条件不能为空")
	}

	if timeout.TimeoutValue == nil || *timeout.TimeoutValue <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "超时值必须大于0")
	}

	if timeout.Priority < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "优先级不能为负数")
	}

	return result
}
