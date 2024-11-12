package ecs

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

type TreeEcsDAO interface {
	// Create 创建一个新的 ResourceEcs 实例
	Create(ctx context.Context, resource *model.ResourceEcs) error
	// Delete 删除指定的 ResourceEcs 实例（软删除）
	Delete(ctx context.Context, id int) error
	// DeleteByInstanceName 根据 InstanceName 删除 ResourceEcs 实例（软删除）
	DeleteByInstanceName(ctx context.Context, name string) error
	// Upsert 创建或更新 ResourceEcs 实例
	Upsert(ctx context.Context, resource *model.ResourceEcs) error
	// Update 更新指定的 ResourceEcs 实例
	Update(ctx context.Context, resource *model.ResourceEcs) error
	// UpdateEcsResourceStatusByHash 更新 ECS 资源的状态
	UpdateEcsResourceStatusByHash(ctx context.Context, resource *model.ResourceEcs) error
	// UpdateByHash 通过 Hash 更新 ResourceEcs 实例
	UpdateByHash(ctx context.Context, resource *model.ResourceEcs) error
	// UpdateBindNodes 更新 ResourceEcs 绑定的 TreeNode 节点
	UpdateBindNodes(ctx context.Context, resource *model.ResourceEcs, nodes []*model.TreeNode) error
	// GetAll 获取所有 ResourceEcs 实例，预加载绑定的 TreeNodes
	GetAll(ctx context.Context) ([]*model.ResourceEcs, error)
	// GetAllNoPreload 获取所有 ResourceEcs 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.ResourceEcs, error)
	// GetByLevel 根据层级获取 ResourceEcs 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.ResourceEcs, error)
	// GetByIDsWithPagination 根据 IDs 获取 ResourceEcs 实例，支持分页
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceEcs, error)
	// GetByInstanceID 根据 InstanceID 获取单个 ResourceEcs 实例，预加载绑定的 TreeNodes
	GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceEcs, error)
	// GetByID 根据 ID 获取单个 ResourceEcs 实例，预加载绑定的 TreeNodes
	GetByID(ctx context.Context, id int) (*model.ResourceEcs, error)
	// GetByIDNoPreload 根据 ID 获取单个 ResourceEcs 实例
	GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceEcs, error)
	// GetUidAndHashMap 获取所有 ResourceEcs 的 InstanceID 和 Hash 映射
	GetUidAndHashMap(ctx context.Context) (map[string]string, error)
	// AddBindNodes 添加 ResourceEcs 绑定的 TreeNode 节点
	AddBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error
	// RemoveBindNodes 移除 ResourceEcs 绑定的 TreeNode 节点
	RemoveBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error
	// GetByHash 根据 Hash 获取单个 ResourceEcs 实例
	GetByHash(ctx context.Context, hash string) (*model.ResourceEcs, error)
}

type treeEcsDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func (t *treeEcsDAO) UpdateEcsResourceStatusByHash(ctx context.Context, resource *model.ResourceEcs) error {
	if err := t.db.WithContext(ctx).Model(model.ResourceEcs{}).Where("hash = ?", resource.Hash).Update("status", resource.Status).Error; err != nil {
		t.l.Error("更新 ECS 资源状态失败", zap.Error(err))
		return err
	}

	return nil
}

func NewTreeEcsDAO(db *gorm.DB, l *zap.Logger) TreeEcsDAO {
	return &treeEcsDAO{
		db: db,
		l:  l,
	}
}

func (t *treeEcsDAO) applyPreloads(query *gorm.DB) *gorm.DB {
	return query.Preload("BindNodes")
}

