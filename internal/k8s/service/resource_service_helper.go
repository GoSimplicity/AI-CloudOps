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

package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// parsePeriod 解析时间周期
func (r *resourceService) parsePeriod(period string) (time.Duration, error) {
	switch period {
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("不支持的时间周期: %s", period)
	}
}

// generateMockTrendData 生成模拟趋势数据
func (r *resourceService) generateMockTrendData(resourceType string, duration time.Duration) model.TrendData {
	points := int(duration.Hours())
	if points > 168 { // 超过一周，减少数据点
		points = points / 24 // 按天
	}
	if points < 10 {
		points = 10 // 至少10个点
	}

	timestamps := make([]string, points)
	values := make([]float64, points)

	baseValue := 50.0
	switch resourceType {
	case "CPU":
		baseValue = 45.0
	case "Memory":
		baseValue = 60.0
	case "Pod":
		baseValue = 100.0
	case "Node":
		baseValue = 5.0
	}

	var max, min, sum float64
	for i := 0; i < points; i++ {
		timestamp := time.Now().Add(-duration + time.Duration(i)*duration/time.Duration(points))
		timestamps[i] = timestamp.Format(time.RFC3339)

		// 生成带有随机波动的数值
		variation := (float64(i%10) - 5) * 2
		value := baseValue + variation
		if value < 0 {
			value = 0
		}
		values[i] = value

		sum += value
		if i == 0 || value > max {
			max = value
		}
		if i == 0 || value < min {
			min = value
		}
	}

	unit := "%"
	if resourceType == "Pod" || resourceType == "Node" {
		unit = "count"
	}

	return model.TrendData{
		Timestamps: timestamps,
		Values:     values,
		Unit:       unit,
		Max:        max,
		Min:        min,
		Avg:        sum / float64(points),
	}
}

// generateResourcePredictions 生成资源预测
func (r *resourceService) generateResourcePredictions() []model.ResourcePredict {
	return []model.ResourcePredict{
		{
			Resource:    "CPU",
			PredictDays: 7,
			Tendency:    "increasing",
			Confidence:  0.75,
			Value:       65.0,
			Suggestion:  "建议监控CPU使用情况，考虑增加节点",
		},
		{
			Resource:    "Memory",
			PredictDays: 7,
			Tendency:    "stable",
			Confidence:  0.85,
			Value:       62.0,
			Suggestion:  "内存使用稳定，暂无需调整",
		},
		{
			Resource:    "Storage",
			PredictDays: 30,
			Tendency:    "increasing",
			Confidence:  0.60,
			Value:       80.0,
			Suggestion:  "存储增长较快，建议规划存储扩容",
		},
	}
}

// getNodeDistribution 获取节点分布
func (r *resourceService) getNodeDistribution(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.NodeResourceDistrib, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 按节点统计Pod数量
	podsByNode := make(map[string]int)
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" {
			podsByNode[pod.Spec.NodeName]++
		}
	}

	result := make([]model.NodeResourceDistrib, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		role := "worker"
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			role = "master"
		} else if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			role = "master"
		}

		status := "Ready"
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				status = "NotReady"
				break
			}
		}

		distrib := model.NodeResourceDistrib{
			NodeName:   node.Name,
			Role:       role,
			CPU:        node.Status.Capacity.Cpu().String(),
			Memory:     fmt.Sprintf("%.1fGi", node.Status.Capacity.Memory().AsApproximateFloat64()/(1024*1024*1024)),
			Storage:    fmt.Sprintf("%.1fGi", node.Status.Capacity.StorageEphemeral().AsApproximateFloat64()/(1024*1024*1024)),
			PodCount:   podsByNode[node.Name],
			CPUUtil:    0.0, // 需要metrics-server
			MemoryUtil: 0.0, // 需要metrics-server
			Status:     status,
		}
		result = append(result, distrib)
	}

	return result, nil
}

