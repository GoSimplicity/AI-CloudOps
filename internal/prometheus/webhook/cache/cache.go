package cache

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"go.uber.org/zap"
)

type WebhookCache interface {
	RenewMapOnDutyGroup(ctx context.Context)
	GetOnDutyGroupById(id int) *model.MonitorOnDutyGroup

	RenewMapRule(ctx context.Context)
	GetRuleById(id int) *model.MonitorAlertRule

	RenewMapSendGroup(ctx context.Context)
	GetSendGroupById(id int) *model.MonitorSendGroup

	RenewMapUser(ctx context.Context)
	GetUserById(id int) *model.User
}

type webhookCache struct {
	l   *zap.Logger
	dao dao.WebhookDao
}

func NewWebhookCache(l *zap.Logger, dao dao.WebhookDao) WebhookCache {
	return &webhookCache{
		l:   l,
		dao: dao,
	}
}

func (wc *webhookCache) RenewMapOnDutyGroup(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) GetOnDutyGroupById(id int) *model.MonitorOnDutyGroup {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) RenewMapRule(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) GetRuleById(id int) *model.MonitorAlertRule {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) RenewMapSendGroup(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) GetSendGroupById(id int) *model.MonitorSendGroup {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) RenewMapUser(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (wc *webhookCache) GetUserById(id int) *model.User {
	//TODO implement me
	panic("implement me")
}
