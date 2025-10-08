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

type TreeCloudService interface {
	GetTreeCloudResourceList(ctx context.Context, req *model.GetTreeCloudResourceListReq) (model.ListResp[*model.TreeCloudResource], error)
	GetTreeCloudResourceDetail(ctx context.Context, req *model.GetTreeCloudResourceDetailReq) (*model.TreeCloudResource, error)
	GetTreeCloudResourceForConnection(ctx context.Context, req *model.GetTreeCloudResourceDetailReq) (*model.TreeCloudResource, error)
	CreateTreeCloudResource(ctx context.Context, req *model.CreateTreeCloudResourceReq) error
	UpdateTreeCloudResource(ctx context.Context, req *model.UpdateTreeCloudResourceReq) error
	DeleteTreeCloudResource(ctx context.Context, req *model.DeleteTreeCloudResourceReq) error
	BindTreeCloudResource(ctx context.Context, req *model.BindTreeCloudResourceReq) error
	UnBindTreeCloudResource(ctx context.Context, req *model.UnBindTreeCloudResourceReq) error
	GetTreeNodeCloudResources(ctx context.Context, req *model.GetTreeNodeCloudResourcesReq) ([]*model.TreeCloudResource, error)
	UpdateCloudResourceStatus(ctx context.Context, req *model.UpdateCloudResourceStatusReq) error
	VerifyCloudCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq) error
	SyncTreeCloudResource(ctx context.Context, req *model.SyncTreeCloudResourceReq) error
	BatchImportCloudResource(ctx context.Context, req *model.BatchImportCloudResourceReq) ([]int, error)
}

type treeCloudService struct {
	logger          *zap.Logger
	dao             dao.TreeCloudDAO
	cloudAccountDAO dao.CloudAccountDAO
}

func NewTreeCloudService(logger *zap.Logger, dao dao.TreeCloudDAO, cloudAccountDAO dao.CloudAccountDAO) TreeCloudService {
	return &treeCloudService{
		logger:          logger,
		dao:             dao,
		cloudAccountDAO: cloudAccountDAO,
	}
}

// GetTreeCloudResourceList 获取云资源列表
func (s *treeCloudService) GetTreeCloudResourceList(ctx context.Context, req *model.GetTreeCloudResourceListReq) (model.ListResp[*model.TreeCloudResource], error) {
	// 兜底分页参数，避免offset为负或size为0
	treeUtils.ValidateAndSetPaginationDefaults(&req.Page, &req.Size)

	clouds, total, err := s.dao.GetList(ctx, req)
	if err != nil {
		s.logger.Error("获取云资源列表失败", zap.Error(err))
		return model.ListResp[*model.TreeCloudResource]{}, err
	}

	return model.ListResp[*model.TreeCloudResource]{
		Items: clouds,
		Total: total,
	}, nil
}

// GetTreeCloudResourceDetail 获取云资源详情
func (s *treeCloudService) GetTreeCloudResourceDetail(ctx context.Context, req *model.GetTreeCloudResourceDetailReq) (*model.TreeCloudResource, error) {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return nil, fmt.Errorf("无效的云资源ID: %w", err)
	}

	cloud, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云资源不存在")
		}
		s.logger.Error("获取云资源详情失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, err
	}

	return cloud, nil
}

// GetTreeCloudResourceForConnection 获取用于连接的云资源详情(包含解密后的密码)
func (s *treeCloudService) GetTreeCloudResourceForConnection(ctx context.Context, req *model.GetTreeCloudResourceDetailReq) (*model.TreeCloudResource, error) {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return nil, fmt.Errorf("无效的云资源ID: %w", err)
	}

	cloud, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云资源不存在")
		}
		s.logger.Error("获取云资源详情失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, err
	}

	// 解密SSH密码（针对ECS类型）
	if cloud.AuthMode == model.AuthModePassword && cloud.Password != "" {
		plainPassword, err := treeUtils.DecryptPassword(cloud.Password)
		if err != nil {
			s.logger.Error("SSH密码解密失败", zap.Int("id", req.ID), zap.Error(err))
			return nil, fmt.Errorf("SSH密码解密失败: %w", err)
		}
		cloud.Password = plainPassword
	}

	return cloud, nil
}

