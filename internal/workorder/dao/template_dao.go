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

var (
	ErrTemplateNotFound   = errors.New("模板不存在")
	ErrTemplateNameExists = errors.New("模板名称已存在")
	ErrInvalidStatus      = errors.New("无效的状态值")
	ErrInvalidID          = errors.New("无效的ID")
)

type WorkorderTemplateDAO interface {
	CreateTemplate(ctx context.Context, template *model.WorkorderTemplate) error
	UpdateTemplate(ctx context.Context, template *model.WorkorderTemplate) error
	DeleteTemplate(ctx context.Context, id int) error
	GetTemplate(ctx context.Context, id int) (*model.WorkorderTemplate, error)
	ListTemplate(ctx context.Context, req *model.ListWorkorderTemplateReq) ([]*model.WorkorderTemplate, int64, error)
	UpdateTemplateStatus(ctx context.Context, id int, status int8) error
	IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error)
}

type templateDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTemplateDAO(db *gorm.DB, logger *zap.Logger) WorkorderTemplateDAO {
	return &templateDAO{
		db:     db,
		logger: logger,
	}
}

func (d *templateDAO) CreateTemplate(ctx context.Context, template *model.WorkorderTemplate) error {
	if len(template.DefaultValues) == 0 {
		template.DefaultValues = model.JSONMap{}
	}

	if err := DBWithContext(ctx, d.db).Create(template).Error; err != nil {
		d.logger.Error("创建模板失败", zap.Error(err), zap.String("name", template.Name))
		if d.isDuplicateKeyError(err) {
			return ErrTemplateNameExists
		}
		return fmt.Errorf("创建模板失败: %w", err)
	}

	return nil
}

func (d *templateDAO) UpdateTemplate(ctx context.Context, template *model.WorkorderTemplate) error {
	if template.ID <= 0 {
		return ErrInvalidID
	}

	if len(template.DefaultValues) == 0 {
		template.DefaultValues = model.JSONMap{}
	}

	updates := map[string]any{
		"name":           template.Name,
		"description":    template.Description,
		"process_id":     template.ProcessID,
		"default_values": template.DefaultValues,
		"status":         template.Status,
		"category_id":    template.CategoryID,
	}

	result := DBWithContext(ctx, d.db).Model(&model.WorkorderTemplate{}).
		Where("id = ?", template.ID).
		Updates(updates)

	if result.Error != nil {
		d.logger.Error("更新模板失败", zap.Error(result.Error), zap.Int("id", template.ID))
		if d.isDuplicateKeyError(result.Error) {
			return ErrTemplateNameExists
		}
		return fmt.Errorf("更新模板失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

func (d *templateDAO) DeleteTemplate(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidID
	}

	result := DBWithContext(ctx, d.db).Where("id = ?", id).Delete(&model.WorkorderTemplate{})
	if result.Error != nil {
		d.logger.Error("删除模板失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除模板失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

func (d *templateDAO) GetTemplate(ctx context.Context, id int) (*model.WorkorderTemplate, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}

	var template model.WorkorderTemplate
	err := DBWithContext(ctx, d.db).
		Where("id = ?", id).
		Preload("Process").
		Preload("Category").
		Preload("FormDesign").
		First(&template).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		d.logger.Error("获取模板失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	// 确保默认值不为空
	if len(template.DefaultValues) == 0 {
		template.DefaultValues = model.JSONMap{}
	}

	return &template, nil
}

func (d *templateDAO) ListTemplate(ctx context.Context, req *model.ListWorkorderTemplateReq) ([]*model.WorkorderTemplate, int64, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("请求参数不能为空")
	}

	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	var templates []*model.WorkorderTemplate
	var total int64

	db := DBWithContext(ctx, d.db).Model(&model.WorkorderTemplate{})

	db = d.buildListQuery(db, req)

	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取模板总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取模板总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.Size
	err := db.Offset(offset).
		Preload("Process").
		Preload("Category").
		Preload("FormDesign").
		Limit(req.Size).
		Order("created_at DESC").
		Find(&templates).Error

	if err != nil {
		d.logger.Error("查询模板列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("查询模板列表失败: %w", err)
	}

	// 确保所有模板的默认值不为空
	for _, template := range templates {
		if len(template.DefaultValues) == 0 {
			template.DefaultValues = model.JSONMap{}
		}
	}

	return templates, total, nil
}

func (d *templateDAO) UpdateTemplateStatus(ctx context.Context, id int, status int8) error {
	if id <= 0 {
		return ErrInvalidID
	}

	if !d.isValidStatus(status) {
		return ErrInvalidStatus
	}

	result := DBWithContext(ctx, d.db).Model(&model.WorkorderTemplate{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		d.logger.Error("更新模板状态失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("更新模板状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

func (d *templateDAO) IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("模板名称不能为空")
	}

	var count int64
	query := DBWithContext(ctx, d.db).Model(&model.WorkorderTemplate{}).Where("name = ?", name)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		d.logger.Error("检查模板名称是否存在失败", zap.Error(err))
		return false, fmt.Errorf("检查模板名称是否存在失败: %w", err)
	}

	return count > 0, nil
}

func (d *templateDAO) buildListQuery(db *gorm.DB, req *model.ListWorkorderTemplateReq) *gorm.DB {
	// 通用搜索
	if req.Search != "" {
		searchTerm := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", searchTerm, searchTerm)
	}

	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 分类筛选
	if req.CategoryID != nil && *req.CategoryID > 0 {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 流程筛选
	if req.ProcessID != nil && *req.ProcessID > 0 {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	// 表单设计筛选
	if req.FormDesignID != nil && *req.FormDesignID > 0 {
		db = db.Where("form_design_id = ?", *req.FormDesignID)
	}

	return db
}

// isDuplicateKeyError 判断是否为重复键错误
func (d *templateDAO) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate entry") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "unique constraint")
}

func (d *templateDAO) isValidStatus(status int8) bool {
	return status == model.TemplateStatusEnabled || status == model.TemplateStatusDisabled
}
