package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestNotificationTriggerValidation(t *testing.T) {
	now := time.Now().UTC().Add(time.Hour)
	east := now.In(time.FixedZone("east", 3600))
	delay, interval, count, invalid := 5, 10, 3, 0
	for _, tc := range []struct {
		name, trigger          string
		scheduled              *time.Time
		delay, interval, count *int
		condition              model.JSONMap
		valid                  bool
	}{
		{"immediate", "immediate", nil, nil, nil, nil, nil, true},
		{"delay", "delayed", nil, &delay, nil, nil, nil, true},
		{"delay missing", "delayed", nil, nil, nil, nil, nil, false},
		{"delay zero", "delayed", nil, &invalid, nil, nil, nil, false},
		{"scheduled", "scheduled", &now, nil, nil, nil, nil, true},
		{"scheduled DB local time", "scheduled", &east, nil, nil, nil, nil, true},
		{"conditional", "conditional", nil, nil, nil, nil, model.JSONMap{"status": float64(2), "priority": float64(1)}, true},
		{"conditional unknown field", "conditional", nil, nil, nil, nil, model.JSONMap{"script": "run"}, false},
		{"conditional fractional", "conditional", nil, nil, nil, nil, model.JSONMap{"status": 1.5}, false},
		{"repeat", "immediate", nil, nil, &interval, &count, nil, true},
		{"repeat missing count", "immediate", nil, nil, &interval, nil, nil, false},
		{"unknown", "execute", nil, nil, nil, nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNotificationTrigger(tc.trigger, tc.scheduled, tc.delay, tc.interval, tc.count, tc.condition)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
	if validateInputScheduleTime(&east) == nil {
		t.Fatal("non-UTC client input was accepted")
	}
}

func scheduleFixture(t *testing.T, config *model.WorkorderNotification, sender notificationSender) (*NotificationEventWorker, *model.WorkorderNotificationEvent) {
	t.Helper()
	db := eventTestDB(t)
	instance := model.WorkorderInstance{Model: model.Model{ID: 12}, ProcessID: 1, Status: model.InstanceStatusPending, Priority: model.PriorityHigh}
	snapshot, err := json.Marshal(instance)
	if err != nil {
		t.Fatal(err)
	}
	event := &model.WorkorderNotificationEvent{InstanceID: 12, EventType: model.EventTypeInstanceCreated, Snapshot: snapshot, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
	if err := db.Create(event).Error; err != nil {
		t.Fatal(err)
	}
	worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: config}, notificationMgr: sender, logger: zap.NewNop()}, logger: zap.NewNop()}
	return worker, event
}

func scheduleConfig(trigger string) *model.WorkorderNotification {
	return &model.WorkorderNotification{Model: model.Model{ID: 8}, TriggerType: trigger, TriggerVersion: 1, Status: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "recipient@example.test"}}, MessageTemplate: "fixture"}
}

func TestDelayedSchedulePersistsAndDoesNotSendEarly(t *testing.T) {
	delay := 5
	config := scheduleConfig(model.TriggerTypeDelayed)
	config.DelayMinutes = &delay
	sender := &eventTestSender{attempts: map[string]int{}}
	worker, parent := scheduleFixture(t, config, sender)
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var parentStatus model.WorkorderNotificationEvent
	if err := worker.db.First(&parentStatus, parent.ID).Error; err != nil || parentStatus.Status != "planned" {
		t.Fatalf("parent schedule state=%s error=%v", parentStatus.Status, err)
	}
	var child model.WorkorderNotificationEvent
	if err := worker.db.Where("parent_event_id = ?", parent.ID).First(&child).Error; err != nil {
		t.Fatal(err)
	}
	if child.Status != "pending" || child.NotificationID == nil || *child.NotificationID != 8 || child.Occurrence == nil || *child.Occurrence != 1 {
		t.Fatalf("bad child: %+v", child)
	}
	if diff := child.NextAttemptAt.Sub(parent.CreatedAt); diff != 5*time.Minute {
		t.Fatalf("delay=%v", diff)
	}
	if err := worker.processOne(context.Background()); !errors.Is(err, errNoNotificationEvent) {
		t.Fatalf("delivered early: %v", err)
	}
	if len(sender.attempts) != 0 {
		t.Fatal("unexpected delivery")
	}
	if err := worker.db.Model(&child).Update("next_attempt_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.attempts["recipient@example.test"] != 1 {
		t.Fatal("missing delivery")
	}
}

