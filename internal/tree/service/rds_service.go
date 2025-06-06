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

type TreeRdsService interface {
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error)
	GetRdsResourceById(ctx context.Context, req *model.GetRdsDetailReq) (*model.ResourceRDSResp, error)
	CreateRdsResource(ctx context.Context, req *model.RdsCreationParams) error
	DeleteRdsResource(ctx context.Context, req *model.DeleteRdsReq) error
	StartRdsResource(ctx context.Context, req *model.StartRdsReq) error
	StopRdsResource(ctx context.Context, req *model.StopRdsReq) error
	RestartRdsResource(ctx context.Context, req *model.RestartRdsReq) error
}

type treeRdsService struct {
	logger *zap.Logger
	dao    dao.TreeRdsDAO
}

func NewTreeRdsService(logger *zap.Logger, dao dao.TreeRdsDAO) TreeRdsService {
	return &treeRdsService{
		logger: logger,
		dao:    dao,
	}
}

// CreateRdsResource implements RdsService.
func (r *treeRdsService) CreateRdsResource(ctx context.Context, req *model.RdsCreationParams) error {
	panic("unimplemented")
}

// DeleteRdsResource implements RdsService.
func (r *treeRdsService) DeleteRdsResource(ctx context.Context, req *model.DeleteRdsReq) error {
	panic("unimplemented")
}

// GetRdsResourceById implements RdsService.
func (r *treeRdsService) GetRdsResourceById(ctx context.Context, req *model.GetRdsDetailReq) (*model.ResourceRDSResp, error) {
	panic("unimplemented")
}

// ListRdsResources implements RdsService.
func (r *treeRdsService) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error) {
	panic("unimplemented")
}

// RestartRdsResource implements RdsService.
func (r *treeRdsService) RestartRdsResource(ctx context.Context, req *model.RestartRdsReq) error {
	panic("unimplemented")
}

// StartRdsResource implements RdsService.
func (r *treeRdsService) StartRdsResource(ctx context.Context, req *model.StartRdsReq) error {
	panic("unimplemented")
}

// StopRdsResource implements RdsService.
func (r *treeRdsService) StopRdsResource(ctx context.Context, req *model.StopRdsReq) error {
	panic("unimplemented")
}
