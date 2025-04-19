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

type VpcService interface {
	GetVpcResourceById(ctx context.Context, req *model.GetVpcDetailReq) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error
	DeleteVpcResource(ctx context.Context, req *model.DeleteVpcReq) error
}

type vpcService struct {
	logger          *zap.Logger
	dao             dao.VpcDAO
	providerFactory *provider.ProviderFactory
}

// CreateVpcResource implements VpcService.
func (v *vpcService) CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error {
	panic("unimplemented")
}

// DeleteVpcResource implements VpcService.
func (v *vpcService) DeleteVpcResource(ctx context.Context, req *model.DeleteVpcReq) error {
	panic("unimplemented")
}

// GetVpcResourceById implements VpcService.
func (v *vpcService) GetVpcResourceById(ctx context.Context, req *model.GetVpcDetailReq) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

func NewVpcService(logger *zap.Logger, dao dao.VpcDAO, providerFactory *provider.ProviderFactory) VpcService {
	return &vpcService{
		logger:          logger,
		dao:             dao,
		providerFactory: providerFactory,
	}
}
