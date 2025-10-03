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
	"strings"

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

type TreeNodeService interface {
	// 树结构相关接口
	GetTreeList(ctx context.Context, req *model.GetTreeNodeListReq) (model.ListResp[*model.TreeNode], error)
	GetNodeDetail(ctx context.Context, id int) (*model.TreeNode, error)
	GetChildNodes(ctx context.Context, parentID int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error)

	// 节点管理接口
	CreateNode(ctx context.Context, req *model.CreateTreeNodeReq) error
	UpdateNode(ctx context.Context, req *model.UpdateTreeNodeReq) error
	DeleteNode(ctx context.Context, id int) error
	MoveNode(ctx context.Context, nodeId, newParentId int) error

	// 资源绑定接口
	BindResource(ctx context.Context, req *model.BindTreeNodeResourceReq) error
	UnbindResource(ctx context.Context, req *model.UnbindTreeNodeResourceReq) error

	// 成员管理接口
	GetNodeMembers(ctx context.Context, nodeId int, memberType string) (model.ListResp[*model.User], error)
	AddNodeMember(ctx context.Context, req *model.AddTreeNodeMemberReq) error
	RemoveNodeMember(ctx context.Context, req *model.RemoveTreeNodeMemberReq) error
}

type treeService struct {
	logger  *zap.Logger
	dao     dao.TreeNodeDAO
	userDao userDao.UserDAO
}

func NewTreeNodeService(logger *zap.Logger, dao dao.TreeNodeDAO, userDao userDao.UserDAO) TreeNodeService {
	return &treeService{
		logger:  logger,
		dao:     dao,
		userDao: userDao,
	}
}

// GetTreeList 获取树节点列表
func (s *treeService) GetTreeList(ctx context.Context, req *model.GetTreeNodeListReq) (model.ListResp[*model.TreeNode], error) {
	if req.Level < 0 {
		return model.ListResp[*model.TreeNode]{}, errors.New("层级不能为负数")
	}

	s.logger.Debug("获取树节点列表", zap.Int("level", req.Level), zap.Int("status", int(req.Status)))

	trees, total, err := s.dao.GetTreeList(ctx, req)
	if err != nil {
		s.logger.Error("获取树节点列表失败", zap.Error(err))
		return model.ListResp[*model.TreeNode]{}, err
	}

	return model.ListResp[*model.TreeNode]{
		Items: trees,
		Total: total,
	}, nil
}

