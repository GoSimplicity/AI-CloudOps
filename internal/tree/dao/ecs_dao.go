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

type EcsDAO interface {
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, error)
	GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error)
	CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error
	DeleteEcsResource(ctx context.Context, instanceId string) error
}

type ecsDAO struct {
	db *gorm.DB
}

func NewEcsDAO(db *gorm.DB) EcsDAO {
	return &ecsDAO{
		db: db,
	}
}

// CreateEcsResource 创建ECS资源
func (e *ecsDAO) CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	if err := e.db.WithContext(ctx).Create(resource).Error; err != nil {
		return err
	}
	return nil
}

// GetEcsResourceById implements EcsDAO.
func (e *ecsDAO) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error) {
	var result model.ResourceEcs
	if err := e.db.Where("id = ?", id).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

// ListEcsResources 获取ECS资源列表
func (e *ecsDAO) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, error) {
	var result []*model.ResourceEcs
	var total int64

	query := e.db.WithContext(ctx).Model(&model.ResourceEcs{})

	if req.Provider != "" {
		query = query.Where("provider = ?", req.Provider)
	}

	if req.Region != "" {
		query = query.Where("region = ?", req.Region)
	}

	if req.Search != "" {
		query = query.Where("name LIKE ? OR instance_id LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (e *ecsDAO) DeleteEcsResource(ctx context.Context, instanceId string) error {
	if err := e.db.WithContext(ctx).Where("id = ?", instanceId).Delete(&model.ResourceEcs{}).Error; err != nil {
		return err
	}
	return nil
}
