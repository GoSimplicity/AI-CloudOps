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

// import (
// 	"context"
// 	"fmt"
// 	"sort"
// 	"strconv"
// 	"time"

// 	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/model"
// 	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
// 	"go.uber.org/zap"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/client-go/kubernetes"
// )

// type ResourceService interface {
// 	// 基础资源概览功能
// 	GetResourceOverview(ctx context.Context, clusterID int) (*model.ResourceOverview, error)
// 	GetResourceStatistics(ctx context.Context, clusterID int) (*model.ClusterStats, error)
// 	GetResourceDistribution(ctx context.Context, clusterID int) (*model.ResourceDistribution, error)

// 	// 资源分析功能
// 	GetResourceTrend(ctx context.Context, req *model.ResourceTrendReq) (*model.ResourceTrend, error)
// 	GetResourceUtilization(ctx context.Context, clusterID int) (*model.ResourceUtilization, error)
// 	GetResourceHealth(ctx context.Context, clusterID int) (*model.ResourceHealth, error)

// 	// 工作负载和命名空间功能
// 	GetWorkloadDistribution(ctx context.Context, clusterID int) (*model.WorkloadDistribution, error)
// 	GetNamespaceResources(ctx context.Context, clusterID int) ([]*model.NamespaceUsage, error)

// 	// 存储和网络功能
// 	GetStorageOverview(ctx context.Context, clusterID int) (*model.StorageStats, error)
// 	GetNetworkOverview(ctx context.Context, clusterID int) (*model.NetworkStats, error)

// 	// 多集群功能
// 	CompareClusterResources(ctx context.Context, clusterIDs []int) (*model.ResourceComparisonChart, error)
// 	GetAllClustersSummary(ctx context.Context) (*model.AllClustersSummary, error)
// }

// type resourceService struct {
// 	dao    dao.ClusterDAO
// 	client client.K8sClient
// 	logger *zap.Logger
// }

// func NewResourceService(dao dao.ClusterDAO, client client.K8sClient, logger *zap.Logger) ResourceService {
// 	return &resourceService{
// 		dao:    dao,
// 		client: client,
// 		logger: logger,
// 	}
// }

// // GetResourceOverview 获取集群资源概览
// func (r *resourceService) GetResourceOverview(ctx context.Context, clusterID int) (*model.ResourceOverview, error) {
// 	// 获取集群信息
// 	cluster, err := r.dao.GetClusterByID(ctx, clusterID)
// 	if err != nil {
// 		r.logger.Error("获取集群信息失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "集群不存在")
// 	}

// 	if cluster == nil {
// 		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "集群不存在")
// 	}

// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	// 收集统计信息
// 	stats := &model.ClusterStats{
// 		ClusterID:   clusterID,
// 		ClusterName: cluster.Name,
// 	}

// 	// 并发收集各种统计信息
// 	errChan := make(chan error, 6)

// 	go func() {
// 		errChan <- utils.CollectNodeStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectPodStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectResourceStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectNamespaceStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectEventStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		_, _, err := utils.GetComponentStatuses(ctx, kubeClient)
// 		errChan <- err
// 	}()

// 	// 等待所有goroutine完成
// 	var errors []error
// 	for i := 0; i < 6; i++ {
// 		if err := <-errChan; err != nil {
// 			errors = append(errors, err)
// 		}
// 	}

// 	if len(errors) > 0 {
// 		r.logger.Warn("部分统计信息收集失败", zap.Int("errorCount", len(errors)))
// 		// 不返回错误，使用部分数据
// 	}