func TestConditionAndPastScheduleArePersistedSkipped(t *testing.T) {
	for _, tc := range []string{"condition", "past"} {
		t.Run(tc, func(t *testing.T) {
			config := scheduleConfig(model.TriggerTypeConditional)
			config.TriggerCondition = model.JSONMap{"status": float64(model.InstanceStatusCompleted)}
			if tc == "past" {
				past := time.Now().UTC().Add(-time.Hour)
				config.TriggerType = model.TriggerTypeScheduled
				config.TriggerCondition = nil
				config.ScheduledTime = &past
			}
			sender := &eventTestSender{attempts: map[string]int{}}
			worker, parent := scheduleFixture(t, config, sender)
			if err := worker.processOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			var child model.WorkorderNotificationEvent
			if err := worker.db.Where("parent_event_id = ?", parent.ID).First(&child).Error; err != nil {
				t.Fatal(err)
			}
			if child.Status != "skipped" || len(sender.attempts) != 0 || len(child.Plan) != 0 || string(child.Snapshot) != "{}" {
				t.Fatalf("bad skip: %+v", child)
			}
		})
	}
}

type unknownScheduleSender struct{ calls int }

func (s *unknownScheduleSender) GetAvailableChannels() []string { return []string{"email"} }
func (s *unknownScheduleSender) SendNotification(context.Context, *notification.SendRequest) (*notification.SendResponse, error) {
	s.calls++
	return nil, io.EOF
}

func TestRepeatCreatesNextOnlyAfterConfirmedSuccess(t *testing.T) {
	interval, count := 2, 3
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "unknown"}[unknown], func(t *testing.T) {
			config := scheduleConfig(model.TriggerTypeImmediate)
			config.RepeatInterval, config.RepeatCount = &interval, &count
			sender := &eventTestSender{attempts: map[string]int{}}
			uncertain := &unknownScheduleSender{}
			var selected notificationSender = sender
			if unknown {
				selected = uncertain
			}
			worker, parent := scheduleFixture(t, config, selected)
			if err := worker.processOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := worker.processOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			var children []model.WorkorderNotificationEvent
			if err := worker.db.Where("parent_event_id = ?", parent.ID).Order("occurrence").Find(&children).Error; err != nil {
				t.Fatal(err)
			}
			if unknown {
				if len(children) != 1 || children[0].Status != "uncertain" || uncertain.calls != 1 {
					t.Fatalf("unknown children=%v calls=%d", children, uncertain.calls)
				}
			} else {
				if len(children) != 2 || children[0].Status != "success" || children[1].Status != "pending" || *children[1].Occurrence != 2 {
					t.Fatalf("success children=%v", children)
				}
				if children[1].NextAttemptAt.Before(time.Now().UTC().Add(time.Minute)) {
					t.Fatal("repeat interval lost")
				}
				for occurrence := 2; occurrence <= 3; occurrence++ {
					if err := worker.db.Model(&model.WorkorderNotificationEvent{}).Where("parent_event_id = ? AND occurrence = ?", parent.ID, occurrence).Update("next_attempt_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
						t.Fatal(err)
					}
					if err := worker.processOne(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
				var total int64
				if err := worker.db.Model(&model.WorkorderNotificationEvent{}).Where("parent_event_id = ?", parent.ID).Count(&total).Error; err != nil || total != 3 {
					t.Fatalf("repeat total=%d error=%v", total, err)
				}
				if sender.attempts["recipient@example.test"] != 3 {
					t.Fatal("each occurrence must send once")
				}
			}
		})
	}
}