func (t *treeEcsDAO) Create(ctx context.Context, resource *model.ResourceEcs) error {
	// 创建资源
	if err := t.db.WithContext(ctx).Create(resource).Error; err != nil {
		t.l.Error("创建 ECS 失败", zap.Error(err))
		return err
	}

	// 如果存在 BindNodes，则添加关联
	if len(resource.BindNodes) > 0 {
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, node := range resource.BindNodes {
				if err := t.AddBindNodes(ctx, resource, node); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.l.Error("添加 ECS 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}
	}

	return nil
}

func (t *treeEcsDAO) Delete(ctx context.Context, id int) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除关联关系
		if err := tx.Where("id = ?", id).Select(clause.Associations).Delete(&model.ResourceEcs{}).Error; err != nil {
			t.l.Error("删除 ECS 关联关系失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 物理删除资源
		if err := tx.Unscoped().Where("id = ?", id).Delete(&model.ResourceEcs{}).Error; err != nil {
			t.l.Error("物理删除 ECS 失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeEcsDAO) DeleteByInstanceName(ctx context.Context, name string) error {
	// 删除关联关系
	if err := t.db.WithContext(ctx).Where("instance_name = ?", name).Select(clause.Associations).Delete(&model.ResourceEcs{}).Error; err != nil {
		t.l.Error("删除 ECS 失败", zap.String("instance_name", name), zap.Error(err))
		return err
	}

	// 物理删除资源
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_name = ?", name).Unscoped().Delete(&model.ResourceEcs{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("物理删除 ECS 失败", zap.String("instance_name", name), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeEcsDAO) Upsert(ctx context.Context, resource *model.ResourceEcs) error {
	// 插入或更新资源
	if err := t.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(resource).Error; err != nil {
		t.l.Error("Upsert ECS 失败", zap.Error(err))
		return err
	}

	// 更新关联关系
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := t.UpdateBindNodes(ctx, resource, resource.BindNodes); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("更新关联关系失败", zap.Error(err))
		return err
	}

	return nil
}

func (t *treeEcsDAO) Update(ctx context.Context, resource *model.ResourceEcs) error {
	// 更新资源信息
	if err := t.db.WithContext(ctx).Where("id = ?", resource.ID).Updates(resource).Error; err != nil {
		t.l.Error("更新 ECS 失败", zap.Error(err))
		return err
	}

	if len(resource.BindNodes) > 0 {
		// 更新关联关系
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := t.UpdateBindNodes(ctx, resource, resource.BindNodes); err != nil {
				return err
			}
			return nil
		}); err != nil {
			t.l.Error("更新关联关系失败", zap.Error(err))
			return err
		}
	}

	return nil
}

func (t *treeEcsDAO) UpdateBindNodes(ctx context.Context, resource *model.ResourceEcs, nodes []*model.TreeNode) error {
	// 更新关联对象的字段
	for _, node := range nodes {
		if err := t.db.WithContext(ctx).Model(node).Updates(node).Error; err != nil {
			t.l.Error("更新 TreeNode 字段失败", zap.Error(err))
			return err
		}
	}

	// 同步关联集合
	if err := t.db.WithContext(ctx).Model(resource).Association("BindNodes").Replace(nodes); err != nil {
		t.l.Error("同步 ECS 绑定的 TreeNode 失败", zap.Error(err))
		return err
	}

	return nil
}

func (t *treeEcsDAO) GetAll(ctx context.Context) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	query := t.applyPreloads(t.db.WithContext(ctx))

	if err := query.Find(&ecs).Error; err != nil {
		t.l.Error("获取所有 ECS 失败", zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetAllNoPreload(ctx context.Context) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Find(&ecs).Error; err != nil {
		t.l.Error("获取所有 ECS 失败", zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("level = ?", level)

	if err := query.Find(&ecs).Error; err != nil {
		t.l.Error("根据层级获取 ECS 失败", zap.Int("level", level), zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("id IN ?", ids).Limit(limit).Offset(offset)

	if err := query.Find(&ecs).Error; err != nil {
		t.l.Error("根据 IDs 获取 ECS 失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceEcs, error) {
	var ecs model.ResourceEcs

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("instance_id = ?", instanceID)

	if err := query.First(&ecs).Error; err != nil {
		t.l.Error("根据 InstanceID 获取 ECS 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return nil, err
	}

	return &ecs, nil
}

func (t *treeEcsDAO) GetByID(ctx context.Context, id int) (*model.ResourceEcs, error) {
	var ecs model.ResourceEcs

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("id = ?", id)

	if err := query.First(&ecs).Error; err != nil {
		t.l.Error("根据 ID 获取 ECS 失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &ecs, nil
}

func (t *treeEcsDAO) GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceEcs, error) {
	ecs := new(model.ResourceEcs)

	if err := t.db.WithContext(ctx).First(ecs, id).Error; err != nil {
		t.l.Error("根据 ID 获取 ECS 失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetUidAndHashMap(ctx context.Context) (map[string]string, error) {
	return nil, nil
}

func (t *treeEcsDAO) AddBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error {
	// 使用事务更新 ECS 和树节点的关联关系
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(ecs).Association("BindNodes").Append(node); err != nil {
			t.l.Error("BindEcs 更新 ECS 失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindEcs").Append(ecs); err != nil {
			t.l.Error("BindEcs 更新树节点失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeEcsDAO) RemoveBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error {
	// 使用事务更新 ECS 和树节点的关联关系
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(ecs).Association("BindNodes").Delete(node); err != nil {
			t.l.Error("BindEcs 更新 ECS 失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindEcs").Delete(ecs); err != nil {
			t.l.Error("BindEcs 更新树节点失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeEcsDAO) GetByHash(ctx context.Context, hash string) (*model.ResourceEcs, error) {
	var modelEcs *model.ResourceEcs

	if err := t.db.WithContext(ctx).Where("hash = ?", hash).First(&modelEcs).Error; err != nil {
		t.l.Error("根据 Hash 获取 ECS 失败", zap.String("hash", hash), zap.Error(err))
		return nil, err
	}

	return modelEcs, nil
}

func (t *treeEcsDAO) UpdateByHash(ctx context.Context, resource *model.ResourceEcs) error {
	// 更新资源信息
	if err := t.db.WithContext(ctx).Where("hash = ?", resource.Hash).Updates(resource).Error; err != nil {
		t.l.Error("更新 ECS 失败", zap.Error(err))
		return err
	}

	return nil
}
