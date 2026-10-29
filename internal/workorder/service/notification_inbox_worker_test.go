package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

func setupInboxWorker(t *testing.T) (*NotificationEventWorker, *model.WorkorderNotificationEvent) {
	t.Helper()
	db := eventTestDB(t)
	if err := db.AutoMigrate(&model.User{}, &model.WorkorderInboxMessage{}, &model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{7, 8} {
		user := &model.User{Model: model.Model{ID: id}, Username: fmt.Sprintf("user-%d", id), Enable: 1}
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	instance := model.WorkorderInstance{Model: model.Model{ID: 12}, ProcessID: 1, Status: model.InstanceStatusPending, OperatorID: 7, OperatorName: "fixture"}
	snapshot, err := json.Marshal(instance)
	if err != nil {
		t.Fatal(err)
	}
	event := &model.WorkorderNotificationEvent{InstanceID: 12, EventType: model.EventTypeInstanceCreated, Snapshot: snapshot, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
	if err := db.Create(event).Error; err != nil {
		t.Fatal(err)
	}
	delay := 5
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeDelayed, TriggerVersion: 1, DelayMinutes: &delay, Status: 1, Channels: model.StringList{"in_app"}, RecipientTypes: model.StringList{"user"}, RecipientUsers: model.StringList{"7", "8"}, MessageTemplate: "fixture message"}
	store := workdao.NewWorkorderInstanceDAO(db, zap.NewNop())
	svc := &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, instanceDAO: store, userDAO: userdao.NewUserDAO(db, zap.NewNop()), logger: zap.NewNop()}
	return &NotificationEventWorker{db: db, service: svc, logger: zap.NewNop()}, event
}

func TestInboxDelayedDeliveryAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			worker, parent := setupInboxWorker(t)
			if err := worker.processOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			var child model.WorkorderNotificationEvent
			if err := worker.db.Where("parent_event_id = ?", parent.ID).First(&child).Error; err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := worker.db.Model(&model.WorkorderInboxMessage{}).Count(&count).Error; err != nil || count != 0 {
				t.Fatal("delayed message delivered early")
			}
			if cancelled {
				store := worker.service.instanceDAO.(interface {
					CancelScheduledNotification(context.Context, uint64, string, int) error
				})
				if err := store.CancelScheduledNotification(context.Background(), child.ID, "verified cancellation", 7); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := worker.db.Model(&child).Update("next_attempt_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
					t.Fatal(err)
				}
				if err := worker.processOne(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if err := worker.db.Model(&model.WorkorderInboxMessage{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			want := int64(2)
			if cancelled {
				want = 0
			}
			if count != want {
				t.Fatalf("messages=%d want=%d", count, want)
			}
		})
	}
}

func TestInboxPartialSuccessManualRetryIsFencedAndDoesNotResurrect(t *testing.T) {
	worker, event := setupInboxWorker(t)
	store := worker.service.instanceDAO
	inbox := store.(inboxStore)
	lease := time.Now().UTC().Add(time.Minute)
	plan := []eventPlanChannel{{Config: model.WorkorderNotification{Model: model.Model{ID: 3}}, Channel: "in_app", Recipients: []RecipientInfo{{ID: "7", Address: "7", Channel: "in_app"}, {ID: "8", Address: "8", Channel: "in_app"}}, Subject: "subject", Content: "body"}}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.db.Model(event).Updates(map[string]interface{}{"status": "processing", "lease_token": "fixture", "lease_until": lease, "plan": encoded}).Error; err != nil {
		t.Fatal(err)
	}
	if err := inbox.DeliverInboxMessage(context.Background(), event.ID, "fixture", 3, 7, 12, event.EventType, "subject", "body"); err != nil {
		t.Fatal(err)
	}
	rows, err := inbox.ListInboxMessages(context.Background(), 7, 1, 20, false)
	if err != nil || rows.Total != 1 {
		t.Fatal("first message absent")
	}
	if err := inbox.DeleteInboxMessage(context.Background(), 7, rows.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := worker.db.Model(event).Updates(map[string]interface{}{"status": "failed", "lease_token": "", "lease_until": nil, "last_error": "通知处理失败，需人工检查发送日志"}).Error; err != nil {
		t.Fatal(err)
	}
	retry := store.(interface {
		RetryNotificationEvent(context.Context, uint64, string, int) error
	})
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := retry.RetryNotificationEvent(cancelled, event.ID, "verified retry", 7); err == nil {
		t.Fatal("cancelled retry accepted")
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- retry.RetryNotificationEvent(context.Background(), event.ID, "verified retry", 7)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("retry winners=%d", wins)
	}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	rows, err = inbox.ListInboxMessages(context.Background(), 7, 1, 20, false)
	if err != nil || rows.Total != 0 {
		t.Fatal("deleted message resurrected")
	}
	rows, err = inbox.ListInboxMessages(context.Background(), 8, 1, 20, false)
	if err != nil || rows.Total != 1 {
		t.Fatal("remaining user did not receive message")
	}
	var audits []model.WorkorderNotificationResolution
	if err := worker.db.Find(&audits).Error; err != nil || len(audits) != 1 || audits[0].Action != "retry_inbox" {
		t.Fatal("retry audit missing")
	}
}

func TestInboxRetryRejectsUnresolvedExternalDelivery(t *testing.T) {
	for _, status := range []string{"sending", "uncertain", "failed"} {
		t.Run(status, func(t *testing.T) {
			worker, event := setupInboxWorker(t)
			plan, err := json.Marshal([]eventPlanChannel{{Config: model.WorkorderNotification{Model: model.Model{ID: 3}}, Channel: "in_app", Recipients: []RecipientInfo{{ID: "7", Address: "7", Channel: "in_app"}}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := worker.db.Model(event).Updates(map[string]interface{}{"status": "failed", "plan": plan}).Error; err != nil {
				t.Fatal(err)
			}
			if err := worker.db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 3, Channel: "email", RecipientHash: "opaque", Status: status}).Error; err != nil {
				t.Fatal(err)
			}
			retry := worker.service.instanceDAO.(interface {
				RetryNotificationEvent(context.Context, uint64, string, int) error
			})
			if err := retry.RetryNotificationEvent(context.Background(), event.ID, "verified retry", 7); err == nil {
				t.Fatal("unresolved external delivery accepted")
			}
			var auditCount int64
			if err := worker.db.Model(&model.WorkorderNotificationResolution{}).Count(&auditCount).Error; err != nil || auditCount != 0 {
				t.Fatal("rejected retry wrote audit")
			}
		})
	}
}

func TestInboxWorkerRecoversAfterDeliveryCommitAndExpiredLease(t *testing.T) {
	worker, event := setupInboxWorker(t)
	store := worker.service.instanceDAO.(inboxStore)
	plan, err := json.Marshal([]eventPlanChannel{{Config: model.WorkorderNotification{Model: model.Model{ID: 3}}, Channel: "in_app", Recipients: []RecipientInfo{{ID: "7", Address: "7", Channel: "in_app"}, {ID: "8", Address: "8", Channel: "in_app"}}, Subject: "subject", Content: "body"}})
	if err != nil {
		t.Fatal(err)
	}
	lease := time.Now().UTC().Add(time.Minute)
	if err := worker.db.Model(event).Updates(map[string]interface{}{"status": "processing", "lease_token": "old-token", "lease_until": lease, "plan": plan}).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.DeliverInboxMessage(context.Background(), event.ID, "old-token", 3, 7, 12, event.EventType, "subject", "body"); err != nil {
		t.Fatal(err)
	}
	if err := worker.db.Model(event).Update("lease_until", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := worker.db.Model(&model.WorkorderInboxMessage{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("recovery messages=%d err=%v", count, err)
	}
	var saved model.WorkorderNotificationEvent
	if err := worker.db.First(&saved, event.ID).Error; err != nil || saved.Status != "success" {
		t.Fatal("recovered event not completed")
	}
}

func TestInboxDestinationValidationRejectsCustomOnly(t *testing.T) {
	if err := validateNotificationDestinations(model.StringList{"in_app"}, model.StringList{"custom"}, nil, nil, nil, model.NotificationAddresses{{Channel: "email", Address: "user@example.test"}}); err == nil {
		t.Fatal("external custom destination accepted for inbox")
	}
	if err := validateNotificationDestinations(model.StringList{"in_app"}, model.StringList{"user"}, model.StringList{"7"}, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestInboxResolvesAndDeduplicatesInternalRecipients(t *testing.T) {
	service := &workorderNotificationService{userDAO: mixedRecipientDAO{}, logger: zap.NewNop()}
	config := &model.WorkorderNotification{RecipientTypes: model.StringList{"creator", "role", "dept", "custom"}, RecipientRoles: model.StringList{"1"}, RecipientDepts: model.StringList{"2"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "external@example.test"}}}
	recipients, err := service.getRecipients(context.Background(), config, &model.WorkorderInstance{OperatorID: 7})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := service.resolveChannelRecipients(context.Background(), "in_app", recipients)
	if err != nil || len(resolved) != 2 {
		t.Fatalf("unexpected internal recipients: %d %v", len(resolved), err)
	}
	if resolved[0].ID != "7" || resolved[0].Address != "7" || resolved[1].ID != "8" || resolved[1].Address != "8" {
		t.Fatal("user IDs were not canonicalized")
	}
}
