package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"sync"
)

type ClusterService interface {
	// ListAllClusters 获取所有 Kubernetes 集群
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	// CreateCluster 创建一个新的 Kubernetes 集群
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	// UpdateCluster 更新指定 ID 的 Kubernetes 集群
	UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error
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

func (c *clusterService) DeleteCluster(ctx context.Context, id int) error {
	return c.dao.DeleteCluster(ctx, id)
}

func NewClusterService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) ClusterService {
	return &clusterService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

func (c *clusterService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	return c.dao.ListAllClusters(ctx)
}

func (c *clusterService) CreateCluster(ctx context.Context, cluster *model.K8sCluster) (err error) {
	// 创建集群记录
	if err := c.dao.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 确保后续操作如果出现错误时回滚集群记录
	defer func() {
		if err != nil {
			c.l.Info("CreateCluster: 回滚集群记录", zap.Int("clusterID", cluster.ID))
			if rollbackErr := c.dao.DeleteCluster(ctx, cluster.ID); rollbackErr != nil {
				c.l.Error("CreateCluster: 回滚集群记录失败", zap.Error(rollbackErr))
			}
		}
	}()

	// 解析 kubeconfig 并手动初始化 Kubernetes 客户端
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		c.l.Error("CreateCluster: 解析 kubeconfig 失败", zap.Error(err))
		return fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}

	// 初始化 Kubernetes 客户端
	if err = c.client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		c.l.Error("CreateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("初始化 Kubernetes 客户端失败: %w", err)
	}

	// 获取 Kubernetes 客户端
	kubeClient, err := c.client.GetKubeClient(cluster.ID)
	if err != nil {
		c.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	const maxConcurrent = 5
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx1, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, namespace := range cluster.RestrictedNameSpace {
		wg.Add(1)
		ns := namespace

		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 确保命名空间存在
			if err := pkg.EnsureNamespace(ctx1, kubeClient, ns); err != nil {
				errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 LimitRange
			if err := pkg.ApplyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 ResourceQuota
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

func (c *clusterService) UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error {
	// 初始化 Kubernetes 客户端
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		c.l.Error("UpdateCluster: 解析 kubeconfig 失败", zap.Error(err))
		return fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}

	if err = c.client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		c.l.Error("UpdateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("初始化 Kubernetes 客户端失败: %w", err)
	}

	kubeClient, err := c.client.GetKubeClient(cluster.ID)
	if err != nil {
		c.l.Error("UpdateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	const maxConcurrent = 5
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx1, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, namespace := range cluster.RestrictedNameSpace {
		wg.Add(1)
		ns := namespace

		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 确保命名空间存在
			if err := pkg.EnsureNamespace(ctx1, kubeClient, ns); err != nil {
				errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 LimitRange
			if err := pkg.ApplyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 ResourceQuota
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
			c.l.Error("UpdateCluster: 处理命名空间时发生错误", zap.Error(e))
			return e
		}
	}

	// 更新集群记录
	if err := c.dao.UpdateCluster(ctx, id, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群失败", zap.Error(err))
		return fmt.Errorf("更新集群失败: %w", err)
	}

	c.l.Info("UpdateCluster: 成功更新 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

func (c *clusterService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	return c.dao.BatchDeleteClusters(ctx, ids)
}

func (c *clusterService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	return c.dao.GetClusterByID(ctx, id)
}
