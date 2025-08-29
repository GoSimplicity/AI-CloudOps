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

// SecretManager Secret管理器接口
type SecretManager interface {
	// 基础CRUD操作
	GetSecret(ctx context.Context, clusterID int, namespace, name string) (*corev1.Secret, error)
	ListSecrets(ctx context.Context, clusterID int, namespace string) (*corev1.SecretList, error)
	CreateSecret(ctx context.Context, clusterID int, secret *corev1.Secret) (*corev1.Secret, error)
	UpdateSecret(ctx context.Context, clusterID int, secret *corev1.Secret) (*corev1.Secret, error)
	DeleteSecret(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error

	// 批量操作
	BatchDeleteSecrets(ctx context.Context, clusterID int, namespace string, secretNames []string, options metav1.DeleteOptions) error

	// 业务功能
	ListSecretsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.SecretList, error)
	ListSecretsByType(ctx context.Context, clusterID int, namespace string, secretType corev1.SecretType) (*corev1.SecretList, error)
	GetSecretData(ctx context.Context, clusterID int, namespace, name string, key string) ([]byte, error)
	UpdateSecretData(ctx context.Context, clusterID int, namespace, name string, data map[string][]byte) (*corev1.Secret, error)
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

// GetSecret 获取单个Secret
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

// ListSecrets 获取Secret列表
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

// CreateSecret 创建Secret
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

// UpdateSecret 更新Secret
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

// DeleteSecret 删除Secret
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
func (m *secretManager) BatchDeleteSecrets(ctx context.Context, clusterID int, namespace string, secretNames []string, options metav1.DeleteOptions) error {
	if len(secretNames) == 0 {
		return nil
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var errors []error
	for _, secretName := range secretNames {
		if err := clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, options); err != nil {
			m.logger.Error("批量删除Secret失败", zap.Error(err),
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", secretName))
			errors = append(errors, fmt.Errorf("删除Secret %s/%s 失败: %w", namespace, secretName, err))
		} else {
			m.logger.Info("成功删除Secret",
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", secretName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("批量删除Secret时发生 %d 个错误: %v", len(errors), errors)
	}

	return nil
}

// ListSecretsBySelector 根据选择器获取Secret列表
func (m *secretManager) ListSecretsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.SecretList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("根据选择器获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("selector", selector))
		return nil, fmt.Errorf("根据选择器获取Secret列表失败: %w", err)
	}

	return secrets, nil
}

// ListSecretsByType 根据类型获取Secret列表
func (m *secretManager) ListSecretsByType(ctx context.Context, clusterID int, namespace string, secretType corev1.SecretType) (*corev1.SecretList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	secretList, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Secret列表失败: %w", err)
	}

	// 过滤指定类型的Secret
	filteredSecrets := &corev1.SecretList{
		TypeMeta: secretList.TypeMeta,
		ListMeta: secretList.ListMeta,
		Items:    []corev1.Secret{},
	}

	for _, secret := range secretList.Items {
		if secret.Type == secretType {
			filteredSecrets.Items = append(filteredSecrets.Items, secret)
		}
	}

	return filteredSecrets, nil
}

// GetSecretData 获取Secret中特定键的数据
func (m *secretManager) GetSecretData(ctx context.Context, clusterID int, namespace, name string, key string) ([]byte, error) {
	secret, err := m.GetSecret(ctx, clusterID, namespace, name)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("secret %s/%s 没有数据", namespace, name)
	}

	value, exists := secret.Data[key]
	if !exists {
		return nil, fmt.Errorf("secret %s/%s 中不存在键 %s", namespace, name, key)
	}

	return value, nil
}

// UpdateSecretData 更新Secret的数据
func (m *secretManager) UpdateSecretData(ctx context.Context, clusterID int, namespace, name string, data map[string][]byte) (*corev1.Secret, error) {
	secret, err := m.GetSecret(ctx, clusterID, namespace, name)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	// 更新数据
	for key, value := range data {
		secret.Data[key] = value
	}

	return m.UpdateSecret(ctx, clusterID, secret)
}
