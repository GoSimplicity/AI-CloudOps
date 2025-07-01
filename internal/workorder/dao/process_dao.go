package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrProcessNotFound      = fmt.Errorf("流程不存在")
	ErrProcessNameExists    = fmt.Errorf("流程名称已存在")
	ErrProcessCannotPublish = fmt.Errorf("流程状态不是草稿，无法发布")
	ErrProcessInvalidID     = fmt.Errorf("流程ID无效")
	ErrProcessNilPointer    = fmt.Errorf("流程对象为空")
	ErrProcessInUse         = fmt.Errorf("流程正在使用中，无法删除")
)

type ProcessDAO interface {
	CreateProcess(ctx context.Context, process *model.Process) error
	UpdateProcess(ctx context.Context, process *model.Process) error
	DeleteProcess(ctx context.Context, id int) error
	ListProcess(ctx context.Context, req *model.ListProcessReq) ([]*model.Process, int64, error)
	GetProcess(ctx context.Context, id int) (*model.Process, error)
	GetProcessWithRelations(ctx context.Context, id int) (*model.Process, error)
	PublishProcess(ctx context.Context, id int) error
	CloneProcess(ctx context.Context, id int, name string, creatorID int) (*model.Process, error)
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
func (d *processDAO) CreateProcess(ctx context.Context, process *model.Process) error {
	// 验证关联的表单设计是否存在
	if process.FormDesignID <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.FormDesign{}).
		Where("id = ?", process.FormDesignID).
		Count(&count).Error

	if err != nil {
		d.logger.Error("验证表单设计存在性失败", zap.Error(err), zap.Int("formDesignID", process.FormDesignID))
		return fmt.Errorf("验证表单设计存在性失败: %w", err)
	}

	if count == 0 {
		d.logger.Warn("关联的表单设计不存在", zap.Int("formDesignID", process.FormDesignID))
		return ErrFormDesignNotFound
	}

	if err := d.db.WithContext(ctx).Create(process).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "Duplicate entry") ||
			err == gorm.ErrDuplicatedKey {
			d.logger.Warn("流程名称已存在", zap.String("name", process.Name))
			return ErrProcessNameExists
		}
		d.logger.Error("创建流程失败", zap.Error(err), zap.String("name", process.Name))
		return fmt.Errorf("创建流程失败: %w", err)
	}

	return nil
}

// UpdateProcess 更新流程
func (d *processDAO) UpdateProcess(ctx context.Context, process *model.Process) error {
	if process.ID == 0 {
		return ErrProcessInvalidID
	}

	// 验证关联的表单设计是否存在
	if process.FormDesignID <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.FormDesign{}).
		Where("id = ?", process.FormDesignID).
		Count(&count).Error

	if err != nil {
		d.logger.Error("验证表单设计存在性失败", zap.Error(err), zap.Int("formDesignID", process.FormDesignID))
		return fmt.Errorf("验证表单设计存在性失败: %w", err)
	}

	if count == 0 {
		d.logger.Warn("关联的表单设计不存在", zap.Int("formDesignID", process.FormDesignID))
		return ErrFormDesignNotFound
	}

	updateData := map[string]interface{}{
		"name":           process.Name,
		"description":    process.Description,
		"form_design_id": process.FormDesignID,
		"definition":     process.Definition,
		"version":        process.Version,
		"status":         process.Status,
		"category_id":    process.CategoryID,
	}

	result := d.db.WithContext(ctx).
		Model(&model.Process{}).
		Where("id = ?", process.ID).
		Updates(updateData)

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed") ||
			strings.Contains(result.Error.Error(), "Duplicate entry") ||
			result.Error == gorm.ErrDuplicatedKey {
			d.logger.Warn("流程名称已存在", zap.String("name", process.Name), zap.Int("id", process.ID))
			return ErrProcessNameExists
		}
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

	// 使用事务确保数据一致性
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查流程是否正在使用中
		var instanceCount int64
		if err := tx.Model(&model.Instance{}).Where("process_id = ?", id).Count(&instanceCount).Error; err != nil {
			return fmt.Errorf("检查流程使用情况失败: %w", err)
		}

		if instanceCount > 0 {
			return ErrProcessInUse
		}

		// 检查是否有模板在使用此流程
		var templateCount int64
		if err := tx.Model(&model.Template{}).Where("process_id = ?", id).Count(&templateCount).Error; err != nil {
			return fmt.Errorf("检查模板使用情况失败: %w", err)
		}

		if templateCount > 0 {
			return ErrProcessInUse
		}

		// 删除流程
		result := tx.Delete(&model.Process{}, id)
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

