package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"
)

// 磁盘管理相关方法
// ListDisks, GetDisk, CreateDisk, DeleteDisk, AttachDisk, DetachDisk 及相关辅助函数

// ListDisks 获取指定region下的EBS卷列表，支持分页。
func (a *AWSProviderImpl) ListDisks(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceDisk, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, 0, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, 0, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.EBSService == nil {
		return nil, 0, fmt.Errorf("AWS EBS SDK未初始化，请先调用InitializeProvider")
	}

	req := &aws.ListVolumesRequest{
		Region: region,
		Page:   pageNumber,
		Size:   pageSize,
	}

	resp, total, err := a.EBSService.ListVolumes(ctx, req)
	if err != nil {
		a.logger.Error("failed to list EBS volumes", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list EBS volumes failed: %w", err)
	}

	if resp == nil || len(resp.Volumes) == 0 {
		return nil, 0, nil
	}

	result := make([]*model.ResourceDisk, 0, len(resp.Volumes))
	for _, volume := range resp.Volumes {
		result = append(result, a.convertToResourceDiskFromListVolume(volume, region))
	}

	return result, total, nil
}

// GetDisk 获取指定region下的EBS卷详情。
func (a *AWSProviderImpl) GetDisk(ctx context.Context, region string, diskID string) (*model.ResourceDisk, error) {
	if region == "" || diskID == "" {
		return nil, fmt.Errorf("region and diskID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.EBSService == nil {
		return nil, fmt.Errorf("AWS EBS SDK未初始化，请先调用InitializeProvider")
	}

	volume, err := a.EBSService.GetVolumeDetail(ctx, region, diskID)
	if err != nil {
		a.logger.Error("failed to get EBS volume detail", zap.Error(err), zap.String("diskID", diskID))
		return nil, fmt.Errorf("get EBS volume detail failed: %w", err)
	}

	if volume == nil {
		return nil, fmt.Errorf("EBS volume not found")
	}

	return a.convertToResourceDiskFromDetail(volume, region), nil
}

// CreateDisk 创建EBS卷。
func (a *AWSProviderImpl) CreateDisk(ctx context.Context, region string, config *model.CreateDiskReq) (*model.ResourceDisk, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.EBSService == nil {
		return nil, fmt.Errorf("AWS EBS SDK未初始化，请先调用InitializeProvider")
	}

	// 准备标签
	tags := make(map[string]string)
	if config.DiskName != "" {
		tags["Name"] = config.DiskName
	}
	if config.Description != "" {
		tags["Description"] = config.Description
	}

	req := &aws.CreateVolumeRequest{
		Region:           region,
		AvailabilityZone: config.ZoneId,
		VolumeName:       config.DiskName,
		VolumeType:       config.DiskCategory, // 映射磁盘类型
		Size:             int32(config.Size),
		Description:      config.Description,
		Encrypted:        false, // 可以根据需要配置
		Tags:             tags,
	}

	resp, err := a.EBSService.CreateVolume(ctx, req)
	if err != nil {
		a.logger.Error("failed to create EBS volume", zap.Error(err), zap.String("region", region))
		return nil, fmt.Errorf("create EBS volume failed: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("create EBS volume response is nil")
	}

	// 获取创建的卷详情
	volume, err := a.EBSService.GetVolumeDetail(ctx, region, resp.VolumeId)
	if err != nil {
		a.logger.Warn("failed to get created EBS volume detail", zap.Error(err), zap.String("volumeId", resp.VolumeId))
		// 返回基本信息
		return &model.ResourceDisk{
			InstanceName: config.DiskName,
			InstanceID:   resp.VolumeId,
			Provider:     model.CloudProviderAWS,
			RegionId:     region,
			ZoneId:       config.ZoneId,
			DiskName:     config.DiskName,
			Category:     config.DiskCategory,
			Size:         config.Size,
			Description:  config.Description,
			Status:       "creating",
			LastSyncTime: time.Now(),
		}, nil
	}

	return a.convertToResourceDiskFromDetail(volume, region), nil
}

// DeleteDisk 删除指定region下的EBS卷。
func (a *AWSProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
	if region == "" || diskID == "" {
		return fmt.Errorf("region and diskID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.EBSService == nil {
		return fmt.Errorf("AWS EBS SDK未初始化，请先调用InitializeProvider")
	}

	err := a.EBSService.DeleteVolume(ctx, region, diskID)
	if err != nil {
		a.logger.Error("failed to delete EBS volume", zap.Error(err), zap.String("diskID", diskID))
		return fmt.Errorf("delete EBS volume failed: %w", err)
	}

	return nil
}

// AttachDisk 挂载EBS卷到实例。
func (a *AWSProviderImpl) AttachDisk(ctx context.Context, region string, diskID, instanceID string) error {
	if region == "" || diskID == "" || instanceID == "" {
		return fmt.Errorf("region, diskID and instanceID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.EBSService == nil {
		return fmt.Errorf("AWS EBS SDK未初始化，请先调用InitializeProvider")
	}

	// AWS需要指定设备名称，这里使用默认的设备名称生成逻辑
	device := "/dev/sdf" // 可以根据需要动态生成

	err := a.EBSService.AttachVolume(ctx, region, diskID, instanceID, device)
	if err != nil {
		a.logger.Error("failed to attach EBS volume", zap.Error(err), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
		return fmt.Errorf("attach EBS volume failed: %w", err)
	}

	return nil
}

// DetachDisk 卸载EBS卷。
func (a *AWSProviderImpl) DetachDisk(ctx context.Context, region string, diskID, instanceID string) error {
	if region == "" || diskID == "" || instanceID == "" {
		return fmt.Errorf("region, diskID and instanceID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.EBSService == nil {
		return fmt.Errorf("AWS EBS SDK未初始化，请先调用InitializeProvider")
	}

	err := a.EBSService.DetachVolume(ctx, region, diskID, instanceID, false)
	if err != nil {
		a.logger.Error("failed to detach EBS volume", zap.Error(err), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
		return fmt.Errorf("detach EBS volume failed: %w", err)
	}

	return nil
}

// 辅助方法：数据转换

// convertToResourceDiskFromListVolume 将AWS Volume转换为ResourceDisk（列表模式）
func (a *AWSProviderImpl) convertToResourceDiskFromListVolume(volume types.Volume, region string) *model.ResourceDisk {
	var tags []string
	diskName := ""
	description := ""

	for _, tag := range volume.Tags {
		if *tag.Key == "Name" {
			diskName = *tag.Value
		} else if *tag.Key == "Description" {
			description = *tag.Value
		}
		tags = append(tags, fmt.Sprintf("%s=%s", *tag.Key, *tag.Value))
	}

	if diskName == "" {
		diskName = *volume.VolumeId
	}

	// 获取挂载信息
	var attachedInstanceId string
	var device string
	if len(volume.Attachments) > 0 {
		attachedInstanceId = *volume.Attachments[0].InstanceId
		device = *volume.Attachments[0].Device
	}

	// 获取可用区
	zoneId := ""
	if volume.AvailabilityZone != nil {
		zoneId = *volume.AvailabilityZone
	}

	// 磁盘类型映射
	diskCategory := string(volume.VolumeType)

	// 计费类型（AWS默认按需付费）
	chargeType := "PostPaid"

	return &model.ResourceDisk{
		InstanceName:       diskName,
		InstanceID:         attachedInstanceId,
		Provider:           model.CloudProviderAWS,
		RegionId:           region,
		ZoneId:             zoneId,
		DiskID:             *volume.VolumeId,
		DiskName:           diskName,
		Category:           diskCategory,
		Size:               int(*volume.Size),
		Status:             string(volume.State),
		CreationTime:       volume.CreateTime.Format(time.RFC3339),
		InstanceChargeType: chargeType,
		Description:        description,
		LastSyncTime:       time.Now(),
		Tags:               model.StringList(tags),
		Device:             device,
		Encrypted:          *volume.Encrypted,
		DeleteWithInstance: false, // 需要从attachment获取
		ResourceGroupId:    "",    // AWS使用不同的资源组织方式
		PerformanceLevel:   "",    // AWS性能参数在IOPS等字段中
	}
}

// convertToResourceDiskFromDetail 将AWS Volume转换为ResourceDisk（详情模式）
func (a *AWSProviderImpl) convertToResourceDiskFromDetail(volume *types.Volume, region string) *model.ResourceDisk {
	if volume == nil {
		return nil
	}
	return a.convertToResourceDiskFromListVolume(*volume, region)
}
