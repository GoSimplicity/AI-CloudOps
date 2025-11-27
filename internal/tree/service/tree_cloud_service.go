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
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	GetTreeNodeCloudResources(ctx context.Context, req *model.GetTreeNodeCloudResourcesReq) ([]*model.TreeCloudResource, error)
	SyncTreeCloudResource(ctx context.Context, req *model.SyncTreeCloudResourceReq) (*model.SyncCloudResourceResp, error)
	GetSyncHistory(ctx context.Context, req *model.GetCloudResourceSyncHistoryReq) (model.ListResp[*model.CloudResourceSyncHistory], error)
	UpdateTreeCloudResource(ctx context.Context, req *model.UpdateTreeCloudResourceReq) error
	DeleteTreeCloudResource(ctx context.Context, req *model.DeleteTreeCloudResourceReq) error
	BatchDeleteTreeCloudResource(ctx context.Context, req *model.BatchDeleteTreeCloudResourceReq) error
	UpdateCloudResourceStatus(ctx context.Context, req *model.UpdateCloudResourceStatusReq) error
	BatchUpdateCloudResourceStatus(ctx context.Context, req *model.BatchUpdateCloudResourceStatusReq) error
	BindTreeCloudResource(ctx context.Context, req *model.BindTreeCloudResourceReq) error
	UnBindTreeCloudResource(ctx context.Context, req *model.UnBindTreeCloudResourceReq) error
	GetChangeLog(ctx context.Context, req *model.GetCloudResourceChangeLogReq) (model.ListResp[*model.CloudResourceChangeLog], error)
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

	// 解密SSH密码
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

