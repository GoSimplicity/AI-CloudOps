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

type CloudAccountRegionService interface {
	GetCloudAccountRegionList(ctx context.Context, req *model.GetCloudAccountRegionListReq) (model.ListResp[*model.CloudAccountRegion], error)
	GetCloudAccountRegionDetail(ctx context.Context, id int) (*model.CloudAccountRegion, error)
	CreateCloudAccountRegion(ctx context.Context, req *model.CreateCloudAccountRegionReq) error
	BatchCreateCloudAccountRegion(ctx context.Context, req *model.BatchCreateCloudAccountRegionReq) error
	UpdateCloudAccountRegion(ctx context.Context, req *model.UpdateCloudAccountRegionReq) error
	DeleteCloudAccountRegion(ctx context.Context, req *model.DeleteCloudAccountRegionReq) error
	UpdateCloudAccountRegionStatus(ctx context.Context, req *model.UpdateCloudAccountRegionStatusReq) error
	GetAvailableRegions(ctx context.Context, req *model.GetAvailableRegionsReq) (*model.GetAvailableRegionsResp, error)
	GetRegionsByCloudAccountID(ctx context.Context, cloudAccountID int) ([]*model.CloudAccountRegion, error)
}

type cloudAccountRegionService struct {
	logger              *zap.Logger
	dao                 dao.CloudAccountRegionDAO
	cloudAccountService CloudAccountService
}

func NewCloudAccountRegionService(logger *zap.Logger, dao dao.CloudAccountRegionDAO, cloudAccountService CloudAccountService) CloudAccountRegionService {
	return &cloudAccountRegionService{
		logger:              logger,
		dao:                 dao,
		cloudAccountService: cloudAccountService,
	}
}

// GetCloudAccountRegionList 获取云账号区域列表
func (s *cloudAccountRegionService) GetCloudAccountRegionList(ctx context.Context, req *model.GetCloudAccountRegionListReq) (model.ListResp[*model.CloudAccountRegion], error) {
	// 兜底分页参数
	treeUtils.ValidateAndSetPaginationDefaults(&req.Page, &req.Size)

	regions, total, err := s.dao.GetList(ctx, req)
	if err != nil {
		s.logger.Error("获取云账号区域列表失败", zap.Error(err))
		return model.ListResp[*model.CloudAccountRegion]{}, err
	}

	return model.ListResp[*model.CloudAccountRegion]{
		Items: regions,
		Total: total,
	}, nil
}

// GetCloudAccountRegionDetail 获取云账号区域详情
func (s *cloudAccountRegionService) GetCloudAccountRegionDetail(ctx context.Context, id int) (*model.CloudAccountRegion, error) {
	if err := treeUtils.ValidateID(id); err != nil {
		return nil, fmt.Errorf("无效的云账号区域ID: %w", err)
	}

	region, err := s.dao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云账号区域不存在")
		}
		s.logger.Error("获取云账号区域详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return region, nil
}

