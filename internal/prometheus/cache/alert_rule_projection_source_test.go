package cache

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
)

type alertRuleReadBarrier struct {
	alertdao.AlertManagerRuleDAO
	barrier *recordSourceBarrier
}

func (p *alertRuleReadBarrier) GetMonitorAlertRuleByPoolID(ctx context.Context, id int) ([]*model.MonitorAlertRule, int64, error) {
	rows, total, err := p.AlertManagerRuleDAO.GetMonitorAlertRuleByPoolID(ctx, id)
	if err == nil {
		err = p.barrier.wait(ctx)
	}
	return rows, total, err
}
func TestAlertRuleProjectionOlderSourceCannotReplaceNewerCompletedGeneration(t *testing.T) {
	runScrapeRuleOlderSourceContract(t, model.ConfigTypeAlertRule)
}
