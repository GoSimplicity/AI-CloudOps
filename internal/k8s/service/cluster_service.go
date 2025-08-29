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
	"strings"
	"time"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"go.uber.org/zap"
)

type ClusterService interface {
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error
	DeleteCluster(ctx context.Context, id int) error
	BatchDeleteClusters(ctx context.Context, ids []int) error
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
	RefreshClusterStatus(ctx context.Context, id int) error
	CheckClusterHealth(ctx context.Context, id int) (*model.ClusterHealthResponse, error)
	GetClusterStats(ctx context.Context, id int) (*model.ClusterStatsResponse, error)
}

type clusterService struct {
	dao        dao.ClusterDAO
	client     client.K8sClient
	clusterMgr manager.ClusterManager
	l          *zap.Logger
}

func NewClusterService(dao dao.ClusterDAO, client client.K8sClient, clusterMgr manager.ClusterManager, l *zap.Logger) ClusterService {
	return &clusterService{
		dao:        dao,
		client:     client,
		clusterMgr: clusterMgr,
		l:          l,
	}
}

// ListAllClusters 获取所有 Kubernetes 集群
func (c *clusterService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	list, err := c.dao.ListAllClusters(ctx)
	if err != nil {
		c.l.Error("ListAllClusters: 查询所有集群失败", zap.Error(err))
		return nil, fmt.Errorf("查询所有集群失败: %w", err)
	}

	return c.buildListResponse(list), nil
}

