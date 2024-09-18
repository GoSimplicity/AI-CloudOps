package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
	"golang.org/x/sync/errgroup"
	"strconv"

	"github.com/GoSimplicity/CloudOps/internal/constants"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/ecs"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/elb"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/rds"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/tree_node"
	"go.uber.org/zap"
)

type TreeService interface {
	ListTreeNodes(ctx context.Context) ([]*model.TreeNode, error)
	SelectTreeNode(ctx context.Context, level int, levelLt int) ([]*model.TreeNode, error)
	GetTopTreeNode(ctx context.Context) ([]*model.TreeNode, error)
	ListLeafTreeNodes(ctx context.Context) ([]*model.TreeNode, error)
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
	nodeDao tree_node.TreeNodeDAO
	userDao dao.UserDAO
	l       *zap.Logger
}

// NewTreeService 构造函数
func NewTreeService(ecsDao ecs.TreeEcsDAO, elbDao elb.TreeElbDAO, rdsDao rds.TreeRdsDAO, nodeDao tree_node.TreeNodeDAO, l *zap.Logger, userDao dao.UserDAO) TreeService {
	return &treeService{
		ecsDao:  ecsDao,
		elbDao:  elbDao,
		rdsDao:  rdsDao,
		nodeDao: nodeDao,
		userDao: userDao,
		l:       l,
	}
}

// ListTreeNodes 获取所有树节点并构建树结构
func (ts *treeService) ListTreeNodes(ctx context.Context) ([]*model.TreeNode, error) {
	// 获取所有节点
	nodes, err := ts.nodeDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("ListTreeNodes 获取所有节点失败", zap.Error(err))
		return nil, err
	}

	// 初始化顶层节点切片和父节点到子节点的映射
	var topNodes []*model.TreeNode
	childrenMap := make(map[int][]*model.TreeNode)

	// 遍历所有节点，设置 Key 属性并分类为顶层节点或子节点
	for _, node := range nodes {
		// 设置节点的 Key 属性
		node.Key = strconv.Itoa(node.ID)

		if node.Pid == 0 {
			// 如果父节点 ID 为 0，则为顶层节点，添加到 topNodes 切片中
			topNodes = append(topNodes, node)
		} else {
			// 否则，将节点添加到对应父节点的子节点列表中
			childrenMap[node.Pid] = append(childrenMap[node.Pid], node)
		}
	}

	// 遍历所有节点，将子节点关联到各自的父节点
	for _, node := range nodes {
		if children, exists := childrenMap[node.ID]; exists {
			node.Children = children
		}
	}

	return topNodes, nil
}

func (ts *treeService) SelectTreeNode(ctx context.Context, level int, levelLt int) ([]*model.TreeNode, error) {
	// 从数据库中获取所有树节点
	nodes, err := ts.nodeDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("SelectTreeNode failed to retrieve nodes", zap.Error(err))
		return nil, err
	}

	filteredNodes := make([]*model.TreeNode, 0, len(nodes))

	for _, node := range nodes {
		// 如果指定了具体层级，且节点的层级不匹配，则跳过该节点
		if level > 0 && node.Level != level {
			continue
		}

		// 如果指定了最大层级，且节点的层级超过该值，则跳过该节点
		if levelLt > 0 && node.Level > levelLt {
			continue
		}

		// 设置节点的 Value 属性为其 ID
		node.Value = node.ID

		filteredNodes = append(filteredNodes, node)
	}

	return filteredNodes, nil
}

func (ts *treeService) GetTopTreeNode(ctx context.Context) ([]*model.TreeNode, error) {
	// level = 1, 顶级节点
	nodes, err := ts.nodeDao.GetByLevel(ctx, 1)
	if err != nil {
		ts.l.Error("GetTopTreeNode failed", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (ts *treeService) ListLeafTreeNodes(ctx context.Context) ([]*model.TreeNode, error) {
	// 从数据库中获取所有树节点
	nodes, err := ts.nodeDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("ListLeafTreeNodes 获取所有树节点失败", zap.Error(err))
		return nil, err
	}

	leafNodes := make([]*model.TreeNode, 0, len(nodes))

	// 遍历所有节点，筛选出叶子节点
	for _, node := range nodes {
		// 如果 BindEcs 为空或长度为 0，则不是叶子节点，跳过
		if len(node.BindEcs) == 0 {
			continue
		}
		leafNodes = append(leafNodes, node)
	}

	return leafNodes, nil
}

func (ts *treeService) CreateTreeNode(ctx context.Context, obj *model.TreeNode) error {
	return ts.nodeDao.Create(ctx, obj)
}

func (ts *treeService) DeleteTreeNode(ctx context.Context, id int) error {
	treeNode, err := ts.nodeDao.GetByID(ctx, id)
	if err != nil {
		ts.l.Error("DeleteTreeNode failed", zap.Error(err))
		return err
	}

	if treeNode != nil && len(treeNode.Children) > 0 {
		ts.l.Error("DeleteTreeNode failed", zap.Error(errors.New(constants.ErrorTreeNodeHasChildren)))
		return errors.New(constants.ErrorTreeNodeHasChildren)
	}

	return ts.nodeDao.Delete(ctx, id)
}

// UpdateTreeNode 更新树节点的用户信息
func (ts *treeService) UpdateTreeNode(ctx context.Context, obj *model.TreeNode) error {
	// 使用 errgroup 实现并发处理
	g, ctx := errgroup.WithContext(ctx)

	var (
		usersOpsAdmin []*model.User
		usersRdAdmin  []*model.User
		usersRdMember []*model.User
	)

	g.Go(func() error {
		var err error
		usersOpsAdmin, err = ts.fetchUsers(ctx, obj.OpsAdminUsers.Items, "OpsAdmin")
		return err
	})

	g.Go(func() error {
		var err error
		usersRdAdmin, err = ts.fetchUsers(ctx, obj.RdAdminUsers.Items, "RdAdmin")
		return err
	})

	g.Go(func() error {
		var err error
		usersRdMember, err = ts.fetchUsers(ctx, obj.RdMemberUsers.Items, "RdMember")
		return err
	})

	if err := g.Wait(); err != nil {
		ts.l.Error("UpdateTreeNode 获取用户信息失败", zap.Error(err))
		return err
	}

	// 更新树节点的用户信息
	obj.OpsAdmins = usersOpsAdmin
	obj.RdAdmins = usersRdAdmin
	obj.RdMembers = usersRdMember

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

// fetchUsers 根据用户名列表获取用户，role 用于日志记录
func (ts *treeService) fetchUsers(ctx context.Context, userNames []string, role string) ([]*model.User, error) {
	// 预分配切片容量，减少内存重新分配次数
	users := make([]*model.User, 0, len(userNames))

	for _, userName := range userNames {
		user, err := ts.userDao.GetUserByUsername(ctx, userName)
		if err != nil {
			// 记录具体的错误信息，包括角色和用户名
			ts.l.Error(fmt.Sprintf("UpdateTreeNode 获取 %s 用户失败", role), zap.String("userName", userName), zap.Error(err))
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
