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

package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/GoSimplicity/AI-CloudOps/internal/job"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"go.uber.org/zap"
)

type ClusterService interface {
	// ListAllClusters 获取所有 Kubernetes 集群
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	// CreateCluster 创建一个新的 Kubernetes 集群
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	// UpdateCluster 更新指定 ID 的 Kubernetes 集群
	UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error
	// DeleteCluster 删除指定 ID 的 Kubernetes 集群
	DeleteCluster(ctx context.Context, id int) error
	// BatchDeleteClusters 批量删除 Kubernetes 集群
	BatchDeleteClusters(ctx context.Context, ids []int) error
	// GetClusterByID 根据 ID 获取单个 Kubernetes 集群
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
}

type clusterService struct {
	dao         admin.ClusterDAO
	client      client.K8sClient
	asynqClient *asynq.Client
	l           *zap.Logger
}

func NewClusterService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger, asynqClient *asynq.Client) ClusterService {
	return &clusterService{
		dao:         dao,
		client:      client,
		asynqClient: asynqClient,
		l:           l,
	}
}

// ListAllClusters 获取所有 Kubernetes 集群
func (c *clusterService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	list, err := c.dao.ListAllClusters(ctx)
	if err != nil {
		c.l.Error("ListAllClusters: 查询所有集群失败", zap.Error(err))
		return nil, err
	}

	return c.buildListResponse(list), nil
}

// GetClusterByID 根据 ID 获取单个 Kubernetes 集群
func (c *clusterService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	return c.dao.GetClusterByID(ctx, id)
}

// CreateCluster 创建一个新的 Kubernetes 集群
func (c *clusterService) CreateCluster(ctx context.Context, cluster *model.K8sCluster) (err error) {
	// 检查集群是否存在
	existingCluster, err := c.dao.GetClusterByName(ctx, cluster.Name)
	if err != nil && err != gorm.ErrRecordNotFound {
		c.l.Error("CreateCluster: 查询集群失败", zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster != nil {
		c.l.Error("CreateCluster: 集群已存在", zap.Int("clusterID", cluster.ID))
		return fmt.Errorf("集群已存在: %w", err)
	}

	cluster.Status = "PENDING"

	// 创建集群记录
	if err := c.dao.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 放入异步任务队列
	payload := job.CreateK8sClusterPayload{
		Cluster: cluster,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		c.l.Error("CreateCluster: 序列化任务载荷失败", zap.Error(err))
		return fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	task := asynq.NewTask(job.DeferCreateK8sCluster, jsonPayload)
	if _, err := c.asynqClient.Enqueue(task); err != nil {
		c.l.Error("CreateCluster: 放入异步任务队列失败", zap.Error(err))
		return fmt.Errorf("放入异步任务队列失败: %w", err)
	}

	c.l.Info("CreateCluster: 成功创建 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// UpdateCluster 更新指定 ID 的 Kubernetes 集群
func (c *clusterService) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	// 更新集群记录
	if err := c.dao.UpdateCluster(ctx, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群失败", zap.Error(err))
		return fmt.Errorf("更新集群失败: %w", err)
	}

	// 放入异步任务队列
	payload := job.UpdateK8sClusterPayload{
		Cluster: cluster,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		c.l.Error("UpdateCluster: 序列化任务载荷失败", zap.Error(err))
		return fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	task := asynq.NewTask(job.DeferUpdateK8sCluster, jsonPayload)
	if _, err := c.asynqClient.Enqueue(task); err != nil {
		c.l.Error("UpdateCluster: 放入异步任务队列失败", zap.Error(err))
		return fmt.Errorf("放入异步任务队列失败: %w", err)
	}

	c.l.Info("UpdateCluster: 成功更新 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// DeleteCluster 删除指定 ID 的 Kubernetes 集群
func (c *clusterService) DeleteCluster(ctx context.Context, id int) error {
	return c.dao.DeleteCluster(ctx, id)
}

// BatchDeleteClusters 批量删除 Kubernetes 集群
func (c *clusterService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	return c.dao.BatchDeleteClusters(ctx, ids)
}

func (c *clusterService) buildListResponse(clusters []*model.K8sCluster) []*model.K8sCluster {
	for _, cluster := range clusters {
		cluster.KubeConfigContent = ""
	}

	return clusters
}
