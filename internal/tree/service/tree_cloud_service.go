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

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	provider "github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	providerhuawei "github.com/GoSimplicity/AI-CloudOps/internal/tree/provider/huawei"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aliyun"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"go.uber.org/zap"
)

type TreeCloudService interface {
	CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error
	UpdateCloudAccount(ctx context.Context, req *model.UpdateCloudAccountReq) error
	DeleteCloudAccount(ctx context.Context, id int) error
	GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error)
	ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error)
	TestCloudAccount(ctx context.Context, id int) error
	SyncCloudResources(ctx context.Context, req *model.SyncCloudReq) error
	SyncCloudAccountResources(ctx context.Context, req *model.SyncCloudAccountResourcesReq) error
	GetCloudAccountStatistics(ctx context.Context, req *model.GetCloudAccountStatisticsReq) (*model.CloudAccountStatistics, error)
}

type treeCloudService struct {
	logger           *zap.Logger
	dao              dao.TreeCloudDAO
	ecsDao           dao.TreeEcsDAO
	vpcDao           dao.TreeVpcDAO
	securityGroupDao dao.TreeSecurityGroupDAO
	providerFactory  *provider.ProviderFactory
}

func NewTreeCloudService(logger *zap.Logger, dao dao.TreeCloudDAO, providerFactory *provider.ProviderFactory, vpcDao dao.TreeVpcDAO, securityGroupDao dao.TreeSecurityGroupDAO, ecsDao dao.TreeEcsDAO) TreeCloudService {
	return &treeCloudService{
		logger:           logger,
		dao:              dao,
		providerFactory:  providerFactory,
		vpcDao:           vpcDao,
		securityGroupDao: securityGroupDao,
		ecsDao:           ecsDao,
	}
}

// CreateCloudAccount 创建云账户
func (t *treeCloudService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error {
	account := &model.CloudAccount{
		Name:            req.Name,
		Provider:        req.Provider,
		AccountId:       req.AccountId,
		AccessKey:       req.AccessKey,
		EncryptedSecret: req.SecretKey,
		Regions:         req.Regions,
		IsEnabled:       req.IsEnabled,
		Description:     req.Description,
	}

	if err := t.dao.CreateCloudAccount(ctx, account); err != nil {
		t.logger.Error("DAO创建云账户失败", zap.String("name", req.Name), zap.Error(err))
		return fmt.Errorf("创建云账户失败: %w", err)
	}

	return nil
}

// UpdateCloudAccount 更新云账户
func (t *treeCloudService) UpdateCloudAccount(ctx context.Context, req *model.UpdateCloudAccountReq) error {
	// 获取现有账户信息
	existingAccount, err := t.dao.GetCloudAccount(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取现有云账户失败", zap.Int("id", req.ID), zap.Error(err))
		return fmt.Errorf("获取云账户失败: %w", err)
	}

	updateData := &model.CloudAccount{
		Model:       existingAccount.Model,
		Name:        req.Name,
		Provider:    req.Provider,
		AccountId:   req.AccountId,
		AccessKey:   req.AccessKey,
		Regions:     req.Regions,
		IsEnabled:   req.IsEnabled,
		Description: req.Description,
	}

	// 如果提供了新的SecretKey，进行加密
	if req.SecretKey != "" {
		updateData.EncryptedSecret = req.SecretKey
	}

	// 调用DAO更新账户
	if err := t.dao.UpdateCloudAccount(ctx, updateData); err != nil {
		t.logger.Error("DAO更新云账户失败", zap.Int("id", updateData.ID), zap.Error(err))
		return fmt.Errorf("更新云账户失败: %w", err)
	}

	return nil
}

// GetCloudAccount 获取云账户详情（包含解密后的SecretKey）
func (t *treeCloudService) GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error) {
	// 调用DAO获取账户信息
	account, err := t.dao.GetCloudAccount(ctx, id)
	if err != nil {
		t.logger.Error("DAO获取云账户失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("获取云账户失败: %w", err)
	}

	// 创建返回对象，包含解密后的SecretKey
	result := &model.CloudAccount{
		Model:           account.Model,
		Name:            account.Name,
		Provider:        account.Provider,
		AccountId:       account.AccountId,
		AccessKey:       account.AccessKey,
		EncryptedSecret: account.EncryptedSecret,
		Regions:         account.Regions,
		IsEnabled:       account.IsEnabled,
		LastSyncTime:    account.LastSyncTime,
		Description:     account.Description,
	}

	return result, nil
}

// ListCloudAccounts 获取云账户列表（不包含SecretKey）
func (t *treeCloudService) ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error) {
	// 调用DAO获取列表
	result, err := t.dao.ListCloudAccounts(ctx, req)
	if err != nil {
		t.logger.Error("DAO获取云账户列表失败", zap.Error(err))
		return model.ListResp[model.CloudAccount]{}, fmt.Errorf("获取云账户列表失败: %w", err)
	}

	// 清空所有账户的EncryptedSecret字段，确保安全
	for i := range result.Items {
		result.Items[i].EncryptedSecret = ""
	}

	return result, nil
}

