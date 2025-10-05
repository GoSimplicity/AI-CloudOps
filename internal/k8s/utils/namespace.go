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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func BuildNamespaceListOptions(req *model.K8sNamespaceListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 添加标签选择器
	if req.LabelSelector != "" {
		options.LabelSelector = req.LabelSelector
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

func GetNamespaceResourceQuota(namespace *corev1.Namespace) map[string]string {
	if namespace == nil || namespace.Annotations == nil {
		return nil
	}

	quota := make(map[string]string)

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

// FilterNamespacesBySearch 根据搜索关键字过滤命名空间
func FilterNamespacesBySearch(namespaces []corev1.Namespace, search string) []corev1.Namespace {
	if search == "" {
		return namespaces
	}

	search = strings.ToLower(search)
	var filtered []corev1.Namespace
	for _, ns := range namespaces {
		// 搜索命名空间名称
		if strings.Contains(strings.ToLower(ns.Name), search) {
			filtered = append(filtered, ns)
			continue
		}

		// 搜索标签
		for key, value := range ns.Labels {
			if strings.Contains(strings.ToLower(key), search) ||
				strings.Contains(strings.ToLower(value), search) {
				filtered = append(filtered, ns)
				break
			}
		}
	}
	return filtered
}

// FilterNamespacesByName 根据命名空间名称过滤
func FilterNamespacesByName(namespaces []corev1.Namespace, nameFilter string) []corev1.Namespace {
	if nameFilter == "" {
		return namespaces
	}

	nameFilter = strings.ToLower(nameFilter)
	var filtered []corev1.Namespace
	for _, ns := range namespaces {
		if strings.Contains(strings.ToLower(ns.Name), nameFilter) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

func BuildNamespaceListPagination(namespaces []corev1.Namespace, page, size int) ([]corev1.Namespace, int64) {
	total := int64(len(namespaces))
	if total == 0 {
		return []corev1.Namespace{}, 0
	}

	// 设置默认分页参数
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []corev1.Namespace{}, total
	}
	if end > total {
		end = total
	}

	return namespaces[start:end], total
}

func ValidateNamespaceFilters(req *model.K8sNamespaceListReq) error {
	if req == nil {
		return nil
	}

	if req.LabelSelector != "" {
		// 可以添加更复杂的标签选择器格式验证
		if strings.Contains(req.LabelSelector, "..") {
			return fmt.Errorf("invalid label selector format")
		}
	}

	return nil
}
