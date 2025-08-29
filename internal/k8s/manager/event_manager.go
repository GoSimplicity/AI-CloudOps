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
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventManager Event管理器接口
type EventManager interface {
	// 基础操作
	GetEvent(ctx context.Context, clusterID int, namespace, name string) (*corev1.Event, error)
	ListEvents(ctx context.Context, clusterID int, namespace string) (*corev1.EventList, error)
	ListAllEvents(ctx context.Context, clusterID int) (*corev1.EventList, error)
	DeleteEvent(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error

	// 业务功能
	ListEventsByObject(ctx context.Context, clusterID int, namespace string, objectKind, objectName string) (*corev1.EventList, error)
	ListEventsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.EventList, error)
	ListEventsByFieldSelector(ctx context.Context, clusterID int, namespace string, fieldSelector string) (*corev1.EventList, error)
	ListRecentEvents(ctx context.Context, clusterID int, namespace string, limitSeconds int64) (*corev1.EventList, error)
}

// eventManager Event管理器实现
type eventManager struct {
	client client.K8sClient
	logger *zap.Logger
}

// NewEventManager 创建Event管理器
func NewEventManager(client client.K8sClient, logger *zap.Logger) EventManager {
	return &eventManager{
		client: client,
		logger: logger,
	}
}

// GetEvent 获取单个Event
func (m *eventManager) GetEvent(ctx context.Context, clusterID int, namespace, name string) (*corev1.Event, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	event, err := clientset.CoreV1().Events(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取Event失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return nil, fmt.Errorf("获取Event %s/%s 失败: %w", namespace, name, err)
	}

	return event, nil
}

// ListEvents 获取指定命名空间的Event列表
func (m *eventManager) ListEvents(ctx context.Context, clusterID int, namespace string) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取Event列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Event列表失败: %w", err)
	}

	return events, nil
}

// ListAllEvents 获取所有命名空间的Event列表
func (m *eventManager) ListAllEvents(ctx context.Context, clusterID int) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	events, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取所有Event列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取所有Event列表失败: %w", err)
	}

	return events, nil
}

// DeleteEvent 删除Event
func (m *eventManager) DeleteEvent(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	err = clientset.CoreV1().Events(namespace).Delete(ctx, name, options)
	if err != nil {
		m.logger.Error("删除Event失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		return fmt.Errorf("删除Event %s/%s 失败: %w", namespace, name, err)
	}

	m.logger.Info("成功删除Event",
		zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
	return nil
}

// ListEventsByObject 根据对象获取Event列表
func (m *eventManager) ListEventsByObject(ctx context.Context, clusterID int, namespace string, objectKind, objectName string) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	fieldSelector := fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", objectKind, objectName)
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		m.logger.Error("根据对象获取Event列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace),
			zap.String("object_kind", objectKind), zap.String("object_name", objectName))
		return nil, fmt.Errorf("根据对象获取Event列表失败: %w", err)
	}

	return events, nil
}

// ListEventsBySelector 根据标签选择器获取Event列表
func (m *eventManager) ListEventsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("根据选择器获取Event列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("selector", selector))
		return nil, fmt.Errorf("根据选择器获取Event列表失败: %w", err)
	}

	return events, nil
}

// ListEventsByFieldSelector 根据字段选择器获取Event列表
func (m *eventManager) ListEventsByFieldSelector(ctx context.Context, clusterID int, namespace string, fieldSelector string) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := metav1.ListOptions{}
	if fieldSelector != "" {
		listOptions.FieldSelector = fieldSelector
	}

	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("根据字段选择器获取Event列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("field_selector", fieldSelector))
		return nil, fmt.Errorf("根据字段选择器获取Event列表失败: %w", err)
	}

	return events, nil
}

// ListRecentEvents 获取最近的Event列表
func (m *eventManager) ListRecentEvents(ctx context.Context, clusterID int, namespace string, limitSeconds int64) (*corev1.EventList, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	listOptions := metav1.ListOptions{}
	if limitSeconds > 0 {
		listOptions.TimeoutSeconds = &limitSeconds
	}

	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取最近Event列表失败", zap.Error(err),
			zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.Int64("limit_seconds", limitSeconds))
		return nil, fmt.Errorf("获取最近Event列表失败: %w", err)
	}

	return events, nil
}
