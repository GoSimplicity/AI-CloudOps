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
	GetVpcResourceById(ctx context.Context, req *model.GetVpcDetailReq) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error
	DeleteVpcResource(ctx context.Context, req *model.DeleteVpcReq) error
	ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error)
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

// CreateVpcResource 创建VPC
func (v *treeVpcService) CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error {
	provider, err := v.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return err
	}

	return provider.CreateVPC(ctx, req.Region, req)
}

// DeleteVpcResource 删除VPC
func (v *treeVpcService) DeleteVpcResource(ctx context.Context, req *model.DeleteVpcReq) error {
	provider, err := v.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return err
	}

	return provider.DeleteVPC(ctx, req.Region, req.VpcId)
}

// GetVpcResourceById 获取VPC详情
func (v *treeVpcService) GetVpcResourceById(ctx context.Context, req *model.GetVpcDetailReq) (*model.ResourceVpc, error) {
	provider, err := v.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return nil, err
	}

	return provider.GetVpcDetail(ctx, req.Region, req.VpcId)
}

// ListVpcResources 获取VPC列表
func (v *treeVpcService) ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error) {
	provider, err := v.providerFactory.GetProvider(req.Provider)
	if err != nil {
		return model.ListResp[*model.ResourceVpc]{}, err
	}

	vpcList, total, err := provider.ListVPCs(ctx, req.Region, req.PageNumber, req.PageSize)
	if err != nil {
		return model.ListResp[*model.ResourceVpc]{}, err
	}

	return model.ListResp[*model.ResourceVpc]{
		Total: total,
		Items: vpcList,
	}, nil
}
