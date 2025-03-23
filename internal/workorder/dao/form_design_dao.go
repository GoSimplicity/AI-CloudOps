package dao

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type FormDesignDAO interface {
	CreateFormDesign(ctx context.Context, formDesign *model.FormDesign) error
	UpdateFormDesign(ctx context.Context, formDesign *model.FormDesign) error
	DeleteFormDesign(ctx context.Context, id int64) error
	PublishFormDesign(ctx context.Context, id int64) error
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error)
	GetFormDesign(ctx context.Context, id int64) (*model.FormDesign, error)
	CloneFormDesign(ctx context.Context, name string) error
}

type formDesignDAO struct {
	db *gorm.DB
}

func NewFormDesignDAO(db *gorm.DB) FormDesignDAO {
	return &formDesignDAO{
		db: db,
	}
}

// CreateFormDesign 创建表单设计
func (f *formDesignDAO) CreateFormDesign(ctx context.Context, formDesign *model.FormDesign) error {
	if err := f.db.WithContext(ctx).Create(formDesign).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}
	return nil
}

// UpdateFormDesign 更新表单设计
func (f *formDesignDAO) UpdateFormDesign(ctx context.Context, formDesign *model.FormDesign) error {
	result := f.db.WithContext(ctx).Model(&model.FormDesign{}).Where("id = ?", formDesign.ID).Updates(map[string]interface{}{
		"name":        formDesign.Name,
		"description": formDesign.Description,
		"schema":      formDesign.Schema,
		"version":     formDesign.Version,
		"status":      formDesign.Status,
		"category_id": formDesign.CategoryID,
	})

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("表单设计不存在")
		}
		if result.Error == gorm.ErrDuplicatedKey {
			return fmt.Errorf("目标表单设计名称已存在")
		}
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("表单设计不存在")
	}

	return nil
}

// DeleteFormDesign 删除表单设计
func (f *formDesignDAO) DeleteFormDesign(ctx context.Context, id int64) error {
	result := f.db.WithContext(ctx).Delete(&model.FormDesign{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("表单设计不存在")
	}

	return nil
}

// PublishFormDesign 发布表单设计
func (f *formDesignDAO) PublishFormDesign(ctx context.Context, id int64) error {
	result := f.db.WithContext(ctx).Model(&model.FormDesign{}).
		Where("id = ? AND status = 0", id).
		Updates(map[string]interface{}{
			"status": 1,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("表单设计不存在或状态不是草稿，无法发布")
	}

	return nil
}

// GetFormDesign 获取表单设计
func (f *formDesignDAO) GetFormDesign(ctx context.Context, id int64) (*model.FormDesign, error) {
	var formDesign model.FormDesign

	if err := f.db.WithContext(ctx).First(&formDesign, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("表单设计不存在")
		}
		return nil, err
	}

	return &formDesign, nil
}

// CloneFormDesign 克隆表单设计
func (f *formDesignDAO) CloneFormDesign(ctx context.Context, name string) error {
	var originalFormDesign model.FormDesign

	if err := f.db.WithContext(ctx).First(&originalFormDesign).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("表单设计不存在")
		}
		return err
	}

	clonedFormDesign := originalFormDesign
	clonedFormDesign.ID = 0
	clonedFormDesign.Name = name

	if err := f.db.WithContext(ctx).Create(&clonedFormDesign).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}

	return nil
}

// ListFormDesign 获取表单设计列表
func (f *formDesignDAO) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error) {
	var formDesigns []model.FormDesign
	db := f.db.WithContext(ctx).Model(&model.FormDesign{})

	if req.Search != "" {
		db = db.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.Status != 0 {
		db = db.Where("status = ?", req.Status)
	}

	offset := (req.Page - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Find(&formDesigns).Error; err != nil {
		return nil, err
	}

	return formDesigns, nil
}
