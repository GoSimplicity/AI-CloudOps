package cache

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
)

type promJobReadBarrier struct {
	scrapedao.ScrapeJobDAO
	barrier *recordSourceBarrier
}

func (p *promJobReadBarrier) GetMonitorScrapeJobsByPoolId(ctx context.Context, id int) ([]*model.MonitorScrapeJob, error) {
	rows, err := p.ScrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, id)
	if err == nil {
		err = p.barrier.wait(ctx)
	}
	return rows, err
}
func TestPromMainProjectionOlderSourceCannotReplaceNewerCompletedGeneration(t *testing.T) {
	runScrapeRuleOlderSourceContract(t, model.ConfigTypePrometheus)
}
