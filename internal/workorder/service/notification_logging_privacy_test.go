package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type privateLogSender struct{ fail bool }

func (s privateLogSender) GetAvailableChannels() []string { return []string{"email"} }
func (s privateLogSender) SendNotification(context.Context, *notification.SendRequest) (*notification.SendResponse, error) {
	if s.fail {
		return nil, errors.New("private-provider-marker private-recipient@example.test")
	}
	return &notification.SendResponse{Success: true}, nil
}
func TestWorkorderNotificationLogsDoNotExposeRecipientOrProviderError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			config := &model.WorkorderNotification{Model: model.Model{ID: 3}, TriggerType: model.TriggerTypeImmediate, Status: 1, Channels: model.StringList{"email"}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: "email", Address: "private-recipient@example.test"}}, MessageTemplate: "private-message-marker"}
			svc := &workorderNotificationService{dao: &retryConfigDAO{config: config}, notificationMgr: privateLogSender{fail: fail}, logger: zap.New(core)}
			err := svc.processNotification(context.Background(), config, &model.WorkorderInstance{Model: model.Model{ID: 7}}, model.EventTypeInstanceCreated, 1)
			if (err != nil) != fail {
				t.Fatalf("unexpected send result: %v", err)
			}
			if fail && !strings.Contains(err.Error(), "private-recipient@example.test") {
				t.Fatal("send error contract unexpectedly changed")
			}
			var output strings.Builder
			for _, entry := range logs.All() {
				output.WriteString(entry.Message)
				data, _ := json.Marshal(entry.ContextMap())
				output.Write(data)
			}
			for _, marker := range []string{"private-recipient@example.test", "private-provider-marker", "private-message-marker"} {
				if strings.Contains(output.String(), marker) {
					t.Errorf("log exposed private data")
				}
			}
		})
	}
}
