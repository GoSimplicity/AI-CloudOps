package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	pc "github.com/prometheus/prometheus/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type promReconcileSource struct {
	scrapedao.ScrapeJobDAO
	rows    []*model.MonitorScrapeJob
	failure bool
	calls   int
	onRead  func()
}

func (s *promReconcileSource) GetMonitorScrapeJobsByPoolId(context.Context, int) ([]*model.MonitorScrapeJob, error) {
	s.calls++
	if s.failure {
		return nil, errors.New("fixture source failure")
	}
	if s.onRead != nil {
		s.onRead()
	}
	return s.rows, nil
}

type promReconcilePools struct {
	scrapedao.ScrapePoolDAO
	pool    *model.MonitorScrapePool
	failure bool
}

func (s *promReconcilePools) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if s.failure {
		return nil, 0, errors.New("fixture pool failure")
	}
	if req.Page > 1 {
		return nil, 1, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

type promReconcileRedis struct {
	redis.Cmdable
	values  map[string]string
	members map[string]bool
	failure string
	writes  int
	onRead  func()
}

func (s *promReconcileRedis) Get(_ context.Context, key string) *redis.StringCmd {
	if s.onRead != nil {
		s.onRead()
	}
	if s.failure == "hash read" && strings.Contains(key, ":publication:") || s.failure == "artifact read" && key == buildRedisKeyPrometheusMain("127.0.0.1") {
		return redis.NewStringResult("", errors.New("fixture read failure"))
	}
	v, ok := s.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (s *promReconcileRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	s.writes++
	v := value.(string)
	isACK := strings.Contains(key, ":publication:")
	if s.failure == "pending" && isACK && strings.HasPrefix(v, "pending:") || s.failure == "ack" && isACK && !strings.HasPrefix(v, "pending:") || s.failure == "set" && key == buildRedisKeyPrometheusMain("127.0.0.1") {
		return redis.NewStatusResult("", errors.New("fixture write failure"))
	}
	s.values[key] = v
	return redis.NewStatusResult("OK", nil)
}
func (s *promReconcileRedis) SMembers(context.Context, string) *redis.StringSliceCmd {
	if s.failure == "members read" {
		return redis.NewStringSliceResult(nil, errors.New("fixture membership read failure"))
	}
	var ips []string
	for ip := range s.members {
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	return redis.NewStringSliceResult(ips, nil)
}
func (s *promReconcileRedis) SAdd(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	s.writes++
	if s.failure == "sadd" {
		return redis.NewIntResult(0, errors.New("fixture membership write failure"))
	}
	for _, v := range values {
		s.members[v.(string)] = true
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *promReconcileRedis) SRem(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	s.writes++
	if s.failure == "srem" {
		return redis.NewIntResult(0, errors.New("fixture membership removal failure"))
	}
	for _, v := range values {
		delete(s.members, v.(string))
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *promReconcileRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	s.writes++
	if s.failure == "del" {
		return redis.NewIntResult(0, errors.New("fixture artifact removal failure"))
	}
	for _, key := range keys {
		delete(s.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

type promReconcileDB struct {
	configdao.MonitorConfigDAO
	rows         map[int]*model.MonitorConfig
	next, writes int
	failure      string
}

func (s *promReconcileDB) GetMonitorConfigsByInstances(_ context.Context, ips []string, kind int8) ([]*model.MonitorConfig, error) {
	var out []*model.MonitorConfig
	for _, row := range s.rows {
		for _, ip := range ips {
			if row.InstanceIP == ip && row.ConfigType == kind {
				copy := *row
				out = append(out, &copy)
			}
		}
	}
	return out, nil
}
func (s *promReconcileDB) GetMonitorConfigList(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	var rows []*model.MonitorConfig
	for id := 1; id <= s.next; id++ {
		if row := s.rows[id]; row != nil && (req.ConfigType == nil || *req.ConfigType == row.ConfigType) {
			copy := *row
			rows = append(rows, &copy)
		}
	}
	total := len(rows)
	start := (req.Page - 1) * req.Size
	if start >= total {
		return nil, int64(total), nil
	}
	end := start + req.Size
	if end > total {
		end = total
	}
	return rows[start:end], int64(total), nil
}
func (s *promReconcileDB) save(rows []*model.MonitorConfig, mode string) error {
	s.writes++
	if s.failure == mode {
		return errors.New("fixture persistence failure")
	}
	for _, row := range rows {
		copy := *row
		if copy.ID == 0 {
			s.next++
			copy.ID = s.next
		}
		s.rows[copy.ID] = &copy
	}
	return nil
}
func (s *promReconcileDB) BatchCreateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows, "db create")
}
func (s *promReconcileDB) BatchUpdateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows, "db update")
}
func (s *promReconcileDB) DeleteMonitorConfig(_ context.Context, id int) error {
	s.writes++
	if s.failure == "db delete" {
		return errors.New("fixture delete failure")
	}
	delete(s.rows, id)
	return nil
}

func promReconcileRow() *model.MonitorScrapeJob {
	return &model.MonitorScrapeJob{Model: model.Model{ID: 31}, PoolID: 7, Name: "fixture-job", Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, Scheme: "http", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, IpAddress: "127.0.0.1:9100"}
}
func parsePromReconcile(t *testing.T, content string) *pc.Config {
	t.Helper()
	if content == "" {
		t.Fatal("missing Prometheus configuration")
	}
	parsed, err := pc.Load(content, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal("fixed Prometheus parser rejected operational configuration")
	}
	return parsed
}
func newPromReconcile(t *testing.T) (*prometheusConfigCache, *promReconcileSource, *promReconcileRedis, *promReconcileDB, *model.MonitorScrapePool) {
	t.Helper()
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "prom-pool-A", ScrapeInterval: 30, ScrapeTimeout: 10, PrometheusInstances: model.StringList{"127.0.0.1", "127.0.0.2"}}
	source := &promReconcileSource{rows: []*model.MonitorScrapeJob{promReconcileRow()}}
	store := &promReconcileRedis{values: map[string]string{}, members: map[string]bool{}}
	db := &promReconcileDB{rows: map[int]*model.MonitorConfig{}}
	logger := zap.NewNop()
	cache := &prometheusConfigCache{logger: logger, redis: store, scrapeJobDAO: source, scrapePoolDAO: &promReconcilePools{pool: pool}, configDAO: db, batchManager: NewBatchConfigManager(db, logger)}
	return cache, source, store, db, pool
}
func promReconcileACK(poolID int) string {
	return fmt.Sprintf("%s:publication:prometheus:pool:%d", redisPrefix, poolID)
}
func TestPromReconcileFieldOnlyEditsAndEmpty(t *testing.T) {
	for _, field := range []string{"name", "credentials", "metrics", "empty"} {
		t.Run(field, func(t *testing.T) {
			c, source, store, _, pool := newPromReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			before := store.values[buildRedisKeyPrometheusMain("127.0.0.1")]
			beforeACK := store.values[promReconcileACK(pool.ID)]
			switch field {
			case "name":
				source.rows[0].Name = "changed"
			case "metrics":
				source.rows[0].MetricsPath = "/metrics-B"
			case "credentials":
				source.rows[0].BearerToken = "fixture-operational-token"
			case "empty":
				source.rows = nil
			}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			after := store.values[buildRedisKeyPrometheusMain("127.0.0.1")]
			if source.calls != 2 || after == before || store.values[promReconcileACK(pool.ID)] == beforeACK {
				t.Fatal("source-only change did not replace old configuration")
			}
			for _, ip := range pool.PrometheusInstances {
				groups := parsePromReconcile(t, store.values[buildRedisKeyPrometheusMain(ip)])
				if field == "empty" && len(groups.ScrapeConfigs) != 0 {
					t.Fatal("last active rule was retained")
				}
			}
		})
	}
}
func TestPromReconcileSingleSnapshot(t *testing.T) {
	c, source, store, _, _ := newPromReconcile(t)
	store.onRead = func() {
		store.onRead = nil
		source.rows[0].Name = "changed-during-read"
		source.rows[0].MetricsPath = "/later"
	}
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	groups := parsePromReconcile(t, store.values[buildRedisKeyPrometheusMain("127.0.0.1")])
	if source.calls != 1 || groups.ScrapeConfigs[0].JobName == "changed-during-read" {
		t.Fatal("generation and fingerprint did not share a detached single DAO snapshot")
	}
}
func TestPromReconcileReadAndInvalidPreserveLastGood(t *testing.T) {
	for _, stage := range []string{"source", "hash read", "artifact read", "members read", "db read", "all invalid"} {
		t.Run(stage, func(t *testing.T) {
			c, source, store, db, _ := newPromReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			before := copyPromReconcileRedis(store.values)
			dbBefore := copyPromReconcileRows(db.rows)
			writes := store.writes
			dbWrites := db.writes
			store.failure = stage
			db.failure = stage
			if stage == "source" {
				source.failure = true
			}
			if stage == "all invalid" {
				source.rows[0].ScrapeInterval = 5
			}
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("failed/invalid source was acknowledged as empty")
			}
			if writes != store.writes || dbWrites != db.writes || !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(dbBefore, db.rows) {
				t.Fatal("pre-publication failure changed last good artifacts")
			}
		})
	}
}
func TestPromReconcileFailureThenReturnToEitherSnapshot(t *testing.T) {
	for _, stage := range []string{"pending", "sadd", "db create", "db update", "set", "del", "db delete", "srem", "ack"} {
		for _, target := range []string{"A", "B"} {
			for _, restart := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/restart%v", stage, target, restart), func(t *testing.T) {
					c, source, store, db, pool := newPromReconcile(t)
					ctx := context.Background()
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					initialName := pool.Name
					initialHash := store.values[promReconcileACK(pool.ID)]
					source.rows[0].MetricsPath = "/metrics-B"
					pool.Name = "renamed-pool-B"
					pool.PrometheusInstances = model.StringList{"127.0.0.1", "127.0.0.3"}
					store.failure = stage
					db.failure = stage
					if c.GenerateMainConfig(ctx) == nil {
						t.Fatal("failed B reported success")
					}
					store.failure = ""
					db.failure = ""
					if target == "A" {
						source.rows[0].MetricsPath = "/metrics"
						pool.Name = initialName
						pool.PrometheusInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
					}
					if restart {
						replacement, err := NewPrometheusConfigCache(&config.Config{}, zap.NewNop(), c.scrapePoolDAO, source, db, NewBatchConfigManager(db, zap.NewNop()), store)
						if err != nil {
							t.Fatal(err)
						}
						c = replacement.(*prometheusConfigCache)
					}
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					marker := store.values[promReconcileACK(pool.ID)]
					if len(marker) != 64 || strings.HasPrefix(marker, "pending:") {
						t.Fatal("complete recovery lacks success ACK")
					}
					if target == "A" && marker != initialHash {
						t.Fatal("source reversion did not restore A identity")
					}
					for _, ip := range pool.PrometheusInstances {
						parsed := parsePromReconcile(t, store.values[buildRedisKeyPrometheusMain(ip)])
						if ip == "127.0.0.1" && parsed.ScrapeConfigs[0].MetricsPath != source.rows[0].MetricsPath {
							t.Fatal("old ACK skipped repairing source reversion")
						}
						if !store.members[ip] {
							t.Fatal("desired membership missing")
						}
					}
					stale := "127.0.0.2"
					if target == "A" {
						stale = "127.0.0.3"
					}
					if store.values[buildRedisKeyPrometheusMain(stale)] != "" || store.members[stale] {
						t.Fatal("retired pending target remains visible")
					}
					for _, row := range db.rows {
						if row.PoolID == pool.ID && row.InstanceIP == stale {
							t.Fatal("retired metadata remains")
						}
						if row.PoolID == pool.ID && row.ConfigContent != store.values[buildRedisKeyPrometheusMain(row.InstanceIP)] {
							t.Fatal("DB/Redis disagree after recovery")
						}
					}
				})
			}
		}
	}
}
func TestPromReconcilePendingFailureIsZeroArtifact(t *testing.T) {
	c, _, store, db, _ := newPromReconcile(t)
	store.failure = "pending"
	if c.GenerateMainConfig(context.Background()) == nil {
		t.Fatal("pending failure acknowledged")
	}
	if len(store.values) != 0 || len(store.members) != 0 || len(db.rows) != 0 || db.writes != 0 {
		t.Fatal("artifact appeared before reliable pending marker")
	}
}
func TestPromReconcileSameInstanceSerializes(t *testing.T) {
	c, _, _, _, _ := newPromReconcile(t)
	var wg sync.WaitGroup
	failures := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.GenerateMainConfig(context.Background()); err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}

