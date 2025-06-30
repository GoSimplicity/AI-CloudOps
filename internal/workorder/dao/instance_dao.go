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
	"strings"
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
	ErrTransferFailed     = errors.New("工单转移失败")
)

// 常量定义
const (
	DefaultBatchSize = 100
	DefaultPageSize  = 20
	MaxPageSize      = 1000
)

type InstanceDAO interface {
	// 实例CRUD
	CreateInstance(ctx context.Context, instance *model.Instance) error
	UpdateInstance(ctx context.Context, instance *model.Instance) error
	DeleteInstance(ctx context.Context, id int) error
	GetInstance(ctx context.Context, id int) (*model.Instance, error)
	GetInstanceByTemplateID(ctx context.Context, templateID int) ([]model.Instance, error)
	GetInstanceWithRelations(ctx context.Context, id int) (*model.Instance, error)
	ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.Instance], error)
	BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8) error
	GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) ([]*model.Instance, int64, error)
	GetOverdueInstances(ctx context.Context) ([]model.Instance, error)
	TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error
}

type instanceDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceDAO(db *gorm.DB, logger *zap.Logger) InstanceDAO {
	return &instanceDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstance 创建工单实例
func (d *instanceDAO) CreateInstance(ctx context.Context, instance *model.Instance) error {
	if instance == nil {
		return ErrInstanceNilPointer
	}

	// 数据验证
	if err := d.validateInstance(instance); err != nil {
		return fmt.Errorf("实例数据验证失败: %w", err)
	}

	// 处理零值时间
	d.normalizeTimeFields(instance)

	if err := d.db.WithContext(ctx).Create(instance).Error; err != nil {
		if d.isDuplicateKeyError(err) {
			d.logger.Warn("工单实例已存在", zap.String("title", instance.Title))
			return ErrInstanceExists
		}
		d.logger.Error("创建工单实例失败", zap.Error(err), zap.String("title", instance.Title))
		return fmt.Errorf("创建工单实例失败: %w", err)
	}

	d.logger.Info("创建工单实例成功", zap.Int("id", instance.ID), zap.String("title", instance.Title))
	return nil
}

// UpdateInstance 更新工单实例
func (d *instanceDAO) UpdateInstance(ctx context.Context, instance *model.Instance) error {
	if instance == nil {
		return ErrInstanceNilPointer
	}
	if instance.ID <= 0 {
		return ErrInstanceInvalidID
	}

	// 数据验证
	if err := d.validateInstance(instance); err != nil {
		return fmt.Errorf("实例数据验证失败: %w", err)
	}

	// 处理零值时间
	d.normalizeTimeFields(instance)

	updateData := d.buildUpdateData(instance)
	result := d.db.WithContext(ctx).
		Model(&model.Instance{}).
		Where("id = ?", instance.ID).
		Updates(updateData)

	if result.Error != nil {
		d.logger.Error("更新工单实例失败", zap.Error(result.Error), zap.Int("id", instance.ID))
		return fmt.Errorf("更新工单实例失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("工单实例不存在", zap.Int("id", instance.ID))
		return ErrInstanceNotFound
	}

	d.logger.Info("更新工单实例成功", zap.Int("id", instance.ID), zap.String("title", instance.Title))
	return nil
}

// DeleteInstance 删除工单实例
func (d *instanceDAO) DeleteInstance(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInstanceInvalidID
	}

	// 使用事务删除相关数据
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 验证工单是否存在
		var count int64
		if err := tx.Model(&model.Instance{}).Where("id = ?", id).Count(&count).Error; err != nil {
			return fmt.Errorf("查询工单实例失败: %w", err)
		}
		if count == 0 {
			return ErrInstanceNotFound
		}

		// 删除相关数据的顺序很重要，从子表到主表
		if err := d.deleteRelatedData(tx, id); err != nil {
			return err
		}

		// 删除工单实例
		if err := tx.Delete(&model.Instance{}, id).Error; err != nil {
			return fmt.Errorf("删除工单实例失败: %w", err)
		}

		return nil
	})

	if err != nil {
		d.logger.Error("删除工单实例失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	d.logger.Info("删除工单实例成功", zap.Int("id", id))
	return nil
}

// GetInstance 获取工单实例详情
func (d *instanceDAO) GetInstance(ctx context.Context, id int) (*model.Instance, error) {
	if id <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var instance model.Instance
	err := d.db.WithContext(ctx).First(&instance, id).Error

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

// GetInstanceWithRelations 获取工单实例及其关联数据
func (d *instanceDAO) GetInstanceWithRelations(ctx context.Context, id int) (*model.Instance, error) {
	if id <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var instance model.Instance
	err := d.db.WithContext(ctx).
		Preload("Template").
		Preload("Process").
		Preload("Category").
		First(&instance, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			d.logger.Warn("工单实例不存在", zap.Int("id", id))
			return nil, ErrInstanceNotFound
		}
		d.logger.Error("获取工单实例及关联数据失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单实例及关联数据失败: %w", err)
	}

	return &instance, nil
}

// ListInstance 获取工单实例列表
func (d *instanceDAO) ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.Instance], error) {
	if err := d.validateListRequest(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	var instances []model.Instance
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Instance{})
	db = d.buildInstanceListQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取工单实例总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单实例总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Template").
		Preload("Process").
		Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&instances).Error

	if err != nil {
		d.logger.Error("获取工单实例列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单实例列表失败: %w", err)
	}

	result := &model.ListResp[model.Instance]{
		Items: instances,
		Total: total,
	}

	d.logger.Info("获取工单实例列表成功",
		zap.Int("count", len(instances)),
		zap.Int64("total", total),
		zap.Int("page", req.Page),
		zap.Int("size", req.Size))

	return result, nil
}