// GetProcess 获取流程详情
func (d *processDAO) GetProcess(ctx context.Context, id int) (*model.Process, error) {
	if id <= 0 {
		return nil, ErrProcessInvalidID
	}

	var process model.Process

	err := d.db.WithContext(ctx).Preload("FormDesign").First(&process, id).Error

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

// GetProcessWithRelations 获取流程及其关联数据
func (d *processDAO) GetProcessWithRelations(ctx context.Context, id int) (*model.Process, error) {
	if id <= 0 {
		return nil, ErrProcessInvalidID
	}

	var process model.Process
	err := d.db.WithContext(ctx).
		Preload("FormDesign").
		Preload("Category").
		First(&process, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			d.logger.Warn("流程不存在", zap.Int("id", id))
			return nil, ErrProcessNotFound
		}
		d.logger.Error("获取流程及关联数据失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取流程及关联数据失败: %w", err)
	}

	return &process, nil
}

// ListProcess 获取流程列表
func (d *processDAO) ListProcess(ctx context.Context, req *model.ListProcessReq) ([]*model.Process, int64, error) {
	var processes []*model.Process
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Process{})

	// 构建查询条件 - 内联构建避免辅助函数
	if req.Search != "" {
		searchPattern := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("name LIKE ?", searchPattern)
	}

	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	if req.FormDesignID != nil && *req.FormDesignID > 0 {
		db = db.Where("form_design_id = ?", *req.FormDesignID)
	}

	// 计算总数
	err := db.Count(&total).Error
	if err != nil {
		d.logger.Error("获取流程列表总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取流程列表总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err = db.Preload("FormDesign").
		Preload("Category").
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

// PublishProcess 发布流程
func (d *processDAO) PublishProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrProcessInvalidID
	}

	result := d.db.WithContext(ctx).
		Model(&model.Process{}).
		Where("id = ? AND status = ?", id, model.ProcessStatusDraft).
		Updates(map[string]interface{}{
			"status":     model.ProcessStatusPublished,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		d.logger.Error("发布流程失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("发布流程失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("流程不存在或状态不是草稿", zap.Int("id", id))
		return ErrProcessCannotPublish
	}

	return nil
}

// CloneProcess 克隆流程
func (d *processDAO) CloneProcess(ctx context.Context, id int, name string, creatorID int) (*model.Process, error) {
	if id <= 0 {
		return nil, ErrProcessInvalidID
	}

	// 使用事务确保数据一致性
	var clonedProcess *model.Process
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取原始流程
		var originalProcess model.Process
		if err := tx.Where("id = ?", id).First(&originalProcess).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProcessNotFound
			}
			return fmt.Errorf("获取原始流程失败: %w", err)
		}

		// 创建克隆对象
		clonedProcess = &model.Process{
			Name:         name,
			Description:  originalProcess.Description,
			FormDesignID: originalProcess.FormDesignID,
			Definition:   originalProcess.Definition,
			Version:      originalProcess.Version,
			Status:       model.ProcessStatusDraft,
			CategoryID:   originalProcess.CategoryID,
			CreatorID:    creatorID,
		}

		// 创建克隆记录
		if err := tx.Create(clonedProcess).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
				strings.Contains(err.Error(), "Duplicate entry") ||
				err == gorm.ErrDuplicatedKey {
				return ErrProcessNameExists
			}
			return fmt.Errorf("创建克隆流程失败: %w", err)
		}

		return nil
	})

	if err != nil {
		d.logger.Error("克隆流程失败", zap.Error(err), zap.Int("originalID", id), zap.String("newName", name))
		return nil, err
	}

	return clonedProcess, nil
}

// CheckProcessNameExists 检查流程名称是否存在
func (d *processDAO) CheckProcessNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	var count int64
	db := d.db.WithContext(ctx).Model(&model.Process{}).Where("name = ?", name)

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
		return fmt.Errorf("流程必须包含至少一个步骤")
	}

	// 验证步骤
	stepIDs := make(map[string]bool)
	hasStartNode := false
	hasEndNode := false

	for _, step := range definition.Steps {
		if step.ID == "" {
			return fmt.Errorf("步骤ID不能为空")
		}
		if step.Name == "" {
			return fmt.Errorf("步骤名称不能为空")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("步骤ID重复: %s", step.ID)
		}
		stepIDs[step.ID] = true

		// 检查是否有开始和结束节点
		switch step.Type {
		case model.StepTypeStart:
			hasStartNode = true
		case model.StepTypeEnd:
			hasEndNode = true
		}
	}

	// 确保有开始和结束节点
	if !hasStartNode {
		return fmt.Errorf("流程必须包含一个开始节点")
	}
	if !hasEndNode {
		return fmt.Errorf("流程必须包含一个结束节点")
	}

	// 验证连接
	for _, conn := range definition.Connections {
		if conn.From == "" || conn.To == "" {
			return fmt.Errorf("连接的起始和目标步骤ID不能为空")
		}
		if !stepIDs[conn.From] {
			return fmt.Errorf("连接的起始步骤不存在: %s", conn.From)
		}
		if !stepIDs[conn.To] {
			return fmt.Errorf("连接的目标步骤不存在: %s", conn.To)
		}
	}

	return nil
}
