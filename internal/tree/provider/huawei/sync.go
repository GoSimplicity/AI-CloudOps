package provider

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// 资源同步相关方法
// SyncResources, syncEcsInstances, syncVpcResources, syncSecurityGroupResources, syncDiskResources 及相关辅助函数

func (h *HuaweiProviderImpl) SyncResources(ctx context.Context, region string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}

	h.logger.Info("开始同步华为云资源", zap.String("region", region))

	if h.sdk == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	errChan := make(chan error, 4)

	go func() {
		if err := h.syncEcsInstances(ctx, region); err != nil {
			h.logger.Error("同步ECS实例失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步ECS实例失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	go func() {
		if err := h.syncVpcResources(ctx, region); err != nil {
			h.logger.Error("同步VPC资源失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步VPC资源失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	go func() {
		if err := h.syncSecurityGroupResources(ctx, region); err != nil {
			h.logger.Error("同步安全组资源失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步安全组资源失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	go func() {
		if err := h.syncDiskResources(ctx, region); err != nil {
			h.logger.Error("同步磁盘资源失败", zap.Error(err), zap.String("region", region))
			errChan <- fmt.Errorf("同步磁盘资源失败: %w", err)
		} else {
			errChan <- nil
		}
	}()

	var errors []error
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("资源同步过程中发生错误: %v", errors)
	}

	h.logger.Info("华为云资源同步完成", zap.String("region", region))
	return nil
}

func (h *HuaweiProviderImpl) syncEcsInstances(ctx context.Context, region string) error {
	h.logger.Debug("开始同步ECS实例", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		instances, total, err := h.ListInstances(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取ECS实例列表失败: %w", err)
		}

		if len(instances) == 0 {
			break
		}

		for _, instance := range instances {
			h.logger.Debug("同步ECS实例",
				zap.String("instanceId", instance.InstanceId),
				zap.String("instanceName", instance.InstanceName),
				zap.String("status", instance.Status))
		}

		totalSynced += len(instances)
		h.logger.Debug("ECS实例同步进度",
			zap.Int("synced", totalSynced),
			zap.Int64("total", total),
			zap.String("region", region))

		if totalSynced >= int(total) || len(instances) < pageSize {
			break
		}
		page++
	}

	h.logger.Info("ECS实例同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

func (h *HuaweiProviderImpl) syncVpcResources(ctx context.Context, region string) error {
	h.logger.Debug("开始同步VPC资源", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		vpcs, err := h.ListVPCs(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取VPC列表失败: %w", err)
		}

		if len(vpcs) == 0 {
			break
		}

		for _, vpc := range vpcs {
			h.logger.Debug("同步VPC",
				zap.String("vpcId", vpc.VpcId),
				zap.String("vpcName", vpc.VpcName),
				zap.String("status", vpc.Status))
		}

		totalSynced += len(vpcs)
		h.logger.Debug("VPC同步进度",
			zap.Int("synced", totalSynced),
			zap.String("region", region))

		if len(vpcs) < pageSize {
			break
		}
		page++
	}

	h.logger.Info("VPC资源同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

func (h *HuaweiProviderImpl) syncSecurityGroupResources(ctx context.Context, region string) error {
	h.logger.Debug("开始同步安全组资源", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		securityGroups, err := h.ListSecurityGroups(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取安全组列表失败: %w", err)
		}

		if len(securityGroups) == 0 {
			break
		}

		for _, sg := range securityGroups {
			h.logger.Debug("同步安全组",
				zap.String("securityGroupId", sg.InstanceId),
				zap.String("securityGroupName", sg.SecurityGroupName),
				zap.String("status", sg.Status))
		}

		totalSynced += len(securityGroups)
		h.logger.Debug("安全组同步进度",
			zap.Int("synced", totalSynced),
			zap.String("region", region))

		if len(securityGroups) < pageSize {
			break
		}
		page++
	}

	h.logger.Info("安全组资源同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}

func (h *HuaweiProviderImpl) syncDiskResources(ctx context.Context, region string) error {
	h.logger.Debug("开始同步磁盘资源", zap.String("region", region))

	page := 1
	pageSize := 50
	totalSynced := 0

	for {
		disks, err := h.ListDisks(ctx, region, page, pageSize)
		if err != nil {
			return fmt.Errorf("获取磁盘列表失败: %w", err)
		}

		if len(disks) == 0 {
			break
		}

		for _, disk := range disks {
			h.logger.Debug("同步磁盘",
				zap.String("diskId", disk.DiskID),
				zap.String("diskName", disk.DiskName),
				zap.Int("size", disk.Size),
				zap.String("status", disk.Status))
		}

		totalSynced += len(disks)
		h.logger.Debug("磁盘同步进度",
			zap.Int("synced", totalSynced),
			zap.String("region", region))

		if len(disks) < pageSize {
			break
		}
		page++
	}

	h.logger.Info("磁盘资源同步完成", zap.Int("totalSynced", totalSynced), zap.String("region", region))
	return nil
}