// UpdateTreeCloudResource 更新云资源本地元数据
func (s *treeCloudService) UpdateTreeCloudResource(ctx context.Context, req *model.UpdateTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	// 检查资源是否存在
	_, err := s.dao.GetByID(ctx, req.ID)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return errors.New("云资源不存在")
	case err != nil:
		s.logger.Error("获取云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	// 构建要更新的字段map
	metadata := make(map[string]interface{})

	// 只添加非空字段
	if req.Environment != "" {
		metadata["environment"] = req.Environment
	}
	if req.Description != "" {
		metadata["description"] = req.Description
	}
	if req.Tags != nil {
		metadata["tags"] = req.Tags
	}
	if req.Port > 0 {
		metadata["port"] = req.Port
	}
	if req.Username != "" {
		metadata["username"] = req.Username
	}
	if req.Password != "" {
		// 加密SSH密码
		encrypted, err := treeUtils.EncryptPassword(req.Password)
		if err != nil {
			s.logger.Error("密码加密失败", zap.Error(err))
			return fmt.Errorf("密码加密失败: %w", err)
		}
		metadata["password"] = encrypted
	}
	if req.Key != "" {
		metadata["key"] = req.Key
	}
	if req.AuthMode > 0 {
		metadata["auth_mode"] = req.AuthMode
	}

	// 如果没有字段需要更新
	if len(metadata) == 0 {
		s.logger.Info("没有字段需要更新", zap.Int("id", req.ID))
		return nil
	}

	// 更新元数据
	if err := s.dao.UpdateMetadata(ctx, req.ID, metadata); err != nil {
		s.logger.Error("更新云资源元数据失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	// 记录变更日志
	// 获取资源实例ID用于日志
	resource, _ := s.dao.GetByID(ctx, req.ID)
	instanceID := ""
	if resource != nil {
		instanceID = resource.InstanceID
	}

	// 为每个更新的字段创建变更日志
	for fieldName, newValue := range metadata {
		changeLog := &model.CloudResourceChangeLog{
			ResourceID:   req.ID,
			InstanceID:   instanceID,
			ChangeType:   model.ChangeTypeUpdated,
			FieldName:    fieldName,
			OldValue:     "",
			NewValue:     fmt.Sprintf("%v", newValue),
			ChangeSource: model.ChangeSourceManual,
			OperatorID:   req.OperatorID,
			OperatorName: req.OperatorName,
			ChangeTime:   time.Now(),
		}
		// 异步记录，不影响主流程
		go func(log *model.CloudResourceChangeLog) {
			if err := s.dao.CreateChangeLog(context.Background(), log); err != nil {
				s.logger.Error("记录变更日志失败", zap.Error(err))
			}
		}(changeLog)
	}

	s.logger.Info("更新云资源元数据成功", zap.Int("id", req.ID), zap.Int("fields", len(metadata)))
	return nil
}

// DeleteTreeCloudResource 删除云资源
func (s *treeCloudService) DeleteTreeCloudResource(ctx context.Context, req *model.DeleteTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	// 获取资源信息用于日志记录
	cloud, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云资源不存在")
		}
		s.logger.Error("获取云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	// 记录删除日志
	s.recordChangeLog(ctx, cloud, nil, model.ChangeSourceManual, req.OperatorID, req.OperatorName)

	if err := s.dao.Delete(ctx, req.ID); err != nil {
		s.logger.Error("删除云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	s.logger.Info("从平台删除云资源成功",
		zap.Int("id", req.ID),
		zap.String("instanceID", cloud.InstanceID),
		zap.String("name", cloud.Name))
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
func (s *treeCloudService) SyncTreeCloudResource(ctx context.Context, req *model.SyncTreeCloudResourceReq) (*model.SyncCloudResourceResp, error) {
	startTime := time.Now()

	// 设置默认的同步模式
	if req.SyncMode == "" {
		req.SyncMode = model.SyncModeIncremental
	}

	// 初始化同步响应
	resp := &model.SyncCloudResourceResp{
		SyncTime:        startTime,
		FailedInstances: []string{},
	}

	// 创建同步历史记录
	syncHistory := &model.CloudResourceSyncHistory{
		CloudAccountID: req.CloudAccountID,
		SyncMode:       req.SyncMode,
		StartTime:      startTime,
		SyncStatus:     "running",
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
		return nil, errors.New("云账户已禁用，无法同步资源")
	}

	// 解密密钥
	accessKey, err := treeUtils.DecryptPassword(account.AccessKey)
	if err != nil {
		s.logger.Error("解密AccessKey失败", zap.Error(err))
		syncHistory.SyncStatus = "failed"
		syncHistory.ErrorMessage = fmt.Sprintf("解密AccessKey失败: %v", err)
		s.saveSyncHistory(ctx, syncHistory)
		return nil, fmt.Errorf("解密AccessKey失败: %w", err)
	}

	secretKey, err := treeUtils.DecryptPassword(account.SecretKey)
	if err != nil {
		s.logger.Error("解密SecretKey失败", zap.Error(err))
		syncHistory.SyncStatus = "failed"
		syncHistory.ErrorMessage = fmt.Sprintf("解密SecretKey失败: %v", err)
		s.saveSyncHistory(ctx, syncHistory)
		return nil, fmt.Errorf("解密SecretKey失败: %w", err)
	}

	// 获取要同步的区域列表
	var regionsToSync []*model.CloudAccountRegion
	if len(req.CloudAccountRegionIDs) > 0 {
		// 同步指定的区域
		for _, regionID := range req.CloudAccountRegionIDs {
			for _, region := range account.Regions {
				if region.ID == regionID {
					regionsToSync = append(regionsToSync, region)
					break
				}
			}
		}
	} else {
		// 同步账号下的所有启用区域
		for _, region := range account.Regions {
			if region.Status == model.CloudAccountRegionEnabled {
				regionsToSync = append(regionsToSync, region)
			}
		}
	}

	if len(regionsToSync) == 0 {
		return nil, errors.New("没有可用的区域进行同步")
	}

	s.logger.Info("开始同步云资源",
		zap.Int("cloudAccountID", req.CloudAccountID),
		zap.Int8("provider", int8(account.Provider)),
		zap.Int("regionCount", len(regionsToSync)),
		zap.String("syncMode", string(req.SyncMode)))

	// 根据不同的云厂商调用对应的同步逻辑，遍历所有区域
	var syncErr error
	switch account.Provider {
	case model.ProviderAliyun:
		syncErr = s.syncAliyunResourcesForMultipleRegions(ctx, account, accessKey, secretKey, regionsToSync, req, resp)
	case model.ProviderTencent:
		syncErr = errors.New("腾讯云资源同步功能暂未实现")
	case model.ProviderAWS:
		syncErr = errors.New("AWS资源同步功能暂未实现")
	case model.ProviderHuawei:
		syncErr = errors.New("华为云资源同步功能暂未实现")
	case model.ProviderAzure:
		syncErr = errors.New("Azure资源同步功能暂未实现")
	case model.ProviderGCP:
		syncErr = errors.New("GCP资源同步功能暂未实现")
	default:
		syncErr = fmt.Errorf("不支持的云厂商: %d", account.Provider)
	}

	// 更新同步历史记录
	endTime := time.Now()
	syncHistory.EndTime = &endTime
	syncHistory.Duration = int(endTime.Sub(startTime).Seconds())
	syncHistory.TotalCount = resp.TotalCount
	syncHistory.NewCount = resp.NewCount
	syncHistory.UpdateCount = resp.UpdateCount
	syncHistory.DeleteCount = resp.DeleteCount
	syncHistory.FailedCount = resp.FailedCount

	if len(resp.FailedInstances) > 0 {
		// 将失败的实例ID列表转为JSON字符串
		failedJSON, _ := json.Marshal(resp.FailedInstances)
		syncHistory.FailedInstances = string(failedJSON)
	}

	if syncErr != nil {
		syncHistory.SyncStatus = "failed"
		syncHistory.ErrorMessage = syncErr.Error()
		s.saveSyncHistory(ctx, syncHistory)
		return resp, syncErr
	}

	if resp.FailedCount > 0 {
		syncHistory.SyncStatus = "partial"
	} else {
		syncHistory.SyncStatus = "success"
	}

	s.saveSyncHistory(ctx, syncHistory)

	s.logger.Info("云资源同步完成",
		zap.Int("total", resp.TotalCount),
		zap.Int("new", resp.NewCount),
		zap.Int("update", resp.UpdateCount),
		zap.Int("delete", resp.DeleteCount),
		zap.Int("failed", resp.FailedCount),
		zap.Duration("duration", endTime.Sub(startTime)))

	return resp, nil
}

// syncAliyunResourcesWithStats 同步阿里云资源并返回统计信息
// syncAliyunResourcesForMultipleRegions 多区域阿里云资源同步
func (s *treeCloudService) syncAliyunResourcesForMultipleRegions(ctx context.Context, account *model.CloudAccount, accessKey, secretKey string, regions []*model.CloudAccountRegion, req *model.SyncTreeCloudResourceReq, resp *model.SyncCloudResourceResp) error {
	// 遍历每个区域进行同步
	for _, region := range regions {
		s.logger.Info("开始同步区域资源",
			zap.String("region", region.Region),
			zap.String("regionName", region.RegionName))

		// 构建同步配置
		config := &treeUtils.AliyunSyncConfig{
			AccessKey:      accessKey,
			SecretKey:      secretKey,
			Region:         region.Region,
			CloudAccountID: account.ID,
			ResourceType:   0, // 暂时只同步ECS
			InstanceIDs:    req.InstanceIDs,
			SyncMode:       req.SyncMode,
		}

		// 从阿里云获取资源列表
		resources, err := treeUtils.SyncAliyunResources(ctx, config, s.logger)
		if err != nil {
			s.logger.Error("同步区域资源失败",
				zap.String("region", region.Region),
				zap.Error(err))
			// 继续同步其他区域，不直接返回错误
			continue
		}

		// 为资源设置区域关联信息
		for _, resource := range resources {
			resource.CloudAccountRegionID = region.ID
			resource.Region = region.Region // 冗余字段，便于查询
		}

		// 根据同步模式处理资源
		if req.SyncMode == model.SyncModeFull {
			// 全量同步：先删除该区域下的所有ECS资源，再重新创建
			err = s.fullSyncResourcesForRegion(ctx, region.ID, resources, resp, req.AutoBind, req.BindNodeID, req.OperatorID, req.OperatorName)
		} else {
			// 增量同步：更新已存在的资源，创建不存在的资源
			err = s.incrementalSyncResourcesForRegion(ctx, region.ID, resources, resp, req.AutoBind, req.BindNodeID, req.OperatorID, req.OperatorName)
		}

		if err != nil {
			s.logger.Error("处理区域资源失败",
				zap.String("region", region.Region),
				zap.Error(err))
			// 继续同步其他区域
			continue
		}

		s.logger.Info("区域资源同步完成",
			zap.String("region", region.Region),
			zap.Int("resourceCount", len(resources)))
	}

	return nil
}

func (s *treeCloudService) syncAliyunResourcesWithStats(ctx context.Context, account *model.CloudAccount, accessKey, secretKey string, req *model.SyncTreeCloudResourceReq, resp *model.SyncCloudResourceResp) error {
	// 获取默认区域或第一个区域（向后兼容）
	var region *model.CloudAccountRegion
	for _, r := range account.Regions {
		if r.IsDefault {
			region = r
			break
		}
	}
	if region == nil && len(account.Regions) > 0 {
		region = account.Regions[0]
	}
	if region == nil {
		return errors.New("云账户没有配置区域")
	}

	// 构建同步配置
	config := &treeUtils.AliyunSyncConfig{
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		Region:         region.Region,
		CloudAccountID: account.ID,
		ResourceType:   0, // 暂时只同步ECS
		InstanceIDs:    req.InstanceIDs,
		SyncMode:       req.SyncMode,
	}

	// 从阿里云获取资源列表
	resources, err := treeUtils.SyncAliyunResources(ctx, config, s.logger)
	if err != nil {
		return err
	}

	// 为资源设置区域关联信息
	for _, resource := range resources {
		resource.CloudAccountRegionID = region.ID
		resource.Region = region.Region // 冗余字段，便于查询
	}

	// 根据同步模式处理资源
	if req.SyncMode == model.SyncModeFull {
		// 全量同步：先删除该云账户下的所有ECS资源，再重新创建
		return s.fullSyncResources(ctx, account.ID, resources, resp, req.AutoBind, req.BindNodeID, req.OperatorID, req.OperatorName)
	}

	// 增量同步：更新已存在的资源，创建不存在的资源
	return s.incrementalSyncResources(ctx, account.ID, resources, resp, req.AutoBind, req.BindNodeID, req.OperatorID, req.OperatorName)
}

// fullSyncResources 全量同步资源
func (s *treeCloudService) fullSyncResources(ctx context.Context, cloudAccountID int, resources []*model.TreeCloudResource, resp *model.SyncCloudResourceResp, autoBind bool, bindNodeID int, operatorID int, operatorName string) error {
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
				resp.FailedCount++
				resp.FailedInstances = append(resp.FailedInstances, existingResource.InstanceID)
			} else {
				resp.DeleteCount++
				// 记录删除日志
				s.recordChangeLog(ctx, existingResource, nil, model.ChangeSourceSync, operatorID, operatorName)
			}
		}
	}

	// 更新或创建资源
	return s.incrementalSyncResources(ctx, cloudAccountID, resources, resp, autoBind, bindNodeID, operatorID, operatorName)
}

// fullSyncResourcesForRegion 基于区域的全量同步资源
func (s *treeCloudService) fullSyncResourcesForRegion(ctx context.Context, regionID int, resources []*model.TreeCloudResource, resp *model.SyncCloudResourceResp, autoBind bool, bindNodeID int, operatorID int, operatorName string) error {
	// 通过DAO层查询指定区域的资源
	existingResources, err := s.dao.GetResourcesByRegion(ctx, regionID, model.ResourceTypeECS)
	if err != nil {
		s.logger.Error("获取区域现有资源失败", zap.Int("regionID", regionID), zap.Error(err))
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
				resp.FailedCount++
				resp.FailedInstances = append(resp.FailedInstances, existingResource.InstanceID)
			} else {
				resp.DeleteCount++
				// 记录删除日志
				s.recordChangeLog(ctx, existingResource, nil, model.ChangeSourceSync, operatorID, operatorName)
			}
		}
	}

	// 更新或创建资源
	return s.incrementalSyncResourcesForRegion(ctx, regionID, resources, resp, autoBind, bindNodeID, operatorID, operatorName)
}