func TestPromReconcileOwnershipConflictsArePreflighted(t *testing.T) {
	for _, retired := range []bool{false, true} {
		for _, foreign := range []bool{false, true} {
			t.Run(fmt.Sprintf("retired%v/foreign%v", retired, foreign), func(t *testing.T) {
				c, source, store, db, pool := newPromReconcile(t)
				ctx := context.Background()
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				ip := "127.0.0.1"
				if retired {
					ip = "127.0.0.2"
				}
				db.next++
				row := &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: pool.ID, ConfigType: model.ConfigTypePrometheus, InstanceIP: ip, Name: "custom-preserved", ConfigContent: "keep"}
				if foreign {
					row.PoolID = 99
				}
				db.rows[row.ID] = row
				before := copyPromReconcileRedis(store.values)
				beforeRows := copyPromReconcileRows(db.rows)
				writes, dbWrites := store.writes, db.writes
				source.rows[0].MetricsPath = "/metrics-B"
				if retired {
					pool.PrometheusInstances = model.StringList{"127.0.0.1"}
				}
				if c.GenerateMainConfig(ctx) == nil {
					t.Fatal("ambiguous ownership acknowledged")
				}
				if writes != store.writes || dbWrites != db.writes || !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(beforeRows, db.rows) {
					t.Fatal("ownership conflict modified an artifact or ACK")
				}
			})
		}
	}
}
func TestPromReconcileRetiresCanonicalUnindexedOrphanAcrossPages(t *testing.T) {
	c, _, store, db, pool := newPromReconcile(t)
	for i := 0; i < 110; i++ {
		db.next++
		db.rows[db.next] = &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 99, ConfigType: model.ConfigTypePrometheus, InstanceIP: fmt.Sprintf("custom-%d", i), Name: "custom-preserved", ConfigContent: "keep"}
	}
	db.next++
	orphanID := db.next
	ip := "127.0.0.9"
	db.rows[orphanID] = &model.MonitorConfig{Model: model.Model{ID: orphanID}, PoolID: pool.ID, ConfigType: model.ConfigTypePrometheus, InstanceIP: ip, Name: fmt.Sprintf(ConfigNamePrometheus, pool.ID, ip), ConfigContent: "old"}
	store.values[buildRedisKeyPrometheusMain(ip)] = "old"
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if db.rows[orphanID] != nil || store.values[buildRedisKeyPrometheusMain(ip)] != "" || len(db.rows) != 112 {
		t.Fatal("paged orphan retirement lost ownership boundaries")
	}
}
func TestPromReconcileCancellationBeforePendingIsReadOnly(t *testing.T) {
	c, _, store, db, _ := newPromReconcile(t)
	ctx, cancel := context.WithCancel(context.Background())
	store.onRead = func() { store.onRead = nil; cancel() }
	if !errors.Is(c.GenerateMainConfig(ctx), context.Canceled) {
		t.Fatal("cancellation was not propagated")
	}
	if len(store.values) != 0 || len(store.members) != 0 || len(db.rows) != 0 || db.writes != 0 {
		t.Fatal("cancellation published artifacts")
	}
}

