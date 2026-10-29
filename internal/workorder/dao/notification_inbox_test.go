package dao

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func inboxTestDB(t *testing.T) (*gorm.DB, *workorderInstanceDAO) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "inbox.db")+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}, &model.WorkorderInboxMessage{}, &model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	return db, NewWorkorderInstanceDAO(db, zap.NewNop()).(*workorderInstanceDAO)
}

func TestInboxAtomicDeliveryAndUserIsolation(t *testing.T) {
	db, store := inboxTestDB(t)
	for _, user := range []model.User{{Model: model.Model{ID: 7}, Username: "one", Enable: 1}, {Model: model.Model{ID: 8}, Username: "two", Enable: 1}} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().UTC().Add(time.Minute)
	event := model.WorkorderNotificationEvent{InstanceID: 12, EventType: "instance_created", Status: "processing", LeaseToken: "fixture", LeaseUntil: &deadline, Snapshot: []byte("{}"), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	deliver := func(user int, token string) error {
		return store.DeliverInboxMessage(context.Background(), event.ID, token, 3, user, 12, event.EventType, "subject", "body")
	}
	if err := deliver(7, "invalid"); err == nil {
		t.Fatal("stale lease accepted")
	}
	if err := deliver(7, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := deliver(7, "fixture"); err != nil {
		t.Fatal("replay was not idempotent")
	}
	if err := deliver(8, "fixture"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.WorkorderInboxMessage{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("messages=%d error=%v", count, err)
	}
	if err := db.Model(&model.WorkorderNotificationDelivery{}).Where("status = ? AND channel = ?", "success", "in_app").Count(&count).Error; err != nil || count != 2 {
		t.Fatal("missing delivery ledger")
	}
	rows, err := store.ListInboxMessages(context.Background(), 7, 1, 20, false)
	if err != nil || rows.Total != 1 || len(rows.Items) != 1 || rows.Items[0].UserID != 7 {
		t.Fatal("user list escaped ownership")
	}
	firstID := rows.Items[0].ID
	if err := store.MarkInboxRead(context.Background(), 8, firstID); err == nil {
		t.Fatal("cross-user read accepted")
	}
	if err := store.DeleteInboxMessage(context.Background(), 8, firstID); err == nil {
		t.Fatal("cross-user delete accepted")
	}
	if err := store.MarkInboxRead(context.Background(), 7, firstID); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkInboxRead(context.Background(), 7, firstID); err != nil {
		t.Fatal("read not idempotent")
	}
	if got, err := store.CountUnreadInboxMessages(context.Background(), 7); err != nil || got != 0 {
		t.Fatal("wrong unread count")
	}
	if got, err := store.CountUnreadInboxMessages(context.Background(), 8); err != nil || got != 1 {
		t.Fatal("another user unread state changed")
	}
	if err := store.MarkAllInboxRead(context.Background(), 8); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteInboxMessage(context.Background(), 7, firstID); err != nil {
		t.Fatal(err)
	}
	if err := deliver(7, "fixture"); err != nil {
		t.Fatal("tombstone replay failed")
	}
	rows, err = store.ListInboxMessages(context.Background(), 7, 1, 20, false)
	if err != nil || rows.Total != 0 {
		t.Fatal("deleted message resurrected")
	}
	var tombstone model.WorkorderInboxMessage
	if err := db.First(&tombstone, firstID).Error; err != nil || tombstone.DeletedAt == nil || tombstone.Subject != "" || tombstone.Content != "" {
		t.Fatal("deleted message did not retain a scrubbed tombstone")
	}
}

func TestInboxLedgerFailureRollsBackMessage(t *testing.T) {
	db, store := inboxTestDB(t)
	if err := db.Create(&model.User{Model: model.Model{ID: 7}, Username: "user", Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().UTC().Add(time.Minute)
	event := model.WorkorderNotificationEvent{InstanceID: 12, EventType: "instance_created", Status: "processing", LeaseToken: "fixture", LeaseUntil: &deadline, Snapshot: []byte("{}"), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TRIGGER reject_inbox_ledger BEFORE INSERT ON cl_workorder_notification_delivery BEGIN SELECT RAISE(ABORT, 'fixture'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := store.DeliverInboxMessage(context.Background(), event.ID, "fixture", 3, 7, 12, event.EventType, "subject", "body"); err == nil {
		t.Fatal("injected ledger failure accepted")
	}
	var count int64
	if err := db.Model(&model.WorkorderInboxMessage{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("message escaped failed ledger transaction")
	}
}

func TestInboxConcurrentDuplicateDeliveryCreatesOneMessage(t *testing.T) {
	db, store := inboxTestDB(t)
	if err := db.Create(&model.User{Model: model.Model{ID: 7}, Username: "user", Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().UTC().Add(time.Minute)
	event := model.WorkorderNotificationEvent{InstanceID: 12, EventType: "created", Status: "processing", LeaseToken: "fixture", LeaseUntil: &deadline, Snapshot: []byte("{}"), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- store.DeliverInboxMessage(context.Background(), event.ID, "fixture", 3, 7, 12, event.EventType, "subject", "body")
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	var messages, deliveries int64
	if err := db.Model(&model.WorkorderInboxMessage{}).Count(&messages).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.WorkorderNotificationDelivery{}).Count(&deliveries).Error; err != nil {
		t.Fatal(err)
	}
	if messages != 1 || deliveries != 1 {
		t.Fatalf("messages=%d deliveries=%d", messages, deliveries)
	}
}
