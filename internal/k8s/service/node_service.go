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
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeService interface {
	ListNodeByClusterName(ctx context.Context, id int) ([]*model.K8sNode, error)
	GetNodeDetail(ctx context.Context, id int, name string) (*model.K8sNode, error)
	AddOrUpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesReq) error
	GetNodeResources(ctx context.Context, id int) (*model.NodeResources, error)
	GetNodeEvents(ctx context.Context, id int, nodeName string) ([]model.OneEvent, error)
	DrainNode(ctx context.Context, req *model.NodeDrainReq) (*model.NodeDrainResponse, error)
	CordonNode(ctx context.Context, req *model.NodeCordonReq) (*model.NodeCordonResponse, error)
	UncordonNode(ctx context.Context, req *model.NodeUncordonReq) (*model.NodeUncordonResponse, error)
	GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]model.NodeTaintEntity, error)
}

type nodeService struct {
	clusterDao  dao.ClusterDAO
	client      client.K8sClient
	nodeManager manager.NodeManager
	l           *zap.Logger
}

func NewNodeService(clusterDao dao.ClusterDAO, client client.K8sClient, nodeManager manager.NodeManager, l *zap.Logger) NodeService {
	return &nodeService{
		clusterDao:  clusterDao,
		client:      client,
		nodeManager: nodeManager,
		l:           l,
	}
}

// ListNodeByClusterName 获取集群的节点列表
func (n *nodeService) ListNodeByClusterName(ctx context.Context, id int) ([]*model.K8sNode, error) {
	// 使用 NodeManager 获取节点列表
	nodeList, err := n.nodeManager.GetNodeList(ctx, id, metav1.ListOptions{})
	if err != nil {
		n.l.Error("获取节点列表失败", zap.Error(err), zap.Int("clusterID", id))
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}

	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	g, ctx := errgroup.WithContext(ctx)
	k8sNodes := make([]*model.K8sNode, len(nodeList.Items))

	// 使用 Worker Pool 模式优化并发性能
	for i := range nodeList.Items {
		index := i
		node := nodeList.Items[i] // 避免闭包变量问题
		g.Go(func() error {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
			}()

			// 使用 NodeManager 构建 K8sNode
			k8sNode, err := n.nodeManager.BuildK8sNode(ctx, id, node)
			if err != nil {
				n.l.Error("构建 K8sNode 失败", zap.Error(err), zap.String("node", node.Name))
				return err
			}
			k8sNodes[index] = k8sNode
			return nil
		})
	}

	// 等待所有协程完成
	if err := g.Wait(); err != nil {
		n.l.Error("并发处理节点信息失败", zap.Error(err))
		return nil, err
	}

	return k8sNodes, nil
}

// GetNodeDetail 获取指定节点详情
func (n *nodeService) GetNodeDetail(ctx context.Context, id int, name string) (*model.K8sNode, error) {
	// 使用 NodeManager 获取指定节点
	node, err := n.nodeManager.GetNode(ctx, id, name)
	if err != nil {
		n.l.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", id), zap.String("nodeName", name))
		return nil, constants.ErrorNodeNotFound
	}

	// 使用 NodeManager 构建 K8sNode
	return n.nodeManager.BuildK8sNode(ctx, id, *node)
}

