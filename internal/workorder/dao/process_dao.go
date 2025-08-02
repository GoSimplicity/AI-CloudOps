package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrProcessNotFound   = fmt.Errorf("流程不存在")
	ErrProcessNameExists = fmt.Errorf("流程名称已存在")
	ErrProcessInvalidID  = fmt.Errorf("流程ID无效")
	ErrProcessInUse      = fmt.Errorf("流程正在使用中，无法删除")
)

type ProcessDAO interface {
	CreateProcess(ctx context.Context, process *model.WorkorderProcess) error
	UpdateProcess(ctx context.Context, process *model.WorkorderProcess) error
	DeleteProcess(ctx context.Context, id int) error
	ListProcess(ctx context.Context, req *model.ListWorkorderProcessReq) ([]*model.WorkorderProcess, int64, error)
	GetProcessByID(ctx context.Context, id int) (*model.WorkorderProcess, error)
	CheckProcessNameExists(ctx context.Context, name string, excludeID ...int) (bool, error)
	ValidateProcessDefinition(ctx context.Context, definition *model.ProcessDefinition) error
}

type processDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewProcessDAO(db *gorm.DB, logger *zap.Logger) ProcessDAO {
	return &processDAO{
		db:     db,
		logger: logger,
	}
}

// CreateProcess 创建流程
func (d *processDAO) CreateProcess(ctx context.Context, process *model.WorkorderProcess) error {
	if process.Name == "" {
		return fmt.Errorf("流程名称不能为空")
	}
	// 检查名称唯一性
	exists, err := d.CheckProcessNameExists(ctx, process.Name)
	if err != nil {
		return err
	}
	if exists {
		return ErrProcessNameExists
	}
	if err := d.db.WithContext(ctx).Create(process).Error; err != nil {
		d.logger.Error("创建流程失败", zap.Error(err), zap.String("name", process.Name))
		return fmt.Errorf("创建流程失败: %w", err)
	}
	return nil
}

// UpdateProcess 更新流程
func (d *processDAO) UpdateProcess(ctx context.Context, process *model.WorkorderProcess) error {
	if process.ID <= 0 {
		return ErrProcessInvalidID
	}
	// 检查名称唯一性（排除自己）
	exists, err := d.CheckProcessNameExists(ctx, process.Name, process.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrProcessNameExists
	}
	updateData := map[string]any{
		"name":           process.Name,
		"description":    process.Description,
		"form_design_id": process.FormDesignID,
		"definition":     process.Definition,
		"status":         process.Status,
		"category_id":    process.CategoryID,
		"tags":           process.Tags,
		"is_default":     process.IsDefault,
	}

	result := d.db.WithContext(ctx).
		Model(&model.WorkorderProcess{}).
		Where("id = ?", process.ID).
		Updates(updateData)

	if result.Error != nil {
		d.logger.Error("更新流程失败", zap.Error(result.Error), zap.Int("id", process.ID))
		return fmt.Errorf("更新流程失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("流程不存在", zap.Int("id", process.ID))
		return ErrProcessNotFound
	}

	return nil
}

// DeleteProcess 删除流程
func (d *processDAO) DeleteProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrProcessInvalidID
	}

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var instanceCount int64

		if err := tx.Model(&model.WorkorderInstance{}).Where("process_id = ?", id).Count(&instanceCount).Error; err != nil {
			return fmt.Errorf("检查流程使用情况失败: %w", err)
		}

		if instanceCount > 0 {
			return ErrProcessInUse
		}

		var templateCount int64
		if err := tx.Model(&model.WorkorderTemplate{}).Where("process_id = ?", id).Count(&templateCount).Error; err != nil {
			return fmt.Errorf("检查模板使用情况失败: %w", err)
		}

		if templateCount > 0 {
			return ErrProcessInUse
		}

		result := tx.Delete(&model.WorkorderProcess{}, id)
		if result.Error != nil {
			return fmt.Errorf("删除流程失败: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return ErrProcessNotFound
		}

		return nil
	})

	if err != nil {
		d.logger.Error("删除流程失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetProcessByID 获取流程详情
func (d *processDAO) GetProcessByID(ctx context.Context, id int) (*model.WorkorderProcess, error) {
	if id <= 0 {
		return nil, ErrProcessInvalidID
	}

	var process model.WorkorderProcess

	err := d.db.WithContext(ctx).Preload("Category").Preload("FormDesign").Where("id = ?", id).First(&process).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			d.logger.Warn("流程不存在", zap.Int("id", id))
			return nil, ErrProcessNotFound
		}
		d.logger.Error("获取流程失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取流程失败: %w", err)
	}

	return &process, nil
}

