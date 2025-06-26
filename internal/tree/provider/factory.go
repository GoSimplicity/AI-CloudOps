/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	providerhuawei "github.com/GoSimplicity/AI-CloudOps/internal/tree/provider/huawei"
	"go.uber.org/zap"
)

// CloudProvider 统一的云厂商接口
type CloudProvider interface {
	// 基础方法
	SyncResources(ctx context.Context, region string) error
	ListRegions(ctx context.Context) ([]*model.RegionResp, error)
	GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error)

	// ECS实例管理
	ListInstances(ctx context.Context, region string, page, size int) ([]*model.ResourceEcs, int64, error)
	GetInstance(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error)
	CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error
	DeleteInstance(ctx context.Context, region string, instanceID string) error
	StartInstance(ctx context.Context, region string, instanceID string) error
	StopInstance(ctx context.Context, region string, instanceID string) error
	RestartInstance(ctx context.Context, region string, instanceID string) error

	// VPC网络管理
	ListVPCs(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceVpc, int64, error)
	GetVPC(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error)
	CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error
	DeleteVPC(ctx context.Context, region string, vpcID string) error

	// 安全组管理
	ListSecurityGroups(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceSecurityGroup, int64, error)
	GetSecurityGroup(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error)
	CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) error
	DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error
}

// ProviderFactory 支持动态创建多云多账户 Provider 实例
type ProviderFactory struct {
	logger    *zap.Logger
	providers map[string]CloudProvider
	mu        sync.RWMutex
}

func NewProviderFactory(logger *zap.Logger) *ProviderFactory {
	return &ProviderFactory{
		logger:    logger,
		providers: make(map[string]CloudProvider),
	}
}

// GetProvider 获取指定账户的云厂商Provider
func (f *ProviderFactory) GetProvider(account *model.CloudAccount) (CloudProvider, error) {
	if account == nil {
		return nil, fmt.Errorf("账户信息不能为空")
	}

	// 生成缓存键：provider_accountId
	cacheKey := fmt.Sprintf("%s_%d", account.Provider, account.ID)

	// 尝试从缓存获取
	f.mu.RLock()
	if provider, exists := f.providers[cacheKey]; exists {
		f.mu.RUnlock()
		return provider, nil
	}
	f.mu.RUnlock()

	// 创建新的Provider
	provider, err := f.createProvider(account)
	if err != nil {
		return nil, err
	}

	// 加入缓存
	f.mu.Lock()
	f.providers[cacheKey] = provider
	f.mu.Unlock()

	return provider, nil
}

// createProvider 根据账户信息创建对应的Provider
func (f *ProviderFactory) createProvider(account *model.CloudAccount) (CloudProvider, error) {
	switch account.Provider {
	case model.CloudProviderAliyun:
		provider := NewAliyunProvider(f.logger, account)
		if provider == nil {
			return nil, fmt.Errorf("创建阿里云Provider失败")
		}
		return provider, nil
	case model.CloudProviderHuawei:
		provider := providerhuawei.NewHuaweiProvider(f.logger, account)
		if provider == nil {
			return nil, fmt.Errorf("创建华为云Provider失败")
		}
		return nil, nil
	case model.CloudProviderLocal:
		return nil, fmt.Errorf("本地Provider暂未实现")
	default:
		return nil, fmt.Errorf("不支持的云厂商: %s", account.Provider)
	}
}

// CreateProvider 根据 CloudAccount 和解密后的 SecretKey 动态创建 Provider 实例
func (f *ProviderFactory) CreateProvider(account *model.CloudAccount, decryptedSecret string) (CloudProvider, error) {
	if account == nil {
		return nil, fmt.Errorf("CloudAccount 不能为空")
	}
	acc := *account // 拷贝，避免外部副作用
	acc.EncryptedSecret = decryptedSecret

	switch acc.Provider {
	case model.CloudProviderAliyun:
		provider := NewAliyunProvider(f.logger, &acc)
		if provider == nil {
			return nil, fmt.Errorf("创建阿里云Provider失败")
		}
		return provider, nil
	case model.CloudProviderHuawei:
		provider := providerhuawei.NewHuaweiProvider(f.logger, &acc)
		if provider == nil {
			return nil, fmt.Errorf("创建华为云Provider失败")
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("不支持的云提供商: %s", acc.Provider)
	}
}

// ClearCache 清理指定账户的Provider缓存
func (f *ProviderFactory) ClearCache(account *model.CloudAccount) {
	if account == nil {
		return
	}

	cacheKey := fmt.Sprintf("%s_%d", account.Provider, account.ID)
	f.mu.Lock()
	delete(f.providers, cacheKey)
	f.mu.Unlock()

	f.logger.Debug("清理Provider缓存", zap.String("cacheKey", cacheKey))
}

// ClearAllCache 清理所有Provider缓存
func (f *ProviderFactory) ClearAllCache() {
	f.mu.Lock()
	f.providers = make(map[string]CloudProvider)
	f.mu.Unlock()

	f.logger.Debug("清理所有Provider缓存")
}

// ValidateAccount 验证账户配置
func (f *ProviderFactory) ValidateAccount(account *model.CloudAccount) error {
	if account == nil {
		return fmt.Errorf("账户信息不能为空")
	}

	if account.AccessKey == "" {
		return fmt.Errorf("AccessKey不能为空")
	}

	if account.EncryptedSecret == "" {
		return fmt.Errorf("SecretKey不能为空")
	}

	// 验证云厂商类型是否支持
	supportedProviders := []model.CloudProvider{
		model.CloudProviderAliyun,
		model.CloudProviderHuawei,
		model.CloudProviderLocal,
	}

	isSupported := false
	for _, provider := range supportedProviders {
		if provider == account.Provider {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("不支持的云厂商: %s", account.Provider)
	}

	return nil
}

// TestConnection 测试账户连接
func (f *ProviderFactory) TestConnection(ctx context.Context, account *model.CloudAccount) error {
	if err := f.ValidateAccount(account); err != nil {
		return fmt.Errorf("账户验证失败: %w", err)
	}

	provider, err := f.GetProvider(account)
	if err != nil {
		return fmt.Errorf("获取Provider失败: %w", err)
	}

	// 尝试获取区域列表来测试连接
	_, err = provider.ListRegions(ctx)
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}

	f.logger.Info("云账户连接测试成功",
		zap.String("provider", string(account.Provider)),
		zap.Int("accountId", account.ID))

	return nil
}
