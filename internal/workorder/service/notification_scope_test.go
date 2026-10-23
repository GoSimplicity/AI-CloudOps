package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
)

type scopeUsers struct {
	userdao.UserDAO
	calls atomic.Int32
}

func (d *scopeUsers) GetByID(context.Context, int) (*model.User, error) {
	d.calls.Add(1)
	return &model.User{Email: "owned@example.test", Enable: 1}, nil
}

type scopeSender struct{ calls atomic.Int32 }

func (d *scopeSender) GetAvailableChannels() []string { return []string{"email"} }
func (d *scopeSender) SendNotification(context.Context, *notification.SendRequest) (*notification.SendResponse, error) {
	d.calls.Add(1)
	return &notification.SendResponse{Success: true}, nil
}

type scopeLogs struct {
	workdao.WorkorderNotificationDAO
}

func (scopeLogs) AddSendLog(context.Context, *model.WorkorderNotificationLog) error { return nil }

func TestNotificationScopeGateBeforeRecipientsAndDelivery(t *testing.T) {
	process, category, template, other, invalid := 11, 22, 33, 99, 0
	future := time.Now().Add(time.Hour)
	repeat := 60
	for _, tc := range []struct {
		name                                  string
		process, category, template           *int
		unknown, condition, scheduled, repeat bool
		want                                  int32
		fail                                  bool
	}{
		{name: "global unknown remains global", unknown: true, want: 1},
		{name: "all scopes match", process: &process, category: &category, template: &template, want: 1},
		{name: "different process", process: &other},
		{name: "different category", category: &other},
		{name: "different template", template: &other},
		{name: "unknown category", category: &category, unknown: true},
		{name: "unknown template", template: &template, unknown: true},
		{name: "all scopes use AND", process: &process, category: &other, template: &template},
		{name: "invalid concrete zero", category: &invalid, unknown: true},
		{name: "legacy condition", condition: true, fail: true},
		{name: "legacy future schedule", scheduled: true, fail: true},
		{name: "legacy repeat", repeat: true, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users, sender := &scopeUsers{}, &scopeSender{}
			s := &workorderNotificationService{userDAO: users, notificationMgr: sender, dao: scopeLogs{}, logger: zap.NewNop()}
			instance := &model.WorkorderInstance{Model: model.Model{ID: 7}, ProcessID: process, CategoryID: &category, TemplateID: &template, OperatorID: 1}
			if tc.unknown {
				instance.CategoryID = nil
				instance.TemplateID = nil
			}
			config := &model.WorkorderNotification{ProcessID: tc.process, CategoryID: tc.category, TemplateID: tc.template, TriggerType: model.TriggerTypeImmediate, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}, MessageTemplate: "owned fixture"}
			if tc.condition {
				config.TriggerCondition = model.JSONMap{"status": "completed"}
			}
			if tc.scheduled {
				config.ScheduledTime = &future
			}
			if tc.repeat {
				config.RepeatInterval = &repeat
			}
			err := s.processNotification(context.Background(), config, instance, "instance_created", 0)
			if (err != nil) != tc.fail || users.calls.Load() != tc.want || sender.calls.Load() != tc.want {
				t.Fatalf("guard mismatch: error=%v lookups=%d sends=%d", err, users.calls.Load(), sender.calls.Load())
			}
		})
	}
}

func TestNotificationActivationMergesHistoricalAdvancedParameters(t *testing.T) {
	future := time.Now().Add(time.Hour)
	repeat := 60
	for _, tc := range []struct {
		name     string
		existing model.WorkorderNotification
		request  model.UpdateWorkorderNotificationReq
		fail     bool
	}{
		{"metadata update allowed", model.WorkorderNotification{TriggerType: "scheduled", ScheduledTime: &future}, model.UpdateWorkorderNotificationReq{}, false},
		{"disable allowed", model.WorkorderNotification{TriggerType: "scheduled", ScheduledTime: &future}, model.UpdateWorkorderNotificationReq{Status: 2}, false},
		{"legacy scheduled cannot enable", model.WorkorderNotification{TriggerType: "immediate", ScheduledTime: &future}, model.UpdateWorkorderNotificationReq{Status: 1}, true},
		{"legacy repeat cannot enable", model.WorkorderNotification{TriggerType: "immediate", RepeatInterval: &repeat}, model.UpdateWorkorderNotificationReq{Status: 1}, true},
		{"legacy condition cannot enable", model.WorkorderNotification{TriggerType: "immediate", TriggerCondition: model.JSONMap{"status": "completed"}}, model.UpdateWorkorderNotificationReq{Status: 1}, true},
		{"explicit condition clear", model.WorkorderNotification{TriggerType: "immediate", TriggerCondition: model.JSONMap{"status": "completed"}}, model.UpdateWorkorderNotificationReq{Status: 1, TriggerCondition: model.JSONMap{}}, false},
		{"explicit immediate conversion", model.WorkorderNotification{TriggerType: "scheduled"}, model.UpdateWorkorderNotificationReq{Status: 1, TriggerType: "immediate"}, false},
		{"conversion retains old schedule and blocks", model.WorkorderNotification{TriggerType: "scheduled", ScheduledTime: &future}, model.UpdateWorkorderNotificationReq{Status: 1, TriggerType: "immediate"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateNotificationActivation(&tc.existing, &tc.request); (err != nil) != tc.fail {
				t.Fatalf("activation error=%v", err)
			}
		})
	}
}