// 	// 转换为ResourceOverview格式
// 	overview := &model.ResourceOverview{
// 		ClusterID:   clusterID,
// 		ClusterName: cluster.Name,
// 		Status:      strconv.Itoa(int(cluster.Status)),
// 		Version:     cluster.Version,
// 		NodeSummary: model.NodeSummary{
// 			Total:      stats.NodeStats.TotalNodes,
// 			Ready:      stats.NodeStats.ReadyNodes,
// 			NotReady:   stats.NodeStats.NotReadyNodes,
// 			Master:     stats.NodeStats.MasterNodes,
// 			Worker:     stats.NodeStats.WorkerNodes,
// 			HealthRate: calculateHealthRate(stats.NodeStats.ReadyNodes, stats.NodeStats.TotalNodes),
// 		},
// 		PodSummary: model.PodSummary{
// 			Total:      stats.PodStats.TotalPods,
// 			Running:    stats.PodStats.RunningPods,
// 			Pending:    stats.PodStats.PendingPods,
// 			Failed:     stats.PodStats.FailedPods,
// 			Succeeded:  stats.PodStats.SucceededPods,
// 			HealthRate: calculateHealthRate(stats.PodStats.RunningPods, stats.PodStats.TotalPods),
// 		},
// 		ResourceSummary: model.ResourceSummary{
// 			CPU: model.ResourceUsage{
// 				Total:       stats.ResourceStats.TotalCPU,
// 				Used:        stats.ResourceStats.UsedCPU,
// 				Utilization: stats.ResourceStats.CPUUtilization,
// 			},
// 			Memory: model.ResourceUsage{
// 				Total:       stats.ResourceStats.TotalMemory,
// 				Used:        stats.ResourceStats.UsedMemory,
// 				Utilization: stats.ResourceStats.MemoryUtilization,
// 			},
// 			Storage: model.ResourceUsage{
// 				Total:       stats.ResourceStats.TotalStorage,
// 				Used:        stats.ResourceStats.UsedStorage,
// 				Utilization: stats.ResourceStats.StorageUtilization,
// 			},
// 		},
// 		HealthStatus: model.ClusterHealth{
// 			OverallStatus: r.calculateOverallHealthStatus(stats),
// 			Score:         r.calculateHealthScore(stats).Score,
// 		},
// 	}

// 	// 获取Top命名空间
// 	overview.TopNamespaces = r.buildTopNamespaces(stats.NamespaceStats.TopNamespaces, stats)

// 	// 获取最近事件
// 	overview.RecentEvents = r.buildRecentEvents(ctx, kubeClient, 10)

// 	r.logger.Info("成功获取集群资源概览",
// 		zap.Int("clusterID", clusterID),
// 		zap.String("clusterName", cluster.Name))

// 	return overview, nil
// }

// // GetResourceStatistics 获取资源统计信息
// func (r *resourceService) GetResourceStatistics(ctx context.Context, clusterID int) (*model.ClusterStats, error) {
// 	// 获取集群信息
// 	cluster, err := r.dao.GetClusterByID(ctx, clusterID)
// 	if err != nil || cluster == nil {
// 		r.logger.Error("获取集群信息失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "集群不存在")
// 	}

// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	// 收集完整的统计信息
// 	stats := &model.ClusterStats{
// 		ClusterID:      clusterID,
// 		ClusterName:    cluster.Name,
// 		LastUpdateTime: time.Now().Format(time.DateTime),
// 	}

// 	// 并发收集所有统计信息
// 	errChan := make(chan error, 8)

// 	go func() {
// 		errChan <- utils.CollectNodeStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectPodStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectNamespaceStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectWorkloadStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectResourceStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectStorageStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectNetworkStats(ctx, kubeClient, stats)
// 	}()

// 	go func() {
// 		errChan <- utils.CollectEventStats(ctx, kubeClient, stats)
// 	}()

// 	// 等待所有收集完成
// 	var errors []error
// 	for i := 0; i < 8; i++ {
// 		if err := <-errChan; err != nil {
// 			errors = append(errors, err)
// 			r.logger.Warn("统计信息收集出错", zap.Error(err))
// 		}
// 	}

// 	if len(errors) > 0 {
// 		r.logger.Warn("部分统计信息收集失败", zap.Int("errorCount", len(errors)))
// 		// 继续返回部分数据
// 	}

// 	r.logger.Info("成功获取集群资源统计",
// 		zap.Int("clusterID", clusterID),
// 		zap.String("clusterName", cluster.Name))

// 	return stats, nil
// }

// // GetResourceDistribution 获取资源分布信息
// func (r *resourceService) GetResourceDistribution(ctx context.Context, clusterID int) (*model.ResourceDistribution, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	distribution := &model.ResourceDistribution{
// 		ClusterID: clusterID,
// 	}

