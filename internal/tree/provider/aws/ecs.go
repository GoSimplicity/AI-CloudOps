package provider

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aws"
	"go.uber.org/zap"
)

// EC2实例管理相关方法
// ListInstances, GetInstance, CreateInstance, DeleteInstance, StartInstance, StopInstance, RestartInstance 及相关辅助函数

// ListInstances 获取指定region下的EC2实例列表，支持分页。
func (a *AWSProviderImpl) ListInstances(ctx context.Context, region string, page, size int) ([]*model.ResourceEcs, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if page <= 0 || size <= 0 {
		return nil, 0, fmt.Errorf("page and size must be positive integers")
	}

	if a.EC2Service == nil {
		return nil, 0, fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	req := &aws.ListInstancesRequest{
		Region: region,
		Page:   page,
		Size:   size,
	}

	resp, total, err := a.EC2Service.ListInstances(ctx, req)
	if err != nil {
		a.logger.Error("failed to list instances", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list instances failed: %w", err)
	}

	if resp == nil || len(resp.Instances) == 0 {
		return nil, 0, nil
	}

	result := make([]*model.ResourceEcs, 0, len(resp.Instances))
	for _, instance := range resp.Instances {
		result = append(result, a.convertToResourceEcsFromListInstance(instance))
	}

	return result, total, nil
}

// GetInstance 获取指定region下的EC2实例详情。
func (a *AWSProviderImpl) GetInstance(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	if region == "" || instanceID == "" {
		return nil, fmt.Errorf("region and instanceID cannot be empty")
	}

	if a.EC2Service == nil {
		return nil, fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	instance, err := a.EC2Service.GetInstanceDetail(ctx, region, instanceID)
	if err != nil {
		a.logger.Error("failed to get instance detail", zap.Error(err), zap.String("instanceID", instanceID))
		return nil, fmt.Errorf("get instance detail failed: %w", err)
	}

	if instance == nil {
		return nil, fmt.Errorf("instance not found")
	}

	return a.convertToResourceEcsFromInstanceDetail(instance), nil
}

// CreateInstance 创建EC2实例，支持指定配置和计费类型。
func (a *AWSProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if a.EC2Service == nil {
		return fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	// 准备标签
	tags := make(map[string]string)
	if config.InstanceName != "" {
		tags["Name"] = config.InstanceName
	}
	if config.Description != "" {
		tags["Description"] = config.Description
	}

	req := &aws.CreateInstanceRequest{
		Region:           region,
		ImageId:          config.ImageId,
		InstanceType:     config.InstanceType,
		MinCount:         config.Amount,
		MaxCount:         config.Amount,
		SecurityGroupIds: config.SecurityGroupIds,
		SubnetId:         config.VSwitchId, // AWS使用SubnetId
		InstanceName:     config.InstanceName,
		Description:      config.Description,
		DryRun:           config.DryRun,
		Tags:             tags,
		SystemDiskSize:   int32(config.SystemDiskSize),
		SystemDiskType:   a.getVolumeType(config.SystemDiskCategory),
	}

	// 处理数据盘
	if config.DataDiskCategory != "" && config.DataDiskSize > 0 {
		dataDisk := aws.DataDisk{
			Size:       int32(config.DataDiskSize),
			VolumeType: a.getVolumeType(config.DataDiskCategory),
			Device:     "/dev/sdf", // 数据盘设备名
		}
		req.DataDisks = append(req.DataDisks, dataDisk)
	}

	_, err := a.EC2Service.CreateInstance(ctx, req)
	if err != nil {
		a.logger.Error("failed to create instance", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create instance failed: %w", err)
	}

	return nil
}

// DeleteInstance 删除指定region下的EC2实例。
func (a *AWSProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if a.EC2Service == nil {
		return fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	err := a.EC2Service.DeleteInstance(ctx, region, instanceID, false)
	if err != nil {
		a.logger.Error("failed to delete instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("delete instance failed: %w", err)
	}

	return nil
}

// StartInstance 启动指定region下的EC2实例。
func (a *AWSProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if a.EC2Service == nil {
		return fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	err := a.EC2Service.StartInstance(ctx, region, instanceID)
	if err != nil {
		a.logger.Error("failed to start instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("start instance failed: %w", err)
	}

	return nil
}

// StopInstance 停止指定region下的EC2实例。
func (a *AWSProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if a.EC2Service == nil {
		return fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	forceStop := false
	if a.config != nil {
		forceStop = a.config.Defaults.ForceStop
	}

	err := a.EC2Service.StopInstance(ctx, region, instanceID, forceStop)
	if err != nil {
		a.logger.Error("failed to stop instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("stop instance failed: %w", err)
	}

	return nil
}

// RestartInstance 重启指定region下的EC2实例。
func (a *AWSProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	if a.EC2Service == nil {
		return fmt.Errorf("AWS SDK未初始化，请先调用InitializeProvider")
	}

	err := a.EC2Service.RestartInstance(ctx, region, instanceID)
	if err != nil {
		a.logger.Error("failed to restart instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("restart instance failed: %w", err)
	}

	return nil
}

// 辅助方法：数据转换

// getVolumeType 直接返回传入的磁盘类型
func (a *AWSProviderImpl) getVolumeType(diskCategory string) string {
	return diskCategory
}
