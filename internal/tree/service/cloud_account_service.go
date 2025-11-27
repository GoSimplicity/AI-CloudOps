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
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CloudAccountService interface {
	GetCloudAccountList(ctx context.Context, req *model.GetCloudAccountListReq) (model.ListResp[*model.CloudAccount], error)
	GetCloudAccountDetail(ctx context.Context, req *model.GetCloudAccountDetailReq) (*model.CloudAccount, error)
	CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq, createUserID int, createUserName string) error
	UpdateCloudAccount(ctx context.Context, req *model.UpdateCloudAccountReq) error
	DeleteCloudAccount(ctx context.Context, req *model.DeleteCloudAccountReq) error
	UpdateCloudAccountStatus(ctx context.Context, req *model.UpdateCloudAccountStatusReq) error
	VerifyCloudAccount(ctx context.Context, req *model.VerifyCloudAccountReq) error
	BatchDeleteCloudAccount(ctx context.Context, req *model.BatchDeleteCloudAccountReq) error
	BatchUpdateCloudAccountStatus(ctx context.Context, req *model.BatchUpdateCloudAccountStatusReq) error
	ImportCloudAccount(ctx context.Context, req *model.ImportCloudAccountReq, createUserID int, createUserName string) (*model.ImportCloudAccountResp, error)
	ExportCloudAccount(ctx context.Context, req *model.ExportCloudAccountReq) (interface{}, error)
}

type cloudAccountService struct {
	logger *zap.Logger
	dao    dao.CloudAccountDAO
}

func NewCloudAccountService(logger *zap.Logger, dao dao.CloudAccountDAO) CloudAccountService {
	return &cloudAccountService{
		logger: logger,
		dao:    dao,
	}
}

// GetCloudAccountList 获取云账户列表
func (s *cloudAccountService) GetCloudAccountList(ctx context.Context, req *model.GetCloudAccountListReq) (model.ListResp[*model.CloudAccount], error) {
	// 兜底分页参数
	treeUtils.ValidateAndSetPaginationDefaults(&req.Page, &req.Size)

	// 设置默认排序
	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}

	// 记录查询参数
	s.logger.Debug("获取云账户列表",
		zap.Int("page", req.Page),
		zap.Int("size", req.Size),
		zap.String("search", req.Search),
		zap.Int8("provider", int8(req.Provider)),
		zap.Int8("status", int8(req.Status)),
		zap.String("order_by", req.OrderBy),
		zap.String("order", req.Order))

	accounts, total, err := s.dao.GetList(ctx, req)
	if err != nil {
		s.logger.Error("获取云账户列表失败",
			zap.Int("page", req.Page),
			zap.Int("size", req.Size),
			zap.Error(err))
		return model.ListResp[*model.CloudAccount]{}, fmt.Errorf("获取云账户列表失败: %w", err)
	}

	// 清理敏感信息（双重保险，虽然json:"-"标签已经防止序列化）
	treeUtils.SanitizeCloudAccounts(accounts)

	// 记录成功日志
	s.logger.Info("成功获取云账户列表",
		zap.Int64("total", total),
		zap.Int("returned", len(accounts)),
		zap.Int("page", req.Page),
		zap.Int("size", req.Size))

	return model.ListResp[*model.CloudAccount]{
		Items: accounts,
		Total: total,
	}, nil
}

// GetCloudAccountDetail 获取云账户详情
func (s *cloudAccountService) GetCloudAccountDetail(ctx context.Context, req *model.GetCloudAccountDetailReq) (*model.CloudAccount, error) {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return nil, fmt.Errorf("无效的云账户ID: %w", err)
	}

	s.logger.Debug("获取云账户详情", zap.Int("id", req.ID))

	account, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("云账户不存在", zap.Int("id", req.ID))
			return nil, errors.New("云账户不存在")
		}
		s.logger.Error("获取云账户详情失败",
			zap.Int("id", req.ID),
			zap.Error(err))
		return nil, fmt.Errorf("获取云账户详情失败: %w", err)
	}

	// 清理敏感信息（双重保险，虽然json:"-"标签已经防止序列化）
	treeUtils.SanitizeCloudAccount(account)

	// 记录成功日志（包含关键信息，但不包含敏感数据）
	s.logger.Info("成功获取云账户详情",
		zap.Int("id", account.ID),
		zap.String("name", account.Name),
		zap.Int8("provider", int8(account.Provider)),
		zap.Int("region_count", len(account.Regions)),
		zap.Int("resource_count", len(account.CloudResources)))

	return account, nil
}

