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
	EcsService           *huawei.EcsService
	VpcService           *huawei.VpcService
	DiskService          *huawei.DiskService
	SecurityGroupService *huawei.SecurityGroupService
	config               *HuaweiCloudConfig
	cachedRegions        []*model.RegionResp          // 缓存的区域列表
	regionsCacheTime     time.Time                    // 区域缓存时间
	discoveredRegions    map[string]*HuaweiRegionInfo // 动态发现的区域信息
}

// NewHuaweiProvider 创建一个基于账号信息的华为云Provider实例
func NewHuaweiProvider(logger *zap.Logger, account *model.CloudAccount) *HuaweiProviderImpl {
	if account == nil {
		logger.Error("CloudAccount 不能为空")
		return nil
	}
	if account.AccessKey == "" || account.EncryptedSecret == "" {
		logger.Error("AccessKey 和 SecretKey 不能为空")
		return nil
	}
	// 这里假设 EncryptedSecret 已经是明文 SecretKey，实际可根据需要解密
	// 如果需要解密，可在外部先解密后传入

	sdk := huawei.NewSDK(account.AccessKey, account.EncryptedSecret)
	return &HuaweiProviderImpl{
		logger:               logger,
		sdk:                  sdk,
		EcsService:           huawei.NewEcsService(sdk),
		VpcService:           huawei.NewVpcService(sdk),
		DiskService:          huawei.NewDiskService(sdk),
		SecurityGroupService: huawei.NewSecurityGroupService(sdk),
		config:               getDefaultHuaweiConfig(),
		discoveredRegions:    make(map[string]*HuaweiRegionInfo),
	}
}

// NewHuaweiProviderImpl 创建一个基本的华为云Provider实例用于依赖注入
func NewHuaweiProviderImpl(logger *zap.Logger) *HuaweiProviderImpl {
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
	sdk := huawei.NewSDK(accessKey, secretKey)
	// 初始化各个服务
	h.sdk = sdk
	h.EcsService = huawei.NewEcsService(sdk)
	h.VpcService = huawei.NewVpcService(sdk)
	h.DiskService = huawei.NewDiskService(sdk)
	h.SecurityGroupService = huawei.NewSecurityGroupService(sdk)
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
