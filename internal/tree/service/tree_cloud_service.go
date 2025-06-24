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
	"strings"

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
	UpdateCloudAccount(ctx context.Context, id int, req *model.UpdateCloudAccountReq) error
	DeleteCloudAccount(ctx context.Context, id int) error
	GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error)
	ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error)
	TestCloudAccount(ctx context.Context, id int) error
	SyncCloudResources(ctx context.Context, req *model.SyncCloudReq) error

	// 批量操作方法
	BatchDeleteCloudAccounts(ctx context.Context, accountIDs []int) error
	BatchTestCloudAccounts(ctx context.Context, accountIDs []int) (map[int]error, error)

	// 同步方法
	SyncCloudAccountResources(ctx context.Context, req *model.SyncCloudAccountResourcesReq) error

	// 统计方法
	GetCloudAccountStatistics(ctx context.Context) (*model.CloudAccountStatistics, error)

	// 加密相关方法
	GetDecryptedSecretKey(ctx context.Context, accountId int) (string, error)
	ReEncryptAccount(ctx context.Context, accountId int) error
}

type treeCloudService struct {
	logger          *zap.Logger
	dao             dao.TreeCloudDAO
	cryptoManager   utils.CryptoManager
	providerFactory *provider.ProviderFactory
}

func NewTreeCloudService(logger *zap.Logger, dao dao.TreeCloudDAO, cryptoManager utils.CryptoManager, providerFactory *provider.ProviderFactory) TreeCloudService {
	return &treeCloudService{
		logger:          logger,
		dao:             dao,
		cryptoManager:   cryptoManager,
		providerFactory: providerFactory,
	}
}

// CreateCloudAccount 创建云账户
func (t *treeCloudService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	t.logger.Info("开始创建云账户", zap.String("name", req.Name), zap.String("provider", string(req.Provider)))

	// 参数校验
	if err := t.validateCreateRequest(req); err != nil {
		t.logger.Error("创建云账户参数校验失败", zap.Error(err))
		return fmt.Errorf("参数校验失败: %w", err)
	}

	// 加密SecretKey
	encryptedSecret, err := t.cryptoManager.EncryptSecretKey(req.SecretKey)
	if err != nil {
		t.logger.Error("加密SecretKey失败", zap.String("name", req.Name), zap.Error(err))
		return fmt.Errorf("加密SecretKey失败: %w", err)
	}

	// 构造CloudAccount模型
	account := &model.CloudAccount{
		Name:            req.Name,
		Provider:        req.Provider,
		AccountId:       req.AccountId,
		AccessKey:       req.AccessKey,
		EncryptedSecret: encryptedSecret,
		Regions:         req.Regions,
		IsEnabled:       req.IsEnabled,
		Description:     req.Description,
	}

	// 调用DAO创建账户
	if err := t.dao.CreateCloudAccount(ctx, account); err != nil {
		t.logger.Error("DAO创建云账户失败", zap.String("name", req.Name), zap.Error(err))
		return fmt.Errorf("创建云账户失败: %w", err)
	}

	// 获取用户信息
	userInfo := t.getUserInfoFromContext(ctx)

	// 创建审计日志
	auditLog := &model.CloudAccountAuditLog{
		AccountId: int(account.ID),
		Operation: model.OperationCreate,
		Operator:  userInfo.Username,
		Details:   fmt.Sprintf("创建云账户: %s, 云厂商: %s", req.Name, req.Provider),
		IPAddress: userInfo.IP,
		UserAgent: userInfo.UserAgent,
	}

	if err := t.dao.CreateAuditLog(ctx, auditLog); err != nil {
		t.logger.Warn("创建审计日志失败", zap.Error(err))
		// 不返回错误，因为主要操作已成功
	}

	t.logger.Info("云账户创建成功", zap.Int("id", int(account.ID)), zap.String("name", req.Name))
	return nil
}

