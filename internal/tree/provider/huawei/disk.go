package provider

// 磁盘管理相关方法
// ListDisks, GetDisk, CreateDisk, DeleteDisk, AttachDisk, DetachDisk 及相关辅助函数

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
	"go.uber.org/zap"
)

// ListDisks 获取指定region下的云硬盘列表，支持分页。
func (h *HuaweiProviderImpl) ListDisks(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceDisk, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	// 检查SDK服务是否已初始化
	if h.DiskService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.ListDisksRequest{
		Region: region,
		Page:   pageNumber,
		Size:   pageSize,
	}

	resp, err := h.DiskService.ListDisks(ctx, req)
	if err != nil {
		h.logger.Error("failed to list disks", zap.Error(err), zap.String("region", region))
		return nil, fmt.Errorf("list disks failed: %w", err)
	}

	if resp == nil || len(resp.Disks) == 0 {
		return nil, nil
	}

	result := make([]*model.ResourceDisk, 0, len(resp.Disks))
	for _, disk := range resp.Disks {
		result = append(result, h.convertToResourceDiskFromList(disk, region))
	}

	return result, nil
}

// GetDisk
func (h *HuaweiProviderImpl) GetDisk(ctx context.Context, region string, diskID string) (*model.ResourceDisk, error) {
	if region == "" || diskID == "" {
		return nil, fmt.Errorf("region and diskID cannot be empty")
	}

	// 检查SDK服务是否已初始化
	if h.DiskService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	disk, err := h.DiskService.GetDisk(ctx, region, diskID)
	if err != nil {
		h.logger.Error("failed to get disk detail", zap.Error(err), zap.String("diskID", diskID))
		return nil, fmt.Errorf("get disk detail failed: %w", err)
	}

	if disk == nil {
		return nil, fmt.Errorf("disk not found")
	}

	return h.convertToResourceDiskFromDetail(disk, region), nil
}

// CreateDisk
func (h *HuaweiProviderImpl) CreateDisk(ctx context.Context, region string, config *model.CreateDiskReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 检查SDK服务是否已初始化
	if h.DiskService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.CreateDiskRequest{
		Region:       region,
		ZoneId:       config.ZoneId,
		DiskName:     config.DiskName,
		DiskCategory: config.DiskCategory,
		Size:         config.Size,
		Description:  config.Description,
	}

	_, err := h.DiskService.CreateDisk(ctx, req)
	if err != nil {
		h.logger.Error("failed to create disk", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create disk failed: %w", err)
	}

	return nil
}

// DeleteDisk
func (h *HuaweiProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
	if region == "" || diskID == "" {
		return fmt.Errorf("region and diskID cannot be empty")
	}

	// 检查SDK服务是否已初始化
	if h.DiskService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.DiskService.DeleteDisk(ctx, region, diskID)
	if err != nil {
		h.logger.Error("failed to delete disk", zap.Error(err), zap.String("diskID", diskID))
		return fmt.Errorf("delete disk failed: %w", err)
	}

	return nil
}

// AttachDisk
func (h *HuaweiProviderImpl) AttachDisk(ctx context.Context, region string, diskID, instanceID string) error {
	if region == "" || diskID == "" || instanceID == "" {
		return fmt.Errorf("region, diskID and instanceID cannot be empty")
	}

	// 检查SDK服务是否已初始化
	if h.DiskService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.DiskService.AttachDisk(ctx, region, diskID, instanceID)
	if err != nil {
		h.logger.Error("failed to attach disk", zap.Error(err), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
		return fmt.Errorf("attach disk failed: %w", err)
	}

	return nil
}

// DetachDisk
func (h *HuaweiProviderImpl) DetachDisk(ctx context.Context, region string, diskID, instanceID string) error {
	if region == "" || diskID == "" || instanceID == "" {
		return fmt.Errorf("region, diskID and instanceID cannot be empty")
	}

	// 检查SDK服务是否已初始化
	if h.DiskService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.DiskService.DetachDisk(ctx, region, diskID, instanceID)
	if err != nil {
		h.logger.Error("failed to detach disk", zap.Error(err), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
		return fmt.Errorf("detach disk failed: %w", err)
	}

	return nil
}
