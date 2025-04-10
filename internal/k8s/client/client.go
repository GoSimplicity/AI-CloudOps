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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/openkruise/kruise-api/client/clientset/versioned"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	discovery2 "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K8sClient interface {
	// 初始化客户端
	InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error
	GetKubeClient(clusterID int) (*kubernetes.Clientset, error)
	GetKruiseClient(clusterID int) (*versioned.Clientset, error)
	GetMetricsClient(clusterID int) (*metricsClient.Clientset, error)
	GetDynamicClient(clusterID int) (*dynamic.DynamicClient, error)
	GetDiscoveryClient(clusterID int) (*discovery2.DiscoveryClient, error)
	RefreshClients(ctx context.Context) error
	// 创建资源
	CreateDeployment(ctx context.Context, namespace string, clusterID int, deployment *appsv1.Deployment) error
	CreateStatefulSet(ctx context.Context, namespace string, clusterID int, statefulset *appsv1.StatefulSet) error
	CreateDaemonSet(ctx context.Context, namespace string, clusterID int, daemonset *appsv1.DaemonSet) error
	CreateJob(ctx context.Context, namespace string, clusterID int, job *batchv1.Job) error
	CreateCronJob(ctx context.Context, namespace string, clusterID int, cronjob *batchv1.CronJob) error
	// 删除资源
	DeleteDeployment(ctx context.Context, namespace string, name string, clusterID int) error
	DeleteStatefulSet(ctx context.Context, namespace string, name string, clusterID int) error
	DeleteDaemonSet(ctx context.Context, namespace string, name string, clusterID int) error
	DeleteJob(ctx context.Context, namespace string, name string, clusterID int) error
	DeleteCronJob(ctx context.Context, namespace string, name string, clusterID int) error
	// 更新资源
	UpdateDeployment(ctx context.Context, namespace string, clusterID int, deployment *appsv1.Deployment) error
	UpdateStatefulSet(ctx context.Context, namespace string, clusterID int, statefulset *appsv1.StatefulSet) error
	UpdateDaemonSet(ctx context.Context, namespace string, clusterID int, daemonset *appsv1.DaemonSet) error
	UpdateJob(ctx context.Context, namespace string, clusterID int, job *batchv1.Job) error
	UpdateCronJob(ctx context.Context, namespace string, clusterID int, cronjob *batchv1.CronJob) error
	// 重启资源
	RestartDeployment(ctx context.Context, namespace string, name string, clusterID int) error
	RestartStatefulSet(ctx context.Context, namespace string, name string, clusterID int) error
	RestartDaemonSet(ctx context.Context, namespace string, name string, clusterID int) error
	RestartJob(ctx context.Context, namespace string, name string, clusterID int) error
	RestartCronJob(ctx context.Context, namespace string, name string, clusterID int) error
	// 获取资源
	GetDeployment(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.Deployment, error)
	GetStatefulSet(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.StatefulSet, error)
	GetDaemonSet(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.DaemonSet, error)
	GetJob(ctx context.Context, namespace string, name string, clusterID int) (*batchv1.Job, error)
	GetCronJob(ctx context.Context, namespace string, name string, clusterID int) (*batchv1.CronJob, error)
	// 获取资源列表
	GetDeploymentList(ctx context.Context, namespace string, clusterID int) ([]appsv1.Deployment, error)
	GetStatefulSetList(ctx context.Context, namespace string, clusterID int) ([]appsv1.StatefulSet, error)
	GetDaemonSetList(ctx context.Context, namespace string, clusterID int) ([]appsv1.DaemonSet, error)
	GetJobList(ctx context.Context, namespace string, clusterID int) ([]batchv1.Job, error)
	GetCronJobList(ctx context.Context, namespace string, clusterID int) ([]batchv1.CronJob, error)
}

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
	dao               admin.ClusterDAO
}



