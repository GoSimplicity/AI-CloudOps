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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type TreeRdsDAO interface {
	// 基础CRUD操作
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error)
	GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRds, error)
	CreateRdsResource(ctx context.Context, resource *model.ResourceRds) error
	UpdateRdsResource(ctx context.Context, resource *model.ResourceRds) error
	DeleteRdsResource(ctx context.Context, id int) error

	// RDS实例状态操作
	StartRdsInstance(ctx context.Context, instanceId string) error
	StopRdsInstance(ctx context.Context, instanceId string) error
	RestartRdsInstance(ctx context.Context, instanceId string) error

	// RDS实例管理操作
	ResizeRdsInstance(ctx context.Context, instanceId string, dbInstanceClass string, allocatedStorage int) error
	ResetRdsPassword(ctx context.Context, instanceId string, username string, password string) error
	RenewRdsInstance(ctx context.Context, instanceId string, period int, periodUnit string) error

	// RDS备份恢复操作
	BackupRdsInstance(ctx context.Context, instanceId string, backupName string) error
	RestoreRdsInstance(ctx context.Context, instanceId string, backupId string, restoreTime string) error

	// 辅助查询方法
	GetRdsInstanceStatus(ctx context.Context, id int) (string, error)
	CheckRdsInstanceExists(ctx context.Context, id int) (bool, error)
	UpdateRdsInstanceStatus(ctx context.Context, id int, status string) error

	// 批量操作
	BatchUpdateRdsStatus(ctx context.Context, ids []int, status string) error
	GetRdsInstancesByStatus(ctx context.Context, status string) ([]*model.ResourceRds, error)
}

type treeRdsDAO struct {
	db *gorm.DB
}

func NewTreeRdsDAO(db *gorm.DB) TreeRdsDAO {
	return &treeRdsDAO{
		db: db,
	}
}

// BackupRdsInstance 备份RDS实例
func (t *treeRdsDAO) BackupRdsInstance(ctx context.Context, instanceId string, backupName string) error {
	// 备份操作通常不改变实例状态，只是创建备份任务
	updates := map[string]any{
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// BatchUpdateRdsStatus 批量更新RDS状态
func (t *treeRdsDAO) BatchUpdateRdsStatus(ctx context.Context, ids []int, status string) error {
	if len(ids) == 0 {
		return nil
	}

	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("id IN ?", ids).Updates(updates).Error
}

// CheckRdsInstanceExists 检查RDS实例是否存在
func (t *treeRdsDAO) CheckRdsInstanceExists(ctx context.Context, id int) (bool, error) {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateRdsResource 创建RDS资源
func (t *treeRdsDAO) CreateRdsResource(ctx context.Context, resource *model.ResourceRds) error {
	return t.db.WithContext(ctx).Create(resource).Error
}

// DeleteRdsResource 删除RDS资源
func (t *treeRdsDAO) DeleteRdsResource(ctx context.Context, id int) error {
	return t.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ResourceRds{}).Error
}

// GetRdsInstanceStatus 获取RDS实例状态
func (t *treeRdsDAO) GetRdsInstanceStatus(ctx context.Context, id int) (string, error) {
	var status string
	if err := t.db.WithContext(ctx).Model(&model.ResourceRds{}).Select("status").Where("id = ?", id).Scan(&status).Error; err != nil {
		return "", err
	}
	return status, nil
}

// GetRdsInstancesByStatus 根据状态获取RDS实例列表
func (t *treeRdsDAO) GetRdsInstancesByStatus(ctx context.Context, status string) ([]*model.ResourceRds, error) {
	var instances []*model.ResourceRds
	if err := t.db.WithContext(ctx).Where("status = ?", status).Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

// GetRdsResourceById 根据ID获取RDS资源
func (t *treeRdsDAO) GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRds, error) {
	var resource model.ResourceRds
	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// ListRdsResources 获取RDS资源列表
func (t *treeRdsDAO) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error) {
	var resources []*model.ResourceRds
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceRds{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}
	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}
	if req.ZoneId != "" {
		db = db.Where("zone_id = ?", req.ZoneId)
	}
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.Engine != "" {
		db = db.Where("engine = ?", req.Engine)
	}
	if req.TreeNodeId > 0 {
		db = db.Where("tree_node_id = ?", req.TreeNodeId)
	}
	if req.InstanceName != "" {
		db = db.Where("instance_name LIKE ?", "%"+req.InstanceName+"%")
	}
	if req.Environment != "" {
		db = db.Where("env = ?", req.Environment)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return model.ListResp[*model.ResourceRds]{}, err
	}

	// 分页
	if req.PageSize > 0 && req.PageNumber > 0 {
		offset := (req.PageNumber - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	// 排序
	db = db.Order("created_at DESC")

	if err := db.Find(&resources).Error; err != nil {
		return model.ListResp[*model.ResourceRds]{}, err
	}

	return model.ListResp[*model.ResourceRds]{
		Items: resources,
		Total: total,
	}, nil
}

// RenewRdsInstance 续费RDS实例
func (t *treeRdsDAO) RenewRdsInstance(ctx context.Context, instanceId string, period int, periodUnit string) error {
	// 续费操作通常不会改变实例状态，只是更新更新时间
	updates := map[string]any{
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// ResetRdsPassword 重置RDS实例密码
func (t *treeRdsDAO) ResetRdsPassword(ctx context.Context, instanceId string, username string, password string) error {
	// 更新状态为修改中
	updates := map[string]any{
		"status":     "Modifying",
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// ResizeRdsInstance 调整RDS实例规格
func (t *treeRdsDAO) ResizeRdsInstance(ctx context.Context, instanceId string, dbInstanceClass string, allocatedStorage int) error {
	updates := map[string]any{
		"db_instance_class": dbInstanceClass,
		"status":            "Modifying",
		"updated_at":        time.Now(),
	}

	if allocatedStorage > 0 {
		updates["allocated_storage"] = allocatedStorage
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// RestartRdsInstance 重启RDS实例
func (t *treeRdsDAO) RestartRdsInstance(ctx context.Context, instanceId string) error {
	return t.UpdateRdsInstanceStatusByInstanceId(ctx, instanceId, "Restarting")
}

// RestoreRdsInstance 恢复RDS实例
func (t *treeRdsDAO) RestoreRdsInstance(ctx context.Context, instanceId string, backupId string, restoreTime string) error {
	return t.UpdateRdsInstanceStatusByInstanceId(ctx, instanceId, "Restoring")
}

// StartRdsInstance 启动RDS实例
func (t *treeRdsDAO) StartRdsInstance(ctx context.Context, instanceId string) error {
	return t.UpdateRdsInstanceStatusByInstanceId(ctx, instanceId, "Starting")
}

// StopRdsInstance 停止RDS实例
func (t *treeRdsDAO) StopRdsInstance(ctx context.Context, instanceId string) error {
	return t.UpdateRdsInstanceStatusByInstanceId(ctx, instanceId, "Stopping")
}

// UpdateRdsInstanceStatus 更新RDS实例状态
func (t *treeRdsDAO) UpdateRdsInstanceStatus(ctx context.Context, id int, status string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateRdsInstanceStatusByInstanceId 根据实例ID更新RDS实例状态
func (t *treeRdsDAO) UpdateRdsInstanceStatusByInstanceId(ctx context.Context, instanceId string, status string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceRds{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// UpdateRdsResource 更新RDS资源
func (t *treeRdsDAO) UpdateRdsResource(ctx context.Context, resource *model.ResourceRds) error {
	return t.db.WithContext(ctx).Save(resource).Error
}
