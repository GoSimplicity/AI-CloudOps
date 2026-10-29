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

func cancelTestDB(t *testing.T) (*gorm.DB, *workorderInstanceDAO) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cancel.db")+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err = db.AutoMigrate(&model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}, &model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	return db, NewWorkorderInstanceDAO(db, zap.NewNop()).(*workorderInstanceDAO)
}

func TestCancelScheduledNotificationStateAndAudit(t *testing.T) {
	for _, tc := range []struct {
		name, status, delivery string
		parent                 bool
		allowed                bool
	}{
		{"pending child", "pending", "", true, true},
		{"explicitly rejected", "pending", "failed", true, true},
		{"root event", "pending", "", false, false},
		{"processing", "processing", "", true, false},
		{"success event", "success", "", true, false},
		{"uncertain event", "uncertain", "", true, false},
		{"sending recipient", "pending", "sending", true, false},
		{"uncertain recipient", "pending", "uncertain", true, false},
		{"delivered recipient", "pending", "success", true, false},
		{"unknown recipient", "pending", "unexpected", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, store := cancelTestDB(t)
			parent := uint64(123)
			event := model.WorkorderNotificationEvent{InstanceID: 9, EventType: "fixture", Status: tc.status, Snapshot: []byte("sensitive-snapshot"), Plan: []byte("sensitive-plan"), Content: "sensitive-content", NextAttemptAt: time.Now().UTC().Add(time.Hour)}
			if tc.parent {
				event.ParentEventID = &parent
			}
			if err := db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			if tc.delivery != "" {
				if err := db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 1, Channel: "email", RecipientHash: "fixture", Status: tc.delivery}).Error; err != nil {
					t.Fatal(err)
				}
			}
			err := store.CancelScheduledNotification(context.Background(), event.ID, "  verified cancellation  ", 7)
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, err)
			}
			var saved model.WorkorderNotificationEvent
			if err := db.First(&saved, event.ID).Error; err != nil {
				t.Fatal(err)
			}
			var audits []model.WorkorderNotificationResolution
			if err := db.Find(&audits).Error; err != nil {
				t.Fatal(err)
			}
			if tc.allowed {
				if saved.Status != "cancelled" || len(saved.Snapshot) != 0 || len(saved.Plan) != 0 || saved.Content != "" {
					t.Fatalf("uncleared event: %+v", saved)
				}
				if len(audits) != 1 || audits[0].Action != "cancel_scheduled" || audits[0].OperatorID != 7 || audits[0].Reason != "verified cancellation" {
					t.Fatalf("bad audit: %+v", audits)
				}
				if err := store.CancelScheduledNotification(context.Background(), event.ID, "duplicate cancellation", 7); err == nil {
					t.Fatal("cancelled twice")
				}
			} else if saved.Status != tc.status || string(saved.Snapshot) != "sensitive-snapshot" || string(saved.Plan) != "sensitive-plan" || len(audits) != 0 {
				t.Fatal("rejected cancellation modified state")
			}
		})
	}
}

func TestCancelAuditFailureRollsBackSnapshotAndStatus(t *testing.T) {
	db, store := cancelTestDB(t)
	parent := uint64(123)
	event := model.WorkorderNotificationEvent{ParentEventID: &parent, Status: "pending", Snapshot: []byte("snapshot"), Plan: []byte("plan"), Content: "body", NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TRIGGER reject_cancel_audit BEFORE INSERT ON cl_workorder_notification_resolution BEGIN SELECT RAISE(ABORT, 'fixture'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := store.CancelScheduledNotification(context.Background(), event.ID, "verified cancellation", 7); err == nil {
		t.Fatal("audit failure accepted")
	}
	var got model.WorkorderNotificationEvent
	if err := db.First(&got, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != "pending" || string(got.Snapshot) != "snapshot" || string(got.Plan) != "plan" || got.Content != "body" {
		t.Fatal("event was not rolled back")
	}
}

func TestCancelAndClaimCannotBothWin(t *testing.T) {
	db, store := cancelTestDB(t)
	parent := uint64(123)
	event := model.WorkorderNotificationEvent{ParentEventID: &parent, Status: "pending", Snapshot: []byte("snapshot"), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var cancelErr error
	var claimed int64
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		cancelErr = store.CancelScheduledNotification(context.Background(), event.ID, "verified cancellation", 7)
	}()
	go func() {
		defer wg.Done()
		<-start
		result := db.Model(&model.WorkorderNotificationEvent{}).Where("id = ? AND status = ?", event.ID, "pending").Update("status", "processing")
		if result.Error == nil {
			claimed = result.RowsAffected
		}
	}()
	close(start)
	wg.Wait()
	if cancelErr == nil && claimed != 0 {
		t.Fatal("both cancellation and claim succeeded")
	}
	if cancelErr != nil && claimed != 1 {
		t.Fatalf("neither cancellation nor claim succeeded: %v", cancelErr)
	}
	var got model.WorkorderNotificationEvent
	if err := db.First(&got, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if cancelErr == nil && got.Status != "cancelled" {
		t.Fatal("cancel success lost")
	}
	if claimed == 1 && got.Status != "processing" {
		t.Fatal("claim success lost")
	}
}