// 	// 获取节点分布
// 	nodeDistrib, err := r.getNodeDistribution(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取节点分布失败", zap.Error(err))
// 	} else {
// 		distribution.NodeDistribution = nodeDistrib
// 	}

// 	// 获取命名空间分布
// 	nsDistrib, err := r.getNamespaceDistribution(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取命名空间分布失败", zap.Error(err))
// 	} else {
// 		distribution.NSDistribution = nsDistrib
// 	}

// 	// 获取工作负载分布
// 	workloadDistrib, err := r.getDetailedWorkloadDistribution(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取工作负载分布失败", zap.Error(err))
// 	} else {
// 		distribution.WorkloadDistrib = *workloadDistrib
// 	}

// 	// 生成资源分配图表
// 	distribution.ResourceAllocation = r.generateResourceAllocationChart(nodeDistrib, nsDistrib)

// 	r.logger.Info("成功获取资源分布信息", zap.Int("clusterID", clusterID))
// 	return distribution, nil
// }

// // GetResourceTrend 获取资源趋势（模拟实现）
// func (r *resourceService) GetResourceTrend(ctx context.Context, req *model.ResourceTrendReq) (*model.ResourceTrend, error) {
// 	// 验证时间周期
// 	if req.Period == "" {
// 		req.Period = "24h"
// 	}

// 	// 解析时间周期
// 	duration, err := r.parsePeriod(req.Period)
// 	if err != nil {
// 		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "无效的时间周期参数")
// 	}

// 	// 获取k8s客户端验证集群存在
// 	_, err = r.client.GetKubeClient(req.ClusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", req.ClusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	// 模拟生成趋势数据
// 	trend := &model.ResourceTrend{
// 		ClusterID: req.ClusterID,
// 		Period:    req.Period,
// 		TimeRange: model.TimeRange{
// 			Start: time.Now().Add(-duration),
// 			End:   time.Now(),
// 		},
// 		CPUTrend:    r.generateMockTrendData("CPU", duration),
// 		MemoryTrend: r.generateMockTrendData("Memory", duration),
// 		PodTrend:    r.generateMockTrendData("Pod", duration),
// 		NodeTrend:   r.generateMockTrendData("Node", duration),
// 	}

// 	// 生成预测数据
// 	trend.Predictions = r.generateResourcePredictions()

// 	r.logger.Info("成功获取资源趋势", zap.Int("clusterID", req.ClusterID), zap.String("period", req.Period))
// 	return trend, nil
// }

// // GetResourceUtilization 获取资源利用率
// func (r *resourceService) GetResourceUtilization(ctx context.Context, clusterID int) (*model.ResourceUtilization, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	utilization := &model.ResourceUtilization{
// 		ClusterID: clusterID,
// 	}

// 	// 收集基本统计信息
// 	stats := &model.ClusterStats{}
// 	utils.CollectResourceStats(ctx, kubeClient, stats)
// 	utils.CollectNodeStats(ctx, kubeClient, stats)
// 	utils.CollectNamespaceStats(ctx, kubeClient, stats)

// 	// 计算总体利用率
// 	utilization.OverallUtil = model.UtilizationSummary{
// 		CPU:     stats.ResourceStats.CPUUtilization,
// 		Memory:  stats.ResourceStats.MemoryUtilization,
// 		Storage: stats.ResourceStats.StorageUtilization,
// 		Network: 0.0, // 需要metrics-server支持
// 		Overall: (stats.ResourceStats.CPUUtilization + stats.ResourceStats.MemoryUtilization) / 2,
// 	}

// 	// 获取节点利用率
// 	nodeUtils, err := r.getNodeUtilizations(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取节点利用率失败", zap.Error(err))
// 	} else {
// 		utilization.NodeUtils = nodeUtils
// 	}

// 	// 获取命名空间利用率
// 	nsUtils, err := r.getNamespaceUtilizations(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取命名空间利用率失败", zap.Error(err))
// 	} else {
// 		utilization.NSUtils = nsUtils
// 	}