func TestPromReconcileEmptyTargets(t *testing.T) {
	for _, previous := range []bool{false, true} {
		t.Run(fmt.Sprintf("previous%v", previous), func(t *testing.T) {
			c, source, store, db, pool := newPromReconcile(t)
			ctx := context.Background()
			if previous {
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
			}
			pool.PrometheusInstances = nil
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if len(db.rows) != 0 || len(store.members) != 0 || store.values[buildRedisKeyPrometheusMain("127.0.0.1")] != "" || store.values[buildRedisKeyPrometheusMain("127.0.0.2")] != "" || len(store.values[promReconcileACK(pool.ID)]) != 64 {
				t.Fatal("empty targets did not retire owned artifacts and acknowledge the empty desired set")
			}
			wantReads := 1
			if previous {
				wantReads = 2
			}
			if source.calls != wantReads {
				t.Fatal("empty targets skipped the authoritative rule read")
			}
		})
	}
}
func TestPromReconcileEmptyTargetFailureRecovery(t *testing.T) {
	for _, stage := range []string{"pending", "del", "db delete", "srem", "ack"} {
		for _, target := range []string{"A", "B"} {
			for _, restart := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/restart%v", stage, target, restart), func(t *testing.T) {
					c, source, store, db, pool := newPromReconcile(t)
					ctx := context.Background()
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					hashA := store.values[promReconcileACK(pool.ID)]
					nameA := pool.Name
					pool.Name = "renamed-empty-B"
					pool.PrometheusInstances = nil
					store.failure, db.failure = stage, stage
					if c.GenerateMainConfig(ctx) == nil {
						t.Fatal("failed empty-target retirement reported success")
					}
					store.failure, db.failure = "", ""
					if target == "A" {
						pool.Name = nameA
						pool.PrometheusInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
					}
					if restart {
						replacement, err := NewPrometheusConfigCache(&config.Config{}, zap.NewNop(), c.scrapePoolDAO, source, db, NewBatchConfigManager(db, zap.NewNop()), store)
						if err != nil {
							t.Fatal(err)
						}
						c = replacement.(*prometheusConfigCache)
					}
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					marker := store.values[promReconcileACK(pool.ID)]
					if len(marker) != 64 || strings.HasPrefix(marker, "pending:") {
						t.Fatal("recovery retained pending state")
					}
					if target == "A" {
						if marker != hashA || len(db.rows) != 2 || len(store.members) != 2 {
							t.Fatal("A was not restored after failed empty B")
						}
						for _, ip := range pool.PrometheusInstances {
							parsePromReconcile(t, store.values[buildRedisKeyPrometheusMain(ip)])
						}
					} else if len(db.rows) != 0 || len(store.members) != 0 || store.values[buildRedisKeyPrometheusMain("127.0.0.1")] != "" || store.values[buildRedisKeyPrometheusMain("127.0.0.2")] != "" {
						t.Fatal("B retained retired artifacts")
					}
				})
			}
		}
	}
}
func TestPromReconcileEmptyTargetsDoNotTurnReadFailureIntoRemoval(t *testing.T) {
	c, source, store, db, pool := newPromReconcile(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	before, beforeRows := copyPromReconcileRedis(store.values), copyPromReconcileRows(db.rows)
	members := len(store.members)
	writes, dbWrites := store.writes, db.writes
	pool.PrometheusInstances = nil
	source.failure = true
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("failed empty-target read acknowledged")
	}
	if !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(beforeRows, db.rows) || len(store.members) != members || store.writes != writes || db.writes != dbWrites {
		t.Fatal("failed read removed last-good artifacts")
	}
}

