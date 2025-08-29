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

package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SvcService interface {
	// GetServicesByNamespace 获取指定命名空间的 Service 列表
	GetServicesByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Service, error)
	// GetServiceYaml 获取指定 Service 的 YAML 配置
	GetServiceYaml(ctx context.Context, id int, namespace, serviceName string) (*corev1.Service, error)
	// DeleteService 删除 Service
	DeleteService(ctx context.Context, id int, namespace string, serviceNames string) error
	// BatchDeleteService 批量删除 Service
	BatchDeleteService(ctx context.Context, id int, namespace string, serviceNames []string) error
	// UpdateService 更新 Service
	UpdateService(ctx context.Context, serviceResource *model.K8sServiceReq) error
	// CreateService 创建 Service
	//CreateService(ctx context.Context, serviceRequest *model.K8sServiceRequest) error
}

type svcService struct {
	dao            dao.ClusterDAO
	client         client.K8sClient       // 保持向后兼容
	serviceManager manager.ServiceManager // 新的依赖注入
	l              *zap.Logger
}

func NewSvcService(dao dao.ClusterDAO, client client.K8sClient, l *zap.Logger) SvcService {
	return &svcService{
		dao:            dao,
		client:         client,
		serviceManager: manager.NewServiceManager(client, l),
		l:              l,
	}
}

// GetServicesByNamespace 获取指定命名空间中的 Service 列表
func (s *svcService) GetServicesByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Service, error) {
	// 使用 ServiceManager 获取 Service 列表
	serviceList, err := s.serviceManager.ListServices(ctx, id, namespace)
	if err != nil {
		s.l.Error("获取 Service 列表失败", zap.Error(err))
		return nil, err
	}

	services := make([]*corev1.Service, len(serviceList.Items))
	for i := range serviceList.Items {
		services[i] = &serviceList.Items[i]
	}

	return services, nil
}

// GetServiceYaml 获取指定 Service 的 YAML 配置
func (s *svcService) GetServiceYaml(ctx context.Context, id int, namespace, serviceName string) (*corev1.Service, error) {
	// 使用 ServiceManager 获取 Service 详情
	service, err := s.serviceManager.GetService(ctx, id, namespace, serviceName)
	if err != nil {
		s.l.Error("获取 Service 失败", zap.Error(err))
		return nil, err
	}

	return service, nil
}

// UpdateService 更新指定的 Service
func (s *svcService) UpdateService(ctx context.Context, serviceResource *model.K8sServiceReq) error {
	kubeClient, err := s.client.GetKubeClient(serviceResource.ClusterId)
	if err != nil {
		s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 获取现有 Service
	service, err := kubeClient.CoreV1().Services(serviceResource.ServiceYaml.Namespace).Get(ctx, serviceResource.ServiceYaml.Name, metav1.GetOptions{})
	if err != nil {
		s.l.Error("获取 Service 失败", zap.Error(err))
		return err
	}

	service.Spec = serviceResource.ServiceYaml.Spec

	_, err = kubeClient.CoreV1().Services(serviceResource.ServiceYaml.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		s.l.Error("更新 Service 失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *svcService) DeleteService(ctx context.Context, id int, namespace string, serviceNames string) error {
	// 使用 ServiceManager 删除 Service
	return s.serviceManager.DeleteService(ctx, id, namespace, serviceNames, metav1.DeleteOptions{})
}

// BatchDeleteService 批量删除指定的 Service
func (s *svcService) BatchDeleteService(ctx context.Context, id int, namespace string, serviceNames []string) error {
	// 使用 ServiceManager 批量删除 Service
	return s.serviceManager.BatchDeleteServices(ctx, id, namespace, serviceNames, metav1.DeleteOptions{})
}