// 	// 生成利用率图表
// 	utilization.UtilChart = r.generateUtilizationChart(nodeUtils, nsUtils)

// 	// 生成优化建议
// 	utilization.Recommendations = r.generateUtilizationAdvice(utilization.OverallUtil, nodeUtils)

// 	r.logger.Info("成功获取资源利用率", zap.Int("clusterID", clusterID))
// 	return utilization, nil
// }

// // GetResourceHealth 获取资源健康状态
// func (r *resourceService) GetResourceHealth(ctx context.Context, clusterID int) (*model.ResourceHealth, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	health := &model.ResourceHealth{
// 		ClusterID: clusterID,
// 	}

// 	// 收集健康相关信息
// 	stats := &model.ClusterStats{}
// 	utils.CollectNodeStats(ctx, kubeClient, stats)
// 	utils.CollectPodStats(ctx, kubeClient, stats)
// 	utils.CollectEventStats(ctx, kubeClient, stats)

// 	// 计算总体健康评分
// 	health.OverallHealth = r.calculateHealthScore(stats)

// 	// 获取组件健康状态
// 	components, err := r.getComponentHealth(ctx, kubeClient)
// 	if err != nil {
// 		r.logger.Warn("获取组件健康状态失败", zap.Error(err))
// 	} else {
// 		health.ComponentHealth = components
// 	}

// 	// 识别资源问题
// 	health.ResourceIssues = r.identifyResourceIssues(ctx, kubeClient, stats)

// 	// 生成健康趋势（模拟）
// 	health.HealthTrend = r.generateHealthTrend()

// 	// 生成可操作警报
// 	health.ActionableAlerts = r.generateActionableAlerts(health.ResourceIssues)

// 	r.logger.Info("成功获取资源健康状态", zap.Int("clusterID", clusterID))
// 	return health, nil
// }

// // GetWorkloadDistribution 获取工作负载分布
// func (r *resourceService) GetWorkloadDistribution(ctx context.Context, clusterID int) (*model.WorkloadDistribution, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	return r.getDetailedWorkloadDistribution(ctx, kubeClient)
// }

// // GetNamespaceResources 获取命名空间资源信息
// func (r *resourceService) GetNamespaceResources(ctx context.Context, clusterID int) ([]*model.NamespaceUsage, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	// 获取所有命名空间
// 	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		r.logger.Error("获取命名空间列表失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取命名空间列表失败")
// 	}

// 	// 获取所有Pod用于统计
// 	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		r.logger.Error("获取Pod列表失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Pod列表失败")
// 	}

// 	// 按命名空间统计Pod和资源使用
// 	nsResources := make(map[string]*model.NamespaceUsage)
// 	for _, ns := range namespaces.Items {
// 		nsResources[ns.Name] = &model.NamespaceUsage{
// 			Name:     ns.Name,
// 			IsSystem: utils.IsSystemNamespace(ns.Name),
// 			Status:   string(ns.Status.Phase),
// 		}
// 	}

// 	// 统计每个命名空间的资源使用
// 	for _, pod := range pods.Items {
// 		if nsUsage, exists := nsResources[pod.Namespace]; exists {
// 			nsUsage.PodCount++

// 			// 计算CPU和内存使用（基于requests）
// 			for _, container := range pod.Spec.Containers {
// 				if cpu := container.Resources.Requests.Cpu(); !cpu.IsZero() {
// 					nsUsage.CPUUsage = fmt.Sprintf("%.2f cores", cpu.AsApproximateFloat64())
// 				}
// 				if mem := container.Resources.Requests.Memory(); !mem.IsZero() {
// 					nsUsage.MemUsage = fmt.Sprintf("%.2f Gi", mem.AsApproximateFloat64()/(1024*1024*1024))
// 				}
// 			}
// 		}
// 	}

// 	// 转换为切片并排序
// 	var result []*model.NamespaceUsage
// 	for _, nsUsage := range nsResources {
// 		result = append(result, nsUsage)
// 	}

// 	sort.Slice(result, func(i, j int) bool {
// 		return result[i].PodCount > result[j].PodCount
// 	})

