package tree_node

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TreeNodeDAO interface {
	// Create 创建一个新的 TreeNode 实例
	Create(ctx context.Context, obj *model.TreeNode) error
	// Delete 删除指定的 TreeNode 实例（软删除）
	Delete(ctx context.Context, id int) error
	// Upsert 创建或更新 TreeNode 实例
	Upsert(ctx context.Context, obj *model.TreeNode) error
	// Update 更新指定的 TreeNode 实例
	Update(ctx context.Context, obj *model.TreeNode) error
	// UpdateBindNode 更新 TreeNode 绑定的 ResourceEcs 节点
	UpdateBindNode(ctx context.Context, obj *model.TreeNode, ecs []*model.ResourceEcs) error
	// GetAll 获取所有 TreeNode 实例，预加载绑定的资源和用户
	GetAll(ctx context.Context) ([]*model.TreeNode, error)
	// GetAllNoPreload 获取所有 TreeNode 实例，不预加载任何关联
	GetAllNoPreload(ctx context.Context) ([]*model.TreeNode, error)
	// GetByLevel 根据层级获取 TreeNode 实例，预加载相关数据
	GetByLevel(ctx context.Context, level int) ([]*model.TreeNode, error)
	// GetByIDs 根据 IDs 获取 TreeNode 实例，支持分页
	GetByIDs(ctx context.Context, ids []int) ([]*model.TreeNode, error)
	// GetByID 根据 ID 获取单个 TreeNode 实例，预加载相关数据
	GetByID(ctx context.Context, id int) (*model.TreeNode, error)
	// GetByIDNoPreload 根据 ID 获取单个 TreeNode 实例
	GetByIDNoPreload(ctx context.Context, id int) (*model.TreeNode, error)
	// GetByPid 获取指定 TreeNode 的子节点
	GetByPid(ctx context.Context, pid int) ([]*model.TreeNode, error)
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

// applyPreloads 应用所有需要的 Preload
func (t *treeNodeDAO) applyPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("BindEcs").
		Preload("BindElb").
		Preload("BindRds").
		Preload("OpsAdmins").
		Preload("RdAdmins").
		Preload("RdMembers")
}

func (t *treeNodeDAO) Create(ctx context.Context, obj *model.TreeNode) error {
	if err := t.db.WithContext(ctx).Create(obj).Error; err != nil {
		t.l.Error("创建树节点失败", zap.Error(err), zap.Any("TreeNode", obj))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Delete(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).
		Where("id = ?", id).
		Select(clause.Associations).
		Delete(&model.TreeNode{}).
		Error; err != nil {
		t.l.Error("删除树节点失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Upsert(ctx context.Context, obj *model.TreeNode) error {
	// 使用 Clauses 来实现原子性的 Upsert 操作
	err := t.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "title"}, {Name: "pid"}},
		UpdateAll: true,
	}).Create(obj).Error

	if err != nil {
		t.l.Error("Upsert 树节点失败", zap.Error(err), zap.Any("TreeNode", obj))
		return err
	}

	return nil
}

func (t *treeNodeDAO) Update(ctx context.Context, obj *model.TreeNode) error {
	result := t.db.WithContext(ctx).
		Model(&model.TreeNode{}).
		Where("id = ?", obj.ID).
		Updates(obj)

	if result.Error != nil {
		t.l.Error("更新树节点失败", zap.Int("id", obj.ID), zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		err := errors.New("没有找到对应的树节点进行更新")
		t.l.Warn("更新树节点未找到目标", zap.Int("id", obj.ID))
		return err
	}

	return nil
}

func (t *treeNodeDAO) UpdateBindNode(ctx context.Context, obj *model.TreeNode, ecs []*model.ResourceEcs) error {
	association := t.db.WithContext(ctx).Model(obj).Association("BindEcs")
	if err := association.Replace(ecs); err != nil {
		t.l.Error("更新树节点绑定的 Ecs 失败", zap.Int("id", obj.ID), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeNodeDAO) GetAll(ctx context.Context) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	query := t.applyPreloads(t.db.WithContext(ctx))

	if err := query.Find(&nodes).Error; err != nil {
		t.l.Error("获取所有树节点失败", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetAllNoPreload(ctx context.Context) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if err := t.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		t.l.Error("获取所有树节点（无预加载）失败", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetByLevel(ctx context.Context, level int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("level = ?", level)

	if err := query.Find(&nodes).Error; err != nil {
		t.l.Error("根据层级获取树节点失败", zap.Int("level", level), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetByIDs(ctx context.Context, ids []int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	if len(ids) == 0 {
		t.l.Info("未提供 IDs，返回空结果")
		return []*model.TreeNode{}, nil
	}

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("id IN ?", ids)

	if err := query.Find(&nodes).Error; err != nil {
		t.l.Error("根据 IDs 获取树节点失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (t *treeNodeDAO) GetByID(ctx context.Context, id int) (*model.TreeNode, error) {
	node := &model.TreeNode{}

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("id = ?", id)

	if err := query.First(node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.l.Warn("未找到对应的树节点", zap.Int("id", id))
			return nil, nil
		}
		t.l.Error("根据 ID 获取树节点失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return node, nil
}

func (t *treeNodeDAO) GetByIDNoPreload(ctx context.Context, id int) (*model.TreeNode, error) {
	node := &model.TreeNode{}

	if err := t.db.WithContext(ctx).First(node, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.l.Warn("未找到对应的树节点", zap.Int("id", id))
			return nil, nil
		}
		t.l.Error("根据 ID 获取树节点失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return node, nil
}

func (t *treeNodeDAO) GetByPid(ctx context.Context, pid int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode

	query := t.applyPreloads(t.db.WithContext(ctx)).Where("pid = ?", pid)

	if err := query.Find(&nodes).Error; err != nil {
		t.l.Error("根据 pid 获取树节点失败", zap.Int("pid", pid), zap.Error(err))
		return nil, err
	}

	return nodes, nil
}
