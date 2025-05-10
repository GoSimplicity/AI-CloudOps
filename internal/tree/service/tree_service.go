/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package service

import (
	"context"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

const (
	NodeAdminRole      = "admin"    // 管理员角色
	NodeMemberRole     = "member"   // 普通成员角色
	NodeStatusActive   = "active"   // 活跃状态
	NodeStatusInactive = "inactive" // 非活跃状态
)

type TreeService interface {
	// 树结构相关接口
	GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNodeListResp, error)
	GetNodeDetail(ctx context.Context, id int) (*model.TreeNodeDetailResp, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error)

	// 节点管理接口
	CreateNode(ctx context.Context, node *model.TreeNodeCreateReq) error
	UpdateNode(ctx context.Context, node *model.TreeNodeUpdateReq) error
	DeleteNode(ctx context.Context, id int) error

	// 资源绑定接口
	GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error)
	BindResource(ctx context.Context, req *model.TreeNodeResourceBindReq) error
	UnbindResource(ctx context.Context, req *model.TreeNodeResourceUnbindReq) error
	GetResourceTypes(ctx context.Context) ([]string, error)

	// 成员管理接口
	AddNodeMember(ctx context.Context, req *model.TreeNodeMemberReq) error
	RemoveNodeMember(ctx context.Context, req *model.TreeNodeMemberReq) error
}

type treeService struct {
	logger  *zap.Logger
	dao     dao.TreeDAO
	userDao userDao.UserDAO
}

func NewTreeService(logger *zap.Logger, dao dao.TreeDAO, userDao userDao.UserDAO) TreeService {
	return &treeService{
		logger:  logger,
		dao:     dao,
		userDao: userDao,
	}
}

// validateID 验证ID是否有效
func validateID(ids ...int) error {
	for _, id := range ids {
		if id <= 0 {
			return errors.New("ID必须大于0")
		}
	}
	return nil
}

// BindResource 绑定资源到节点
func (t *treeService) BindResource(ctx context.Context, req *model.TreeNodeResourceBindReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.NodeID); err != nil {
		return err
	}

	if req.ResourceType == "" || len(req.ResourceIDs) == 0 {
		return errors.New("资源类型不能为空且至少需要一个资源ID")
	}

	t.logger.Debug("绑定资源",
		zap.Int("nodeId", req.NodeID),
		zap.String("resourceType", req.ResourceType),
		zap.Any("resourceIds", req.ResourceIDs))

	return t.dao.BindResource(ctx, req.NodeID, req.ResourceType, req.ResourceIDs)
}

// CreateNode 创建节点
func (t *treeService) CreateNode(ctx context.Context, req *model.TreeNodeCreateReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if req.Name == "" {
		return errors.New("节点名称不能为空")
	}

	var level int

	if req.ParentID != 0 {
		// 查询父节点的level
		parent, err := t.dao.GetNode(ctx, req.ParentID)
		if err != nil {
			t.logger.Error("获取父节点失败", zap.Int("parentId", req.ParentID), zap.Error(err))
			return err
		}
		level = parent.Level + 1
	}

	// 创建节点实体
	node := &model.TreeNode{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		Level:       level,
		Status:      NodeStatusActive,
		CreatorID:   req.CreatorID,
		IsLeaf:      req.IsLeaf,
	}

	t.logger.Debug("创建节点",
		zap.String("name", node.Name),
		zap.Int("parentId", req.ParentID),
		zap.Int("level", level))

	return t.dao.Transaction(ctx, func(ctx context.Context) error {
		if err := t.dao.CreateNode(ctx, node); err != nil {
			t.logger.Error("创建节点失败",
				zap.String("name", node.Name),
				zap.Int("parentId", req.ParentID),
				zap.Error(err))
			return err
		}
		return nil
	})
}

