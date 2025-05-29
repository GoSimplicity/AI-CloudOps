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
	CreateCategory(ctx context.Context, req *model.CreateCategoryReq, creatorID int, creatorName string) (*model.CategoryResp, error)
	UpdateCategory(ctx context.Context, req *model.UpdateCategoryReq, userID int) (*model.CategoryResp, error)
	DeleteCategory(ctx context.Context, id int, userID int) error
	ListCategory(ctx context.Context, req model.ListCategoryReq) (*model.ListResp[model.CategoryResp], error)
	GetCategory(ctx context.Context, id int) (*model.CategoryResp, error)
	GetCategoryTree(ctx context.Context) ([]model.CategoryResp, error)
}

type categoryGroupService struct {
	categoryDAO dao.CategoryDAO
	userDAO     userdao.UserDAO
	logger      *zap.Logger
}

func NewCategoryGroupService(categoryDAO dao.CategoryDAO, userDAO userdao.UserDAO, logger *zap.Logger) CategoryGroupService {
	return &categoryGroupService{
		categoryDAO: categoryDAO,
		userDAO:     userDAO,
		logger:      logger,
	}
}

// convertToCategoryResp 将 model.Category 转换为 model.CategoryResp
func (s *categoryGroupService) convertToCategoryResp(category *model.Category) *model.CategoryResp {
	if category == nil {
		return nil
	}
	return &model.CategoryResp{
		ID:          category.ID,
		Name:        category.Name,
		ParentID:    category.ParentID,
		Icon:        category.Icon,
		SortOrder:   category.SortOrder,
		Status:      category.Status,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
		CreatorName: category.CreatorName,
		Children:    make([]model.CategoryResp, 0), // 确保初始化为空切片而不是nil
	}
}

// convertToCategoryRespList 将 []model.Category 转换为 []model.CategoryResp
func (s *categoryGroupService) convertToCategoryRespList(categories []model.Category) []model.CategoryResp {
	resps := make([]model.CategoryResp, 0, len(categories))
	for _, category := range categories {
		resp := s.convertToCategoryResp(&category)
		if resp != nil {
			resps = append(resps, *resp)
		}
	}
	return resps
}

// CreateCategory 创建分类的实现
func (s *categoryGroupService) CreateCategory(ctx context.Context, req *model.CreateCategoryReq, creatorID int, creatorName string) (*model.CategoryResp, error) {
	s.logger.Info("开始创建分类",
		zap.String("name", req.Name),
		zap.Int("creatorID", creatorID),
		zap.String("creatorName", creatorName))

	category := &model.Category{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
		Description: req.Description,
		Status:      1, // 默认为启用状态
		CreatorID:   creatorID,
		CreatorName: creatorName,
	}

	createdCategory, err := s.categoryDAO.CreateCategory(ctx, category)
	if err != nil {
		s.logger.Error("创建分类失败", zap.Error(err), zap.String("name", req.Name))
		return nil, fmt.Errorf("创建分类 '%s' 失败: %w", req.Name, err)
	}

	s.logger.Info("分类创建成功",
		zap.Int("id", createdCategory.ID),
		zap.String("name", createdCategory.Name))

	return s.convertToCategoryResp(createdCategory), nil
}

