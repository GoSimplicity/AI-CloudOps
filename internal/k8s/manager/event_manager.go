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
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventManager interface {
	GetEvent(ctx context.Context, clusterID int, namespace, name string) (*corev1.Event, error)
	ListEvents(ctx context.Context, clusterID int, namespace string) (*corev1.EventList, error)
	ListEventsWithTotal(ctx context.Context, clusterID int, namespace string) (*corev1.EventList, int64, error)
	ListAllEvents(ctx context.Context, clusterID int) (*corev1.EventList, error)
	ListAllEventsWithTotal(ctx context.Context, clusterID int) (*corev1.EventList, int64, error)
	DeleteEvent(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error

	// 业务功能
	ListEventsByObject(ctx context.Context, clusterID int, namespace string, objectKind, objectName string) (*corev1.EventList, error)
	ListEventsByObjectWithTotal(ctx context.Context, clusterID int, namespace string, objectKind, objectName string) (*corev1.EventList, int64, error)
	ListEventsBySelector(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.EventList, error)
	ListEventsBySelectorWithTotal(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.EventList, int64, error)
	ListEventsByFieldSelector(ctx context.Context, clusterID int, namespace string, fieldSelector string) (*corev1.EventList, error)
	ListEventsByFieldSelectorWithTotal(ctx context.Context, clusterID int, namespace string, fieldSelector string) (*corev1.EventList, int64, error)
	ListRecentEvents(ctx context.Context, clusterID int, namespace string, limitSeconds int64) (*corev1.EventList, error)
	ListRecentEventsWithTotal(ctx context.Context, clusterID int, namespace string, limitSeconds int64) (*corev1.EventList, int64, error)

	// 高级业务功能
	GetEventStatistics(ctx context.Context, clusterID int, namespace string, startTime, endTime time.Time) (*model.EventStatistics, error)
	GetEventSummary(ctx context.Context, clusterID int, namespace string, startTime, endTime time.Time) (*model.EventSummary, error)
	GetEventTimeline(ctx context.Context, clusterID int, namespace, objectKind, objectName string) ([]*model.EventTimelineItem, error)
	GetEventTrends(ctx context.Context, clusterID int, namespace, eventType, interval string, startTime, endTime time.Time) ([]*model.EventTrend, error)
	GetEventGroupData(ctx context.Context, clusterID int, namespace, groupBy string, startTime, endTime time.Time, limit int) ([]*model.EventGroupData, error)
	CleanupOldEvents(ctx context.Context, clusterID int, namespace string, beforeTime time.Time) error
	ConvertEventToK8sEvent(event *corev1.Event, clusterID int) *model.K8sEvent
}

type eventManager struct {
	client client.K8sClient
	logger *zap.Logger
}

func NewEventManager(client client.K8sClient, logger *zap.Logger) EventManager {
	return &eventManager{
		client: client,
		logger: logger,
	}
}

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

func (m *eventManager) ListEventsWithTotal(ctx context.Context, clusterID int, namespace string) (*corev1.EventList, int64, error) {
	events, err := m.ListEvents(ctx, clusterID, namespace)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(events.Items))
	return events, total, nil
}

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

func (m *eventManager) ListAllEventsWithTotal(ctx context.Context, clusterID int) (*corev1.EventList, int64, error) {
	events, err := m.ListAllEvents(ctx, clusterID)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(events.Items))
	return events, total, nil
}

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

func (m *eventManager) ListEventsByObjectWithTotal(ctx context.Context, clusterID int, namespace string, objectKind, objectName string) (*corev1.EventList, int64, error) {
	events, err := m.ListEventsByObject(ctx, clusterID, namespace, objectKind, objectName)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(events.Items))
	return events, total, nil
}

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

func (m *eventManager) ListEventsBySelectorWithTotal(ctx context.Context, clusterID int, namespace string, selector string) (*corev1.EventList, int64, error) {
	events, err := m.ListEventsBySelector(ctx, clusterID, namespace, selector)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(events.Items))
	return events, total, nil
}

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

func (m *eventManager) ListEventsByFieldSelectorWithTotal(ctx context.Context, clusterID int, namespace string, fieldSelector string) (*corev1.EventList, int64, error) {
	events, err := m.ListEventsByFieldSelector(ctx, clusterID, namespace, fieldSelector)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(events.Items))
	return events, total, nil
}

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

func (m *eventManager) ListRecentEventsWithTotal(ctx context.Context, clusterID int, namespace string, limitSeconds int64) (*corev1.EventList, int64, error) {
	events, err := m.ListRecentEvents(ctx, clusterID, namespace, limitSeconds)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(events.Items))
	return events, total, nil
}

func (m *eventManager) GetEventStatistics(ctx context.Context, clusterID int, namespace string, startTime, endTime time.Time) (*model.EventStatistics, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取所有事件
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取事件列表失败", zap.Error(err))
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
			Start: startTime,
			End:   endTime,
		},
		Summary:   *summary,
		GroupData: []model.EventGroupData{},
		Trends:    []model.EventTrend{},
	}

	return stats, nil
}

