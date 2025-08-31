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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventService interface {
	GetEventList(ctx context.Context, req *model.GetEventListReq) (model.ListResp[*model.K8sEvent], error)
	GetEventsByNamespace(ctx context.Context, clusterID int, namespace string) (model.ListResp[*model.K8sEvent], error)
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
		eventEntity := e.convertEventToK8sEvent(&event, req.ClusterID)

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

// GetEventsByNamespace 根据命名空间获取Event列表（保持向后兼容）
func (e *eventService) GetEventsByNamespace(ctx context.Context, clusterID int, namespace string) (model.ListResp[*model.K8sEvent], error) {
	req := &model.GetEventListReq{
		ClusterID: clusterID,
		Namespace: namespace,
	}
	return e.GetEventList(ctx, req)
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

	return e.convertEventToK8sEvent(event, req.ClusterID), nil
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

	summary := &model.EventSummary{
		TotalEvents:   int64(len(events.Items)),
		UniqueEvents:  0,
		WarningEvents: 0,
		NormalEvents:  0,
		Distribution:  make(map[string]int64),
		TopReasons:    []model.CountItem{},
		TopObjects:    []model.CountItem{},
	}

	// 统计事件
	reasonCounts := make(map[string]int)
	sourceCounts := make(map[string]int)

	for _, event := range events.Items {
		switch event.Type {
		case "Normal":
			summary.NormalEvents++
		case "Warning":
			summary.WarningEvents++
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

	// 转换为排序列表 - Top原因
	for reason, count := range reasonCounts {
		percentage := float64(count) / float64(summary.TotalEvents) * 100
		summary.TopReasons = append(summary.TopReasons, model.CountItem{
			Name:       reason,
			Count:      int64(count),
			Percentage: percentage,
		})
	}

	// 填充分布信息
	summary.Distribution["Normal"] = summary.NormalEvents
	summary.Distribution["Warning"] = summary.WarningEvents

	stats := &model.EventStatistics{
		TimeRange: model.TimeRange{
			Start: req.StartTime,
			End:   req.EndTime,
		},
		Summary:   *summary,
		GroupData: []model.EventGroupData{},
		Trends:    []model.EventTrend{},
	}

	return stats, nil
}

// GetEventSummary 获取事件汇总
func (e *eventService) GetEventSummary(ctx context.Context, req *model.GetEventSummaryReq) (*model.EventSummary, error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	// 获取所有事件
	listOptions := metav1.ListOptions{}
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, listOptions)
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	summary := &model.EventSummary{
		TotalEvents:   int64(len(events.Items)),
		UniqueEvents:  0,
		WarningEvents: 0,
		NormalEvents:  0,
		Distribution:  make(map[string]int64),
		TopReasons:    []model.CountItem{},
		TopObjects:    []model.CountItem{},
	}

	// 统计数据
	uniqueEvents := make(map[string]bool)
	reasonCounts := make(map[string]int64)
	objectCounts := make(map[string]int64)

	for _, event := range events.Items {
		// 时间过滤
		if !req.StartTime.IsZero() && event.CreationTimestamp.Time.Before(req.StartTime) {
			continue
		}
		if !req.EndTime.IsZero() && event.CreationTimestamp.Time.After(req.EndTime) {
			continue
		}

		// 统计事件类型
		switch event.Type {
		case "Normal":
			summary.NormalEvents++
		case "Warning":
			summary.WarningEvents++
		}

		// 统计唯一事件
		eventKey := fmt.Sprintf("%s/%s/%s", event.Namespace, event.InvolvedObject.Kind, event.InvolvedObject.Name)
		uniqueEvents[eventKey] = true

		// 统计原因
		if event.Reason != "" {
			reasonCounts[event.Reason]++
		}

		// 统计对象
		objectKey := fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)
		objectCounts[objectKey]++
	}

	summary.UniqueEvents = int64(len(uniqueEvents))

	// 生成Top原因
	type reasonCount struct {
		reason string
		count  int64
	}
	var reasons []reasonCount
	for reason, count := range reasonCounts {
		reasons = append(reasons, reasonCount{reason: reason, count: count})
	}
	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].count > reasons[j].count
	})

	for i, rc := range reasons {
		if i >= 10 { // 限制前10个
			break
		}
		percentage := float64(rc.count) / float64(summary.TotalEvents) * 100
		summary.TopReasons = append(summary.TopReasons, model.CountItem{
			Name:       rc.reason,
			Count:      rc.count,
			Percentage: percentage,
		})
	}

	// 生成Top对象
	type objectCount struct {
		object string
		count  int64
	}
	var objects []objectCount
	for object, count := range objectCounts {
		objects = append(objects, objectCount{object: object, count: count})
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].count > objects[j].count
	})

	for i, oc := range objects {
		if i >= 10 { // 限制前10个
			break
		}
		percentage := float64(oc.count) / float64(summary.TotalEvents) * 100
		summary.TopObjects = append(summary.TopObjects, model.CountItem{
			Name:       oc.object,
			Count:      oc.count,
			Percentage: percentage,
		})
	}

	// 分布统计
	summary.Distribution["Normal"] = summary.NormalEvents
	summary.Distribution["Warning"] = summary.WarningEvents

	return summary, nil
}