// BatchUpdateInstanceStatus 批量更新工单状态
func (d *instanceDAO) BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8) error {
	if len(ids) == 0 {
		return errors.New("参数无效")
	}

	// 验证ID的有效性
	for _, id := range ids {
		if id <= 0 {
			return ErrInstanceInvalidID
		}
	}

	result := d.db.WithContext(ctx).
		Model(&model.Instance{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		d.logger.Error("批量更新工单状态失败", zap.Error(result.Error), zap.Ints("ids", ids), zap.Int8("status", status))
		return fmt.Errorf("批量更新工单状态失败: %w", result.Error)
	}

	d.logger.Info("批量更新工单状态成功", zap.Ints("ids", ids), zap.Int8("status", status), zap.Int64("affected", result.RowsAffected))
	return nil
}

// GetInstanceByTemplateID 获取关联工单
func (d *instanceDAO) GetInstanceByTemplateID(ctx context.Context, templateID int) ([]model.Instance, error) {
	var instances []model.Instance
	err := d.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&instances).Error
	if err != nil {
		d.logger.Error("获取关联工单失败", zap.Error(err), zap.Int("templateID", templateID))
		return nil, fmt.Errorf("获取关联工单失败: %w", err)
	}
	return instances, nil
}

// GetMyInstances 获取我的工单
func (d *instanceDAO) GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) ([]*model.Instance, int64, error) {
	if userID <= 0 {
		return nil, 0, errors.New("参数无效")
	}

	if err := d.validateMyInstanceRequest(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	var instances []*model.Instance
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Instance{})

	// 根据类型过滤
	switch req.Type {
	case "created":
		db = db.Where("creator_id = ?", userID)
	case "assigned":
		db = db.Where("assignee_id = ?", userID)
	default:
		db = db.Where("creator_id = ? OR assignee_id = ?", userID, userID)
	}

	// 构建其他查询条件
	db = d.buildMyInstanceQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取我的工单总数失败", zap.Error(err), zap.Int("userID", userID))
		return nil, 0, fmt.Errorf("获取我的工单总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Template").
		Preload("Process").
		Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&instances).Error

	if err != nil {
		d.logger.Error("获取我的工单列表失败", zap.Error(err), zap.Int("userID", userID))
		return nil, 0, fmt.Errorf("获取我的工单列表失败: %w", err)
	}

	return instances, total, nil
}

