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

package dao

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeCloudDAO interface {
	Create(ctx context.Context, cloud *model.TreeCloudResource) error
	Update(ctx context.Context, cloud *model.TreeCloudResource) error
	UpdateMetadata(ctx context.Context, id int, metadata map[string]interface{}) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.TreeCloudResource, error)
	GetList(ctx context.Context, req *model.GetTreeCloudResourceListReq) ([]*model.TreeCloudResource, int64, error)
	GetByAccountAndInstanceID(ctx context.Context, cloudAccountID int, instanceID string) (*model.TreeCloudResource, error)
	GetByNodeID(ctx context.Context, nodeID int, req *model.GetTreeNodeCloudResourcesReq) ([]*model.TreeCloudResource, error)
	BindTreeNodes(ctx context.Context, cloudID int, treeNodeIds []int) error
	UnBindTreeNodes(ctx context.Context, cloudID int, treeNodeIds []int) error
	BatchGetByIDs(ctx context.Context, ids []int) ([]*model.TreeCloudResource, error)
	BatchCreate(ctx context.Context, clouds []*model.TreeCloudResource) error
	BatchDelete(ctx context.Context, ids []int) error
	UpdateStatus(ctx context.Context, id int, status model.CloudResourceStatus) error
	BatchUpdateStatus(ctx context.Context, ids []int, status model.CloudResourceStatus) error
	CreateSyncHistory(ctx context.Context, history *model.CloudResourceSyncHistory) error
	GetSyncHistoryList(ctx context.Context, req *model.GetCloudResourceSyncHistoryReq) ([]*model.CloudResourceSyncHistory, int64, error)
	CreateChangeLog(ctx context.Context, log *model.CloudResourceChangeLog) error
	GetChangeLogList(ctx context.Context, req *model.GetCloudResourceChangeLogReq) ([]*model.CloudResourceChangeLog, int64, error)
	GetResourcesByRegion(ctx context.Context, regionID int, resourceType model.CloudResourceType) ([]*model.TreeCloudResource, error)
	GetByRegionAndInstanceID(ctx context.Context, regionID int, instanceID string) (*model.TreeCloudResource, error)
}

type treeCloudDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewTreeCloudDAO(db *gorm.DB, logger *zap.Logger) TreeCloudDAO {
	return &treeCloudDAO{
		logger: logger,
		db:     db,
	}
}

// Create 创建云资源
func (d *treeCloudDAO) Create(ctx context.Context, cloud *model.TreeCloudResource) error {
	if err := d.db.WithContext(ctx).Create(cloud).Error; err != nil {
		d.logger.Error("创建云资源失败", zap.Error(err))
		return err
	}

	return nil
}