// UpdateCategory 更新分类的实现
func (s *categoryGroupService) UpdateCategory(ctx context.Context, req *model.UpdateCategoryReq, userID int) (*model.CategoryResp, error) {
	s.logger.Info("开始更新分类", zap.Int("id", req.ID), zap.Int("userID", userID))

	// 检查分类是否存在
	existingCategory, err := s.categoryDAO.GetCategory(ctx, req.ID)
	if err != nil {
		s.logger.Error("更新分类失败：获取分类信息失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("获取分类信息失败 (ID: %d): %w", req.ID, err)
	}
	if existingCategory == nil {
		s.logger.Warn("更新分类失败：分类不存在", zap.Int("id", req.ID))
		return nil, fmt.Errorf("分类 (ID: %d) 不存在", req.ID)
	}

	// 构建更新的分类对象
	category := &model.Category{
		Model: model.Model{
			ID: req.ID,
		},
		Name:        req.Name,
		ParentID:    req.ParentID,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
		Description: req.Description,
		Status:      *req.Status,
	}

	updatedCategory, err := s.categoryDAO.UpdateCategory(ctx, category)
	if err != nil {
		s.logger.Error("更新分类失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("更新分类 (ID: %d) 失败: %w", req.ID, err)
	}

	s.logger.Info("分类更新成功",
		zap.Int("id", updatedCategory.ID),
		zap.String("name", updatedCategory.Name))

	return s.convertToCategoryResp(updatedCategory), nil
}

// DeleteCategory 删除分类的实现
func (s *categoryGroupService) DeleteCategory(ctx context.Context, id int, userID int) error {
	s.logger.Info("开始删除分类", zap.Int("id", id), zap.Int("userID", userID))

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

	s.logger.Info("分类删除成功", zap.Int("id", id))
	return nil
}

// ListCategory 列出分类的实现
func (s *categoryGroupService) ListCategory(ctx context.Context, req model.ListCategoryReq) (*model.ListResp[model.CategoryResp], error) {
	s.logger.Debug("开始列出分类", zap.Any("request", req))

	categories, total, err := s.categoryDAO.ListCategory(ctx, req)
	if err != nil {
		s.logger.Error("列出分类失败", zap.Error(err))
		return nil, fmt.Errorf("列出分类失败: %w", err)
	}

	categoryResps := s.convertToCategoryRespList(categories)

	s.logger.Debug("分类列表获取成功",
		zap.Int("count", len(categoryResps)),
		zap.Int64("total", total))

	return &model.ListResp[model.CategoryResp]{
		Total: total,
		Items: categoryResps,
	}, nil
}

// GetCategory 获取单个分类详情的实现
func (s *categoryGroupService) GetCategory(ctx context.Context, id int) (*model.CategoryResp, error) {
	s.logger.Debug("开始获取分类详情", zap.Int("id", id))

	category, err := s.categoryDAO.GetCategory(ctx, id)
	if err != nil {
		s.logger.Error("获取分类详情失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取分类详情 (ID: %d) 失败: %w", id, err)
	}
	if category == nil {
		s.logger.Warn("获取分类详情失败：分类不存在", zap.Int("id", id))
		return nil, fmt.Errorf("分类 (ID: %d) 不存在", id)
	}

	s.logger.Debug("分类详情获取成功", zap.Int("id", category.ID))
	return s.convertToCategoryResp(category), nil
}

// GetCategoryTree 获取分类树结构的实现
func (s *categoryGroupService) GetCategoryTree(ctx context.Context) ([]model.CategoryResp, error) {
	s.logger.Debug("开始获取分类树")

	allCategories, err := s.categoryDAO.GetAllCategories(ctx)
	if err != nil {
		s.logger.Error("获取所有分类失败（用于构建树）", zap.Error(err))
		return nil, fmt.Errorf("获取所有分类失败: %w", err)
	}

	// 构建分类映射，使用指针
	categoryMap := make(map[int]*model.CategoryResp)
	for i := range allCategories {
		categoryResp := s.convertToCategoryResp(&allCategories[i])
		if categoryResp != nil {
			// 确保Children不为nil
			categoryResp.Children = make([]model.CategoryResp, 0)
			categoryMap[allCategories[i].ID] = categoryResp
		}
	}

	// 递归构建树结构
	var buildTree func(parentID *int) []model.CategoryResp
	buildTree = func(parentID *int) []model.CategoryResp {
		var children []model.CategoryResp

		for _, category := range allCategories {
			// 判断是否为当前父节点的直接子节点
			if (parentID == nil && (category.ParentID == nil || *category.ParentID == 0)) ||
				(parentID != nil && category.ParentID != nil && *category.ParentID == *parentID) {

				if node, exists := categoryMap[category.ID]; exists {
					// 递归构建当前节点的子树
					node.Children = buildTree(&category.ID)
					children = append(children, *node)
				}
			}
		}

		return children
	}

	// 构建完整的树结构，从根节点开始
	rootCategories := buildTree(nil)

	s.logger.Debug("分类树获取成功",
		zap.Int("rootCategoriesCount", len(rootCategories)),
		zap.Int("totalCategories", len(allCategories)))

	return rootCategories, nil
}