// AddOrUpdateNodeLabel 更新节点标签（添加、删除或更新）
func (n *nodeService) AddOrUpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesReq) error {
	// TODO: 实现GetKubeClient函数
	// kubeClient, err := k8sutils.GetKubeClient(req.ClusterId, n.client, n.l)
	// if err != nil {
	// 	n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	// 	return err
	// }

	// 临时实现：直接通过client获取
	cluster, err := n.clusterDao.GetClusterByID(ctx, req.ClusterID)
	if err != nil {
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 校验传入的 Labels 数组长度是否为偶数
	if len(req.Labels)%2 != 0 {
		n.l.Error("传入的 Labels 数组不合法", zap.Int("labelsLength", len(req.Labels)))
		return fmt.Errorf("传入的 Labels 数组必须是偶数个元素")
	}

	// 将传入的 labels 转换为 map[string]string
	labelsMap := make(map[string]string)
	for i := 0; i < len(req.Labels); i += 2 {
		labelsMap[req.Labels[i]] = req.Labels[i+1]
	}

	// 获取指定节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		n.l.Error("获取节点信息失败", zap.Error(err))
		return fmt.Errorf("获取节点 %s 信息失败: %w", req.NodeName, err)
	}

	// 根据操作类型进行标签处理
	switch req.ModType {
	case "add":
		if node.Labels == nil {
			node.Labels = map[string]string{}
		}
		for k, v := range labelsMap {
			node.Labels[k] = v
		}
	case "del":
		// 删除标签
		for key := range labelsMap {
			delete(node.Labels, key)
		}
	case "update":
		// 更新标签
		for key, value := range labelsMap {
			if _, exists := node.Labels[key]; exists {
				node.Labels[key] = value
			} else {
				n.l.Warn("标签键不存在，无法更新", zap.String("key", key))
				return fmt.Errorf("节点 %s 不存在标签键 %s，无法更新", req.NodeName, key)
			}
		}
	default:
		errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
		n.l.Error(errMsg)
		return errors.New(errMsg)
	}

	// 更新节点信息
	if _, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		n.l.Error("更新节点信息失败", zap.Error(err))
		return fmt.Errorf("更新节点 %s 信息失败: %w", req.NodeName, err)
	}

	n.l.Info("更新节点Label成功", zap.String("nodeName", req.NodeName))

	// 刷新客户端
	if err := n.client.RefreshClients(ctx); err != nil {
		return fmt.Errorf("刷新客户端失败: %w", err)
	}

	return nil
}

// 辅助函数
func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getNodeIP(node *corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func getNodeAge(node *corev1.Node) string {
	// 简化实现，返回创建时间
	if node.CreationTimestamp.IsZero() {
		return "Unknown"
	}
	return "Created"
}

func getNodeLabels(node *corev1.Node) []string {
	var labels []string
	for k, v := range node.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	return labels
}

// GetNodeResources 获取节点资源信息
func (n *nodeService) GetNodeResources(ctx context.Context, id int) (*model.NodeResources, error) {
	cluster, err := n.clusterDao.GetClusterByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}

	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("集群中没有节点")
	}

	// 返回第一个节点的资源信息（简化实现）
	node := &nodes.Items[0]

	cpu := node.Status.Capacity[corev1.ResourceCPU]
	memory := node.Status.Capacity[corev1.ResourceMemory]
	storage := node.Status.Capacity[corev1.ResourceEphemeralStorage]
	pods := node.Status.Capacity[corev1.ResourcePods]

	return &model.NodeResources{
		NodeName: node.Name,
		Status:   getNodeStatus(node),
		Ready:    getNodeStatus(node) == "Ready",
		CPU:      cpu.String(),
		Memory:   memory.String(),
		Storage:  storage.String(),
		Pods:     pods.String(),
	}, nil
}

// GetNodeEvents 获取节点事件
func (n *nodeService) GetNodeEvents(ctx context.Context, id int, nodeName string) ([]model.OneEvent, error) {
	cluster, err := n.clusterDao.GetClusterByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	eventList, err := kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName),
	})
	if err != nil {
		return nil, fmt.Errorf("获取节点事件失败: %w", err)
	}

	var events []model.OneEvent
	for _, event := range eventList.Items {
		events = append(events, model.OneEvent{
			Type:      event.Type,
			Component: event.Source.Component,
			Reason:    event.Reason,
			Message:   event.Message,
			FirstTime: event.FirstTimestamp.Format("2006-01-02 15:04:05"),
			LastTime:  event.LastTimestamp.Format("2006-01-02 15:04:05"),
			Object:    fmt.Sprintf("kind:%s name:%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Count:     int(event.Count),
		})
	}

	return events, nil
}

