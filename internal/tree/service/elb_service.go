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

type ElbService interface {
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (*model.PageResp, error)
	GetElbResourceById(ctx context.Context, id int) (*model.ResourceELBResp, error)
	CreateElbResource(ctx context.Context, params *model.ElbCreationParams) error
}

type elbService struct {
	logger *zap.Logger
	dao    *dao.ElbDAO
}

func NewElbService(logger *zap.Logger, dao *dao.ElbDAO) ElbService {
	return &elbService{
		logger: logger,
		dao:    dao,
	}
}

// CreateElbResource implements ElbService.
func (e *elbService) CreateElbResource(ctx context.Context, params *model.ElbCreationParams) error {
	panic("unimplemented")
}

// GetElbResourceById implements ElbService.
func (e *elbService) GetElbResourceById(ctx context.Context, id int) (*model.ResourceELBResp, error) {
	panic("unimplemented")
}

// ListElbResources implements ElbService.
func (e *elbService) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}
