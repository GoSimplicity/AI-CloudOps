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
	"strings"
	"sync"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressService interface {
	GetIngressesByNamespace(ctx context.Context, id int, namespace string) ([]*networkingv1.Ingress, error)
	CreateIngress(ctx context.Context, req *model.K8sIngressRequest) error
	UpdateIngress(ctx context.Context, req *model.K8sIngressRequest) error
	DeleteIngress(ctx context.Context, id int, namespace, ingressName string) error
	BatchDeleteIngress(ctx context.Context, id int, namespace string, ingressNames []string) error
	GetIngressYaml(ctx context.Context, id int, namespace, ingressName string) (string, error)
	GetIngressStatus(ctx context.Context, id int, namespace, ingressName string) (*model.K8sIngressStatus, error)
	GetIngressRules(ctx context.Context, id int, namespace, ingressName string) ([]networkingv1.IngressRule, error)
	GetIngressTLS(ctx context.Context, id int, namespace, ingressName string) ([]networkingv1.IngressTLS, error)
	GetIngressEndpoints(ctx context.Context, id int, namespace, ingressName string) (map[string]interface{}, error)
}

type ingressService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewIngressService 创建新的 IngressService 实例
func NewIngressService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) IngressService {
	return &ingressService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetIngressesByNamespace 获取指定命名空间下的所有 Ingress
func (i *ingressService) GetIngressesByNamespace(ctx context.Context, id int, namespace string) ([]*networkingv1.Ingress, error) {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	ingresses, err := kubeClient.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		i.logger.Error("获取 Ingress 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get Ingress list: %w", err)
	}

	result := make([]*networkingv1.Ingress, len(ingresses.Items))
	for idx := range ingresses.Items {
		result[idx] = &ingresses.Items[idx]
	}

	i.logger.Info("成功获取 Ingress 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreateIngress 创建 Ingress
func (i *ingressService) CreateIngress(ctx context.Context, req *model.K8sIngressRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.NetworkingV1().Ingresses(req.Namespace).Create(ctx, req.IngressYaml, metav1.CreateOptions{})
	if err != nil {
		i.logger.Error("创建 Ingress 失败", zap.Error(err), zap.String("ingress_name", req.IngressYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create Ingress: %w", err)
	}

	i.logger.Info("成功创建 Ingress", zap.String("ingress_name", req.IngressYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// UpdateIngress 更新 Ingress
func (i *ingressService) UpdateIngress(ctx context.Context, req *model.K8sIngressRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	existingIngress, err := kubeClient.NetworkingV1().Ingresses(req.Namespace).Get(ctx, req.IngressYaml.Name, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取现有 Ingress 失败", zap.Error(err), zap.String("ingress_name", req.IngressYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing Ingress: %w", err)
	}

	existingIngress.Spec = req.IngressYaml.Spec

	if _, err := kubeClient.NetworkingV1().Ingresses(req.Namespace).Update(ctx, existingIngress, metav1.UpdateOptions{}); err != nil {
		i.logger.Error("更新 Ingress 失败", zap.Error(err), zap.String("ingress_name", req.IngressYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update Ingress: %w", err)
	}

	i.logger.Info("成功更新 Ingress", zap.String("ingress_name", req.IngressYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetIngressYaml 获取指定 Ingress 的 YAML 定义
func (i *ingressService) GetIngressYaml(ctx context.Context, id int, namespace, ingressName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取 Ingress 失败", zap.Error(err), zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Ingress: %w", err)
	}

	yamlData, err := yaml.Marshal(ingress)
	if err != nil {
		i.logger.Error("序列化 Ingress YAML 失败", zap.Error(err), zap.String("ingress_name", ingressName))
		return "", fmt.Errorf("failed to serialize Ingress YAML: %w", err)
	}

	i.logger.Info("成功获取 Ingress YAML", zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeleteIngress 批量删除 Ingress
func (i *ingressService) BatchDeleteIngress(ctx context.Context, id int, namespace string, ingressNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(ingressNames))

	for _, name := range ingressNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				i.logger.Error("删除 Ingress 失败", zap.Error(err), zap.String("ingress_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete Ingress '%s': %w", name, err)
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
		i.logger.Error("批量删除 Ingress 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(ingressNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting Ingresses: %v", errs)
	}

	i.logger.Info("成功批量删除 Ingress", zap.Int("count", len(ingressNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeleteIngress 删除指定的 Ingress
func (i *ingressService) DeleteIngress(ctx context.Context, id int, namespace, ingressName string) error {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, ingressName, metav1.DeleteOptions{}); err != nil {
		i.logger.Error("删除 Ingress 失败", zap.Error(err), zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete Ingress '%s': %w", ingressName, err)
	}

	i.logger.Info("成功删除 Ingress", zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// GetIngressStatus 获取 Ingress 状态
func (i *ingressService) GetIngressStatus(ctx context.Context, id int, namespace, ingressName string) (*model.K8sIngressStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取 Ingress 失败", zap.Error(err), zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Ingress: %w", err)
	}

	// 提取主机和路径信息
	var hosts []string
	var paths []string

	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Path != "" {
					paths = append(paths, path.Path)
				}
			}
		}
	}

	status := &model.K8sIngressStatus{
		Name:              ingress.Name,
		Namespace:         ingress.Namespace,
		IngressClass:      ingress.Spec.IngressClassName,
		Rules:             ingress.Spec.Rules,
		TLS:               ingress.Spec.TLS,
		Hosts:             hosts,
		Paths:             paths,
		LoadBalancer:      ingress.Status.LoadBalancer,
		CreationTimestamp: ingress.CreationTimestamp.Time,
	}

	i.logger.Info("成功获取 Ingress 状态", zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Strings("hosts", hosts), zap.Int("rules_count", len(ingress.Spec.Rules)), zap.Int("cluster_id", id))
	return status, nil
}

// GetIngressRules 获取 Ingress 规则
func (i *ingressService) GetIngressRules(ctx context.Context, id int, namespace, ingressName string) ([]networkingv1.IngressRule, error) {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取 Ingress 失败", zap.Error(err), zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Ingress: %w", err)
	}

	i.logger.Info("成功获取 Ingress 规则", zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("rules_count", len(ingress.Spec.Rules)), zap.Int("cluster_id", id))
	return ingress.Spec.Rules, nil
}

// GetIngressTLS 获取 Ingress TLS 配置
func (i *ingressService) GetIngressTLS(ctx context.Context, id int, namespace, ingressName string) ([]networkingv1.IngressTLS, error) {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取 Ingress 失败", zap.Error(err), zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Ingress: %w", err)
	}

	i.logger.Info("成功获取 Ingress TLS 配置", zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("tls_count", len(ingress.Spec.TLS)), zap.Int("cluster_id", id))
	return ingress.Spec.TLS, nil
}

// GetIngressEndpoints 获取 Ingress 后端端点
func (i *ingressService) GetIngressEndpoints(ctx context.Context, id int, namespace, ingressName string) (map[string]interface{}, error) {
	kubeClient, err := pkg.GetKubeClient(id, i.client, i.logger)
	if err != nil {
		i.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取 Ingress 失败", zap.Error(err), zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Ingress: %w", err)
	}

	endpoints := map[string]interface{}{
		"ingress_name":     ingressName,
		"namespace":        namespace,
		"backend_services": make([]map[string]interface{}, 0),
		"load_balancer":    ingress.Status.LoadBalancer,
	}

	backendServices := make([]map[string]interface{}, 0)

	// 处理默认后端
	if ingress.Spec.DefaultBackend != nil && ingress.Spec.DefaultBackend.Service != nil {
		defaultBackend := map[string]interface{}{
			"type":         "default",
			"service_name": ingress.Spec.DefaultBackend.Service.Name,
			"service_port": ingress.Spec.DefaultBackend.Service.Port,
			"host":         "*",
			"path":         "/",
		}
		backendServices = append(backendServices, defaultBackend)
	}

	// 处理规则中的后端
	for _, rule := range ingress.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}

		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					backend := map[string]interface{}{
						"type":         "rule",
						"service_name": path.Backend.Service.Name,
						"service_port": path.Backend.Service.Port,
						"host":         host,
						"path":         path.Path,
						"path_type":    string(*path.PathType),
					}
					backendServices = append(backendServices, backend)

					// 尝试获取对应的 Service 和 Endpoints 信息
					serviceName := path.Backend.Service.Name
					service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
					if err == nil {
						backend["service_type"] = string(service.Spec.Type)
						backend["service_cluster_ip"] = service.Spec.ClusterIP

						// 获取 Endpoints 信息
						endpoint, err := kubeClient.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
						if err == nil {
							endpointAddresses := make([]string, 0)
							for _, subset := range endpoint.Subsets {
								for _, addr := range subset.Addresses {
									endpointAddresses = append(endpointAddresses, addr.IP)
								}
							}
							backend["endpoint_addresses"] = endpointAddresses
							backend["endpoint_count"] = len(endpointAddresses)
						}
					}
				}
			}
		}
	}

	endpoints["backend_services"] = backendServices

	// 统计信息
	uniqueServices := make(map[string]bool)
	var totalEndpoints int
	for _, backend := range backendServices {
		if serviceName, ok := backend["service_name"].(string); ok {
			uniqueServices[serviceName] = true
		}
		if count, ok := backend["endpoint_count"].(int); ok {
			totalEndpoints += count
		}
	}

	endpoints["summary"] = map[string]interface{}{
		"total_backend_services": len(backendServices),
		"unique_services":        len(uniqueServices),
		"total_endpoints":        totalEndpoints,
		"hosts": strings.Join(func() []string {
			var hosts []string
			for _, rule := range ingress.Spec.Rules {
				if rule.Host != "" {
					hosts = append(hosts, rule.Host)
				}
			}
			return hosts
		}(), ", "),
	}

	i.logger.Info("成功获取 Ingress 后端端点", zap.String("ingress_name", ingressName), zap.String("namespace", namespace), zap.Int("backend_services_count", len(backendServices)), zap.Int("unique_services", len(uniqueServices)), zap.Int("cluster_id", id))
	return endpoints, nil
}