func (m *eventManager) GetEventSummary(ctx context.Context, clusterID int, namespace string, startTime, endTime time.Time) (*model.EventSummary, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取所有事件
	listOptions := metav1.ListOptions{}
	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取事件列表失败", zap.Error(err))
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
		if !startTime.IsZero() && event.CreationTimestamp.Time.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && event.CreationTimestamp.Time.After(endTime) {
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

func (m *eventManager) GetEventTimeline(ctx context.Context, clusterID int, namespace, objectKind, objectName string) ([]*model.EventTimelineItem, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	if objectName != "" && objectKind != "" {
		listOptions.FieldSelector = fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s",
			objectName, objectKind)
	}

	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
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

	return timelineItems, nil
}

func (m *eventManager) GetEventTrends(ctx context.Context, clusterID int, namespace, eventType, interval string, startTime, endTime time.Time) ([]*model.EventTrend, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	// 解析时间间隔
	var intervalDuration time.Duration
	switch interval {
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
		intervalDuration = time.Hour
	}

	// 计算时间范围
	now := time.Now()
	if startTime.IsZero() {
		startTime = now.Add(-24 * time.Hour)
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
		if eventType != "" && event.Type != eventType {
			continue
		}

		// 找到对应的时间桶
		bucketTime := eventTime.Truncate(intervalDuration)
		buckets[bucketTime]++
	}

	var trends []*model.EventTrend
	for timestamp, count := range buckets {
		trends = append(trends, &model.EventTrend{
			Timestamp: timestamp,
			Count:     count,
			Type:      eventType,
		})
	}

	// 按时间排序
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Timestamp.Before(trends[j].Timestamp)
	})

	return trends, nil
}

func (m *eventManager) GetEventGroupData(ctx context.Context, clusterID int, namespace, groupBy string, startTime, endTime time.Time, limit int) ([]*model.EventGroupData, error) {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取事件列表
	listOptions := metav1.ListOptions{}
	events, err := clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取事件列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	// 创建分组映射
	groups := make(map[string][]corev1.Event)

	for _, event := range events.Items {
		// 时间过滤
		if !startTime.IsZero() && event.CreationTimestamp.Time.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && event.CreationTimestamp.Time.After(endTime) {
			continue
		}

		// 根据分组方式确定分组键
		var groupKey string
		switch groupBy {
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

	var groupData []*model.EventGroupData
	for group, eventList := range groups {

		var k8sEvents []model.K8sEvent
		for i, event := range eventList {
			if limit > 0 && i >= limit {
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

	return groupData, nil
}

// CleanupOldEvents 清理旧事件
func (m *eventManager) CleanupOldEvents(ctx context.Context, clusterID int, namespace string, beforeTime time.Time) error {
	clientset, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取所有事件
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取事件列表失败", zap.Error(err))
		return fmt.Errorf("获取事件列表失败: %w", err)
	}

	cleanedCount := 0
	errorCount := 0
	var errorMessages []string

	// 删除旧事件
	for _, event := range events.Items {
		if event.LastTimestamp.Time.Before(beforeTime) {
			err := clientset.CoreV1().Events(namespace).Delete(ctx, event.Name, metav1.DeleteOptions{})
			if err != nil {
				// 如果事件已经不存在（NotFound错误），则认为删除成功
				if apierrors.IsNotFound(err) {
					m.logger.Debug("事件已经不存在，跳过删除",
						zap.String("event", event.Name))
					cleanedCount++
				} else {
					// 其他类型的错误才记录为失败
					m.logger.Warn("删除事件失败",
						zap.String("event", event.Name),
						zap.Error(err))
					errorCount++
					errorMessages = append(errorMessages, fmt.Sprintf("删除事件 %s 失败: %v", event.Name, err))
				}
			} else {
				cleanedCount++
			}
		}
	}

	m.logger.Info("事件清理完成",
		zap.String("namespace", namespace),
		zap.Int("total", len(events.Items)),
		zap.Int("cleaned", cleanedCount),
		zap.Int("failed", errorCount))

	if errorCount > 0 {
		return fmt.Errorf("清理过程中出现 %d 个错误: %v", errorCount, errorMessages)
	}
	return nil
}

func (m *eventManager) ConvertEventToK8sEvent(event *corev1.Event, clusterID int) *model.K8sEvent {

	var eventType model.EventType
	if event.Type == "Warning" {
		eventType = model.EventTypeWarning
	} else {
		eventType = model.EventTypeNormal
	}

	eventReason := model.EventReasonOther
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
		ReportingComponent: event.Source.Component,
		ReportingInstance:  event.Source.Host,
		Labels:             event.Labels,
		Annotations:        event.Annotations,
	}
}
