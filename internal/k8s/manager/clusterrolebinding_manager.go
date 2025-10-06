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

type ClusterRoleBindingManager interface {
	CreateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error
	GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRoleBinding, error)
	GetClusterRoleBindingList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) ([]*model.K8sClusterRoleBinding, error)
	GetClusterRoleBindingListRaw(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error)
	UpdateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error
	DeleteClusterRoleBinding(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error
}

type clusterRoleBindingManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewClusterRoleBindingManager(clientFactory client.K8sClient, logger *zap.Logger) ClusterRoleBindingManager {
	return &clusterRoleBindingManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

func (m *clusterRoleBindingManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

func (m *clusterRoleBindingManager) CreateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	if clusterRoleBinding == nil {
		return fmt.Errorf("clusterRoleBinding 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建 ClusterRoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", clusterRoleBinding.Name),
			zap.Error(err))
		return fmt.Errorf("创建 ClusterRoleBinding 失败: %w", err)
	}

	m.logger.Info("成功创建 ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", clusterRoleBinding.Name))
	return nil
}

func (m *clusterRoleBindingManager) GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRoleBinding, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	clusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 ClusterRoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 ClusterRoleBinding 失败: %w", err)
	}

	m.logger.Debug("成功获取 ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))
	return clusterRoleBinding, nil
}

func (m *clusterRoleBindingManager) GetClusterRoleBindingList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) ([]*model.K8sClusterRoleBinding, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	clusterRoleBindingList, err := kubeClient.RbacV1().ClusterRoleBindings().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 ClusterRoleBinding 列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取 ClusterRoleBinding 列表失败: %w", err)
	}

	var k8sClusterRoleBindings []*model.K8sClusterRoleBinding
	for _, crb := range clusterRoleBindingList.Items {
		k8sClusterRoleBinding := utils.ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(&crb, clusterID)
		// 添加原始对象引用
		k8sClusterRoleBinding.RawClusterRoleBinding = &crb
		// 计算 Age 使用统一的utils函数
		k8sClusterRoleBinding.Age = utils.CalculateAge(crb.CreationTimestamp.Time)
		k8sClusterRoleBindings = append(k8sClusterRoleBindings, &k8sClusterRoleBinding)
	}

	m.logger.Debug("成功获取 ClusterRoleBinding 列表",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(k8sClusterRoleBindings)))
	return k8sClusterRoleBindings, nil
}

func (m *clusterRoleBindingManager) GetClusterRoleBindingListRaw(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	clusterRoleBindingList, err := kubeClient.RbacV1().ClusterRoleBindings().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 ClusterRoleBinding 列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取 ClusterRoleBinding 列表失败: %w", err)
	}

	m.logger.Debug("成功获取 ClusterRoleBinding 列表",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(clusterRoleBindingList.Items)))
	return clusterRoleBindingList, nil
}

func (m *clusterRoleBindingManager) UpdateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	if clusterRoleBinding == nil {
		return fmt.Errorf("clusterRoleBinding 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新 ClusterRoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", clusterRoleBinding.Name),
			zap.Error(err))
		return fmt.Errorf("更新 ClusterRoleBinding 失败: %w", err)
	}

	m.logger.Info("成功更新 ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", clusterRoleBinding.Name))
	return nil
}

func (m *clusterRoleBindingManager) DeleteClusterRoleBinding(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 ClusterRoleBinding 失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 ClusterRoleBinding 失败: %w", err)
	}

	m.logger.Info("成功删除 ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))
	return nil
}
