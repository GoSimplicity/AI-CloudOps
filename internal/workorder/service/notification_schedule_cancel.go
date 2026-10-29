package service

import (
	"context"
	"errors"
	"strings"
)

// CancelScheduledNotification 不发送通知，权限由独立 API 路由控制。
func (s *workorderNotificationService) CancelScheduledNotification(ctx context.Context, eventID uint64, reason string, operatorID int) error {
	reason = strings.TrimSpace(reason)
	if eventID == 0 || operatorID <= 0 || len(reason) < 5 || len(reason) > 500 {
		return errors.New("无效的排期取消请求")
	}
	store, ok := s.instanceDAO.(interface {
		CancelScheduledNotification(context.Context, uint64, string, int) error
	})
	if !ok {
		return errors.New("排期取消存储不可用")
	}
	return store.CancelScheduledNotification(ctx, eventID, reason, operatorID)
}
