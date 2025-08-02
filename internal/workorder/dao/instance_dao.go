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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrInstanceNotFound   = errors.New("工单实例不存在")
	ErrInstanceExists     = errors.New("工单实例已存在")
	ErrInstanceInvalidID  = errors.New("工单实例ID无效")
	ErrInstanceNilPointer = errors.New("工单实例对象为空")
)

type WorkorderInstanceDAO interface {
	CreateInstance(ctx context.Context, instance *model.WorkorderInstance) error
	UpdateInstance(ctx context.Context, instance *model.WorkorderInstance) error
	DeleteInstance(ctx context.Context, id int) error
	GetInstanceByID(ctx context.Context, id int) (*model.WorkorderInstance, error)
	GetInstanceBySerialNumber(ctx context.Context, serialNumber string) (*model.WorkorderInstance, error)
	ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) ([]*model.WorkorderInstance, int64, error)
	GenerateSerialNumber(ctx context.Context) (string, error)
	UpdateInstanceStatus(ctx context.Context, id int, status int8) error
	UpdateInstanceAssignee(ctx context.Context, id int, assigneeID *int) error
	ListInstanceByAssignee(ctx context.Context, assigneeID int, req *model.ListWorkorderInstanceReq) ([]*model.WorkorderInstance, int64, error)
}

type workorderInstanceDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewWorkorderInstanceDAO(db *gorm.DB, logger *zap.Logger) WorkorderInstanceDAO {
	return &workorderInstanceDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstance 创建工单实例
func (d *workorderInstanceDAO) CreateInstance(ctx context.Context, instance *model.WorkorderInstance) error {
	// 检查唯一性
	var count int64

	if err := d.db.WithContext(ctx).Model(&model.WorkorderInstance{}).Where("serial_number = ?", instance.SerialNumber).Count(&count).Error; err != nil {
		d.logger.Error("检查工单编号唯一性失败", zap.Error(err), zap.String("serial_number", instance.SerialNumber))
		return fmt.Errorf("检查工单编号唯一性失败: %w", err)
	}
	if count > 0 {
		d.logger.Warn("工单实例已存在", zap.String("serial_number", instance.SerialNumber))
		return ErrInstanceExists
	}
	if err := d.db.WithContext(ctx).Create(instance).Error; err != nil {
		d.logger.Error("创建工单实例失败", zap.Error(err), zap.String("title", instance.Title))
		return fmt.Errorf("创建工单实例失败: %w", err)
	}

	return nil
}

// UpdateInstance 更新工单实例
func (d *workorderInstanceDAO) UpdateInstance(ctx context.Context, instance *model.WorkorderInstance) error {
	if instance.ID <= 0 {
		d.logger.Error("更新工单实例失败: ID无效", zap.Any("instance", instance))
		return ErrInstanceInvalidID
	}

	// 只更新非零字段
	result := d.db.WithContext(ctx).
		Model(&model.WorkorderInstance{}).
		Where("id = ?", instance.ID).
		Updates(instance)
	if result.Error != nil {
		d.logger.Error("更新工单实例失败", zap.Error(result.Error), zap.Int("id", instance.ID))
		return fmt.Errorf("更新工单实例失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("工单实例不存在", zap.Int("id", instance.ID))
		return ErrInstanceNotFound
	}

	return nil
}

// DeleteInstance 删除工单实例
func (d *workorderInstanceDAO) DeleteInstance(ctx context.Context, id int) error {
	if id <= 0 {
		d.logger.Error("删除工单实例失败: ID无效", zap.Int("id", id))
		return ErrInstanceInvalidID
	}

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64

		if err := tx.Model(&model.WorkorderInstance{}).Where("id = ?", id).Count(&count).Error; err != nil {
			return fmt.Errorf("查询工单实例失败: %w", err)
		}

		if count == 0 {
			return ErrInstanceNotFound
		}

		if err := tx.Delete(&model.WorkorderInstance{}, id).Error; err != nil {
			return fmt.Errorf("删除工单实例失败: %w", err)
		}

		return nil
	})

	if err != nil {
		d.logger.Error("删除工单实例失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetInstanceByID 获取工单实例详情
func (d *workorderInstanceDAO) GetInstanceByID(ctx context.Context, id int) (*model.WorkorderInstance, error) {
	if id <= 0 {
		d.logger.Error("获取工单实例失败: ID无效", zap.Int("id", id))
		return nil, ErrInstanceInvalidID
	}

	var instance model.WorkorderInstance

	err := d.db.WithContext(ctx).
		Where("id = ?", id).
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("FlowLogs", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("Timeline", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		First(&instance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			d.logger.Warn("工单实例不存在", zap.Int("id", id))
			return nil, ErrInstanceNotFound
		}
		d.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	return &instance, nil
}

// ListInstance 获取工单实例列表
func (d *workorderInstanceDAO) ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) ([]*model.WorkorderInstance, int64, error) {
	var instances []*model.WorkorderInstance
	var total int64

	if req == nil {
		d.logger.Error("获取工单实例列表失败: 请求参数为空")
		return nil, 0, fmt.Errorf("请求参数为空")
	}

	// 验证分页参数
	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	db := d.db.WithContext(ctx).Model(&model.WorkorderInstance{})

	// 动态条件
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if req.Priority != nil {
		db = db.Where("priority = ?", *req.Priority)
	}

	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	if req.Search != "" {
		search := sanitizeSearchInput(req.Search)
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取工单实例总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取工单实例总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.Size
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&instances).Error
	if err != nil {
		d.logger.Error("获取工单实例列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取工单实例列表失败: %w", err)
	}

	return instances, total, nil
}

// GetInstanceBySerialNumber 根据工单编号获取工单实例
func (d *workorderInstanceDAO) GetInstanceBySerialNumber(ctx context.Context, serialNumber string) (*model.WorkorderInstance, error) {
	if serialNumber == "" {
		d.logger.Error("获取工单实例失败: 工单编号为空")
		return nil, fmt.Errorf("工单编号不能为空")
	}

	var instance model.WorkorderInstance
	err := d.db.WithContext(ctx).
		Where("serial_number = ?", serialNumber).
		Preload("Comments").
		Preload("FlowLogs").
		Preload("Timeline").
		First(&instance).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			d.logger.Warn("工单实例不存在", zap.String("serial_number", serialNumber))
			return nil, ErrInstanceNotFound
		}
		d.logger.Error("获取工单实例失败", zap.Error(err), zap.String("serial_number", serialNumber))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	return &instance, nil
}

// GenerateSerialNumber 生成工单编号
func (d *workorderInstanceDAO) GenerateSerialNumber(ctx context.Context) (string, error) {
	now := time.Now()
	prefix := "WO" + now.Format("20060102")

	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.WorkorderInstance{}).
		Where("serial_number LIKE ?", prefix+"%").
		Count(&count).Error

	if err != nil {
		d.logger.Error("生成工单编号失败", zap.Error(err))
		return "", fmt.Errorf("生成工单编号失败: %w", err)
	}

	serialNumber := fmt.Sprintf("%s%04d", prefix, count+1)
	return serialNumber, nil
}

// UpdateInstanceStatus 更新工单状态
func (d *workorderInstanceDAO) UpdateInstanceStatus(ctx context.Context, id int, status int8) error {
	if id <= 0 {
		return ErrInstanceInvalidID
	}

	result := d.db.WithContext(ctx).
		Model(&model.WorkorderInstance{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		d.logger.Error("更新工单状态失败", zap.Error(result.Error), zap.Int("id", id), zap.Int8("status", status))
		return fmt.Errorf("更新工单状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrInstanceNotFound
	}

	return nil
}

// UpdateInstanceAssignee 更新工单处理人
func (d *workorderInstanceDAO) UpdateInstanceAssignee(ctx context.Context, id int, assigneeID *int) error {
	if id <= 0 {
		return ErrInstanceInvalidID
	}

	result := d.db.WithContext(ctx).
		Model(&model.WorkorderInstance{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"assignee_id": assigneeID,
			"updated_at":  time.Now(),
		})

	if result.Error != nil {
		d.logger.Error("更新工单处理人失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("更新工单处理人失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrInstanceNotFound
	}

	return nil
}

// ListInstanceByAssignee 根据处理人获取工单列表
func (d *workorderInstanceDAO) ListInstanceByAssignee(ctx context.Context, assigneeID int, req *model.ListWorkorderInstanceReq) ([]*model.WorkorderInstance, int64, error) {
	var instances []*model.WorkorderInstance
	var total int64

	if assigneeID <= 0 {
		return nil, 0, fmt.Errorf("处理人ID无效")
	}

	if req == nil {
		req = &model.ListWorkorderInstanceReq{
			ListReq: model.ListReq{Page: 1, Size: 10},
		}
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	db := d.db.WithContext(ctx).Model(&model.WorkorderInstance{}).Where("assignee_id = ?", assigneeID)

	// 动态条件
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.Priority != nil {
		db = db.Where("priority = ?", *req.Priority)
	}
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}
	if req.Search != "" {
		search := sanitizeSearchInput(req.Search)
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取处理人工单总数失败", zap.Error(err), zap.Int("assignee_id", assigneeID))
		return nil, 0, fmt.Errorf("获取处理人工单总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.Size
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&instances).Error
	if err != nil {
		d.logger.Error("获取处理人工单列表失败", zap.Error(err), zap.Int("assignee_id", assigneeID))
		return nil, 0, fmt.Errorf("获取处理人工单列表失败: %w", err)
	}

	return instances, total, nil
}
