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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ==================== 资源计算相关工具函数 ====================

// CalculateResourceUtilization 计算资源利用率
func CalculateResourceUtilization(requested, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(requested) / float64(total) * 100
}

// FormatResourceSize 格式化资源大小
func FormatResourceSize(bytes int64, unit string) string {
	switch unit {
	case "Mi", "MiB":
		return fmt.Sprintf("%.1fMi", float64(bytes)/(1024*1024))
	case "Gi", "GiB":
		return fmt.Sprintf("%.1fGi", float64(bytes)/(1024*1024*1024))
	case "Ki", "KiB":
		return fmt.Sprintf("%.1fKi", float64(bytes)/1024)
	case "cores":
		return fmt.Sprintf("%.2f cores", float64(bytes)/1000)
	default:
		return fmt.Sprintf("%d%s", bytes, unit)
	}
}

// CalculateHealthRate 计算健康率
func CalculateHealthRate(healthy, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(healthy) / float64(total) * 100
}

// GetResourceStatus 获取资源状态
func GetResourceStatus(current, threshold float64) string {
	switch {
	case current < 50:
		return "low"
	case current < 80:
		return "normal"
	case current < 95:
		return "high"
	default:
		return "critical"
	}
}

// ==================== 资源统计聚合工具函数 ====================

// AggregateClusterResources 聚合集群资源信息
func AggregateClusterResources(stats []*model.ClusterStats) *model.GlobalResourceSummary {
	if len(stats) == 0 {
		return &model.GlobalResourceSummary{}
	}

	summary := &model.GlobalResourceSummary{}
	var totalCPUUtil, totalMemUtil float64
	validClusters := 0

	for _, stat := range stats {
		summary.TotalNodes += stat.NodeStats.TotalNodes
		summary.TotalPods += stat.PodStats.TotalPods

		if stat.ResourceStats.CPUUtilization > 0 {
			totalCPUUtil += stat.ResourceStats.CPUUtilization
			validClusters++
		}
		if stat.ResourceStats.MemoryUtilization > 0 {
			totalMemUtil += stat.ResourceStats.MemoryUtilization
		}
	}

	if validClusters > 0 {
		summary.AvgCPUUtil = totalCPUUtil / float64(validClusters)
		summary.AvgMemUtil = totalMemUtil / float64(validClusters)
	}

	return summary
}

// CalculateClusterEfficiency 计算集群效率
func CalculateClusterEfficiency(stats *model.ClusterStats) float64 {
	if stats == nil {
		return 0.0
	}

	// 基于多个指标计算效率
	nodeEfficiency := CalculateHealthRate(stats.NodeStats.ReadyNodes, stats.NodeStats.TotalNodes)
	podEfficiency := CalculateHealthRate(stats.PodStats.RunningPods, stats.PodStats.TotalPods)
	resourceEfficiency := (stats.ResourceStats.CPUUtilization + stats.ResourceStats.MemoryUtilization) / 2

	// 权重：节点健康 30%, Pod健康 30%, 资源利用 40%
	return (nodeEfficiency*0.3 + podEfficiency*0.3 + resourceEfficiency*0.4)
}

// ==================== 资源预测和趋势分析 ====================

// PredictResourceGrowth 预测资源增长（简化版）
func PredictResourceGrowth(historicalData []float64, days int) *model.ResourcePredict {
	if len(historicalData) < 3 {
		return &model.ResourcePredict{
			Tendency:   "unknown",
			Confidence: 0.0,
			Value:      0.0,
			Suggestion: "数据不足，无法预测",
		}
	}

	// 简单线性趋势分析
	n := len(historicalData)
	recent := historicalData[n-3:]
	trend := (recent[2] - recent[0]) / 2 // 平均变化率

	predictValue := historicalData[n-1] + trend*float64(days)
	if predictValue < 0 {
		predictValue = 0
	}

	tendency := "stable"
	confidence := 0.7
	suggestion := "继续监控"

	if trend > 2 {
		tendency = "increasing"
		confidence = 0.8
		suggestion = "考虑扩容或优化"
	} else if trend < -2 {
		tendency = "decreasing"
		confidence = 0.75
		suggestion = "可考虑资源回收"
	}

	return &model.ResourcePredict{
		PredictDays: days,
		Tendency:    tendency,
		Confidence:  confidence,
		Value:       predictValue,
		Suggestion:  suggestion,
	}
}

// AnalyzeTrendData 分析趋势数据
func AnalyzeTrendData(timestamps []string, values []float64) *model.TrendData {
	if len(timestamps) != len(values) || len(values) == 0 {
		return &model.TrendData{}
	}

	var max, min, sum float64
	max = values[0]
	min = values[0]

	for _, value := range values {
		sum += value
		if value > max {
			max = value
		}
		if value < min {
			min = value
		}
	}

	return &model.TrendData{
		Timestamps: timestamps,
		Values:     values,
		Max:        max,
		Min:        min,
		Avg:        sum / float64(len(values)),
	}
}