// CreateCloudAccountRegion 创建云账号区域关联
func (s *cloudAccountRegionService) CreateCloudAccountRegion(ctx context.Context, req *model.CreateCloudAccountRegionReq) error {
	// 验证云账号是否存在
	_, err := s.cloudAccountService.GetCloudAccountDetail(ctx, &model.GetCloudAccountDetailReq{ID: req.CloudAccountID})
	if err != nil {
		return fmt.Errorf("云账号不存在: %w", err)
	}

	// 检查同一账号下区域是否已存在
	existing, err := s.dao.GetByCloudAccountAndRegion(ctx, req.CloudAccountID, req.Region)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("检查区域是否存在失败", zap.Error(err))
		return err
	}
	if existing != nil {
		return fmt.Errorf("云账号 %d 已配置区域 %s", req.CloudAccountID, req.Region)
	}

	// 如果设置为默认区域，需要先取消其他区域的默认状态
	if req.IsDefault {
		if err := s.dao.ClearDefaultRegion(ctx, req.CloudAccountID); err != nil {
			s.logger.Error("清除默认区域失败", zap.Error(err))
			return fmt.Errorf("清除默认区域失败: %w", err)
		}
	}

	// 创建云账号区域关联
	region := &model.CloudAccountRegion{
		CloudAccountID: req.CloudAccountID,
		Region:         req.Region,
		RegionName:     req.RegionName,
		IsDefault:      req.IsDefault,
		Description:    req.Description,
		Status:         model.CloudAccountRegionEnabled, // 默认启用
		CreateUserID:   req.CreateUserID,
		CreateUserName: req.CreateUserName,
	}

	if err := s.dao.Create(ctx, region); err != nil {
		s.logger.Error("创建云账号区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchCreateCloudAccountRegion 批量创建云账号区域关联
func (s *cloudAccountRegionService) BatchCreateCloudAccountRegion(ctx context.Context, req *model.BatchCreateCloudAccountRegionReq) error {
	// 验证云账号是否存在
	_, err := s.cloudAccountService.GetCloudAccountDetail(ctx, &model.GetCloudAccountDetailReq{ID: req.CloudAccountID})
	if err != nil {
		return fmt.Errorf("云账号不存在: %w", err)
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

	// 确保只有一个默认区域
	if defaultCount > 1 {
		return errors.New("只能设置一个默认区域")
	}

	// 检查区域是否已存在
	for _, regionItem := range req.Regions {
		existing, err := s.dao.GetByCloudAccountAndRegion(ctx, req.CloudAccountID, regionItem.Region)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("检查区域是否存在失败", zap.Error(err))
			return err
		}
		if existing != nil {
			return fmt.Errorf("云账号 %d 已配置区域 %s", req.CloudAccountID, regionItem.Region)
		}
	}

	// 如果有默认区域，需要先清除其他区域的默认状态
	if defaultCount > 0 {
		if err := s.dao.ClearDefaultRegion(ctx, req.CloudAccountID); err != nil {
			s.logger.Error("清除默认区域失败", zap.Error(err))
			return fmt.Errorf("清除默认区域失败: %w", err)
		}
	}

	// 批量创建区域关联
	var regions []*model.CloudAccountRegion
	for _, regionItem := range req.Regions {
		region := &model.CloudAccountRegion{
			CloudAccountID: req.CloudAccountID,
			Region:         regionItem.Region,
			RegionName:     regionItem.RegionName,
			IsDefault:      regionItem.IsDefault,
			Description:    regionItem.Description,
			Status:         model.CloudAccountRegionEnabled, // 默认启用
			CreateUserID:   req.CreateUserID,
			CreateUserName: req.CreateUserName,
		}
		regions = append(regions, region)
	}

	if err := s.dao.BatchCreate(ctx, regions); err != nil {
		s.logger.Error("批量创建云账号区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateCloudAccountRegion 更新云账号区域关联
func (s *cloudAccountRegionService) UpdateCloudAccountRegion(ctx context.Context, req *model.UpdateCloudAccountRegionReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账号区域ID: %w", err)
	}

	// 检查区域是否存在
	existing, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账号区域不存在")
		}
		return err
	}

	// 如果设置为默认区域，需要先取消其他区域的默认状态
	if req.IsDefault {
		if err := s.dao.ClearDefaultRegion(ctx, existing.CloudAccountID); err != nil {
			s.logger.Error("清除默认区域失败", zap.Error(err))
			return fmt.Errorf("清除默认区域失败: %w", err)
		}
	}

	// 构建更新对象
	region := &model.CloudAccountRegion{
		Model:       model.Model{ID: req.ID},
		RegionName:  req.RegionName,
		IsDefault:   req.IsDefault,
		Description: req.Description,
	}

	if err := s.dao.Update(ctx, region); err != nil {
		s.logger.Error("更新云账号区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteCloudAccountRegion 删除云账号区域关联
func (s *cloudAccountRegionService) DeleteCloudAccountRegion(ctx context.Context, req *model.DeleteCloudAccountRegionReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账号区域ID: %w", err)
	}

	// 检查区域是否存在
	region, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云账号区域不存在")
		}
		return err
	}

	// 检查是否有关联的云资源
	resourceCount, err := s.dao.GetResourceCountByRegion(ctx, req.ID)
	if err != nil {
		s.logger.Error("检查关联资源失败", zap.Error(err))
		return err
	}

	if resourceCount > 0 {
		return fmt.Errorf("该区域下还有 %d 个云资源，请先删除相关资源", resourceCount)
	}

	if err := s.dao.Delete(ctx, req.ID); err != nil {
		s.logger.Error("删除云账号区域关联失败", zap.Error(err))
		return err
	}

	// 如果删除的是默认区域，需要设置一个新的默认区域
	if region.IsDefault {
		regions, err := s.dao.GetByCloudAccountID(ctx, region.CloudAccountID)
		if err != nil {
			s.logger.Error("获取云账号区域列表失败", zap.Error(err))
			return err
		}

		// 如果还有其他区域，设置第一个为默认区域
		if len(regions) > 0 {
			newDefault := regions[0]
			newDefault.IsDefault = true
			if err := s.dao.Update(ctx, newDefault); err != nil {
				s.logger.Error("设置新的默认区域失败", zap.Error(err))
				return err
			}
		}
	}

	return nil
}

// UpdateCloudAccountRegionStatus 更新云账号区域状态
func (s *cloudAccountRegionService) UpdateCloudAccountRegionStatus(ctx context.Context, req *model.UpdateCloudAccountRegionStatusReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云账号区域ID: %w", err)
	}

	if err := s.dao.UpdateStatus(ctx, req.ID, req.Status); err != nil {
		s.logger.Error("更新云账号区域状态失败", zap.Error(err))
		return err
	}

	return nil
}

// GetAvailableRegions 获取指定云厂商的可用区域列表
func (s *cloudAccountRegionService) GetAvailableRegions(ctx context.Context, req *model.GetAvailableRegionsReq) (*model.GetAvailableRegionsResp, error) {
	// 如果请求中包含了凭证信息，尝试通过API动态获取
	if req.AccessKey != "" && req.SecretKey != "" {
		regions, err := treeUtils.GetAvailableRegionsByProvider(ctx, req.Provider, req.AccessKey, req.SecretKey, s.logger)
		if err != nil {
			s.logger.Warn("通过API获取区域列表失败，返回默认区域列表",
				zap.String("provider", req.Provider.String()),
				zap.Error(err))
			// API调用失败时返回默认区域列表
			regions = treeUtils.GetAvailableRegionsByProviderWithoutCredentials(req.Provider)
		}

		return &model.GetAvailableRegionsResp{
			Regions: regions,
		}, nil
	}

	// 没有凭证信息时，返回默认区域列表
	regions := treeUtils.GetAvailableRegionsByProviderWithoutCredentials(req.Provider)

	return &model.GetAvailableRegionsResp{
		Regions: regions,
	}, nil
}

// GetRegionsByCloudAccountID 根据云账号ID获取区域列表（内部使用）
func (s *cloudAccountRegionService) GetRegionsByCloudAccountID(ctx context.Context, cloudAccountID int) ([]*model.CloudAccountRegion, error) {
	return s.dao.GetByCloudAccountID(ctx, cloudAccountID)
}
