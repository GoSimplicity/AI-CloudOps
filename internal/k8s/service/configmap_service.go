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

package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ConfigMapService interface {
	GetConfigMapsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.ConfigMap, error)
	UpdateConfigMap(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	GetConfigMapYaml(ctx context.Context, id int, namespace, configMapName string) (*corev1.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, id int, namespace, configMapName string) error
	BatchDeleteConfigMap(ctx context.Context, id int, namespace string, configMapNames []string) error

	// 版本管理
	CreateConfigMapVersion(ctx context.Context, req *model.K8sConfigMapVersionRequest) error
	GetConfigMapVersions(ctx context.Context, id int, namespace, configMapName string) ([]*model.K8sConfigMapVersion, error)
	GetConfigMapVersion(ctx context.Context, id int, namespace, configMapName, version string) (*model.K8sConfigMapVersion, error)
	DeleteConfigMapVersion(ctx context.Context, id int, namespace, configMapName, version string) error

	// 热更新
	HotReloadConfigMap(ctx context.Context, req *model.K8sConfigMapHotReloadRequest) (map[string]interface{}, error)

	// 回滚
	RollbackConfigMap(ctx context.Context, req *model.K8sConfigMapRollbackRequest) error
}

type configMapService struct {
	dao    dao.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
	// 版本存储（实际生产中应使用数据库或其他持久化存储）
	versionStore map[string]map[string][]*model.K8sConfigMapVersion
	versionMutex sync.RWMutex
}

func NewConfigMapService(dao dao.ClusterDAO, client client.K8sClient, logger *zap.Logger) ConfigMapService {
	return &configMapService{
		dao:          dao,
		client:       client,
		logger:       logger,
		versionStore: make(map[string]map[string][]*model.K8sConfigMapVersion),
	}
}

// GetConfigMapsByNamespace 获取命名空间的所有 ConfigMap
func (c *configMapService) GetConfigMapsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.ConfigMap, error) {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get kubeClient: %w", err)
	}

	configMapList, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("获取 ConfigMap 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to list ConfigMaps: %w", err)
	}

	configMaps := make([]*corev1.ConfigMap, len(configMapList.Items))
	for i := range configMapList.Items {
		configMaps[i] = &configMapList.Items[i]
	}

	c.logger.Info("成功获取 ConfigMap 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(configMaps)))
	return configMaps, nil
}

