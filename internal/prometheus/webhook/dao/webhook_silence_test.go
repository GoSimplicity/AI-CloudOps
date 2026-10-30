package dao

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func silenceStore(t *testing.T) (*gorm.DB, WebhookDao, model.MonitorAlertEvent) {
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
	event := model.MonitorAlertEvent{AlertName: "fixture", Fingerprint: "fixture", Status: model.MonitorAlertEventStatusSilenced, SilenceID: "old-id", Labels: model.StringList{"name=fixture"}}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	return db, NewWebhookDao(zap.NewNop(), db), event
}

func TestSilenceStatePersistence(t *testing.T) {
	for _, test := range []struct {
		name                string
		missing, legacyNULL bool
	}{{"clear persisted ID", false, false}, {"missing row", true, false}, {"legacy NULL comparison", false, true}} {
		t.Run(test.name, func(t *testing.T) {
			db, store, event := silenceStore(t)
			expected := AlertEventSilenceState{Status: event.Status, SilenceID: event.SilenceID}
			desired := AlertEventSilenceState{Status: model.MonitorAlertEventStatusFiring, SilenceID: ""}
			if test.missing {
				if err := db.Unscoped().Delete(&event).Error; err != nil {
					t.Fatal(err)
				}
			}
			if test.legacyNULL {
				if err := db.Model(&event).UpdateColumn("silence_id", nil).Error; err != nil {
					t.Fatal(err)
				}
				expected.SilenceID = ""
			}
			err := store.UpdateMonitorAlertEventSilence(context.Background(), event.ID, expected, desired)
			if test.missing {
				if !errors.Is(err, ErrAlertSilenceConflict) {
					t.Fatal("missing row returned success", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var empty bool
			if err := db.Raw("SELECT silence_id IS NOT NULL AND silence_id = '' FROM cl_monitor_alert_events WHERE id = ?", event.ID).Scan(&empty).Error; err != nil || !empty {
				t.Fatal("empty silence ID was omitted from SQL update")
			}
			// The old-state predicate now misses. Reading back the exact desired state
			// must accept an explicit duplicate without treating every zero as missing.
			if err := store.UpdateMonitorAlertEventSilence(context.Background(), event.ID, expected, desired); err != nil {
				t.Fatal("same desired state was not idempotent", err)
			}
		})
	}
}

func TestConcurrentSilenceStateUpdates(t *testing.T) {
	for _, different := range []bool{false, true} {
		name := "same desired state"
		if different {
			name = "conflicting desired state"
		}
		t.Run(name, func(t *testing.T) {
			db, store, event := silenceStore(t)
			expected := AlertEventSilenceState{Status: event.Status, SilenceID: event.SilenceID}
			desired := [2]AlertEventSilenceState{{Status: model.MonitorAlertEventStatusSilenced, SilenceID: "new-a"}, {Status: model.MonitorAlertEventStatusSilenced, SilenceID: "new-a"}}
			if different {
				desired[1].SilenceID = "new-b"
			}
			start := make(chan struct{})
			var wg sync.WaitGroup
			results := make([]error, 2)
			for i := range desired {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					results[i] = store.UpdateMonitorAlertEventSilence(context.Background(), event.ID, expected, desired[i])
				}()
			}
			close(start)
			wg.Wait()
			successes := 0
			for _, err := range results {
				if err == nil {
					successes++
				} else if !errors.Is(err, ErrAlertSilenceConflict) {
					t.Fatal(err)
				}
			}
			want := 2
			if different {
				want = 1
			}
			if successes != want {
				t.Fatalf("unsafe success count %d", successes)
			}
			var current model.MonitorAlertEvent
			if err := db.First(&current, event.ID).Error; err != nil {
				t.Fatal(err)
			}
			if current.SilenceID != "new-a" && (!different || current.SilenceID != "new-b") {
				t.Fatal("unexpected persisted silence state")
			}
		})
	}
}

func TestGenericAlertUpdateRetainsUpgradeFields(t *testing.T) {
	db, store, event := silenceStore(t)
	event.Status = model.MonitorAlertEventStatusUpgraded
	event.RenLingUserID = 9
	event.AlertName = "updated-name"
	if err := store.UpdateMonitorAlertEvent(context.Background(), &event); err != nil {
		t.Fatal(err)
	}
	var current model.MonitorAlertEvent
	if err := db.First(&current, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Status != event.Status || current.RenLingUserID != 9 || current.AlertName != "updated-name" {
		t.Fatal("generic event update was narrowed to silence fields")
	}
}

func TestSilenceCASUsesExactIDWithCaseInsensitiveColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	if err := db.Exec("CREATE TABLE cl_monitor_alert_events (id INTEGER PRIMARY KEY,status INTEGER,silence_id TEXT COLLATE NOCASE,updated_at DATETIME,deleted_at INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO cl_monitor_alert_events(id,status,silence_id,deleted_at) VALUES (7,2,'NEW-ID',0)").Error; err != nil {
		t.Fatal(err)
	}
	store := NewWebhookDao(zap.NewNop(), db)
	err = store.UpdateMonitorAlertEventSilence(context.Background(), 7, AlertEventSilenceState{Status: model.MonitorAlertEventStatusSilenced, SilenceID: "new-id"}, AlertEventSilenceState{Status: model.MonitorAlertEventStatusFiring})
	if !errors.Is(err, ErrAlertSilenceConflict) {
		t.Fatal("case-insensitive SQL comparison accepted a different original ID", err)
	}
	var id string
	if err := db.Raw("SELECT silence_id FROM cl_monitor_alert_events WHERE id=7").Scan(&id).Error; err != nil || id != "NEW-ID" {
		t.Fatal("stale completion changed the differently-cased ID")
	}
}
