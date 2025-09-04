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
	"sort"
	"strings"
	"time"

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

// CollectNodeStats 收集节点统计信息
func CollectNodeStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("收集节点统计信息失败: %w", err)
	}

	stats.NodeStats.TotalNodes = len(nodes.Items)

	for _, node := range nodes.Items {
		// 检查节点就绪状态
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status == corev1.ConditionTrue {
					stats.NodeStats.ReadyNodes++
				} else {
					stats.NodeStats.NotReadyNodes++
				}
				break
			}
		}

		// 检查节点角色
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			stats.NodeStats.MasterNodes++
		} else if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			stats.NodeStats.MasterNodes++
		} else {
			stats.NodeStats.WorkerNodes++
		}
	}

	return nil
}

// CollectPodStats 收集Pod统计信息
func CollectPodStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("收集Pod统计信息失败: %w", err)
	}

	stats.PodStats.TotalPods = len(pods.Items)

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			stats.PodStats.RunningPods++
		case corev1.PodPending:
			stats.PodStats.PendingPods++
		case corev1.PodSucceeded:
			stats.PodStats.SucceededPods++
		case corev1.PodFailed:
			stats.PodStats.FailedPods++
		case corev1.PodUnknown:
			stats.PodStats.UnknownPods++
		}
	}

	return nil
}

// NamespaceResourceUsage 命名空间资源使用统计
type NamespaceResourceUsage struct {
	Name     string
	PodCount int
}

// CollectNamespaceStats 收集命名空间统计信息
func CollectNamespaceStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("收集命名空间统计信息失败: %w", err)
	}

	stats.NamespaceStats.TotalNamespaces = len(namespaces.Items)

	// 统计各命名空间的pod数量
	podsByNamespace := make(map[string]int)
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			podsByNamespace[pod.Namespace]++
		}
	}

	var nsUsages []NamespaceResourceUsage
	for _, ns := range namespaces.Items {
		if ns.Status.Phase == corev1.NamespaceActive {
			stats.NamespaceStats.ActiveNamespaces++
		}

		// 检查是否为系统命名空间
		if IsSystemNamespace(ns.Name) {
			stats.NamespaceStats.SystemNamespaces++
		} else {
			stats.NamespaceStats.UserNamespaces++
		}

		// 收集每个命名空间的pod数量
		nsUsages = append(nsUsages, NamespaceResourceUsage{
			Name:     ns.Name,
			PodCount: podsByNamespace[ns.Name],
		})
	}

	// 按pod数量排序，获取前5个最活跃的命名空间
	sort.Slice(nsUsages, func(i, j int) bool {
		return nsUsages[i].PodCount > nsUsages[j].PodCount
	})

	var topNamespaces []string
	for i, ns := range nsUsages {
		if i >= 5 {
			break
		}
		topNamespaces = append(topNamespaces, ns.Name)
	}
	stats.NamespaceStats.TopNamespaces = topNamespaces

	return nil
}

// CollectWorkloadStats 收集工作负载统计信息
func CollectWorkloadStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	var errs []error

	// Deployments
	deployments, err := kubeClient.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Deployments失败: %w", err))
	} else {
		stats.WorkloadStats.Deployments = len(deployments.Items)
	}

	// StatefulSets
	statefulsets, err := kubeClient.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取StatefulSets失败: %w", err))
	} else {
		stats.WorkloadStats.StatefulSets = len(statefulsets.Items)
	}

	// DaemonSets
	daemonsets, err := kubeClient.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取DaemonSets失败: %w", err))
	} else {
		stats.WorkloadStats.DaemonSets = len(daemonsets.Items)
	}

	// Jobs
	jobs, err := kubeClient.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Jobs失败: %w", err))
	} else {
		stats.WorkloadStats.Jobs = len(jobs.Items)
	}

	// CronJobs
	cronjobs, err := kubeClient.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取CronJobs失败: %w", err))
	} else {
		stats.WorkloadStats.CronJobs = len(cronjobs.Items)
	}

	// Services
	services, err := kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Services失败: %w", err))
	} else {
		stats.WorkloadStats.Services = len(services.Items)
	}

	// ConfigMaps
	configmaps, err := kubeClient.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取ConfigMaps失败: %w", err))
	} else {
		stats.WorkloadStats.ConfigMaps = len(configmaps.Items)
	}

	// Secrets
	secrets, err := kubeClient.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Secrets失败: %w", err))
	} else {
		stats.WorkloadStats.Secrets = len(secrets.Items)
	}

	// Ingresses
	ingresses, err := kubeClient.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Ingresses失败: %w", err))
	} else {
		stats.WorkloadStats.Ingresses = len(ingresses.Items)
	}

	// 如果有错误，返回合并的错误信息
	if len(errs) > 0 {
		var errMessages []string
		for _, e := range errs {
			errMessages = append(errMessages, e.Error())
		}
		return fmt.Errorf("收集工作负载统计时出现错误: %s", strings.Join(errMessages, "; "))
	}

	return nil
}