// incrementalSyncResourcesForRegion 基于区域的增量同步资源
func (s *treeCloudService) incrementalSyncResourcesForRegion(ctx context.Context, regionID int, resources []*model.TreeCloudResource, resp *model.SyncCloudResourceResp, autoBind bool, bindNodeID int, operatorID int, operatorName string) error {
	for _, resource := range resources {
		resp.TotalCount++

		// 检查资源是否已存在（通过区域和实例ID查询）
		existing, err := s.dao.GetByRegionAndInstanceID(ctx, regionID, resource.InstanceID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("查询区域资源失败",
				zap.Int("regionID", regionID),
				zap.String("instanceID", resource.InstanceID),
				zap.Error(err))
			resp.FailedCount++
			resp.FailedInstances = append(resp.FailedInstances, resource.InstanceID)
			continue
		}

		if existing != nil {
			// 更新现有资源
			resource.ID = existing.ID
			if err := s.dao.Update(ctx, resource); err != nil {
				s.logger.Error("更新区域资源失败", zap.Int("id", existing.ID), zap.Error(err))
				resp.FailedCount++
				resp.FailedInstances = append(resp.FailedInstances, resource.InstanceID)
			} else {
				resp.UpdateCount++
				// 记录更新日志
				s.recordChangeLog(ctx, existing, resource, model.ChangeSourceSync, operatorID, operatorName)
			}
		} else {
			// 创建新资源
			if err := s.dao.Create(ctx, resource); err != nil {
				s.logger.Error("创建区域资源失败",
					zap.Int("regionID", regionID),
					zap.String("instanceID", resource.InstanceID),
					zap.Error(err))
				resp.FailedCount++
				resp.FailedInstances = append(resp.FailedInstances, resource.InstanceID)
			} else {
				resp.NewCount++
				// 记录创建日志
				s.recordChangeLog(ctx, nil, resource, model.ChangeSourceSync, operatorID, operatorName)

				// 自动绑定到服务树节点
				if autoBind && bindNodeID > 0 {
					bindReq := &model.BindTreeCloudResourceReq{
						ID:          resource.ID,
						TreeNodeIDs: []int{bindNodeID},
					}
					if err := s.BindTreeCloudResource(ctx, bindReq); err != nil {
						s.logger.Warn("自动绑定资源到节点失败",
							zap.Int("resourceID", resource.ID),
							zap.Int("nodeID", bindNodeID),
							zap.Error(err))
					}
				}
			}
		}
	}

	return nil
}

