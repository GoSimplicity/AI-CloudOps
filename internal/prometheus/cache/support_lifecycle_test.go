package cache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type supportInventory struct {
	scrapedao.ScrapePoolDAO
	pool *model.MonitorScrapePool
	mode string
}

func (s *supportInventory) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.SupportAlert != nil || req.SupportRecord != nil {
		return nil, 0, errors.New("inventory must not be support-filtered")
	}
	if s.mode == "read failure" {
		return nil, 0, errors.New("fixture inventory failure")
	}
	if s.mode == "missing pool" {
		return nil, 0, nil
	}
	if req.Page > 1 {
		return nil, 1, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

// The old filtered APIs are implemented only for the HEAD red comparison.
func (s *supportInventory) GetMonitorScrapePoolSupportedAlert(context.Context) ([]*model.MonitorScrapePool, int64, error) {
	if s.pool.SupportAlert == 1 {
		return []*model.MonitorScrapePool{s.pool}, 1, nil
	}
	return nil, 0, nil
}
func (s *supportInventory) GetMonitorScrapePoolSupportedRecord(context.Context) ([]*model.MonitorScrapePool, int64, error) {
	if s.pool.SupportRecord == 1 {
		return []*model.MonitorScrapePool{s.pool}, 1, nil
	}
	return nil, 0, nil
}

type supportRedis struct {
	redis.Cmdable
	values  map[string]string
	sets    map[string]map[string]bool
	failure string
	writes  int
}

func (s *supportRedis) Get(_ context.Context, key string) *redis.StringCmd {
	v, ok := s.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (s *supportRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	s.writes++
	text := value.(string)
	marker := strings.Contains(key, ":publication:")
	if marker && (s.failure == "pending" && strings.HasPrefix(text, "pending:") || s.failure == "ack" && !strings.HasPrefix(text, "pending:")) {
		return redis.NewStatusResult("", errors.New("fixture marker failure"))
	}
	s.values[key] = text
	return redis.NewStatusResult("OK", nil)
}
func (s *supportRedis) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	var ips []string
	for ip := range s.sets[key] {
		ips = append(ips, ip)
	}
	return redis.NewStringSliceResult(ips, nil)
}
func (s *supportRedis) SAdd(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	s.writes++
	if s.sets[key] == nil {
		s.sets[key] = map[string]bool{}
	}
	for _, v := range values {
		s.sets[key][v.(string)] = true
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *supportRedis) SRem(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	s.writes++
	if s.failure == "srem" {
		return redis.NewIntResult(0, errors.New("fixture membership failure"))
	}
	for _, v := range values {
		delete(s.sets[key], v.(string))
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *supportRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	s.writes++
	if s.failure == "del" {
		return redis.NewIntResult(0, errors.New("fixture delete failure"))
	}
	for _, key := range keys {
		delete(s.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

type supportFixture struct {
	pool         *model.MonitorScrapePool
	inventory    *supportInventory
	store        *supportRedis
	db           *alertRulePubDB
	ruleSource   *alertRulePubSource
	recordSource *recordReconcileSource
	rule         *alertRuleConfigCache
	record       *recordRuleConfigCache
}

func newSupportFixture() *supportFixture {
	f := &supportFixture{pool: &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "support-A", SupportAlert: 1, SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.1", "127.0.0.2"}}, store: &supportRedis{values: map[string]string{}, sets: map[string]map[string]bool{}}, db: &alertRulePubDB{rows: map[int]*model.MonitorConfig{}}, ruleSource: &alertRulePubSource{rows: []*model.MonitorAlertRule{alertingContractRow(31)}}, recordSource: &recordReconcileSource{rows: []*model.MonitorRecordRule{recordReconcileRow()}}}
	f.inventory = &supportInventory{pool: f.pool}
	f.rebuild()
	return f
}
func (f *supportFixture) rebuild() {
	logger := zap.NewNop()
	f.rule = NewAlertRuleConfigCache(logger, f.inventory, f.ruleSource, f.db, NewBatchConfigManager(f.db, logger), f.store).(*alertRuleConfigCache)
	f.record = NewRecordRuleConfigCache(logger, f.inventory, f.recordSource, f.db, NewBatchConfigManager(f.db, logger), f.store).(*recordRuleConfigCache)
}
func (f *supportFixture) publish(kind string) error {
	if kind == "rule" {
		return f.rule.GenerateMainConfig(context.Background())
	}
	return f.record.GenerateMainConfig(context.Background())
}
func (f *supportFixture) setFlag(kind string, value int8) {
	if kind == "rule" {
		f.pool.SupportAlert = value
	} else {
		f.pool.SupportRecord = value
	}
}
func (f *supportFixture) ack(kind string) string {
	if kind == "rule" {
		return alertRulePublicationKey(f.pool.ID)
	}
	return recordPublicationKey(f.pool.ID)
}
func (f *supportFixture) memberKey(kind string) string {
	if kind == "rule" {
		return buildRedisSetKeyAlertRulePoolIPs(f.pool.ID)
	}
	return buildRedisSetKeyRecordRulePoolIPs(f.pool.ID)
}
func supportKind(kind string) int8 {
	if kind == "rule" {
		return model.ConfigTypeAlertRule
	}
	return model.ConfigTypeRecordRule
}
func supportArtifact(kind, ip string) string {
	if kind == "rule" {
		return buildRedisKeyAlertRule(ip)
	}
	return buildRedisKeyRecordRule(ip)
}
func (f *supportFixture) sourceCalls(kind string) int {
	if kind == "rule" {
		return f.ruleSource.calls
	}
	return f.recordSource.calls
}
func (f *supportFixture) assertRetired(t *testing.T, kind string) {
	t.Helper()
	if len(f.store.sets[f.memberKey(kind)]) != 0 {
		t.Fatal("closed feature retains membership")
	}
	for _, ip := range []string{"127.0.0.1", "127.0.0.2"} {
		if f.store.values[supportArtifact(kind, ip)] != "" {
			t.Fatal("closed feature retains an artifact")
		}
	}
	for _, row := range f.db.rows {
		if row.PoolID == f.pool.ID && row.ConfigType == supportKind(kind) {
			t.Fatal("closed feature retains owned metadata")
		}
	}
}
func copySupportSets(source map[string]map[string]bool) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for key, set := range source {
		out[key] = map[string]bool{}
		for value, exists := range set {
			out[key][value] = exists
		}
	}
	return out
}
func TestSupportLifecycleOnOffOnIsFeatureScoped(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		t.Run(kind, func(t *testing.T) {
			f := newSupportFixture()
			if err := f.publish("rule"); err != nil {
				t.Fatal(err)
			}
			if err := f.publish("record"); err != nil {
				t.Fatal(err)
			}
			f.store.values[buildRedisKeyPrometheusMain("127.0.0.1")] = "prometheus-untouched"
			other := "record"
			if kind == "record" {
				other = "rule"
			}
			f.db.next++
			f.db.rows[f.db.next] = &model.MonitorConfig{Model: model.Model{ID: f.db.next}, PoolID: f.pool.ID, ConfigType: model.ConfigTypePrometheus, InstanceIP: "127.0.0.1", Name: fmt.Sprintf(ConfigNamePrometheus, f.pool.ID, "127.0.0.1"), ConfigContent: "prometheus-untouched"}
			otherRows := map[int]*model.MonitorConfig{}
			for id, row := range f.db.rows {
				if row.ConfigType != supportKind(kind) {
					copy := *row
					otherRows[id] = &copy
				}
			}
			otherMembers := copySupportSets(f.store.sets)[f.memberKey(other)]
			otherACK := f.store.values[f.ack(other)]
			otherBody := f.store.values[supportArtifact(other, "127.0.0.1")]
			initial := f.store.values[f.ack(kind)]
			calls := f.sourceCalls(kind)
			f.setFlag(kind, 2)
			if err := f.publish(kind); err != nil {
				t.Fatal(err)
			}
			f.assertRetired(t, kind)
			if f.sourceCalls(kind) != calls || f.store.values[f.ack(kind)] == initial {
				t.Fatal("off read rules or omitted the explicit state fingerprint")
			}
			if f.store.values[f.ack(other)] != otherACK || f.store.values[supportArtifact(other, "127.0.0.1")] != otherBody || f.store.values[buildRedisKeyPrometheusMain("127.0.0.1")] != "prometheus-untouched" {
				t.Fatal("closing one feature touched another publisher")
			}
			for id, before := range otherRows {
				if !reflect.DeepEqual(before, f.db.rows[id]) {
					t.Fatal("closing one feature touched another type's metadata")
				}
			}
			if !reflect.DeepEqual(otherMembers, f.store.sets[f.memberKey(other)]) {
				t.Fatal("closing one feature changed another membership index")
			}
			f.setFlag(kind, 1)
			if err := f.publish(kind); err != nil {
				t.Fatal(err)
			}
			if f.store.values[f.ack(kind)] != initial || f.store.values[supportArtifact(kind, "127.0.0.1")] == "" {
				t.Fatal("on state was not restored")
			}
		})
	}
}
func TestSupportLifecycleOffBypassesBadRulesAndUntrustedCurrentTargets(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		for _, mode := range []string{"DAO failure", "bad rows"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				f := newSupportFixture()
				if err := f.publish(kind); err != nil {
					t.Fatal(err)
				}
				calls := f.sourceCalls(kind)
				f.setFlag(kind, 2)
				f.pool.PrometheusInstances = model.StringList{"not-an-ip", "another-untrusted-target"}
				if mode == "DAO failure" {
					f.ruleSource.failure = true
					f.recordSource.failure = true
				} else {
					f.ruleSource.rows[0].ForTime = "later"
					f.recordSource.rows[0].Expr = "up[5m]"
				}
				if err := f.publish(kind); err != nil {
					t.Fatal(err)
				}
				f.assertRetired(t, kind)
				if f.sourceCalls(kind) != calls {
					t.Fatal("explicit off consulted disabled rule contents")
				}
				for key := range f.store.values {
					if strings.Contains(key, "not-an-ip") {
						t.Fatal("untrusted disabled targets formed a key")
					}
				}
			})
		}
	}
}
func TestSupportLifecycleUnknownFlagsAreReadOnly(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		for _, flag := range []int8{0, -1, 3, 127} {
			t.Run(fmt.Sprintf("%s/%d", kind, flag), func(t *testing.T) {
				f := newSupportFixture()
				if err := f.publish(kind); err != nil {
					t.Fatal(err)
				}
				before, rows, sets := copyAlertRedis(f.store.values), copyAlertRows(f.db.rows), copySupportSets(f.store.sets)
				writes, dbWrites, calls := f.store.writes, f.db.writes, f.sourceCalls(kind)
				f.setFlag(kind, flag)
				if f.publish(kind) == nil {
					t.Fatal("unknown feature flag was acknowledged")
				}
				if f.store.writes != writes || f.db.writes != dbWrites || f.sourceCalls(kind) != calls || !reflect.DeepEqual(before, f.store.values) || !reflect.DeepEqual(rows, f.db.rows) || !reflect.DeepEqual(sets, f.store.sets) {
					t.Fatal("unknown flag mutated the last-good state")
				}
			})
		}
	}
}
func TestSupportLifecycleOffFailureReturnsToOnOrCompletesOff(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		for _, stage := range []string{"pending", "del", "db delete", "srem", "ack"} {
			for _, target := range []string{"on", "off"} {
				for _, restart := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/restart%v", kind, stage, target, restart), func(t *testing.T) {
						f := newSupportFixture()
						if err := f.publish(kind); err != nil {
							t.Fatal(err)
						}
						initial := f.store.values[f.ack(kind)]
						name := f.pool.Name
						f.setFlag(kind, 2)
						f.pool.Name = "renamed-off-B"
						f.pool.PrometheusInstances = model.StringList{"invalid-disabled-target"}
						f.ruleSource.failure = true
						f.recordSource.failure = true
						f.store.failure, f.db.failure = stage, stage
						if f.publish(kind) == nil {
							t.Fatal("failed off publication reported success")
						}
						f.store.failure, f.db.failure = "", ""
						if target == "on" {
							f.setFlag(kind, 1)
							f.pool.Name = name
							f.pool.PrometheusInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
							f.ruleSource.failure = false
							f.recordSource.failure = false
						}
						if restart {
							f.rebuild()
						}
						if err := f.publish(kind); err != nil {
							t.Fatal(err)
						}
						marker := f.store.values[f.ack(kind)]
						if len(marker) != 64 || strings.HasPrefix(marker, "pending:") {
							t.Fatal("recovery did not acknowledge a complete state")
						}
						if target == "off" {
							f.assertRetired(t, kind)
						} else if marker != initial || f.store.values[supportArtifact(kind, "127.0.0.1")] == "" {
							t.Fatal("failed off prevented restoring on")
						}
					})
				}
			}
		}
	}
}
func TestSupportLifecyclePendingFailureAndOwnershipConflictDoNotWrite(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		for _, state := range []int8{1, 2} {
			for _, mode := range []string{"pending", "custom rename", "foreign mixed"} {
				t.Run(fmt.Sprintf("%s/state%d/%s", kind, state, mode), func(t *testing.T) {
					f := newSupportFixture()
					if err := f.publish(kind); err != nil {
						t.Fatal(err)
					}
					if mode == "custom rename" {
						for _, row := range f.db.rows {
							if row.ConfigType == supportKind(kind) {
								row.Name = "operator-custom-name"
							}
						}
					}
					if mode == "foreign mixed" {
						f.db.next++
						f.db.rows[f.db.next] = &model.MonitorConfig{Model: model.Model{ID: f.db.next}, PoolID: 99, ConfigType: supportKind(kind), InstanceIP: "127.0.0.1", Name: "foreign", ConfigContent: "keep"}
					}
					before, rows, sets := copyAlertRedis(f.store.values), copyAlertRows(f.db.rows), copySupportSets(f.store.sets)
					writes, dbWrites := f.store.writes, f.db.writes
					f.setFlag(kind, state)
					if state == 1 {
						f.ruleSource.rows[0].Expr = "sum(up)"
						f.recordSource.rows[0].Expr = "sum(up)"
					}
					if mode == "pending" {
						f.store.failure = "pending"
					}
					if f.publish(kind) == nil {
						t.Fatal("failed precondition acknowledged publication")
					}
					extra := 0
					if mode == "pending" {
						extra = 1
					}
					if f.store.writes != writes+extra || f.db.writes != dbWrites || !reflect.DeepEqual(before, f.store.values) || !reflect.DeepEqual(rows, f.db.rows) || !reflect.DeepEqual(sets, f.store.sets) {
						t.Fatal("precondition failure changed an artifact or acknowledged state")
					}
				})
			}
		}
	}
}
func TestSupportLifecycleReadFailureAndMissingPoolAreNotOff(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		for _, mode := range []string{"read failure", "missing pool"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				f := newSupportFixture()
				if err := f.publish(kind); err != nil {
					t.Fatal(err)
				}
				before, rows := copyAlertRedis(f.store.values), copyAlertRows(f.db.rows)
				writes, dbWrites := f.store.writes, f.db.writes
				f.inventory.mode = mode
				err := f.publish(kind)
				if mode == "read failure" && err == nil {
					t.Fatal("inventory error ignored")
				}
				if !reflect.DeepEqual(before, f.store.values) || !reflect.DeepEqual(rows, f.db.rows) || f.store.writes != writes || f.db.writes != dbWrites {
					t.Fatal("absence or a failed inventory was treated as explicit off")
				}
			})
		}
	}
}
func TestSupportLifecycleEmptyOnAndOffHaveDistinctVersions(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		t.Run(kind, func(t *testing.T) {
			f := newSupportFixture()
			f.pool.PrometheusInstances = nil
			f.ruleSource.rows = nil
			f.recordSource.rows = nil
			if err := f.publish(kind); err != nil {
				t.Fatal(err)
			}
			on := f.store.values[f.ack(kind)]
			f.setFlag(kind, 2)
			if err := f.publish(kind); err != nil {
				t.Fatal(err)
			}
			if len(on) != 64 || f.store.values[f.ack(kind)] == on {
				t.Fatal("empty on and explicit off have the same identity")
			}
		})
	}
}

