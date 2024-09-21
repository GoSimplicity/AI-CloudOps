package elb

import (
	"context"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeElbDAO interface {
	// Create 创建一个新的 ResourceElb 实例
	Create(ctx context.Context, obj *model.ResourceElb) error
	// Delete 删除指定的 ResourceElb 实例（软删除）
	Delete(ctx context.Context, obj *model.ResourceElb) error
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
	// GetUidAndHashMap 获取所有 ResourceElb 的 InstanceID 和 Hash 映射
	GetUidAndHashMap(ctx context.Context) (map[string]string, error)
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
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) Delete(ctx context.Context, obj *model.ResourceElb) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) Upsert(ctx context.Context, obj *model.ResourceElb) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) Update(ctx context.Context, obj *model.ResourceElb) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) UpdateBindNodes(ctx context.Context, obj *model.ResourceElb, nodes []*model.TreeNode) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) GetAll(ctx context.Context) ([]*model.ResourceElb, error) {
	var elb []*model.ResourceElb

	query := t.applyPreloads(t.db.WithContext(ctx))

	if err := query.Find(&elb).Error; err != nil {
		t.l.Error("获取所有 ELB 实例失败", zap.Error(err))
		return nil, err
	}

	return elb, nil
}

func (t *treeElbDAO) GetAllNoPreload(ctx context.Context) ([]*model.ResourceElb, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) GetByLevel(ctx context.Context, level int) ([]*model.ResourceElb, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceElb, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceElb, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) GetByID(ctx context.Context, id int) (*model.ResourceElb, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeElbDAO) GetUidAndHashMap(ctx context.Context) (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}
