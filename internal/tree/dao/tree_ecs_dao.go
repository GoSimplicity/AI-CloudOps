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
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type TreeEcsDAO interface {
	// 基础CRUD操作
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, int64, error)
	GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error)
	GetEcsResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceEcs, error)
	CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error
	UpdateEcsResource(ctx context.Context, resource *model.ResourceEcs) error
	DeleteEcsResource(ctx context.Context, id int) error

	// 状态更新操作
	UpdateEcsStatus(ctx context.Context, instanceId string, status string) error
	UpdateEcsPassword(ctx context.Context, instanceId string, passwordHash string) error
	UpdateEcsConfiguration(ctx context.Context, instanceId string, cpu int, memory int, diskSize int) error
	UpdateEcsRenewalInfo(ctx context.Context, instanceId string, expireTime string, renewalDuration int) error

	// 查询操作
	GetEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) ([]*model.ListEcsResourceOptionsResp, int64, error)
	GetEcsResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceEcs, error)
	GetEcsResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceEcs, error)
	GetEcsResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceEcs, error)

	// 批量操作
	BatchUpdateEcsStatus(ctx context.Context, instanceIds []string, status string) error
	BatchDeleteEcsResources(ctx context.Context, instanceIds []string) error

	// 统计操作
	CountEcsResourcesByProvider(ctx context.Context, provider string) (int64, error)
	CountEcsResourcesByRegion(ctx context.Context, region string) (int64, error)
	CountEcsResourcesByStatus(ctx context.Context, status string) (int64, error)

	// 同步操作
	SyncEcsResources(ctx context.Context, resources []*model.ResourceEcs, total int64) error

	// 事务操作
	WithTx(tx *gorm.DB) TreeEcsDAO
}

type treeEcsDAO struct {
	db *gorm.DB
}

func NewTreeEcsDAO(db *gorm.DB) TreeEcsDAO {
	return &treeEcsDAO{
		db: db,
	}
}

// CreateEcsResource 创建ECS资源
func (t *treeEcsDAO) CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	if resource.Provider == model.CloudProviderLocal {
		resource.InstanceId = fmt.Sprintf("%d", resource.ID)
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建ECS资源
		if err := tx.Create(resource).Error; err != nil {
			return err
		}

		// 如果指定了节点ID，则创建关联关系
		if resource.TreeNodeID > 0 {
			// 创建节点资源关联
			nodeResource := &model.TreeNodeResource{
				TreeNodeID:   resource.TreeNodeID,
				ResourceID:   resource.InstanceId,
				ResourceType: string(resource.Provider),
			}

			if err := tx.Create(nodeResource).Error; err != nil {
				return fmt.Errorf("创建节点资源关联失败: %w", err)
			}
		}

		return nil
	})
}

// DeleteEcsResource 删除ECS资源
func (t *treeEcsDAO) DeleteEcsResource(ctx context.Context, id int) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先查询资源信息，获取实例ID
		var resource model.ResourceEcs
		if err := tx.Where("id = ?", id).First(&resource).Error; err != nil {
			return fmt.Errorf("查询资源信息失败: %w", err)
		}

		// 删除节点资源关联关系
		if err := tx.Where("resource_id = ? AND resource_type = ?", resource.InstanceId, string(resource.Provider)).Delete(&model.TreeNodeResource{}).Error; err != nil {
			return fmt.Errorf("删除资源关联关系失败: %w", err)
		}

		// 删除ECS资源
		if err := tx.Where("id = ?", id).Delete(&model.ResourceEcs{}).Error; err != nil {
			return fmt.Errorf("删除资源失败: %w", err)
		}

		return nil
	})
}

// GetEcsResourceById 根据ID获取ECS资源
func (t *treeEcsDAO) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error) {
	var resource model.ResourceEcs

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&resource).Error; err != nil {
		return nil, err
	}

	return &resource, nil
}

