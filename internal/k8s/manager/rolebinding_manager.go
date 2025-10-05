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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleBindingManager interface {
	CreateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error
	GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.RoleBinding, error)
	GetRoleBindingList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRoleBinding, error)
	UpdateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error
	DeleteRoleBinding(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
}

type roleBindingManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewRoleBindingManager(client client.K8sClient, logger *zap.Logger) RoleBindingManager {
	return &roleBindingManager{
		client: client,
		logger: logger,
	}
}

func (m *roleBindingManager) CreateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建RoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", roleBinding.Name))
		return fmt.Errorf("创建RoleBinding %s/%s 失败: %w", namespace, roleBinding.Name, err)
	}

	m.logger.Info("成功创建RoleBinding",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", roleBinding.Name))
	return nil
}

func (m *roleBindingManager) GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.RoleBinding, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	roleBinding, err := clientset.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取RoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取RoleBinding %s/%s 失败: %w", namespace, name, err)
	}

	return roleBinding, nil
}

func (m *roleBindingManager) GetRoleBindingList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRoleBinding, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取RoleBinding列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取RoleBinding列表失败: %w", err)
	}

	var k8sRoleBindings []*model.K8sRoleBinding
	for _, rb := range roleBindings.Items {

		k8sRoleBinding := &model.K8sRoleBinding{
			ClusterID:   clusterID,
			Name:        rb.Name,
			Namespace:   rb.Namespace,
			CreatedAt:   rb.CreationTimestamp.Time.Format(time.RFC3339),
			Labels:      rb.Labels,
			Annotations: rb.Annotations,
		}
		k8sRoleBindings = append(k8sRoleBindings, k8sRoleBinding)
	}

	m.logger.Debug("成功获取RoleBinding列表",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.Int("count", len(roleBindings.Items)))

	return k8sRoleBindings, nil
}

func (m *roleBindingManager) UpdateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新RoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", roleBinding.Name))
		return fmt.Errorf("更新RoleBinding %s/%s 失败: %w", namespace, roleBinding.Name, err)
	}

	m.logger.Info("成功更新RoleBinding",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", roleBinding.Name))
	return nil
}

func (m *roleBindingManager) DeleteRoleBinding(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.RbacV1().RoleBindings(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除RoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除RoleBinding %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除RoleBinding",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
	return nil
}
