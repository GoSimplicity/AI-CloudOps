//go:build smoke

package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBootstrapAdminDBErrorDoesNotExposeDriverValue(t *testing.T) {
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "fixture-strong-password")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	table := (&model.User{}).TableName()
	if err := db.Exec("CREATE TRIGGER reject_private BEFORE INSERT ON " + table + " BEGIN SELECT RAISE(ABORT, 'bootstrap-private-canary'); END").Error; err != nil {
		t.Fatal("fixture failed to create constraint")
	}
	app := &App{db: db}
	err = app.bootstrapAdmin(context.Background())
	if err == nil {
		t.Fatal("fixture did not fail")
	}
	if strings.Contains(err.Error(), "bootstrap-private-canary") {
		t.Fatal("bootstrap return/stderr boundary exposed original driver value")
	}
	if !strings.Contains(err.Error(), "创建用户失败") {
		t.Fatal("bootstrap error lost its static stage")
	}
	var driverErr sqlite3.Error
	if !errors.As(err, &driverErr) {
		t.Fatal("redacted boundary lost the native error chain")
	}
}

func TestBootstrapAdminQueryErrorPreservesCancellation(t *testing.T) {
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "fixture-strong-password")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = (&App{db: db}).bootstrapAdmin(ctx)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "查询用户失败") {
		t.Fatal("bootstrap query boundary lost safe stage or cancellation identity")
	}
}