// ListEcsResources 获取ECS资源列表
func (t *treeEcsDAO) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, int64, error) {
	var resources []*model.ResourceEcs
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceEcs{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}

	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}

	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 处理分页和排序
	if req.Size > 0 && req.Page > 0 {
		offset := (req.Page - 1) * req.Size
		db = db.Offset(offset).Limit(req.Size)
	}

	// 按照创建时间排序
	db = db.Order("created_at DESC")

	if err := db.Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	return resources, total, nil
}

// BatchDeleteEcsResources 批量删除ECS资源
func (t *treeEcsDAO) BatchDeleteEcsResources(ctx context.Context, instanceIds []string) error {
	if len(instanceIds) == 0 {
		return nil
	}

	return t.db.WithContext(ctx).Where("instance_id IN ?", instanceIds).Delete(&model.ResourceEcs{}).Error
}

// BatchUpdateEcsStatus 批量更新ECS状态
func (t *treeEcsDAO) BatchUpdateEcsStatus(ctx context.Context, instanceIds []string, status string) error {
	if len(instanceIds) == 0 {
		return nil
	}

	return t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("instance_id IN ?", instanceIds).Update("status", status).Error
}

// CountEcsResourcesByProvider 按云厂商统计ECS资源数量
func (t *treeEcsDAO) CountEcsResourcesByProvider(ctx context.Context, provider string) (int64, error) {
	var count int64
	err := t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("provider = ?", provider).Count(&count).Error
	return count, err
}

// CountEcsResourcesByRegion 按区域统计ECS资源数量
func (t *treeEcsDAO) CountEcsResourcesByRegion(ctx context.Context, region string) (int64, error) {
	var count int64
	err := t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("region_id = ?", region).Count(&count).Error
	return count, err
}

// CountEcsResourcesByStatus 按状态统计ECS资源数量
func (t *treeEcsDAO) CountEcsResourcesByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// GetEcsResourceByInstanceId 根据实例ID获取ECS资源
func (t *treeEcsDAO) GetEcsResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceEcs, error) {
	var resource model.ResourceEcs

	if err := t.db.WithContext(ctx).Where("instance_id = ?", instanceId).First(&resource).Error; err != nil {
		return nil, err
	}

	return &resource, nil
}

// GetEcsResourceOptions 获取ECS资源选项列表
func (t *treeEcsDAO) GetEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) ([]*model.ListEcsResourceOptionsResp, int64, error) {
	var options []*model.ListEcsResourceOptionsResp
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceEcs{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}
	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}
	if req.Zone != "" {
		db = db.Where("zone_id = ?", req.Zone)
	}
	if req.InstanceType != "" {
		db = db.Where("instance_type = ?", req.InstanceType)
	}
	if req.ImageId != "" {
		db = db.Where("image_id = ?", req.ImageId)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if req.Size > 0 && req.Page > 0 {
		offset := (req.Page - 1) * req.Size
		db = db.Offset(offset).Limit(req.Size)
	}

	// 查询数据并转换为选项格式
	var resources []*model.ResourceEcs
	if err := db.Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	for _, resource := range resources {
		options = append(options, &model.ListEcsResourceOptionsResp{
			Value:        resource.InstanceId,
			Label:        resource.InstanceName,
			InstanceType: resource.InstanceType,
			Region:       resource.RegionId,
			Zone:         resource.ZoneId,
			ImageId:      resource.ImageId,
			OSName:       resource.OSName,
			OSType:       resource.OsType,
			Cpu:          resource.Cpu,
			Memory:       resource.Memory,
			Valid:        resource.Status == "RUNNING",
		})
	}

	return options, total, nil
}

// GetEcsResourcesByProvider 根据云厂商获取ECS资源列表
func (t *treeEcsDAO) GetEcsResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceEcs, error) {
	var resources []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Where("provider = ?", provider).Find(&resources).Error; err != nil {
		return nil, err
	}

	return resources, nil
}

// GetEcsResourcesByRegion 根据区域获取ECS资源列表
func (t *treeEcsDAO) GetEcsResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceEcs, error) {
	var resources []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Where("region_id = ?", region).Find(&resources).Error; err != nil {
		return nil, err
	}

	return resources, nil
}

