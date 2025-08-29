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

package manager

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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

const (
	TaskPending   = "Pending"
	TaskFailed    = "Failed"
	TaskSucceeded = "Succeeded"
)

// YamlManager YAML模板和任务统一管理接口
type YamlManager interface {
	// ========== YAML 模板管理 ==========
	// GetYamlTemplateList 获取 YAML 模板列表
	GetYamlTemplateList(ctx context.Context, clusterID int) ([]*model.K8sYamlTemplate, error)
	// CreateYamlTemplate 创建 YAML 模板
	CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// CheckYamlTemplate 检查 YAML 模板是否正确
	CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// UpdateYamlTemplate 更新 YAML 模板
	UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// DeleteYamlTemplate 删除 YAML 模板
	DeleteYamlTemplate(ctx context.Context, templateID int, clusterID int) error
	// GetYamlTemplateDetail 获取 YAML 模板详情
	GetYamlTemplateDetail(ctx context.Context, templateID int, clusterID int) (string, error)

	// ========== YAML 任务管理 ==========
	// GetYamlTaskList 获取 YAML 任务列表
	GetYamlTaskList(ctx context.Context) ([]*model.K8sYamlTask, error)
	// CreateYamlTask 创建 YAML 任务
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// UpdateYamlTask 更新 YAML 任务
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// DeleteYamlTask 删除 YAML 任务
	DeleteYamlTask(ctx context.Context, taskID int) error
	// ApplyYamlTask 应用 YAML 任务
	ApplyYamlTask(ctx context.Context, taskID int) error

	// ========== 工具方法 ==========
	// ValidateYamlContent 验证 YAML 内容格式
	ValidateYamlContent(ctx context.Context, content string) error
	// ParseYamlTemplate 解析模板并替换变量
	ParseYamlTemplate(ctx context.Context, templateContent string, variables []string) (string, error)
}

type yamlManager struct {
	yamlTemplateDao dao.YamlTemplateDAO
	yamlTaskDao     dao.YamlTaskDAO
	clusterDao      dao.ClusterDAO
	client          client.K8sClient
	logger          *zap.Logger
}

// NewYamlManager 创建 YamlManager 实例
func NewYamlManager(
	yamlTemplateDao dao.YamlTemplateDAO,
	yamlTaskDao dao.YamlTaskDAO,
	clusterDao dao.ClusterDAO,
	client client.K8sClient,
	logger *zap.Logger,
) YamlManager {
	return &yamlManager{
		yamlTemplateDao: yamlTemplateDao,
		yamlTaskDao:     yamlTaskDao,
		clusterDao:      clusterDao,
		client:          client,
		logger:          logger,
	}
}

// ========== YAML 模板管理实现 ==========

// GetYamlTemplateList 获取 YAML 模板列表
func (ym *yamlManager) GetYamlTemplateList(ctx context.Context, clusterID int) ([]*model.K8sYamlTemplate, error) {
	templates, err := ym.yamlTemplateDao.ListAllYamlTemplates(ctx, clusterID)
	if err != nil {
		ym.logger.Error("获取 YAML 模板列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取 YAML 模板列表失败: %w", err)
	}

	return templates, nil
}

// CreateYamlTemplate 创建 YAML 模板
func (ym *yamlManager) CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式
	if err := ym.ValidateYamlContent(ctx, template.Content); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 创建模板
	if err := ym.yamlTemplateDao.CreateYamlTemplate(ctx, template); err != nil {
		ym.logger.Error("创建 YAML 模板失败",
			zap.String("templateName", template.Name),
			zap.Int("clusterID", template.ClusterId),
			zap.Error(err))
		return fmt.Errorf("创建 YAML 模板失败: %w", err)
	}

	ym.logger.Info("创建 YAML 模板成功",
		zap.String("templateName", template.Name),
		zap.Int("clusterID", template.ClusterId))
	return nil
}

// CheckYamlTemplate 检查 YAML 模板是否正确
func (ym *yamlManager) CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
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

	// YAML 格式验证
	jsonData, err := yaml.ToJSON([]byte(template.Content))
	if err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 解析为 Unstructured 对象
	var obj unstructured.Unstructured
	if err := obj.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("JSON 解析错误: %w", err)
	}

	// 检查必要字段
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" {
		return fmt.Errorf("YAML 内容缺少必要的 apiVersion 或 kind 字段")
	}

	// 获取 Kubernetes 客户端进行验证
	discoveryClient, err := ym.client.GetDiscoveryClient(template.ClusterId)
	if err != nil {
		return fmt.Errorf("获取 discovery client 失败: %w", err)
	}

	// 构建 REST mapper
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

	// 获取动态客户端
	dynamicClient, err := ym.client.GetDynamicClient(template.ClusterId)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	// 构建资源接口
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

