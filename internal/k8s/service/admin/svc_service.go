package admin

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

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SvcService interface {
	// GetServicesByNamespace 获取指定命名空间的 Service 列表
	GetServicesByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Service, error)
	// GetServiceYaml 获取指定 Service 的 YAML 配置
	GetServiceYaml(ctx context.Context, id int, namespace, serviceName string) (*corev1.Service, error)
	// CreateOrUpdateService 创建或更新 Service
	CreateOrUpdateService(ctx context.Context, service *model.K8sServiceRequest) error
	// DeleteService 删除 Service
	DeleteService(ctx context.Context, id int, namespace string, serviceNames []string) error
}

type svcService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewSvcService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) SvcService {
	return &svcService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// GetServicesByNamespace 获取指定命名空间中的 Service 列表
func (s *svcService) GetServicesByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Service, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.l)
	if err != nil {
		s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	serviceList, err := kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
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
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.l)
	if err != nil {
		s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		s.l.Error("获取 Service 失败", zap.Error(err))
		return nil, err
	}

	return service, nil
}

// CreateOrUpdateService 创建或更新指定 Service
func (s *svcService) CreateOrUpdateService(ctx context.Context, serviceResource *model.K8sServiceRequest) error {
	kubeClient, err := pkg.GetKubeClient(serviceResource.ClusterId, s.client, s.l)
	if err != nil {
		s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 检查 Service 是否已存在
	_, err = kubeClient.CoreV1().Services(serviceResource.ServiceYaml.Namespace).Get(ctx, serviceResource.ServiceYaml.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			// Service 不存在，创建新 Service
			_, err = kubeClient.CoreV1().Services(serviceResource.ServiceYaml.Namespace).Create(ctx, serviceResource.ServiceYaml, metav1.CreateOptions{})
			if err != nil {
				s.l.Error("创建 Service 失败", zap.Error(err))
				return err
			}
			s.l.Info("创建 Service 成功", zap.String("serviceName", serviceResource.ServiceYaml.Name))
			return nil
		}
		s.l.Error("获取 Service 失败", zap.Error(err))
		return err
	}

	return s.updateService(ctx, serviceResource)
}

// updateService 更新指定的 Service
func (s *svcService) updateService(ctx context.Context, serviceResource *model.K8sServiceRequest) error {
	kubeClient, err := pkg.GetKubeClient(serviceResource.ClusterId, s.client, s.l)
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

// DeleteService 删除指定的 Service
func (s *svcService) DeleteService(ctx context.Context, id int, namespace string, serviceNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.l)
	if err != nil {
		s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, name := range serviceNames {
		name := name // 避免闭包中使用相同的 name
		g.Go(func() error {
			err := kubeClient.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			if err != nil {
				s.l.Error("删除 Service 失败", zap.String("serviceName", name), zap.Error(err))
				return err
			}
			s.l.Info("删除 Service 成功", zap.String("serviceName", name))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("在删除 Service 时遇到错误: %v", err)
	}

	return nil
}
