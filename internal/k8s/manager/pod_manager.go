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
	"io"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodManager Pod 资源管理器，负责 Pod 相关的业务逻辑
// 通过依赖注入接收客户端工厂，实现业务逻辑与客户端创建的解耦
type PodManager interface {
	// Pod 查询
	GetPod(ctx context.Context, clusterID int, namespace, name string) (*corev1.Pod, error)
	GetPodList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.PodList, error)
	GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) (*corev1.PodList, error)

	// Pod 操作
	DeletePod(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// Pod 日志
	GetPodLogs(ctx context.Context, clusterID int, namespace, name string, logOptions *corev1.PodLogOptions) (string, error)

	// 批量操作
	BatchDeletePods(ctx context.Context, clusterID int, namespace string, podNames []string) error
}

type podManager struct {
	clientFactory client.K8sClientFactory
	logger        *zap.Logger
}

// NewPodManager 创建新的 Pod 管理器实例
// 通过构造函数注入客户端工厂依赖
func NewPodManager(clientFactory client.K8sClientFactory, logger *zap.Logger) PodManager {
	return &podManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 私有方法：获取 Kubernetes 客户端
// 封装客户端获取逻辑，统一错误处理
func (p *podManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := p.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}
	return kubeClient, nil
}

// GetPod 获取单个 Pod
func (p *podManager) GetPod(ctx context.Context, clusterID int, namespace, name string) (*corev1.Pod, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Pod 失败: %w", err)
	}

	p.logger.Debug("成功获取 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return pod, nil
}

// GetPodList 获取 Pod 列表
func (p *podManager) GetPodList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.PodList, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取 Pod 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Pod 列表失败: %w", err)
	}

	p.logger.Debug("成功获取 Pod 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(podList.Items)))
	return podList, nil
}

// GetPodsByNodeName 获取指定节点上的 Pod 列表
func (p *podManager) GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) (*corev1.PodList, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	}

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取节点 Pod 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return nil, fmt.Errorf("获取节点 Pod 列表失败: %w", err)
	}

	p.logger.Debug("成功获取节点 Pod 列表",
		zap.Int("clusterID", clusterID),
		zap.String("nodeName", nodeName),
		zap.Int("count", len(pods.Items)))
	return pods, nil
}

// DeletePod 删除 Pod
func (p *podManager) DeletePod(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		p.logger.Error("删除 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Pod 失败: %w", err)
	}

	p.logger.Info("成功删除 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// GetPodLogs 获取 Pod 日志
func (p *podManager) GetPodLogs(ctx context.Context, clusterID int, namespace, name string, logOptions *corev1.PodLogOptions) (string, error) {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return "", err
	}

	podLogRequest := kubeClient.CoreV1().Pods(namespace).GetLogs(name, logOptions)
	podLogs, err := podLogRequest.Stream(ctx)
	if err != nil {
		p.logger.Error("获取 Pod 日志流失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return "", fmt.Errorf("获取 Pod 日志流失败: %w", err)
	}
	defer podLogs.Close()

	logData, err := io.ReadAll(podLogs)
	if err != nil {
		p.logger.Error("读取 Pod 日志数据失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return "", fmt.Errorf("读取 Pod 日志数据失败: %w", err)
	}

	p.logger.Debug("成功获取 Pod 日志",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int("logSize", len(logData)))
	return string(logData), nil
}

// BatchDeletePods 批量删除 Pod
func (p *podManager) BatchDeletePods(ctx context.Context, clusterID int, namespace string, podNames []string) error {
	kubeClient, err := p.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var errors []string
	for _, podName := range podNames {
		err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		if err != nil {
			errorMsg := fmt.Sprintf("删除 Pod %s 失败: %v", podName, err)
			errors = append(errors, errorMsg)
			p.logger.Error("批量删除中的单个 Pod 失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("podName", podName),
				zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("批量删除失败，详情: %s", strings.Join(errors, "; "))
	}

	p.logger.Info("成功批量删除 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(podNames)))
	return nil
}
