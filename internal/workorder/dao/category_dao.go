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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CategoryDAO interface {
	CreateCategory(ctx context.Context, category *model.Category) error
	UpdateCategory(ctx context.Context, category *model.Category) error
	DeleteCategory(ctx context.Context, id int) error
	ListCategory(ctx context.Context, req model.ListCategoryReq) ([]*model.Category, int64, error)
	GetCategory(ctx context.Context, id int) (*model.Category, error)
	GetCategoryByIDs(ctx context.Context, ids []int) ([]*model.Category, error)
	GetAllCategories(ctx context.Context) ([]*model.Category, error)
	GetCategoriesByIDs(ctx context.Context, ids []int) ([]*model.Category, error)
	CheckCategoryExists(ctx context.Context, id int) (bool, error)
	CheckNameExists(ctx context.Context, name string, excludeID *int) (bool, error)
	GetCategoryStatistics(ctx context.Context) (*model.CategoryStatistics, error)
}

type categoryDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCategoryDAO(db *gorm.DB, logger *zap.Logger) CategoryDAO {
	return &categoryDAO{
		db:     db,
		logger: logger,
	}
}

// CreateCategory 创建分类
func (dao *categoryDAO) CreateCategory(ctx context.Context, category *model.Category) error {
	dao.logger.Debug("开始创建分类",
		zap.String("name", category.Name),
		zap.Any("parent_id", category.ParentID))

	if err := dao.validateCategory(ctx, category, true); err != nil {
		dao.logger.Error("分类验证失败", zap.Error(err))
		return fmt.Errorf("分类信息有误：%v", err)
	}

	if err := dao.db.WithContext(ctx).Create(category).Error; err != nil {
		dao.logger.Error("创建分类失败",
			zap.Error(err),
			zap.String("name", category.Name))
		return fmt.Errorf("创建分类失败，请稍后重试，错误信息：%w", err)
	}

	dao.logger.Info("分类创建成功",
		zap.Int("id", category.ID),
		zap.String("name", category.Name))
	return nil
}

// UpdateCategory 更新分类
func (dao *categoryDAO) UpdateCategory(ctx context.Context, category *model.Category) error {
	dao.logger.Debug("开始更新分类",
		zap.Int("id", category.ID),
		zap.String("name", category.Name))

	if err := dao.validateCategory(ctx, category, false); err != nil {
		dao.logger.Error("分类验证失败", zap.Error(err))
		return fmt.Errorf("分类信息有误：%v", err)
	}

	// 构建更新数据，避免零值问题
	updateData := dao.buildUpdateData(category)

	result := dao.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("id = ?", category.ID).
		Updates(updateData)

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
func (dao *categoryDAO) DeleteCategory(ctx context.Context, id int) error {
	dao.logger.Debug("开始删除分类", zap.Int("id", id))

	result := dao.db.WithContext(ctx).Delete(&model.Category{}, id)
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

	dao.logger.Info("分类删除成功", zap.Int("id", id))
	return nil
}

// ListCategory 列出分类 (分页)
func (dao *categoryDAO) ListCategory(ctx context.Context, req model.ListCategoryReq) ([]*model.Category, int64, error) {
	dao.logger.Debug("开始列出分类",
		zap.String("name", req.Search),
		zap.Any("status", req.Status),
		zap.Int("page", req.Page),
		zap.Int("page_size", req.Size))

	var categories []*model.Category
	var total int64

	// 构建查询
	db := dao.buildListQuery(ctx, req)

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		dao.logger.Error("计算分类总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取分类总数失败，请稍后重试，错误信息：%w", err)
	}

	// 分页参数验证和设置
	page, pageSize := dao.validatePagination(req.Page, req.Size)
	offset := (page - 1) * pageSize

	// 执行查询
	if err := db.Offset(offset).
		Limit(pageSize).
		Order("sort_order ASC, id ASC").
		Find(&categories).Error; err != nil {
		dao.logger.Error("查询分类列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取分类列表失败，请稍后重试，错误信息：%w", err)
	}

	dao.logger.Debug("分类列表获取成功",
		zap.Int("count", len(categories)),
		zap.Int64("total", total))
	return categories, total, nil
}

