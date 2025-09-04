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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
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
	GetServiceMetrics(ctx context.Context, clusterID int, namespace, serviceName string) (*model.K8sServiceMetrics, error)
}

// serviceManager Service管理器实现
type serviceManager struct {
	client client.K8sClient
	logger *zap.Logger
}

// NewServiceManager 创建Service管理器
func NewServiceManager(client client.K8sClient, logger *zap.Logger) ServiceManager {
	return &serviceManager{
		client: client,
		logger: logger,
	}
}

// GetService 获取单个Service
func (m *serviceManager) GetService(ctx context.Context, clusterID int, namespace, name string) (*corev1.Service, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
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
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
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
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	createdService, err := clientset.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", service.Namespace), zap.String("name", service.Name))
		return nil, fmt.Errorf("创建Service %s/%s 失败: %w", service.Namespace, service.Name, err)
	}

	m.logger.Info("成功创建Service",
		zap.Int("cluster_id", clusterID), zap.String("namespace", createdService.Namespace), zap.String("name", createdService.Name))
	return createdService, nil
}

// UpdateService 更新Service
func (m *serviceManager) UpdateService(ctx context.Context, clusterID int, service *corev1.Service) (*corev1.Service, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	updatedService, err := clientset.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", service.Namespace), zap.String("name", service.Name))
		return nil, fmt.Errorf("更新Service %s/%s 失败: %w", service.Namespace, service.Name, err)
	}

	m.logger.Info("成功更新Service",
		zap.Int("cluster_id", clusterID), zap.String("namespace", updatedService.Namespace), zap.String("name", updatedService.Name))
	return updatedService, nil
}

// DeleteService 删除Service
func (m *serviceManager) DeleteService(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().Services(namespace).Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除Service %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除Service",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
	return nil
}

