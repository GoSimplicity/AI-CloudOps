package content

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/prometheus/alertmanager/template"
	"go.uber.org/zap"
)

type WebhookContent interface {
	GenerateFeishuCardContentOneAlert(alert template.Alert, event *model.MonitorAlertEvent, rule *model.MonitorAlertRule, sendGroup *model.MonitorSendGroup) error
	SentFeishuGroup(msg string, rebotToken string)
}

type webhookContent struct {
	l *zap.Logger
}

func NewWebhookContent(l *zap.Logger) WebhookContent {
	return &webhookContent{
		l: l,
	}
}

func (w webhookContent) GenerateFeishuCardContentOneAlert(alert template.Alert, event *model.MonitorAlertEvent, rule *model.MonitorAlertRule, sendGroup *model.MonitorSendGroup) error {
	//TODO implement me
	panic("implement me")
}

func (w webhookContent) SentFeishuGroup(msg string, rebotToken string) {
	//TODO implement me
	panic("implement me")
}
