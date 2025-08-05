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
	"strconv"
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
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error)

	// 节点管理接口
	CreateNode(ctx context.Context, req *model.CreateTreeNodeReq) error
	UpdateNode(ctx context.Context, req *model.UpdateTreeNodeReq) error
	UpdateNodeStatus(ctx context.Context, req *model.UpdateTreeNodeStatusReq) error
	DeleteNode(ctx context.Context, id int) error
	MoveNode(ctx context.Context, nodeId, newParentId int) error

	// 资源绑定接口
	GetNodeResources(ctx context.Context, nodeId int) (model.ListResp[*model.ResourceItems], error)
	BindResource(ctx context.Context, req *model.BindTreeNodeResourceReq) error
	UnbindResource(ctx context.Context, req *model.UnbindTreeNodeResourceReq) error

	// 成员管理接口
	GetNodeMembers(ctx context.Context, nodeId int, memberType string) ([]*model.User, error)
	AddNodeMember(ctx context.Context, req *model.AddTreeNodeMemberReq) error
	RemoveNodeMember(ctx context.Context, req *model.RemoveTreeNodeMemberReq) error
}

type treeService struct {
	logger   *zap.Logger
	dao      dao.TreeNodeDAO
	userDao  userDao.UserDAO
	localDao dao.TreeLocalDAO
}

func NewTreeNodeService(logger *zap.Logger, dao dao.TreeNodeDAO, userDao userDao.UserDAO, localDao dao.TreeLocalDAO) TreeNodeService {
	return &treeService{
		logger:   logger,
		dao:      dao,
		userDao:  userDao,
		localDao: localDao,
	}
}

// GetTreeList 获取树节点列表
func (t *treeService) GetTreeList(ctx context.Context, req *model.GetTreeNodeListReq) (model.ListResp[*model.TreeNode], error) {
	if req.Level < 0 {
		return model.ListResp[*model.TreeNode]{}, errors.New("层级不能为负数")
	}

	t.logger.Debug("获取树节点列表", zap.Int("level", req.Level), zap.String("status", req.Status))

	trees, total, err := t.dao.GetTreeList(ctx, req)
	if err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return model.ListResp[*model.TreeNode]{}, err
	}

	return model.ListResp[*model.TreeNode]{
		Items: trees,
		Total: total,
	}, nil
}

