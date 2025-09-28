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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

const (
	TaskPending   = "Pending"
	TaskFailed    = "Failed"
	TaskSucceeded = "Succeeded"
)

// YamlManager YAML模板和任务管理
type YamlManager interface {
	GetYamlTemplateList(ctx context.Context, req *model.YamlTemplateListReq) ([]*model.K8sYamlTemplate, error)
	CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	DeleteYamlTemplate(ctx context.Context, templateID int, clusterID int) error
	GetYamlTemplateDetail(ctx context.Context, templateID int, clusterID int) (*model.K8sYamlTemplate, error)
	GetYamlTaskList(ctx context.Context, req *model.YamlTaskListReq) ([]*model.K8sYamlTask, error)
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	DeleteYamlTask(ctx context.Context, taskID int) error
	ApplyYamlTask(ctx context.Context, taskID int) error
}

type yamlManager struct {
	yamlTemplateDao dao.YamlTemplateDAO
	yamlTaskDao     dao.YamlTaskDAO
	clusterDao      dao.ClusterDAO
	client          client.K8sClient
	logger          *zap.Logger
}

// NewYamlManager 创建YamlManager实例
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

// GetYamlTemplateList 获取模板列表
func (ym *yamlManager) GetYamlTemplateList(ctx context.Context, req *model.YamlTemplateListReq) ([]*model.K8sYamlTemplate, error) {
	templates, err := ym.yamlTemplateDao.ListAllYamlTemplates(ctx, req)
	if err != nil {
		ym.logger.Error("获取 YAML 模板列表失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取 YAML 模板列表失败: %w", err)
	}

	return templates, nil
}

// CreateYamlTemplate 创建模板
func (ym *yamlManager) CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式
	if err := utils.ValidateYamlContent(template.Content); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 创建模板
	if err := ym.yamlTemplateDao.CreateYamlTemplate(ctx, template); err != nil {
		ym.logger.Error("创建 YAML 模板失败",
			zap.String("templateName", template.Name),
			zap.Int("clusterID", template.ClusterID),
			zap.Error(err))
		return fmt.Errorf("创建 YAML 模板失败: %w", err)
	}

	ym.logger.Info("创建 YAML 模板成功",
		zap.String("templateName", template.Name),
		zap.Int("clusterID", template.ClusterID))
	return nil
}

// CheckYamlTemplate 检查模板格式
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

	// 获取 Kubernetes 客户端进行验证
	discoveryClient, err := ym.client.GetDiscoveryClient(template.ClusterID)
	if err != nil {
		return fmt.Errorf("获取 discovery client 失败: %w", err)
	}

	dynamicClient, err := ym.client.GetDynamicClient(template.ClusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	// 使用工具方法验证YAML
	return utils.ValidateYamlWithCluster(ctx, discoveryClient, dynamicClient, template.Content)
}

// UpdateYamlTemplate 更新模板
func (ym *yamlManager) UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式
	if err := utils.ValidateYamlContent(template.Content); err != nil {
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

// DeleteYamlTemplate 删除模板
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

// GetYamlTemplateDetail 获取模板详情
func (ym *yamlManager) GetYamlTemplateDetail(ctx context.Context, templateID int, clusterID int) (*model.K8sYamlTemplate, error) {
	template, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, templateID, clusterID)
	if err != nil {
		ym.logger.Error("获取 YAML 模板详情失败",
			zap.Int("templateID", templateID),
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取 YAML 模板详情失败: %w", err)
	}

	return template, nil
}

// GetYamlTaskList 获取任务列表
func (ym *yamlManager) GetYamlTaskList(ctx context.Context, req *model.YamlTaskListReq) ([]*model.K8sYamlTask, error) {
	tasks, err := ym.yamlTaskDao.ListAllYamlTasks(ctx, req)
	if err != nil {
		ym.logger.Error("获取 YAML 任务列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取 YAML 任务列表失败: %w", err)
	}

	return tasks, nil
}

// CreateYamlTask 创建任务
func (ym *yamlManager) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	// 验证模板存在
	if _, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterID); err != nil {
		return fmt.Errorf("YAML 模板不存在: %w", err)
	}

	// 验证集群存在
	if _, err := ym.clusterDao.GetClusterByID(ctx, task.ClusterID); err != nil {
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

// UpdateYamlTask 更新任务
func (ym *yamlManager) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	// 验证任务存在
	if _, err := ym.yamlTaskDao.GetYamlTaskByID(ctx, task.ID); err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	// 如果更新了模板ID，验证模板存在
	if task.TemplateID > 0 {
		if _, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterID); err != nil {
			return fmt.Errorf("YAML 模板不存在: %w", err)
		}
	}

	// 如果更新了集群ID，验证集群存在
	if task.ClusterID > 0 {
		if _, err := ym.clusterDao.GetClusterByID(ctx, task.ClusterID); err != nil {
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

// DeleteYamlTask 删除任务
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

// ApplyYamlTask 应用任务
func (ym *yamlManager) ApplyYamlTask(ctx context.Context, taskID int) error {
	// 获取任务信息
	task, err := ym.yamlTaskDao.GetYamlTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	// 获取模板内容
	template, err := ym.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterID)
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
	yamlContent, err := utils.ParseYamlTemplate(template.Content, task.Variables)
	if err != nil {
		ym.logger.Error("解析 YAML 模板失败",
			zap.Int("taskID", taskID),
			zap.Error(err))
		task.Status = TaskFailed
		task.ApplyResult = fmt.Sprintf("解析模板失败: %v", err)
		ym.yamlTaskDao.UpdateYamlTask(ctx, task)
		return fmt.Errorf("解析 YAML 模板失败: %w", err)
	}

	// 获取Kubernetes客户端
	discoveryClient, err := ym.client.GetDiscoveryClient(task.ClusterID)
	if err != nil {
		ym.logger.Error("获取 discovery client 失败",
			zap.Int("taskID", taskID),
			zap.Int("clusterID", task.ClusterID),
			zap.Error(err))
		task.Status = TaskFailed
		task.ApplyResult = fmt.Sprintf("获取客户端失败: %v", err)
		ym.yamlTaskDao.UpdateYamlTask(ctx, task)
		return fmt.Errorf("获取 discovery client 失败: %w", err)
	}

	dynamicClient, err := ym.client.GetDynamicClient(task.ClusterID)
	if err != nil {
		ym.logger.Error("获取 dynamic client 失败",
			zap.Int("taskID", taskID),
			zap.Int("clusterID", task.ClusterID),
			zap.Error(err))
		task.Status = TaskFailed
		task.ApplyResult = fmt.Sprintf("获取客户端失败: %v", err)
		ym.yamlTaskDao.UpdateYamlTask(ctx, task)
		return fmt.Errorf("获取 dynamic client 失败: %w", err)
	}

	// 应用 YAML 到集群
	if err := utils.ApplyYamlToCluster(ctx, discoveryClient, dynamicClient, yamlContent); err != nil {
		ym.logger.Error("应用 YAML 到集群失败",
			zap.Int("taskID", taskID),
			zap.Int("clusterID", task.ClusterID),
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
