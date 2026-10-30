package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAlertEventDAOFailureLogsAreRedacted(t *testing.T) {
	for _, operation := range []string{"query", "update", "silence_update", "silence_readback"} {
		t.Run(operation, func(t *testing.T) {
			const canary = "PRIVATE_DAO_CANARY"
			core, logs := observer.New(zap.DebugLevel)
			logger := zap.New(core)
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: databaseLogger.NewGORMLogger(logger)})
			if err != nil {
				t.Fatal(err)
			}
			pool, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			if operation == "silence_readback" {
				if err := db.AutoMigrate(&model.MonitorAlertEvent{}); err != nil {
					t.Fatal(err)
				}
				row := model.MonitorAlertEvent{Model: model.Model{ID: 7}, Status: model.MonitorAlertEventStatusResolved, SilenceID: "other-id", AlertName: "fixture", Fingerprint: "fixture", Labels: model.StringList{"fixture"}}
				if err := db.Create(&row).Error; err != nil {
					t.Fatal(err)
				}
			}
			inject := func(tx *gorm.DB) { tx.AddError(errors.New(canary)) }
			if operation == "query" || operation == "silence_readback" {
				if err := db.Callback().Query().Before("gorm:query").Register("fixture_private_error", inject); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := db.Callback().Update().Before("gorm:update").Register("fixture_private_error", inject); err != nil {
					t.Fatal(err)
				}
			}
			dao := NewWebhookDao(logger, db)
			if operation == "query" {
				_, err = dao.GetMonitorAlertEventByFingerprintId(context.Background(), canary)
			} else if operation == "silence_update" || operation == "silence_readback" {
				err = dao.UpdateMonitorAlertEventSilence(context.Background(), 7, AlertEventSilenceState{Status: model.MonitorAlertEventStatusFiring}, AlertEventSilenceState{Status: model.MonitorAlertEventStatusSilenced, SilenceID: canary})
			} else {
				err = dao.UpdateMonitorAlertEvent(context.Background(), &model.MonitorAlertEvent{Model: model.Model{ID: 7}, Fingerprint: canary, AlertName: canary, SilenceID: canary, Labels: model.StringList{canary}})
			}
			if err == nil || strings.Contains(err.Error(), canary) {
				t.Fatal("DAO did not return a sanitized failure")
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
					t.Fatal("DAO exposed private error, fingerprint or event")
				}
			}
		})
	}
}