// GetNodeDetail 获取节点详情
func (t *treeService) GetNodeDetail(ctx context.Context, id int) (*model.TreeNode, error) {
	node, err := t.dao.GetNode(ctx, id)
	if err != nil {
		t.logger.Error("获取节点详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return node, nil
}

// GetChildNodes 获取子节点列表
func (t *treeService) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error) {
	if parentId < 0 {
		return nil, errors.New("父节点ID不能为负数")
	}

	children, err := t.dao.GetChildNodes(ctx, parentId)
	if err != nil {
		t.logger.Error("获取子节点列表失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, err
	}

	return children, nil
}

// GetTreeStatistics 获取树统计信息
func (t *treeService) GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error) {
	stats, err := t.dao.GetTreeStatistics(ctx)
	if err != nil {
		t.logger.Error("获取树统计信息失败", zap.Error(err))
		return nil, err
	}

	return stats, nil
}

// CreateNode 创建节点
func (t *treeService) CreateNode(ctx context.Context, req *model.CreateTreeNodeReq) error {
	// 设置默认值
	if req.Status == "" {
		req.Status = DefaultStatus
	}

	// 创建节点实体
	node := &model.TreeNode{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		ParentID:    req.ParentID,
		Status:      req.Status,
		CreatorID:   req.CreatorID,
		IsLeaf:      req.IsLeaf,
	}

	return t.dao.CreateNode(ctx, node)
}

// UpdateNode 更新节点
func (t *treeService) UpdateNode(ctx context.Context, req *model.UpdateTreeNodeReq) error {
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

	return t.dao.UpdateNode(ctx, node)
}

// DeleteNode 删除节点
func (t *treeService) DeleteNode(ctx context.Context, id int) error {
	return t.dao.DeleteNode(ctx, id)
}

// MoveNode 移动节点
func (t *treeService) MoveNode(ctx context.Context, nodeId, newParentId int) error {
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
	if memberType != "" && memberType != NodeAdminRole && memberType != NodeMemberRole && memberType != "all" {
		return nil, errors.New("成员类型只能是admin、member或all")
	}

	t.logger.Debug("获取节点成员", zap.Int("nodeId", nodeId), zap.String("memberType", memberType))

	users, err := t.dao.GetNodeMembers(ctx, nodeId, memberType)
	if err != nil {
		t.logger.Error("获取节点成员失败", zap.Int("nodeId", nodeId), zap.String("memberType", memberType), zap.Error(err))
		return nil, err
	}

	return users, nil
}

// AddNodeMember 添加节点成员
func (t *treeService) AddNodeMember(ctx context.Context, req *model.AddTreeNodeMemberReq) error {
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
func (t *treeService) RemoveNodeMember(ctx context.Context, req *model.RemoveTreeNodeMemberReq) error {
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
func (t *treeService) UpdateNodeStatus(ctx context.Context, req *model.UpdateTreeNodeStatusReq) error {
	return t.dao.UpdateNodeStatus(ctx, req.ID, req.Status)
}

// GetNodeResources 获取节点资源列表
func (t *treeService) GetNodeResources(ctx context.Context, nodeId int) (model.ListResp[*model.ResourceItems], error) {
	resources, err := t.dao.GetNodeResources(ctx, nodeId)
	if err != nil {
		t.logger.Error("获取节点资源失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return model.ListResp[*model.ResourceItems]{}, err
	}

	items := make([]*model.ResourceItems, 0, len(resources))

	// 获取资源信息
	for _, r := range resources {
		id, err := strconv.Atoi(r.ResourceID)
		if err != nil {
			t.logger.Error("获取资源ID失败", zap.String("resourceId", r.ResourceID), zap.Error(err))
			return model.ListResp[*model.ResourceItems]{}, err
		}
		resource, err := t.localDao.GetByID(ctx, id)
		if err != nil {
			t.logger.Error("获取资源信息失败", zap.String("resourceId", r.ResourceID), zap.Error(err))
			return model.ListResp[*model.ResourceItems]{}, err
		}
		items = append(items, &model.ResourceItems{
			ResourceID:   r.ResourceID,
			ResourceName: resource.Name,
			ResourceType: r.ResourceType,
			Status:       string(resource.Status),
			CreatedAt:    resource.CreatedAt,
			UpdatedAt:    resource.UpdatedAt,
		})
	}

	return model.ListResp[*model.ResourceItems]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// BindResource 绑定资源到节点
func (t *treeService) BindResource(ctx context.Context, req *model.BindTreeNodeResourceReq) error {
	if len(req.ResourceIDs) == 0 {
		return errors.New("资源ID列表不能为空")
	}

	if req.ResourceType == "" {
		return errors.New("资源类型不能为空")
	}

	return t.dao.BindResource(ctx, req.NodeID, req.ResourceType, req.ResourceIDs)
}

// UnbindResource 解绑资源
func (t *treeService) UnbindResource(ctx context.Context, req *model.UnbindTreeNodeResourceReq) error {
	if strings.TrimSpace(string(req.ResourceType)) == "" {
		return errors.New("资源类型不能为空")
	}

	if strings.TrimSpace(req.ResourceID) == "" {
		return errors.New("资源ID不能为空")
	}

	t.logger.Info("解绑资源",
		zap.Int("nodeId", req.NodeID),
		zap.String("resourceType", string(req.ResourceType)),
		zap.String("resourceId", req.ResourceID))

	return t.dao.UnbindResource(ctx, req.NodeID, string(req.ResourceType), req.ResourceID)
}