func NewK8sClient(logger *zap.Logger, dao admin.ClusterDAO) K8sClient {
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

// InitClient 初始化指定集群 ID 的 Kubernetes 客户端
func (k *k8sClient) InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error {
	k.Lock()
	defer k.Unlock()

	// 检查客户端是否已经初始化
	if _, exists := k.KubeClients[clusterID]; exists {
		return nil
	}

	// 保存 REST 配置
	k.RestConfigs[clusterID] = kubeConfig

	// 创建 Kubernetes 原生客户端
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建 Kubernetes 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Kubernetes 客户端失败: %w", err)
	}
	k.KubeClients[clusterID] = kubeClient

	// 创建 Kruise 客户端
	kruiseClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建 Kruise 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Kruise 客户端失败: %w", err)
	}
	k.KruiseClients[clusterID] = kruiseClient

	// 创建 Metrics 客户端
	metricsClientSet, err := metricsClient.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建 Metrics 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Metrics 客户端失败: %w", err)
	}
	k.MetricsClients[clusterID] = metricsClientSet

	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建动态客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建动态客户端失败: %w", err)
	}
	k.DynamicClients[clusterID] = dynamicClient

	discoveryClient, err := discovery2.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建 Discovery 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Discovery 客户端失败: %w", err)
	}
	k.DiscoveryClients[clusterID] = discoveryClient

	// 获取并保存命名空间，直接使用 kubeClient
	namespaces, err := k.getNamespacesDirectly(ctx, kubeClient)
	if err != nil {
		k.LastProbeErrors[clusterID] = err.Error()
	} else {
		host := kubeConfig.Host
		if host == "" {
			host = "unknown"
		}
		k.ClusterNamespaces[host] = namespaces
	}

	return nil
}

// getNamespacesDirectly 直接使用 kubeClient 获取命名空间
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
		return nil, fmt.Errorf("获取DiscoveryClient失败: %d", clusterID)
	}

	return discoveryClient, nil
}

// RefreshClients 刷新所有集群的客户端
func (k *k8sClient) RefreshClients(ctx context.Context) error {
	clusters, err := k.dao.ListAllClusters(ctx)
	if err != nil {
		k.logger.Error("获取所有集群失败", zap.Error(err))
		return err
	}

	for _, cluster := range clusters {
		restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
		if err != nil {
			k.logger.Error("解析 kubeconfig 失败", zap.Int("ClusterID", cluster.ID), zap.Error(err))
			continue
		}
		if err = k.InitClient(ctx, cluster.ID, restConfig); err != nil {
			k.logger.Error("初始化 Kubernetes 客户端失败", zap.Int("ClusterID", cluster.ID), zap.Error(err))
		}
	}

	return nil
}

// CreateCronJob 创建 CronJob 资源
func (k *k8sClient) CreateCronJob(ctx context.Context, namespace string, clusterID int, cronjob *batchv1.CronJob) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().CronJobs(namespace).Create(ctx, cronjob, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 CronJob 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 CronJob 失败: %w", err)
	}

	return nil
}

// CreateDaemonSet 创建 DaemonSet 资源
func (k *k8sClient) CreateDaemonSet(ctx context.Context, namespace string, clusterID int, daemonset *appsv1.DaemonSet) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().DaemonSets(namespace).Create(ctx, daemonset, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 DaemonSet 失败: %w", err)
	}

	return nil
}

// CreateDeployment 创建 Deployment 资源
func (k *k8sClient) CreateDeployment(ctx context.Context, namespace string, clusterID int, deployment *appsv1.Deployment) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 Deployment 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	return nil
}

// CreateJob 创建 Job 资源
func (k *k8sClient) CreateJob(ctx context.Context, namespace string, clusterID int, job *batchv1.Job) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 Job 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 Job 失败: %w", err)
	}

	return nil
}

