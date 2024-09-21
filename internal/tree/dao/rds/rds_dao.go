package rds

import (
	"context"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeRdsDAO interface {
	// Create 创建一个新的 ResourceRds 实例
	Create(ctx context.Context, obj *model.ResourceRds) error
	// Delete 删除指定的 ResourceRds 实例（软删除）
	Delete(ctx context.Context, obj *model.ResourceRds) error
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
	// GetInstanceIDHashMap 获取所有 ResourceRds 的 InstanceID 和 Hash 映射
	GetInstanceIDHashMap(ctx context.Context) (map[string]string, error)
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
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) Delete(ctx context.Context, obj *model.ResourceRds) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) Upsert(ctx context.Context, obj *model.ResourceRds) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeRdsDAO) Update(ctx context.Context, obj *model.ResourceRds) error {
	//TODO implement me
	panic("implement me")
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

func (t *treeRdsDAO) GetInstanceIDHashMap(ctx context.Context) (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}