// UpdateCloudAccount 更新云账户
func (t *treeCloudService) UpdateCloudAccount(ctx context.Context, id int, req *model.UpdateCloudAccountReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	t.logger.Info("开始更新云账户", zap.Int("id", id))

	// 参数校验
	if err := t.validateUpdateRequest(req); err != nil {
		t.logger.Error("更新云账户参数校验失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("参数校验失败: %w", err)
	}

	// 获取现有账户信息
	existingAccount, err := t.dao.GetCloudAccount(ctx, id)
	if err != nil {
		t.logger.Error("获取现有云账户失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取云账户失败: %w", err)
	}

	// 构造更新数据
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
		encryptedSecret, err := t.cryptoManager.EncryptSecretKey(req.SecretKey)
		if err != nil {
			t.logger.Error("加密SecretKey失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("加密SecretKey失败: %w", err)
		}
		updateData.EncryptedSecret = encryptedSecret
	} else {
		// 保持原有加密密钥
		updateData.EncryptedSecret = existingAccount.EncryptedSecret
	}

	// 调用DAO更新账户
	if err := t.dao.UpdateCloudAccount(ctx, id, updateData); err != nil {
		t.logger.Error("DAO更新云账户失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("更新云账户失败: %w", err)
	}

	// 获取用户信息
	userInfo := t.getUserInfoFromContext(ctx)

	// 创建审计日志
	auditLog := &model.CloudAccountAuditLog{
		AccountId: id,
		Operation: model.OperationUpdate,
		Operator:  userInfo.Username,
		Details:   fmt.Sprintf("更新云账户: %s", req.Name),
		IPAddress: userInfo.IP,
		UserAgent: userInfo.UserAgent,
	}

	if err := t.dao.CreateAuditLog(ctx, auditLog); err != nil {
		t.logger.Warn("创建审计日志失败", zap.Error(err))
		// 不返回错误，因为主要操作已成功
	}

	t.logger.Info("云账户更新成功", zap.Int("id", id))
	return nil
}

// GetCloudAccount 获取云账户详情（包含解密后的SecretKey）
func (t *treeCloudService) GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context 不能为空")
	}
	t.logger.Debug("获取云账户详情", zap.Int("id", id))

	// 调用DAO获取账户信息
	account, err := t.dao.GetCloudAccount(ctx, id)
	if err != nil {
		t.logger.Error("DAO获取云账户失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("获取云账户失败: %w", err)
	}

	// 解密SecretKey
	decryptedSecret, err := t.cryptoManager.DecryptSecretKey(account.EncryptedSecret)
	if err != nil {
		t.logger.Error("解密SecretKey失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("解密SecretKey失败: %w", err)
	}

	// 创建返回对象，包含解密后的SecretKey
	result := &model.CloudAccount{
		Model:           account.Model,
		Name:            account.Name,
		Provider:        account.Provider,
		AccountId:       account.AccountId,
		AccessKey:       account.AccessKey,
		EncryptedSecret: decryptedSecret, // 返回解密后的明文
		Regions:         account.Regions,
		IsEnabled:       account.IsEnabled,
		LastSyncTime:    account.LastSyncTime,
		Description:     account.Description,
	}

	t.logger.Debug("云账户详情获取成功", zap.Int("id", id))
	return result, nil
}

// ListCloudAccounts 获取云账户列表（不包含SecretKey）
func (t *treeCloudService) ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error) {
	if req == nil {
		return model.ListResp[model.CloudAccount]{}, fmt.Errorf("请求参数不能为空")
	}
	t.logger.Debug("获取云账户列表",
		zap.Int("page", req.Page),
		zap.Int("pageSize", req.PageSize),
		zap.String("name", req.Name),
		zap.String("provider", string(req.Provider)),
		zap.Bool("enabled", req.Enabled))

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

	t.logger.Debug("云账户列表获取成功", zap.Int64("total", result.Total), zap.Int("count", len(result.Items)))
	return result, nil
}

// DeleteCloudAccount 删除云账户
func (t *treeCloudService) DeleteCloudAccount(ctx context.Context, id int) error {
	if ctx == nil {
		return fmt.Errorf("context 不能为空")
	}
	t.logger.Info("开始删除云账户", zap.Int("id", id))

	// 获取账户信息用于审计日志
	account, err := t.dao.GetCloudAccount(ctx, id)
	if err != nil {
		t.logger.Error("获取云账户信息失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取云账户失败: %w", err)
	}

	// 调用DAO删除账户
	if err := t.dao.DeleteCloudAccount(ctx, id); err != nil {
		t.logger.Error("DAO删除云账户失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("删除云账户失败: %w", err)
	}

	// 获取用户信息
	userInfo := t.getUserInfoFromContext(ctx)

	// 创建审计日志
	auditLog := &model.CloudAccountAuditLog{
		AccountId: id,
		Operation: model.OperationDelete,
		Operator:  userInfo.Username,
		Details:   fmt.Sprintf("删除云账户: %s", account.Name),
		IPAddress: userInfo.IP,
		UserAgent: userInfo.UserAgent,
	}

	if err := t.dao.CreateAuditLog(ctx, auditLog); err != nil {
		t.logger.Warn("创建审计日志失败", zap.Error(err))
		// 不返回错误，因为主要操作已成功
	}

	t.logger.Info("云账户删除成功", zap.Int("id", id), zap.String("name", account.Name))
	return nil
}

// TestCloudAccount 测试云账户连接
func (t *treeCloudService) TestCloudAccount(ctx context.Context, id int) error {
	if ctx == nil {
		return fmt.Errorf("context 不能为空")
	}

	t.logger.Info("开始测试云账户连接", zap.Int("id", id))

	// 获取账户信息
	account, err := t.dao.GetCloudAccount(ctx, id)
	if err != nil {
		t.logger.Error("获取云账户信息失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取云账户信息失败: %w", err)
	}

	// 获取解密后的SecretKey
	secretKey, err := t.dao.GetDecryptedSecretKey(ctx, id)
	if err != nil {
		t.logger.Error("获取解密后的SecretKey失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取SecretKey失败: %w", err)
	}

	switch account.Provider {
	case model.CloudProviderHuawei:
		region := "cn-north-4"
		if len(account.Regions) > 0 && account.Regions[0] != "" {
			region = account.Regions[0]
		}
		sdk := huawei.NewSDK(t.logger, account.AccessKey, secretKey)
		ecsService := huawei.NewEcsService(sdk)
		_, err := ecsService.ListInstances(ctx, &huawei.ListInstancesRequest{
			Region: region,
			Page:   1,
			Size:   1,
		})
		if err != nil {
			t.logger.Error("华为云连接测试失败", zap.Int("id", id), zap.String("region", region), zap.Error(err))
			return fmt.Errorf("华为云连接测试失败: %w", err)
		}
		t.logger.Info("华为云连接测试成功", zap.Int("id", id), zap.String("region", region))
		return nil
	case model.CloudProviderAliyun:
		region := "cn-hangzhou"
		if len(account.Regions) > 0 && account.Regions[0] != "" {
			region = account.Regions[0]
		}
		aliyunSDK := aliyun.NewSDK(t.logger, account.AccessKey, secretKey)
		ecsClient, err := aliyunSDK.CreateEcsClient(region)
		if err != nil {
			t.logger.Error("阿里云ECS客户端创建失败", zap.Int("id", id), zap.String("region", region), zap.Error(err))
			return fmt.Errorf("阿里云ECS客户端创建失败: %w", err)
		}
		// 调用DescribeRegions做连接测试
		request := &ecs.DescribeRegionsRequest{}
		_, err = ecsClient.DescribeRegions(request)
		if err != nil {
			t.logger.Error("阿里云连接测试失败", zap.Int("id", id), zap.String("region", region), zap.Error(err))
			return fmt.Errorf("阿里云连接测试失败: %w", err)
		}
		t.logger.Info("阿里云连接测试成功", zap.Int("id", id), zap.String("region", region))
		return nil
	default:
		return fmt.Errorf("暂不支持该云厂商的连接测试: %s", account.Provider)
	}
}

// SyncCloudResources 同步云资源
func (t *treeCloudService) SyncCloudResources(ctx context.Context, req *model.SyncCloudReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	t.logger.Info("开始同步云资源",
		zap.Ints("accountIds", req.AccountIds),
		zap.String("resourceType", req.ResourceType),
		zap.Strings("regions", req.Regions),
		zap.Bool("force", req.Force))

	// 1. 获取账号列表
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
		// 2. 解密SecretKey
		secretKey, err := t.dao.GetDecryptedSecretKey(ctx, int(account.ID))
		if err != nil {
			t.logger.Error("解密SecretKey失败", zap.Int("accountId", int(account.ID)), zap.Error(err))
			continue
		}
		account.EncryptedSecret = secretKey

		// 3. 构造Provider
		var (
			aliyunProvider *provider.AliyunProviderImpl
			huaweiProvider *providerhuawei.HuaweiProviderImpl
		)
		switch account.Provider {
		case model.CloudProviderAliyun:
			aliyunProvider = provider.NewAliyunProvider(t.logger, account)
			if aliyunProvider == nil {
				t.logger.Error("阿里云Provider初始化失败", zap.Int("accountId", int(account.ID)))
				continue
			}
		case model.CloudProviderHuawei:
			huaweiProvider = providerhuawei.NewHuaweiProvider(t.logger, account)
			if huaweiProvider == nil {
				t.logger.Error("华为云Provider初始化失败", zap.Int("accountId", int(account.ID)))
				continue
			}
		default:
			t.logger.Warn("暂不支持的云厂商", zap.String("provider", string(account.Provider)))
			continue
		}

		// 4. 区域遍历
		regions := req.Regions
		if len(regions) == 0 {
			regions = account.Regions
		}
		if len(regions) == 0 {
			t.logger.Warn("账号无可用区域", zap.Int("accountId", int(account.ID)))
			continue
		}

		for _, region := range regions {
			// 5. 按资源类型同步
			if req.ResourceType == "" || req.ResourceType == "ecs" {
				switch account.Provider {
				case model.CloudProviderAliyun:
					_, total, err := aliyunProvider.ListInstances(ctx, region, 1, 10)
					if err != nil {
						t.logger.Error("同步阿里云ECS失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云ECS成功", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Int64("total", total))
					}
				case model.CloudProviderHuawei:
					_, err := huaweiProvider.EcsService.ListInstances(ctx, &huawei.ListInstancesRequest{
						Region: region,
						Page:   1,
						Size:   10,
					})
					if err != nil {
						t.logger.Error("同步华为云ECS失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云ECS成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				}
			}
			if req.ResourceType == "" || req.ResourceType == "vpc" {
				switch account.Provider {
				case model.CloudProviderAliyun:
					_, err := aliyunProvider.ListVPCs(ctx, region, 1, 10)
					if err != nil {
						t.logger.Error("同步阿里云VPC失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云VPC成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				case model.CloudProviderHuawei:
					_, err := huaweiProvider.VpcService.ListVpcs(ctx, &huawei.ListVpcsRequest{
						Region: region,
						Page:   1,
						Size:   10,
					})
					if err != nil {
						t.logger.Error("同步华为云VPC失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云VPC成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				}
			}
			if req.ResourceType == "" || req.ResourceType == "disk" {
				switch account.Provider {
				case model.CloudProviderAliyun:
					_, err := aliyunProvider.ListDisks(ctx, region, 1, 10)
					if err != nil {
						t.logger.Error("同步阿里云Disk失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云Disk成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				case model.CloudProviderHuawei:
					_, err := huaweiProvider.DiskService.ListDisks(ctx, &huawei.ListDisksRequest{
						Region: region,
						Page:   1,
						Size:   10,
					})
					if err != nil {
						t.logger.Error("同步华为云Disk失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云Disk成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				}
			}
			if req.ResourceType == "" || req.ResourceType == "sg" {
				switch account.Provider {
				case model.CloudProviderAliyun:
					_, err := aliyunProvider.ListSecurityGroups(ctx, region, 1, 10)
					if err != nil {
						t.logger.Error("同步阿里云SG失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步阿里云SG成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				case model.CloudProviderHuawei:
					_, err := huaweiProvider.SecurityGroupService.ListSecurityGroups(ctx, &huawei.ListSecurityGroupsRequest{
						Region:     region,
						PageNumber: 1,
						PageSize:   10,
					})
					if err != nil {
						t.logger.Error("同步华为云SG失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
					} else {
						t.logger.Info("同步华为云SG成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
					}
				}
			}
		}
	}

	t.logger.Info("云资源同步完成")
	return nil
}

// GetDecryptedSecretKey 获取解密后的SecretKey
func (t *treeCloudService) GetDecryptedSecretKey(ctx context.Context, accountId int) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context 不能为空")
	}
	return t.dao.GetDecryptedSecretKey(ctx, accountId)
}

// ReEncryptAccount 重新加密账户
func (t *treeCloudService) ReEncryptAccount(ctx context.Context, accountId int) error {
	if ctx == nil {
		return fmt.Errorf("context 不能为空")
	}
	return t.dao.ReEncryptAccount(ctx, accountId)
}

// ==================== 私有方法 ====================

// getUserInfoFromContext 从上下文中获取用户信息
func (t *treeCloudService) getUserInfoFromContext(ctx context.Context) *utils.UserInfo {
	// 尝试从context中获取用户信息
	userInfo := utils.GetUserInfoFromContextGeneric(ctx)

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

// validateCreateRequest 校验创建请求参数
func (t *treeCloudService) validateCreateRequest(req *model.CreateCloudAccountReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	if req.Name == "" {
		return fmt.Errorf("账户名称不能为空")
	}
	if req.Provider == "" {
		return fmt.Errorf("云厂商不能为空")
	}
	if req.AccountId == "" {
		return fmt.Errorf("账户ID不能为空")
	}
	if req.AccessKey == "" {
		return fmt.Errorf("AccessKey不能为空")
	}
	if req.SecretKey == "" {
		return fmt.Errorf("SecretKey不能为空")
	}
	return nil
}

// validateUpdateRequest 校验更新请求参数
func (t *treeCloudService) validateUpdateRequest(req *model.UpdateCloudAccountReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	if req.Name == "" {
		return fmt.Errorf("账户名称不能为空")
	}
	if req.Provider == "" {
		return fmt.Errorf("云厂商不能为空")
	}
	if req.AccountId == "" {
		return fmt.Errorf("账户ID不能为空")
	}
	if req.AccessKey == "" {
		return fmt.Errorf("AccessKey不能为空")
	}
	return nil
}

// BatchDeleteCloudAccounts 批量删除云账号
func (t *treeCloudService) BatchDeleteCloudAccounts(ctx context.Context, accountIDs []int) error {
	if len(accountIDs) == 0 {
		return fmt.Errorf("账号ID列表不能为空")
	}

	t.logger.Info("开始批量删除云账号", zap.Ints("accountIDs", accountIDs))

	// 获取用户信息
	userInfo := t.getUserInfoFromContext(ctx)

	var errors []string
	for _, id := range accountIDs {
		// 获取账户信息用于审计日志
		account, err := t.dao.GetCloudAccount(ctx, id)
		if err != nil {
			t.logger.Error("获取云账户信息失败", zap.Int("id", id), zap.Error(err))
			errors = append(errors, fmt.Sprintf("账号ID %d: 获取账户信息失败", id))
			continue
		}

		// 删除账户
		if err := t.dao.DeleteCloudAccount(ctx, id); err != nil {
			t.logger.Error("删除云账户失败", zap.Int("id", id), zap.Error(err))
			errors = append(errors, fmt.Sprintf("账号ID %d: 删除失败", id))
			continue
		}

		// 创建审计日志
		auditLog := &model.CloudAccountAuditLog{
			AccountId: id,
			Operation: model.OperationDelete,
			Operator:  userInfo.Username,
			Details:   fmt.Sprintf("批量删除云账户: %s", account.Name),
			IPAddress: userInfo.IP,
			UserAgent: userInfo.UserAgent,
		}

		if err := t.dao.CreateAuditLog(ctx, auditLog); err != nil {
			t.logger.Warn("创建审计日志失败", zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("批量删除失败: %s", strings.Join(errors, "; "))
	}

	t.logger.Info("批量删除云账号成功", zap.Ints("accountIDs", accountIDs))
	return nil
}

// BatchTestCloudAccounts 批量测试云账号连接
func (t *treeCloudService) BatchTestCloudAccounts(ctx context.Context, accountIDs []int) (map[int]error, error) {
	if len(accountIDs) == 0 {
		return nil, fmt.Errorf("账号ID列表不能为空")
	}

	t.logger.Info("开始批量测试云账号连接", zap.Ints("accountIDs", accountIDs))

	results := make(map[int]error)
	for _, id := range accountIDs {
		if err := t.TestCloudAccount(ctx, id); err != nil {
			results[id] = err
		}
	}

	t.logger.Info("批量测试云账号连接完成", zap.Int("total", len(accountIDs)), zap.Int("failed", len(results)))
	return results, nil
}

// SyncCloudAccountResources 同步指定云账号的资源
func (t *treeCloudService) SyncCloudAccountResources(ctx context.Context, req *model.SyncCloudAccountResourcesReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	t.logger.Info("开始同步云账号资源",
		zap.Int("accountId", req.AccountID),
		zap.String("resourceType", req.ResourceType),
		zap.Strings("regions", req.Regions),
		zap.Bool("force", req.Force),
	)

	// 1. 获取账号
	account, err := t.dao.GetCloudAccount(ctx, req.AccountID)
	if err != nil {
		t.logger.Error("获取云账号失败", zap.Int("accountId", req.AccountID), zap.Error(err))
		return err
	}

	// 2. 解密 SecretKey
	secretKey, err := t.dao.GetDecryptedSecretKey(ctx, req.AccountID)
	if err != nil {
		t.logger.Error("解密SecretKey失败", zap.Int("accountId", req.AccountID), zap.Error(err))
		return err
	}
	account.EncryptedSecret = secretKey

	// 3. 构造 Provider
	var (
		aliyunProvider *provider.AliyunProviderImpl
		huaweiProvider *providerhuawei.HuaweiProviderImpl
	)
	switch account.Provider {
	case model.CloudProviderAliyun:
		aliyunProvider = provider.NewAliyunProvider(t.logger, account)
		if aliyunProvider == nil {
			t.logger.Error("阿里云Provider初始化失败", zap.Int("accountId", int(account.ID)))
			return fmt.Errorf("阿里云Provider初始化失败")
		}
	case model.CloudProviderHuawei:
		huaweiProvider = providerhuawei.NewHuaweiProvider(t.logger, account)
		if huaweiProvider == nil {
			t.logger.Error("华为云Provider初始化失败", zap.Int("accountId", int(account.ID)))
			return fmt.Errorf("华为云Provider初始化失败")
		}
	default:
		t.logger.Warn("暂不支持的云厂商", zap.String("provider", string(account.Provider)))
		return nil
	}

	// 4. 区域
	regions := req.Regions
	if len(regions) == 0 {
		regions = account.Regions
	}
	if len(regions) == 0 {
		t.logger.Warn("账号无可用区域", zap.Int("accountId", int(account.ID)))
		return nil
	}

	// 5. 资源同步
	for _, region := range regions {
		if req.ResourceType == "" || req.ResourceType == "ecs" || req.ResourceType == "all" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				_, total, err := aliyunProvider.ListInstances(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云ECS失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步阿里云ECS成功", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Int64("total", total))
				}
			case model.CloudProviderHuawei:
				_, err := huaweiProvider.EcsService.ListInstances(ctx, &huawei.ListInstancesRequest{
					Region: region,
					Page:   1,
					Size:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云ECS失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步华为云ECS成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			}
		}
		if req.ResourceType == "" || req.ResourceType == "vpc" || req.ResourceType == "all" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				_, err := aliyunProvider.ListVPCs(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云VPC失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步阿里云VPC成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			case model.CloudProviderHuawei:
				_, err := huaweiProvider.VpcService.ListVpcs(ctx, &huawei.ListVpcsRequest{
					Region: region,
					Page:   1,
					Size:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云VPC失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步华为云VPC成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			}
		}
		if req.ResourceType == "" || req.ResourceType == "disk" || req.ResourceType == "all" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				_, err := aliyunProvider.ListDisks(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云Disk失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步阿里云Disk成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			case model.CloudProviderHuawei:
				_, err := huaweiProvider.DiskService.ListDisks(ctx, &huawei.ListDisksRequest{
					Region: region,
					Page:   1,
					Size:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云Disk失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步华为云Disk成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			}
		}
		if req.ResourceType == "" || req.ResourceType == "sg" || req.ResourceType == "all" {
			switch account.Provider {
			case model.CloudProviderAliyun:
				_, err := aliyunProvider.ListSecurityGroups(ctx, region, 1, 10)
				if err != nil {
					t.logger.Error("同步阿里云SG失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步阿里云SG成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			case model.CloudProviderHuawei:
				_, err := huaweiProvider.SecurityGroupService.ListSecurityGroups(ctx, &huawei.ListSecurityGroupsRequest{
					Region:     region,
					PageNumber: 1,
					PageSize:   10,
				})
				if err != nil {
					t.logger.Error("同步华为云SG失败", zap.Int("accountId", int(account.ID)), zap.String("region", region), zap.Error(err))
				} else {
					t.logger.Info("同步华为云SG成功", zap.Int("accountId", int(account.ID)), zap.String("region", region))
				}
			}
		}
	}

	t.logger.Info("云账号资源同步完成", zap.Int("accountId", req.AccountID))
	return nil
}

// GetCloudAccountStatistics 获取云账号统计信息
func (t *treeCloudService) GetCloudAccountStatistics(ctx context.Context) (*model.CloudAccountStatistics, error) {
	t.logger.Debug("获取云账号统计信息")

	// 获取所有账号
	allAccounts, err := t.dao.GetEnabledCloudAccounts(ctx)
	if err != nil {
		t.logger.Error("获取云账号列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取云账号列表失败: %w", err)
	}

	// 获取所有账号（包括禁用的）
	allReq := &model.ListCloudAccountsReq{
		Page:     1,
		PageSize: 1000, // 假设不会有超过1000个账号
	}
	allAccountsResp, err := t.dao.ListCloudAccounts(ctx, allReq)
	if err != nil {
		t.logger.Error("获取所有云账号失败", zap.Error(err))
		return nil, fmt.Errorf("获取所有云账号失败: %w", err)
	}

	stats := &model.CloudAccountStatistics{
		TotalAccounts:    allAccountsResp.Total,
		EnabledAccounts:  int64(len(allAccounts)),
		DisabledAccounts: allAccountsResp.Total - int64(len(allAccounts)),
		ProviderStats:    make(map[string]int64),
		RegionStats:      make(map[string]int64),
		SyncStatus:       make(map[string]int64),
	}

	// 统计各云厂商账号数
	for _, account := range allAccountsResp.Items {
		provider := string(account.Provider)
		stats.ProviderStats[provider]++

		// 统计各区域账号数
		for _, region := range account.Regions {
			stats.RegionStats[region]++
		}
	}

	// 获取同步状态统计
	for _, account := range allAccountsResp.Items {
		statuses, err := t.dao.ListSyncStatus(ctx, int(account.ID))
		if err != nil {
			t.logger.Warn("获取同步状态失败", zap.Int("accountId", int(account.ID)), zap.Error(err))
			continue
		}
		for _, status := range statuses {
			stats.SyncStatus[status.Status]++
		}
	}

	// 获取最近活动（取最近20条审计日志）
	recentLogs := make([]*model.CloudAccountAuditLog, 0, 20)
	for _, account := range allAccountsResp.Items {
		logsResp, err := t.dao.ListAuditLogs(ctx, int(account.ID), 1, 5)
		if err != nil {
			t.logger.Warn("获取审计日志失败", zap.Int("accountId", int(account.ID)), zap.Error(err))
			continue
		}
		for _, log := range logsResp.Items {
			if len(recentLogs) < 20 {
				recentLogs = append(recentLogs, &log)
			}
		}
	}
	stats.RecentActivities = recentLogs

	t.logger.Debug("云账号统计信息获取成功")
	return stats, nil
}
