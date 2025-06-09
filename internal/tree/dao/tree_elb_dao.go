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

type TreeElbDAO interface {
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error)
	GetElbResourceById(ctx context.Context, id int) (*model.ResourceElb, error)
	GetElbResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceElb, error)
	CreateElbResource(ctx context.Context, resource *model.ResourceElb) error
	UpdateElbResource(ctx context.Context, resource *model.ResourceElb) error
	DeleteElbResource(ctx context.Context, id int) error

	// 服务器绑定管理
	GetElbHealthCheck(ctx context.Context, elbId int) (*model.ElbHealthCheck, error)
	CreateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error
	UpdateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error
	GetElbResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceElb, error)
	GetElbResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceElb, error)
	GetElbResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceElb, error)
	GetElbResourcesByVpcId(ctx context.Context, vpcId string) ([]*model.ResourceElb, error)

	CountElbResourcesByProvider(ctx context.Context, provider string) (int64, error)
	CountElbResourcesByRegion(ctx context.Context, region string) (int64, error)
	CountElbResourcesByStatus(ctx context.Context, status string) (int64, error)

	BatchDeleteElbResources(ctx context.Context, ids []int) error

	WithTx(tx *gorm.DB) TreeElbDAO

	GetElbListeners(ctx context.Context, elbId int) ([]*model.ElbListener, error)
	CreateElbListener(ctx context.Context, listener *model.ElbListener) error
	UpdateElbListener(ctx context.Context, listener *model.ElbListener) error
	DeleteElbListener(ctx context.Context, listenerId int) error

	GetElbRules(ctx context.Context, listenerId int) ([]*model.ElbRule, error)
	CreateElbRule(ctx context.Context, rule *model.ElbRule) error
	UpdateElbRule(ctx context.Context, rule *model.ElbRule) error
	DeleteElbRule(ctx context.Context, ruleId int) error
}

type treeElbDAO struct {
	db *gorm.DB
}

func NewTreeElbDAO(db *gorm.DB) TreeElbDAO {
	return &treeElbDAO{
		db: db,
	}
}

// BatchDeleteElbResources implements TreeElbDAO.
func (t *treeElbDAO) BatchDeleteElbResources(ctx context.Context, ids []int) error {
	panic("unimplemented")
}

// CountElbResourcesByProvider implements TreeElbDAO.
func (t *treeElbDAO) CountElbResourcesByProvider(ctx context.Context, provider string) (int64, error) {
	panic("unimplemented")
}

// CountElbResourcesByRegion implements TreeElbDAO.
func (t *treeElbDAO) CountElbResourcesByRegion(ctx context.Context, region string) (int64, error) {
	panic("unimplemented")
}

// CountElbResourcesByStatus implements TreeElbDAO.
func (t *treeElbDAO) CountElbResourcesByStatus(ctx context.Context, status string) (int64, error) {
	panic("unimplemented")
}

// CreateElbHealthCheck implements TreeElbDAO.
func (t *treeElbDAO) CreateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error {
	panic("unimplemented")
}

// CreateElbListener implements TreeElbDAO.
func (t *treeElbDAO) CreateElbListener(ctx context.Context, listener *model.ElbListener) error {
	panic("unimplemented")
}

// CreateElbResource implements TreeElbDAO.
func (t *treeElbDAO) CreateElbResource(ctx context.Context, resource *model.ResourceElb) error {
	panic("unimplemented")
}

// CreateElbRule implements TreeElbDAO.
func (t *treeElbDAO) CreateElbRule(ctx context.Context, rule *model.ElbRule) error {
	panic("unimplemented")
}

// DeleteElbListener implements TreeElbDAO.
func (t *treeElbDAO) DeleteElbListener(ctx context.Context, listenerId int) error {
	panic("unimplemented")
}

// DeleteElbResource implements TreeElbDAO.
func (t *treeElbDAO) DeleteElbResource(ctx context.Context, id int) error {
	panic("unimplemented")
}

// DeleteElbRule implements TreeElbDAO.
func (t *treeElbDAO) DeleteElbRule(ctx context.Context, ruleId int) error {
	panic("unimplemented")
}

// GetElbHealthCheck implements TreeElbDAO.
func (t *treeElbDAO) GetElbHealthCheck(ctx context.Context, elbId int) (*model.ElbHealthCheck, error) {
	panic("unimplemented")
}

// GetElbListeners implements TreeElbDAO.
func (t *treeElbDAO) GetElbListeners(ctx context.Context, elbId int) ([]*model.ElbListener, error) {
	panic("unimplemented")
}

// GetElbResourceById implements TreeElbDAO.
func (t *treeElbDAO) GetElbResourceById(ctx context.Context, id int) (*model.ResourceElb, error) {
	panic("unimplemented")
}

// GetElbResourceByInstanceId implements TreeElbDAO.
func (t *treeElbDAO) GetElbResourceByInstanceId(ctx context.Context, instanceId string) (*model.ResourceElb, error) {
	panic("unimplemented")
}

// GetElbResourcesByProvider implements TreeElbDAO.
func (t *treeElbDAO) GetElbResourcesByProvider(ctx context.Context, provider string) ([]*model.ResourceElb, error) {
	panic("unimplemented")
}

// GetElbResourcesByRegion implements TreeElbDAO.
func (t *treeElbDAO) GetElbResourcesByRegion(ctx context.Context, region string) ([]*model.ResourceElb, error) {
	panic("unimplemented")
}

// GetElbResourcesByStatus implements TreeElbDAO.
func (t *treeElbDAO) GetElbResourcesByStatus(ctx context.Context, status string) ([]*model.ResourceElb, error) {
	panic("unimplemented")
}

// GetElbResourcesByVpcId implements TreeElbDAO.
func (t *treeElbDAO) GetElbResourcesByVpcId(ctx context.Context, vpcId string) ([]*model.ResourceElb, error) {
	panic("unimplemented")
}

// GetElbRules implements TreeElbDAO.
func (t *treeElbDAO) GetElbRules(ctx context.Context, listenerId int) ([]*model.ElbRule, error) {
	panic("unimplemented")
}

// ListElbResources implements TreeElbDAO.
func (t *treeElbDAO) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error) {
	panic("unimplemented")
}

// UpdateElbHealthCheck implements TreeElbDAO.
func (t *treeElbDAO) UpdateElbHealthCheck(ctx context.Context, healthCheck *model.ElbHealthCheck) error {
	panic("unimplemented")
}

// UpdateElbListener implements TreeElbDAO.
func (t *treeElbDAO) UpdateElbListener(ctx context.Context, listener *model.ElbListener) error {
	panic("unimplemented")
}

// UpdateElbResource implements TreeElbDAO.
func (t *treeElbDAO) UpdateElbResource(ctx context.Context, resource *model.ResourceElb) error {
	panic("unimplemented")
}

// UpdateElbRule implements TreeElbDAO.
func (t *treeElbDAO) UpdateElbRule(ctx context.Context, rule *model.ElbRule) error {
	panic("unimplemented")
}

// WithTx implements TreeElbDAO.
func (t *treeElbDAO) WithTx(tx *gorm.DB) TreeElbDAO {
	panic("unimplemented")
}
