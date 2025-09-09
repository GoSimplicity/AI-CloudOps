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

// BuildTaintYaml 构建污点YAML字符串 (从NodeTaintEntity)
func BuildTaintYaml(taints []model.NodeTaint) (string, error) {
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

// BuildTaintYamlFromK8sTaints 构建污点YAML字符串 (从corev1.Taint)
func BuildTaintYamlFromK8sTaints(taints []corev1.Taint) (string, error) {
	yamlData, err := yaml.Marshal(taints)
	if err != nil {
		return "", fmt.Errorf("序列化污点YAML失败: %w", err)
	}

	return string(yamlData), nil
}

// ParseTaintYaml 解析污点YAML字符串为 corev1.Taint 切片
func ParseTaintYaml(taintYaml string) (taintsResult []corev1.Taint, returnErr error) {
	// 转换为自定义错误
	defer func() {
		if r := recover(); r != nil {
			returnErr = fmt.Errorf("YAML解析过程中发生错误: %v。请检查YAML格式是否正确，例如:\n- key: \"example-key\"\n  value: \"example-value\"\n  effect: \"NoSchedule\"", r)
		}
	}()

	if taintYaml == "" {
		return nil, fmt.Errorf("YAML数据不能为空")
	}

	// 预处理：检查输入是否包含明显的非YAML字符
	if len(taintYaml) > 0 {
		// 清理可能的控制字符和非打印字符
		cleanedYaml := ""
		for _, r := range taintYaml {
			if r == '\n' || r == '\r' || r == '\t' || r == ' ' || (r >= 32 && r <= 126) || r > 126 {
				cleanedYaml += string(r)
			}
		}
		taintYaml = cleanedYaml
	}

	// 首先尝试解析为 []corev1.Taint
	var taintsToProcess []corev1.Taint
	err := yaml.UnmarshalStrict([]byte(taintYaml), &taintsToProcess)
	if err == nil {
		// 检查解析结果是否为空
		if len(taintsToProcess) == 0 {
			return nil, fmt.Errorf("未找到有效的污点配置")
		}
		// 验证解析结果的有效性
		for i, taint := range taintsToProcess {
			if taint.Key == "" {
				return nil, fmt.Errorf("第%d个污点的键不能为空", i+1)
			}
			if taint.Effect == "" {
				return nil, fmt.Errorf("第%d个污点的效果不能为空", i+1)
			}
			// 验证Effect是否为有效值
			if taint.Effect != corev1.TaintEffectNoSchedule &&
				taint.Effect != corev1.TaintEffectPreferNoSchedule &&
				taint.Effect != corev1.TaintEffectNoExecute {
				return nil, fmt.Errorf("第%d个污点的效果无效: %s，支持的效果: NoSchedule, PreferNoSchedule, NoExecute", i+1, string(taint.Effect))
			}
		}
		return taintsToProcess, nil
	}

	// 如果直接解析失败，尝试解析为自定义结构
	var customTaints []struct {
		Key    string `yaml:"key" json:"key"`
		Value  string `yaml:"value" json:"value"`
		Effect string `yaml:"effect" json:"effect"`
	}

	customErr := yaml.UnmarshalStrict([]byte(taintYaml), &customTaints)
	if customErr != nil {
		// 尝试更宽松的解析来提供更好的错误信息
		looseTaints := make([]map[string]interface{}, 0)
		looseErr := yaml.Unmarshal([]byte(taintYaml), &looseTaints)
		if looseErr != nil {
			// 检查常见的格式错误
			if len(taintYaml) < 50 { // 对于短字符串，显示完整内容
				return nil, fmt.Errorf("YAML格式错误，无法解析输入: '%s'。请确保YAML格式正确，例如:\n- key: \"example-key\"\n  value: \"example-value\"\n  effect: \"NoSchedule\"", taintYaml)
			} else {
				return nil, fmt.Errorf("YAML格式错误: %s。请确保YAML格式正确，例如:\n- key: \"example-key\"\n  value: \"example-value\"\n  effect: \"NoSchedule\"", looseErr.Error())
			}
		}

		// 如果宽松解析成功，但严格解析失败，说明格式有问题
		missingFields := []string{}
		if len(looseTaints) > 0 {
			for i, item := range looseTaints {
				if _, hasKey := item["key"]; !hasKey {
					missingFields = append(missingFields, fmt.Sprintf("第%d个污点缺少'key'字段", i+1))
				}
				if _, hasEffect := item["effect"]; !hasEffect {
					missingFields = append(missingFields, fmt.Sprintf("第%d个污点缺少'effect'字段", i+1))
				}
			}
		}

		if len(missingFields) > 0 {
			return nil, fmt.Errorf("YAML格式不完整: %s。请确保每个污点都包含 key、value 和 effect 字段", strings.Join(missingFields, "; "))
		}

		return nil, fmt.Errorf("YAML格式不完整或字段类型错误。请确保每个污点都包含正确的 key、value 和 effect 字段")
	}

	// 验证解析结果
	if len(customTaints) == 0 {
		return nil, fmt.Errorf("未找到有效的污点配置")
	}

	// 将自定义结构转换为 corev1.Taint
	taintsToProcess = make([]corev1.Taint, 0, len(customTaints))
	for i, customTaint := range customTaints {
		if customTaint.Key == "" {
			return nil, fmt.Errorf("第%d个污点的键不能为空", i+1)
		}
		if customTaint.Effect == "" {
			return nil, fmt.Errorf("第%d个污点的效果不能为空", i+1)
		}

		// 验证污点效果的有效性
		var effect corev1.TaintEffect
		switch customTaint.Effect {
		case "NoSchedule":
			effect = corev1.TaintEffectNoSchedule
		case "PreferNoSchedule":
			effect = corev1.TaintEffectPreferNoSchedule
		case "NoExecute":
			effect = corev1.TaintEffectNoExecute
		default:
			return nil, fmt.Errorf("第%d个污点的效果无效: %s，支持的效果: NoSchedule, PreferNoSchedule, NoExecute", i+1, customTaint.Effect)
		}

		taint := corev1.Taint{
			Key:    customTaint.Key,
			Value:  customTaint.Value,
			Effect: effect,
		}
		taintsToProcess = append(taintsToProcess, taint)
	}

	return taintsToProcess, nil
}
