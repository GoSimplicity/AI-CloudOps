package resource

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeResourceDAO interface {
	// Create 创建一个新的 ResourceTree 实例
	Create(ctx context.Context, obj *model.ResourceTree) error
	// Delete 删除指定的 ResourceTree 实例（软删除）
	Delete(ctx context.Context, obj *model.ResourceTree) error
	// DeleteByInstanceID 根据 InstanceID 删除 ResourceTree 实例（软删除）
	DeleteByInstanceID(ctx context.Context, instanceID string) error
	// Upsert 创建或更新 ResourceTree 实例
	Upsert(ctx context.Context, obj *model.ResourceTree) error
	// Update 更新指定的 ResourceTree 实例
	Update(ctx context.Context, obj *model.ResourceTree) error
	// UpdateBindNodes 更新 ResourceTree 绑定的 TreeNode 节点
	UpdateBindNodes(ctx context.Context, obj *model.ResourceTree, nodes []*model.TreeNode) error
	// GetAll 获取所有 ResourceTree 实例，预加载绑定的 TreeNodes
	GetAll(ctx context.Context) ([]*model.ResourceTree, error)
	// GetAllNoPreload 获取所有 ResourceTree 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.ResourceTree, error)
	// GetByLevel 根据层级获取 ResourceTree 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.ResourceTree, error)
	// GetByIDsWithPagination 根据 IDs 获取 ResourceTree 实例，支持分页
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceTree, error)
	// GetByInstanceID 根据 InstanceID 获取单个 ResourceTree 实例，预加载绑定的 TreeNodes
	GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceTree, error)
	// GetByID 根据 ID 获取单个 ResourceTree 实例，预加载绑定的 TreeNodes
	GetByID(ctx context.Context, id int) (*model.ResourceTree, error)
	// GetUidAndHashMap 获取所有 ResourceTree 的 InstanceID 和 Hash 映射
	GetUidAndHashMap(ctx context.Context) (map[string]string, error)
}

type treeResourceDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeResourceDAO(db *gorm.DB, l *zap.Logger) TreeResourceDAO {
	return &treeResourceDAO{
		db: db,
		l:  l,
	}
}

func (t *treeResourceDAO) Create(ctx context.Context, obj *model.ResourceTree) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) Delete(ctx context.Context, obj *model.ResourceTree) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) Upsert(ctx context.Context, obj *model.ResourceTree) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) Update(ctx context.Context, obj *model.ResourceTree) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) UpdateBindNodes(ctx context.Context, obj *model.ResourceTree, nodes []*model.TreeNode) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetAll(ctx context.Context) ([]*model.ResourceTree, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetAllNoPreload(ctx context.Context) ([]*model.ResourceTree, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceTree, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceTree, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceTree, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetByID(ctx context.Context, id int) (*model.ResourceTree, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeResourceDAO) GetUidAndHashMap(ctx context.Context) (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}
