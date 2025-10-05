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

package manager

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapManager interface {
	GetConfigMap(ctx context.Context, clusterID int, namespace, name string) (*corev1.ConfigMap, error)
	ListConfigMaps(ctx context.Context, clusterID int, namespace string) (*corev1.ConfigMapList, error)
	CreateConfigMap(ctx context.Context, clusterID int, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	UpdateConfigMap(ctx context.Context, clusterID int, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error

	ListConfigMapsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.ConfigMapList, error)
}

type configMapManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewConfigMapManager(client client.K8sClient, logger *zap.Logger) ConfigMapManager {
	return &configMapManager{
		client: client,
		logger: logger,
	}
}

func (m *configMapManager) GetConfigMap(ctx context.Context, clusterID int, namespace, name string) (*corev1.ConfigMap, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取ConfigMap %s/%s 失败: %w", namespace, name, err)
	}

	return configMap, nil
}

func (m *configMapManager) ListConfigMaps(ctx context.Context, clusterID int, namespace string) (*corev1.ConfigMapList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取ConfigMap列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取ConfigMap列表失败: %w", err)
	}

	return configMaps, nil
}

func (m *configMapManager) CreateConfigMap(ctx context.Context, clusterID int, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	createdConfigMap, err := clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", configMap.Namespace), zap.String("name", configMap.Name))
		return nil, fmt.Errorf("创建ConfigMap %s/%s 失败: %w", configMap.Namespace, configMap.Name, err)
	}

	m.logger.Info("成功创建ConfigMap",
		zap.Int("cluster_id", clusterID), zap.String("namespace", createdConfigMap.Namespace), zap.String("name", createdConfigMap.Name))
	return createdConfigMap, nil
}

func (m *configMapManager) UpdateConfigMap(ctx context.Context, clusterID int, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	updatedConfigMap, err := clientset.CoreV1().ConfigMaps(configMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", configMap.Namespace), zap.String("name", configMap.Name))
		return nil, fmt.Errorf("更新ConfigMap %s/%s 失败: %w", configMap.Namespace, configMap.Name, err)
	}

	m.logger.Info("成功更新ConfigMap",
		zap.Int("cluster_id", clusterID), zap.String("namespace", updatedConfigMap.Namespace), zap.String("name", updatedConfigMap.Name))
	return updatedConfigMap, nil
}

func (m *configMapManager) DeleteConfigMap(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除ConfigMap失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除ConfigMap %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除ConfigMap",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
	return nil
}

func (m *configMapManager) ListConfigMapsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.ConfigMapList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("根据选择器获取ConfigMap列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("selector", selector))
		return nil, fmt.Errorf("根据选择器获取ConfigMap列表失败: %w", err)
	}

	return configMaps, nil
}