// DeleteCloudAccount 删除云账户
func (t *treeCloudService) DeleteCloudAccount(ctx context.Context, id int) error {
	// 调用DAO删除账户
	if err := t.dao.DeleteCloudAccount(ctx, id); err != nil {
		t.logger.Error("DAO删除云账户失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("删除云账户失败: %w", err)
	}

	return nil
}

// TestCloudAccount 测试云账户连接
func (t *treeCloudService) TestCloudAccount(ctx context.Context, id int) error {
	// 获取账户信息
	account, err := t.dao.GetCloudAccount(ctx, id)
	if err != nil {
		t.logger.Error("获取云账户信息失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取云账户信息失败: %w", err)
	}

	switch account.Provider {
	case model.CloudProviderHuawei:
		sdk := huawei.NewSDK(account.AccessKey, account.EncryptedSecret)
		ecsService := huawei.NewEcsService(sdk)
		_, _, err := ecsService.ListInstances(ctx, &huawei.ListInstancesRequest{
			Region: account.Regions[0],
		})
		if err != nil {
			t.logger.Error("华为云连接测试失败", zap.Int("id", id), zap.String("region", account.Regions[0]), zap.Error(err))
			return fmt.Errorf("华为云连接测试失败: %w", err)
		}
		return nil
	case model.CloudProviderAliyun:
		aliyunSDK := aliyun.NewSDK(account.AccessKey, account.EncryptedSecret)
		ecsClient, err := aliyunSDK.CreateEcsClient(account.Regions[0])
		if err != nil {
			t.logger.Error("阿里云ECS客户端创建失败", zap.Int("id", id), zap.String("region", account.Regions[0]), zap.Error(err))
			return fmt.Errorf("阿里云ECS客户端创建失败: %w", err)
		}
		// 调用DescribeRegions做连接测试
		request := &ecs.DescribeRegionsRequest{}
		_, err = ecsClient.DescribeRegions(request)
		if err != nil {
			t.logger.Error("阿里云连接测试失败", zap.Int("id", id), zap.String("region", account.Regions[0]), zap.Error(err))
			return fmt.Errorf("阿里云连接测试失败: %w", err)
		}
		t.logger.Info("阿里云连接测试成功", zap.Int("id", id), zap.String("region", account.Regions[0]))
		return nil
	default:
		return fmt.Errorf("暂不支持该云厂商的连接测试: %s", account.Provider)
	}
}

// SyncCloudResources 同步云资源
func (t *treeCloudService) SyncCloudResources(ctx context.Context, req *model.SyncCloudReq) error {
	// 获取账号列表
	var accounts []*model.CloudAccount
	var err error
	if len(req.AccountIds) == 0 {
		accounts, err = t.dao.GetEnabledCloudAccounts(ctx)
		if err != nil {
			t.logger.Error("获取启用账号失败", zap.Error(err))
			return err
		}
	} else {
		accounts, err = t.dao.BatchGetCloudAccounts(ctx, req.AccountIds)
		if err != nil {
			t.logger.Error("批量获取账号失败", zap.Error(err))
			return err
		}
	}

	for _, account := range accounts {
		// 构造Provider
		var (
			aliyunProvider *provider.AliyunProviderImpl
			huaweiProvider *providerhuawei.HuaweiProviderImpl
		)
		switch account.Provider {
		case model.CloudProviderAliyun:
			aliyunProvider = provider.NewAliyunProvider(t.logger)
			if aliyunProvider == nil {
				t.logger.Error("阿里云Provider初始化失败", zap.Int("accountId", account.ID))
				continue
			}
		case model.CloudProviderHuawei:
			huaweiProvider = providerhuawei.NewHuaweiProvider(t.logger, account)
			if huaweiProvider == nil {
				t.logger.Error("华为云Provider初始化失败", zap.Int("accountId", account.ID))
				continue
			}
		default:
			t.logger.Warn("暂不支持的云厂商", zap.String("provider", string(account.Provider)))
			continue
		}

		// 区域遍历
		regions := req.Regions
		if len(regions) == 0 {
			regions = account.Regions
		}

		if len(regions) == 0 {
			t.logger.Warn("账号无可用区域", zap.Int("accountId", account.ID))
			continue
		}

		for _, region := range regions {
			// 创建同步状态记录
			syncStatus := &model.CloudAccountSyncStatus{
				AccountId:    account.ID,
				ResourceType: req.ResourceType,
				Region:       region,
				Status:       model.SyncStatusRunning,
				LastSyncTime: time.Now(),
			}
			t.dao.CreateSyncStatus(ctx, syncStatus)

			// 按资源类型同步
			if req.ResourceType == "" || req.ResourceType == model.ResourceTypeECS {
				if err := t.syncECSResources(ctx, account, aliyunProvider, huaweiProvider, region); err != nil {
					t.updateSyncStatus(ctx, account.ID, model.ResourceTypeECS, region, model.SyncStatusFailed, err.Error(), 0)
					continue
				}
			}
			if req.ResourceType == "" || req.ResourceType == model.ResourceTypeVPC {
				if err := t.syncVPCResources(ctx, account, aliyunProvider, huaweiProvider, region); err != nil {
					t.updateSyncStatus(ctx, account.ID, model.ResourceTypeVPC, region, model.SyncStatusFailed, err.Error(), 0)
					continue
				}
			}
			if req.ResourceType == "" || req.ResourceType == model.ResourceTypeSecurityGroup {
				if err := t.syncSecurityGroupResources(ctx, account, aliyunProvider, huaweiProvider, region); err != nil {
					t.updateSyncStatus(ctx, account.ID, model.ResourceTypeSecurityGroup, region, model.SyncStatusFailed, err.Error(), 0)
					continue
				}
			}

			// 更新同步状态为成功
			t.updateSyncStatus(ctx, account.ID, req.ResourceType, region, model.SyncStatusSuccess, "", 0)
		}
	}
	return nil
}

// getUserInfoFromContext 从上下文中获取用户信息
func (t *treeCloudService) getUserInfoFromContext(ctx context.Context) *utils.UserInfo {
	// 尝试从context中获取用户信息
	userInfo := utils.GetUserInfoFromContext(ctx)

	// 如果获取不到用户信息，使用默认值
	if userInfo.Username == "" {
		userInfo.Username = "system"
	}
	if userInfo.IP == "" {
		userInfo.IP = "unknown"
	}
	if userInfo.UserAgent == "" {
		userInfo.UserAgent = "system"
	}

	return userInfo
}

// SyncCloudAccountResources 同步指定云账号的资源
func (t *treeCloudService) SyncCloudAccountResources(ctx context.Context, req *model.SyncCloudAccountResourcesReq) error {
	// 获取账号
	account, err := t.dao.GetCloudAccount(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取云账号失败", zap.Int("ID", req.ID), zap.Error(err))
		return err
	}

	var (
		aliyunProvider *provider.AliyunProviderImpl
		huaweiProvider *providerhuawei.HuaweiProviderImpl
	)
	switch account.Provider {
	case model.CloudProviderAliyun:
		aliyunProvider = provider.NewAliyunProvider(t.logger)
		if aliyunProvider == nil {
			t.logger.Error("阿里云Provider初始化失败", zap.Int("accountId", account.ID))
			return fmt.Errorf("阿里云Provider初始化失败")
		}
	case model.CloudProviderHuawei:
		huaweiProvider = providerhuawei.NewHuaweiProvider(t.logger, account)
		if huaweiProvider == nil {
			t.logger.Error("华为云Provider初始化失败", zap.Int("accountId", account.ID))
			return fmt.Errorf("华为云Provider初始化失败")
		}
	default:
		t.logger.Warn("暂不支持的云厂商", zap.String("provider", string(account.Provider)))
		return nil
	}

	regions := req.Regions
	if len(regions) == 0 {
		regions = account.Regions
	}
	if len(regions) == 0 {
		t.logger.Warn("账号无可用区域", zap.Int("accountId", account.ID))
		return nil
	}

	// 资源同步
	for _, region := range regions {
		// ECS同步
		if req.ResourceType == "" || req.ResourceType == "ecs" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				ecss, total, err := aliyunProvider.ListInstances(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云ECS失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
				} else {
					if err := t.ecsDao.SyncEcsResources(ctx, ecss, total); err != nil {
						t.logger.Error("同步阿里云ECS到数据库失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云ECS成功", zap.Int("accountId", account.ID), zap.String("region", region))
					}
				}
			case model.CloudProviderHuawei:
				resp, total, err := huaweiProvider.EcsService.ListInstances(ctx, &huawei.ListInstancesRequest{
					Region: region,
					Page:   1,
					Size:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云ECS失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
				} else {
					var ecsInstances []*model.ResourceEcs
					if resp != nil && resp.Instances != nil {
						for _, instance := range resp.Instances {
							ecsResource := t.convertHuaweiECSToResourceEcs(instance, account, region)
							ecsInstances = append(ecsInstances, ecsResource)
						}
					}
					if err := t.ecsDao.SyncEcsResources(ctx, ecsInstances, total); err != nil {
						t.logger.Error("同步华为云ECS到数据库失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云ECS成功", zap.Int("accountId", account.ID), zap.String("region", region))
					}
				}
			}
		}
		// VPC同步
		if req.ResourceType == "" || req.ResourceType == "vpc" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				vpcs, total, err := aliyunProvider.ListVPCs(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云VPC失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
				} else {
					if err := t.vpcDao.SyncVPCResources(ctx, vpcs, total); err != nil {
						t.logger.Error("同步阿里云VPC到数据库失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云VPC成功", zap.Int("accountId", account.ID), zap.String("region", region))
					}
				}
			case model.CloudProviderHuawei:
				resp, total, err := huaweiProvider.VpcService.ListVpcs(ctx, &huawei.ListVpcsRequest{
					Region: region,
					Page:   1,
					Size:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云VPC失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
				} else {
					var vpcResources []*model.ResourceVpc
					if resp != nil && resp.Vpcs != nil {
						for _, vpc := range resp.Vpcs {
							vpcResource := t.convertHuaweiVPCToResourceVpc(vpc, account, region)
							vpcResources = append(vpcResources, vpcResource)
						}
					}
					if err := t.vpcDao.SyncVPCResources(ctx, vpcResources, total); err != nil {
						t.logger.Error("同步华为云VPC到数据库失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云VPC成功", zap.Int("accountId", account.ID), zap.String("region", region))
					}
				}
			}
		}
		// SG同步
		if req.ResourceType == "" || req.ResourceType == "sg" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				sgs, total, err := aliyunProvider.ListSecurityGroups(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云SG失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
				} else {
					if err := t.securityGroupDao.SyncSecurityGroupResources(ctx, sgs, total); err != nil {
						t.logger.Error("同步阿里云SG到数据库失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云SG成功", zap.Int("accountId", account.ID), zap.String("region", region))
					}
				}
			case model.CloudProviderHuawei:
				resp, total, err := huaweiProvider.SecurityGroupService.ListSecurityGroups(ctx, &huawei.ListSecurityGroupsRequest{
					Region:     region,
					PageNumber: 1,
					PageSize:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云SG失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
				} else {
					var sgResources []*model.ResourceSecurityGroup
					if resp != nil && resp.SecurityGroups != nil {
						for _, sg := range resp.SecurityGroups {
							sgResource := t.convertHuaweiSecurityGroupToResourceSecurityGroup(sg, account, region)
							sgResources = append(sgResources, sgResource)
						}
					}
					if err := t.securityGroupDao.SyncSecurityGroupResources(ctx, sgResources, total); err != nil {
						t.logger.Error("同步华为云SG到数据库失败", zap.Int("accountId", account.ID), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云SG成功", zap.Int("accountId", account.ID), zap.String("region", region))
					}
				}
			}
		}
	}
	t.logger.Info("云账号资源同步完成", zap.Int("accountId", req.ID))
	return nil
}