func TestRepeatPersistenceFailureCanResumeWithoutResending(t *testing.T) {
	interval, count := 2, 2
	config := scheduleConfig(model.TriggerTypeImmediate)
	config.RepeatInterval, config.RepeatCount = &interval, &count
	sender := &eventTestSender{attempts: map[string]int{}}
	worker, parent := scheduleFixture(t, config, sender)
	if err := worker.db.AutoMigrate(&model.WorkorderNotificationResolution{}); err != nil {
		t.Fatal(err)
	}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var child model.WorkorderNotificationEvent
	if err := worker.db.Where("parent_event_id = ?", parent.ID).First(&child).Error; err != nil {
		t.Fatal(err)
	}
	if err := worker.db.Model(&child).Update("attempts", eventMaxAttempts-1).Error; err != nil {
		t.Fatal(err)
	}
	callback := "test:fail-next-occurrence"
	if err := worker.db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if event, ok := tx.Statement.Dest.(*model.WorkorderNotificationEvent); ok && event.Occurrence != nil && *event.Occurrence == 2 {
			tx.AddError(errors.New("injected persistence failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := worker.processOne(context.Background()); err == nil {
		t.Fatal("expected injected error")
	}
	if err := worker.db.First(&child, child.ID).Error; err != nil {
		t.Fatal(err)
	}
	if child.Status != "failed" || len(child.Plan) == 0 || sender.attempts["recipient@example.test"] != 1 {
		t.Fatalf("outcome=%s sends=%d", child.Status, sender.attempts["recipient@example.test"])
	}
	if err := worker.db.Callback().Create().Remove(callback); err != nil {
		t.Fatal(err)
	}
	instanceDAO := workdao.NewWorkorderInstanceDAO(worker.db, zap.NewNop())
	worker.service.instanceDAO = instanceDAO
	if err := worker.service.RetryNotificationEvent(context.Background(), child.ID, "verified schedule failure", 9); err != nil {
		t.Fatal(err)
	}
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.attempts["recipient@example.test"] != 1 {
		t.Fatal("confirmed delivery was resent")
	}
	var children []model.WorkorderNotificationEvent
	if err := worker.db.Where("parent_event_id = ?", parent.ID).Order("occurrence").Find(&children).Error; err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 || children[0].Status != "success" || children[1].Status != "pending" {
		t.Fatalf("children=%v", children)
	}
	var audit model.WorkorderNotificationResolution
	if err := worker.db.Where("event_id = ?", child.ID).First(&audit).Error; err != nil || audit.Action != "retry_schedule" {
		t.Fatalf("audit=%+v err=%v", audit, err)
	}
}

func TestScheduleInsertionFailureRollsBackParentPlan(t *testing.T) {
	delay := 5
	config := scheduleConfig(model.TriggerTypeDelayed)
	config.DelayMinutes = &delay
	sender := &eventTestSender{attempts: map[string]int{}}
	worker, parent := scheduleFixture(t, config, sender)
	callback := "test:reject-schedule"
	if err := worker.db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if event, ok := tx.Statement.Dest.(*model.WorkorderNotificationEvent); ok && event.ParentEventID != nil {
			tx.AddError(errors.New("injected child failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer worker.db.Callback().Create().Remove(callback)
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.db.First(parent, parent.ID).Error; err != nil {
		t.Fatal(err)
	}
	if parent.Status != "pending" || len(parent.Plan) != 0 {
		t.Fatalf("partial parent commit: %+v", parent)
	}
	var count int64
	if err := worker.db.Model(&model.WorkorderNotificationEvent{}).Where("parent_event_id = ?", parent.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("child was committed")
	}
	if len(sender.attempts) != 0 {
		t.Fatal("sent before transaction completed")
	}
}

func TestScheduleServiceAcceptsValidTriggersAndRejectsUnsafeUpdates(t *testing.T) {
	delay, interval, count := 3, 10, 2
	store := &triggerNotificationDAO{trigger: model.TriggerTypeImmediate}
	svc := &workorderNotificationService{dao: store, logger: zap.NewNop()}
	request := &model.CreateWorkorderNotificationReq{TriggerType: model.TriggerTypeDelayed, DelayMinutes: &delay, RepeatInterval: &interval, RepeatCount: &count, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}}
	if err := svc.CreateNotification(context.Background(), request); err != nil || store.creates != 1 {
		t.Fatalf("valid schedule rejected: %v", err)
	}
	if err := svc.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: 1, TriggerType: model.TriggerTypeDelayed, DelayMinutes: &delay, ClearTrigger: true}); err != nil || store.updates != 1 {
		t.Fatalf("valid update rejected: %v", err)
	}
	if err := svc.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: 1, TriggerType: model.TriggerTypeConditional, TriggerCondition: model.JSONMap{"script": "alert()"}, ClearTrigger: true}); err == nil || store.updates != 1 {
		t.Fatal("unsafe condition was accepted")
	}
}

