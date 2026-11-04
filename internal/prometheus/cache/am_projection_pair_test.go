package cache

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

type amPairTestFixture struct {
	sql     *recordProjectionFixture
	store   *alertContractRedis
	pool    *model.MonitorAlertManagerPool
	group   *model.MonitorSendGroup
	pools   alertdao.AlertManagerPoolDAO
	groups  alertdao.AlertManagerSendDAO
	journal configdao.AMProjectionJournalDAO
}

func newAMPairTestFixture(t *testing.T) *amPairTestFixture {
	t.Helper()
	base := newRecordProjectionFixture(t, nil, nil, nil, nil)
	if err := base.db.AutoMigrate(&model.MonitorAlertManagerPool{}, &model.MonitorSendGroup{}); err != nil {
		t.Fatal(err)
	}
	log := zap.NewNop()
	f := &amPairTestFixture{sql: base, store: &alertContractRedis{values: map[string]string{}, members: map[string]map[string]bool{}}, pools: alertdao.NewAlertManagerPoolDAO(base.db, log), groups: alertdao.NewAlertManagerSendDAO(base.db, log), journal: configdao.NewAMProjectionJournalDAO(log, base.db)}
	f.pool = &model.MonitorAlertManagerPool{Model: model.Model{ID: 7}, Name: "pair-fixture", AlertManagerInstances: model.StringList{"127.0.0.47"}, ResolveTimeout: "5m", GroupWait: "30s", GroupInterval: "5m", RepeatInterval: "1h", GroupBy: model.StringList{"alertname"}, Receiver: "pair-group"}
	f.group = &model.MonitorSendGroup{Model: model.Model{ID: 3}, PoolID: 7, Name: "pair-group", NameZh: "pair", Enable: 1, RepeatInterval: "1h", SendResolved: 1}
	if err := f.pools.CreateMonitorAlertManagerPool(context.Background(), f.pool); err != nil {
		t.Fatal(err)
	}
	if err := f.groups.CreateMonitorSendGroup(context.Background(), f.group); err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *amPairTestFixture) cache(t *testing.T, address string, pools alertdao.AlertManagerPoolDAO, groups alertdao.AlertManagerSendDAO) *alertManagerConfigCache {
	t.Helper()
	if pools == nil {
		pools = f.pools
	}
	if groups == nil {
		groups = f.groups
	}
	journal, history, projection := AMProjectionDependenciesForTest(f.sql.db, f.store)
	c, err := NewAlertManagerConfigCache(&config.Config{Server: config.ServerConfig{Port: "8891"}, Prometheus: config.PrometheusConfig{AlertWebhookAddr: address, AlertWebhookFileDir: "/typed/pair"}}, zap.NewNop(), pools, groups, f.sql.MonitorConfigDAO, nil, f.store, journal, history, projection)
	if err != nil {
		t.Fatal(err)
	}
	return c.(*alertManagerConfigCache)
}

type amPoolReadBarrier struct {
	alertdao.AlertManagerPoolDAO
	which   string
	barrier *recordSourceBarrier
}

func (p *amPoolReadBarrier) GetMonitorAlertManagerPoolList(ctx context.Context, req *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	rows, total, err := p.AlertManagerPoolDAO.GetMonitorAlertManagerPoolList(ctx, req)
	if err == nil && p.which == "full" && req.Page == 1 {
		err = p.barrier.wait(ctx)
	}
	return rows, total, err
}
func (p *amPoolReadBarrier) GetAlertPoolByID(ctx context.Context, id int) (*model.MonitorAlertManagerPool, error) {
	row, err := p.AlertManagerPoolDAO.GetAlertPoolByID(ctx, id)
	if err == nil && p.which == "subset" {
		err = p.barrier.wait(ctx)
	}
	return row, err
}

type amGroupReadBarrier struct {
	alertdao.AlertManagerSendDAO
	barrier *recordSourceBarrier
}

