package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type notificationActorKey struct{}

// WithNotificationActor 仅由认证入口注入可信操作者
func WithNotificationActor(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, notificationActorKey{}, userID)
}

type instanceEventStore interface {
	InTransaction(context.Context, func(context.Context) error) error
	SaveNotificationEvent(context.Context, *model.WorkorderNotificationEvent) error
}

// recordNotificationEvent 与工单变更处于同一事务，不依赖请求结束后的 goroutine
func (s *instanceService) recordNotificationEvent(ctx context.Context, instanceID int, eventType string, senderID int, content string, snapshot *model.WorkorderInstance) error {
	store, ok := s.dao.(instanceEventStore)
	if !ok {
		return errors.New("工单通知事件存储不可用")
	}
	if snapshot == nil {
		var err error
		snapshot, err = s.dao.GetInstanceByID(ctx, instanceID)
		if err != nil {
			return err
		}
	}
	copy := *snapshot
	copy.FormData = nil
	copy.Process = nil
	copy.Comments = nil
	copy.FlowLogs = nil
	copy.Timeline = nil
	data, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	if senderID == 0 {
		senderID, _ = ctx.Value(notificationActorKey{}).(int)
	}
	event := &model.WorkorderNotificationEvent{InstanceID: instanceID, EventType: eventType, Snapshot: data, Content: content, SenderID: senderID, Status: "pending", NextAttemptAt: time.Now().UTC()}
	return store.SaveNotificationEvent(ctx, event)
}
