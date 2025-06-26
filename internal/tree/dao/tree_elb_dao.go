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

type TreeElbDAO interface {
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error)
	GetElbResourceById(ctx context.Context, id int) (*model.ResourceElb, error)
	GetElbResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceElb, error)
	CreateElbResource(ctx context.Context, resource *model.ResourceElb) error
	UpdateElbResource(ctx context.Context, resource *model.ResourceElb) error
	DeleteElbResource(ctx context.Context, id int) error

	// 服务器绑定管理
	GetElbHealthCheck(ctx context.Context, elbId int) (*model.ElbHealthCheck, error)
	CreateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error
	UpdateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error
	GetElbResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceElb, error)
	GetElbResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceElb, error)
	GetElbResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceElb, error)
	GetElbResourcesByVpcId(ctx context.Context, vpcId string) ([]*model.ResourceElb, error)

	CountElbResourcesByProvider(ctx context.Context, provider string) (int64, error)
	CountElbResourcesByRegion(ctx context.Context, region string) (int64, error)
	CountElbResourcesByStatus(ctx context.Context, status string) (int64, error)

	BatchDeleteElbResources(ctx context.Context, ids []int) error

	WithTx(tx *gorm.DB) TreeElbDAO

	GetElbListeners(ctx context.Context, elbId int) ([]*model.ElbListener, error)
	CreateElbListener(ctx context.Context, listener *model.ElbListener) error
	UpdateElbListener(ctx context.Context, listener *model.ElbListener) error
	DeleteElbListener(ctx context.Context, listenerId int) error

	GetElbRules(ctx context.Context, listenerId int) ([]*model.ElbRule, error)
	CreateElbRule(ctx context.Context, rule *model.ElbRule) error
	UpdateElbRule(ctx context.Context, rule *model.ElbRule) error
	DeleteElbRule(ctx context.Context, ruleId int) error
}

type treeElbDAO struct {
	db *gorm.DB
}

func NewTreeElbDAO(db *gorm.DB) TreeElbDAO {
	return &treeElbDAO{
		db: db,
	}
}

// BatchDeleteElbResources 批量删除ELB资源
func (t *treeElbDAO) BatchDeleteElbResources(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return t.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.ResourceElb{}).Error
}

// CountElbResourcesByProvider 按云厂商统计ELB资源数量
func (t *treeElbDAO) CountElbResourcesByProvider(ctx context.Context, provider string) (int64, error) {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.ResourceElb{}).Where("provider = ?", provider).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountElbResourcesByRegion 按区域统计ELB资源数量
func (t *treeElbDAO) CountElbResourcesByRegion(ctx context.Context, region string) (int64, error) {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.ResourceElb{}).Where("region_id = ?", region).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountElbResourcesByStatus 按状态统计ELB资源数量
func (t *treeElbDAO) CountElbResourcesByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.ResourceElb{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CreateElbHealthCheck 创建健康检查配置
func (t *treeElbDAO) CreateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error {
	// 这里可以根据实际需求实现健康检查配置的创建逻辑
	// 如果需要存储在数据库中，可以创建相应的表结构
	return nil
}

// CreateElbListener 创建监听器
func (t *treeElbDAO) CreateElbListener(ctx context.Context, listener *model.ElbListener) error {
	// 这里可以根据实际需求实现监听器的创建逻辑
	return nil
}

// CreateElbResource 创建ELB资源
func (t *treeElbDAO) CreateElbResource(ctx context.Context, resource *model.ResourceElb) error {
	resource.LastSyncTime = time.Now()
	return t.db.WithContext(ctx).Create(resource).Error
}

// CreateElbRule 创建监听器规则
func (t *treeElbDAO) CreateElbRule(ctx context.Context, rule *model.ElbRule) error {
	// 这里可以根据实际需求实现规则的创建逻辑
	return nil
}

// DeleteElbListener 删除监听器
func (t *treeElbDAO) DeleteElbListener(ctx context.Context, listenerId int) error {
	// 这里可以根据实际需求实现监听器的删除逻辑
	return nil
}

// DeleteElbResource 删除ELB资源
func (t *treeElbDAO) DeleteElbResource(ctx context.Context, id int) error {
	return t.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ResourceElb{}).Error
}

// DeleteElbRule 删除监听器规则
func (t *treeElbDAO) DeleteElbRule(ctx context.Context, ruleId int) error {
	// 这里可以根据实际需求实现规则的删除逻辑
	return nil
}

// GetElbHealthCheck 获取ELB健康检查配置
func (t *treeElbDAO) GetElbHealthCheck(ctx context.Context, elbId int) (*model.ElbHealthCheck, error) {
	// 这里可以根据实际需求实现健康检查的查询逻辑
	// 如果健康检查配置存储在单独的表中，则查询该表
	// 这里为示例，返回默认配置
	return &model.ElbHealthCheck{
		Enabled:            true,
		Type:               "http",
		Port:               80,
		Path:               "/health",
		Interval:           30,
		Timeout:            5,
		HealthyThreshold:   3,
		UnhealthyThreshold: 3,
		HttpCode:           "200",
	}, nil
}