// DrainNode 驱逐节点上的所有Pod
func (n *nodeService) DrainNode(ctx context.Context, req *model.NodeDrainReq) (*model.NodeDrainResponse, error) {
	cluster, err := n.clusterDao.GetClusterByID(ctx, req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 首先获取节点信息
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点 %s 信息失败: %w", req.NodeName, err)
	}

	// 标记节点为不可调度（cordon）
	node.Spec.Unschedulable = true
	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("标记节点 %s 不可调度失败: %w", req.NodeName, err)
	}

	// 获取节点上的所有Pod
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", req.NodeName),
	})
	if err != nil {
		return nil, fmt.Errorf("获取节点 %s 上的Pod列表失败: %w", req.NodeName, err)
	}

	var drainedPods []string
	var skippedPods []string

	// 驱逐Pod
	for _, pod := range pods.Items {
		// 跳过系统Pod和DaemonSet Pod（如果配置了忽略）
		if req.IgnoreDaemonsets && isDaemonSetPod(&pod) {
			skippedPods = append(skippedPods, pod.Name)
			continue
		}

		// 跳过静态Pod
		if isStaticPod(&pod) {
			skippedPods = append(skippedPods, pod.Name)
			continue
		}

		// 删除Pod（模拟驱逐）
		err := kubeClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: func() *int64 {
				if req.GracePeriodSeconds > 0 {
					grace := int64(req.GracePeriodSeconds)
					return &grace
				}
				return nil
			}(),
		})
		if err != nil {
			n.l.Warn("删除Pod失败", zap.Error(err), zap.String("pod", pod.Name))
			skippedPods = append(skippedPods, pod.Name)
		} else {
			drainedPods = append(drainedPods, pod.Name)
		}
	}

	n.l.Info("节点驱逐完成", zap.String("nodeName", req.NodeName), zap.Int("drainedCount", len(drainedPods)))

	return &model.NodeDrainResponse{
		NodeName:    req.NodeName,
		DrainedPods: drainedPods,
		SkippedPods: skippedPods,
		Status:      "success",
		Message:     "节点驱逐完成",
		Duration:    "completed", // 简化实现
	}, nil
}

// CordonNode 禁止节点调度新的Pod
func (n *nodeService) CordonNode(ctx context.Context, req *model.NodeCordonReq) (*model.NodeCordonResponse, error) {
	cluster, err := n.clusterDao.GetClusterByID(ctx, req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取节点信息
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点 %s 信息失败: %w", req.NodeName, err)
	}

	// 检查节点是否已经被封锁
	if node.Spec.Unschedulable {
		return &model.NodeCordonResponse{
			NodeName: req.NodeName,
			Status:   "already_cordoned",
			Message:  "节点已经被封锁",
		}, nil
	}

	// 标记节点为不可调度
	node.Spec.Unschedulable = true
	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("封锁节点 %s 失败: %w", req.NodeName, err)
	}

	n.l.Info("节点封锁成功", zap.String("nodeName", req.NodeName))

	return &model.NodeCordonResponse{
		NodeName: req.NodeName,
		Status:   "success",
		Message:  "节点已成功封锁",
	}, nil
}

// UncordonNode 解除节点调度限制
func (n *nodeService) UncordonNode(ctx context.Context, req *model.NodeUncordonReq) (*model.NodeUncordonResponse, error) {
	cluster, err := n.clusterDao.GetClusterByID(ctx, req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取节点信息
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点 %s 信息失败: %w", req.NodeName, err)
	}

	// 检查节点是否需要解封
	if !node.Spec.Unschedulable {
		return &model.NodeUncordonResponse{
			NodeName: req.NodeName,
			Status:   "already_uncordoned",
			Message:  "节点已经可以调度",
		}, nil
	}

	// 移除不可调度标记
	node.Spec.Unschedulable = false
	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("解封节点 %s 失败: %w", req.NodeName, err)
	}

	n.l.Info("节点解封成功", zap.String("nodeName", req.NodeName))

	return &model.NodeUncordonResponse{
		NodeName: req.NodeName,
		Status:   "success",
		Message:  "节点已成功解封",
	}, nil
}

// 辅助函数：检查是否为DaemonSet Pod
func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// 辅助函数：检查是否为静态Pod
func isStaticPod(pod *corev1.Pod) bool {
	return pod.Annotations["kubernetes.io/config.source"] == "file"
}

// GetNodeTaints 获取节点污点列表
func (n *nodeService) GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]model.NodeTaintEntity, error) {
	cluster, err := n.clusterDao.GetClusterByID(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := n.client.GetKubeClient(cluster.ID)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取节点信息
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err)
	}

	// 转换污点信息为实体格式
	var taintEntities []model.NodeTaintEntity
	for _, taint := range node.Spec.Taints {
		taintEntity := model.NodeTaintEntity{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
		if taint.TimeAdded != nil {
			taintEntity.TimeAdded = taint.TimeAdded.Format("2006-01-02 15:04:05")
		}
		taintEntities = append(taintEntities, taintEntity)
	}

	return taintEntities, nil
}