// getNamespaceDistribution 获取命名空间分布
func (r *resourceService) getNamespaceDistribution(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.NSResourceDistrib, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 按命名空间统计资源
	nsStats := make(map[string]*model.NSResourceDistrib)
	for _, ns := range namespaces.Items {
		nsStats[ns.Name] = &model.NSResourceDistrib{
			Namespace: ns.Name,
			IsSystem:  utils.IsSystemNamespace(ns.Name),
		}
	}

	// 统计每个命名空间的Pod和资源
	for _, pod := range pods.Items {
		if nsDistrib, exists := nsStats[pod.Namespace]; exists {
			nsDistrib.PodCount++

			var cpuRequest, cpuLimit, memRequest, memLimit int64
			for _, container := range pod.Spec.Containers {
				if req := container.Resources.Requests; req != nil {
					if cpu := req.Cpu(); !cpu.IsZero() {
						cpuRequest += cpu.MilliValue()
					}
					if mem := req.Memory(); !mem.IsZero() {
						memRequest += mem.Value()
					}
				}
				if limit := container.Resources.Limits; limit != nil {
					if cpu := limit.Cpu(); !cpu.IsZero() {
						cpuLimit += cpu.MilliValue()
					}
					if mem := limit.Memory(); !mem.IsZero() {
						memLimit += mem.Value()
					}
				}
			}

			nsDistrib.CPURequest = fmt.Sprintf("%.2f cores", float64(cpuRequest)/1000)
			nsDistrib.CPULimit = fmt.Sprintf("%.2f cores", float64(cpuLimit)/1000)
			nsDistrib.MemRequest = fmt.Sprintf("%.2f Gi", float64(memRequest)/(1024*1024*1024))
			nsDistrib.MemLimit = fmt.Sprintf("%.2f Gi", float64(memLimit)/(1024*1024*1024))
		}
	}

	result := make([]model.NSResourceDistrib, 0, len(nsStats))
	for _, nsDistrib := range nsStats {
		result = append(result, *nsDistrib)
	}

	// 按Pod数量排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].PodCount > result[j].PodCount
	})

	return result, nil
}

// getDetailedWorkloadDistribution 获取详细工作负载分布
func (r *resourceService) getDetailedWorkloadDistribution(ctx context.Context, kubeClient *kubernetes.Clientset) (*model.WorkloadDistribution, error) {
	stats := &model.ClusterStats{}
	err := utils.CollectWorkloadStats(ctx, kubeClient, stats)
	if err != nil {
		return nil, err
	}

	// 获取按命名空间的工作负载统计
	workloadsByNS, err := r.getWorkloadsByNamespace(ctx, kubeClient)
	if err != nil {
		r.logger.Warn("获取命名空间工作负载统计失败", zap.Error(err))
	}

	// 获取按类型的资源使用统计
	resourcesByType, err := r.getResourcesByWorkloadType(ctx, kubeClient)
	if err != nil {
		r.logger.Warn("获取工作负载资源统计失败", zap.Error(err))
	}

	return &model.WorkloadDistribution{
		Deployments:     stats.WorkloadStats.Deployments,
		StatefulSets:    stats.WorkloadStats.StatefulSets,
		DaemonSets:      stats.WorkloadStats.DaemonSets,
		Jobs:            stats.WorkloadStats.Jobs,
		CronJobs:        stats.WorkloadStats.CronJobs,
		Services:        stats.WorkloadStats.Services,
		ConfigMaps:      stats.WorkloadStats.ConfigMaps,
		Secrets:         stats.WorkloadStats.Secrets,
		Ingresses:       stats.WorkloadStats.Ingresses,
		WorkloadsByNS:   workloadsByNS,
		ResourcesByType: resourcesByType,
	}, nil
}

// getWorkloadsByNamespace 获取按命名空间的工作负载统计
func (r *resourceService) getWorkloadsByNamespace(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.NSWorkloadCount, error) {
	// 获取所有命名空间
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]model.NSWorkloadCount, 0, len(namespaces.Items))

	for _, ns := range namespaces.Items {
		count := model.NSWorkloadCount{
			Namespace: ns.Name,
			Types:     make(map[string]int),
		}

		// 统计Deployments
		deployments, _ := kubeClient.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
		if len(deployments.Items) > 0 {
			count.Types["Deployment"] = len(deployments.Items)
			count.Count += len(deployments.Items)
		}

		// 统计StatefulSets
		statefulsets, _ := kubeClient.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{})
		if len(statefulsets.Items) > 0 {
			count.Types["StatefulSet"] = len(statefulsets.Items)
			count.Count += len(statefulsets.Items)
		}

		// 统计DaemonSets
		daemonsets, _ := kubeClient.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{})
		if len(daemonsets.Items) > 0 {
			count.Types["DaemonSet"] = len(daemonsets.Items)
			count.Count += len(daemonsets.Items)
		}

		// 统计Services
		services, _ := kubeClient.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{})
		if len(services.Items) > 0 {
			count.Types["Service"] = len(services.Items)
			count.Count += len(services.Items)
		}

		if count.Count > 0 {
			result = append(result, count)
		}
	}

	// 按数量排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result, nil
}

