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
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ModTypeAdd    = "add"
	ModTypeDelete = "delete"
	ModTypeUpdate = "update"
)

type TaintManager interface {
	CheckTaintYaml(ctx context.Context, clusterID int, nodeName string, taintYaml string) error
	AddOrUpdateNodeTaint(ctx context.Context, clusterID int, nodeName string, taintYaml string, modType string) error
	DrainPods(ctx context.Context, clusterID int, nodeName string) error
	GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]corev1.Taint, error)
	DeleteNodeTaintsByKeys(ctx context.Context, clusterID int, nodeName string, taintKeys []string) error
}

type taintManager struct {
	client     client.K8sClient
	clusterDao dao.ClusterDAO
	logger     *zap.Logger
}

func NewTaintManager(client client.K8sClient, clusterDao dao.ClusterDAO, logger *zap.Logger) TaintManager {
	return &taintManager{
		client:     client,
		clusterDao: clusterDao,
		logger:     logger,
	}
}

func (tm *taintManager) CheckTaintYaml(ctx context.Context, clusterID int, nodeName string, taintYaml string) error {
	taintsToProcess, err := utils.ParseTaintYaml(taintYaml)
	if err != nil {
		tm.logger.Error("解析 Taint YAML 配置失败", zap.Error(err), zap.String("nodeName", nodeName), zap.String("yamlData", taintYaml))
		return fmt.Errorf("解析 Taint YAML 配置失败: %w", err)
	}

	taintsKey := make(map[string]struct{})
	for _, taint := range taintsToProcess {
		if _, exists := taintsKey[taint.Key]; exists {
			return constants.ErrorTaintsKeyDuplicate
		}
		taintsKey[taint.Key] = struct{}{}
	}

	cluster, err := tm.clusterDao.GetClusterByID(ctx, clusterID)
	if err != nil {
		tm.logger.Error("获取集群信息失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := tm.client.GetKubeClient(cluster.ID)
	if err != nil {
		tm.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	_, err = kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		tm.logger.Error("获取节点信息失败", zap.Error(err), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err)
	}

	return nil
}

func (tm *taintManager) AddOrUpdateNodeTaint(ctx context.Context, clusterID int, nodeName string, taintYaml string, modType string) error {
	cluster, err := tm.clusterDao.GetClusterByID(ctx, clusterID)
	if err != nil {
		tm.logger.Error("获取集群信息失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := tm.client.GetKubeClient(cluster.ID)
	if err != nil {
		tm.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	taintsToProcess, err := utils.ParseTaintYaml(taintYaml)
	if err != nil {
		tm.logger.Error("解析 Taint YAML 配置失败", zap.Error(err), zap.String("nodeName", nodeName), zap.String("yamlData", taintYaml))
		return fmt.Errorf("解析 Taint YAML 配置失败: %w", err)
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		tm.logger.Error("获取节点信息失败", zap.Error(err), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err)
	}

	switch modType {
	case ModTypeAdd:
		node.Spec.Taints = tm.mergeTaints(node.Spec.Taints, taintsToProcess)
	case ModTypeDelete:
		node.Spec.Taints = tm.removeTaints(node.Spec.Taints, taintsToProcess)
	case ModTypeUpdate:
		node.Spec.Taints = tm.updateTaints(node.Spec.Taints, taintsToProcess)
	default:
		errMsg := fmt.Sprintf("未知的修改类型: %s", modType)
		tm.logger.Error(errMsg, zap.String("nodeName", nodeName))
		return errors.New(errMsg)
	}

	if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		tm.logger.Error("更新节点 Taint 失败", zap.Error(err),
			zap.String("nodeName", nodeName), zap.String("modType", modType))
		return fmt.Errorf("更新节点 %s Taint 失败: %w", nodeName, err)
	}

	tm.logger.Info("更新节点 Taint 成功", zap.String("nodeName", nodeName), zap.String("modType", modType))
	return nil
}

func (tm *taintManager) DrainPods(ctx context.Context, clusterID int, nodeName string) error {
	cluster, err := tm.clusterDao.GetClusterByID(ctx, clusterID)
	if err != nil {
		tm.logger.Error("获取集群信息失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := tm.client.GetKubeClient(cluster.ID)
	if err != nil {
		tm.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		tm.logger.Error("获取节点 Pod 列表失败", zap.Error(err), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点 %s Pod 列表失败: %w", nodeName, err)
	}

	if len(pods.Items) == 0 {
		tm.logger.Info("节点上没有需要驱逐的 Pod", zap.String("nodeName", nodeName))
		return nil
	}

	evictionTemplate := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "Eviction",
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		},
	}

	g, ctx := errgroup.WithContext(ctx)
	var evictedCount int

	for _, pod := range pods.Items {
		pod := pod
		g.Go(func() error {
			eviction := evictionTemplate.DeepCopy()
			eviction.Name = pod.Name
			eviction.Namespace = pod.Namespace

			if err := kubeClient.PolicyV1().Evictions(eviction.Namespace).Evict(ctx, eviction); err != nil {
				tm.logger.Error("驱逐 Pod 失败", zap.Error(err),
					zap.String("nodeName", nodeName), zap.String("podName", pod.Name), zap.String("namespace", pod.Namespace))
				return fmt.Errorf("驱逐 Pod %s/%s 失败: %w", pod.Namespace, pod.Name, err)
			}

			evictedCount++
			tm.logger.Debug("驱逐 Pod 成功",
				zap.String("nodeName", nodeName), zap.String("podName", pod.Name), zap.String("namespace", pod.Namespace))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("在驱逐节点 %s 的 Pod 时遇到错误: %w", nodeName, err)
	}

	tm.logger.Info("节点 Pod 驱逐完成",
		zap.String("nodeName", nodeName), zap.Int("totalPods", len(pods.Items)), zap.Int("evictedPods", evictedCount))

	return nil
}

func (tm *taintManager) GetNodeTaints(ctx context.Context, clusterID int, nodeName string) ([]corev1.Taint, error) {
	cluster, err := tm.clusterDao.GetClusterByID(ctx, clusterID)
	if err != nil {
		tm.logger.Error("获取集群信息失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := tm.client.GetKubeClient(cluster.ID)
	if err != nil {
		tm.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		tm.logger.Error("获取节点信息失败", zap.Error(err), zap.String("nodeName", nodeName))
		return nil, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err)
	}

	return node.Spec.Taints, nil
}

func (tm *taintManager) mergeTaints(existingTaints, newTaints []corev1.Taint) []corev1.Taint {
	result := make([]corev1.Taint, 0, len(existingTaints)+len(newTaints))

	newTaintKeys := make(map[string]struct{})
	for _, taint := range newTaints {
		newTaintKeys[taint.Key] = struct{}{}
	}

	for _, taint := range existingTaints {
		if _, exists := newTaintKeys[taint.Key]; !exists {
			result = append(result, taint)
		}
	}

	result = append(result, newTaints...)
	return result
}

func (tm *taintManager) removeTaints(existingTaints, taintsToRemove []corev1.Taint) []corev1.Taint {
	removeKeys := make(map[string]struct{})
	for _, taint := range taintsToRemove {
		removeKeys[taint.Key] = struct{}{}
	}

	var result []corev1.Taint
	for _, taint := range existingTaints {
		if _, shouldRemove := removeKeys[taint.Key]; !shouldRemove {
			result = append(result, taint)
		}
	}

	return result
}

func (tm *taintManager) updateTaints(existingTaints, newTaints []corev1.Taint) []corev1.Taint {
	// 创建新污点的键映射
	newTaintMap := make(map[string]corev1.Taint)
	for _, taint := range newTaints {
		newTaintMap[taint.Key] = taint
	}

	result := make([]corev1.Taint, 0, len(existingTaints))
	for _, taint := range existingTaints {
		if newTaint, shouldUpdate := newTaintMap[taint.Key]; !shouldUpdate {
			result = append(result, taint)
		} else {
			// 如果存在新的污点，使用新的污点替换旧的
			result = append(result, newTaint)
		}
	}

	// 添加新的污点（不在现有污点中的）
	existingKeys := make(map[string]struct{})
	for _, taint := range existingTaints {
		existingKeys[taint.Key] = struct{}{}
	}

	for _, taint := range newTaints {
		if _, exists := existingKeys[taint.Key]; !exists {
			result = append(result, taint)
		}
	}

	return result
}

func (tm *taintManager) DeleteNodeTaintsByKeys(ctx context.Context, clusterID int, nodeName string, taintKeys []string) error {
	cluster, err := tm.clusterDao.GetClusterByID(ctx, clusterID)
	if err != nil {
		tm.logger.Error("获取集群信息失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取集群信息失败: %w", err)
	}

	kubeClient, err := tm.client.GetKubeClient(cluster.ID)
	if err != nil {
		tm.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("clusterID", clusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		tm.logger.Error("获取节点信息失败", zap.Error(err), zap.String("nodeName", nodeName))
		return fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err)
	}

	// 直接根据键删除污点，不需要解析 YAML
	removeKeys := make(map[string]struct{})
	for _, key := range taintKeys {
		removeKeys[key] = struct{}{}
	}

	var remainingTaints []corev1.Taint
	for _, taint := range node.Spec.Taints {
		if _, shouldRemove := removeKeys[taint.Key]; !shouldRemove {
			remainingTaints = append(remainingTaints, taint)
		}
	}

	node.Spec.Taints = remainingTaints

	if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		tm.logger.Error("删除节点 Taint 失败", zap.Error(err),
			zap.String("nodeName", nodeName), zap.Strings("taintKeys", taintKeys))
		return fmt.Errorf("删除节点 %s Taint 失败: %w", nodeName, err)
	}

	tm.logger.Info("删除节点 Taint 成功", zap.String("nodeName", nodeName), zap.Strings("taintKeys", taintKeys))
	return nil
}