// Update 更新云资源
func (d *treeCloudDAO) Update(ctx context.Context, cloud *model.TreeCloudResource) error {
	if err := d.db.WithContext(ctx).Model(cloud).Updates(cloud).Error; err != nil {
		d.logger.Error("更新云资源失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMetadata 更新云资源元数据
func (d *treeCloudDAO) UpdateMetadata(ctx context.Context, id int, metadata map[string]interface{}) error {
	if err := d.db.WithContext(ctx).
		Model(&model.TreeCloudResource{}).
		Where("id = ?", id).
		Updates(metadata).Error; err != nil {
		d.logger.Error("更新云资源元数据失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// Delete 删除云资源
func (d *treeCloudDAO) Delete(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Delete(&model.TreeCloudResource{}, id).Error; err != nil {
		d.logger.Error("删除云资源失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetByID 根据ID获取云资源详情
func (d *treeCloudDAO) GetByID(ctx context.Context, id int) (*model.TreeCloudResource, error) {
	var cloud model.TreeCloudResource

	err := d.db.WithContext(ctx).Preload("TreeNodes").Where("id = ?", id).First(&cloud).Error
	if err != nil {
		d.logger.Error("根据ID获取云资源详情失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &cloud, nil
}

// GetList 获取云资源列表
func (d *treeCloudDAO) GetList(ctx context.Context, req *model.GetTreeCloudResourceListReq) ([]*model.TreeCloudResource, int64, error) {
	var clouds []*model.TreeCloudResource
	var total int64

	query := d.db.WithContext(ctx).Model(&model.TreeCloudResource{})

	// 添加查询条件
	if req.CloudAccountID != 0 {
		query = query.Where("cloud_account_id = ?", req.CloudAccountID)
	}

	if req.ResourceType != 0 {
		query = query.Where("resource_type = ?", req.ResourceType)
	}

	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}

	if req.Environment != "" {
		query = query.Where("environment = ?", req.Environment)
	}

	if req.Search != "" {
		query = query.Where("name LIKE ? OR instance_id LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 计算总数
	err := query.Count(&total).Error
	if err != nil {
		d.logger.Error("获取云资源总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询，关联云账户信息
	offset := (req.Page - 1) * req.Size
	err = query.
		Order("created_at DESC").
		Preload("CloudAccount").
		Preload("TreeNodes").
		Limit(req.Size).
		Offset(offset).
		Find(&clouds).Error
	if err != nil {
		d.logger.Error("获取云资源列表失败", zap.Error(err))
		return nil, 0, err
	}

	return clouds, total, nil
}

// GetByAccountAndInstanceID 根据云账户ID和实例ID获取云资源
func (d *treeCloudDAO) GetByAccountAndInstanceID(ctx context.Context, cloudAccountID int, instanceID string) (*model.TreeCloudResource, error) {
	var cloud model.TreeCloudResource

	err := d.db.WithContext(ctx).
		Where("cloud_account_id = ? AND instance_id = ?", cloudAccountID, instanceID).
		First(&cloud).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		d.logger.Error("根据云账户和实例ID获取云资源失败", zap.Error(err), zap.Int("cloudAccountID", cloudAccountID), zap.String("instanceID", instanceID))
		return nil, err
	}

	return &cloud, nil
}

// GetByNodeID 根据树节点ID获取云资源列表
func (d *treeCloudDAO) GetByNodeID(ctx context.Context, nodeID int, req *model.GetTreeNodeCloudResourcesReq) ([]*model.TreeCloudResource, error) {
	var clouds []*model.TreeCloudResource

	query := d.db.WithContext(ctx).
		Joins("JOIN cl_tree_node_cloud ON cl_tree_node_cloud.tree_cloud_resource_id = cl_tree_cloud_resource.id").
		Where("cl_tree_node_cloud.tree_node_id = ?", nodeID)

	// 添加过滤条件
	if req.CloudAccountID != 0 {
		query = query.Where("cl_tree_cloud_resource.cloud_account_id = ?", req.CloudAccountID)
	}

	if req.ResourceType != 0 {
		query = query.Where("cl_tree_cloud_resource.resource_type = ?", req.ResourceType)
	}

	if req.Status != 0 {
		query = query.Where("cl_tree_cloud_resource.status = ?", req.Status)
	}

	err := query.Preload("CloudAccount").Find(&clouds).Error
	if err != nil {
		d.logger.Error("根据节点ID获取云资源失败", zap.Error(err), zap.Int("nodeID", nodeID))
		return nil, err
	}

	return clouds, nil
}

// BatchGetByIDs 批量获取云资源
func (d *treeCloudDAO) BatchGetByIDs(ctx context.Context, ids []int) ([]*model.TreeCloudResource, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var clouds []*model.TreeCloudResource

	if err := d.db.WithContext(ctx).Where("id IN ?", ids).Find(&clouds).Error; err != nil {
		d.logger.Error("批量获取云资源失败", zap.Error(err), zap.Ints("ids", ids))
		return nil, err
	}

	return clouds, nil
}

// BatchCreate 批量创建云资源
func (d *treeCloudDAO) BatchCreate(ctx context.Context, clouds []*model.TreeCloudResource) error {
	if len(clouds) == 0 {
		return nil
	}

	if err := d.db.WithContext(ctx).Create(&clouds).Error; err != nil {
		d.logger.Error("批量创建云资源失败", zap.Error(err))
		return err
	}

	d.logger.Info("批量创建云资源成功", zap.Int("count", len(clouds)))
	return nil
}

// UpdateStatus 更新云资源状态
func (d *treeCloudDAO) UpdateStatus(ctx context.Context, id int, status model.CloudResourceStatus) error {
	if err := d.db.WithContext(ctx).
		Model(&model.TreeCloudResource{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		d.logger.Error("更新云资源状态失败", zap.Error(err), zap.Int("id", id), zap.Int8("status", int8(status)))
		return err
	}

	return nil
}

// BindTreeNodes 绑定树节点
func (d *treeCloudDAO) BindTreeNodes(ctx context.Context, cloudID int, treeNodeIds []int) error {
	if !treeUtils.ValidateTreeNodeIDs(treeNodeIds) {
		d.logger.Info("没有需要绑定的树节点")
		return nil
	}

	// 获取云资源
	var cloud model.TreeCloudResource
	if err := d.db.WithContext(ctx).First(&cloud, cloudID).Error; err != nil {
		d.logger.Error("获取云资源失败", zap.Error(err), zap.Int("cloudID", cloudID))
		return err
	}

	// 构建要绑定的树节点列表
	var treeNodes []model.TreeNode
	for _, nodeID := range treeNodeIds {
		treeNodes = append(treeNodes, model.TreeNode{Model: model.Model{ID: nodeID}})
	}

	// 通过many2many关系绑定树节点
	if err := d.db.WithContext(ctx).Model(&cloud).Association("TreeNodes").Append(treeNodes); err != nil {
		d.logger.Error("绑定树节点失败", zap.Error(err), zap.Int("cloudID", cloudID), zap.Ints("treeNodeIds", treeNodeIds))
		return err
	}

	d.logger.Info("绑定树节点成功", zap.Int("cloudID", cloudID), zap.Ints("treeNodeIds", treeNodeIds))

	return nil
}

// UnBindTreeNodes 解绑树节点
func (d *treeCloudDAO) UnBindTreeNodes(ctx context.Context, cloudID int, treeNodeIds []int) error {
	if !treeUtils.ValidateTreeNodeIDs(treeNodeIds) {
		d.logger.Info("没有需要解绑的树节点")
		return nil
	}

	// 获取云资源
	var cloud model.TreeCloudResource
	if err := d.db.WithContext(ctx).First(&cloud, cloudID).Error; err != nil {
		d.logger.Error("获取云资源失败", zap.Error(err), zap.Int("cloudID", cloudID))
		return err
	}

	// 构建要解绑的树节点列表
	var treeNodes []model.TreeNode
	for _, nodeID := range treeNodeIds {
		treeNodes = append(treeNodes, model.TreeNode{Model: model.Model{ID: nodeID}})
	}

	// 通过many2many关系解绑树节点
	if err := d.db.WithContext(ctx).Model(&cloud).Association("TreeNodes").Delete(treeNodes); err != nil {
		d.logger.Error("解绑树节点失败", zap.Error(err), zap.Int("cloudID", cloudID), zap.Ints("treeNodeIds", treeNodeIds))
		return err
	}

	d.logger.Info("解绑树节点成功", zap.Int("cloudID", cloudID), zap.Ints("treeNodeIds", treeNodeIds))

	return nil
}

// CreateSyncHistory 创建同步历史记录
func (d *treeCloudDAO) CreateSyncHistory(ctx context.Context, history *model.CloudResourceSyncHistory) error {
	if err := d.db.WithContext(ctx).Create(history).Error; err != nil {
		d.logger.Error("创建同步历史失败", zap.Error(err))
		return err
	}

	d.logger.Info("创建同步历史成功",
		zap.Int("cloudAccountID", history.CloudAccountID),
		zap.String("syncStatus", string(history.SyncStatus)))
	return nil
}

// GetSyncHistoryList 获取同步历史列表
func (d *treeCloudDAO) GetSyncHistoryList(ctx context.Context, req *model.GetCloudResourceSyncHistoryReq) ([]*model.CloudResourceSyncHistory, int64, error) {
	var histories []*model.CloudResourceSyncHistory
	var total int64

	query := d.db.WithContext(ctx).Model(&model.CloudResourceSyncHistory{})

	// 添加查询条件
	if req.CloudAccountID != 0 {
		query = query.Where("cloud_account_id = ?", req.CloudAccountID)
	}

	if req.SyncStatus != "" {
		query = query.Where("sync_status = ?", req.SyncStatus)
	}

	if req.Search != "" {
		query = query.Where("error_message LIKE ?", "%"+req.Search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		d.logger.Error("获取同步历史总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Order("created_at DESC").
		Limit(req.Size).
		Offset(offset).
		Find(&histories).Error; err != nil {
		d.logger.Error("获取同步历史列表失败", zap.Error(err))
		return nil, 0, err
	}

	return histories, total, nil
}

// CreateChangeLog 创建变更日志
func (d *treeCloudDAO) CreateChangeLog(ctx context.Context, log *model.CloudResourceChangeLog) error {
	if err := d.db.WithContext(ctx).Create(log).Error; err != nil {
		d.logger.Error("创建变更日志失败", zap.Error(err))
		return err
	}

	return nil
}

// GetChangeLogList 获取变更日志列表
func (d *treeCloudDAO) GetChangeLogList(ctx context.Context, req *model.GetCloudResourceChangeLogReq) ([]*model.CloudResourceChangeLog, int64, error) {
	var logs []*model.CloudResourceChangeLog
	var total int64

	query := d.db.WithContext(ctx).Model(&model.CloudResourceChangeLog{})

	// 添加查询条件
	if req.ResourceID != 0 {
		query = query.Where("resource_id = ?", req.ResourceID)
	}

	if req.ChangeType != "" {
		query = query.Where("change_type = ?", req.ChangeType)
	}

	if req.Search != "" {
		query = query.Where("instance_id LIKE ? OR field_name LIKE ?",
			"%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		d.logger.Error("获取变更日志总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Order("change_time DESC").
		Limit(req.Size).
		Offset(offset).
		Find(&logs).Error; err != nil {
		d.logger.Error("获取变更日志列表失败", zap.Error(err))
		return nil, 0, err
	}

	return logs, total, nil
}

// GetResourcesByRegion 根据区域ID和资源类型获取资源列表
func (d *treeCloudDAO) GetResourcesByRegion(ctx context.Context, regionID int, resourceType model.CloudResourceType) ([]*model.TreeCloudResource, error) {
	var resources []*model.TreeCloudResource

	query := d.db.WithContext(ctx).Where("cloud_account_region_id = ?", regionID)
	if resourceType != 0 {
		query = query.Where("resource_type = ?", resourceType)
	}

	err := query.Find(&resources).Error
	if err != nil {
		d.logger.Error("根据区域获取资源列表失败",
			zap.Int("regionID", regionID),
			zap.Int8("resourceType", int8(resourceType)),
			zap.Error(err))
		return nil, err
	}

	return resources, nil
}

// GetByRegionAndInstanceID 根据区域ID和实例ID获取资源
func (d *treeCloudDAO) GetByRegionAndInstanceID(ctx context.Context, regionID int, instanceID string) (*model.TreeCloudResource, error) {
	var resource model.TreeCloudResource

	err := d.db.WithContext(ctx).
		Where("cloud_account_region_id = ? AND instance_id = ?", regionID, instanceID).
		First(&resource).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		d.logger.Error("根据区域和实例ID获取资源失败",
			zap.Int("regionID", regionID),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return nil, err
	}

	return &resource, nil
}

// BatchDelete 批量删除云资源
func (d *treeCloudDAO) BatchDelete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return errors.New("批量删除ID列表不能为空")
	}

	if err := d.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.TreeCloudResource{}).Error; err != nil {
		d.logger.Error("批量删除云资源失败", zap.Error(err), zap.Ints("ids", ids))
		return err
	}

	d.logger.Info("批量删除云资源成功", zap.Int("count", len(ids)))
	return nil
}

// BatchUpdateStatus 批量更新云资源状态
func (d *treeCloudDAO) BatchUpdateStatus(ctx context.Context, ids []int, status model.CloudResourceStatus) error {
	if len(ids) == 0 {
		return errors.New("批量更新ID列表不能为空")
	}

	if err := d.db.WithContext(ctx).
		Model(&model.TreeCloudResource{}).
		Where("id IN ?", ids).
		Update("status", status).Error; err != nil {
		d.logger.Error("批量更新云资源状态失败", zap.Error(err), zap.Ints("ids", ids), zap.Int8("status", int8(status)))
		return err
	}

	d.logger.Info("批量更新云资源状态成功", zap.Int("count", len(ids)), zap.Int8("status", int8(status)))
	return nil
}