// GetOverdueInstances 获取超时工单
func (d *instanceDAO) GetOverdueInstances(ctx context.Context) ([]model.Instance, error) {
	var instances []model.Instance

	err := d.db.WithContext(ctx).
		Where("due_date < ? AND status NOT IN ?", time.Now(),
			[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}).
		Preload("Template").
		Preload("Process").
		Preload("Category").
		Find(&instances).Error

	if err != nil {
		d.logger.Error("获取超时工单失败", zap.Error(err))
		return nil, fmt.Errorf("获取超时工单失败: %w", err)
	}

	return instances, nil
}

// TransferInstance 转移工单
func (d *instanceDAO) TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error {
	if instanceID <= 0 || fromUserID <= 0 || toUserID <= 0 {
		return errors.New("参数无效")
	}

	if fromUserID == toUserID {
		return fmt.Errorf("转移人和接收人不能为同一人")
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 验证工单是否存在且属于fromUser
		var instance model.Instance
		if err := tx.Where("id = ? AND assignee_id = ?", instanceID, fromUserID).First(&instance).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("工单不存在或不属于当前用户")
			}
			return fmt.Errorf("查询工单失败: %w", err)
		}

		// 更新工单处理人
		if err := tx.Model(&instance).Updates(map[string]interface{}{
			"assignee_id": toUserID,
			"updated_at":  time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("更新工单处理人失败: %w", err)
		}

		// 创建转移记录
		flow := &model.InstanceFlow{
			InstanceID: instanceID,
			StepID:     "transfer",
			StepName:   "工单转移",
			Action:     "transfer",
			OperatorID: fromUserID,
			Comment:    comment,
			FromStepID: "current",
			ToStepID:   "current",
		}

		if err := tx.Create(flow).Error; err != nil {
			return fmt.Errorf("创建转移记录失败: %w", err)
		}

		d.logger.Info("工单转移成功",
			zap.Int("instanceID", instanceID),
			zap.Int("fromUserID", fromUserID),
			zap.Int("toUserID", toUserID))

		return nil
	})
}

// 私有辅助方法

// validateInstance 验证工单实例数据
func (d *instanceDAO) validateInstance(instance *model.Instance) error {
	if strings.TrimSpace(instance.Title) == "" {
		return fmt.Errorf("工单标题不能为空")
	}
	if len(instance.Title) > 200 {
		return fmt.Errorf("工单标题过长")
	}
	if instance.ProcessID <= 0 {
		return fmt.Errorf("流程ID无效")
	}
	if instance.CreatorID <= 0 {
		return fmt.Errorf("创建人ID无效")
	}
	return nil
}

// validateListRequest 验证列表请求参数
func (d *instanceDAO) validateListRequest(req *model.ListInstanceReq) error {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = DefaultPageSize
	}
	if req.Size > MaxPageSize {
		req.Size = MaxPageSize
	}
	return nil
}

// validateMyInstanceRequest 验证我的工单请求参数
func (d *instanceDAO) validateMyInstanceRequest(req *model.MyInstanceReq) error {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = DefaultPageSize
	}
	if req.Size > MaxPageSize {
		req.Size = MaxPageSize
	}
	return nil
}

