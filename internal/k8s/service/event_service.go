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
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventService interface {
	// 获取Event列表
	GetEventList(ctx context.Context, req *model.K8sEventListReq) ([]*model.K8sEventEntity, error)
	GetEventsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sEventEntity, error)

	// 获取Event详情
	GetEvent(ctx context.Context, clusterID int, namespace, name string) (*model.K8sEventEntity, error)

	// 根据对象获取相关事件
	GetEventsByObject(ctx context.Context, req *model.K8sEventByObjectReq) ([]*model.K8sEventEntity, error)
	GetEventsByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sEventEntity, error)
	GetEventsByDeployment(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sEventEntity, error)
	GetEventsByService(ctx context.Context, clusterID int, namespace, serviceName string) ([]*model.K8sEventEntity, error)
	GetEventsByNode(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sEventEntity, error)
	GetEventsByIngress(ctx context.Context, clusterID int, ns, ingressName string) ([]*model.K8sEventEntity, error)

	// 事件统计和分析
	GetEventStatistics(ctx context.Context, req *model.K8sEventStatisticsReq) (*model.K8sEventStatistics, error)
	GetEventTimeline(ctx context.Context, req *model.K8sEventTimelineReq) ([]*model.K8sEventTimelineItem, error)

	// 事件清理
	CleanupOldEvents(ctx context.Context, req *model.K8sEventCleanupReq) (*model.K8sEventCleanupResult, error)
}

type eventService struct {
	dao          dao.ClusterDAO       // 保持对DAO的依赖
	client       client.K8sClient     // 保持向后兼容
	eventManager manager.EventManager // 新的依赖注入
	logger       *zap.Logger
}

// NewEventService 创建新的 EventService 实例
func NewEventService(dao dao.ClusterDAO, client client.K8sClient, eventManager manager.EventManager, logger *zap.Logger) EventService {
	return &eventService{
		dao:          dao,
		client:       client,
		eventManager: eventManager,
		logger:       logger,
	}
}

// GetEventList 获取Event列表
func (e *eventService) GetEventList(ctx context.Context, req *model.K8sEventListReq) ([]*model.K8sEventEntity, error) {
	// 使用 EventManager 获取 Event 列表
	eventList, err := e.eventManager.ListEvents(ctx, req.ClusterID, req.Namespace)
	if err != nil {
		e.logger.Error("获取Event列表失败",
			zap.String("Namespace", req.Namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取Event列表失败: %w", err)
	}

	events := make([]*model.K8sEventEntity, 0, len(eventList.Items))
	for _, event := range eventList.Items {
		eventEntity := e.convertEventToEntity(&event, req.ClusterID)

		// 根据请求参数进行过滤
		if req.EventType != "" && event.Type != req.EventType {
			continue
		}
		if req.Reason != "" && event.Reason != req.Reason {
			continue
		}
		if req.Source != "" && event.Source.Component != req.Source {
			continue
		}
		if req.InvolvedObjectKind != "" && event.InvolvedObject.Kind != req.InvolvedObjectKind {
			continue
		}
		if req.InvolvedObjectName != "" && event.InvolvedObject.Name != req.InvolvedObjectName {
			continue
		}

		// 时间过滤
		if req.LimitDays > 0 {
			limitTime := time.Now().AddDate(0, 0, -req.LimitDays)
			if event.CreationTimestamp.Time.Before(limitTime) {
				continue
			}
		}

		events = append(events, eventEntity)
	}

	// 按时间排序（最新的在前）
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreationTimestamp.After(events[j].CreationTimestamp)
	})

	return events, nil
}

// GetEventsByNamespace 根据命名空间获取Event列表（保持向后兼容）
func (e *eventService) GetEventsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sEventEntity, error) {
	req := &model.K8sEventListReq{
		ClusterID: clusterID,
		Namespace: namespace,
	}
	return e.GetEventList(ctx, req)
}

