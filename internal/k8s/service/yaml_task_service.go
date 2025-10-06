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
	GetYamlTaskDetail(ctx context.Context, req *model.YamlTaskDetailReq) (*model.K8sYamlTask, error)
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

func (s *yamlTaskService) GetYamlTaskList(ctx context.Context, req *model.YamlTaskListReq) (model.ListResp[*model.K8sYamlTask], error) {
	list, err := s.manager.GetYamlTaskList(ctx, req)
	if err != nil {
		return model.ListResp[*model.K8sYamlTask]{}, err
	}
	return model.ListResp[*model.K8sYamlTask]{
		Items: list,
		Total: int64(len(list)),
	}, nil
}

func (s *yamlTaskService) CreateYamlTask(ctx context.Context, req *model.YamlTaskCreateReq) error {
	task := &model.K8sYamlTask{
		Name:       req.Name,
		UserID:     req.UserID,
		TemplateID: req.TemplateID,
		ClusterID:  req.ClusterID,
		Variables:  req.Variables,
		Status:     TaskPending,
	}
	return s.manager.CreateYamlTask(ctx, task)
}

func (s *yamlTaskService) UpdateYamlTask(ctx context.Context, req *model.YamlTaskUpdateReq) error {
	// 将请求转换为任务模型
	task := &model.K8sYamlTask{
		Model:      model.Model{ID: req.ID},
		Name:       req.Name,
		UserID:     req.UserID,
		TemplateID: req.TemplateID,
		ClusterID:  req.ClusterID,
		Variables:  req.Variables,
	}
	return s.manager.UpdateYamlTask(ctx, task)
}

func (s *yamlTaskService) DeleteYamlTask(ctx context.Context, req *model.YamlTaskDeleteReq) error {
	return s.manager.DeleteYamlTask(ctx, req.ID, req.ClusterID)
}

func (s *yamlTaskService) ApplyYamlTask(ctx context.Context, req *model.YamlTaskExecuteReq) error {
	return s.manager.ApplyYamlTask(ctx, req.ID, req.ClusterID, req.DryRun)
}

func (s *yamlTaskService) GetYamlTaskDetail(ctx context.Context, req *model.YamlTaskDetailReq) (*model.K8sYamlTask, error) {
	return s.manager.GetYamlTaskDetail(ctx, req.ID, req.ClusterID)
}
