package alert_test

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	alertservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	common "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type namespaceRules struct {
	alertdao.AlertManagerRuleDAO
	row              *model.MonitorAlertRule
	creates, updates int
	requestedPool    int
}

func (s *namespaceRules) CreateMonitorAlertRule(_ context.Context, row *model.MonitorAlertRule) error {
	s.creates++
	copy := *row
	copy.ID = 31
	s.row = &copy
	return nil
}
func (s *namespaceRules) GetMonitorAlertRuleByID(context.Context, int) (*model.MonitorAlertRule, error) {
	if s.row == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *s.row
	return &copy, nil
}
func (s *namespaceRules) UpdateMonitorAlertRule(_ context.Context, req *model.UpdateMonitorAlertRuleReq) error {
	s.updates++
	s.row.PoolID = req.PoolID
	s.row.Expr = req.Expr
	s.row.Name = req.Name
	return nil
}
func (s *namespaceRules) GetMonitorAlertRuleList(context.Context, *model.GetMonitorAlertRuleListReq) ([]*model.MonitorAlertRule, int64, error) {
	if s.row == nil {
		return nil, 0, nil
	}
	copy := *s.row
	return []*model.MonitorAlertRule{&copy}, 1, nil
}
func (s *namespaceRules) GetMonitorAlertRuleByPoolID(_ context.Context, id int) ([]*model.MonitorAlertRule, int64, error) {
	s.requestedPool = id
	if s.row == nil || s.row.PoolID != id || s.row.Enable != 1 {
		return nil, 0, nil
	}
	return []*model.MonitorAlertRule{s.row}, 1, nil
}

type namespacePools struct {
	scrapedao.ScrapePoolDAO
	alertRows map[int]*model.MonitorAlertManagerPool
	rows      map[int]*model.MonitorScrapePool
	lookup    func(int) (*model.MonitorScrapePool, error)
	calls     []int
}

