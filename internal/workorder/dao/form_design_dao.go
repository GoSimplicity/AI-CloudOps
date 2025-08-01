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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrFormDesignNotFound   = fmt.Errorf("表单设计不存在")
	ErrFormDesignNameExists = fmt.Errorf("表单设计名称已存在")
)

type WorkorderFormDesignDAO interface {
	CreateFormDesign(ctx context.Context, formDesign *model.WorkorderFormDesign) error
	UpdateFormDesign(ctx context.Context, formDesign *model.WorkorderFormDesign) error
	DeleteFormDesign(ctx context.Context, id int) error
	GetFormDesign(ctx context.Context, id int) (*model.WorkorderFormDesign, error)
	ListFormDesign(ctx context.Context, req *model.ListWorkorderFormDesignReq) ([]*model.WorkorderFormDesign, int64, error)
	CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error)
}

type workorderFormDesignDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewWorkorderFormDesignDAO(db *gorm.DB, logger *zap.Logger) WorkorderFormDesignDAO {
	return &workorderFormDesignDAO{
		db:     db,
		logger: logger,
	}
}

// CreateFormDesign 创建表单设计
func (f *workorderFormDesignDAO) CreateFormDesign(ctx context.Context, formDesign *model.WorkorderFormDesign) error {
	if err := f.db.WithContext(ctx).Create(formDesign).Error; err != nil {
		f.logger.Error("创建表单设计失败", zap.Error(err), zap.String("name", formDesign.Name))
		return fmt.Errorf("创建表单设计失败: %w", err)
	}

	return nil
}

// UpdateFormDesign 更新表单设计
func (f *workorderFormDesignDAO) UpdateFormDesign(ctx context.Context, formDesign *model.WorkorderFormDesign) error {
	updateData := map[string]any{
		"name":        formDesign.Name,
		"description": formDesign.Description,
		"schema":      formDesign.Schema,
		"category_id": formDesign.CategoryID,
		"status":      formDesign.Status,
		"tags":        formDesign.Tags,
		"is_template": formDesign.IsTemplate,
	}
	result := f.db.WithContext(ctx).
		Model(&model.WorkorderFormDesign{}).
		Where("id = ?", formDesign.ID).
		Updates(updateData)

	if result.Error != nil {
		f.logger.Error("更新表单设计失败", zap.Error(result.Error), zap.Int("id", int(formDesign.ID)))
		return fmt.Errorf("更新表单设计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		f.logger.Warn("表单设计不存在", zap.Int("id", int(formDesign.ID)))
		return ErrFormDesignNotFound
	}

	return nil
}

// DeleteFormDesign 删除表单设计（软删除）
func (f *workorderFormDesignDAO) DeleteFormDesign(ctx context.Context, id int) error {
	result := f.db.WithContext(ctx).Delete(&model.WorkorderFormDesign{}, id)
	if result.Error != nil {
		f.logger.Error("删除表单设计失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除表单设计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		f.logger.Warn("表单设计不存在", zap.Int("id", id))
		return ErrFormDesignNotFound
	}

	return nil
}

// GetFormDesign 获取表单设计
func (f *workorderFormDesignDAO) GetFormDesign(ctx context.Context, id int) (*model.WorkorderFormDesign, error) {
	var formDesign model.WorkorderFormDesign

	err := f.db.WithContext(ctx).
		Preload("Category").
		First(&formDesign, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			f.logger.Warn("表单设计不存在", zap.Int("id", id))
			return nil, ErrFormDesignNotFound
		}
		f.logger.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取表单设计失败: %w", err)
	}

	return &formDesign, nil
}

// ListFormDesign 获取表单设计列表
func (f *workorderFormDesignDAO) ListFormDesign(ctx context.Context, req *model.ListWorkorderFormDesignReq) ([]*model.WorkorderFormDesign, int64, error) {
	var formDesigns []*model.WorkorderFormDesign
	var total int64

	db := f.db.WithContext(ctx).Model(&model.WorkorderFormDesign{})

	// 构建查询条件
	db = f.buildListQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		f.logger.Error("获取表单设计总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取表单设计总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&formDesigns).Error

	if err != nil {
		f.logger.Error("获取表单设计列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取表单设计列表失败: %w", err)
	}

	return formDesigns, total, nil
}

// CheckFormDesignNameExists 检查表单设计名称是否存在
func (f *workorderFormDesignDAO) CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("表单设计名称不能为空")
	}

	var count int64
	db := f.db.WithContext(ctx).Model(&model.WorkorderFormDesign{}).Where("name = ?", name)

	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}

	if err := db.Count(&count).Error; err != nil {
		f.logger.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", name))
		return false, fmt.Errorf("检查表单设计名称是否存在失败: %w", err)
	}

	return count > 0, nil
}

// buildListQuery 构建列表查询条件
func (f *workorderFormDesignDAO) buildListQuery(db *gorm.DB, req *model.ListWorkorderFormDesignReq) *gorm.DB {
	if req.Search != "" {
		searchTerm := sanitizeSearchInput(req.Search)
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+searchTerm+"%", "%"+searchTerm+"%")
	}

	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if req.CategoryID != nil {
		if *req.CategoryID == 0 {
			db = db.Where("category_id IS NULL")
		} else {
			db = db.Where("category_id = ?", *req.CategoryID)
		}
	}

	return db
}
