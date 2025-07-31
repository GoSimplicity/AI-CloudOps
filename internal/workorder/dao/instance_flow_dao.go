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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrFlowNilPointer = errors.New("流程记录对象为空")
)

type InstanceFlowDAO interface {
	CreateInstanceFlow(ctx context.Context, flow *model.WorkorderInstanceFlow) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.WorkorderInstanceFlow, error)
}

type instanceFlowDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceFlowDAO(db *gorm.DB, logger *zap.Logger) InstanceFlowDAO {
	return &instanceFlowDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstanceFlow 创建工单流程记录
func (d *instanceFlowDAO) CreateInstanceFlow(ctx context.Context, flow *model.WorkorderInstanceFlow) error {
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

// GetInstanceFlows 获取工单流程记录
func (d *instanceFlowDAO) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.WorkorderInstanceFlow, error) {
	if instanceID <= 0 {
		return nil, ErrInstanceInvalidID
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

// validateFlow 验证流程记录数据
func (d *instanceFlowDAO) validateFlow(flow *model.WorkorderInstanceFlow) error {
	if flow.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}
	if strings.TrimSpace(flow.StepID) == "" {
		return fmt.Errorf("步骤ID不能为空")
	}
	if flow.OperatorID <= 0 {
		return fmt.Errorf("操作人ID无效")
	}
	return nil
}
