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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	CheckClusterConnection(clusterID int) error
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
	// 清理资源
	RemoveCluster(clusterID int)
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
	dao               dao.ClusterDAO
}

func NewK8sClient(logger *zap.Logger, dao dao.ClusterDAO) K8sClient {
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
	if kubeConfig == nil {
		return fmt.Errorf("kubeConfig 不能为空")
	}

	k.Lock()
	defer k.Unlock()

	// 检查客户端是否已经初始化
	if _, exists := k.KubeClients[clusterID]; exists {
		k.logger.Debug("客户端已初始化，跳过", zap.Int("ClusterID", clusterID))
		return nil
	}

	// 设置超时
	if kubeConfig.Timeout == 0 {
		kubeConfig.Timeout = 10 * time.Second
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

	// 创建 Kruise 客户端（非关键组件，失败不阻塞）
	kruiseClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Warn("创建 Kruise 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
	} else {
		k.KruiseClients[clusterID] = kruiseClient
	}

	// 创建 Metrics 客户端（非关键组件，失败不阻塞）
	metricsClientSet, err := metricsClient.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Warn("创建 Metrics 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
	} else {
		k.MetricsClients[clusterID] = metricsClientSet
	}

	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建动态客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建动态客户端失败: %w", err)
	}
	k.DynamicClients[clusterID] = dynamicClient

	// 创建 Discovery 客户端
	discoveryClient, err := discovery2.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		k.logger.Error("创建 Discovery 客户端失败", zap.Error(err), zap.Int("ClusterID", clusterID))
		return fmt.Errorf("创建 Discovery 客户端失败: %w", err)
	}
	k.DiscoveryClients[clusterID] = discoveryClient

	// 测试连接并获取命名空间
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

	if exists {
		return client, nil
	}

	// 尝试初始化客户端
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
	client, exists := k.DiscoveryClients[clusterID]
	k.RUnlock()

	if !exists {
		return nil, fmt.Errorf("集群 %d 的 DiscoveryClient 未初始化", clusterID)
	}

	return client, nil
}

// RefreshClients 刷新所有集群的客户端
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

	// 收集所有错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("刷新客户端时发生 %d 个错误，第一个错误: %w", len(errs), errs[0])
	}

	return nil
}

// RemoveCluster 清理指定集群的客户端
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

// validateInputs 验证输入参数
func (k *k8sClient) validateInputs(namespace, name string, clusterID int) error {
	if namespace == "" {
		return fmt.Errorf("namespace 不能为空")
	}
	if name == "" {
		return fmt.Errorf("资源名称不能为空")
	}
	if clusterID <= 0 {
		return fmt.Errorf("clusterID 必须大于 0")
	}
	return nil
}

// CreateDeployment 创建 Deployment 资源
func (k *k8sClient) CreateDeployment(ctx context.Context, namespace string, clusterID int, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", deployment.Name))
		return fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	return nil
}

// CreateStatefulSet 创建 StatefulSet 资源
func (k *k8sClient) CreateStatefulSet(ctx context.Context, namespace string, clusterID int, statefulset *appsv1.StatefulSet) error {
	if statefulset == nil {
		return fmt.Errorf("statefulset 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().StatefulSets(namespace).Create(ctx, statefulset, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", statefulset.Name))
		return fmt.Errorf("创建 StatefulSet 失败: %w", err)
	}

	return nil
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

// CreateDaemonSet 创建 DaemonSet 资源
func (k *k8sClient) CreateDaemonSet(ctx context.Context, namespace string, clusterID int, daemonset *appsv1.DaemonSet) error {
	if daemonset == nil {
		return fmt.Errorf("daemonset 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().DaemonSets(namespace).Create(ctx, daemonset, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", daemonset.Name))
		return fmt.Errorf("创建 DaemonSet 失败: %w", err)
	}

	return nil
}

// CreateJob 创建 Job 资源
func (k *k8sClient) CreateJob(ctx context.Context, namespace string, clusterID int, job *batchv1.Job) error {
	if job == nil {
		return fmt.Errorf("job 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", job.Name))
		return fmt.Errorf("创建 Job 失败: %w", err)
	}

	return nil
}