// GetCloudAccountStatistics 获取云账号统计信息
func (t *treeCloudService) GetCloudAccountStatistics(ctx context.Context, req *model.GetCloudAccountStatisticsReq) (*model.CloudAccountStatistics, error) {
	// 获取所有云账号
	allAccountsReq := &model.ListCloudAccountsReq{}
	allAccounts, err := t.dao.ListCloudAccounts(ctx, allAccountsReq)
	if err != nil {
		t.logger.Error("获取所有云账号失败", zap.Error(err))
		return nil, fmt.Errorf("获取所有云账号失败: %w", err)
	}

	// 获取启用的云账号
	enabledAccountsReq := &model.ListCloudAccountsReq{Enabled: true}
	enabledAccounts, err := t.dao.ListCloudAccounts(ctx, enabledAccountsReq)
	if err != nil {
		t.logger.Error("获取启用云账号失败", zap.Error(err))
		return nil, fmt.Errorf("获取启用云账号失败: %w", err)
	}

	stats := &model.CloudAccountStatistics{
		TotalAccounts:    allAccounts.Total,
		EnabledAccounts:  enabledAccounts.Total,
		DisabledAccounts: allAccounts.Total - enabledAccounts.Total,
		ProviderStats:    make(map[string]int64),
		RegionStats:      make(map[string]int64),
		SyncStatus:       make(map[string]int64),
	}

	// 统计各云厂商账号数
	for _, account := range allAccounts.Items {
		provider := string(account.Provider)
		stats.ProviderStats[provider]++

		// 统计各区域账号数
		for _, region := range account.Regions {
			stats.RegionStats[region]++
		}
	}

	// 获取同步状态统计
	for _, account := range allAccounts.Items {
		statuses, err := t.dao.ListSyncStatus(ctx, account.ID)
		if err != nil {
			t.logger.Warn("获取同步状态失败", zap.Int("accountId", account.ID), zap.Error(err))
			continue
		}
		for _, status := range statuses {
			stats.SyncStatus[status.Status]++
		}
	}

	return stats, nil
}

