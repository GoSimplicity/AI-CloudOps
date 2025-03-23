package dao

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type FormDesignDAO interface {
	CreateFormDesign(ctx context.Context, formDesign *model.FormDesign) error
	UpdateFormDesign(ctx context.Context, formDesign *model.FormDesign) error
	DeleteFormDesign(ctx context.Context, id int64) error
	PublishFormDesign(ctx context.Context, id int64) error
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error)
	GetFormDesign(ctx context.Context, id int64) (*model.FormDesign, error)
	CloneFormDesign(ctx context.Context, id int64, name string) (*model.FormDesign, error)
}

type formDesignDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewFormDesignDAO(db *gorm.DB, l *zap.Logger) FormDesignDAO {
	return &formDesignDAO{
		db: db,
		l:  l,
	}
}

// CreateFormDesign implements FormDesignDAO.
func (f *formDesignDAO) CreateFormDesign(ctx context.Context, formDesign *model.FormDesign) error {

	if err := f.db.WithContext(ctx).Create(formDesign).Error; err != nil {
		f.l.Error("CreateFormDesign 创建表单失败", zap.Error(err))
		return err
	}
	return nil

}

// UpdateFormDesign implements FormDesignDAO.
func (f *formDesignDAO) UpdateFormDesign(ctx context.Context, formDesign *model.FormDesign) error {
	// 检查记录是否存在
	var existingFormDesign model.FormDesign
	if err := f.db.WithContext(ctx).Where("id = ? ", formDesign.ID).First(&existingFormDesign).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.l.Error("FormDesign记录不存在", zap.Int64("id", formDesign.ID))
			return fmt.Errorf("record not found with id %d", formDesign.ID)
		}
		f.l.Error("查询FormDesign失败", zap.Int64("id", formDesign.ID), zap.Error(err))
		return err
	}

	// 更新表单设计数据
	if err := f.db.WithContext(ctx).Model(&model.FormDesign{}).Where("id = ?", formDesign.ID).Updates(formDesign).Error; err != nil {
		f.l.Error("UpdateFormDesign 更新表单失败", zap.Int64("id", formDesign.ID), zap.Error(err))
		return err
	}

	// 返回成功
	return nil
}

// DeleteFormDesign implements FormDesignDAO.
func (f *formDesignDAO) DeleteFormDesign(ctx context.Context, id int64) error {
	// 检查记录是否存在
	var existingFormDesign model.FormDesign
	if err := f.db.WithContext(ctx).Where("id =? ", id).First(&existingFormDesign).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.l.Error("FormDesign记录不存在", zap.Int64("id", id))
		}
		f.l.Error("查询FormDesign失败", zap.Int64("id", id), zap.Error(err))
		return err
	}
	// 删除表单设计数据
	if err := f.db.WithContext(ctx).Delete(&model.FormDesign{}, id).Error; err != nil {
		f.l.Error("DeleteFormDesign 删除表单失败", zap.Int64("id", id), zap.Error(err))
		return err
	}
	// 返回成功
	return nil
}

// PublishFormDesign implements FormDesignDAO.
// PublishFormDesign implements FormDesignDAO.
func (f *formDesignDAO) PublishFormDesign(ctx context.Context, id int64) error {
	// 直接更新表单设计数据，将状态从草稿（status = 0）更新为已发布（status = 1）
	result := f.db.WithContext(ctx).Model(&model.FormDesign{}).
		Where("id = ? AND status = 0", id).
		Update("status", 1)

	if result.Error != nil {
		f.l.Error("PublishFormDesign 更新表单失败", zap.Int64("id", id), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		// 没有记录被更新，可能是记录不存在或者状态不是草稿
		f.l.Error("FormDesign记录不存在或状态不为草稿，无法发布", zap.Int64("id", id))
		return fmt.Errorf("record not found or status is not draft with id %d", id)
	}

	// 返回成功
	return nil
}

// GetFormDesign implements FormDesignDAO.
func (f *formDesignDAO) GetFormDesign(ctx context.Context, id int64) (*model.FormDesign, error) {
	var formDesign model.FormDesign
	// 查询表单设计
	if err := f.db.WithContext(ctx).First(&formDesign, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.l.Error("DetailFormDesign 未找到指定 ID 的表单设计", zap.Int64("id", id), zap.Error(err))
			return nil, err
		}
		f.l.Error("DetailFormDesign 查询表单设计失败", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	return &formDesign, nil
}

// CloneFormDesign implements FormDesignDAO.
func (f *formDesignDAO) CloneFormDesign(ctx context.Context, id int64, name string) (*model.FormDesign, error) {
	// 1. 根据 id 查询原始表单设计记录
	var originalFormDesign model.FormDesign
	if err := f.db.WithContext(ctx).First(&originalFormDesign, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.l.Error("CloneFormDesign 未找到指定 ID 的表单设计", zap.Int64("id", id), zap.Error(err))
			return nil, err
		}
		f.l.Error("CloneFormDesign 查询表单设计失败", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}

	// 2. 创建新的表单设计对象并复制字段
	clonedFormDesign := originalFormDesign
	clonedFormDesign.Name = name

	// 3. 重置新对象的 ID 字段
	clonedFormDesign.ID = 0

	// 4. 将新对象插入数据库
	if err := f.db.WithContext(ctx).Create(&clonedFormDesign).Error; err != nil {
		f.l.Error("CloneFormDesign 插入新表单设计失败", zap.Int64("original_id", id), zap.String("new_name", name), zap.Error(err))
		return nil, err
	}

	return &clonedFormDesign, nil
}
func (f *formDesignDAO) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error) {
	var formDesigns []model.FormDesign
	query := f.db.WithContext(ctx).Model(&model.FormDesign{})
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}
	if err := query.Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&formDesigns).Error; err != nil {
		f.l.Error("ListFormDesign 查询表单设计失败", zap.Error(err))
		return nil, err
	}
	return formDesigns, nil
}