// 	r.logger.Info("成功获取命名空间资源信息",
// 		zap.Int("clusterID", clusterID),
// 		zap.Int("namespaceCount", len(result)))

// 	return result, nil
// }

// // GetStorageOverview 获取存储概览
// func (r *resourceService) GetStorageOverview(ctx context.Context, clusterID int) (*model.StorageStats, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	// 收集存储统计信息
// 	stats := &model.ClusterStats{}
// 	err = utils.CollectStorageStats(ctx, kubeClient, stats)
// 	if err != nil {
// 		r.logger.Error("收集存储统计信息失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sResourceOperation, "获取存储信息失败")
// 	}

// 	r.logger.Info("成功获取存储概览", zap.Int("clusterID", clusterID))
// 	return &stats.StorageStats, nil
// }

// // GetNetworkOverview 获取网络概览
// func (r *resourceService) GetNetworkOverview(ctx context.Context, clusterID int) (*model.NetworkStats, error) {
// 	// 获取k8s客户端
// 	kubeClient, err := r.client.GetKubeClient(clusterID)
// 	if err != nil {
// 		r.logger.Error("获取Kubernetes客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	// 收集网络统计信息
// 	stats := &model.ClusterStats{}
// 	err = utils.CollectNetworkStats(ctx, kubeClient, stats)
// 	if err != nil {
// 		r.logger.Error("收集网络统计信息失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sResourceOperation, "获取网络信息失败")
// 	}

// 	r.logger.Info("成功获取网络概览", zap.Int("clusterID", clusterID))
// 	return &stats.NetworkStats, nil
// }

// // CompareClusterResources 对比多个集群的资源使用情况
// func (r *resourceService) CompareClusterResources(ctx context.Context, clusterIDs []int) (*model.ResourceComparisonChart, error) {
// 	if len(clusterIDs) < 2 || len(clusterIDs) > 10 {
// 		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "集群数量必须在2-10之间")
// 	}

// 	comparison := &model.ResourceComparisonChart{
// 		ClusterNames: make([]string, 0, len(clusterIDs)),
// 		CPUData:      make([]float64, 0, len(clusterIDs)),
// 		MemoryData:   make([]float64, 0, len(clusterIDs)),
// 		PodData:      make([]float64, 0, len(clusterIDs)),
// 		Detailed:     make([]model.ClusterComparison, 0, len(clusterIDs)),
// 	}

// 	for _, clusterID := range clusterIDs {
// 		// 获取集群信息
// 		cluster, err := r.dao.GetClusterByID(ctx, clusterID)
// 		if err != nil || cluster == nil {
// 			r.logger.Warn("获取集群信息失败", zap.Int("clusterID", clusterID))
// 			continue
// 		}

// 		// 获取集群统计信息
// 		stats, err := r.GetResourceStatistics(ctx, clusterID)
// 		if err != nil {
// 			r.logger.Warn("获取集群统计失败", zap.Int("clusterID", clusterID))
// 			continue
// 		}

// 		comparison.ClusterNames = append(comparison.ClusterNames, cluster.Name)
// 		comparison.CPUData = append(comparison.CPUData, stats.ResourceStats.CPUUtilization)
// 		comparison.MemoryData = append(comparison.MemoryData, stats.ResourceStats.MemoryUtilization)
// 		comparison.PodData = append(comparison.PodData, float64(stats.PodStats.TotalPods))

// 		// 计算效率得分（简化计算）
// 		efficiency := (stats.ResourceStats.CPUUtilization + stats.ResourceStats.MemoryUtilization) / 200 * 100

// 		comparison.Detailed = append(comparison.Detailed, model.ClusterComparison{
// 			ClusterName: cluster.Name,
// 			CPU:         stats.ResourceStats.TotalCPU,
// 			Memory:      stats.ResourceStats.TotalMemory,
// 			Nodes:       stats.NodeStats.TotalNodes,
// 			Pods:        stats.PodStats.TotalPods,
// 			Efficiency:  efficiency,
// 		})
// 	}

// 	r.logger.Info("成功对比集群资源", zap.Int("clusterCount", len(comparison.ClusterNames)))
// 	return comparison, nil
// }

