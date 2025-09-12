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
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// ConvertToK8sIngress 将 Kubernetes Ingress 转换为内部 Ingress 模型
func ConvertToK8sIngress(ingress *networkingv1.Ingress, clusterID int) *model.K8sIngress {
	if ingress == nil {
		return nil
	}
	// 提取主机列表
	hosts := make([]string, 0)
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}

	// 计算年龄
	age := pkg.GetAge(ingress.CreationTimestamp.Time)

	// 确定状态
	status := IngressStatus(ingress)

	// 获取Ingress类名
	ingressClassName := ""
	if ingress.Spec.IngressClassName != nil {
		ingressClassName = *ingress.Spec.IngressClassName
	}

	// 转换规则（简化处理）
	rules := make([]model.IngressRule, 0, len(ingress.Spec.Rules))
	for _, rule := range ingress.Spec.Rules {
		ingressRule := model.IngressRule{
			Host: rule.Host,
		}

		if rule.HTTP != nil {
			paths := make([]model.IngressHTTPIngressPath, 0, len(rule.HTTP.Paths))
			for _, path := range rule.HTTP.Paths {
				ingressPath := model.IngressHTTPIngressPath{
					Path:    path.Path,
					Backend: path.Backend,
				}
				if path.PathType != nil {
					ingressPath.PathType = path.PathType
				}
				paths = append(paths, ingressPath)
			}
			ingressRule.HTTP = model.IngressHTTPRuleValue{
				Paths: paths,
			}
		}

		rules = append(rules, ingressRule)
	}

	// 转换TLS配置（简化处理）
	tls := make([]model.IngressTLS, 0, len(ingress.Spec.TLS))
	for _, tlsConfig := range ingress.Spec.TLS {
		ingressTLS := model.IngressTLS{
			Hosts:      tlsConfig.Hosts,
			SecretName: tlsConfig.SecretName,
		}
		tls = append(tls, ingressTLS)
	}

	// 负载均衡器信息（简化处理）
	loadBalancer := model.IngressLoadBalancer{}

	return &model.K8sIngress{
		Name:             ingress.Name,
		Namespace:        ingress.Namespace,
		ClusterID:        clusterID,
		UID:              string(ingress.UID),
		IngressClassName: &ingressClassName,
		Rules:            rules,
		TLS:              tls,
		LoadBalancer:     loadBalancer,
		Labels:           ingress.Labels,
		Annotations:      ingress.Annotations,
		CreatedAt:        ingress.CreationTimestamp.Time,
		Age:              age,
		Status:           convertIngressStatusToEnum(status),
		Hosts:            hosts,
		RawIngress:       ingress,
	}
}

// IngressStatus 获取Ingress状态
func IngressStatus(item *networkingv1.Ingress) string {
	if item == nil {
		return StatusUnknown
	}
	// 如果正在删除
	if item.DeletionTimestamp != nil {
		return StatusTerminating
	}
	lb := item.Status.LoadBalancer
	ingressList := lb.Ingress
	if len(ingressList) == 0 {
		return StatusPending
	}
	for _, entry := range ingressList {
		if entry.IP != "" || entry.Hostname != "" {
			return StatusReady
		}
	}
	return StatusUnknown
}

// ValidateIngress 验证Ingress配置
func ValidateIngress(ingress *networkingv1.Ingress) error {
	if ingress == nil {
		return fmt.Errorf("ingress不能为空")
	}

	if ingress.Name == "" {
		return fmt.Errorf("ingress名称不能为空")
	}

	if ingress.Namespace == "" {
		return fmt.Errorf("namespace不能为空")
	}

	// 验证规则
	for i, rule := range ingress.Spec.Rules {
		if rule.HTTP != nil {
			for j, path := range rule.HTTP.Paths {
				if path.Backend.Service == nil {
					return fmt.Errorf("规则%d路径%d的后端服务不能为空", i, j)
				}
				if path.Backend.Service.Name == "" {
					return fmt.Errorf("规则%d路径%d的后端服务名称不能为空", i, j)
				}
			}
		}
	}

	return nil
}

// IngressToYAML 将Ingress转换为YAML
func IngressToYAML(ingress *networkingv1.Ingress) (string, error) {
	if ingress == nil {
		return "", fmt.Errorf("ingress不能为空")
	}

	// 清理不需要的字段
	cleanIngress := ingress.DeepCopy()
	cleanIngress.Status = networkingv1.IngressStatus{}
	cleanIngress.ManagedFields = nil
	cleanIngress.ResourceVersion = ""
	cleanIngress.UID = ""
	cleanIngress.CreationTimestamp = metav1.Time{}
	cleanIngress.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanIngress)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(yamlBytes), nil
}

// YAMLToIngress 将YAML转换为Ingress
func YAMLToIngress(yamlContent string) (*networkingv1.Ingress, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	var ingress networkingv1.Ingress
	err := yaml.Unmarshal([]byte(yamlContent), &ingress)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &ingress, nil
}

// IsIngressReady 判断Ingress是否就绪
func IsIngressReady(ingress networkingv1.Ingress) bool {
	return IngressStatus(&ingress) == StatusReady
}

