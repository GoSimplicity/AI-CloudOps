//go:build smoke

package logger

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func TestGORMLogsDoNotExposeSQLOrDriverValues(t *testing.T) {
	var output bytes.Buffer
	outputLogger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&output), zap.DebugLevel))
	candidate := NewGORMLogger(outputLogger)
	candidate.slowThreshold = time.Nanosecond
	candidate.level = glogger.Info
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: candidate})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	if err := db.Exec("CREATE TABLE fixture_rows (value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	// SQLite真实约束错误携带驱动原文，模拟MySQL Duplicate entry中的字段值
	if err := db.Exec("CREATE TRIGGER reject_private BEFORE INSERT ON fixture_rows BEGIN SELECT RAISE(ABORT, 'driver-private-canary'); END").Error; err != nil {
		t.Fatal("fixture constraint creation failed")
	}
	output.Reset()
	if err := db.Exec("INSERT INTO fixture_rows(value) VALUES (?)", "bound-private-canary").Error; err == nil {
		t.Fatal("fixture failed to produce database error")
	}
	var value string
	if err := db.Raw("SELECT 'literal-private-canary'").Scan(&value).Error; err != nil {
		t.Fatal("fixture slow query failed")
	}
	candidate.Info(context.Background(), "message-private-canary %s", "argument-private-canary")
	candidate.Warn(context.Background(), "message-private-canary %s", "argument-private-canary")
	candidate.Error(context.Background(), "message-private-canary %v", errors.New("error-private-canary"))
	candidate.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT 'trace-private-canary'", 1 }, errors.New("wrapped-private-canary"))
	for _, marker := range []string{"driver-private-canary", "bound-private-canary", "literal-private-canary", "message-private-canary", "argument-private-canary", "error-private-canary", "trace-private-canary", "wrapped-private-canary", "INSERT INTO", "SELECT"} {
		if strings.Contains(output.String(), marker) {
			t.Error("GORM SQL/driver/framework logger exposed a private canary")
		}
	}
	if output.Len() == 0 {
		t.Fatal("suppressing all database diagnostics is not the logger contract")
	}
}