// DeleteNode 删除节点
func (t *treeService) DeleteNode(ctx context.Context, id int) error {
	if err := validateID(id); err != nil {
		return err
	}

	t.logger.Debug("删除节点", zap.Int("id", id))

	// 使用事务确保操作的原子性
	return t.dao.Transaction(ctx, func(ctx context.Context) error {
		// 检查节点是否绑定资源
		resources, err := t.dao.GetNodeResources(ctx, id)
		if err != nil {
			t.logger.Error("获取节点资源失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 检查当前节点是否有子节点
		children, err := t.dao.GetChildNodes(ctx, id)
		if err != nil {
			t.logger.Error("获取子节点失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		if len(children) > 0 {
			t.logger.Warn("节点有子节点，无法删除", zap.Int("id", id), zap.Int("childCount", len(children)))
			return errors.New("节点有子节点，无法删除")
		}

		if len(resources) > 0 {
			t.logger.Warn("节点有绑定资源，无法删除", zap.Int("id", id), zap.Int("resourceCount", len(resources)))
			return errors.New("节点有绑定资源，无法删除")
		}

		// 删除节点
		if err := t.dao.DeleteNode(ctx, id); err != nil {
			t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		return nil
	})
}

// GetNodeDetail 获取节点详情
func (t *treeService) GetNodeDetail(ctx context.Context, id int) (*model.TreeNodeDetailResp, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	t.logger.Debug("获取节点详情", zap.Int("id", id))

	// 调用DAO获取节点详情
	node, err := t.dao.GetNode(ctx, id)
	if err != nil {
		t.logger.Error("获取节点详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 如果节点不存在，返回错误
	if node == nil {
		return nil, errors.New("节点不存在")
	}

	// 并行获取管理员和成员信息以提高性能
	type memberResult struct {
		users []*model.User
		err   error
		role  string
	}

	ch := make(chan memberResult, 2)

	// 获取管理员
	go func() {
		users, err := t.dao.GetNodeMembers(ctx, id, 0, NodeAdminRole)
		ch <- memberResult{users: users, err: err, role: NodeAdminRole}
	}()

	// 获取成员
	go func() {
		users, err := t.dao.GetNodeMembers(ctx, id, 0, NodeMemberRole)
		ch <- memberResult{users: users, err: err, role: NodeMemberRole}
	}()

	var adminUsers, memberUsers []*model.User
	for i := 0; i < 2; i++ {
		result := <-ch
		if result.err != nil {
			t.logger.Error("获取节点成员失败",
				zap.Int("id", id),
				zap.String("role", result.role),
				zap.Error(result.err))
			return nil, result.err
		}

		if result.role == NodeAdminRole {
			adminUsers = result.users
		} else {
			memberUsers = result.users
		}
	}

	// 提取用户名
	adminUserNames := make([]string, 0, len(adminUsers))
	for _, user := range adminUsers {
		adminUserNames = append(adminUserNames, user.RealName)
	}

	memberUserNames := make([]string, 0, len(memberUsers))
	for _, user := range memberUsers {
		memberUserNames = append(memberUserNames, user.RealName)
	}

	return &model.TreeNodeDetailResp{
		TreeNodeResp: model.TreeNodeResp{
			ID:          node.ID,
			Name:        node.Name,
			Description: node.Description,
			ParentID:    node.ParentID,
			ParentName:  node.ParentName,
			Level:       node.Level,
			CreatorID:   node.CreatorID,
			Status:      node.Status,
			ChildCount:  node.ChildCount,
			CreatedAt:   node.CreatedAt,
			UpdatedAt:   node.UpdatedAt,
		},
		AdminUsers:    adminUserNames,
		MemberUsers:   memberUserNames,
		ResourceCount: node.ResourceCount,
	}, nil
}

// GetNodeResources 获取节点资源列表
func (t *treeService) GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error) {
	if err := validateID(nodeId); err != nil {
		return nil, err
	}

	t.logger.Debug("获取节点资源", zap.Int("nodeId", nodeId))
	resources, err := t.dao.GetNodeResources(ctx, nodeId)
	if err != nil {
		t.logger.Error("获取节点资源失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return nil, err
	}

	return resources, nil
}

// GetResourceTypes 获取资源类型列表
func (t *treeService) GetResourceTypes(ctx context.Context) ([]string, error) {
	t.logger.Debug("获取资源类型列表")
	types, err := t.dao.GetResourceTypes(ctx)
	if err != nil {
		t.logger.Error("获取资源类型列表失败", zap.Error(err))
		return nil, err
	}

	return types, nil
}

// GetTreeList 获取树节点列表
func (t *treeService) GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNodeListResp, error) {
	if req == nil {
		return nil, errors.New("请求参数不能为空")
	}

	if req.Level < 0 {
		return nil, errors.New("层级不允许小于0")
	}

	trees, err := t.dao.GetTreeList(ctx, req)
	if err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, err
	}

	// 转换为响应对象
	resp := make([]*model.TreeNodeListResp, len(trees))
	for i, tree := range trees {
		resp[i] = convertToTreeNodeListResp(tree)
	}

	return resp, nil
}

// convertToTreeNodeListResp 将数据模型转换为响应模型
func convertToTreeNodeListResp(tree *model.TreeNode) *model.TreeNodeListResp {
	resp := &model.TreeNodeListResp{
		ID:        tree.ID,
		CreatedAt: tree.CreatedAt,
		UpdatedAt: tree.UpdatedAt,
		Name:      tree.Name,
		ParentID:  tree.ParentID,
		Level:     tree.Level,
		CreatorID: tree.CreatorID,
		Status:    tree.Status,
		IsLeaf:    tree.IsLeaf,
		Children:  make([]*model.TreeNodeListResp, 0, len(tree.Children)),
	}

	// 递归处理子节点
	for _, child := range tree.Children {
		resp.Children = append(resp.Children, convertToTreeNodeListResp(child))
	}

	return resp
}

// GetTreeStatistics 获取树统计信息
func (t *treeService) GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error) {
	t.logger.Debug("获取树统计信息")
	stats, err := t.dao.GetTreeStatistics(ctx)
	if err != nil {
		t.logger.Error("获取树统计信息失败", zap.Error(err))
		return nil, err
	}

	return stats, nil
}

// AddNodeMember 添加节点成员
func (t *treeService) AddNodeMember(ctx context.Context, req *model.TreeNodeMemberReq) error {
	if err := validateID(req.NodeID, req.UserID); err != nil {
		return err
	}

	return t.dao.AddNodeMember(ctx, req.NodeID, req.UserID, req.Type)
}

// RemoveNodeMember 移除节点成员
func (t *treeService) RemoveNodeMember(ctx context.Context, req *model.TreeNodeMemberReq) error {
	if err := validateID(req.NodeID, req.UserID); err != nil {
		return err
	}

	return t.dao.RemoveNodeMember(ctx, req.NodeID, req.UserID, req.Type)
}

// UnbindResource 解绑资源
func (t *treeService) UnbindResource(ctx context.Context, req *model.TreeNodeResourceUnbindReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.NodeID); err != nil {
		return err
	}

	if req.ResourceType == "" || req.ResourceID == "" {
		return errors.New("资源类型和资源ID不能为空")
	}

	t.logger.Debug("解绑资源",
		zap.Int("nodeId", req.NodeID),
		zap.String("resourceType", req.ResourceType),
		zap.String("resourceId", req.ResourceID))

	return t.dao.UnbindResource(ctx, req.NodeID, req.ResourceType, req.ResourceID)
}

// UpdateNode 更新节点
func (t *treeService) UpdateNode(ctx context.Context, req *model.TreeNodeUpdateReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.ID); err != nil {
		return err
	}

	if req.Name == "" {
		return errors.New("节点名称不能为空")
	}

	// 创建更新实体
	node := &model.TreeNode{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Model: model.Model{
			ID:        req.ID,
			UpdatedAt: time.Now(),
		},
	}

	t.logger.Debug("更新节点",
		zap.Int("id", req.ID),
		zap.String("name", req.Name),
		zap.String("status", req.Status))

	return t.dao.UpdateNode(ctx, node)
}
