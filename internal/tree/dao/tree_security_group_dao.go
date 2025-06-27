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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type TreeSecurityGroupDAO interface {
	// 安全组基础操作
	CreateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error
	DeleteSecurityGroup(ctx context.Context, securityGroupID string) error
	GetSecurityGroupDetail(ctx context.Context, securityGroupID string) (*model.ResourceSecurityGroup, error)
	ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) ([]*model.ResourceSecurityGroup, error)
	UpdateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error

	// 安全组规则操作
	AddSecurityGroupRule(ctx context.Context, rule *model.SecurityGroupRule) error
	RemoveSecurityGroupRule(ctx context.Context, securityGroupID string, ruleID string) error
	GetSecurityGroupRules(ctx context.Context, securityGroupID string) ([]*model.SecurityGroupRule, error)

	// 实例绑定操作
	BindInstanceToSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error
	UnbindInstanceFromSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error
	GetInstanceSecurityGroups(ctx context.Context, instanceID string) ([]*model.ResourceSecurityGroup, error)

	// 辅助查询方法
	SecurityGroupExists(ctx context.Context, securityGroupID string) (bool, error)
	GetSecurityGroupByName(ctx context.Context, name string) (*model.ResourceSecurityGroup, error)
	GetSecurityGroupsByRegion(ctx context.Context, region string) ([]*model.ResourceSecurityGroup, error)

	// 同步操作
	SyncSecurityGroupResources(ctx context.Context, resources []*model.ResourceSecurityGroup, total int64) error
}

type treeSecurityGroupDAO struct {
	db *gorm.DB
}

func NewTreeSecurityGroupDAO(db *gorm.DB) TreeSecurityGroupDAO {
	return &treeSecurityGroupDAO{
		db: db,
	}
}

// AddSecurityGroupRule 添加安全组规则
func (t *treeSecurityGroupDAO) AddSecurityGroupRule(ctx context.Context, rule *model.SecurityGroupRule) error {
	return t.db.WithContext(ctx).Create(rule).Error
}

// BindInstanceToSecurityGroup 绑定实例到安全组
func (t *treeSecurityGroupDAO) BindInstanceToSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error {
	// 这里需要更新ECS实例的安全组关联
	// 由于安全组绑定通常是在ECS表中维护security_group_ids字段
	// 这里可以根据实际业务需求实现
	return nil
}

// CreateSecurityGroup 创建安全组
func (t *treeSecurityGroupDAO) CreateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error {
	return t.db.WithContext(ctx).Create(securityGroup).Error
}

// DeleteSecurityGroup 删除安全组
func (t *treeSecurityGroupDAO) DeleteSecurityGroup(ctx context.Context, securityGroupID string) error {
	return t.db.WithContext(ctx).Where("instance_id = ?", securityGroupID).Delete(&model.ResourceSecurityGroup{}).Error
}

// GetInstanceSecurityGroups 获取实例关联的安全组
func (t *treeSecurityGroupDAO) GetInstanceSecurityGroups(ctx context.Context, instanceID string) ([]*model.ResourceSecurityGroup, error) {
	// 先获取实例信息
	var ecs model.ResourceEcs
	if err := t.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&ecs).Error; err != nil {
		return nil, err
	}

	// 根据安全组ID列表查询安全组详情
	var securityGroups []*model.ResourceSecurityGroup
	if len(ecs.SecurityGroupIds) > 0 {
		if err := t.db.WithContext(ctx).Where("instance_id IN ?", ecs.SecurityGroupIds).Find(&securityGroups).Error; err != nil {
			return nil, err
		}
	}

	return securityGroups, nil
}

// GetSecurityGroupByName 根据名称获取安全组
func (t *treeSecurityGroupDAO) GetSecurityGroupByName(ctx context.Context, name string) (*model.ResourceSecurityGroup, error) {
	var sg model.ResourceSecurityGroup
	if err := t.db.WithContext(ctx).Where("security_group_name = ?", name).First(&sg).Error; err != nil {
		return nil, err
	}
	return &sg, nil
}

