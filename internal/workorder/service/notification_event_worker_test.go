package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type eventTestConfigDAO struct {
	workdao.WorkorderNotificationDAO
	config *model.WorkorderNotification
}

func (d *eventTestConfigDAO) GetActiveNotificationsByEventType(context.Context, string, int) ([]*model.WorkorderNotification, error) {
	return []*model.WorkorderNotification{d.config}, nil
}
func (*eventTestConfigDAO) IncrementSentCount(context.Context, int) error { return nil }
func (*eventTestConfigDAO) AddSendLog(context.Context, *model.WorkorderNotificationLog) error {
	return nil
}

type eventTestSender struct {
	mu        sync.Mutex
	attempts  map[string]int
	failFirst string
}

func (s *eventTestSender) GetAvailableChannels() []string { return []string{"email"} }
func (s *eventTestSender) SendNotification(ctx context.Context, r *notification.SendRequest) (*notification.SendResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[r.RecipientAddr]++
	if r.RecipientAddr == s.failFirst && s.attempts[r.RecipientAddr] == 1 {
		return nil, notification.ErrDeliveryRejected
	}
	return &notification.SendResponse{Success: true}, nil
}
func eventTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "outbox.db")+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err = db.AutoMigrate(&model.WorkorderInstance{}, &model.WorkorderProcess{}, &model.WorkorderFormDesign{}, &model.WorkorderCategory{}, &model.WorkorderInstanceComment{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}, &model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestDeleteEventSnapshotAndAtomicRollback(t *testing.T) {
	db := eventTestDB(t)
	instance := model.WorkorderInstance{Title: "owned", SerialNumber: "owned-1", ProcessID: 1, Status: model.InstanceStatusDraft, OperatorID: 9, OperatorName: "owner"}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	s := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), notificationService: &workorderNotificationService{}, logger: zap.NewNop()}
	requestCtx, cancel := context.WithCancel(context.Background())
	if err := s.DeleteInstance(requestCtx, instance.ID); err != nil {
		t.Fatal(err)
	}
	cancel()
	var event model.WorkorderNotificationEvent
	if err := db.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var snapshot model.WorkorderInstance
	if err := json.Unmarshal(event.Snapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if event.EventType != model.EventTypeInstanceDeleted || snapshot.ID != instance.ID || snapshot.Title != "owned" {
		t.Fatal("deleted instance snapshot missing")
	}
	sender := &eventTestSender{attempts: map[string]int{}}
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "first@example.test"}}, MessageTemplate: "owned"}
	worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}, logger: zap.NewNop()}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.attempts["first@example.test"] != 1 {
		t.Fatal("cancelled request prevented snapshot delivery")
	}
	var active int64
	db.Model(&model.WorkorderInstance{}).Where("id = ?", instance.ID).Count(&active)
	if active != 0 {
		t.Fatal("instance remains active")
	}
	second := model.WorkorderInstance{Title: "second", SerialNumber: "owned-2", ProcessID: 1, Status: model.InstanceStatusDraft}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.WorkorderNotificationEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteInstance(context.Background(), second.ID); err == nil {
		t.Fatal("outbox insertion failure must abort delete")
	}
	db.Model(&model.WorkorderInstance{}).Where("id = ?", second.ID).Count(&active)
	if active != 1 {
		t.Fatal("delete was not rolled back")
	}
}
func TestEventRetrySkipsSuccessfulRecipientAndRecoversExpiredLease(t *testing.T) {
	db := eventTestDB(t)
	snapshot := model.WorkorderInstance{Model: model.Model{ID: 7}, Title: "owned", ProcessID: 1, OperatorID: 1}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	event := model.WorkorderNotificationEvent{InstanceID: 7, EventType: model.EventTypeInstanceCreated, Snapshot: data, Status: "processing", LeaseUntil: ptrTime(time.Now().UTC().Add(-time.Minute)), NextAttemptAt: time.Now().UTC().Add(-time.Minute)}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	sender := &eventTestSender{attempts: map[string]int{}, failFirst: "second@example.test"}
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, MaxRetries: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "first@example.test"}, {Channel: "email", Address: "second@example.test"}}, MessageTemplate: "owned"}
	svc := &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}
	worker := &NotificationEventWorker{db: db, service: svc, logger: zap.NewNop()}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "pending" || event.Attempts != 1 {
		t.Fatalf("first state %s attempts %d", event.Status, event.Attempts)
	}
	if err := db.Model(&event).Update("next_attempt_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	// 配置在首次发送后改变；重试必须继续使用持久计划中的原地址与内容
	config.RecipientAddresses = model.NotificationAddresses{{Channel: "email", Address: "new@example.test"}}
	config.MessageTemplate = "changed after first delivery"
	restarted := &NotificationEventWorker{db: db, service: svc, logger: zap.NewNop()}
	if err := restarted.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "success" || event.Attempts != 2 {
		var rows []model.WorkorderNotificationDelivery
		db.Where("event_id = ?", event.ID).Find(&rows)
		t.Fatalf("retry state %s attempts %d error %s rows %+v", event.Status, event.Attempts, event.LastError, rows)
	}
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if sender.attempts["new@example.test"] != 0 {
		t.Fatal("retry used changed recipient configuration")
	}
	if sender.attempts["first@example.test"] != 1 || sender.attempts["second@example.test"] != 2 {
		t.Fatalf("unexpected recipient attempts: %#v", sender.attempts)
	}
	var count int64
	db.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ? AND status = ?", event.ID, "success").Count(&count)
	if count != 2 {
		t.Fatalf("delivery ledger records=%d", count)
	}
}
func ptrTime(v time.Time) *time.Time { return &v }

