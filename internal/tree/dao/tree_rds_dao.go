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
	// 基础CRUD操作
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error)
	GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRds, error)
	CreateRdsResource(ctx context.Context, params *model.CreateRdsResourceReq) error
	UpdateRdsResource(ctx context.Context, id int, req *model.UpdateRdsReq) error
	DeleteRdsResource(ctx context.Context, id int) error

	// RDS实例状态操作
	StartRdsInstance(ctx context.Context, id int, req *model.StartRdsReq) error
	StopRdsInstance(ctx context.Context, id int, req *model.StopRdsReq) error
	RestartRdsInstance(ctx context.Context, id int, req *model.RestartRdsReq) error

	// RDS实例管理操作
	ResizeRdsInstance(ctx context.Context, id int, req *model.ResizeRdsReq) error
	ResetRdsPassword(ctx context.Context, id int, req *model.ResetRdsPasswordReq) error
	RenewRdsInstance(ctx context.Context, id int, req *model.RenewRdsReq) error

	// RDS备份恢复操作
	BackupRdsInstance(ctx context.Context, id int, req *model.BackupRdsReq) error
	RestoreRdsInstance(ctx context.Context, id int, req *model.RestoreRdsReq) error

	// 辅助查询方法
	GetRdsInstanceStatus(ctx context.Context, id int) (string, error)
	CheckRdsInstanceExists(ctx context.Context, id int) (bool, error)
	UpdateRdsInstanceStatus(ctx context.Context, id int, status string) error

	// 批量操作
	BatchUpdateRdsStatus(ctx context.Context, ids []int, status string) error
	GetRdsInstancesByStatus(ctx context.Context, status string) ([]*model.ResourceRds, error)
}

type treeRdsDAO struct {
	db *gorm.DB
}

func NewTreeRdsDAO(db *gorm.DB) TreeRdsDAO {
	return &treeRdsDAO{
		db: db,
	}
}

// BackupRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) BackupRdsInstance(ctx context.Context, id int, req *model.BackupRdsReq) error {
	panic("unimplemented")
}

// BatchUpdateRdsStatus implements TreeRdsDAO.
func (t *treeRdsDAO) BatchUpdateRdsStatus(ctx context.Context, ids []int, status string) error {
	panic("unimplemented")
}

// CheckRdsInstanceExists implements TreeRdsDAO.
func (t *treeRdsDAO) CheckRdsInstanceExists(ctx context.Context, id int) (bool, error) {
	panic("unimplemented")
}

// CreateRdsResource implements TreeRdsDAO.
func (t *treeRdsDAO) CreateRdsResource(ctx context.Context, params *model.CreateRdsResourceReq) error {
	panic("unimplemented")
}

// DeleteRdsResource implements TreeRdsDAO.
func (t *treeRdsDAO) DeleteRdsResource(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetRdsInstanceStatus implements TreeRdsDAO.
func (t *treeRdsDAO) GetRdsInstanceStatus(ctx context.Context, id int) (string, error) {
	panic("unimplemented")
}

// GetRdsInstancesByStatus implements TreeRdsDAO.
func (t *treeRdsDAO) GetRdsInstancesByStatus(ctx context.Context, status string) ([]*model.ResourceRds, error) {
	panic("unimplemented")
}

// GetRdsResourceById implements TreeRdsDAO.
func (t *treeRdsDAO) GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRds, error) {
	panic("unimplemented")
}

// ListRdsResources implements TreeRdsDAO.
func (t *treeRdsDAO) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error) {
	panic("unimplemented")
}

// RenewRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) RenewRdsInstance(ctx context.Context, id int, req *model.RenewRdsReq) error {
	panic("unimplemented")
}

// ResetRdsPassword implements TreeRdsDAO.
func (t *treeRdsDAO) ResetRdsPassword(ctx context.Context, id int, req *model.ResetRdsPasswordReq) error {
	panic("unimplemented")
}

// ResizeRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) ResizeRdsInstance(ctx context.Context, id int, req *model.ResizeRdsReq) error {
	panic("unimplemented")
}

// RestartRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) RestartRdsInstance(ctx context.Context, id int, req *model.RestartRdsReq) error {
	panic("unimplemented")
}

// RestoreRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) RestoreRdsInstance(ctx context.Context, id int, req *model.RestoreRdsReq) error {
	panic("unimplemented")
}

// StartRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) StartRdsInstance(ctx context.Context, id int, req *model.StartRdsReq) error {
	panic("unimplemented")
}

// StopRdsInstance implements TreeRdsDAO.
func (t *treeRdsDAO) StopRdsInstance(ctx context.Context, id int, req *model.StopRdsReq) error {
	panic("unimplemented")
}

// UpdateRdsInstanceStatus implements TreeRdsDAO.
func (t *treeRdsDAO) UpdateRdsInstanceStatus(ctx context.Context, id int, status string) error {
	panic("unimplemented")
}

// UpdateRdsResource implements TreeRdsDAO.
func (t *treeRdsDAO) UpdateRdsResource(ctx context.Context, id int, req *model.UpdateRdsReq) error {
	panic("unimplemented")
}
