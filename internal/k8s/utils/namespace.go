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

package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
)

// ValidateNamespaceName 验证命名空间名称是否符合Kubernetes规范
func ValidateNamespaceName(name string) error {
	if name == "" {
		return fmt.Errorf("命名空间名称不能为空")
	}

	if len(name) > 253 {
		return fmt.Errorf("命名空间名称长度不能超过253个字符")
	}

	// Kubernetes命名空间名称必须符合DNS-1123标准
	dnsPattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !dnsPattern.MatchString(name) {
		return fmt.Errorf("命名空间名称只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾")
	}

	// 检查保留的命名空间名称
	reservedNames := []string{"kube-system", "kube-public", "kube-node-lease", "default"}
	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("不能使用保留的命名空间名称: %s", reserved)
		}
	}

	return nil
}

// ConvertLabelsToKeyValueList 将Kubernetes标签转换为KeyValueList
func ConvertLabelsToKeyValueList(labels map[string]string) model.KeyValueList {
	if labels == nil {
		return model.KeyValueList{}
	}

	var result model.KeyValueList
	for key, value := range labels {
		result = append(result, model.KeyValue{
			Key:   key,
			Value: value,
		})
	}
	return result
}

// ConvertKeyValueListToLabels 将KeyValueList转换为Kubernetes标签
func ConvertKeyValueListToLabels(kvList model.KeyValueList) map[string]string {
	result := make(map[string]string)
	for _, kv := range kvList {
		if strings.TrimSpace(kv.Key) != "" {
			result[kv.Key] = kv.Value
		}
	}
	return result
}

// IsNamespaceActive 检查命名空间是否处于活跃状态
func IsNamespaceActive(namespace *corev1.Namespace) bool {
	return namespace != nil && namespace.Status.Phase == corev1.NamespaceActive
}

// IsNamespaceTerminating 检查命名空间是否正在终止
func IsNamespaceTerminating(namespace *corev1.Namespace) bool {
	return namespace != nil && namespace.Status.Phase == corev1.NamespaceTerminating
}

// GetNamespaceStatus 获取命名空间状态的中文描述
func GetNamespaceStatus(phase corev1.NamespacePhase) string {
	switch phase {
	case corev1.NamespaceActive:
		return "活跃"
	case corev1.NamespaceTerminating:
		return "终止中"
	default:
		return "未知"
	}
}

// ValidateLabels 验证标签是否符合Kubernetes规范
func ValidateLabels(labels model.KeyValueList) error {
	for _, label := range labels {
		if err := validateLabelKey(label.Key); err != nil {
			return fmt.Errorf("标签键 %s 无效: %w", label.Key, err)
		}
		if err := validateLabelValue(label.Value); err != nil {
			return fmt.Errorf("标签值 %s 无效: %w", label.Value, err)
		}
	}
	return nil
}

// validateLabelKey 验证标签键
func validateLabelKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("标签键不能为空")
	}

	// 检查长度限制
	if len(key) > 253 {
		return fmt.Errorf("标签键长度不能超过253个字符")
	}

	// 检查前缀和名称部分
	parts := strings.Split(key, "/")
	if len(parts) > 2 {
		return fmt.Errorf("标签键只能包含一个'/'分隔符")
	}

	if len(parts) == 2 {
		// 有前缀的情况
		prefix, name := parts[0], parts[1]
		if err := validateLabelPrefix(prefix); err != nil {
			return err
		}
		if err := validateLabelName(name); err != nil {
			return err
		}
	} else {
		// 无前缀的情况
		if err := validateLabelName(key); err != nil {
			return err
		}
	}

	return nil
}

// validateLabelPrefix 验证标签前缀
func validateLabelPrefix(prefix string) error {
	if len(prefix) > 253 {
		return fmt.Errorf("标签前缀长度不能超过253个字符")
	}

	// DNS子域名格式
	dnsPattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	if !dnsPattern.MatchString(prefix) {
		return fmt.Errorf("标签前缀必须是有效的DNS子域名")
	}

	return nil
}

// validateLabelName 验证标签名称
func validateLabelName(name string) error {
	if name == "" {
		return fmt.Errorf("标签名称不能为空")
	}

	if len(name) > 63 {
		return fmt.Errorf("标签名称长度不能超过63个字符")
	}

	// 字母数字格式，可以包含连字符、下划线和点
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)
	if !namePattern.MatchString(name) {
		return fmt.Errorf("标签名称只能包含字母、数字、连字符、下划线和点，且必须以字母或数字开头和结尾")
	}

	return nil
}

// validateLabelValue 验证标签值
func validateLabelValue(value string) error {
	if len(value) > 63 {
		return fmt.Errorf("标签值长度不能超过63个字符")
	}

	if value == "" {
		return nil // 空值是允许的
	}

	// 字母数字格式，可以包含连字符、下划线和点
	valuePattern := regexp.MustCompile(`^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)
	if !valuePattern.MatchString(value) {
		return fmt.Errorf("标签值只能包含字母、数字、连字符、下划线和点，且必须以字母或数字开头和结尾")
	}

	return nil
}

// ValidateAnnotations 验证注解是否符合Kubernetes规范
func ValidateAnnotations(annotations model.KeyValueList) error {
	for _, annotation := range annotations {
		if err := validateAnnotationKey(annotation.Key); err != nil {
			return fmt.Errorf("注解键 %s 无效: %w", annotation.Key, err)
		}
		// 注解值没有特殊限制，只需要检查长度
		if len(annotation.Value) > 262144 {
			return fmt.Errorf("注解值长度不能超过262144个字符")
		}
	}
	return nil
}

// validateAnnotationKey 验证注解键（与标签键规则相同）
func validateAnnotationKey(key string) error {
	return validateLabelKey(key)
}

// FilterNamespacesByStatus 根据状态过滤命名空间
func FilterNamespacesByStatus(namespaces []corev1.Namespace, status string) []corev1.Namespace {
	if status == "" || len(namespaces) == 0 {
		return namespaces
	}

	filtered := make([]corev1.Namespace, 0, len(namespaces))
	for _, ns := range namespaces {
		if string(ns.Status.Phase) == status {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

// FilterNamespacesByLabels 根据标签过滤命名空间
func FilterNamespacesByLabels(namespaces []corev1.Namespace, filterLabels model.KeyValueList) []corev1.Namespace {
	if len(filterLabels) == 0 || len(namespaces) == 0 {
		return namespaces
	}

	filtered := make([]corev1.Namespace, 0, len(namespaces))
	for _, ns := range namespaces {
		if hasLabels(ns.Labels, filterLabels) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

// hasLabels 检查命名空间是否包含指定的标签
func hasLabels(nsLabels map[string]string, filterLabels model.KeyValueList) bool {
	for _, filter := range filterLabels {
		value, exists := nsLabels[filter.Key]
		if !exists || value != filter.Value {
			return false
		}
	}
	return true
}
