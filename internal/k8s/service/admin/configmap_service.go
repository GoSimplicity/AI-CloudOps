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

package admin

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapService interface {
	GetConfigMapsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.ConfigMap, error)
	UpdateConfigMap(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	GetConfigMapYaml(ctx context.Context, id int, namespace, configMapName string) (*corev1.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, id int, namespace, configMapName string) error
	BatchDeleteConfigMap(ctx context.Context, id int, namespace string, configMapNames []string) error
}

type configMapService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewConfigMapService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) ConfigMapService {
	return &configMapService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetConfigMapsByNamespace 获取命名空间的所有 ConfigMap
func (c *configMapService) GetConfigMapsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.ConfigMap, error) {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeClient: %w", err)
	}

	configMapList, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to get ConfigMap list", zap.Error(err))
		return nil, fmt.Errorf("failed to list ConfigMaps: %w", err)
	}

	configMaps := make([]*corev1.ConfigMap, len(configMapList.Items))
	for i := range configMapList.Items {
		configMaps[i] = &configMapList.Items[i]
	}

	return configMaps, nil
}

// UpdateConfigMap 更新 ConfigMap
func (c *configMapService) UpdateConfigMap(ctx context.Context, configMapRequest *model.K8sConfigMapRequest) error {
	kubeClient, err := pkg.GetKubeClient(configMapRequest.ClusterId, c.client, c.logger)
	if err != nil {
		return fmt.Errorf("failed to get kubeClient: %w", err)
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(configMapRequest.ConfigMap.Namespace).Get(ctx, configMapRequest.ConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		c.logger.Error("Failed to get ConfigMap", zap.Error(err))
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
		c.logger.Error("Failed to update ConfigMap", zap.Error(err))
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	return nil
}

// GetConfigMapYaml 获取 ConfigMap 详情
func (c *configMapService) GetConfigMapYaml(ctx context.Context, id int, namespace, configMapName string) (*corev1.ConfigMap, error) {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeClient: %w", err)
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		c.logger.Error("Failed to get ConfigMap", zap.Error(err))
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	return configMap, nil
}

func (c *configMapService) DeleteConfigMap(ctx context.Context, id int, namespace, configMapName string) error {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		return fmt.Errorf("failed to get kubeClient: %w", err)
	}

	// 删除指定的 ConfigMap
	if err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil {
		c.logger.Error("Failed to delete ConfigMap", zap.String("configMapName", configMapName), zap.Error(err))
		return fmt.Errorf("failed to delete ConfigMap '%s': %w", configMapName, err)
	}

	return nil
}

// BatchDeleteConfigMap 批量删除指定的 ConfigMap
func (c *configMapService) BatchDeleteConfigMap(ctx context.Context, id int, namespace string, configMapNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.logger)
	if err != nil {
		return fmt.Errorf("failed to get kubeClient: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(configMapNames))

	for _, name := range configMapNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				c.logger.Error("Failed to delete ConfigMap", zap.String("configMapName", name), zap.Error(err))
				errCh <- fmt.Errorf("failed to delete ConfigMap '%s': %w", name, err)
			} else {
				c.logger.Info("Successfully deleted ConfigMap", zap.String("configMapName", name))
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
		return fmt.Errorf("errors occurred while deleting ConfigMaps: %v", errs)
	}

	return nil
}
