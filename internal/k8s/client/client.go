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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
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

type K8sClient interface {
	// InitClient 初始化指定集群 ID 的 Kubernetes 客户端
	InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error
	// GetKubeClient 获取指定集群 ID 的 Kubernetes 客户端
	GetKubeClient(clusterID int) (*kubernetes.Clientset, error)
	// GetKruiseClient 获取指定集群 ID 的 Kruise 客户端
	GetKruiseClient(clusterID int) (*versioned.Clientset, error)
	// GetMetricsClient 获取指定集群 ID 的 Metrics 客户端
	GetMetricsClient(clusterID int) (*metricsClient.Clientset, error)
	// GetDynamicClient 获取指定集群 ID 的动态客户端
	GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error)
	// GetDiscoveryClient 获取指定集群 ID 的 Discovery 客户端
	GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error)
	// RefreshClients 刷新客户端
	RefreshClients(ctx context.Context) error
}

type k8sClient struct {
	sync.RWMutex
	KubeClients       map[int]*kubernetes.Clientset    // K8s原生客户端集合
	KruiseClients     map[int]*versioned.Clientset     // Kruise扩展客户端集合
	MetricsClients    map[int]*metricsClient.Clientset // Metrics客户端集合
	DynamicClients    map[int]*dynamic.DynamicClient   // 动态客户端集合
	RestConfigs       map[int]*rest.Config             // REST配置集合
	DiscoveryClients  map[int]*discovery2.DiscoveryClient
	ClusterNamespaces map[string][]string // 集群命名空间集合
	LastProbeErrors   map[int]string      // 集群探针错误信息
	l                 *zap.Logger         // 日志记录器
	dao               admin.ClusterDAO
}

func NewK8sClient(l *zap.Logger, dao admin.ClusterDAO) K8sClient {
	return &k8sClient{
		KubeClients:       make(map[int]*kubernetes.Clientset),
		KruiseClients:     make(map[int]*versioned.Clientset),
		MetricsClients:    make(map[int]*metricsClient.Clientset),
		DynamicClients:    make(map[int]*dynamic.DynamicClient),
		RestConfigs:       make(map[int]*rest.Config),
		DiscoveryClients:  make(map[int]*discovery2.DiscoveryClient),
		ClusterNamespaces: make(map[string][]string),
		LastProbeErrors:   make(map[int]string),
		l:                 l,
		dao:               dao,
	}
}

// InitClient 初始化指定集群 ID 的 Kubernetes 客户端
func (k *k8sClient) InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error {
	k.Lock()
	defer k.Unlock()

	k.l.Info("Initializing client for cluster", zap.Int("ClusterID", clusterID))

	// 检查客户端是否已经初始化
	if _, exists := k.KubeClients[clusterID]; exists {
		k.l.Debug("InitClient: Client already initialized for clusterID", zap.Int("ClusterID", clusterID))
		return nil
	}

	// 保存 REST 配置
	k.RestConfigs[clusterID] = kubeConfig

	// 创建 Kubernetes 原生客户端
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		k.l.Error("创建 Kubernetes 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Kubernetes 客户端失败: %w", err)
	}
	k.KubeClients[clusterID] = kubeClient

	// 创建 Kruise 客户端
	kruiseClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		k.l.Error("创建 Kruise 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Kruise 客户端失败: %w", err)
	}
	k.KruiseClients[clusterID] = kruiseClient

	// 创建 Metrics 客户端
	metricsClientSet, err := metricsClient.NewForConfig(kubeConfig)
	if err != nil {
		k.l.Error("创建 Metrics 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Metrics 客户端失败: %w", err)
	}
	k.MetricsClients[clusterID] = metricsClientSet

	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		k.l.Error("创建动态客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建动态客户端失败: %w", err)
	}
	k.DynamicClients[clusterID] = dynamicClient

	discoveryClient, err := discovery2.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		k.l.Error("创建 Discovery 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Discovery 客户端失败: %w", err)
	}
	k.DiscoveryClients[clusterID] = discoveryClient

	// 获取并保存命名空间，直接使用 kubeClient
	namespaces, err := k.getNamespacesDirectly(ctx, clusterID, kubeClient)
	if err != nil {
		k.l.Debug("获取命名空间失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		k.LastProbeErrors[clusterID] = err.Error()
	} else {
		host := kubeConfig.Host
		if host == "" {
			host = "unknown"
		}
		k.ClusterNamespaces[host] = namespaces
		k.l.Info("Namespaces retrieved", zap.Int("ClusterID", clusterID), zap.Int("NamespaceCount", len(namespaces)))
	}

	k.l.Info("初始化 Kubernetes 客户端成功", zap.Int("ClusterID", clusterID))

	return nil
}

// getNamespacesDirectly 直接使用 kubeClient 获取命名空间
func (k *k8sClient) getNamespacesDirectly(ctx context.Context, clusterID int, kubeClient *kubernetes.Clientset) ([]string, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		k.l.Debug("获取命名空间失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return nil, fmt.Errorf("获取命名空间失败: %w", err)
	}

	nsList := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		nsList[i] = ns.Name
	}
	k.l.Debug("获取到到命名空间为：", zap.Strings("Namespaces", nsList))
	return nsList, nil
}

// GetKubeClient 获取指定集群 ID 的 Kubernetes 客户端
func (k *k8sClient) GetKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	k.RLock()
	client, exists := k.KubeClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 KubeClient 未初始化", clusterID)
	}

	return client, nil
}

// GetKruiseClient 获取指定集群 ID 的 Kruise 客户端
func (k *k8sClient) GetKruiseClient(clusterID int) (*versioned.Clientset, error) {
	k.RLock()
	client, exists := k.KruiseClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 KruiseClient 未初始化", clusterID)
	}

	return client, nil
}

// GetMetricsClient 获取指定集群 ID 的 Metrics 客户端
func (k *k8sClient) GetMetricsClient(clusterID int) (*metricsClient.Clientset, error) {
	k.RLock()
	client, exists := k.MetricsClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 MetricsClient 未初始化", clusterID)
	}

	return client, nil
}

// GetDynamicClient 获取指定集群 ID 的动态客户端
func (k *k8sClient) GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error) {
	k.RLock()
	client, exists := k.DynamicClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 DynamicClient 未初始化", clusterID)
	}

	return client, nil
}

// GetDiscoveryClient 获取指定集群 ID 的 Discovery 客户端
func (k *k8sClient) GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error) {
	k.RLock()
	defer k.RUnlock()

	discoveryClient, exists := k.DiscoveryClients[clusterID]
	if !exists {
		return nil, fmt.Errorf("discovery client not found for cluster ID: %d", clusterID)
	}

	return discoveryClient, nil
}

// RefreshClients 刷新所有集群的客户端
func (k *k8sClient) RefreshClients(ctx context.Context) error {
	clusters, err := k.dao.ListAllClusters(ctx)
	if err != nil {
		k.l.Error("RefreshClients: 获取所有集群失败", zap.Error(err))
		return err
	}

	for _, cluster := range clusters {
		restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
		if err != nil {
			k.l.Error("RefreshClients: 解析 kubeconfig 失败", zap.Int("ClusterID", cluster.ID), zap.Error(err))
			continue
		}
		err = k.InitClient(ctx, cluster.ID, restConfig)
		if err != nil {
			k.l.Error("RefreshClients: 初始化 Kubernetes 客户端失败", zap.Int("ClusterID", cluster.ID), zap.Error(err))
			continue
		}
	}

	k.l.Info("RefreshClients: 所有集群的 Kubernetes 客户端刷新完成")

	return nil
}
