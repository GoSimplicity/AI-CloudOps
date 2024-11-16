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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/general"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync"
)

type NamespaceService interface {
	// GetClusterNamespacesList 获取命名空间列表
	GetClusterNamespacesList(ctx context.Context) ([]model.ClusterNamespaces, error)
	// GetClusterNamespacesById 获取指定集群的所有命名空间
	GetClusterNamespacesById(ctx context.Context, id int) ([]model.Namespace, error)
	// CreateNamespace 创建新的命名空间
	CreateNamespace(ctx context.Context, req model.CreateNamespaceRequest) error
	// DeleteNamespace 删除指定的命名空间
	DeleteNamespace(ctx context.Context, name string, id int) error
	// GetNamespaceDetails 获取指定命名空间的详情
	GetNamespaceDetails(ctx context.Context, name string, id int) (model.Namespace, error)
	// UpdateNamespace 更新指定命名空间
	UpdateNamespace(ctx context.Context, req model.UpdateNamespaceRequest) error
	// GetNamespaceResources 获取指定命名空间中的资源
	GetNamespaceResources(ctx context.Context, name string, id int) ([]model.Resource, error)
	// GetNamespaceEvents 获取指定命名空间中的事件
	GetNamespaceEvents(ctx context.Context, name string, id int) ([]model.Event, error)
}

