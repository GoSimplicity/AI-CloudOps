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

type RoleManager interface {
	CreateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error
	GetRole(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.Role, error)
	GetRoleList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRole, error)
	GetRoleListRaw(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleList, error)
	UpdateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error
	DeleteRole(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
}

type roleManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewRoleManager(client client.K8sClient, logger *zap.Logger) RoleManager {
	return &roleManager{
		client: client,
		logger: logger,
	}
}

func (m *roleManager) CreateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建Role失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", role.Name))
		return fmt.Errorf("创建Role %s/%s 失败: %w", namespace, role.Name, err)
	}

	m.logger.Info("成功创建Role",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", role.Name))
	return nil
}

func (m *roleManager) GetRole(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.Role, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	role, err := clientset.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Role失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取Role %s/%s 失败: %w", namespace, name, err)
	}

	return role, nil
}

func (m *roleManager) GetRoleList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRole, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	roles, err := clientset.RbacV1().Roles(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取Role列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Role列表失败: %w", err)
	}

	var k8sRoles []*model.K8sRole
	for _, role := range roles.Items {
		k8sRole := &model.K8sRole{
			ClusterID:       clusterID,
			Name:            role.Name,
			Namespace:       role.Namespace,
			UID:             string(role.UID),
			CreatedAt:       role.CreationTimestamp.Time.Format(time.RFC3339),
			Labels:          role.Labels,
			Annotations:     role.Annotations,
			ResourceVersion: role.ResourceVersion,
			RawRole:         &role,
		}
		k8sRoles = append(k8sRoles, k8sRole)
	}

	m.logger.Debug("成功获取Role列表",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.Int("count", len(roles.Items)))

	return k8sRoles, nil
}

func (m *roleManager) GetRoleListRaw(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	roles, err := clientset.RbacV1().Roles(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取Role列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Role列表失败: %w", err)
	}

	m.logger.Debug("成功获取Role列表",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.Int("count", len(roles.Items)))

	return roles, nil
}

func (m *roleManager) UpdateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新Role失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", role.Name))
		return fmt.Errorf("更新Role %s/%s 失败: %w", namespace, role.Name, err)
	}

	m.logger.Info("成功更新Role",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", role.Name))
	return nil
}

func (m *roleManager) DeleteRole(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.RbacV1().Roles(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除Role失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除Role %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除Role",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
	return nil
}
