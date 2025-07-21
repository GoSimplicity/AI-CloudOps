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

type TreeLocalDAO interface {
	Create(ctx context.Context, local *model.TreeLocal) error
	Update(ctx context.Context, local *model.TreeLocal) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.TreeLocal, error)
	GetList(ctx context.Context, req *model.GetTreeLocalListReq) ([]*model.TreeLocal, int64, error)
	UpdateStatus(ctx context.Context, id int, status string) error
	UpdateTreeNodes(ctx context.Context, id int, treeNodeIDs []string) error
	GetByIP(ctx context.Context, ip string) (*model.TreeLocal, error)
	BatchDelete(ctx context.Context, ids []int) error
}

type treeLocalDAO struct {
	db *gorm.DB
}

func NewTreeLocalDAO(db *gorm.DB) TreeLocalDAO {
	return &treeLocalDAO{
		db: db,
	}
}

// Create 创建本地主机
func (d *treeLocalDAO) Create(ctx context.Context, local *model.TreeLocal) error {
	return d.db.WithContext(ctx).Create(local).Error
}

// Update 更新本地主机
func (d *treeLocalDAO) Update(ctx context.Context, local *model.TreeLocal) error {
	return d.db.WithContext(ctx).Model(local).Updates(local).Error
}

// Delete 删除本地主机
func (d *treeLocalDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.TreeLocal{}, id).Error
}

// GetByID 根据ID获取本地主机详情
func (d *treeLocalDAO) GetByID(ctx context.Context, id int) (*model.TreeLocal, error) {
	var local model.TreeLocal
	err := d.db.WithContext(ctx).Preload("EcsTreeNodes").First(&local, id).Error
	if err != nil {
		return nil, err
	}
	return &local, nil
}

// GetList 获取本地主机列表
func (d *treeLocalDAO) GetList(ctx context.Context, req *model.GetTreeLocalListReq) ([]*model.TreeLocal, int64, error) {
	var locals []*model.TreeLocal
	var total int64

	query := d.db.WithContext(ctx).Model(&model.TreeLocal{})

	// 添加查询条件
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Env != "" {
		query = query.Where("environment = ?", req.Env)
	}
	if req.Search != "" {
		query = query.Where("name LIKE ?",
			"%"+req.Search+"%")
	}

	// 计算总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err = query.Preload("EcsTreeNodes").
		Order("created_at DESC").
		Limit(req.Size).
		Offset(offset).
		Find(&locals).Error
	if err != nil {
		return nil, 0, err
	}

	return locals, total, nil
}

// UpdateStatus 更新状态
func (d *treeLocalDAO) UpdateStatus(ctx context.Context, id int, status string) error {
	return d.db.WithContext(ctx).Model(&model.TreeLocal{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateTreeNodes 更新关联的服务树节点
func (d *treeLocalDAO) UpdateTreeNodes(ctx context.Context, id int, treeNodeIDs []string) error {
	return d.db.WithContext(ctx).Model(model.TreeLocal{}).Where("id = ?", id).
		Update("tree_node_ids", treeNodeIDs).Error
}

// GetByIP 根据IP地址获取主机
func (d *treeLocalDAO) GetByIP(ctx context.Context, ip string) (*model.TreeLocal, error) {
	var local model.TreeLocal
	err := d.db.WithContext(ctx).Where("ip_addr = ?", ip).First(&local).Error
	if err != nil {
		return nil, err
	}
	return &local, nil
}

// BatchDelete 批量删除
func (d *treeLocalDAO) BatchDelete(ctx context.Context, ids []int) error {
	return d.db.WithContext(ctx).Delete(&model.TreeLocal{}, ids).Error
}
