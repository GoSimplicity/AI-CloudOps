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

package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func BuildK8sServiceFromCore(clusterID int, service corev1.Service) *model.K8sService {
	status := getServiceStatus(service)
	age := getServiceAge(service)
	ports := buildServicePorts(service.Spec.Ports)

	externalIPs := service.Spec.ExternalIPs
	if externalIPs == nil {
		externalIPs = []string{}
	}

	selector := service.Spec.Selector
	if selector == nil {
		selector = map[string]string{}
	}

	labels := service.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	annotations := service.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	return &model.K8sService{
		Name:           service.Name,
		Namespace:      service.Namespace,
		ClusterID:      clusterID,
		UID:            string(service.UID),
		Type:           string(service.Spec.Type),
		ClusterIP:      service.Spec.ClusterIP,
		ExternalIPs:    externalIPs,
		LoadBalancerIP: service.Spec.LoadBalancerIP,
		Ports:          ports,
		Selector:       selector,
		Labels:         labels,
		Annotations:    annotations,
		CreatedAt:      service.CreationTimestamp.Time,
		Age:            age,
		Status:         status,
		Endpoints:      []model.K8sServiceEndpoint{},
	}
}

func BuildServiceFromRequest(req *model.CreateServiceReq) (*corev1.Service, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML != "" {
		return YAMLToService(req.YAML)
	}

	labels := req.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	annotations := req.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(req.Type),
			Selector: req.Selector,
			Ports:    ConvertToCorePorts(req.Ports),
		},
	}

	return service, nil
}

func ValidateService(service *corev1.Service) error {
	if service == nil {
		return fmt.Errorf("Service对象不能为空")
	}

	if service.Name == "" {
		return fmt.Errorf("Service名称不能为空")
	}

	if service.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if len(service.Spec.Ports) == 0 {
		return fmt.Errorf("Service端口配置不能为空")
	}

	for _, port := range service.Spec.Ports {
		if port.Port <= 0 {
			return fmt.Errorf("Service端口必须大于0")
		}
		if port.Protocol != corev1.ProtocolTCP && port.Protocol != corev1.ProtocolUDP && port.Protocol != corev1.ProtocolSCTP {
			return fmt.Errorf("无效的协议类型: %s", port.Protocol)
		}
	}

	switch service.Spec.Type {
	case corev1.ServiceTypeClusterIP, corev1.ServiceTypeNodePort, corev1.ServiceTypeLoadBalancer, corev1.ServiceTypeExternalName:
		// 有效类型
	default:
		return fmt.Errorf("无效的Service类型: %s", service.Spec.Type)
	}

	return nil
}

// YAMLToService 将YAML转换为Service对象
func YAMLToService(yamlContent string) (*corev1.Service, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	var service corev1.Service
	err := yaml.Unmarshal([]byte(yamlContent), &service)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &service, nil
}

// ServiceToYAML 将Service对象转换为YAML
func ServiceToYAML(service *corev1.Service) (string, error) {
	if service == nil {
		return "", fmt.Errorf("Service对象不能为空")
	}

	cleanService := service.DeepCopy()
	cleanService.Status = corev1.ServiceStatus{}
	cleanService.ManagedFields = nil
	cleanService.ResourceVersion = ""
	cleanService.UID = ""
	cleanService.CreationTimestamp = metav1.Time{}
	cleanService.Generation = 0
	cleanService.SelfLink = ""

	yamlBytes, err := yaml.Marshal(cleanService)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(yamlBytes), nil
}

func BuildServiceListOptions(req *model.GetServiceListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	if len(req.Labels) > 0 {
		var labelSelector []string
		for key, value := range req.Labels {
			labelSelector = append(labelSelector, fmt.Sprintf("%s=%s", key, value))
		}
		options.LabelSelector = strings.Join(labelSelector, ",")
	}

	return options
}

