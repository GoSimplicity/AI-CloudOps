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

package admin

import (
	"context"
	"fmt"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlTask "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	TaskPending   = "Pending"
	TaskFailed    = "Failed"
	TaskSucceeded = "Succeeded"
)

type YamlTaskService interface {
	// GetYamlTaskList 获取 YAML 任务列表
	GetYamlTaskList(ctx context.Context) ([]*model.K8sYamlTask, error)
	// CreateYamlTask 创建 YAML 任务
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// UpdateYamlTask 更新 YAML 任务
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// DeleteYamlTask 删除 YAML 任务
	DeleteYamlTask(ctx context.Context, id int) error
	// ApplyYamlTask 应用 YAML 任务
	ApplyYamlTask(ctx context.Context, id int) error
}

type yamlTaskService struct {
	yamlTaskDao     admin.YamlTaskDAO
	clusterDao      admin.ClusterDAO
	yamlTemplateDao admin.YamlTemplateDAO
	client          client.K8sClient
	l               *zap.Logger
}

func NewYamlTaskService(yamlTaskDao admin.YamlTaskDAO, clusterDao admin.ClusterDAO, yamlTemplateDao admin.YamlTemplateDAO, client client.K8sClient, l *zap.Logger) YamlTaskService {
	return &yamlTaskService{
		yamlTaskDao:     yamlTaskDao,
		clusterDao:      clusterDao,
		yamlTemplateDao: yamlTemplateDao,
		client:          client,
		l:               l,
	}
}

// GetYamlTaskList 获取 YAML 任务列表
func (y *yamlTaskService) GetYamlTaskList(ctx context.Context) ([]*model.K8sYamlTask, error) {
	return y.yamlTaskDao.ListAllYamlTasks(ctx)
}

// CreateYamlTask 创建 YAML 任务
func (y *yamlTaskService) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if _, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterId); err != nil {
		return fmt.Errorf("YAML 模板不存在: %w", err)
	}

	if _, err := y.clusterDao.GetClusterByID(ctx, task.ClusterId); err != nil {
		return fmt.Errorf("集群不存在: %w", err)
	}

	return y.yamlTaskDao.CreateYamlTask(ctx, task)
}

// UpdateYamlTask 更新 YAML 任务
func (y *yamlTaskService) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if _, err := y.yamlTaskDao.GetYamlTaskByID(ctx, task.ID); err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	if task.TemplateID > 0 {
		if _, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterId); err != nil {
			return fmt.Errorf("YAML 模板不存在: %w", err)
		}
	}

	if task.ClusterId > 0 {
		if _, err := y.clusterDao.GetClusterByID(ctx, task.ClusterId); err != nil {
			return fmt.Errorf("集群不存在: %w", err)
		}
	}

	// 重置任务状态为待处理，并清空应用结果
	task.Status = TaskPending
	task.ApplyResult = ""

	return y.yamlTaskDao.UpdateYamlTask(ctx, task)
}

// DeleteYamlTask 删除 YAML 任务
func (y *yamlTaskService) DeleteYamlTask(ctx context.Context, id int) error {
	return y.yamlTaskDao.DeleteYamlTask(ctx, id)
}

// ApplyYamlTask 应用 YAML 任务
func (y *yamlTaskService) ApplyYamlTask(ctx context.Context, id int) error {
	task, err := y.yamlTaskDao.GetYamlTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	dynClient, err := pkg.GetDynamicClient(ctx, task.ClusterId, y.clusterDao, y.client)
	if err != nil {
		y.l.Error("获取动态客户端失败", zap.Error(err))
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	taskTemplate, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID, task.ClusterId)
	if err != nil {
		y.l.Error("获取 YAML 模板失败", zap.Error(err))
		return fmt.Errorf("获取 YAML 模板失败: %w", err)
	}

	// 变量替换处理
	yamlContent := taskTemplate.Content
	for _, variable := range task.Variables {
		parts := strings.SplitN(variable, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			yamlContent = strings.ReplaceAll(yamlContent, fmt.Sprintf("${%s}", key), value)
		}
	}

	jsonData, err := yamlTask.ToJSON([]byte(yamlContent))
	if err != nil {
		y.l.Error("YAML 转换 JSON 失败", zap.Error(err))
		return fmt.Errorf("YAML 转换 JSON 失败: %w", err)
	}

	obj := &unstructured.Unstructured{}
	if _, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonData, nil, obj); err != nil {
		y.l.Error("解析 JSON 失败", zap.Error(err))
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if obj.GetNamespace() == "" {
		obj.SetNamespace("default")
	}

	// 获取 GVR (GroupVersionResource)
	gvr := schema.GroupVersionResource{
		Group:    obj.GetObjectKind().GroupVersionKind().Group,
		Version:  obj.GetObjectKind().GroupVersionKind().Version,
		Resource: pkg.GetResourceName(obj.GetObjectKind().GroupVersionKind().Kind),
	}

	// 应用资源到集群
	_, err = dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		if k8sErr.IsAlreadyExists(err) {
			y.l.Warn("资源已存在，跳过创建", zap.Error(err))
		} else {
			y.l.Error("应用 YAML 任务失败", zap.Error(err))
			task.Status = TaskFailed
			task.ApplyResult = err.Error()
		}
	} else {
		task.Status = TaskSucceeded
		task.ApplyResult = "应用成功"
	}

	if updateErr := y.yamlTaskDao.UpdateYamlTask(ctx, task); updateErr != nil {
		y.l.Error("更新 YAML 任务状态失败", zap.Error(updateErr))
	}

	return err
}