// GetEvent 获取单个Event详情
func (e *eventService) GetEvent(ctx context.Context, clusterID int, namespace, name string) (*model.K8sEventEntity, error) {
	// 使用 EventManager 获取单个 Event
	event, err := e.eventManager.GetEvent(ctx, clusterID, namespace, name)
	if err != nil {
		e.logger.Error("获取Event详情失败",
			zap.String("Namespace", namespace),
			zap.String("Name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取Event详情失败: %w", err)
	}

	return e.convertEventToEntity(event, clusterID), nil
}

// GetEventsByObject 根据对象获取相关事件
func (e *eventService) GetEventsByObject(ctx context.Context, req *model.K8sEventByObjectReq) ([]*model.K8sEventEntity, error) {
	fieldSelector := "involvedObject.name=" + req.ObjectName + ",involvedObject.kind=" + req.ObjectKind
	if req.ObjectUID != "" {
		fieldSelector += ",involvedObject.uid=" + req.ObjectUID
	}

	eventReq := &model.K8sEventListReq{
		ClusterID:     req.ClusterID,
		Namespace:     req.Namespace,
		FieldSelector: fieldSelector,
		LimitDays:     req.LimitDays,
	}

	return e.GetEventList(ctx, eventReq)
}

// GetEventsByPod 获取Pod相关事件
func (e *eventService) GetEventsByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sEventEntity, error) {
	req := &model.K8sEventByObjectReq{
		ClusterID:  clusterID,
		Namespace:  namespace,
		ObjectName: podName,
		ObjectKind: "Pod",
		LimitDays:  7, // 默认7天内的事件
	}
	return e.GetEventsByObject(ctx, req)
}

// GetEventsByDeployment 获取Deployment相关事件
func (e *eventService) GetEventsByDeployment(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sEventEntity, error) {
	req := &model.K8sEventByObjectReq{
		ClusterID:  clusterID,
		Namespace:  namespace,
		ObjectName: deploymentName,
		ObjectKind: "Deployment",
		LimitDays:  7, // 默认7天内的事件
	}
	return e.GetEventsByObject(ctx, req)
}

// GetEventsByService 获取Service相关事件
func (e *eventService) GetEventsByService(ctx context.Context, clusterID int, namespace, serviceName string) ([]*model.K8sEventEntity, error) {
	req := &model.K8sEventByObjectReq{
		ClusterID:  clusterID,
		Namespace:  namespace,
		ObjectName: serviceName,
		ObjectKind: "Service",
		LimitDays:  7, // 默认7天内的事件
	}
	return e.GetEventsByObject(ctx, req)
}

// GetEventsByNode 获取Node相关事件
func (e *eventService) GetEventsByNode(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sEventEntity, error) {
	req := &model.K8sEventByObjectReq{
		ClusterID:  clusterID,
		Namespace:  "", // Node是集群级别资源，不需要namespace
		ObjectName: nodeName,
		ObjectKind: "Node",
		LimitDays:  7, // 默认7天内的事件
	}
	return e.GetEventsByObject(ctx, req)
}

// GetEventsByIngress 获取Ingress相关事件
func (e *eventService) GetEventsByIngress(ctx context.Context, clusterID int, namespace, ingressName string) ([]*model.K8sEventEntity, error) {
	req := &model.K8sEventByObjectReq{
		ClusterID:  clusterID,
		Namespace:  namespace,
		ObjectName: ingressName,
		ObjectKind: "Ingress",
		LimitDays:  7, // 默认7天内的事件
	}
	return e.GetEventsByObject(ctx, req)
}

