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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

const (
	NodeAdminRole      = "admin"
	NodeMemberRole     = "member"
	NodeStatusActive   = "active"
	NodeStatusInactive = "inactive"
)

type TreeService interface {
	// 树结构相关接口
	GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNode, error)
	GetNodeDetail(ctx context.Context, id int) (*model.TreeNodeDetailResp, error)
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNodeResp, error)
	GetNodePath(ctx context.Context, nodeId int) (*model.TreeNodePathResp, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error)

	// 节点管理接口
	CreateNode(ctx context.Context, node *model.TreeNodeCreateReq) error
	CreateChildNode(ctx context.Context, parentId int, node *model.TreeNodeCreateReq) (*model.TreeNodeResp, error)
	UpdateNode(ctx context.Context, node *model.TreeNodeUpdateReq) error
	DeleteNode(ctx context.Context, id int) error

	// 资源绑定接口
	GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error)
	BindResource(ctx context.Context, req *model.TreeNodeResourceBindReq) error
	UnbindResource(ctx context.Context, req *model.TreeNodeResourceUnbindReq) error
	GetResourceTypes(ctx context.Context) ([]string, error)

	// 成员管理接口
	GetNodeMembers(ctx context.Context, req *model.TreeNodeMemberReq) ([]*model.User, error)
	AddNodeMember(ctx context.Context, req *model.TreeNodeMemberReq) error
	RemoveNodeMember(ctx context.Context, nodeId int, userId int) error
	AddNodeAdmin(ctx context.Context, nodeId int, userId int) error
	RemoveNodeAdmin(ctx context.Context, nodeId int, userId int) error
}

type treeService struct {
	logger *zap.Logger
	dao    dao.TreeDAO
}

func NewTreeService(logger *zap.Logger, dao dao.TreeDAO) TreeService {
	return &treeService{
		logger: logger,
		dao:    dao,
	}
}

// AddNodeAdmin 添加节点管理员
func (t *treeService) AddNodeAdmin(ctx context.Context, nodeId int, userId int) error {
	return t.dao.AddNodeMember(ctx, nodeId, userId, NodeAdminRole)
}

// AddNodeMember 添加节点成员
func (t *treeService) AddNodeMember(ctx context.Context, req *model.TreeNodeMemberReq) error {
	return t.dao.AddNodeMember(ctx, req.NodeID, req.UserID, NodeMemberRole)
}

// BindResource 绑定资源到节点
func (t *treeService) BindResource(ctx context.Context, req *model.TreeNodeResourceBindReq) error {
	return t.dao.BindResource(ctx, req.NodeID, req.ResourceType, req.ResourceIDs)
}