func TestUnconfirmedDeliveryAfterCrashIsNotAutomaticallyResent(t *testing.T) {
	db := eventTestDB(t)
	snapshot := model.WorkorderInstance{Model: model.Model{ID: 8}, Title: "owned", ProcessID: 1, OperatorID: 1}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	event := model.WorkorderNotificationEvent{InstanceID: 8, EventType: model.EventTypeInstanceCreated, Snapshot: data, Status: "processing", LeaseUntil: ptrTime(time.Now().UTC().Add(-time.Minute)), NextAttemptAt: time.Now().UTC().Add(-time.Minute)}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("first@example.test")))
	if err := db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 3, Channel: "email", RecipientHash: hash, Status: "sending"}).Error; err != nil {
		t.Fatal(err)
	}
	sender := &eventTestSender{attempts: map[string]int{}}
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "first@example.test"}}, MessageTemplate: "owned"}
	worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}, logger: zap.NewNop()}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "uncertain" {
		t.Fatalf("unknown delivery should require review, got %s", event.Status)
	}
	if len(sender.attempts) != 0 {
		t.Fatal("unconfirmed send was automatically repeated")
	}
}

func TestEventStatusQueryDoesNotExposeSnapshotOrMessage(t *testing.T) {
	db := eventTestDB(t)
	event := model.WorkorderNotificationEvent{InstanceID: 10, EventType: "created", Status: "uncertain", Snapshot: []byte(`{"secret":"private-fixture"}`), Content: "private-fixture", NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	store := workdao.NewWorkorderInstanceDAO(db, zap.NewNop())
	svc := &workorderNotificationService{instanceDAO: store}
	result, err := svc.ListNotificationEvents(context.Background(), &model.ListWorkorderNotificationEventsReq{Status: "uncertain"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private-fixture") || len(result.Items) != 1 || result.Items[0].Status != "uncertain" {
		t.Fatal("event list exposed private snapshot or lost status")
	}
}

func TestManualResolutionIsAuditedAndRequiresTerminalEvent(t *testing.T) {
	db := eventTestDB(t)
	if err := db.AutoMigrate(&model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: []byte(`{}`), Status: "uncertain", NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	delivery := model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 1, Channel: "email", RecipientHash: "fixture", Status: "uncertain", Attempts: 1}
	if err := db.Create(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	dao := workdao.NewWorkorderInstanceDAO(db, zap.NewNop())
	svc := &workorderNotificationService{instanceDAO: dao}
	req := &model.ResolveWorkorderNotificationDeliveryReq{DeliveryID: delivery.ID, Action: "retry", Reason: "operator verified"}
	if err := svc.ResolveNotificationDelivery(context.Background(), event.ID, req, 7); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveNotificationDelivery(context.Background(), event.ID, req, 7); err == nil {
		t.Fatal("repeat resolution must fail after state change")
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "pending" {
		t.Fatal("approved retry did not return to queue")
	}
	if err := db.First(&delivery, delivery.ID).Error; err != nil {
		t.Fatal(err)
	}
	if delivery.Status != "failed" || delivery.Attempts != 0 {
		t.Fatal("retry budget not reset for explicit operator action")
	}
	audits, err := svc.ListNotificationEventResolutions(context.Background(), event.ID)
	if err != nil || len(audits) != 1 || audits[0].OperatorID != 7 || audits[0].Action != "retry" {
		t.Fatal("missing audit")
	}
}
func TestExpiredWorkerCannotOverwriteDeliveryStatus(t *testing.T) {
	db := eventTestDB(t)
	until := time.Now().UTC().Add(time.Minute)
	event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: []byte(`{}`), Status: "processing", LeaseToken: "new-owner", LeaseUntil: &until, NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	stale := &eventDeliveryLedger{db: db, eventID: event.ID, token: "old-owner"}
	if err := stale.markDelivery(1, "email", "fixture", "sending", 0, 0); err == nil {
		t.Fatal("old worker acquired new lease")
	}
	var count int64
	db.Model(&model.WorkorderNotificationDelivery{}).Count(&count)
	if count != 0 {
		t.Fatal("expired worker created delivery")
	}
}

type alwaysRejectedSender struct {
	calls     int
	sawSingle bool
}

func (s *alwaysRejectedSender) GetAvailableChannels() []string { return []string{"email"} }
func (s *alwaysRejectedSender) SendNotification(_ context.Context, r *notification.SendRequest) (*notification.SendResponse, error) {
	s.calls++
	s.sawSingle = r.MaxRetries != nil && *r.MaxRetries == 0
	return nil, notification.ErrDeliveryRejected
}
func TestConfirmedRejectionHonorsExtraRetryBudget(t *testing.T) {
	for _, max := range []int{0, 2} {
		t.Run(fmt.Sprintf("retries-%d", max), func(t *testing.T) {
			db := eventTestDB(t)
			snapshot := model.WorkorderInstance{Model: model.Model{ID: 1}, ProcessID: 1}
			payload, _ := json.Marshal(snapshot)
			event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: payload, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
			if err := db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, MaxRetries: max, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "first@example.test"}}, MessageTemplate: "owned"}
			sender := &alwaysRejectedSender{}
			worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}, logger: zap.NewNop()}
			for i := 0; i <= max; i++ {
				if err := worker.processOne(context.Background()); err != nil {
					t.Fatal(err)
				}
				if err := db.First(&event, event.ID).Error; err != nil {
					t.Fatal(err)
				}
			}
			if event.Status != "failed" || sender.calls != max+1 || !sender.sawSingle {
				t.Fatalf("status %s sends %d single=%v", event.Status, sender.calls, sender.sawSingle)
			}
			if err := worker.processOne(context.Background()); !errors.Is(err, errNoNotificationEvent) {
				t.Fatal("exhausted delivery remained eligible")
			}
		})
	}
}

