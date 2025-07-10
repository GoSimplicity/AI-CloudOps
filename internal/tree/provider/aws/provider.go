package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aws"
)

// 确保AWSProviderImpl实现了provider.Provider接口
var _ provider.Provider = (*AWSProviderImpl)(nil)

// AWSProviderImpl AWS云资源管理的核心Provider实现，负责EC2、VPC、EBS、安全组等资源的统一管理和服务聚合。
// 该实现已在config.go中定义，这里重新整理并增强其功能
func (a *AWSProviderImpl) ensureServicesInitialized() error {
	if a.sdk == nil {
		return fmt.Errorf("AWS SDK未初始化")
	}

	// 确保所有服务都已初始化
	if a.EC2Service == nil {
		a.EC2Service = aws.NewEC2Service(a.sdk)
	}
	if a.VpcService == nil {
		a.VpcService = aws.NewVpcService(a.sdk)
	}
	if a.SecurityGroupService == nil {
		a.SecurityGroupService = aws.NewSecurityGroupService(a.sdk)
	}
	if a.EBSService == nil {
		a.EBSService = aws.NewEBSService(a.sdk)
	}

	return nil
}

// ValidateCredentials 验证AWS凭证
func (a *AWSProviderImpl) ValidateCredentials(ctx context.Context) error {
	if a.sdk == nil {
		return fmt.Errorf("AWS SDK未初始化")
	}

	// 通过获取区域列表来验证凭证
	_, err := a.getAWSRegionsFromSDK()
	if err != nil {
		a.logger.Error("AWS凭证验证失败", zap.Error(err))
		return fmt.Errorf("AWS凭证验证失败: %w", err)
	}

	a.logger.Info("AWS凭证验证成功")
	return nil
}

// GetSDK 获取AWS SDK实例（内部使用）
func (a *AWSProviderImpl) GetSDK() *aws.SDK {
	return a.sdk
}

// GetLogger 获取日志实例（内部使用）
func (a *AWSProviderImpl) GetLogger() *zap.Logger {
	return a.logger
}

// SetRegionDiscovered 设置区域发现信息（内部使用）
func (a *AWSProviderImpl) SetRegionDiscovered(regionID string, info *AWSRegionInfo) {
	if a.discoveredRegions == nil {
		a.discoveredRegions = make(map[string]*AWSRegionInfo)
	}
	a.discoveredRegions[regionID] = info
}

// GetRegionDiscovered 获取区域发现信息（内部使用）
func (a *AWSProviderImpl) GetRegionDiscovered(regionID string) (*AWSRegionInfo, bool) {
	if a.discoveredRegions == nil {
		return nil, false
	}
	info, exists := a.discoveredRegions[regionID]
	return info, exists
}

// ClearRegionCache 清除区域缓存（内部使用）
func (a *AWSProviderImpl) ClearRegionCache() {
	a.cachedRegions = nil
	a.regionsCacheTime = time.Time{}
}

// GetCachedRegions 获取缓存的区域列表（内部使用）
func (a *AWSProviderImpl) GetCachedRegions() ([]*model.RegionResp, time.Time) {
	return a.cachedRegions, a.regionsCacheTime
}

// SetCachedRegions 设置缓存的区域列表（内部使用）
func (a *AWSProviderImpl) SetCachedRegions(regions []*model.RegionResp) {
	a.cachedRegions = regions
	a.regionsCacheTime = time.Now()
}