type promIncompleteMetadata struct {
	*promReconcileDB
	mode  string
	first []*model.MonitorConfig
	owned []*model.MonitorConfig
}

func (d promIncompleteMetadata) GetMonitorConfigList(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	if d.mode == "negative total" {
		return nil, -1, nil
	}
	if d.mode == "first empty" {
		return nil, 102, nil
	}
	if req.Page == 1 {
		if d.mode == "total smaller than page" {
			return d.first, 99, nil
		}
		if d.mode == "final count mismatch" {
			return d.first, 103, nil
		}
		return d.first, 102, nil
	}
	switch d.mode {
	case "later empty":
		return nil, 102, nil
	case "total drift":
		return d.owned, 101, nil
	case "duplicate ID":
		return []*model.MonitorConfig{d.first[0], d.owned[0]}, 102, nil
	case "final count mismatch":
		return d.owned, 103, nil
	}
	return d.owned, 102, nil
}
func TestPromReconcileIncompleteMetadataCannotAcknowledgeOrDuplicate(t *testing.T) {
	for _, mode := range []string{"negative total", "first empty", "total smaller than page", "later empty", "total drift", "duplicate ID", "final count mismatch"} {
		t.Run(mode, func(t *testing.T) {
			c, source, store, db, _ := newPromReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			var owned []*model.MonitorConfig
			for id := 1; id <= db.next; id++ {
				if row := db.rows[id]; row != nil {
					copy := *row
					owned = append(owned, &copy)
				}
			}
			var first []*model.MonitorConfig
			for i := 0; i < 100; i++ {
				db.next++
				row := &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 99, ConfigType: model.ConfigTypePrometheus, InstanceIP: fmt.Sprintf("unrelated-%d", i), Name: "custom-preserved", ConfigContent: "keep"}
				db.rows[row.ID] = row
				first = append(first, row)
			}
			c.configDAO = promIncompleteMetadata{promReconcileDB: db, mode: mode, first: first, owned: owned}
			before, beforeRows := copyPromReconcileRedis(store.values), copyPromReconcileRows(db.rows)
			writes, dbWrites := store.writes, db.writes
			source.rows[0].MetricsPath = "/metrics-B"
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("incomplete or drifting metadata was acknowledged")
			}
			if writes != store.writes || dbWrites != db.writes || !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(beforeRows, db.rows) {
				t.Fatal("rejected metadata changed pending/artifacts or created duplicate owners")
			}
		})
	}
}

