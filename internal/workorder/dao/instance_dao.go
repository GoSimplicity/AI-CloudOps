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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type InstanceDAO interface {
	CreateInstance(ctx context.Context, req *model.Instance) error
	UpdateInstance(ctx context.Context, req *model.Instance) error
	DeleteInstance(ctx context.Context, id int64) error
	ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error)
	GetInstance(ctx context.Context, id int64) (model.Instance, error)
	CreateInstanceFlow(ctx context.Context, req *model.InstanceFlow) error
	CreateInstanceComment(ctx context.Context, req *model.InstanceComment) error
}

type instanceDAO struct {
	db *gorm.DB
}

func NewInstanceDAO(db *gorm.DB) InstanceDAO {
	return &instanceDAO{
		db: db,
	}
}

// CreateInstance implements InstanceDAO.
func (i *instanceDAO) CreateInstance(ctx context.Context, instance *model.Instance) error {
	if err := i.db.WithContext(ctx).Create(instance).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}
	return nil
}

// DeleteInstance implements InstanceDAO.
func (i *instanceDAO) DeleteInstance(ctx context.Context, id int64) error {
	if err := i.db.WithContext(ctx).Delete(&model.Instance{}, id).Error; err != nil {
		return err
	}
	return nil
}

// GetInstance implements InstanceDAO.
func (i *instanceDAO) GetInstance(ctx context.Context, id int64) (model.Instance, error) {
	var instance model.Instance
	if err := i.db.WithContext(ctx).Where("id = ?", id).First(&instance).Error; err != nil {
		return instance, err
	}
	return instance, nil
}

// ListInstance implements InstanceDAO.
func (i *instanceDAO) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) {
	var instances []model.Instance
	db := i.db.WithContext(ctx).Model(&model.Instance{})

	// 关键字搜索
	if req.Keyword != "" {
		db = db.Where("title LIKE ?", "%"+req.Keyword+"%")
	}

	// 状态过滤
	if req.Status != 0 {
		db = db.Where("status = ?", req.Status)
	}

	// 日期范围过滤
	if len(req.DateRange) == 2 {
		db = db.Where("created_at BETWEEN ? AND ?", req.DateRange[0], req.DateRange[1])
	}

	// 创建人过滤
	if req.CreatorID != 0 {
		db = db.Where("creator_id = ?", req.CreatorID)
	}

	// 处理人过滤
	if req.AssigneeID != 0 {
		db = db.Where("assignee_id = ?", req.AssigneeID)
	}

	// 分页处理
	offset := (req.Page - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Find(&instances).Error; err != nil {
		return nil, err
	}

	return instances, nil
}

// UpdateInstance implements InstanceDAO.
func (i *instanceDAO) UpdateInstance(ctx context.Context, instance *model.Instance) error {
	// 检查 instance 是否为空
	if instance == nil {
		return fmt.Errorf("instance is nil")
	}
	// 检查 instance 的 ID 是否有效
	if instance.ID == 0 {
		return fmt.Errorf("invalid instance ID")
	}

	// 执行更新操作
	result := i.db.WithContext(ctx).Model(&model.Instance{}).Where("id = ?", instance.ID).Updates(instance)
	if result.Error != nil {
		return result.Error
	}

	// 检查是否有记录被更新
	if result.RowsAffected == 0 {
		return fmt.Errorf("no instance record found with ID %d", instance.ID)
	}

	return nil
}
func (i *instanceDAO) CreateInstanceFlow(ctx context.Context, req *model.InstanceFlow) error {
	if err := i.db.WithContext(ctx).Create(req).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}
	return nil
}
func (i *instanceDAO) CreateInstanceComment(ctx context.Context, req *model.InstanceComment) error {
	if err := i.db.WithContext(ctx).Create(req).Error; err != nil {
		if err == gorm.ErrDuplicatedKey {
			return fmt.Errorf("表单设计名称已存在")
		}
		return err
	}
	return nil
}
