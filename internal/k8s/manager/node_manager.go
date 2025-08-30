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

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// NodeManager Node 资源管理器，负责节点相关的业务逻辑
// 通过依赖注入接收客户端工厂，实现业务逻辑与客户端创建的解耦
type NodeManager interface {
	// Node 查询
	GetNode(ctx context.Context, clusterID int, nodeName string) (*corev1.Node, error)
	GetNodeList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.NodeList, error)

	// Node 详细信息构建
	BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node) (*model.K8sNode, error)

	// Node 操作
	DrainNode(ctx context.Context, clusterID int, nodeName string) error
	CordonNode(ctx context.Context, clusterID int, nodeName string) error
	UncordonNode(ctx context.Context, clusterID int, nodeName string) error
}

type nodeManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

// NewNodeManager 创建新的 Node 管理器实例
func NewNodeManager(clientFactory client.K8sClient, logger *zap.Logger) NodeManager {
	return &nodeManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getClients 私有方法：获取 Kubernetes 客户端和 Metrics 客户端
func (n *nodeManager) getClients(clusterID int) (*kubernetes.Clientset, *metricsClient.Clientset, error) {
	kubeClient, err := n.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// Metrics 客户端是可选的，获取失败不影响基本功能
	metricsClient, err := n.clientFactory.GetMetricsClient(clusterID)
	if err != nil {
		n.logger.Warn("获取 Metrics 客户端失败，将在无指标模式下运行",
			zap.Int("clusterID", clusterID), zap.Error(err))
		return kubeClient, nil, nil
	}

	return kubeClient, metricsClient, nil
}

// GetNode 获取单个节点
func (n *nodeManager) GetNode(ctx context.Context, clusterID int, nodeName string) (*corev1.Node, error) {
	kubeClient, _, err := n.getClients(clusterID)
	if err != nil {
		return nil, err
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 Node 失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Node 失败: %w", err)
	}

	n.logger.Debug("成功获取 Node",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName))
	return node, nil
}

// GetNodeList 获取节点列表
func (n *nodeManager) GetNodeList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.NodeList, error) {
	kubeClient, _, err := n.getClients(clusterID)
	if err != nil {
		return nil, err
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		n.logger.Error("获取 Node 列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Node 列表失败: %w", err)
	}

	n.logger.Debug("成功获取 Node 列表",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(nodeList.Items)))
	return nodeList, nil
}

// BuildK8sNode 构建详细的 K8sNode 模型
// 整合节点基本信息、Pod 列表、事件和资源使用情况
func (n *nodeManager) BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node) (*model.K8sNode, error) {
	// kubeClient, metricsClient, err := n.getClients(clusterID)
	// if err != nil {
	// 	return nil, err
	// }

	// // 使用 utils 包中的工具函数构建 K8sNode
	// k8sNode, err := utils.BuildK8sNode(ctx, clusterID, node, kubeClient, metricsClient)
	// if err != nil {
	// 	n.logger.Error("构建 K8sNode 失败",
	// 		zap.Int("clusterID", clusterID),
	// 		zap.String("nodeName", node.Name),
	// 		zap.Error(err))
	// 	return nil, fmt.Errorf("构建 K8sNode 失败: %w", err)
	// }

	// n.logger.Debug("成功构建 K8sNode",
	// 	zap.Int("clusterID", clusterID),
	// 	zap.String("nodeName", node.Name))
	return nil, nil
}

// DrainNode 驱逐节点上的所有 Pod（排水）
func (n *nodeManager) DrainNode(ctx context.Context, clusterID int, nodeName string) error {
	kubeClient, _, err := n.getClients(clusterID)
	if err != nil {
		return err
	}

	// 获取节点上的所有 Pod
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		n.logger.Error("获取节点 Pod 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return fmt.Errorf("获取节点 Pod 列表失败: %w", err)
	}

	// 删除非系统 Pod（驱逐）
	for _, pod := range pods.Items {
		// 跳过系统命名空间的 Pod
		if pod.Namespace == "kube-system" || pod.Namespace == "kube-public" || pod.Namespace == "kube-node-lease" {
			continue
		}

		// 跳过由 DaemonSet 管理的 Pod
		isDaemonSetPod := false
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Kind == "DaemonSet" {
				isDaemonSetPod = true
				break
			}
		}
		if isDaemonSetPod {
			continue
		}

		err := kubeClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			n.logger.Error("驱逐 Pod 失败",
				zap.Int("clusterID", clusterID),
				zap.String("nodeName", nodeName),
				zap.String("podName", pod.Name),
				zap.String("namespace", pod.Namespace),
				zap.Error(err))
			// 继续处理其他 Pod，不因单个 Pod 失败而停止整个操作
		}
	}

	n.logger.Info("成功驱逐节点",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName))
	return nil
}

// CordonNode 标记节点为不可调度
func (n *nodeManager) CordonNode(ctx context.Context, clusterID int, nodeName string) error {
	kubeClient, _, err := n.getClients(clusterID)
	if err != nil {
		return err
	}

	// 获取当前节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 Node 失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return fmt.Errorf("获取 Node 失败: %w", err)
	}

	// 设置节点为不可调度
	node.Spec.Unschedulable = true
	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		n.logger.Error("标记节点不可调度失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return fmt.Errorf("标记节点不可调度失败: %w", err)
	}

	n.logger.Info("成功标记节点为不可调度",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName))
	return nil
}

// UncordonNode 标记节点为可调度
func (n *nodeManager) UncordonNode(ctx context.Context, clusterID int, nodeName string) error {
	kubeClient, _, err := n.getClients(clusterID)
	if err != nil {
		return err
	}

	// 获取当前节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 Node 失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return fmt.Errorf("获取 Node 失败: %w", err)
	}

	// 设置节点为可调度
	node.Spec.Unschedulable = false
	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		n.logger.Error("标记节点可调度失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return fmt.Errorf("标记节点可调度失败: %w", err)
	}

	n.logger.Info("成功标记节点为可调度",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName))
	return nil
}
