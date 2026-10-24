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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"k8s.io/client-go/kubernetes"
)

func ValidateClusterExists(cluster *model.K8sCluster, clusterID int) error {
	if cluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", clusterID)
	}
	return nil
}

// Bare CPU and memory values follow the existing API units: millicores and MiB.
func clusterQuotaResources(cluster *model.K8sCluster) (corev1.ResourceList, error) {
	if cluster == nil {
		return nil, fmt.Errorf("集群配置不能为空")
	}
	hard := corev1.ResourceList{}
	fields := []struct {
		name      corev1.ResourceName
		raw, unit string
	}{
		{corev1.ResourceRequestsCPU, cluster.CpuRequest, "m"}, {corev1.ResourceLimitsCPU, cluster.CpuLimit, "m"},
		{corev1.ResourceRequestsMemory, cluster.MemoryRequest, "Mi"}, {corev1.ResourceLimitsMemory, cluster.MemoryLimit, "Mi"},
	}
	for _, field := range fields {
		value := strings.TrimSpace(field.raw)
		if value == "" {
			continue
		}
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			value += field.unit
		}
		quantity, err := resource.ParseQuantity(value)
		if err != nil || quantity.Sign() < 0 {
			return nil, fmt.Errorf("%s 必须为合法非负资源数量", field.name)
		}
		if field.unit == "m" && (quantity.MilliValue() < 0 || quantity.Cmp(*resource.NewMilliQuantity(quantity.MilliValue(), resource.DecimalSI)) != 0) {
			return nil, fmt.Errorf("CPU数量超出范围或精度小于1m")
		}
		hard[field.name] = quantity
	}
	for _, pair := range [][2]corev1.ResourceName{{corev1.ResourceRequestsCPU, corev1.ResourceLimitsCPU}, {corev1.ResourceRequestsMemory, corev1.ResourceLimitsMemory}} {
		request, hasRequest := hard[pair[0]]
		limit, hasLimit := hard[pair[1]]
		if hasRequest && hasLimit && request.Cmp(limit) > 0 {
			return nil, fmt.Errorf("%s 不能超过 %s", pair[0], pair[1])
		}
	}
	return hard, nil
}

func ValidateResourceQuantities(cluster *model.K8sCluster) error {
	if _, err := clusterQuotaResources(cluster); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, namespace := range cluster.RestrictNamespace {
		if len(validation.IsDNS1123Label(namespace)) > 0 {
			return fmt.Errorf("无效的配额命名空间: %s", namespace)
		}
		if seen[namespace] {
			return fmt.Errorf("配额命名空间重复: %s", namespace)
		}
		seen[namespace] = true
	}
	return nil
}

const quotaOwnerLabel = "cloudops.io/cluster-id"

func clusterQuotaName(id int) string { return "cloudops-quota-" + strconv.Itoa(id) }

// AddClusterResourceLimit enforces aggregate namespace quotas, never Pod defaults.
// An empty namespace selection never expands to every namespace.
func AddClusterResourceLimit(ctx context.Context, client kubernetes.Interface, cluster *model.K8sCluster) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateResourceQuantities(cluster); err != nil {
		return err
	}
	hard, _ := clusterQuotaResources(cluster)
	if len(hard) == 0 || len(cluster.RestrictNamespace) == 0 {
		return nil
	}
	if client == nil || cluster.ID <= 0 {
		return fmt.Errorf("配额客户端和集群ID不能为空")
	}
	for _, namespace := range cluster.RestrictNamespace {
		quotas := client.CoreV1().ResourceQuotas(namespace)
		existing, err := quotas.Get(ctx, clusterQuotaName(cluster.ID), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = quotas.Create(ctx, &corev1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{Name: clusterQuotaName(cluster.ID), Labels: map[string]string{quotaOwnerLabel: strconv.Itoa(cluster.ID)}}, Spec: corev1.ResourceQuotaSpec{Hard: hard.DeepCopy()}}, metav1.CreateOptions{})
		} else if err == nil {
			if existing.Labels[quotaOwnerLabel] != strconv.Itoa(cluster.ID) {
				return fmt.Errorf("命名空间 %s 存在非本平台管理的同名配额", namespace)
			}
			existing.Spec.Hard = hard.DeepCopy()
			_, err = quotas.Update(ctx, existing, metav1.UpdateOptions{})
		}
		if err != nil {
			return fmt.Errorf("命名空间 %s 配额应用失败: %w", namespace, err)
		}
	}
	return nil
}

// RemoveClusterResourceLimits deletes only quota objects owned by this registration.
func RemoveClusterResourceLimits(ctx context.Context, client kubernetes.Interface, id int, namespaces []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(namespaces) == 0 {
		return nil
	}
	if client == nil || id <= 0 {
		return fmt.Errorf("配额客户端和集群ID不能为空")
	}
	for _, namespace := range namespaces {
		// A legacy invalid namespace cannot contain any Kubernetes quota object.
		if len(validation.IsDNS1123Label(namespace)) > 0 {
			continue
		}
		quotas := client.CoreV1().ResourceQuotas(namespace)
		existing, err := quotas.Get(ctx, clusterQuotaName(id), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if existing.Labels[quotaOwnerLabel] != strconv.Itoa(id) {
			continue
		}
		uid := existing.UID
		version := existing.ResourceVersion
		if err := quotas.Delete(ctx, existing.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &version}}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
