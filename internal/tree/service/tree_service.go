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
	"fmt"
	"strings"
	"sync"

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
	NodeStatusDeleted  = "deleted"  // 删除状态

	// 默认值
	DefaultLevel  = 1
	DefaultStatus = NodeStatusActive
)

type TreeService interface {
	// 树结构相关接口
	GetTreeList(ctx context.Context, req *model.GetTreeListReq) (model.ListResp[*model.TreeNodeListResp], error)
	GetNodeDetail(ctx context.Context, id int) (*model.TreeNodeDetailResp, error)
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNodeListResp, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error)

	// 节点管理接口
	CreateNode(ctx context.Context, req *model.CreateNodeReq) error
	UpdateNode(ctx context.Context, req *model.UpdateNodeReq) error
	UpdateNodeStatus(ctx context.Context, req *model.UpdateNodeStatusReq) error
	DeleteNode(ctx context.Context, id int) error
	MoveNode(ctx context.Context, nodeId, newParentId int) error

	// 资源绑定接口
	GetNodeResources(ctx context.Context, nodeId int) (model.ListResp[*model.ResourceBase], error)
	BindResource(ctx context.Context, req *model.BindResourceReq) error
	UnbindResource(ctx context.Context, req *model.UnbindResourceReq) error

	// 成员管理接口
	GetNodeMembers(ctx context.Context, nodeId int, memberType string) ([]*model.User, error)
	AddNodeMember(ctx context.Context, req *model.AddNodeMemberReq) error
	RemoveNodeMember(ctx context.Context, req *model.RemoveNodeMemberReq) error
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

// GetTreeList 获取树节点列表
func (t *treeService) GetTreeList(ctx context.Context, req *model.GetTreeListReq) (model.ListResp[*model.TreeNodeListResp], error) {
	if req.Level < 0 {
		return model.ListResp[*model.TreeNodeListResp]{}, errors.New("层级不能为负数")
	}

	t.logger.Debug("获取树节点列表", zap.Int("level", req.Level), zap.String("status", req.Status))

	trees, err := t.dao.GetTreeList(ctx, req)
	if err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return model.ListResp[*model.TreeNodeListResp]{}, err
	}

	// 转换为响应对象
	resp := make([]*model.TreeNodeListResp, 0, len(trees))
	for _, tree := range trees {
		resp = append(resp, convertToTreeNodeListResp(tree))
	}

	return model.ListResp[*model.TreeNodeListResp]{
		Items: resp,
		Total: int64(len(trees)),
	}, nil
}

// GetNodeDetail 获取节点详情
func (t *treeService) GetNodeDetail(ctx context.Context, id int) (*model.TreeNodeDetailResp, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	t.logger.Debug("获取节点详情", zap.Int("id", id))

	// 获取节点基本信息
	node, err := t.dao.GetNode(ctx, id)
	if err != nil {
		t.logger.Error("获取节点详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 并行获取相关信息
	type Result struct {
		AdminUsers    []*model.User
		MemberUsers   []*model.User
		Creator       *model.User
		ResourceCount int
		Err           error
	}

	var wg sync.WaitGroup
	result := &Result{}

	// 获取管理员
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := t.dao.GetNodeMembers(ctx, id, 0, NodeAdminRole)
		if err != nil {
			result.Err = fmt.Errorf("获取管理员失败: %w", err)
			return
		}
		result.AdminUsers = users
	}()

	// 获取成员
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := t.dao.GetNodeMembers(ctx, id, 0, NodeMemberRole)
		if err != nil {
			result.Err = fmt.Errorf("获取成员失败: %w", err)
			return
		}
		result.MemberUsers = users
	}()

	// 获取创建者信息
	if node.CreatorID > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			creator, err := t.userDao.GetUserByID(ctx, node.CreatorID)
			if err != nil {
				t.logger.Warn("获取创建者信息失败", zap.Int("creatorId", node.CreatorID), zap.Error(err))
			} else {
				result.Creator = creator
			}
		}()
	}

	// 获取资源数量
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := t.dao.GetNodeResources(ctx, id)
		if err != nil {
			t.logger.Warn("获取资源数量失败", zap.Int("id", id), zap.Error(err))
		} else {
			result.ResourceCount = len(resources)
		}
	}()

	wg.Wait()

	if result.Err != nil {
		return nil, result.Err
	}

	// 提取用户名
	adminUserNames := make([]string, 0, len(result.AdminUsers))
	for _, user := range result.AdminUsers {
		adminUserNames = append(adminUserNames, user.Username)
	}

	memberUserNames := make([]string, 0, len(result.MemberUsers))
	for _, user := range result.MemberUsers {
		memberUserNames = append(memberUserNames, user.Username)
	}

	creatorName := ""
	if result.Creator != nil {
		creatorName = result.Creator.Username
	}

	return &model.TreeNodeDetailResp{
		TreeNodeResp: model.TreeNodeResp{
			ID:          node.ID,
			Name:        node.Name,
			Description: node.Description,
			ParentID:    node.ParentID,
			ParentName:  node.ParentName,
			Level:       node.Level,
			CreatorName: creatorName,
			CreatorID:   node.CreatorID,
			Status:      node.Status,
			ChildCount:  node.ChildCount,
			IsLeaf:      node.IsLeaf,
			CreatedAt:   node.CreatedAt,
			UpdatedAt:   node.UpdatedAt,
		},
		AdminUsers:    adminUserNames,
		MemberUsers:   memberUserNames,
		ResourceCount: result.ResourceCount,
	}, nil
}