// CreateCronJob 创建 CronJob 资源
func (k *k8sClient) CreateCronJob(ctx context.Context, namespace string, clusterID int, cronjob *batchv1.CronJob) error {
	if cronjob == nil {
		return fmt.Errorf("cronjob 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().CronJobs(namespace).Create(ctx, cronjob, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("创建 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", cronjob.Name))
		return fmt.Errorf("创建 CronJob 失败: %w", err)
	}

	return nil
}

// DeleteDeployment 删除 Deployment 资源
func (k *k8sClient) DeleteDeployment(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		k.logger.Error("删除 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 Deployment 失败: %w", err)
	}

	return nil
}

// DeleteStatefulSet 删除 StatefulSet 资源
func (k *k8sClient) DeleteStatefulSet(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		k.logger.Error("删除 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 StatefulSet 失败: %w", err)
	}

	return nil
}

// DeleteDaemonSet 删除 DaemonSet 资源
func (k *k8sClient) DeleteDaemonSet(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		k.logger.Error("删除 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 DaemonSet 失败: %w", err)
	}

	return nil
}

// DeleteJob 删除 Job 资源
func (k *k8sClient) DeleteJob(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		k.logger.Error("删除 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 Job 失败: %w", err)
	}

	return nil
}

// DeleteCronJob 删除 CronJob 资源
func (k *k8sClient) DeleteCronJob(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = client.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		k.logger.Error("删除 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除 CronJob 失败: %w", err)
	}

	return nil
}

// GetDeployment 获取 Deployment 资源
func (k *k8sClient) GetDeployment(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.Deployment, error) {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return nil, err
	}

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

// GetStatefulSet 获取 StatefulSet 资源
func (k *k8sClient) GetStatefulSet(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.StatefulSet, error) {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return nil, err
	}

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

// GetDaemonSet 获取 DaemonSet 资源
func (k *k8sClient) GetDaemonSet(ctx context.Context, namespace string, name string, clusterID int) (*appsv1.DaemonSet, error) {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return nil, err
	}

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

// GetJob 获取 Job 资源
func (k *k8sClient) GetJob(ctx context.Context, namespace string, name string, clusterID int) (*batchv1.Job, error) {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return nil, err
	}

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

// GetCronJob 获取 CronJob 资源
func (k *k8sClient) GetCronJob(ctx context.Context, namespace string, name string, clusterID int) (*batchv1.CronJob, error) {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return nil, err
	}

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

// GetDeploymentList 获取 Deployment 资源列表
func (k *k8sClient) GetDeploymentList(ctx context.Context, namespace string, clusterID int) ([]appsv1.Deployment, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}

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

// GetStatefulSetList 获取 StatefulSet 资源列表
func (k *k8sClient) GetStatefulSetList(ctx context.Context, namespace string, clusterID int) ([]appsv1.StatefulSet, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}

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

// GetDaemonSetList 获取 DaemonSet 资源列表
func (k *k8sClient) GetDaemonSetList(ctx context.Context, namespace string, clusterID int) ([]appsv1.DaemonSet, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}

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

// GetJobList 获取 Job 资源列表
func (k *k8sClient) GetJobList(ctx context.Context, namespace string, clusterID int) ([]batchv1.Job, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}

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

// GetCronJobList 获取 CronJob 资源列表
func (k *k8sClient) GetCronJobList(ctx context.Context, namespace string, clusterID int) ([]batchv1.CronJob, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}

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

// RestartDeployment 重启 Deployment 资源
func (k *k8sClient) RestartDeployment(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Deployment 失败: %w", err)
	}

	return nil
}

// RestartStatefulSet 重启 StatefulSet 资源
func (k *k8sClient) RestartStatefulSet(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 StatefulSet 失败: %w", err)
	}

	return nil
}

// RestartDaemonSet 重启 DaemonSet 资源
func (k *k8sClient) RestartDaemonSet(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 DaemonSet 失败: %w", err)
	}

	return nil
}