func (p *amGroupReadBarrier) GetMonitorSendGroupByPoolID(ctx context.Context, id int) ([]*model.MonitorSendGroup, int64, error) {
	rows, total, err := p.AlertManagerSendDAO.GetMonitorSendGroupByPoolID(ctx, id)
	if err == nil {
		err = p.barrier.wait(ctx)
	}
	return rows, total, err
}

type amPublishedState struct {
	Full, Subset *configdao.ProjectionReceipt
	Rows         []model.MonitorConfig
	Values       map[string]string
	Sets         map[string]map[string]bool
}

func (f *amPairTestFixture) state(t *testing.T) amPublishedState {
	t.Helper()
	s := amPublishedState{Values: copyAlertRedis(f.store.values), Sets: copySupportSets(f.store.members)}
	var err error
	s.Full, err = f.journal.LoadLastCommittedProjection(context.Background(), 2, 7)
	if err != nil {
		t.Fatal(err)
	}
	s.Subset, err = f.journal.LoadLastCommittedProjection(context.Background(), 5, 7)
	if err != nil {
		t.Fatal(err)
	}
	if f.sql.db.Unscoped().Order("id").Find(&s.Rows).Error != nil {
		t.Fatal("AM metadata snapshot failed")
	}
	return s
}
func (f *amPairTestFixture) assertState(t *testing.T, want amPublishedState) {
	t.Helper()
	if !reflect.DeepEqual(f.state(t), want) {
		t.Fatal("older AM pair source changed committed metadata/control/Redis facts")
	}
}