func TestConfirmedRejectionRespectsRetryInterval(t *testing.T) {
	db := eventTestDB(t)
	snapshot := model.WorkorderInstance{Model: model.Model{ID: 1}, ProcessID: 1}
	payload, _ := json.Marshal(snapshot)
	event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: payload, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, MaxRetries: 2, RetryInterval: 7, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "first@example.test"}}, MessageTemplate: "owned"}
	sender := &alwaysRejectedSender{}
	worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}, logger: zap.NewNop()}
	before := time.Now().UTC()
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "pending" || event.NextAttemptAt.Before(before.Add(6*time.Minute)) || event.NextAttemptAt.After(before.Add(8*time.Minute)) {
		t.Fatal("configured minute interval not preserved")
	}
	if err := worker.processOne(context.Background()); !errors.Is(err, errNoNotificationEvent) {
		t.Fatal("delivery retried before configured interval")
	}
	if sender.calls != 1 {
		t.Fatal("unexpected immediate retry")
	}
}

type noTransactionInstanceDAO struct {
	workdao.WorkorderInstanceDAO
	lookups int
}

func (d *noTransactionInstanceDAO) GetInstanceByTitle(context.Context, string) (*model.WorkorderInstance, error) {
	d.lookups++
	return nil, workdao.ErrInstanceNotFound
}
func TestMutationWithNotificationsRequiresTransactionBeforeAnyWork(t *testing.T) {
	db := &noTransactionInstanceDAO{}
	svc := &instanceService{dao: db, notificationService: &workorderNotificationService{}, logger: zap.NewNop()}
	err := svc.CreateInstance(context.Background(), &model.CreateWorkorderInstanceReq{Status: model.InstanceStatusDraft, Priority: model.PriorityNormal})
	if err == nil || db.lookups != 0 {
		t.Fatal("missing transaction capability was not rejected before lookup or mutation")
	}
}