// syncECSResources 同步ECS资源
func (t *treeCloudService) syncECSResources(ctx context.Context, account *model.CloudAccount, aliyunProvider *provider.AliyunProviderImpl, huaweiProvider *providerhuawei.HuaweiProviderImpl, region string) error {
	switch account.Provider {
	case model.CloudProviderAliyun:
		if aliyunProvider == nil {
			return fmt.Errorf("阿里云Provider为空")
		}

		// 循环获取所有ECS实例（分页）
		page := 1
		pageSize := 100
		var allInstances []*model.ResourceEcs

		for {
			instances, _, err := aliyunProvider.ListInstances(ctx, region, page, pageSize)
			if err != nil {
				return fmt.Errorf("获取阿里云ECS实例失败: %w", err)
			}

			if len(instances) == 0 {
				break
			}

			allInstances = append(allInstances, instances...)

			// 如果获取的实例数少于页大小，说明已经获取完所有数据
			if len(instances) < pageSize {
				break
			}

			page++
		}

		// 同步到数据库
		if err := t.ecsDao.SyncEcsResources(ctx, allInstances, int64(len(allInstances))); err != nil {
			return fmt.Errorf("同步阿里云ECS到数据库失败: %w", err)
		}

		t.logger.Info("同步阿里云ECS成功", zap.Int("accountId", account.ID), zap.String("region", region), zap.Int("count", len(allInstances)))

	case model.CloudProviderHuawei:
		if huaweiProvider == nil {
			return fmt.Errorf("华为云Provider为空")
		}

		// 循环获取所有ECS实例（分页）
		page := 1
		pageSize := 100
		var allInstances []*model.ResourceEcs

		for {
			resp, _, err := huaweiProvider.EcsService.ListInstances(ctx, &huawei.ListInstancesRequest{
				Region: region,
				Page:   page,
				Size:   pageSize,
			})
			if err != nil {
				return fmt.Errorf("获取华为云ECS实例失败: %w", err)
			}

			if resp == nil || len(resp.Instances) == 0 {
				break
			}

			// 转换华为云实例为统一模型
			for _, instance := range resp.Instances {
				ecsResource := t.convertHuaweiECSToResourceEcs(instance, account, region)
				allInstances = append(allInstances, ecsResource)
			}

			if len(resp.Instances) < pageSize {
				break
			}

			page++
		}

		// 同步到数据库
		if err := t.ecsDao.SyncEcsResources(ctx, allInstances, int64(len(allInstances))); err != nil {
			return fmt.Errorf("同步华为云ECS到数据库失败: %w", err)
		}

		t.logger.Info("同步华为云ECS成功", zap.Int("accountId", account.ID), zap.String("region", region), zap.Int("count", len(allInstances)))
	}

	return nil
}

