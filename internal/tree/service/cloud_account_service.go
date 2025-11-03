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

// CreateCloudAccount 创建云账户（支持多区域）
func (s *cloudAccountService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq, createUserID int, createUserName string) error {
	// 验证区域列表
	if len(req.Regions) == 0 {
		return errors.New("必须至少指定一个区域")
	}

	// 检查账户名称是否已存在（同一云厂商下）
	exists, err := s.dao.CheckNameExists(ctx, req.Name, req.Provider, 0)
	if err != nil {
		s.logger.Error("检查云账户名称是否存在失败", zap.Error(err))
		return err
	}
	if exists {
		return fmt.Errorf("云账户名称 %s 在 %s 下已存在", req.Name, treeUtils.GetProviderName(req.Provider))
	}

	// 检查是否有重复的区域
	regionMap := make(map[string]bool)
	var defaultCount int
	for _, regionItem := range req.Regions {
		if regionMap[regionItem.Region] {
			return fmt.Errorf("区域 %s 重复", regionItem.Region)
		}
		regionMap[regionItem.Region] = true

		if regionItem.IsDefault {
			defaultCount++
		}
	}

	// 确保只有一个默认区域，如果没有指定默认区域，则设置第一个为默认
	if defaultCount == 0 {
		req.Regions[0].IsDefault = true
	} else if defaultCount > 1 {
		return errors.New("只能设置一个默认区域")
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
	return s.dao.CreateWithTransaction(ctx, func(tx interface{}) error {
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
			s.logger.Error("创建云账户失败", zap.Error(err))
			return err
		}

		// 创建区域关联
		for _, regionItem := range req.Regions {
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
				s.logger.Error("创建云账户区域关联失败", zap.Error(err))
				return err
			}
		}

		return nil
	})
}

// UpdateCloudAccount 更新云账户
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
		return err
	}

	// 如果修改了名称，检查新名称是否已存在（同一云厂商下）
	if req.Name != "" && req.Name != account.Name {
		exists, err := s.dao.CheckNameExists(ctx, req.Name, account.Provider, req.ID)
		if err != nil {
			s.logger.Error("检查云账户名称是否存在失败", zap.Error(err))
			return err
		}
		if exists {
			return fmt.Errorf("云账户名称 %s 在 %s 下已存在", req.Name, treeUtils.GetProviderName(account.Provider))
		}
	}

	// 构建更新对象
	updateAccount := &model.CloudAccount{
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
		updateAccount.AccessKey = encryptedAccessKey
	}

	// 如果需要更新 SecretKey
	if req.SecretKey != "" {
		encryptedSecretKey, err := treeUtils.EncryptPassword(req.SecretKey)
		if err != nil {
			s.logger.Error("加密SecretKey失败", zap.Error(err))
			return fmt.Errorf("加密SecretKey失败: %w", err)
		}
		updateAccount.SecretKey = encryptedSecretKey
	}

	if err := s.dao.Update(ctx, updateAccount); err != nil {
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

	// 获取默认区域用于验证凭证
	defaultRegion := "cn-hangzhou" // 默认区域
	if len(account.Regions) > 0 {
		for _, region := range account.Regions {
			if region.IsDefault {
				defaultRegion = region.Region
				break
			}
		}
		// 如果没有找到默认区域，使用第一个区域
		if defaultRegion == "cn-hangzhou" {
			defaultRegion = account.Regions[0].Region
		}
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
		format = "json" // 默认导出为 JSON
	}

	switch format {
	case "json":
		return s.exportAsJSON(accounts), nil
	case "csv":
		return s.exportAsCSV(accounts), nil
	case "excel":
		// TODO: 实现 Excel 导出
		return nil, errors.New("Excel 导出功能暂未实现")
	default:
		return nil, fmt.Errorf("不支持的导出格式: %s", format)
	}
}

// exportAsJSON 导出为 JSON 格式
func (s *cloudAccountService) exportAsJSON(accounts []*model.CloudAccount) interface{} {
	// 为了安全，不导出敏感信息（AccessKey、SecretKey）
	exportAccounts := make([]model.ExportAccount, 0, len(accounts))
	for _, account := range accounts {
		regions := make([]model.ExportRegion, 0, len(account.Regions))
		for _, region := range account.Regions {
			regions = append(regions, model.ExportRegion{
				Region:      region.Region,
				RegionName:  region.RegionName,
				IsDefault:   region.IsDefault,
				Description: region.Description,
			})
		}

		exportAccounts = append(exportAccounts, model.ExportAccount{
			ID:           account.ID,
			Name:         account.Name,
			Provider:     account.Provider,
			ProviderName: treeUtils.GetProviderName(account.Provider),
			AccountID:    account.AccountID,
			AccountName:  account.AccountName,
			AccountAlias: account.AccountAlias,
			Description:  account.Description,
			Status:       int8(account.Status),
			Regions:      regions,
			CreatedAt:    account.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return exportAccounts
}

// exportAsCSV 导出为 CSV 格式
func (s *cloudAccountService) exportAsCSV(accounts []*model.CloudAccount) [][]string {
	// CSV 表头
	csvData := [][]string{
		{"ID", "名称", "云厂商", "账号ID", "账号名称", "账号别名", "描述", "状态", "区域列表", "创建时间"},
	}

	// 数据行
	for _, account := range accounts {
		// 组装区域列表
		regions := ""
		for i, region := range account.Regions {
			if i > 0 {
				regions += ";"
			}
			regions += fmt.Sprintf("%s(%s)", region.Region, region.RegionName)
		}

		status := "禁用"
		if account.Status == model.CloudAccountEnabled {
			status = "启用"
		}

		csvData = append(csvData, []string{
			fmt.Sprintf("%d", account.ID),
			account.Name,
			treeUtils.GetProviderName(account.Provider),
			account.AccountID,
			account.AccountName,
			account.AccountAlias,
			account.Description,
			status,
			regions,
			account.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return csvData
}
