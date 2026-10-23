package di

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWebhookDatabaseInitializationPanicAndLogsAreRedacted(t *testing.T) {
	// 无效时区由真实MySQL驱动在拨号前拒绝，原始错误包含canary
	dsn := MySQLDSN("fixture:private-password@tcp(localhost:3306)/fixture?loc=driver-private-canary")
	core, logs := observer.New(zap.DebugLevel)
	var failure any
	func() { defer func() { failure = recover() }(); InitDB(zap.New(core), dsn) }()
	err, ok := failure.(error)
	if !ok {
		t.Fatal("fixture did not produce initialization error")
	}
	if strings.Contains(err.Error(), "private-") || !strings.Contains(err.Error(), "Webhook数据库初始化失败") {
		t.Fatal("webhook initialization panic leaked configuration or lost stage")
	}
	cause := errors.Unwrap(err)
	if cause == nil || !strings.Contains(cause.Error(), "driver-private-canary") {
		t.Fatal("safe boundary lost real driver cause")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), "private-") {
			t.Fatal("webhook initialization logged driver value")
		}
	}
}
