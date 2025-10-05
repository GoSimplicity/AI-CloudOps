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

package manager

// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"time"

// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/model"
// 	"go.uber.org/zap"
// 	"k8s.io/client-go/kubernetes"
// )

// type ResourceManager interface {
// 	// 资源概览和统计
// 	GetClusterResourceOverview(ctx context.Context, clusterID int) (*model.ClusterStats, error)
// 	GetResourceDistribution(ctx context.Context, clusterID int) (*model.ResourceDistribution, error)
// 	GetResourceUtilization(ctx context.Context, clusterID int) (*model.ResourceUtilization, error)

// 	// 资源健康和监控
// 	GetClusterHealth(ctx context.Context, clusterID int) (*model.ClusterHealth, error)
// 	GetResourceIssues(ctx context.Context, clusterID int) ([]model.ResourceIssue, error)

// 	CompareMultiClusterResources(ctx context.Context, clusterIDs []int) (*model.ResourceComparisonChart, error)
// 	GetAllClustersOverview(ctx context.Context, clusterIDs []int) (*model.AllClustersSummary, error)

// 	// 资源预测和建议
// 	PredictResourceTrend(ctx context.Context, clusterID int, period string) (*model.ResourceTrend, error)
// 	GenerateOptimizationAdvice(ctx context.Context, clusterID int) ([]model.UtilizationAdvice, error)

// 	RefreshResourceCache(ctx context.Context, clusterID int) error
// 	ClearResourceCache(ctx context.Context, clusterID int) error
// }

// type resourceManager struct {
// 	clientFactory client.K8sClient
// 	logger        *zap.Logger
// 	cache         *resourceCache
// }

// // resourceCache 资源缓存结构
// type resourceCache struct {
// 	mu           sync.RWMutex
// 	clusterStats map[int]*cachedStats
// 	cacheTimeout time.Duration
// }

// type cachedStats struct {
// 	stats     *model.ClusterStats
// 	timestamp time.Time
// }

// func NewResourceManager(clientFactory client.K8sClient, logger *zap.Logger) ResourceManager {
// 	return &resourceManager{
// 		clientFactory: clientFactory,
// 		logger:        logger,
// 		cache: &resourceCache{
// 			clusterStats: make(map[int]*cachedStats),
// 			cacheTimeout: 5 * time.Minute, // 5分钟缓存
// 		},
// 	}
// }

// // getKubeClient 获取Kubernetes客户端
// func (r *resourceManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
// 	kubeClient, err := r.clientFactory.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
// 	}
// 	return kubeClient, nil
// }

// // GetClusterResourceOverview
// func (r *resourceManager) GetClusterResourceOverview(ctx context.Context, clusterID int) (*model.ClusterStats, error) {
// 	// 尝试从缓存获取
// 	if stats := r.getCachedStats(clusterID); stats != nil {
// 		r.logger.Debug("从缓存获取集群统计信息", zap.Int("clusterID", clusterID))
// 		return stats, nil
// 	}

// 	// 获取k8s客户端
// 	kubeClient, err := r.getKubeClient(clusterID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 收集统计信息
// 	stats := &model.ClusterStats{
// 		ClusterID:      clusterID,
// 		LastUpdateTime: time.Now().Format(time.DateTime),
// 	}

// 	// 并发收集各种统计信息
// 	errChan := make(chan error, 8)
// 	var wg sync.WaitGroup

// 	wg.Add(8)

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectNodeStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectPodStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectNamespaceStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectWorkloadStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectResourceStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectStorageStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectNetworkStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		errChan <- utils.CollectEventStats(ctx, kubeClient, stats)
// 	}()

// 	wg.Wait()
// 	close(errChan)

//
// 	var errors []error
// 	for err := range errChan {
// 		if err != nil {
// 			errors = append(errors, err)
// 		}
// 	}

