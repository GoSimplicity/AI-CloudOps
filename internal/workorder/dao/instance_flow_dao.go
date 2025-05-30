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
	// 流程方法
	CreateInstanceFlow(ctx context.Context, flow *model.InstanceFlow) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlow, error)
	BatchCreateInstanceFlows(ctx context.Context, flows []model.InstanceFlow) error
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
func (d *instanceFlowDAO) CreateInstanceFlow(ctx context.Context, flow *model.InstanceFlow) error {
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
func (d *instanceFlowDAO) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlow, error) {
	if instanceID <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var flows []model.InstanceFlow
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

// BatchCreateInstanceFlows 批量创建工单流程记录
func (d *instanceFlowDAO) BatchCreateInstanceFlows(ctx context.Context, flows []model.InstanceFlow) error {
	if len(flows) == 0 {
		return nil
	}

	// 验证流程记录
	for i, flow := range flows {
		if err := d.validateFlow(&flow); err != nil {
			return fmt.Errorf("第%d个流程记录验证失败: %w", i+1, err)
		}
	}

	if err := d.db.WithContext(ctx).CreateInBatches(flows, DefaultBatchSize).Error; err != nil {
		d.logger.Error("批量创建工单流程记录失败", zap.Error(err), zap.Int("count", len(flows)))
		return fmt.Errorf("批量创建工单流程记录失败: %w", err)
	}

	d.logger.Info("批量创建工单流程记录成功", zap.Int("count", len(flows)))
	return nil
}

// validateFlow 验证流程记录数据
func (d *instanceFlowDAO) validateFlow(flow *model.InstanceFlow) error {
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
