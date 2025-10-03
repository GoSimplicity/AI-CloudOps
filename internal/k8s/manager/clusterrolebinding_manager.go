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
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterRoleBindingManager ClusterRoleBinding管理器接口
type ClusterRoleBindingManager interface {
	// 基础 CRUD 操作
	CreateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error
	GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRoleBinding, error)
	GetClusterRoleBindingList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) ([]*model.K8sClusterRoleBinding, error)
	GetClusterRoleBindingListRaw(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error)
	UpdateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error
	DeleteClusterRoleBinding(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error
}

type clusterRoleBindingManager struct {
	client client.K8sClient
	logger *zap.Logger
}

// NewClusterRoleBindingManager 创建ClusterRoleBinding管理器
func NewClusterRoleBindingManager(client client.K8sClient, logger *zap.Logger) ClusterRoleBindingManager {
	return &clusterRoleBindingManager{
		client: client,
		logger: logger,
	}
}

// CreateClusterRoleBinding 创建ClusterRoleBinding
func (m *clusterRoleBindingManager) CreateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建ClusterRoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", clusterRoleBinding.Name))
		return fmt.Errorf("创建ClusterRoleBinding %s 失败: %w", clusterRoleBinding.Name, err)
	}

	m.logger.Info("成功创建ClusterRoleBinding",
		zap.Int("cluster_id", clusterID), zap.String("name", clusterRoleBinding.Name))
	return nil
}

// GetClusterRoleBinding 获取单个ClusterRoleBinding
func (m *clusterRoleBindingManager) GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRoleBinding, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取ClusterRoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", name))
		return nil, fmt.Errorf("获取ClusterRoleBinding %s 失败: %w", name, err)
	}

	return clusterRoleBinding, nil
}

// GetClusterRoleBindingList 获取ClusterRoleBinding列表（转换为模型）
func (m *clusterRoleBindingManager) GetClusterRoleBindingList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) ([]*model.K8sClusterRoleBinding, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取ClusterRoleBinding列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取ClusterRoleBinding列表失败: %w", err)
	}

	// 转换为模型格式
	var k8sClusterRoleBindings []*model.K8sClusterRoleBinding
	for _, crb := range clusterRoleBindings.Items {
		// 使用 utils 中的转换函数，确保所有字段都被正确填充
		k8sClusterRoleBinding := k8sutils.ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(&crb, clusterID)
		// 添加原始对象引用
		k8sClusterRoleBinding.RawClusterRoleBinding = &crb
		// 计算 Age
		k8sClusterRoleBinding.Age = calculateAge(crb.CreationTimestamp.Time)
		k8sClusterRoleBindings = append(k8sClusterRoleBindings, &k8sClusterRoleBinding)
	}

	m.logger.Debug("成功获取ClusterRoleBinding列表",
		zap.Int("cluster_id", clusterID), zap.Int("count", len(clusterRoleBindings.Items)))

	return k8sClusterRoleBindings, nil
}

// GetClusterRoleBindingListRaw 获取ClusterRoleBinding原始列表
func (m *clusterRoleBindingManager) GetClusterRoleBindingListRaw(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取ClusterRoleBinding列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取ClusterRoleBinding列表失败: %w", err)
	}

	m.logger.Debug("成功获取ClusterRoleBinding列表",
		zap.Int("cluster_id", clusterID), zap.Int("count", len(clusterRoleBindings.Items)))

	return clusterRoleBindings, nil
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
func (m *clusterRoleBindingManager) UpdateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新ClusterRoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", clusterRoleBinding.Name))
		return fmt.Errorf("更新ClusterRoleBinding %s 失败: %w", clusterRoleBinding.Name, err)
	}

	m.logger.Info("成功更新ClusterRoleBinding",
		zap.Int("cluster_id", clusterID), zap.String("name", clusterRoleBinding.Name))
	return nil
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (m *clusterRoleBindingManager) DeleteClusterRoleBinding(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除ClusterRoleBinding失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("name", name))
		return fmt.Errorf("删除ClusterRoleBinding %s 失败: %w", name, err)
	}

	m.logger.Info("成功删除ClusterRoleBinding",
		zap.Int("cluster_id", clusterID), zap.String("name", name))
	return nil
}

// calculateAge 计算资源的年龄，返回可读的时间格式
func calculateAge(creationTime time.Time) string {
	duration := time.Since(creationTime)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		if days == 1 {
			return "1d"
		}
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		if hours == 1 {
			return "1h"
		}
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", minutes)
	} else {
		seconds := int(duration.Seconds())
		if seconds <= 1 {
			return "1s"
		}
		return fmt.Sprintf("%ds", seconds)
	}
}
