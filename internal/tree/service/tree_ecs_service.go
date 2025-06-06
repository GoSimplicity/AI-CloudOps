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

type TreeEcsService interface {
	// 资源管理
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error)
	GetEcsDetail(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceECSDetailResp, error)
	CreateEcsResource(ctx context.Context, req *model.CreateEcsResourceReq) error
	UpdateEcs(ctx context.Context, req *model.UpdateEcsReq) error
	DeleteEcs(ctx context.Context, req *model.DeleteEcsReq) error
	StartEcs(ctx context.Context, req *model.StartEcsReq) error
	StopEcs(ctx context.Context, req *model.StopEcsReq) error
	RestartEcs(ctx context.Context, req *model.RestartEcsReq) error
	ResizeEcs(ctx context.Context, req *model.ResizeEcsReq) error
	ResetEcsPassword(ctx context.Context, req *model.ResetEcsPasswordReq) error
	RenewEcs(ctx context.Context, req *model.RenewEcsReq) error
	ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error)
}

type treeEcsService struct {
	providerFactory *provider.ProviderFactory
	logger          *zap.Logger
	dao             dao.TreeEcsDAO
}

func NewTreeEcsService(logger *zap.Logger, dao dao.TreeEcsDAO, providerFactory *provider.ProviderFactory) TreeEcsService {
	return &treeEcsService{
		logger:          logger,
		dao:             dao,
		providerFactory: providerFactory,
	}
}

// CreateEcsResource implements TreeEcsService.
func (t *treeEcsService) CreateEcsResource(ctx context.Context, req *model.CreateEcsResourceReq) error {
	panic("unimplemented")
}

// DeleteEcs implements TreeEcsService.
func (t *treeEcsService) DeleteEcs(ctx context.Context, req *model.DeleteEcsReq) error {
	panic("unimplemented")
}

// GetEcsDetail implements TreeEcsService.
func (t *treeEcsService) GetEcsDetail(ctx context.Context, req *model.GetEcsDetailReq) (*model.ResourceECSDetailResp, error) {
	panic("unimplemented")
}

// ListEcsResourceOptions implements TreeEcsService.
func (t *treeEcsService) ListEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) (model.ListResp[*model.ListEcsResourceOptionsResp], error) {
	panic("unimplemented")
}

// ListEcsResources implements TreeEcsService.
func (t *treeEcsService) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (model.ListResp[*model.ResourceEcs], error) {
	panic("unimplemented")
}

// RenewEcs implements TreeEcsService.
func (t *treeEcsService) RenewEcs(ctx context.Context, req *model.RenewEcsReq) error {
	panic("unimplemented")
}

// ResetEcsPassword implements TreeEcsService.
func (t *treeEcsService) ResetEcsPassword(ctx context.Context, req *model.ResetEcsPasswordReq) error {
	panic("unimplemented")
}

// ResizeEcs implements TreeEcsService.
func (t *treeEcsService) ResizeEcs(ctx context.Context, req *model.ResizeEcsReq) error {
	panic("unimplemented")
}

// RestartEcs implements TreeEcsService.
func (t *treeEcsService) RestartEcs(ctx context.Context, req *model.RestartEcsReq) error {
	panic("unimplemented")
}

// StartEcs implements TreeEcsService.
func (t *treeEcsService) StartEcs(ctx context.Context, req *model.StartEcsReq) error {
	panic("unimplemented")
}

// StopEcs implements TreeEcsService.
func (t *treeEcsService) StopEcs(ctx context.Context, req *model.StopEcsReq) error {
	panic("unimplemented")
}

// UpdateEcs implements TreeEcsService.
func (t *treeEcsService) UpdateEcs(ctx context.Context, req *model.UpdateEcsReq) error {
	panic("unimplemented")
}