// CreateCloudAccount 创建云账户（支持多区域）
func (s *cloudAccountService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq, createUserID int, createUserName string) error {
	// 验证和规范化区域列表
	normalizedRegions, err := treeUtils.ValidateAndNormalizeRegions(req.Regions)
	if err != nil {
		return fmt.Errorf("区域验证失败: %w", err)
	}

	// 检查账户名称是否已存在（同一云厂商下）
	exists, err := s.dao.CheckNameExists(ctx, req.Name, req.Provider, 0)
	if err != nil {
		s.logger.Error("检查云账户名称是否存在失败", zap.Error(err))
		return fmt.Errorf("检查云账户名称失败: %w", err)
	}

	if exists {
		return fmt.Errorf("云账户名称 %s 在 %s 下已存在", req.Name, treeUtils.GetProviderName(req.Provider))
	}

	// 加密 AccessKey 和 SecretKey
	encryptedAccessKey, err := treeUtils.EncryptPassword(req.AccessKey)
	if err != nil {
		s.logger.Error("加密AccessKey失败", zap.Error(err))
		return fmt.Errorf("加密AccessKey失败: %w", err)
	}

	encryptedSecretKey, err := treeUtils.EncryptPassword(req.SecretKey)
	if err != nil {
		s.logger.Error("加密SecretKey失败", zap.Error(err))
		return fmt.Errorf("加密SecretKey失败: %w", err)
	}

	// 使用事务创建云账户和区域关联
	if err := s.dao.CreateWithTransaction(ctx, func(tx interface{}) error {
		// 创建云账户对象
		account := &model.CloudAccount{
			Name:           req.Name,
			Provider:       req.Provider,
			AccessKey:      encryptedAccessKey,
			SecretKey:      encryptedSecretKey,
			AccountID:      req.AccountID,
			AccountName:    req.AccountName,
			AccountAlias:   req.AccountAlias,
			Description:    req.Description,
			Status:         model.CloudAccountEnabled, // 默认启用
			CreateUserID:   createUserID,
			CreateUserName: createUserName,
		}

		if err := s.dao.CreateInTransaction(ctx, account, tx); err != nil {
			s.logger.Error("在事务中创建云账户失败",
				zap.String("name", req.Name),
				zap.Error(err))
			return fmt.Errorf("创建云账户失败: %w", err)
		}

		// 创建区域关联（使用规范化后的区域列表）
		for _, regionItem := range normalizedRegions {
			region := &model.CloudAccountRegion{
				CloudAccountID: account.ID,
				Region:         regionItem.Region,
				RegionName:     regionItem.RegionName,
				IsDefault:      regionItem.IsDefault,
				Description:    regionItem.Description,
				Status:         model.CloudAccountRegionEnabled,
				CreateUserID:   createUserID,
				CreateUserName: createUserName,
			}

			if err := s.dao.CreateRegionInTransaction(ctx, region, tx); err != nil {
				s.logger.Error("在事务中创建云账户区域关联失败",
					zap.Int("account_id", account.ID),
					zap.String("region", regionItem.Region),
					zap.Error(err))
				return fmt.Errorf("创建云账户区域关联失败: %w", err)
			}
		}

		s.logger.Info("成功创建云账户",
			zap.Int("account_id", account.ID),
			zap.String("name", account.Name),
			zap.Int8("provider", int8(account.Provider)),
			zap.Int("region_count", len(normalizedRegions)))

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// UpdateCloudAccount 更新云账户（支持更新区域）
func (s *cloudAccountService) UpdateCloudAccount(ctx context.Context, req *model.UpdateCloudAccountReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账户ID: %w", err)
	}

	// 检查云账户是否存在
	account, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账户不存在")
		}
		s.logger.Error("获取云账户失败", zap.Int("id", req.ID), zap.Error(err))
		return fmt.Errorf("获取云账户失败: %w", err)
	}

	// 如果修改了名称，检查新名称是否已存在（同一云厂商下）
	if req.Name != "" && req.Name != account.Name {
		exists, err := s.dao.CheckNameExists(ctx, req.Name, account.Provider, req.ID)
		if err != nil {
			s.logger.Error("检查云账户名称是否存在失败", zap.Error(err))
			return fmt.Errorf("检查云账户名称失败: %w", err)
		}
		if exists {
			return fmt.Errorf("云账户名称 %s 在 %s 下已存在", req.Name, treeUtils.GetProviderName(account.Provider))
		}
	}

	// 如果需要更新区域，验证区域列表
	var normalizedRegions []model.CreateCloudAccountRegionItem
	if len(req.Regions) > 0 {
		normalizedRegions, err = treeUtils.ValidateAndNormalizeRegions(req.Regions)
		if err != nil {
			return fmt.Errorf("区域验证失败: %w", err)
		}
	}

	// 使用事务更新云账户和区域
	if err := s.dao.CreateWithTransaction(ctx, func(tx interface{}) error {
		// 构建更新对象和字段列表
		updateAccount := &model.CloudAccount{
			Model: model.Model{ID: req.ID},
		}
		updateFields := make([]string, 0)

		// 基本信息字段
		if req.Name != "" {
			updateAccount.Name = req.Name
			updateFields = append(updateFields, "name")
		}
		if req.AccountID != "" {
			updateAccount.AccountID = req.AccountID
			updateFields = append(updateFields, "account_id")
		}
		if req.AccountName != "" {
			updateAccount.AccountName = req.AccountName
			updateFields = append(updateFields, "account_name")
		}
		if req.AccountAlias != "" {
			updateAccount.AccountAlias = req.AccountAlias
			updateFields = append(updateFields, "account_alias")
		}
		if req.Description != "" {
			updateAccount.Description = req.Description
			updateFields = append(updateFields, "description")
		}

		// 加密并更新 AccessKey
		if req.AccessKey != "" {
			encryptedAccessKey, err := treeUtils.EncryptPassword(req.AccessKey)
			if err != nil {
				s.logger.Error("加密AccessKey失败", zap.Error(err))
				return fmt.Errorf("加密AccessKey失败: %w", err)
			}
			updateAccount.AccessKey = encryptedAccessKey
			updateFields = append(updateFields, "access_key")
		}

		// 加密并更新 SecretKey
		if req.SecretKey != "" {
			encryptedSecretKey, err := treeUtils.EncryptPassword(req.SecretKey)
			if err != nil {
				s.logger.Error("加密SecretKey失败", zap.Error(err))
				return fmt.Errorf("加密SecretKey失败: %w", err)
			}
			updateAccount.SecretKey = encryptedSecretKey
			updateFields = append(updateFields, "secret_key")
		}

		// 更新云账户基本信息（如果有字段需要更新）
		if len(updateFields) > 0 {
			if err := s.dao.UpdateWithFields(ctx, updateAccount, updateFields); err != nil {
				s.logger.Error("更新云账户基本信息失败",
					zap.Int("id", req.ID),
					zap.Strings("fields", updateFields),
					zap.Error(err))
				return fmt.Errorf("更新云账户基本信息失败: %w", err)
			}
		}

		// 如果需要更新区域，先删除旧的区域关联，再创建新的
		if len(normalizedRegions) > 0 {
			// 删除旧的区域关联
			if err := s.dao.DeleteRegionsByAccountIDInTransaction(ctx, req.ID, tx); err != nil {
				s.logger.Error("删除旧区域关联失败",
					zap.Int("account_id", req.ID),
					zap.Error(err))
				return fmt.Errorf("删除旧区域关联失败: %w", err)
			}

			// 创建新的区域关联
			for _, regionItem := range normalizedRegions {
				region := &model.CloudAccountRegion{
					CloudAccountID: req.ID,
					Region:         regionItem.Region,
					RegionName:     regionItem.RegionName,
					IsDefault:      regionItem.IsDefault,
					Description:    regionItem.Description,
					Status:         model.CloudAccountRegionEnabled,
					CreateUserID:   account.CreateUserID, // 保持原创建者
					CreateUserName: account.CreateUserName,
				}

				if err := s.dao.CreateRegionInTransaction(ctx, region, tx); err != nil {
					s.logger.Error("创建新区域关联失败",
						zap.Int("account_id", req.ID),
						zap.String("region", regionItem.Region),
						zap.Error(err))
					return fmt.Errorf("创建新区域关联失败: %w", err)
				}
			}

			s.logger.Info("成功更新云账户区域",
				zap.Int("account_id", req.ID),
				zap.Int("region_count", len(normalizedRegions)))
		}

		s.logger.Info("成功更新云账户",
			zap.Int("account_id", req.ID),
			zap.String("name", account.Name),
			zap.Strings("updated_fields", updateFields),
			zap.Bool("regions_updated", len(normalizedRegions) > 0))

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// DeleteCloudAccount 删除云账户
func (s *cloudAccountService) DeleteCloudAccount(ctx context.Context, req *model.DeleteCloudAccountReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账户ID: %w", err)
	}

	// 检查云账户是否存在
	account, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账户不存在")
		}
		return err
	}

	// 检查是否有关联的云资源
	if len(account.CloudResources) > 0 {
		return fmt.Errorf("云账户下还有 %d 个云资源，请先删除相关资源", len(account.CloudResources))
	}

	if err := s.dao.Delete(ctx, req.ID); err != nil {
		s.logger.Error("删除云账户失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateCloudAccountStatus 更新云账户状态
func (s *cloudAccountService) UpdateCloudAccountStatus(ctx context.Context, req *model.UpdateCloudAccountStatusReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账户ID: %w", err)
	}

	if err := s.dao.UpdateStatus(ctx, req.ID, req.Status); err != nil {
		s.logger.Error("更新云账户状态失败", zap.Error(err))
		return err
	}

	return nil
}

// VerifyCloudAccount 验证云账户凭证
func (s *cloudAccountService) VerifyCloudAccount(ctx context.Context, req *model.VerifyCloudAccountReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账户ID: %w", err)
	}

	account, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账户不存在")
		}
		return err
	}

	// 解密密钥
	accessKey, err := treeUtils.DecryptPassword(account.AccessKey)
	if err != nil {
		s.logger.Error("解密AccessKey失败", zap.Error(err))
		return fmt.Errorf("解密AccessKey失败: %w", err)
	}

	secretKey, err := treeUtils.DecryptPassword(account.SecretKey)
	if err != nil {
		s.logger.Error("解密SecretKey失败", zap.Error(err))
		return fmt.Errorf("解密SecretKey失败: %w", err)
	}

	// 获取默认区域用于验证凭证
	defaultRegion, err := treeUtils.GetDefaultRegion(account.Regions)
	if err != nil {
		return fmt.Errorf("获取默认区域失败: %w", err)
	}

	// 根据 Provider 调用相应的云厂商 SDK 验证凭证
	verifyReq := &model.VerifyCloudCredentialsReq{
		Provider:  account.Provider,
		Region:    defaultRegion,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	switch account.Provider {
	case model.ProviderAliyun:
		if err := treeUtils.VerifyAliyunCredentials(ctx, verifyReq, s.logger); err != nil {
			s.logger.Error("阿里云凭证验证失败", zap.Int("id", req.ID), zap.Error(err))
			return fmt.Errorf("阿里云凭证验证失败: %w", err)
		}
	case model.ProviderTencent:
		if err := treeUtils.VerifyTencentCredentials(ctx, verifyReq, s.logger); err != nil {
			s.logger.Error("腾讯云凭证验证失败", zap.Int("id", req.ID), zap.Error(err))
			return fmt.Errorf("腾讯云凭证验证失败: %w", err)
		}
	case model.ProviderAWS:
		if err := treeUtils.VerifyAWSCredentials(ctx, verifyReq, s.logger); err != nil {
			s.logger.Error("AWS凭证验证失败", zap.Int("id", req.ID), zap.Error(err))
			return fmt.Errorf("AWS凭证验证失败: %w", err)
		}
	case model.ProviderHuawei:
		if err := treeUtils.VerifyHuaweiCredentials(ctx, verifyReq, s.logger); err != nil {
			s.logger.Error("华为云凭证验证失败", zap.Int("id", req.ID), zap.Error(err))
			return fmt.Errorf("华为云凭证验证失败: %w", err)
		}
	case model.ProviderAzure:
		if err := treeUtils.VerifyAzureCredentials(ctx, verifyReq, s.logger); err != nil {
			s.logger.Error("Azure凭证验证失败", zap.Int("id", req.ID), zap.Error(err))
			return fmt.Errorf("Azure凭证验证失败: %w", err)
		}
	case model.ProviderGCP:
		if err := treeUtils.VerifyGCPCredentials(ctx, verifyReq, s.logger); err != nil {
			s.logger.Error("GCP凭证验证失败", zap.Int("id", req.ID), zap.Error(err))
			return fmt.Errorf("GCP凭证验证失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的云厂商: %d", account.Provider)
	}

	s.logger.Info("云账户凭证验证成功",
		zap.Int("id", req.ID),
		zap.Int8("provider", int8(account.Provider)),
		zap.String("region", defaultRegion))

	return nil
}

// BatchDeleteCloudAccount 批量删除云账户
func (s *cloudAccountService) BatchDeleteCloudAccount(ctx context.Context, req *model.BatchDeleteCloudAccountReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量删除ID列表不能为空")
	}

	// 检查所有云账户是否存在及是否有关联的云资源
	accounts, err := s.dao.GetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(accounts) != len(req.IDs) {
		return errors.New("部分云账户不存在")
	}

	// 检查是否有关联的云资源
	for _, account := range accounts {
		if len(account.CloudResources) > 0 {
			return fmt.Errorf("云账户 %s 下还有 %d 个云资源，无法删除", account.Name, len(account.CloudResources))
		}
	}

	// 执行批量删除
	if err := s.dao.BatchDelete(ctx, req.IDs); err != nil {
		s.logger.Error("批量删除云账户失败", zap.Error(err))
		return err
	}

	s.logger.Info("批量删除云账户成功", zap.Ints("ids", req.IDs))
	return nil
}

