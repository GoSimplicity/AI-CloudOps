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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getNodeUtilizations 获取节点利用率
func (r *resourceService) getNodeUtilizations(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.NodeUtilization, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 统计每个节点的Pod数量
	podsByNode := make(map[string]int)
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" {
			podsByNode[pod.Spec.NodeName]++
		}
	}

	result := make([]model.NodeUtilization, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		status := "Ready"
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				status = "NotReady"
				break
			}
		}

		efficiency := "高效"
		if podsByNode[node.Name] < 5 {
			efficiency = "低效"
		} else if podsByNode[node.Name] > 100 {
			efficiency = "过载"
		}

		util := model.NodeUtilization{
			NodeName:   node.Name,
			CPU:        0.0, // 需要metrics-server
			Memory:     0.0, // 需要metrics-server
			Storage:    0.0, // 需要metrics-server
			PodCount:   podsByNode[node.Name],
			Status:     status,
			Efficiency: efficiency,
		}
		result = append(result, util)
	}

	return result, nil
}

// getNamespaceUtilizations 获取命名空间利用率
func (r *resourceService) getNamespaceUtilizations(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.NSUtilization, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 按命名空间统计
	nsStats := make(map[string]*model.NSUtilization)
	for _, ns := range namespaces.Items {
		nsStats[ns.Name] = &model.NSUtilization{
			Namespace: ns.Name,
			IsSystem:  utils.IsSystemNamespace(ns.Name),
		}
	}

	for _, pod := range pods.Items {
		if nsUtil, exists := nsStats[pod.Namespace]; exists {
			nsUtil.PodCount++
			// CPU和Memory需要metrics-server支持
		}
	}

	result := make([]model.NSUtilization, 0, len(nsStats))
	for _, nsUtil := range nsStats {
		result = append(result, *nsUtil)
	}

	return result, nil
}

// generateUtilizationChart 生成利用率图表
func (r *resourceService) generateUtilizationChart(nodeUtils []model.NodeUtilization, nsUtils []model.NSUtilization) model.UtilizationChart {
	// 简化的热图数据生成
	xLabels := make([]string, 0, len(nodeUtils))
	yLabels := []string{"CPU", "Memory", "Storage"}
	heatmapData := make([][]float64, len(yLabels))

	for i := range heatmapData {
		heatmapData[i] = make([]float64, len(nodeUtils))
	}

	for i, node := range nodeUtils {
		xLabels = append(xLabels, node.NodeName)
		heatmapData[0][i] = node.CPU
		heatmapData[1][i] = node.Memory
		heatmapData[2][i] = node.Storage
	}

	return model.UtilizationChart{
		HeatmapData: heatmapData,
		XLabels:     xLabels,
		YLabels:     yLabels,
	}
}

// generateUtilizationAdvice 生成利用率建议
func (r *resourceService) generateUtilizationAdvice(overall model.UtilizationSummary, nodeUtils []model.NodeUtilization) []model.UtilizationAdvice {
	advice := make([]model.UtilizationAdvice, 0)

	// CPU利用率建议
	if overall.CPU > 80 {
		advice = append(advice, model.UtilizationAdvice{
			Type:        "optimization",
			Priority:    "high",
			Resource:    "CPU",
			Target:      "cluster",
			Current:     overall.CPU,
			Suggested:   70.0,
			Description: "集群CPU利用率过高，建议扩容或优化应用",
			Impact:      "性能影响严重",
		})
	}

	// 内存利用率建议
	if overall.Memory > 85 {
		advice = append(advice, model.UtilizationAdvice{
			Type:        "scaling",
			Priority:    "high",
			Resource:    "Memory",
			Target:      "cluster",
			Current:     overall.Memory,
			Suggested:   75.0,
			Description: "集群内存利用率过高，建议增加内存或扩容节点",
			Impact:      "可能出现OOM",
		})
	}

	// 节点效率建议
	for _, node := range nodeUtils {
		if node.Efficiency == "低效" {
			advice = append(advice, model.UtilizationAdvice{
				Type:        "rebalancing",
				Priority:    "medium",
				Resource:    "Pod",
				Target:      node.NodeName,
				Current:     float64(node.PodCount),
				Suggested:   20.0,
				Description: fmt.Sprintf("节点 %s Pod数量较少，资源利用率低", node.NodeName),
				Impact:      "资源浪费",
			})
		}
	}

	return advice
}

