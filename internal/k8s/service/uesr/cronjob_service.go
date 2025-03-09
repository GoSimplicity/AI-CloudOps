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

package uesr

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/uesr"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type CronjobService interface {
	CreateCronjobOne(ctx context.Context, cornjob model.K8sCronjob) error
	GetCronjobList(ctx context.Context) ([]*model.K8sCronjob, error)
	GetCronjobOne(ctx context.Context, id int64) (*model.K8sCronjob, error)
	UpdateCronjobOne(ctx context.Context, id int64, cornjob model.K8sCronjob) error
}
type cronjobService struct {
	dao           admin.ClusterDAO
	k8scornjobDAO uesr.CornJobDAO
	client        client.K8sClient
	l             *zap.Logger
}

func NewCronjobService(dao admin.ClusterDAO, k8scornjobDAO uesr.CornJobDAO, client client.K8sClient, l *zap.Logger) CronjobService {
	return &cronjobService{
		dao:           dao,
		k8scornjobDAO: k8scornjobDAO,
		client:        client,
		l:             l,
	}
}
func (c *cronjobService) CreateCronjobOne(ctx context.Context, job model.K8sCronjob) error {

	err := c.k8scornjobDAO.CreateCornJobOne(ctx, &job)
	if err != nil {
		return errors.New("Create Cronjob One fail")
	}
	// 构建cornjob
	K8sCluster, err := c.dao.GetClusterByName(ctx, job.Cluster)
	if err != nil {
		return errors.New("Create Cronjob One Get Cluster By Name fail")
	}
	err = pkg.CreateCornJob(ctx, K8sCluster.ID, job, c.client, c.l)
	if err != nil {
		return errors.New("pkg Create Cron job  fail")
	}
	return nil
}
func (c *cronjobService) GetCronjobList(ctx context.Context) ([]*model.K8sCronjob, error) {
	return c.k8scornjobDAO.GetCronjobList(ctx)
}
func (c *cronjobService) GetCronjobOne(ctx context.Context, id int64) (*model.K8sCronjob, error) {
	return c.k8scornjobDAO.GetCronjobOne(ctx, id)
}
func (c *cronjobService) UpdateCronjobOne(ctx context.Context, id int64, cornjob model.K8sCronjob) error {
	// 更新数据库
	c.k8scornjobDAO.UpdateCronjobOne(ctx, id, cornjob)
	// 实例更新
	K8sCluster, err := c.dao.GetClusterByName(ctx, cornjob.Cluster)
	if err != nil {
		return errors.New("Create Cronjob One Get Cluster By Name fail")
	}
	err = pkg.UpdateCronJob(ctx, K8sCluster.ID, cornjob, c.client, c.l)
	if err != nil {
		return errors.New("pkg Update Cron job  fail")
	}
	return nil
}
