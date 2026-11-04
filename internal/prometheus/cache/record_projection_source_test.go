package cache

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"go.uber.org/zap"
)

type recordSourceBarrier struct {
	reached chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func (b *recordSourceBarrier) wait(ctx context.Context) error {
	b.once.Do(func() { close(b.reached) })
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type recordPoolReadBarrier struct {
	scrapedao.ScrapePoolDAO
	mode    string
	barrier *recordSourceBarrier
}

func (p *recordPoolReadBarrier) GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	rows, total, err := p.ScrapePoolDAO.GetMonitorScrapePoolList(ctx, req)
	if err == nil && p.mode == "pool" && req.Page == 1 {
		err = p.barrier.wait(ctx)
	}
	return rows, total, err
}
func (p *recordPoolReadBarrier) GetMonitorScrapePoolLifecycleList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	rows, total, err := p.ScrapePoolDAO.GetMonitorScrapePoolLifecycleList(ctx, req)
	if err == nil && p.mode == "lifecycle" && req.Page == 1 {
		err = p.barrier.wait(ctx)
	}
	return rows, total, err
}
func (p *recordPoolReadBarrier) GetMonitorScrapePoolTombstone(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	row, err := p.ScrapePoolDAO.GetMonitorScrapePoolTombstone(ctx, id)
	if err == nil && p.mode == "tombstone" {
		err = p.barrier.wait(ctx)
	}
	return row, err
}

type recordRuleReadBarrier struct {
	alertdao.AlertManagerRecordDAO
	barrier *recordSourceBarrier
}

func (p *recordRuleReadBarrier) GetMonitorRecordRuleByPoolId(ctx context.Context, id int) ([]*model.MonitorRecordRule, int64, error) {
	rows, total, err := p.AlertManagerRecordDAO.GetMonitorRecordRuleByPoolId(ctx, id)
	if err == nil {
		err = p.barrier.wait(ctx)
	}
	return rows, total, err
}