// GetCategory 获取单个分类详情
func (dao *categoryDAO) GetCategory(ctx context.Context, id int) (*model.Category, error) {
	dao.logger.Debug("开始获取分类详情", zap.Int("id", id))

	var category model.Category
	if err := dao.db.WithContext(ctx).First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dao.logger.Debug("分类不存在", zap.Int("id", id))
			return nil, nil
		}
		dao.logger.Error("获取分类详情失败",
			zap.Error(err),
			zap.Int("id", id))
		return nil, fmt.Errorf("获取分类详情失败，请稍后重试，错误信息：%w", err)
	}

	dao.logger.Debug("分类详情获取成功", zap.Int("id", id))
	return &category, nil
}

// GetAllCategories 获取所有分类
func (dao *categoryDAO) GetAllCategories(ctx context.Context) ([]*model.Category, error) {
	dao.logger.Debug("开始获取所有分类")

	var categories []*model.Category
	if err := dao.db.WithContext(ctx).
		Where("status = ?", 1). // 只获取启用的分类
		Order("sort_order ASC, id ASC").
		Find(&categories).Error; err != nil {
		dao.logger.Error("获取所有分类失败", zap.Error(err))
		return nil, fmt.Errorf("获取所有分类失败，请稍后重试，错误信息：%w", err)
	}

	dao.logger.Debug("所有分类获取成功", zap.Int("count", len(categories)))
	return categories, nil
}

// GetCategoriesByIDs 根据ID列表获取分类
func (dao *categoryDAO) GetCategoriesByIDs(ctx context.Context, ids []int) ([]*model.Category, error) {
	if len(ids) == 0 {
		return []*model.Category{}, nil
	}

	dao.logger.Debug("根据ID列表获取分类", zap.Ints("ids", ids))

	var categories []*model.Category
	if err := dao.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("sort_order ASC, id ASC").
		Find(&categories).Error; err != nil {
		dao.logger.Error("根据ID列表获取分类失败",
			zap.Error(err),
			zap.Ints("ids", ids))
		return nil, fmt.Errorf("获取分类信息失败，请稍后重试，错误信息：%w", err)
	}

	dao.logger.Debug("根据ID列表获取分类成功",
		zap.Int("count", len(categories)))
	return categories, nil
}

// GetCategoryByIDs 根据ID列表获取分类
func (dao *categoryDAO) GetCategoryByIDs(ctx context.Context, ids []int) ([]*model.Category, error) {
	if len(ids) == 0 {
		return []*model.Category{}, nil
	}

	var categories []*model.Category

	if err := dao.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("sort_order ASC, id ASC").
		Find(&categories).Error; err != nil {
		dao.logger.Error("根据ID列表获取分类失败",
			zap.Error(err),
			zap.Ints("ids", ids))
		return nil, fmt.Errorf("获取分类信息失败，请稍后重试，错误信息：%w", err)
	}

	return categories, nil
}

// CheckCategoryExists 检查分类是否存在
func (dao *categoryDAO) CheckCategoryExists(ctx context.Context, id int) (bool, error) {
	var count int64
	if err := dao.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		dao.logger.Error("检查分类是否存在失败",
			zap.Error(err),
			zap.Int("id", id))
		return false, fmt.Errorf("检查分类是否存在失败，请稍后重试，错误信息：%w", err)
	}
	return count > 0, nil
}

// CheckNameExists 检查分类名称是否存在
func (dao *categoryDAO) CheckNameExists(ctx context.Context, name string, excludeID *int) (bool, error) {
	query := dao.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("name = ?", name)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		dao.logger.Error("检查分类名称是否存在失败",
			zap.Error(err),
			zap.String("name", name))
		return false, fmt.Errorf("检查分类名称是否存在失败，请稍后重试，错误信息：%w", err)
	}
	return count > 0, nil
}

