package admin

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type SvcService interface {
	// GetServicesByNamespace 获取指定命名空间的 Service 列表
	GetServicesByNamespace(ctx context.Context, clusterName, namespace string) ([]*corev1.Service, error)
	// GetServiceYaml 获取指定 Service 的 YAML 配置
	GetServiceYaml(ctx context.Context, clusterName, namespace, serviceName string) (*corev1.Service, error)
	// CreateOrUpdateService 创建或更新 Service
	CreateOrUpdateService(ctx context.Context, service *model.K8sServiceRequest) error
	// UpdateService 更新指定 Name Service
	UpdateService(ctx context.Context, service *model.K8sServiceRequest) error
	// DeleteService 删除 Service
	DeleteService(ctx context.Context, clusterName, namespace string, serviceName []string) error
}

type svcService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

// NewSvcService 创建新的 SvcService 实例
func NewSvcService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) SvcService {
	return &svcService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// GetServicesByNamespace 获取指定命名空间中的 Service 列表
func (s *svcService) GetServicesByNamespace(ctx context.Context, clusterName, namespace string) ([]*corev1.Service, error) {
	//kubeClient, err := pkg.GetKubeClient(ctx, clusterName, s.dao, s.client, s.l)
	//if err != nil {
	//	s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return nil, err
	//}
	//
	//serviceList, err := kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	//if err != nil {
	//	s.l.Error("获取 Service 列表失败", zap.Error(err))
	//	return nil, err
	//}
	//
	//// 将 *v1.ServiceList 转换为 []*corev1.Service
	//services := make([]*corev1.Service, len(serviceList.Items))
	//for i, svc := range serviceList.Items {
	//	services[i] = &svc
	//}
	//
	//return services, nil
	return nil, nil
}

// GetServiceYaml 获取指定 Service 的 YAML 配置
func (s *svcService) GetServiceYaml(ctx context.Context, clusterName, namespace, serviceName string) (*corev1.Service, error) {
	//kubeClient, err := pkg.GetKubeClient(ctx, clusterName, s.dao, s.client, s.l)
	//if err != nil {
	//	s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return nil, err
	//}
	//
	//// 获取 Service 对象
	//service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	//if err != nil {
	//	s.l.Error("获取 Service 失败", zap.Error(err))
	//	return nil, err
	//}
	//
	//return service, nil
	return nil, nil
}

// CreateOrUpdateService 创建或更新指定 Service
func (s *svcService) CreateOrUpdateService(ctx context.Context, serviceResource *model.K8sServiceRequest) error {
	//kubeClient, err := pkg.GetKubeClient(ctx, serviceResource.ClusterName, s.dao, s.client, s.l)
	//if err != nil {
	//	s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return err
	//}
	//
	//// 检查 Service 是否已存在
	//_, err = kubeClient.CoreV1().Services(serviceResource.Service.Namespace).Get(ctx, serviceResource.Service.Name, metav1.GetOptions{})
	//if err != nil {
	//	if k8sErr.IsNotFound(err) {
	//		// Service 不存在，创建新 Service
	//		_, err = kubeClient.CoreV1().Services(serviceResource.Service.Namespace).Create(ctx, serviceResource.Service, metav1.CreateOptions{})
	//		if err != nil {
	//			s.l.Error("创建 Service 失败", zap.Error(err))
	//			return err
	//		}
	//		s.l.Info("创建 Service 成功", zap.String("serviceName", serviceResource.Service.Name))
	//		return nil
	//	}
	//	// 其他错误
	//	s.l.Error("获取 Service 失败", zap.Error(err))
	//	return err
	//}
	//
	//// Service 已存在，更新现有 Service
	//return s.UpdateService(ctx, serviceResource)
	return nil
}

// UpdateService 更新指定的 Service
func (s *svcService) UpdateService(ctx context.Context, serviceResource *model.K8sServiceRequest) error {
	//kubeClient, err := pkg.GetKubeClient(ctx, serviceResource.ClusterName, s.dao, s.client, s.l)
	//if err != nil {
	//	s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return err
	//}
	//
	//// 获取现有 Service
	//service, err := kubeClient.CoreV1().Services(serviceResource.Service.Namespace).Get(ctx, serviceResource.Service.Name, metav1.GetOptions{})
	//if err != nil {
	//	s.l.Error("获取 Service 失败", zap.Error(err))
	//	return err
	//}
	//
	//// 更新 Service Spec
	//service.Spec = serviceResource.Service.Spec
	//_, err = kubeClient.CoreV1().Services(serviceResource.Service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	//if err != nil {
	//	s.l.Error("更新 Service 失败", zap.Error(err))
	//	return err
	//}
	//
	//s.l.Info("更新 Service 成功", zap.String("serviceName", serviceResource.Service.Name))
	//return nil
	return nil
}

// DeleteService 删除指定的 Service
func (s *svcService) DeleteService(ctx context.Context, clusterName, namespace string, serviceNames []string) error {
	//kubeClient, err := pkg.GetKubeClient(ctx, clusterName, s.dao, s.client, s.l)
	//if err != nil {
	//	s.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return err
	//}
	//
	//var errs []error
	//// 批量删除服务
	//for _, name := range serviceNames {
	//	err = kubeClient.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	//	if err != nil {
	//		s.l.Error("删除 Service 失败", zap.String("serviceName", name), zap.Error(err))
	//		errs = append(errs, err)
	//		continue
	//	}
	//
	//	s.l.Info("删除 Service 成功", zap.String("serviceName", name))
	//}
	//
	//// 如果存在错误，返回所有错误
	//if len(errs) > 0 {
	//	return fmt.Errorf("在删除 Service 时遇到以下错误: %v", errs)
	//}
	//
	//return nil
	return nil
}
