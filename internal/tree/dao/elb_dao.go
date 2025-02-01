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

	"gorm.io/gorm/clause"

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

type TreeElbDAO interface {
	// Create 创建一个新的 ResourceElb 实例
	Create(ctx context.Context, obj *model.ResourceElb) error
	// Delete 删除指定的 ResourceElb 实例（软删除）
	Delete(ctx context.Context, id int) error
	// DeleteByInstanceID 根据 InstanceID 删除 ResourceElb 实例（软删除）
	DeleteByInstanceID(ctx context.Context, instanceID string) error
	// Upsert 创建或更新 ResourceElb 实例
	Upsert(ctx context.Context, obj *model.ResourceElb) error
	// Update 更新指定的 ResourceElb 实例
	Update(ctx context.Context, obj *model.ResourceElb) error
	// UpdateBindNodes 更新 ResourceElb 绑定的 TreeNode 节点
	UpdateBindNodes(ctx context.Context, obj *model.ResourceElb, nodes []*model.TreeNode) error
	// GetAll 获取所有 ResourceElb 实例，预加载绑定的 TreeNodes
	GetAll(ctx context.Context) ([]*model.ResourceElb, error)
	// GetAllNoPreload 获取所有 ResourceElb 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.ResourceElb, error)
	// GetByLevel 根据层级获取 ResourceElb 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.ResourceElb, error)
	// GetByIDsWithPagination 根据 IDs 获取 ResourceElb 实例，支持分页
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceElb, error)
	// GetByInstanceID 根据 InstanceID 获取单个 ResourceElb 实例，预加载绑定的 TreeNodes
	GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceElb, error)
	// GetByID 根据 ID 获取单个 ResourceElb 实例，预加载绑定的 TreeNodes
	GetByID(ctx context.Context, id int) (*model.ResourceElb, error)
	// GetByIDNoPreload 根据 ID 获取单个 ResourceElb 实例，不预加载任何关联
	GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceElb, error)
	// GetUidAndHashMap 获取所有 ResourceElb 的 InstanceID 和 Hash 映射
	GetUidAndHashMap(ctx context.Context) (map[string]string, error)
	// AddBindNodes 添加 ResourceElb 绑定的 TreeNode 节点
	AddBindNodes(ctx context.Context, elb *model.ResourceElb, node *model.TreeNode) error
	// RemoveBindNodes 移除 ResourceElb 绑定的 TreeNode 节点
	RemoveBindNodes(ctx context.Context, elb *model.ResourceElb, node *model.TreeNode) error
}

type treeElbDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeElbDAO(db *gorm.DB, l *zap.Logger) TreeElbDAO {
	return &treeElbDAO{
		db: db,
		l:  l,
	}
}

func (t *treeElbDAO) applyPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("BindNodes")
}

