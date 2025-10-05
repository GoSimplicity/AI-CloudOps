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

type ClusterRoleManager interface {
	CreateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error
	GetClusterRole(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRole, error)
	GetClusterRoleList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) ([]*model.K8sClusterRole, error)
	UpdateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error
	DeleteClusterRole(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error
}

type clusterRoleManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewClusterRoleManager(client client.K8sClient, logger *zap.Logger) ClusterRoleManager {
	return &clusterRoleManager{
		client: client,
		logger: logger,
	}
}

func (m *clusterRoleManager) CreateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建ClusterRole失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", clusterRole.Name))
		return fmt.Errorf("创建ClusterRole %s 失败: %w", clusterRole.Name, err)
	}

	m.logger.Info("成功创建ClusterRole",
		zap.Int("cluster_id", clusterID), zap.String("name", clusterRole.Name))
	return nil
}

func (m *clusterRoleManager) GetClusterRole(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRole, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	clusterRole, err := clientset.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取ClusterRole失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", name))
		return nil, fmt.Errorf("获取ClusterRole %s 失败: %w", name, err)
	}

	return clusterRole, nil
}

func (m *clusterRoleManager) GetClusterRoleList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) ([]*model.K8sClusterRole, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取ClusterRole列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取ClusterRole列表失败: %w", err)
	}

	var k8sClusterRoles []*model.K8sClusterRole
	for _, cr := range clusterRoles.Items {
		k8sClusterRole := &model.K8sClusterRole{
			ClusterID:       clusterID,
			Name:            cr.Name,
			UID:             string(cr.UID),
			CreatedAt:       cr.CreationTimestamp.Time.Format(time.RFC3339),
			Labels:          cr.Labels,
			Annotations:     cr.Annotations,
			ResourceVersion: cr.ResourceVersion,
			RawClusterRole:  &cr,
		}
		k8sClusterRoles = append(k8sClusterRoles, k8sClusterRole)
	}

	m.logger.Debug("成功获取ClusterRole列表",
		zap.Int("cluster_id", clusterID), zap.Int("count", len(clusterRoles.Items)))

	return k8sClusterRoles, nil
}

func (m *clusterRoleManager) UpdateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新ClusterRole失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", clusterRole.Name))
		return fmt.Errorf("更新ClusterRole %s 失败: %w", clusterRole.Name, err)
	}

	m.logger.Info("成功更新ClusterRole",
		zap.Int("cluster_id", clusterID), zap.String("name", clusterRole.Name))
	return nil
}

func (m *clusterRoleManager) DeleteClusterRole(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.RbacV1().ClusterRoles().Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除ClusterRole失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", name))
		return fmt.Errorf("删除ClusterRole %s 失败: %w", name, err)
	}

	m.logger.Info("成功删除ClusterRole",
		zap.Int("cluster_id", clusterID), zap.String("name", name))
	return nil
}
