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

type TreeCloudService interface {
	CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error
	UpdateCloudAccount(ctx context.Context, id int, req *model.UpdateCloudAccountReq) error
	DeleteCloudAccount(ctx context.Context, id int) error
	GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error)
	ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error)
	TestCloudAccount(ctx context.Context, id int) error
	SyncCloudResources(ctx context.Context, req *model.SyncCloudReq) error
}

type treeCloudService struct {
	logger *zap.Logger
	dao    dao.TreeCloudDAO
}

func NewTreeCloudService(logger *zap.Logger, dao dao.TreeCloudDAO) TreeCloudService {
	return &treeCloudService{
		logger: logger,
		dao:    dao,
	}
}

// CreateCloudAccount implements TreeCloudService.
func (t *treeCloudService) CreateCloudAccount(ctx context.Context, req *model.CreateCloudAccountReq) error {
	panic("unimplemented")
}

// DeleteCloudAccount implements TreeCloudService.
func (t *treeCloudService) DeleteCloudAccount(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetCloudAccount implements TreeCloudService.
func (t *treeCloudService) GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error) {
	panic("unimplemented")
}

// ListCloudAccounts implements TreeCloudService.
func (t *treeCloudService) ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error) {
	panic("unimplemented")
}

// SyncCloudResources implements TreeCloudService.
func (t *treeCloudService) SyncCloudResources(ctx context.Context, req *model.SyncCloudReq) error {
	panic("unimplemented")
}

// TestCloudAccount implements TreeCloudService.
func (t *treeCloudService) TestCloudAccount(ctx context.Context, id int) error {
	panic("unimplemented")
}

// UpdateCloudAccount implements TreeCloudService.
func (t *treeCloudService) UpdateCloudAccount(ctx context.Context, id int, req *model.UpdateCloudAccountReq) error {
	panic("unimplemented")
}
