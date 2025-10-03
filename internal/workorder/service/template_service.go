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
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type WorkorderTemplateService interface {
	CreateTemplate(ctx context.Context, req *model.CreateWorkorderTemplateReq) error
	UpdateTemplate(ctx context.Context, req *model.UpdateWorkorderTemplateReq) error
	DeleteTemplate(ctx context.Context, req *model.DeleteWorkorderTemplateReq) error
	ListTemplate(ctx context.Context, req *model.ListWorkorderTemplateReq) (*model.ListResp[*model.WorkorderTemplate], error)
	DetailTemplate(ctx context.Context, req *model.DetailWorkorderTemplateReq) (*model.WorkorderTemplate, error)
}

type workorderTemplateService struct {
	dao         dao.WorkorderTemplateDAO
	processDao  dao.WorkorderProcessDAO
	categoryDao dao.WorkorderCategoryDAO
	instanceDao dao.WorkorderInstanceDAO
	l           *zap.Logger
}

func NewWorkorderTemplateService(
	dao dao.WorkorderTemplateDAO,
	processDao dao.WorkorderProcessDAO,
	categoryDao dao.WorkorderCategoryDAO,
	instanceDao dao.WorkorderInstanceDAO,
	l *zap.Logger,
) WorkorderTemplateService {
	return &workorderTemplateService{
		dao:         dao,
		processDao:  processDao,
		categoryDao: categoryDao,
		instanceDao: instanceDao,
		l:           l,
	}
}

// CreateTemplate 创建模板
func (s *workorderTemplateService) CreateTemplate(ctx context.Context, req *model.CreateWorkorderTemplateReq) error {
	exists, err := s.checkTemplateNameExists(ctx, req.Name)
	if err != nil {
		s.l.Error("检查模板名称失败", zap.Error(err), zap.String("name", req.Name), zap.Int("creatorID", req.OperatorID))
		return fmt.Errorf("检查模板名称失败: %w", err)
	}
	if exists {
		return fmt.Errorf("模板名称已存在: %s", req.Name)
	}

	if err := s.validateProcessExists(ctx, req.ProcessID); err != nil {
		return err
	}

	if req.CategoryID != nil && *req.CategoryID > 0 {
		if err := s.validateCategoryExists(ctx, *req.CategoryID); err != nil {
			return err
		}
	}

	template := &model.WorkorderTemplate{
		Name:          req.Name,
		Description:   req.Description,
		ProcessID:     req.ProcessID,
		FormDesignID:  req.FormDesignID,
		DefaultValues: req.DefaultValues,
		Status:        1,
		CategoryID:    req.CategoryID,
		OperatorID:    req.OperatorID,
		OperatorName:  req.OperatorName,
		Tags:          req.Tags,
	}

	if err := s.dao.CreateTemplate(ctx, template); err != nil {
		s.l.Error("创建模板失败", zap.Error(err), zap.String("name", req.Name), zap.Int("creatorID", req.OperatorID))
		return fmt.Errorf("创建模板失败: %w", err)
	}

	return nil
}

// UpdateTemplate 更新模板
func (s *workorderTemplateService) UpdateTemplate(ctx context.Context, req *model.UpdateWorkorderTemplateReq) error {
	if req.ID <= 0 {
		return errors.New("模板ID无效")
	}

	existingTemplate, err := s.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取模板失败: %w", err)
	}

	if req.Name != existingTemplate.Name {
		exists, err := s.dao.IsTemplateNameExists(ctx, req.Name, req.ID)
		if err != nil {
			s.l.Error("检查模板名称失败", zap.Error(err), zap.String("name", req.Name))
			return fmt.Errorf("检查模板名称失败: %w", err)
		}

		if exists {
			return fmt.Errorf("模板名称已存在: %s", req.Name)
		}
	}

	if req.ProcessID != existingTemplate.ProcessID {
		if err := s.validateProcessExists(ctx, req.ProcessID); err != nil {
			return err
		}
	}

	if req.CategoryID != nil && *req.CategoryID > 0 {
		if err := s.validateCategoryExists(ctx, *req.CategoryID); err != nil {
			return err
		}
	}

	template := &model.WorkorderTemplate{
		Model:         model.Model{ID: req.ID},
		Name:          req.Name,
		Description:   req.Description,
		ProcessID:     req.ProcessID,
		FormDesignID:  req.FormDesignID,
		DefaultValues: req.DefaultValues,
		Status:        req.Status,
		CategoryID:    req.CategoryID,
		Tags:          req.Tags,
	}

	if err := s.dao.UpdateTemplate(ctx, template); err != nil {
		s.l.Error("更新模板失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新模板失败: %w", err)
	}

	return nil
}

