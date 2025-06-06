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

type TreeEcsDAO interface {
	// 基础CRUD操作
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, int64, error)
	GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error)
	GetEcsResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceEcs, error)
	CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error
	UpdateEcsResource(ctx context.Context, resource *model.ResourceEcs) error
	DeleteEcsResource(ctx context.Context, instanceId string) error

	// 状态更新操作
	UpdateEcsStatus(ctx context.Context, instanceId string, status string) error
	UpdateEcsPassword(ctx context.Context, instanceId string, passwordHash string) error
	UpdateEcsConfiguration(ctx context.Context, instanceId string, cpu int, memory int, diskSize int) error
	UpdateEcsRenewalInfo(ctx context.Context, instanceId string, expireTime string, renewalDuration int) error

	// 查询操作
	GetEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) ([]*model.ListEcsResourceOptionsResp, int64, error)
	GetEcsResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceEcs, error)
	GetEcsResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceEcs, error)
	GetEcsResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceEcs, error)

	// 批量操作
	BatchUpdateEcsStatus(ctx context.Context, instanceIds []string, status string) error
	BatchDeleteEcsResources(ctx context.Context, instanceIds []string) error

	// 统计操作
	CountEcsResourcesByProvider(ctx context.Context, provider string) (int64, error)
	CountEcsResourcesByRegion(ctx context.Context, region string) (int64, error)
	CountEcsResourcesByStatus(ctx context.Context, status string) (int64, error)

	// 事务操作
	WithTx(tx *gorm.DB) TreeEcsDAO
}

type treeEcsDAO struct {
	db *gorm.DB
}

func NewTreeEcsDAO(db *gorm.DB) TreeEcsDAO {
	return &treeEcsDAO{
		db: db,
	}
}

// BatchDeleteEcsResources implements TreeEcsDAO.
func (t *treeEcsDAO) BatchDeleteEcsResources(ctx context.Context, instanceIds []string) error {
	panic("unimplemented")
}

// BatchUpdateEcsStatus implements TreeEcsDAO.
func (t *treeEcsDAO) BatchUpdateEcsStatus(ctx context.Context, instanceIds []string, status string) error {
	panic("unimplemented")
}

// CountEcsResourcesByProvider implements TreeEcsDAO.
func (t *treeEcsDAO) CountEcsResourcesByProvider(ctx context.Context, provider string) (int64, error) {
	panic("unimplemented")
}

// CountEcsResourcesByRegion implements TreeEcsDAO.
func (t *treeEcsDAO) CountEcsResourcesByRegion(ctx context.Context, region string) (int64, error) {
	panic("unimplemented")
}

// CountEcsResourcesByStatus implements TreeEcsDAO.
func (t *treeEcsDAO) CountEcsResourcesByStatus(ctx context.Context, status string) (int64, error) {
	panic("unimplemented")
}

// CreateEcsResource implements TreeEcsDAO.
func (t *treeEcsDAO) CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	panic("unimplemented")
}

// DeleteEcsResource implements TreeEcsDAO.
func (t *treeEcsDAO) DeleteEcsResource(ctx context.Context, instanceId string) error {
	panic("unimplemented")
}

// GetEcsResourceById implements TreeEcsDAO.
func (t *treeEcsDAO) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error) {
	panic("unimplemented")
}

// GetEcsResourceByInstanceId implements TreeEcsDAO.
func (t *treeEcsDAO) GetEcsResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceEcs, error) {
	panic("unimplemented")
}

// GetEcsResourceOptions implements TreeEcsDAO.
func (t *treeEcsDAO) GetEcsResourceOptions(ctx context.Context, req *model.ListEcsResourceOptionsReq) ([]*model.ListEcsResourceOptionsResp, int64, error) {
	panic("unimplemented")
}

// GetEcsResourcesByProvider implements TreeEcsDAO.
func (t *treeEcsDAO) GetEcsResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceEcs, error) {
	panic("unimplemented")
}

// GetEcsResourcesByRegion implements TreeEcsDAO.
func (t *treeEcsDAO) GetEcsResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceEcs, error) {
	panic("unimplemented")
}

// GetEcsResourcesByStatus implements TreeEcsDAO.
func (t *treeEcsDAO) GetEcsResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceEcs, error) {
	panic("unimplemented")
}

// ListEcsResources implements TreeEcsDAO.
func (t *treeEcsDAO) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, int64, error) {
	panic("unimplemented")
}

// UpdateEcsConfiguration implements TreeEcsDAO.
func (t *treeEcsDAO) UpdateEcsConfiguration(ctx context.Context, instanceId string, cpu int, memory int, diskSize int) error {
	panic("unimplemented")
}

// UpdateEcsPassword implements TreeEcsDAO.
func (t *treeEcsDAO) UpdateEcsPassword(ctx context.Context, instanceId string, passwordHash string) error {
	panic("unimplemented")
}

// UpdateEcsRenewalInfo implements TreeEcsDAO.
func (t *treeEcsDAO) UpdateEcsRenewalInfo(ctx context.Context, instanceId string, expireTime string, renewalDuration int) error {
	panic("unimplemented")
}

// UpdateEcsResource implements TreeEcsDAO.
func (t *treeEcsDAO) UpdateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	panic("unimplemented")
}

// UpdateEcsStatus implements TreeEcsDAO.
func (t *treeEcsDAO) UpdateEcsStatus(ctx context.Context, instanceId string, status string) error {
	panic("unimplemented")
}

// WithTx implements TreeEcsDAO.
func (t *treeEcsDAO) WithTx(tx *gorm.DB) TreeEcsDAO {
	panic("unimplemented")
}