// UpdateYamlTemplate 更新 YAML 模板
func (ym *yamlManager) UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式
	if err := ym.ValidateYamlContent(ctx, template.Content); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 更新模板
	if err := ym.yamlTemplateDao.UpdateYamlTemplate(ctx, template); err != nil {
		ym.logger.Error("更新 YAML 模板失败",
			zap.Int("templateID", template.ID),
			zap.String("templateName", template.Name),
			zap.Error(err))
		return fmt.Errorf("更新 YAML 模板失败: %w", err)
	}

	ym.logger.Info("更新 YAML 模板成功",
		zap.Int("templateID", template.ID),
		zap.String("templateName", template.Name))
	return nil
}

// DeleteYamlTemplate 删除 YAML 模板
func (ym *yamlManager) DeleteYamlTemplate(ctx context.Context, templateID int, clusterID int) error {
	// 检查是否有任务正在使用该模板
	tasks, err := ym.yamlTaskDao.GetYamlTaskByTemplateID(ctx, templateID)
	if err != nil {
		ym.logger.Error("检查模板使用情况失败",
			zap.Int("templateID", templateID),
			zap.Error(err))
		return fmt.Errorf("检查模板使用情况失败: %w", err)
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
	if err := ym.yamlTemplateDao.DeleteYamlTemplate(ctx, templateID, clusterID); err != nil {
		ym.logger.Error("删除 YAML 模板失败",
			zap.Int("templateID", templateID),
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return fmt.Errorf("删除 YAML 模板失败: %w", err)
	}

	ym.logger.Info("删除 YAML 模板成功",
		zap.Int("templateID", templateID),
		zap.Int("clusterID", clusterID))
	return nil
}

// GetYamlTemplateDetail 获取 YAML 模板详情
func (ym *yamlManager) GetYamlTemplateDetail(ctx context.Context, templateID int, clusterID int) (string, error) {
	template, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, templateID, clusterID)
	if err != nil {
		ym.logger.Error("获取 YAML 模板详情失败",
			zap.Int("templateID", templateID),
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return "", fmt.Errorf("获取 YAML 模板详情失败: %w", err)
	}

	return template.Content, nil
}

// ========== YAML 任务管理实现 ==========

// GetYamlTaskList 获取 YAML 任务列表
func (ym *yamlManager) GetYamlTaskList(ctx context.Context) ([]*model.K8sYamlTask, error) {
	tasks, err := ym.yamlTaskDao.ListAllYamlTasks(ctx)
	if err != nil {
		ym.logger.Error("获取 YAML 任务列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取 YAML 任务列表失败: %w", err)
	}

	return tasks, nil
}

// CreateYamlTask 创建 YAML 任务
func (ym *yamlManager) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	// 验证模板存在
	if _, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterId); err != nil {
		return fmt.Errorf("YAML 模板不存在: %w", err)
	}

	// 验证集群存在
	if _, err := ym.clusterDao.GetClusterByID(ctx, task.ClusterId); err != nil {
		return fmt.Errorf("集群不存在: %w", err)
	}

	// 创建任务
	if err := ym.yamlTaskDao.CreateYamlTask(ctx, task); err != nil {
		ym.logger.Error("创建 YAML 任务失败",
			zap.String("taskName", task.Name),
			zap.Int("templateID", task.TemplateID),
			zap.Error(err))
		return fmt.Errorf("创建 YAML 任务失败: %w", err)
	}

	ym.logger.Info("创建 YAML 任务成功",
		zap.String("taskName", task.Name),
		zap.Int("templateID", task.TemplateID))
	return nil
}

// UpdateYamlTask 更新 YAML 任务
func (ym *yamlManager) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	// 验证任务存在
	if _, err := ym.yamlTaskDao.GetYamlTaskByID(ctx, task.ID); err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	// 如果更新了模板ID，验证模板存在
	if task.TemplateID > 0 {
		if _, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterId); err != nil {
			return fmt.Errorf("YAML 模板不存在: %w", err)
		}
	}

	// 如果更新了集群ID，验证集群存在
	if task.ClusterId > 0 {
		if _, err := ym.clusterDao.GetClusterByID(ctx, task.ClusterId); err != nil {
			return fmt.Errorf("集群不存在: %w", err)
		}
	}

	// 重置任务状态
	task.Status = TaskPending
	task.ApplyResult = ""

	// 更新任务
	if err := ym.yamlTaskDao.UpdateYamlTask(ctx, task); err != nil {
		ym.logger.Error("更新 YAML 任务失败",
			zap.Int("taskID", task.ID),
			zap.String("taskName", task.Name),
			zap.Error(err))
		return fmt.Errorf("更新 YAML 任务失败: %w", err)
	}

	ym.logger.Info("更新 YAML 任务成功",
		zap.Int("taskID", task.ID),
		zap.String("taskName", task.Name))
	return nil
}

