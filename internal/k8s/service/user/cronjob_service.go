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

package user

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type CronjobService interface {
	CreateCronjob(ctx context.Context, req *model.CreateK8sCronjobRequest) error
	GetCronjobList(ctx context.Context, req *model.GetK8sCronjobListRequest) ([]*model.K8sCronjob, error)
	GetCronjob(ctx context.Context, id int64) (*model.K8sCronjob, error)
	UpdateCronjob(ctx context.Context, req *model.UpdateK8sCronjobRequest) error
	BatchDeleteCronjob(ctx context.Context, ids []int64) error
	GetCronjobLastPod(ctx context.Context, id int64) (model.K8sPod, error)
}
type cronjobService struct {
	dao           admin.ClusterDAO
	k8scornjobDAO user.CornJobDAO
	client        client.K8sClient
	l             *zap.Logger
}

func NewCronjobService(dao admin.ClusterDAO, k8scornjobDAO user.CornJobDAO, client client.K8sClient, l *zap.Logger) CronjobService {
	return &cronjobService{
		dao:           dao,
		k8scornjobDAO: k8scornjobDAO,
		client:        client,
		l:             l,
	}
}

// BatchDeleteCronjob implements CronjobService.
func (c *cronjobService) BatchDeleteCronjob(ctx context.Context, ids []int64) error {
	panic("unimplemented")
}

// CreateCronjob implements CronjobService.
func (c *cronjobService) CreateCronjob(ctx context.Context, req *model.CreateK8sCronjobRequest) error {
	panic("unimplemented")
}

// GetCronjob implements CronjobService.
func (c *cronjobService) GetCronjob(ctx context.Context, id int64) (*model.K8sCronjob, error) {
	panic("unimplemented")
}

// GetCronjobLastPod implements CronjobService.
func (c *cronjobService) GetCronjobLastPod(ctx context.Context, id int64) (model.K8sPod, error) {
	panic("unimplemented")
}

// GetCronjobList implements CronjobService.
func (c *cronjobService) GetCronjobList(ctx context.Context, req *model.GetK8sCronjobListRequest) ([]*model.K8sCronjob, error) {
	panic("unimplemented")
}

// UpdateCronjob implements CronjobService.
func (c *cronjobService) UpdateCronjob(ctx context.Context, req *model.UpdateK8sCronjobRequest) error {
	panic("unimplemented")
}