// GetElbListeners 获取ELB监听器列表
func (t *treeElbDAO) GetElbListeners(ctx context.Context, elbId int) ([]*model.ElbListener, error) {
	// 这里可以根据实际需求实现监听器的查询逻辑
	// 如果监听器信息存储在单独的表中，则查询该表
	return []*model.ElbListener{
		{
			Port:     80,
			Protocol: "HTTP",
			Status:   "active",
		},
		{
			Port:     443,
			Protocol: "HTTPS",
			Status:   "active",
		},
	}, nil
}

// GetElbResourceById 根据ID获取ELB资源
func (t *treeElbDAO) GetElbResourceById(ctx context.Context, id int) (*model.ResourceElb, error) {
	var resource model.ResourceElb
	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetElbResourceByInstanceId 根据实例ID获取ELB资源
func (t *treeElbDAO) GetElbResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceElb, error) {
	var resource model.ResourceElb
	if err := t.db.WithContext(ctx).Where("instance_id = ?", instanceId).First(&resource).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetElbResourcesByProvider 根据云厂商获取ELB资源列表
func (t *treeElbDAO) GetElbResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceElb, error) {
	var resources []*model.ResourceElb
	if err := t.db.WithContext(ctx).Where("provider = ?", provider).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

// GetElbResourcesByRegion 根据区域获取ELB资源列表
func (t *treeElbDAO) GetElbResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceElb, error) {
	var resources []*model.ResourceElb
	if err := t.db.WithContext(ctx).Where("region_id = ?", region).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

// GetElbResourcesByStatus 根据状态获取ELB资源列表
func (t *treeElbDAO) GetElbResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceElb, error) {
	var resources []*model.ResourceElb
	if err := t.db.WithContext(ctx).Where("status = ?", status).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

// GetElbResourcesByVpcId 根据VPC ID获取ELB资源列表
func (t *treeElbDAO) GetElbResourcesByVpcId(ctx context.Context, vpcId string) ([]*model.ResourceElb, error) {
	var resources []*model.ResourceElb
	if err := t.db.WithContext(ctx).Where("vpc_id = ?", vpcId).Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

// GetElbRules 获取监听器规则列表
func (t *treeElbDAO) GetElbRules(ctx context.Context, listenerId int) ([]*model.ElbRule, error) {
	// 这里可以根据实际需求实现规则的查询逻辑
	return []*model.ElbRule{}, nil
}

// ListElbResources 获取ELB资源列表
func (t *treeElbDAO) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error) {
	var resources []*model.ResourceElb
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceElb{})

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
	if req.Env != "" {
		db = db.Where("environment = ?", req.Env)
	}
	if req.TreeNodeID > 0 {
		db = db.Where("tree_node_id = ?", req.TreeNodeID)
	}
	if req.Keyword != "" {
		db = db.Where("instance_name LIKE ? OR instance_id LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return model.ListResp[*model.ResourceElb]{}, err
	}

	// 分页
	if req.PageSize > 0 && req.PageNumber > 0 {
		offset := (req.PageNumber - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	// 排序
	db = db.Order("created_at DESC")

	if err := db.Find(&resources).Error; err != nil {
		return model.ListResp[*model.ResourceElb]{}, err
	}

	return model.ListResp[*model.ResourceElb]{
		Items: resources,
		Total: total,
	}, nil
}

// UpdateElbHealthCheck 更新健康检查配置
func (t *treeElbDAO) UpdateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error {
	// 这里可以根据实际需求实现健康检查配置的更新逻辑
	return nil
}

// UpdateElbListener 更新监听器
func (t *treeElbDAO) UpdateElbListener(ctx context.Context, listener *model.ElbListener) error {
	// 这里可以根据实际需求实现监听器的更新逻辑
	return nil
}

// UpdateElbResource 更新ELB资源
func (t *treeElbDAO) UpdateElbResource(ctx context.Context, resource *model.ResourceElb) error {
	resource.LastSyncTime = time.Now()
	return t.db.WithContext(ctx).Model(&model.ResourceElb{}).Where("id = ?", resource.ID).Updates(resource).Error
}

// UpdateElbRule 更新监听器规则
func (t *treeElbDAO) UpdateElbRule(ctx context.Context, rule *model.ElbRule) error {
	// 这里可以根据实际需求实现规则的更新逻辑
	return nil
}

// WithTx 返回使用事务的DAO实例
func (t *treeElbDAO) WithTx(tx *gorm.DB) TreeElbDAO {
	return &treeElbDAO{
		db: tx,
	}
}