type supportIncompleteInventory struct {
	scrapedao.ScrapePoolDAO
	owned *model.MonitorScrapePool
	mode  string
}

func (s supportIncompleteInventory) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.Page == 1 {
		rows := make([]*model.MonitorScrapePool, 100)
		for i := range rows {
			rows[i] = &model.MonitorScrapePool{Model: model.Model{ID: i + 1}, Name: fmt.Sprintf("pool-%d", i+1), SupportAlert: 2, SupportRecord: 2}
		}
		rows[s.owned.ID-1] = s.owned
		return rows, 101, nil
	}
	if s.mode == "read error" {
		return nil, 0, errors.New("fixture later inventory failure")
	}
	if s.mode == "early empty" {
		return nil, 101, nil
	}
	id := 101
	if s.mode == "duplicate ID" {
		id = s.owned.ID
	}
	total := int64(101)
	if s.mode == "total drift" {
		total = 102
	}
	return []*model.MonitorScrapePool{{Model: model.Model{ID: id}, SupportAlert: 2, SupportRecord: 2}}, total, nil
}
func TestSupportLifecycleWholeInventoryPrecedesAnyOffWrite(t *testing.T) {
	for _, kind := range []string{"rule", "record"} {
		for _, mode := range []string{"read error", "early empty", "duplicate ID", "total drift"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				f := newSupportFixture()
				if err := f.publish(kind); err != nil {
					t.Fatal(err)
				}
				f.setFlag(kind, 2)
				inventory := supportIncompleteInventory{owned: f.pool, mode: mode}
				f.rule.scrapePoolDAO = inventory
				f.record.scrapePoolDAO = inventory
				before, rows, sets := copyAlertRedis(f.store.values), copyAlertRows(f.db.rows), copySupportSets(f.store.sets)
				writes, dbWrites, calls := f.store.writes, f.db.writes, f.sourceCalls(kind)
				if f.publish(kind) == nil {
					t.Fatal("incomplete inventory authorized closing its first page")
				}
				if f.store.writes != writes || f.db.writes != dbWrites || f.sourceCalls(kind) != calls || !reflect.DeepEqual(before, f.store.values) || !reflect.DeepEqual(rows, f.db.rows) || !reflect.DeepEqual(sets, f.store.sets) {
					t.Fatal("inventory failed after a partial off publication")
				}
			})
		}
	}
}