// RestartJob 重启 Job 资源
func (k *k8sClient) RestartJob(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取原始 Job 配置
	job, err := client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		k.logger.Error("获取 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Job 失败，无法获取 Job 配置: %w", err)
	}

	// 创建新的 Job 对象
	newJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        job.Name,
			Namespace:   job.Namespace,
			Labels:      job.Labels,
			Annotations: job.Annotations,
		},
		Spec: job.Spec,
	}

	// 清除运行时字段
	newJob.ObjectMeta.ResourceVersion = ""
	newJob.ObjectMeta.UID = ""
	newJob.ObjectMeta.CreationTimestamp = metav1.Time{}

	// 删除旧的 Job
	propagationPolicy := metav1.DeletePropagationBackground
	err = client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil && !errors.IsNotFound(err) {
		k.logger.Error("删除 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Job 失败，无法删除旧 Job: %w", err)
	}

	// 等待 Job 被完全删除
	time.Sleep(2 * time.Second)

	// 创建新的 Job
	_, err = client.BatchV1().Jobs(namespace).Create(ctx, newJob, metav1.CreateOptions{})
	if err != nil {
		k.logger.Error("重新创建 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 Job 失败，无法重新创建 Job: %w", err)
	}

	return nil
}

// RestartCronJob 重启 CronJob 资源
func (k *k8sClient) RestartCronJob(ctx context.Context, namespace string, name string, clusterID int) error {
	if err := k.validateInputs(namespace, name, clusterID); err != nil {
		return err
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	patchData := fmt.Sprintf(`{"spec":{"jobTemplate":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.BatchV1().CronJobs(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		k.logger.Error("重启 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("重启 CronJob 失败: %w", err)
	}

	return nil
}

// UpdateDeployment 更新 Deployment 资源
func (k *k8sClient) UpdateDeployment(ctx context.Context, namespace string, clusterID int, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 Deployment 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", deployment.Name))
		return fmt.Errorf("更新 Deployment 失败: %w", err)
	}

	return nil
}

// UpdateStatefulSet 更新 StatefulSet 资源
func (k *k8sClient) UpdateStatefulSet(ctx context.Context, namespace string, clusterID int, statefulset *appsv1.StatefulSet) error {
	if statefulset == nil {
		return fmt.Errorf("statefulset 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 StatefulSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", statefulset.Name))
		return fmt.Errorf("更新 StatefulSet 失败: %w", err)
	}

	return nil
}

// UpdateDaemonSet 更新 DaemonSet 资源
func (k *k8sClient) UpdateDaemonSet(ctx context.Context, namespace string, clusterID int, daemonset *appsv1.DaemonSet) error {
	if daemonset == nil {
		return fmt.Errorf("daemonset 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().DaemonSets(namespace).Update(ctx, daemonset, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 DaemonSet 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", daemonset.Name))
		return fmt.Errorf("更新 DaemonSet 失败: %w", err)
	}

	return nil
}

// UpdateJob 更新 Job 资源
func (k *k8sClient) UpdateJob(ctx context.Context, namespace string, clusterID int, job *batchv1.Job) error {
	if job == nil {
		return fmt.Errorf("job 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 Job 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", job.Name))
		return fmt.Errorf("更新 Job 失败: %w", err)
	}

	return nil
}

// UpdateCronJob 更新 CronJob 资源
func (k *k8sClient) UpdateCronJob(ctx context.Context, namespace string, clusterID int, cronjob *batchv1.CronJob) error {
	if cronjob == nil {
		return fmt.Errorf("cronjob 不能为空")
	}

	client, err := k.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = client.BatchV1().CronJobs(namespace).Update(ctx, cronjob, metav1.UpdateOptions{})
	if err != nil {
		k.logger.Error("更新 CronJob 失败", zap.Error(err), zap.String("namespace", namespace), zap.String("name", cronjob.Name))
		return fmt.Errorf("更新 CronJob 失败: %w", err)
	}

	return nil
}