// GetClusterByID 根据 ID 获取单个 Kubernetes 集群
func (c *clusterService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	cluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("GetClusterByID: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}
	return cluster, nil
}

// CreateCluster 创建一个新的 Kubernetes 集群
func (c *clusterService) CreateCluster(ctx context.Context, cluster *model.K8sCluster) (err error) {
	// 检查集群是否存在
	existingCluster, err := c.dao.GetClusterByName(ctx, cluster.Name)
	if err != nil && err != gorm.ErrRecordNotFound {
		c.l.Error("CreateCluster: 查询集群失败", zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster != nil {
		c.l.Error("CreateCluster: 集群已存在", zap.String("clusterName", cluster.Name))
		return fmt.Errorf("集群名称 %s 已存在", cluster.Name)
	}

	cluster.Status = constants.StatusPending

	// 创建集群记录
	if err := c.dao.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 使用集群管理器创建集群
	if err := c.clusterMgr.CreateCluster(ctx, cluster); err != nil {
		c.l.Error("CreateCluster: 创建集群失败", zap.Error(err))
		return fmt.Errorf("创建集群失败: %w", err)
	}

	c.l.Info("CreateCluster: 成功创建 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// UpdateCluster 更新指定 ID 的 Kubernetes 集群
func (c *clusterService) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil {
		return fmt.Errorf("集群参数不能为空")
	}

	// 检查集群是否存在
	existingCluster, err := c.dao.GetClusterByID(ctx, cluster.ID)
	if err != nil {
		c.l.Error("UpdateCluster: 查询集群失败", zap.Int("clusterID", cluster.ID), zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", cluster.ID)
	}

	// 更新集群记录
	if err := c.dao.UpdateCluster(ctx, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群失败", zap.Error(err), zap.Int("clusterID", cluster.ID))
		return fmt.Errorf("更新集群失败: %w", err)
	}

	// 使用集群管理器更新集群
	if err := c.clusterMgr.UpdateCluster(ctx, cluster); err != nil {
		c.l.Error("UpdateCluster: 更新集群客户端失败", zap.Error(err))
		return fmt.Errorf("更新集群客户端失败: %w", err)
	}

	c.l.Info("UpdateCluster: 成功更新 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

// DeleteCluster 删除指定 ID 的 Kubernetes 集群
func (c *clusterService) DeleteCluster(ctx context.Context, id int) error {
	// 检查集群是否存在
	existingCluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("DeleteCluster: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if existingCluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", id)
	}

	// 删除集群客户端
	c.client.RemoveCluster(id)

	// 删除集群记录
	if err := c.dao.DeleteCluster(ctx, id); err != nil {
		c.l.Error("DeleteCluster: 删除集群失败", zap.Int("clusterID", id), zap.Error(err))
		return fmt.Errorf("删除集群失败: %w", err)
	}

	c.l.Info("DeleteCluster: 成功删除 Kubernetes 集群", zap.Int("clusterID", id))
	return nil
}

// BatchDeleteClusters 批量删除 Kubernetes 集群
func (c *clusterService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return fmt.Errorf("删除ID列表不能为空")
	}

	// 删除集群客户端
	for _, id := range ids {
		c.client.RemoveCluster(id)
	}

	if err := c.dao.BatchDeleteClusters(ctx, ids); err != nil {
		c.l.Error("BatchDeleteClusters: 批量删除集群失败", zap.Ints("clusterIDs", ids), zap.Error(err))
		return fmt.Errorf("批量删除集群失败: %w", err)
	}

	c.l.Info("BatchDeleteClusters: 成功批量删除 Kubernetes 集群", zap.Ints("clusterIDs", ids))
	return nil
}

// RefreshClusterStatus 刷新集群状态
func (c *clusterService) RefreshClusterStatus(ctx context.Context, id int) error {
	cluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("RefreshClusterStatus: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return fmt.Errorf("查询集群失败: %w", err)
	}

	if cluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", id)
	}

	// 使用集群管理器刷新集群状态
	if err := c.clusterMgr.RefreshCluster(ctx, id); err != nil {
		c.l.Error("RefreshClusterStatus: 刷新集群状态失败", zap.Error(err))
		return fmt.Errorf("刷新集群状态失败: %w", err)
	}

	c.l.Info("RefreshClusterStatus: 成功刷新集群状态", zap.Int("clusterID", id))
	return nil
}

func (c *clusterService) buildListResponse(clusters []*model.K8sCluster) []*model.K8sCluster {
	result := make([]*model.K8sCluster, len(clusters))
	for i, cluster := range clusters {
		clusterCopy := *cluster
		clusterCopy.KubeConfigContent = ""
		result[i] = &clusterCopy
	}

	return result
}

// CheckClusterHealth 检查集群健康状态
func (c *clusterService) CheckClusterHealth(ctx context.Context, id int) (*model.ClusterHealthResponse, error) {
	cluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("CheckClusterHealth: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}

	if cluster == nil {
		return nil, fmt.Errorf("集群不存在，ID: %d", id)
	}

	startTime := time.Now()
	response := &model.ClusterHealthResponse{
		ClusterID:     cluster.ID,
		ClusterName:   cluster.Name,
		ApiServerAddr: cluster.ApiServerAddr,
		LastCheckTime: startTime.Format("2006-01-02 15:04:05"),
		Status:        "unknown",
		Connected:     false,
	}

	// 检查集群连接
	kubeClient, err := c.client.GetKubeClient(cluster.ID)
	if err != nil {
		response.Status = "unhealthy"
		response.ErrorMessage = fmt.Sprintf("获取客户端失败: %s", err.Error())
		response.ResponseTime = fmt.Sprintf("%.2fms", float64(time.Since(startTime).Nanoseconds())/1e6)
		return response, nil
	}

	// 检查服务器版本
	version, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		response.Status = "unhealthy"
		response.ErrorMessage = fmt.Sprintf("连接失败: %s", err.Error())
		response.ResponseTime = fmt.Sprintf("%.2fms", float64(time.Since(startTime).Nanoseconds())/1e6)
		return response, nil
	}

	response.Connected = true
	response.Version = version.String()
	response.Status = "healthy"
	response.ResponseTime = fmt.Sprintf("%.2fms", float64(time.Since(startTime).Nanoseconds())/1e6)

	// 获取节点数量
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		response.NodeCount = len(nodes.Items)
	}

	// 获取命名空间数量
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		response.NamespaceCount = len(namespaces.Items)
	}

	// 获取组件状态
	componentStatuses, err := kubeClient.CoreV1().ComponentStatuses().List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, cs := range componentStatuses.Items {
			status := "healthy"
			message := "正常"

			for _, condition := range cs.Conditions {
				if condition.Type == "Healthy" {
					if condition.Status != "True" {
						status = "unhealthy"
						message = condition.Message
					}
					break
				}
			}

			response.ComponentStatus = append(response.ComponentStatus, model.ComponentHealthStatus{
				Name:      cs.Name,
				Status:    status,
				Message:   message,
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 获取资源概览
	c.fillResourceSummary(ctx, kubeClient, response)

	c.l.Info("CheckClusterHealth: 成功检查集群健康状态", zap.Int("clusterID", id), zap.String("status", response.Status))
	return response, nil
}

// fillResourceSummary 填充资源概览信息
func (c *clusterService) fillResourceSummary(ctx context.Context, kubeClient kubernetes.Interface, response *model.ClusterHealthResponse) {
	// 获取所有Pod统计
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		response.ResourceSummary.TotalPods = len(pods.Items)

		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case corev1.PodRunning:
				response.ResourceSummary.RunningPods++
			case corev1.PodPending:
				response.ResourceSummary.PendingPods++
			case corev1.PodFailed:
				response.ResourceSummary.FailedPods++
			}
		}
	}

	// 获取节点资源统计
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		var totalCPU, totalMemory int64

		for _, node := range nodes.Items {
			cpu := node.Status.Capacity[corev1.ResourceCPU]
			memory := node.Status.Capacity[corev1.ResourceMemory]

			totalCPU += cpu.MilliValue()
			totalMemory += memory.Value()
		}

		response.ResourceSummary.TotalCPU = fmt.Sprintf("%.1f cores", float64(totalCPU)/1000)
		response.ResourceSummary.TotalMemory = fmt.Sprintf("%.1fGi", float64(totalMemory)/(1024*1024*1024))

		// 这里可以进一步获取实际使用量，需要metrics-server支持
		response.ResourceSummary.UsedCPU = "需要metrics-server"
		response.ResourceSummary.UsedMemory = "需要metrics-server"
	}
}

// GetClusterStats 获取集群统计信息
func (c *clusterService) GetClusterStats(ctx context.Context, id int) (*model.ClusterStatsResponse, error) {
	cluster, err := c.dao.GetClusterByID(ctx, id)
	if err != nil {
		c.l.Error("GetClusterStats: 查询集群失败", zap.Int("clusterID", id), zap.Error(err))
		return nil, fmt.Errorf("查询集群失败: %w", err)
	}

	if cluster == nil {
		return nil, fmt.Errorf("集群不存在，ID: %d", id)
	}

	kubeClient, err := c.client.GetKubeClient(cluster.ID)
	if err != nil {
		c.l.Error("GetClusterStats: 获取客户端失败", zap.Error(err))
		return nil, fmt.Errorf("获取客户端失败: %w", err)
	}

	stats := &model.ClusterStatsResponse{
		ClusterID:      cluster.ID,
		ClusterName:    cluster.Name,
		LastUpdateTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 收集各类统计信息
	c.collectNodeStats(ctx, kubeClient, stats)
	c.collectPodStats(ctx, kubeClient, stats)
	c.collectNamespaceStats(ctx, kubeClient, stats)
	c.collectWorkloadStats(ctx, kubeClient, stats)
	c.collectResourceStats(ctx, kubeClient, stats)
	c.collectStorageStats(ctx, kubeClient, stats)
	c.collectNetworkStats(ctx, kubeClient, stats)
	c.collectEventStats(ctx, kubeClient, stats)

	c.l.Info("GetClusterStats: 成功获取集群统计信息", zap.Int("clusterID", id))
	return stats, nil
}

// collectNodeStats 收集节点统计信息
func (c *clusterService) collectNodeStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Warn("收集节点统计信息失败", zap.Error(err))
		return
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
}

// collectPodStats 收集Pod统计信息
func (c *clusterService) collectPodStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Warn("收集Pod统计信息失败", zap.Error(err))
		return
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
}

