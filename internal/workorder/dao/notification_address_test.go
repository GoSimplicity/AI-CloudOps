package dao

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTypedAddressesPersistAndClear(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "notification.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.WorkorderNotification{}); err != nil {
		t.Fatal(err)
	}
	dao := NewNotificationDAO(db, zap.NewNop())
	ctx := context.Background()
	req := &model.CreateWorkorderNotificationReq{Name: "local", EventTypes: []string{"instance_created"}, Channels: []string{"email"}, RecipientTypes: []string{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "recipient@example.test", Name: "Inbox"}}, TriggerType: "immediate", MessageTemplate: "test", Status: 1}
	if err = dao.CreateNotification(ctx, req); err != nil {
		t.Fatal(err)
	}
	var row model.WorkorderNotification
	if err = db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if len(row.RecipientAddresses) != 1 || row.RecipientAddresses[0].Name != "Inbox" {
		t.Fatal("typed addresses did not persist")
	}
	if err = dao.UpdateNotification(ctx, &model.UpdateWorkorderNotificationReq{ID: row.ID, RecipientAddresses: model.NotificationAddresses{}}); err != nil {
		t.Fatal(err)
	}
	if err = db.First(&row, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(row.RecipientAddresses) != 0 {
		t.Fatal("explicit empty array did not clear addresses")
	}
}
