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

type TemplateDAO interface {
	CreateTemplate(ctx context.Context, template *model.Template) error
	UpdateTemplate(ctx context.Context, template *model.Template) error
	DeleteTemplate(ctx context.Context, id int) error
	GetTemplate(ctx context.Context, id int) (*model.Template, error)
	ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error)
	UpdateTemplateStatus(ctx context.Context, id int, status int8) error
	IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error)
}

type templateDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTemplateDAO(db *gorm.DB, logger *zap.Logger) TemplateDAO {
	return &templateDAO{
		db:     db,
		logger: logger,
	}
}

// CreateTemplate 创建模板
func (t *templateDAO) CreateTemplate(ctx context.Context, template *model.Template) error {
	if template == nil {
		return fmt.Errorf("模板不能为空")
	}

	// 设置默认值
	if template.DefaultValues == "" {
		template.DefaultValues = "{}"
	}

	if err := t.db.WithContext(ctx).Create(template).Error; err != nil {
		t.logger.Error("创建模板失败", zap.Error(err), zap.String("name", template.Name))
		if t.isDuplicateKeyError(err) {
			return ErrTemplateNameExists
		}
		return fmt.Errorf("创建模板失败: %w", err)
	}

	return nil
}

// UpdateTemplate 更新模板
func (t *templateDAO) UpdateTemplate(ctx context.Context, template *model.Template) error {
	if template == nil || template.ID <= 0 {
		return ErrInvalidID
	}

	// 设置默认值
	if template.DefaultValues == "" {
		template.DefaultValues = "{}"
	}

	// 明确指定要更新的字段
	updates := map[string]interface{}{
		"name":           template.Name,
		"description":    template.Description,
		"process_id":     template.ProcessID,
		"default_values": template.DefaultValues,
		"icon":           template.Icon,
		"status":         template.Status,
		"sort_order":     template.SortOrder,
		"category_id":    template.CategoryID,
	}

	result := t.db.WithContext(ctx).Model(&model.Template{}).
		Where("id = ?", template.ID).
		Updates(updates)

	if result.Error != nil {
		t.logger.Error("更新模板失败", zap.Error(result.Error), zap.Int("id", template.ID))
		if t.isDuplicateKeyError(result.Error) {
			return ErrTemplateNameExists
		}
		return fmt.Errorf("更新模板失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// DeleteTemplate 删除模板（软删除）
func (t *templateDAO) DeleteTemplate(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidID
	}

	result := t.db.WithContext(ctx).Delete(&model.Template{}, id)
	if result.Error != nil {
		t.logger.Error("删除模板失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除模板失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// GetTemplate 获取单个模板
func (t *templateDAO) GetTemplate(ctx context.Context, id int) (*model.Template, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}

	var template model.Template
	err := t.db.WithContext(ctx).
		Preload("Process").
		Preload("Category").
		First(&template, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		t.logger.Error("获取模板失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	// 确保默认值不为空
	if template.DefaultValues == "" {
		template.DefaultValues = "{}"
	}

	return &template, nil
}

// ListTemplate 列表查询模板
func (t *templateDAO) ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 { // 限制最大页面大小
		req.Size = 100
	}

	var templates []*model.Template
	var total int64

	db := t.db.WithContext(ctx).Model(&model.Template{})

	// 构建查询条件
	db = t.buildListQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		t.logger.Error("获取模板总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取模板总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Process").
		Preload("Category").
		Offset(offset).
		Limit(req.Size).
		Order("sort_order ASC, created_at DESC").
		Find(&templates).Error

	if err != nil {
		t.logger.Error("查询模板列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}

	// 确保所有模板的默认值不为空
	for _, template := range templates {
		if template.DefaultValues == "" {
			template.DefaultValues = "{}"
		}
	}

	return &model.ListResp[*model.Template]{
		Items: templates,
		Total: total,
	}, nil
}

// UpdateTemplateStatus 更新模板状态
func (t *templateDAO) UpdateTemplateStatus(ctx context.Context, id int, status int8) error {
	if id <= 0 {
		return ErrInvalidID
	}

	if !t.isValidStatus(status) {
		return ErrInvalidStatus
	}

	result := t.db.WithContext(ctx).Model(&model.Template{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		t.logger.Error("更新模板状态失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("更新模板状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// IsTemplateNameExists 检查模板名称是否存在
func (t *templateDAO) IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("模板名称不能为空")
	}

	var count int64
	query := t.db.WithContext(ctx).Model(&model.Template{}).Where("name = ?", name)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		t.logger.Error("检查模板名称是否存在失败", zap.Error(err))
		return false, fmt.Errorf("检查模板名称是否存在失败: %w", err)
	}

	return count > 0, nil
}

// buildListQuery 构建列表查询条件
func (t *templateDAO) buildListQuery(db *gorm.DB, req *model.ListTemplateReq) *gorm.DB {
	// 通用搜索
	if req.Search != "" {
		searchTerm := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", searchTerm, searchTerm)
	}

	// 按名称筛选
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		db = db.Where("name LIKE ?", "%"+strings.TrimSpace(*req.Name)+"%")
	}

	// 状态筛选
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

	return db
}

// isDuplicateKeyError 判断是否为重复键错误
func (t *templateDAO) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate entry") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "unique constraint")
}

// isValidStatus 验证状态值是否有效
func (t *templateDAO) isValidStatus(status int8) bool {
	return status == 0 || status == 1
}
