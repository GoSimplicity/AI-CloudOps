//go:build integration

package dao

import (
	"context"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLInboxConcurrentDelivery(t *testing.T) {
	dsn := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("explicit isolated MySQL DSN required")
	}
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(parsed.DBName) || parsed.User != parsed.DBName {
		t.Fatal("refusing database outside generated isolated fixture")
	}
	parsed.Timeout, parsed.ReadTimeout, parsed.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	db, err := gorm.Open(mysql.Open(parsed.FormatDSN()), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal("isolated MySQL connection failed")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(8)
	defer pool.Close()
	if err := db.AutoMigrate(&model.User{}, &model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}, &model.WorkorderInboxMessage{}); err != nil {
		t.Fatal("isolated schema migration failed")
	}
	if err := db.Create(&model.User{Model: model.Model{ID: 7}, Username: "fixture", Enable: 1}).Error; err != nil {
		t.Fatal("user fixture failed")
	}
	deadline := time.Now().UTC().Add(time.Minute)
	event := model.WorkorderNotificationEvent{InstanceID: 12, EventType: "created", Status: "processing", LeaseToken: "fixture", LeaseUntil: &deadline, Snapshot: []byte("{}"), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal("event fixture failed")
	}
	store := NewWorkorderInstanceDAO(db, zap.NewNop()).(*workorderInstanceDAO)
	start := make(chan struct{})
	results := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- store.DeliverInboxMessage(context.Background(), event.ID, "fixture", 3, 7, 12, event.EventType, "subject", "body")
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal("concurrent delivery failed")
		}
	}
	var messages, deliveries int64
	if err := db.Model(&model.WorkorderInboxMessage{}).Count(&messages).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.WorkorderNotificationDelivery{}).Count(&deliveries).Error; err != nil {
		t.Fatal(err)
	}
	if messages != 1 || deliveries != 1 {
		t.Fatalf("messages=%d deliveries=%d", messages, deliveries)
	}
}