// GetNodeDetail 获取节点详情
func (s *treeService) GetNodeDetail(ctx context.Context, id int) (*model.TreeNode, error) {
	node, err := s.dao.GetNode(ctx, id)
	if err != nil {
		s.logger.Error("获取节点详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return node, nil
}

// GetChildNodes 获取直接子节点
func (s *treeService) GetChildNodes(ctx context.Context, parentID int) ([]*model.TreeNode, error) {
	if parentID < 0 {
		return nil, errors.New("父节点ID无效")
	}
	return s.dao.GetChildNodes(ctx, parentID)
}

// GetTreeStatistics 获取服务树统计信息
func (s *treeService) GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error) {
	return s.dao.GetTreeStatistics(ctx)
}

// CreateNode 创建节点
func (s *treeService) CreateNode(ctx context.Context, req *model.CreateTreeNodeReq) error {
	// 设置默认状态
	status := model.ACTIVE
	if req.Status != 0 {
		status = req.Status
	}

	// 创建节点实体
	node := &model.TreeNode{
		Name:           strings.TrimSpace(req.Name),
		Description:    strings.TrimSpace(req.Description),
		ParentID:       req.ParentID,
		Status:         status,
		IsLeaf:         req.IsLeaf,
		CreateUserID:   req.CreateUserID,
		CreateUserName: req.CreateUserName,
	}

	return s.dao.CreateNode(ctx, node)
}

// UpdateNode 更新节点
func (s *treeService) UpdateNode(ctx context.Context, req *model.UpdateTreeNodeReq) error {
	// 设置默认状态
	status := model.ACTIVE
	if req.Status != 0 {
		status = req.Status
	}

	// 创建更新实体
	node := &model.TreeNode{
		Model:       model.Model{ID: req.ID},
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Status:      status,
		ParentID:    req.ParentID,
		IsLeaf:      req.IsLeaf,
	}

	return s.dao.UpdateNode(ctx, node)
}

// DeleteNode 删除节点
func (s *treeService) DeleteNode(ctx context.Context, id int) error {
	return s.dao.DeleteNode(ctx, id)
}

// MoveNode 移动节点
func (s *treeService) MoveNode(ctx context.Context, nodeId, newParentId int) error {
	if newParentId < 0 {
		return errors.New("新父节点ID不能为负数")
	}

	if nodeId == newParentId {
		return errors.New("节点不能移动到自己")
	}

	s.logger.Info("移动节点", zap.Int("nodeId", nodeId), zap.Int("newParentId", newParentId))

	// 获取当前节点信息
	node, err := s.dao.GetNode(ctx, nodeId)
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

	return s.dao.UpdateNode(ctx, updateReq)
}

// GetNodeMembers 获取节点成员列表
func (s *treeService) GetNodeMembers(ctx context.Context, nodeId int, memberType string) (model.ListResp[*model.User], error) {
	if memberType != "" && memberType != NodeAdminRole && memberType != NodeMemberRole && memberType != "all" {
		return model.ListResp[*model.User]{}, errors.New("成员类型只能是admin、member或all")
	}

	s.logger.Debug("获取节点成员", zap.Int("nodeId", nodeId), zap.String("memberType", memberType))

	users, err := s.dao.GetNodeMembers(ctx, nodeId, memberType)
	if err != nil {
		s.logger.Error("获取节点成员失败", zap.Int("nodeId", nodeId), zap.String("memberType", memberType), zap.Error(err))
		return model.ListResp[*model.User]{}, err
	}

	return model.ListResp[*model.User]{
		Items: users,
		Total: int64(len(users)),
	}, nil
}

// AddNodeMember 添加节点成员
func (s *treeService) AddNodeMember(ctx context.Context, req *model.AddTreeNodeMemberReq) error {
	if req.MemberType != model.AdminRole && req.MemberType != model.MemberRole {
		return errors.New("成员类型只能是admin或member")
	}

	s.logger.Info("添加节点成员",
		zap.Int("nodeId", req.NodeID),
		zap.Int("userId", req.UserID),
		zap.Int8("type", int8(req.MemberType)))

	return s.dao.AddNodeMember(ctx, req.NodeID, req.UserID, req.MemberType)
}

// RemoveNodeMember 移除节点成员
func (s *treeService) RemoveNodeMember(ctx context.Context, req *model.RemoveTreeNodeMemberReq) error {
	if req.MemberType != model.AdminRole && req.MemberType != model.MemberRole {
		return errors.New("成员类型只能是admin或member")
	}

	s.logger.Info("移除节点成员",
		zap.Int("nodeId", req.NodeID),
		zap.Int("userId", req.UserID),
		zap.Int8("type", int8(req.MemberType)))

	return s.dao.RemoveNodeMember(ctx, req.NodeID, req.UserID, req.MemberType)
}

// BindResource 绑定资源到节点
func (s *treeService) BindResource(ctx context.Context, req *model.BindTreeNodeResourceReq) error {
	if len(req.ResourceIDs) == 0 {
		return errors.New("资源ID列表不能为空")
	}

	return s.dao.BindResource(ctx, req.NodeID, req.ResourceIDs)
}

// UnbindResource 解绑资源
func (s *treeService) UnbindResource(ctx context.Context, req *model.UnbindTreeNodeResourceReq) error {
	if req.ResourceID <= 0 {
		return errors.New("资源ID不能为空或小于等于0")
	}

	s.logger.Info("解绑资源",
		zap.Int("nodeId", req.NodeID),
		zap.Int("resourceId", req.ResourceID))

	return s.dao.UnbindResource(ctx, req.NodeID, req.ResourceID)
}