func TestRecordProjectionOlderSourceCannotReplaceNewerCompletedGeneration(t *testing.T) {
	runScrapeRuleOlderSourceContract(t, model.ConfigTypeRecordRule)
}
func runScrapeRuleOlderSourceContract(t *testing.T, family int8) {
	if family != model.ConfigTypeAlertRule && family != model.ConfigTypeRecordRule {
		t.Fatal("unsupported source test family")
	}
	for _, mode := range []string{"pool", "rule", "lifecycle", "tombstone"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecordProjectionFixture(t, nil, nil, nil, nil)
			db := f.db
			var sourceModel any = &model.MonitorRecordRule{}
			var sourceSeed any = &model.MonitorRecordRule{Model: model.Model{ID: 31}, PoolID: 7, Name: "snapshot:up", Expr: "up", Enable: 1}
			if family == model.ConfigTypeAlertRule {
				sourceModel = &model.MonitorAlertRule{}
				sourceSeed = &model.MonitorAlertRule{Model: model.Model{ID: 31}, PoolID: 7, Name: "snapshot alert", Expr: "up", ForTime: "0s", Enable: 1}
			}
			if err := db.AutoMigrate(&model.MonitorScrapePool{}, sourceModel); err != nil {
				t.Fatal("real source tables unavailable")
			}
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "source-snapshot", SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.41"}}
			if family == model.ConfigTypeAlertRule {
				pool.SupportAlert = 1
			}
			deleted := mode == "lifecycle" || mode == "tombstone"
			if deleted {
				pool.DeletedAt = 17
				if family == model.ConfigTypeAlertRule {
					pool.SupportAlert = 2
				} else {
					pool.SupportRecord = 2
				}
			}
			if db.Create(pool).Error != nil || db.Create(sourceSeed).Error != nil {
				t.Fatal("source seed failed")
			}
			log := zap.NewNop()
			pools := scrapedao.NewScrapePoolDAO(db, log, nil)
			rules := alertdao.NewAlertManagerRecordDAO(db, log, nil)
			alerts := alertdao.NewAlertManagerRuleDAO(db, log, nil)
			store := &supportRedis{values: map[string]string{}, sets: map[string]map[string]bool{}}
			makeCache := func(p scrapedao.ScrapePoolDAO, r alertdao.AlertManagerRecordDAO, a alertdao.AlertManagerRuleDAO) scrapeRuleTestPublisher {
				if family == model.ConfigTypeAlertRule {
					journal, history, projection := AlertRuleProjectionDependenciesForTest(db, store)
					return NewAlertRuleConfigCache(log, p, a, f, NewBatchConfigManager(f, log), store, journal, history, projection)
				}
				c := &recordRuleConfigCache{logger: log, redis: store, scrapePoolDAO: p, alertRecordDAO: r}
				f.attach(c)
				return c
			}
			release := make(chan struct{})
			barrier := &recordSourceBarrier{reached: make(chan struct{}), release: release}
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			var aPools scrapedao.ScrapePoolDAO = pools
			var aRules alertdao.AlertManagerRecordDAO = rules
			var aAlerts alertdao.AlertManagerRuleDAO = alerts
			if mode == "rule" {
				if family == model.ConfigTypeAlertRule {
					aAlerts = &alertRuleReadBarrier{AlertManagerRuleDAO: alerts, barrier: barrier}
				} else {
					aRules = &recordRuleReadBarrier{AlertManagerRecordDAO: rules, barrier: barrier}
				}
			} else {
				aPools = &recordPoolReadBarrier{ScrapePoolDAO: pools, mode: mode, barrier: barrier}
			}
			a, b := makeCache(aPools, aRules, aAlerts), makeCache(pools, rules, alerts)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			result := make(chan error, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				switch mode {
				case "lifecycle":
					result <- a.ReconcileDeletedScrapePools(ctx)
				case "tombstone":
					result <- a.RetireDeletedScrapePool(ctx, 7)
				default:
					result <- a.GenerateMainConfig(ctx)
				}
			}()
			defer func() {
				unblock()
				select {
				case <-finished:
				case <-ctx.Done():
					t.Error("older-source publisher did not finish")
				}
			}()
			select {
			case <-barrier.reached:
			case <-ctx.Done():
				t.Fatal("older source read did not reach deterministic barrier")
			}
			// Mutate actual source tables only after A has obtained the old DAO result.
			if deleted {
				if db.Unscoped().Model(&model.MonitorScrapePool{}).Where("id=7").Update("deleted_at", uint64(33)).Error != nil {
					t.Fatal("source tombstone update failed")
				}
			} else if mode == "pool" {
				if db.Model(&model.MonitorScrapePool{}).Where("id=7").Update("prometheus_instances", model.StringList{"127.0.0.42"}).Error != nil {
					t.Fatal("source target update failed")
				}
			} else {
				if db.Model(sourceModel).Where("id=31").Update("expr", "sum(up)").Error != nil {
					t.Fatal("source rule update failed")
				}
			}
			var err error
			if deleted {
				err = b.RetireDeletedScrapePool(ctx, 7)
			} else {
				err = b.GenerateMainConfig(ctx)
			}
			if err != nil {
				t.Fatal("newer B source did not fully publish", err)
			}
			committed, err := f.journal.LoadLastCommittedProjection(ctx, family, 7)
			if err != nil || committed == nil || committed.Phase != model.ConfigPublicationProjected {
				t.Fatal("B has no durable completed generation")
			}
			projected, err := f.journal.LoadLastProjectedProjection(ctx, family, 7)
			if err != nil || projected == nil || projected.Reservation != committed.Reservation {
				t.Fatal("B last-good pointer missing")
			}
			beforeValues, beforeSets := copyAlertRedis(store.values), copySupportSets(store.sets)
			var beforeRows []model.MonitorConfig
			if db.Unscoped().Order("id ASC").Find(&beforeRows).Error != nil {
				t.Fatal("B metadata snapshot failed")
			}
			unblock()
			select {
			case <-result:
			case <-ctx.Done():
				t.Fatal("older A did not return after releasing source snapshot")
			}
			after, loadErr := f.journal.LoadLastCommittedProjection(ctx, family, 7)
			lastGood, goodErr := f.journal.LoadLastProjectedProjection(ctx, family, 7)
			var afterRows []model.MonitorConfig
			if db.Unscoped().Order("id ASC").Find(&afterRows).Error != nil {
				t.Fatal("metadata recheck failed")
			}
			if loadErr != nil || goodErr != nil || after == nil || lastGood == nil || after.Reservation != committed.Reservation || lastGood.Reservation != projected.Reservation || after.SourceHash != committed.SourceHash || !reflect.DeepEqual(beforeRows, afterRows) || !reflect.DeepEqual(beforeValues, store.values) || !reflect.DeepEqual(beforeSets, store.sets) {
				t.Fatal("older source reserved after and reverted B's completed SQL/Redis generation")
			}
		})
	}
}
