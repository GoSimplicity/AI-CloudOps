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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	yamlv2 "go.yaml.in/yaml/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// secretYAMLDocument 与 corev1.Secret 的字段及 JSON 名称一一对应（由测试保证），
// 唯一区别是 data 的值保留原始 JSON 文本：base64 解码前先识别裸占位符，
// 其余值再按 corev1.Secret 相同的规则（encoding/json 的 []byte 解码）解码。
// 不能用嵌入 corev1.Secret 的方式定义：嵌入字段在 sigs.k8s.io/yaml 中会丢失数值转字符串的处理。
type secretYAMLDocument struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Immutable         *bool                      `json:"immutable,omitempty"`
	Data              map[string]json.RawMessage `json:"data,omitempty"`
	StringData        map[string]string          `json:"stringData,omitempty"`
	Type              corev1.SecretType          `json:"type,omitempty"`
}

var (
	errSecretYAMLMultiDoc = errors.New("解析YAML失败: 只支持单个 YAML 文档，请去掉 --- 分隔的其他文档")
	// yaml 库的部分报错会用反引号或 %#v 带出标量值，统一抹掉
	secretYAMLQuotedValue = regexp.MustCompile("`[^`]*`")
)

// sanitizeSecretYAMLError 去掉解析错误中可能回显的值，只保留位置与原因
func sanitizeSecretYAMLError(err error) error {
	msg := err.Error()
	if i := strings.Index(msg, "unsupported map key"); i >= 0 {
		msg = msg[:i] + "unsupported map key（键必须是字符串）"
	}
	msg = secretYAMLQuotedValue.ReplaceAllString(msg, "`...`")
	return fmt.Errorf("解析YAML失败: %s", msg)
}

// checkSingleYAMLDocument 拒绝多文档 YAML（包括末尾多出的空文档 ---）
func checkSingleYAMLDocument(raw []byte) error {
	dec := yamlv2.NewDecoder(bytes.NewReader(raw))
	var first interface{}
	if err := dec.Decode(&first); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return sanitizeSecretYAMLError(err)
	}
	var next interface{}
	if err := dec.Decode(&next); !errors.Is(err, io.EOF) {
		return errSecretYAMLMultiDoc
	}
	return nil
}

// foldSecretFieldName 与 encoding/json 匹配字段名时的大小写折叠规则一致
func foldSecretFieldName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < utf8.RuneSelf {
			if 'a' <= r && r <= 'z' {
				r -= 'a' - 'A'
			}
			b.WriteRune(r)
			continue
		}
		for {
			r2 := unicode.SimpleFold(r)
			if r2 <= r {
				r = r2
				break
			}
			r = r2
		}
		b.WriteRune(r)
	}
	return b.String()
}

// checkFoldedDuplicateKeys 拒绝只差大小写、会被 encoding/json 合并到同一字段的键
func checkFoldedDuplicateKeys(obj map[string]json.RawMessage, prefix string) error {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	seen := make(map[string]string, len(keys))
	for _, k := range keys {
		f := foldSecretFieldName(k)
		if prev, ok := seen[f]; ok {
			return fmt.Errorf("解析YAML失败: 字段 %s%s 与 %s%s 只差大小写，请只保留一个", prefix, prev, prefix, k)
		}
		seen[f] = k
	}
	return nil
}

// checkSecretFieldCaseDuplicates 检查顶层与 metadata 下只差大小写的重复字段
func checkSecretFieldCaseDuplicates(j []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(j, &top); err != nil {
		// 非对象由后续类型化解析报错
		return nil
	}
	if err := checkFoldedDuplicateKeys(top, ""); err != nil {
		return err
	}
	metaName := foldSecretFieldName("metadata")
	for k, v := range top {
		if foldSecretFieldName(k) != metaName {
			continue
		}
		var meta map[string]json.RawMessage
		if err := json.Unmarshal(v, &meta); err != nil {
			continue
		}
		if err := checkFoldedDuplicateKeys(meta, k+"."); err != nil {
			return err
		}
	}
	return nil
}

// parseSecretYAMLTyped 把单文档 YAML 解析为类型化的 corev1.Secret。
// 返回值中 data 里值恰为占位符的键不放入 sec.Data，而是按升序单独返回，供调用方恢复或拒绝。
// 所有错误都不包含 Secret 的值。
func parseSecretYAMLTyped(y string) (*corev1.Secret, []string, error) {
	raw := []byte(y)
	if err := checkSingleYAMLDocument(raw); err != nil {
		return nil, nil, err
	}
	// 严格模式：任意层级完全相同的重复键直接报错（报错只含键名）
	j, err := yaml.YAMLToJSONStrict(raw)
	if err != nil {
		return nil, nil, sanitizeSecretYAMLError(err)
	}
	if err := checkSecretFieldCaseDuplicates(j); err != nil {
		return nil, nil, err
	}
	var doc secretYAMLDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, sanitizeSecretYAMLError(err)
	}
	sec := &corev1.Secret{
		TypeMeta:   doc.TypeMeta,
		ObjectMeta: doc.ObjectMeta,
		Immutable:  doc.Immutable,
		StringData: doc.StringData,
		Type:       doc.Type,
	}
	var placeholders []string
	if doc.Data != nil {
		sec.Data = make(map[string][]byte, len(doc.Data))
		for k, v := range doc.Data {
			var s string
			if json.Unmarshal(v, &s) == nil && s == SecretRedactedPlaceholder {
				placeholders = append(placeholders, k)
				continue
			}
			var b []byte
			if err := json.Unmarshal(v, &b); err != nil {
				return nil, nil, fmt.Errorf("解析YAML失败: data 中键 %s 的值不是合法的 base64 字符串", k)
			}
			sec.Data[k] = b
		}
	}
	sort.Strings(placeholders)
	return sec, placeholders, nil
}

