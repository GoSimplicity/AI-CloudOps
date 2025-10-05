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
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func ValidateYamlContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("YAML 内容不能为空")
	}

	_, err := yaml.ToJSON([]byte(content))
	if err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	return nil
}

func ParseYamlTemplate(templateContent string, variables []string) (string, error) {
	yamlContent := templateContent

	// 变量替换处理
	for _, variable := range variables {
		parts := strings.SplitN(variable, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			yamlContent = strings.ReplaceAll(yamlContent, fmt.Sprintf("${%s}", key), value)
		}
	}

	return yamlContent, nil
}

func ApplyYamlToCluster(ctx context.Context, discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, yamlContent string) error {
	// 分割多文档YAML
	documents := strings.Split(yamlContent, "---")

	// 获取 REST Mapper（复用）
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("获取 API 组资源失败: %w", err)
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

	var errors []string

	// 遍历每个文档
	for i, document := range documents {
		// 跳过空文档
		doc := strings.TrimSpace(document)
		if doc == "" {
			continue
		}

		if err := ApplySingleDocument(ctx, doc, restMapper, dynamicClient, i); err != nil {
			errors = append(errors, fmt.Sprintf("文档 %d: %v", i+1, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("应用YAML失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

func ApplySingleDocument(ctx context.Context, document string, restMapper meta.RESTMapper, dynamicClient dynamic.Interface, docIndex int) error {

	jsonData, err := yaml.ToJSON([]byte(document))
	if err != nil {
		return fmt.Errorf("YAML 转换 JSON 失败: %w", err)
	}

	// 解析为 Unstructured 对象
	obj := &unstructured.Unstructured{}
	if _, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonData, nil, obj); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 获取 GVK (GroupVersionKind)
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" {
		return fmt.Errorf("YAML 内容缺少必要的 apiVersion 或 kind 字段")
	}

	// 获取正确的资源映射
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("无法找到对应的资源: %w", err)
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace && obj.GetNamespace() == "" {
		obj.SetNamespace("default")
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dynamicClient.Resource(mapping.Resource)
	}

	// 尝试创建资源
	_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		// 如果创建失败，尝试更新
		_, updateErr := dr.Update(ctx, obj, metav1.UpdateOptions{})
		if updateErr != nil {
			return fmt.Errorf("创建或更新资源失败: create error: %v, update error: %v", err, updateErr)
		}
	}

	return nil
}

func ValidateYamlWithCluster(ctx context.Context, discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, yamlContent string) error {
	// 基础格式验证
	if err := ValidateYamlContent(yamlContent); err != nil {
		return err
	}

	jsonData, err := yaml.ToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 解析为 Unstructured 对象
	var obj unstructured.Unstructured
	if err := obj.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("JSON 解析错误: %w", err)
	}

	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" {
		return fmt.Errorf("YAML 内容缺少必要的 apiVersion 或 kind 字段")
	}

	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("获取 API 组资源失败: %w", err)
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

	// 获取资源映射
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("无法找到对应的资源: %w", err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			return fmt.Errorf("命名空间资源缺少 namespace 字段")
		}
		dr = dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dynamicClient.Resource(mapping.Resource)
	}

	// 执行 Dry-Run 验证
	_, err = dr.Create(ctx, &obj, metav1.CreateOptions{
		DryRun: []string{metav1.DryRunAll},
	})
	if err != nil {
		return fmt.Errorf("YAML 校验失败: %w", err)
	}

	return nil
}

// SplitYamlDocuments 分割多文档YAML
func SplitYamlDocuments(yamlContent string) []string {
	documents := strings.Split(yamlContent, "---")
	var validDocuments []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc != "" {
			validDocuments = append(validDocuments, doc)
		}
	}

	return validDocuments
}

// HasMultipleDocuments 检查是否为多文档YAML
func HasMultipleDocuments(yamlContent string) bool {
	documents := SplitYamlDocuments(yamlContent)
	return len(documents) > 1
}
