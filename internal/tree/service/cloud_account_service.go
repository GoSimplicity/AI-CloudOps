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
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CloudAccountService interface {
	GetAvailableRegionsForAccount(context.Context, int) ([]model.AvailableRegion, error)
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
	clientFactory     treeUtils.CloudClientFactory
	credentialFactory treeUtils.CloudCredentialClientFactory
	logger            *zap.Logger
	dao               dao.CloudAccountDAO
}

func NewCloudAccountService(logger *zap.Logger, dao dao.CloudAccountDAO) CloudAccountService {
	return &cloudAccountService{
		logger: logger,
		dao:    dao,
	}
}

func (s *cloudAccountService) GetCloudAccountList(ctx context.Context, req *model.GetCloudAccountListReq) (model.ListResp[*model.CloudAccount], error) {
	treeUtils.ValidateAndSetPaginationDefaults(&req.Page, &req.Size)

	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}

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

func (s *cloudAccountService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq, createUserID int, createUserName string) error {
	initialStatus := model.CloudAccountEnabled
	if req.Status != nil {
		if *req.Status != int(model.CloudAccountEnabled) && *req.Status != int(model.CloudAccountDisabled) {
			return errors.New("初始账户状态只能为启用或停用")
		}
		initialStatus = model.CloudAccountStatus(*req.Status)
	}
	credentials := treeUtils.CloudCredentials{AccessKey: req.AccessKey, SecretKey: req.SecretKey, CredentialType: req.CredentialType, SessionToken: req.SessionToken, ExpiresAt: req.CredentialExpiresAt}
	isStructured := req.Provider == model.ProviderAzure || req.Provider == model.ProviderGCP
	if isStructured {
		if req.ProviderCredentials == nil || req.AccessKey != "" || req.SecretKey != "" || req.SessionToken != "" || req.CredentialExpiresAt != nil || (req.CredentialType != "" && req.CredentialType != "long_term") {
			return errors.New("该云厂商必须提供完整的专用凭据")
		}
		credentials = treeUtils.ProviderCredentialsFromInput(req.ProviderCredentials)
	} else if req.ProviderCredentials != nil {
		return errors.New("该云厂商不支持专用凭据")
	}
	if err := treeUtils.ValidateCloudCredentials(req.Provider, credentials, time.Now().UTC()); err != nil {
		return err
	}
	if credentials.CredentialType == "" {
		credentials.CredentialType = "long_term"
	}
	normalizedRegions, err := treeUtils.ValidateAndNormalizeRegions(req.Regions)
	if err != nil {
		return fmt.Errorf("区域验证失败: %w", err)
	}
	for _, region := range normalizedRegions {
		if err := treeUtils.ValidateCloudRegion(req.Provider, region.Region); err != nil {
			return err
		}
	}
	if req.Provider == model.ProviderHuawei {
		for _, region := range normalizedRegions {
			if region.ProjectID == nil || !treeUtils.ValidHuaweiProjectID(*region.ProjectID) {
				return errors.New("华为云地域必须指定项目ID")
			}
		}
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
	if len(encryptedAccessKey) > 500 || len(encryptedSecretKey) > 500 {
		return errors.New("云账户密钥超出存储限制")
	}
	var encryptedProviderCredentials string
	if isStructured {
		encryptedProviderCredentials, err = treeUtils.EncryptProviderCredentials(req.Provider, req.ProviderCredentials)
		if err != nil {
			return err
		}
	}
	var encryptedToken string
	if req.SessionToken != "" {
		encryptedToken, err = treeUtils.EncryptPassword(req.SessionToken)
		if err != nil || len(encryptedToken) > 16000 {
			return errors.New("无法保存云账户令牌")
		}
	}

	// 使用事务创建云账户和区域关联
	if err := s.dao.CreateWithTransaction(ctx, func(tx interface{}) error {
		account := &model.CloudAccount{
			Name:                req.Name,
			Provider:            req.Provider,
			AccessKey:           encryptedAccessKey,
			SecretKey:           encryptedSecretKey,
			CredentialType:      credentials.CredentialType,
			SessionToken:        encryptedToken,
			ProviderCredentials: encryptedProviderCredentials,
			CredentialExpiresAt: credentials.ExpiresAt,
			AccountID:           req.AccountID,
			AccountName:         req.AccountName,
			AccountAlias:        req.AccountAlias,
			Description:         req.Description,
			Status:              initialStatus,
			CreateUserID:        createUserID,
			CreateUserName:      createUserName,
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
				ProjectID:      regionItem.ProjectID,
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
		for _, region := range normalizedRegions {
			if err := treeUtils.ValidateCloudRegion(account.Provider, region.Region); err != nil {
				return err
			}
		}
	}

	updateAccount := &model.CloudAccount{
		Model: model.Model{ID: req.ID},
	}
	updateFields := make([]string, 0)
	oldType := account.CredentialType
	if oldType == "" {
		oldType = "long_term"
	}
	newType := req.CredentialType
	if newType == "" {
		newType = oldType
	}
	isStructured := account.Provider == model.ProviderAzure || account.Provider == model.ProviderGCP
	if isStructured {
		if req.AccessKey != "" || req.SecretKey != "" || req.SessionToken != "" || req.CredentialExpiresAt != nil || (req.CredentialType != "" && req.CredentialType != "long_term") {
			return errors.New("该云厂商必须完整替换专用凭据")
		}
		if req.ProviderCredentials != nil {
			encrypted, err := treeUtils.EncryptProviderCredentials(account.Provider, req.ProviderCredentials)
			if err != nil {
				return err
			}
			version := account.CredentialVersion
			updateAccount.ExpectedCredentialVersion = &version
			updateAccount.ProviderCredentials = encrypted
			updateAccount.CredentialType = "long_term"
			updateAccount.SessionToken = ""
			updateAccount.AccessKey = ""
			updateAccount.SecretKey = ""
			updateAccount.CredentialExpiresAt = nil
			updateFields = append(updateFields, "provider_credentials", "credential_type", "session_token", "credential_expires_at", "access_key", "secret_key")
		}
	} else if req.ProviderCredentials != nil {
		return errors.New("该云厂商不支持专用凭据")
	}
	credentialChange := !isStructured && (req.AccessKey != "" || req.SecretKey != "" || req.SessionToken != "" || req.CredentialExpiresAt != nil || newType != oldType)
	if credentialChange {
		if oldType == "temporary" || newType == "temporary" || newType != oldType {
			if req.AccessKey == "" || req.SecretKey == "" {
				return errors.New("临时凭据轮换或类型切换必须完整提供访问凭据")
			}
			input := treeUtils.CloudCredentials{AccessKey: req.AccessKey, SecretKey: req.SecretKey, SessionToken: req.SessionToken, ExpiresAt: req.CredentialExpiresAt, CredentialType: newType}
			if err := treeUtils.ValidateCloudCredentials(account.Provider, input, time.Now().UTC()); err != nil {
				return err
			}
		} else {
			if req.SessionToken != "" || req.CredentialExpiresAt != nil || (account.Provider == model.ProviderAWS && strings.HasPrefix(req.AccessKey, "ASIA")) {
				return errors.New("长期凭据不能包含临时令牌或到期时间")
			}
		}
		version := account.CredentialVersion
		updateAccount.ExpectedCredentialVersion = &version
		if oldType == "temporary" || newType == "temporary" || newType != oldType {
			updateAccount.CredentialType = newType
			updateAccount.CredentialExpiresAt = req.CredentialExpiresAt
			if req.SessionToken != "" {
				updateAccount.SessionToken, err = treeUtils.EncryptPassword(req.SessionToken)
				if err != nil || len(updateAccount.SessionToken) > 16000 {
					return errors.New("无法保存云账户令牌")
				}
			}
			updateFields = append(updateFields, "credential_type", "credential_expires_at", "session_token")
		}
	}

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
		if len(encryptedAccessKey) > 500 {
			return errors.New("云账户密钥超出存储限制")
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
		if len(encryptedSecretKey) > 500 {
			return errors.New("云账户密钥超出存储限制")
		}
		updateAccount.SecretKey = encryptedSecretKey
		updateFields = append(updateFields, "secret_key")
	}
	if err := s.dao.UpdateAccountAndRegions(ctx, updateAccount, updateFields, normalizedRegions); err != nil {
		s.logger.Error("更新云账户失败", zap.Int("account_id", req.ID), zap.Error(err))
		return err
	}
	s.logger.Info("成功更新云账户", zap.Int("account_id", req.ID), zap.Strings("updated_fields", updateFields), zap.Bool("regions_updated", normalizedRegions != nil))
	return nil
}

func (s *cloudAccountService) DeleteCloudAccount(ctx context.Context, req *model.DeleteCloudAccountReq) error {
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

	if len(account.CloudResources) > 0 {
		return fmt.Errorf("云账户下还有 %d 个云资源，请先删除相关资源", len(account.CloudResources))
	}

	if err := s.dao.Delete(ctx, req.ID); err != nil {
		s.logger.Error("删除云账户失败", zap.Error(err))
		return err
	}

	return nil
}

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
	if account.Status != model.CloudAccountEnabled {
		return errors.New("云账号已禁用")
	}

	credentials, err := treeUtils.LoadCloudAccountCredentials(account)
	if err != nil {
		return err
	}
	region, err := defaultAccountRegion(account.Regions)
	if err != nil {
		return err
	}
	client, err := cloudClient(account.Provider, credentials, region, s.logger, s.credentialFactory, s.clientFactory)
	if err != nil {
		return err
	}
	operationCtx, cancel := treeUtils.CredentialContext(ctx, credentials)
	defer cancel()
	if err := client.VerifyCredentials(operationCtx); err != nil {
		if operationCtx.Err() != nil {
			return operationCtx.Err()
		}
		s.logger.Warn("云账户凭证验证失败", zap.Int("id", req.ID), zap.String("error_type", fmt.Sprintf("%T", err)))
		return errors.New("云账户凭证验证失败")
	}

	return nil
}

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

	for _, account := range accounts {
		if len(account.CloudResources) > 0 {
			return fmt.Errorf("云账户 %s 下还有 %d 个云资源，无法删除", account.Name, len(account.CloudResources))
		}
	}

	if err := s.dao.BatchDelete(ctx, req.IDs); err != nil {
		s.logger.Error("批量删除云账户失败", zap.Error(err))
		return err
	}

	s.logger.Info("批量删除云账户成功", zap.Ints("ids", req.IDs))
	return nil
}

func (s *cloudAccountService) BatchUpdateCloudAccountStatus(ctx context.Context, req *model.BatchUpdateCloudAccountStatusReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量更新ID列表不能为空")
	}

	accounts, err := s.dao.GetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(accounts) != len(req.IDs) {
		return errors.New("部分云账户不存在")
	}

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

// GetAvailableRegionsForAccount never returns or accepts credentials in a URL.
func (s *cloudAccountService) GetAvailableRegionsForAccount(ctx context.Context, id int) ([]model.AvailableRegion, error) {
	account, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account.Status != model.CloudAccountEnabled {
		return nil, fmt.Errorf("云账号已禁用")
	}
	credentials, err := treeUtils.LoadCloudAccountCredentials(account)
	if err != nil {
		return nil, err
	}
	if account.Provider == model.ProviderHuawei || account.Provider == model.ProviderAzure || account.Provider == model.ProviderGCP {
		var configured []*model.CloudAccountRegion
		for _, region := range account.Regions {
			if region == nil || region.Status != model.CloudAccountRegionEnabled {
				continue
			}
			if account.Provider == model.ProviderHuawei && (region.ProjectID == nil || !treeUtils.ValidHuaweiProjectID(*region.ProjectID)) {
				return nil, errors.New("华为云地域缺少项目ID")
			}
			if err := treeUtils.ValidateCloudRegion(account.Provider, region.Region); err != nil {
				return nil, err
			}
			configured = append(configured, region)
		}
		if len(configured) == 0 {
			return nil, errors.New("云账户没有启用的地域")
		}
		operationCtx, cancel := treeUtils.CredentialContext(ctx, credentials)
		defer cancel()
		all := make([]model.AvailableRegion, 0, len(configured))
		for _, region := range configured {
			client, err := cloudClient(account.Provider, credentials, region, s.logger, s.credentialFactory, s.clientFactory)
			if err != nil {
				return nil, err
			}
			part, err := client.GetAvailableRegions(operationCtx)
			if err != nil {
				if operationCtx.Err() != nil {
					return nil, operationCtx.Err()
				}
				return nil, errors.New("云地域查询失败")
			}
			if len(part) != 1 || part[0].Region != region.Region {
				return nil, errors.New("云地域响应与账户配置不一致")
			}
			all = append(all, part...)
		}
		return all, nil
	}
	region, err := defaultAccountRegion(account.Regions)
	if err != nil {
		return nil, err
	}
	client, err := cloudClient(account.Provider, credentials, region, s.logger, s.credentialFactory, s.clientFactory)
	if err != nil {
		return nil, err
	}
	operationCtx, cancel := treeUtils.CredentialContext(ctx, credentials)
	defer cancel()
	regions, err := client.GetAvailableRegions(operationCtx)
	if err != nil {
		if operationCtx.Err() != nil {
			return nil, operationCtx.Err()
		}
		return nil, errors.New("云地域查询失败")
	}
	return regions, nil
}