// CreateTreeCloudResource 创建云资源
func (s *treeCloudService) CreateTreeCloudResource(ctx context.Context, req *model.CreateTreeCloudResourceReq) error {
	// 检查实例ID是否已存在（如果提供了实例ID）
	if req.InstanceID != "" {
		existing, err := s.dao.GetByAccountAndInstanceID(ctx, req.CloudAccountID, req.InstanceID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("检查实例ID是否存在失败", zap.Error(err))
			return fmt.Errorf("检查实例ID失败: %w", err)
		}
		if existing != nil {
			return fmt.Errorf("云账户 %d 下的实例 %s 已存在", req.CloudAccountID, req.InstanceID)
		}
	}

	// 加密SSH密码
	var encryptedPassword string
	var err error

	if req.Password != "" {
		encryptedPassword, err = treeUtils.EncryptPassword(req.Password)
		if err != nil {
			s.logger.Error("密码加密失败", zap.Error(err))
			return fmt.Errorf("密码加密失败: %w", err)
		}
	}

	// 创建云资源对象
	cloud := &model.TreeCloudResource{
		Name:           req.Name,
		ResourceType:   req.ResourceType,
		Status:         model.CloudResourceRunning,
		Environment:    req.Environment,
		Description:    req.Description,
		Tags:           req.Tags,
		CreateUserID:   req.CreateUserID,
		CreateUserName: req.CreateUserName,
		CloudAccountID: req.CloudAccountID,
		InstanceID:     req.InstanceID,
		InstanceType:   req.InstanceType,
		Cpu:            req.Cpu,
		Memory:         req.Memory,
		Disk:           req.Disk,
		PublicIP:       req.PublicIP,
		PrivateIP:      req.PrivateIP,
		VpcID:          req.VpcID,
		ZoneID:         req.ZoneID,
		ChargeType:     req.ChargeType,
		ExpireTime:     req.ExpireTime,
		MonthlyCost:    req.MonthlyCost,
		Currency:       model.Currency(req.Currency),
		OSType:         req.OSType,
		OSName:         req.OSName,
		ImageID:        req.ImageID,
		ImageName:      req.ImageName,
		Port:           req.Port,
		Username:       req.Username,
		Password:       encryptedPassword,
		Key:            req.Key,
		AuthMode:       req.AuthMode,
	}

	// 设置默认值
	treeUtils.SetSSHDefaults(&cloud.Port, &cloud.Username)

	if err := s.dao.Create(ctx, cloud); err != nil {
		s.logger.Error("创建云资源失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateTreeCloudResource 更新云资源
func (s *treeCloudService) UpdateTreeCloudResource(ctx context.Context, req *model.UpdateTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	// 检查是否存在
	_, err := s.dao.GetByID(ctx, req.ID)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return errors.New("云资源不存在")
	case err != nil:
		s.logger.Error("获取云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	// 加密SSH密码
	if req.Password != "" {
		encrypted, err := treeUtils.EncryptPassword(req.Password)
		if err != nil {
			s.logger.Error("密码加密失败", zap.Error(err))
			return fmt.Errorf("密码加密失败: %w", err)
		}
		req.Password = encrypted
	}

	// 构建更新对象
	cloud := &model.TreeCloudResource{
		Model:          model.Model{ID: req.ID},
		Name:           req.Name,
		Environment:    req.Environment,
		Description:    req.Description,
		Tags:           req.Tags,
		ResourceType:   req.ResourceType,
		CloudAccountID: req.CloudAccountID,
		InstanceType:   req.InstanceType,
		PublicIP:       req.PublicIP,
		PrivateIP:      req.PrivateIP,
		ChargeType:     req.ChargeType,
		ExpireTime:     req.ExpireTime,
		MonthlyCost:    req.MonthlyCost,
		Currency:       model.Currency(req.Currency),
		Port:           req.Port,
		Username:       req.Username,
		Password:       req.Password,
		Key:            req.Key,
		AuthMode:       req.AuthMode,
	}

	// 直接更新
	if err := s.dao.Update(ctx, cloud); err != nil {
		s.logger.Error("更新云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteTreeCloudResource 删除云资源
func (s *treeCloudService) DeleteTreeCloudResource(ctx context.Context, req *model.DeleteTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	if err := s.dao.Delete(ctx, req.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云资源不存在")
		}
		s.logger.Error("删除云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// BindTreeCloudResource 绑定云资源到树节点
func (s *treeCloudService) BindTreeCloudResource(ctx context.Context, req *model.BindTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	if err := s.dao.BindTreeNodes(ctx, req.ID, req.TreeNodeIDs); err != nil {
		s.logger.Error("绑定云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// UnBindTreeCloudResource 解绑云资源与树节点
func (s *treeCloudService) UnBindTreeCloudResource(ctx context.Context, req *model.UnBindTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	if err := s.dao.UnBindTreeNodes(ctx, req.ID, req.TreeNodeIDs); err != nil {
		s.logger.Error("解绑云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// GetTreeNodeCloudResources 获取树节点下的云资源
func (s *treeCloudService) GetTreeNodeCloudResources(ctx context.Context, req *model.GetTreeNodeCloudResourcesReq) ([]*model.TreeCloudResource, error) {
	if err := treeUtils.ValidateID(req.NodeID); err != nil {
		return nil, fmt.Errorf("无效的节点ID: %w", err)
	}

	clouds, err := s.dao.GetByNodeID(ctx, req.NodeID, req)
	if err != nil {
		s.logger.Error("获取树节点云资源失败", zap.Int("nodeID", req.NodeID), zap.Error(err))
		return nil, err
	}

	return clouds, nil
}

// UpdateCloudResourceStatus 更新云资源状态
func (s *treeCloudService) UpdateCloudResourceStatus(ctx context.Context, req *model.UpdateCloudResourceStatusReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	// 检查云资源是否存在
	_, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云资源不存在")
		}
		s.logger.Error("获取云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	if err := s.dao.UpdateStatus(ctx, req.ID, req.Status); err != nil {
		s.logger.Error("更新云资源状态失败", zap.Int("id", req.ID), zap.Int8("status", int8(req.Status)), zap.Error(err))
		return err
	}

	return nil
}

// VerifyCloudCredentials 验证云厂商凭证
func (s *treeCloudService) VerifyCloudCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq) error {
	// TODO: 实现具体的云厂商SDK验证逻辑
	// 这里需要根据不同的云厂商（阿里云、腾讯云、AWS等）调用对应的SDK验证凭证
	s.logger.Info("验证云厂商凭证",
		zap.Int8("provider", int8(req.Provider)),
		zap.String("region", req.Region))

	// 根据云厂商类型验证凭证
	switch req.Provider {
	case model.ProviderAliyun:
		return treeUtils.VerifyAliyunCredentials(ctx, req, s.logger)
	case model.ProviderTencent:
		return treeUtils.VerifyTencentCredentials(ctx, req, s.logger)
	case model.ProviderAWS:
		return treeUtils.VerifyAWSCredentials(ctx, req, s.logger)
	case model.ProviderHuawei:
		return treeUtils.VerifyHuaweiCredentials(ctx, req, s.logger)
	case model.ProviderAzure:
		return treeUtils.VerifyAzureCredentials(ctx, req, s.logger)
	case model.ProviderGCP:
		return treeUtils.VerifyGCPCredentials(ctx, req, s.logger)
	default:
		return fmt.Errorf("不支持的云厂商: %d", req.Provider)
	}
}

// SyncTreeCloudResource 从云厂商同步资源
func (s *treeCloudService) SyncTreeCloudResource(ctx context.Context, req *model.SyncTreeCloudResourceReq) error {
	// 获取云账户信息
	account, err := s.cloudAccountDAO.GetByID(ctx, req.CloudAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账户不存在")
		}
		s.logger.Error("获取云账户失败", zap.Int("cloudAccountID", req.CloudAccountID), zap.Error(err))
		return err
	}

	// 检查云账户状态
	if account.Status != model.CloudAccountEnabled {
		return errors.New("云账户已禁用，无法同步资源")
	}

	s.logger.Info("同步云资源",
		zap.Int("cloudAccountID", req.CloudAccountID),
		zap.Int8("provider", int8(account.Provider)),
		zap.String("region", account.Region),
		zap.String("syncMode", string(req.SyncMode)))

	// TODO: 根据不同的云厂商调用对应的同步逻辑
	// 这里需要实现具体的云厂商SDK调用
	switch account.Provider {
	case model.ProviderAliyun:
		return errors.New("阿里云资源同步功能待实现")
	case model.ProviderTencent:
		return errors.New("腾讯云资源同步功能待实现")
	case model.ProviderAWS:
		return errors.New("AWS资源同步功能待实现")
	case model.ProviderHuawei:
		return errors.New("华为云资源同步功能待实现")
	case model.ProviderAzure:
		return errors.New("Azure资源同步功能待实现")
	case model.ProviderGCP:
		return errors.New("GCP资源同步功能待实现")
	default:
		return fmt.Errorf("不支持的云厂商: %d", account.Provider)
	}
}

// BatchImportCloudResource 批量导入云资源
func (s *treeCloudService) BatchImportCloudResource(ctx context.Context, req *model.BatchImportCloudResourceReq) ([]int, error) {
	if len(req.InstanceIDs) == 0 {
		return nil, errors.New("实例ID列表不能为空")
	}

	// 获取云账户信息
	account, err := s.cloudAccountDAO.GetByID(ctx, req.CloudAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云账户不存在")
		}
		s.logger.Error("获取云账户失败", zap.Int("cloudAccountID", req.CloudAccountID), zap.Error(err))
		return nil, err
	}

	// 检查云账户状态
	if account.Status != model.CloudAccountEnabled {
		return nil, errors.New("云账户已禁用，无法导入资源")
	}

	s.logger.Info("批量导入云资源",
		zap.Int("cloudAccountID", req.CloudAccountID),
		zap.Int8("provider", int8(account.Provider)),
		zap.String("region", account.Region),
		zap.Int("count", len(req.InstanceIDs)))

	// TODO: 实现批量导入逻辑
	// 1. 根据云厂商调用对应的SDK获取实例详情
	// 2. 批量创建云资源记录
	// 3. 返回创建的资源ID列表

	// 暂时返回空列表
	return []int{}, errors.New("批量导入云资源功能待实现")
}