// incrementalSyncResources 增量同步资源
func (s *treeCloudService) incrementalSyncResources(ctx context.Context, cloudAccountID int, resources []*model.TreeCloudResource, resp *model.SyncCloudResourceResp, autoBind bool, bindNodeID int, operatorID int, operatorName string) error {
	for _, resource := range resources {
		resp.TotalCount++

		// 检查资源是否已存在
		existing, err := s.dao.GetByAccountAndInstanceID(ctx, cloudAccountID, resource.InstanceID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("查询资源失败", zap.String("instanceID", resource.InstanceID), zap.Error(err))
			resp.FailedCount++
			resp.FailedInstances = append(resp.FailedInstances, resource.InstanceID)
			continue
		}

		if existing != nil {
			// 更新现有资源
			resource.ID = existing.ID
			if err := s.dao.Update(ctx, resource); err != nil {
				s.logger.Error("更新资源失败", zap.Int("id", existing.ID), zap.Error(err))
				resp.FailedCount++
				resp.FailedInstances = append(resp.FailedInstances, resource.InstanceID)
			} else {
				resp.UpdateCount++
				// 记录更新日志
				s.recordChangeLog(ctx, existing, resource, model.ChangeSourceSync, operatorID, operatorName)
			}
		} else {
			// 创建新资源
			if err := s.dao.Create(ctx, resource); err != nil {
				s.logger.Error("创建资源失败", zap.String("instanceID", resource.InstanceID), zap.Error(err))
				resp.FailedCount++
				resp.FailedInstances = append(resp.FailedInstances, resource.InstanceID)
			} else {
				resp.NewCount++
				// 记录创建日志
				s.recordChangeLog(ctx, nil, resource, model.ChangeSourceSync, operatorID, operatorName)

				// 如果启用自动绑定，则绑定到指定节点
				if autoBind && bindNodeID > 0 {
					if err := s.dao.BindTreeNodes(ctx, resource.ID, []int{bindNodeID}); err != nil {
						s.logger.Error("自动绑定资源到节点失败",
							zap.Int("resourceID", resource.ID),
							zap.Int("nodeID", bindNodeID),
							zap.Error(err))
					}
				}
			}
		}
	}

	return nil
}

