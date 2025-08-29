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

// NamespaceManager 命名空间管理器接口
type NamespaceManager interface {
	// 基础CRUD操作
	GetNamespace(ctx context.Context, clusterID int, name string) (*corev1.Namespace, error)
	ListNamespaces(ctx context.Context, clusterID int) (*corev1.NamespaceList, error)
	CreateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error)
	UpdateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error)
	DeleteNamespace(ctx context.Context, clusterID int, name string, options metav1.DeleteOptions) error

	// 业务功能
	GetNamespaceEvents(ctx context.Context, clusterID int, namespaceName string) (*corev1.EventList, error)
	GetNamespaceResourceQuota(ctx context.Context, clusterID int, namespaceName string) (*corev1.ResourceQuotaList, error)
	GetNamespaceLimitRanges(ctx context.Context, clusterID int, namespaceName string) (*corev1.LimitRangeList, error)
}

// namespaceManager 命名空间管理器实现
type namespaceManager struct {
	client client.K8sClient
	logger *zap.Logger
}

// NewNamespaceManager 创建命名空间管理器
func NewNamespaceManager(client client.K8sClient, logger *zap.Logger) NamespaceManager {
	return &namespaceManager{
		client: client,
		logger: logger,
	}
}

// GetNamespace 获取单个命名空间
func (m *namespaceManager) GetNamespace(ctx context.Context, clusterID int, name string) (*corev1.Namespace, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取命名空间失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", name))
		return nil, fmt.Errorf("获取命名空间 %s 失败: %w", name, err)
	}

	return namespace, nil
}

// ListNamespaces 获取命名空间列表
func (m *namespaceManager) ListNamespaces(ctx context.Context, clusterID int) (*corev1.NamespaceList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取命名空间列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取命名空间列表失败: %w", err)
	}

	return namespaces, nil
}

// CreateNamespace 创建命名空间
func (m *namespaceManager) CreateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	createdNamespace, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建命名空间失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace.Name))
		return nil, fmt.Errorf("创建命名空间 %s 失败: %w", namespace.Name, err)
	}

	m.logger.Info("成功创建命名空间",
		zap.Int("cluster_id", clusterID), zap.String("namespace", createdNamespace.Name))
	return createdNamespace, nil
}

// UpdateNamespace 更新命名空间
func (m *namespaceManager) UpdateNamespace(ctx context.Context, clusterID int, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	updatedNamespace, err := clientset.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新命名空间失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace.Name))
		return nil, fmt.Errorf("更新命名空间 %s 失败: %w", namespace.Name, err)
	}

	m.logger.Info("成功更新命名空间",
		zap.Int("cluster_id", clusterID), zap.String("namespace", updatedNamespace.Name))
	return updatedNamespace, nil
}

// DeleteNamespace 删除命名空间
func (m *namespaceManager) DeleteNamespace(ctx context.Context, clusterID int, name string, options metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().Namespaces().Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除命名空间失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", name))
		return fmt.Errorf("删除命名空间 %s 失败: %w", name, err)
	}

	m.logger.Info("成功删除命名空间",
		zap.Int("cluster_id", clusterID), zap.String("namespace", name))
	return nil
}

// GetNamespaceEvents 获取命名空间事件
func (m *namespaceManager) GetNamespaceEvents(ctx context.Context, clusterID int, namespaceName string) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	events, err := clientset.CoreV1().Events(namespaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取命名空间事件失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespaceName))
		return nil, fmt.Errorf("获取命名空间 %s 事件失败: %w", namespaceName, err)
	}

	return events, nil
}

// GetNamespaceResourceQuota 获取命名空间资源配额
func (m *namespaceManager) GetNamespaceResourceQuota(ctx context.Context, clusterID int, namespaceName string) (*corev1.ResourceQuotaList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	quotas, err := clientset.CoreV1().ResourceQuotas(namespaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取资源配额失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespaceName))
		return nil, fmt.Errorf("获取命名空间 %s 资源配额失败: %w", namespaceName, err)
	}

	return quotas, nil
}

// GetNamespaceLimitRanges 获取命名空间限制范围
func (m *namespaceManager) GetNamespaceLimitRanges(ctx context.Context, clusterID int, namespaceName string) (*corev1.LimitRangeList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	limitRanges, err := clientset.CoreV1().LimitRanges(namespaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取限制范围失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespaceName))
		return nil, fmt.Errorf("获取命名空间 %s 限制范围失败: %w", namespaceName, err)
	}

	return limitRanges, nil
}