// DeleteYamlTask 删除 YAML 任务
func (ym *yamlManager) DeleteYamlTask(ctx context.Context, taskID int) error {
	if err := ym.yamlTaskDao.DeleteYamlTask(ctx, taskID); err != nil {
		ym.logger.Error("删除 YAML 任务失败",
			zap.Int("taskID", taskID),
			zap.Error(err))
		return fmt.Errorf("删除 YAML 任务失败: %w", err)
	}

	ym.logger.Info("删除 YAML 任务成功", zap.Int("taskID", taskID))
	return nil
}

// ApplyYamlTask 应用 YAML 任务
func (ym *yamlManager) ApplyYamlTask(ctx context.Context, taskID int) error {
	// 获取任务信息
	task, err := ym.yamlTaskDao.GetYamlTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	// 获取模板内容
	template, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterId)
	if err != nil {
		ym.logger.Error("获取 YAML 模板失败",
			zap.Int("taskID", taskID),
			zap.Int("templateID", task.TemplateID),
			zap.Error(err))
		task.Status = TaskFailed
		task.ApplyResult = fmt.Sprintf("获取模板失败: %v", err)
		ym.yamlTaskDao.UpdateYamlTask(ctx, task)
		return fmt.Errorf("获取 YAML 模板失败: %w", err)
	}

	// 解析模板并替换变量
	yamlContent, err := ym.ParseYamlTemplate(ctx, template.Content, task.Variables)
	if err != nil {
		ym.logger.Error("解析 YAML 模板失败",
			zap.Int("taskID", taskID),
			zap.Error(err))
		task.Status = TaskFailed
		task.ApplyResult = fmt.Sprintf("解析模板失败: %v", err)
		ym.yamlTaskDao.UpdateYamlTask(ctx, task)
		return fmt.Errorf("解析 YAML 模板失败: %w", err)
	}

	// 应用 YAML 到集群
	if err := ym.applyYamlToCluster(ctx, task.ClusterId, yamlContent); err != nil {
		ym.logger.Error("应用 YAML 到集群失败",
			zap.Int("taskID", taskID),
			zap.Int("clusterID", task.ClusterId),
			zap.Error(err))
		task.Status = TaskFailed
		task.ApplyResult = fmt.Sprintf("应用失败: %v", err)
		ym.yamlTaskDao.UpdateYamlTask(ctx, task)
		return fmt.Errorf("应用 YAML 到集群失败: %w", err)
	}

	// 更新任务状态为成功
	task.Status = TaskSucceeded
	task.ApplyResult = "应用成功"
	if err := ym.yamlTaskDao.UpdateYamlTask(ctx, task); err != nil {
		ym.logger.Error("更新任务状态失败",
			zap.Int("taskID", taskID),
			zap.Error(err))
	}

	ym.logger.Info("应用 YAML 任务成功",
		zap.Int("taskID", taskID),
		zap.String("taskName", task.Name))
	return nil
}

// ========== 工具方法实现 ==========

// ValidateYamlContent 验证 YAML 内容格式
func (ym *yamlManager) ValidateYamlContent(ctx context.Context, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("YAML 内容不能为空")
	}

	// 验证 YAML 格式
	_, err := yaml.ToJSON([]byte(content))
	if err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	return nil
}

// ParseYamlTemplate 解析模板并替换变量
func (ym *yamlManager) ParseYamlTemplate(ctx context.Context, templateContent string, variables []string) (string, error) {
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

// applyYamlToCluster 应用 YAML 到 Kubernetes 集群
func (ym *yamlManager) applyYamlToCluster(ctx context.Context, clusterID int, yamlContent string) error {
	// 转换 YAML 为 JSON
	jsonData, err := yaml.ToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("YAML 转换 JSON 失败: %w", err)
	}

	// 解析为 Unstructured 对象
	obj := &unstructured.Unstructured{}
	if _, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonData, nil, obj); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 设置默认命名空间
	if obj.GetNamespace() == "" {
		obj.SetNamespace("default")
	}

	// 获取 GVR (GroupVersionResource)
	gvk := obj.GetObjectKind().GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s", // 简单的复数化，实际应该更复杂
	}

	// 获取动态客户端
	dynamicClient, err := ym.client.GetDynamicClient(clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	// 应用资源到集群
	var dr dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		dr = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		dr = dynamicClient.Resource(gvr)
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
