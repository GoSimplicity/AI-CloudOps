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

type SecretManager interface {
	GetSecret(ctx context.Context, clusterID int, namespace, name string) (*corev1.Secret, error)
	ListSecrets(ctx context.Context, clusterID int, namespace string) (*corev1.SecretList, error)
	CreateSecret(ctx context.Context, clusterID int, secret *corev1.Secret) (*corev1.Secret, error)
	UpdateSecret(ctx context.Context, clusterID int, secret *corev1.Secret) (*corev1.Secret, error)
	DeleteSecret(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error

	// 业务功能
	ListSecretsBySelectors(ctx context.Context, clusterID int, namespace string, labelSelector string, fieldSelector string) (*corev1.SecretList, error)
}

// secretManager Secret管理器实现
type secretManager struct {
	client client.K8sClient
	logger *zap.Logger
}

// NewSecretManager 创建Secret管理器
func NewSecretManager(client client.K8sClient, logger *zap.Logger) SecretManager {
	return &secretManager{
		client: client,
		logger: logger,
	}
}

func (m *secretManager) GetSecret(ctx context.Context, clusterID int, namespace, name string) (*corev1.Secret, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取Secret %s/%s 失败: %w", namespace, name, err)
	}

	return secret, nil
}

func (m *secretManager) ListSecrets(ctx context.Context, clusterID int, namespace string) (*corev1.SecretList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Secret列表失败: %w", err)
	}

	return secrets, nil
}

func (m *secretManager) CreateSecret(ctx context.Context, clusterID int, secret *corev1.Secret) (*corev1.Secret, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	createdSecret, err := clientset.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建Secret失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", secret.Namespace), zap.String("name", secret.Name))
		return nil, fmt.Errorf("创建Secret %s/%s 失败: %w", secret.Namespace, secret.Name, err)
	}

	m.logger.Info("成功创建Secret",
		zap.Int("cluster_id", clusterID), zap.String("namespace", createdSecret.Namespace), zap.String("name", createdSecret.Name))
	return createdSecret, nil
}

func (m *secretManager) UpdateSecret(ctx context.Context, clusterID int, secret *corev1.Secret) (*corev1.Secret, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	updatedSecret, err := clientset.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新Secret失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", secret.Namespace), zap.String("name", secret.Name))
		return nil, fmt.Errorf("更新Secret %s/%s 失败: %w", secret.Namespace, secret.Name, err)
	}

	m.logger.Info("成功更新Secret",
		zap.Int("cluster_id", clusterID), zap.String("namespace", updatedSecret.Namespace), zap.String("name", updatedSecret.Name))
	return updatedSecret, nil
}

func (m *secretManager) DeleteSecret(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().Secrets(namespace).Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除Secret失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除Secret %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除Secret",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
	return nil
}

// BatchDeleteSecrets 批量删除Secret
// 删除未使用的批量删除接口以简化实现

// 删除未使用的单一选择器方法，保留复合选择器方法

func (m *secretManager) ListSecretsBySelectors(ctx context.Context, clusterID int, namespace string, labelSelector string, fieldSelector string) (*corev1.SecretList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}
	if fieldSelector != "" {
		listOptions.FieldSelector = fieldSelector
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("根据选择器获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace),
			zap.String("label_selector", labelSelector), zap.String("field_selector", fieldSelector))
		return nil, fmt.Errorf("根据选择器获取Secret列表失败: %w", err)
	}

	return secrets, nil
}

// 删除未使用的按类型过滤方法

// 删除未使用的数据读取方法

// 删除未使用的数据更新方法