// CollectResourceStats 收集资源统计信息
func CollectResourceStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("收集资源统计信息失败: %w", err)
	}

	var totalCPU, totalMemory, totalStorage int64
	var allocatableCPU, allocatableMemory, allocatableStorage int64

	for _, node := range nodes.Items {
		// 总容量
		cpu := node.Status.Capacity[corev1.ResourceCPU]
		memory := node.Status.Capacity[corev1.ResourceMemory]
		storage := node.Status.Capacity[corev1.ResourceEphemeralStorage]

		totalCPU += cpu.MilliValue()
		totalMemory += memory.Value()
		totalStorage += storage.Value()

		// 可分配容量
		allocCpu := node.Status.Allocatable[corev1.ResourceCPU]
		allocMemory := node.Status.Allocatable[corev1.ResourceMemory]
		allocStorage := node.Status.Allocatable[corev1.ResourceEphemeralStorage]

		allocatableCPU += allocCpu.MilliValue()
		allocatableMemory += allocMemory.Value()
		allocatableStorage += allocStorage.Value()
	}

	stats.ResourceStats.TotalCPU = fmt.Sprintf("%.1f cores", float64(totalCPU)/1000)
	stats.ResourceStats.TotalMemory = fmt.Sprintf("%.1fGi", float64(totalMemory)/(1024*1024*1024))
	stats.ResourceStats.TotalStorage = fmt.Sprintf("%.1fGi", float64(totalStorage)/(1024*1024*1024))

	// 计算已请求的资源
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		var requestedCPU, requestedMemory int64

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
				for _, container := range pod.Spec.Containers {
					if req := container.Resources.Requests; req != nil {
						if cpuReq := req[corev1.ResourceCPU]; !cpuReq.IsZero() {
							requestedCPU += cpuReq.MilliValue()
						}
						if memReq := req[corev1.ResourceMemory]; !memReq.IsZero() {
							requestedMemory += memReq.Value()
						}
					}
				}
			}
		}

		stats.ResourceStats.UsedCPU = fmt.Sprintf("%.1f cores", float64(requestedCPU)/1000)
		stats.ResourceStats.UsedMemory = fmt.Sprintf("%.1fGi", float64(requestedMemory)/(1024*1024*1024))

		// 计算利用率（基于请求量）
		if allocatableCPU > 0 {
			stats.ResourceStats.CPUUtilization = float64(requestedCPU) / float64(allocatableCPU) * 100
		}
		if allocatableMemory > 0 {
			stats.ResourceStats.MemoryUtilization = float64(requestedMemory) / float64(allocatableMemory) * 100
		}
	} else {
		stats.ResourceStats.UsedCPU = "获取失败"
		stats.ResourceStats.UsedMemory = "获取失败"
		stats.ResourceStats.CPUUtilization = 0.0
		stats.ResourceStats.MemoryUtilization = 0.0
	}

	stats.ResourceStats.UsedStorage = "需要metrics-server"
	stats.ResourceStats.StorageUtilization = 0.0

	return nil
}