// 	if len(errors) > 0 {
// 		r.logger.Warn("部分统计信息收集失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.Int("errorCount", len(errors)))
// 		// 继续返回部分数据
// 	}

// 	// 缓存结果
// 	r.setCachedStats(clusterID, stats)

// 	r.logger.Info("成功收集集群资源概览",
// 		zap.Int("clusterID", clusterID),
// 		zap.Int("nodes", stats.NodeStats.TotalNodes),
// 		zap.Int("pods", stats.PodStats.TotalPods))

// 	return stats, nil
// }

// // GetResourceDistribution
// func (r *resourceManager) GetResourceDistribution(ctx context.Context, clusterID int) (*model.ResourceDistribution, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.getKubeClient(clusterID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	distribution := &model.ResourceDistribution{
// 		ClusterID: clusterID,
// 	}

// 	// 并发获取各种分布信息
// 	var wg sync.WaitGroup
// 	errChan := make(chan error, 3)

// 	// 获取节点分布
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		nodeDistrib, err := r.getNodeResourceDistribution(ctx, kubeClient)
// 		if err != nil {
// 			errChan <- fmt.Errorf("获取节点分布失败: %w", err)
// 			return
// 		}
// 		distribution.NodeDistribution = nodeDistrib
// 	}()

// 	// 获取命名空间分布
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		nsDistrib, err := r.getNamespaceResourceDistribution(ctx, kubeClient)
// 		if err != nil {
// 			errChan <- fmt.Errorf("获取命名空间分布失败: %w", err)
// 			return
// 		}
// 		distribution.NSDistribution = nsDistrib
// 	}()

// 	// 获取工作负载分布
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		workloadDistrib, err := r.getWorkloadDistribution(ctx, kubeClient)
// 		if err != nil {
// 			errChan <- fmt.Errorf("获取工作负载分布失败: %w", err)
// 			return
// 		}
// 		distribution.WorkloadDistrib = workloadDistrib
// 	}()

// 	wg.Wait()
// 	close(errChan)

//
// 	var errors []error
// 	for err := range errChan {
// 		if err != nil {
// 			errors = append(errors, err)
// 		}
// 	}

// 	if len(errors) > 0 {
// 		r.logger.Warn("部分分布信息获取失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.Errors("errors", errors))
// 	}

// 	// 生成资源分配图表
// 	distribution.ResourceAllocation = r.generateResourceAllocationChart(
// 		distribution.NodeDistribution,
// 		distribution.NSDistribution,
// 	)

// 	r.logger.Info("成功获取资源分布信息", zap.Int("clusterID", clusterID))
// 	return distribution, nil
// }

// // GetResourceUtilization
// func (r *resourceManager) GetResourceUtilization(ctx context.Context, clusterID int) (*model.ResourceUtilization, error) {
// 	// 获取基本统计信息
// 	stats, err := r.GetClusterResourceOverview(ctx, clusterID)
// 	if err != nil {
// 		return nil, fmt.Errorf("获取集群统计失败: %w", err)
// 	}

// 	utilization := &model.ResourceUtilization{
// 		ClusterID: clusterID,
// 		OverallUtil: model.UtilizationSummary{
// 			CPU:     stats.ResourceStats.CPUUtilization,
// 			Memory:  stats.ResourceStats.MemoryUtilization,
// 			Storage: stats.ResourceStats.StorageUtilization,
// 			Network: 0.0, // 需要metrics-server支持
// 			Overall: (stats.ResourceStats.CPUUtilization + stats.ResourceStats.MemoryUtilization) / 2,
// 		},
// 	}

// 	// 获取k8s客户端
// 	kubeClient, err := r.getKubeClient(clusterID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 并发获取详细利用率信息
// 	var wg sync.WaitGroup
// 	errChan := make(chan error, 2)

// 	// 获取节点利用率
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		nodeUtils, err := r.getNodeUtilizations(ctx, kubeClient)
// 		if err != nil {
// 			errChan <- fmt.Errorf("获取节点利用率失败: %w", err)
// 			return
// 		}
// 		utilization.NodeUtils = nodeUtils
// 	}()

