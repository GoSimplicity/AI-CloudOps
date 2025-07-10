package provider

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// 资源同步相关方法
// SyncResources, syncEC2Instances, syncVpcResources, syncSecurityGroupResources, syncEBSResources 及相关辅助函数

// SyncResources 并发同步指定region下的EC2、VPC、安全组和EBS资源。
func (a *AWSProviderImpl) SyncResources(ctx context.Context, region string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}

	a.logger.Info("开始同步AWS资源", zap.String("region", region))

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	errChan := make(chan error, 4)

	// 并发同步EC2实例
	go func() {
		if err := a.syncEC2Instances(ctx, region); err != nil {
			a.logger.Error("同步EC2实例失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步EC2实例失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// 并发同步VPC资源
	go func() {
		if err := a.syncVpcResources(ctx, region); err != nil {
			a.logger.Error("同步VPC资源失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步VPC资源失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// 并发同步安全组资源
	go func() {
		if err := a.syncSecurityGroupResources(ctx, region); err != nil {
			a.logger.Error("同步安全组资源失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步安全组资源失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// 并发同步EBS资源
	go func() {
		if err := a.syncEBSResources(ctx, region); err != nil {
			a.logger.Error("同步EBS资源失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步EBS资源失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// 等待所有同步任务完成
	var errors []error
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("资源同步过程中发生错误: %v", errors)
	}

	a.logger.Info("AWS资源同步完成", zap.String("region", region))
	return nil
}

// syncEC2Instances 同步EC2实例资源
func (a *AWSProviderImpl) syncEC2Instances(ctx context.Context, region string) error {
	a.logger.Debug("开始同步EC2实例", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		instances, total, err := a.ListInstances(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取EC2实例列表失败: %w", err)
		}

		if len(instances) == 0 {
			break
		}

		for _, instance := range instances {
			a.logger.Debug("同步EC2实例",
				zap.String("instanceId", instance.InstanceId),
				zap.String("instanceName", instance.InstanceName),
				zap.String("status", instance.Status))
		}

		totalSynced += len(instances)
		a.logger.Debug("EC2实例同步进度",
			zap.Int("synced", totalSynced),
			zap.Int64("total", total),
			zap.String("region", region))

		if totalSynced >= int(total) || len(instances) < pageSize {
			break
		}
		page++
	}

	a.logger.Info("EC2实例同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

// syncVpcResources 同步VPC资源
func (a *AWSProviderImpl) syncVpcResources(ctx context.Context, region string) error {
	a.logger.Debug("开始同步VPC资源", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		vpcs, total, err := a.ListVPCs(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取VPC列表失败: %w", err)
		}

		if len(vpcs) == 0 {
			break
		}

		for _, vpc := range vpcs {
			a.logger.Debug("同步VPC",
				zap.String("vpcId", vpc.VpcId),
				zap.String("vpcName", vpc.VpcName),
				zap.String("status", vpc.Status))
		}

		totalSynced += len(vpcs)
		a.logger.Debug("VPC同步进度",
			zap.Int("synced", totalSynced),
			zap.Int64("total", total),
			zap.String("region", region))

		if totalSynced >= int(total) || len(vpcs) < pageSize {
			break
		}
		page++
	}

	a.logger.Info("VPC资源同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

// syncSecurityGroupResources 同步安全组资源
func (a *AWSProviderImpl) syncSecurityGroupResources(ctx context.Context, region string) error {
	a.logger.Debug("开始同步安全组资源", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		securityGroups, total, err := a.ListSecurityGroups(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取安全组列表失败: %w", err)
		}

		if len(securityGroups) == 0 {
			break
		}

		for _, sg := range securityGroups {
			a.logger.Debug("同步安全组",
				zap.String("securityGroupId", sg.InstanceId),
				zap.String("securityGroupName", sg.SecurityGroupName),
				zap.String("status", sg.Status))
		}

		totalSynced += len(securityGroups)
		a.logger.Debug("安全组同步进度",
			zap.Int("synced", totalSynced),
			zap.Int64("total", total),
			zap.String("region", region))

		if totalSynced >= int(total) || len(securityGroups) < pageSize {
			break
		}
		page++
	}

	a.logger.Info("安全组资源同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

// syncEBSResources 同步EBS磁盘资源
func (a *AWSProviderImpl) syncEBSResources(ctx context.Context, region string) error {
	a.logger.Debug("开始同步EBS磁盘资源", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		disks, total, err := a.ListDisks(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取EBS磁盘列表失败: %w", err)
		}

		if len(disks) == 0 {
			break
		}

		for _, disk := range disks {
			a.logger.Debug("同步EBS磁盘",
				zap.String("diskId", disk.DiskID),
				zap.String("diskName", disk.DiskName),
				zap.String("status", disk.Status))
		}

		totalSynced += len(disks)
		a.logger.Debug("EBS磁盘同步进度",
			zap.Int("synced", totalSynced),
			zap.Int64("total", total),
			zap.String("region", region))

		if totalSynced >= int(total) || len(disks) < pageSize {
			break
		}
		page++
	}

	a.logger.Info("EBS磁盘资源同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

// SyncAllRegions 同步所有已配置区域的资源
func (a *AWSProviderImpl) SyncAllRegions(ctx context.Context) error {
	a.logger.Info("开始同步所有AWS区域的资源")

	regions, err := a.ListRegions(ctx)
	if err != nil {
		return fmt.Errorf("获取区域列表失败: %w", err)
	}

	var syncErrors []error
	successCount := 0

	for _, region := range regions {
		a.logger.Info("开始同步区域", zap.String("regionId", region.RegionId))

		if err := a.SyncResources(ctx, region.RegionId); err != nil {
			a.logger.Error("同步区域失败", zap.Error(err), zap.String("regionId", region.RegionId))
			syncErrors = append(syncErrors, fmt.Errorf("同步区域 %s 失败: %w", region.RegionId, err))
			continue
		}

		successCount++
		a.logger.Info("区域同步完成", zap.String("regionId", region.RegionId))
	}

	a.logger.Info("所有区域同步完成",
		zap.Int("successCount", successCount),
		zap.Int("totalRegions", len(regions)),
		zap.Int("errorCount", len(syncErrors)))

	if len(syncErrors) > 0 {
		return fmt.Errorf("部分区域同步失败: %v", syncErrors)
	}

	return nil
}

// SyncSpecificResources 同步指定类型的资源
func (a *AWSProviderImpl) SyncSpecificResources(ctx context.Context, region string, resourceTypes []string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if len(resourceTypes) == 0 {
		return fmt.Errorf("resourceTypes cannot be empty")
	}

	a.logger.Info("开始同步指定类型的AWS资源",
		zap.String("region", region),
		zap.Strings("resourceTypes", resourceTypes))

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	var errors []error

	for _, resourceType := range resourceTypes {
		switch resourceType {
		case "ec2", "instance":
			if err := a.syncEC2Instances(ctx, region); err != nil {
				errors = append(errors, fmt.Errorf("同步EC2实例失败: %w", err))
			}
		case "vpc":
			if err := a.syncVpcResources(ctx, region); err != nil {
				errors = append(errors, fmt.Errorf("同步VPC资源失败: %w", err))
			}
		case "sg", "security-group":
			if err := a.syncSecurityGroupResources(ctx, region); err != nil {
				errors = append(errors, fmt.Errorf("同步安全组资源失败: %w", err))
			}
		case "ebs", "disk":
			if err := a.syncEBSResources(ctx, region); err != nil {
				errors = append(errors, fmt.Errorf("同步EBS资源失败: %w", err))
			}
		default:
			a.logger.Warn("不支持的资源类型", zap.String("resourceType", resourceType))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("同步过程中发生错误: %v", errors)
	}

	a.logger.Info("指定类型资源同步完成",
		zap.String("region", region),
		zap.Strings("resourceTypes", resourceTypes))
	return nil
}

// ValidateResourceConsistency 验证资源一致性
func (a *AWSProviderImpl) ValidateResourceConsistency(ctx context.Context, region string) error {
	a.logger.Info("开始验证AWS资源一致性", zap.String("region", region))

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	// 这里可以添加资源一致性检查逻辑
	// 例如：检查实例与VPC的关联关系、安全组引用、磁盘挂载状态等

	a.logger.Info("AWS资源一致性验证完成", zap.String("region", region))
	return nil
}