// GetEventStatistics 获取事件统计
func (e *eventService) GetEventStatistics(ctx context.Context, req *model.K8sEventStatisticsReq) (*model.K8sEventStatistics, error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	// 获取所有事件
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	stats := &model.K8sEventStatistics{
		TotalEvents:   len(events.Items),
		NormalEvents:  0,
		WarningEvents: 0,
		TopReasons:    []model.EventReasonCount{},
		TopSources:    []model.EventSourceCount{},
	}

	// 统计事件
	reasonCounts := make(map[string]int)
	sourceCounts := make(map[string]int)

	for _, event := range events.Items {
		switch event.Type {
		case "Normal":
			stats.NormalEvents++
		case "Warning":
			stats.WarningEvents++
		}

		// 统计原因
		if event.Reason != "" {
			reasonCounts[event.Reason]++
		}

		// 统计来源
		if event.Source.Component != "" {
			sourceCounts[event.Source.Component]++
		}
	}

	// 转换为排序列表
	for reason, count := range reasonCounts {
		stats.TopReasons = append(stats.TopReasons, model.EventReasonCount{
			Reason: reason,
			Count:  count,
		})
	}

	for source, count := range sourceCounts {
		stats.TopSources = append(stats.TopSources, model.EventSourceCount{
			Source: source,
			Count:  count,
		})
	}

	return stats, nil
}

// GetEventTimeline 获取事件时间线
func (e *eventService) GetEventTimeline(ctx context.Context, req *model.K8sEventTimelineReq) ([]*model.K8sEventTimelineItem, error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	if req.InvolvedObjectName != "" && req.InvolvedObjectKind != "" {
		listOptions.FieldSelector = fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s",
			req.InvolvedObjectName, req.InvolvedObjectKind)
	}

	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, listOptions)
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	var timelineItems []*model.K8sEventTimelineItem
	for _, event := range events.Items {
		timelineItems = append(timelineItems, &model.K8sEventTimelineItem{
			Timestamp: event.LastTimestamp.Time,
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Object:    fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
		})
	}

	// 按时间排序（最新的在前）
	sort.Slice(timelineItems, func(i, j int) bool {
		return timelineItems[i].Timestamp.After(timelineItems[j].Timestamp)
	})

	return timelineItems, nil
}

// CleanupOldEvents 清理旧事件
func (e *eventService) CleanupOldEvents(ctx context.Context, req *model.K8sEventCleanupReq) (*model.K8sEventCleanupResult, error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 获取所有事件
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceOperation, "获取事件列表失败")
	}

	result := &model.K8sEventCleanupResult{
		CleanedCount: 0,
		ErrorCount:   0,
		Errors:       []string{},
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -req.DaysToKeep)

	// 删除旧事件
	for _, event := range events.Items {
		if event.LastTimestamp.Time.Before(cutoffTime) {
			err := kubeClient.CoreV1().Events(req.Namespace).Delete(ctx, event.Name, metav1.DeleteOptions{})
			if err != nil {
				e.logger.Warn("删除事件失败",
					zap.String("event", event.Name),
					zap.Error(err))
				result.ErrorCount++
				result.Errors = append(result.Errors, fmt.Sprintf("删除事件 %s 失败: %v", event.Name, err))
			} else {
				result.CleanedCount++
			}
		}
	}

	e.logger.Info("事件清理完成",
		zap.String("namespace", req.Namespace),
		zap.Int("total", len(events.Items)),
		zap.Int("cleaned", result.CleanedCount),
		zap.Int("failed", result.ErrorCount))

	return result, nil
}

// convertEventToEntity 将Kubernetes Event对象转换为实体模型
func (e *eventService) convertEventToEntity(event *corev1.Event, clusterID int) *model.K8sEventEntity {
	return &model.K8sEventEntity{
		Name:              event.Name,
		Namespace:         event.Namespace,
		ClusterID:         clusterID,
		UID:               string(event.UID),
		Type:              event.Type,
		Reason:            event.Reason,
		Message:           event.Message,
		Source:            event.Source,
		InvolvedObject:    event.InvolvedObject,
		Count:             event.Count,
		FirstTimestamp:    event.FirstTimestamp.Time,
		LastTimestamp:     event.LastTimestamp.Time,
		CreationTimestamp: event.CreationTimestamp.Time,
		Age:               pkg.GetAge(event.CreationTimestamp.Time),
	}
}
