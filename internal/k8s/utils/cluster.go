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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/pingcap/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ValidateResourceQuantities 验证集群资源请求量和限制量
func ValidateResourceQuantities(cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群配置不能为空")
	}

	// 定义资源验证配置
	resources := []struct {
		requestField, limitField *string
		requestName, limitName   string
	}{
		{&cluster.CpuRequest, &cluster.CpuLimit, "CPU 请求量", "CPU 限制量"},
		{&cluster.MemoryRequest, &cluster.MemoryLimit, "内存请求量", "内存限制量"},
	}

	// 验证每种资源类型
	for _, res := range resources {
		if err := validateResourcePair(res.requestField, res.limitField, res.requestName, res.limitName); err != nil {
			return err
		}
	}

	return nil
}

// validateResourcePair 验证单对资源的请求量和限制量
func validateResourcePair(requestField, limitField *string, requestName, limitName string) error {
	// 检查字段是否为空
	if *requestField == "" || *limitField == "" {
		return nil // 允许空值，跳过验证
	}

	// 解析请求量
	reqQuantity, err := resource.ParseQuantity(*requestField)
	if err != nil {
		return fmt.Errorf("%s格式错误: %w", requestName, err)
	}

	// 解析限制量
	limQuantity, err := resource.ParseQuantity(*limitField)
	if err != nil {
		return fmt.Errorf("%s格式错误: %w", limitName, err)
	}

	// 如果请求量大于限制量，自动调整请求量
	if reqQuantity.Cmp(limQuantity) > 0 {
		*requestField = *limitField
	}

	return nil
}

// AddClusterResourceLimit 添加集群资源限制
func AddClusterResourceLimit(ctx context.Context, kubeClient kubernetes.Interface, cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群配置不能为空")
	}

	// 如果没有限制的命名空间，则跳过
	if len(cluster.RestrictNamespace) == 0 {
		return nil
	}

	// 为每个限制的命名空间创建资源配额和限制范围
	for _, namespace := range cluster.RestrictNamespace {
		if namespace == "" {
			continue
		}

		// 创建 ResourceQuota
		if err := createResourceQuota(ctx, kubeClient, namespace, cluster); err != nil {
			return fmt.Errorf("创建命名空间 %s 的资源配额失败: %w", namespace, err)
		}

		// 创建 LimitRange
		if err := createLimitRange(ctx, kubeClient, namespace, cluster); err != nil {
			return fmt.Errorf("创建命名空间 %s 的限制范围失败: %w", namespace, err)
		}
	}

	return nil
}

// createResourceQuota 创建资源配额
func createResourceQuota(ctx context.Context, kubeClient kubernetes.Interface, namespace string, cluster *model.K8sCluster) error {
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-resource-quota",
			Namespace: namespace,
			Labels: map[string]string{
				"managed-by":   "ai-cloudops",
				"cluster-name": cluster.Name,
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{},
		},
	}

	// 设置CPU资源配额
	if cluster.CpuLimit != "" {
		if cpuQuantity, err := resource.ParseQuantity(cluster.CpuLimit); err == nil {
			resourceQuota.Spec.Hard[corev1.ResourceRequestsCPU] = cpuQuantity
			resourceQuota.Spec.Hard[corev1.ResourceLimitsCPU] = cpuQuantity
		}
	}

	// 设置内存资源配额
	if cluster.MemoryLimit != "" {
		if memoryQuantity, err := resource.ParseQuantity(cluster.MemoryLimit); err == nil {
			resourceQuota.Spec.Hard[corev1.ResourceRequestsMemory] = memoryQuantity
			resourceQuota.Spec.Hard[corev1.ResourceLimitsMemory] = memoryQuantity
		}
	}

	// 如果没有设置任何资源限制，则跳过创建
	if len(resourceQuota.Spec.Hard) == 0 {
		return nil
	}

	_, err := kubeClient.CoreV1().ResourceQuotas(namespace).Create(ctx, resourceQuota, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("创建资源配额失败: %w", err)
	}

	return nil
}

// createLimitRange 创建限制范围
func createLimitRange(ctx context.Context, kubeClient kubernetes.Interface, namespace string, cluster *model.K8sCluster) error {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-limit-range",
			Namespace: namespace,
			Labels: map[string]string{
				"managed-by":   "ai-cloudops",
				"cluster-name": cluster.Name,
			},
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{},
		},
	}

	// 创建容器级别的限制
	containerLimit := corev1.LimitRangeItem{
		Type:           corev1.LimitTypeContainer,
		Default:        corev1.ResourceList{},
		DefaultRequest: corev1.ResourceList{},
		Max:            corev1.ResourceList{},
	}

	hasLimits := false

	// 设置CPU限制
	if cluster.CpuRequest != "" {
		if cpuRequestQuantity, err := resource.ParseQuantity(cluster.CpuRequest); err == nil {
			containerLimit.DefaultRequest[corev1.ResourceCPU] = cpuRequestQuantity
			hasLimits = true
		}
	}

	if cluster.CpuLimit != "" {
		if cpuLimitQuantity, err := resource.ParseQuantity(cluster.CpuLimit); err == nil {
			containerLimit.Default[corev1.ResourceCPU] = cpuLimitQuantity
			containerLimit.Max[corev1.ResourceCPU] = cpuLimitQuantity
			hasLimits = true
		}
	}

	// 设置内存限制
	if cluster.MemoryRequest != "" {
		if memoryRequestQuantity, err := resource.ParseQuantity(cluster.MemoryRequest); err == nil {
			containerLimit.DefaultRequest[corev1.ResourceMemory] = memoryRequestQuantity
			hasLimits = true
		}
	}

	if cluster.MemoryLimit != "" {
		if memoryLimitQuantity, err := resource.ParseQuantity(cluster.MemoryLimit); err == nil {
			containerLimit.Default[corev1.ResourceMemory] = memoryLimitQuantity
			containerLimit.Max[corev1.ResourceMemory] = memoryLimitQuantity
			hasLimits = true
		}
	}

	// 如果没有设置任何限制，则跳过创建
	if !hasLimits {
		return nil
	}

	limitRange.Spec.Limits = append(limitRange.Spec.Limits, containerLimit)

	_, err := kubeClient.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("创建限制范围失败: %w", err)
	}

	return nil
}

// IsSystemNamespace 判断是否为系统命名空间
func IsSystemNamespace(name string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"kubernetes-dashboard",
		"istio-system",
		"prometheus-system",
		"monitoring",
		"logging",
		"cert-manager",
		"ingress-nginx",
		"metallb-system",
		"argocd",
		"gitlab-system",
		"harbor-system",
	}

	for _, sysNs := range systemNamespaces {
		if name == sysNs || strings.HasPrefix(name, sysNs) {
			return true
		}
	}

	return false
}
