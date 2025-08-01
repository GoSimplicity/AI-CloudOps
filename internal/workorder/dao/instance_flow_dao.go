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

// 错误定义
var (
	ErrFlowNilPointer       = errors.New("流程记录对象为空")
	ErrInstanceFlowNotFound = errors.New("工单流程记录不存在")
)

type WorkorderInstanceFlowDAO interface {
	Create(ctx context.Context, flow *model.WorkorderInstanceFlow) error
	GetByInstanceID(ctx context.Context, instanceID int) ([]model.WorkorderInstanceFlow, error)
	GetByID(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error)
	List(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) ([]model.WorkorderInstanceFlow, int64, error)
}

type instanceFlowDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceFlowDAO(db *gorm.DB, logger *zap.Logger) WorkorderInstanceFlowDAO {
	return &instanceFlowDAO{
		db:     db,
		logger: logger,
	}
}

// Create 创建工单流程记录
func (d *instanceFlowDAO) Create(ctx context.Context, flow *model.WorkorderInstanceFlow) error {
	if flow == nil {
		return ErrFlowNilPointer
	}

	if err := d.validateFlow(flow); err != nil {
		return fmt.Errorf("流程记录验证失败: %w", err)
	}

	if err := d.db.WithContext(ctx).Create(flow).Error; err != nil {
		d.logger.Error("创建工单流程记录失败", zap.Error(err), zap.Int("instanceID", flow.InstanceID))
		return fmt.Errorf("创建工单流程记录失败: %w", err)
	}

	d.logger.Info("创建工单流程记录成功", zap.Int("id", flow.ID), zap.Int("instanceID", flow.InstanceID))
	return nil
}

// GetByInstanceID 获取工单流程记录
func (d *instanceFlowDAO) GetByInstanceID(ctx context.Context, instanceID int) ([]model.WorkorderInstanceFlow, error) {
	if instanceID <= 0 {
		return nil, errors.New("工单实例ID无效")
	}

	var flows []model.WorkorderInstanceFlow
	err := d.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at ASC").
		Find(&flows).Error

	if err != nil {
		d.logger.Error("获取工单流程记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单流程记录失败: %w", err)
	}

	return flows, nil
}

// GetByID 根据ID获取工单流程记录
func (d *instanceFlowDAO) GetByID(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error) {
	if id <= 0 {
		return nil, errors.New("工单实例ID无效")
	}

	var flow model.WorkorderInstanceFlow
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&flow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceFlowNotFound
		}
		d.logger.Error("根据ID获取工单流程记录失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("根据ID获取工单流程记录失败: %w", err)
	}

	return &flow, nil
}

// List 分页获取工单流程记录列表
func (d *instanceFlowDAO) List(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) ([]model.WorkorderInstanceFlow, int64, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("请求参数为空")
	}

	query := d.db.WithContext(ctx).Model(&model.WorkorderInstanceFlow{})

	// 添加过滤条件
	if req.InstanceID != nil {
		query = query.Where("instance_id = ?", *req.InstanceID)
	}
	if req.StepID != nil {
		query = query.Where("step_id = ?", *req.StepID)
	}
	if req.Action != nil {
		query = query.Where("action = ?", *req.Action)
	}
	if req.OperatorID != nil {
		query = query.Where("operator_id = ?", *req.OperatorID)
	}
	if req.Result != nil {
		query = query.Where("result = ?", *req.Result)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		d.logger.Error("获取工单流程记录总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取工单流程记录总数失败: %w", err)
	}

	// 分页查询
	var flows []model.WorkorderInstanceFlow
	offset := (req.Page - 1) * req.Size
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&flows).Error

	if err != nil {
		d.logger.Error("分页获取工单流程记录失败", zap.Error(err))
		return nil, 0, fmt.Errorf("分页获取工单流程记录失败: %w", err)
	}

	return flows, total, nil
}

// validateFlow 验证流程记录数据
func (d *instanceFlowDAO) validateFlow(flow *model.WorkorderInstanceFlow) error {
	if flow.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}
	if flow.StepID == "" {
		return fmt.Errorf("步骤ID不能为空")
	}
	if flow.OperatorID <= 0 {
		return fmt.Errorf("操作人ID无效")
	}
	return nil
}
