//go:build integration

package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	mysqldriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMySQLOutboxAtomicAndDelivery(t *testing.T) {
	raw := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if raw == "" {
		t.Skip("explicit isolated MySQL DSN required")
	}
	cfg, err := mysqldriver.ParseDSN(raw)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(cfg.DBName) || cfg.User != cfg.DBName {
		t.Fatal("refusing non-isolated MySQL database")
	}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 10 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err = db.AutoMigrate(&model.WorkorderInstance{}, &model.WorkorderProcess{}, &model.WorkorderFormDesign{}, &model.WorkorderCategory{}, &model.WorkorderInstanceComment{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}, &model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}, &model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	form := model.WorkorderFormDesign{Name: "mysql-fixture", Schema: model.JSONMap{"fields": []any{}}, Status: 2}
	if err = db.Create(&form).Error; err != nil {
		t.Fatal(err)
	}
	process := model.WorkorderProcess{Name: "mysql-fixture", FormDesignID: form.ID, Definition: model.JSONMap{"steps": []any{}, "connections": []any{}}, Status: 2}
	if err = db.Create(&process).Error; err != nil {
		t.Fatal(err)
	}
	instance := model.WorkorderInstance{Title: "owned", SerialNumber: "outbox-owned-1", ProcessID: process.ID, Status: model.InstanceStatusDraft}
	if err = db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	svc := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), notificationService: &workorderNotificationService{}, logger: zap.NewNop()}
	if err = svc.DeleteInstance(context.Background(), instance.ID); err != nil {
		t.Fatal(err)
	}
	var event model.WorkorderNotificationEvent
	if err = db.Where("instance_id = ?", instance.ID).First(&event).Error; err != nil {
		t.Fatal(err)
	}
	until := time.Now().UTC().Add(time.Minute)
	if err := db.Model(&event).Updates(map[string]interface{}{"status": "processing", "lease_token": "mysql-test", "lease_until": until}).Error; err != nil {
		t.Fatal(err)
	}
	ledger := &eventDeliveryLedger{db: db, eventID: event.ID, token: "mysql-test"}
	if err = ledger.markDelivery(7, "email", "testhash", "sending", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err = ledger.markDelivery(7, "email", "testhash", "success", 0, 0); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ?", event.ID).Count(&count)
	if count != 1 {
		t.Fatalf("upsert count=%d", count)
	}
	sent, err := ledger.delivered(context.Background(), 7, "email", "testhash")
	if err != nil || !sent {
		t.Fatal("MySQL upsert did not persist success")
	}
	reviewEvent := model.WorkorderNotificationEvent{InstanceID: instance.ID, EventType: "created", Snapshot: []byte(`{}`), Status: "uncertain", NextAttemptAt: time.Now().UTC()}
	if err = db.Create(&reviewEvent).Error; err != nil {
		t.Fatal(err)
	}
	reviewDelivery := model.WorkorderNotificationDelivery{EventID: reviewEvent.ID, NotificationID: 7, Channel: "email", RecipientHash: "reviewhash", Status: "uncertain", Attempts: 1}
	if err = db.Create(&reviewDelivery).Error; err != nil {
		t.Fatal(err)
	}
	store := workdao.NewWorkorderInstanceDAO(db, zap.NewNop())
	reviewSvc := &workorderNotificationService{instanceDAO: store}
	reviewReq := &model.ResolveWorkorderNotificationDeliveryReq{DeliveryID: reviewDelivery.ID, Action: "delivered", Reason: "verified by operator"}
	if err = reviewSvc.ResolveNotificationDelivery(context.Background(), reviewEvent.ID, reviewReq, 7); err != nil {
		t.Fatal(err)
	}
	if err = reviewSvc.ResolveNotificationDelivery(context.Background(), reviewEvent.ID, reviewReq, 7); err == nil {
		t.Fatal("duplicate manual resolution succeeded")
	}
	var reviews int64
	db.Model(&model.WorkorderNotificationResolution{}).Where("event_id = ? AND operator_id = ?", reviewEvent.ID, 7).Count(&reviews)
	if reviews != 1 {
		t.Fatal("manual resolution audit missing")
	}
	if err = db.Model(&reviewEvent).Update("status", "success").Error; err != nil {
		t.Fatal(err)
	}
	rawSnapshot, marshalErr := json.Marshal(instance)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	plannedEvent := model.WorkorderNotificationEvent{InstanceID: instance.ID, EventType: model.EventTypeInstanceCreated, Snapshot: rawSnapshot, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Minute)}
	if err = db.Create(&plannedEvent).Error; err != nil {
		t.Fatal(err)
	}
	cfgNotification := &model.WorkorderNotification{Model: model.Model{ID: 14}, TriggerType: model.TriggerTypeImmediate, Status: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "mysql-fixture@example.test"}}, MessageTemplate: "fixture"}
	testSender := &eventTestSender{attempts: map[string]int{}}
	workerSvc := &workorderNotificationService{dao: &eventTestConfigDAO{config: cfgNotification}, notificationMgr: testSender, logger: zap.NewNop()}
	eventWorker := &NotificationEventWorker{db: db, service: workerSvc, logger: zap.NewNop()}
	if err = eventWorker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = db.First(&plannedEvent, plannedEvent.ID).Error; err != nil {
		t.Fatal(err)
	}
	if plannedEvent.Status != "success" || len(plannedEvent.Plan) != 0 || string(plannedEvent.Snapshot) != "{}" || testSender.attempts["mysql-fixture@example.test"] != 1 {
		t.Fatal("MySQL worker plan/snapshot behavior failed")
	}
	old := time.Now().UTC().AddDate(0, 0, -100)
	oldEvent := model.WorkorderNotificationEvent{InstanceID: instance.ID, EventType: "created", Snapshot: []byte(`{}`), Status: "success", NextAttemptAt: old, CreatedAt: old, UpdatedAt: old}
	if err = db.Create(&oldEvent).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.WorkorderNotificationDelivery{EventID: oldEvent.ID, NotificationID: 7, Channel: "email", RecipientHash: "oldhash", Status: "success"}).Error; err != nil {
		t.Fatal(err)
	}
	clean := &NotificationEventWorker{db: db, retentionDays: 90}
	deleted, cleanErr := clean.cleanupCompleted(context.Background(), time.Now().UTC())
	if cleanErr != nil || deleted != 1 {
		t.Fatalf("mysql retention failed: count=%d err=%v", deleted, cleanErr)
	}
	var remains int64
	db.Model(&model.WorkorderNotificationDelivery{}).Where("event_id = ?", oldEvent.ID).Count(&remains)
	if remains != 0 {
		t.Fatal("retention left orphan delivery")
	}
	other := model.WorkorderInstance{Title: "rollback", SerialNumber: "outbox-owned-2", ProcessID: process.ID, Status: model.InstanceStatusDraft}
	if err = db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Callback().Create().Before("gorm:create").Register("outbox:test:reject", func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_workorder_notification_event" {
			tx.AddError(errors.New("test injected failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err = svc.DeleteInstance(context.Background(), other.ID); err == nil {
		t.Fatal("missing outbox failure")
	}
	db.Model(&model.WorkorderInstance{}).Where("id = ?", other.ID).Count(&count)
	if count != 1 {
		t.Fatal("delete not rolled back with outbox failure")
	}
}
