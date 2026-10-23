package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationFieldPresenceSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "presence.db")), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	assertNotificationFieldPresence(t, db)
}

func assertNotificationFieldPresence(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&model.WorkorderNotification{}); err != nil {
		t.Fatal(err)
	}
	store := NewNotificationDAO(db, zap.NewNop())
	ctx := context.Background()
	create := func(t *testing.T, name, optional string) model.WorkorderNotification {
		t.Helper()
		var request model.CreateWorkorderNotificationReq
		body := fmt.Sprintf(`{"name":%q,"event_types":["instance_created"],"channels":["email"],"recipient_types":["creator"],"trigger_type":"immediate","message_template":"fixture","description":"before","subject_template":"before","status":2,"is_default":1%s}`, name, optional)
		if err := json.Unmarshal([]byte(body), &request); err != nil {
			t.Fatal(err)
		}
		if err := store.CreateNotification(ctx, &request); err != nil {
			t.Fatal("notification fixture create failed")
		}
		var saved model.WorkorderNotification
		if err := db.Where("name = ?", name).First(&saved).Error; err != nil {
			t.Fatal(err)
		}
		return saved
	}
	update := func(t *testing.T, id int, optional string) {
		t.Helper()
		var request model.UpdateWorkorderNotificationReq
		if err := json.Unmarshal([]byte(fmt.Sprintf(`{"id":%d%s}`, id, optional)), &request); err != nil {
			t.Fatal(err)
		}
		if err := store.UpdateNotification(ctx, &request); err != nil {
			t.Fatal("notification fixture update failed")
		}
	}
	t.Run("omitted create uses three", func(t *testing.T) {
		if create(t, "presence-default", "").MaxRetries != 3 {
			t.Fatal("omitted retry default changed")
		}
	})
	t.Run("explicit create zero persists", func(t *testing.T) {
		if create(t, "presence-zero", `,"max_retries":0`).MaxRetries != 0 {
			t.Fatal("explicit zero retry became default three")
		}
	})
	t.Run("status only preserves optional fields", func(t *testing.T) {
		row := create(t, "presence-omitted", `,"max_retries":5`)
		update(t, row.ID, `,"status":2`)
		var saved model.WorkorderNotification
		db.First(&saved, row.ID)
		if saved.IsDefault != 1 || saved.Description != "before" || saved.SubjectTemplate != "before" || saved.MaxRetries != 5 {
			t.Fatal("omitted metadata or default flag overwritten")
		}
	})
	t.Run("explicit clears and zero persist", func(t *testing.T) {
		row := create(t, "presence-clear", `,"max_retries":5`)
		update(t, row.ID, `,"description":"","subject_template":"","max_retries":0,"is_default":2`)
		var saved model.WorkorderNotification
		db.First(&saved, row.ID)
		if saved.Description != "" || saved.SubjectTemplate != "" || saved.MaxRetries != 0 || saved.IsDefault != 2 {
			t.Fatal("explicit empty or zero values did not persist")
		}
	})
	t.Run("create zero override failure rolls back insert", func(t *testing.T) {
		setup := "CREATE TRIGGER fixture_reject_zero BEFORE UPDATE ON cl_workorder_notification WHEN NEW.name = 'rejected-zero' AND NEW.max_retries = 0 BEGIN SELECT RAISE(ABORT, 'fixture rejected'); END"
		cleanup := "DROP TRIGGER fixture_reject_zero"
		if db.Dialector.Name() == "mysql" {
			setup = "ALTER TABLE cl_workorder_notification ADD CONSTRAINT fixture_reject_zero CHECK (name <> 'rejected-zero' OR max_retries <> 0)"
			cleanup = "ALTER TABLE cl_workorder_notification DROP CHECK fixture_reject_zero"
		}
		if err := db.Exec(setup).Error; err != nil {
			t.Fatal("failure fixture setup failed")
		}
		defer db.Exec(cleanup)
		var request model.CreateWorkorderNotificationReq
		if err := json.Unmarshal([]byte(`{"name":"rejected-zero","status":2,"max_retries":0}`), &request); err != nil {
			t.Fatal(err)
		}
		if err := store.CreateNotification(ctx, &request); err == nil {
			t.Fatal("failed zero override reported success")
		}
		var count int64
		if err := db.Model(&model.WorkorderNotification{}).Where("name = ?", request.Name).Count(&count).Error; err != nil || count != 0 {
			t.Fatal("failed zero override left a default-three row")
		}
	})
}

func TestNotificationOptionalValueValidation(t *testing.T) {
	for _, body := range []string{`{"id":1,"max_retries":-1}`, `{"id":1,"max_retries":101}`, `{"id":1,"is_default":0}`, `{"id":1,"message_template":""}`} {
		var request model.UpdateWorkorderNotificationReq
		if err := json.Unmarshal([]byte(body), &request); err != nil {
			t.Fatal(err)
		}
		if err := binding.Validator.ValidateStruct(&request); err == nil {
			t.Errorf("invalid explicit notification value accepted: %s", body)
		}
	}
}
