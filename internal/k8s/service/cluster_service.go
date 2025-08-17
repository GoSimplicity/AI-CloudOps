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

package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"go.uber.org/zap"
)

type ClusterService interface {
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error
	DeleteCluster(ctx context.Context, id int) error
	BatchDeleteClusters(ctx context.Context, ids []int) error
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
	RefreshClusterStatus(ctx context.Context, id int) error
}

type clusterService struct {
	dao        dao.ClusterDAO
	client     client.K8sClient
	clusterMgr manager.ClusterManager
	l          *zap.Logger
}

func NewClusterService(dao dao.ClusterDAO, client client.K8sClient, clusterMgr manager.ClusterManager, l *zap.Logger) ClusterService {
	return &clusterService{
		dao:        dao,
		client:     client,
		clusterMgr: clusterMgr,
		l:          l,
	}
}

// ListAllClusters 获取所有 Kubernetes 集群
func (c *clusterService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	list, err := c.dao.ListAllClusters(ctx)
	if err != nil {
		c.l.Error("ListAllClusters: 查询所有集群失败", zap.Error(err))
		return nil, fmt.Errorf("查询所有集群失败: %w", err)
	}

	return c.buildListResponse(list), nil
}

// GetClusterByID 根据 ID 获取单个 Kubernetes 集群
func (c *clusterService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	cluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("GetClusterByID: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}
	return cluster, nil
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
		c.l.Error("CreateCluster: 集群已存在", zap.String("clusterName", cluster.Name))
		return fmt.Errorf("集群名称 %s 已存在", cluster.Name)
	}

	cluster.Status = "PENDING"

	// 创建集群记录
	if err := c.dao.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 使用集群管理器创建集群
	if err := c.clusterMgr.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群失败", zap.Error(err))
		return fmt.Errorf("创建集群失败: %w", err)
	}

	c.l.Info("CreateCluster: 成功创建 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// UpdateCluster 更新指定 ID 的 Kubernetes 集群
func (c *clusterService) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群参数不能为空")
	}

	// 检查集群是否存在
	existingCluster, err := c.dao.GetClusterByID(ctx, cluster.ID)
	if err != nil {
		c.l.Error("UpdateCluster: 查询集群失败", zap.Int("clusterID", cluster.ID), zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", cluster.ID)
	}

	// 更新集群记录
	if err := c.dao.UpdateCluster(ctx, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群失败", zap.Error(err), zap.Int("clusterID", cluster.ID))
		return fmt.Errorf("更新集群失败: %w", err)
	}

	// 使用集群管理器更新集群
	if err := c.clusterMgr.UpdateCluster(ctx, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群客户端失败", zap.Error(err))
		return fmt.Errorf("更新集群客户端失败: %w", err)
	}

	c.l.Info("UpdateCluster: 成功更新 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// DeleteCluster 删除指定 ID 的 Kubernetes 集群
func (c *clusterService) DeleteCluster(ctx context.Context, id int) error {
	// 检查集群是否存在
	existingCluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("DeleteCluster: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", id)
	}

	// 删除集群客户端
	c.client.RemoveCluster(id)

	// 删除集群记录
	if err := c.dao.DeleteCluster(ctx, id); err != nil {
		c.l.Error("DeleteCluster: 删除集群失败", zap.Int("clusterID", id), zap.Error(err))
		return fmt.Errorf("删除集群失败: %w", err)
	}

	c.l.Info("DeleteCluster: 成功删除 Kubernetes 集群", zap.Int("clusterID", id))
	return nil
}

// BatchDeleteClusters 批量删除 Kubernetes 集群
func (c *clusterService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return fmt.Errorf("删除ID列表不能为空")
	}

	// 删除集群客户端
	for _, id := range ids {
		c.client.RemoveCluster(id)
	}

	if err := c.dao.BatchDeleteClusters(ctx, ids); err != nil {
		c.l.Error("BatchDeleteClusters: 批量删除集群失败", zap.Ints("clusterIDs", ids), zap.Error(err))
		return fmt.Errorf("批量删除集群失败: %w", err)
	}

	c.l.Info("BatchDeleteClusters: 成功批量删除 Kubernetes 集群", zap.Ints("clusterIDs", ids))
	return nil
}

// RefreshClusterStatus 刷新集群状态
func (c *clusterService) RefreshClusterStatus(ctx context.Context, id int) error {
	cluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("RefreshClusterStatus: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if cluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", id)
	}

	// 使用集群管理器刷新集群状态
	if err := c.clusterMgr.RefreshCluster(ctx, id); err != nil {
		c.l.Error("RefreshClusterStatus: 刷新集群状态失败", zap.Error(err))
		return fmt.Errorf("刷新集群状态失败: %w", err)
	}

	c.l.Info("RefreshClusterStatus: 成功刷新集群状态", zap.Int("clusterID", id))
	return nil
}

func (c *clusterService) buildListResponse(clusters []*model.K8sCluster) []*model.K8sCluster {
	result := make([]*model.K8sCluster, len(clusters))
	for i, cluster := range clusters {
		clusterCopy := *cluster
		clusterCopy.KubeConfigContent = ""
		result[i] = &clusterCopy
	}

	return result
}
