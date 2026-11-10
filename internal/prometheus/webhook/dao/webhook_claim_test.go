package dao

import (
	"context"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func claimStore(t *testing.T, status model.MonitorAlertEventStatus) (*gorm.DB, WebhookDao, model.MonitorAlertEvent) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.MonitorAlertEvent{}); err != nil {
		t.Fatal(err)
	}
	event := model.MonitorAlertEvent{
		AlertName: "keep-name", Fingerprint: "fixture", Status: status, SilenceID: "keep-silence",
		RenLingUserID: 4, Labels: model.StringList{"team=a=b"},
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	return db, NewWebhookDao(zap.NewNop(), db), event
}

func TestClaimUpdatesOnlyStatusFromFiringOrUpgraded(t *testing.T) {
	for _, status := range []model.MonitorAlertEventStatus{model.MonitorAlertEventStatusFiring, model.MonitorAlertEventStatusUpgraded} {
		db, store, event := claimStore(t, status)
		if err := store.ClaimMonitorAlertEvent(context.Background(), event.ID, status); err != nil {
			t.Fatal(err)
		}
		var current model.MonitorAlertEvent
		if err := db.First(&current, event.ID).Error; err != nil {
			t.Fatal(err)
		}
		if current.Status != model.MonitorAlertEventStatusClaimed || current.SilenceID != "keep-silence" || current.RenLingUserID != 4 || current.AlertName != "keep-name" || len(current.Labels) != 1 || current.Labels[0] != "team=a=b" {
			t.Fatalf("claim rewrote more than status: %+v", current)
		}
		if err := store.ClaimMonitorAlertEvent(context.Background(), event.ID, status); err != nil {
			t.Fatal("duplicate claim was not idempotent", err)
		}
	}
}

func TestClaimRejectsOtherStatesAndStaleExpected(t *testing.T) {
	for _, status := range []model.MonitorAlertEventStatus{model.MonitorAlertEventStatusSilenced, model.MonitorAlertEventStatusResolved} {
		db, store, event := claimStore(t, status)
		if err := store.ClaimMonitorAlertEvent(context.Background(), event.ID, model.MonitorAlertEventStatusFiring); err == nil {
			t.Fatal("mismatched claim succeeded")
		}
		var current model.MonitorAlertEvent
		if err := db.First(&current, event.ID).Error; err != nil {
			t.Fatal(err)
		}
		if current.Status != status || current.SilenceID != "keep-silence" || current.RenLingUserID != 4 {
			t.Fatal("rejected claim changed the row")
		}
	}
	db, store, event := claimStore(t, model.MonitorAlertEventStatusClaimed)
	if err := store.ClaimMonitorAlertEvent(context.Background(), event.ID, model.MonitorAlertEventStatusFiring); err != nil {
		t.Fatal("claim that already landed was not idempotent", err)
	}
	var current model.MonitorAlertEvent
	if err := db.First(&current, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Status != model.MonitorAlertEventStatusClaimed || current.RenLingUserID != 4 {
		t.Fatal("idempotent claim rewrote the row")
	}
	if _, store, event := claimStore(t, model.MonitorAlertEventStatusFiring); store.ClaimMonitorAlertEvent(context.Background(), event.ID, model.MonitorAlertEventStatusSilenced) == nil {
		t.Fatal("claim accepted a status it must not write")
	}
}

func TestConcurrentClaimKeepsOneStatus(t *testing.T) {
	db, store, event := claimStore(t, model.MonitorAlertEventStatusFiring)
	var group sync.WaitGroup
	errorsSeen := make(chan error, 8)
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			errorsSeen <- store.ClaimMonitorAlertEvent(context.Background(), event.ID, model.MonitorAlertEventStatusFiring)
		}()
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	var current model.MonitorAlertEvent
	if err := db.First(&current, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Status != model.MonitorAlertEventStatusClaimed || current.SilenceID != "keep-silence" || current.RenLingUserID != 4 {
		t.Fatal("concurrent claim changed more than status")
	}
}
