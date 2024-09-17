package service

import (
	"context"
	"errors"

	"github.com/GoSimplicity/CloudOps/internal/constants"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/ecs"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/elb"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/node"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/rds"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeService interface {
	ListTreeNodes(ctx context.Context) ([]*model.TreeNode, error)
	SelectTreeNode(ctx context.Context, id int) (*model.TreeNode, error)
	GetTopTreeNode(ctx context.Context) ([]*model.TreeNode, error)
	GetAllTreeNodes(ctx context.Context) ([]*model.TreeNode, error)
	CreateTreeNode(ctx context.Context, obj *model.TreeNode) error
	DeleteTreeNode(ctx context.Context, id int) error
	UpdateTreeNode(ctx context.Context, obj *model.TreeNode) error
	GetChildrenTreeNodes(ctx context.Context, pid int) ([]*model.TreeNode, error)

	GetEcsUnbindList(ctx context.Context) ([]*model.ResourceEcs, error)
	GetEcsList(ctx context.Context) ([]*model.ResourceEcs, error)
	GetElbUnbindList(ctx context.Context) ([]*model.ResourceElb, error)
	GetElbList(ctx context.Context) ([]*model.ResourceElb, error)
	GetRdsUnbindList(ctx context.Context) ([]*model.ResourceRds, error)
	GetAllResources(ctx context.Context) ([]*model.ResourceTree, error)
	GetRdsList(ctx context.Context) ([]*model.ResourceRds, error)

	BindEcs(ctx context.Context, ecsID int, treeNodeID int) error
	BindElb(ctx context.Context, elbID int, treeNodeID int) error
	BindRds(ctx context.Context, rdsID int, treeNodeID int) error
	UnBindEcs(ctx context.Context, ecsID int, treeNodeID int) error
	UnBindElb(ctx context.Context, elbID int, treeNodeID int) error
	UnBindRds(ctx context.Context, rdsID int, treeNodeID int) error
}

type treeService struct {
	ecsDao  ecs.TreeEcsDAO
	elbDao  elb.TreeElbDAO
	rdsDao  rds.TreeRdsDAO
	nodeDao node.TreeNodeDAO
	l       *zap.Logger
}

// NewTreeService 构造函数
func NewTreeService(ecsDao ecs.TreeEcsDAO, elbDao elb.TreeElbDAO, rdsDao rds.TreeRdsDAO, nodeDao node.TreeNodeDAO, l *zap.Logger) TreeService {
	return &treeService{
		ecsDao:  ecsDao,
		elbDao:  elbDao,
		rdsDao:  rdsDao,
		nodeDao: nodeDao,
		l:       l,
	}
}

func (ts *treeService) ListTreeNodes(ctx context.Context) ([]*model.TreeNode, error) {
	// TODO 获取全部TreeNode列表

	// TODO 初始化映射并分类节点

	// TODO 回填Children列表

	// TODO 返回结果

	return nil, nil
}

func (ts *treeService) SelectTreeNode(ctx context.Context, id int) (*model.TreeNode, error) {
	treeNode, err := ts.nodeDao.GetByID(ctx, id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrorTreeNodeNotExist
		}
		ts.l.Error("SelectTreeNode failed", zap.Error(err))
		return nil, err
	}

	return treeNode, nil
}

func (ts *treeService) GetTopTreeNode(ctx context.Context) ([]*model.TreeNode, error) {
	// pid = 1, 顶级节点
	nodes, err := ts.nodeDao.GetByPid(ctx, 1)

	if err != nil {
		ts.l.Error("GetTopTreeNode failed", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (ts *treeService) GetAllTreeNodes(ctx context.Context) ([]*model.TreeNode, error) {
	nodes, err := ts.nodeDao.GetAll(ctx)

	if err != nil {
		ts.l.Error("GetAllTreeNodes failed", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (ts *treeService) CreateTreeNode(ctx context.Context, obj *model.TreeNode) error {
	// TODO 验证权限

	// TODO 执行创建

	// TODO 返回创建结果

	return nil
}

func (ts *treeService) DeleteTreeNode(ctx context.Context, id int) error {
	// TODO 验证权限

	// TODO 判断是否有子节点

	// TODO 执行删除

	// TODO 返回删除结果

	return nil
}

func (ts *treeService) UpdateTreeNode(ctx context.Context, obj *model.TreeNode) error {
	// TODO 验证权限

	// TODO 获取并验证关联用户（运维管理员、研发管理员、研发成员）

	// TODO 更新节点关联用户

	// TODO 返回更新结果

	return nil
}

func (ts *treeService) GetChildrenTreeNodes(ctx context.Context, pid int) ([]*model.TreeNode, error) {
	list, err := ts.nodeDao.GetByPid(ctx, pid)
	if err != nil {
		ts.l.Error("GetChildrenTreeNodes failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (ts *treeService) GetEcsUnbindList(ctx context.Context) ([]*model.ResourceEcs, error) {
	return nil, nil
}

func (ts *treeService) GetEcsList(ctx context.Context) ([]*model.ResourceEcs, error) {
	return nil, nil
}

func (ts *treeService) GetElbUnbindList(ctx context.Context) ([]*model.ResourceElb, error) {
	return nil, nil
}

func (ts *treeService) GetElbList(ctx context.Context) ([]*model.ResourceElb, error) {
	return nil, nil
}

func (ts *treeService) GetRdsUnbindList(ctx context.Context) ([]*model.ResourceRds, error) {
	return nil, nil
}

func (ts *treeService) GetAllResources(ctx context.Context) ([]*model.ResourceTree, error) {
	return nil, nil
}

func (ts *treeService) GetRdsList(ctx context.Context) ([]*model.ResourceRds, error) {
	return nil, nil
}

func (ts *treeService) BindEcs(ctx context.Context, ecsID int, treeNodeID int) error {
	return nil
}

func (ts *treeService) BindElb(ctx context.Context, elbID int, treeNodeID int) error {
	return nil
}

func (ts *treeService) BindRds(ctx context.Context, rdsID int, treeNodeID int) error {
	return nil
}

func (ts *treeService) UnBindEcs(ctx context.Context, ecsID int, treeNodeID int) error {
	return nil
}

func (ts *treeService) UnBindElb(ctx context.Context, elbID int, treeNodeID int) error {
	return nil
}

func (ts *treeService) UnBindRds(ctx context.Context, rdsID int, treeNodeID int) error {
	return nil
}