func TestAMProjectionSourcePairRejectsBothDirectionsOfOlderSnapshots(t *testing.T) {
	for _, direction := range []string{"full", "subset"} {
		for _, point := range []string{"pool", "groups"} {
			t.Run(direction+"/"+point, func(t *testing.T) {
				f := newAMPairTestFixture(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				baseline := f.cache(t, "http://127.0.0.1:8891", nil, nil)
				if err := baseline.GenerateMainConfig(ctx); err != nil {
					t.Fatal("baseline full AM publication failed", err)
				}
				release := make(chan struct{})
				barrier := &recordSourceBarrier{reached: make(chan struct{}), release: release}
				var once sync.Once
				unblock := func() { once.Do(func() { close(release) }) }
				pools, groups := f.pools, f.groups
				if point == "pool" {
					pools = &amPoolReadBarrier{AlertManagerPoolDAO: pools, which: direction, barrier: barrier}
				} else {
					groups = &amGroupReadBarrier{AlertManagerSendDAO: groups, barrier: barrier}
				}
				older := f.cache(t, "http://127.0.0.1:8891", pools, groups)
				olderLocator := f.locator(t)
				finished := make(chan struct{})
				result := make(chan bool, 1)
				go func() {
					defer close(finished)
					if direction == "full" {
						result <- older.GenerateMainConfig(ctx) == nil
					} else {
						routes, receivers := older.GenerateRouteConfigForPool(ctx, olderLocator)
						result <- routes != nil && receivers != nil
					}
				}()
				defer func() {
					unblock()
					select {
					case <-finished:
					case <-ctx.Done():
						t.Error("older AM publication did not finish")
					}
				}()
				select {
				case <-barrier.reached:
				case <-ctx.Done():
					t.Fatal("actual AM source DAO did not reach barrier")
				}
				if point == "pool" {
					if f.sql.db.Model(&model.MonitorAlertManagerPool{}).Where("id=7").Update("name", "pair-after").Error != nil {
						t.Fatal("actual pool source update failed")
					}
				}
				if f.sql.db.Model(&model.MonitorSendGroup{}).Where("id=3").Update("repeat_interval", "2h").Error != nil {
					t.Fatal("actual group source update failed")
				}
				newer := f.cache(t, "http://127.0.0.1:8892", nil, nil)
				if direction == "full" {
					routes, receivers := newer.GenerateRouteConfigForPool(ctx, f.locator(t))
					if len(routes) == 0 || len(receivers) == 0 {
						t.Fatal("newer subset failed")
					}
					receipt, err := f.journal.LoadLastProjectedProjection(ctx, 5, 7)
					if err != nil || receipt == nil {
						t.Fatal("newer subset lacks actual completed SQL receipt")
					}
				} else {
					if err := newer.GenerateMainConfig(ctx); err != nil {
						t.Fatal("newer full publication failed", err)
					}
				}
				before := f.state(t)
				unblock()
				select {
				case <-result:
				case <-ctx.Done():
					t.Fatal("older AM source did not return")
				}
				f.assertState(t, before)
			})
		}
	}
}

func TestAMProjectionSameSourceAlternationIsTrustedNoop(t *testing.T) {
	f := newAMPairTestFixture(t)
	ctx := context.Background()
	c := f.cache(t, "http://127.0.0.1:8891", nil, nil)
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	routes, receivers := c.GenerateRouteConfigForPool(ctx, f.locator(t))
	if len(routes) == 0 || len(receivers) == 0 {
		t.Fatal("initial subset failed")
	}
	before := f.state(t)
	var controls []model.MonitorConfigPoolControl
	if f.sql.db.Order("family,pool_id").Find(&controls).Error != nil {
		t.Fatal("control snapshot failed")
	}
	for i := 0; i < 3; i++ {
		next := f.cache(t, "http://127.0.0.1:8891", nil, nil)
		if err := next.GenerateMainConfig(ctx); err != nil {
			t.Fatal(err)
		}
		r, s := next.GenerateRouteConfigForPool(ctx, f.locator(t))
		if len(r) == 0 || len(s) == 0 {
			t.Fatal("same-source subset was not available")
		}
	}
	f.assertState(t, before)
	var after []model.MonitorConfigPoolControl
	if f.sql.db.Order("family,pool_id").Find(&after).Error != nil || !reflect.DeepEqual(controls, after) {
		t.Fatal("same-source full/subset alternation ping-ponged generations")
	}
}

func TestAMProjectionFailedReservedPeerDoesNotStarveLastGood(t *testing.T) {
	f := newAMPairTestFixture(t)
	ctx := context.Background()
	c := f.cache(t, "http://127.0.0.1:8891", nil, nil)
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	routes, _ := c.GenerateRouteConfigForPool(ctx, f.locator(t))
	if len(routes) == 0 {
		t.Fatal("subset baseline failed")
	}
	versions, err := f.journal.ReadAMSourceVersions(ctx)
	if err != nil || len(versions) != 1 {
		t.Fatal("pair version missing")
	}
	if _, err := f.journal.ReserveAMPoolOperation(ctx, configdao.PoolOperationRequest{Family: 5, PoolID: 7, Nonce: [16]byte{0xf3}}, versions[0]); err != nil {
		t.Fatal("actual failed-attempt reservation fixture failed", err)
	}
	before := f.state(t)
	if err := f.cache(t, "http://127.0.0.1:8891", nil, nil).GenerateMainConfig(ctx); err != nil {
		t.Fatal("uncommitted peer reservation starved full last-good", err)
	}
	routes, _ = f.cache(t, "http://127.0.0.1:8891", nil, nil).GenerateRouteConfigForPool(ctx, f.locator(t))
	if len(routes) == 0 {
		t.Fatal("uncommitted reservation starved subset last-good")
	}
	f.assertState(t, before)
}

func (f *amPairTestFixture) locator(t *testing.T) *model.MonitorAlertManagerPool {
	t.Helper()
	row, err := f.pools.GetAlertPoolByID(context.Background(), 7)
	if err != nil || row == nil {
		t.Fatal("fresh saved AM locator missing", err)
	}
	return cloneAlertPoolLifecycle(row)
}
