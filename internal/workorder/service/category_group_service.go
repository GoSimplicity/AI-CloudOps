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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type CategoryGroupService interface {
	CreateCategory(ctx context.Context, req *model.CreateWorkorderCategoryReq) error
	UpdateCategory(ctx context.Context, req *model.UpdateWorkorderCategoryReq) error
	DeleteCategory(ctx context.Context, id int) error
	ListCategory(ctx context.Context, req model.ListWorkorderCategoryReq) (*model.ListResp[*model.WorkorderCategory], error)
	GetCategory(ctx context.Context, id int) (*model.WorkorderCategory, error)
}

type categoryGroupService struct {
	categoryDAO dao.WorkorderCategoryDAO
	userDAO     userdao.UserDAO
	logger      *zap.Logger
}

func NewCategoryGroupService(categoryDAO dao.WorkorderCategoryDAO, userDAO userdao.UserDAO, logger *zap.Logger) CategoryGroupService {
	return &categoryGroupService{
		categoryDAO: categoryDAO,
		userDAO:     userDAO,
		logger:      logger,
	}
}

// CreateCategory 创建分类的实现
func (s *categoryGroupService) CreateCategory(ctx context.Context, req *model.CreateWorkorderCategoryReq) error {
	// 检查分类是否存在
	existingCategory, err := s.categoryDAO.GetCategoryByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("创建分类失败：获取分类信息失败", zap.Error(err), zap.String("name", req.Name))
		return fmt.Errorf("获取分类信息失败 (name: %s): %w", req.Name, err)
	}
	
	if existingCategory != nil {
		s.logger.Warn("创建分类失败：分类已存在", zap.String("name", req.Name))
		return fmt.Errorf("分类已存在: %s", req.Name)
	}

	category := &model.WorkorderCategory{
		Name:         req.Name,
		Description:  req.Description,
		Status:       req.Status,
		OperatorID:   req.OperatorID,
		OperatorName: req.OperatorName,
	}

	err = s.categoryDAO.CreateCategory(ctx, category)
	if err != nil {
		s.logger.Error("创建分类失败", zap.Error(err), zap.String("name", req.Name))
		return fmt.Errorf("创建分类 '%s' 失败: %w", req.Name, err)
	}

	return nil
}

// UpdateCategory 更新分类的实现
func (s *categoryGroupService) UpdateCategory(ctx context.Context, req *model.UpdateWorkorderCategoryReq) error {
	// 检查分类是否存在
	existingCategory, err := s.categoryDAO.GetCategory(ctx, req.ID)
	if err != nil {
		s.logger.Error("更新分类失败：获取分类信息失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("获取分类信息失败 (ID: %d): %w", req.ID, err)
	}
	if existingCategory == nil {
		s.logger.Warn("更新分类失败：分类不存在", zap.Int("id", req.ID))
		return fmt.Errorf("分类 (ID: %d) 不存在", req.ID)
	}

	// 构建更新的分类对象
	category := &model.WorkorderCategory{
		Model: model.Model{
			ID: req.ID,
		},
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	}

	err = s.categoryDAO.UpdateCategory(ctx, category)
	if err != nil {
		s.logger.Error("更新分类失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新分类 (ID: %d) 失败: %w", req.ID, err)
	}

	return nil
}

// DeleteCategory 删除分类的实现
func (s *categoryGroupService) DeleteCategory(ctx context.Context, id int) error {
	// 检查分类是否存在
	existingCategory, err := s.categoryDAO.GetCategory(ctx, id)
	if err != nil {
		s.logger.Error("删除分类失败：获取分类信息失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("获取分类信息失败 (ID: %d): %w", id, err)
	}

	if existingCategory == nil {
		s.logger.Warn("删除分类失败：分类不存在", zap.Int("id", id))
		return fmt.Errorf("分类 (ID: %d) 不存在", id)
	}

	err = s.categoryDAO.DeleteCategory(ctx, id)
	if err != nil {
		s.logger.Error("删除分类失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除分类 (ID: %d) 失败: %w", id, err)
	}

	return nil
}

// ListCategory 列出分类的实现
func (s *categoryGroupService) ListCategory(ctx context.Context, req model.ListWorkorderCategoryReq) (*model.ListResp[*model.WorkorderCategory], error) {
	categories, total, err := s.categoryDAO.ListCategory(ctx, req)
	if err != nil {
		s.logger.Error("列出分类失败", zap.Error(err))
		return nil, fmt.Errorf("列出分类失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderCategory]{
		Total: total,
		Items: categories,
	}, nil
}

// GetCategory 获取单个分类详情的实现
func (s *categoryGroupService) GetCategory(ctx context.Context, id int) (*model.WorkorderCategory, error) {
	category, err := s.categoryDAO.GetCategory(ctx, id)
	if err != nil {
		s.logger.Error("获取分类详情失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取分类详情 (ID: %d) 失败: %w", id, err)
	}
	if category == nil {
		s.logger.Warn("获取分类详情失败：分类不存在", zap.Int("id", id))
		return nil, fmt.Errorf("分类 (ID: %d) 不存在", id)
	}

	return category, nil
}