// GetSecurityGroupDetail 获取安全组详情
func (t *treeSecurityGroupDAO) GetSecurityGroupDetail(ctx context.Context, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	var sg model.ResourceSecurityGroup
	if err := t.db.WithContext(ctx).Preload("SecurityGroupRules").Where("instance_id = ?", securityGroupID).First(&sg).Error; err != nil {
		return nil, err
	}
	return &sg, nil
}

// GetSecurityGroupRules 获取安全组规则列表
func (t *treeSecurityGroupDAO) GetSecurityGroupRules(ctx context.Context, securityGroupID string) ([]*model.SecurityGroupRule, error) {
	var rules []*model.SecurityGroupRule
	if err := t.db.WithContext(ctx).Where("security_group_id = ?", securityGroupID).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetSecurityGroupsByRegion 根据区域获取安全组列表
func (t *treeSecurityGroupDAO) GetSecurityGroupsByRegion(ctx context.Context, region string) ([]*model.ResourceSecurityGroup, error) {
	var securityGroups []*model.ResourceSecurityGroup
	if err := t.db.WithContext(ctx).Where("region_id = ?", region).Find(&securityGroups).Error; err != nil {
		return nil, err
	}
	return securityGroups, nil
}

// ListSecurityGroups 获取安全组列表
func (t *treeSecurityGroupDAO) ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) ([]*model.ResourceSecurityGroup, error) {
	var securityGroups []*model.ResourceSecurityGroup

	db := t.db.WithContext(ctx).Model(&model.ResourceSecurityGroup{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}
	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}
	if req.VpcId != "" {
		db = db.Where("vpc_id = ?", req.VpcId)
	}
	if req.TreeNodeId > 0 {
		db = db.Where("tree_node_id = ?", req.TreeNodeId)
	}
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.Env != "" {
		db = db.Where("environment = ?", req.Env)
	}
	if req.Search != "" {
		db = db.Where("security_group_name LIKE ? OR description LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 分页
	if req.Size > 0 && req.Page > 0 {
		offset := (req.Page - 1) * req.Size
		db = db.Offset(offset).Limit(req.Size)
	}

	db = db.Order("created_at DESC")

	if err := db.Find(&securityGroups).Error; err != nil {
		return nil, err
	}

	return securityGroups, nil
}

// RemoveSecurityGroupRule 删除安全组规则
func (t *treeSecurityGroupDAO) RemoveSecurityGroupRule(ctx context.Context, securityGroupID string, ruleID string) error {
	return t.db.WithContext(ctx).Where("security_group_id = ? AND id = ?", securityGroupID, ruleID).Delete(&model.SecurityGroupRule{}).Error
}

// SecurityGroupExists 检查安全组是否存在
func (t *treeSecurityGroupDAO) SecurityGroupExists(ctx context.Context, securityGroupID string) (bool, error) {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.ResourceSecurityGroup{}).Where("instance_id = ?", securityGroupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UnbindInstanceFromSecurityGroup 从安全组解绑实例
func (t *treeSecurityGroupDAO) UnbindInstanceFromSecurityGroup(ctx context.Context, securityGroupID string, instanceID string) error {
	// 这里需要更新ECS实例的安全组关联
	// 由于安全组绑定通常是在ECS表中维护security_group_ids字段
	// 这里可以根据实际业务需求实现
	return nil
}

// UpdateSecurityGroup 更新安全组
func (t *treeSecurityGroupDAO) UpdateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error {
	return t.db.WithContext(ctx).Model(&model.ResourceSecurityGroup{}).Where("id = ?", securityGroup.ID).Updates(securityGroup).Error
}

// SyncSecurityGroupResources 同步安全组资源到数据库
func (t *treeSecurityGroupDAO) SyncSecurityGroupResources(ctx context.Context, resources []*model.ResourceSecurityGroup, total int64) error {
	if len(resources) == 0 {
		return nil
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, resource := range resources {
			var existingResource model.ResourceSecurityGroup
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