// 	// 获取命名空间利用率
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		nsUtils, err := r.getNamespaceUtilizations(ctx, kubeClient)
// 		if err != nil {
// 			errChan <- fmt.Errorf("获取命名空间利用率失败: %w", err)
// 			return
// 		}
// 		utilization.NSUtils = nsUtils
// 	}()

// 	wg.Wait()
// 	close(errChan)

//
// 	for err := range errChan {
// 		if err != nil {
// 			r.logger.Warn("获取利用率信息时出错", zap.Error(err))
// 		}
// 	}

// 	// 生成利用率图表和建议
// 	utilization.UtilChart = r.generateUtilizationChart(utilization.NodeUtils, utilization.NSUtils)
// 	utilization.Recommendations = []model.UtilizationAdvice{} // 简化实现

// 	r.logger.Info("成功获取资源利用率",
// 		zap.Int("clusterID", clusterID),
// 		zap.Float64("overallCPU", utilization.OverallUtil.CPU),
// 		zap.Float64("overallMemory", utilization.OverallUtil.Memory))

// 	return utilization, nil
// }

// // GetClusterHealth
// func (r *resourceManager) GetClusterHealth(ctx context.Context, clusterID int) (*model.ClusterHealth, error) {
// 	// 获取基本统计信息
// 	stats, err := r.GetClusterResourceOverview(ctx, clusterID)
// 	if err != nil {
// 		return nil, fmt.Errorf("获取集群统计失败: %w", err)
// 	}

// 	// 计算健康评分
// 	healthScore := r.calculateHealthScore(stats)

// 	// 获取组件健康状态
// 	kubeClient, err := r.getKubeClient(clusterID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	components, _, err := utils.GetComponentStatuses(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取组件状态失败", zap.Error(err))
// 		components = []*model.ComponentHealthStatus{}
// 	}

// 	// 识别问题
// 	issues := r.identifyHealthIssues(stats)

//
// 	componentStatuses := make([]model.ComponentHealthStatus, len(components))
// 	for i, comp := range components {
// 		componentStatuses[i] = *comp
// 	}

// 	health := &model.ClusterHealth{
// 		OverallStatus: r.determineOverallStatus(healthScore),
// 		Score:         healthScore,
// 		Components:    componentStatuses,
// 		Issues:        issues,
// 	}

// 	r.logger.Info("成功获取集群健康状态",
// 		zap.Int("clusterID", clusterID),
// 		zap.Int("healthScore", healthScore),
// 		zap.String("status", health.OverallStatus))

// 	return health, nil
// }

// // GetResourceIssues
// func (r *resourceManager) GetResourceIssues(ctx context.Context, clusterID int) ([]model.ResourceIssue, error) {
// 	// 获取统计信息
// 	stats, err := r.GetClusterResourceOverview(ctx, clusterID)
// 	if err != nil {
// 		return nil, fmt.Errorf("获取集群统计失败: %w", err)
// 	}

// 	issues := make([]model.ResourceIssue, 0)

//
// 	if stats.NodeStats.NotReadyNodes > 0 {
// 		issues = append(issues, model.ResourceIssue{
// 			Type:        "node",
// 			Severity:    "critical",
// 			Resource:    "Node",
// 			Description: fmt.Sprintf("%d个节点处于NotReady状态", stats.NodeStats.NotReadyNodes),
// 			Since:       time.Now().Add(-time.Hour).Format(time.RFC3339),
// 			Suggestion:  "检查节点状态并修复网络或资源问题",
// 		})
// 	}

//
// 	if stats.PodStats.FailedPods > 0 {
// 		issues = append(issues, model.ResourceIssue{
// 			Type:        "pod",
// 			Severity:    "warning",
// 			Resource:    "Pod",
// 			Description: fmt.Sprintf("%d个Pod处于失败状态", stats.PodStats.FailedPods),
// 			Since:       time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
// 			Suggestion:  "检查Pod日志并修复应用问题",
// 		})
// 	}

