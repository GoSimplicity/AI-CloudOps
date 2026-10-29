package service

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

func TestCommentAndEventCommitTogetherAndRecoverAfterRequest(t *testing.T) {
	db := eventTestDB(t)
	if err := db.AutoMigrate(&model.User{}, &model.WorkorderInboxMessage{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.User{Model: model.Model{ID: 7}, Username: "fixture", Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	instance := model.WorkorderInstance{Title: "fixture", SerialNumber: "comment-outbox", ProcessID: 1, Status: model.InstanceStatusPending, OperatorID: 7, OperatorName: "fixture"}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	instanceDAO := workdao.NewWorkorderInstanceDAO(db, zap.NewNop())
	config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, TriggerVersion: 1, Status: 1, Channels: model.StringList{"in_app"}, RecipientTypes: model.StringList{"creator"}, MessageTemplate: "comment received"}
	service := &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, instanceDAO: instanceDAO, userDAO: userdao.NewUserDAO(db, zap.NewNop()), logger: zap.NewNop()}
	commentService := NewInstanceCommentService(workdao.NewWorkorderInstanceCommentDAO(db, zap.NewNop()), instanceDAO, service, zap.NewNop())
	req := &model.CreateWorkorderInstanceCommentReq{InstanceID: instance.ID, OperatorID: 7, OperatorName: "fixture", Content: "test comment", Type: model.CommentTypeNormal, IsSystem: 2}
	ctx, cancel := context.WithCancel(context.Background())
	if err := commentService.CreateInstanceComment(ctx, req); err != nil {
		t.Fatal(err)
	}
	cancel()
	var events []model.WorkorderNotificationEvent
	if err := db.Find(&events).Error; err != nil || len(events) != 1 || events[0].EventType != model.EventTypeInstanceCommented || events[0].SenderID != 7 {
		t.Fatal("comment event not persisted with trusted sender")
	}
	worker := &NotificationEventWorker{db: db, service: service, logger: zap.NewNop()}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var messages int64
	if err := db.Model(&model.WorkorderInboxMessage{}).Where("user_id = ?", 7).Count(&messages).Error; err != nil || messages != 1 {
		t.Fatal("comment notification did not recover after request cancellation")
	}
	if err := db.Exec("CREATE TRIGGER reject_comment_event BEFORE INSERT ON cl_workorder_notification_event BEGIN SELECT RAISE(ABORT, 'fixture'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := commentService.CreateInstanceComment(context.Background(), req); err == nil {
		t.Fatal("event insert failure accepted")
	}
	var comments int64
	if err := db.Model(&model.WorkorderInstanceComment{}).Count(&comments).Error; err != nil || comments != 1 {
		t.Fatal("comment escaped failed outbox transaction")
	}
	if err := db.Exec("DROP TRIGGER reject_comment_event").Error; err != nil {
		t.Fatal(err)
	}
	internal := *req
	internal.IsSystem = 1
	internal.Type = model.CommentTypeSystem
	if err := commentService.CreateInstanceComment(context.Background(), &internal); err != nil {
		t.Fatal("internal system comment contract changed")
	}
	if err := db.Model(&model.WorkorderNotificationEvent{}).Count(&messages).Error; err != nil || messages != 1 {
		t.Fatal("internal system comment generated notification")
	}
}