// getResourcesByWorkloadType 获取按工作负载类型的资源使用
func (r *resourceService) getResourcesByWorkloadType(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.WorkloadResource, error) {
	result := []model.WorkloadResource{}

	// 统计Deployments资源使用
	deployments, err := kubeClient.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		depResource := r.calculateWorkloadResource("Deployment", len(deployments.Items), deployments.Items)
		result = append(result, depResource)
	}

	// 统计StatefulSets资源使用
	statefulsets, err := kubeClient.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stsResource := r.calculateWorkloadResource("StatefulSet", len(statefulsets.Items), statefulsets.Items)
		result = append(result, stsResource)
	}

	// 统计DaemonSets资源使用
	daemonsets, err := kubeClient.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		dsResource := r.calculateWorkloadResource("DaemonSet", len(daemonsets.Items), daemonsets.Items)
		result = append(result, dsResource)
	}

	return result, nil
}

// calculateWorkloadResource 计算工作负载资源使用（泛型方法）
func (r *resourceService) calculateWorkloadResource(workloadType string, count int, workloads interface{}) model.WorkloadResource {
	// 这里简化处理，实际应该根据不同工作负载类型分别计算
	return model.WorkloadResource{
		Type:       workloadType,
		Count:      count,
		CPURequest: "计算中...",
		CPULimit:   "计算中...",
		MemRequest: "计算中...",
		MemLimit:   "计算中...",
	}
}

// generateResourceAllocationChart 生成资源分配图表
func (r *resourceService) generateResourceAllocationChart(nodeDistrib []model.NodeResourceDistrib, nsDistrib []model.NSResourceDistrib) model.ResourceAllocationChart {
	// CPU分配图表
	cpuLabels := make([]string, 0, len(nodeDistrib))
	cpuValues := make([]float64, 0, len(nodeDistrib))
	for _, node := range nodeDistrib {
		cpuLabels = append(cpuLabels, node.NodeName)
		cpuValues = append(cpuValues, node.CPUUtil)
	}

	// Memory分配图表
	memLabels := make([]string, 0, len(nodeDistrib))
	memValues := make([]float64, 0, len(nodeDistrib))
	for _, node := range nodeDistrib {
		memLabels = append(memLabels, node.NodeName)
		memValues = append(memValues, node.MemoryUtil)
	}

	// Pod分配图表
	podLabels := make([]string, 0, len(nsDistrib))
	podValues := make([]float64, 0, len(nsDistrib))
	for _, ns := range nsDistrib {
		if ns.PodCount > 0 {
			podLabels = append(podLabels, ns.Namespace)
			podValues = append(podValues, float64(ns.PodCount))
		}
	}

	return model.ResourceAllocationChart{
		CPUChart: model.PieChartData{
			Labels: cpuLabels,
			Values: cpuValues,
			Colors: generateColors(len(cpuLabels)),
		},
		MemoryChart: model.PieChartData{
			Labels: memLabels,
			Values: memValues,
			Colors: generateColors(len(memLabels)),
		},
		PodChart: model.PieChartData{
			Labels: podLabels,
			Values: podValues,
			Colors: generateColors(len(podLabels)),
		},
	}
}

// generateColors 生成图表颜色
func generateColors(count int) []string {
	colors := []string{
		"#FF6384", "#36A2EB", "#FFCE56", "#4BC0C0",
		"#9966FF", "#FF9F40", "#FF6384", "#C9CBCF",
		"#4BC0C0", "#FF6384", "#36A2EB", "#FFCE56",
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = colors[i%len(colors)]
	}
	return result
}