// BatchUpdateCloudAccountStatus 批量更新云账户状态
func (s *cloudAccountService) BatchUpdateCloudAccountStatus(ctx context.Context, req *model.BatchUpdateCloudAccountStatusReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量更新ID列表不能为空")
	}

	// 检查所有云账户是否存在
	accounts, err := s.dao.GetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(accounts) != len(req.IDs) {
		return errors.New("部分云账户不存在")
	}

	// 执行批量更新状态
	if err := s.dao.BatchUpdateStatus(ctx, req.IDs, req.Status); err != nil {
		s.logger.Error("批量更新云账户状态失败", zap.Error(err))
		return err
	}

	s.logger.Info("批量更新云账户状态成功",
		zap.Ints("ids", req.IDs),
		zap.Int8("status", int8(req.Status)))
	return nil
}

// ImportCloudAccount 导入云账户
func (s *cloudAccountService) ImportCloudAccount(ctx context.Context, req *model.ImportCloudAccountReq, createUserID int, createUserName string) (*model.ImportCloudAccountResp, error) {
	if len(req.Accounts) == 0 {
		return nil, errors.New("导入账户列表不能为空")
	}

	resp := &model.ImportCloudAccountResp{
		SuccessCount: 0,
		FailedCount:  0,
		FailedItems:  make([]string, 0),
	}

	// 逐个导入云账户
	for _, accountReq := range req.Accounts {
		// 检查账户名称是否已存在
		exists, err := s.dao.CheckNameExists(ctx, accountReq.Name, accountReq.Provider, 0)
		if err != nil {
			s.logger.Error("检查账户名称是否存在失败", zap.Error(err))
			resp.FailedCount++
			resp.FailedItems = append(resp.FailedItems, fmt.Sprintf("%s (检查失败)", accountReq.Name))
			continue
		}

		if exists {
			s.logger.Warn("云账户已存在", zap.String("name", accountReq.Name))
			resp.FailedCount++
			resp.FailedItems = append(resp.FailedItems, fmt.Sprintf("%s (已存在)", accountReq.Name))
			continue
		}

		// 创建云账户
		if err := s.CreateCloudAccount(ctx, &accountReq, createUserID, createUserName); err != nil {
			s.logger.Error("导入云账户失败",
				zap.String("name", accountReq.Name),
				zap.Error(err))
			resp.FailedCount++
			resp.FailedItems = append(resp.FailedItems, fmt.Sprintf("%s (%s)", accountReq.Name, err.Error()))
			continue
		}

		resp.SuccessCount++
	}

	// 生成提示信息
	if resp.FailedCount == 0 {
		resp.Message = fmt.Sprintf("成功导入 %d 个云账户", resp.SuccessCount)
	} else {
		resp.Message = fmt.Sprintf("成功导入 %d 个云账户，失败 %d 个", resp.SuccessCount, resp.FailedCount)
	}

	s.logger.Info("云账户导入完成",
		zap.Int("success", resp.SuccessCount),
		zap.Int("failed", resp.FailedCount))

	return resp, nil
}

// ExportCloudAccount 导出云账户
func (s *cloudAccountService) ExportCloudAccount(ctx context.Context, req *model.ExportCloudAccountReq) (interface{}, error) {
	var accounts []*model.CloudAccount
	var err error

	// 根据条件获取云账户列表
	if len(req.IDs) > 0 {
		// 导出指定的云账户
		accounts, err = s.dao.GetByIDs(ctx, req.IDs)
		if err != nil {
			s.logger.Error("获取指定云账户失败", zap.Error(err))
			return nil, err
		}
	} else {
		// 导出所有云账户（支持按云厂商过滤）
		accounts, err = s.dao.GetAll(ctx, req.Provider)
		if err != nil {
			s.logger.Error("获取所有云账户失败", zap.Error(err))
			return nil, err
		}
	}

	if len(accounts) == 0 {
		return nil, errors.New("没有可导出的云账户")
	}

	// 根据导出格式处理数据
	format := req.Format
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		return treeUtils.ExportAsJSON(accounts), nil
	case "csv":
		return treeUtils.ExportAsCSV(accounts), nil
	default:
		return nil, fmt.Errorf("不支持的导出格式: %s，仅支持json和csv", format)
	}
}