// CreateChildNode 创建子节点
func (t *treeService) CreateChildNode(ctx context.Context, parentId int, req *model.TreeNodeCreateReq) (*model.TreeNodeResp, error) {
	// 获取父节点信息以确定子节点的层级
	parentNode, err := t.dao.GetNodeDetail(ctx, parentId)
	if err != nil {
		t.logger.Error("获取父节点失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, err
	}

	// 创建节点实体
	node := &model.TreeNode{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    parentId,
		Level:       parentNode.Level + 1,
		Status:      NodeStatusActive,
		Model: model.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// 调用DAO创建节点
	if err := t.dao.CreateNode(ctx, node); err != nil {
		t.logger.Error("创建子节点失败", zap.Error(err))
		return nil, err
	}

	// 返回创建的节点信息
	return &model.TreeNodeResp{
		ID:          int(node.ID),
		Name:        node.Name,
		Description: node.Description,
		ParentID:    node.ParentID,
		ParentName:  parentNode.Name,
		Level:       node.Level,
		Status:      node.Status,
		CreatedAt:   node.CreatedAt,
		UpdatedAt:   node.UpdatedAt,
	}, nil
}

// CreateNode 创建根节点
func (t *treeService) CreateNode(ctx context.Context, req *model.TreeNodeCreateReq) error {
	var parentPath string
	var level int

	if req.ParentID != 0 {
		// 直接查询父节点的level以及path
		parent, err := t.dao.GetNodeDetail(ctx, req.ParentID)
		if err != nil {
			t.logger.Error("获取父节点失败", zap.Int("parentId", req.ParentID), zap.Error(err))
			return err
		}
		parentPath = parent.Path
		level = parent.Level + 1
	}

	// 创建节点实体
	node := &model.TreeNode{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		Level:       level,
		Status:      NodeStatusActive,
	}

	err := t.dao.Transaction(ctx, func(ctx context.Context) error {
		if err := t.dao.CreateNode(ctx, node); err != nil {
			t.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Int("parentId", req.ParentID), zap.Error(err))
			return err
		}

		// 节点创建成功后，更新路径
		// 之所以这么做，是因为在创建节点时，节点路径是未知的，所以需要先创建节点，再更新路径
		// 路径的格式为：/1/2/3，表示节点3是节点2的子节点，节点2是节点1的子节点
		// 这样做的好处是，当需要获取节点路径时，只需要根据节点ID获取路径，而不需要递归获取
		// 缺点是，当节点路径发生变化时，需要手动更新路径
		var path string
		if parentPath == "" {
			path = fmt.Sprintf("/%d", node.ID)
		} else {
			path = fmt.Sprintf("%s/%d", parentPath, node.ID)
		}

		if err := t.dao.UpdateNodePath(ctx, int(node.ID), path); err != nil {
			t.logger.Error("更新节点路径失败", zap.Int("nodeId", int(node.ID)), zap.String("path", path), zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		t.logger.Error("创建节点事务执行失败", zap.String("name", req.Name), zap.Int("parentId", req.ParentID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteNode 删除节点
func (t *treeService) DeleteNode(ctx context.Context, id int) error {
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

	if len(children) > 0 || len(resources) > 0 {
		t.logger.Error("节点有子节点或绑定资源，无法删除", zap.Int("id", id))
		return errors.New("节点有子节点或绑定资源，无法删除")
	}

	// 删除节点
	err = t.dao.DeleteNode(ctx, id)
	if err != nil {
		t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	return nil
}

// GetChildNodes 获取子节点列表
func (t *treeService) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNodeResp, error) {
	// 调用DAO获取子节点
	nodes, err := t.dao.GetChildNodes(ctx, parentId)
	if err != nil {
		t.logger.Error("获取子节点失败", zap.Error(err))
		return nil, err
	}

	// 转换为响应格式
	result := make([]*model.TreeNodeResp, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, &model.TreeNodeResp{
			ID:          int(node.ID),
			Name:        node.Name,
			Description: node.Description,
			ParentID:    node.ParentID,
			ParentName:  node.ParentName,
			Level:       node.Level,
			Status:      node.Status,
			ChildCount:  node.ChildCount,
			CreatedAt:   node.CreatedAt,
			UpdatedAt:   node.UpdatedAt,
		})
	}

	return result, nil
}

// GetNodeDetail 获取节点详情
func (t *treeService) GetNodeDetail(ctx context.Context, id int) (*model.TreeNodeDetailResp, error) {
	// 调用DAO获取节点详情
	node, err := t.dao.GetNodeDetail(ctx, id)
	if err != nil {
		t.logger.Error("获取节点详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	adminUsers := make(model.StringList, 0, len(node.Admins))
	memberUsers := make(model.StringList, 0, len(node.Members))

	// 提取管理员和成员用户名
	for _, admin := range node.Admins {
		adminUsers = append(adminUsers, admin.Username)
	}

	for _, member := range node.Members {
		memberUsers = append(memberUsers, member.Username)
	}

	return &model.TreeNodeDetailResp{
		TreeNodeResp: model.TreeNodeResp{
			ID:          int(node.ID),
			Name:        node.Name,
			Description: node.Description,
			ParentID:    node.ParentID,
			ParentName:  node.ParentName,
			Path:        node.Path,
			Level:       node.Level,
			ServiceCode: node.ServiceCode,
			Status:      node.Status,
			ChildCount:  node.ChildCount,
			CreatedAt:   node.CreatedAt,
			UpdatedAt:   node.UpdatedAt,
		},
		AdminUsers:    adminUsers,
		MemberUsers:   memberUsers,
		ResourceCount: node.ResourceCount,
	}, nil
}

// GetNodeMembers 获取节点成员列表
func (t *treeService) GetNodeMembers(ctx context.Context, req *model.TreeNodeMemberReq) ([]*model.User, error) {
	members, err := t.dao.GetNodeMembers(ctx, req.NodeID, req.UserID, req.Type)
	if err != nil {
		t.logger.Error("获取节点成员失败", zap.Error(err))
		return nil, err
	}

	return members, nil
}

// GetNodePath 获取节点路径
func (t *treeService) GetNodePath(ctx context.Context, nodeId int) (*model.TreeNodePathResp, error) {
	// 调用DAO获取节点路径
	path, err := t.dao.GetNodePath(ctx, nodeId)
	if err != nil {
		t.logger.Error("获取节点路径失败", zap.Error(err))
		return nil, err
	}

	// 转换为响应格式
	nodes := make([]*model.TreeNodeResp, 0, len(path))
	for _, node := range path {
		nodes = append(nodes, &model.TreeNodeResp{
			ID:   int(node.ID),
			Name: node.Name,
		})
	}

	return &model.TreeNodePathResp{
		Path:  nodes,
		Total: len(nodes),
	}, nil
}

// GetNodeResources 获取节点资源列表
func (t *treeService) GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error) {
	return t.dao.GetNodeResources(ctx, nodeId)
}

// GetResourceTypes 获取资源类型列表
func (t *treeService) GetResourceTypes(ctx context.Context) ([]string, error) {
	return t.dao.GetResourceTypes(ctx)
}

// GetTreeList 获取树节点列表
func (t *treeService) GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNode, error) {
	// 检查请求参数
	if req.ParentID < 0 || req.Level < 0 {
		return nil, errors.New("父节点ID或层级不允许小于0")
	}

	resp, err := t.dao.GetTreeList(ctx, req)
	if err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

// GetTreeStatistics 获取树统计信息
func (t *treeService) GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error) {
	return t.dao.GetTreeStatistics(ctx)
}

// RemoveNodeAdmin 移除节点管理员
func (t *treeService) RemoveNodeAdmin(ctx context.Context, nodeId int, userId int) error {
	return t.dao.RemoveNodeMember(ctx, nodeId, userId, "admin")
}

// RemoveNodeMember 移除节点成员
func (t *treeService) RemoveNodeMember(ctx context.Context, nodeId int, userId int) error {
	return t.dao.RemoveNodeMember(ctx, nodeId, userId, "member")
}

// UnbindResource 解绑资源
func (t *treeService) UnbindResource(ctx context.Context, req *model.TreeNodeResourceUnbindReq) error {
	return t.dao.UnbindResource(ctx, req.NodeID, req.ResourceType, req.ResourceID)
}

// UpdateNode 更新节点
func (t *treeService) UpdateNode(ctx context.Context, req *model.TreeNodeUpdateReq) error {
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

	return t.dao.UpdateNode(ctx, node)
}
