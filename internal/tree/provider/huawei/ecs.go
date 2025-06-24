package provider

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
	"go.uber.org/zap"
)

// ECS实例管理相关方法
// ListInstances, GetInstance, CreateInstance, DeleteInstance, StartInstance, StopInstance, RestartInstance 及相关辅助函数

func (h *HuaweiProviderImpl) ListInstances(ctx context.Context, region string, page, size int) ([]*model.ResourceEcs, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if page <= 0 || size <= 0 {
		return nil, 0, fmt.Errorf("page and size must be positive integers")
	}

	if h.EcsService == nil {
		return nil, 0, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.ListInstancesRequest{
		Region: region,
		Page:   page,
		Size:   size,
	}

	resp, err := h.EcsService.ListInstances(ctx, req)
	if err != nil {
		h.logger.Error("failed to list instances", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list instances failed: %w", err)
	}

	if resp == nil || len(resp.Instances) == 0 {
		return nil, 0, nil
	}

	result := make([]*model.ResourceEcs, 0, len(resp.Instances))
	for _, instance := range resp.Instances {
		result = append(result, h.convertToResourceEcsFromListInstance(instance))
	}

	return result, int64(resp.Total), nil
}

// GetInstance 获取指定region下的ECS实例详情。
func (h *HuaweiProviderImpl) GetInstance(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	if region == "" || instanceID == "" {
		return nil, fmt.Errorf("region and instanceID cannot be empty")
	}

	if h.EcsService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	instance, err := h.EcsService.GetInstanceDetail(ctx, region, instanceID)
	if err != nil {
		h.logger.Error("failed to get instance detail", zap.Error(err), zap.String("instanceID", instanceID))
		return nil, fmt.Errorf("get instance detail failed: %w", err)
	}

	if instance == nil {
		return nil, fmt.Errorf("instance not found")
	}

	return h.convertToResourceEcsFromInstanceDetail(instance), nil
}

// CreateInstance 创建ECS实例，支持指定配置和计费类型。
func (h *HuaweiProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if h.EcsService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.CreateInstanceRequest{
		Region:             region,
		ZoneId:             config.ZoneId,
		ImageId:            config.ImageId,
		InstanceType:       config.InstanceType,
		SecurityGroupIds:   config.SecurityGroupIds,
		SubnetId:           config.VSwitchId, // 华为云使用SubnetId而不是VSwitchId
		InstanceName:       config.InstanceName,
		Hostname:           config.Hostname,
		Password:           config.Password,
		Description:        config.Description,
		Amount:             config.Amount,
		DryRun:             config.DryRun,
		InstanceChargeType: h.getInstanceChargeType(config.InstanceChargeType),
		SystemDiskCategory: config.SystemDiskCategory,
		SystemDiskSize:     config.SystemDiskSize,
		DataDiskCategory:   config.DataDiskCategory,
		DataDiskSize:       config.DataDiskSize,
	}

	_, err := h.EcsService.CreateInstance(ctx, req)
	if err != nil {
		h.logger.Error("failed to create instance", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create instance failed: %w", err)
	}

	return nil
}

// DeleteInstance 删除指定region下的ECS实例。
func (h *HuaweiProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if h.EcsService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.EcsService.DeleteInstance(ctx, region, instanceID, h.config.Defaults.ForceDelete)
	if err != nil {
		h.logger.Error("failed to delete instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("delete instance failed: %w", err)
	}

	return nil
}

// StartInstance 启动指定region下的ECS实例。
func (h *HuaweiProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if h.EcsService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.EcsService.StartInstance(ctx, region, instanceID)
	if err != nil {
		h.logger.Error("failed to start instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("start instance failed: %w", err)
	}

	return nil
}

// StopInstance 停止指定region下的ECS实例。
func (h *HuaweiProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if h.EcsService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.EcsService.StopInstance(ctx, region, instanceID, h.config.Defaults.ForceStop)
	if err != nil {
		h.logger.Error("failed to stop instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("stop instance failed: %w", err)
	}

	return nil
}

// RestartInstance 重启指定region下的ECS实例。
func (h *HuaweiProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if h.EcsService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.EcsService.RestartInstance(ctx, region, instanceID)
	if err != nil {
		h.logger.Error("failed to restart instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("restart instance failed: %w", err)
	}

	return nil
}

// getInstanceChargeType 安全获取计费类型，若传入为空则返回默认配置。
func (h *HuaweiProviderImpl) getInstanceChargeType(chargeType interface{}) string {
	if chargeType != nil {
		if ct, ok := chargeType.(string); ok && ct != "" {
			return ct
		}
	}
	return h.config.Defaults.InstanceChargeType
}