// GetEventTimeline 获取事件时间线
func (e *eventService) GetEventTimeline(ctx context.Context, req *model.GetEventTimelineReq) (model.ListResp[*model.EventTimelineItem], error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return model.ListResp[*model.EventTimelineItem]{}, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	if req.ObjectName != "" && req.ObjectKind != "" {
		listOptions.FieldSelector = fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s",
			req.ObjectName, req.ObjectKind)
	}

	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, listOptions)
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return model.ListResp[*model.EventTimelineItem]{}, fmt.Errorf("获取事件列表失败: %w", err)
	}

	var timelineItems []*model.EventTimelineItem
	for _, event := range events.Items {
		timelineItems = append(timelineItems, &model.EventTimelineItem{
			Timestamp: event.LastTimestamp.Time,
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Count:     int64(event.Count),
		})
	}

	// 按时间排序（最新的在前）
	sort.Slice(timelineItems, func(i, j int) bool {
		return timelineItems[i].Timestamp.After(timelineItems[j].Timestamp)
	})

	return model.ListResp[*model.EventTimelineItem]{Items: timelineItems, Total: int64(len(timelineItems))}, nil
}

// GetEventTrends 获取事件趋势
func (e *eventService) GetEventTrends(ctx context.Context, req *model.GetEventTrendsReq) (model.ListResp[*model.EventTrend], error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return model.ListResp[*model.EventTrend]{}, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, listOptions)
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return model.ListResp[*model.EventTrend]{}, fmt.Errorf("获取事件列表失败: %w", err)
	}

	// 解析时间间隔
	var intervalDuration time.Duration
	switch req.Interval {
	case "1m":
		intervalDuration = time.Minute
	case "5m":
		intervalDuration = 5 * time.Minute
	case "15m":
		intervalDuration = 15 * time.Minute
	case "1h":
		intervalDuration = time.Hour
	case "1d":
		intervalDuration = 24 * time.Hour
	default:
		intervalDuration = time.Hour // 默认1小时
	}

	// 计算时间范围
	now := time.Now()
	startTime := req.StartTime
	endTime := req.EndTime
	if startTime.IsZero() {
		startTime = now.Add(-24 * time.Hour) // 默认过去24小时
	}
	if endTime.IsZero() {
		endTime = now
	}

	// 创建时间桶
	buckets := make(map[time.Time]int64)
	for t := startTime.Truncate(intervalDuration); t.Before(endTime); t = t.Add(intervalDuration) {
		buckets[t] = 0
	}

	// 统计事件
	for _, event := range events.Items {
		eventTime := event.LastTimestamp.Time
		if eventTime.Before(startTime) || eventTime.After(endTime) {
			continue
		}

		// 事件类型过滤
		if req.EventType != "" && event.Type != req.EventType {
			continue
		}

		// 找到对应的时间桶
		bucketTime := eventTime.Truncate(intervalDuration)
		buckets[bucketTime]++
	}

	// 转换为趋势数据
	var trends []*model.EventTrend
	for timestamp, count := range buckets {
		trends = append(trends, &model.EventTrend{
			Timestamp: timestamp,
			Count:     count,
			Type:      req.EventType,
		})
	}

	// 按时间排序
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Timestamp.Before(trends[j].Timestamp)
	})

	return model.ListResp[*model.EventTrend]{Items: trends, Total: int64(len(trends))}, nil
}

