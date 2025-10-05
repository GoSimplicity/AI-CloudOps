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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func BuildConfigMapListOptions(labelSelector string) metav1.ListOptions {
	options := metav1.ListOptions{}
	if labelSelector != "" {
		options.LabelSelector = labelSelector
	}
	return options
}

// CleanConfigMapForYAML 清理系统字段，便于导出 YAML
func CleanConfigMapForYAML(cm *corev1.ConfigMap) *corev1.ConfigMap {
	if cm == nil {
		return nil
	}
	cleaned := cm.DeepCopy()
	cleaned.ObjectMeta.ResourceVersion = ""
	cleaned.ObjectMeta.UID = ""
	cleaned.ObjectMeta.SelfLink = ""
	cleaned.ObjectMeta.CreationTimestamp = metav1.Time{}
	cleaned.ObjectMeta.Generation = 0
	cleaned.ObjectMeta.ManagedFields = nil
	return cleaned
}

// ConfigMapToYAML 将 ConfigMap 转换为 YAML 字符串
func ConfigMapToYAML(cm *corev1.ConfigMap) (string, error) {
	if cm == nil {
		return "", fmt.Errorf("ConfigMap 不能为空")
	}
	clean := CleanConfigMapForYAML(cm)
	b, err := yaml.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}
	return string(b), nil
}

// YAMLToConfigMap 将 YAML 反序列化为 ConfigMap
func YAMLToConfigMap(y string) (*corev1.ConfigMap, error) {
	if y == "" {
		return nil, fmt.Errorf("YAML 字符串不能为空")
	}
	var cm corev1.ConfigMap
	if err := yaml.Unmarshal([]byte(y), &cm); err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}
	return &cm, nil
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
