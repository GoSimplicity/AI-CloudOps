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
	discovery2 "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K8sClient interface {
	GetKubeClient(clusterID int) (*kubernetes.Clientset, error)
	GetKruiseClient(clusterID int) (*versioned.Clientset, error)
	GetMetricsClient(clusterID int) (*metricsClient.Clientset, error)
	GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error)
	GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error)
	RefreshClients(ctx context.Context) error
	RemoveCluster(clusterID int)
	CheckClusterConnection(clusterID int) error
}

type k8sClient struct {
	mu      sync.RWMutex
	clients map[int]*clusterClients
	dao     dao.ClusterDAO
	logger  *zap.Logger
}

type clusterClients struct {
	kube      *kubernetes.Clientset
	kruise    *versioned.Clientset
	metrics   *metricsClient.Clientset
	dynamic   *dynamic.DynamicClient
	discovery *discovery2.DiscoveryClient
	config    *rest.Config
}

func NewK8sClient(logger *zap.Logger, dao dao.ClusterDAO) K8sClient {
	return &k8sClient{
		clients: make(map[int]*clusterClients),
		dao:     dao,
		logger:  logger,
	}
}

func (k *k8sClient) GetKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	k.mu.RLock()
	clients, exists := k.clients[clusterID]
	k.mu.RUnlock()

	if exists && clients.kube != nil {
		return clients.kube, nil
	}

	return k.initClusterClients(clusterID)
}

func (k *k8sClient) GetKruiseClient(clusterID int) (*versioned.Clientset, error) {
	k.mu.RLock()
	clients, exists := k.clients[clusterID]
	k.mu.RUnlock()

	if !exists || clients.kruise == nil {
		return nil, fmt.Errorf("集群%d的kruise client不可用", clusterID)
	}

	return clients.kruise, nil
}

func (k *k8sClient) GetMetricsClient(clusterID int) (*metricsClient.Clientset, error) {
	k.mu.RLock()
	clients, exists := k.clients[clusterID]
	k.mu.RUnlock()

	if !exists || clients.metrics == nil {
		return nil, fmt.Errorf("集群%d的metrics client不可用", clusterID)
	}

	return clients.metrics, nil
}

func (k *k8sClient) GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error) {
	k.mu.RLock()
	clients, exists := k.clients[clusterID]
	k.mu.RUnlock()

	if !exists || clients.dynamic == nil {
		return nil, fmt.Errorf("集群%d的dynamic client不可用", clusterID)
	}

	return clients.dynamic, nil
}

func (k *k8sClient) GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error) {
	k.mu.RLock()
	clients, exists := k.clients[clusterID]
	k.mu.RUnlock()

	if !exists || clients.discovery == nil {
		return nil, fmt.Errorf("cluster %d discovery client not available", clusterID)
	}

	return clients.discovery, nil
}

func (k *k8sClient) RefreshClients(ctx context.Context) error {
	page := 1
	size := 10
	var allErrors []error

	for {
		req := &model.ListClustersReq{
			ListReq: model.ListReq{
				Page: page,
				Size: size,
			},
		}

		clusters, total, err := k.dao.GetClusterList(ctx, req)
		if err != nil {
			return fmt.Errorf("获取集群列表失败: %w", err)
		}

		// 如果没有集群了，退出循环
		if len(clusters) == 0 {
			break
		}

		var errors []error
		for _, cluster := range clusters {
			if cluster.KubeConfigContent == "" {
				continue
			}

			_, err := k.initClusterClients(cluster.ID)
			if err != nil {
				errors = append(errors, fmt.Errorf("集群%d: %w", cluster.ID, err))
			}
		}

		allErrors = append(allErrors, errors...)

		// 如果已经处理完所有集群，退出循环
		if int64(page*size) >= total {
			break
		}

		page++
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("刷新%d个集群失败", len(allErrors))
	}

	return nil
}

func (k *k8sClient) RemoveCluster(clusterID int) {
	k.mu.Lock()
	delete(k.clients, clusterID)
	k.mu.Unlock()

	k.logger.Info("removed cluster clients", zap.Int("clusterID", clusterID))
}

func (k *k8sClient) CheckClusterConnection(clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return fmt.Errorf("获取kube client失败: %w", err)
	}

	_, err = client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("连接集群失败: %w", err)
	}

	return nil
}

func (k *k8sClient) initClusterClients(clusterID int) (*kubernetes.Clientset, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cluster, err := k.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("获取集群失败: %w", err)
	}

	if cluster.KubeConfigContent == "" {
		return nil, fmt.Errorf("集群%d的kubeconfig为空", clusterID)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		return nil, fmt.Errorf("解析kubeconfig失败: %w", err)
	}

	config.Timeout = 10 * time.Second

	clients := &clusterClients{config: config}

	// 创建 kubernetes 客户端（必需）
	clients.kube, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("创建kubernetes client失败: %w", err)
	}

	// 创建其他客户端（可选）
	clients.kruise, _ = versioned.NewForConfig(config)
	clients.metrics, _ = metricsClient.NewForConfig(config)
	clients.dynamic, _ = dynamic.NewForConfig(config)
	clients.discovery, _ = discovery2.NewDiscoveryClientForConfig(config)

	k.mu.Lock()
	k.clients[clusterID] = clients
	k.mu.Unlock()

	k.logger.Info("initialized cluster clients", zap.Int("clusterID", clusterID))
	return clients.kube, nil
}
