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
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strconv"
)

type TreeLocalDAO interface {
	Create(ctx context.Context, local *model.TreeLocal) error
	Update(ctx context.Context, local *model.TreeLocal) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.TreeLocal, error)
	GetList(ctx context.Context, req *model.GetTreeLocalListReq) ([]*model.TreeLocal, int64, error)
	GetByIP(ctx context.Context, ip string) (*model.TreeLocal, error)
	BindTreeNodes(ctx context.Context, localID int, treeNodeIds []int) error
	UnBindTreeNodes(ctx context.Context, localID int, treeNodeIds []int) error
	BatchGetByIDs(ctx context.Context, ids []int) ([]*model.TreeLocal, error)
}

type treeLocalDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewTreeLocalDAO(db *gorm.DB, logger *zap.Logger) TreeLocalDAO {
	return &treeLocalDAO{
		logger: logger,
		db:     db,
	}
}

// Create 创建本地主机
func (d *treeLocalDAO) Create(ctx context.Context, local *model.TreeLocal) error {
	if err := d.db.WithContext(ctx).Create(local).Error; err != nil {
		d.logger.Error("创建本地主机失败", zap.Error(err))
		return err
	}

	return nil
}

// Update 更新本地主机
func (d *treeLocalDAO) Update(ctx context.Context, local *model.TreeLocal) error {
	if err := d.db.WithContext(ctx).Model(local).Updates(local).Error; err != nil {
		d.logger.Error("更新本地主机失败", zap.Error(err))
		return err
	}

	return nil
}

// Delete 删除本地主机
func (d *treeLocalDAO) Delete(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Delete(&model.TreeLocal{}, id).Error; err != nil {
		d.logger.Error("删除本地主机失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetByID 根据ID获取本地主机详情
func (d *treeLocalDAO) GetByID(ctx context.Context, id int) (*model.TreeLocal, error) {
	var local model.TreeLocal

	err := d.db.WithContext(ctx).Where("id = ?", id).First(&local).Error
	if err != nil {
		d.logger.Error("根据ID获取本地主机详情失败", zap.Error(err), zap.Int("id", id))
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

	if req.Environment != "" {
		query = query.Where("environment = ?", req.Environment)
	}

	if req.Search != "" {
		query = query.Where("name LIKE ?",
			"%"+req.Search+"%")
	}

	// 计算总数
	err := query.Count(&total).Error
	if err != nil {
		d.logger.Error("获取本地主机总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err = query.
		Order("created_at DESC").
		Limit(req.Size).
		Offset(offset).
		Find(&locals).Error
	if err != nil {
		d.logger.Error("获取本地主机列表失败", zap.Error(err))
		return nil, 0, err
	}

	return locals, total, nil
}

// GetByIP 根据IP地址获取主机
func (d *treeLocalDAO) GetByIP(ctx context.Context, ip string) (*model.TreeLocal, error) {
	var local model.TreeLocal

	err := d.db.WithContext(ctx).Where("ip_addr = ?", ip).First(&local).Error
	if err != nil {
		d.logger.Error("根据IP地址获取本地主机失败", zap.Error(err), zap.String("ip", ip))
		return nil, err
	}

	return &local, nil
}

func (d *treeLocalDAO) BatchGetByIDs(ctx context.Context, ids []int) ([]*model.TreeLocal, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var locals []*model.TreeLocal

	if err := d.db.WithContext(ctx).Where("id IN (?)", ids).Find(&locals).Error; err != nil {
		d.logger.Error("批量获取本地主机失败", zap.Error(err), zap.Ints("ids", ids))
		return nil, err
	}

	return locals, nil
}

// BindTreeNodes 绑定树节点
func (d *treeLocalDAO) BindTreeNodes(ctx context.Context, localID int, treeNodeIds []int) error {
	var resource []*model.TreeNodeResource

	for _, id := range treeNodeIds {
		// 先检查是否已经绑定,不允许重复绑定
		var existingResource model.TreeNodeResource
		err := d.db.WithContext(ctx).Where(
			"tree_node_id = ? AND resource_id = ? AND resource_type = ?",
			id, strconv.Itoa(localID), model.CloudProviderLocal,
		).First(&existingResource).Error

		if err == nil {
			// 如果找到了记录，说明已经绑定过了
			d.logger.Warn("树节点已经绑定，跳过",
				zap.Int("treeNodeId", id),
				zap.Int("localID", localID))
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果是其他错误（非记录不存在），返回错误
			d.logger.Error("检查绑定状态失败", zap.Error(err),
				zap.Int("treeNodeId", id),
				zap.Int("localID", localID))
			return err
		}

		// 如果记录不存在，添加到待创建列表
		resource = append(resource, &model.TreeNodeResource{
			TreeNodeID:   id,
			ResourceID:   strconv.Itoa(localID),
			ResourceType: model.CloudProviderLocal,
		})
	}

	// 如果没有需要创建的资源，直接返回
	if len(resource) == 0 {
		d.logger.Info("所有树节点都已绑定，无需创建新绑定")
		return nil
	}

	if err := d.db.WithContext(ctx).Create(&resource).Error; err != nil {
		d.logger.Error("绑定树节点失败", zap.Error(err), zap.Int("localID", localID), zap.Ints("treeNodeIds", treeNodeIds))
		return err
	}

	return nil
}

// UnBindTreeNodes 解绑树节点
func (d *treeLocalDAO) UnBindTreeNodes(ctx context.Context, localID int, treeNodeIds []int) error {
	if len(treeNodeIds) == 0 {
		d.logger.Info("没有需要解绑的树节点")
		return nil
	}

	result := d.db.WithContext(ctx).
		Where("resource_id = ? AND tree_node_id IN (?) AND resource_type = ?",
			strconv.Itoa(localID), treeNodeIds, model.CloudProviderLocal).
		Delete(&model.TreeNodeResource{})

	if result.Error != nil {
		d.logger.Error("解绑树节点失败", zap.Error(result.Error), zap.Int("localID", localID), zap.Ints("treeNodeIds", treeNodeIds))
		return result.Error
	}

	d.logger.Info("解绑树节点成功",
		zap.Int("localID", localID),
		zap.Ints("treeNodeIds", treeNodeIds),
		zap.Int64("deletedCount", result.RowsAffected))

	return nil
}