// getComponentHealth 获取组件健康状态
func (r *resourceService) getComponentHealth(ctx context.Context, kubeClient *kubernetes.Clientset) ([]model.ComponentHealth, error) {
	components, _, err := utils.GetComponentStatuses(ctx, kubeClient)
	if err != nil {
		return nil, err
	}

	result := make([]model.ComponentHealth, 0, len(components))
	for _, comp := range components {
		score := 100
		issues := 0
		details := []string{}

		if comp.Status != "healthy" {
			score = 0
			issues = 1
			details = append(details, comp.Message)
		}

		health := model.ComponentHealth{
			Component: comp.Name,
			Status:    comp.Status,
			Score:     score,
			Issues:    issues,
			LastCheck: comp.Timestamp,
			Details:   details,
		}
		result = append(result, health)
	}

	return result, nil
}

// identifyResourceIssues 识别资源问题
func (r *resourceService) identifyResourceIssues(ctx context.Context, kubeClient *kubernetes.Clientset, stats *model.ClusterStats) []model.ResourceIssue {
	issues := make([]model.ResourceIssue, 0)

	// 检查节点问题
	if stats.NodeStats.NotReadyNodes > 0 {
		issues = append(issues, model.ResourceIssue{
			Type:        "node",
			Severity:    "critical",
			Resource:    "Node",
			Description: fmt.Sprintf("%d个节点处于NotReady状态", stats.NodeStats.NotReadyNodes),
			Since:       time.Now().Add(-time.Hour).Format(time.RFC3339),
			Suggestion:  "检查节点状态并修复网络或资源问题",
		})
	}

	// 检查Pod问题
	if stats.PodStats.FailedPods > 0 {
		issues = append(issues, model.ResourceIssue{
			Type:        "pod",
			Severity:    "warning",
			Resource:    "Pod",
			Description: fmt.Sprintf("%d个Pod处于失败状态", stats.PodStats.FailedPods),
			Since:       time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			Suggestion:  "检查Pod日志并修复应用问题",
		})
	}

	// 检查资源利用率问题
	if stats.ResourceStats.CPUUtilization > 90 {
		issues = append(issues, model.ResourceIssue{
			Type:        "resource",
			Severity:    "critical",
			Resource:    "CPU",
			Description: fmt.Sprintf("CPU利用率过高: %.1f%%", stats.ResourceStats.CPUUtilization),
			Since:       time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
			Suggestion:  "考虑扩容或优化高CPU使用的应用",
		})
	}

	if stats.ResourceStats.MemoryUtilization > 90 {
		issues = append(issues, model.ResourceIssue{
			Type:        "resource",
			Severity:    "critical",
			Resource:    "Memory",
			Description: fmt.Sprintf("内存利用率过高: %.1f%%", stats.ResourceStats.MemoryUtilization),
			Since:       time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
			Suggestion:  "考虑增加内存或扩容节点",
		})
	}

	return issues
}

// generateHealthTrend 生成健康趋势
func (r *resourceService) generateHealthTrend() []model.HealthTrendPoint {
	trend := make([]model.HealthTrendPoint, 24)
	baseScore := 85

	for i := 0; i < 24; i++ {
		timestamp := time.Now().Add(-time.Duration(23-i) * time.Hour)
		score := baseScore + (i%5-2)*3 // 模拟波动
		issues := 0
		if score < 80 {
			issues = 1
		}

		trend[i] = model.HealthTrendPoint{
			Timestamp: timestamp.Format(time.RFC3339),
			Score:     score,
			Issues:    issues,
		}
	}

	return trend
}

// generateActionableAlerts 生成可操作警报
func (r *resourceService) generateActionableAlerts(issues []model.ResourceIssue) []model.ActionableAlert {
	alerts := make([]model.ActionableAlert, 0, len(issues))

	for i, issue := range issues {
		actions := []string{}
		switch issue.Type {
		case "node":
			actions = []string{"检查节点状态", "重启节点", "检查网络连接"}
		case "pod":
			actions = []string{"查看Pod日志", "重启Pod", "检查资源配置"}
		case "resource":
			actions = []string{"监控资源使用", "扩容集群", "优化应用"}
		}

		alert := model.ActionableAlert{
			ID:          fmt.Sprintf("alert-%d-%d", time.Now().Unix(), i),
			Title:       fmt.Sprintf("%s问题", issue.Resource),
			Description: issue.Description,
			Severity:    issue.Severity,
			Actions:     actions,
			CreatedAt:   issue.Since,
		}
		alerts = append(alerts, alert)
	}

	return alerts
}
