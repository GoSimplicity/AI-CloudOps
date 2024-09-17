package ecs

import (
	"context"
	"errors"

	"github.com/GoSimplicity/CloudOps/internal/constants"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/go-sql-driver/mysql"
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
	// GetUidAndHashMap 获取所有 ResourceEcs 的 InstanceID 和 Hash 映射
	GetUidAndHashMap(ctx context.Context) (map[string]string, error)
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

func (t *treeEcsDAO) Create(ctx context.Context, obj *model.ResourceEcs) error {
	if err := t.db.WithContext(ctx).Create(obj).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == constants.ErrCodeDuplicate {
			return constants.ErrorResourceEcsExist
		}
		t.l.Error("create resource ecs failed", zap.Error(err))
		return err
	}
	return nil
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
	//TODO implement me
	panic("implement me")
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

func (t *treeEcsDAO) GetUidAndHashMap(ctx context.Context) (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}
