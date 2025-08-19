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

package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	yamlTask "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

type YamlTemplateService interface {
	// GetYamlTemplateList 获取 YAML 模板列表
	GetYamlTemplateList(ctx context.Context, clusterId int) ([]*model.K8sYamlTemplate, error)
	// CreateYamlTemplate 创建 YAML 模板
	CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// CheckYamlTemplate 检查 YAML 模板是否正确
	CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// UpdateYamlTemplate 更新 YAML 模板
	UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// DeleteYamlTemplate 删除 YAML 模板
	DeleteYamlTemplate(ctx context.Context, id int, clusterId int) error
	// GetYamlTemplateDetail 获取 YAML 模板详情
	GetYamlTemplateDetail(ctx context.Context, id int, clusterId int) (string, error)
}

type yamlTemplateService struct {
	yamlTemplateDao dao.YamlTemplateDAO
	yamlTaskDao     dao.YamlTaskDAO
	client          client.K8sClient
	l               *zap.Logger
}

func NewYamlTemplateService(yamlTemplateDao dao.YamlTemplateDAO, yamlTaskDao dao.YamlTaskDAO, client client.K8sClient, l *zap.Logger) YamlTemplateService {
	return &yamlTemplateService{
		yamlTemplateDao: yamlTemplateDao,
		yamlTaskDao:     yamlTaskDao,
		client:          client,
		l:               l,
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
func (y *yamlTemplateService) GetYamlTemplateList(ctx context.Context, clusterId int) ([]*model.K8sYamlTemplate, error) {
	return y.yamlTemplateDao.ListAllYamlTemplates(ctx, clusterId)
}

// CreateYamlTemplate 创建 YAML 模板
func (y *yamlTemplateService) CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式是否正确
	if _, err := yamlTask.ToJSON([]byte(template.Content)); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	return y.yamlTemplateDao.CreateYamlTemplate(ctx, template)
}

// CheckYamlTemplate 检查 YAML 模板是否正确
func (y *yamlTemplateService) CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 基础校验
	if template == nil {
		return fmt.Errorf("模板不能为空")
	}
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	if strings.TrimSpace(template.Content) == "" {
		return fmt.Errorf("模板内容不能为空")
	}

	jsonData, err := yaml.ToJSON([]byte(template.Content))
	if err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 解析 JSON 数据到 Unstructured 对象
	var obj unstructured.Unstructured
	if err := obj.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("JSON 解析错误: %w", err)
	}

	// 获取 GVK
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" {
		return fmt.Errorf("YAML 内容缺少必要的 apiVersion 或 kind 字段")
	}

	// 获取 discovery client
	discoveryClient, err := y.client.GetDiscoveryClient(template.ClusterId)
	if err != nil {
		return fmt.Errorf("获取 discovery client 失败: %w", err)
	}

	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("获取 API 组资源失败: %w", err)
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

	// 获取资源的 GVR（Group-Version-Resource）
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("无法找到对应的资源: %w", err)
	}

	// 获取动态客户端
	dynamicClient, err := y.client.GetDynamicClient(template.ClusterId)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			return fmt.Errorf("命名空间缺失但资源需要命名空间")
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

// UpdateYamlTemplate 更新 YAML 模板
func (y *yamlTemplateService) UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式是否正确
	if _, err := yamlTask.ToJSON([]byte(template.Content)); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 更新模板
	return y.yamlTemplateDao.UpdateYamlTemplate(ctx, template)
}

// DeleteYamlTemplate 删除 YAML 模板
func (y *yamlTemplateService) DeleteYamlTemplate(ctx context.Context, id int, clusterId int) error {
	// 检查是否有任务正在使用该模板
	tasks, err := y.yamlTaskDao.GetYamlTaskByTemplateID(ctx, id)
	if err != nil {
		return err
	}

	// 如果有任务使用该模板，返回错误
	if len(tasks) > 0 {
		taskNames := make([]string, len(tasks))
		for i, task := range tasks {
			taskNames[i] = task.Name
		}
		return fmt.Errorf("该模板正在被以下任务使用: %v, 删除失败", taskNames)
	}

	// 删除模板
	return y.yamlTemplateDao.DeleteYamlTemplate(ctx, id, clusterId)
}

func (y *yamlTemplateService) GetYamlTemplateDetail(ctx context.Context, id int, clusterId int) (string, error) {
	content, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, id, clusterId)
	if err != nil {
		y.l.Error("GetYamlTemplateDetail 查询Yaml模板失败", zap.Int("yamlID", id), zap.Error(err))
		return "", err
	}

	return content.Content, nil
}
