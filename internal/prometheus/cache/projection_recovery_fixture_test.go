package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"go.uber.org/zap"
)

type recoverySourceFixture struct {
	base   *recordProjectionFixture
	store  *supportRedis
	family int8
}

func newRecoverySourceFixture(t *testing.T, family int8) *recoverySourceFixture {
	t.Helper()
	f := &recoverySourceFixture{base: newRecordProjectionFixture(t, nil, nil, nil, nil), store: &supportRedis{values: map[string]string{}, sets: map[string]map[string]bool{}}, family: family}
	if err := f.base.db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}, &model.MonitorAlertRule{}, &model.MonitorRecordRule{}); err != nil {
		t.Fatal(err)
	}
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "independent-recovery", SupportAlert: 1, SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.73"}, ScrapeInterval: 30, ScrapeTimeout: 10}
	if err := f.base.db.Create(pool).Error; err != nil {
		t.Fatal(err)
	}
	var source any
	switch family {
	case 1:
		source = &model.MonitorScrapeJob{Model: model.Model{ID: 31}, PoolID: 7, Name: "recovery_job", Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, Scheme: "http", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, IpAddress: "127.0.0.1:9100"}
	case 3:
		source = &model.MonitorAlertRule{Model: model.Model{ID: 31}, PoolID: 7, Name: "recovery alert", Expr: "up", ForTime: "0s", Enable: 1}
	case 4:
		source = &model.MonitorRecordRule{Model: model.Model{ID: 31}, PoolID: 7, Name: "recovery:up", Expr: "up", Enable: 1}
	default:
		t.Fatal("unsupported recovery fixture family")
	}
	if err := f.base.db.Create(source).Error; err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *recoverySourceFixture) publisher(t *testing.T) scrapeRuleTestPublisher {
	t.Helper()
	log := zap.NewNop()
	pools := scrapedao.NewScrapePoolDAO(f.base.db, log, nil)
	switch f.family {
	case 1:
		journal, history, projection := PromMainProjectionDependenciesForTest(f.base.db, f.store)
		p, err := NewPrometheusConfigCache(&config.Config{}, log, pools, scrapedao.NewScrapeJobDAO(f.base.db, log), f.base, nil, f.store, journal, history, projection)
		if err != nil {
			t.Fatal(err)
		}
		return p
	case 3:
		journal, history, projection := AlertRuleProjectionDependenciesForTest(f.base.db, f.store)
		return NewAlertRuleConfigCache(log, pools, alertdao.NewAlertManagerRuleDAO(f.base.db, log, nil), f.base, nil, f.store, journal, history, projection)
	case 4:
		p := &recordRuleConfigCache{logger: log, redis: f.store, scrapePoolDAO: pools, alertRecordDAO: alertdao.NewAlertManagerRecordDAO(f.base.db, log, nil)}
		f.base.attach(p)
		return p
	}
	t.Fatal("unsupported recovery publisher")
	return nil
}
func (f *recoverySourceFixture) last(t *testing.T) configdao.ProjectionReceipt {
	t.Helper()
	receipt, err := configdao.NewProjectionPlanJournalDAO(zap.NewNop(), f.base.db).LoadLastCommittedProjection(context.Background(), f.family, 7)
	if err != nil || receipt == nil {
		t.Fatal("missing actual durable recovery receipt", err)
	}
	return *receipt
}
func (f *recoverySourceFixture) changeSource(t *testing.T, version int) {
	t.Helper()
	var source any
	field, value := "expr", fmt.Sprintf("up == %d", version)
	switch f.family {
	case 1:
		source = &model.MonitorScrapeJob{}
		field = "metrics_path"
		value = fmt.Sprintf("/metrics_%d", version)
	case 3:
		source = &model.MonitorAlertRule{}
	case 4:
		source = &model.MonitorRecordRule{}
	}
	if err := f.base.db.Model(source).Where("id=31").Update(field, value).Error; err != nil {
		t.Fatal(err)
	}
}
func (f *recoverySourceFixture) recoverer(t *testing.T) PoolProjectionRecoverer {
	t.Helper()
	result, ok := f.publisher(t).(PoolProjectionRecoverer)
	if !ok {
		t.Fatal("real publisher missing durable recovery capability")
	}
	return result
}

type recoveryRecordJournalHook struct {
	configdao.RecordProjectionJournalDAO
	configdao.ProjectionRecoveryJournalDAO
	beforeReserve func()
	commit        func(context.Context, configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error)
}

func (j *recoveryRecordJournalHook) ReserveRecordPoolOperation(ctx context.Context, req configdao.PoolOperationRequest, expected uint64) (configdao.PoolReservation, error) {
	if j.beforeReserve != nil {
		hook := j.beforeReserve
		j.beforeReserve = nil
		hook()
	}
	return j.RecordProjectionJournalDAO.ReserveRecordPoolOperation(ctx, req, expected)
}
func (j *recoveryRecordJournalHook) CommitRecordProjectionPlan(ctx context.Context, req configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	if j.commit != nil {
		return j.commit(ctx, req)
	}
	return j.RecordProjectionJournalDAO.CommitRecordProjectionPlan(ctx, req)
}
