package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapeDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	common "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
)

type recordingContractRules struct {
	alertDAO.AlertManagerRecordDAO
	rows  []*model.MonitorRecordRule
	err   error
	calls int
}

func (s *recordingContractRules) GetMonitorRecordRuleByPoolId(context.Context, int) ([]*model.MonitorRecordRule, int64, error) {
	s.calls++
	return s.rows, int64(len(s.rows)), s.err
}

type recordingContractPools struct {
	scrapeDAO.ScrapePoolDAO
	pool *model.MonitorScrapePool
}

func (s recordingContractPools) GetMonitorScrapePoolSupportedRecord(context.Context) ([]*model.MonitorScrapePool, int64, error) {
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

func (s recordingContractPools) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.Page > 1 {
		return nil, 1, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

func newRecordingContractCache(rows []*model.MonitorRecordRule) (*recordRuleConfigCache, *recordingContractRules, *refreshRedis, *model.MonitorScrapePool) {
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "recording-fixture", SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.1"}}
	rules := &recordingContractRules{rows: rows}
	storage := &refreshRedis{values: map[string]string{}}
	logger := zap.NewNop()
	cache := &recordRuleConfigCache{
		logger: logger, redis: &recordingPublicationRedis{refreshRedis: storage, members: map[string]bool{}}, alertRecordDAO: rules,
		scrapePoolDAO: recordingContractPools{pool: pool},
		batchManager:  NewBatchConfigManager(&recordingPublicationConfigs{rows: map[string]*model.MonitorConfig{}}, logger),
	}
	return cache, rules, storage, pool
}

func parseRecordingContract(t *testing.T, content string) []rulefmt.Rule {
	t.Helper()
	if content == "" {
		t.Fatal("expected generated recording configuration")
	}
	groups, errs := rulefmt.Parse([]byte(content), false, common.UTF8Validation, parser.NewParser(parser.Options{}))
	if len(errs) != 0 {
		t.Fatalf("generated configuration rejected by Prometheus: %v", errs)
	}
	var rules []rulefmt.Rule
	for _, group := range groups.Groups {
		rules = append(rules, group.Rules...)
	}
	return rules
}

func TestRecordingContractValidNamesExpressionsAndLabels(t *testing.T) {
	for _, test := range []struct{ name, expression string }{
		{"服务:请求总量", "sum(up)"},
		{"bad-name", "up"},
		{"record:scalar", "1 + 2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cache, _, _, pool := newRecordingContractCache([]*model.MonitorRecordRule{{
				Name: test.name, Expr: test.expression, Enable: 1,
				Labels: model.StringList{"team=ops", "url=a=b", "empty="},
			}})
			rules := parseRecordingContract(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
			if len(rules) != 1 || rules[0].Record != test.name || rules[0].Expr != test.expression {
				t.Fatalf("recording rule changed: %+v", rules)
			}
			want := map[string]string{"team": "ops", "url": "a=b", "empty": ""}
			if !reflect.DeepEqual(rules[0].Labels, want) {
				t.Fatalf("labels = %#v, want %#v", rules[0].Labels, want)
			}
		})
	}
}