//
// 	if stats.ResourceStats.CPUUtilization > 90 {
// 		issues = append(issues, model.ResourceIssue{
// 			Type:        "resource",
// 			Severity:    "critical",
// 			Resource:    "CPU",
// 			Description: fmt.Sprintf("CPU利用率过高: %.1f%%", stats.ResourceStats.CPUUtilization),
// 			Since:       time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
// 			Suggestion:  "考虑扩容或优化高CPU使用的应用",
// 		})
// 	}

// 	if stats.ResourceStats.MemoryUtilization > 90 {
// 		issues = append(issues, model.ResourceIssue{
// 			Type:        "resource",
// 			Severity:    "critical",
// 			Resource:    "Memory",
// 			Description: fmt.Sprintf("内存利用率过高: %.1f%%", stats.ResourceStats.MemoryUtilization),
// 			Since:       time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
// 			Suggestion:  "考虑增加内存或扩容节点",
// 		})
// 	}

// 	r.logger.Info("成功识别资源问题",
// 		zap.Int("clusterID", clusterID),
// 		zap.Int("issueCount", len(issues)))

// 	return issues, nil
// }

// // CompareMultiClusterResources 对比多集群资源
// func (r *resourceManager) CompareMultiClusterResources(ctx context.Context, clusterIDs []int) (*model.ResourceComparisonChart, error) {
// 	if len(clusterIDs) < 2 {
// 		return nil, fmt.Errorf("至少需要2个集群进行对比")
// 	}

// 	comparison := &model.ResourceComparisonChart{
// 		ClusterNames: make([]string, 0, len(clusterIDs)),
// 		CPUData:      make([]float64, 0, len(clusterIDs)),
// 		MemoryData:   make([]float64, 0, len(clusterIDs)),
// 		PodData:      make([]float64, 0, len(clusterIDs)),
// 		Detailed:     make([]model.ClusterComparison, 0, len(clusterIDs)),
// 	}

// 	// 并发获取各集群的统计信息
// 	var wg sync.WaitGroup
// 	mu := sync.Mutex{}

// 	for _, clusterID := range clusterIDs {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()

// 			stats, err := r.GetClusterResourceOverview(ctx, id)
// 			if err != nil {
// 				r.logger.Warn("获取集群统计失败", zap.Int("clusterID", id), zap.Error(err))
// 				return
// 			}

// 			// 计算效率得分（简化）
// 			efficiency := (stats.ResourceStats.CPUUtilization + stats.ResourceStats.MemoryUtilization) / 200 * 100

// 			mu.Lock()
// 			comparison.ClusterNames = append(comparison.ClusterNames, stats.ClusterName)
// 			comparison.CPUData = append(comparison.CPUData, stats.ResourceStats.CPUUtilization)
// 			comparison.MemoryData = append(comparison.MemoryData, stats.ResourceStats.MemoryUtilization)
// 			comparison.PodData = append(comparison.PodData, float64(stats.PodStats.TotalPods))

// 			comparison.Detailed = append(comparison.Detailed, model.ClusterComparison{
// 				ClusterName: stats.ClusterName,
// 				CPU:         stats.ResourceStats.TotalCPU,
// 				Memory:      stats.ResourceStats.TotalMemory,
// 				Nodes:       stats.NodeStats.TotalNodes,
// 				Pods:        stats.PodStats.TotalPods,
// 				Efficiency:  efficiency,
// 			})
// 			mu.Unlock()
// 		}(clusterID)
// 	}

// 	wg.Wait()

// 	r.logger.Info("成功对比多集群资源",
// 		zap.Int("clusterCount", len(comparison.ClusterNames)))

// 	return comparison, nil
// }

