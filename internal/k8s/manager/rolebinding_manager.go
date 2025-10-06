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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RoleBindingManager interface {
	CreateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error
	GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.RoleBinding, error)
	GetRoleBindingList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRoleBinding, error)
	UpdateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error
	DeleteRoleBinding(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
}

type roleBindingManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewRoleBindingManager(clientFactory client.K8sClient, logger *zap.Logger) RoleBindingManager {
	return &roleBindingManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

func (m *roleBindingManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

func (m *roleBindingManager) CreateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error {
	if roleBinding == nil {
		return fmt.Errorf("roleBinding 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建 RoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", roleBinding.Name),
			zap.Error(err))
		return fmt.Errorf("创建 RoleBinding 失败: %w", err)
	}

	m.logger.Info("成功创建 RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", roleBinding.Name))
	return nil
}

func (m *roleBindingManager) GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.RoleBinding, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	roleBinding, err := kubeClient.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 RoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 RoleBinding 失败: %w", err)
	}

	m.logger.Debug("成功获取 RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return roleBinding, nil
}

func (m *roleBindingManager) GetRoleBindingList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRoleBinding, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	roleBindingList, err := kubeClient.RbacV1().RoleBindings(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 RoleBinding 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 RoleBinding 列表失败: %w", err)
	}

	var k8sRoleBindings []*model.K8sRoleBinding
	for _, rb := range roleBindingList.Items {
		// 使用统一的转换函数确保所有字段都被正确填充
		k8sRoleBinding := utils.ConvertK8sRoleBindingToRoleBindingInfo(&rb, clusterID)
		k8sRoleBindings = append(k8sRoleBindings, k8sRoleBinding)
	}

	m.logger.Debug("成功获取 RoleBinding 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sRoleBindings)))
	return k8sRoleBindings, nil
}

func (m *roleBindingManager) UpdateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error {
	if roleBinding == nil {
		return fmt.Errorf("roleBinding 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新 RoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", roleBinding.Name),
			zap.Error(err))
		return fmt.Errorf("更新 RoleBinding 失败: %w", err)
	}

	m.logger.Info("成功更新 RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", roleBinding.Name))
	return nil
}

func (m *roleBindingManager) DeleteRoleBinding(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.RbacV1().RoleBindings(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 RoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 RoleBinding 失败: %w", err)
	}

	m.logger.Info("成功删除 RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}
