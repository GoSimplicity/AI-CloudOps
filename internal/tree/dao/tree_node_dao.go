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
	"gorm.io/gorm/clause"
)

type TreeNodeDAO interface {
	// Create 创建一个新的 TreeNode 实例
	Create(ctx context.Context, obj *model.TreeNode) error
	// Delete 删除指定的 TreeNode 实例（软删除）
	Delete(ctx context.Context, id int) error
	// Upsert 创建或更新 TreeNode 实例
	Upsert(ctx context.Context, obj *model.TreeNode) error
	// Update 更新指定的 TreeNode 实例
	Update(ctx context.Context, obj *model.TreeNode) error
	// UpdateBindNode 更新 TreeNode 绑定的 ResourceEcs 节点
	UpdateBindNode(ctx context.Context, obj *model.TreeNode, ecs []*model.ResourceEcs) error
	// GetAll 获取所有 TreeNode 实例，预加载绑定的资源和用户
	GetAll(ctx context.Context) ([]*model.TreeNode, error)
	// GetAllNoPreload 获取所有 TreeNode 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.TreeNode, error)
	// GetByLevel 根据层级获取 TreeNode 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.TreeNode, error)
	// GetByIDs 根据 IDs 获取 TreeNode 实例，支持分页
	GetByIDs(ctx context.Context, ids []int) ([]*model.TreeNode, error)
	// GetByID 根据 ID 获取单个 TreeNode 实例，预加载相关数据
	GetByID(ctx context.Context, id int) (*model.TreeNode, error)
	// GetByIDNoPreload 根据 ID 获取单个 TreeNode 实例
	GetByIDNoPreload(ctx context.Context, id int) (*model.TreeNode, error)
	// GetByPid 获取指定 TreeNode 的子节点
	GetByPid(ctx context.Context, pid int) ([]*model.TreeNode, error)
	// HasChildren 判断指定的 TreeNode 是否有子节点
	HasChildren(ctx context.Context, id int) (bool, error)
}

type treeNodeDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeNodeDAO(db *gorm.DB, l *zap.Logger) TreeNodeDAO {
	return &treeNodeDAO{
		db: db,
		l:  l,
	}
}

// applyPreloads 应用所有需要的 Preload
func (t *treeNodeDAO) applyPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("BindEcs").
		Preload("BindElb").
		Preload("BindRds").
		Preload("OpsAdmins").
		Preload("RdAdmins").
		Preload("RdMembers")
}

func (t *treeNodeDAO) Create(ctx context.Context, obj *model.TreeNode) error {
	if err := t.db.WithContext(ctx).Create(obj).Error; err != nil {
		t.l.Error("创建树节点失败", zap.Error(err), zap.Any("TreeNode", obj))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Delete(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除关联关系
		if err := tx.Where("id = ?", id).Select(clause.Associations).Delete(&model.TreeNode{}).Error; err != nil {
			return err
		}
		// 再删除节点本身
		if err := tx.Where("id = ?", id).Delete(&model.TreeNode{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("删除树节点失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Upsert(ctx context.Context, obj *model.TreeNode) error {
	// 使用事务确保原子性
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用 Clauses 来实现原子性的 Upsert 操作
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "title"}, {Name: "pid"}},
			UpdateAll: true,
		}).Create(obj).Error; err != nil {
			t.l.Error("Upsert 树节点失败", zap.Error(err), zap.Any("TreeNode", obj))
			return err
		}

		// 更新关联关系
		if err := t.UpdateBindNode(ctx, obj, obj.BindEcs); err != nil {
			return err
		}

		return nil
	})
}

func (t *treeNodeDAO) Update(ctx context.Context, obj *model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新基本字段
		result := tx.Model(&model.TreeNode{}).Where("id = ?", obj.ID).Updates(obj)
		if result.Error != nil {
			t.l.Error("更新树节点失败", zap.Int("id", obj.ID), zap.Error(result.Error))
			return result.Error
		}

		if result.RowsAffected == 0 {
			err := errors.New("没有找到对应的树节点进行更新")
			t.l.Warn("更新树节点未找到目标", zap.Int("id", obj.ID))
			return err
		}

		// 更新关联关系
		if err := tx.Model(&obj).Association("OpsAdmins").Replace(obj.OpsAdmins); err != nil {
			t.l.Error("更新运维负责人失败", zap.Int("id", obj.ID), zap.Error(err))
			return err
		}

		if err := tx.Model(&obj).Association("RdAdmins").Replace(obj.RdAdmins); err != nil {
			t.l.Error("更新研发负责人失败", zap.Int("id", obj.ID), zap.Error(err))
			return err
		}

		if err := tx.Model(&obj).Association("RdMembers").Replace(obj.RdMembers); err != nil {
			t.l.Error("更新研发工程师列表失败", zap.Int("id", obj.ID), zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeNodeDAO) UpdateBindNode(ctx context.Context, obj *model.TreeNode, ecs []*model.ResourceEcs) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(obj).Association("BindEcs").Replace(ecs); err != nil {
			t.l.Error("更新树节点绑定的 Ecs 失败", zap.Int("id", obj.ID), zap.Error(err))
			return err
		}
		return nil
	})
}

func (t *treeNodeDAO) GetAll(ctx context.Context) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.applyPreloads(t.db.WithContext(ctx)).Find(&nodes).Error; err != nil {
		t.l.Error("获取所有树节点失败", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetAllNoPreload(ctx context.Context) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		t.l.Error("获取所有树节点（无预加载）失败", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetByLevel(ctx context.Context, level int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.applyPreloads(t.db.WithContext(ctx)).Where("level = ?", level).Find(&nodes).Error; err != nil {
		t.l.Error("根据层级获取树节点失败", zap.Int("level", level), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetByIDs(ctx context.Context, ids []int) ([]*model.TreeNode, error) {
	if len(ids) == 0 {
		t.l.Info("未提供 IDs，返回空结果")
		return []*model.TreeNode{}, nil
	}

	var nodes []*model.TreeNode
	if err := t.applyPreloads(t.db.WithContext(ctx)).Where("id IN ?", ids).Find(&nodes).Error; err != nil {
		t.l.Error("根据 IDs 获取树节点失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetByID(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode

	if err := t.applyPreloads(t.db.WithContext(ctx)).Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.l.Warn("未找到对应的树节点", zap.Int("id", id))
			return nil, nil
		}
		t.l.Error("根据 ID 获取树节点失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &node, nil
}

func (t *treeNodeDAO) GetByIDNoPreload(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.l.Warn("未找到对应的树节点", zap.Int("id", id))
			return nil, nil
		}
		t.l.Error("根据 ID 获取树节点失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &node, nil
}

func (t *treeNodeDAO) GetByPid(ctx context.Context, pid int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.applyPreloads(t.db.WithContext(ctx)).Where("pid = ?", pid).Find(&nodes).Error; err != nil {
		t.l.Error("根据 pid 获取树节点失败", zap.Int("pid", pid), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) HasChildren(ctx context.Context, id int) (bool, error) {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("pid = ?", id).Count(&count).Error; err != nil {
		t.l.Error("检查子节点失败", zap.Int("id", id), zap.Error(err))
		return false, err
	}

	return count > 0, nil
}