// GetIngressAge 获取Ingress年龄
func GetIngressAge(ingress networkingv1.Ingress) string {
	age := time.Since(ingress.CreationTimestamp.Time)
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

// BuildIngressFromSpec 从CreateIngressReq构建Ingress对象
func BuildIngressFromSpec(req *model.CreateIngressReq) (*networkingv1.Ingress, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: req.IngressClassName,
		},
	}

	// 构建规则
	if len(req.Rules) > 0 {
		ingress.Spec.Rules = make([]networkingv1.IngressRule, 0, len(req.Rules))
		for _, rule := range req.Rules {
			ingressRule := networkingv1.IngressRule{
				Host: rule.Host,
			}

			if len(rule.HTTP.Paths) > 0 {
				httpRule := &networkingv1.HTTPIngressRuleValue{
					Paths: make([]networkingv1.HTTPIngressPath, 0, len(rule.HTTP.Paths)),
				}

				for _, path := range rule.HTTP.Paths {
					httpPath := networkingv1.HTTPIngressPath{
						Path:    path.Path,
						Backend: path.Backend,
					}
					if path.PathType != nil {
						pathType := networkingv1.PathType(*path.PathType)
						httpPath.PathType = &pathType
					}
					httpRule.Paths = append(httpRule.Paths, httpPath)
				}
				ingressRule.HTTP = httpRule
			}

			ingress.Spec.Rules = append(ingress.Spec.Rules, ingressRule)
		}
	}

	// 构建TLS配置
	if len(req.TLS) > 0 {
		ingress.Spec.TLS = make([]networkingv1.IngressTLS, 0, len(req.TLS))
		for _, tls := range req.TLS {
			ingress.Spec.TLS = append(ingress.Spec.TLS, networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			})
		}
	}

	return ingress, nil
}

// BuildIngressFromUpdateSpec 从UpdateIngressReq构建Ingress对象
func BuildIngressFromUpdateSpec(req *model.UpdateIngressReq) (*networkingv1.Ingress, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: req.IngressClassName,
		},
	}

	// 构建规则
	if len(req.Rules) > 0 {
		ingress.Spec.Rules = make([]networkingv1.IngressRule, 0, len(req.Rules))
		for _, rule := range req.Rules {
			ingressRule := networkingv1.IngressRule{
				Host: rule.Host,
			}

			if len(rule.HTTP.Paths) > 0 {
				httpRule := &networkingv1.HTTPIngressRuleValue{
					Paths: make([]networkingv1.HTTPIngressPath, 0, len(rule.HTTP.Paths)),
				}

				for _, path := range rule.HTTP.Paths {
					httpPath := networkingv1.HTTPIngressPath{
						Path:    path.Path,
						Backend: path.Backend,
					}
					if path.PathType != nil {
						pathType := networkingv1.PathType(*path.PathType)
						httpPath.PathType = &pathType
					}
					httpRule.Paths = append(httpRule.Paths, httpPath)
				}
				ingressRule.HTTP = httpRule
			}

			ingress.Spec.Rules = append(ingress.Spec.Rules, ingressRule)
		}
	}

	// 构建TLS配置
	if len(req.TLS) > 0 {
		ingress.Spec.TLS = make([]networkingv1.IngressTLS, 0, len(req.TLS))
		for _, tls := range req.TLS {
			ingress.Spec.TLS = append(ingress.Spec.TLS, networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			})
		}
	}

	return ingress, nil
}

// convertIngressStatusToEnum 转换状态字符串为枚举值
func convertIngressStatusToEnum(status string) model.K8sIngressStatus {
	switch status {
	case StatusRunning, StatusReady:
		return model.K8sIngressStatusRunning
	case StatusPending:
		return model.K8sIngressStatusPending
	case StatusTerminating, StatusFailed:
		return model.K8sIngressStatusFailed
	default:
		return model.K8sIngressStatusPending
	}
}

// BuildIngressListOptions 构建Ingress列表查询选项
func BuildIngressListOptions(req *model.GetIngressListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 构建标签选择器
	var labelSelectors []string
	for key, value := range req.Labels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
	}
	if len(labelSelectors) > 0 {
		options.LabelSelector = strings.Join(labelSelectors, ",")
	}

	return options
}

// FilterIngressesByStatus 根据Ingress状态过滤
func FilterIngressesByStatus(ingresses []networkingv1.Ingress, status string) []networkingv1.Ingress {
	if status == "" {
		return ingresses
	}

	var filtered []networkingv1.Ingress
	for _, ingress := range ingresses {
		ingressStatus := IngressStatus(&ingress)
		if strings.EqualFold(ingressStatus, status) {
			filtered = append(filtered, ingress)
		}
	}

	return filtered
}

// PaginateK8sIngresses 对 K8sIngress 列表进行分页
func PaginateK8sIngresses(ingresses []*model.K8sIngress, page, size int) ([]*model.K8sIngress, int64) {
	total := int64(len(ingresses))
	if total == 0 {
		return []*model.K8sIngress{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return ingresses, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []*model.K8sIngress{}, total
	}
	if end > total {
		end = total
	}

	return ingresses[start:end], total
}

// BuildIngressListPagination 构建Ingress列表分页逻辑
func BuildIngressListPagination(ingresses []networkingv1.Ingress, page, size int) ([]networkingv1.Ingress, int64) {
	total := int64(len(ingresses))
	if total == 0 {
		return []networkingv1.Ingress{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return ingresses, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []networkingv1.Ingress{}, total
	}
	if end > total {
		end = total
	}

	return ingresses[start:end], total
}
