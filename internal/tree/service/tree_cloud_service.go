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

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
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
	passwordEncryptionKey string
	clientFactory         treeUtils.CloudClientFactory
	credentialFactory     treeUtils.CloudCredentialClientFactory
	logger                *zap.Logger
	dao                   dao.TreeCloudDAO
	cloudAccountDAO       dao.CloudAccountDAO
}

func NewTreeCloudService(cfg *config.Config, logger *zap.Logger, dao dao.TreeCloudDAO, cloudAccountDAO dao.CloudAccountDAO) (TreeCloudService, error) {
	if cfg == nil {
		return nil, errors.New("配置不能为空")
	}
	return &treeCloudService{
		passwordEncryptionKey: cfg.Tree.PasswordEncryptionKey,
		logger:                logger,
		dao:                   dao,
		cloudAccountDAO:       cloudAccountDAO,
	}, nil
}

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
		plainPassword, err := treeUtils.DecryptPassword(cloud.Password, s.passwordEncryptionKey)
		if err != nil {
			s.logger.Error("SSH密码解密失败", zap.Int("id", req.ID), zap.Error(err))
			return nil, fmt.Errorf("SSH密码解密失败: %w", err)
		}
		cloud.Password = plainPassword
	}

	return cloud, nil
}

func (s *treeCloudService) UpdateTreeCloudResource(ctx context.Context, req *model.UpdateTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

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
		encrypted, err := treeUtils.EncryptPassword(req.Password, s.passwordEncryptionKey)
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

	if len(metadata) == 0 {
		s.logger.Info("没有字段需要更新", zap.Int("id", req.ID))
		return nil
	}

	if err := s.dao.UpdateMetadata(ctx, req.ID, metadata); err != nil {
		s.logger.Error("更新云资源元数据失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

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
		go func(log *model.CloudResourceChangeLog) {
			if err := s.dao.CreateChangeLog(context.Background(), log); err != nil {
				s.logger.Error("记录变更日志失败", zap.Error(err))
			}
		}(changeLog)
	}

	s.logger.Info("更新云资源元数据成功", zap.Int("id", req.ID), zap.Int("fields", len(metadata)))
	return nil
}

func (s *treeCloudService) DeleteTreeCloudResource(ctx context.Context, req *model.DeleteTreeCloudResourceReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

	cloud, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("云资源不存在")
		}
		s.logger.Error("获取云资源失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

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

func (s *treeCloudService) UpdateCloudResourceStatus(ctx context.Context, req *model.UpdateCloudResourceStatusReq) error {
	if err := treeUtils.ValidateID(req.ID); err != nil {
		return fmt.Errorf("无效的云资源ID: %w", err)
	}

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

func (s *treeCloudService) SyncTreeCloudResource(ctx context.Context, req *model.SyncTreeCloudResourceReq) (*model.SyncCloudResourceResp, error) {
	if req == nil {
		return nil, errors.New("同步请求不能为空")
	}
	if req.SyncMode != "" && req.SyncMode != model.SyncModeFull && req.SyncMode != model.SyncModeIncremental {
		return nil, errors.New("不支持的同步模式")
	}
	if req.AutoBind && req.BindNodeID <= 0 {
		return nil, errors.New("自动绑定需要有效节点")
	}
	for _, resourceType := range req.ResourceTypes {
		if resourceType != model.ResourceTypeECS {
			return nil, errors.New("当前仅支持同步云服务器")
		}
	}
	if req.SyncMode == model.SyncModeFull && len(req.InstanceIDs) > 0 {
		return nil, fmt.Errorf("指定实例ID只能使用增量同步，避免删除未查询的资源")
	}
	startTime := time.Now()

	if req.SyncMode == "" {
		req.SyncMode = model.SyncModeIncremental
	}

	resp := &model.SyncCloudResourceResp{
		SyncTime:        startTime,
		FailedInstances: []string{},
		Regions:         []model.CloudRegionSyncResult{},
	}

	syncHistory := &model.CloudResourceSyncHistory{
		CloudAccountID: req.CloudAccountID,
		SyncMode:       req.SyncMode,
		StartTime:      startTime,
		SyncStatus:     "running",
		OperatorID:     req.OperatorID,
		OperatorName:   req.OperatorName,
	}

	account, err := s.cloudAccountDAO.GetByID(ctx, req.CloudAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("云账户不存在")
		}
		s.logger.Error("获取云账户失败", zap.Int("cloudAccountID", req.CloudAccountID), zap.Error(err))
		return nil, err
	}

	if account.Status != model.CloudAccountEnabled {
		return nil, errors.New("云账户已禁用，无法同步资源")
	}
	if account.Provider == model.ProviderHuawei && (req.SyncMode == model.SyncModeFull || len(req.InstanceIDs) == 0) {
		return nil, errors.New("华为云仅支持指定实例ID的增量同步：无法证明跨企业项目资源列表完整")
	}
	if (account.Provider == model.ProviderAzure || account.Provider == model.ProviderGCP) && (req.SyncMode == model.SyncModeFull || len(req.InstanceIDs) == 0) {
		return nil, errors.New("该云厂商仅支持指定实例ID的增量同步：无法证明全量覆盖")
	}

	// 校验整个地域集合后再解密或访问云端，避免部分有效输入触发部分同步。
	var regionsToSync []*model.CloudAccountRegion
	if len(req.CloudAccountRegionIDs) > 0 {
		regionsByID := make(map[int]*model.CloudAccountRegion, len(account.Regions))
		for _, region := range account.Regions {
			if region != nil && region.CloudAccountID == req.CloudAccountID {
				regionsByID[region.ID] = region
			}
		}
		seen := make(map[int]bool, len(req.CloudAccountRegionIDs))
		for _, regionID := range req.CloudAccountRegionIDs {
			region := regionsByID[regionID]
			if regionID <= 0 || region == nil || region.Status != model.CloudAccountRegionEnabled || seen[regionID] {
				return nil, fmt.Errorf("指定的地域 %d 不存在、不属于该账户、已禁用或重复", regionID)
			}
			seen[regionID] = true
			regionsToSync = append(regionsToSync, region)
		}
	} else {
		for _, region := range account.Regions {
			if region != nil && region.CloudAccountID == req.CloudAccountID && region.Status == model.CloudAccountRegionEnabled {
				regionsToSync = append(regionsToSync, region)
			}
		}
	}
	if len(regionsToSync) == 0 {
		return nil, errors.New("没有可用的区域进行同步")
	}

	credentials, err := treeUtils.LoadCloudAccountCredentials(account, s.passwordEncryptionKey)
	if err != nil {
		return nil, err
	}
	operationCtx, cancel := treeUtils.CredentialContext(ctx, credentials)
	defer cancel()

	s.logger.Info("开始同步云资源",
		zap.Int("cloudAccountID", req.CloudAccountID),
		zap.Int8("provider", int8(account.Provider)),
		zap.Int("regionCount", len(regionsToSync)),
		zap.String("syncMode", string(req.SyncMode)))

	// 根据不同的云厂商调用对应的同步逻辑，遍历所有区域
	var syncErr error
	switch account.Provider {
	case model.ProviderAliyun, model.ProviderTencent, model.ProviderAWS, model.ProviderHuawei, model.ProviderAzure, model.ProviderGCP:
		syncErr = s.syncCloudResourcesWithCredentials(operationCtx, account, credentials, regionsToSync, req, resp)
	default:
		syncErr = fmt.Errorf("不支持的云厂商: %d", account.Provider)
	}

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

	if syncErr != nil && len(resp.Regions) == 0 {
		return nil, syncErr
	}
	successful, unknown := 0, false
	for _, region := range resp.Regions {
		if region.Status == "success" {
			successful++
		}
		if region.Status == "unknown" {
			unknown = true
		}
	}
	switch {
	case unknown:
		resp.Status, resp.Message = "unknown", "部分地域提交结果未知，请核对后处理"
		syncHistory.SyncStatus, syncHistory.ErrorMessage = "failed", resp.Message
	case resp.FailedCount > 0 && successful > 0:
		resp.Status, resp.Message = "partial", "部分地域同步成功，其余地域未完成"
		syncHistory.SyncStatus, syncHistory.ErrorMessage = "partial", resp.Message
	case resp.FailedCount > 0:
		resp.Status, resp.Message = "failed", "所有选定地域均未完成"
		syncHistory.SyncStatus, syncHistory.ErrorMessage = "failed", resp.Message
	default:
		resp.Status, resp.Message = "success", "选定地域同步完成"
		syncHistory.SyncStatus = "success"
	}
	historyCtx, historyCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	s.saveSyncHistory(historyCtx, syncHistory)
	historyCancel()

	s.logger.Info("云资源同步结果",
		zap.String("status", resp.Status),
		zap.Int("total", resp.TotalCount),
		zap.Int("new", resp.NewCount),
		zap.Int("update", resp.UpdateCount),
		zap.Int("delete", resp.DeleteCount),
		zap.Int("failed", resp.FailedCount),
		zap.Duration("duration", endTime.Sub(startTime)))

	return resp, nil
}

func (s *treeCloudService) syncCloudResourcesForMultipleRegions(ctx context.Context, account *model.CloudAccount, accessKey, secretKey string, regions []*model.CloudAccountRegion, req *model.SyncTreeCloudResourceReq, resp *model.SyncCloudResourceResp) error {
	return s.syncCloudResourcesWithCredentials(ctx, account, treeUtils.CloudCredentials{AccessKey: accessKey, SecretKey: secretKey}, regions, req, resp)
}

func (s *treeCloudService) syncCloudResourcesWithCredentials(ctx context.Context, account *model.CloudAccount, credentials treeUtils.CloudCredentials, regions []*model.CloudAccountRegion, req *model.SyncTreeCloudResourceReq, resp *model.SyncCloudResourceResp) error {
	var failures []error
	for _, region := range regions {
		stats, err := s.syncOneRegion(ctx, account, credentials, region, req)
		if err != nil {
			result := stats
			result.Status = "failed"
			switch {
			case errors.Is(err, dao.ErrRegionSyncCommitUnknown):
				result.Status, result.ErrorCategory = "unknown", "commit_unknown"
				if errors.Is(err, errRegionAuditUnavailable) {
					result.ErrorCategory = "commit_unknown_audit_unavailable"
				}
			case errors.Is(err, errRegionAuditUnavailable):
				result.ErrorCategory, result.RetrySafe = "audit_unavailable", false
			case errors.Is(err, dao.ErrRegionSyncBusy):
				result.ErrorCategory = "busy"
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				result.Status, result.ErrorCategory = "cancelled", "cancelled"
			default:
				result.ErrorCategory = "sync_failed"
			}
			resp.FailedCount++
			resp.Regions = append(resp.Regions, result)
			failures = append(failures, fmt.Errorf("地域 %d 同步未完成", region.ID))
			s.logger.Warn("地域同步未完成", zap.Int("region_id", region.ID), zap.String("category", result.ErrorCategory))
			continue
		}
		stats.Region = region.Region
		resp.Regions = append(resp.Regions, stats)
		resp.TotalCount += stats.TotalCount
		resp.NewCount += stats.NewCount
		resp.UpdateCount += stats.UpdateCount
		resp.DeleteCount += stats.DeleteCount
	}
	return errors.Join(failures...)
}

var errRegionAuditUnavailable = errors.New("地域同步历史不可用")

func (s *treeCloudService) syncOneRegion(ctx context.Context, account *model.CloudAccount, credentials treeUtils.CloudCredentials, region *model.CloudAccountRegion, req *model.SyncTreeCloudResourceReq) (result model.CloudRegionSyncResult, resultErr error) {
	result = model.CloudRegionSyncResult{RegionID: region.ID, Region: region.Region, RetrySafe: true}
	if s.dao == nil {
		result.RetrySafe = false
		return result, errRegionAuditUnavailable
	}
	operationCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	lockCtx, stopLock := context.WithTimeout(operationCtx, 5*time.Second)
	lease, err := s.dao.AcquireRegionSyncLease(lockCtx, region.ID, 5*time.Minute)
	stopLock()
	if err != nil {
		auditCtx, stopAudit := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer stopAudit()
		now := time.Now().UTC()
		status, category := model.SyncStatusFailed, "busy_or_acquisition_failed"
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			status, category = "cancelled", "cancelled"
		}
		history := &model.CloudResourceSyncHistory{CloudAccountID: account.ID, RegionID: region.ID, Region: region.Region, SyncMode: req.SyncMode, SyncStatus: status, StartTime: now, EndTime: &now, FailedCount: 1, ErrorMessage: category, OperatorID: req.OperatorID, OperatorName: req.OperatorName}
		if auditErr := s.dao.CreateSyncHistory(auditCtx, history); auditErr != nil {
			result.RetrySafe = false
			return result, errors.Join(err, errRegionAuditUnavailable)
		}
		result.HistoryID = history.ID
		return result, err
	}
	release := true
	defer func() {
		if release {
			releaseCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
			defer stop()
			if err := s.dao.ReleaseRegionSyncLease(releaseCtx, lease); err != nil {
				s.logger.Warn("地域租约释放失败", zap.Int("region_id", region.ID))
			}
		}
	}()
	history := &model.CloudResourceSyncHistory{CloudAccountID: account.ID, RegionID: region.ID, Region: region.Region, OperationID: lease.Token, LeaseExpiresMS: lease.ExpiresEpochMS, SyncMode: req.SyncMode, SyncStatus: "running", StartTime: time.Now().UTC(), OperatorID: req.OperatorID, OperatorName: req.OperatorName}
	if err := s.dao.CreateSyncHistory(operationCtx, history); err != nil {
		result.RetrySafe = false
		return result, errRegionAuditUnavailable
	}
	result.HistoryID = history.ID
	defer func() {
		if resultErr == nil {
			return
		}
		status, category := "failed", "sync_failed"
		if errors.Is(resultErr, dao.ErrRegionSyncCommitUnknown) {
			status, category = "unknown", "commit_unknown"
			release = false
			result.RetrySafe = false
		} else if errors.Is(resultErr, context.Canceled) || errors.Is(resultErr, context.DeadlineExceeded) {
			status, category = "cancelled", "cancelled"
		}
		auditCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := s.dao.FinishRegionSyncHistory(auditCtx, history.ID, lease.Token, status, category); err != nil {
			result.RetrySafe = false
			resultErr = errors.Join(resultErr, errRegionAuditUnavailable)
		}
	}()
	client, err := cloudClient(account.Provider, credentials, region, s.logger, s.credentialFactory, s.clientFactory)
	if err != nil {
		return result, err
	}
	resources, err := client.ListInstances(operationCtx, req.InstanceIDs)
	if err != nil {
		return result, err
	}
	if resources == nil {
		return result, errors.New("云端快照为空响应")
	}
	if err := operationCtx.Err(); err != nil {
		return result, err
	}
	for _, resource := range resources {
		if resource == nil {
			return result, errors.New("云端快照无效")
		}
		resource.CloudAccountID, resource.CloudAccountRegionID, resource.Region = account.ID, region.ID, region.Region
	}
	var committed model.CloudRegionSyncResult
	err = s.dao.ApplyRegionSync(operationCtx, account, region, lease, func(tx dao.TreeCloudDAO) error {
		stats, applyErr := reconcileRegion(operationCtx, tx, region.ID, resources, req.SyncMode == model.SyncModeFull, req.AutoBind, req.BindNodeID, req.OperatorID, req.OperatorName)
		if applyErr != nil {
			return applyErr
		}
		if auditErr := tx.CompleteRegionSyncHistory(operationCtx, history.ID, lease.Token, stats); auditErr != nil {
			return auditErr
		}
		committed = stats
		return nil
	})
	if err != nil {
		return result, err
	}
	committed.Region = region.Region
	committed.HistoryID = history.ID
	committed.RetrySafe = true
	return committed, nil
}

func (s *treeCloudService) recordChangeLog(ctx context.Context, oldResource, newResource *model.TreeCloudResource, source string, operatorID int, operatorName string) {
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
		if err := s.dao.CreateChangeLog(ctx, changeLog); err != nil {
			s.logger.Error("保存删除日志失败", zap.Error(err))
		}
		return
	}

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
		if err := s.dao.CreateChangeLog(ctx, changeLog); err != nil {
			s.logger.Error("保存创建日志失败", zap.Error(err))
		}
		return
	}

	// 如果是更新操作，比较字段变化
	if oldResource != nil && newResource != nil {
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
			if err := s.dao.CreateChangeLog(ctx, changeLog); err != nil {
				s.logger.Error("保存状态变更日志失败", zap.Error(err))
			}
		}
		// 可以继续比较其他字段...
	}
}

