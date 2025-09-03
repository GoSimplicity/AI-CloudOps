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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeManager interface {
	GetNode(ctx context.Context, clusterID int, nodeName string) (*corev1.Node, error)
	GetNodeList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.NodeList, int64, error)
	BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node) (*model.K8sNode, error)
	DrainNode(ctx context.Context, clusterID int, nodeName string, options *utils.DrainOptions) error
	CordonNode(ctx context.Context, clusterID int, nodeName string) error
	UncordonNode(ctx context.Context, clusterID int, nodeName string) error
	AddOrUpdateNodeLabels(ctx context.Context, clusterID int, nodeName string, labels map[string]string, overwrite int8) error
	DeleteNodeLabels(ctx context.Context, clusterID int, nodeName string, labelKeys []string) error
	GetNodeResource(ctx context.Context, clusterID int, nodeName string) (*model.NodeResource, error)
	GetNodeEvents(ctx context.Context, clusterID int, nodeName string, limit int) ([]*model.NodeEvent, int64, error)
	GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]*model.NodeTaintEntity, int64, error)
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
	if nodeName == "" {
		return nil, fmt.Errorf("节点名称不能为空")
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
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
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

	for _, pod := range pods.Items {
		if utils.ShouldSkipPodDrain(pod, options) {
			continue
		}

		deleteOptions := utils.BuildDeleteOptions(options.GracePeriodSeconds)

		err := clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOptions)
		if err != nil {
			m.logger.Error("驱逐Pod失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName), zap.String("podName", pod.Name))
		}
	}

	m.logger.Info("节点驱逐完成", zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
	return nil
}

func (m *nodeManager) CordonNode(ctx context.Context, clusterID int, nodeName string) error {
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
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
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
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

func (m *nodeManager) AddOrUpdateNodeLabels(ctx context.Context, clusterID int, nodeName string, labels map[string]string, overwrite int8) error {
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(labels) == 0 {
		return fmt.Errorf("标签不能为空")
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

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	for key, value := range labels {
		if _, exists := node.Labels[key]; exists && overwrite == 1 {
			continue
		}
		node.Labels[key] = value
	}

	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新节点标签失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("更新节点标签失败: %w", err)
	}

	m.logger.Info("节点标签更新成功", zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
	return nil
}

func (m *nodeManager) DeleteNodeLabels(ctx context.Context, clusterID int, nodeName string, labelKeys []string) error {
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(labelKeys) == 0 {
		return fmt.Errorf("标签键不能为空")
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

	if node.Labels != nil {
		for _, key := range labelKeys {
			delete(node.Labels, key)
		}
	}

	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("删除节点标签失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return fmt.Errorf("删除节点标签失败: %w", err)
	}

	m.logger.Info("节点标签删除成功", zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
	return nil
}

func (m *nodeManager) GetNodeResource(ctx context.Context, clusterID int, nodeName string) (*model.NodeResource, error) {
	if nodeName == "" {
		return nil, fmt.Errorf("节点名称不能为空")
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

	resource := utils.BuildNodeResource(ctx, clientset, node)
	return resource, nil
}

func (m *nodeManager) GetNodeEvents(ctx context.Context, clusterID int, nodeName string, limit int) ([]*model.NodeEvent, int64, error) {
	if nodeName == "" {
		return nil, 0, fmt.Errorf("节点名称不能为空")
	}

	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, 0, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	eventList, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName),
	})
	if err != nil {
		m.logger.Error("获取节点事件失败", zap.Error(err), zap.Int("clusterID", clusterID), zap.String("nodeName", nodeName))
		return nil, 0, fmt.Errorf("获取节点事件失败: %w", err)
	}

	var events []*model.NodeEvent
	count := 0
	for _, event := range eventList.Items {
		if limit > 0 && count >= limit {
			break
		}

		nodeEvent := &model.NodeEvent{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Component:      event.Source.Component,
			Host:           event.Source.Host,
			FirstTimestamp: event.FirstTimestamp.Time,
			LastTimestamp:  event.LastTimestamp.Time,
			Count:          event.Count,
		}
		events = append(events, nodeEvent)
		count++
	}

	return events, int64(count), nil
}

func (m *nodeManager) GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]*model.NodeTaintEntity, int64, error) {
	if nodeName == "" {
		return nil, 0, fmt.Errorf("节点名称不能为空")
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

	var taints []*model.NodeTaintEntity
	for _, taint := range node.Spec.Taints {
		taintEntity := &model.NodeTaintEntity{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
		taints = append(taints, taintEntity)
	}

	return taints, int64(len(taints)), nil
}
