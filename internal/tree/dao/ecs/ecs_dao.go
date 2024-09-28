package ecs

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeEcsDAO interface {
	// Create 创建一个新的 ResourceEcs 实例
	Create(ctx context.Context, obj *model.ResourceEcs) error
	// Delete 删除指定的 ResourceEcs 实例（软删除）
	Delete(ctx context.Context, obj *model.ResourceEcs) error
	// DeleteByInstanceID 根据 InstanceID 删除 ResourceEcs 实例（软删除）
	DeleteByInstanceID(ctx context.Context, instanceID string) error
	// Upsert 创建或更新 ResourceEcs 实例
	Upsert(ctx context.Context, obj *model.ResourceEcs) error
	// Update 更新指定的 ResourceEcs 实例
	Update(ctx context.Context, obj *model.ResourceEcs) error
	// UpdateBindNodes 更新 ResourceEcs 绑定的 TreeNode 节点
	UpdateBindNodes(ctx context.Context, obj *model.ResourceEcs, nodes []*model.TreeNode) error
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

func (t *treeEcsDAO) applyPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("BindNodes")
}

func (t *treeEcsDAO) Create(ctx context.Context, obj *model.ResourceEcs) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) Delete(ctx context.Context, obj *model.ResourceEcs) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) Upsert(ctx context.Context, obj *model.ResourceEcs) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) Update(ctx context.Context, obj *model.ResourceEcs) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) UpdateBindNodes(ctx context.Context, obj *model.ResourceEcs, nodes []*model.TreeNode) error {
	//TODO implement me
	panic("implement me")
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
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceEcs, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceEcs, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceEcs, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeEcsDAO) GetByID(ctx context.Context, id int) (*model.ResourceEcs, error) {
	//TODO implement me
	panic("implement me")
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
	//TODO implement me
	panic("implement me")
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