// syncVPCResources 同步VPC资源
func (t *treeCloudService) syncVPCResources(ctx context.Context, account *model.CloudAccount, aliyunProvider *provider.AliyunProviderImpl, huaweiProvider *providerhuawei.HuaweiProviderImpl, region string) error {
	switch account.Provider {
	case model.CloudProviderAliyun:
		if aliyunProvider == nil {
			return fmt.Errorf("阿里云Provider为空")
		}

		// 循环获取所有VPC
		page := 1
		pageSize := 100
		var allVpcs []*model.ResourceVpc

		for {
			vpcs, _, err := aliyunProvider.ListVPCs(ctx, region, page, pageSize)
			if err != nil {
				return fmt.Errorf("获取阿里云VPC失败: %w", err)
			}

			if len(vpcs) == 0 {
				break
			}

			allVpcs = append(allVpcs, vpcs...)

			if len(vpcs) < pageSize {
				break
			}

			page++
		}

		// 同步到数据库
		if err := t.vpcDao.SyncVPCResources(ctx, allVpcs, int64(len(allVpcs))); err != nil {
			return fmt.Errorf("同步阿里云VPC到数据库失败: %w", err)
		}

		t.logger.Info("同步阿里云VPC成功", zap.Int("accountId", account.ID), zap.String("region", region), zap.Int("count", len(allVpcs)))

	case model.CloudProviderHuawei:
		if huaweiProvider == nil {
			return fmt.Errorf("华为云Provider为空")
		}

		// 循环获取所有VPC
		page := 1
		pageSize := 100
		var allVpcs []*model.ResourceVpc

		for {
			resp, _, err := huaweiProvider.VpcService.ListVpcs(ctx, &huawei.ListVpcsRequest{
				Region: region,
				Page:   page,
				Size:   pageSize,
			})
			if err != nil {
				return fmt.Errorf("获取华为云VPC失败: %w", err)
			}

			if resp == nil || len(resp.Vpcs) == 0 {
				break
			}

			// 转换华为云VPC为统一模型
			for _, vpc := range resp.Vpcs {
				vpcResource := t.convertHuaweiVPCToResourceVpc(vpc, account, region)
				allVpcs = append(allVpcs, vpcResource)
			}

			if len(resp.Vpcs) < pageSize {
				break
			}

			page++
		}

		// 同步到数据库
		if err := t.vpcDao.SyncVPCResources(ctx, allVpcs, int64(len(allVpcs))); err != nil {
			return fmt.Errorf("同步华为云VPC到数据库失败: %w", err)
		}

		t.logger.Info("同步华为云VPC成功", zap.Int("accountId", account.ID), zap.String("region", region), zap.Int("count", len(allVpcs)))
	}

	return nil
}

