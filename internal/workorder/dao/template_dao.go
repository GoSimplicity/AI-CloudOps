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
	"encoding/json"
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
	GetTemplatesByProcessID(ctx context.Context, processID int) ([]*model.Template, error)
	GetTemplatesByCategory(ctx context.Context, categoryID int) ([]*model.Template, error)
	BatchUpdateStatus(ctx context.Context, ids []int, status int8) error
	GetTemplateCount(ctx context.Context) (int64, error)
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
		return fmt.Errorf("template cannot be nil")
	}

	t.logger.Debug("创建模板",
		zap.String("name", template.Name),
		zap.Int("process_id", template.ProcessID),
		zap.Int("creator_id", template.CreatorID))

	// 序列化默认值
	if err := t.serializeDefaultValues(template); err != nil {
		t.logger.Error("序列化默认值失败", zap.Error(err))
		return fmt.Errorf("序列化默认值失败: %w", err)
	}

	if err := t.db.WithContext(ctx).Create(template).Error; err != nil {
		t.logger.Error("创建模板失败", zap.Error(err), zap.String("name", template.Name))
		if t.isDuplicateKeyError(err) {
			return ErrTemplateNameExists
		}
		return fmt.Errorf("创建模板失败: %w", err)
	}

	t.logger.Info("模板创建成功", zap.Int("id", template.ID), zap.String("name", template.Name))
	return nil
}

