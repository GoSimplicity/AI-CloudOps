package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
)

type retryCapturingSender struct{ requests []*notification.SendRequest }

func (s *retryCapturingSender) SendNotification(_ context.Context, request *notification.SendRequest) (*notification.SendResponse, error) {
	s.requests = append(s.requests, request)
	return &notification.SendResponse{Success: true, MessageID: request.MessageID}, nil
}
func (*retryCapturingSender) GetAvailableChannels() []string { return []string{"email"} }

type retryConfigDAO struct {
	workdao.WorkorderNotificationDAO
	config *model.WorkorderNotification
}

func (d *retryConfigDAO) GetNotificationByID(context.Context, int) (*model.WorkorderNotification, error) {
	return d.config, nil
}
func (*retryConfigDAO) IncrementSentCount(context.Context, int) error                     { return nil }
func (*retryConfigDAO) AddSendLog(context.Context, *model.WorkorderNotificationLog) error { return nil }

func TestConfiguredSendRequestsCarryZeroAndExtraRetries(t *testing.T) {
	for _, retries := range []int{0, 2, 100} {
		for _, method := range []string{"automatic", "explicit test"} {
			t.Run(method+strconv.Itoa(retries), func(t *testing.T) {
				config := &model.WorkorderNotification{MaxRetries: retries, RetryInterval: 7, Status: 1, TriggerType: model.TriggerTypeImmediate, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "owned@example.test"}}, MessageTemplate: "owned"}
				sender := &retryCapturingSender{}
				service := &workorderNotificationService{notificationMgr: sender, dao: &retryConfigDAO{config: config}, logger: zap.NewNop()}
				var err error
				if method == "automatic" {
					err = service.processNotification(context.Background(), config, &model.WorkorderInstance{}, "instance_created", 0)
				} else {
					err = service.TestSendNotification(context.Background(), &model.TestSendWorkorderNotificationReq{Channel: "email", Recipient: "owned@example.test"})
				}
				if err != nil || len(sender.requests) != 1 {
					t.Fatalf("requests=%d err=%v", len(sender.requests), err)
				}
				request := sender.requests[0]
				if request.MaxRetries == nil || *request.MaxRetries != retries || request.RetryIntervalMinutes == nil || *request.RetryIntervalMinutes != 7 {
					t.Fatal("configuration retry override missing or changed units")
				}
			})
		}
	}
}
func TestDirectSendKeepsChannelRetryDefaults(t *testing.T) {
	sender := &retryCapturingSender{}
	service := &workorderNotificationService{notificationMgr: sender, logger: zap.NewNop()}
	if err := service.SendNotificationByChannels(context.Background(), []string{"email"}, "owned@example.test", "subject", "body"); err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 1 || sender.requests[0].MaxRetries != nil || sender.requests[0].RetryIntervalMinutes != nil {
		t.Fatal("manual send fabricated a configuration override")
	}
}