// // GetAllClustersOverview
// func (r *resourceManager) GetAllClustersOverview(ctx context.Context, clusterIDs []int) (*model.AllClustersSummary, error) {
// 	summary := &model.AllClustersSummary{
// 		TotalClusters:    len(clusterIDs),
// 		TotalResources:   model.GlobalResourceSummary{},
// 		ClustersOverview: make([]model.ClusterBriefSummary, 0, len(clusterIDs)),
// 		AlertsSummary: model.GlobalAlertsSummary{
// 			AlertsByCluster: make(map[string]int),
// 		},
// 	}

// 	var wg sync.WaitGroup
// 	mu := sync.Mutex{}
// 	var healthyCount, unhealthyCount int
// 	var totalNodes, totalPods int
// 	var totalCPUUtil, totalMemUtil float64
// 	validClusters := 0

// 	for _, clusterID := range clusterIDs {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()

// 			stats, err := r.GetClusterResourceOverview(ctx, id)
// 			if err != nil {
// 				r.logger.Warn("获取集群统计失败", zap.Int("clusterID", id))
// 				mu.Lock()
// 				unhealthyCount++
// 				mu.Unlock()
// 				return
// 			}

// 			// 计算健康状态
// 			healthScore := r.calculateHealthScore(stats)

// 			mu.Lock()
// 			defer mu.Unlock()

// 			if healthScore > 70 {
// 				healthyCount++
// 			} else {
// 				unhealthyCount++
// 			}

// 			// 累计资源
// 			totalNodes += stats.NodeStats.TotalNodes
// 			totalPods += stats.PodStats.TotalPods
// 			totalCPUUtil += stats.ResourceStats.CPUUtilization
// 			totalMemUtil += stats.ResourceStats.MemoryUtilization
// 			validClusters++

// 			// 添加到概览
// 			summary.ClustersOverview = append(summary.ClustersOverview, model.ClusterBriefSummary{
// 				ClusterID:   id,
// 				ClusterName: stats.ClusterName,
// 				Status:      "Running", // 简化
// 				HealthScore: healthScore,
// 				NodeCount:   stats.NodeStats.TotalNodes,
// 				PodCount:    stats.PodStats.TotalPods,
// 				CPUUtil:     stats.ResourceStats.CPUUtilization,
// 				MemoryUtil:  stats.ResourceStats.MemoryUtilization,
// 				Issues:      stats.EventStats.WarningEvents,
// 			})

// 			// 统计警报
// 			summary.AlertsSummary.AlertsByCluster[stats.ClusterName] = stats.EventStats.WarningEvents
// 			summary.AlertsSummary.TotalAlerts += stats.EventStats.WarningEvents
// 		}(clusterID)
// 	}

// 	wg.Wait()

// 	// 设置汇总信息
// 	summary.HealthyClusters = healthyCount
// 	summary.UnhealthyClusters = unhealthyCount
// 	summary.TotalResources.TotalNodes = totalNodes
// 	summary.TotalResources.TotalPods = totalPods

// 	if validClusters > 0 {
// 		summary.TotalResources.AvgCPUUtil = totalCPUUtil / float64(validClusters)
// 		summary.TotalResources.AvgMemUtil = totalMemUtil / float64(validClusters)
// 	}

// 	r.logger.Info("成功获取所有集群概览",
// 		zap.Int("totalClusters", summary.TotalClusters),
// 		zap.Int("healthyClusters", summary.HealthyClusters))

// 	return summary, nil
// }

// // PredictResourceTrend 预测资源趋势（模拟实现）
// func (r *resourceManager) PredictResourceTrend(ctx context.Context, clusterID int, period string) (*model.ResourceTrend, error) {
//
// 	duration, err := r.parsePeriod(period)
// 	if err != nil {
// 		return nil, fmt.Errorf("无效的时间周期: %w", err)
// 	}

