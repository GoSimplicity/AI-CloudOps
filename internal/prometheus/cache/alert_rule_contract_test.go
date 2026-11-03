package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	common "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
)

type alertingContractSource struct {
	alertdao.AlertManagerRuleDAO
	rows []*model.MonitorAlertRule
}

func (s alertingContractSource) GetMonitorAlertRuleByPoolID(_ context.Context, poolID int) ([]*model.MonitorAlertRule, int64, error) {
	if poolID != 7 {
		return nil, 0, errors.New("unexpected fixture scrape pool")
	}
	return s.rows, int64(len(s.rows)), nil
}
func alertingContractCache(rows []*model.MonitorAlertRule) *alertRuleConfigCache {
	return &alertRuleConfigCache{logger: zap.NewNop(), alertRuleDAO: alertingContractSource{rows: rows}}
}
func alertingContractRow(id int) *model.MonitorAlertRule {
	return &model.MonitorAlertRule{Model: model.Model{ID: id}, PoolID: 7, Name: "服务{告警}", Expr: "up", ForTime: "0s", Enable: 1}
}
func alertingContractParse(t *testing.T, content string) *rulefmt.RuleGroups {
	t.Helper()
	if content == "" {
		t.Fatal("missing generated configuration")
	}
	groups, errs := rulefmt.Parse([]byte(content), false, common.UTF8Validation, parser.NewParser(parser.Options{}))
	if len(errs) > 0 {
		t.Fatalf("actual Prometheus parser rejected complete generated file: %v", errs)
	}
	return groups
}
func TestAlertingContractPreservesValuesAndZeroDuration(t *testing.T) {
	row := alertingContractRow(31)
	row.Labels = model.StringList{"team=ops", "url=https://demo.invalid/?q=a=b,c", "empty="}
	row.Annotations = model.StringList{"summary={{ $labels.team }}=正常,ready", "description=a=b,c"}
	cache := alertingContractCache([]*model.MonitorAlertRule{row})
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, PrometheusInstances: model.StringList{"127.0.0.1"}}
	groups := alertingContractParse(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
	actual := groups.Groups[0].Rules[0]
	if actual.Alert != row.Name || actual.For != 0 || actual.Labels["url"] != "https://demo.invalid/?q=a=b,c" || actual.Labels["empty"] != "" || actual.Annotations["description"] != "a=b,c" {
		t.Fatal("alert construction changed saved values")
	}
	if _, ok := actual.Labels["empty"]; !ok {
		t.Fatal("empty label value was discarded")
	}
}
func TestAlertingContractDuplicateNamesHaveStableUniqueDatabaseIdentity(t *testing.T) {
	first, second := alertingContractRow(31), alertingContractRow(32)
	cache := alertingContractCache([]*model.MonitorAlertRule{first, second})
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, PrometheusInstances: model.StringList{"127.0.0.1"}}
	before := alertingContractParse(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
	if len(before.Groups) != 2 || before.Groups[0].Name == before.Groups[1].Name {
		t.Fatal("same display name collided in generated groups")
	}
	first.Name = "renamed"
	after := alertingContractParse(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
	if before.Groups[0].Name != after.Groups[0].Name || after.Groups[0].Rules[0].Alert != "renamed" {
		t.Fatal("display-name edit changed stable group identity")
	}
}
func TestAlertingContractBadLegacyRowCannotPoisonValidRules(t *testing.T) {
	tests := []struct {
		name string
		bad  *model.MonitorAlertRule
	}{
		{name: "nil", bad: nil},
		{name: "missing ID", bad: alertingContractRow(0)},
		{name: "duration", bad: &model.MonitorAlertRule{Model: model.Model{ID: 32}, PoolID: 7, Enable: 1, Name: "bad", Expr: "up", ForTime: "later"}},
		{name: "matrix", bad: &model.MonitorAlertRule{Model: model.Model{ID: 32}, PoolID: 7, Enable: 1, Name: "bad", Expr: "up[5m]", ForTime: "1m"}},
		{name: "string", bad: &model.MonitorAlertRule{Model: model.Model{ID: 32}, PoolID: 7, Enable: 1, Name: "bad", Expr: `"hello"`, ForTime: "1m"}},
		{name: "label", bad: &model.MonitorAlertRule{Model: model.Model{ID: 32}, PoolID: 7, Enable: 1, Name: "bad", Expr: "up", ForTime: "1m", Labels: model.StringList{"=oops"}}},
		{name: "annotation template", bad: &model.MonitorAlertRule{Model: model.Model{ID: 32}, PoolID: 7, Enable: 1, Name: "bad", Expr: "up", ForTime: "1m", Annotations: model.StringList{"summary={{"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			good := alertingContractRow(31)
			cache := alertingContractCache([]*model.MonitorAlertRule{good, test.bad})
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, PrometheusInstances: model.StringList{"127.0.0.1"}}
			groups := alertingContractParse(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
			if len(groups.Groups) != 1 || groups.Groups[0].Rules[0].Alert != good.Name {
				t.Fatal("bad row leaked or removed valid rule")
			}
		})
	}
}
func TestAlertingContractDuplicateIDsDoNotPoisonCompleteFile(t *testing.T) {
	row := alertingContractRow(31)
	cache := alertingContractCache([]*model.MonitorAlertRule{row, row, alertingContractRow(32)})
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, PrometheusInstances: model.StringList{"127.0.0.1"}}
	groups := alertingContractParse(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
	if len(groups.Groups) != 2 {
		t.Fatal("duplicate database ID was not isolated")
	}
}
func TestAlertingContractAllInvalidRetainsPrePublicationFailureBoundary(t *testing.T) {
	cache := alertingContractCache([]*model.MonitorAlertRule{{Model: model.Model{ID: 31}, PoolID: 7, Enable: 1, Name: "invalid", Expr: "up", ForTime: "later"}})
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, PrometheusInstances: model.StringList{"127.0.0.1"}}
	if actual := cache.GenerateConfigForPool(context.Background(), pool); actual != nil {
		t.Fatal("invalid-only input must not masquerade as a successfully generated empty snapshot")
	}
}
func TestAlertingContractDoesNotMutateSavedRows(t *testing.T) {
	row := alertingContractRow(31)
	row.Labels = model.StringList{"team=ops", "team=last"}
	before := append(model.StringList(nil), row.Labels...)
	cache := alertingContractCache([]*model.MonitorAlertRule{row})
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, PrometheusInstances: model.StringList{"127.0.0.1"}}
	alertingContractParse(t, cache.GenerateConfigForPool(context.Background(), pool)["127.0.0.1"])
	if !reflect.DeepEqual(row.Labels, before) {
		t.Fatal("generation mutated saved labels")
	}
}
