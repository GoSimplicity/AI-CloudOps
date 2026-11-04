package cache

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	common "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type recordReconcileSource struct {
	alertdao.AlertManagerRecordDAO
	rows    []*model.MonitorRecordRule
	failure bool
	calls   int
}

func (s *recordReconcileSource) GetMonitorRecordRuleByPoolId(context.Context, int) ([]*model.MonitorRecordRule, int64, error) {
	s.calls++
	if s.failure {
		return nil, 0, errors.New("fixture source failure")
	}
	return s.rows, int64(len(s.rows)), nil
}

type recordReconcilePools struct {
	scrapedao.ScrapePoolDAO
	pool    *model.MonitorScrapePool
	failure bool
}

func (s *recordReconcilePools) GetMonitorScrapePoolSupportedRecord(context.Context) ([]*model.MonitorScrapePool, int64, error) {
	if s.failure {
		return nil, 0, errors.New("fixture pool failure")
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

func (s *recordReconcilePools) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if s.failure {
		return nil, 0, errors.New("fixture pool failure")
	}
	if req.Page > 1 {
		return nil, 1, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

type recordReconcileRedis struct {
	redis.Cmdable
	values  map[string]string
	members map[string]bool
	failure string
	writes  int
	onRead  func()
}

func (s *recordReconcileRedis) Get(_ context.Context, key string) *redis.StringCmd {
	if s.onRead != nil {
		s.onRead()
	}
	if s.failure == "hash read" && strings.Contains(key, ":publication:") || s.failure == "artifact read" && key == buildRedisKeyRecordRule("127.0.0.1") {
		return redis.NewStringResult("", errors.New("fixture read failure"))
	}
	v, ok := s.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (s *recordReconcileRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	s.writes++
	v := value.(string)
	isACK := strings.Contains(key, ":publication:")
	if s.failure == "pending" && isACK && strings.HasPrefix(v, "pending:") || s.failure == "ack" && isACK && !strings.HasPrefix(v, "pending:") || s.failure == "set" && key == buildRedisKeyRecordRule("127.0.0.1") {
		return redis.NewStatusResult("", errors.New("fixture write failure"))
	}
	s.values[key] = v
	return redis.NewStatusResult("OK", nil)
}
func (s *recordReconcileRedis) SMembers(context.Context, string) *redis.StringSliceCmd {
	if s.onRead != nil {
		s.onRead()
	}
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
func (s *recordReconcileRedis) SAdd(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	s.writes++
	if s.failure == "sadd" {
		return redis.NewIntResult(0, errors.New("fixture membership write failure"))
	}
	for _, v := range values {
		s.members[v.(string)] = true
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *recordReconcileRedis) SRem(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	s.writes++
	if s.failure == "srem" {
		return redis.NewIntResult(0, errors.New("fixture membership removal failure"))
	}
	for _, v := range values {
		delete(s.members, v.(string))
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *recordReconcileRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	s.writes++
	if s.failure == "del" {
		return redis.NewIntResult(0, errors.New("fixture artifact removal failure"))
	}
	for _, key := range keys {
		delete(s.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

type recordReconcileDB struct {
	projection *recordProjectionFixture
	configdao.MonitorConfigDAO
	rows         map[int]*model.MonitorConfig
	next, writes int
	failure      string
}

func (s *recordReconcileDB) GetMonitorConfigsByInstances(_ context.Context, ips []string, kind int8) ([]*model.MonitorConfig, error) {
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
func (s *recordReconcileDB) GetMonitorConfigList(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
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
func (s *recordReconcileDB) save(rows []*model.MonitorConfig, mode string) error {
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
func (s *recordReconcileDB) BatchCreateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows, "db create")
}
func (s *recordReconcileDB) BatchUpdateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows, "db update")
}
func (s *recordReconcileDB) DeleteMonitorConfig(_ context.Context, id int) error {
	s.writes++
	if s.failure == "db delete" {
		return errors.New("fixture delete failure")
	}
	delete(s.rows, id)
	return nil
}

func recordReconcileRow() *model.MonitorRecordRule {
	return &model.MonitorRecordRule{Model: model.Model{ID: 31}, PoolID: 7, Name: "fixture:up", Expr: "up", Enable: 1}
}
func parseRecordReconcile(t *testing.T, content string) *rulefmt.RuleGroups {
	t.Helper()
	if content == "" {
		t.Fatal("missing recording-rule configuration")
	}
	groups, errs := rulefmt.Parse([]byte(content), false, common.UTF8Validation, parser.NewParser(parser.Options{}))
	if len(errs) > 0 {
		t.Fatalf("fixed Prometheus parser rejected recording rules: %v", errs)
	}
	return groups
}
func newRecordReconcile(t *testing.T) (*recordRuleConfigCache, *recordReconcileSource, *recordReconcileRedis, *recordReconcileDB, *model.MonitorScrapePool) {
	t.Helper()
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "record-pool-A", SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.1", "127.0.0.2"}}
	source := &recordReconcileSource{rows: []*model.MonitorRecordRule{recordReconcileRow()}}
	store := &recordReconcileRedis{values: map[string]string{}, members: map[string]bool{}}
	db := &recordReconcileDB{rows: map[int]*model.MonitorConfig{}}
	logger := zap.NewNop()
	cache := &recordRuleConfigCache{logger: logger, redis: store, alertRecordDAO: source, scrapePoolDAO: &recordReconcilePools{pool: pool}, configDAO: db, batchManager: NewBatchConfigManager(db, logger)}
	attachRecordReconcileProjection(t, cache, db)
	return cache, source, store, db, pool
}
func recordReconcileACK(poolID int) string {
	return fmt.Sprintf("%s:publication:record_rule:pool:%d", redisPrefix, poolID)
}
func TestRecordReconcileFieldOnlyEditsAndEmpty(t *testing.T) {
	for _, field := range []string{"name", "expr", "labels", "empty"} {
		t.Run(field, func(t *testing.T) {
			c, source, store, _, pool := newRecordReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			before := store.values[buildRedisKeyRecordRule("127.0.0.1")]
			switch field {
			case "name":
				source.rows[0].Name = "changed"
			case "expr":
				source.rows[0].Expr = "sum(up)"
			case "labels":
				source.rows[0].Labels = model.StringList{"team=changed"}
			case "empty":
				source.rows = nil
			}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			after := store.values[buildRedisKeyRecordRule("127.0.0.1")]
			if source.calls != 2 || after == before {
				t.Fatal("source-only change did not replace old configuration")
			}
			for _, ip := range pool.PrometheusInstances {
				groups := parseRecordReconcile(t, store.values[buildRedisKeyRecordRule(ip)])
				if field == "empty" && len(groups.Groups) != 0 {
					t.Fatal("last active rule was retained")
				}
			}
		})
	}
}
func TestRecordReconcileSingleSnapshot(t *testing.T) {
	c, source, store, _, _ := newRecordReconcile(t)
	store.onRead = func() {
		store.onRead = nil
		source.rows[0].Name = "changed-during-read"
		source.rows[0].Labels = model.StringList{"team=later"}
	}
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	groups := parseRecordReconcile(t, store.values[buildRedisKeyRecordRule("127.0.0.1")])
	if source.calls != 1 || groups.Groups[0].Rules[0].Record == "changed-during-read" {
		t.Fatal("generation and fingerprint did not share a detached single DAO snapshot")
	}
}
func TestRecordReconcileReadAndInvalidPreserveLastGood(t *testing.T) {
	for _, stage := range []string{"source", "hash read", "artifact read", "members read", "db read", "all invalid"} {
		t.Run(stage, func(t *testing.T) {
			c, source, store, db, _ := newRecordReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			before := copyRecordReconcileRedis(store.values)
			dbBefore := copyRecordReconcileRows(db.rows)
			writes := store.writes
			dbWrites := db.writes
			store.failure = stage
			db.failure = stage
			if stage == "source" {
				source.failure = true
			}
			if stage == "all invalid" {
				source.rows[0].Expr = "up[5m]"
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
func TestRecordReconcileFailureThenReturnToEitherSnapshot(t *testing.T) {
	for _, stage := range []string{"pending", "sadd", "db create", "db update", "set", "del", "db delete", "srem", "ack"} {
		for _, target := range []string{"A", "B"} {
			for _, restart := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/restart%v", stage, target, restart), func(t *testing.T) {
					c, source, store, db, pool := newRecordReconcile(t)
					ctx := context.Background()
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					initialName := pool.Name
					initialHash := store.values[recordReconcileACK(pool.ID)]
					source.rows[0].Expr = "sum(up)"
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
						source.rows[0].Expr = "up"
						pool.Name = initialName
						pool.PrometheusInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
					}
					if restart {
						c = NewRecordRuleConfigCache(zap.NewNop(), c.scrapePoolDAO, source, db, NewBatchConfigManager(db, zap.NewNop()), store, db.projection.journal, db.projection.history, nil).(*recordRuleConfigCache)
						db.projection.attach(c)
					}
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					marker := store.values[recordReconcileACK(pool.ID)]
					if len(marker) != 64 || strings.HasPrefix(marker, "pending:") {
						t.Fatal("complete recovery lacks success ACK")
					}
					if target == "A" && marker != initialHash {
						t.Fatal("source reversion did not restore A identity")
					}
					for _, ip := range pool.PrometheusInstances {
						parsed := parseRecordReconcile(t, store.values[buildRedisKeyRecordRule(ip)])
						if ip == "127.0.0.1" && parsed.Groups[0].Rules[0].Expr != source.rows[0].Expr {
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
					if store.values[buildRedisKeyRecordRule(stale)] != "" || store.members[stale] {
						t.Fatal("retired pending target remains visible")
					}
					for _, row := range db.rows {
						if row.PoolID == pool.ID && row.InstanceIP == stale {
							t.Fatal("retired metadata remains")
						}
						if row.PoolID == pool.ID && row.ConfigContent != store.values[buildRedisKeyRecordRule(row.InstanceIP)] {
							t.Fatal("DB/Redis disagree after recovery")
						}
					}
				})
			}
		}
	}
}
func TestRecordReconcilePendingFailureIsZeroArtifact(t *testing.T) {
	c, _, store, db, _ := newRecordReconcile(t)
	store.failure = "pending"
	if c.GenerateMainConfig(context.Background()) == nil {
		t.Fatal("pending failure acknowledged")
	}
	if len(store.values) != 0 || len(store.members) != 0 || !recordFixturePending(c) || len(db.rows) != 2 {
		t.Fatal("failed prepare published raw/ACK or lost its committed, unprojected SQL plan")
	}
}
func TestRecordReconcileIgnoresLegacyNameKeys(t *testing.T) {
	c, source, store, db, pool := newRecordReconcile(t)
	pool.Name = "pool:7"
	legacy := buildRedisHashKeyRecordRule(pool.Name)
	snapshotHash := calculateRecordPoolHash(pool, source.rows)
	store.values[legacy] = snapshotHash
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if legacy == recordReconcileACK(pool.ID) || store.values[legacy] != snapshotHash || len(store.values[recordReconcileACK(pool.ID)]) != 64 {
		t.Fatal("legacy name controlled trusted publication identity")
	}
	if len(db.rows) != 2 || store.values[buildRedisKeyRecordRule("127.0.0.1")] == "" {
		t.Fatal("a legacy SHA incorrectly acknowledged missing artifacts")
	}
}

func TestRecordReconcileSameInstanceSerializes(t *testing.T) {
	c, _, _, _, _ := newRecordReconcile(t)
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

func TestRecordReconcileOwnershipConflictsArePreflighted(t *testing.T) {
	for _, retired := range []bool{false, true} {
		for _, foreign := range []bool{false, true} {
			t.Run(fmt.Sprintf("retired%v/foreign%v", retired, foreign), func(t *testing.T) {
				c, source, store, db, pool := newRecordReconcile(t)
				ctx := context.Background()
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				ip := "127.0.0.1"
				if retired {
					ip = "127.0.0.2"
				}
				db.next++
				row := &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: pool.ID, ConfigType: model.ConfigTypeRecordRule, InstanceIP: ip, Name: "custom-preserved", ConfigContent: "keep"}
				if foreign {
					row.PoolID = 99
				}
				db.rows[row.ID] = row
				before := copyRecordReconcileRedis(store.values)
				beforeRows := copyRecordReconcileRows(db.rows)
				writes, dbWrites := store.writes, db.writes
				source.rows[0].Expr = "sum(up)"
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
func TestRecordReconcileRetiresCanonicalUnindexedOrphanAcrossPages(t *testing.T) {
	c, _, store, db, pool := newRecordReconcile(t)
	for i := 0; i < 110; i++ {
		db.next++
		db.rows[db.next] = &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 99, ConfigType: model.ConfigTypeRecordRule, InstanceIP: fmt.Sprintf("custom-%d", i), Name: "custom-preserved", ConfigContent: "keep"}
	}
	db.next++
	orphanID := db.next
	ip := "127.0.0.9"
	db.rows[orphanID] = &model.MonitorConfig{Model: model.Model{ID: orphanID}, PoolID: pool.ID, ConfigType: model.ConfigTypeRecordRule, InstanceIP: ip, Name: fmt.Sprintf(ConfigNameRecordRule, pool.ID, ip), ConfigContent: "old"}
	store.values[buildRedisKeyRecordRule(ip)] = "old"
	if err := c.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if db.rows[orphanID] != nil || store.values[buildRedisKeyRecordRule(ip)] != "" || len(db.rows) != 112 {
		t.Fatal("paged orphan retirement lost ownership boundaries")
	}
}
func TestRecordReconcileCancellationBeforePendingIsReadOnly(t *testing.T) {
	c, _, store, db, _ := newRecordReconcile(t)
	ctx, cancel := context.WithCancel(context.Background())
	store.onRead = func() { store.onRead = nil; cancel() }
	if !errors.Is(c.GenerateMainConfig(ctx), context.Canceled) {
		t.Fatal("cancellation was not propagated")
	}
	if len(store.values) != 0 || len(store.members) != 0 || len(db.rows) != 0 || db.writes != 0 {
		t.Fatal("cancellation published artifacts")
	}
}

func TestRecordReconcileEmptyTargets(t *testing.T) {
	for _, previous := range []bool{false, true} {
		t.Run(fmt.Sprintf("previous%v", previous), func(t *testing.T) {
			c, source, store, db, pool := newRecordReconcile(t)
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
			if len(db.rows) != 0 || len(store.members) != 0 || store.values[buildRedisKeyRecordRule("127.0.0.1")] != "" || store.values[buildRedisKeyRecordRule("127.0.0.2")] != "" || len(store.values[recordReconcileACK(pool.ID)]) != 64 {
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
func TestRecordReconcileEmptyTargetFailureRecovery(t *testing.T) {
	for _, stage := range []string{"pending", "del", "db delete", "srem", "ack"} {
		for _, target := range []string{"A", "B"} {
			for _, restart := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/restart%v", stage, target, restart), func(t *testing.T) {
					c, source, store, db, pool := newRecordReconcile(t)
					ctx := context.Background()
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					hashA := store.values[recordReconcileACK(pool.ID)]
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
						c = NewRecordRuleConfigCache(zap.NewNop(), c.scrapePoolDAO, source, db, NewBatchConfigManager(db, zap.NewNop()), store, db.projection.journal, db.projection.history, nil).(*recordRuleConfigCache)
						db.projection.attach(c)
					}
					if err := c.GenerateMainConfig(ctx); err != nil {
						t.Fatal(err)
					}
					marker := store.values[recordReconcileACK(pool.ID)]
					if len(marker) != 64 || strings.HasPrefix(marker, "pending:") {
						t.Fatal("recovery retained pending state")
					}
					if target == "A" {
						if marker != hashA || len(db.rows) != 2 || len(store.members) != 2 {
							t.Fatal("A was not restored after failed empty B")
						}
						for _, ip := range pool.PrometheusInstances {
							parseRecordReconcile(t, store.values[buildRedisKeyRecordRule(ip)])
						}
					} else if len(db.rows) != 0 || len(store.members) != 0 || store.values[buildRedisKeyRecordRule("127.0.0.1")] != "" || store.values[buildRedisKeyRecordRule("127.0.0.2")] != "" {
						t.Fatal("B retained retired artifacts")
					}
				})
			}
		}
	}
}
func TestRecordReconcileEmptyTargetsDoNotTurnReadFailureIntoRemoval(t *testing.T) {
	c, source, store, db, pool := newRecordReconcile(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	before, beforeRows := copyRecordReconcileRedis(store.values), copyRecordReconcileRows(db.rows)
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

type recordIncompleteMetadata struct {
	*recordReconcileDB
	mode  string
	first []*model.MonitorConfig
	owned []*model.MonitorConfig
}

func (d recordIncompleteMetadata) GetMonitorConfigList(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
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
func TestRecordReconcileIncompleteMetadataCannotAcknowledgeOrDuplicate(t *testing.T) {
	for _, mode := range []string{"negative total", "first empty", "total smaller than page", "later empty", "total drift", "duplicate ID", "final count mismatch"} {
		t.Run(mode, func(t *testing.T) {
			c, source, store, db, _ := newRecordReconcile(t)
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
				row := &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 7, ConfigType: model.ConfigTypeRecordRule, InstanceIP: fmt.Sprintf("unrelated-%d", i), Name: "custom-preserved", ConfigContent: "keep"}
				db.rows[row.ID] = row
				first = append(first, row)
			}
			c.configDAO = recordIncompleteMetadata{recordReconcileDB: db, mode: mode, first: first, owned: owned}
			before, beforeRows := copyRecordReconcileRedis(store.values), copyRecordReconcileRows(db.rows)
			writes, dbWrites := store.writes, db.writes
			source.rows[0].Expr = "sum(up)"
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("incomplete or drifting metadata was acknowledged")
			}
			if writes != store.writes || dbWrites != db.writes || !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(beforeRows, db.rows) {
				t.Fatal("rejected metadata changed pending/artifacts or created duplicate owners")
			}
		})
	}
}

func TestRecordReconcileSameACKRepairsMissingOrDriftingArtifacts(t *testing.T) {
	for _, mode := range []string{"cache missing", "cache drift", "member missing", "metadata content", "metadata hash", "metadata status"} {
		t.Run(mode, func(t *testing.T) {
			c, _, store, db, pool := newRecordReconcile(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			key := buildRedisKeyRecordRule("127.0.0.1")
			body, ack := store.values[key], store.values[recordReconcileACK(pool.ID)]
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
			if store.values[key] != body || !store.members["127.0.0.1"] || store.values[recordReconcileACK(pool.ID)] != ack {
				t.Fatal("same ACK incorrectly skipped artifact repair")
			}
			for _, saved := range db.rows {
				if saved.InstanceIP == "127.0.0.1" && (saved.ConfigContent != body || saved.ConfigHash != calculateConfigHash(body) || saved.Status != model.ConfigStatusActive) {
					t.Fatal("metadata drift was not repaired")
				}
			}
			parseRecordReconcile(t, store.values[key])
		})
	}
}
func TestRecordReconcileNoncanonicalNameRemainsAnUnownedConflict(t *testing.T) {
	c, _, store, db, _ := newRecordReconcile(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	for _, row := range db.rows {
		if row.InstanceIP == "127.0.0.1" {
			row.Name = "operator-custom-name"
		}
	}
	before, beforeRows := copyRecordReconcileRedis(store.values), copyRecordReconcileRows(db.rows)
	writes, dbWrites := store.writes, db.writes
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("noncanonical name was guessed to be generated")
	}
	if store.writes != writes || db.writes != dbWrites || !reflect.DeepEqual(before, store.values) || !reflect.DeepEqual(beforeRows, db.rows) {
		t.Fatal("ambiguous name was overwritten")
	}
}

func copyRecordReconcileRedis(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
func copyRecordReconcileRows(source map[int]*model.MonitorConfig) map[int]*model.MonitorConfig {
	copy := make(map[int]*model.MonitorConfig, len(source))
	for id, row := range source {
		value := *row
		copy[id] = &value
	}
	return copy
}

func TestRecordProjectionMemberOnlyLocatorCannotAuthorizeDeletion(t *testing.T) {
	for _, mode := range []string{"empty targets", "feature off"} {
		t.Run(mode, func(t *testing.T) {
			c, _, store, db, pool := newRecordReconcile(t)
			const ip = "127.0.0.9"
			store.members[ip] = true
			store.values[buildRedisKeyRecordRule(ip)] = "preserve unproved legacy bytes"
			pool.PrometheusInstances = nil
			if mode == "feature off" {
				pool.SupportRecord = 2
			}
			before := copyRecordReconcileRedis(store.values)
			if err := c.GenerateMainConfig(context.Background()); err == nil {
				t.Fatal("member without SQL owner/history was treated as delete authority")
			}
			if !reflect.DeepEqual(before, store.values) || !store.members[ip] || len(db.rows) != 0 {
				t.Fatal("unproved member cleanup changed raw data or metadata")
			}
			var operations int64
			if db.projection.db.Model(&model.MonitorConfigPoolOperation{}).Count(&operations).Error != nil || operations != 0 {
				t.Fatal("unproved member acquired a committed publication authority")
			}
		})
	}
}

func TestRecordProjectionHistoricalTombstoneRequiresUnambiguousOwner(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprintf("foreign-history-%v", foreign), func(t *testing.T) {
			c, _, store, db, pool := newRecordReconcile(t)
			const ip = "127.0.0.9"
			pool.SupportRecord = 2
			pool.PrometheusInstances = nil
			owned := &model.MonitorConfig{Model: model.Model{ID: 91, DeletedAt: 19}, PoolID: pool.ID, ConfigType: 4, InstanceIP: ip, Name: fmt.Sprintf(ConfigNameRecordRule, pool.ID, ip), ConfigContent: "historical synthetic", Status: 2}
			if db.projection.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(owned).Error != nil {
				t.Fatal("real historical tombstone fixture failed")
			}
			if foreign {
				row := *owned
				row.ID = 92
				row.PoolID = 99
				row.Name = fmt.Sprintf(ConfigNameRecordRule, 99, ip)
				if db.projection.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(&row).Error != nil {
					t.Fatal("foreign historical tombstone fixture failed")
				}
			}
			store.members[ip] = true
			store.values[buildRedisKeyRecordRule(ip)] = "historical synthetic"
			before := copyRecordReconcileRedis(store.values)
			err := c.GenerateMainConfig(context.Background())
			if foreign {
				if err == nil || !reflect.DeepEqual(before, store.values) || !store.members[ip] {
					t.Fatal("ambiguous cross-pool history was guessed into cleanup authority")
				}
				var operations int64
				if db.projection.db.Model(&model.MonitorConfigPoolOperation{}).Count(&operations).Error != nil || operations != 0 {
					t.Fatal("ambiguous history committed a cleanup plan")
				}
			} else {
				if err != nil || store.members[ip] || store.values[buildRedisKeyRecordRule(ip)] != "" {
					t.Fatal("proven historical owner did not complete retirement", err)
				}
				receipt, err := c.journal.LoadLastProjectedProjection(context.Background(), 4, pool.ID)
				if err != nil || receipt == nil || receipt.Phase != model.ConfigPublicationProjected {
					t.Fatal("historical retirement lacks its actual projected SQL receipt")
				}
			}
			var saved model.MonitorConfig
			if db.projection.db.Unscoped().First(&saved, owned.ID).Error != nil || saved.DeletedAt != owned.DeletedAt || saved.Name != owned.Name {
				t.Fatal("historical owner audit row was changed or removed")
			}
		})
	}
}

func TestRecordProjectionDiscoversUnindexedHistoricalArtifacts(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprintf("foreign-history-%v", foreign), func(t *testing.T) {
			c, source, store, db, pool := newRecordReconcile(t)
			const ip = "127.0.0.19"
			pool.SupportRecord = 2
			pool.PrometheusInstances = nil
			source.failure = true
			owned := &model.MonitorConfig{Model: model.Model{ID: 191, DeletedAt: 29}, PoolID: pool.ID, ConfigType: 4, InstanceIP: ip, Name: fmt.Sprintf(ConfigNameRecordRule, pool.ID, ip), ConfigContent: "unindexed historical synthetic", Status: 2}
			if db.projection.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(owned).Error != nil {
				t.Fatal("unindexed tombstone fixture failed")
			}
			if foreign {
				row := *owned
				row.ID = 192
				row.PoolID = 99
				row.Name = "custom historical owner"
				if db.projection.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(&row).Error != nil {
					t.Fatal("foreign unindexed history fixture failed")
				}
			}
			store.values[buildRedisKeyRecordRule(ip)] = "unindexed historical synthetic"
			if len(store.members) != 0 {
				t.Fatal("fixture unexpectedly has a member locator")
			}
			prior, err := c.journal.LoadLastCommittedProjection(context.Background(), 4, pool.ID)
			if err != nil || prior != nil {
				t.Fatal("fixture unexpectedly has a durable prior manifest")
			}
			before := copyRecordReconcileRedis(store.values)
			err = c.GenerateMainConfig(context.Background())
			if foreign {
				if err == nil || !reflect.DeepEqual(before, store.values) {
					t.Fatal("unindexed foreign history was acknowledged or modified")
				}
				var operations int64
				if db.projection.db.Model(&model.MonitorConfigPoolOperation{}).Count(&operations).Error != nil || operations != 0 {
					t.Fatal("ambiguous unindexed history acquired publication authority")
				}
			} else {
				if err != nil || store.values[buildRedisKeyRecordRule(ip)] != "" || len(store.members) != 0 {
					t.Fatal("unindexed canonical tombstone was not discovered and retired", err)
				}
				receipt, err := c.journal.LoadLastProjectedProjection(context.Background(), 4, pool.ID)
				if err != nil || receipt == nil || len(receipt.Artifacts) != 1 || receipt.Artifacts[0].InstanceKey != ip || receipt.Artifacts[0].Action != model.ConfigProjectionRetire {
					t.Fatal("unindexed retirement lacks its exact projected SQL artifact receipt")
				}
			}
			var audit model.MonitorConfig
			if db.projection.db.Unscoped().First(&audit, owned.ID).Error != nil || audit.DeletedAt != owned.DeletedAt || audit.Name != owned.Name {
				t.Fatal("unindexed historical audit row changed")
			}
			if source.calls != 0 {
				t.Fatal("explicit off consulted broken record definitions")
			}
		})
	}
}

func TestRecordProjectionCommitRechecksHistoryAfterForeignABA(t *testing.T) {
	for _, state := range []string{"historical tombstone", "active owner deleted after preflight"} {
		t.Run(state, func(t *testing.T) {
			c, _, store, db, pool := newRecordReconcile(t)
			ctx := context.Background()
			const ip = "127.0.0.1"
			var owned model.MonitorConfig
			if state == "historical tombstone" {
				owned = model.MonitorConfig{Model: model.Model{ID: 91, DeletedAt: 19}, PoolID: pool.ID, ConfigType: 4, InstanceIP: ip, Name: fmt.Sprintf(ConfigNameRecordRule, pool.ID, ip), ConfigContent: "original synthetic", Status: 2}
				if db.projection.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Create(&owned).Error != nil {
					t.Fatal("real tombstone fixture failed")
				}
				store.members[ip] = true
				store.values[buildRedisKeyRecordRule(ip)] = "original synthetic"
			} else {
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				for _, row := range db.rows {
					if row.InstanceIP == ip {
						owned = *row
					}
				}
				if owned.ID <= 0 {
					t.Fatal("active SQL owner missing")
				}
			}
			oldSHA := store.values[recordPublicationKey(pool.ID)]
			oldReceipt := store.values[projectionPoolKey(4, pool.ID, "ack")]
			prior, err := c.journal.LoadLastCommittedProjection(ctx, 4, pool.ID)
			if err != nil {
				t.Fatal(err)
			}
			pool.SupportRecord = 2
			pool.PrometheusInstances = nil
			called := false
			hook := &recordFixtureCommitHook{RecordProjectionJournalDAO: db.projection.journal}
			hook.before = func() {
				if called {
					t.Fatal("typed commit interleaving repeated")
				}
				called = true
				if state == "active owner deleted after preflight" {
					if err := db.projection.MonitorConfigDAO.DeleteMonitorConfig(ctx, owned.ID); err != nil {
						t.Fatal("ordinary owner deletion failed", err)
					}
				}
				foreign := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: ip, Name: "foreign transient owner", ConfigContent: "foreign transient synthetic", Status: 2}
				if err := db.projection.MonitorConfigDAO.CreateMonitorConfig(ctx, foreign); err != nil {
					t.Fatal("ordinary foreign create failed", err)
				}
				if err := db.projection.MonitorConfigDAO.DeleteMonitorConfig(ctx, foreign.ID); err != nil {
					t.Fatal("ordinary foreign delete failed", err)
				}
				store.values[buildRedisKeyRecordRule(ip)] = "foreign transient synthetic"
			}
			c.journal = hook
			publishErr := c.GenerateMainConfig(ctx)
			if !called {
				t.Fatal("interleaving did not reach actual typed commit boundary")
			}
			if publishErr == nil {
				t.Fatal("preflight proof survived foreign create-delete ABA")
			}
			if store.values[buildRedisKeyRecordRule(ip)] != "foreign transient synthetic" || store.values[recordPublicationKey(pool.ID)] != oldSHA || store.values[projectionPoolKey(4, pool.ID, "ack")] != oldReceipt {
				t.Fatal("rejected history proof changed raw data or acknowledged a new projection")
			}
			current, err := c.journal.LoadLastCommittedProjection(ctx, 4, pool.ID)
			if err != nil {
				t.Fatal(err)
			}
			if prior == nil {
				if current != nil {
					t.Fatal("failed history guard committed a new manifest")
				}
			} else if current == nil || current.Reservation.OperationID != prior.Reservation.OperationID {
				t.Fatal("failed history guard replaced the last-good committed operation")
			}
			var history []*model.MonitorConfig
			if db.projection.db.Unscoped().Where("config_type=4 AND instance_ip=?", ip).Find(&history).Error != nil || len(history) != 2 {
				t.Fatal("ABA audit history was removed")
			}
			for _, row := range history {
				if row.DeletedAt == 0 {
					t.Fatal("foreign ABA did not leave a vacant namespace fixture")
				}
			}
		})
	}
}
