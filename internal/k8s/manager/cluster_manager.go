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

package manager

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type ClusterManager interface {
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error
	RefreshCluster(ctx context.Context, clusterID int) error
	RefreshAllClusters(ctx context.Context) error
	InitializeAllClusters(ctx context.Context) error
	CheckClusterStatus(ctx context.Context, clusterID int) error
}

type clusterManager struct {
	client client.K8sClient
	dao    dao.ClusterDAO
	logger *zap.Logger
}

func NewClusterManager(logger *zap.Logger, client client.K8sClient, dao dao.ClusterDAO) ClusterManager {
	return &clusterManager{
		client: client,
		dao:    dao,
		logger: logger,
	}
}

func (cm *clusterManager) CreateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群配置不能为空")
	}

	// 验证资源配额格式
	if err := utils.ValidateResourceQuantities(cluster); err != nil {
		cm.logger.Warn("资源配额格式验证失败", zap.Error(err))
	}

	// 初始化k8s客户端
	client, err := cm.client.GetKubeClient(cluster.ID)
	if err != nil {
		cm.dao.UpdateClusterStatus(ctx, cluster.ID, model.StatusError)
		return fmt.Errorf("初始化客户端失败: %w", err)
	}

	// 添加集群资源限制
	if err := utils.AddClusterResourceLimit(ctx, client, cluster); err != nil {
		cm.logger.Warn("添加集群资源限制失败", zap.Error(err))
	}

	// 更新集群状态为运行中
	cm.dao.UpdateClusterStatus(ctx, cluster.ID, model.StatusRunning)

	cm.logger.Info("创建集群成功", zap.Int("clusterID", cluster.ID))
	return nil
}

func (cm *clusterManager) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群配置不能为空")
	}

	// 验证资源配额格式
	if err := utils.ValidateResourceQuantities(cluster); err != nil {
		cm.logger.Warn("资源配额验证失败", zap.Error(err))
	}

	// 先移除旧的客户端
	cm.client.RemoveCluster(cluster.ID)

	// 重新初始化k8s客户端
	client, err := cm.client.GetKubeClient(cluster.ID)
	if err != nil {
		cm.logger.Error("初始化客户端失败", zap.Error(err))
		return fmt.Errorf("初始化客户端失败: %w", err)
	}

	// 添加集群资源限制
	if err := utils.AddClusterResourceLimit(ctx, client, cluster); err != nil {
		cm.logger.Warn("添加集群资源限制失败", zap.Error(err))
	}

	return nil
}

func (cm *clusterManager) RefreshCluster(ctx context.Context, clusterID int) error {
	cluster, err := cm.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		cm.logger.Error("获取集群信息失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return err
	}

	if cluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", clusterID)
	}

	// 检查集群连接
	if err := cm.client.CheckClusterConnection(clusterID); err != nil {
		cm.logger.Error("集群连接检查失败", zap.Int("clusterID", clusterID), zap.Error(err))
		cm.dao.UpdateClusterStatus(ctx, clusterID, model.StatusError)
		return fmt.Errorf("集群连接检查失败: %w", err)
	}

	// 更新集群状态
	if err := cm.dao.UpdateClusterStatus(ctx, clusterID, model.StatusRunning); err != nil {
		cm.logger.Error("更新集群状态失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return fmt.Errorf("更新集群状态失败: %w", err)
	}

	return nil
}

func (cm *clusterManager) RefreshAllClusters(ctx context.Context) error {
	return cm.client.RefreshClients(ctx)
}

func (cm *clusterManager) InitializeAllClusters(ctx context.Context) error {
	page := 1
	size := 10

	for {
		clusters, total, err := cm.dao.GetClusterList(ctx, &model.ListClustersReq{
			ListReq: model.ListReq{
				Page: page,
				Size: size,
			},
		})
		if err != nil {
			cm.logger.Error("获取所有集群失败", zap.Error(err))
			return err
		}

		// 如果没有集群了，退出循环
		if len(clusters) == 0 {
			break
		}

		for _, cluster := range clusters {
			if cluster.KubeConfigContent == "" {
				cm.logger.Warn("集群的 KubeConfig 内容为空，跳过初始化", zap.Int("ClusterID", cluster.ID))
				continue
			}

			_, err := cm.client.GetKubeClient(cluster.ID)
			if err != nil {
				cm.logger.Error("初始化 Kubernetes 客户端失败", zap.Int("ClusterID", cluster.ID), zap.Error(err))
				continue
			}
		}

		// 如果已经处理完所有集群，退出循环
		if int64(page*size) >= total {
			break
		}

		page++
	}

	return nil
}

func (cm *clusterManager) CheckClusterStatus(ctx context.Context, clusterID int) error {
	return cm.client.CheckClusterConnection(clusterID)
}
