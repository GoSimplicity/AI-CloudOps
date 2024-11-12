package rds

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

import (
	"context"
	"gorm.io/gorm/clause"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeRdsDAO interface {
	// Create 创建一个新的 ResourceRds 实例
	Create(ctx context.Context, obj *model.ResourceRds) error
	// Delete 删除指定的 ResourceRds 实例（软删除）
	Delete(ctx context.Context, id int) error
	// DeleteByInstanceID 根据 InstanceID 删除 ResourceRds 实例（软删除）
	DeleteByInstanceID(ctx context.Context, instanceID string) error
	// Upsert 创建或更新 ResourceRds 实例
	Upsert(ctx context.Context, obj *model.ResourceRds) error
	// Update 更新指定的 ResourceRds 实例
	Update(ctx context.Context, obj *model.ResourceRds) error
	// UpdateBindNodes 更新 ResourceRds 绑定的 TreeNode 节点
	UpdateBindNodes(ctx context.Context, obj *model.ResourceRds, nodes []*model.TreeNode) error
	// GetAll 获取所有 ResourceRds 实例，预加载绑定的 TreeNodes
	GetAll(ctx context.Context) ([]*model.ResourceRds, error)
	// GetAllNoPreload 获取所有 ResourceRds 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.ResourceRds, error)
	// GetByLevel 根据层级获取 ResourceRds 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.ResourceRds, error)
	// GetByIDsWithPagination 根据 IDs 获取 ResourceRds 实例，支持分页
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceRds, error)
	// GetByInstanceID 根据 InstanceID 获取单个 ResourceRds 实例，预加载绑定的 TreeNodes
	GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceRds, error)
	// GetByID 根据 ID 获取单个 ResourceRds 实例，预加载绑定的 TreeNodes
	GetByID(ctx context.Context, id int) (*model.ResourceRds, error)
	// GetByIDNoPreload 根据 ID 获取单个 ResourceRds 实例, 不预加载任何关联
	GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceRds, error)
	// GetInstanceIDHashMap 获取所有 ResourceRds 的 InstanceID 和 Hash 映射
	GetInstanceIDHashMap(ctx context.Context) (map[string]string, error)
	// AddBindNodes 添加 ResourceRds 绑定的 TreeNode 节点
	AddBindNodes(ctx context.Context, rds *model.ResourceRds, node *model.TreeNode) error
	// RemoveBindNodes 移除 ResourceRds 绑定的 TreeNode 节点
	RemoveBindNodes(ctx context.Context, rds *model.ResourceRds, node *model.TreeNode) error
}

type treeRdsDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeRdsDAO(db *gorm.DB, l *zap.Logger) TreeRdsDAO {
	return &treeRdsDAO{
		db: db,
		l:  l,
	}
}

func (t *treeRdsDAO) applyPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("BindNodes")
}

func (t *treeRdsDAO) Create(ctx context.Context, obj *model.ResourceRds) error {
	if err := t.db.WithContext(ctx).Create(obj).Error; err != nil {
		t.l.Error("创建 Rds 实例失败", zap.Error(err))
		return err
	}
	//如果存在BindNodes，则添加关联
	if len(obj.BindNodes) > 0 {
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, node := range obj.BindNodes {
				if err := t.AddBindNodes(ctx, obj, node); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.l.Error("添加 Rds 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}
	}
	return nil
}

func (t *treeRdsDAO) Delete(ctx context.Context, id int) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//删除关联关系
		if err := tx.Where("id = ?", id).Select(clause.Associations).Delete(&model.ResourceRds{}).Error; err != nil {
			t.l.Error("删除 Rds 关联关系失败", zap.Int("id", id), zap.Error(err))
			return err
		}
		//删除物理资源
		if err := tx.Unscoped().Where("id = ?", id).Delete(&model.ResourceRds{}).Error; err != nil {
			t.l.Error("删除 Rds 物理资源失败", zap.Int("id", id), zap.Error(err))
			return err
		}
		return nil
	})
}

func (t *treeRdsDAO) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	// 删除关联关系
	if err := t.db.WithContext(ctx).Where("instanceID = ?", instanceID).Select(clause.Associations).Delete(&model.ResourceEcs{}).Error; err != nil {
		t.l.Error("删除 ECS 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return err
	}

	// 物理删除资源
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instanceID = ?", instanceID).Unscoped().Delete(&model.ResourceEcs{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("物理删除 ECS 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeRdsDAO) Upsert(ctx context.Context, obj *model.ResourceRds) error {
	//插入或更新资源
	if err := t.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(obj).Error; err != nil {
		t.l.Error("Upsert Rds 失败", zap.Error(err))
		return err
	}
	//更新关联关系
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := t.UpdateBindNodes(ctx, obj, obj.BindNodes); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("更新关联关系失败", zap.Error(err))
		return err
	}
	return nil
}

func (t *treeRdsDAO) Update(ctx context.Context, obj *model.ResourceRds) error {
	//更新资源信息
	if err := t.db.WithContext(ctx).Where("id = ?", obj.ID).Updates(obj).Error; err != nil {
		t.l.Error("更新 Rds 失败", zap.Error(err))
		return err
	}
	if len(obj.BindNodes) > 0 {
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, node := range obj.BindNodes {
				if err := t.AddBindNodes(ctx, obj, node); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.l.Error("添加 Rds 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}
	}
	return nil
}

func (t *treeRdsDAO) UpdateBindNodes(ctx context.Context, obj *model.ResourceRds, nodes []*model.TreeNode) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) GetAll(ctx context.Context) ([]*model.ResourceRds, error) {
	var rds []*model.ResourceRds

	query := t.applyPreloads(t.db.WithContext(ctx))

	if err := query.Find(&rds).Error; err != nil {
		t.l.Error("获取所有 ResourceRds 实例失败", zap.Error(err))
		return nil, err
	}

	return rds, nil
}

func (t *treeRdsDAO) GetAllNoPreload(ctx context.Context) ([]*model.ResourceRds, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceRds, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceRds, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceRds, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) GetByID(ctx context.Context, id int) (*model.ResourceRds, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceRds, error) {
	rds := &model.ResourceRds{}

	if err := t.db.WithContext(ctx).First(&rds, id).Error; err != nil {
		t.l.Error("根据 ID 获取 ResourceRds 实例失败", zap.Error(err))
		return nil, err
	}

	return rds, nil
}

func (t *treeRdsDAO) GetInstanceIDHashMap(ctx context.Context) (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) AddBindNodes(ctx context.Context, rds *model.ResourceRds, node *model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(rds).Association("BindNodes").Append(node); err != nil {
			t.l.Error("添加 ResourceRds 绑定的 TreeNode 节点失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindRds").Append(rds); err != nil {
			t.l.Error("添加 TreeNode 绑定的 ResourceRds 实例失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeRdsDAO) RemoveBindNodes(ctx context.Context, rds *model.ResourceRds, node *model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(rds).Association("BindNodes").Delete(node); err != nil {
			t.l.Error("移除 ResourceRds 绑定的 TreeNode 节点失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindRds").Delete(rds); err != nil {
			t.l.Error("移除 TreeNode 绑定的 ResourceRds 实例失败", zap.Error(err))
			return err
		}

		return nil
	})
}