// DeleteTemplate 删除模板
func (s *workorderTemplateService) DeleteTemplate(ctx context.Context, req *model.DeleteWorkorderTemplateReq) error {
	if req.ID <= 0 {
		return errors.New("模板ID无效")
	}

	template, err := s.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取模板失败: %w", err)
	}

	instances, _, err := s.instanceDao.ListInstance(ctx, &model.ListWorkorderInstanceReq{
		ProcessID: &template.ProcessID,
	})
	if err != nil {
		s.l.Error("获取关联工单失败", zap.Error(err), zap.Int("templateID", req.ID))
		return fmt.Errorf("获取关联工单失败: %w", err)
	}

	if len(instances) > 0 {
		s.l.Warn("模板有关联的工单，无法删除", zap.Int("templateID", req.ID), zap.Int("instanceCount", len(instances)))
		return errors.New("模板有关联的工单，无法删除")
	}

	if err := s.dao.DeleteTemplate(ctx, req.ID); err != nil {
		s.l.Error("删除模板失败", zap.Error(err), zap.Int("id", req.ID), zap.String("name", template.Name))
		return fmt.Errorf("删除模板失败: %w", err)
	}

	return nil
}

// ListTemplate 获取模板列表
func (s *workorderTemplateService) ListTemplate(ctx context.Context, req *model.ListWorkorderTemplateReq) (*model.ListResp[*model.WorkorderTemplate], error) {
	templates, total, err := s.dao.ListTemplate(ctx, req)
	if err != nil {
		s.l.Error("获取模板列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取模板列表失败: %w", err)
	}

	result := &model.ListResp[*model.WorkorderTemplate]{
		Items: templates,
		Total: total,
	}

	return result, nil
}

// DetailTemplate 获取模板
func (s *workorderTemplateService) DetailTemplate(ctx context.Context, req *model.DetailWorkorderTemplateReq) (*model.WorkorderTemplate, error) {
	if req.ID <= 0 {
		return nil, errors.New("模板ID无效")
	}

	template, err := s.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		s.l.Error("获取模板详情失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("获取模板详情失败: %w", err)
	}

	return template, nil
}

// checkTemplateNameExists 检查模板名称
func (s *workorderTemplateService) checkTemplateNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if name == "" {
		return false, errors.New("模板名称不能为空")
	}

	var id int

	if len(excludeID) > 0 {
		id = excludeID[0]
	}

	return s.dao.IsTemplateNameExists(ctx, name, id)
}

// validateProcessExists 验证流程是否存在
func (s *workorderTemplateService) validateProcessExists(ctx context.Context, processID int) error {
	if processID <= 0 {
		return errors.New("流程ID无效")
	}

	_, err := s.processDao.GetProcessByID(ctx, processID)
	if err != nil {
		s.l.Error("验证流程发生错误", zap.Error(err))
		return errors.New("关联的流程不存在或无效")
	}

	return nil
}

// validateCategoryExists 验证分类
func (s *workorderTemplateService) validateCategoryExists(ctx context.Context, categoryID int) error {
	if categoryID <= 0 {
		return errors.New("分类ID无效")
	}

	_, err := s.categoryDao.GetCategory(ctx, categoryID)
	if err != nil {
		s.l.Error("验证分类发生错误", zap.Error(err))
		return errors.New("关联的分类不存在或无效")
	}

	return nil
}