// FilterServicesByType 根据Service类型过滤
func FilterServicesByType(services []corev1.Service, serviceType string) []corev1.Service {
	if serviceType == "" {
		return services
	}

	var filtered []corev1.Service
	for _, service := range services {
		if string(service.Spec.Type) == serviceType {
			filtered = append(filtered, service)
		}
	}

	return filtered
}

func BuildServiceListPagination(services []corev1.Service, page int, size int) ([]corev1.Service, int64) {
	total := int64(len(services))
	if total == 0 {
		return []corev1.Service{}, 0
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := int64(page-1) * int64(size)
	end := start + int64(size)

	if start >= total {
		return []corev1.Service{}, total
	}

	if end > total {
		end = total
	}

	return services[start:end], total
}

// getServiceStatus 获取Service状态
func getServiceStatus(service corev1.Service) model.K8sSvcStatus {
	// 根据Service类型和状态判断
	switch service.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			return model.K8sSvcStatusRunning
		}
		return model.K8sSvcStatusStopped
	case corev1.ServiceTypeExternalName:
		if service.Spec.ExternalName != "" {
			return model.K8sSvcStatusRunning
		}
		return model.K8sSvcStatusError
	default:
		// ClusterIP 和 NodePort 类型通常都是运行状态
		if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" {
			return model.K8sSvcStatusRunning
		}
		return model.K8sSvcStatusStopped
	}
}

// getServiceAge 获取Service年龄
func getServiceAge(service corev1.Service) string {
	age := time.Since(service.CreationTimestamp.Time)
	days := int(age.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(age.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(age.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

// buildServicePorts 构建Service端口配置
func buildServicePorts(ports []corev1.ServicePort) []model.ServicePort {
	servicePorts := make([]model.ServicePort, 0, len(ports))
	for _, port := range ports {
		servicePort := model.ServicePort{
			Name:        port.Name,
			Protocol:    port.Protocol,
			Port:        port.Port,
			TargetPort:  port.TargetPort,
			NodePort:    port.NodePort,
			AppProtocol: port.AppProtocol,
		}
		servicePorts = append(servicePorts, servicePort)
	}
	return servicePorts
}

func ConvertToCorePorts(ports []model.ServicePort) []corev1.ServicePort {
	var corePorts []corev1.ServicePort
	for _, port := range ports {
		corePort := corev1.ServicePort{
			Name:        port.Name,
			Protocol:    port.Protocol,
			Port:        port.Port,
			TargetPort:  port.TargetPort,
			NodePort:    port.NodePort,
			AppProtocol: port.AppProtocol,
		}
		corePorts = append(corePorts, corePort)
	}
	return corePorts
}

func ConvertEndpointsToModel(endpoints *corev1.Endpoints) []model.K8sServiceEndpoint {
	if endpoints == nil {
		return []model.K8sServiceEndpoint{}
	}

	serviceEndpoints := make([]model.K8sServiceEndpoint, 0)
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				endpoint := model.K8sServiceEndpoint{
					IP:       address.IP,
					Port:     port.Port,
					Protocol: string(port.Protocol),
					Ready:    true,
				}
				serviceEndpoints = append(serviceEndpoints, endpoint)
			}
		}

		for _, address := range subset.NotReadyAddresses {
			for _, port := range subset.Ports {
				endpoint := model.K8sServiceEndpoint{
					IP:       address.IP,
					Port:     port.Port,
					Protocol: string(port.Protocol),
					Ready:    false,
				}
				serviceEndpoints = append(serviceEndpoints, endpoint)
			}
		}
	}

	return serviceEndpoints
}

// getServiceEndpoints 获取Service端点
func getServiceEndpoints(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, serviceName string) ([]model.K8sServiceEndpoint, error) {
	endpoints, err := kubeClient.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return []model.K8sServiceEndpoint{}, nil
		}
		return nil, err
	}

	return ConvertEndpointsToModel(endpoints), nil
}