func TestLegacyAdvancedTriggerRequiresExplicitRevalidation(t *testing.T) {
	delay := 5
	legacy := scheduleConfig(model.TriggerTypeDelayed)
	legacy.TriggerVersion = 0
	legacy.DelayMinutes = &delay
	if err := validateAutomaticNotification(legacy); err == nil {
		t.Fatal("legacy advanced trigger unexpectedly enabled")
	}
	sender := &eventTestSender{attempts: map[string]int{}}
	worker, parent := scheduleFixture(t, legacy, sender)
	if err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var skipped model.WorkorderNotificationEvent
	if err := worker.db.Where("parent_event_id = ?", parent.ID).First(&skipped).Error; err != nil || skipped.Status != "skipped" || skipped.LastError == "" || len(sender.attempts) != 0 {
		t.Fatalf("legacy did not fail closed: %+v error=%v", skipped, err)
	}
	request := &model.UpdateWorkorderNotificationReq{Status: model.NotificationStatusEnabled}
	if err := validateNotificationActivation(legacy, request); err != nil {
		t.Fatalf("valid explicit activation rejected: %v", err)
	}
	legacy.TriggerVersion = 1
	if err := validateAutomaticNotification(legacy); err != nil {
		t.Fatalf("validated trigger rejected: %v", err)
	}
}

type legacyUpdateStore struct {
	workdao.WorkorderNotificationDAO
	config *model.WorkorderNotification
	marked bool
}

func (s *legacyUpdateStore) GetNotificationByID(context.Context, int) (*model.WorkorderNotification, error) {
	return s.config, nil
}
func (s *legacyUpdateStore) UpdateNotification(_ context.Context, req *model.UpdateWorkorderNotificationReq) error {
	s.marked = req.MarkTriggerVersion
	return nil
}
func TestLegacyMetadataAndRepeatedEnableDoNotActivateTrigger(t *testing.T) {
	delay := 5
	existing := scheduleConfig(model.TriggerTypeDelayed)
	existing.DelayMinutes = &delay
	existing.TriggerVersion = 0
	existing.Status = model.NotificationStatusEnabled
	store := &legacyUpdateStore{config: existing}
	svc := &workorderNotificationService{dao: store, logger: zap.NewNop()}
	name := "renamed"
	if err := svc.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: existing.ID, Name: name}); err != nil || store.marked {
		t.Fatalf("metadata changed activation: %v", err)
	}
	if err := svc.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: existing.ID, Status: model.NotificationStatusEnabled}); err != nil || store.marked {
		t.Fatalf("unchanged enabled state activated legacy: %v", err)
	}
	existing.Status = model.NotificationStatusDisabled
	if err := svc.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: existing.ID, Status: model.NotificationStatusEnabled}); err != nil || !store.marked {
		t.Fatalf("explicit enable failed: %v", err)
	}
}
