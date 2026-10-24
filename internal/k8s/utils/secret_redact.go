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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	// SecretRedactedPlaceholder YAML 中替换 data/stringData 值的占位符。
	// 它不是合法的 base64，因此不可能被当作真实值静默写回集群。
	SecretRedactedPlaceholder = "<redacted>"
	// LastAppliedConfigAnnotation kubectl apply 写入的注解，内容为完整对象（含 Secret 明文）
	LastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"
)

// SecretDataKeyInfos 返回 data 的键与值字节数（按 key 升序），不包含值
func SecretDataKeyInfos(data map[string][]byte) []model.SecretKeyInfo {
	infos := make([]model.SecretKeyInfo, 0, len(data))
	for k, v := range data {
		infos = append(infos, model.SecretKeyInfo{Key: k, Size: len(v)})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Key < infos[j].Key })
	return infos
}

// SecretStringDataKeyInfos 返回 stringData 的键与值字节数（按 key 升序），不包含值
func SecretStringDataKeyInfos(data map[string]string) []model.SecretKeyInfo {
	infos := make([]model.SecretKeyInfo, 0, len(data))
	for k, v := range data {
		infos = append(infos, model.SecretKeyInfo{Key: k, Size: len(v)})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Key < infos[j].Key })
	return infos
}

// RedactSecretData 仅保留键，值置为空
func RedactSecretData(data map[string][]byte) model.BinaryDataMap {
	if data == nil {
		return nil
	}
	out := make(model.BinaryDataMap, len(data))
	for k := range data {
		out[k] = []byte{}
	}
	return out
}

// RedactSecretStringData 仅保留键，值置为空
func RedactSecretStringData(data map[string]string) map[string]string {
	if data == nil {
		return nil
	}
	out := make(map[string]string, len(data))
	for k := range data {
		out[k] = ""
	}
	return out
}

// RedactSecretAnnotations 复制注解并去掉可能包含明文的 last-applied-configuration
func RedactSecretAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}
	out := make(map[string]string, len(annotations))
	for k, v := range annotations {
		if k == LastAppliedConfigAnnotation {
			continue
		}
		out[k] = v
	}
	return out
}

// redactedSecretYAMLObject 将清理后的 Secret 转为通用对象，并把 data/stringData 的值替换为占位符
func redactedSecretYAMLObject(sec *corev1.Secret) (map[string]interface{}, error) {
	clean := CleanSecretForYAML(sec)
	raw, err := json.Marshal(clean)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	for _, field := range []string{"data", "stringData"} {
		values, ok := obj[field].(map[string]interface{})
		if !ok {
			continue
		}
		for k := range values {
			values[k] = SecretRedactedPlaceholder
		}
	}
	return obj, nil
}

// secretYAMLValues 解析 YAML 中的 data 与 stringData（值统一视为字符串）
func secretYAMLValues(obj map[string]interface{}, field string) map[string]interface{} {
	values, _ := obj[field].(map[string]interface{})
	return values
}

// SecretYAMLHasRedactedPlaceholder 判断 YAML 的 data/stringData 中是否仍含脱敏占位符
func SecretYAMLHasRedactedPlaceholder(y string) (bool, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(y), &obj); err != nil {
		return false, fmt.Errorf("解析YAML失败: %w", err)
	}
	for _, field := range []string{"data", "stringData"} {
		for _, v := range secretYAMLValues(obj, field) {
			if s, ok := v.(string); ok && s == SecretRedactedPlaceholder {
				return true, nil
			}
		}
	}
	return false, nil
}

// RestoreRedactedSecretYAML 将 YAML 中仍为占位符的键恢复为集群中的现有值。
// 前端拿到脱敏 YAML 后只改了部分字段再提交时，未改动的键保持原值而不是被占位符覆盖。
// 占位符对应的键在现有 Secret 中不存在时返回错误。
func RestoreRedactedSecretYAML(y string, existing *corev1.Secret) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(y), &obj); err != nil {
		return "", fmt.Errorf("解析YAML失败: %w", err)
	}
	if obj == nil {
		return y, nil
	}
	var existingData map[string][]byte
	var existingStringData map[string]string
	if existing != nil {
		existingData = existing.Data
		existingStringData = existing.StringData
	}
	changed := false

	data := secretYAMLValues(obj, "data")
	for k, v := range data {
		if s, ok := v.(string); ok && s == SecretRedactedPlaceholder {
			if old, exists := existingData[k]; exists {
				data[k] = base64.StdEncoding.EncodeToString(old)
			} else if old, exists := existingStringData[k]; exists {
				data[k] = base64.StdEncoding.EncodeToString([]byte(old))
			} else {
				return "", fmt.Errorf("键 %s 仍为脱敏占位符且在现有Secret中不存在，请填写真实值", k)
			}
			changed = true
		}
	}

	stringData := secretYAMLValues(obj, "stringData")
	for k, v := range stringData {
		if s, ok := v.(string); ok && s == SecretRedactedPlaceholder {
			if old, exists := existingStringData[k]; exists {
				stringData[k] = old
				changed = true
				continue
			}
			old, exists := existingData[k]
			if !exists {
				return "", fmt.Errorf("键 %s 仍为脱敏占位符且在现有Secret中不存在，请填写真实值", k)
			}
			// stringData 写入后会合并到 data，这里直接以 data 形式恢复原值
			delete(stringData, k)
			if data == nil {
				data = map[string]interface{}{}
				obj["data"] = data
			}
			if _, set := data[k]; !set {
				data[k] = base64.StdEncoding.EncodeToString(old)
			}
			changed = true
		}
	}

	if !changed {
		return y, nil
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("生成YAML失败: %w", err)
	}
	return string(out), nil
}
