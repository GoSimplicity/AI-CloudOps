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
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	dao    admin.ClusterDAO
	logger *zap.Logger
}

func NewClusterManager(logger *zap.Logger, client client.K8sClient, dao admin.ClusterDAO) ClusterManager {
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

	const (
		maxRetries     = 5
		baseRetryDelay = 5 * time.Second
		maxConcurrent  = 5
		initTimeout    = 5 * time.Second
	)

	var (
		retryCount int
		lastError  error
	)

	if err := cm.validateResourceQuantities(cluster); err != nil {
		cm.dao.UpdateClusterStatus(ctx, cluster.ID, "ERROR")
		cm.logger.Error("资源配额格式验证失败", zap.Error(err))
		return err
	}

	for retryCount < maxRetries {
		select {
		case <-ctx.Done():
			cm.dao.UpdateClusterStatus(ctx, cluster.ID, "ERROR")
			return ctx.Err()
		default:
			if err := cm.processClusterConfig(ctx, cluster, retryCount, initTimeout, maxConcurrent); err != nil {
				lastError = err
				retryCount++

				if retryCount < maxRetries {
					delay := time.Duration(retryCount) * baseRetryDelay
					cm.logger.Info("任务重试",
						zap.Int("重试次数", retryCount),
						zap.Duration("延迟时间", delay),
						zap.Error(lastError))
					time.Sleep(delay)
					continue
				}

				cm.dao.UpdateClusterStatus(ctx, cluster.ID, "ERROR")
				cm.logger.Error("达到最大重试次数，任务失败",
					zap.Int("最大重试次数", maxRetries),
					zap.Error(lastError))
				return lastError
			}

			cm.dao.UpdateClusterStatus(ctx, cluster.ID, "SUCCESS")
			return nil
		}
	}

	return nil
}

func (cm *clusterManager) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群配置不能为空")
	}

	if err := cm.validateResourceQuantities(cluster); err != nil {
		return fmt.Errorf("资源配额验证失败: %w", err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		return fmt.Errorf("解析kubeconfig失败: %w", err)
	}

	if err := cm.client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		return fmt.Errorf("初始化客户端失败: %w", err)
	}

	return nil
}

func (cm *clusterManager) RefreshCluster(ctx context.Context, clusterID int) error {
	const (
		maxRetries     = 3
		baseRetryDelay = 3 * time.Second
	)

	var (
		retryCount int
		lastError  error
	)

	for retryCount < maxRetries {
		select {
		case <-ctx.Done():
			cm.logger.Error("刷新集群任务被取消", zap.Int("clusterID", clusterID))
			return ctx.Err()
		default:
			cluster, err := cm.dao.GetClusterByID(ctx, clusterID)
			if err != nil {
				lastError = fmt.Errorf("获取集群信息失败: %w", err)
				cm.logger.Error("获取集群信息失败",
					zap.Int("clusterID", clusterID),
					zap.Error(err))
				retryCount++
				if retryCount < maxRetries {
					delay := time.Duration(retryCount) * baseRetryDelay
					cm.logger.Info("任务重试",
						zap.Int("重试次数", retryCount),
						zap.Duration("延迟时间", delay),
						zap.Error(lastError))
					time.Sleep(delay)
					continue
				}
				return lastError
			}

			if cluster == nil {
				cm.logger.Error("集群不存在", zap.Int("clusterID", clusterID))
				return fmt.Errorf("集群不存在，ID: %d", clusterID)
			}

			if err := cm.client.CheckClusterConnection(clusterID); err != nil {
				cm.logger.Error("集群连接检查失败",
					zap.Int("clusterID", clusterID),
					zap.Error(err))

				if updateErr := cm.dao.UpdateClusterStatus(ctx, clusterID, "ERROR"); updateErr != nil {
					cm.logger.Error("更新集群状态失败",
						zap.Int("clusterID", clusterID),
						zap.Error(updateErr))
				}

				lastError = fmt.Errorf("集群连接检查失败: %w", err)
				retryCount++
				if retryCount < maxRetries {
					delay := time.Duration(retryCount) * baseRetryDelay
					cm.logger.Info("任务重试",
						zap.Int("重试次数", retryCount),
						zap.Duration("延迟时间", delay),
						zap.Error(lastError))
					time.Sleep(delay)
					continue
				}
				return lastError
			}

			if err := cm.dao.UpdateClusterStatus(ctx, clusterID, "SUCCESS"); err != nil {
				cm.logger.Error("更新集群状态失败",
					zap.Int("clusterID", clusterID),
					zap.Error(err))
				return fmt.Errorf("更新集群状态失败: %w", err)
			}

			cm.logger.Info("成功刷新集群状态", zap.Int("clusterID", clusterID))
			return nil
		}
	}

	return fmt.Errorf("达到最大重试次数，任务失败: %w", lastError)
}

