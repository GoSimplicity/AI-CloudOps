package admin

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

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"

	"go.uber.org/zap"
	"sync"
	"time"
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
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

// NewClusterService 创建并返回一个 ClusterService 实例
func NewClusterService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) ClusterService {
	return &clusterService{
		dao:    dao,
		client: client,
		l:      l,
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
	// 创建集群记录
	if err := c.dao.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 后续操作如果出现错误时回滚集群记录
	defer func() {
		if err != nil {
			c.l.Info("CreateCluster: 回滚集群记录", zap.Int("clusterID", cluster.ID))
			if rollbackErr := c.dao.DeleteCluster(ctx, cluster.ID); rollbackErr != nil {
				c.l.Error("CreateCluster: 回滚集群记录失败", zap.Error(rollbackErr))
			}
		}
	}()

	// 初始化 Kubernetes 客户端
	kubeClient, err := pkg.InitAadGetKubeClient(ctx, cluster, c.l, c.client)
	if err != nil {
		c.l.Error("CreateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 限制并发数，避免过多并发请求
	const maxConcurrent = 5
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx1, cancel := context.WithTimeout(ctx, time.Duration(cluster.ActionTimeoutSeconds)*time.Second)
	defer cancel()

	// 如果 RestrictedNameSpace 为空，默认为 "default" 命名空间
	if cluster.RestrictedNameSpace == nil || len(cluster.RestrictedNameSpace) == 0 {
		cluster.RestrictedNameSpace = []string{"default"}
	}

	// 为每个命名空间启动并发处理
	for _, namespace := range cluster.RestrictedNameSpace {
		wg.Add(1)
		ns := namespace

		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() {
				<-semaphore // 确保在退出时释放信号量
			}()

			// 确保命名空间存在
			if err := pkg.EnsureNamespace(ctx1, kubeClient, ns); err != nil {
				errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 LimitRange 配置
			if err := pkg.ApplyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 ResourceQuota 配置
			if err := pkg.ApplyResourceQuota(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 ResourceQuota 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for e := range errChan {
		if e != nil {
			c.l.Error("CreateCluster: 处理命名空间时发生错误", zap.Error(e))
			return e
		}
	}

	c.l.Info("CreateCluster: 成功创建 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// UpdateCluster 更新指定 ID 的 Kubernetes 集群
func (c *clusterService) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	// 如果 RestrictedNameSpace 为空，则跳过命名空间操作
	if cluster.RestrictedNameSpace != nil && len(cluster.RestrictedNameSpace) > 0 {
		kubeClient, err := pkg.InitAadGetKubeClient(ctx, cluster, c.l, c.client)
		if err != nil {
			c.l.Error("UpdateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
			return err
		}

		// 限制并发数，避免过多并发
		const maxConcurrent = 5
		semaphore := make(chan struct{}, maxConcurrent)

		var wg sync.WaitGroup
		errChan := make(chan error, len(cluster.RestrictedNameSpace))

		// 设置超时控制
		ctx1, cancel := context.WithTimeout(ctx, time.Duration(cluster.ActionTimeoutSeconds)*time.Second)
		defer cancel()

		// 并发处理命名空间
		for _, ns := range cluster.RestrictedNameSpace {
			if ns == "" {
				continue
			}

			wg.Add(1)

			go func(ns string) {
				defer wg.Done()

				semaphore <- struct{}{} // 信号量控制并发数
				defer func() {
					<-semaphore
				}()

				// 确保命名空间存在
				if err := pkg.EnsureNamespace(ctx1, kubeClient, ns); err != nil {
					errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
					return
				}

				// 应用 LimitRange
				if err := pkg.ApplyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
					errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
					return
				}

				// 应用 ResourceQuota
				if err := pkg.ApplyResourceQuota(ctx1, kubeClient, ns, cluster); err != nil {
					errChan <- fmt.Errorf("应用 ResourceQuota 到命名空间 %s 失败: %w", ns, err)
					return
				}
			}(ns)
		}

		wg.Wait()
		close(errChan)

		// 检查并处理并发任务中的错误
		for e := range errChan {
			if e != nil {
				c.l.Error("UpdateCluster: 处理命名空间时发生错误", zap.Error(e))
				return e
			}
		}
	}

	// 更新集群记录
	if err := c.dao.UpdateCluster(ctx, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群失败", zap.Error(err))
		return fmt.Errorf("更新集群失败: %w", err)
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