// UpdateTemplate 更新模板
func (t *templateDAO) UpdateTemplate(ctx context.Context, template *model.Template) error {
	if template == nil || template.ID == 0 {
		return ErrInvalidID
	}

	t.logger.Debug("更新模板",
		zap.Int("id", template.ID),
		zap.String("name", template.Name))

	// 序列化默认值
	if err := t.serializeDefaultValues(template); err != nil {
		t.logger.Error("序列化默认值失败", zap.Error(err))
		return fmt.Errorf("序列化默认值失败: %w", err)
	}

	// 使用 Select 明确指定要更新的字段，避免零值问题
	result := t.db.WithContext(ctx).Model(&model.Template{}).
		Where("id = ?", template.ID).
		Select("name", "description", "process_id", "default_values", "icon", "status", "sort_order", "category_id", "updated_at").
		Updates(template)

	if result.Error != nil {
		t.logger.Error("更新模板失败", zap.Error(result.Error), zap.Int("id", template.ID))
		if t.isDuplicateKeyError(result.Error) {
			return ErrTemplateNameExists
		}
		return fmt.Errorf("更新模板失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		t.logger.Warn("更新模板：未找到记录", zap.Int("id", template.ID))
		return ErrTemplateNotFound
	}

	t.logger.Info("模板更新成功", zap.Int("id", template.ID))
	return nil
}

// DeleteTemplate 删除模板（软删除）
func (t *templateDAO) DeleteTemplate(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidID
	}

	t.logger.Debug("删除模板", zap.Int("id", id))

	result := t.db.WithContext(ctx).Delete(&model.Template{}, id)
	if result.Error != nil {
		t.logger.Error("删除模板失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除模板失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		t.logger.Warn("删除模板：未找到记录", zap.Int("id", id))
		return ErrTemplateNotFound
	}

	t.logger.Info("模板删除成功", zap.Int("id", id))
	return nil
}

// GetTemplate 获取单个模板
func (t *templateDAO) GetTemplate(ctx context.Context, id int) (*model.Template, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}

	t.logger.Debug("获取模板", zap.Int("id", id))

	var template model.Template
	err := t.db.WithContext(ctx).
		Preload("Process").
		Preload("Category").
		First(&template, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.logger.Warn("模板不存在", zap.Int("id", id))
			return nil, ErrTemplateNotFound
		}
		t.logger.Error("获取模板失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	// 反序列化默认值
	if err := t.deserializeDefaultValues(&template); err != nil {
		t.logger.Error("反序列化默认值失败", zap.Error(err))
		return nil, fmt.Errorf("反序列化默认值失败: %w", err)
	}

	return &template, nil
}

// ListTemplate 列表查询模板
func (t *templateDAO) ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	t.logger.Debug("查询模板列表",
		zap.Int("page", req.Page),
		zap.Int("size", req.Size),
		zap.String("search", req.Search))

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

	// 反序列化默认值
	for _, template := range templates {
		if err := t.deserializeDefaultValues(template); err != nil {
			t.logger.Error("反序列化默认值失败", zap.Error(err), zap.Int("template_id", template.ID))
			// 不中断整个查询，只记录错误
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

	t.logger.Debug("更新模板状态", zap.Int("id", id), zap.Int8("status", status))

	result := t.db.WithContext(ctx).Model(&model.Template{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		t.logger.Error("更新模板状态失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("更新模板状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		t.logger.Warn("更新模板状态：未找到记录", zap.Int("id", id))
		return ErrTemplateNotFound
	}

	t.logger.Info("模板状态更新成功", zap.Int("id", id), zap.Int8("status", status))
	return nil
}

// GetTemplatesByProcessID 根据流程ID获取模板列表
func (t *templateDAO) GetTemplatesByProcessID(ctx context.Context, processID int) ([]*model.Template, error) {
	if processID <= 0 {
		return nil, ErrInvalidID
	}

	t.logger.Debug("根据流程ID获取模板", zap.Int("process_id", processID))

	var templates []*model.Template
	err := t.db.WithContext(ctx).
		Where("process_id = ? AND status = ?", processID, 1). // 只获取启用的模板
		Order("sort_order ASC, created_at DESC").
		Find(&templates).Error

	if err != nil {
		t.logger.Error("根据流程ID获取模板失败", zap.Error(err), zap.Int("process_id", processID))
		return nil, fmt.Errorf("根据流程ID获取模板失败: %w", err)
	}

	return templates, nil
}

// GetTemplatesByCategory 根据分类ID获取模板列表
func (t *templateDAO) GetTemplatesByCategory(ctx context.Context, categoryID int) ([]*model.Template, error) {
	if categoryID <= 0 {
		return nil, ErrInvalidID
	}

	t.logger.Debug("根据分类ID获取模板", zap.Int("category_id", categoryID))

	var templates []*model.Template
	err := t.db.WithContext(ctx).
		Where("category_id = ? AND status = ?", categoryID, 1). // 只获取启用的模板
		Order("sort_order ASC, created_at DESC").
		Find(&templates).Error

	if err != nil {
		t.logger.Error("根据分类ID获取模板失败", zap.Error(err), zap.Int("category_id", categoryID))
		return nil, fmt.Errorf("根据分类ID获取模板失败: %w", err)
	}

	return templates, nil
}

// BatchUpdateStatus 批量更新状态
func (t *templateDAO) BatchUpdateStatus(ctx context.Context, ids []int, status int8) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids cannot be empty")
	}

	if !t.isValidStatus(status) {
		return ErrInvalidStatus
	}

	t.logger.Debug("批量更新模板状态", zap.Ints("ids", ids), zap.Int8("status", status))

	result := t.db.WithContext(ctx).Model(&model.Template{}).
		Where("id IN ?", ids).
		Update("status", status)

	if result.Error != nil {
		t.logger.Error("批量更新模板状态失败", zap.Error(result.Error))
		return fmt.Errorf("批量更新模板状态失败: %w", result.Error)
	}

	t.logger.Info("批量更新模板状态成功",
		zap.Ints("ids", ids),
		zap.Int8("status", status),
		zap.Int64("affected_rows", result.RowsAffected))

	return nil
}

// GetTemplateCount 获取模板总数
func (t *templateDAO) GetTemplateCount(ctx context.Context) (int64, error) {
	var count int64
	err := t.db.WithContext(ctx).Model(&model.Template{}).Count(&count).Error
	if err != nil {
		t.logger.Error("获取模板总数失败", zap.Error(err))
		return 0, fmt.Errorf("获取模板总数失败: %w", err)
	}
	return count, nil
}

// IsTemplateNameExists 检查模板名称是否存在
func (t *templateDAO) IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	if strings.TrimSpace(name) == "" {
		return false, fmt.Errorf("name cannot be empty")
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

// 辅助方法

// buildListQuery 构建列表查询条件
func (t *templateDAO) buildListQuery(db *gorm.DB, req *model.ListTemplateReq) *gorm.DB {
	// 搜索条件
	if req.Search != "" {
		db = db.Where("name LIKE ? OR description LIKE ?",
			"%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 状态筛选
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 分类筛选
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 流程筛选
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	return db
}

// serializeDefaultValues 序列化默认值
func (t *templateDAO) serializeDefaultValues(template *model.Template) error {
	if template.DefaultValues == "" {
		// 如果为空，设置空JSON对象
		template.DefaultValues = "{}"
		return nil
	}

	// 验证是否为有效的JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(template.DefaultValues), &temp); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	return nil
}

// deserializeDefaultValues 反序列化默认值
func (t *templateDAO) deserializeDefaultValues(template *model.Template) error {
	if template.DefaultValues == "" {
		template.DefaultValues = "{}"
		return nil
	}

	// 验证JSON格式
	var temp interface{}
	if err := json.Unmarshal([]byte(template.DefaultValues), &temp); err != nil {
		t.logger.Warn("模板默认值JSON格式无效",
			zap.Int("template_id", template.ID),
			zap.String("default_values", template.DefaultValues),
			zap.Error(err))
		template.DefaultValues = "{}"
	}

	return nil
}

// isDuplicateKeyError 判断是否为重复键错误
func (t *templateDAO) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate entry") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint")
}

// isValidStatus 验证状态值是否有效
func (t *templateDAO) isValidStatus(status int8) bool {
	return status == 0 || status == 1 // 0-禁用，1-启用
}