func (s *namespacePools) GetMonitorScrapePoolById(_ context.Context, id int) (*model.MonitorScrapePool, error) {
	s.calls = append(s.calls, id)
	if s.lookup != nil {
		return s.lookup(id)
	}
	row := s.rows[id]
	if row == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

type namespaceSendGroups struct{ alertdao.AlertManagerSendDAO }

func (namespaceSendGroups) GetMonitorSendGroupByID(context.Context, int) (*model.MonitorSendGroup, error) {
	return &model.MonitorSendGroup{}, nil
}

type namespaceMonitorCache struct{}

func (namespaceMonitorCache) MonitorCacheManager(context.Context) error { return nil }
func namespaceService() (alertservice.AlertManagerRuleService, *namespaceRules, *namespacePools) {
	rules := &namespaceRules{}
	pools := &namespacePools{
		alertRows: map[int]*model.MonitorAlertManagerPool{
			99: {Model: model.Model{ID: 99}, Name: "unrelated-alertmanager-99"},
		},
		rows: map[int]*model.MonitorScrapePool{
			7: {
				Model:               model.Model{ID: 7},
				Name:                "actual-scrape-seven",
				SupportAlert:        1,
				PrometheusInstances: model.StringList{"127.0.0.1"},
			},
		},
	}
	return alertservice.NewAlertManagerRuleService(zap.NewNop(), rules, pools, namespaceSendGroups{}, namespaceMonitorCache{}), rules, pools
}
func namespaceCreate(poolID int) *model.CreateMonitorAlertRuleReq {
	return &model.CreateMonitorAlertRuleReq{
		Name: "NamespaceAlert", PoolID: poolID, SendGroupID: 1, Enable: 1,
		Expr: "sum(up)", ForTime: "1m", Labels: model.StringList{"team=ops"},
	}
}
func namespaceUpdate(poolID int) *model.UpdateMonitorAlertRuleReq {
	return &model.UpdateMonitorAlertRuleReq{
		ID: 31, Name: "NamespaceAlert", PoolID: poolID, SendGroupID: 1, Enable: 1,
		Expr: "sum(up)", ForTime: "1m",
	}
}

func TestAlertRuleSaveUsesScrapePoolNamespace(t *testing.T) {
	for _, mode := range []string{"create", "update"} {
		for _, test := range []struct {
			name string
			id   int
			ok   bool
		}{
			{name: "existing Scrape7", id: 7, ok: true},
			{name: "AlertManager99 is not a scrape pool", id: 99},
			{name: "zero", id: 0},
			{name: "negative", id: -1},
		} {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				service, rules, pools := namespaceService()
				if pools.alertRows[99] == nil || pools.rows[99] != nil || pools.rows[7] == nil {
					t.Fatal("non-overlapping namespace fixtures are missing")
				}
				if mode == "update" {
					rules.row = &model.MonitorAlertRule{Model: model.Model{ID: 31}, PoolID: 99, Name: "legacy-unmapped"}
				}
				var err error
				if mode == "create" {
					err = service.CreateMonitorAlertRule(context.Background(), namespaceCreate(test.id))
				} else {
					err = service.UpdateMonitorAlertRule(context.Background(), namespaceUpdate(test.id))
				}
				if (err == nil) != test.ok {
					t.Fatalf("success=%v, want %v", err == nil, test.ok)
				}
				if test.ok {
					if rules.row.PoolID != 7 {
						t.Fatal("explicit scrape selection changed")
					}
				} else if rules.creates+rules.updates != 0 {
					t.Fatal("invalid namespace reached persistence")
				}
				if !test.ok && mode == "update" && rules.row.PoolID != 99 {
					t.Fatal("legacy ID was rewritten without explicit selection")
				}
			})
		}
	}
}
func TestAlertRulePoolLookupFailsClosed(t *testing.T) {
	for _, mode := range []string{"create", "update"} {
		for _, test := range []struct {
			name   string
			lookup func(int) (*model.MonitorScrapePool, error)
		}{
			{name: "nil successful lookup", lookup: func(int) (*model.MonitorScrapePool, error) { return nil, nil }},
			{name: "mismatched entity", lookup: func(int) (*model.MonitorScrapePool, error) {
				return &model.MonitorScrapePool{Model: model.Model{ID: 99}}, nil
			}},
			{name: "database failure", lookup: func(int) (*model.MonitorScrapePool, error) {
				return nil, errors.New("fixture lookup failure")
			}},
		} {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				service, rules, pools := namespaceService()
				pools.lookup = test.lookup
				rules.row = &model.MonitorAlertRule{Model: model.Model{ID: 31}, PoolID: 99}
				var err error
				if mode == "create" {
					err = service.CreateMonitorAlertRule(context.Background(), namespaceCreate(7))
				} else {
					err = service.UpdateMonitorAlertRule(context.Background(), namespaceUpdate(7))
				}
				if err == nil || rules.creates+rules.updates != 0 {
					t.Fatal("bad lookup reached persistence")
				}
			})
		}
	}
}
func TestAlertRuleNamesComeOnlyFromScrapePools(t *testing.T) {
	for _, test := range []struct {
		name   string
		poolID int
		want   string
	}{{"valid scrape", 7, "actual-scrape-seven"}, {"legacy AlertManager-only ID", 99, ""}} {
		t.Run(test.name, func(t *testing.T) {
			service, rules, _ := namespaceService()
			rules.row = &model.MonitorAlertRule{Model: model.Model{ID: 31}, PoolID: test.poolID, PoolName: "old-alertmanager-name", Name: "unchanged"}
			list, err := service.GetMonitorAlertRuleList(context.Background(), &model.GetMonitorAlertRuleListReq{})
			if err != nil || len(list.Items) != 1 || list.Items[0].PoolName != test.want {
				t.Fatal("list borrowed another pool namespace")
			}
			detail, err := service.GetMonitorAlertRule(context.Background(), &model.GetMonitorAlertRuleReq{ID: 31})
			if err != nil || detail.PoolName != test.want || detail.PoolID != test.poolID {
				t.Fatal("detail rewrote legacy identity or borrowed its label")
			}
		})
	}
}
func TestAlertRuleSavedScrapeSevenIsReachableByRealGenerator(t *testing.T) {
	service, rules, pools := namespaceService()
	if err := service.CreateMonitorAlertRule(context.Background(), namespaceCreate(7)); err != nil {
		t.Fatal(err)
	}
	generator := cache.NewAlertRuleConfigCache(zap.NewNop(), pools, rules, nil, nil, nil)
	content := generator.GenerateConfigForPool(context.Background(), pools.rows[7])["127.0.0.1"]
	if rules.requestedPool != 7 || content == "" {
		t.Fatal("saved scrape ID did not reach actual generator")
	}
	groups, errs := rulefmt.Parse([]byte(content), false, common.UTF8Validation, parser.NewParser(parser.Options{}))
	if len(errs) > 0 || len(groups.Groups) != 1 || groups.Groups[0].Rules[0].Expr != "sum(up)" {
		t.Fatalf("generated rule invalid: %v", errs)
	}
}