// CreateStatefulSet 创建 StatefulSet 资源
func (k *k8sClient) CreateStatefulSet(ctx context.Context, namespace string, clusterID int, statefulset *appsv1.StatefulSet) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().StatefulSets(namespace).Create(ctx, statefulset, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 StatefulSet 失败: %w", err)
	}

	return nil
}

// DeleteCronJob 删除 CronJob 资源
func (k *k8sClient) DeleteCronJob(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		k.logger.Error("删除 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 CronJob 失败: %w", err)
	}

	return nil
}

// DeleteDaemonSet 删除 DaemonSet 资源
func (k *k8sClient) DeleteDaemonSet(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		k.logger.Error("删除 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 DaemonSet 失败: %w", err)
	}

	return nil
}

// DeleteDeployment 删除 Deployment 资源
func (k *k8sClient) DeleteDeployment(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		k.logger.Error("删除 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 Deployment 失败: %w", err)
	}

	return nil
}

// DeleteJob 删除 Job 资源
func (k *k8sClient) DeleteJob(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		k.logger.Error("删除 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 Job 失败: %w", err)
	}

	return nil
}

// DeleteStatefulSet 删除 StatefulSet 资源
func (k *k8sClient) DeleteStatefulSet(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		k.logger.Error("删除 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 StatefulSet 失败: %w", err)
	}

	return nil
}

// GetCronJob 获取 CronJob 资源
func (k *k8sClient) GetCronJob(ctx context.Context, namespace string, name string, clusterID int) (*batchv1.CronJob, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	cronJob, err := client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		k.logger.Error("获取 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取 CronJob 失败: %w", err)
	}

	return cronJob, nil
}

// GetCronJobList 获取 CronJob 资源列表
func (k *k8sClient) GetCronJobList(ctx context.Context, namespace string, clusterID int) ([]batchv1.CronJob, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	cronJobList, err := client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		k.logger.Error("获取 CronJob 列表失败", zap.Error(err), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 CronJob 列表失败: %w", err)
	}

	return cronJobList.Items, nil
}

// GetDaemonSet 获取 DaemonSet 资源
func (k *k8sClient) GetDaemonSet(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.DaemonSet, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		k.logger.Error("获取 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取 DaemonSet 失败: %w", err)
	}

	return daemonSet, nil
}

// GetDaemonSetList 获取 DaemonSet 资源列表
func (k *k8sClient) GetDaemonSetList(ctx context.Context, namespace string, clusterID int) ([]appsv1.DaemonSet, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	daemonSetList, err := client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		k.logger.Error("获取 DaemonSet 列表失败", zap.Error(err), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 DaemonSet 列表失败: %w", err)
	}

	return daemonSetList.Items, nil
}

// GetDeployment 获取 Deployment 资源
func (k *k8sClient) GetDeployment(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.Deployment, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		k.logger.Error("获取 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	return deployment, nil
}

// GetDeploymentList 获取 Deployment 资源列表
func (k *k8sClient) GetDeploymentList(ctx context.Context, namespace string, clusterID int) ([]appsv1.Deployment, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	deploymentList, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		k.logger.Error("获取 Deployment 列表失败", zap.Error(err), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 Deployment 列表失败: %w", err)
	}

	return deploymentList.Items, nil
}

// GetJob 获取 Job 资源
func (k *k8sClient) GetJob(ctx context.Context, namespace string, name string, clusterID int) (*batchv1.Job, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	job, err := client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		k.logger.Error("获取 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取 Job 失败: %w", err)
	}

	return job, nil
}

// GetJobList 获取 Job 资源列表
func (k *k8sClient) GetJobList(ctx context.Context, namespace string, clusterID int) ([]batchv1.Job, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	jobList, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		k.logger.Error("获取 Job 列表失败", zap.Error(err), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 Job 列表失败: %w", err)
	}

	return jobList.Items, nil
}