// recordChangeLog 记录资源变更日志
func (s *treeCloudService) recordChangeLog(ctx context.Context, oldResource, newResource *model.TreeCloudResource, source string, operatorID int, operatorName string) {
	// 如果是删除操作
	if oldResource != nil && newResource == nil {
		changeLog := &model.CloudResourceChangeLog{
			ResourceID:   oldResource.ID,
			InstanceID:   oldResource.InstanceID,
			ChangeType:   model.ChangeTypeDeleted,
			FieldName:    "",
			OldValue:     oldResource.Name,
			NewValue:     "",
			ChangeSource: source,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			ChangeTime:   time.Now(),
		}
		// 保存变更日志
		if err := s.dao.CreateChangeLog(ctx, changeLog); err != nil {
			s.logger.Error("保存删除日志失败", zap.Error(err))
		}
		return
	}

	// 如果是创建操作
	if oldResource == nil && newResource != nil {
		changeLog := &model.CloudResourceChangeLog{
			ResourceID:   newResource.ID,
			InstanceID:   newResource.InstanceID,
			ChangeType:   model.ChangeTypeCreated,
			FieldName:    "",
			OldValue:     "",
			NewValue:     newResource.Name,
			ChangeSource: source,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			ChangeTime:   time.Now(),
		}
		// 保存变更日志
		if err := s.dao.CreateChangeLog(ctx, changeLog); err != nil {
			s.logger.Error("保存创建日志失败", zap.Error(err))
		}
		return
	}

	// 如果是更新操作，比较字段变化
	if oldResource != nil && newResource != nil {
		// 比较状态
		if oldResource.Status != newResource.Status {
			changeLog := &model.CloudResourceChangeLog{
				ResourceID:   newResource.ID,
				InstanceID:   newResource.InstanceID,
				ChangeType:   model.ChangeTypeStatusChanged,
				FieldName:    "status",
				OldValue:     fmt.Sprintf("%d", oldResource.Status),
				NewValue:     fmt.Sprintf("%d", newResource.Status),
				ChangeSource: source,
				OperatorID:   operatorID,
				OperatorName: operatorName,
				ChangeTime:   time.Now(),
			}
			// 保存变更日志
			if err := s.dao.CreateChangeLog(ctx, changeLog); err != nil {
				s.logger.Error("保存状态变更日志失败", zap.Error(err))
			}
		}
		// 可以继续比较其他字段...
	}
}

