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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventService interface {
	GetEventList(ctx context.Context, req *model.GetEventListReq) (model.ListResp[*model.K8sEvent], error)
	GetEvent(ctx context.Context, req *model.GetEventDetailReq) (*model.K8sEvent, error)
	GetEventsByPod(ctx context.Context, req *model.GetEventsByPodReq) (model.ListResp[*model.K8sEvent], error)
	GetEventsByDeployment(ctx context.Context, req *model.GetEventsByDeploymentReq) (model.ListResp[*model.K8sEvent], error)
	GetEventsByService(ctx context.Context, req *model.GetEventsByServiceReq) (model.ListResp[*model.K8sEvent], error)
	GetEventsByNode(ctx context.Context, req *model.GetEventsByNodeReq) (model.ListResp[*model.K8sEvent], error)
	GetEventStatistics(ctx context.Context, req *model.GetEventStatisticsReq) (*model.EventStatistics, error)
	GetEventSummary(ctx context.Context, req *model.GetEventSummaryReq) (*model.EventSummary, error)
	GetEventTimeline(ctx context.Context, req *model.GetEventTimelineReq) (model.ListResp[*model.EventTimelineItem], error)
	GetEventTrends(ctx context.Context, req *model.GetEventTrendsReq) (model.ListResp[*model.EventTrend], error)
	GetEventGroupData(ctx context.Context, req *model.GetEventGroupDataReq) (model.ListResp[*model.EventGroupData], error)
	DeleteEvent(ctx context.Context, req *model.DeleteEventReq) error
	CleanupOldEvents(ctx context.Context, req *model.CleanupOldEventsReq) error
}

type eventService struct {
	eventManager manager.EventManager // 新的依赖注入
	logger       *zap.Logger
}

// NewEventService 创建新的 EventService 实例
func NewEventService(eventManager manager.EventManager, logger *zap.Logger) EventService {
	return &eventService{
		eventManager: eventManager,
		logger:       logger,
	}
}

// GetEventList 获取Event列表
func (e *eventService) GetEventList(ctx context.Context, req *model.GetEventListReq) (model.ListResp[*model.K8sEvent], error) {
	// 使用 EventManager 获取 Event 列表和总数
	eventList, total, err := e.eventManager.ListEventsWithTotal(ctx, req.ClusterID, req.Namespace)
	if err != nil {
		e.logger.Error("获取Event列表失败",
			zap.String("Namespace", req.Namespace),
			zap.Error(err))
		return model.ListResp[*model.K8sEvent]{}, fmt.Errorf("获取Event列表失败: %w", err)
	}

	events := make([]*model.K8sEvent, 0, len(eventList.Items))
	for _, event := range eventList.Items {
		eventEntity := e.eventManager.ConvertEventToK8sEvent(&event, req.ClusterID)

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
		return events[i].LastTimestamp.After(events[j].LastTimestamp)
	})

	// 如果没有过滤条件，使用原始total；否则使用过滤后的数量
	filteredTotal := total
	if req.EventType != "" || req.Reason != "" || req.Source != "" ||
		req.InvolvedObjectKind != "" || req.InvolvedObjectName != "" || req.LimitDays > 0 {
		filteredTotal = int64(len(events))
	}

	return model.ListResp[*model.K8sEvent]{Items: events, Total: filteredTotal}, nil
}

// GetEvent 获取单个Event详情
func (e *eventService) GetEvent(ctx context.Context, req *model.GetEventDetailReq) (*model.K8sEvent, error) {
	// 使用 EventManager 获取单个 Event
	event, err := e.eventManager.GetEvent(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		e.logger.Error("获取Event详情失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return nil, fmt.Errorf("获取Event详情失败: %w", err)
	}

	return e.eventManager.ConvertEventToK8sEvent(event, req.ClusterID), nil
}

// GetEventsByObject 根据对象获取相关事件
func (e *eventService) GetEventsByObject(ctx context.Context, clusterID int, namespace, objectKind, objectName, objectUID string, limitDays int) (model.ListResp[*model.K8sEvent], error) {
	fieldSelector := "involvedObject.name=" + objectName + ",involvedObject.kind=" + objectKind
	if objectUID != "" {
		fieldSelector += ",involvedObject.uid=" + objectUID
	}

	eventReq := &model.GetEventListReq{
		ClusterID:     clusterID,
		Namespace:     namespace,
		FieldSelector: fieldSelector,
		LimitDays:     limitDays,
	}

	return e.GetEventList(ctx, eventReq)
}

// GetEventsByPod 获取Pod相关事件
func (e *eventService) GetEventsByPod(ctx context.Context, req *model.GetEventsByPodReq) (model.ListResp[*model.K8sEvent], error) {
	return e.GetEventsByObject(ctx, req.ClusterID, req.Namespace, "Pod", req.PodName, "", 7)
}

// GetEventsByDeployment 获取Deployment相关事件
func (e *eventService) GetEventsByDeployment(ctx context.Context, req *model.GetEventsByDeploymentReq) (model.ListResp[*model.K8sEvent], error) {
	return e.GetEventsByObject(ctx, req.ClusterID, req.Namespace, "Deployment", req.DeploymentName, "", 7)
}

// GetEventsByService 获取Service相关事件
func (e *eventService) GetEventsByService(ctx context.Context, req *model.GetEventsByServiceReq) (model.ListResp[*model.K8sEvent], error) {
	return e.GetEventsByObject(ctx, req.ClusterID, req.Namespace, "Service", req.ServiceName, "", 7)
}

// GetEventsByNode 获取Node相关事件
func (e *eventService) GetEventsByNode(ctx context.Context, req *model.GetEventsByNodeReq) (model.ListResp[*model.K8sEvent], error) {
	return e.GetEventsByObject(ctx, req.ClusterID, "", "Node", req.NodeName, "", 7)
}

// GetEventStatistics 获取事件统计
func (e *eventService) GetEventStatistics(ctx context.Context, req *model.GetEventStatisticsReq) (*model.EventStatistics, error) {
	// 参数验证
	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", req.ClusterID)
	}

	// 调用EventManager的方法
	return e.eventManager.GetEventStatistics(ctx, req.ClusterID, req.Namespace, req.StartTime, req.EndTime)
}