func (dao *categoryDAO) GetCategoryStatistics(ctx context.Context) (*model.CategoryStatistics, error) {
	var statistics model.CategoryStatistics

	if err := dao.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("status = ?", 1). // 状态为1表示启用
		Count(&statistics.EnabledCount).Error; err != nil {
		return nil, fmt.Errorf("获取分类统计失败，请稍后重试，错误信息：%w", err)
	}

	if err := dao.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("status = ?", 2). // 状态为2表示禁用
		Count(&statistics.DisabledCount).Error; err != nil {
		return nil, fmt.Errorf("获取分类统计失败，请稍后重试，错误信息：%w", err)
	}

	return &statistics, nil
}

// validateCategory 验证分类数据
func (dao *categoryDAO) validateCategory(ctx context.Context, category *model.Category, isCreate bool) error {
	// 验证名称
	if strings.TrimSpace(category.Name) == "" {
		return errors.New("分类名称不能为空")
	}

	// 检查名称是否重复
	var excludeID *int
	if !isCreate {
		excludeID = &category.ID
	}

	exists, err := dao.CheckNameExists(ctx, category.Name, excludeID)
	if err != nil {
		return fmt.Errorf("检查分类名称时发生错误，请稍后重试，错误信息：%w", err)
	}
	if exists {
		return errors.New("分类名称已存在")
	}

	// 验证父分类
	if category.ParentID != nil && *category.ParentID > 0 {
		exists, err := dao.CheckCategoryExists(ctx, *category.ParentID)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("父分类不存在")
		}

		// 防止循环引用（更新时）
		if !isCreate && category.ID == *category.ParentID {
			return errors.New("不能将自己设为父分类")
		}

		// 防止将父分类设为自己的子分类
		if !isCreate {
			// 获取当前数据库中的分类
			dbCategory, err := dao.GetCategory(ctx, category.ID)
			if err != nil {
				return err
			}
			if dbCategory == nil {
				return errors.New("要更新的分类不存在")
			}
			// 只有当 parentID 发生变化时才检查循环引用
			var dbParentID int
			if dbCategory.ParentID != nil {
				dbParentID = *dbCategory.ParentID
			}
			if dbParentID != *category.ParentID {
				if err := dao.checkCircularReference(ctx, category.ID, *category.ParentID); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// checkCircularReference 检查循环引用
func (dao *categoryDAO) checkCircularReference(ctx context.Context, categoryID, parentID int) error {
	// 如果 parentID 为 0 或 nil，直接返回
	if parentID == 0 {
		return nil
	}
	visited := make(map[int]bool)
	return dao.dfsCheckCircular(ctx, categoryID, parentID, visited)
}

// dfsCheckCircular 深度优先搜索检查循环引用
func (dao *categoryDAO) dfsCheckCircular(ctx context.Context, targetID, currentID int, visited map[int]bool) error {
	if currentID == targetID {
		return errors.New("设置父分类会产生循环引用")
	}

	if visited[currentID] {
		return nil
	}
	visited[currentID] = true

	// 获取当前分类的父ID，向上递归
	category, err := dao.GetCategory(ctx, currentID)
	if err != nil {
		return err
	}
	if category == nil || category.ParentID == nil || *category.ParentID == 0 {
		return nil
	}
	return dao.dfsCheckCircular(ctx, targetID, *category.ParentID, visited)
}

// buildUpdateData 构建更新数据
func (dao *categoryDAO) buildUpdateData(category *model.Category) map[string]interface{} {
	updateData := map[string]interface{}{
		"name":        category.Name,
		"icon":        category.Icon,
		"sort_order":  category.SortOrder,
		"status":      category.Status,
		"description": category.Description,
	}

	// 处理 ParentID，允许设置为 NULL
	if category.ParentID != nil {
		updateData["parent_id"] = *category.ParentID
	} else {
		updateData["parent_id"] = nil
	}

	return updateData
}

// buildListQuery 构建列表查询
func (dao *categoryDAO) buildListQuery(ctx context.Context, req model.ListCategoryReq) *gorm.DB {
	db := dao.db.WithContext(ctx).Model(&model.Category{})

	// 名称搜索
	if req.Search != "" {
		db = db.Where("name LIKE ?", "%"+strings.TrimSpace(req.Search)+"%")
	}

	// 状态过滤
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	return db
}

// validatePagination 验证和设置分页参数
func (dao *categoryDAO) validatePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
