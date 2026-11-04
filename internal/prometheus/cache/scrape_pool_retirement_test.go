package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type deletedPoolPublisher interface {
	GenerateMainConfig(context.Context) error
	RetireDeletedScrapePool(context.Context, int) error
	ReconcileDeletedScrapePools(context.Context) error
}
type retirementInventory struct {
	scrapedao.ScrapePoolDAO
	pool             *model.MonitorScrapePool
	rows             []*model.MonitorScrapePool
	readFailure      bool
	tombstoneFailure bool
	page             func(int) ([]*model.MonitorScrapePool, int64, error)
}

func (s *retirementInventory) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if s.readFailure {
		return nil, 0, errors.New("fixture read failure")
	}
	if s.pool == nil || s.pool.DeletedAt > 0 {
		return nil, 0, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}
func (s *retirementInventory) GetMonitorScrapePoolLifecycleList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if s.readFailure {
		return nil, 0, errors.New("fixture lifecycle failure")
	}
	if s.page != nil {
		return s.page(req.Page)
	}
	if s.rows != nil {
		return s.rows, int64(len(s.rows)), nil
	}
	if s.pool == nil {
		return nil, 0, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}
func (s *retirementInventory) GetMonitorScrapePoolTombstone(_ context.Context, id int) (*model.MonitorScrapePool, error) {
	if s.tombstoneFailure || s.pool == nil || s.pool.ID != id || s.pool.DeletedAt == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return cloneScrapePoolSnapshot(s.pool), nil
}

type retirementFixture struct {
	publisher     deletedPoolPublisher
	inventory     *retirementInventory
	pool          *model.MonitorScrapePool
	values        map[string]string
	members       map[string]bool
	rows          map[int]*model.MonitorConfig
	kind          int8
	ack           string
	artifact      func(string) string
	failure       func(string)
	sourceFailure func()
	sourceCalls   func() int
	writes        func() int
	onRead        func(func())
	rebuild       func()
}

func newRetirementFixture(t *testing.T, kind string) *retirementFixture {
	t.Helper()
	h := &retirementFixture{}
	switch kind {
	case "prom":
		c, source, store, db, pool := newPromReconcile(t)
		h.inventory = &retirementInventory{pool: pool}
		c.scrapePoolDAO = h.inventory
		h.publisher = c
		h.pool = pool
		h.values = store.values
		h.members = store.members
		h.rows = db.rows
		h.kind = 1
		h.ack = promReconcileACK(7)
		h.artifact = buildRedisKeyPrometheusMain
		h.failure = func(stage string) { store.failure = stage; db.failure = stage }
		h.sourceFailure = func() { source.failure = true }
		h.sourceCalls = func() int { return source.calls }
		h.writes = func() int { return store.writes + db.writes }
		h.onRead = func(fn func()) { store.onRead = fn }
		h.rebuild = func() {
			c = &prometheusConfigCache{logger: c.logger, redis: store, scrapePoolDAO: h.inventory, scrapeJobDAO: source, configDAO: c.configDAO, batchManager: c.batchManager, journal: c.journal, history: c.history, projection: c.projection}
			h.publisher = c
		}
	case "alert":
		c, source, store, db, pool := newAlertRulePublication(t)
		h.inventory = &retirementInventory{pool: pool}
		c.scrapePoolDAO = h.inventory
		h.publisher = c
		h.pool = pool
		h.values = store.values
		h.members = store.members
		h.rows = db.rows
		h.kind = 3
		h.ack = alertRuleACK(7)
		h.artifact = buildRedisKeyAlertRule
		h.failure = func(stage string) { store.failure = stage; db.failure = stage }
		h.sourceFailure = func() { source.failure = true }
		h.sourceCalls = func() int { return source.calls }
		h.writes = func() int { return store.writes + db.writes }
		h.onRead = func(fn func()) { store.onRead = fn }
		h.rebuild = func() {
			c = &alertRuleConfigCache{logger: c.logger, redis: store, scrapePoolDAO: h.inventory, alertRuleDAO: source, configDAO: c.configDAO, batchManager: c.batchManager, journal: c.journal, history: c.history, projection: c.projection}
			h.publisher = c
		}
	case "record":
		c, source, store, db, pool := newRecordReconcile(t)
		h.inventory = &retirementInventory{pool: pool}
		c.scrapePoolDAO = h.inventory
		h.publisher = c
		h.pool = pool
		h.values = store.values
		h.members = store.members
		h.rows = db.rows
		h.kind = 4
		h.ack = recordReconcileACK(7)
		h.artifact = buildRedisKeyRecordRule
		h.failure = func(stage string) { store.failure = stage; db.failure = stage }
		h.sourceFailure = func() { source.failure = true }
		h.sourceCalls = func() int { return source.calls }
		h.writes = func() int { return store.writes + db.writes }
		h.onRead = func(fn func()) { store.onRead = fn }
		h.rebuild = func() {
			c = &recordRuleConfigCache{logger: c.logger, redis: store, scrapePoolDAO: h.inventory, alertRecordDAO: source, configDAO: c.configDAO, batchManager: c.batchManager, journal: c.journal, history: c.history, projection: c.projection}
			h.publisher = c
		}
	default:
		t.Fatal("unknown fixture")
	}
	return h
}
func (h *retirementFixture) markDeleted() {
	h.pool.DeletedAt = soft_delete.DeletedAt(1730000000)
	h.pool.Name = "renamed-after-publish"
	h.pool.SupportAlert = 0
	h.pool.SupportRecord = 0
	h.pool.PrometheusInstances = model.StringList{"invalid old target"}
	h.sourceFailure()
}
func (h *retirementFixture) assertRetired(t *testing.T) {
	t.Helper()
	if len(h.members) != 0 {
		t.Fatal("retired members remain")
	}
	for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
		if h.values[h.artifact(ip)] != "" {
			t.Fatal("retired artifact remains")
		}
	}
	for _, row := range h.rows {
		if row.PoolID == 7 && row.ConfigType == h.kind {
			t.Fatal("retired owned metadata remains")
		}
	}
	if h.values[h.ack] == "" || len(h.values[h.ack]) > 8 && h.values[h.ack][:8] == "pending:" {
		t.Fatal("retirement ACK missing")
	}
}

