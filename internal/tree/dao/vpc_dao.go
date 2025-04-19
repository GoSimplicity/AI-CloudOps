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

type VpcDAO interface {
	ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (*model.PageResp, error)
	GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error
	DeleteVpcResource(ctx context.Context, id int) error
}

type vpcDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewVpcDAO(logger *zap.Logger, db *gorm.DB) VpcDAO {
	return &vpcDAO{
		logger: logger,
		db:     db,
	}
}

// CreateVpcResource implements VpcDAO.
func (v *vpcDAO) CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error {
	panic("unimplemented")
}

// DeleteVpcResource implements VpcDAO.
func (v *vpcDAO) DeleteVpcResource(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetVpcResourceById implements VpcDAO.
func (v *vpcDAO) GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

// ListVpcResources implements VpcDAO.
func (v *vpcDAO) ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}
