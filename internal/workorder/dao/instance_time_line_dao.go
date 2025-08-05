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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderInstanceTimelineDAO interface {
	Create(ctx context.Context, timeline *model.WorkorderInstanceTimeline) error
	GetByID(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error)
	GetByInstanceID(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceTimeline, error)
	List(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) ([]*model.WorkorderInstanceTimeline, int64, error)
}

type instanceTimeLineDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceTimeLineDAO(db *gorm.DB, logger *zap.Logger) WorkorderInstanceTimelineDAO {
	return &instanceTimeLineDAO{
		db:     db,
		logger: logger,
	}
}

// Create 创建时间线记录
func (i *instanceTimeLineDAO) Create(ctx context.Context, timeline *model.WorkorderInstanceTimeline) error {
	if timeline == nil {
		return fmt.Errorf("时间线记录不能为空")
	}

	if err := i.db.WithContext(ctx).Create(timeline).Error; err != nil {
		i.logger.Error("创建时间线记录失败", zap.Error(err), zap.Int("instanceID", timeline.InstanceID))
		return fmt.Errorf("创建时间线记录失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取时间线记录
func (i *instanceTimeLineDAO) GetByID(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error) {
	if id <= 0 {
		return nil, fmt.Errorf("时间线记录ID无效")
	}

	var timeline model.WorkorderInstanceTimeline
	err := i.db.WithContext(ctx).
		Where("id = ?", id).
		First(&timeline).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("时间线记录不存在")
		}
		i.logger.Error("获取时间线记录失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取时间线记录失败: %w", err)
	}

	return &timeline, nil
}

// GetByInstanceID 根据工单ID获取时间线记录列表
func (i *instanceTimeLineDAO) GetByInstanceID(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceTimeline, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	var timelines []*model.WorkorderInstanceTimeline
	err := i.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at DESC").
		Find(&timelines).Error

	if err != nil {
		i.logger.Error("获取工单时间线记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单时间线记录失败: %w", err)
	}

	return timelines, nil
}

// List 获取时间线记录列表
func (i *instanceTimeLineDAO) List(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) ([]*model.WorkorderInstanceTimeline, int64, error) {
	var timelines []*model.WorkorderInstanceTimeline
	var total int64

	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	db := i.db.WithContext(ctx).Model(&model.WorkorderInstanceTimeline{})

	// 构建查询条件
	if req.InstanceID != nil {
		db = db.Where("instance_id = ?", *req.InstanceID)
	}
	if req.Action != nil {
		db = db.Where("action = ?", *req.Action)
	}
	if req.StartDate != nil {
		db = db.Where("created_at >= ?", *req.StartDate)
	}
	if req.EndDate != nil {
		db = db.Where("created_at <= ?", *req.EndDate)
	}
	if req.Search != "" {
		searchTerm := sanitizeSearchInput(req.Search)
		db = db.Where("comment LIKE ? OR operator_name LIKE ?", "%"+searchTerm+"%", "%"+searchTerm+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		i.logger.Error("获取时间线记录总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取时间线记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&timelines).Error

	if err != nil {
		i.logger.Error("获取时间线记录列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取时间线记录列表失败: %w", err)
	}

	return timelines, total, nil
}

// UpdateInstanceTimeLine 更新时间线记录
func (i *instanceTimeLineDAO) UpdateInstanceTimeLine(ctx context.Context, timeline *model.WorkorderInstanceTimeline) error {
	if timeline == nil || timeline.ID <= 0 {
		return fmt.Errorf("时间线记录ID无效")
	}

	result := i.db.WithContext(ctx).
		Model(&model.WorkorderInstanceTimeline{}).
		Where("id = ?", timeline.ID).
		Updates(map[string]any{
			"comment": timeline.Comment,
		})

	if result.Error != nil {
		i.logger.Error("更新时间线记录失败", zap.Error(result.Error), zap.Int("id", timeline.ID))
		return fmt.Errorf("更新时间线记录失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("时间线记录不存在")
	}

	return nil
}

// DeleteInstanceTimeLine 删除时间线记录
func (i *instanceTimeLineDAO) DeleteInstanceTimeLine(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("时间线记录ID无效")
	}

	result := i.db.WithContext(ctx).Delete(&model.WorkorderInstanceTimeline{}, id)
	if result.Error != nil {
		i.logger.Error("删除时间线记录失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除时间线记录失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("时间线记录不存在")
	}

	return nil
}

// GetInstanceTimeLine 获取时间线记录详情
func (i *instanceTimeLineDAO) GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error) {
	if id <= 0 {
		return nil, fmt.Errorf("时间线记录ID无效")
	}

	var timeline model.WorkorderInstanceTimeline
	err := i.db.WithContext(ctx).
		Where("id = ?", id).
		First(&timeline).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("时间线记录不存在")
		}
		i.logger.Error("获取时间线记录失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取时间线记录失败: %w", err)
	}

	return &timeline, nil
}

// ListInstanceTimeLine 获取时间线记录列表
func (i *instanceTimeLineDAO) ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) ([]*model.WorkorderInstanceTimeline, int64, error) {
	var timelines []*model.WorkorderInstanceTimeline
	var total int64

	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	db := i.db.WithContext(ctx).Model(&model.WorkorderInstanceTimeline{})

	// 构建查询条件
	if req.InstanceID != nil {
		db = db.Where("instance_id = ?", *req.InstanceID)
	}
	if req.Action != nil {
		db = db.Where("action = ?", *req.Action)
	}
	if req.StartDate != nil {
		db = db.Where("created_at >= ?", *req.StartDate)
	}
	if req.EndDate != nil {
		db = db.Where("created_at <= ?", *req.EndDate)
	}
	if req.Search != "" {
		searchTerm := sanitizeSearchInput(req.Search)
		db = db.Where("comment LIKE ? OR operator_name LIKE ?", "%"+searchTerm+"%", "%"+searchTerm+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		i.logger.Error("获取时间线记录总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取时间线记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&timelines).Error

	if err != nil {
		i.logger.Error("获取时间线记录列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取时间线记录列表失败: %w", err)
	}

	return timelines, total, nil
}
