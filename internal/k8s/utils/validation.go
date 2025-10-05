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
)

// Kubernetes资源名称验证正则表达式
var (
	// DNS-1123 subdomain格式：小写字母、数字、'-'，以字母或数字开头和结尾
	kubernetesNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	// DNS-1035 label格式：小写字母、数字、'-'，以字母开头，以字母或数字结尾
	kubernetesLabelRegex = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
)

func ValidateKubernetesName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 253 {
		return fmt.Errorf("name length cannot exceed 253 characters")
	}

	if !kubernetesNameRegex.MatchString(name) {
		return fmt.Errorf("name must consist of lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character")
	}

	return nil
}

func ValidateNamespaceName(name string) error {
	if err := ValidateKubernetesName(name); err != nil {
		return fmt.Errorf("invalid namespace name: %w", err)
	}

	// 命名空间名称长度限制为63字符
	if len(name) > 63 {
		return fmt.Errorf("namespace name length cannot exceed 63 characters")
	}

	reservedNames := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
	}

	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("namespace name '%s' is reserved", name)
		}
	}

	return nil
}

func ValidateLabels(labels []model.KeyValue) error {
	if len(labels) == 0 {
		return nil // 标签是可选的
	}

	for i, label := range labels {
		if err := ValidateLabelKey(label.Key); err != nil {
			return fmt.Errorf("invalid label key at index %d: %w", i, err)
		}

		if err := ValidateLabelValue(label.Value); err != nil {
			return fmt.Errorf("invalid label value at index %d: %w", i, err)
		}
	}

	return nil
}

func ValidateLabelKey(key string) error {
	if key == "" {
		return fmt.Errorf("label key cannot be empty")
	}

	if strings.Contains(key, "/") {
		parts := strings.SplitN(key, "/", 2)
		prefix, name := parts[0], parts[1]

		if err := ValidateLabelPrefix(prefix); err != nil {
			return err
		}

		if err := validateLabelNamePart(name); err != nil {
			return err
		}
	} else {

		if err := validateLabelNamePart(key); err != nil {
			return err
		}
	}

	return nil
}

// validateLabelNamePart 验证标签名称部分
func validateLabelNamePart(name string) error {

	if len(name) > 63 {
		return fmt.Errorf("label key name part length cannot exceed 63 characters")
	}

	if !kubernetesLabelRegex.MatchString(name) && name != "" {
		return fmt.Errorf("label key name part must consist of lowercase alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character")
	}

	return nil
}

func ValidateLabelPrefix(prefix string) error {
	if prefix == "" {
		return nil
	}

	if len(prefix) > 253 {
		return fmt.Errorf("label key prefix length cannot exceed 253 characters")
	}

	// 前缀必须是DNS子域名
	if !kubernetesNameRegex.MatchString(prefix) {
		return fmt.Errorf("label key prefix must be a valid DNS subdomain")
	}

	return nil
}

func ValidateLabelValue(value string) error {
	if len(value) > 63 {
		return fmt.Errorf("label value length cannot exceed 63 characters")
	}

	// 标签值可以为空
	if value == "" {
		return nil
	}

	// 标签值格式验证
	labelValueRegex := regexp.MustCompile(`^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)
	if !labelValueRegex.MatchString(value) {
		return fmt.Errorf("label value must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character")
	}

	return nil
}

func ValidateAnnotations(annotations []model.KeyValue) error {
	if len(annotations) == 0 {
		return nil // 注解是可选的
	}

	for i, annotation := range annotations {
		if err := ValidateAnnotationKey(annotation.Key); err != nil {
			return fmt.Errorf("invalid annotation key at index %d: %w", i, err)
		}

		if err := ValidateAnnotationValue(annotation.Value); err != nil {
			return fmt.Errorf("invalid annotation value at index %d: %w", i, err)
		}
	}

	return nil
}

func ValidateAnnotationKey(key string) error {
	if key == "" {
		return fmt.Errorf("annotation key cannot be empty")
	}

	// 注解键的验证规则与标签键相同
	return ValidateLabelKey(key)
}

func ValidateAnnotationValue(value string) error {
	// 注解值没有长度限制，但建议不要太长
	if len(value) > 262144 { // 256KB
		return fmt.Errorf("annotation value is too long (exceeds 256KB)")
	}

	return nil
}

func ConvertKeyValueListToLabels(keyValues []model.KeyValue) map[string]string {
	if len(keyValues) == 0 {
		return nil
	}

	result := make(map[string]string, len(keyValues))
	for _, kv := range keyValues {
		if kv.Key != "" {
			result[kv.Key] = kv.Value
		}
	}

	return result
}

func ValidateResourceQuota(resources map[string]string) error {
	if len(resources) == 0 {
		return nil
	}

	validResources := map[string]bool{
		"cpu":                    true,
		"memory":                 true,
		"storage":                true,
		"ephemeral-storage":      true,
		"pods":                   true,
		"services":               true,
		"replicationcontrollers": true,
		"resourcequotas":         true,
		"secrets":                true,
		"configmaps":             true,
		"persistentvolumeclaims": true,
		"services.nodeports":     true,
		"services.loadbalancers": true,
	}

	for resource := range resources {
		if !validResources[resource] {
			return fmt.Errorf("invalid resource type: %s", resource)
		}
	}

	return nil
}

func ValidateContainerName(name string) error {
	if name == "" {
		return fmt.Errorf("container name cannot be empty")
	}

	if len(name) > 253 {
		return fmt.Errorf("container name length cannot exceed 253 characters")
	}

	// 容器名称格式验证
	containerNameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !containerNameRegex.MatchString(name) {
		return fmt.Errorf("container name must consist of lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character")
	}

	return nil
}

func ValidateImageName(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// 简单的镜像名称格式验证
	// 格式：[registry/]name[:tag]
	if len(image) > 1024 {
		return fmt.Errorf("image name is too long")
	}

	return nil
}

func ValidateClusterCreateParams(name, apiServerAddr, kubeConfigContent string) error {
	if name == "" {
		return fmt.Errorf("集群名称不能为空")
	}

	if apiServerAddr == "" {
		return fmt.Errorf("API Server 地址不能为空")
	}

	if kubeConfigContent == "" {
		return fmt.Errorf("KubeConfig 内容不能为空")
	}

	return nil
}

func ValidateClusterUpdateParams(id int) error {
	if id <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	return nil
}