// GetStatefulSet 获取 StatefulSet 资源
func (k *k8sClient) GetStatefulSet(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.StatefulSet, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		k.logger.Error("获取 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取 StatefulSet 失败: %w", err)
	}

	return statefulSet, nil
}

// GetStatefulSetList 获取 StatefulSet 资源列表
func (k *k8sClient) GetStatefulSetList(ctx context.Context, namespace string, clusterID int) ([]appsv1.StatefulSet, error) {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	statefulSetList, err := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		k.logger.Error("获取 StatefulSet 列表失败", zap.Error(err), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 StatefulSet 列表失败: %w", err)
	}

	return statefulSetList.Items, nil
}

// RestartCronJob 重启 CronJob 资源
func (k *k8sClient) RestartCronJob(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 通过添加重启注解来重启 CronJob
	patchData := fmt.Sprintf(`{"spec":{"jobTemplate":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.BatchV1().CronJobs(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 CronJob 失败: %w", err)
	}

	return nil
}

// RestartDaemonSet 重启 DaemonSet 资源
func (k *k8sClient) RestartDaemonSet(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 通过添加重启注解来重启 DaemonSet
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 DaemonSet 失败: %w", err)
	}

	return nil
}

// RestartDeployment 重启 Deployment 资源
func (k *k8sClient) RestartDeployment(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 通过添加重启注解来重启 Deployment
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Deployment 失败: %w", err)
	}

	return nil
}

// RestartJob 重启 Job 资源
func (k *k8sClient) RestartJob(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 删除旧的 Job
	err = client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		k.logger.Error("删除 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Job 失败，无法删除旧 Job: %w", err)
	}

	// 获取原始 Job 配置并重新创建
	job, err := k.GetJob(ctx, namespace, name, clusterID)
	if err != nil {
		return fmt.Errorf("重启 Job 失败，无法获取 Job 配置: %w", err)
	}

	// 清除不需要的字段
	job.ResourceVersion = ""
	job.UID = ""
	job.CreationTimestamp = metav1.Time{}
	job.Status = batchv1.JobStatus{}

	_, err = client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("重新创建 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Job 失败，无法重新创建 Job: %w", err)
	}

	return nil
}

// RestartStatefulSet 重启 StatefulSet 资源
func (k *k8sClient) RestartStatefulSet(ctx context.Context, namespace string, name string, clusterID int) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 通过添加重启注解来重启 StatefulSet
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 StatefulSet 失败: %w", err)
	}

	return nil
}

// UpdateCronJob 更新 CronJob 资源
func (k *k8sClient) UpdateCronJob(ctx context.Context, namespace string, clusterID int, cronjob *batchv1.CronJob) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().CronJobs(namespace).Update(ctx, cronjob, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 CronJob 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("更新 CronJob 失败: %w", err)
	}

	return nil
}

// UpdateDaemonSet 更新 DaemonSet 资源
func (k *k8sClient) UpdateDaemonSet(ctx context.Context, namespace string, clusterID int, daemonset *appsv1.DaemonSet) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().DaemonSets(namespace).Update(ctx, daemonset, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("更新 DaemonSet 失败: %w", err)
	}

	return nil
}

// UpdateDeployment 更新 Deployment 资源
func (k *k8sClient) UpdateDeployment(ctx context.Context, namespace string, clusterID int, deployment *appsv1.Deployment) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 Deployment 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("更新 Deployment 失败: %w", err)
	}

	return nil
}

// UpdateJob 更新 Job 资源
func (k *k8sClient) UpdateJob(ctx context.Context, namespace string, clusterID int, job *batchv1.Job) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 Job 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("更新 Job 失败: %w", err)
	}

	return nil
}

// UpdateStatefulSet 更新 StatefulSet 资源
func (k *k8sClient) UpdateStatefulSet(ctx context.Context, namespace string, clusterID int, statefulset *appsv1.StatefulSet) error {
	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("更新 StatefulSet 失败: %w", err)
	}

	return nil
}