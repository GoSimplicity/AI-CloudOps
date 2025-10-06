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

type RoleManager interface {
	CreateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error
	GetRole(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.Role, error)
	GetRoleList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRole, error)
	GetRoleListRaw(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleList, error)
	UpdateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error
	DeleteRole(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
}

type roleManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewRoleManager(clientFactory client.K8sClient, logger *zap.Logger) RoleManager {
	return &roleManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

func (m *roleManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

func (m *roleManager) CreateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error {
	if role == nil {
		return fmt.Errorf("role 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建 Role 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", role.Name),
			zap.Error(err))
		return fmt.Errorf("创建 Role 失败: %w", err)
	}

	m.logger.Info("成功创建 Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", role.Name))
	return nil
}

func (m *roleManager) GetRole(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.Role, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	role, err := kubeClient.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Role 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Role 失败: %w", err)
	}

	m.logger.Info("成功获取 Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int("rulesCount", len(role.Rules)))

	// 详细记录规则信息
	if len(role.Rules) > 0 {
		for i, rule := range role.Rules {
			m.logger.Debug("Role规则详情",
				zap.Int("ruleIndex", i),
				zap.Strings("verbs", rule.Verbs),
				zap.Strings("apiGroups", rule.APIGroups),
				zap.Strings("resources", rule.Resources),
				zap.Strings("resourceNames", rule.ResourceNames))
		}
	}

	return role, nil
}

func (m *roleManager) GetRoleList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sRole, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	roleList, err := kubeClient.RbacV1().Roles(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 Role 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Role 列表失败: %w", err)
	}

	var k8sRoles []*model.K8sRole
	for _, role := range roleList.Items {
		// 使用统一的转换函数确保所有字段都被正确填充
		roleInfo := utils.ConvertK8sRoleToRoleInfo(&role, clusterID)
		k8sRoles = append(k8sRoles, &roleInfo)
	}

	m.logger.Debug("成功获取 Role 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sRoles)))
	return k8sRoles, nil
}

func (m *roleManager) GetRoleListRaw(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleList, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	roleList, err := kubeClient.RbacV1().Roles(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 Role 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Role 列表失败: %w", err)
	}

	m.logger.Debug("成功获取 Role 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(roleList.Items)))
	return roleList, nil
}

func (m *roleManager) UpdateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error {
	if role == nil {
		return fmt.Errorf("role 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新 Role 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", role.Name),
			zap.Error(err))
		return fmt.Errorf("更新 Role 失败: %w", err)
	}

	m.logger.Info("成功更新 Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", role.Name))
	return nil
}

func (m *roleManager) DeleteRole(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.RbacV1().Roles(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 Role 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Role 失败: %w", err)
	}

	m.logger.Info("成功删除 Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}