func TestCreateUpdateDeleteEventsCommitWithMutations(t *testing.T) {
	db := eventTestDB(t)
	form := model.WorkorderFormDesign{Name: "owned-form", Schema: model.JSONMap{"fields": []any{}}, Status: model.ProcessStatusPublished, OperatorID: 1, OperatorName: "tester"}
	if err := db.Create(&form).Error; err != nil {
		t.Fatal(err)
	}
	process := model.WorkorderProcess{Name: "owned-process", FormDesignID: form.ID, Definition: model.JSONMap{"steps": []any{}, "connections": []any{}}, Status: model.ProcessStatusPublished, OperatorID: 1, OperatorName: "tester"}
	if err := db.Create(&process).Error; err != nil {
		t.Fatal(err)
	}
	svc := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), processDao: workdao.NewProcessDAO(db, zap.NewNop()), formDesignDao: workdao.NewWorkorderFormDesignDAO(db, zap.NewNop()), flowDao: workdao.NewInstanceFlowDAO(db, zap.NewNop()), timelineDao: workdao.NewInstanceTimeLineDAO(db, zap.NewNop()), notificationService: &workorderNotificationService{}, logger: zap.NewNop()}
	req := &model.CreateWorkorderInstanceReq{Title: "owned-create", ProcessID: process.ID, FormData: model.JSONMap{}, Status: model.InstanceStatusDraft, Priority: model.PriorityNormal, OperatorID: 1, OperatorName: "tester"}
	if err := svc.CreateInstance(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	var instance model.WorkorderInstance
	if err := db.Where("title = ?", req.Title).First(&instance).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateInstance(context.Background(), &model.UpdateWorkorderInstanceReq{ID: instance.ID, Title: "owned-updated"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("reject-event", func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_workorder_notification_event" {
			tx.AddError(errors.New("isolated event failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateInstance(context.Background(), &model.UpdateWorkorderInstanceReq{ID: instance.ID, Title: "should-rollback"}); err == nil {
		t.Fatal("event insertion failure accepted")
	}
	var after model.WorkorderInstance
	if err := db.First(&after, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Title != "owned-updated" {
		t.Fatal("update persisted despite event failure")
	}
	if err := db.Callback().Create().Remove("reject-event"); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteInstance(context.Background(), instance.ID); err != nil {
		t.Fatal(err)
	}
	var events []model.WorkorderNotificationEvent
	if err := db.Where("instance_id = ?", instance.ID).Order("id").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].EventType != model.EventTypeInstanceCreated || events[1].EventType != model.EventTypeInstanceUpdated || events[2].EventType != model.EventTypeInstanceDeleted {
		t.Fatalf("event sequence %+v", events)
	}
	var deleted model.WorkorderInstance
	if err := json.Unmarshal(events[2].Snapshot, &deleted); err != nil {
		t.Fatal(err)
	}
	if deleted.Title != "owned-updated" {
		t.Fatal("delete event lost pre-delete state")
	}
}

func TestManualRetryOnlyForFailedEventWithoutDeliveries(t *testing.T) {
	db := eventTestDB(t)
	if err := db.AutoMigrate(&model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: []byte(`{}`), Status: "failed", Attempts: 10, NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	svc := &workorderNotificationService{instanceDAO: workdao.NewWorkorderInstanceDAO(db, zap.NewNop())}
	if err := svc.RetryNotificationEvent(context.Background(), event.ID, "verified configuration", 7); err != nil {
		t.Fatal(err)
	}
	if err := svc.RetryNotificationEvent(context.Background(), event.ID, "verified configuration", 7); err == nil {
		t.Fatal("second retry accepted")
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "pending" || event.Attempts != 0 {
		t.Fatal("manual retry did not reset failure")
	}
	audits, err := svc.ListNotificationEventResolutions(context.Background(), event.ID)
	if err != nil || len(audits) != 1 || audits[0].Action != "retry_event" || audits[0].OperatorID != 7 {
		t.Fatal("missing retry audit")
	}
	another := model.WorkorderNotificationEvent{InstanceID: 2, EventType: "created", Snapshot: []byte(`{}`), Status: "failed", NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&another).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.WorkorderNotificationDelivery{EventID: another.ID, NotificationID: 1, Channel: "email", RecipientHash: "fixture", Status: "success"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.RetryNotificationEvent(context.Background(), another.ID, "verified configuration", 7); err == nil {
		t.Fatal("event with delivery records must use per-recipient resolution")
	}
}

type noEventConfigDAO struct {
	workdao.WorkorderNotificationDAO
}

func (*noEventConfigDAO) GetActiveNotificationsByEventType(context.Context, string, int) ([]*model.WorkorderNotification, error) {
	return nil, nil
}
func TestRemovedConfigCannotHideFailedOrUnknownDelivery(t *testing.T) {
	for _, oldStatus := range []string{"failed", "uncertain", "sending"} {
		t.Run(oldStatus, func(t *testing.T) {
			db := eventTestDB(t)
			snapshot := model.WorkorderInstance{Model: model.Model{ID: 1}, ProcessID: 1}
			payload, _ := json.Marshal(snapshot)
			event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: payload, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
			if err := db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			delivery := model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 3, Channel: "email", RecipientHash: "fixture", Status: oldStatus, Attempts: 1}
			if err := db.Create(&delivery).Error; err != nil {
				t.Fatal(err)
			}
			worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &noEventConfigDAO{}, logger: zap.NewNop()}, logger: zap.NewNop()}
			if err := worker.processOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := db.First(&event, event.ID).Error; err != nil {
				t.Fatal(err)
			}
			expected := "uncertain"
			if oldStatus == "failed" {
				expected = "failed"
			}
			if event.Status != expected {
				t.Fatalf("old %s became %s", oldStatus, event.Status)
			}
		})
	}
}

func TestEventRetentionRemovesOnlyUnauditedCompletedRows(t *testing.T) {
	db := eventTestDB(t)
	if err := db.AutoMigrate(&model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().AddDate(0, 0, -100)
	statuses := []string{"success", "skipped", "pending", "processing", "failed", "uncertain", "success"}
	var events []model.WorkorderNotificationEvent
	for i, status := range statuses {
		event := model.WorkorderNotificationEvent{InstanceID: i + 1, EventType: "created", Snapshot: []byte(`{}`), Status: status, CreatedAt: old, UpdatedAt: old, NextAttemptAt: old}
		if err := db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if err := db.Create(&model.WorkorderNotificationResolution{EventID: events[6].ID, Action: "delivered", OperatorID: 1, Reason: "verified operator"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if err := db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 1, Channel: "email", RecipientHash: "fixture", Status: "success"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	worker := &NotificationEventWorker{db: db, retentionDays: 90}
	count, err := worker.cleanupCompleted(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("deleted %d instead of 2", count)
	}
	for i, event := range events {
		var n int64
		db.Model(&model.WorkorderNotificationEvent{}).Where("id = ?", event.ID).Count(&n)
		if (i < 2 && n != 0) || (i >= 2 && n != 1) {
			t.Fatalf("incorrect event retention for %s", event.Status)
		}
		db.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ?", event.ID).Count(&n)
		if (i < 2 && n != 0) || (i >= 2 && n != 1) {
			t.Fatal("incorrect delivery retention")
		}
	}
	var audits int64
	db.Model(&model.WorkorderNotificationResolution{}).Count(&audits)
	if audits != 1 {
		t.Fatal("manual audit was deleted")
	}
}
func TestEventRetentionConfig(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  int
		valid bool
	}{{"", 90, true}, {"1", 1, true}, {"3650", 3650, true}, {"0", 0, false}, {"3651", 0, false}, {"no", 0, false}} {
		got, err := notificationRetentionDays(tt.value)
		if (err == nil) != tt.valid || got != tt.want {
			t.Fatalf("retention %q got %d err=%v", tt.value, got, err)
		}
	}
}

type uncertainOnceSender struct {
	calls            int
	address, content string
}

func (s *uncertainOnceSender) GetAvailableChannels() []string { return []string{"email"} }
func (s *uncertainOnceSender) SendNotification(_ context.Context, r *notification.SendRequest) (*notification.SendResponse, error) {
	s.calls++
	s.address = r.RecipientAddr
	s.content = r.Content
	if s.calls == 1 {
		return nil, io.EOF
	}
	return &notification.SendResponse{Success: true}, nil
}
func TestUnknownManualRetryUsesFrozenPlanAndClearsAfterSuccess(t *testing.T) {
	db := eventTestDB(t)
	if err := db.AutoMigrate(&model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	snapshot := model.WorkorderInstance{Model: model.Model{ID: 1}, Title: "frozen-title", ProcessID: 1}
	payload, _ := json.Marshal(snapshot)
	event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "created", Snapshot: payload, Content: "original event", Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "original@example.test"}}, MessageTemplate: "original template"}
	sender := &uncertainOnceSender{}
	svc := &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, instanceDAO: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), notificationMgr: sender, logger: zap.NewNop()}
	worker := &NotificationEventWorker{db: db, service: svc, logger: zap.NewNop()}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "uncertain" || len(event.Plan) == 0 || event.Content != "original event" || !strings.Contains(string(event.Snapshot), "frozen-title") {
		t.Fatal("unknown outcome lost recovery snapshot")
	}
	var delivery model.WorkorderNotificationDelivery
	if err := db.Where("event_id = ?", event.ID).First(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	config.MessageTemplate = "changed template"
	config.RecipientAddresses = model.NotificationAddresses{{Channel: "email", Address: "changed@example.test"}}
	if err := svc.ResolveNotificationDelivery(context.Background(), event.ID, &model.ResolveWorkorderNotificationDeliveryReq{DeliveryID: delivery.ID, Action: "retry", Reason: "verified not delivered"}, 7); err != nil {
		t.Fatal(err)
	}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "success" || sender.calls != 2 || sender.address != "original@example.test" || sender.content != "original template" {
		t.Fatalf("retry changed plan or outcome: %s", event.Status)
	}
	if len(event.Plan) != 0 || string(event.Snapshot) != "{}" || event.Content != "" {
		t.Fatal("completed event retained sensitive snapshot or plan")
	}
}