func TestPromReconcileSameACKRepairsMissingOrDriftingArtifacts(t *testing.T) {
	for _, mode := range []string{"cache missing", "cache drift", "member missing", "metadata content", "metadata hash", "metadata status"} {
		t.Run(mode, func(t *testing.T) {
			c, _, store, db, pool := newPromReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			key := buildRedisKeyPrometheusMain("127.0.0.1")
			body, ack := store.values[key], store.values[promReconcileACK(pool.ID)]
			var row *model.MonitorConfig
			for _, saved := range db.rows {
				if saved.InstanceIP == "127.0.0.1" {
					row = saved
				}
			}
			if row == nil || body == "" || len(ack) != 64 {
				t.Fatal("initial acknowledged artifacts missing")
			}
			switch mode {
			case "cache missing":
				delete(store.values, key)
			case "cache drift":
				store.values[key] = "corrupted"
			case "member missing":
				delete(store.members, "127.0.0.1")
			case "metadata content":
				row.ConfigContent = "drift"
			case "metadata hash":
				row.ConfigHash = "drift"
			case "metadata status":
				row.Status = model.ConfigStatusInactive
			}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if store.values[key] != body || !store.members["127.0.0.1"] || store.values[promReconcileACK(pool.ID)] != ack {
				t.Fatal("same ACK incorrectly skipped artifact repair")
			}
			for _, saved := range db.rows {
				if saved.InstanceIP == "127.0.0.1" && (saved.ConfigContent != body || saved.ConfigHash != calculateConfigHash(body) || saved.Status != model.ConfigStatusActive) {
					t.Fatal("metadata drift was not repaired")
				}
			}
			parsePromReconcile(t, store.values[key])
		})
	}
}
func TestPromReconcileNoncanonicalNameRemainsAnUnownedConflict(t *testing.T) {
	c, _, store, db, _ := newPromReconcile(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	for _, row := range db.rows {
		if row.InstanceIP == "127.0.0.1" {
			row.Name = "operator-custom-name"
		}
	}
	before, beforeRows := copyPromReconcileRedis(store.values), copyPromReconcileRows(db.rows)
	writes, dbWrites := store.writes, db.writes
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("noncanonical name was guessed to be generated")
	}
	if store.writes != writes || db.writes != dbWrites || !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(beforeRows, db.rows) {
		t.Fatal("ambiguous name was overwritten")
	}
}