// ListProcess 获取流程列表
func (d *processDAO) ListProcess(ctx context.Context, req *model.ListWorkorderProcessReq) ([]*model.WorkorderProcess, int64, error) {
	var processes []*model.WorkorderProcess
	var total int64

	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	db := d.db.WithContext(ctx).Model(&model.WorkorderProcess{})

	if req.Search != "" {
		searchTerm := sanitizeSearchInput(req.Search)
		db = db.Where("name LIKE ?", "%"+searchTerm+"%")
	}

	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	if req.FormDesignID != nil && *req.FormDesignID > 0 {
		db = db.Where("form_design_id = ?", *req.FormDesignID)
	}

	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if req.IsDefault != nil {
		db = db.Where("is_default = ?", *req.IsDefault)
	}

	// 计算总数
	err := db.Count(&total).Error
	if err != nil {
		d.logger.Error("获取流程列表总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取流程列表总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.Size
	err = db.Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&processes).Error

	if err != nil {
		d.logger.Error("获取流程列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取流程列表失败: %w", err)
	}

	return processes, total, nil
}

// CheckProcessNameExists 检查流程名称是否存在
func (d *processDAO) CheckProcessNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("流程名称不能为空")
	}

	var count int64

	db := d.db.WithContext(ctx).Model(&model.WorkorderProcess{}).Where("name = ?", name)

	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}

	if err := db.Count(&count).Error; err != nil {
		d.logger.Error("检查流程名称是否存在失败", zap.Error(err), zap.String("name", name))
		return false, fmt.Errorf("检查流程名称是否存在失败: %w", err)
	}

	return count > 0, nil
}

// ValidateProcessDefinition 验证流程定义
func (d *processDAO) ValidateProcessDefinition(ctx context.Context, definition *model.ProcessDefinition) error {
	if len(definition.Steps) == 0 {
		return fmt.Errorf("流程定义必须包含至少一个步骤")
	}

	if len(definition.Connections) == 0 {
		return fmt.Errorf("流程定义必须包含至少一条连接")
	}

	stepIDSet := make(map[string]struct{})
	startCount := 0
	endCount := 0

	for i, step := range definition.Steps {
		if step.ID == "" {
			return fmt.Errorf("第%d个步骤ID不能为空", i+1)
		}

		if step.Type == "" {
			return fmt.Errorf("第%d个步骤类型不能为空", i+1)
		}

		if step.Name == "" {
			return fmt.Errorf("第%d个步骤名称不能为空", i+1)
		}

		if _, exists := stepIDSet[step.ID]; exists {
			return fmt.Errorf("步骤ID重复: %s", step.ID)
		}

		stepIDSet[step.ID] = struct{}{}
		switch step.Type {
		case model.ProcessStepTypeStart:
			startCount++
		case model.ProcessStepTypeEnd:
			endCount++
		}
	}

	if startCount == 0 {
		return fmt.Errorf("流程必须包含一个开始步骤")
	}

	if startCount > 1 {
		return fmt.Errorf("流程只能有一个开始步骤")
	}

	if endCount == 0 {
		return fmt.Errorf("流程必须包含至少一个结束步骤")
	}

	// 校验连线
	for i, conn := range definition.Connections {
		if conn.From == "" {
			return fmt.Errorf("第%d条连接的来源步骤ID不能为空", i+1)
		}

		if conn.To == "" {
			return fmt.Errorf("第%d条连接的目标步骤ID不能为空", i+1)
		}

		if _, ok := stepIDSet[conn.From]; !ok {
			return fmt.Errorf("第%d条连接的来源步骤ID不存在: %s", i+1, conn.From)
		}

		if _, ok := stepIDSet[conn.To]; !ok {
			return fmt.Errorf("第%d条连接的目标步骤ID不存在: %s", i+1, conn.To)
		}
	}

	return nil
}
