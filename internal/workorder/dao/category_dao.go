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

package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderCategoryDAO interface {
	CreateCategory(ctx context.Context, category *model.WorkorderCategory) error
	UpdateCategory(ctx context.Context, category *model.WorkorderCategory) error
	DeleteCategory(ctx context.Context, id int) error
	ListCategory(ctx context.Context, req model.ListWorkorderCategoryReq) ([]*model.WorkorderCategory, int64, error)
	ListCategoryByIDs(ctx context.Context, ids []int) ([]*model.WorkorderCategory, error)
	GetCategory(ctx context.Context, id int) (*model.WorkorderCategory, error)
}

type workorderCategoryDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewWorkorderCategoryDAO(db *gorm.DB, logger *zap.Logger) WorkorderCategoryDAO {
	return &workorderCategoryDAO{
		db:     db,
		logger: logger,
	}
}

// CreateCategory 创建分类
func (dao *workorderCategoryDAO) CreateCategory(ctx context.Context, category *model.WorkorderCategory) error {
	if err := dao.db.WithContext(ctx).Create(category).Error; err != nil {
		dao.logger.Error("创建分类失败", zap.Error(err))
		return fmt.Errorf("创建分类失败，请稍后重试，错误信息：%w", err)
	}

	return nil
}

// UpdateCategory 更新分类
func (dao *workorderCategoryDAO) UpdateCategory(ctx context.Context, category *model.WorkorderCategory) error {
	result := dao.db.WithContext(ctx).
		Model(&model.WorkorderCategory{}).
		Where("id = ?", category.ID).
		Updates(map[string]interface{}{
			"name":        category.Name,
			"description": category.Description,
			"status":      category.Status,
		})

	if err := result.Error; err != nil {
		dao.logger.Error("更新分类失败",
			zap.Error(err),
			zap.Int("id", category.ID))
		return fmt.Errorf("更新分类失败，请稍后重试，错误信息：%w", err)
	}

	if result.RowsAffected == 0 {
		dao.logger.Warn("更新分类：未找到记录", zap.Int("id", category.ID))
		return fmt.Errorf("未找到要更新的分类，id=%d", category.ID)
	}

	return nil
}

// DeleteCategory 删除分类 (软删除)
func (dao *workorderCategoryDAO) DeleteCategory(ctx context.Context, id int) error {
	result := dao.db.WithContext(ctx).Delete(&model.WorkorderCategory{}, id)
	if err := result.Error; err != nil {
		dao.logger.Error("删除分类失败",
			zap.Error(err),
			zap.Int("id", id))
		return fmt.Errorf("删除分类失败，请稍后重试，错误信息：%w", err)
	}

	if result.RowsAffected == 0 {
		dao.logger.Warn("删除分类：未找到记录", zap.Int("id", id))
		return fmt.Errorf("未找到要删除的分类，id=%d", id)
	}

	return nil
}

// ListCategory 列出分类 (分页)
func (dao *workorderCategoryDAO) ListCategory(ctx context.Context, req model.ListWorkorderCategoryReq) ([]*model.WorkorderCategory, int64, error) {
	var categories []*model.WorkorderCategory
	var total int64

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 {
		req.Size = 100
	}

	query := dao.db.WithContext(ctx).Model(&model.WorkorderCategory{})

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		dao.logger.Error("计算分类总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取分类总数失败，请稍后重试，错误信息：%w", err)
	}

	// 分页参数验证和设置
	offset := (req.Page - 1) * req.Size

	// 执行查询
	if err := query.Offset(offset).
		Limit(req.Size).
		Order("id ASC").
		Find(&categories).Error; err != nil {
		dao.logger.Error("查询分类列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取分类列表失败，请稍后重试，错误信息：%w", err)
	}

	return categories, total, nil
}

// ListCategoryByIDs 根据IDs列表获取分类列表
func (dao *workorderCategoryDAO) ListCategoryByIDs(ctx context.Context, ids []int) ([]*model.WorkorderCategory, error) {
	var categories []*model.WorkorderCategory

	if err := dao.db.WithContext(ctx).Where("id IN (?)", ids).Find(&categories).Error; err != nil {
		dao.logger.Error("根据IDs列表获取分类列表失败", zap.Error(err))
		return nil, fmt.Errorf("根据IDs列表获取分类列表失败，请稍后重试，错误信息：%w", err)
	}

	return categories, nil
}

// GetCategory 获取单个分类详情
func (dao *workorderCategoryDAO) GetCategory(ctx context.Context, id int) (*model.WorkorderCategory, error) {
	var category model.WorkorderCategory
	if err := dao.db.WithContext(ctx).Where("id = ?", id).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		dao.logger.Error("获取分类详情失败",
			zap.Error(err),
			zap.Int("id", id))
		return nil, fmt.Errorf("获取分类详情失败，请稍后重试，错误信息：%w", err)
	}

	return &category, nil
}
