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

	s.logger.Info("同步云资源",
		zap.Int("cloudAccountID", req.CloudAccountID),
		zap.Int8("provider", int8(account.Provider)),
		zap.String("region", account.Region),
		zap.String("syncMode", string(req.SyncMode)))

	// 根据不同的云厂商调用对应的同步逻辑
	switch account.Provider {
	case model.ProviderAliyun:
		return s.syncAliyunResources(ctx, account, accessKey, secretKey, req)
	case model.ProviderTencent:
		return errors.New("腾讯云资源同步功能暂未实现")
	case model.ProviderAWS:
		return errors.New("AWS资源同步功能暂未实现")
	case model.ProviderHuawei:
		return errors.New("华为云资源同步功能暂未实现")
	case model.ProviderAzure:
		return errors.New("Azure资源同步功能暂未实现")
	case model.ProviderGCP:
		return errors.New("GCP资源同步功能暂未实现")
	default:
		return fmt.Errorf("不支持的云厂商: %d", account.Provider)
	}
}

// syncAliyunResources 同步阿里云资源
func (s *treeCloudService) syncAliyunResources(ctx context.Context, account *model.CloudAccount, accessKey, secretKey string, req *model.SyncTreeCloudResourceReq) error {
	// 构建同步配置
	config := &treeUtils.AliyunSyncConfig{
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		Region:         account.Region,
		CloudAccountID: account.ID,
		ResourceType:   req.ResourceType,
		InstanceIDs:    req.InstanceIDs,
		SyncMode:       req.SyncMode,
	}

	// 从阿里云获取资源列表
	resources, err := treeUtils.SyncAliyunResources(ctx, config, s.logger)
	if err != nil {
		return err
	}

	// 根据同步模式处理资源
	if req.SyncMode == model.SyncModeFull {
		// 全量同步：先删除该云账户下的所有ECS资源，再重新创建
		return s.fullSyncResources(ctx, account.ID, resources)
	}

	// 增量同步：更新已存在的资源，创建不存在的资源
	return s.incrementalSyncResources(ctx, account.ID, resources)
}

// fullSyncResources 全量同步资源
func (s *treeCloudService) fullSyncResources(ctx context.Context, cloudAccountID int, resources []*model.TreeCloudResource) error {
	// 获取该云账户下的所有ECS资源
	req := &model.GetTreeCloudResourceListReq{
		ListReq: model.ListReq{
			Page: 1,
			Size: 10000, // 获取所有资源
		},
		CloudAccountID: cloudAccountID,
		ResourceType:   model.ResourceTypeECS,
	}
	existingResources, _, err := s.dao.GetList(ctx, req)
	if err != nil {
		s.logger.Error("获取现有资源失败", zap.Error(err))
		return err
	}

	// 删除不在新资源列表中的资源
	newInstanceIDSet := make(map[string]bool)
	for _, resource := range resources {
		newInstanceIDSet[resource.InstanceID] = true
	}

	for _, existingResource := range existingResources {
		if !newInstanceIDSet[existingResource.InstanceID] {
			if err := s.dao.Delete(ctx, existingResource.ID); err != nil {
				s.logger.Error("删除资源失败", zap.Int("id", existingResource.ID), zap.Error(err))
			}
		}
	}

	// 更新或创建资源
	return s.incrementalSyncResources(ctx, cloudAccountID, resources)
}