// ==================== 资源问题检测 ====================

// DetectResourceIssues 检测资源问题
func DetectResourceIssues(stats *model.ClusterStats) []model.ResourceIssue {
	issues := make([]model.ResourceIssue, 0)

	// 检查节点问题
	if stats.NodeStats.NotReadyNodes > 0 {
		severity := "warning"
		if stats.NodeStats.NotReadyNodes > stats.NodeStats.TotalNodes/2 {
			severity = "critical"
		}

		issues = append(issues, model.ResourceIssue{
			Type:        "node",
			Severity:    severity,
			Resource:    "Node",
			Description: fmt.Sprintf("%d个节点处于NotReady状态", stats.NodeStats.NotReadyNodes),
			Since:       time.Now().Add(-time.Hour).Format(time.RFC3339),
			Suggestion:  "检查节点状态，修复网络或资源问题",
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
			Suggestion:  "检查Pod日志，修复应用问题",
		})
	}

	// 检查资源利用率
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

// CheckResourceThresholds 检查资源阈值
func CheckResourceThresholds(current, warning, critical float64) string {
	switch {
	case current >= critical:
		return "critical"
	case current >= warning:
		return "warning"
	default:
		return "normal"
	}
}

// ==================== 优化建议生成 ====================

// GenerateResourceOptimizationAdvice 生成资源优化建议
func GenerateResourceOptimizationAdvice(stats *model.ClusterStats) []model.UtilizationAdvice {
	advice := make([]model.UtilizationAdvice, 0)

	// CPU优化建议
	if stats.ResourceStats.CPUUtilization > 80 {
		priority := "medium"
		if stats.ResourceStats.CPUUtilization > 90 {
			priority = "high"
		}

		advice = append(advice, model.UtilizationAdvice{
			Type:        "optimization",
			Priority:    priority,
			Resource:    "CPU",
			Target:      "cluster",
			Current:     stats.ResourceStats.CPUUtilization,
			Suggested:   70.0,
			Description: "CPU利用率较高，建议优化或扩容",
			Impact:      "可能影响应用性能",
		})
	} else if stats.ResourceStats.CPUUtilization < 20 {
		advice = append(advice, model.UtilizationAdvice{
			Type:        "rightsizing",
			Priority:    "low",
			Resource:    "CPU",
			Target:      "cluster",
			Current:     stats.ResourceStats.CPUUtilization,
			Suggested:   40.0,
			Description: "CPU利用率较低，可考虑缩减资源",
			Impact:      "可降低成本",
		})
	}

	// 内存优化建议
	if stats.ResourceStats.MemoryUtilization > 85 {
		priority := "medium"
		if stats.ResourceStats.MemoryUtilization > 95 {
			priority = "high"
		}

		advice = append(advice, model.UtilizationAdvice{
			Type:        "scaling",
			Priority:    priority,
			Resource:    "Memory",
			Target:      "cluster",
			Current:     stats.ResourceStats.MemoryUtilization,
			Suggested:   75.0,
			Description: "内存利用率较高，建议增加内存或扩容",
			Impact:      "可能出现OOM错误",
		})
	}

	// 节点优化建议
	if stats.NodeStats.TotalNodes > 0 {
		nodeUtilization := float64(stats.PodStats.TotalPods) / float64(stats.NodeStats.TotalNodes*100) * 100
		if nodeUtilization < 30 {
			advice = append(advice, model.UtilizationAdvice{
				Type:        "consolidation",
				Priority:    "low",
				Resource:    "Node",
				Target:      "cluster",
				Current:     nodeUtilization,
				Suggested:   50.0,
				Description: "节点利用率较低，可考虑合并节点",
				Impact:      "优化资源配置，降低成本",
			})
		}
	}

	return advice
}

// ==================== 数据转换和格式化 ====================

// ConvertToClusterComparison 转换为集群比较数据
func ConvertToClusterComparison(stats []*model.ClusterStats) []model.ClusterComparison {
	comparisons := make([]model.ClusterComparison, 0, len(stats))

	for _, stat := range stats {
		efficiency := CalculateClusterEfficiency(stat)
		comparison := model.ClusterComparison{
			ClusterName: stat.ClusterName,
			CPU:         stat.ResourceStats.TotalCPU,
			Memory:      stat.ResourceStats.TotalMemory,
			Nodes:       stat.NodeStats.TotalNodes,
			Pods:        stat.PodStats.TotalPods,
			Efficiency:  efficiency,
		}
		comparisons = append(comparisons, comparison)
	}

	// 按效率排序
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].Efficiency > comparisons[j].Efficiency
	})

	return comparisons
}

