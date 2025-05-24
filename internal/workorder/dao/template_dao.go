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

	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap" // Added for logging
	"gorm.io/gorm"
)

type TemplateDAO interface {
	CreateTemplate(ctx context.Context, template *model.Template) error
	UpdateTemplate(ctx context.Context, template *model.Template) error
	DeleteTemplate(ctx context.Context, id int) error
	ListTemplate(ctx context.Context, req model.ListTemplateReq) ([]model.Template, error)
	GetTemplate(ctx context.Context, id int) (model.Template, error)
	UpdateTemplateStatus(ctx context.Context, id int, status int8) error // Added UpdateTemplateStatus
}

type templateDAO struct {
	db     *gorm.DB
	logger *zap.Logger // Added logger
}

func NewTemplateDAO(db *gorm.DB, logger *zap.Logger) TemplateDAO { // Updated constructor
	return &templateDAO{
		db:     db,
		logger: logger, // Set logger
	}
}

// CreateTemplate implements TemplateDAO.
func (t *templateDAO) CreateTemplate(ctx context.Context, template *model.Template) error {
	if err := t.db.WithContext(ctx).Create(template).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}
	return nil
}

// DeleteTemplate implements TemplateDAO.
func (t *templateDAO) DeleteTemplate(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).Delete(&model.Template{}, id).Error; err != nil {
		return err
	}
	return nil
}

// GetTemplate implements TemplateDAO.
func (t *templateDAO) GetTemplate(ctx context.Context, id int) (model.Template, error) {
	var template model.Template
	if err := t.db.WithContext(ctx).First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return template, fmt.Errorf("表单设计不存在")
		}
		return template, err
	}
	return template, nil
}

// ListTemplate implements TemplateDAO.
func (t *templateDAO) ListTemplate(ctx context.Context, req model.ListTemplateReq) ([]model.Template, error) {
	var templates []model.Template
	db := t.db.WithContext(ctx).Model(&model.Template{})

	// 搜索条件
	if req.Search != "" {
		db = db.Where("name LIKE ?", "%"+req.Search+"%")
	}

	// 状态筛选
	if req.Status != 0 {
		db = db.Where("status = ?", req.Status)
	}

	// 分页
	offset := (req.Page - 1) * req.Size // Changed PageSize to Size
	if err := db.Offset(offset).Limit(req.Size).Find(&templates).Error; err != nil { // Changed PageSize to Size
		return nil, err
	}

	return templates, nil
}

// UpdateTemplate implements TemplateDAO.
func (t *templateDAO) UpdateTemplate(ctx context.Context, template *model.Template) error {
	result := t.db.WithContext(ctx).Model(&model.Template{}).Where("id = ?", template.ID).Updates(template)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("表单设计不存在")
		}
		if result.Error == gorm.ErrDuplicatedKey {
			return fmt.Errorf("目标表单设计名称已存在")
		}
		return result.Error
	}
	return nil

}

// UpdateTemplateStatus 更新模板状态
func (t *templateDAO) UpdateTemplateStatus(ctx context.Context, id int, status int8) error {
	t.logger.Debug("开始更新模板状态 (DAO)", zap.Int("id", id), zap.Int8("status", status))
	result := t.db.WithContext(ctx).Model(&model.Template{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		t.logger.Error("更新模板状态失败 (DAO)", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("更新模板 (ID: %d) 状态失败: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		t.logger.Warn("更新模板状态：未找到记录 (DAO)", zap.Int("id", id))
		return fmt.Errorf("未找到模板 (ID: %d)", id) // Or return nil if "not found" is not an error for status update
	}
	t.logger.Debug("模板状态更新成功 (DAO)", zap.Int("id", id))
	return nil
}
