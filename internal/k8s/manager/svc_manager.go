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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServiceManager Service管理器接口
type ServiceManager interface {
	// 基础CRUD操作
	GetService(ctx context.Context, clusterID int, namespace, name string) (*corev1.Service, error)
	ListServices(ctx context.Context, clusterID int, namespace string) (*corev1.ServiceList, error)
	CreateService(ctx context.Context, clusterID int, service *corev1.Service) (*corev1.Service, error)
	UpdateService(ctx context.Context, clusterID int, service *corev1.Service) (*corev1.Service, error)
	DeleteService(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error

	// 批量操作
	BatchDeleteServices(ctx context.Context, clusterID int, namespace string, serviceNames []string, options metav1.DeleteOptions) error

	// 业务功能
	GetServiceEndpoints(ctx context.Context, clusterID int, namespace, serviceName string) (*corev1.Endpoints, error)
	ListServicesBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.ServiceList, error)
}

// serviceManager Service管理器实现
type serviceManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

// NewServiceManager 创建Service管理器
func NewServiceManager(clientFactory client.K8sClient, logger *zap.Logger) ServiceManager {
	return &serviceManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 私有方法：获取Kubernetes客户端
func (m *serviceManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

// GetService 获取单个Service
func (m *serviceManager) GetService(ctx context.Context, clusterID int, namespace, name string) (*corev1.Service, error) {
	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	service, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取Service %s/%s 失败: %w", namespace, name, err)
	}

	return service, nil
}

// ListServices 获取Service列表
func (m *serviceManager) ListServices(ctx context.Context, clusterID int, namespace string) (*corev1.ServiceList, error) {
	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取Service列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Service列表失败: %w", err)
	}

	return services, nil
}

// CreateService 创建Service
func (m *serviceManager) CreateService(ctx context.Context, clusterID int, service *corev1.Service) (*corev1.Service, error) {
	if service == nil {
		return nil, fmt.Errorf("service 不能为空")
	}

	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	targetNamespace := service.Namespace
	if targetNamespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}

	createdService, err := clientset.CoreV1().Services(targetNamespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", service.Namespace), zap.String("name", service.Name))
		return nil, fmt.Errorf("创建Service %s/%s 失败: %w", targetNamespace, service.Name, err)
	}

	m.logger.Info("成功创建Service",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", targetNamespace),
		zap.String("name", createdService.Name))
	return createdService, nil
}

// UpdateService 更新Service
func (m *serviceManager) UpdateService(ctx context.Context, clusterID int, service *corev1.Service) (*corev1.Service, error) {
	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	updatedService, err := clientset.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", service.Namespace), zap.String("name", service.Name))
		return nil, fmt.Errorf("更新Service %s/%s 失败: %w", service.Namespace, service.Name, err)
	}

	m.logger.Info("成功更新Service",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", updatedService.Namespace),
		zap.String("name", updatedService.Name))
	return updatedService, nil
}

// DeleteService 删除Service
func (m *serviceManager) DeleteService(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error {
	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = clientset.CoreV1().Services(namespace).Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除Service %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除Service",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// BatchDeleteServices 批量删除Service
func (m *serviceManager) BatchDeleteServices(ctx context.Context, clusterID int, namespace string, serviceNames []string, options metav1.DeleteOptions) error {
	if len(serviceNames) == 0 {
		return nil
	}

	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var errors []error
	for _, serviceName := range serviceNames {
		if err := clientset.CoreV1().Services(namespace).Delete(ctx, serviceName, options); err != nil {
			m.logger.Error("批量删除Service失败", zap.Error(err),
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", serviceName))
			errors = append(errors, fmt.Errorf("删除Service %s/%s 失败: %w", namespace, serviceName, err))
		} else {
			m.logger.Info("成功删除Service",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", serviceName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("批量删除Service时发生 %d 个错误: %v", len(errors), errors)
	}

	return nil
}

// GetServiceEndpoints 获取Service的Endpoints
func (m *serviceManager) GetServiceEndpoints(ctx context.Context, clusterID int, namespace, serviceName string) (*corev1.Endpoints, error) {
	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			m.logger.Info("Service Endpoints不存在，返回空Endpoints",
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName))
			return &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: namespace,
				},
				Subsets: []corev1.EndpointSubset{},
			}, nil
		}

		m.logger.Error("获取Service Endpoints失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName))
		return nil, fmt.Errorf("获取Service %s/%s Endpoints失败: %w", namespace, serviceName, err)
	}

	return endpoints, nil
}

// ListServicesBySelector 根据选择器获取Service列表
func (m *serviceManager) ListServicesBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.ServiceList, error) {
	clientset, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	services, err := clientset.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("根据选择器获取Service列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("selector", selector))
		return nil, fmt.Errorf("根据选择器获取Service列表失败: %w", err)
	}

	return services, nil
}