// BuildResourceOverview 构建资源概览
func BuildResourceOverview(stats *model.ClusterStats, clusterName string) *model.ResourceOverview {
	return &model.ResourceOverview{
		ClusterID:   stats.ClusterID,
		ClusterName: clusterName,
		NodeSummary: model.NodeSummary{
			Total:      stats.NodeStats.TotalNodes,
			Ready:      stats.NodeStats.ReadyNodes,
			NotReady:   stats.NodeStats.NotReadyNodes,
			Master:     stats.NodeStats.MasterNodes,
			Worker:     stats.NodeStats.WorkerNodes,
			HealthRate: CalculateHealthRate(stats.NodeStats.ReadyNodes, stats.NodeStats.TotalNodes),
		},
		PodSummary: model.PodSummary{
			Total:      stats.PodStats.TotalPods,
			Running:    stats.PodStats.RunningPods,
			Pending:    stats.PodStats.PendingPods,
			Failed:     stats.PodStats.FailedPods,
			Succeeded:  stats.PodStats.SucceededPods,
			HealthRate: CalculateHealthRate(stats.PodStats.RunningPods, stats.PodStats.TotalPods),
		},
		ResourceSummary: model.ResourceSummary{
			CPU: model.ResourceUsage{
				Total:       stats.ResourceStats.TotalCPU,
				Used:        stats.ResourceStats.UsedCPU,
				Utilization: stats.ResourceStats.CPUUtilization,
			},
			Memory: model.ResourceUsage{
				Total:       stats.ResourceStats.TotalMemory,
				Used:        stats.ResourceStats.UsedMemory,
				Utilization: stats.ResourceStats.MemoryUtilization,
			},
			Storage: model.ResourceUsage{
				Total:       stats.ResourceStats.TotalStorage,
				Used:        stats.ResourceStats.UsedStorage,
				Utilization: stats.ResourceStats.StorageUtilization,
			},
		},
	}
}

// ==================== 图表数据生成工具 ====================

// GeneratePieChartData 生成饼图数据
func GeneratePieChartData(data map[string]float64) model.PieChartData {
	labels := make([]string, 0, len(data))
	values := make([]float64, 0, len(data))

	for label, value := range data {
		labels = append(labels, label)
		values = append(values, value)
	}

	return model.PieChartData{
		Labels: labels,
		Values: values,
		Colors: GenerateColorScheme(len(labels)),
	}
}

// GenerateColorScheme 生成颜色方案
func GenerateColorScheme(count int) []string {
	baseColors := []string{
		"#FF6384", "#36A2EB", "#FFCE56", "#4BC0C0", "#9966FF",
		"#FF9F40", "#FF6384", "#C9CBCF", "#4BC0C0", "#FF6384",
		"#36A2EB", "#FFCE56", "#FF8C00", "#32CD32", "#8A2BE2",
	}

	colors := make([]string, count)
	for i := 0; i < count; i++ {
		colors[i] = baseColors[i%len(baseColors)]
	}

	return colors
}

// ==================== 命名空间和工作负载分析 ====================

// AnalyzeNamespaceResources 分析命名空间资源使用
func AnalyzeNamespaceResources(ctx context.Context, kubeClient kubernetes.Interface) ([]model.NamespaceUsage, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间列表失败: %w", err)
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	// 统计每个命名空间的资源使用
	nsUsage := make(map[string]*model.NamespaceUsage)
	for _, ns := range namespaces.Items {
		nsUsage[ns.Name] = &model.NamespaceUsage{
			Name:     ns.Name,
			IsSystem: IsSystemNamespace(ns.Name),
			Status:   string(ns.Status.Phase),
		}
	}

	// 统计Pod和资源使用
	for _, pod := range pods.Items {
		if usage, exists := nsUsage[pod.Namespace]; exists {
			usage.PodCount++

			// 计算资源使用（基于requests）
			var cpuRequests, memRequests int64
			for _, container := range pod.Spec.Containers {
				if req := container.Resources.Requests; req != nil {
					if cpu := req.Cpu(); !cpu.IsZero() {
						cpuRequests += cpu.MilliValue()
					}
					if mem := req.Memory(); !mem.IsZero() {
						memRequests += mem.Value()
					}
				}
			}

			if cpuRequests > 0 {
				usage.CPUUsage = fmt.Sprintf("%.2f cores", float64(cpuRequests)/1000)
			}
			if memRequests > 0 {
				usage.MemUsage = fmt.Sprintf("%.2f Gi", float64(memRequests)/(1024*1024*1024))
			}
		}
	}

	// 转换为切片并排序
	result := make([]model.NamespaceUsage, 0, len(nsUsage))
	for _, usage := range nsUsage {
		result = append(result, *usage)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].PodCount > result[j].PodCount
	})

	return result, nil
}