func TestRecordingContractBadRowDoesNotPoisonPool(t *testing.T) {
	for _, test := range []struct {
		name string
		bad  *model.MonitorRecordRule
	}{
		{"name", &model.MonitorRecordRule{Name: "bad{rule}", Expr: "up", Enable: 1}},
		{"syntax", &model.MonitorRecordRule{Name: "bad:syntax", Expr: "sum(", Enable: 1}},
		{"string", &model.MonitorRecordRule{Name: "bad:string", Expr: `"hello"`, Enable: 1}},
		{"matrix", &model.MonitorRecordRule{Name: "bad:matrix", Expr: "up[5m]", Enable: 1}},
		{"reserved label", &model.MonitorRecordRule{Name: "bad:label", Expr: "up", Enable: 1, Labels: model.StringList{"__name__=override"}}},
		{"missing label separator", &model.MonitorRecordRule{Name: "bad:label", Expr: "up", Enable: 1, Labels: model.StringList{"team"}}},
		{"empty label name", &model.MonitorRecordRule{Name: "bad:label", Expr: "up", Enable: 1, Labels: model.StringList{"=ops"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cache, _, _, pool := newRecordingContractCache([]*model.MonitorRecordRule{
				{Name: "good:up", Expr: "up", Enable: 1, Labels: model.StringList{"team=ops"}}, test.bad,
			})
			rules := parseRecordingContract(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
			if len(rules) != 1 || rules[0].Record != "good:up" || rules[0].Labels["team"] != "ops" {
				t.Fatalf("bad row affected valid recording rule: %+v", rules)
			}
		})
	}
}

func TestRecordingContractInvalidSnapshotPreservesLastGood(t *testing.T) {
	for _, test := range []struct {
		name string
		rows []*model.MonitorRecordRule
		err  error
	}{
		{"invalid name", []*model.MonitorRecordRule{{Name: "bad{rule}", Expr: "up", Enable: 1}}, nil},
		{"invalid expression", []*model.MonitorRecordRule{{Name: "bad:syntax", Expr: "sum(", Enable: 1}}, nil},
		{"invalid label", []*model.MonitorRecordRule{{Name: "bad:label", Expr: "up", Enable: 1, Labels: model.StringList{"__name__=override"}}}, nil},
		{"blank fields", []*model.MonitorRecordRule{{Name: " ", Expr: " ", Enable: 1}}, nil},
		{"string expression", []*model.MonitorRecordRule{{Name: "bad:string", Expr: `"hello"`, Enable: 1}}, nil},
		{"matrix expression", []*model.MonitorRecordRule{{Name: "bad:matrix", Expr: "up[5m]", Enable: 1}}, nil},
		{"DAO error", nil, errors.New("fixture read failure")},
	} {
		t.Run(test.name, func(t *testing.T) {
			cache, source, storage, pool := newRecordingContractCache([]*model.MonitorRecordRule{{Name: "good:up", Expr: "up", Enable: 1}})
			if err := cache.GenerateMainConfig(context.Background()); err != nil {
				t.Fatal(err)
			}
			configKey, hashKey := buildRedisKeyRecordRule("127.0.0.1"), recordPublicationKey(pool.ID)
			previousConfig, previousHash, previousWrites := storage.values[configKey], storage.values[hashKey], storage.writes
			parseRecordingContract(t, previousConfig)
			source.rows, source.err = test.rows, test.err
			previousReads := source.calls
			// Failure may be returned or logged; neither may publish an invalid snapshot.
			_ = cache.GenerateMainConfig(context.Background())
			if storage.values[configKey] != previousConfig || storage.values[hashKey] != previousHash || storage.writes != previousWrites {
				t.Fatal("invalid snapshot overwrote last-good configuration or advanced its hash")
			}
			if source.calls-previousReads != 1 {
				t.Fatal("failed snapshot was read more than once")
			}
		})
	}
}

func TestRecordingContractEmptyActiveListClearsRules(t *testing.T) {
	cache, source, storage, pool := newRecordingContractCache([]*model.MonitorRecordRule{{Name: "good:up", Expr: "up", Enable: 1}})
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	previousHash := storage.values[recordPublicationKey(pool.ID)]
	source.rows = nil
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rules := parseRecordingContract(t, storage.values[buildRedisKeyRecordRule("127.0.0.1")]); len(rules) != 0 {
		t.Fatalf("empty active list retained rules: %+v", rules)
	}
	groups, errs := rulefmt.Parse([]byte(storage.values[buildRedisKeyRecordRule("127.0.0.1")]), false, common.UTF8Validation, parser.NewParser(parser.Options{}))
	if len(errs) != 0 || groups == nil || groups.Groups == nil || len(groups.Groups) != 0 {
		t.Fatal("empty active list must publish an explicit groups array")
	}
	if storage.values[recordPublicationKey(pool.ID)] == previousHash {
		t.Fatal("empty active list did not advance the published snapshot hash")
	}
}

func TestRecordingContractLabelsOnlyRefreshAndUnchangedNoWrite(t *testing.T) {
	cache, source, storage, pool := newRecordingContractCache([]*model.MonitorRecordRule{{Name: "good:up", Expr: "up", Enable: 1, Labels: model.StringList{"team=before"}}})
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	oldHash, oldWrites := storage.values[recordPublicationKey(pool.ID)], storage.writes
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if storage.writes != oldWrites {
		t.Fatal("unchanged recording snapshot rewrote Redis")
	}
	source.rows = []*model.MonitorRecordRule{{Name: "good:up", Expr: "up", Enable: 1, Labels: model.StringList{"team=after", "url=a=b", "empty="}}}
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	rules := parseRecordingContract(t, storage.values[buildRedisKeyRecordRule("127.0.0.1")])
	want := map[string]string{"team": "after", "url": "a=b", "empty": ""}
	if len(rules) != 1 || !reflect.DeepEqual(rules[0].Labels, want) || storage.writes <= oldWrites || storage.values[recordPublicationKey(pool.ID)] == oldHash {
		t.Fatalf("labels-only change was not published: %+v", rules)
	}
}

func TestRecordingContractHashSeparatorsDoNotCollide(t *testing.T) {
	cache, source, storage, pool := newRecordingContractCache([]*model.MonitorRecordRule{{Name: "a", Expr: "up", Enable: 1}, {Name: "b", Expr: "up", Enable: 1}})
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	oldHash := storage.values[recordPublicationKey(pool.ID)]
	source.rows = []*model.MonitorRecordRule{{Name: "a|up|b", Expr: "up", Enable: 1}}
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	rules := parseRecordingContract(t, storage.values[buildRedisKeyRecordRule("127.0.0.1")])
	if len(rules) != 1 || rules[0].Record != "a|up|b" || storage.values[recordPublicationKey(pool.ID)] == oldHash {
		t.Fatalf("distinct snapshots collided: %+v", rules)
	}
}

func TestRecordingContractMainUsesOneDAOSnapshot(t *testing.T) {
	cache, source, _, _ := newRecordingContractCache([]*model.MonitorRecordRule{{Name: "good:up", Expr: "up", Enable: 1}})
	for iteration := 1; iteration <= 2; iteration++ {
		before := source.calls
		if err := cache.GenerateMainConfig(context.Background()); err != nil {
			t.Fatal(err)
		}
		if reads := source.calls - before; reads != 1 {
			t.Fatalf("refresh %d used %d DAO reads, want one coherent snapshot", iteration, reads)
		}
	}
}