func (s *treeCloudService) saveSyncHistory(ctx context.Context, history *model.CloudResourceSyncHistory) {
	if err := s.dao.CreateSyncHistory(ctx, history); err != nil {
		s.logger.Error("保存同步历史失败", zap.Error(err))
	}
}

func (s *treeCloudService) GetSyncHistory(ctx context.Context, req *model.GetCloudResourceSyncHistoryReq) (model.ListResp[*model.CloudResourceSyncHistory], error) {
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

func (s *treeCloudService) GetChangeLog(ctx context.Context, req *model.GetCloudResourceChangeLogReq) (model.ListResp[*model.CloudResourceChangeLog], error) {
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

func (s *treeCloudService) BatchDeleteTreeCloudResource(ctx context.Context, req *model.BatchDeleteTreeCloudResourceReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量删除ID列表不能为空")
	}

	resources, err := s.dao.BatchGetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(resources) != len(req.IDs) {
		return errors.New("部分云资源不存在")
	}

	if err := s.dao.BatchDelete(ctx, req.IDs); err != nil {
		s.logger.Error("批量删除云资源失败", zap.Error(err))
		return err
	}

	for _, resource := range resources {
		s.recordChangeLog(ctx, resource, nil, model.ChangeSourceManual, req.OperatorID, req.OperatorName)
	}

	s.logger.Info("批量删除云资源成功", zap.Ints("ids", req.IDs))
	return nil
}

func (s *treeCloudService) BatchUpdateCloudResourceStatus(ctx context.Context, req *model.BatchUpdateCloudResourceStatusReq) error {
	if len(req.IDs) == 0 {
		return errors.New("批量更新ID列表不能为空")
	}

	resources, err := s.dao.BatchGetByIDs(ctx, req.IDs)
	if err != nil {
		return err
	}

	if len(resources) != len(req.IDs) {
		return errors.New("部分云资源不存在")
	}

	if err := s.dao.BatchUpdateStatus(ctx, req.IDs, req.Status); err != nil {
		s.logger.Error("批量更新云资源状态失败", zap.Error(err))
		return err
	}

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

// Only provider-owned fields are reconciled. Explicit empty values clear stale
// public IPs and expiry dates without touching local SSH credentials or bindings.
func providerResourceFields(r *model.TreeCloudResource) map[string]interface{} {
	return map[string]interface{}{
		"name": r.Name, "instance_type": r.InstanceType, "status": r.Status,
		"public_ip": r.PublicIP, "private_ip": r.PrivateIP, "vpc_id": r.VpcID, "zone_id": r.ZoneID,
		"cpu": r.Cpu, "memory": r.Memory, "disk": r.Disk, "os_type": r.OSType, "os_name": r.OSName,
		"image_id": r.ImageID, "image_name": r.ImageName, "charge_type": r.ChargeType, "expire_time": r.ExpireTime,
		"currency": r.Currency, "tags": r.Tags, "last_sync_time": r.LastSyncTime,
	}
}