// // GetAllClustersSummary 获取所有集群资源汇总
// func (r *resourceService) GetAllClustersSummary(ctx context.Context) (*model.AllClustersSummary, error) {
// 	// 获取所有集群
// 	clusters, total, err := r.dao.GetClusterList(ctx, &model.ListClustersReq{
// 		ListReq: model.ListReq{Page: 1, Size: 100}, // 假设最多100个集群
// 	})
// 	if err != nil {
// 		r.logger.Error("获取集群列表失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "获取集群列表失败")
// 	}

// 	summary := &model.AllClustersSummary{
// 		TotalClusters:    int(total),
// 		TotalResources:   model.GlobalResourceSummary{},
// 		ClustersOverview: make([]model.ClusterBriefSummary, 0, len(clusters)),
// 		AlertsSummary: model.GlobalAlertsSummary{
// 			AlertsByCluster: make(map[string]int),
// 		},
// 	}

// 	// 初始化资源对比图表
// 	comparison := &model.ResourceComparisonChart{
// 		ClusterNames: make([]string, 0, len(clusters)),
// 		CPUData:      make([]float64, 0, len(clusters)),
// 		MemoryData:   make([]float64, 0, len(clusters)),
// 		PodData:      make([]float64, 0, len(clusters)),
// 	}

// 	var healthyCount, unhealthyCount int
// 	var totalNodes, totalPods int
// 	var totalCPUUtil, totalMemUtil float64
// 	validClusters := 0

// 	for _, cluster := range clusters {
// 		// 获取集群统计
// 		stats, err := r.GetResourceStatistics(ctx, cluster.ID)
// 		if err != nil {
// 			r.logger.Warn("获取集群统计失败", zap.Int("clusterID", cluster.ID))
// 			unhealthyCount++
// 			continue
// 		}

// 		// 计算健康状态
// 		healthScore := r.calculateHealthScore(stats)
// 		if healthScore.Score > 70 {
// 			healthyCount++
// 		} else {
// 			unhealthyCount++
// 		}

// 		// 累计资源
// 		totalNodes += stats.NodeStats.TotalNodes
// 		totalPods += stats.PodStats.TotalPods
// 		totalCPUUtil += stats.ResourceStats.CPUUtilization
// 		totalMemUtil += stats.ResourceStats.MemoryUtilization
// 		validClusters++

// 		// 添加到概览
// 		summary.ClustersOverview = append(summary.ClustersOverview, model.ClusterBriefSummary{
// 			ClusterID:   cluster.ID,
// 			ClusterName: cluster.Name,
// 			Status:      strconv.Itoa(int(cluster.Status)),
// 			HealthScore: healthScore.Score,
// 			NodeCount:   stats.NodeStats.TotalNodes,
// 			PodCount:    stats.PodStats.TotalPods,
// 			CPUUtil:     stats.ResourceStats.CPUUtilization,
// 			MemoryUtil:  stats.ResourceStats.MemoryUtilization,
// 			Issues:      stats.EventStats.WarningEvents,
// 		})

// 		// 添加到对比图表
// 		comparison.ClusterNames = append(comparison.ClusterNames, cluster.Name)
// 		comparison.CPUData = append(comparison.CPUData, stats.ResourceStats.CPUUtilization)
// 		comparison.MemoryData = append(comparison.MemoryData, stats.ResourceStats.MemoryUtilization)
// 		comparison.PodData = append(comparison.PodData, float64(stats.PodStats.TotalPods))

// 		// 统计警报
// 		summary.AlertsSummary.AlertsByCluster[cluster.Name] = stats.EventStats.WarningEvents
// 		summary.AlertsSummary.TotalAlerts += stats.EventStats.WarningEvents
// 		if stats.EventStats.WarningEvents > 10 {
// 			summary.AlertsSummary.CriticalAlerts++
// 		}
// 	}

// 	// 设置汇总信息
// 	summary.HealthyClusters = healthyCount
// 	summary.UnhealthyClusters = unhealthyCount
// 	summary.TotalResources.TotalNodes = totalNodes
// 	summary.TotalResources.TotalPods = totalPods