type supportReusedInventory struct {
	scrapedao.ScrapePoolDAO
	first []*model.MonitorScrapePool
}

func (s *supportReusedInventory) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.Page == 1 {
		s.first = make([]*model.MonitorScrapePool, 100)
		for i := range s.first {
			s.first[i] = &model.MonitorScrapePool{Model: model.Model{ID: i + 1}, Name: fmt.Sprintf("before-%d", i+1), SupportAlert: 1, SupportRecord: 2, PrometheusInstances: model.StringList{"192.0.2.1"}, Tags: model.StringList{"page=before"}}
		}
		return s.first, 101, nil
	}
	reused := s.first[0]
	reused.ID = 101
	reused.Name = "later"
	reused.SupportAlert = 2
	reused.SupportRecord = 1
	reused.PrometheusInstances[0] = "192.0.2.9"
	reused.Tags[0] = "page=later"
	return []*model.MonitorScrapePool{reused}, 101, nil
}
func TestSupportLifecycleInventoryCopiesPagesBeforeReuse(t *testing.T) {
	rows, err := readScrapePoolSnapshot(context.Background(), &supportReusedInventory{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 101 || rows[0].ID != 1 || rows[0].Name != "before-1" || rows[0].SupportAlert != 1 || rows[0].SupportRecord != 2 || rows[0].PrometheusInstances[0] != "192.0.2.1" || rows[0].Tags[0] != "page=before" || rows[100].ID != 101 {
		t.Fatal("later pages mutated the already collected inventory")
	}
}
