package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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

type TreeEcsDAO interface {
	// 绑定节点相关操作
	UpdateBindNodes(ctx context.Context, resource *model.ResourceEcs, nodes []*model.TreeNode) error
	AddBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error
	RemoveBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error

	// 带预加载的查询操作
	GetAll(ctx context.Context) ([]*model.ResourceEcs, error)
	GetByLevel(ctx context.Context, level int) ([]*model.ResourceEcs, error)
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceEcs, error)
	GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceEcs, error)
	GetByID(ctx context.Context, id int) (*model.ResourceEcs, error)
}

type treeEcsDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeEcsDAO(db *gorm.DB, l *zap.Logger) TreeEcsDAO {
	return &treeEcsDAO{
		db: db,
		l:  l,
	}
}

// 绑定节点相关操作实现
func (t *treeEcsDAO) UpdateBindNodes(ctx context.Context, resource *model.ResourceEcs, nodes []*model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新关联对象的字段
		for _, node := range nodes {
			if err := tx.Model(node).Updates(node).Error; err != nil {
				t.l.Error("更新 TreeNode 字段失败", zap.Error(err))
				return err
			}
		}

		// 同步关联集合
		if err := tx.Model(resource).Association("BindNodes").Replace(nodes); err != nil {
			t.l.Error("同步 ECS 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeEcsDAO) AddBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error {
	// 使用事务更新 ECS 和树节点的关联关系
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(ecs).Association("BindNodes").Append(node); err != nil {
			t.l.Error("添加 ECS 绑定节点失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindEcs").Append(ecs); err != nil {
			t.l.Error("添加节点绑定 ECS 失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeEcsDAO) RemoveBindNodes(ctx context.Context, ecs *model.ResourceEcs, node *model.TreeNode) error {
	// 使用事务更新 ECS 和树节点的关联关系
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(ecs).Association("BindNodes").Delete(node); err != nil {
			t.l.Error("移除 ECS 绑定节点失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindEcs").Delete(ecs); err != nil {
			t.l.Error("移除节点绑定 ECS 失败", zap.Error(err))
			return err
		}

		return nil
	})
}

// 带预加载的查询操作实现
func (t *treeEcsDAO) GetAll(ctx context.Context) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Preload("BindNodes").Find(&ecs).Error; err != nil {
		t.l.Error("获取所有 ECS 失败", zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Preload("BindNodes").Where("level = ?", level).Find(&ecs).Error; err != nil {
		t.l.Error("根据层级获取 ECS 失败", zap.Int("level", level), zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	if err := t.db.WithContext(ctx).Preload("BindNodes").Where("id IN ?", ids).Limit(limit).Offset(offset).Find(&ecs).Error; err != nil {
		t.l.Error("根据 IDs 获取 ECS 失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (t *treeEcsDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceEcs, error) {
	var ecs model.ResourceEcs

	if err := t.db.WithContext(ctx).Preload("BindNodes").Where("instance_id = ?", instanceID).First(&ecs).Error; err != nil {
		t.l.Error("根据 InstanceID 获取 ECS 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return nil, err
	}

	return &ecs, nil
}

func (t *treeEcsDAO) GetByID(ctx context.Context, id int) (*model.ResourceEcs, error) {
	var ecs model.ResourceEcs

	if err := t.db.WithContext(ctx).Preload("BindNodes").Where("id = ?", id).First(&ecs).Error; err != nil {
		t.l.Error("根据 ID 获取 ECS 失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &ecs, nil
}