// GetEcsResourcesByStatus 根据状态获取ECS资源列表
func (t *treeEcsDAO) GetEcsResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceEcs, error) {
	var resources []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Where("status = ?", status).Find(&resources).Error; err != nil {
		return nil, err
	}

	return resources, nil
}

// UpdateEcsConfiguration 更新ECS配置信息
func (t *treeEcsDAO) UpdateEcsConfiguration(ctx context.Context, instanceId string, cpu int, memory int, diskSize int) error {
	updates := map[string]any{
		"cpu":        cpu,
		"memory":     memory,
		"disk":       diskSize,
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// UpdateEcsPassword 更新ECS密码
func (t *treeEcsDAO) UpdateEcsPassword(ctx context.Context, instanceId string, passwordHash string) error {
	updates := map[string]any{
		"password":   passwordHash,
		"updated_at": time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// UpdateEcsRenewalInfo 更新ECS续费信息
func (t *treeEcsDAO) UpdateEcsRenewalInfo(ctx context.Context, instanceId string, expireTime string, renewalDuration int) error {
	updates := map[string]any{
		"auto_release_time": expireTime,
		"updated_at":        time.Now(),
	}

	return t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// UpdateEcsResource 更新ECS资源信息
func (t *treeEcsDAO) UpdateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	if resource.Provider == model.CloudProviderLocal {
		resource.InstanceId = fmt.Sprintf("%d", resource.ID)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新ECS资源基本信息
		if err := tx.Model(&model.ResourceEcs{}).Where("id = ?", resource.ID).Updates(resource).Error; err != nil {
			return fmt.Errorf("更新ECS资源信息失败: %w", err)
		}

		// 如果指定了节点ID，则更新关联关系
		if resource.TreeNodeID > 0 {
			// 先查询是否已存在关联
			var count int64
			if err := tx.Model(&model.TreeNodeResource{}).
				Where("resource_id = ? AND resource_type = ?", resource.InstanceId, string(resource.Provider)).
				Count(&count).Error; err != nil {
				return fmt.Errorf("查询资源关联关系失败: %w", err)
			}

			if count == 0 {
				// 不存在则创建新关联
				nodeResource := &model.TreeNodeResource{
					TreeNodeID:   resource.TreeNodeID,
					ResourceID:   resource.InstanceId,
					ResourceType: string(resource.Provider),
				}
				if err := tx.Create(nodeResource).Error; err != nil {
					return fmt.Errorf("创建节点资源关联失败: %w", err)
				}
			} else {
				// 存在则更新关联
				if err := tx.Model(&model.TreeNodeResource{}).
					Where("resource_id = ? AND resource_type = ?", resource.InstanceId, string(resource.Provider)).
					Update("tree_node_id", resource.TreeNodeID).Error; err != nil {
					return fmt.Errorf("更新节点资源关联失败: %w", err)
				}
			}
		}

		return nil
	})
}

// UpdateEcsStatus 更新ECS状态
func (t *treeEcsDAO) UpdateEcsStatus(ctx context.Context, instanceId string, status string) error {
	updates := map[string]any{
		"status": status,
	}

	return t.db.WithContext(ctx).Model(&model.ResourceEcs{}).Where("instance_id = ?", instanceId).Updates(updates).Error
}

// SyncEcsResources 同步ECS资源到数据库
func (t *treeEcsDAO) SyncEcsResources(ctx context.Context, resources []*model.ResourceEcs, total int64) error {
	if len(resources) == 0 {
		return nil
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, resource := range resources {
			var existingResource model.ResourceEcs
			err := tx.Where("instance_id = ? AND provider = ? AND region_id = ?",
				resource.InstanceId, resource.Provider, resource.RegionId).First(&existingResource).Error

			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(resource).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				resource.ID = existingResource.ID
				resource.CreatedAt = existingResource.CreatedAt
				if err := tx.Model(&existingResource).Updates(resource).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// WithTx 返回使用事务的DAO实例
func (t *treeEcsDAO) WithTx(tx *gorm.DB) TreeEcsDAO {
	return &treeEcsDAO{
		db: tx,
	}
}
