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
	GetRdsDetail(ctx context.Context, req *model.GetRdsDetailReq) (*model.ResourceRds, error)
	CreateRdsResource(ctx context.Context, req *model.CreateRdsResourceReq) error
	DeleteRds(ctx context.Context, req *model.DeleteRdsReq) error
	StartRds(ctx context.Context, req *model.StartRdsReq) error
	StopRds(ctx context.Context, req *model.StopRdsReq) error
	RestartRds(ctx context.Context, req *model.RestartRdsReq) error
	UpdateRds(ctx context.Context, req *model.UpdateRdsReq) error
	ResizeRds(ctx context.Context, req *model.ResizeRdsReq) error
	BackupRds(ctx context.Context, req *model.BackupRdsReq) error
	RestoreRds(ctx context.Context, req *model.RestoreRdsReq) error
	ResetRdsPassword(ctx context.Context, req *model.ResetRdsPasswordReq) error
	RenewRds(ctx context.Context, req *model.RenewRdsReq) error
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

// BackupRds implements TreeRdsService.
func (t *treeRdsService) BackupRds(ctx context.Context, req *model.BackupRdsReq) error {
	panic("unimplemented")
}

// CreateRdsResource implements TreeRdsService.
func (t *treeRdsService) CreateRdsResource(ctx context.Context, req *model.CreateRdsResourceReq) error {
	panic("unimplemented")
}

// DeleteRds implements TreeRdsService.
func (t *treeRdsService) DeleteRds(ctx context.Context, req *model.DeleteRdsReq) error {
	panic("unimplemented")
}

// GetRdsDetail implements TreeRdsService.
func (t *treeRdsService) GetRdsDetail(ctx context.Context, req *model.GetRdsDetailReq) (*model.ResourceRds, error) {
	panic("unimplemented")
}

// ListRdsResources implements TreeRdsService.
func (t *treeRdsService) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error) {
	panic("unimplemented")
}

// RenewRds implements TreeRdsService.
func (t *treeRdsService) RenewRds(ctx context.Context, req *model.RenewRdsReq) error {
	panic("unimplemented")
}

// ResetRdsPassword implements TreeRdsService.
func (t *treeRdsService) ResetRdsPassword(ctx context.Context, req *model.ResetRdsPasswordReq) error {
	panic("unimplemented")
}

// ResizeRds implements TreeRdsService.
func (t *treeRdsService) ResizeRds(ctx context.Context, req *model.ResizeRdsReq) error {
	panic("unimplemented")
}

// RestartRds implements TreeRdsService.
func (t *treeRdsService) RestartRds(ctx context.Context, req *model.RestartRdsReq) error {
	panic("unimplemented")
}

// RestoreRds implements TreeRdsService.
func (t *treeRdsService) RestoreRds(ctx context.Context, req *model.RestoreRdsReq) error {
	panic("unimplemented")
}

// StartRds implements TreeRdsService.
func (t *treeRdsService) StartRds(ctx context.Context, req *model.StartRdsReq) error {
	panic("unimplemented")
}

// StopRds implements TreeRdsService.
func (t *treeRdsService) StopRds(ctx context.Context, req *model.StopRdsReq) error {
	panic("unimplemented")
}

// UpdateRds implements TreeRdsService.
func (t *treeRdsService) UpdateRds(ctx context.Context, req *model.UpdateRdsReq) error {
	panic("unimplemented")
}
