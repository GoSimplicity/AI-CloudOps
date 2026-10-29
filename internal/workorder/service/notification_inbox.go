package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type inboxStore interface {
	DeliverInboxMessage(context.Context, uint64, string, int, int, int, string, string, string) error
	ListInboxMessages(context.Context, int, int, int, bool) (*model.ListResp[*model.WorkorderInboxMessage], error)
	CountUnreadInboxMessages(context.Context, int) (int64, error)
	MarkInboxRead(context.Context, int, uint64) error
	MarkAllInboxRead(context.Context, int) error
	DeleteInboxMessage(context.Context, int, uint64) error
}

func (s *workorderNotificationService) inboxStore() (inboxStore, error) {
	store, ok := s.instanceDAO.(inboxStore)
	if !ok {
		return nil, errors.New("站内通知存储不可用")
	}
	return store, nil
}

func (s *workorderNotificationService) deliverInboxPlan(ctx context.Context, event *model.WorkorderNotificationEvent, channel *eventPlanChannel) error {
	ledger, ok := ctx.Value(eventDeliveryKey{}).(*eventDeliveryLedger)
	if !ok || ledger == nil || event == nil || ledger.eventID != event.ID || channel == nil {
		return errors.New("站内通知缺少有效事件租约")
	}
	store, err := s.inboxStore()
	if err != nil {
		return err
	}
	if len(channel.Recipients) == 0 || len(channel.Recipients) > 1000 {
		return errors.New("站内通知接收人数量无效")
	}
	for _, recipient := range channel.Recipients {
		if err := ctx.Err(); err != nil {
			return err
		}
		userID, err := strconv.Atoi(recipient.Address)
		if err != nil || userID <= 0 || strconv.Itoa(userID) != recipient.Address || recipient.ID != recipient.Address || recipient.Channel != "in_app" {
			return errors.New("站内通知接收人无效")
		}
		if err := store.DeliverInboxMessage(ctx, event.ID, ledger.token, channel.Config.ID, userID, event.InstanceID, event.EventType, channel.Subject, channel.Content); err != nil {
			return err
		}
	}
	return nil
}

func (s *workorderNotificationService) ListInbox(ctx context.Context, userID int, req *model.ListWorkorderInboxReq) (*model.ListResp[*model.WorkorderInboxMessage], error) {
	if userID <= 0 || req == nil {
		return nil, errors.New("用户身份或查询无效")
	}
	store, err := s.inboxStore()
	if err != nil {
		return nil, err
	}
	return store.ListInboxMessages(ctx, userID, req.Page, req.Size, req.Unread)
}
func (s *workorderNotificationService) CountUnreadInbox(ctx context.Context, userID int) (int64, error) {
	if userID <= 0 {
		return 0, errors.New("用户身份无效")
	}
	store, err := s.inboxStore()
	if err != nil {
		return 0, err
	}
	return store.CountUnreadInboxMessages(ctx, userID)
}
func (s *workorderNotificationService) ReadInbox(ctx context.Context, userID int, id uint64) error {
	if userID <= 0 || id == 0 {
		return errors.New("用户身份或消息无效")
	}
	store, err := s.inboxStore()
	if err != nil {
		return err
	}
	return store.MarkInboxRead(ctx, userID, id)
}
func (s *workorderNotificationService) ReadAllInbox(ctx context.Context, userID int) error {
	if userID <= 0 {
		return errors.New("用户身份无效")
	}
	store, err := s.inboxStore()
	if err != nil {
		return err
	}
	return store.MarkAllInboxRead(ctx, userID)
}
func (s *workorderNotificationService) DeleteInbox(ctx context.Context, userID int, id uint64) error {
	if userID <= 0 || id == 0 {
		return errors.New("用户身份或消息无效")
	}
	store, err := s.inboxStore()
	if err != nil {
		return err
	}
	return store.DeleteInboxMessage(ctx, userID, id)
}