// BatchDeleteServices 批量删除Service
func (m *serviceManager) BatchDeleteServices(ctx context.Context, clusterID int, namespace string, serviceNames []string, options metav1.DeleteOptions) error {
	if len(serviceNames) == 0 {
		return nil
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var errors []error
	for _, serviceName := range serviceNames {
		if err := clientset.CoreV1().Services(namespace).Delete(ctx, serviceName, options); err != nil {
			m.logger.Error("批量删除Service失败", zap.Error(err),
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", serviceName))
			errors = append(errors, fmt.Errorf("删除Service %s/%s 失败: %w", namespace, serviceName, err))
		} else {
			m.logger.Info("成功删除Service",
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", serviceName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("批量删除Service时发生 %d 个错误: %v", len(errors), errors)
	}

	return nil
}

// GetServiceEndpoints 获取Service的Endpoints
func (m *serviceManager) GetServiceEndpoints(ctx context.Context, clusterID int, namespace, serviceName string) (*corev1.Endpoints, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		// 如果Endpoints不存在，返回空的Endpoints对象而不是错误
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
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
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

// GetServiceMetrics 获取Service指标
func (m *serviceManager) GetServiceMetrics(ctx context.Context, clusterID int, namespace, serviceName string) (*model.K8sServiceMetrics, error) {
	// 获取Service信息
	service, err := m.GetService(ctx, clusterID, namespace, serviceName)
	if err != nil {
		m.logger.Error("获取Service失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName))
		return nil, fmt.Errorf("获取Service失败: %w", err)
	}

	// 获取Kubernetes客户端
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取Metrics客户端
	metricsClient, err := m.client.GetMetricsClient(clusterID)
	if err != nil {
		m.logger.Warn("获取Metrics客户端失败，将返回基础指标", zap.Error(err), zap.Int("cluster_id", clusterID))
		// 如果无法获取Metrics客户端，返回基础指标
		return &model.K8sServiceMetrics{
			RequestCount:    0,
			RequestRate:     0.0,
			ResponseTime:    0.0,
			ErrorRate:       0.0,
			ConnectionCount: 0,
			BandwidthIn:     0.0,
			BandwidthOut:    0.0,
			LastUpdated:     time.Now(),
		}, nil
	}

	// 根据Service的Selector获取对应的Pod
	var podList *corev1.PodList
	if len(service.Spec.Selector) > 0 {
		labelSelector := labels.SelectorFromSet(service.Spec.Selector)
		podList, err = clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
		if err != nil {
			m.logger.Error("获取Service对应的Pod列表失败", zap.Error(err),
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName))
			return nil, fmt.Errorf("获取Service对应的Pod列表失败: %w", err)
		}
	} else {
		// 如果Service没有Selector（如headless service），返回基础指标
		m.logger.Info("Service没有Selector，返回基础指标",
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName))
		return &model.K8sServiceMetrics{
			RequestCount:    0,
			RequestRate:     0.0,
			ResponseTime:    0.0,
			ErrorRate:       0.0,
			ConnectionCount: 0,
			BandwidthIn:     0.0,
			BandwidthOut:    0.0,
			LastUpdated:     time.Now(),
		}, nil
	}

	// 获取Pod指标
	var podMetricsList *metricsv1beta1.PodMetricsList
	if len(podList.Items) > 0 {
		labelSelector := labels.SelectorFromSet(service.Spec.Selector)
		podMetricsList, err = metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
		if err != nil {
			m.logger.Error("获取Pod指标失败", zap.Error(err),
				zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName))
			// 指标获取失败，返回基础指标
			return &model.K8sServiceMetrics{
				RequestCount:    0,
				RequestRate:     0.0,
				ResponseTime:    0.0,
				ErrorRate:       0.0,
				ConnectionCount: int64(len(podList.Items)), // 至少可以显示Pod数量作为连接数
				BandwidthIn:     0.0,
				BandwidthOut:    0.0,
				LastUpdated:     time.Now(),
			}, nil
		}
	}

	// 计算Service指标
	metrics := m.calculateServiceMetrics(service, podList.Items, podMetricsList)

	m.logger.Debug("成功获取Service指标",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service", serviceName),
		zap.Int("pod_count", len(podList.Items)))

	return metrics, nil
}

// calculateServiceMetrics 计算Service指标
func (m *serviceManager) calculateServiceMetrics(service *corev1.Service, pods []corev1.Pod, podMetrics *metricsv1beta1.PodMetricsList) *model.K8sServiceMetrics {
	metrics := &model.K8sServiceMetrics{
		RequestCount:    0,
		RequestRate:     0.0,
		ResponseTime:    0.0,
		ErrorRate:       0.0,
		ConnectionCount: 0,
		BandwidthIn:     0.0,
		BandwidthOut:    0.0,
		LastUpdated:     time.Now(),
	}

	// 基于Pod数量估算连接数（每个运行的Pod可以处理连接）
	runningPods := 0
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods++
		}
	}
	metrics.ConnectionCount = int64(runningPods)

	// 如果有Pod指标，可以基于资源使用情况估算一些指标
	if podMetrics != nil && len(podMetrics.Items) > 0 {
		// 基于CPU使用量估算请求率（这是一个简化的估算）
		totalCPUMilliCores := int64(0)
		for _, podMetric := range podMetrics.Items {
			for _, container := range podMetric.Containers {
				if cpu, ok := container.Usage[corev1.ResourceCPU]; ok {
					totalCPUMilliCores += cpu.MilliValue()
				}
			}
		}

		// 简化估算：假设每100毫核CPU对应1个请求/秒
		if totalCPUMilliCores > 0 {
			metrics.RequestRate = float64(totalCPUMilliCores) / 100.0
			// 基于请求率估算请求总数（假设服务运行了1小时）
			metrics.RequestCount = int64(metrics.RequestRate * 3600)
		}

		// 基于Pod数量和负载估算响应时间（简化模型）
		if runningPods > 0 && totalCPUMilliCores > 0 {
			// 负载越高，响应时间越长（简化模型）
			avgCPUPerPod := float64(totalCPUMilliCores) / float64(runningPods)
			metrics.ResponseTime = avgCPUPerPod / 10.0 // 简化计算
			if metrics.ResponseTime < 10.0 {
				metrics.ResponseTime = 10.0 // 最低响应时间
			}
		} else {
			metrics.ResponseTime = 50.0 // 默认响应时间
		}

		// 基于Service类型估算带宽（简化模型）
		if service.Spec.Type == corev1.ServiceTypeLoadBalancer || service.Spec.Type == corev1.ServiceTypeNodePort {
			// 外部服务通常有更多流量
			metrics.BandwidthIn = float64(runningPods) * 0.5  // MB/s
			metrics.BandwidthOut = float64(runningPods) * 0.3 // MB/s
		} else {
			// 内部服务流量较少
			metrics.BandwidthIn = float64(runningPods) * 0.1
			metrics.BandwidthOut = float64(runningPods) * 0.1
		}

		// 基于Pod健康状态估算错误率
		totalPods := len(pods)
		if totalPods > 0 {
			unhealthyPods := totalPods - runningPods
			metrics.ErrorRate = float64(unhealthyPods) / float64(totalPods) * 100.0
		}
	} else {
		// 没有指标时的默认值
		metrics.ResponseTime = 50.0
		metrics.ErrorRate = 0.0
	}

	return metrics
}
