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

package admin

import (
	"context"
	"fmt"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EndpointService interface {
	GetEndpointsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Endpoints, error)
	CreateEndpoint(ctx context.Context, req *model.K8sEndpointRequest) error
	DeleteEndpoint(ctx context.Context, id int, namespace, endpointName string) error
	BatchDeleteEndpoint(ctx context.Context, id int, namespace string, endpointNames []string) error
	GetEndpointYaml(ctx context.Context, id int, namespace, endpointName string) (string, error)
	GetEndpointStatus(ctx context.Context, id int, namespace, endpointName string) (*model.K8sEndpointStatus, error)
	CheckEndpointHealth(ctx context.Context, id int, namespace, endpointName string) (map[string]interface{}, error)
	GetEndpointService(ctx context.Context, id int, namespace, endpointName string) (*corev1.Service, error)
}

type endpointService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewEndpointService 创建新的 EndpointService 实例
func NewEndpointService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) EndpointService {
	return &endpointService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetEndpointsByNamespace 获取指定命名空间下的所有 Endpoint
func (e *endpointService) GetEndpointsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.Endpoints, error) {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	endpoints, err := kubeClient.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		e.logger.Error("获取 Endpoint 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get Endpoint list: %w", err)
	}

	result := make([]*corev1.Endpoints, len(endpoints.Items))
	for i := range endpoints.Items {
		result[i] = &endpoints.Items[i]
	}

	e.logger.Info("成功获取 Endpoint 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreateEndpoint 创建 Endpoint
func (e *endpointService) CreateEndpoint(ctx context.Context, req *model.K8sEndpointRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.CoreV1().Endpoints(req.Namespace).Create(ctx, req.EndpointYaml, metav1.CreateOptions{})
	if err != nil {
		e.logger.Error("创建 Endpoint 失败", zap.Error(err), zap.String("endpoint_name", req.EndpointYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create Endpoint: %w", err)
	}

	e.logger.Info("成功创建 Endpoint", zap.String("endpoint_name", req.EndpointYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetEndpointYaml 获取指定 Endpoint 的 YAML 定义
func (e *endpointService) GetEndpointYaml(ctx context.Context, id int, namespace, endpointName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	endpoint, err := kubeClient.CoreV1().Endpoints(namespace).Get(ctx, endpointName, metav1.GetOptions{})
	if err != nil {
		e.logger.Error("获取 Endpoint 失败", zap.Error(err), zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Endpoint: %w", err)
	}

	yamlData, err := yaml.Marshal(endpoint)
	if err != nil {
		e.logger.Error("序列化 Endpoint YAML 失败", zap.Error(err), zap.String("endpoint_name", endpointName))
		return "", fmt.Errorf("failed to serialize Endpoint YAML: %w", err)
	}

	e.logger.Info("成功获取 Endpoint YAML", zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeleteEndpoint 批量删除 Endpoint
func (e *endpointService) BatchDeleteEndpoint(ctx context.Context, id int, namespace string, endpointNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(endpointNames))

	for _, name := range endpointNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().Endpoints(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				e.logger.Error("删除 Endpoint 失败", zap.Error(err), zap.String("endpoint_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete Endpoint '%s': %w", name, err)
			}
		}(name)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		e.logger.Error("批量删除 Endpoint 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(endpointNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting Endpoints: %v", errs)
	}

	e.logger.Info("成功批量删除 Endpoint", zap.Int("count", len(endpointNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeleteEndpoint 删除指定的 Endpoint
func (e *endpointService) DeleteEndpoint(ctx context.Context, id int, namespace, endpointName string) error {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.CoreV1().Endpoints(namespace).Delete(ctx, endpointName, metav1.DeleteOptions{}); err != nil {
		e.logger.Error("删除 Endpoint 失败", zap.Error(err), zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete Endpoint '%s': %w", endpointName, err)
	}

	e.logger.Info("成功删除 Endpoint", zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// GetEndpointStatus 获取 Endpoint 状态
func (e *endpointService) GetEndpointStatus(ctx context.Context, id int, namespace, endpointName string) (*model.K8sEndpointStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	endpoint, err := kubeClient.CoreV1().Endpoints(namespace).Get(ctx, endpointName, metav1.GetOptions{})
	if err != nil {
		e.logger.Error("获取 Endpoint 失败", zap.Error(err), zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Endpoint: %w", err)
	}

	// 提取地址信息
	var addresses []string
	var ports []corev1.EndpointPort
	healthyCount := 0
	unhealthyCount := 0

	for _, subset := range endpoint.Subsets {
		// 收集健康地址
		for _, addr := range subset.Addresses {
			addresses = append(addresses, addr.IP)
			healthyCount++
		}
		// 收集不健康地址
		for _, addr := range subset.NotReadyAddresses {
			addresses = append(addresses, addr.IP)
			unhealthyCount++
		}
		// 收集端口信息
		ports = append(ports, subset.Ports...)
	}

	// 尝试找到关联的 Service
	serviceName := endpointName // 通常 Endpoint 名称与 Service 名称相同

	status := &model.K8sEndpointStatus{
		Name:               endpoint.Name,
		Namespace:          endpoint.Namespace,
		Subsets:            endpoint.Subsets,
		Addresses:          addresses,
		Ports:              ports,
		ServiceName:        serviceName,
		HealthyEndpoints:   healthyCount,
		UnhealthyEndpoints: unhealthyCount,
		CreationTimestamp:  endpoint.CreationTimestamp.Time,
	}

	e.logger.Info("成功获取 Endpoint 状态", zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("healthy_count", healthyCount), zap.Int("unhealthy_count", unhealthyCount), zap.Int("cluster_id", id))
	return status, nil
}

// CheckEndpointHealth 检查 Endpoint 健康状态
func (e *endpointService) CheckEndpointHealth(ctx context.Context, id int, namespace, endpointName string) (map[string]interface{}, error) {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	endpoint, err := kubeClient.CoreV1().Endpoints(namespace).Get(ctx, endpointName, metav1.GetOptions{})
	if err != nil {
		e.logger.Error("获取 Endpoint 失败", zap.Error(err), zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Endpoint: %w", err)
	}

	healthStatus := map[string]interface{}{
		"name":      endpoint.Name,
		"namespace": endpoint.Namespace,
		"healthy":   true,
		"details":   make(map[string]interface{}),
	}

	totalHealthy := 0
	totalUnhealthy := 0
	subsetDetails := make([]map[string]interface{}, 0)

	for i, subset := range endpoint.Subsets {
		subsetInfo := map[string]interface{}{
			"subset_index":     i,
			"healthy_count":    len(subset.Addresses),
			"unhealthy_count":  len(subset.NotReadyAddresses),
			"ports":           subset.Ports,
			"healthy_addresses": make([]string, 0),
			"unhealthy_addresses": make([]string, 0),
		}

		for _, addr := range subset.Addresses {
			subsetInfo["healthy_addresses"] = append(subsetInfo["healthy_addresses"].([]string), addr.IP)
			totalHealthy++
		}

		for _, addr := range subset.NotReadyAddresses {
			subsetInfo["unhealthy_addresses"] = append(subsetInfo["unhealthy_addresses"].([]string), addr.IP)
			totalUnhealthy++
		}

		subsetDetails = append(subsetDetails, subsetInfo)
	}

	healthStatus["details"] = map[string]interface{}{
		"total_healthy_endpoints":   totalHealthy,
		"total_unhealthy_endpoints": totalUnhealthy,
		"total_endpoints":          totalHealthy + totalUnhealthy,
		"subsets":                  subsetDetails,
	}

	// 如果有不健康的端点，标记为不健康
	if totalUnhealthy > 0 {
		healthStatus["healthy"] = false
	}

	// 如果没有任何端点，也标记为不健康
	if totalHealthy == 0 && totalUnhealthy == 0 {
		healthStatus["healthy"] = false
	}

	e.logger.Info("成功检查 Endpoint 健康状态", zap.String("endpoint_name", endpointName), zap.String("namespace", namespace), zap.Int("healthy_count", totalHealthy), zap.Int("unhealthy_count", totalUnhealthy), zap.Bool("overall_healthy", healthStatus["healthy"].(bool)), zap.Int("cluster_id", id))
	return healthStatus, nil
}

// GetEndpointService 获取 Endpoint 关联的 Service
func (e *endpointService) GetEndpointService(ctx context.Context, id int, namespace, endpointName string) (*corev1.Service, error) {
	kubeClient, err := pkg.GetKubeClient(id, e.client, e.logger)
	if err != nil {
		e.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 通常 Endpoint 名称与其关联的 Service 名称相同
	service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, endpointName, metav1.GetOptions{})
	if err != nil {
		e.logger.Error("获取关联的 Service 失败", zap.Error(err), zap.String("service_name", endpointName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get associated Service '%s': %w", endpointName, err)
	}

	e.logger.Info("成功获取 Endpoint 关联的 Service", zap.String("endpoint_name", endpointName), zap.String("service_name", service.Name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return service, nil
}