func (t *treeElbDAO) Create(ctx context.Context, obj *model.ResourceElb) error {
	//创建资源
	if err := t.db.WithContext(ctx).Create(obj).Error; err != nil {
		t.l.Error("创建 ELB 失败", zap.Error(err))
		return err
	}

	//如果存在BindNodes,则添加关联
	if len(obj.BindNodes) > 0 {
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, node := range obj.BindNodes {
				if err := t.AddBindNodes(ctx, obj, node); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.l.Error("添加 ELB 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}
	}
	return nil
}

func (t *treeElbDAO) Delete(ctx context.Context, id int) error {
	//删除关联关系
	if err := t.db.WithContext(ctx).Where("id = ?", id).Select(clause.Associations).Delete(&model.ResourceElb{}).Error; err != nil {
		t.l.Error("删除 ELB 失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	//物理删除资源
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Unscoped().Delete(&model.ResourceElb{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("物理删除 ELB 失败", zap.Int("id", id), zap.Error(err))
		return err
	}
	return nil
}

func (t *treeElbDAO) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	//删除关联关系
	if err := t.db.WithContext(ctx).Where("instance_id = ?", instanceID).Select(clause.Associations).Delete(&model.ResourceElb{}).Error; err != nil {
		t.l.Error("根据 InstanceID 删除 ELB 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return err
	}

	//物理删除资源
	if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_id = ?", instanceID).Unscoped().Delete(&model.ResourceElb{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.l.Error("物理删除 ELB 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return err
	}
	return nil
}

func (t *treeElbDAO) Upsert(ctx context.Context, obj *model.ResourceElb) error {
	//插入或者更新资源
	if err := t.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(obj).Error; err != nil {
		t.l.Error("Upsert ELB 失败", zap.Error(err))
		return err
	}

	//更新关联关系
	if len(obj.BindNodes) > 0 {
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := t.UpdateBindNodes(ctx, obj, obj.BindNodes); err != nil {
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

func (t *treeElbDAO) Update(ctx context.Context, obj *model.ResourceElb) error {
	if err := t.db.WithContext(ctx).Updates(obj).Error; err != nil {
		t.l.Error("更新 ELB 失败", zap.Error(err))
		return err
	}

	//更新关联关系
	if len(obj.BindNodes) > 0 {
		if err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := t.UpdateBindNodes(ctx, obj, obj.BindNodes); err != nil {
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

func (t *treeElbDAO) UpdateBindNodes(ctx context.Context, obj *model.ResourceElb, nodes []*model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新关联对象的字段
		for _, node := range nodes {
			if err := tx.Model(node).Updates(node).Error; err != nil {
				t.l.Error("更新 TreeNode 字段失败", zap.Error(err))
				return err
			}
		}

		// 同步关联集合
		if err := tx.Model(obj).Association("BindNodes").Replace(nodes); err != nil {
			t.l.Error("同步 ELB 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (t *treeElbDAO) GetAll(ctx context.Context) ([]*model.ResourceElb, error) {
	var elb []*model.ResourceElb

	query := t.applyPreloads(t.db.WithContext(ctx))

	if err := query.Find(&elb).Error; err != nil {
		t.l.Error("获取所有 ELB 失败", zap.Error(err))
		return nil, err
	}

	return elb, nil
}

func (t *treeElbDAO) GetAllNoPreload(ctx context.Context) ([]*model.ResourceElb, error) {
	var elb []*model.ResourceElb

	if err := t.db.WithContext(ctx).Find(&elb).Error; err != nil {
		t.l.Error("获取所有 ELB 失败", zap.Error(err))
		return nil, err
	}
	return elb, nil
}

func (t *treeElbDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceElb, error) {
	var elb []*model.ResourceElb

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("level = ?", level)

	if err := query.Find(&elb).Error; err != nil {
		t.l.Error("根据层级获取 ELB 失败", zap.Int("level", level), zap.Error(err))
		return nil, err
	}
	return elb, nil
}

func (t *treeElbDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceElb, error) {
	var elb []*model.ResourceElb

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("id IN ?", ids).Limit(limit).Offset(offset)

	if err := query.Find(&elb).Error; err != nil {
		t.l.Error("根据 IDs 获取 ELB 失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}
	return elb, nil
}

func (t *treeElbDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceElb, error) {
	var elb model.ResourceElb

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("instance_id = ?", instanceID)

	if err := query.First(&elb).Error; err != nil {
		t.l.Error("根据 InstanceID 获取 ELB 失败", zap.String("instanceID", instanceID), zap.Error(err))
		return nil, err
	}
	return &elb, nil
}

func (t *treeElbDAO) GetByID(ctx context.Context, id int) (*model.ResourceElb, error) {
	var elb model.ResourceElb

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("id = ?", id)

	if err := query.First(&elb).Error; err != nil {
		t.l.Error("根据 ID 获取 ELB 失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	return &elb, nil
}

func (t *treeElbDAO) GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceElb, error) {
	var elb model.ResourceElb

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&elb).Error; err != nil {
		t.l.Error("根据ID获取 ELB 失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &elb, nil
}

func (t *treeElbDAO) GetUidAndHashMap(ctx context.Context) (map[string]string, error) {
	return nil, nil
}

func (t *treeElbDAO) AddBindNodes(ctx context.Context, elb *model.ResourceElb, node *model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(elb).Association("BindNodes").Append(node); err != nil {
			t.l.Error("添加 ELB 绑定节点失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindElb").Append(elb); err != nil {
			t.l.Error("添加节点绑定 ELB 失败", zap.Error(err))
			return err
		}
		return nil
	})
}

func (t *treeElbDAO) RemoveBindNodes(ctx context.Context, elb *model.ResourceElb, node *model.TreeNode) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(elb).Association("BindNodes").Delete(node); err != nil {
			t.l.Error("移除 ELB 绑定节点失败", zap.Error(err))
			return err
		}

		if err := tx.Model(node).Association("BindElb").Delete(elb); err != nil {
			t.l.Error("移除节点绑定 ELB 失败", zap.Error(err))
			return err
		}
		return nil
	})
}