type namespaceService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewNamespaceService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) NamespaceService {
	return &namespaceService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetClusterNamespacesById 获取指定集群的命名空间列表，返回详细的 Namespace 结构体
func (n *namespaceService) GetClusterNamespacesById(ctx context.Context, id int) ([]model.Namespace, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取命名空间列表
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		n.logger.Error("获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	// 转换为 []model.Namespace
	nsList := make([]model.Namespace, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		// 转换 Labels 和 Annotations 为 []string
		labels, err := general.MapToStringSlice(ns.Labels)
		if err != nil {
			n.logger.Error("转换 Labels 失败", zap.Error(err))
			return nil, err
		}

		annotations, err := general.MapToStringSlice(ns.Annotations)
		if err != nil {
			n.logger.Error("转换 Annotations 失败", zap.Error(err))
			return nil, err
		}

		nsList[i] = model.Namespace{
			Name:         ns.Name,
			UID:          string(ns.UID),
			Status:       string(ns.Status.Phase),
			CreationTime: ns.CreationTimestamp.Time,
			Labels:       labels,
			Annotations:  annotations,
		}
	}

	return nsList, nil
}

// GetClusterNamespacesList 获取所有集群的命名空间列表，返回包含 cluster_name 和 namespaces 的列表
func (n *namespaceService) GetClusterNamespacesList(ctx context.Context) ([]model.ClusterNamespaces, error) {
	// 获取集群列表
	clusters, err := n.dao.ListAllClusters(ctx)
	if err != nil {
		n.logger.Error("获取集群列表失败", zap.Error(err))
		return nil, err
	}

	// 初始化结果切片
	var clusterNamespacesList []model.ClusterNamespaces
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // 限制并发数为 10

	// 遍历集群列表并发获取命名空间
	for _, cluster := range clusters {
		cluster := cluster // 避免闭包变量捕获问题
		g.Go(func() error {
			// 获取指定集群的命名空间
			namespaces, err := n.GetClusterNamespacesById(ctx, cluster.ID)
			if err != nil {
				n.logger.Error("获取命名空间列表失败",
					zap.Error(err),
					zap.String("clusterName", cluster.Name))
				return err
			}

			// 创建 ClusterNamespaces 实例
			cn := model.ClusterNamespaces{
				ClusterName: cluster.Name,
				ClusterId:   cluster.ID,
				Namespaces:  namespaces,
			}

			// 加锁写入结果切片
			mu.Lock()
			clusterNamespacesList = append(clusterNamespacesList, cn)
			mu.Unlock()

			return nil
		})
	}

	// 等待所有 goroutines 执行完成
	if err := g.Wait(); err != nil {
		n.logger.Error("并发获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	return clusterNamespacesList, nil
}

func (n *namespaceService) CreateNamespace(ctx context.Context, req model.CreateNamespaceRequest) error {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(req.ClusterId, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 将 []string 转换为 map[string]string
	labelsMap, err := general.StringSliceToMap(req.Labels)
	if err != nil {
		n.logger.Error("转换 Labels 失败", zap.Error(err))
		return err
	}

	annotationsMap, err := general.StringSliceToMap(req.Annotations)
	if err != nil {
		n.logger.Error("转换 Annotations 失败", zap.Error(err))
		return err
	}

	// 创建命名空间对象
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      labelsMap,
			Annotations: annotationsMap,
		},
	}

	// 提交创建请求
	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		n.logger.Error("创建命名空间失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteNamespace 删除指定的命名空间
func (n *namespaceService) DeleteNamespace(ctx context.Context, name string, id int) error {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 删除命名空间
	err = kubeClient.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		n.logger.Error("删除命名空间失败", zap.Error(err))
		return err
	}

	return nil
}

// GetNamespaceDetails 获取指定命名空间的详细信息
func (n *namespaceService) GetNamespaceDetails(ctx context.Context, name string, id int) (model.Namespace, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return model.Namespace{}, err
	}

	// 获取命名空间详情
	namespace, err := kubeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取命名空间详情失败", zap.Error(err))
		return model.Namespace{}, err
	}

	labels, err := general.MapToStringSlice(namespace.Labels)
	if err != nil {
		n.logger.Error("转换 Labels 失败", zap.Error(err))
		return model.Namespace{}, err
	}

	annotations, err := general.MapToStringSlice(namespace.Annotations)
	if err != nil {
		n.logger.Error("转换 Annotations 失败", zap.Error(err))
		return model.Namespace{}, err
	}

	return model.Namespace{
		Name:         namespace.Name,
		UID:          string(namespace.UID),
		Status:       string(namespace.Status.Phase),
		CreationTime: namespace.CreationTimestamp.Time,
		Labels:       labels,
		Annotations:  annotations,
	}, nil
}

// UpdateNamespace 更新指定命名空间
func (n *namespaceService) UpdateNamespace(ctx context.Context, req model.UpdateNamespaceRequest) error {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(req.ClusterId, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 获取现有命名空间
	namespace, err := kubeClient.CoreV1().Namespaces().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取命名空间失败", zap.Error(err))
		return err
	}

	labelsMap, err := general.StringSliceToMap(req.Labels)
	if err != nil {
		n.logger.Error("转换 Labels 失败", zap.Error(err))
		return err
	}

	annotationsMap, err := general.StringSliceToMap(req.Annotations)
	if err != nil {
		n.logger.Error("转换 Annotations 失败", zap.Error(err))
		return err
	}

	// 更新命名空间标签或注释
	namespace.Labels = labelsMap
	namespace.Annotations = annotationsMap

	// 提交更新请求
	_, err = kubeClient.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		n.logger.Error("更新命名空间失败", zap.Error(err))
		return err
	}

	return nil
}

// GetNamespaceResources 获取指定命名空间中的所有资源
func (n *namespaceService) GetNamespaceResources(ctx context.Context, namespace string, id int) ([]model.Resource, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 定义资源类型和对应的获取函数
	resourceTypes := map[string]func(context.Context, *kubernetes.Clientset, string) ([]model.Resource, error){
		"pods":         pkg.GetPodResources,
		"services":     pkg.GetServiceResources,
		"deployments":  pkg.GetDeploymentResources,
		"replicasets":  pkg.GetReplicaSetResources,
		"statefulsets": pkg.GetStatefulSetResources,
		"daemonsets":   pkg.GetDaemonSetResources,
	}

	var resources []model.Resource
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // 限制并发数为 10

	// 并发获取各类资源
	for resourceType, getResources := range resourceTypes {
		resourceType := resourceType // 避免闭包变量捕获问题
		g.Go(func() error {
			resourceList, err := getResources(ctx, kubeClient, namespace)
			if err != nil {
				n.logger.Error("获取资源失败", zap.String("resourceType", resourceType), zap.Error(err))
				return err
			}

			// 确保资源列表非空后再合并
			if len(resourceList) > 0 {
				mu.Lock()
				resources = append(resources, resourceList...)
				mu.Unlock()
			}
			return nil
		})
	}

	// 等待并发任务完成
	if err := g.Wait(); err != nil {
		n.logger.Error("并发获取资源失败", zap.Error(err))
		return nil, err
	}

	return resources, nil
}

// GetNamespaceEvents 获取指定命名空间中的事件
func (n *namespaceService) GetNamespaceEvents(ctx context.Context, namespace string, id int) ([]model.Event, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取事件列表
	events, err := kubeClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		n.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, err
	}

	// 提取事件信息
	eventList := make([]model.Event, len(events.Items))
	for i, event := range events.Items {
		eventList[i] = model.Event{
			Reason:         event.Reason,
			Message:        event.Message,
			Type:           event.Type,
			FirstTimestamp: event.FirstTimestamp.Time,
			LastTimestamp:  event.LastTimestamp.Time,
			Count:          event.Count,
			Source:         event.Source,
		}
	}

	return eventList, nil
}
