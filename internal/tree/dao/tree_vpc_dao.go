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

package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeVpcDAO interface {
	// VPC资源管理
	ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error)
	GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error
	UpdateVpcResource(ctx context.Context, req *model.UpdateVpcReq) error
	DeleteVpcResource(ctx context.Context, id int) error

	// 子网管理
	ListSubnets(ctx context.Context, req *model.ListSubnetsReq) (model.ListResp[*model.ResourceSubnet], error)
	GetSubnetById(ctx context.Context, id int) (*model.ResourceSubnet, error)
	CreateSubnet(ctx context.Context, req *model.CreateSubnetReq) error
	UpdateSubnet(ctx context.Context, req *model.UpdateSubnetReq) error
	DeleteSubnet(ctx context.Context, id int) error

	// VPC对等连接管理
	ListVpcPeerings(ctx context.Context, req *model.ListVpcPeeringsReq) (model.ListResp[*model.ResourceVpcPeering], error)
	GetVpcPeeringById(ctx context.Context, id int) (*model.ResourceVpcPeering, error)
	CreateVpcPeering(ctx context.Context, req *model.CreateVpcPeeringReq) error
	DeleteVpcPeering(ctx context.Context, id int) error
}

type treeVpcDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewTreeVpcDAO(logger *zap.Logger, db *gorm.DB) TreeVpcDAO {
	return &treeVpcDAO{
		logger: logger,
		db:     db,
	}
}

// CreateSubnet implements TreeVpcDAO.
func (t *treeVpcDAO) CreateSubnet(ctx context.Context, req *model.CreateSubnetReq) error {
	panic("unimplemented")
}

// CreateVpcPeering implements TreeVpcDAO.
func (t *treeVpcDAO) CreateVpcPeering(ctx context.Context, req *model.CreateVpcPeeringReq) error {
	panic("unimplemented")
}

// CreateVpcResource implements TreeVpcDAO.
func (t *treeVpcDAO) CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error {
	panic("unimplemented")
}

// DeleteSubnet implements TreeVpcDAO.
func (t *treeVpcDAO) DeleteSubnet(ctx context.Context, id int) error {
	panic("unimplemented")
}

// DeleteVpcPeering implements TreeVpcDAO.
func (t *treeVpcDAO) DeleteVpcPeering(ctx context.Context, id int) error {
	panic("unimplemented")
}

// DeleteVpcResource implements TreeVpcDAO.
func (t *treeVpcDAO) DeleteVpcResource(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetSubnetById implements TreeVpcDAO.
func (t *treeVpcDAO) GetSubnetById(ctx context.Context, id int) (*model.ResourceSubnet, error) {
	panic("unimplemented")
}

// GetVpcPeeringById implements TreeVpcDAO.
func (t *treeVpcDAO) GetVpcPeeringById(ctx context.Context, id int) (*model.ResourceVpcPeering, error) {
	panic("unimplemented")
}

// GetVpcResourceById implements TreeVpcDAO.
func (t *treeVpcDAO) GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

// ListSubnets implements TreeVpcDAO.
func (t *treeVpcDAO) ListSubnets(ctx context.Context, req *model.ListSubnetsReq) (model.ListResp[*model.ResourceSubnet], error) {
	panic("unimplemented")
}

// ListVpcPeerings implements TreeVpcDAO.
func (t *treeVpcDAO) ListVpcPeerings(ctx context.Context, req *model.ListVpcPeeringsReq) (model.ListResp[*model.ResourceVpcPeering], error) {
	panic("unimplemented")
}

// ListVpcResources implements TreeVpcDAO.
func (t *treeVpcDAO) ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error) {
	panic("unimplemented")
}

// UpdateSubnet implements TreeVpcDAO.
func (t *treeVpcDAO) UpdateSubnet(ctx context.Context, req *model.UpdateSubnetReq) error {
	panic("unimplemented")
}

// UpdateVpcResource implements TreeVpcDAO.
func (t *treeVpcDAO) UpdateVpcResource(ctx context.Context, req *model.UpdateVpcReq) error {
	panic("unimplemented")
}
