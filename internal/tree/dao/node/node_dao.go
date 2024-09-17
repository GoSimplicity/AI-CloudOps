package node

import (
	"context"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeNodeDAO interface {
	// Create 创建一个新的 TreeNode 实例
	Create(ctx context.Context, obj *model.TreeNode) error
	// Delete 删除指定的 TreeNode 实例（软删除）
	Delete(ctx context.Context, obj *model.TreeNode) error
	// DeleteByID 根据 ID 删除 TreeNode 实例（软删除）
	DeleteByID(ctx context.Context, id int) error
	// Upsert 创建或更新 TreeNode 实例
	Upsert(ctx context.Context, obj *model.TreeNode) error
	// Update 更新指定的 TreeNode 实例
	Update(ctx context.Context, obj *model.TreeNode) error
	// UpdateBindEcs 更新 TreeNode 绑定的 ResourceEcs 节点
	UpdateBindEcs(ctx context.Context, obj *model.TreeNode, ecss []*model.ResourceEcs) error
	// UpdateBindElb 更新 TreeNode 绑定的 ResourceElb 节点
	UpdateBindElb(ctx context.Context, obj *model.TreeNode, elbs []*model.ResourceElb) error
	// UpdateBindRds 更新 TreeNode 绑定的 ResourceRds 节点
	UpdateBindRds(ctx context.Context, obj *model.TreeNode, rdss []*model.ResourceRds) error
	// GetAll 获取所有 TreeNode 实例，预加载绑定的资源和用户
	GetAll(ctx context.Context) ([]*model.TreeNode, error)
	// GetAllNoPreload 获取所有 TreeNode 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.TreeNode, error)
	// GetByLevel 根据层级获取 TreeNode 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.TreeNode, error)
	// GetByIDsWithPagination 根据 IDs 获取 TreeNode 实例，支持分页
	GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.TreeNode, error)
	// GetByID 根据 ID 获取单个 TreeNode 实例，预加载相关数据
	GetByID(ctx context.Context, id int) (*model.TreeNode, error)
	// GetIDTitleHashMap 获取所有 TreeNode 的 ID 和 Title 映射
	GetIDTitleHashMap(ctx context.Context) (map[int]string, error)
	// GetTop 获取顶级 TreeNode 实例
	GetTop(ctx context.Context) (*model.TreeNode, error)
	// GetChildren 获取指定 TreeNode 的子节点
	GetChildren(ctx context.Context, pid int) ([]*model.TreeNode, error)
}

type treeNodeDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewTreeNodeDAO(db *gorm.DB, l *zap.Logger) TreeNodeDAO {
	return &treeNodeDAO{
		db: db,
		l:  l,
	}
}

func (t *treeNodeDAO) Create(ctx context.Context, obj *model.TreeNode) error {
	if err := t.db.WithContext(ctx).Create(obj).Error; err != nil {
		t.l.Error("failed to create tree node", zap.Error(err))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Delete(ctx context.Context, obj *model.TreeNode) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) DeleteByID(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).Where("id = ?", id).Delete(&model.TreeNode{}).Error; err != nil {
		t.l.Error("failed to delete tree node by ID", zap.Int("id", id), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Upsert(ctx context.Context, obj *model.TreeNode) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) Update(ctx context.Context, obj *model.TreeNode) error {
	if err := t.db.WithContext(ctx).Model(obj).Where("id = ?", obj.ID).Updates(map[string]interface{}{
		"title":   obj.Title,
		"pid":     obj.Pid,
		"level":   obj.Level,
		"is_leaf": obj.IsLeaf,
		"desc":    obj.Desc,
	}).Error; err != nil {
		t.l.Error("failed to update tree node", zap.Error(err))
		return err
	}

	return nil
}

func (t *treeNodeDAO) UpdateBindEcs(ctx context.Context, obj *model.TreeNode, ecss []*model.ResourceEcs) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) UpdateBindElb(ctx context.Context, obj *model.TreeNode, elbs []*model.ResourceElb) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) UpdateBindRds(ctx context.Context, obj *model.TreeNode, rdss []*model.ResourceRds) error {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) GetAll(ctx context.Context) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		t.l.Error("failed to get all tree nodes", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetAllNoPreload(ctx context.Context) ([]*model.TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) GetByLevel(ctx context.Context, level int) ([]*model.TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) GetByIDsWithPagination(ctx context.Context, ids []int, limit, offset int) ([]*model.TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) GetByID(ctx context.Context, id int) (*model.TreeNode, error) {
	var node *model.TreeNode

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&node).Error; err != nil {
		t.l.Error("failed to get tree node by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return node, nil
}

func (t *treeNodeDAO) GetIDTitleHashMap(ctx context.Context) (map[int]string, error) {
	//TODO implement me
	panic("implement me")
}

func (t *treeNodeDAO) GetTop(ctx context.Context) (*model.TreeNode, error) {
	var node *model.TreeNode

	if err := t.db.WithContext(ctx).First(&node).Error; err != nil {
		t.l.Error("failed to get top tree node", zap.Error(err))
		return nil, err
	}

	return node, nil
}

func (t *treeNodeDAO) GetChildren(ctx context.Context, pid int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.db.WithContext(ctx).Where("pid = ?", pid).Find(&nodes).Error; err != nil {
		t.l.Error("failed to get children tree nodes", zap.Int("pid", pid), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}