// saveSyncHistory 保存同步历史
func (s *treeCloudService) saveSyncHistory(ctx context.Context, history *model.CloudResourceSyncHistory) {
	if err := s.dao.CreateSyncHistory(ctx, history); err != nil {
		s.logger.Error("保存同步历史失败", zap.Error(err))
	}
}

// GetSyncHistory 获取同步历史
func (s *treeCloudService) GetSyncHistory(ctx context.Context, req *model.GetCloudResourceSyncHistoryReq) (model.ListResp[*model.CloudResourceSyncHistory], error) {
	// 兜底分页参数
	treeUtils.ValidateAndSetPaginationDefaults(&req.Page, &req.Size)

	histories, total, err := s.dao.GetSyncHistoryList(ctx, req)
	if err != nil {
		s.logger.Error("获取同步历史失败", zap.Error(err))
		return model.ListResp[*model.CloudResourceSyncHistory]{}, err
	}

	return model.ListResp[*model.CloudResourceSyncHistory]{
		Items: histories,
		Total: total,
	}, nil
}

// GetChangeLog 获取资源变更日志
func (s *treeCloudService) GetChangeLog(ctx context.Context, req *model.GetCloudResourceChangeLogReq) (model.ListResp[*model.CloudResourceChangeLog], error) {
	// 兜底分页参数
	treeUtils.ValidateAndSetPaginationDefaults(&req.Page, &req.Size)

	logs, total, err := s.dao.GetChangeLogList(ctx, req)
	if err != nil {
		s.logger.Error("获取变更日志失败", zap.Error(err))
		return model.ListResp[*model.CloudResourceChangeLog]{}, err
	}

	return model.ListResp[*model.CloudResourceChangeLog]{
		Items: logs,
		Total: total,
	}, nil
}