// GetEventGroupData 获取事件分组数据
func (e *eventService) GetEventGroupData(ctx context.Context, req *model.GetEventGroupDataReq) (model.ListResp[*model.EventGroupData], error) {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return model.ListResp[*model.EventGroupData]{}, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, listOptions)
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return model.ListResp[*model.EventGroupData]{}, fmt.Errorf("获取事件列表失败: %w", err)
	}

	// 创建分组映射
	groups := make(map[string][]corev1.Event)

	for _, event := range events.Items {
		// 时间过滤
		if !req.StartTime.IsZero() && event.CreationTimestamp.Time.Before(req.StartTime) {
			continue
		}
		if !req.EndTime.IsZero() && event.CreationTimestamp.Time.After(req.EndTime) {
			continue
		}

		// 根据分组方式确定分组键
		var groupKey string
		switch req.GroupBy {
		case "type":
			groupKey = event.Type
		case "reason":
			groupKey = event.Reason
		case "object":
			groupKey = fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)
		case "severity":
			// 根据事件类型和原因判断严重程度
			if event.Type == "Warning" {
				groupKey = "High"
			} else {
				groupKey = "Low"
			}
		default:
			groupKey = event.Type
		}

		groups[groupKey] = append(groups[groupKey], event)
	}

	// 转换为分组数据
	var groupData []*model.EventGroupData
	for group, eventList := range groups {
		// 转换事件为K8sEvent格式（如果需要包含事件样本）
		var k8sEvents []model.K8sEvent
		for i, event := range eventList {
			if req.Limit > 0 && i >= req.Limit {
				break
			}
			k8sEvents = append(k8sEvents, model.K8sEvent{
				Name:           event.Name,
				Namespace:      event.Namespace,
				UID:            string(event.UID),
				Message:        event.Message,
				FirstTimestamp: event.FirstTimestamp.Time,
				LastTimestamp:  event.LastTimestamp.Time,
				Count:          int64(event.Count),
				InvolvedObject: model.InvolvedObject{
					Kind:       event.InvolvedObject.Kind,
					Name:       event.InvolvedObject.Name,
					Namespace:  event.InvolvedObject.Namespace,
					UID:        string(event.InvolvedObject.UID),
					APIVersion: event.InvolvedObject.APIVersion,
					FieldPath:  event.InvolvedObject.FieldPath,
				},
				Source: model.EventSource{
					Component: event.Source.Component,
					Host:      event.Source.Host,
				},
			})
		}

		groupData = append(groupData, &model.EventGroupData{
			Group:  group,
			Count:  int64(len(eventList)),
			Events: k8sEvents,
		})
	}

	// 按计数排序
	sort.Slice(groupData, func(i, j int) bool {
		return groupData[i].Count > groupData[j].Count
	})

	return model.ListResp[*model.EventGroupData]{Items: groupData, Total: int64(len(groupData))}, nil
}

