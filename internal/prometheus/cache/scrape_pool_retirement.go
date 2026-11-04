package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
)

// The caller holds its publisher's instance mutex throughout validation and
// publication. A source snapshot captured by an older run must settle first.
func retireDeletedScrapePool(ctx context.Context, dao scrapedao.ScrapePoolDAO, poolID int, publish func(context.Context, *model.MonitorScrapePool) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if dao == nil || poolID <= 0 {
		return errors.New("采集池删除授权不可用")
	}
	pool, err := dao.GetMonitorScrapePoolTombstone(ctx, poolID)
	if err != nil {
		return errors.New("无法确认采集池删除记录")
	}
	if pool == nil || pool.ID != poolID || pool.DeletedAt == 0 {
		return errors.New("缺少真实采集池删除授权")
	}
	return publish(ctx, cloneScrapePoolSnapshot(pool))
}

func reconcileDeletedScrapePools(ctx context.Context, dao scrapedao.ScrapePoolDAO, publish func(context.Context, *model.MonitorScrapePool) error) error {
	// Finish every page, including live rows, before the first visible write.
	pools, err := readScrapePoolLifecycleSnapshot(ctx, dao)
	if err != nil {
		return err
	}
	var failures []error
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if pool.DeletedAt == 0 {
			continue
		}
		if err := publish(ctx, pool); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

func (p *prometheusConfigCache) RetireDeletedScrapePool(ctx context.Context, poolID int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return retireDeletedScrapePool(ctx, p.scrapePoolDAO, poolID, p.publishPromPool)
}
func (p *prometheusConfigCache) ReconcileDeletedScrapePools(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return reconcileDeletedScrapePools(ctx, p.scrapePoolDAO, p.publishPromPool)
}
func (r *alertRuleConfigCache) RetireDeletedScrapePool(ctx context.Context, poolID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	versions, err := r.alertRuleSourceVersions(ctx)
	if err != nil {
		return err
	}
	return retireDeletedScrapePool(ctx, r.scrapePoolDAO, poolID, func(ctx context.Context, pool *model.MonitorScrapePool) error {
		return r.publishAlertRulePool(ctx, pool, versions[pool.ID])
	})
}
func (r *alertRuleConfigCache) ReconcileDeletedScrapePools(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	versions, err := r.alertRuleSourceVersions(ctx)
	if err != nil {
		return err
	}
	return reconcileDeletedScrapePools(ctx, r.scrapePoolDAO, func(ctx context.Context, pool *model.MonitorScrapePool) error {
		return r.publishAlertRulePool(ctx, pool, versions[pool.ID])
	})
}
func (r *recordRuleConfigCache) RetireDeletedScrapePool(ctx context.Context, poolID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	versions, err := r.recordSourceVersions(ctx)
	if err != nil {
		return err
	}
	return retireDeletedScrapePool(ctx, r.scrapePoolDAO, poolID, func(ctx context.Context, pool *model.MonitorScrapePool) error {
		return r.publishRecordPool(ctx, pool, versions[pool.ID])
	})
}
func (r *recordRuleConfigCache) ReconcileDeletedScrapePools(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	versions, err := r.recordSourceVersions(ctx)
	if err != nil {
		return err
	}
	return reconcileDeletedScrapePools(ctx, r.scrapePoolDAO, func(ctx context.Context, pool *model.MonitorScrapePool) error {
		return r.publishRecordPool(ctx, pool, versions[pool.ID])
	})
}
