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

package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/openkruise/kruise-api/client/clientset/versioned"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discovery2 "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// K8sClientFactory 是 Kubernetes 客户端工厂接口，负责客户端的创建、管理和连接维护
// 遵循单一职责原则：仅负责客户端的生命周期管理，不包含业务逻辑
type K8sClientFactory interface {
	// 客户端初始化与管理
	InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error
	RefreshClients(ctx context.Context) error
	RemoveCluster(clusterID int)

	// 客户端获取方法 - 各种类型的 Kubernetes 客户端
	GetKubeClient(clusterID int) (*kubernetes.Clientset, error)
	GetKruiseClient(clusterID int) (*versioned.Clientset, error)
	GetMetricsClient(clusterID int) (*metricsClient.Clientset, error)
	GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error)
	GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error)

	// 连接状态检查与管理
	CheckClusterConnection(clusterID int) error
	UpdateClusterMetaFromLive(ctx context.Context, clusterID int) error
}

// K8sClient 保持向后兼容的接口别名
// 建议新代码使用 K8sClientFactory
type K8sClient = K8sClientFactory

type k8sClient struct {
	sync.RWMutex
	KubeClients       map[int]*kubernetes.Clientset
	KruiseClients     map[int]*versioned.Clientset
	MetricsClients    map[int]*metricsClient.Clientset
	DynamicClients    map[int]*dynamic.DynamicClient
	RestConfigs       map[int]*rest.Config
	DiscoveryClients  map[int]*discovery2.DiscoveryClient
	ClusterNamespaces map[string][]string
	LastProbeErrors   map[int]string
	logger            *zap.Logger
	dao               dao.ClusterDAO
}

// NewK8sClientFactory 创建新的 Kubernetes 客户端工厂实例
func NewK8sClientFactory(logger *zap.Logger, dao dao.ClusterDAO) K8sClientFactory {
	return &k8sClient{
		KubeClients:       make(map[int]*kubernetes.Clientset),
		KruiseClients:     make(map[int]*versioned.Clientset),
		MetricsClients:    make(map[int]*metricsClient.Clientset),
		DynamicClients:    make(map[int]*dynamic.DynamicClient),
		RestConfigs:       make(map[int]*rest.Config),
		DiscoveryClients:  make(map[int]*discovery2.DiscoveryClient),
		ClusterNamespaces: make(map[string][]string),
		LastProbeErrors:   make(map[int]string),
		logger:            logger,
		dao:               dao,
	}
}

// NewK8sClient 保持向后兼容的构造函数
// 建议新代码使用 NewK8sClientFactory
func NewK8sClient(logger *zap.Logger, dao dao.ClusterDAO) K8sClient {
	return NewK8sClientFactory(logger, dao)
}

// InitClient 初始化Kubernetes客户端
func (k *k8sClient) InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error {
	if kubeConfig == nil {
		return fmt.Errorf("kubeConfig 不能为空")
	}

	k.Lock()
	defer k.Unlock()

	if _, exists := k.KubeClients[clusterID]; exists {
		k.logger.Debug("客户端已初始化，跳过", zap.Int("ClusterID", clusterID))
		return nil
	}

	if kubeConfig.Timeout == 0 {
		kubeConfig.Timeout = 10 * time.Second
	}

	k.RestConfigs[clusterID] = kubeConfig

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建Kubernetes客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}
	k.KubeClients[clusterID] = kubeClient

	kruiseClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Warn("创建Kruise客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
	} else {
		k.KruiseClients[clusterID] = kruiseClient
	}

	metricsClientSet, err := metricsClient.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Warn("创建Metrics客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
	} else {
		k.MetricsClients[clusterID] = metricsClientSet
	}

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建动态客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建动态客户端失败: %w", err)
	}
	k.DynamicClients[clusterID] = dynamicClient

	discoveryClient, err := discovery2.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建Discovery客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建Discovery客户端失败: %w", err)
	}
	k.DiscoveryClients[clusterID] = discoveryClient

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	namespaces, err := k.getNamespacesDirectly(ctx, kubeClient)
	if err != nil {
		k.LastProbeErrors[clusterID] = err.Error()
		k.logger.Warn("获取命名空间失败", zap.Error(err), zap.Int("ClusterID", clusterID))
	} else {
		host := kubeConfig.Host
		if host == "" {
			host = fmt.Sprintf("cluster-%d", clusterID)
		}
		k.ClusterNamespaces[host] = namespaces
		delete(k.LastProbeErrors, clusterID)
	}

	k.logger.Info("客户端初始化成功", zap.Int("ClusterID", clusterID))
	return nil
}

// getNamespacesDirectly 获取命名空间列表
func (k *k8sClient) getNamespacesDirectly(ctx context.Context, kubeClient *kubernetes.Clientset) ([]string, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间失败: %w", err)
	}

	nsList := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		nsList[i] = ns.Name
	}
	return nsList, nil
}

// GetKubeClient 获取Kubernetes客户端
func (k *k8sClient) GetKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	k.RLock()
	client, exists := k.KubeClients[clusterID]
	k.RUnlock()

	if exists {
		return client, nil
	}

	return k.initClientFromDB(clusterID)
}

