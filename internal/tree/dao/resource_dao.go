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

type ResourceDAO interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)

	SyncResources(ctx context.Context, provider model.CloudProvider, region string) error
	DeleteResource(ctx context.Context, resourceType string, id int) error
	StartResource(ctx context.Context, resourceType string, id int) error
	StopResource(ctx context.Context, resourceType string, id int) error
	RestartResource(ctx context.Context, resourceType string, id int) error
	GetResourceById(ctx context.Context, resourceType string, id string) (*model.ResourceBase, error)
	SaveOrUpdateResource(ctx context.Context, resource interface{}) error

	SaveRegion(ctx context.Context, tx *gorm.DB, region string) error
	SaveZones(ctx context.Context, tx *gorm.DB, region string, azs []string) error
	SaveInstanceTypes(ctx context.Context, tx *gorm.DB, region string, instanceTypes []string) error
	SaveCombinations(ctx context.Context, tx *gorm.DB, combinations []string) error
}

type resourceDAO struct {
	db *gorm.DB
}

func NewResourceDAO(db *gorm.DB) ResourceDAO {
	return &resourceDAO{
		db: db,
	}
}

// DeleteResource implements ResourceDAO.
func (r *resourceDAO) DeleteResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// RestartResource implements ResourceDAO.
func (r *resourceDAO) RestartResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// StartResource implements ResourceDAO.
func (r *resourceDAO) StartResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// StopResource implements ResourceDAO.
func (r *resourceDAO) StopResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// SyncResources implements ResourceDAO.
func (r *resourceDAO) SyncResources(ctx context.Context, provider model.CloudProvider, region string) error {
	panic("unimplemented")
}

// GetResourceById implements ResourceDAO.
func (r *resourceDAO) GetResourceById(ctx context.Context, resourceType string, id string) (*model.ResourceBase, error) {
	panic("unimplemented")
}

// SaveOrUpdateResource implements ResourceDAO.
func (r *resourceDAO) SaveOrUpdateResource(ctx context.Context, resource interface{}) error {
	panic("unimplemented")
}

// BeginTx 实现 ResourceDAO 接口，开启数据库事务
func (r *resourceDAO) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

// SaveCombinations implements ResourceDAO.
func (r *resourceDAO) SaveCombinations(ctx context.Context, tx *gorm.DB, combinations []string) error {
	panic("unimplemented")
}

// SaveInstanceTypes implements ResourceDAO.
func (r *resourceDAO) SaveInstanceTypes(ctx context.Context, tx *gorm.DB, region string, instanceTypes []string) error {
	panic("unimplemented")
}

// SaveRegion implements ResourceDAO.
func (r *resourceDAO) SaveRegion(ctx context.Context, tx *gorm.DB, region string) error {
	panic("unimplemented")
}

// SaveZones implements ResourceDAO.
func (r *resourceDAO) SaveZones(ctx context.Context, tx *gorm.DB, region string, azs []string) error {
	panic("unimplemented")
}