func TestDeletedScrapePoolsRetireWithoutReadingDisabledOrBrokenDefinitions(t *testing.T) {
	for _, kind := range []string{"prom", "alert", "record"} {
		for _, path := range []string{"explicit", "periodic"} {
			t.Run(kind+"/"+path, func(t *testing.T) {
				h := newRetirementFixture(t, kind)
				ctx := context.Background()
				if err := h.publisher.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				calls := h.sourceCalls()
				oldACK := h.values[h.ack]
				// Same numeric ID is not ownership of either AM namespace.
				h.rows[98] = &model.MonitorConfig{Model: model.Model{ID: 98}, PoolID: 7, ConfigType: 2, Name: "AM-main", InstanceIP: "127.0.0.1"}
				h.rows[99] = &model.MonitorConfig{Model: model.Model{ID: 99}, PoolID: 7, ConfigType: 5, Name: "AM-alert", InstanceIP: "127.0.0.1"}
				h.markDeleted()
				var err error
				if path == "explicit" {
					err = h.publisher.RetireDeletedScrapePool(ctx, 7)
				} else {
					err = h.publisher.ReconcileDeletedScrapePools(ctx)
				}
				if err != nil {
					t.Fatal(err)
				}
				h.assertRetired(t)
				if h.sourceCalls() != calls || oldACK == h.values[h.ack] {
					t.Fatal("deletion read broken definitions or reused active version")
				}
				if h.rows[98] == nil || h.rows[99] == nil {
					t.Fatal("same-ID AM metadata was retired")
				}
				writes := h.writes()
				h.rebuild()
				if err := h.publisher.ReconcileDeletedScrapePools(ctx); err != nil {
					t.Fatal(err)
				}
				if h.writes() != writes {
					t.Fatal("completed tombstone not idempotent after rebuild")
				}
			})
		}
	}
}
func TestDeletedScrapePoolPartialFailuresRecoverAfterRebuild(t *testing.T) {
	for _, kind := range []string{"prom", "alert", "record"} {
		for _, stage := range []string{"hash read", "members read", "db read", "pending", "sadd", "db delete", "del", "srem", "ack"} {
			t.Run(kind+"/"+stage, func(t *testing.T) {
				h := newRetirementFixture(t, kind)
				ctx := context.Background()
				if err := h.publisher.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				h.markDeleted()
				h.failure(stage)
				if err := h.publisher.RetireDeletedScrapePool(ctx, 7); err == nil {
					t.Fatal("failure reported completed retirement")
				}
				if stage == "pending" {
					for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
						if h.values[h.artifact(ip)] == "" {
							t.Fatal("failed pending write changed visible artifact")
						}
					}
					if kind == "record" || kind == "alert" || kind == "prom" {
						pending := false
						if kind == "record" {
							pending = recordFixturePending(h.publisher.(*recordRuleConfigCache))
						} else if kind == "alert" {
							pending = alertFixturePending(h.publisher.(*alertRuleConfigCache))
						} else {
							pending = promFixturePending(h.publisher.(*prometheusConfigCache), 7)
						}
						if !pending || len(h.members) != 2 {
							t.Fatal("failed prepare lost durable retirement or changed members")
						}
					} else if len(h.rows) != 2 || len(h.members) != 2 {
						t.Fatal("failed pending write changed metadata/members")
					}
				}
				h.failure("")
				h.rebuild()
				if err := h.publisher.ReconcileDeletedScrapePools(ctx); err != nil {
					t.Fatal(err)
				}
				h.assertRetired(t)
			})
		}
	}
}
func TestDeletedScrapePoolRequiresTombstoneAndPreservesForeignCustomOwners(t *testing.T) {
	for _, kind := range []string{"prom", "alert", "record"} {
		for _, mode := range []string{"active", "absent", "read failure", "foreign", "custom"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				h := newRetirementFixture(t, kind)
				ctx := context.Background()
				if err := h.publisher.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				if mode != "active" {
					h.markDeleted()
				}
				switch mode {
				case "absent":
					h.inventory.pool = nil
				case "read failure":
					h.inventory.tombstoneFailure = true
				case "foreign":
					for _, row := range h.rows {
						row.PoolID = 99
					}
				case "custom":
					for _, row := range h.rows {
						row.Name = "custom"
					}
				}
				writes := h.writes()
				if err := h.publisher.RetireDeletedScrapePool(ctx, 7); err == nil {
					t.Fatal("missing authorization/ownership accepted")
				}
				if h.writes() != writes {
					t.Fatal("authorization conflict performed writes")
				}
			})
		}
	}
}
func TestDeletedScrapePoolFullInventoryFailureNeverPublishesEarlyPages(t *testing.T) {
	for _, kind := range []string{"prom", "alert", "record"} {
		for _, mode := range []string{"error", "early empty", "total drift", "duplicate"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				h := newRetirementFixture(t, kind)
				ctx := context.Background()
				if err := h.publisher.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				h.markDeleted()
				first := []*model.MonitorScrapePool{h.pool}
				for id := 10; len(first) < 100; id++ {
					first = append(first, &model.MonitorScrapePool{Model: model.Model{ID: id}})
				}
				h.inventory.page = func(page int) ([]*model.MonitorScrapePool, int64, error) {
					if page == 1 {
						return first, 101, nil
					}
					switch mode {
					case "error":
						return nil, 0, errors.New("fixture page failure")
					case "early empty":
						return nil, 101, nil
					case "total drift":
						return []*model.MonitorScrapePool{{Model: model.Model{ID: 1000}}}, 100, nil
					default:
						return []*model.MonitorScrapePool{h.pool}, 101, nil
					}
				}
				writes := h.writes()
				if err := h.publisher.ReconcileDeletedScrapePools(ctx); err == nil {
					t.Fatal("incomplete lifecycle inventory accepted")
				}
				if h.writes() != writes {
					t.Fatal("early page authorized retirement before inventory completed")
				}
			})
		}
	}
}
func TestDeletedScrapePoolRetirementWaitsForOldInflightPublisher(t *testing.T) {
	for _, kind := range []string{"prom", "alert", "record"} {
		t.Run(kind, func(t *testing.T) {
			h := newRetirementFixture(t, kind)
			ctx := context.Background()
			held, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			h.onRead(func() { once.Do(func() { close(held); <-release }) })
			published := make(chan error, 1)
			go func() { published <- h.publisher.GenerateMainConfig(ctx) }()
			<-held
			h.markDeleted()
			retired := make(chan error, 1)
			go func() { retired <- h.publisher.RetireDeletedScrapePool(ctx, 7) }()
			close(release)
			if err := <-published; err != nil {
				t.Fatal(err)
			}
			if err := <-retired; err != nil {
				t.Fatal(err)
			}
			h.assertRetired(t)
			if err := h.publisher.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			h.assertRetired(t)
		})
	}
}
func TestDeletedScrapePoolInventoryCopiesPageBeforeReuse(t *testing.T) {
	first := make([]*model.MonitorScrapePool, 100)
	for i := range first {
		first[i] = &model.MonitorScrapePool{Model: model.Model{ID: i + 1}}
	}
	first[0].DeletedAt = 10
	source := &retirementInventory{page: func(page int) ([]*model.MonitorScrapePool, int64, error) {
		if page == 1 {
			return first, 101, nil
		}
		first[0].DeletedAt = 0
		first[0].ID = 999
		return []*model.MonitorScrapePool{{Model: model.Model{ID: 101}}}, 101, nil
	}}
	rows, err := readScrapePoolLifecycleSnapshot(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].ID != 1 || rows[0].DeletedAt != 10 {
		t.Fatal(fmt.Sprintf("first page mutated during subsequent read: id=%d", rows[0].ID))
	}
}