// isSecretPlaceholderValue 判断值去掉首尾空白后是否为占位符
func isSecretPlaceholderValue(v string) bool {
	return strings.TrimSpace(v) == SecretRedactedPlaceholder
}

// checkNoSecretPlaceholder 确认 data（解码后）与 stringData 中没有占位符或其带空白的变体
func checkNoSecretPlaceholder(sec *corev1.Secret) error {
	dataKeys := make([]string, 0, len(sec.Data))
	for k := range sec.Data {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)
	for _, k := range dataKeys {
		if isSecretPlaceholderValue(string(sec.Data[k])) {
			return fmt.Errorf("data 中键 %s 解码后仍为脱敏占位符 %s，请填写真实值", k, SecretRedactedPlaceholder)
		}
	}
	stringKeys := make([]string, 0, len(sec.StringData))
	for k := range sec.StringData {
		stringKeys = append(stringKeys, k)
	}
	sort.Strings(stringKeys)
	for _, k := range stringKeys {
		if isSecretPlaceholderValue(sec.StringData[k]) {
			return fmt.Errorf("stringData 中键 %s 仍为脱敏占位符 %s（或带空白的变体），请填写真实值", k, SecretRedactedPlaceholder)
		}
	}
	return nil
}

// SecretYAMLHasRedactedPlaceholder 判断 YAML 的 data/stringData 中是否仍含脱敏占位符。
// 先解析为类型化的 Secret（字段名大小写变体按 Kubernetes 解码规则归并），
// data 中的裸占位符、解码后为占位符的值、stringData 中带空白的占位符都算。
func SecretYAMLHasRedactedPlaceholder(y string) (bool, error) {
	sec, placeholders, err := parseSecretYAMLTyped(y)
	if err != nil {
		return false, err
	}
	if len(placeholders) > 0 || checkNoSecretPlaceholder(sec) != nil {
		return true, nil
	}
	return false, nil
}

// RestoreRedactedSecretYAML 将 YAML 中仍为占位符的键恢复为集群中的现有值。
// 前端拿到脱敏 YAML 后只改了部分字段再提交时，未改动的键保持原值而不是被占位符覆盖。
// 占位符对应的键在现有 Secret 中不存在、或恢复后仍残留占位符变体时返回错误。
// 有恢复时返回由类型化 Secret 重新生成的 YAML，否则原样返回。
func RestoreRedactedSecretYAML(y string, existing *corev1.Secret) (string, error) {
	sec, placeholders, err := parseSecretYAMLTyped(y)
	if err != nil {
		return "", err
	}
	var existingData map[string][]byte
	var existingStringData map[string]string
	if existing != nil {
		existingData = existing.Data
		existingStringData = existing.StringData
	}
	changed := false

	for _, k := range placeholders {
		var restored []byte
		if old, exists := existingData[k]; exists {
			restored = old
		} else if old, exists := existingStringData[k]; exists {
			restored = []byte(old)
		} else {
			return "", fmt.Errorf("键 %s 仍为脱敏占位符且在现有Secret中不存在，请填写真实值", k)
		}
		if sec.Data == nil {
			sec.Data = map[string][]byte{}
		}
		sec.Data[k] = restored
		changed = true
	}

	stringKeys := make([]string, 0, len(sec.StringData))
	for k := range sec.StringData {
		stringKeys = append(stringKeys, k)
	}
	sort.Strings(stringKeys)
	for _, k := range stringKeys {
		if sec.StringData[k] != SecretRedactedPlaceholder {
			continue
		}
		if old, exists := existingStringData[k]; exists {
			sec.StringData[k] = old
			changed = true
			continue
		}
		old, exists := existingData[k]
		if !exists {
			return "", fmt.Errorf("键 %s 仍为脱敏占位符且在现有Secret中不存在，请填写真实值", k)
		}
		// stringData 写入后会合并到 data，这里直接以 data 形式恢复原值
		delete(sec.StringData, k)
		if sec.Data == nil {
			sec.Data = map[string][]byte{}
		}
		if _, set := sec.Data[k]; !set {
			sec.Data[k] = old
		}
		changed = true
	}

	if err := checkNoSecretPlaceholder(sec); err != nil {
		return "", err
	}
	if !changed {
		return y, nil
	}
	out, err := yaml.Marshal(sec)
	if err != nil {
		return "", fmt.Errorf("生成YAML失败: %w", err)
	}
	return string(out), nil
}