// syncSecurityGroupResources 同步安全组资源
func (t *treeCloudService) syncSecurityGroupResources(ctx context.Context, account *model.CloudAccount, aliyunProvider *provider.AliyunProviderImpl, huaweiProvider *providerhuawei.HuaweiProviderImpl, region string) error {
	switch account.Provider {
	case model.CloudProviderAliyun:
		if aliyunProvider == nil {
			return fmt.Errorf("阿里云Provider为空")
		}

		// 循环获取所有安全组
		page := 1
		pageSize := 100
		var allSecurityGroups []*model.ResourceSecurityGroup

		for {
			sgs, _, err := aliyunProvider.ListSecurityGroups(ctx, region, page, pageSize)
			if err != nil {
				return fmt.Errorf("获取阿里云安全组失败: %w", err)
			}

			if len(sgs) == 0 {
				break
			}

			allSecurityGroups = append(allSecurityGroups, sgs...)

			if len(sgs) < pageSize {
				break
			}

			page++
		}

		// 同步到数据库
		if err := t.securityGroupDao.SyncSecurityGroupResources(ctx, allSecurityGroups, int64(len(allSecurityGroups))); err != nil {
			return fmt.Errorf("同步阿里云安全组到数据库失败: %w", err)
		}

		t.logger.Info("同步阿里云安全组成功", zap.Int("accountId", account.ID), zap.String("region", region), zap.Int("count", len(allSecurityGroups)))

	case model.CloudProviderHuawei:
		if huaweiProvider == nil {
			return fmt.Errorf("华为云Provider为空")
		}

		// 循环获取所有安全组
		page := 1
		pageSize := 100
		var allSecurityGroups []*model.ResourceSecurityGroup

		for {
			resp, _, err := huaweiProvider.SecurityGroupService.ListSecurityGroups(ctx, &huawei.ListSecurityGroupsRequest{
				Region:     region,
				PageNumber: page,
				PageSize:   pageSize,
			})
			if err != nil {
				return fmt.Errorf("获取华为云安全组失败: %w", err)
			}

			if resp == nil || len(resp.SecurityGroups) == 0 {
				break
			}

			// 转换华为云安全组为统一模型
			for _, sg := range resp.SecurityGroups {
				sgResource := t.convertHuaweiSecurityGroupToResourceSecurityGroup(sg, account, region)
				allSecurityGroups = append(allSecurityGroups, sgResource)
			}

			if len(resp.SecurityGroups) < pageSize {
				break
			}

			page++
		}

		// 同步到数据库
		if err := t.securityGroupDao.SyncSecurityGroupResources(ctx, allSecurityGroups, int64(len(allSecurityGroups))); err != nil {
			return fmt.Errorf("同步华为云安全组到数据库失败: %w", err)
		}

		t.logger.Info("同步华为云安全组成功", zap.Int("accountId", account.ID), zap.String("region", region), zap.Int("count", len(allSecurityGroups)))
	}

	return nil
}

