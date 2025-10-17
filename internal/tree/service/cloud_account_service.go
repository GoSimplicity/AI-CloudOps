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
	CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error
	UpdateCloudAccount(ctx context.Context, req *model.UpdateCloudAccountReq) error
	DeleteCloudAccount(ctx context.Context, req *model.DeleteCloudAccountReq) error
	UpdateCloudAccountStatus(ctx context.Context, req *model.UpdateCloudAccountStatusReq) error
	VerifyCloudAccount(ctx context.Context, req *model.VerifyCloudAccountReq) error
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

	accounts, total, err := s.dao.GetList(ctx, req)
	if err != nil {
		s.logger.Error("获取云账户列表失败", zap.Error(err))
		return model.ListResp[*model.CloudAccount]{}, err
	}

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

	account, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云账户不存在")
		}
		s.logger.Error("获取云账户详情失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, err
	}

	return account, nil
}

// CreateCloudAccount 创建云账户
func (s *cloudAccountService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error {
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

	// 创建云账户对象
	account := &model.CloudAccount{
		Name:           req.Name,
		Provider:       req.Provider,
		Region:         req.Region,
		AccessKey:      encryptedAccessKey,
		SecretKey:      encryptedSecretKey,
		AccountID:      req.AccountID,
		AccountName:    req.AccountName,
		AccountAlias:   req.AccountAlias,
		Description:    req.Description,
		Status:         model.CloudAccountEnabled, // 默认启用
		CreateUserID:   req.CreateUserID,
		CreateUserName: req.CreateUserName,
	}

	if err := s.dao.Create(ctx, account); err != nil {
		s.logger.Error("创建云账户失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateCloudAccount 更新云账户
func (s *cloudAccountService) UpdateCloudAccount(ctx context.Context, req *model.UpdateCloudAccountReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账户ID: %w", err)
	}

	// 检查云账户是否存在
	_, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账户不存在")
		}
		return err
	}

	// 构建更新对象
	account := &model.CloudAccount{
		Model:        model.Model{ID: req.ID},
		Name:         req.Name,
		AccountID:    req.AccountID,
		AccountName:  req.AccountName,
		AccountAlias: req.AccountAlias,
		Description:  req.Description,
	}

	// 如果需要更新 AccessKey
	if req.AccessKey != "" {
		encryptedAccessKey, err := treeUtils.EncryptPassword(req.AccessKey)
		if err != nil {
			s.logger.Error("加密AccessKey失败", zap.Error(err))
			return fmt.Errorf("加密AccessKey失败: %w", err)
		}
		account.AccessKey = encryptedAccessKey
	}

	// 如果需要更新 SecretKey
	if req.SecretKey != "" {
		encryptedSecretKey, err := treeUtils.EncryptPassword(req.SecretKey)
		if err != nil {
			s.logger.Error("加密SecretKey失败", zap.Error(err))
			return fmt.Errorf("加密SecretKey失败: %w", err)
		}
		account.SecretKey = encryptedSecretKey
	}

	if err := s.dao.Update(ctx, account); err != nil {
		s.logger.Error("更新云账户失败", zap.Error(err))
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

	// 根据 Provider 调用相应的云厂商 SDK 验证凭证
	verifyReq := &model.VerifyCloudCredentialsReq{
		Provider:  account.Provider,
		Region:    account.Region,
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
		zap.String("region", account.Region))

	return nil
}