// AnalyzeWorkloadDistribution 分析工作负载分布
func AnalyzeWorkloadDistribution(ctx context.Context, kubeClient kubernetes.Interface) (*model.WorkloadDistribution, error) {
	stats := &model.ClusterStats{}

	// 收集工作负载统计
	if err := CollectWorkloadStats(ctx, kubeClient, stats); err != nil {
		return nil, fmt.Errorf("收集工作负载统计失败: %w", err)
	}

	distribution := &model.WorkloadDistribution{
		Deployments:  stats.WorkloadStats.Deployments,
		StatefulSets: stats.WorkloadStats.StatefulSets,
		DaemonSets:   stats.WorkloadStats.DaemonSets,
		Jobs:         stats.WorkloadStats.Jobs,
		CronJobs:     stats.WorkloadStats.CronJobs,
		Services:     stats.WorkloadStats.Services,
		ConfigMaps:   stats.WorkloadStats.ConfigMaps,
		Secrets:      stats.WorkloadStats.Secrets,
		Ingresses:    stats.WorkloadStats.Ingresses,
	}

	return distribution, nil
}

// ==================== 健康评估工具 ====================

// AssessClusterHealth 评估集群健康状况
func AssessClusterHealth(stats *model.ClusterStats) model.HealthScore {
	score := 100
	factors := make([]string, 0)

	// 节点健康检查
	if stats.NodeStats.TotalNodes > 0 {
		nodeHealthRate := CalculateHealthRate(stats.NodeStats.ReadyNodes, stats.NodeStats.TotalNodes)
		if nodeHealthRate < 90 {
			deduction := int((90 - nodeHealthRate) / 3)
			score -= deduction
			factors = append(factors, "节点健康状态")
		}
	}

	// Pod健康检查
	if stats.PodStats.TotalPods > 0 {
		podHealthRate := CalculateHealthRate(stats.PodStats.RunningPods, stats.PodStats.TotalPods)
		if podHealthRate < 80 {
			deduction := int((80 - podHealthRate) / 2)
			score -= deduction
			factors = append(factors, "Pod健康状态")
		}
	}

	// 资源利用率检查
	if stats.ResourceStats.CPUUtilization > 90 {
		score -= 15
		factors = append(factors, "CPU利用率过高")
	} else if stats.ResourceStats.CPUUtilization < 10 {
		score -= 5
		factors = append(factors, "CPU利用率过低")
	}

	if stats.ResourceStats.MemoryUtilization > 90 {
		score -= 15
		factors = append(factors, "内存利用率过高")
	}

	// 事件检查
	if stats.EventStats.WarningEvents > 10 {
		score -= 10
		factors = append(factors, "警告事件过多")
	}

	if score < 0 {
		score = 0
	}

	// 确定健康级别
	level := "excellent"
	description := "集群运行状态优秀"

	switch {
	case score < 50:
		level = "critical"
		description = "集群存在严重问题，需要立即处理"
	case score < 70:
		level = "warning"
		description = "集群需要关注和优化"
	case score < 90:
		level = "good"
		description = "集群运行状态良好"
	}

	return model.HealthScore{
		Score:       score,
		Level:       level,
		Description: description,
		Factors:     factors,
	}
}

// ==================== 字符串和验证工具 ====================

// SanitizeClusterName 清理集群名称
func SanitizeClusterName(name string) string {
	// 移除特殊字符，只保留字母数字和连字符
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, name)

	// 确保长度不超过63字符（Kubernetes限制）
	if len(cleaned) > 63 {
		cleaned = cleaned[:63]
	}

	return strings.ToLower(cleaned)
}

// ValidateResourceRequest 验证资源请求
func ValidateResourceRequest(cpu, memory, storage string) error {
	if cpu != "" {
		if _, err := parseResourceQuantity(cpu); err != nil {
			return fmt.Errorf("无效的CPU请求量: %s", cpu)
		}
	}

	if memory != "" {
		if _, err := parseResourceQuantity(memory); err != nil {
			return fmt.Errorf("无效的内存请求量: %s", memory)
		}
	}

	if storage != "" {
		if _, err := parseResourceQuantity(storage); err != nil {
			return fmt.Errorf("无效的存储请求量: %s", storage)
		}
	}

	return nil
}

// parseResourceQuantity 解析资源数量（简化实现）
func parseResourceQuantity(quantity string) (int64, error) {
	// 简化的资源解析，实际应该使用k8s.io/apimachinery/pkg/api/resource
	if quantity == "" {
		return 0, nil
	}

	// 这里应该实现完整的Kubernetes资源量解析
	// 为了简化，这里只做基本验证
	if strings.Contains(quantity, "..") || strings.HasPrefix(quantity, "-") {
		return 0, fmt.Errorf("无效的资源格式")
	}

	return 0, nil
}