// GetChildNodes 获取子节点列表
func (t *treeService) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNodeListResp, error) {
	if parentId < 0 {
		return nil, errors.New("父节点ID不能为负数")
	}

	t.logger.Debug("获取子节点列表", zap.Int("parentId", parentId))

	children, err := t.dao.GetChildNodes(ctx, parentId)
	if err != nil {
		t.logger.Error("获取子节点列表失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, err
	}

	// 转换为响应对象
	resp := make([]*model.TreeNodeListResp, 0, len(children))
	for _, child := range children {
		resp = append(resp, convertToTreeNodeListResp(child))
	}

	return resp, nil
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

// CreateNode 创建节点
func (t *treeService) CreateNode(ctx context.Context, req *model.CreateNodeReq) error {
	if err := t.validateCreateNodeReq(req); err != nil {
		return err
	}

	// 设置默认值
	if req.Status == "" {
		req.Status = DefaultStatus
	}

	// 计算层级
	level := DefaultLevel
	if req.ParentID != 0 {
		parentLevel, err := t.dao.GetNodeLevel(ctx, req.ParentID)
		if err != nil {
			t.logger.Error("获取父节点层级失败", zap.Int("parentId", req.ParentID), zap.Error(err))
			return fmt.Errorf("获取父节点层级失败: %w", err)
		}
		level = parentLevel + 1
	}

	// 创建节点实体
	node := &model.TreeNode{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		ParentID:    req.ParentID,
		Level:       level,
		Status:      req.Status,
		CreatorID:   req.CreatorID,
		IsLeaf:      req.IsLeaf,
	}

	t.logger.Info("创建节点",
		zap.String("name", node.Name),
		zap.Int("parentId", req.ParentID),
		zap.Int("level", level),
		zap.Int("creatorId", req.CreatorID))

	return t.dao.CreateNode(ctx, node)
}

// UpdateNode 更新节点
func (t *treeService) UpdateNode(ctx context.Context, req *model.UpdateNodeReq) error {
	if err := t.validateUpdateNodeReq(req); err != nil {
		return err
	}

	// 设置默认状态
	if req.Status == "" {
		req.Status = NodeStatusActive
	}

	// 创建更新实体
	node := &model.TreeNode{
		Model:       model.Model{ID: req.ID},
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Status:      req.Status,
		ParentID:    req.ParentID,
	}

	t.logger.Info("更新节点",
		zap.Int("id", req.ID),
		zap.String("name", req.Name),
		zap.String("status", req.Status),
		zap.Int("parentId", req.ParentID))

	return t.dao.UpdateNode(ctx, node)
}

// DeleteNode 删除节点
func (t *treeService) DeleteNode(ctx context.Context, id int) error {
	if err := validateID(id); err != nil {
		return err
	}

	t.logger.Info("删除节点", zap.Int("id", id))

	// 使用事务确保操作的原子性
	return t.dao.Transaction(ctx, func(txCtx context.Context) error {
		// 检查节点是否存在
		node, err := t.dao.GetNode(txCtx, id)
		if err != nil {
			return err
		}

		// 检查节点是否有子节点
		children, err := t.dao.GetChildNodes(txCtx, id)
		if err != nil {
			t.logger.Error("获取子节点失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("获取子节点失败: %w", err)
		}

		if len(children) > 0 {
			t.logger.Warn("节点有子节点，无法删除", zap.Int("id", id), zap.Int("childCount", len(children)))
			return errors.New("节点有子节点，无法删除")
		}

		// 检查节点是否绑定了资源
		resources, err := t.dao.GetNodeResources(txCtx, id)
		if err != nil {
			t.logger.Error("获取节点资源失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("获取节点资源失败: %w", err)
		}

		if len(resources) > 0 {
			t.logger.Warn("节点有绑定资源，无法删除", zap.Int("id", id), zap.Int("resourceCount", len(resources)))
			return errors.New("节点有绑定资源，无法删除")
		}

		// 删除节点
		if err := t.dao.DeleteNode(txCtx, id); err != nil {
			t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除节点失败: %w", err)
		}

		t.logger.Info("节点删除成功", zap.Int("id", id), zap.String("name", node.Name))
		return nil
	})
}

// MoveNode 移动节点
func (t *treeService) MoveNode(ctx context.Context, nodeId, newParentId int) error {
	if err := validateID(nodeId); err != nil {
		return err
	}

	if newParentId < 0 {
		return errors.New("新父节点ID不能为负数")
	}

	if nodeId == newParentId {
		return errors.New("节点不能移动到自己")
	}

	t.logger.Info("移动节点", zap.Int("nodeId", nodeId), zap.Int("newParentId", newParentId))

	// 获取当前节点信息
	node, err := t.dao.GetNode(ctx, nodeId)
	if err != nil {
		return err
	}

	// 创建更新请求
	updateReq := &model.TreeNode{
		Model:       model.Model{ID: nodeId},
		Name:        node.Name,
		Description: node.Description,
		Status:      node.Status,
		ParentID:    newParentId,
	}

	return t.dao.UpdateNode(ctx, updateReq)
}

// GetNodeMembers 获取节点成员列表
func (t *treeService) GetNodeMembers(ctx context.Context, nodeId int, memberType string) ([]*model.User, error) {
	if err := validateID(nodeId); err != nil {
		return nil, err
	}

	if memberType != "" && memberType != NodeAdminRole && memberType != NodeMemberRole && memberType != "all" {
		return nil, errors.New("成员类型只能是admin、member或all")
	}

	t.logger.Debug("获取节点成员", zap.Int("nodeId", nodeId), zap.String("memberType", memberType))

	users, err := t.dao.GetNodeMembers(ctx, nodeId, 0, memberType)
	if err != nil {
		t.logger.Error("获取节点成员失败", zap.Int("nodeId", nodeId), zap.String("memberType", memberType), zap.Error(err))
		return nil, err
	}

	return users, nil
}

// AddNodeMember 添加节点成员
func (t *treeService) AddNodeMember(ctx context.Context, req *model.AddNodeMemberReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.NodeID, req.UserID); err != nil {
		return err
	}

	if req.MemberType != NodeAdminRole && req.MemberType != NodeMemberRole {
		return errors.New("成员类型只能是admin或member")
	}

	t.logger.Info("添加节点成员",
		zap.Int("nodeId", req.NodeID),
		zap.Int("userId", req.UserID),
		zap.String("type", req.MemberType))

	return t.dao.AddNodeMember(ctx, req.NodeID, req.UserID, req.MemberType)
}

// RemoveNodeMember 移除节点成员
func (t *treeService) RemoveNodeMember(ctx context.Context, req *model.RemoveNodeMemberReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.NodeID, req.UserID); err != nil {
		return err
	}

	if req.MemberType != NodeAdminRole && req.MemberType != NodeMemberRole {
		return errors.New("成员类型只能是admin或member")
	}

	t.logger.Info("移除节点成员",
		zap.Int("nodeId", req.NodeID),
		zap.Int("userId", req.UserID),
		zap.String("type", req.MemberType))

	return t.dao.RemoveNodeMember(ctx, req.NodeID, req.UserID, req.MemberType)
}

// UpdateNodeStatus 更新节点状态
func (t *treeService) UpdateNodeStatus(ctx context.Context, req *model.UpdateNodeStatusReq) error {
	return t.dao.UpdateNodeStatus(ctx, req.ID, req.Status)
}

// GetNodeResources 获取节点资源列表
func (t *treeService) GetNodeResources(ctx context.Context, nodeId int) (model.ListResp[*model.ResourceBase], error) {
	if err := validateID(nodeId); err != nil {
		return model.ListResp[*model.ResourceBase]{}, err
	}

	resources, err := t.dao.GetNodeResources(ctx, nodeId)
	if err != nil {
		t.logger.Error("获取节点资源失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return model.ListResp[*model.ResourceBase]{}, err
	}

	return model.ListResp[*model.ResourceBase]{
		Items: resources,
		Total: int64(len(resources)),
	}, nil
}

// BindResource 绑定资源到节点
func (t *treeService) BindResource(ctx context.Context, req *model.BindResourceReq) error {
	if err := validateID(req.NodeID); err != nil {
		return err
	}

	if len(req.ResourceIDs) == 0 {
		return errors.New("资源ID列表不能为空")
	}

	if req.ResourceType == "" {
		return errors.New("资源类型不能为空")
	}

	return t.dao.BindResource(ctx, req.NodeID, req.ResourceType, req.ResourceIDs)
}

// UnbindResource 解绑资源
func (t *treeService) UnbindResource(ctx context.Context, req *model.UnbindResourceReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.NodeID); err != nil {
		return err
	}

	if strings.TrimSpace(req.ResourceType) == "" {
		return errors.New("资源类型不能为空")
	}

	if strings.TrimSpace(req.ResourceID) == "" {
		return errors.New("资源ID不能为空")
	}

	t.logger.Info("解绑资源",
		zap.Int("nodeId", req.NodeID),
		zap.String("resourceType", req.ResourceType),
		zap.String("resourceId", req.ResourceID))

	return t.dao.UnbindResource(ctx, req.NodeID, req.ResourceType, req.ResourceID)
}

// convertToTreeNodeListResp 将数据模型转换为响应模型
func convertToTreeNodeListResp(tree *model.TreeNode) *model.TreeNodeListResp {
	if tree == nil {
		return nil
	}

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
		if childResp := convertToTreeNodeListResp(child); childResp != nil {
			resp.Children = append(resp.Children, childResp)
		}
	}

	return resp
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

// validateCreateNodeReq 验证创建节点请求
func (t *treeService) validateCreateNodeReq(req *model.CreateNodeReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if strings.TrimSpace(req.Name) == "" {
		return errors.New("节点名称不能为空")
	}

	if len(req.Name) > 50 {
		return errors.New("节点名称长度不能超过50个字符")
	}

	if req.ParentID < 0 {
		return errors.New("父节点ID不能为负数")
	}

	if req.Status != "" && req.Status != NodeStatusActive && req.Status != NodeStatusInactive {
		return errors.New("节点状态只能是active或inactive")
	}

	return nil
}

// validateUpdateNodeReq 验证更新节点请求
func (t *treeService) validateUpdateNodeReq(req *model.UpdateNodeReq) error {
	if req == nil {
		return errors.New("请求参数不能为空")
	}

	if err := validateID(req.ID); err != nil {
		return err
	}

	if strings.TrimSpace(req.Name) == "" {
		return errors.New("节点名称不能为空")
	}

	if len(req.Name) > 50 {
		return errors.New("节点名称长度不能超过50个字符")
	}

	if req.ParentID < 0 {
		return errors.New("父节点ID不能为负数")
	}

	if req.ParentID == req.ID {
		return errors.New("节点不能成为自己的父节点")
	}

	if req.Status != "" && req.Status != NodeStatusActive && req.Status != NodeStatusInactive {
		return errors.New("节点状态只能是active或inactive")
	}

	return nil
}