// GetEventSummary 获取事件汇总
func (e *eventService) GetEventSummary(ctx context.Context, req *model.GetEventSummaryReq) (*model.EventSummary, error) {
	// 参数验证
	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", req.ClusterID)
	}

	// 调用EventManager的方法
	return e.eventManager.GetEventSummary(ctx, req.ClusterID, req.Namespace, req.StartTime, req.EndTime)
}

// GetEventTimeline 获取事件时间线
func (e *eventService) GetEventTimeline(ctx context.Context, req *model.GetEventTimelineReq) (model.ListResp[*model.EventTimelineItem], error) {
	// 参数验证
	if req.ClusterID <= 0 {
		return model.ListResp[*model.EventTimelineItem]{}, fmt.Errorf("无效的集群ID: %d", req.ClusterID)
	}

	// 调用EventManager的方法
	timelineItems, err := e.eventManager.GetEventTimeline(ctx, req.ClusterID, req.Namespace, req.ObjectKind, req.ObjectName)
	if err != nil {
		return model.ListResp[*model.EventTimelineItem]{}, err
	}

	return model.ListResp[*model.EventTimelineItem]{Items: timelineItems, Total: int64(len(timelineItems))}, nil
}

// GetEventTrends 获取事件趋势
func (e *eventService) GetEventTrends(ctx context.Context, req *model.GetEventTrendsReq) (model.ListResp[*model.EventTrend], error) {
	// 参数验证
	if req.ClusterID <= 0 {
		return model.ListResp[*model.EventTrend]{}, fmt.Errorf("无效的集群ID: %d", req.ClusterID)
	}

	// 调用EventManager的方法
	trends, err := e.eventManager.GetEventTrends(ctx, req.ClusterID, req.Namespace, req.EventType, req.Interval, req.StartTime, req.EndTime)
	if err != nil {
		return model.ListResp[*model.EventTrend]{}, err
	}

	return model.ListResp[*model.EventTrend]{Items: trends, Total: int64(len(trends))}, nil
}

// GetEventGroupData 获取事件分组数据
func (e *eventService) GetEventGroupData(ctx context.Context, req *model.GetEventGroupDataReq) (model.ListResp[*model.EventGroupData], error) {
	// 参数验证
	if req.ClusterID <= 0 {
		return model.ListResp[*model.EventGroupData]{}, fmt.Errorf("无效的集群ID: %d", req.ClusterID)
	}

	// 调用EventManager的方法
	groupData, err := e.eventManager.GetEventGroupData(ctx, req.ClusterID, req.Namespace, req.GroupBy, req.StartTime, req.EndTime, req.Limit)
	if err != nil {
		return model.ListResp[*model.EventGroupData]{}, err
	}

	return model.ListResp[*model.EventGroupData]{Items: groupData, Total: int64(len(groupData))}, nil
}

// DeleteEvent 删除事件
func (e *eventService) DeleteEvent(ctx context.Context, req *model.DeleteEventReq) error {
	return e.eventManager.DeleteEvent(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
}

// CleanupOldEvents 清理旧事件
func (e *eventService) CleanupOldEvents(ctx context.Context, req *model.CleanupOldEventsReq) error {
	// 参数验证
	if req.ClusterID <= 0 {
		return fmt.Errorf("无效的集群ID: %d", req.ClusterID)
	}
	if req.BeforeTime.IsZero() {
		return fmt.Errorf("清理截止时间不能为空")
	}

	// 调用EventManager的方法
	return e.eventManager.CleanupOldEvents(ctx, req.ClusterID, req.Namespace, req.BeforeTime)
}
