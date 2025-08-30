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

type NamespaceManager interface {
	GetNamespace(ctx context.Context, clusterID int, name string) (*corev1.Namespace, error)
	ListNamespaces(ctx context.Context, clusterID int) (*corev1.NamespaceList, int64, error)
	CreateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error)
	UpdateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error)
	DeleteNamespace(ctx context.Context, clusterID int, name string, options metav1.DeleteOptions) error
}

type namespaceManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewNamespaceManager(client client.K8sClient, logger *zap.Logger) NamespaceManager {
	return &namespaceManager{
		client: client,
		logger: logger,
	}
}

func (m *namespaceManager) GetNamespace(ctx context.Context, clusterID int, name string) (*corev1.Namespace, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取命名空间失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("namespace", name))
		return nil, fmt.Errorf("获取命名空间失败: %w", err)
	}

	return namespace, nil
}

func (m *namespaceManager) ListNamespaces(ctx context.Context, clusterID int) (*corev1.NamespaceList, int64, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, 0, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取命名空间列表失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, 0, fmt.Errorf("获取命名空间列表失败: %w", err)
	}

	// 获取命名空间总数
	total := int64(len(namespaces.Items))

	return namespaces, total, nil
}

func (m *namespaceManager) CreateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	createdNamespace, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建命名空间失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("namespace", namespace.Name))
		return nil, fmt.Errorf("创建命名空间失败: %w", err)
	}

	return createdNamespace, nil
}

func (m *namespaceManager) UpdateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	updatedNamespace, err := clientset.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新命名空间失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("namespace", namespace.Name))
		return nil, fmt.Errorf("更新命名空间失败: %w", err)
	}

	return updatedNamespace, nil
}

func (m *namespaceManager) DeleteNamespace(ctx context.Context, clusterID int, name string, options metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().Namespaces().Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除命名空间失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("namespace", name))
		return fmt.Errorf("删除命名空间失败: %w", err)
	}

	return nil
}
