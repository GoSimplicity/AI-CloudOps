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
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type TreeVpcService interface {
	// VPC资源管理
	GetVpcDetail(ctx context.Context, req *model.GetVpcDetailReq) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error
	DeleteVpc(ctx context.Context, req *model.DeleteVpcReq) error
	ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error)
	UpdateVpc(ctx context.Context, req *model.UpdateVpcReq) error

	// 子网管理
	CreateSubnet(ctx context.Context, req *model.CreateSubnetReq) error
	DeleteSubnet(ctx context.Context, req *model.DeleteSubnetReq) error
	ListSubnets(ctx context.Context, req *model.ListSubnetsReq) (model.ListResp[*model.ResourceSubnet], error)
	GetSubnetDetail(ctx context.Context, req *model.GetSubnetDetailReq) (*model.ResourceSubnet, error)
	UpdateSubnet(ctx context.Context, req *model.UpdateSubnetReq) error

	// VPC对等连接管理
	CreateVpcPeering(ctx context.Context, req *model.CreateVpcPeeringReq) error
	DeleteVpcPeering(ctx context.Context, req *model.DeleteVpcPeeringReq) error
	ListVpcPeerings(ctx context.Context, req *model.ListVpcPeeringsReq) (model.ListResp[*model.ResourceVpcPeering], error)
}

type treeVpcService struct {
	logger          *zap.Logger
	dao             dao.TreeVpcDAO
	providerFactory *provider.ProviderFactory
}

func NewTreeVpcService(logger *zap.Logger, dao dao.TreeVpcDAO, providerFactory *provider.ProviderFactory) TreeVpcService {
	return &treeVpcService{
		logger:          logger,
		dao:             dao,
		providerFactory: providerFactory,
	}
}

// CreateSubnet implements TreeVpcService.
func (t *treeVpcService) CreateSubnet(ctx context.Context, req *model.CreateSubnetReq) error {
	panic("unimplemented")
}

// CreateVpcPeering implements TreeVpcService.
func (t *treeVpcService) CreateVpcPeering(ctx context.Context, req *model.CreateVpcPeeringReq) error {
	panic("unimplemented")
}

// CreateVpcResource implements TreeVpcService.
func (t *treeVpcService) CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error {
	panic("unimplemented")
}

// DeleteSubnet implements TreeVpcService.
func (t *treeVpcService) DeleteSubnet(ctx context.Context, req *model.DeleteSubnetReq) error {
	panic("unimplemented")
}

// DeleteVpc implements TreeVpcService.
func (t *treeVpcService) DeleteVpc(ctx context.Context, req *model.DeleteVpcReq) error {
	panic("unimplemented")
}

// DeleteVpcPeering implements TreeVpcService.
func (t *treeVpcService) DeleteVpcPeering(ctx context.Context, req *model.DeleteVpcPeeringReq) error {
	panic("unimplemented")
}

// GetSubnetDetail implements TreeVpcService.
func (t *treeVpcService) GetSubnetDetail(ctx context.Context, req *model.GetSubnetDetailReq) (*model.ResourceSubnet, error) {
	panic("unimplemented")
}

// GetVpcDetail implements TreeVpcService.
func (t *treeVpcService) GetVpcDetail(ctx context.Context, req *model.GetVpcDetailReq) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

// ListSubnets implements TreeVpcService.
func (t *treeVpcService) ListSubnets(ctx context.Context, req *model.ListSubnetsReq) (model.ListResp[*model.ResourceSubnet], error) {
	panic("unimplemented")
}

// ListVpcPeerings implements TreeVpcService.
func (t *treeVpcService) ListVpcPeerings(ctx context.Context, req *model.ListVpcPeeringsReq) (model.ListResp[*model.ResourceVpcPeering], error) {
	panic("unimplemented")
}

// ListVpcResources implements TreeVpcService.
func (t *treeVpcService) ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error) {
	panic("unimplemented")
}

// UpdateSubnet implements TreeVpcService.
func (t *treeVpcService) UpdateSubnet(ctx context.Context, req *model.UpdateSubnetReq) error {
	panic("unimplemented")
}

// UpdateVpc implements TreeVpcService.
func (t *treeVpcService) UpdateVpc(ctx context.Context, req *model.UpdateVpcReq) error {
	panic("unimplemented")
}
