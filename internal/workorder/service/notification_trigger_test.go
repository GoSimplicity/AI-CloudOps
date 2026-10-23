package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type triggerNotificationDAO struct {
	workorderDao.WorkorderNotificationDAO
	creates, updates int
	trigger          string
}

func (d *triggerNotificationDAO) CreateNotification(context.Context, *model.CreateWorkorderNotificationReq) error {
	d.creates++
	return nil
}
func (d *triggerNotificationDAO) UpdateNotification(context.Context, *model.UpdateWorkorderNotificationReq) error {
	d.updates++
	return nil
}
func (d *triggerNotificationDAO) GetNotificationByID(context.Context, int) (*model.WorkorderNotification, error) {
	return &model.WorkorderNotification{TriggerType: d.trigger, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}}, nil
}

type triggerRecipientDAO struct {
	userDao.UserDAO
	calls int
}

func (d *triggerRecipientDAO) GetByID(context.Context, int) (*model.User, error) {
	d.calls++
	return nil, errors.New("unexpected recipient lookup")
}

func TestUnsupportedNotificationTriggersRejectWritesAndHistoricalAutomaticSends(t *testing.T) {
	for _, trigger := range []string{"delayed", "scheduled", "conditional", "unknown"} {
		t.Run(trigger, func(t *testing.T) {
			store := &triggerNotificationDAO{trigger: model.TriggerTypeImmediate}
			users := &triggerRecipientDAO{}
			s := &workorderNotificationService{dao: store, userDAO: users, logger: zap.NewNop()}
			if err := s.CreateNotification(context.Background(), &model.CreateWorkorderNotificationReq{TriggerType: trigger, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}}); err == nil || store.creates != 0 {
				t.Error("unsupported trigger reached create persistence")
			}
			if err := s.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: 1, TriggerType: trigger}); err == nil || store.updates != 0 {
				t.Error("unsupported trigger reached update persistence")
			}
			if err := s.processNotification(context.Background(), &model.WorkorderNotification{TriggerType: trigger, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"creator"}}, &model.WorkorderInstance{OperatorID: 7}, "instance_created", 0); err == nil || users.calls != 0 {
				t.Error("historical non-immediate trigger entered automatic recipient processing")
			}
		})
	}
}

func TestImmediateNotificationDoesNotAcceptIgnoredSchedulingParameters(t *testing.T) {
	now := time.Now()
	interval := 10
	for _, request := range []*model.CreateWorkorderNotificationReq{
		{TriggerType: model.TriggerTypeImmediate, ScheduledTime: &now},
		{TriggerType: model.TriggerTypeImmediate, RepeatInterval: &interval},
		{TriggerType: model.TriggerTypeImmediate, TriggerCondition: model.JSONMap{"status": "completed"}},
	} {
		store := &triggerNotificationDAO{trigger: model.TriggerTypeImmediate}
		s := &workorderNotificationService{dao: store, logger: zap.NewNop()}
		request.Channels = model.StringList{"email"}
		request.RecipientTypes = model.StringList{"creator"}
		if err := s.CreateNotification(context.Background(), request); err == nil || store.creates != 0 {
			t.Error("ignored scheduling parameters accepted as immediate notification")
		}
	}
}

func TestHistoricalUnsupportedNotificationCanBeDisabledButNotEnabled(t *testing.T) {
	store := &triggerNotificationDAO{trigger: model.TriggerTypeScheduled}
	s := &workorderNotificationService{dao: store, logger: zap.NewNop()}
	if err := s.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: 1, Status: 2}); err != nil {
		t.Fatal("must allow disabling legacy unsupported notification")
	}
	if err := s.UpdateNotification(context.Background(), &model.UpdateWorkorderNotificationReq{ID: 1, Status: 1}); err == nil || store.updates != 1 {
		t.Fatal("unsupported legacy trigger can be enabled")
	}
}
