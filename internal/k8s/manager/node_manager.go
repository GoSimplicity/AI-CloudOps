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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeManager interface {
	GetNode(ctx context.Context, clusterID int, nodeName string) (*corev1.Node, error)
	GetNodeList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.NodeList, int64, error)
	BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node) (*model.K8sNode, error)
	DrainNode(ctx context.Context, clusterID int, nodeName string, options *utils.DrainOptions) error
	CordonNode(ctx context.Context, clusterID int, nodeName string) error
	UncordonNode(ctx context.Context, clusterID int, nodeName string) error
	UpdateNodeLabels(ctx context.Context, clusterID int, nodeName string, labels map[string]string) error
	GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]*model.NodeTaint, int64, error)
}

type nodeManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewNodeManager(client client.K8sClient, logger *zap.Logger) NodeManager {
	return &nodeManager{
		client: client,
		logger: logger,
	}
}

func (m *nodeManager) GetNode(ctx context.Context, clusterID int, nodeName string) (*corev1.Node, error) {
	if err := utils.ValidateNodeName(nodeName); err != nil {
		return nil, err
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	return node, nil
}

func (m *nodeManager) GetNodeList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.NodeList, int64, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, 0, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	nodeList, err := clientset.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取节点列表失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, 0, fmt.Errorf("获取节点列表失败: %w", err)
	}

	return nodeList, int64(len(nodeList.Items)), nil
}

func (m *nodeManager) BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node) (*model.K8sNode, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	k8sNode, err := utils.BuildK8sNode(ctx, clusterID, node, clientset, nil)
	if err != nil {
		m.logger.Error("构建K8sNode失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", node.Name))
		return nil, fmt.Errorf("构建K8sNode失败: %w", err)
	}

	return k8sNode, nil
}

func (m *nodeManager) DrainNode(ctx context.Context, clusterID int, nodeName string, options *utils.DrainOptions) error {
	if err := utils.ValidateNodeName(nodeName); err != nil {
		return err
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		m.logger.Error("获取节点Pod列表失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点Pod列表失败: %w", err)
	}

	var drainErrors []error
	successCount := 0
	skipCount := 0

	for _, pod := range pods.Items {
		if utils.ShouldSkipPodDrain(pod, options) {
			skipCount++
			m.logger.Debug("跳过Pod驱逐",
				zap.String("podName", pod.Name),
				zap.String("namespace", pod.Namespace),
				zap.String("reason", "shouldSkip"))
			continue
		}

		// 使用Eviction API而不是直接删除，这样会遵守PDB
		deleteOptions := utils.BuildDeleteOptions(options.GracePeriodSeconds)

		eviction := &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
			DeleteOptions: &deleteOptions,
		}

		err := clientset.PolicyV1().Evictions(pod.Namespace).Evict(ctx, eviction)
		if err != nil {
			m.logger.Error("驱逐Pod失败",
				zap.Error(err),
				zap.Int("clusterID", clusterID),
				zap.String("nodeName", nodeName),
				zap.String("podName", pod.Name),
				zap.String("namespace", pod.Namespace))
			drainErrors = append(drainErrors, fmt.Errorf("驱逐Pod %s/%s 失败: %w", pod.Namespace, pod.Name, err))

			// 如果强制驱逐，即使失败也继续
			if options.Force == 2 {
				continue
			}
			// 非强制模式下，遇到错误就停止
			return fmt.Errorf("驱逐节点失败: %w", err)
		}

		successCount++
		m.logger.Info("成功驱逐Pod",
			zap.String("podName", pod.Name),
			zap.String("namespace", pod.Namespace),
			zap.String("nodeName", nodeName))
	}

	m.logger.Info("节点驱逐完成",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName),
		zap.Int("successCount", successCount),
		zap.Int("skipCount", skipCount),
		zap.Int("errorCount", len(drainErrors)))

	// 如果有错误但是是强制模式，返回成功但记录错误
	if len(drainErrors) > 0 {
		m.logger.Warn("部分Pod驱逐失败", zap.Errors("errors", drainErrors))
	}

	return nil
}

func (m *nodeManager) CordonNode(ctx context.Context, clusterID int, nodeName string) error {
	if err := utils.ValidateNodeName(nodeName); err != nil {
		return err
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点失败: %w", err)
	}

	node.Spec.Unschedulable = true
	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("标记节点不可调度失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("标记节点不可调度失败: %w", err)
	}

	m.logger.Info("节点已标记为不可调度", zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
	return nil
}

func (m *nodeManager) UncordonNode(ctx context.Context, clusterID int, nodeName string) error {
	if err := utils.ValidateNodeName(nodeName); err != nil {
		return err
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点失败: %w", err)
	}

	node.Spec.Unschedulable = false
	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("标记节点可调度失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("标记节点可调度失败: %w", err)
	}

	m.logger.Info("节点已标记为可调度", zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
	return nil
}

// UpdateNodeLabels 更新节点标签
func (m *nodeManager) UpdateNodeLabels(ctx context.Context, clusterID int, nodeName string, labels map[string]string) error {
	if err := utils.ValidateNodeName(nodeName); err != nil {
		return err
	}

	// 允许传入空标签，用于清空所有标签
	if labels != nil {
		if err := utils.ValidateNodeLabelsMap(labels); err != nil {
			return err
		}
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点失败: %w", err)
	}

	// 完全覆盖原有标签
	if labels == nil {
		node.Labels = make(map[string]string)
	} else {
		node.Labels = labels
	}

	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新节点标签失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("更新节点标签失败: %w", err)
	}

	m.logger.Info("节点标签更新成功", zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName), zap.Any("labels", labels))
	return nil
}

func (m *nodeManager) GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]*model.NodeTaint, int64, error) {
	if err := utils.ValidateNodeName(nodeName); err != nil {
		return nil, 0, err
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, 0, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return nil, 0, fmt.Errorf("获取节点失败: %w", err)
	}

	taints, total := utils.BuildNodeTaints(node.Spec.Taints)
	return taints, total, nil
}
