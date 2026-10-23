//go:build integration

package dao

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"os"
	"regexp"
	"testing"
	"time"

	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLNotificationIdempotentUpdate(t *testing.T) {
	dsn := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("explicit isolated MySQL DSN required")
	}
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(parsed.DBName) || parsed.User != parsed.DBName {
		t.Fatal("refusing database outside generated isolated fixture")
	}
	parsed.Timeout = 5 * time.Second
	parsed.ReadTimeout = 10 * time.Second
	parsed.WriteTimeout = 10 * time.Second
	db, err := gorm.Open(mysql.Open(parsed.FormatDSN()), &gorm.Config{NowFunc: func() time.Time { return time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC) }, Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal("isolated MySQL connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotification{}); err != nil {
		t.Fatal("migration failed")
	}
	ctx := context.Background()
	store := NewNotificationDAO(db, zap.NewNop())
	row := &model.WorkorderNotification{Name: "owned-idempotent", Status: 1}
	if err := db.Create(row).Error; err != nil {
		t.Fatal("fixture create failed")
	}
	request := &model.UpdateWorkorderNotificationReq{ID: row.ID, Status: 1}
	for i := 0; i < 2; i++ {
		if err := store.UpdateNotification(ctx, request); err != nil {
			t.Fatal("existing same-value update incorrectly failed")
		}
	}
	// 固定UpdatedAt且值相同，确认MySQL确实报告零变更而非依靠时间戳掩盖问题
	result := db.Model(&model.WorkorderNotification{}).Where("id = ?", row.ID).Updates(map[string]any{"status": 1})
	if result.Error != nil || result.RowsAffected != 0 {
		t.Fatal("fixture did not reproduce MySQL unchanged-row semantics")
	}
	var count int64
	if err := db.Model(&model.WorkorderNotification{}).Where("id = ? AND status = ?", row.ID, 1).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("idempotent update lost row")
	}
	if err := store.UpdateNotification(ctx, &model.UpdateWorkorderNotificationReq{ID: row.ID + 9999, Status: 1}); err == nil {
		t.Fatal("missing row reported successful")
	}
	if err := db.Delete(row).Error; err != nil {
		t.Fatal("fixture delete failed")
	}
	if err := store.UpdateNotification(ctx, request); err == nil {
		t.Fatal("deleted row reported successful")
	}
}
