//go:build integration

package di

import (
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMySQLNotificationTimeZoneGuard(t *testing.T) {
	raw := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if raw == "" {
		t.Skip("isolated MySQL required")
	}
	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(cfg.DBName) || cfg.User != cfg.DBName {
		t.Fatal("refusing non-isolated MySQL")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotificationClock{}); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, zone := range []string{"UTC", "Asia/Shanghai"} {
		wg.Add(1)
		go func(zone string) { defer wg.Done(); results <- CheckNotificationTimeZone(db, zone) }(zone)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("conflicting first registrations succeeded=%d", success)
	}
	if err := db.Where("id = ?", 1).Delete(&model.WorkorderNotificationClock{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := CheckNotificationTimeZone(db, "UTC"); err != nil {
		t.Fatalf("new database: %v", err)
	}
	if err := CheckNotificationTimeZone(db, "UTC"); err != nil {
		t.Fatalf("same timezone: %v", err)
	}
	if err := CheckNotificationTimeZone(db, "Asia/Shanghai"); err == nil {
		t.Fatal("different instance timezone accepted")
	}
	if err := db.Where("id = ?", 1).Delete(&model.WorkorderNotificationClock{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.WorkorderNotificationEvent{}); err != nil {
		t.Fatal(err)
	}
	event := model.WorkorderNotificationEvent{InstanceID: 1, EventType: "fixture", Status: "pending", Snapshot: []byte("{}"), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := CheckNotificationTimeZone(db, "Asia/Shanghai"); err == nil {
		t.Fatal("unverified historical values accepted")
	}
	var n int64
	db.Model(&model.WorkorderNotificationClock{}).Count(&n)
	if n != 0 {
		t.Fatal("legacy zone was implicitly marked")
	}
	if err := db.Create(&model.WorkorderNotificationClock{ID: 1, TimeZone: "Asia/Shanghai"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := CheckNotificationTimeZone(db, "Asia/Shanghai"); err != nil {
		t.Fatalf("verified zone rejected: %v", err)
	}
	if err := CheckNotificationTimeZone(db, "UTC"); err == nil {
		t.Fatal("different zone accepted")
	}
}
