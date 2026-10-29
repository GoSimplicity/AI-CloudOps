package service

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type cancelStoreStub struct {
	dao.WorkorderInstanceDAO
	called  bool
	eventID uint64
	reason  string
	actor   int
}

func (s *cancelStoreStub) CancelScheduledNotification(_ context.Context, id uint64, reason string, actor int) error {
	s.called, s.eventID, s.reason, s.actor = true, id, reason, actor
	return nil
}

func TestCancelScheduledServiceValidatesAndForwardsTrustedIdentity(t *testing.T) {
	store := &cancelStoreStub{}
	service := &workorderNotificationService{instanceDAO: store}
	for _, input := range []struct {
		id     uint64
		reason string
		actor  int
	}{{0, "verified", 7}, {1, "short", 0}, {1, "    ", 7}} {
		if err := service.CancelScheduledNotification(context.Background(), input.id, input.reason, input.actor); err == nil {
			t.Fatal("invalid request accepted")
		}
		if store.called {
			t.Fatal("invalid request reached storage")
		}
	}
	if err := service.CancelScheduledNotification(context.Background(), 12, "  verified cancellation  ", 7); err != nil {
		t.Fatal(err)
	}
	if !store.called || store.eventID != 12 || store.reason != "verified cancellation" || store.actor != 7 {
		t.Fatal("identity or request not preserved")
	}
	missing := &workorderNotificationService{instanceDAO: &stateGuardDAO{}}
	if err := missing.CancelScheduledNotification(context.Background(), 12, "verified cancellation", 7); err == nil {
		t.Fatal("unavailable store accepted")
	}
}
