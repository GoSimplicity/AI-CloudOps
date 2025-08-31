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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// GetNodeTaintsByEffect 根据污点效果获取节点污点
func GetNodeTaintsByEffect(taints []corev1.Taint, effect corev1.TaintEffect) []corev1.Taint {
	var filtered []corev1.Taint
	for _, taint := range taints {
		if taint.Effect == effect {
			filtered = append(filtered, taint)
		}
	}
	return filtered
}

// ValidateTaint 验证污点的有效性
func ValidateTaint(taint *corev1.Taint) error {
	if taint == nil {
		return fmt.Errorf("污点不能为空")
	}
	if taint.Key == "" {
		return fmt.Errorf("污点键不能为空")
	}
	if len(taint.Key) > 253 {
		return fmt.Errorf("污点键长度不能超过253个字符")
	}
	if len(taint.Value) > 63 {
		return fmt.Errorf("污点值长度不能超过63个字符")
	}
	return nil
}

// FindTaintByKey 根据键查找污点
func FindTaintByKey(taints []corev1.Taint, key string) (*corev1.Taint, bool) {
	for i := range taints {
		if taints[i].Key == key {
			return &taints[i], true
		}
	}
	return nil, false
}

// RemoveTaintByKey 根据键移除污点
func RemoveTaintByKey(taints []corev1.Taint, key string) []corev1.Taint {
	var result []corev1.Taint
	for _, taint := range taints {
		if taint.Key != key {
			result = append(result, taint)
		}
	}
	return result
}

// TaintExists 检查污点是否存在
func TaintExists(taints []corev1.Taint, targetTaint corev1.Taint) bool {
	for _, taint := range taints {
		if taint.Key == targetTaint.Key &&
			taint.Value == targetTaint.Value &&
			taint.Effect == targetTaint.Effect {
			return true
		}
	}
	return false
}

// AddOrUpdateTaint 添加或更新污点
func AddOrUpdateTaint(taints []corev1.Taint, newTaint corev1.Taint) []corev1.Taint {
	for i := range taints {
		if taints[i].Key == newTaint.Key {
			taints[i] = newTaint
			return taints
		}
	}
	return append(taints, newTaint)
}

// GetTaintsByKeys 根据键列表获取污点
func GetTaintsByKeys(taints []corev1.Taint, keys []string) []corev1.Taint {
	var result []corev1.Taint
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	for _, taint := range taints {
		if keySet[taint.Key] {
			result = append(result, taint)
		}
	}
	return result
}

// BuildTaintYaml 构建污点YAML字符串
func BuildTaintYaml(taints []model.NodeTaintEntity) (string, error) {
	var k8sTaints []corev1.Taint
	for _, taint := range taints {
		k8sTaint := corev1.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: corev1.TaintEffect(taint.Effect),
		}
		k8sTaints = append(k8sTaints, k8sTaint)
	}

	yamlData, err := yaml.Marshal(k8sTaints)
	if err != nil {
		return "", fmt.Errorf("序列化污点YAML失败: %w", err)
	}

	return string(yamlData), nil
}