// collectNamespaceStats 收集命名空间统计信息
func (c *clusterService) collectNamespaceStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Warn("收集命名空间统计信息失败", zap.Error(err))
		return
	}

	stats.NamespaceStats.TotalNamespaces = len(namespaces.Items)

	for _, ns := range namespaces.Items {
		if ns.Status.Phase == corev1.NamespaceActive {
			stats.NamespaceStats.ActiveNamespaces++
		}

		// 检查是否为系统命名空间
		if isSystemNamespace(ns.Name) {
			stats.NamespaceStats.SystemNamespaces++
		} else {
			stats.NamespaceStats.UserNamespaces++
		}
	}

	// 获取资源使用量较多的命名空间（这里简化处理）
	stats.NamespaceStats.TopNamespaces = []string{"default", "kube-system", "kube-public"}
}

// collectWorkloadStats 收集工作负载统计信息
func (c *clusterService) collectWorkloadStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	// Deployments
	deployments, err := kubeClient.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.Deployments = len(deployments.Items)
	}

	// StatefulSets
	statefulsets, err := kubeClient.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.StatefulSets = len(statefulsets.Items)
	}

	// DaemonSets
	daemonsets, err := kubeClient.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.DaemonSets = len(daemonsets.Items)
	}

	// Jobs
	jobs, err := kubeClient.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.Jobs = len(jobs.Items)
	}

	// CronJobs
	cronjobs, err := kubeClient.BatchV1beta1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.CronJobs = len(cronjobs.Items)
	}

	// Services
	services, err := kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.Services = len(services.Items)
	}

	// ConfigMaps
	configmaps, err := kubeClient.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.ConfigMaps = len(configmaps.Items)
	}

	// Secrets
	secrets, err := kubeClient.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.Secrets = len(secrets.Items)
	}

	// Ingresses
	ingresses, err := kubeClient.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.WorkloadStats.Ingresses = len(ingresses.Items)
	}
}

