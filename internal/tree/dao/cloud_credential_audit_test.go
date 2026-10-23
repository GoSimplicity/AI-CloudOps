package dao

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCloudCredentialAuditNeverPersistsOrReturnsValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "audit.db")), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.Migrator().CreateTable(&model.CloudResourceChangeLog{}); err != nil {
		t.Fatal(err)
	}
	store := NewTreeCloudDAO(db, zap.NewNop())
	for _, field := range []string{"key", "password"} {
		t.Run(field, func(t *testing.T) {
			entry := &model.CloudResourceChangeLog{ResourceID: 1, FieldName: field, OldValue: "old-sensitive-marker", NewValue: "new-sensitive-marker"}
			if err := store.CreateChangeLog(context.Background(), entry); err != nil {
				t.Fatal(err)
			}
			var saved model.CloudResourceChangeLog
			if err := db.Where("field_name = ?", field).First(&saved).Error; err != nil {
				t.Fatal(err)
			}
			if saved.NewValue != "[REDACTED]" || saved.OldValue != "[REDACTED]" {
				t.Error("审计持久化包含凭据值")
			}
			// 历史对象即使绕过正常 DAO，也不能在读取 JSON 时回显值
			encoded, err := json.Marshal(model.CloudResourceChangeLog{FieldName: field, OldValue: "old-sensitive-marker", NewValue: "new-sensitive-marker"})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "sensitive-marker") {
				t.Error("历史审计 JSON 返回凭据值")
			}
		})
	}
	visible, err := json.Marshal(model.CloudResourceChangeLog{FieldName: "environment", NewValue: "local"})
	if err != nil || !strings.Contains(string(visible), `"new_value":"local"`) {
		t.Fatal("普通元数据审计丢失")
	}
}