// incrementalSyncResources 增量同步资源
func (s *treeCloudService) incrementalSyncResources(ctx context.Context, cloudAccountID int, resources []*model.TreeCloudResource) error {
	for _, resource := range resources {
		// 检查资源是否已存在
		existing, err := s.dao.GetByAccountAndInstanceID(ctx, cloudAccountID, resource.InstanceID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("查询资源失败", zap.String("instanceID", resource.InstanceID), zap.Error(err))
			continue
		}

		if existing != nil {
			// 更新现有资源
			resource.ID = existing.ID
			if err := s.dao.Update(ctx, resource); err != nil {
				s.logger.Error("更新资源失败", zap.Int("id", existing.ID), zap.Error(err))
			}
		} else {
			// 创建新资源
			if err := s.dao.Create(ctx, resource); err != nil {
				s.logger.Error("创建资源失败", zap.String("instanceID", resource.InstanceID), zap.Error(err))
			}
		}
	}

	return nil
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

	// 解密密钥
	accessKey, err := treeUtils.DecryptPassword(account.AccessKey)
	if err != nil {
		s.logger.Error("解密AccessKey失败", zap.Error(err))
		return nil, fmt.Errorf("解密AccessKey失败: %w", err)
	}

	secretKey, err := treeUtils.DecryptPassword(account.SecretKey)
	if err != nil {
		s.logger.Error("解密SecretKey失败", zap.Error(err))
		return nil, fmt.Errorf("解密SecretKey失败: %w", err)
	}

	s.logger.Info("批量导入云资源",
		zap.Int("cloudAccountID", req.CloudAccountID),
		zap.Int8("provider", int8(account.Provider)),
		zap.String("region", account.Region),
		zap.Int("count", len(req.InstanceIDs)))

	// 根据云厂商调用对应的导入逻辑
	switch account.Provider {
	case model.ProviderAliyun:
		return s.batchImportAliyunResources(ctx, account, accessKey, secretKey, req)
	case model.ProviderTencent:
		return nil, errors.New("腾讯云批量导入功能暂未实现")
	case model.ProviderAWS:
		return nil, errors.New("AWS批量导入功能暂未实现")
	case model.ProviderHuawei:
		return nil, errors.New("华为云批量导入功能暂未实现")
	case model.ProviderAzure:
		return nil, errors.New("Azure批量导入功能暂未实现")
	case model.ProviderGCP:
		return nil, errors.New("GCP批量导入功能暂未实现")
	default:
		return nil, fmt.Errorf("不支持的云厂商: %d", account.Provider)
	}
}

// batchImportAliyunResources 批量导入阿里云资源
func (s *treeCloudService) batchImportAliyunResources(ctx context.Context, account *model.CloudAccount, accessKey, secretKey string, req *model.BatchImportCloudResourceReq) ([]int, error) {
	// 构建同步配置，指定要导入的实例ID
	config := &treeUtils.AliyunSyncConfig{
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		Region:         account.Region,
		CloudAccountID: account.ID,
		ResourceType:   model.ResourceTypeECS,
		InstanceIDs:    req.InstanceIDs,
		SyncMode:       model.SyncModeIncremental,
	}

	// 从阿里云获取指定实例的详情
	resources, err := treeUtils.SyncAliyunResources(ctx, config, s.logger)
	if err != nil {
		return nil, err
	}

	// 批量导入资源
	var importedIDs []int
	for _, resource := range resources {
		// 检查资源是否已存在
		existing, err := s.dao.GetByAccountAndInstanceID(ctx, account.ID, resource.InstanceID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("查询资源失败", zap.String("instanceID", resource.InstanceID), zap.Error(err))
			continue
		}

		if existing != nil {
			// 资源已存在，更新
			resource.ID = existing.ID
			if err := s.dao.Update(ctx, resource); err != nil {
				s.logger.Error("更新资源失败", zap.Int("id", existing.ID), zap.Error(err))
				continue
			}
			importedIDs = append(importedIDs, existing.ID)
		} else {
			// 创建新资源
			if err := s.dao.Create(ctx, resource); err != nil {
				s.logger.Error("创建资源失败", zap.String("instanceID", resource.InstanceID), zap.Error(err))
				continue
			}
			importedIDs = append(importedIDs, resource.ID)
		}
	}

	s.logger.Info("批量导入阿里云资源成功", zap.Int("count", len(importedIDs)))
	return importedIDs, nil
}
