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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

type TreeService interface {
	// 树结构操作
	GetTree(ctx context.Context) ([]model.TreeNodeResp, error)
	GetNodeById(ctx context.Context, id int) (*model.TreeNodeDetailResp, error)
	GetChildNodes(ctx context.Context, parentId int) ([]model.TreeNodeResp, error)
	GetNodePath(ctx context.Context, nodeId int) ([]model.TreeNodePathResp, error)
	GetNodeResources(ctx context.Context, nodeId int) ([]model.NodeResourceResp, error)

	// 节点管理
	CreateNode(ctx context.Context, req *model.CreateNodeReq) (*model.TreeNodeResp, error)
	UpdateNode(ctx context.Context, req *model.UpdateNodeReq) error
	DeleteNode(ctx context.Context, id int) error

	// 资源绑定
	BindResource(ctx context.Context, req *model.ResourceBindingRequest) error
	UnbindResource(ctx context.Context, req *model.ResourceBindingRequest) error

	// 成员管理
	AddNodeAdmin(ctx context.Context, req *model.NodeAdminReq) error
	RemoveNodeAdmin(ctx context.Context, req *model.NodeAdminReq) error
	AddNodeMember(ctx context.Context, req *model.NodeMemberReq) error
	RemoveNodeMember(ctx context.Context, req *model.NodeMemberReq) error
}

type treeService struct {
	logger *zap.Logger
	dao    *dao.TreeDAO
}

func NewTreeService(logger *zap.Logger, dao *dao.TreeDAO) TreeService {
	return &treeService{
		logger: logger,
		dao:    dao,
	}
}

// AddNodeAdmin implements TreeService.
func (t *treeService) AddNodeAdmin(ctx context.Context, req *model.NodeAdminReq) error {
	panic("unimplemented")
}

// AddNodeMember implements TreeService.
func (t *treeService) AddNodeMember(ctx context.Context, req *model.NodeMemberReq) error {
	panic("unimplemented")
}

// BindResource implements TreeService.
func (t *treeService) BindResource(ctx context.Context, req *model.ResourceBindingRequest) error {
	panic("unimplemented")
}

// CreateNode implements TreeService.
func (t *treeService) CreateNode(ctx context.Context, req *model.CreateNodeReq) (*model.TreeNodeResp, error) {
	panic("unimplemented")
}

// DeleteNode implements TreeService.
func (t *treeService) DeleteNode(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetChildNodes implements TreeService.
func (t *treeService) GetChildNodes(ctx context.Context, parentId int) ([]model.TreeNodeResp, error) {
	panic("unimplemented")
}

// GetNodeById implements TreeService.
func (t *treeService) GetNodeById(ctx context.Context, id int) (*model.TreeNodeDetailResp, error) {
	panic("unimplemented")
}

// GetNodePath implements TreeService.
func (t *treeService) GetNodePath(ctx context.Context, nodeId int) ([]model.TreeNodePathResp, error) {
	panic("unimplemented")
}

// GetNodeResources implements TreeService.
func (t *treeService) GetNodeResources(ctx context.Context, nodeId int) ([]model.NodeResourceResp, error) {
	panic("unimplemented")
}

// GetTree implements TreeService.
func (t *treeService) GetTree(ctx context.Context) ([]model.TreeNodeResp, error) {
	panic("unimplemented")
}

// RemoveNodeAdmin implements TreeService.
func (t *treeService) RemoveNodeAdmin(ctx context.Context, req *model.NodeAdminReq) error {
	panic("unimplemented")
}

// RemoveNodeMember implements TreeService.
func (t *treeService) RemoveNodeMember(ctx context.Context, req *model.NodeMemberReq) error {
	panic("unimplemented")
}

// UnbindResource implements TreeService.
func (t *treeService) UnbindResource(ctx context.Context, req *model.ResourceBindingRequest) error {
	panic("unimplemented")
}

// UpdateNode implements TreeService.
func (t *treeService) UpdateNode(ctx context.Context, req *model.UpdateNodeReq) error {
	panic("unimplemented")
}
