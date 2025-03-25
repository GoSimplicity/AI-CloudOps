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
	DeleteTemplate(ctx context.Context, id int64) error
	ListTemplate(ctx context.Context, req model.ListTemplateReq) ([]model.Template, error)
	GetTemplate(ctx context.Context, id int64) (model.Template, error)
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
func (t *templateDAO) DeleteTemplate(ctx context.Context, id int64) error {
	if err := t.db.WithContext(ctx).Delete(&model.Template{}, id).Error; err != nil {
		return err
	}
	return nil
}

// GetTemplate implements TemplateDAO.
func (t *templateDAO) GetTemplate(ctx context.Context, id int64) (model.Template, error) {
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