// UpdateConfigMap 更新 ConfigMap
func (c *configMapService) UpdateConfigMap(ctx context.Context, configMapRequest *model.K8sConfigMapRequest) error {
	kubeClient, err := pkg.GetKubeClient(configMapRequest.ClusterId, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", configMapRequest.ClusterId))
		return fmt.Errorf("failed to get kubeClient: %w", err)
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(configMapRequest.ConfigMap.Namespace).Get(ctx, configMapRequest.ConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		c.logger.Error("获取 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", configMapRequest.ConfigMap.Name), zap.String("namespace", configMapRequest.ConfigMap.Namespace), zap.Int("cluster_id", configMapRequest.ClusterId))
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	for key, value := range configMapRequest.ConfigMap.Data {
		configMap.Data[key] = value
	}

	_, err = kubeClient.CoreV1().ConfigMaps(configMapRequest.ConfigMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		c.logger.Error("更新 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", configMapRequest.ConfigMap.Name), zap.String("namespace", configMapRequest.ConfigMap.Namespace), zap.Int("cluster_id", configMapRequest.ClusterId))
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	c.logger.Info("成功更新 ConfigMap", zap.String("configmap_name", configMapRequest.ConfigMap.Name), zap.String("namespace", configMapRequest.ConfigMap.Namespace), zap.Int("cluster_id", configMapRequest.ClusterId))
	return nil
}

// GetConfigMapYaml 获取 ConfigMap 详情
func (c *configMapService) GetConfigMapYaml(ctx context.Context, id int, namespace, configMapName string) (*corev1.ConfigMap, error) {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get kubeClient: %w", err)
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		c.logger.Error("获取 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	c.logger.Info("成功获取 ConfigMap YAML", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return configMap, nil
}

func (c *configMapService) DeleteConfigMap(ctx context.Context, id int, namespace, configMapName string) error {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get kubeClient: %w", err)
	}

	if err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil {
		c.logger.Error("删除 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete ConfigMap '%s': %w", configMapName, err)
	}

	c.logger.Info("成功删除 ConfigMap", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// BatchDeleteConfigMap 批量删除指定的 ConfigMap
func (c *configMapService) BatchDeleteConfigMap(ctx context.Context, id int, namespace string, configMapNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get kubeClient: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(configMapNames))

	for _, name := range configMapNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				c.logger.Error("删除 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errCh <- fmt.Errorf("failed to delete ConfigMap '%s': %w", name, err)
			} else {
				c.logger.Info("成功删除 ConfigMap", zap.String("configmap_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
			}
		}(name)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		c.logger.Error("批量删除 ConfigMap 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(configMapNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting ConfigMaps: %v", errs)
	}

	c.logger.Info("成功批量删除 ConfigMap", zap.Int("count", len(configMapNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// CreateConfigMapVersion 创建 ConfigMap 版本
func (c *configMapService) CreateConfigMapVersion(ctx context.Context, req *model.K8sConfigMapVersionRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取当前 ConfigMap
	currentConfigMap, err := kubeClient.CoreV1().ConfigMaps(req.Namespace).Get(ctx, req.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		c.logger.Error("获取当前 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get current ConfigMap: %w", err)
	}

	// 生成版本号（如果未提供）
	version := req.Version
	if version == "" {
		version = c.generateVersion(req.ClusterID, req.Namespace, req.ConfigMapName)
	}

	// 创建版本记录
	versionRecord := &model.K8sConfigMapVersion{
		Version:           version,
		Description:       req.Description,
		ConfigMap:         currentConfigMap.DeepCopy(),
		CreationTimestamp: time.Now(),
		Author:            "system", // 在实际应用中应获取真实用户信息
	}

	// 保存版本记录
	c.saveVersion(req.ClusterID, req.Namespace, req.ConfigMapName, versionRecord)

	c.logger.Info("成功创建 ConfigMap 版本", zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.String("version", version), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetConfigMapVersions 获取 ConfigMap 版本列表
func (c *configMapService) GetConfigMapVersions(ctx context.Context, id int, namespace, configMapName string) ([]*model.K8sConfigMapVersion, error) {
	c.versionMutex.RLock()
	defer c.versionMutex.RUnlock()

	key := c.getVersionKey(id, namespace, configMapName)
	if versions, exists := c.versionStore[key]; exists {
		var result []*model.K8sConfigMapVersion
		for _, version := range versions {
			result = append(result, version...)
		}
		c.logger.Info("成功获取 ConfigMap 版本列表", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.Int("versions_count", len(result)), zap.Int("cluster_id", id))
		return result, nil
	}

	c.logger.Info("未找到 ConfigMap 版本", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return []*model.K8sConfigMapVersion{}, nil
}

// GetConfigMapVersion 获取特定版本的 ConfigMap
func (c *configMapService) GetConfigMapVersion(ctx context.Context, id int, namespace, configMapName, version string) (*model.K8sConfigMapVersion, error) {
	versions, err := c.GetConfigMapVersions(ctx, id, namespace, configMapName)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if v.Version == version {
			c.logger.Info("成功获取 ConfigMap 版本", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.String("version", version), zap.Int("cluster_id", id))
			return v, nil
		}
	}

	c.logger.Error("未找到指定版本的 ConfigMap", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.String("version", version), zap.Int("cluster_id", id))
	return nil, fmt.Errorf("version %s not found for ConfigMap %s", version, configMapName)
}

// DeleteConfigMapVersion 删除 ConfigMap 版本
func (c *configMapService) DeleteConfigMapVersion(ctx context.Context, id int, namespace, configMapName, version string) error {
	c.versionMutex.Lock()
	defer c.versionMutex.Unlock()

	key := c.getVersionKey(id, namespace, configMapName)
	if namespaceVersions, exists := c.versionStore[key]; exists {
		for versionKey, versions := range namespaceVersions {
			for i, v := range versions {
				if v.Version == version {
					// 删除版本
					namespaceVersions[versionKey] = append(versions[:i], versions[i+1:]...)
					if len(namespaceVersions[versionKey]) == 0 {
						delete(namespaceVersions, versionKey)
					}
					if len(namespaceVersions) == 0 {
						delete(c.versionStore, key)
					}

					c.logger.Info("成功删除 ConfigMap 版本", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.String("version", version), zap.Int("cluster_id", id))
					return nil
				}
			}
		}
	}

	c.logger.Error("未找到要删除的 ConfigMap 版本", zap.String("configmap_name", configMapName), zap.String("namespace", namespace), zap.String("version", version), zap.Int("cluster_id", id))
	return fmt.Errorf("version %s not found for ConfigMap %s", version, configMapName)
}

// HotReloadConfigMap 热重载 ConfigMap
func (c *configMapService) HotReloadConfigMap(ctx context.Context, req *model.K8sConfigMapHotReloadRequest) (map[string]interface{}, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	result := map[string]interface{}{
		"configmap_name":     req.ConfigMapName,
		"namespace":          req.Namespace,
		"reload_type":        req.ReloadType,
		"reloaded_resources": make([]map[string]interface{}, 0),
		"summary":            make(map[string]interface{}),
	}

	var reloadedResources []map[string]interface{}
	var totalReloaded int

	switch req.ReloadType {
	case "pods":
		pods, err := c.reloadPodsUsingConfigMap(ctx, kubeClient, req.Namespace, req.ConfigMapName, req.TargetSelector)
		if err != nil {
			c.logger.Error("重载使用 ConfigMap 的 Pod 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
			return nil, fmt.Errorf("failed to reload pods: %w", err)
		}
		reloadedResources = append(reloadedResources, pods...)
		totalReloaded = len(pods)

	case "deployments":
		deployments, err := c.reloadDeploymentsUsingConfigMap(ctx, kubeClient, req.Namespace, req.ConfigMapName, req.TargetSelector)
		if err != nil {
			c.logger.Error("重载使用 ConfigMap 的 Deployment 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
			return nil, fmt.Errorf("failed to reload deployments: %w", err)
		}
		reloadedResources = append(reloadedResources, deployments...)
		totalReloaded = len(deployments)

	case "all":
		// 重载所有相关资源
		pods, err := c.reloadPodsUsingConfigMap(ctx, kubeClient, req.Namespace, req.ConfigMapName, req.TargetSelector)
		if err != nil {
			c.logger.Error("重载使用 ConfigMap 的 Pod 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
			return nil, fmt.Errorf("failed to reload pods: %w", err)
		}

		deployments, err := c.reloadDeploymentsUsingConfigMap(ctx, kubeClient, req.Namespace, req.ConfigMapName, req.TargetSelector)
		if err != nil {
			c.logger.Error("重载使用 ConfigMap 的 Deployment 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
			return nil, fmt.Errorf("failed to reload deployments: %w", err)
		}

		reloadedResources = append(reloadedResources, pods...)
		reloadedResources = append(reloadedResources, deployments...)
		totalReloaded = len(pods) + len(deployments)

	default:
		c.logger.Error("不支持的重载类型", zap.String("reload_type", req.ReloadType), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("unsupported reload type: %s", req.ReloadType)
	}

	result["reloaded_resources"] = reloadedResources
	result["summary"] = map[string]interface{}{
		"total_reloaded": totalReloaded,
		"reload_time":    time.Now(),
		"status":         "success",
	}

	c.logger.Info("成功热重载 ConfigMap", zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.String("reload_type", req.ReloadType), zap.Int("total_reloaded", totalReloaded), zap.Int("cluster_id", req.ClusterID))
	return result, nil
}

// RollbackConfigMap 回滚 ConfigMap
func (c *configMapService) RollbackConfigMap(ctx context.Context, req *model.K8sConfigMapRollbackRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, c.client, c.logger)
	if err != nil {
		c.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取目标版本
	targetVersion, err := c.GetConfigMapVersion(ctx, req.ClusterID, req.Namespace, req.ConfigMapName, req.TargetVersion)
	if err != nil {
		c.logger.Error("获取目标版本失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.String("target_version", req.TargetVersion), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get target version: %w", err)
	}

	// 获取当前 ConfigMap
	currentConfigMap, err := kubeClient.CoreV1().ConfigMaps(req.Namespace).Get(ctx, req.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		c.logger.Error("获取当前 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get current ConfigMap: %w", err)
	}

	// 先创建当前版本的备份
	backupReq := &model.K8sConfigMapVersionRequest{
		ClusterID:     req.ClusterID,
		Namespace:     req.Namespace,
		ConfigMapName: req.ConfigMapName,
		Version:       fmt.Sprintf("backup-before-rollback-%d", time.Now().Unix()),
		Description:   fmt.Sprintf("Backup before rollback to version %s", req.TargetVersion),
	}
	if err := c.CreateConfigMapVersion(ctx, backupReq); err != nil {
		c.logger.Error("创建回滚前备份失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create backup before rollback: %w", err)
	}

	// 执行回滚
	currentConfigMap.Data = targetVersion.ConfigMap.Data
	currentConfigMap.BinaryData = targetVersion.ConfigMap.BinaryData

	_, err = kubeClient.CoreV1().ConfigMaps(req.Namespace).Update(ctx, currentConfigMap, metav1.UpdateOptions{})
	if err != nil {
		c.logger.Error("回滚 ConfigMap 失败", zap.Error(err), zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.String("target_version", req.TargetVersion), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to rollback ConfigMap: %w", err)
	}

	c.logger.Info("成功回滚 ConfigMap", zap.String("configmap_name", req.ConfigMapName), zap.String("namespace", req.Namespace), zap.String("target_version", req.TargetVersion), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// 辅助方法

func (c *configMapService) generateVersion(clusterID int, namespace, configMapName string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("v%d-%d", timestamp, clusterID)
}

func (c *configMapService) getVersionKey(clusterID int, namespace, configMapName string) string {
	return fmt.Sprintf("%d:%s:%s", clusterID, namespace, configMapName)
}

func (c *configMapService) saveVersion(clusterID int, namespace, configMapName string, version *model.K8sConfigMapVersion) {
	c.versionMutex.Lock()
	defer c.versionMutex.Unlock()

	key := c.getVersionKey(clusterID, namespace, configMapName)
	if c.versionStore[key] == nil {
		c.versionStore[key] = make(map[string][]*model.K8sConfigMapVersion)
	}
	if c.versionStore[key][configMapName] == nil {
		c.versionStore[key][configMapName] = make([]*model.K8sConfigMapVersion, 0)
	}

	c.versionStore[key][configMapName] = append(c.versionStore[key][configMapName], version)
}

func (c *configMapService) reloadPodsUsingConfigMap(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, configMapName string, selector map[string]string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// 获取所有 Pod
	listOptions := metav1.ListOptions{}
	if len(selector) > 0 {
		listOptions.LabelSelector = labels.Set(selector).String()
	}

	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// 检查 Pod 是否使用了指定的 ConfigMap
		usesConfigMap := false

		// 检查卷挂载
		for _, volume := range pod.Spec.Volumes {
			if volume.ConfigMap != nil && volume.ConfigMap.Name == configMapName {
				usesConfigMap = true
				break
			}
		}

		// 检查环境变量
		if !usesConfigMap {
			for _, container := range pod.Spec.Containers {
				for _, env := range container.Env {
					if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == configMapName {
						usesConfigMap = true
						break
					}
				}
				if usesConfigMap {
					break
				}
			}
		}

		if usesConfigMap {
			// 重启 Pod（通过删除让控制器重新创建）
			err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil {
				c.logger.Error("删除 Pod 失败", zap.Error(err), zap.String("pod_name", pod.Name), zap.String("namespace", namespace))
				continue
			}

			result = append(result, map[string]interface{}{
				"type":      "Pod",
				"name":      pod.Name,
				"namespace": namespace,
				"action":    "restarted",
			})
		}
	}

	return result, nil
}

func (c *configMapService) reloadDeploymentsUsingConfigMap(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, configMapName string, selector map[string]string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// 获取所有 Deployment
	listOptions := metav1.ListOptions{}
	if len(selector) > 0 {
		listOptions.LabelSelector = labels.Set(selector).String()
	}

	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		// 检查 Deployment 是否使用了指定的 ConfigMap
		usesConfigMap := false

		// 检查卷挂载
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil && volume.ConfigMap.Name == configMapName {
				usesConfigMap = true
				break
			}
		}

		// 检查环境变量
		if !usesConfigMap {
			for _, container := range deployment.Spec.Template.Spec.Containers {
				for _, env := range container.Env {
					if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == configMapName {
						usesConfigMap = true
						break
					}
				}
				if usesConfigMap {
					break
				}
			}
		}

		if usesConfigMap {
			// 触发 Deployment 重新部署（通过更新注解）
			if deployment.Spec.Template.Annotations == nil {
				deployment.Spec.Template.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

			_, err := kubeClient.AppsV1().Deployments(namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
			if err != nil {
				c.logger.Error("更新 Deployment 失败", zap.Error(err), zap.String("deployment_name", deployment.Name), zap.String("namespace", namespace))
				continue
			}

			result = append(result, map[string]interface{}{
				"type":      "Deployment",
				"name":      deployment.Name,
				"namespace": namespace,
				"action":    "restarted",
			})
		}
	}

	return result, nil
}