// buildUpdateData 构建更新数据
func (d *instanceDAO) buildUpdateData(instance *model.Instance) map[string]interface{} {
	updateData := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if instance.Title != "" {
		updateData["title"] = instance.Title
	}
	if instance.FormData != nil {
		updateData["form_data"] = instance.FormData
	}
	if instance.CurrentStep != "" {
		updateData["current_step"] = instance.CurrentStep
	}
	updateData["status"] = instance.Status
	updateData["priority"] = instance.Priority
	updateData["description"] = instance.Description
	updateData["assignee_id"] = instance.AssigneeID
	updateData["completed_at"] = instance.CompletedAt
	updateData["due_date"] = instance.DueDate
	updateData["tags"] = instance.Tags
	updateData["process_data"] = instance.ProcessData

	return updateData
}

// deleteRelatedData 删除相关数据
func (d *instanceDAO) deleteRelatedData(tx *gorm.DB, instanceID int) error {
	// 删除相关的流转记录
	if err := tx.Where("instance_id = ?", instanceID).Delete(&model.InstanceFlow{}).Error; err != nil {
		return fmt.Errorf("删除工单流转记录失败: %w", err)
	}

	// 删除相关的评论
	if err := tx.Where("instance_id = ?", instanceID).Delete(&model.InstanceComment{}).Error; err != nil {
		return fmt.Errorf("删除工单评论失败: %w", err)
	}

	// 删除相关的附件记录
	if err := tx.Where("instance_id = ?", instanceID).Delete(&model.InstanceAttachment{}).Error; err != nil {
		return fmt.Errorf("删除工单附件记录失败: %w", err)
	}

	return nil
}

// buildInstanceListQuery 构建工单列表查询条件
func (d *instanceDAO) buildInstanceListQuery(db *gorm.DB, req *model.ListInstanceReq) *gorm.DB {
	// 搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("title LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// 状态过滤
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 优先级过滤
	if req.Priority != nil {
		db = db.Where("priority = ?", *req.Priority)
	}

	// 分类过滤
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 创建人过滤
	if req.CreatorID != nil {
		db = db.Where("creator_id = ?", *req.CreatorID)
	}

	// 处理人过滤
	if req.AssigneeID != nil {
		db = db.Where("assignee_id = ?", *req.AssigneeID)
	}

	// 流程过滤
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	// 模板过滤
	if req.TemplateID != nil {
		db = db.Where("template_id = ?", *req.TemplateID)
	}

	// 时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	} else if req.StartDate != nil {
		db = db.Where("created_at >= ?", req.StartDate)
	} else if req.EndDate != nil {
		db = db.Where("created_at <= ?", req.EndDate)
	}

	// 标签过滤
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			db = db.Where("tags LIKE ?", "%"+tag+"%")
		}
	}

	// 超时过滤
	if req.Overdue != nil {
		if *req.Overdue {
			db = db.Where("due_date < ? AND status NOT IN ?", time.Now(),
				[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected})
		} else {
			db = db.Where("due_date >= ? OR status IN ?", time.Now(),
				[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected})
		}
	}

	return db
}

// buildMyInstanceQuery 构建我的工单查询条件
func (d *instanceDAO) buildMyInstanceQuery(db *gorm.DB, req *model.MyInstanceReq) *gorm.DB {
	// 搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("title LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// 状态过滤
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 优先级过滤
	if req.Priority != nil {
		db = db.Where("priority = ?", *req.Priority)
	}

	// 分类过滤
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	// 流程过滤
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}

	// 时间范围过滤
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	} else if req.StartDate != nil {
		db = db.Where("created_at >= ?", req.StartDate)
	} else if req.EndDate != nil {
		db = db.Where("created_at <= ?", req.EndDate)
	}

	return db
}

// normalizeTimeFields 处理零值时间字段
func (d *instanceDAO) normalizeTimeFields(instance *model.Instance) {
	if instance.CompletedAt != nil && instance.CompletedAt.IsZero() {
		instance.CompletedAt = nil
	}
	if instance.DueDate != nil && instance.DueDate.IsZero() {
		instance.DueDate = nil
	}
}

// isDuplicateKeyError 判断是否为重复键错误
func (d *instanceDAO) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "duplicate key")
}