// BatchDeleteTreeCloudResource 批量删除云资源
func (s *treeCloudService) BatchDeleteTreeCloudResource(ctx context.Context, req *model.BatchDeleteTreeCloudResourceReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量删除ID列表不能为空")
	}

	// 检查所有云资源是否存在
	resources, err := s.dao.BatchGetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(resources) != len(req.IDs) {
		return errors.New("部分云资源不存在")
	}

	// 执行批量删除
	if err := s.dao.BatchDelete(ctx, req.IDs); err != nil {
		s.logger.Error("批量删除云资源失败", zap.Error(err))
		return err
	}

	// 记录删除日志
	for _, resource := range resources {
		s.recordChangeLog(ctx, resource, nil, model.ChangeSourceManual, req.OperatorID, req.OperatorName)
	}

	s.logger.Info("批量删除云资源成功", zap.Ints("ids", req.IDs))
	return nil
}

// BatchUpdateCloudResourceStatus 批量更新云资源状态
func (s *treeCloudService) BatchUpdateCloudResourceStatus(ctx context.Context, req *model.BatchUpdateCloudResourceStatusReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量更新ID列表不能为空")
	}

	// 检查所有云资源是否存在
	resources, err := s.dao.BatchGetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(resources) != len(req.IDs) {
		return errors.New("部分云资源不存在")
	}

	// 执行批量更新状态
	if err := s.dao.BatchUpdateStatus(ctx, req.IDs, req.Status); err != nil {
		s.logger.Error("批量更新云资源状态失败", zap.Error(err))
		return err
	}

	// 记录状态变更日志
	for _, resource := range resources {
		if resource.Status != req.Status {
			changeLog := &model.CloudResourceChangeLog{
				ResourceID:   resource.ID,
				InstanceID:   resource.InstanceID,
				ChangeType:   model.ChangeTypeStatusChanged,
				FieldName:    "status",
				OldValue:     fmt.Sprintf("%d", resource.Status),
				NewValue:     fmt.Sprintf("%d", req.Status),
				ChangeSource: model.ChangeSourceManual,
				OperatorID:   req.OperatorID,
				OperatorName: req.OperatorName,
				ChangeTime:   time.Now(),
			}
			go func(log *model.CloudResourceChangeLog) {
				if err := s.dao.CreateChangeLog(context.Background(), log); err != nil {
					s.logger.Error("记录状态变更日志失败", zap.Error(err))
				}
			}(changeLog)
		}
	}

	s.logger.Info("批量更新云资源状态成功",
		zap.Ints("ids", req.IDs),
		zap.Int8("status", int8(req.Status)))
	return nil
}
