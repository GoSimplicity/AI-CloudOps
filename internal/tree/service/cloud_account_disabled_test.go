package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDisabledCloudAccountCannotVerify(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "accounts.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal(err)
	}
	account := &model.CloudAccount{Name: "disabled", Provider: model.ProviderAWS, Status: 2, AccessKey: "not-valid-ciphertext", SecretKey: "not-valid-ciphertext"}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	svc := mustCloudAccountService(t, "", zap.NewNop(), dao.NewCloudAccountDAO(db, zap.NewNop()))
	err = svc.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: account.ID})
	if err == nil || !strings.Contains(err.Error(), "禁用") {
		t.Fatalf("disabled account must be rejected before decrypt/network: %v", err)
	}
}
