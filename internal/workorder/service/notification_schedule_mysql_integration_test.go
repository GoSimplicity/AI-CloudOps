//go:build integration

package service

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMySQLScheduleAtomicConcurrentRestart(t *testing.T) {
	raw := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if raw == "" {
		t.Skip("explicit isolated MySQL DSN required")
	}
	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(cfg.DBName) || cfg.User != cfg.DBName {
		t.Fatal("refusing non-isolated database")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotification{}, &model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}, &model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	instance := model.WorkorderInstance{Model: model.Model{ID: 12}, ProcessID: 1, Status: model.InstanceStatusPending}
	snapshot, _ := json.Marshal(instance)
	parent := model.WorkorderNotificationEvent{InstanceID: 12, EventType: model.EventTypeInstanceCreated, Snapshot: snapshot, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	interval, count := 1, 2
	config := scheduleConfig(model.TriggerTypeImmediate)
	config.RepeatInterval, config.RepeatCount = &interval, &count
	sender := &eventTestSender{attempts: map[string]int{}}
	service := &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}
	runTwo := func() {
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker := &NotificationEventWorker{db: db, service: service, logger: zap.NewNop()}
				_ = worker.processOne(context.Background())
			}()
		}
		wg.Wait()
	}
	runTwo()
	var children []model.WorkorderNotificationEvent
	if err := db.Where("parent_event_id = ?", parent.ID).Find(&children).Error; err != nil {
		t.Fatal(err)
	}
	// One worker can also claim the freshly due child; check no duplicate occurrence.
	seen := map[int]bool{}
	for _, child := range children {
		if child.Occurrence == nil || seen[*child.Occurrence] {
			t.Fatal("duplicate occurrence")
		}
		seen[*child.Occurrence] = true
	}
	if !seen[1] || len(children) > 2 {
		t.Fatalf("invalid child count=%d", len(children))
	}
	// Restarted worker processes only due records. Ensure first occurrence is due.
	if err := db.Model(&model.WorkorderNotificationEvent{}).Where("parent_event_id = ? AND occurrence = 1 AND status = ?", parent.ID, "pending").Update("next_attempt_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	runTwo()
	children = nil
	if err := db.Where("parent_event_id = ?", parent.ID).Order("occurrence").Find(&children).Error; err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 || children[0].Status != "success" || children[1].Status != "pending" {
		t.Fatalf("unexpected state count=%d", len(children))
	}
	if sender.attempts["recipient@example.test"] != 1 {
		t.Fatal("first occurrence duplicated")
	}
	initial := time.Now().UTC().Add(time.Hour)
	row := model.WorkorderNotification{Name: "mysql-cas-fixture", TriggerType: model.TriggerTypeScheduled, TriggerVersion: 1, ScheduledTime: &initial, EventTypes: model.StringList{"instance_created"}, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}, MessageTemplate: "fixture"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	version := row.Version
	later := initial.Add(time.Hour)
	requests := []*model.UpdateWorkorderNotificationReq{
		{ID: row.ID, TriggerType: model.TriggerTypeImmediate, ClearTrigger: true, ExpectedVersion: &version},
		{ID: row.ID, ScheduledTime: &later, ExpectedVersion: &version},
	}
	store := workdao.NewNotificationDAO(db, zap.NewNop())
	errors := make(chan error, len(requests))
	var wg sync.WaitGroup
	for _, request := range requests {
		wg.Add(1)
		go func(request *model.UpdateWorkorderNotificationReq) {
			defer wg.Done()
			errors <- store.UpdateNotification(context.Background(), request)
		}(request)
	}
	wg.Wait()
	close(errors)
	succeeded := 0
	for err := range errors {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("concurrent version CAS successes=%d", succeeded)
	}
	var final model.WorkorderNotification
	if err := db.First(&final, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateAutomaticNotification(&final); err != nil {
		t.Fatalf("concurrent writes produced invalid config: %v", err)
	}
	// The two loc modes are tested sequentially on fresh records; mixing loc
	// against existing outstanding DATETIME rows is explicitly unsupported.
	if err := db.Model(&model.WorkorderNotificationEvent{}).Where("status = ?", "pending").Update("status", "skipped").Error; err != nil {
		t.Fatal(err)
	}

	localCfg := *cfg
	localCfg.Loc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	localDB, err := gorm.Open(mysql.Open(localCfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("local-zone test connection failed")
	}
	localPool, _ := localDB.DB()
	defer localPool.Close()
	future := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	localRow := model.WorkorderNotification{Name: "timezone-fixture", TriggerType: model.TriggerTypeScheduled, TriggerVersion: 1, ScheduledTime: &future, EventTypes: model.StringList{"instance_created"}, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}, MessageTemplate: "fixture"}
	if err := localDB.Create(&localRow).Error; err != nil {
		t.Fatal(err)
	}
	var loaded model.WorkorderNotification
	if err := localDB.First(&loaded, localRow.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateAutomaticNotification(&loaded); err != nil {
		t.Fatalf("local DB time rejected: %v", err)
	}
	if !loaded.ScheduledTime.Equal(future) {
		t.Fatal("local DB time changed absolute instant")
	}
	loaded.RecipientTypes = model.StringList{"custom"}
	loaded.RecipientAddresses = model.NotificationAddresses{{Channel: "email", Address: "local@example.test"}}
	localParent := model.WorkorderNotificationEvent{InstanceID: 12, EventType: model.EventTypeInstanceCreated, Snapshot: snapshot, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
	if err := localDB.Create(&localParent).Error; err != nil {
		t.Fatal(err)
	}
	localWorker := &NotificationEventWorker{db: localDB, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: &loaded}, notificationMgr: sender, logger: zap.NewNop()}, logger: zap.NewNop()}
	if err := localWorker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var localChild model.WorkorderNotificationEvent
	if err := localDB.Where("parent_event_id = ?", localParent.ID).First(&localChild).Error; err != nil {
		t.Fatal(err)
	}
	if localChild.Status != "pending" || !localChild.NextAttemptAt.Equal(future) {
		t.Fatal("local-zone schedule due instant changed")
	}
}