// initClientFromDB 从数据库初始化客户端
func (k *k8sClient) initClientFromDB(clusterID int) (*kubernetes.Clientset, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cluster, err := k.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("获取集群失败: %w", err)
	}

	if cluster.KubeConfigContent == "" {
		return nil, fmt.Errorf("集群 %d 的 KubeConfig 内容为空", clusterID)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		return nil, fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}

	if err := k.InitClient(ctx, clusterID, restConfig); err != nil {
		return nil, fmt.Errorf("初始化 Kubernetes 客户端失败: %w", err)
	}

	k.RLock()
	client, exists := k.KubeClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 KubeClient 初始化失败", clusterID)
	}

	return client, nil
}

// GetKruiseClient 获取Kruise客户端
func (k *k8sClient) GetKruiseClient(clusterID int) (*versioned.Clientset, error) {
	k.RLock()
	client, exists := k.KruiseClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 KruiseClient 未初始化", clusterID)
	}

	return client, nil
}

// GetMetricsClient 获取Metrics客户端
func (k *k8sClient) GetMetricsClient(clusterID int) (*metricsClient.Clientset, error) {
	k.RLock()
	client, exists := k.MetricsClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 MetricsClient 未初始化", clusterID)
	}

	return client, nil
}

// GetDynamicClient 获取动态客户端
func (k *k8sClient) GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error) {
	k.RLock()
	client, exists := k.DynamicClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 DynamicClient 未初始化", clusterID)
	}

	return client, nil
}

// GetDiscoveryClient 获取Discovery客户端
func (k *k8sClient) GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error) {
	k.RLock()
	client, exists := k.DiscoveryClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 DiscoveryClient 未初始化", clusterID)
	}

	return client, nil
}

// RefreshClients 刷新所有客户端
func (k *k8sClient) RefreshClients(ctx context.Context) error {
	clusters, err := k.dao.ListAllClusters(ctx)
	if err != nil {
		k.logger.Error("获取所有集群失败", zap.Error(err))
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(clusters))

	for _, cluster := range clusters {
		if cluster.KubeConfigContent == "" {
			k.logger.Warn("集群的 KubeConfig 内容为空，跳过初始化", zap.Int("ClusterID", cluster.ID))
			continue
		}

		wg.Add(1)
		go func(c *model.K8sCluster) {
			defer wg.Done()

			restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(c.KubeConfigContent))
			if err != nil {
				k.logger.Error("解析 kubeconfig 失败", zap.Int("ClusterID", c.ID), zap.Error(err))
				errChan <- fmt.Errorf("解析集群 %d 的 kubeconfig 失败: %w", c.ID, err)
				return
			}

			if err := k.InitClient(ctx, c.ID, restConfig); err != nil {
				k.logger.Error("初始化 Kubernetes 客户端失败", zap.Int("ClusterID", c.ID), zap.Error(err))
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
		return fmt.Errorf("刷新客户端时发生 %d 个错误，第一个错误: %w", len(errs), errs[0])
	}

	return nil
}

// RemoveCluster 清理集群客户端
func (k *k8sClient) RemoveCluster(clusterID int) {
	k.Lock()
	defer k.Unlock()

	delete(k.KubeClients, clusterID)
	delete(k.KruiseClients, clusterID)
	delete(k.MetricsClients, clusterID)
	delete(k.DynamicClients, clusterID)
	delete(k.RestConfigs, clusterID)
	delete(k.DiscoveryClients, clusterID)
	delete(k.LastProbeErrors, clusterID)

	k.logger.Info("已清理集群客户端", zap.Int("ClusterID", clusterID))
}

// CheckClusterConnection 检查集群连接
func (k *k8sClient) CheckClusterConnection(clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		k.logger.Error("获取集群客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		k.LastProbeErrors[clusterID] = err.Error()
		return fmt.Errorf("获取集群客户端失败: %w", err)
	}

	// 检查集群版本
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		k.logger.Error("检查集群连接失败", zap.Int("clusterID", clusterID), zap.Error(err))
		k.LastProbeErrors[clusterID] = err.Error()
		return fmt.Errorf("检查集群连接失败: %w", err)
	}

	k.logger.Debug("集群连接成功", zap.Int("clusterID", clusterID), zap.String("version", version.String()))
	delete(k.LastProbeErrors, clusterID)
	return nil
}

// UpdateClusterMetaFromLive 更新集群元信息
func (k *k8sClient) UpdateClusterMetaFromLive(ctx context.Context, clusterID int) error {
	kubeClient, err := k.GetKubeClient(clusterID)
	if err != nil {
		return fmt.Errorf("获取集群客户端失败: %w", err)
	}

	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("获取集群版本失败: %w", err)
	}

	k.RLock()
	restCfg := k.RestConfigs[clusterID]
	k.RUnlock()
	host := ""
	if restCfg != nil {
		host = restCfg.Host
	}

	if err := k.dao.UpdateCluster(ctx, &model.K8sCluster{Model: model.Model{ID: clusterID}, Version: v.String(), ApiServerAddr: host}); err != nil {
		k.logger.Warn("回写集群版本信息失败", zap.Int("ClusterID", clusterID), zap.Error(err))
	}
	return nil
}