func (cm *clusterManager) RefreshAllClusters(ctx context.Context) error {
	return cm.client.RefreshClients(ctx)
}

func (cm *clusterManager) InitializeAllClusters(ctx context.Context) error {
	clusters, err := cm.dao.ListAllClusters(ctx)
	if err != nil {
		cm.logger.Error("获取所有集群失败", zap.Error(err))
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(clusters))

	for _, cluster := range clusters {
		if cluster.KubeConfigContent == "" {
			cm.logger.Warn("集群的 KubeConfig 内容为空，跳过初始化", zap.Int("ClusterID", cluster.ID))
			continue
		}

		wg.Add(1)
		go func(c *model.K8sCluster) {
			defer wg.Done()

			restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(c.KubeConfigContent))
			if err != nil {
				cm.logger.Error("解析 kubeconfig 失败", zap.Int("ClusterID", c.ID), zap.Error(err))
				errChan <- fmt.Errorf("解析集群 %d 的 kubeconfig 失败: %w", c.ID, err)
				return
			}

			if err := cm.client.InitClient(ctx, c.ID, restConfig); err != nil {
				cm.logger.Error("初始化 Kubernetes 客户端失败", zap.Int("ClusterID", c.ID), zap.Error(err))
				errChan <- fmt.Errorf("初始化集群 %d 的客户端失败: %w", c.ID, err)
			}
		}(cluster)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("初始化客户端时发生 %d 个错误，第一个错误: %w", len(errs), errs[0])
	}

	return nil
}

func (cm *clusterManager) CheckClusterStatus(ctx context.Context, clusterID int) error {
	return cm.client.CheckClusterConnection(clusterID)
}

func (cm *clusterManager) validateResourceQuantities(cluster *model.K8sCluster) error {
	if cluster.CpuRequest == "" {
		cluster.CpuRequest = "500m"
	}
	if cluster.MemoryRequest == "" {
		cluster.MemoryRequest = "512Mi"
	}
	if cluster.CpuLimit == "" {
		cluster.CpuLimit = "1000m"
	}
	if cluster.MemoryLimit == "" {
		cluster.MemoryLimit = "1Gi"
	}

	if cluster.CpuRequest > cluster.CpuLimit {
		cluster.CpuRequest = cluster.CpuLimit
	}
	if cluster.MemoryRequest > cluster.MemoryLimit {
		cluster.MemoryRequest = cluster.MemoryLimit
	}

	return nil
}

func (cm *clusterManager) processClusterConfig(ctx context.Context, cluster *model.K8sCluster, _ int, initTimeout time.Duration, maxConcurrent int) error {
	ctx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		return err
	}

	if err := cm.client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		return err
	}

	kubeClient, err := cm.client.GetKubeClient(cluster.ID)
	if err != nil {
		return err
	}

	if len(cluster.RestrictedNameSpace) == 0 {
		cluster.RestrictedNameSpace = []string{"default"}
	}

	return cm.processNamespaces(ctx, kubeClient, cluster, maxConcurrent)
}

func (cm *clusterManager) processNamespaces(ctx context.Context, kubeClient *kubernetes.Clientset, cluster *model.K8sCluster, maxConcurrent int) error {
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxConcurrent)
	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cluster.ActionTimeoutSeconds)*time.Second)
	defer cancel()

	for _, ns := range cluster.RestrictedNameSpace {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			wg.Add(1)
			go func(namespace string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if err := cm.configureNamespace(ctx, kubeClient, namespace, cluster); err != nil {
					select {
					case errChan <- err:
					default:
					}
					cancel()
				}
			}(ns)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-done:
	}

	return nil
}

func (cm *clusterManager) configureNamespace(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	if namespace == "" {
		return fmt.Errorf("命名空间名称为空")
	}

	if err := utils.EnsureNamespace(ctx, kubeClient, namespace); err != nil {
		return fmt.Errorf("确保命名空间 %s 存在失败: %w", namespace, err)
	}

	if err := utils.ApplyLimitRange(ctx, kubeClient, namespace, cluster); err != nil {
		return fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", namespace, err)
	}

	if err := utils.ApplyResourceQuota(ctx, kubeClient, namespace, cluster); err != nil {
		return fmt.Errorf("应用 ResourceQuota 到命名空间 %s 失败: %w", namespace, err)
	}

	return nil
}