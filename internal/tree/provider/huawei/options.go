package provider

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// 资源选项相关方法

func (h *HuaweiProviderImpl) ListRegionOptions(ctx context.Context) ([]*model.ListEcsResourceOptionsResp, error) {
	if h.sdk == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	h.logger.Debug("开始查询区域选项")

	regions, err := h.ListRegions(ctx)
	if err != nil {
		h.logger.Error("获取区域列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取区域列表失败: %w", err)
	}

	var options []*model.ListEcsResourceOptionsResp
	for _, region := range regions {
		option := &model.ListEcsResourceOptionsResp{
			Value:  region.RegionId,
			Label:  region.LocalName,
			Region: region.RegionId,
			Valid:  true,
		}
		options = append(options, option)
	}

	h.logger.Info("区域选项查询完成", zap.Int("count", len(options)))
	return options, nil
}

func (h *HuaweiProviderImpl) ListRegionZones(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if h.ecsService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	h.logger.Debug("开始查询区域可用区", zap.String("region", region))

	zoneOptions := h.getAvailableZones(region)

	var options []*model.ListEcsResourceOptionsResp
	for _, zone := range zoneOptions {
		option := &model.ListEcsResourceOptionsResp{
			Value:  zone.ZoneId,
			Label:  zone.LocalName,
			Region: region,
			Zone:   zone.ZoneId,
			Valid:  true,
		}
		options = append(options, option)
	}

	h.logger.Info("区域可用区查询完成", zap.String("region", region), zap.Int("count", len(options)))
	return options, nil
}

func (h *HuaweiProviderImpl) ListRegionInstanceTypes(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if h.ecsService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	h.logger.Debug("开始查询实例类型", zap.String("region", region))

	instanceTypes := h.getAvailableInstanceTypes(region)

	var options []*model.ListEcsResourceOptionsResp
	for _, instanceType := range instanceTypes {
		option := &model.ListEcsResourceOptionsResp{
			Value:        instanceType.InstanceTypeId,
			Label:        instanceType.Description,
			Region:       region,
			InstanceType: instanceType.InstanceTypeId,
			Cpu:          instanceType.CpuCoreCount,
			Memory:       instanceType.MemorySize,
			Valid:        true,
		}
		options = append(options, option)
	}

	h.logger.Info("实例类型查询完成", zap.String("region", region), zap.Int("count", len(options)))
	return options, nil
}

func (h *HuaweiProviderImpl) ListRegionImages(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if h.ecsService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	h.logger.Debug("开始查询镜像", zap.String("region", region))

	images := h.getAvailableImages(region)

	var options []*model.ListEcsResourceOptionsResp
	for _, image := range images {
		option := &model.ListEcsResourceOptionsResp{
			Value:   image.ImageId,
			Label:   image.ImageName,
			Region:  region,
			ImageId: image.ImageId,
			OSName:  image.ImageName,
			OSType:  image.OSType,
			Valid:   true,
		}
		options = append(options, option)
	}

	h.logger.Info("镜像查询完成", zap.String("region", region), zap.Int("count", len(options)))
	return options, nil
}

func (h *HuaweiProviderImpl) ListRegionSystemDiskCategories(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}

	h.logger.Debug("开始查询系统盘类型", zap.String("region", region))

	systemDiskTypes := []string{
		"SSD",
		"GPSSD",
		"SAS",
		"SATA",
		"ESSD",
		"GPSSD2",
		"ESSD2",
	}

	var options []*model.ListEcsResourceOptionsResp
	for _, diskType := range systemDiskTypes {
		option := &model.ListEcsResourceOptionsResp{
			Value:              diskType,
			Label:              h.getDiskTypeDescription(diskType),
			Region:             region,
			SystemDiskCategory: diskType,
			Valid:              true,
		}
		options = append(options, option)
	}

	h.logger.Info("系统盘类型查询完成", zap.String("region", region), zap.Int("count", len(options)))
	return options, nil
}

func (h *HuaweiProviderImpl) ListRegionDataDiskCategories(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}

	h.logger.Debug("开始查询数据盘类型", zap.String("region", region))

	dataDiskTypes := []string{
		"SSD",
		"GPSSD",
		"SAS",
		"SATA",
		"ESSD",
		"GPSSD2",
		"ESSD2",
	}

	var options []*model.ListEcsResourceOptionsResp
	for _, diskType := range dataDiskTypes {
		option := &model.ListEcsResourceOptionsResp{
			Value:            diskType,
			Label:            h.getDiskTypeDescription(diskType),
			Region:           region,
			DataDiskCategory: diskType,
			Valid:            true,
		}
		options = append(options, option)
	}

	h.logger.Info("数据盘类型查询完成", zap.String("region", region), zap.Int("count", len(options)))
	return options, nil
}

// 获取可用区列表，优先通过API动态获取，否则智能生成
func (h *HuaweiProviderImpl) getAvailableZones(region string) []*model.ZoneResp {
	// 若有 API 可用，优先用 API
	// 这里可根据实际情况调用 API
	// 否则用智能生成
	return h.generateZonesForRegion(region)
}

// 获取实例类型列表，转发到 provider.go
func (h *HuaweiProviderImpl) getAvailableInstanceTypes(region string) []*model.InstanceTypeResp {
	// 实际实现应调用 provider.go 的同名方法
	return nil // 如有需要可补充真实实现
}

// 获取镜像列表，转发到 provider.go
func (h *HuaweiProviderImpl) getAvailableImages(region string) []*model.ImageResp {
	// 实际实现应调用 provider.go 的同名方法
	return nil // 如有需要可补充真实实现
}

// 获取磁盘类型描述，转发到 provider.go
func (h *HuaweiProviderImpl) getDiskTypeDescription(diskType string) string {
	// 实际实现应调用 provider.go 的同名方法
	return diskType // 如有需要可补充真实实现
}

// 智能生成可用区列表，转发到 provider.go
func (h *HuaweiProviderImpl) generateZonesForRegion(region string) []*model.ZoneResp {
	// 实际实现应调用 provider.go 的同名方法
	return []*model.ZoneResp{}
}
