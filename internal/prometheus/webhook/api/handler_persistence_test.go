package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSilenceHandlersPersistThroughRealDAO(t *testing.T) {
	for _, test := range []struct {
		name, action, mutation string
		success                bool
	}{
		{"silence", "silence", "", true}, {"unsilence clears DB ID", "unsilence", "", true},
		{"silence deleted row", "silence", "delete", false}, {"unsilence deleted row", "unsilence", "delete", false},
		{"late silence cannot replace newer ID", "silence", "new-id", false}, {"late unsilence cannot clear newer ID", "unsilence", "new-id", false},
		{"identical local completion is idempotent", "unsilence", "already-desired", true},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			defer pool.Close()
			if err := db.AutoMigrate(&model.MonitorAlertEvent{}); err != nil {
				t.Fatal(err)
			}
			initial := model.MonitorAlertEventStatusFiring
			silenceID := ""
			if test.action == "unsilence" {
				initial = model.MonitorAlertEventStatusSilenced
				silenceID = "old-id"
			}
			event := model.MonitorAlertEvent{AlertName: "fixture", Fingerprint: "fixture", Status: initial, SilenceID: silenceID, Labels: model.StringList{"name=fixture"}}
			if err := db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			handler, err := NewWebHookHandler(&config.WebhookConfig{AlertManagerAPI: "https://fixture.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}, zap.NewNop(), dao.NewWebhookDao(zap.NewNop(), db), nil)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
				calls++
				// Another real database writer changes/deletes the row while the handler
				// awaits its successful upstream response. No fake CAS result is injected.
				switch test.mutation {
				case "delete":
					if err := db.Unscoped().Delete(&event).Error; err != nil {
						t.Fatal(err)
					}
				case "new-id":
					if err := db.Model(&event).Updates(map[string]any{"status": model.MonitorAlertEventStatusSilenced, "silence_id": "newer-id"}).Error; err != nil {
						t.Fatal(err)
					}
				case "already-desired":
					if err := db.Model(&event).Updates(map[string]any{"status": model.MonitorAlertEventStatusFiring, "silence_id": ""}).Error; err != nil {
						t.Fatal(err)
					}
				}
				return fixtureResponse(request, 200, `{"silenceID":"created-id"}`), nil
			})
			engine := gin.New()
			handler.RegisterRouters(engine)
			recorder := httptest.NewRecorder()
			path := "/api/v1/alerts/" + test.action + "?fingerprint=fixture&hour=1"
			engine.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
			var result struct {
				Code    int
				Message string
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("handler retried an upstream side effect")
			}
			if (result.Code == 0) != test.success {
				t.Fatal("database missing/conflict was falsely confirmed")
			}
			if !test.success {
				if result.Message == "" {
					t.Fatal("unconfirmed outcome has no message")
				}
				if test.mutation == "new-id" {
					var current model.MonitorAlertEvent
					if err := db.WithContext(context.Background()).First(&current, event.ID).Error; err != nil {
						t.Fatal(err)
					}
					if current.Status != model.MonitorAlertEventStatusSilenced || current.SilenceID != "newer-id" {
						t.Fatal("stale upstream completion overwrote newer local state")
					}
				}
				return
			}
			var current model.MonitorAlertEvent
			if err := db.First(&current, event.ID).Error; err != nil {
				t.Fatal(err)
			}
			if test.action == "unsilence" {
				if current.Status != model.MonitorAlertEventStatusFiring || current.SilenceID != "" {
					t.Fatal("unsilence did not persist the empty ID")
				}
			} else if current.Status != model.MonitorAlertEventStatusSilenced || current.SilenceID != "created-id" {
				t.Fatal("silence desired state was not persisted")
			}
		})
	}
}
