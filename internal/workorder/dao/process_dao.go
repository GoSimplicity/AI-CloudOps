package dao

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type ProcessDAO interface {
	CreateProcess(ctx context.Context, process *model.Process) error
	UpdateProcess(ctx context.Context, process *model.Process) error
	DeleteProcess(ctx context.Context, id int64) error
	ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error)
	GetProcess(ctx context.Context, id int64) (model.Process, error)
	PublishProcess(ctx context.Context, id int64) error
}

type processDAO struct {
	db *gorm.DB
}

func NewProcessDAO(db *gorm.DB) ProcessDAO {
	return &processDAO{
		db: db,
	}

}

// CreateProcess implements ProcessDAO.
func (p *processDAO) CreateProcess(ctx context.Context, process *model.Process) error {
	if err := p.db.WithContext(ctx).Create(process).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}
	return nil
}

// DeleteProcess implements ProcessDAO.
func (p *processDAO) DeleteProcess(ctx context.Context, id int64) error {
	result := p.db.WithContext(ctx).Delete(&model.Process{}, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("表单设计不存在")
		}
		return result.Error
	}
	return nil
}

// GetProcess implements ProcessDAO.
func (p *processDAO) GetProcess(ctx context.Context, id int64) (model.Process, error) {
	var process model.Process
	result := p.db.WithContext(ctx).First(&process, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return model.Process{}, fmt.Errorf("表单设计不存在")
		}
		return model.Process{}, result.Error
	}
	return process, nil
}

// ListProcess implements ProcessDAO.
func (p *processDAO) ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error) {
	var processes []model.Process
	db := p.db.WithContext(ctx).Model(&model.Process{})

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
	if err := db.Offset(offset).Limit(req.PageSize).Find(&processes).Error; err != nil {
		return nil, err
	}

	return processes, nil
}

// UpdateProcess implements ProcessDAO.
func (p *processDAO) UpdateProcess(ctx context.Context, process *model.Process) error {
	result := p.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", process.ID).Updates(process)
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

func (p *processDAO) PublishProcess(ctx context.Context, id int64) error {
	result := p.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", id).Update("status", 1)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("表单设计不存在")
		}
		return result.Error
	}
	return nil
}
