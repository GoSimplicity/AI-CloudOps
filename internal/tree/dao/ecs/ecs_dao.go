package dao

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeDAO interface {
	// Create 创建一个新的 ResourceEcs 实例。
	Create(ctx context.Context, obj *model.ResourceEcs) error
	// Delete 删除指定的 ResourceEcs 实例（软删除）。
	Delete(ctx context.Context, obj *model.ResourceEcs) error
	// DeleteByInstanceID 根据 InstanceID 删除 ResourceEcs 实例（软删除）。
	DeleteByInstanceID(ctx context.Context, instanceID string) error
	// Upsert 创建或更新 ResourceEcs 实例。
	Upsert(ctx context.Context, obj *model.ResourceEcs) error
	// Update 更新指定的 ResourceEcs 实例。
	Update(ctx context.Context, obj *model.ResourceEcs) error
	// UpdateBindNodes 更新 ResourceEcs 绑定的 TreeNode 节点。
	UpdateBindNodes(ctx context.Context, obj *model.ResourceEcs, nodes []*model.TreeNode) error
	// GetAll 获取所有 ResourceEcs 实例，预加载绑定的 TreeNodes。
	GetAll(ctx context.Context) ([]*model.ResourceEcs, error)
	// GetAllNoPreload 获取所有 ResourceEcs 实例，不预加载。
	GetAllNoPreload(ctx context.Context) ([]*model.ResourceEcs, error)
	// GetByLevel 根据层级获取 ResourceEcs 实例，预加载运维负责人。
	GetByLevel(ctx context.Context, level int) ([]*model.ResourceEcs, error)
	// GetByIDsWithPagination 根据 IDs 获取 ResourceEcs 实例，支持分页。
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.ResourceEcs, error)
	// GetByInstanceID 根据 InstanceID 获取单个 ResourceEcs 实例，预加载绑定的 TreeNodes。
	GetByInstanceID(ctx context.Context, instanceID string) (*model.ResourceEcs, error)
	// GetByID 根据 ID 获取单个 ResourceEcs 实例，预加载绑定的 TreeNodes。
	GetByID(ctx context.Context, id int) (*model.ResourceEcs, error)
	// GetInstanceIDHashMap 获取所有 ResourceEcs 的 InstanceID 和 Hash 映射。
	GetInstanceIDHashMap(ctx context.Context) (map[string]string, error)
}

type treeDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeDAO(db *gorm.DB, l *zap.Logger) TreeDAO {
	return &treeDAO{
		db: db,
		l:  l,
	}
}
