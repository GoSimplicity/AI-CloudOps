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

type TreeElbService interface {
	// 资源管理
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error)
	GetElbDetail(ctx context.Context, req *model.GetElbDetailReq) (*model.ResourceElb, error)
	CreateElbResource(ctx context.Context, req *model.CreateElbResourceReq) error
	UpdateElb(ctx context.Context, req *model.UpdateElbReq) error
	DeleteElb(ctx context.Context, req *model.DeleteElbReq) error
	StartElb(ctx context.Context, req *model.StartElbReq) error
	StopElb(ctx context.Context, req *model.StopElbReq) error
	RestartElb(ctx context.Context, req *model.RestartElbReq) error
	ResizeElb(ctx context.Context, req *model.ResizeElbReq) error
	BindServersToElb(ctx context.Context, req *model.BindServersToElbReq) error
	UnbindServersFromElb(ctx context.Context, req *model.UnbindServersFromElbReq) error
	ConfigureHealthCheck(ctx context.Context, req *model.ConfigureHealthCheckReq) error
}

type elbService struct {
	logger *zap.Logger
	dao    dao.TreeElbDAO
}

func NewTreeElbService(logger *zap.Logger, dao dao.TreeElbDAO) TreeElbService {
	return &elbService{
		logger: logger,
		dao:    dao,
	}
}

// BindServersToElb implements TreeElbService.
func (e *elbService) BindServersToElb(ctx context.Context, req *model.BindServersToElbReq) error {
	panic("unimplemented")
}

// ConfigureHealthCheck implements TreeElbService.
func (e *elbService) ConfigureHealthCheck(ctx context.Context, req *model.ConfigureHealthCheckReq) error {
	panic("unimplemented")
}

// CreateElbResource implements TreeElbService.
func (e *elbService) CreateElbResource(ctx context.Context, req *model.CreateElbResourceReq) error {
	panic("unimplemented")
}

// DeleteElb implements TreeElbService.
func (e *elbService) DeleteElb(ctx context.Context, req *model.DeleteElbReq) error {
	panic("unimplemented")
}

// GetElbDetail implements TreeElbService.
func (e *elbService) GetElbDetail(ctx context.Context, req *model.GetElbDetailReq) (*model.ResourceElb, error) {
	panic("unimplemented")
}

// ListElbResources implements TreeElbService.
func (e *elbService) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error) {
	panic("unimplemented")
}

// ResizeElb implements TreeElbService.
func (e *elbService) ResizeElb(ctx context.Context, req *model.ResizeElbReq) error {
	panic("unimplemented")
}

// RestartElb implements TreeElbService.
func (e *elbService) RestartElb(ctx context.Context, req *model.RestartElbReq) error {
	panic("unimplemented")
}

// StartElb implements TreeElbService.
func (e *elbService) StartElb(ctx context.Context, req *model.StartElbReq) error {
	panic("unimplemented")
}

// StopElb implements TreeElbService.
func (e *elbService) StopElb(ctx context.Context, req *model.StopElbReq) error {
	panic("unimplemented")
}

// UnbindServersFromElb implements TreeElbService.
func (e *elbService) UnbindServersFromElb(ctx context.Context, req *model.UnbindServersFromElbReq) error {
	panic("unimplemented")
}

// UpdateElb implements TreeElbService.
func (e *elbService) UpdateElb(ctx context.Context, req *model.UpdateElbReq) error {
	panic("unimplemented")
}