// 	if validClusters > 0 {
// 		summary.TotalResources.AvgCPUUtil = totalCPUUtil / float64(validClusters)
// 		summary.TotalResources.AvgMemUtil = totalMemUtil / float64(validClusters)
// 	}

// 	summary.ResourceComparison = *comparison

// 	r.logger.Info("成功获取所有集群汇总",
// 		zap.Int("totalClusters", summary.TotalClusters),
// 		zap.Int("healthyClusters", summary.HealthyClusters))

// 	return summary, nil
// }

// // ==================== 私有辅助方法 ====================

// // calculateHealthRate 计算健康率
// func calculateHealthRate(healthy, total int) float64 {
// 	if total == 0 {
// 		return 0
// 	}
// 	return float64(healthy) / float64(total) * 100
// }

// // calculateOverallHealthStatus 计算总体健康状态
// func (r *resourceService) calculateOverallHealthStatus(stats *model.ClusterStats) string {
// 	score := r.calculateHealthScore(stats).Score
// 	switch {
// 	case score >= 90:
// 		return "excellent"
// 	case score >= 75:
// 		return "good"
// 	case score >= 60:
// 		return "fair"
// 	default:
// 		return "poor"
// 	}
// }

// // calculateHealthScore 计算健康评分
// func (r *resourceService) calculateHealthScore(stats *model.ClusterStats) model.HealthScore {
// 	score := 100
// 	factors := make([]string, 0)

// 	// 节点健康检查
// 	if stats.NodeStats.TotalNodes > 0 {
// 		nodeHealthRate := float64(stats.NodeStats.ReadyNodes) / float64(stats.NodeStats.TotalNodes)
// 		if nodeHealthRate < 0.9 {
// 			score -= int((0.9 - nodeHealthRate) * 30)
// 			factors = append(factors, "节点健康状态")
// 		}
// 	}

// 	// Pod健康检查
// 	if stats.PodStats.TotalPods > 0 {
// 		podHealthRate := float64(stats.PodStats.RunningPods) / float64(stats.PodStats.TotalPods)
// 		if podHealthRate < 0.8 {
// 			score -= int((0.8 - podHealthRate) * 25)
// 			factors = append(factors, "Pod健康状态")
// 		}
// 	}

// 	// 资源利用率检查
// 	if stats.ResourceStats.CPUUtilization > 90 {
// 		score -= 15
// 		factors = append(factors, "CPU利用率过高")
// 	}
// 	if stats.ResourceStats.MemoryUtilization > 90 {
// 		score -= 15
// 		factors = append(factors, "内存利用率过高")
// 	}

// 	// 事件检查
// 	if stats.EventStats.WarningEvents > 10 {
// 		score -= 10
// 		factors = append(factors, "警告事件过多")
// 	}

// 	if score < 0 {
// 		score = 0
// 	}

// 	level := "excellent"
// 	description := "集群运行状态优秀"
// 	switch {
// 	case score < 50:
// 		level = "critical"
// 		description = "集群存在严重问题"
// 	case score < 70:
// 		level = "warning"
// 		description = "集群需要关注"
// 	case score < 90:
// 		level = "good"
// 		description = "集群运行状态良好"
// 	}

// 	return model.HealthScore{
// 		Score:       score,
// 		Level:       level,
// 		Description: description,
// 		Factors:     factors,
// 	}
// }

// // buildTopNamespaces 构建Top命名空间列表
// func (r *resourceService) buildTopNamespaces(topNS []string, stats *model.ClusterStats) []model.NamespaceUsage {
// 	result := make([]model.NamespaceUsage, 0, len(topNS))
// 	for _, nsName := range topNS {
// 		usage := model.NamespaceUsage{
// 			Name:     nsName,
// 			IsSystem: utils.IsSystemNamespace(nsName),
// 			Status:   "Active",
// 		}
// 		result = append(result, usage)
// 	}
// 	return result
// }

// // buildRecentEvents 构建最近事件列表（简化版）
// func (r *resourceService) buildRecentEvents(ctx context.Context, kubeClient *kubernetes.Clientset, limit int) []model.EventSummary {
// 	// 暂时返回空列表，避免字段不匹配问题
// 	return []model.EventSummary{}
// }