// collectResourceStats 收集资源统计信息
func (c *clusterService) collectResourceStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Warn("收集资源统计信息失败", zap.Error(err))
		return
	}

	var totalCPU, totalMemory, totalStorage int64

	for _, node := range nodes.Items {
		cpu := node.Status.Capacity[corev1.ResourceCPU]
		memory := node.Status.Capacity[corev1.ResourceMemory]
		storage := node.Status.Capacity[corev1.ResourceEphemeralStorage]

		totalCPU += cpu.MilliValue()
		totalMemory += memory.Value()
		totalStorage += storage.Value()
	}

	stats.ResourceStats.TotalCPU = fmt.Sprintf("%.1f cores", float64(totalCPU)/1000)
	stats.ResourceStats.TotalMemory = fmt.Sprintf("%.1fGi", float64(totalMemory)/(1024*1024*1024))
	stats.ResourceStats.TotalStorage = fmt.Sprintf("%.1fGi", float64(totalStorage)/(1024*1024*1024))

	// 使用量需要metrics-server支持
	stats.ResourceStats.UsedCPU = "需要metrics-server"
	stats.ResourceStats.UsedMemory = "需要metrics-server"
	stats.ResourceStats.UsedStorage = "需要metrics-server"
	stats.ResourceStats.CPUUtilization = 0.0
	stats.ResourceStats.MemoryUtilization = 0.0
	stats.ResourceStats.StorageUtilization = 0.0
}

// collectStorageStats 收集存储统计信息
func (c *clusterService) collectStorageStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	// PersistentVolumes
	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.StorageStats.TotalPV = len(pvs.Items)
		for _, pv := range pvs.Items {
			switch pv.Status.Phase {
			case corev1.VolumeBound:
				stats.StorageStats.BoundPV++
			case corev1.VolumeAvailable:
				stats.StorageStats.AvailablePV++
			}
		}
	}

	// PersistentVolumeClaims
	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err == nil {
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
	if err == nil {
		stats.StorageStats.StorageClasses = len(scList.Items)
	}

	stats.StorageStats.TotalCapacity = "需要metrics-server"
}

// collectNetworkStats 收集网络统计信息
func (c *clusterService) collectNetworkStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	// Services
	services, err := kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.NetworkStats.Services = len(services.Items)
	}

	// Endpoints
	endpoints, err := kubeClient.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.NetworkStats.Endpoints = len(endpoints.Items)
	}

	// Ingresses
	ingresses, err := kubeClient.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.NetworkStats.Ingresses = len(ingresses.Items)
	}

	// NetworkPolicies
	netpols, err := kubeClient.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.NetworkStats.NetworkPolicies = len(netpols.Items)
	}
}

// collectEventStats 收集事件统计信息
func (c *clusterService) collectEventStats(ctx context.Context, kubeClient kubernetes.Interface, stats *model.ClusterStatsResponse) {
	events, err := kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Warn("收集事件统计信息失败", zap.Error(err))
		return
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
}

// isSystemNamespace 判断是否为系统命名空间
func isSystemNamespace(name string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"kubernetes-dashboard",
		"istio-system",
		"prometheus-system",
		"monitoring",
		"logging",
	}

	for _, sysNs := range systemNamespaces {
		if name == sysNs || strings.HasPrefix(name, sysNs) {
			return true
		}
	}

	return false
}
