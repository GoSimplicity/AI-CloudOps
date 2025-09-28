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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

const (
	TaskPending   = "Pending"
	TaskFailed    = "Failed"
	TaskSucceeded = "Succeeded"
)

type YamlTaskService interface {
	GetYamlTaskList(ctx context.Context, req *model.YamlTaskListReq) (model.ListResp[*model.K8sYamlTask], error)
	CreateYamlTask(ctx context.Context, req *model.YamlTaskCreateReq) error
	UpdateYamlTask(ctx context.Context, req *model.YamlTaskUpdateReq) error
	DeleteYamlTask(ctx context.Context, req *model.YamlTaskDeleteReq) error
	ApplyYamlTask(ctx context.Context, req *model.YamlTaskExecuteReq) error
}

type yamlTaskService struct {
	manager manager.YamlManager
	logger  *zap.Logger
}

func NewYamlTaskService(manager manager.YamlManager, logger *zap.Logger) YamlTaskService {
	return &yamlTaskService{
		manager: manager,
		logger:  logger,
	}
}

// GetYamlTaskList 获取 YAML 任务列表
func (y *yamlTaskService) GetYamlTaskList(ctx context.Context, req *model.YamlTaskListReq) (model.ListResp[*model.K8sYamlTask], error) {
	list, err := y.manager.GetYamlTaskList(ctx, req)
	if err != nil {
		return model.ListResp[*model.K8sYamlTask]{}, err
	}
	return model.ListResp[*model.K8sYamlTask]{
		Items: list,
		Total: int64(len(list)),
	}, nil
}

// CreateYamlTask 创建 YAML 任务
func (y *yamlTaskService) CreateYamlTask(ctx context.Context, req *model.YamlTaskCreateReq) error {
	task := &model.K8sYamlTask{
		Name:       req.Name,
		UserID:     req.UserID,
		TemplateID: req.TemplateID,
		ClusterID:  req.ClusterID,
		Variables:  req.Variables,
		Status:     TaskPending,
	}
	return y.manager.CreateYamlTask(ctx, task)
}

// UpdateYamlTask 更新 YAML 任务
func (y *yamlTaskService) UpdateYamlTask(ctx context.Context, req *model.YamlTaskUpdateReq) error {
	// 将请求转换为任务模型
	task := &model.K8sYamlTask{
		Model:      model.Model{ID: req.ID},
		Name:       req.Name,
		UserID:     req.UserID,
		TemplateID: req.TemplateID,
		ClusterID:  req.ClusterID,
		Variables:  req.Variables,
	}
	return y.manager.UpdateYamlTask(ctx, task)
}

// DeleteYamlTask 删除 YAML 任务
func (y *yamlTaskService) DeleteYamlTask(ctx context.Context, req *model.YamlTaskDeleteReq) error {
	return y.manager.DeleteYamlTask(ctx, req.ID)
}

// ApplyYamlTask 应用 YAML 任务
func (y *yamlTaskService) ApplyYamlTask(ctx context.Context, req *model.YamlTaskExecuteReq) error {
	return y.manager.ApplyYamlTask(ctx, req.ID)
}
