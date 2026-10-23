package logger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func TestGORMLoggerPreservesSafeMetadataAndLevels(t *testing.T) {
	core, events := observer.New(zap.DebugLevel)
	original := NewGORMLogger(zap.New(core))
	verbose := original.LogMode(glogger.Info)
	ctx := context.Background()
	query := func() (string, int64) { return "private-sql-text", 7 }
	original.Trace(ctx, time.Now(), query, nil)
	if events.Len() != 0 {
		t.Fatal("default level should not log fast successful queries")
	}
	verbose.Trace(ctx, time.Now(), query, fmt.Errorf("private-value: %w", context.DeadlineExceeded))
	entry := events.All()[0]
	if entry.Level != zap.ErrorLevel || entry.ContextMap()["rows"] != int64(7) || entry.ContextMap()["error_type"] != "deadline_exceeded" {
		t.Fatal("safe failure diagnosis lost")
	}
	if _, ok := entry.ContextMap()["elapsed"]; !ok {
		t.Fatal("elapsed diagnostic missing")
	}
	for _, entry := range events.All() {
		for key := range entry.ContextMap() {
			if key != "rows" && key != "elapsed" && key != "error_type" {
				t.Fatal("unexpected query metadata field")
			}
		}
	}
	silent := original.LogMode(glogger.Silent)
	called := false
	silent.Trace(ctx, time.Now(), func() (string, int64) { called = true; return "private", 0 }, errors.New("private"))
	silent.Info(ctx, "private")
	silent.Warn(ctx, "private")
	silent.Error(ctx, "private")
	if called || events.Len() != 1 {
		t.Fatal("silent mode executed formatter or logged")
	}
	original.Info(ctx, "private")
	if events.Len() != 1 {
		t.Fatal("LogMode mutated original logger")
	}
}
func TestGORMLoggerConcurrentModesDoNotShareMutableLevels(t *testing.T) {
	core, events := observer.New(zap.DebugLevel)
	logger := NewGORMLogger(zap.New(core))
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			local := logger.LogMode(glogger.Info)
			local.Trace(context.Background(), time.Now(), func() (string, int64) { return "private", 1 }, nil)
		}()
	}
	group.Wait()
	if events.Len() != 32 || logger.level != glogger.Warn {
		t.Fatal("concurrent mode changed logger behavior")
	}
}
func TestDatabaseErrorTypesDoNotUseErrorText(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want string
	}{
		{fmt.Errorf("private: %w", context.Canceled), "context_canceled"},
		{fmt.Errorf("private: %w", context.DeadlineExceeded), "deadline_exceeded"},
		{gorm.ErrRecordNotFound, "record_not_found"}, {sql.ErrNoRows, "record_not_found"},
		{errors.New("Duplicate entry 'private-token' for key 'credential'"), "*errors.errorString"},
	} {
		if DatabaseErrorType(tt.err) != tt.want {
			t.Fatal("error classifier lost safe identity")
		}
	}
}

func TestDatabaseErrorWrapRedactsTextAndPreservesChain(t *testing.T) {
	original := fmt.Errorf("driver-private: %w", context.DeadlineExceeded)
	wrapped := WrapDatabaseError("数据库测试阶段失败", original)
	if !errors.Is(wrapped, context.DeadlineExceeded) || errors.Unwrap(wrapped) != original {
		t.Fatal("safe boundary lost error identity")
	}
	if wrapped.Error() != "数据库测试阶段失败 (deadline_exceeded)" {
		t.Fatal("safe boundary lost stage or exposed driver message")
	}
	if WrapDatabaseError("unused", nil) != nil {
		t.Fatal("nil error converted to failure")
	}
}