func TestLiveScrapePoolRetirementAnchorsDBOnlyArtifactsBeforeSQLDelete(t *testing.T) {
	for _, kind := range []string{"prom", "alert", "record"} {
		modes := []string{"empty targets", "move targets"}
		if kind != "prom" {
			modes = append(modes, "feature off")
		}
		for _, mode := range modes {
			for _, stage := range []string{"sadd", "del"} {
				t.Run(kind+"/"+mode+"/"+stage, func(t *testing.T) {
					h := newRetirementFixture(t, kind)
					ctx := context.Background()
					if err := h.publisher.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					// Canonical metadata is authoritative even when the old member index was
					// never written. Do not fabricate a new index before exercising the code.
					clear(h.members)
					switch mode {
					case "empty targets":
						h.pool.PrometheusInstances = nil
					case "move targets":
						h.pool.PrometheusInstances = model.StringList{"127.0.0.3"}
					case "feature off":
						h.pool.SupportAlert = 2
						h.pool.SupportRecord = 2
					}
					h.failure(stage)
					if err := h.publisher.GenerateMainConfig(ctx); err == nil {
						t.Fatal("injected retirement failure reported complete")
					}
					if stage == "sadd" {
						if kind == "record" || kind == "alert" || kind == "prom" {
							pending := false
							if kind == "record" {
								pending = recordFixturePending(h.publisher.(*recordRuleConfigCache))
							} else if kind == "alert" {
								pending = alertFixturePending(h.publisher.(*alertRuleConfigCache))
							} else {
								pending = promFixturePending(h.publisher.(*prometheusConfigCache), 7)
							}
							if !pending {
								t.Fatal("failed projection control lost durable retirement")
							}
						} else if len(h.rows) != 2 {
							t.Fatal("failed retirement anchor allowed SQL deletion")
						}
						for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
							if h.values[h.artifact(ip)] == "" {
								t.Fatal("failed retirement anchor deleted visible artifact")
							}
						}
					}
					h.failure("")
					h.rebuild()
					if err := h.publisher.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
						if h.values[h.artifact(ip)] != "" || h.members[ip] {
							t.Fatal("DB-only retired target lost across failure/rebuild")
						}
					}
					for _, row := range h.rows {
						if row.InstanceIP != "127.0.0.3" {
							t.Fatal("stale metadata remains")
						}
					}
					if mode == "move targets" {
						if len(h.rows) != 1 || len(h.members) != 1 || !h.members["127.0.0.3"] || h.values[h.artifact("127.0.0.3")] == "" {
							t.Fatal("current target was lost while retiring old targets")
						}
					} else if len(h.rows) != 0 || len(h.members) != 0 {
						t.Fatal("empty desired did not retire all known owned targets")
					}
				})
			}
		}
	}
}