// CollectStorageStats 收集存储统计信息
func CollectStorageStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	var errs []error

	// PersistentVolumes
	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取PersistentVolumes失败: %w", err))
	} else {
		stats.StorageStats.TotalPV = len(pvs.Items)
		var totalPVCapacity int64

		for _, pv := range pvs.Items {
			switch pv.Status.Phase {
			case corev1.VolumeBound:
				stats.StorageStats.BoundPV++
			case corev1.VolumeAvailable:
				stats.StorageStats.AvailablePV++
			}

			// 计算总容量
			if capacity := pv.Spec.Capacity[corev1.ResourceStorage]; !capacity.IsZero() {
				totalPVCapacity += capacity.Value()
			}
		}

		if totalPVCapacity > 0 {
			stats.StorageStats.TotalCapacity = fmt.Sprintf("%.1fGi", float64(totalPVCapacity)/(1024*1024*1024))
		} else {
			stats.StorageStats.TotalCapacity = "0Gi"
		}
	}

	// PersistentVolumeClaims
	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取PersistentVolumeClaims失败: %w", err))
	} else {
		stats.StorageStats.TotalPVC = len(pvcs.Items)
		for _, pvc := range pvcs.Items {
			switch pvc.Status.Phase {
			case corev1.ClaimBound:
				stats.StorageStats.BoundPVC++
			case corev1.ClaimPending:
				stats.StorageStats.PendingPVC++
			}
		}
	}

	// StorageClasses
	scList, err := kubeClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取StorageClasses失败: %w", err))
	} else {
		stats.StorageStats.StorageClasses = len(scList.Items)
	}

	// 如果TotalCapacity仍未设置，则设置默认值
	if stats.StorageStats.TotalCapacity == "" {
		stats.StorageStats.TotalCapacity = "0Gi"
	}

	// 如果有错误，返回合并的错误信息
	if len(errs) > 0 {
		var errMessages []string
		for _, e := range errs {
			errMessages = append(errMessages, e.Error())
		}
		return fmt.Errorf("收集存储统计时出现错误: %s", strings.Join(errMessages, "; "))
	}

	return nil
}

// CollectNetworkStats 收集网络统计信息
func CollectNetworkStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	var errs []error

	// Services
	services, err := kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Services失败: %w", err))
	} else {
		stats.NetworkStats.Services = len(services.Items)
	}

	// Endpoints
	endpoints, err := kubeClient.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Endpoints失败: %w", err))
	} else {
		stats.NetworkStats.Endpoints = len(endpoints.Items)
	}

	// Ingresses
	ingresses, err := kubeClient.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取Ingresses失败: %w", err))
	} else {
		stats.NetworkStats.Ingresses = len(ingresses.Items)
	}

	// NetworkPolicies
	netpols, err := kubeClient.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("获取NetworkPolicies失败: %w", err))
	} else {
		stats.NetworkStats.NetworkPolicies = len(netpols.Items)
	}

	// 如果有错误，返回合并的错误信息
	if len(errs) > 0 {
		var errMessages []string
		for _, e := range errs {
			errMessages = append(errMessages, e.Error())
		}
		return fmt.Errorf("收集网络统计时出现错误: %s", strings.Join(errMessages, "; "))
	}

	return nil
}

// CollectEventStats 收集事件统计信息
func CollectEventStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStats) error {
	events, err := kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("收集事件统计信息失败: %w", err)
	}

	stats.EventStats.TotalEvents = len(events.Items)

	recentTime := time.Now().Add(-1 * time.Hour)

	for _, event := range events.Items {
		switch event.Type {
		case "Warning":
			stats.EventStats.WarningEvents++
		case "Normal":
			stats.EventStats.NormalEvents++
		}

		// 统计最近1小时的事件
		if event.CreationTimestamp.Time.After(recentTime) {
			stats.EventStats.RecentEvents++
		}
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

func GetComponentStatuses(ctx context.Context, kubeClient kubernetes.Interface) ([]*model.ComponentHealthStatus, int64, error) {
	componentStatuses, err := kubeClient.CoreV1().ComponentStatuses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取组件状态失败: %w", err)
	}

	var componentStatusList []*model.ComponentHealthStatus
	for _, cs := range componentStatuses.Items {
		status := "healthy"
		message := "正常"

		// 检查健康状态
		for _, condition := range cs.Conditions {
			if condition.Type == "Healthy" && condition.Status != "True" {
				status = "unhealthy"
				message = condition.Message
				break
			}
		}

		componentStatusList = append(componentStatusList, &model.ComponentHealthStatus{
			Name:      cs.Name,
			Status:    status,
			Message:   message,
			Timestamp: time.Now().Format(time.DateTime),
		})
	}
	return componentStatusList, int64(len(componentStatuses.Items)), nil
}
