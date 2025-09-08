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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// ConvertToK8sNamespace 将名称和标签转换为Kubernetes Namespace对象
func ConvertToK8sNamespace(name string, labels, annotations model.KeyValueList) *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	// 添加标签
	if len(labels) > 0 {
		namespace.Labels = ConvertKeyValueListToLabels(labels)
	}

	// 添加注解
	if len(annotations) > 0 {
		namespace.Annotations = ConvertKeyValueListToLabels(annotations)
	}

	return namespace
}

// BuildNamespaceListOptions 构建Namespace列表选项
func BuildNamespaceListOptions(labelSelector, fieldSelector string) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 添加标签选择器
	if labelSelector != "" {
		options.LabelSelector = labelSelector
	}

	// 添加字段选择器
	if fieldSelector != "" {
		options.FieldSelector = fieldSelector
	}

	return options
}

// FilterNamespacesByStatus 根据状态过滤命名空间
func FilterNamespacesByStatus(namespaces []corev1.Namespace, status string) []corev1.Namespace {
	if status == "" {
		return namespaces
	}

	var filtered []corev1.Namespace
	for _, ns := range namespaces {
		switch status {
		case "Active":
			if IsNamespaceActive(&ns) {
				filtered = append(filtered, ns)
			}
		case "Terminating":
			if IsNamespaceTerminating(&ns) {
				filtered = append(filtered, ns)
			}
		}
	}
	return filtered
}

// FilterNamespacesByLabels 根据标签过滤命名空间
func FilterNamespacesByLabels(namespaces []corev1.Namespace, labels map[string]string) []corev1.Namespace {
	if len(labels) == 0 {
		return namespaces
	}

	var filtered []corev1.Namespace
	for _, ns := range namespaces {
		match := true
		for key, value := range labels {
			if nsValue, exists := ns.Labels[key]; !exists || nsValue != value {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

// GetNamespaceResourceQuota 获取命名空间资源配额信息
func GetNamespaceResourceQuota(namespace *corev1.Namespace) map[string]string {
	if namespace == nil || namespace.Annotations == nil {
		return nil
	}

	quota := make(map[string]string)

	// 检查常见的资源配额注解
	quotaKeys := []string{
		"quota.cpu",
		"quota.memory",
		"quota.pods",
		"quota.storage",
	}

	for _, key := range quotaKeys {
		if value, exists := namespace.Annotations[key]; exists {
			quota[key] = value
		}
	}

	return quota
}