//
// 	_, err = r.getKubeClient(clusterID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 生成模拟趋势数据
// 	trend := &model.ResourceTrend{
// 		ClusterID: clusterID,
// 		Period:    period,
// 		TimeRange: model.TimeRange{
// 			Start: time.Now().Add(-duration),
// 			End:   time.Now(),
// 		},
// 		CPUTrend:    r.generateTrendData("CPU", duration),
// 		MemoryTrend: r.generateTrendData("Memory", duration),
// 		PodTrend:    r.generateTrendData("Pod", duration),
// 		NodeTrend:   r.generateTrendData("Node", duration),
// 		Predictions: r.generateResourcePredictions(),
// 	}

// 	r.logger.Info("成功预测资源趋势",
// 		zap.Int("clusterID", clusterID),
// 		zap.String("period", period))

// 	return trend, nil
// }

// // GenerateOptimizationAdvice 生成优化建议
// func (r *resourceManager) GenerateOptimizationAdvice(ctx context.Context, clusterID int) ([]model.UtilizationAdvice, error) {
// 	// 获取利用率信息
// 	utilization, err := r.GetResourceUtilization(ctx, clusterID)
// 	if err != nil {
// 		return nil, fmt.Errorf("获取利用率信息失败: %w", err)
// 	}

// 	advice := make([]model.UtilizationAdvice, 0)

// 	// CPU利用率建议
// 	if utilization.OverallUtil.CPU > 80 {
// 		advice = append(advice, model.UtilizationAdvice{
// 			Type:        "optimization",
// 			Priority:    "high",
// 			Resource:    "CPU",
// 			Target:      "cluster",
// 			Current:     utilization.OverallUtil.CPU,
// 			Suggested:   70.0,
// 			Description: "集群CPU利用率过高，建议扩容或优化应用",
// 			Impact:      "性能影响严重",
// 		})
// 	}

// 	// 内存利用率建议
// 	if utilization.OverallUtil.Memory > 85 {
// 		advice = append(advice, model.UtilizationAdvice{
// 			Type:        "scaling",
// 			Priority:    "high",
// 			Resource:    "Memory",
// 			Target:      "cluster",
// 			Current:     utilization.OverallUtil.Memory,
// 			Suggested:   75.0,
// 			Description: "集群内存利用率过高，建议增加内存或扩容节点",
// 			Impact:      "可能出现OOM",
// 		})
// 	}

// 	// 节点效率建议
// 	for _, node := range utilization.NodeUtils {
// 		if node.Efficiency == "低效" {
// 			advice = append(advice, model.UtilizationAdvice{
// 				Type:        "rebalancing",
// 				Priority:    "medium",
// 				Resource:    "Pod",
// 				Target:      node.NodeName,
// 				Current:     float64(node.PodCount),
// 				Suggested:   20.0,
// 				Description: fmt.Sprintf("节点 %s Pod数量较少，资源利用率低", node.NodeName),
// 				Impact:      "资源浪费",
// 			})
// 		}
// 	}

// 	r.logger.Info("成功生成优化建议",
// 		zap.Int("clusterID", clusterID),
// 		zap.Int("adviceCount", len(advice)))

// 	return advice, nil
// }

// // RefreshResourceCache 刷新资源缓存
// func (r *resourceManager) RefreshResourceCache(ctx context.Context, clusterID int) error {
// 	// 清除旧缓存
// 	r.clearCachedStats(clusterID)

// 	// 重新收集统计信息
// 	_, err := r.GetClusterResourceOverview(ctx, clusterID)
// 	if err != nil {
// 		return fmt.Errorf("刷新缓存失败: %w", err)
// 	}

// 	r.logger.Info("成功刷新资源缓存", zap.Int("clusterID", clusterID))
// 	return nil
// }

// // ClearResourceCache 清除资源缓存
// func (r *resourceManager) ClearResourceCache(ctx context.Context, clusterID int) error {
// 	r.clearCachedStats(clusterID)
// 	r.logger.Info("成功清除资源缓存", zap.Int("clusterID", clusterID))
// 	return nil
// }