// updateSyncStatus 更新同步状态
func (t *treeCloudService) updateSyncStatus(ctx context.Context, accountId int, resourceType, region, status, errorMessage string, syncCount int64) {
	syncStatus := &model.CloudAccountSyncStatus{
		AccountId:    accountId,
		ResourceType: resourceType,
		Region:       region,
		Status:       status,
		LastSyncTime: time.Now(),
		ErrorMessage: errorMessage,
		SyncCount:    syncCount,
	}

	// 查找现有状态记录
	existingStatus, err := t.dao.GetSyncStatus(ctx, accountId, resourceType, region)
	if err == nil && existingStatus != nil {
		// 更新现有记录
		t.dao.UpdateSyncStatus(ctx, int(existingStatus.ID), syncStatus)
	} else {
		// 创建新记录
		t.dao.CreateSyncStatus(ctx, syncStatus)
	}
}

// 华为云资源转换方法
func (t *treeCloudService) convertHuaweiECSToResourceEcs(instance interface{}, account *model.CloudAccount, region string) *model.ResourceEcs {
	// 需要先转换为华为云SDK的ServerDetail类型
	// 这里使用interface{}是因为从不同的包导入类型会有问题
	// 实际使用中应该有适当的类型转换
	now := time.Now()
	return &model.ResourceEcs{
		Provider:     model.CloudProviderHuawei,
		RegionId:     region,
		LastSyncTime: &now,
		// TODO: 根据实际华为云ECS字段进行映射
		// InstanceId: instance.Id,
		// InstanceName: instance.Name,
		// Status: instance.Status,
		// 其他字段映射...
	}
}

func (t *treeCloudService) convertHuaweiVPCToResourceVpc(vpc interface{}, account *model.CloudAccount, region string) *model.ResourceVpc {
	// TODO: 根据华为云VPC结构实现转换
	now := time.Now()
	return &model.ResourceVpc{
		Provider:     model.CloudProviderHuawei,
		RegionId:     region,
		LastSyncTime: now,
		// TODO: 根据实际华为云VPC字段进行映射
		// VpcId: vpc.Id,
		// VpcName: vpc.Name,
		// Status: vpc.Status,
		// 其他字段映射...
	}
}

func (t *treeCloudService) convertHuaweiSecurityGroupToResourceSecurityGroup(sg interface{}, account *model.CloudAccount, region string) *model.ResourceSecurityGroup {
	// TODO: 根据华为云安全组结构实现转换
	now := time.Now()
	return &model.ResourceSecurityGroup{
		Provider:     model.CloudProviderHuawei,
		RegionId:     region,
		LastSyncTime: now,
		// TODO: 根据实际华为云安全组字段进行映射
		// InstanceId: sg.Id,
		// SecurityGroupName: sg.Name,
		// Status: sg.Status,
		// 其他字段映射...
	}
}
