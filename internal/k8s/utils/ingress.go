package utils

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	networkingv1 "k8s.io/api/networking/v1"
)

func ConvertToK8sIngress(ingress *networkingv1.Ingress, clusterID int) *model.K8sIngress {
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
					Path: path.Path,
				}
				if path.PathType != nil {
					ingressPath.PathType = string(*path.PathType)
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
		Name:              ingress.Name,
		Namespace:         ingress.Namespace,
		ClusterID:         clusterID,
		UID:               string(ingress.UID),
		IngressClassName:  ingressClassName,
		Rules:             rules,
		TLS:               tls,
		LoadBalancer:      loadBalancer,
		Labels:            ingress.Labels,
		Annotations:       ingress.Annotations,
		CreationTimestamp: ingress.CreationTimestamp.Time,
		Age:               age,
		Status:            status,
		Hosts:             hosts,
	}
}

func IngressStatus(item *networkingv1.Ingress) string {
	if item == nil {
		return statusUnknown
	}
	// 如果正在删除
	if item.DeletionTimestamp != nil {
		return statusTerminating
	}
	lb := item.Status.LoadBalancer
	ingressList := lb.Ingress
	if len(ingressList) == 0 {
		return statusPending
	}
	for _, entry := range ingressList {
		if entry.IP != "" || entry.Hostname != "" {
			return statusReady
		}
	}
	return statusUnknown
}
