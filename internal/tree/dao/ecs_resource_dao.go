package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type TreeEcsResourceDAO interface {
	Create(ctx context.Context, resource *model.ResourceEcs) error
	Delete(ctx context.Context, id int) error
	DeleteByInstanceName(ctx context.Context, name string) error
	Update(ctx context.Context, resource *model.ResourceEcs) error

	UpdateEcsResourceStatusByHash(ctx context.Context, resource *model.ResourceEcs) error
	UpdateByHash(ctx context.Context, resource *model.ResourceEcs) error
	GetByHash(ctx context.Context, hash string) (*model.ResourceEcs, error)

	Upsert(ctx context.Context, resource *model.ResourceEcs) error
	GetAllNoPreload(ctx context.Context) ([]*model.ResourceEcs, error)
	GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceEcs, error)
	GetUidAndHashMap(ctx context.Context) (map[string]string, error)
}

type treeEcsResourceDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewEcsResourceDAO(logger *zap.Logger, db *gorm.DB) TreeEcsResourceDAO {
	return &treeEcsResourceDAO{
		logger: logger,
		db:     db,
	}
}

func (e *treeEcsResourceDAO) Create(ctx context.Context, resource *model.ResourceEcs) error {
	// 创建资源
	if err := e.db.WithContext(ctx).Create(resource).Error; err != nil {
		e.logger.Error("创建 ECS 失败", zap.Error(err))
		return err
	}

	// 如果存在 BindNodes，则添加关联
	if len(resource.BindNodes) > 0 {
		if err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, node := range resource.BindNodes {
				if err := tx.Model(resource).Association("BindNodes").Append(node); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			e.logger.Error("添加 ECS 绑定的 TreeNode 失败", zap.Error(err))
			return err
		}
	}

	return nil
}

func (e *treeEcsResourceDAO) Delete(ctx context.Context, id int) error {
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除关联关系
		if err := tx.Where("id = ?", id).Select(clause.Associations).Delete(&model.ResourceEcs{}).Error; err != nil {
			e.logger.Error("删除 ECS 关联关系失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 物理删除资源
		if err := tx.Unscoped().Where("id = ?", id).Delete(&model.ResourceEcs{}).Error; err != nil {
			e.logger.Error("物理删除 ECS 失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		return nil
	})
}

func (e *treeEcsResourceDAO) DeleteByInstanceName(ctx context.Context, name string) error {
	// 删除关联关系
	if err := e.db.WithContext(ctx).Where("instance_name = ?", name).Select(clause.Associations).Delete(&model.ResourceEcs{}).Error; err != nil {
		e.logger.Error("删除 ECS 失败", zap.String("instance_name", name), zap.Error(err))
		return err
	}

	// 物理删除资源
	if err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_name = ?", name).Unscoped().Delete(&model.ResourceEcs{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		e.logger.Error("物理删除 ECS 失败", zap.String("instance_name", name), zap.Error(err))
		return err
	}

	return nil
}

func (e *treeEcsResourceDAO) UpdateEcsResourceStatusByHash(ctx context.Context, resource *model.ResourceEcs) error {
	if err := e.db.WithContext(ctx).Model(model.ResourceEcs{}).Where("hash = ?", resource.Hash).Update("status", resource.Status).Error; err != nil {
		e.logger.Error("更新 ECS 资源状态失败", zap.Error(err))
		return err
	}

	return nil
}

func (e *treeEcsResourceDAO) Upsert(ctx context.Context, resource *model.ResourceEcs) error {
	// 插入或更新资源
	if err := e.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(resource).Error; err != nil {
		e.logger.Error("Upsert ECS 失败", zap.Error(err))
		return err
	}

	// 更新关联关系
	if err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(resource).Association("BindNodes").Replace(resource.BindNodes); err != nil {
			return err
		}
		return nil
	}); err != nil {
		e.logger.Error("更新关联关系失败", zap.Error(err))
		return err
	}

	return nil
}

func (e *treeEcsResourceDAO) Update(ctx context.Context, resource *model.ResourceEcs) error {
	// 更新资源信息
	if err := e.db.WithContext(ctx).Where("id = ?", resource.ID).Updates(resource).Error; err != nil {
		e.logger.Error("更新 ECS 失败", zap.Error(err))
		return err
	}

	if len(resource.BindNodes) > 0 {
		// 更新关联关系
		if err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(resource).Association("BindNodes").Replace(resource.BindNodes); err != nil {
				return err
			}
			return nil
		}); err != nil {
			e.logger.Error("更新关联关系失败", zap.Error(err))
			return err
		}
	}

	return nil
}

func (e *treeEcsResourceDAO) GetAllNoPreload(ctx context.Context) ([]*model.ResourceEcs, error) {
	var ecs []*model.ResourceEcs

	if err := e.db.WithContext(ctx).Find(&ecs).Error; err != nil {
		e.logger.Error("获取所有 ECS 失败", zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (e *treeEcsResourceDAO) GetByIDNoPreload(ctx context.Context, id int) (*model.ResourceEcs, error) {
	ecs := new(model.ResourceEcs)

	if err := e.db.WithContext(ctx).First(ecs, id).Error; err != nil {
		e.logger.Error("根据 ID 获取 ECS 失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return ecs, nil
}

func (e *treeEcsResourceDAO) GetUidAndHashMap(ctx context.Context) (map[string]string, error) {
	return nil, nil
}

func (e *treeEcsResourceDAO) GetByHash(ctx context.Context, hash string) (*model.ResourceEcs, error) {
	var modelEcs *model.ResourceEcs

	if err := e.db.WithContext(ctx).Where("hash = ?", hash).First(&modelEcs).Error; err != nil {
		e.logger.Error("根据 Hash 获取 ECS 失败", zap.String("hash", hash), zap.Error(err))
		return nil, err
	}

	return modelEcs, nil
}

func (e *treeEcsResourceDAO) UpdateByHash(ctx context.Context, resource *model.ResourceEcs) error {
	// 更新资源信息
	if err := e.db.WithContext(ctx).Where("hash = ?", resource.Hash).Updates(resource).Error; err != nil {
		e.logger.Error("更新 ECS 失败", zap.Error(err))
		return err
	}

	return nil
}