func copyPromReconcileRedis(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
func copyPromReconcileRows(source map[int]*model.MonitorConfig) map[int]*model.MonitorConfig {
	copy := make(map[int]*model.MonitorConfig, len(source))
	for id, row := range source {
		value := *row
		copy[id] = &value
	}
	return copy
}

func TestPromReconcileFingerprintContainsActualPerInstanceShards(t *testing.T) {
	c, _, store, _, pool := newPromReconcile(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	first, second := store.values[buildRedisKeyPrometheusMain("127.0.0.1")], store.values[buildRedisKeyPrometheusMain("127.0.0.2")]
	version := store.values[promReconcileACK(pool.ID)]
	if first == second {
		t.Fatal("fixture lacks distinct shard renderings")
	}
	pool.PrometheusInstances = model.StringList{"127.0.0.2", "127.0.0.1"}
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if store.values[promReconcileACK(pool.ID)] == version || store.values[buildRedisKeyPrometheusMain("127.0.0.1")] != second || store.values[buildRedisKeyPrometheusMain("127.0.0.2")] != first {
		t.Fatal("order-sensitive rendered shard bytes were omitted from the fingerprint")
	}
	parsePromReconcile(t, store.values[buildRedisKeyPrometheusMain("127.0.0.1")])
}
func TestPromReconcilePoolInputsAreDetachedBeforeJobRead(t *testing.T) {
	c, source, store, _, pool := newPromReconcile(t)
	pool.Tags = model.StringList{"zone=before"}
	source.onRead = func() {
		source.onRead = nil
		pool.Tags[0] = "zone=after"
		pool.PrometheusInstances[0] = "127.0.0.9"
		pool.ScrapeInterval = 90
		pool.ScrapeTimeout = 20
		pool.RemoteWriteUrl = "https://later.invalid/write"
	}
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	parsed := parsePromReconcile(t, store.values[buildRedisKeyPrometheusMain("127.0.0.1")])
	if parsed.GlobalConfig.ExternalLabels.Get("zone") != "before" || parsed.GlobalConfig.ScrapeInterval.String() != "30s" || len(parsed.RemoteWriteConfigs) != 0 || store.values[buildRedisKeyPrometheusMain("127.0.0.9")] != "" {
		t.Fatal("mutable pool fields changed the detached publication snapshot")
	}
}

type promReconcileIncompletePools struct {
	scrapedao.ScrapePoolDAO
	mode string
}

func (s promReconcileIncompletePools) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.Page == 1 {
		rows := make([]*model.MonitorScrapePool, 100)
		for i := range rows {
			rows[i] = &model.MonitorScrapePool{Model: model.Model{ID: i + 1}, Name: fmt.Sprintf("pool-%d", i+1), PrometheusInstances: model.StringList{fmt.Sprintf("192.0.2.%d", i+1)}, ScrapeInterval: 30, ScrapeTimeout: 10}
		}
		return rows, 101, nil
	}
	if s.mode == "early empty" {
		return nil, 101, nil
	}
	id := 101
	if s.mode == "duplicate ID" {
		id = 1
	}
	row := &model.MonitorScrapePool{Model: model.Model{ID: id}}
	total := int64(101)
	if s.mode == "total drift" {
		total = 102
	}
	return []*model.MonitorScrapePool{row}, total, nil
}
func TestPromReconcilePoolInventoryMustCompleteBeforePublication(t *testing.T) {
	for _, mode := range []string{"early empty", "duplicate ID", "total drift"} {
		t.Run(mode, func(t *testing.T) {
			c, source, store, db, _ := newPromReconcile(t)
			c.scrapePoolDAO = promReconcileIncompletePools{mode: mode}
			if c.GenerateMainConfig(context.Background()) == nil {
				t.Fatal("incomplete pool inventory accepted")
			}
			if source.calls != 0 || store.writes != 0 || db.writes != 0 || len(store.members) != 0 {
				t.Fatal("pool inventory failure published a partial inventory")
			}
		})
	}
}
