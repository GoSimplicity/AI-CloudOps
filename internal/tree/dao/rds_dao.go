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
	"gorm.io/gorm"
)

type TreeRdsDAO interface {
	// RDS资源接口
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error)
	GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRDSResp, error)
	CreateRdsResource(ctx context.Context, params *model.RdsCreationParams) error
}

type treeRdsDAO struct {
	db *gorm.DB
}

func NewTreeRdsDAO(db *gorm.DB) TreeRdsDAO {
	return &treeRdsDAO{
		db: db,
	}
}

// CreateRdsResource implements RdsDAO.
func (r *treeRdsDAO) CreateRdsResource(ctx context.Context, params *model.RdsCreationParams) error {
	panic("unimplemented")
}

// GetRdsResourceById implements RdsDAO.
func (r *treeRdsDAO) GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRDSResp, error) {
	panic("unimplemented")
}

// ListRdsResources implements RdsDAO.
func (r *treeRdsDAO) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error) {
	panic("unimplemented")
}