// DeleteEvent 删除事件
func (e *eventService) DeleteEvent(ctx context.Context, req *model.DeleteEventReq) error {
	return e.eventManager.DeleteEvent(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
}

// CleanupOldEvents 清理旧事件
func (e *eventService) CleanupOldEvents(ctx context.Context, req *model.CleanupOldEventsReq) error {
	kubeClient, err := e.client.GetKubeClient(req.ClusterID)
	if err != nil {
		e.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 获取所有事件
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		e.logger.Error("获取事件列表失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "获取事件列表失败")
	}

	type CleanupResult struct {
		CleanedCount int      `json:"cleaned_count"`
		ErrorCount   int      `json:"error_count"`
		Errors       []string `json:"errors"`
	}

	result := &CleanupResult{
		CleanedCount: 0,
		ErrorCount:   0,
		Errors:       []string{},
	}

	// 使用请求中的截止时间
	cutoffTime := req.BeforeTime

	// 删除旧事件
	for _, event := range events.Items {
		if event.LastTimestamp.Time.Before(cutoffTime) {
			err := kubeClient.CoreV1().Events(req.Namespace).Delete(ctx, event.Name, metav1.DeleteOptions{})
			if err != nil {
				// 如果事件已经不存在（NotFound错误），则认为删除成功
				if errors.IsNotFound(err) {
					e.logger.Debug("事件已经不存在，跳过删除",
						zap.String("event", event.Name))
					result.CleanedCount++
				} else {
					// 其他类型的错误才记录为失败
					e.logger.Warn("删除事件失败",
						zap.String("event", event.Name),
						zap.Error(err))
					result.ErrorCount++
					result.Errors = append(result.Errors, fmt.Sprintf("删除事件 %s 失败: %v", event.Name, err))
				}
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

	if result.ErrorCount > 0 {
		return fmt.Errorf("清理过程中出现 %d 个错误: %v", result.ErrorCount, result.Errors)
	}
	return nil
}

// convertEventToK8sEvent 将Kubernetes Event对象转换为K8sEvent模型
func (e *eventService) convertEventToK8sEvent(event *corev1.Event, clusterID int) *model.K8sEvent {
	// 转换事件类型
	var eventType model.EventType
	if event.Type == "Warning" {
		eventType = model.EventTypeWarning
	} else {
		eventType = model.EventTypeNormal
	}

	// 转换事件原因
	eventReason := model.EventReasonOther // 默认值
	switch event.Reason {
	case "BackOff":
		eventReason = model.EventReasonBackOff
	case "Pulled":
		eventReason = model.EventReasonPulled
	case "Created":
		eventReason = model.EventReasonCreated
	case "Deleted":
		eventReason = model.EventReasonDeleted
	case "Updated":
		eventReason = model.EventReasonUpdated
	case "Started":
		eventReason = model.EventReasonStarted
	case "Stopped":
		eventReason = model.EventReasonStopped
	case "Failed":
		eventReason = model.EventReasonFailed
	case "Succeeded":
		eventReason = model.EventReasonSucceeded
	case "Unknown":
		eventReason = model.EventReasonUnknown
	case "Warning":
		eventReason = model.EventReasonWarning
	case "Error":
		eventReason = model.EventReasonError
	case "Fatal":
		eventReason = model.EventReasonFatal
	case "Panic":
		eventReason = model.EventReasonPanic
	case "Timeout":
		eventReason = model.EventReasonTimeout
	case "Cancelled":
		eventReason = model.EventReasonCancelled
	case "Interrupted":
		eventReason = model.EventReasonInterrupted
	case "Aborted":
		eventReason = model.EventReasonAborted
	case "Ignored":
		eventReason = model.EventReasonIgnored
	default:
		eventReason = model.EventReasonOther
	}

	// 确定严重程度
	var severity model.EventSeverity
	if event.Type == "Warning" {
		severity = model.EventSeverityHigh
	} else {
		severity = model.EventSeverityLow
	}

	return &model.K8sEvent{
		Name:           event.Name,
		Namespace:      event.Namespace,
		UID:            string(event.UID),
		ClusterID:      clusterID,
		Type:           eventType,
		Reason:         eventReason,
		Message:        event.Message,
		Severity:       severity,
		FirstTimestamp: event.FirstTimestamp.Time,
		LastTimestamp:  event.LastTimestamp.Time,
		Count:          int64(event.Count),
		InvolvedObject: model.InvolvedObject{
			Kind:       event.InvolvedObject.Kind,
			Name:       event.InvolvedObject.Name,
			Namespace:  event.InvolvedObject.Namespace,
			UID:        string(event.InvolvedObject.UID),
			APIVersion: event.InvolvedObject.APIVersion,
			FieldPath:  event.InvolvedObject.FieldPath,
		},
		Source: model.EventSource{
			Component: event.Source.Component,
			Host:      event.Source.Host,
		},
		Action:             event.Action,
		ReportingComponent: event.Source.Component, // 使用Source.Component作为ReportingComponent
		ReportingInstance:  event.Source.Host,      // 使用Source.Host作为ReportingInstance
		Labels:             event.Labels,
		Annotations:        event.Annotations,
	}
}
