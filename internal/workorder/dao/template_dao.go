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
	"gorm.io/gorm"
)

type TemplateDAO interface {
	CreateTemplate(ctx context.Context, template *model.Template) error
	UpdateTemplate(ctx context.Context, template *model.Template) error
	DeleteTemplate(ctx context.Context, id int) error
	ListTemplate(ctx context.Context, req model.ListTemplateReq) ([]model.Template, error)
	GetTemplate(ctx context.Context, id int) (model.Template, error)
}

type templateDAO struct {
	db *gorm.DB
}

func NewTemplateDAO(db *gorm.DB) TemplateDAO {
	return &templateDAO{
		db: db,
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
	offset := (req.Page - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Find(&templates).Error; err != nil {
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
