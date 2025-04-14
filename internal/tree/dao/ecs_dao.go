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

type EcsDAO interface {
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (*model.PageResp, error)
	GetEcsResourceById(ctx context.Context, id int) (*model.ResourceECSResp, error)
	CreateEcsResource(ctx context.Context, params *model.EcsCreationParams) error
}

type ecsDAO struct {
	db *gorm.DB
}

func NewEcsDAO(db *gorm.DB) EcsDAO {
	return &ecsDAO{
		db: db,
	}
}

// CreateEcsResource implements EcsDAO.
func (e *ecsDAO) CreateEcsResource(ctx context.Context, params *model.EcsCreationParams) error {
	if err := e.db.Create(params).Error; err != nil {
		return err
	}
	return nil
}

// GetEcsResourceById implements EcsDAO.
func (e *ecsDAO) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceECSResp, error) {
	panic("unimplemented")
}

// ListEcsResources implements EcsDAO.
func (e *ecsDAO) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}
