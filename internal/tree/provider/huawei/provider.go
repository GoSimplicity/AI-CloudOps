package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
)

// HuaweiProviderImpl 是华为云资源管理的核心Provider实现，负责ECS、VPC、磁盘、安全组等资源的统一管理和服务聚合。
type HuaweiProviderImpl struct {
	logger               *zap.Logger
	sdk                  *huawei.SDK
	ecsService           *huawei.EcsService
	vpcService           *huawei.VpcService
	diskService          *huawei.DiskService
	securityGroupService *huawei.SecurityGroupService
	config               *HuaweiCloudConfig
	cachedRegions        []*model.RegionResp          // 缓存的区域列表
	regionsCacheTime     time.Time                    // 区域缓存时间
	discoveredRegions    map[string]*HuaweiRegionInfo // 动态发现的区域信息
}

// NewHuaweiProvider 创建一个未初始化的华为云Provider实例（需后续调用InitializeProvider注入AK/SK）。
func NewHuaweiProvider(logger *zap.Logger) *HuaweiProviderImpl {
	return &HuaweiProviderImpl{
		logger:            logger,
		config:            getDefaultHuaweiConfig(),
		discoveredRegions: make(map[string]*HuaweiRegionInfo),
	}
}

// InitializeProvider 初始化Provider，注入AK/SK并完成SDK和各服务的初始化。
func (h *HuaweiProviderImpl) InitializeProvider(accessKey, secretKey string) error {
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("华为云访问密钥不能为空")
	}
	// 创建SDK实例
	sdk := huawei.NewSDK(h.logger, accessKey, secretKey)
	// 初始化各个服务
	h.sdk = sdk
	h.ecsService = huawei.NewEcsService(sdk)
	h.vpcService = huawei.NewVpcService(sdk)
	h.diskService = huawei.NewDiskService(sdk)
	h.securityGroupService = huawei.NewSecurityGroupService(sdk)
	h.logger.Info("华为云提供商初始化成功")
	return nil
}

// 验证华为云凭证
func (h *HuaweiProviderImpl) ValidateCredentials(ctx context.Context) error {
	if h.sdk == nil {
		return fmt.Errorf("华为云SDK未初始化")
	}
	_, err := h.getHuaweiRegionsFromSDK()
	if err != nil {
		h.logger.Error("华为云凭证验证失败", zap.Error(err))
		return fmt.Errorf("华为云凭证验证失败: %w", err)
	}
	h.logger.Info("华为云凭证验证成功")
	return nil
}
