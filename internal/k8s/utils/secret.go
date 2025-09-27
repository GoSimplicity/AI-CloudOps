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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildSecretFromRequest 从请求构建secret
func BuildSecretFromRequest(req *model.CreateSecretReq) (*corev1.Secret, error) {
	if req == nil {
		return nil, fmt.Errorf("创建请求不能为空")
	}

	// 构建 Secret 对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Type:       corev1.SecretType(req.Type),
		Data:       req.Data,
		StringData: req.StringData,
	}

	// 如果没有指定类型，默认为 Opaque
	if secret.Type == "" {
		secret.Type = corev1.SecretTypeOpaque
	}

	// 设置不可变标志
	if req.Immutable {
		secret.Immutable = &req.Immutable
	}

	return secret, nil
}

// UpdateSecretFromRequest 从更新请求更新 Kubernetes Secret
func UpdateSecretFromRequest(existing *corev1.Secret, req *model.UpdateSecretReq) (*corev1.Secret, error) {
	if existing == nil {
		return nil, fmt.Errorf("现有Secret不能为空")
	}
	if req == nil {
		return nil, fmt.Errorf("更新请求不能为空")
	}

	// 创建一个副本用于更新
	updated := existing.DeepCopy()

	// 更新数据
	if req.Data != nil {
		updated.Data = req.Data
	}
	if req.StringData != nil {
		updated.StringData = req.StringData
	}

	// 更新标签
	if req.Labels != nil {
		updated.Labels = req.Labels
	}

	// 更新注解
	if req.Annotations != nil {
		updated.Annotations = req.Annotations
	}

	return updated, nil
}

// CleanSecretForYAML 清理 Secret 对象中的系统字段，用于YAML输出
func CleanSecretForYAML(secret *corev1.Secret) *corev1.Secret {
	cleaned := secret.DeepCopy()

	// 清理 metadata 中的系统字段
	cleaned.ObjectMeta.ResourceVersion = ""
	cleaned.ObjectMeta.UID = ""
	cleaned.ObjectMeta.SelfLink = ""
	cleaned.ObjectMeta.CreationTimestamp = metav1.Time{}
	cleaned.ObjectMeta.Generation = 0
	cleaned.ObjectMeta.ManagedFields = nil

	// 清理状态相关的注解
	if cleaned.Annotations != nil {
		delete(cleaned.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}

	return cleaned
}

// SecretToYAML 将 Secret 转换为 YAML 字符串
func SecretToYAML(sec *corev1.Secret) (string, error) {
	if sec == nil {
		return "", fmt.Errorf("Secret 不能为空")
	}
	clean := CleanSecretForYAML(sec)
	b, err := yaml.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}
	return string(b), nil
}

// YAMLToSecret 将 YAML 反序列化为 Secret
func YAMLToSecret(y string) (*corev1.Secret, error) {
	if y == "" {
		return nil, fmt.Errorf("YAML 字符串不能为空")
	}
	var sec corev1.Secret
	if err := yaml.Unmarshal([]byte(y), &sec); err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}
	return &sec, nil
}

// ValidateSecretData 验证 Secret 数据的有效性
func ValidateSecretData(secretType corev1.SecretType, data map[string][]byte, stringData map[string]string) error {
	switch secretType {
	case corev1.SecretTypeServiceAccountToken:
		// ServiceAccount token 应该包含特定的键
		requiredKeys := []string{"token"}
		for _, key := range requiredKeys {
			if _, exists := data[key]; !exists {
				if _, exists := stringData[key]; !exists {
					return fmt.Errorf("ServiceAccount token Secret 必须包含 %s 键", key)
				}
			}
		}
	case corev1.SecretTypeDockerConfigJson:
		// Docker config 应该包含 .dockerconfigjson 键
		if _, exists := data[".dockerconfigjson"]; !exists {
			if _, exists := stringData[".dockerconfigjson"]; !exists {
				return fmt.Errorf("Docker config Secret 必须包含 .dockerconfigjson 键")
			}
		}
	case corev1.SecretTypeTLS:
		// TLS Secret 应该包含 tls.crt 和 tls.key
		requiredKeys := []string{"tls.crt", "tls.key"}
		for _, key := range requiredKeys {
			if _, exists := data[key]; !exists {
				if _, exists := stringData[key]; !exists {
					return fmt.Errorf("TLS Secret 必须包含 %s 键", key)
				}
			}
		}
	}

	return nil
}
