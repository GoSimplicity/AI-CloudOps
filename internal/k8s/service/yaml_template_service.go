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

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"go.uber.org/zap"
)

type YamlTemplateService interface {
	GetYamlTemplateList(ctx context.Context, req *model.YamlTemplateListReq) (model.ListResp[*model.K8sYamlTemplate], error)
	CreateYamlTemplate(ctx context.Context, req *model.YamlTemplateCreateReq) error
	CheckYamlTemplate(ctx context.Context, req *model.YamlTemplateCheckReq) error
	UpdateYamlTemplate(ctx context.Context, req *model.YamlTemplateUpdateReq, super bool) error
	DeleteYamlTemplate(ctx context.Context, req *model.YamlTemplateDeleteReq, super bool) error
	GetYamlTemplateDetail(ctx context.Context, req *model.YamlTemplateDetailReq) (*model.K8sYamlTemplate, error)
}

type yamlTemplateService struct {
	manager manager.YamlManager
	logger  *zap.Logger
}

func NewYamlTemplateService(manager manager.YamlManager, logger *zap.Logger) YamlTemplateService {
	return &yamlTemplateService{
		manager: manager,
		logger:  logger,
	}
}

func (s *yamlTemplateService) GetYamlTemplateList(ctx context.Context, req *model.YamlTemplateListReq) (model.ListResp[*model.K8sYamlTemplate], error) {
	list, err := s.manager.GetYamlTemplateList(ctx, req)
	if err != nil {
		return model.ListResp[*model.K8sYamlTemplate]{}, err
	}
	return model.ListResp[*model.K8sYamlTemplate]{
		Items: list,
		Total: int64(len(list)),
	}, nil
}

func (s *yamlTemplateService) CreateYamlTemplate(ctx context.Context, req *model.YamlTemplateCreateReq) error {
	template := &model.K8sYamlTemplate{
		Name:      req.Name,
		UserID:    req.UserID,
		Content:   req.Content,
		ClusterID: req.ClusterID,
	}
	return s.manager.CreateYamlTemplate(ctx, template)
}

func (s *yamlTemplateService) CheckYamlTemplate(ctx context.Context, req *model.YamlTemplateCheckReq) error {
	template := &model.K8sYamlTemplate{
		Name:      req.Name,
		Content:   req.Content,
		ClusterID: req.ClusterID,
	}
	return s.manager.CheckYamlTemplate(ctx, template)
}

func (s *yamlTemplateService) UpdateYamlTemplate(ctx context.Context, req *model.YamlTemplateUpdateReq, super bool) error {
	if req == nil {
		return base.NewBusinessError(constants.ErrInvalidParam, "更新YAML模板请求不能为空")
	}
	template, err := s.ownedYamlTemplate(ctx, req.ID, req.ClusterID, req.UserID, super)
	if err != nil {
		return err
	}
	updated := &model.K8sYamlTemplate{
		Model:     model.Model{ID: req.ID},
		Name:      req.Name,
		UserID:    template.UserID,
		Content:   req.Content,
		ClusterID: req.ClusterID,
	}
	return s.manager.UpdateYamlTemplate(ctx, updated, template.UserID)
}

func (s *yamlTemplateService) DeleteYamlTemplate(ctx context.Context, req *model.YamlTemplateDeleteReq, super bool) error {
	if req == nil {
		return base.NewBusinessError(constants.ErrInvalidParam, "删除YAML模板请求不能为空")
	}
	template, err := s.ownedYamlTemplate(ctx, req.ID, req.ClusterID, req.UserID, super)
	if err != nil {
		return err
	}
	return s.manager.DeleteYamlTemplate(ctx, req.ID, req.ClusterID, template.UserID)
}

func (s *yamlTemplateService) ownedYamlTemplate(ctx context.Context, id, clusterID, actorID int, super bool) (*model.K8sYamlTemplate, error) {
	template, err := s.manager.GetYamlTemplateDetail(ctx, id, clusterID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("YAML 模板不存在或不属于指定集群")
	}
	if err := authorizeYamlOwner(s.logger, actorID, template.UserID, template.ID, yamlObjectTemplate, super); err != nil {
		return nil, err
	}
	return template, nil
}

func (s *yamlTemplateService) GetYamlTemplateDetail(ctx context.Context, req *model.YamlTemplateDetailReq) (*model.K8sYamlTemplate, error) {
	return s.manager.GetYamlTemplateDetail(ctx, req.ID, req.ClusterID)
}
