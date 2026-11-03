package alert

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type recordingValidationDAO struct {
	alertdao.AlertManagerRecordDAO
	reads, checks, creates, updates int
	stored                          *model.MonitorRecordRule
	failure                         string
}

func (d *recordingValidationDAO) CheckMonitorRecordRuleNameExists(context.Context, *model.MonitorRecordRule) (bool, error) {
	d.checks++
	if d.failure == "check" {
		return false, errors.New("fixture lookup failure")
	}
	return d.failure == "conflict", nil
}
func (d *recordingValidationDAO) GetMonitorRecordRuleById(context.Context, int) (*model.MonitorRecordRule, error) {
	d.reads++
	if d.failure == "missing" {
		return nil, gorm.ErrRecordNotFound
	}
	return &model.MonitorRecordRule{Model: model.Model{ID: 17}, Name: "original", Enable: 2}, nil
}
func (d *recordingValidationDAO) CreateMonitorRecordRule(_ context.Context, row *model.MonitorRecordRule) error {
	d.creates++
	if d.failure == "write" {
		return errors.New("fixture write failure")
	}
	d.stored = row
	return nil
}
func (d *recordingValidationDAO) UpdateMonitorRecordRule(_ context.Context, row *model.MonitorRecordRule) error {
	d.updates++
	if d.failure == "write" {
		return errors.New("fixture write failure")
	}
	d.stored = row
	return nil
}

type recordingValidationCache struct {
	cache.MonitorCache
	refreshed chan struct{}
}

func (c *recordingValidationCache) MonitorCacheManager(context.Context) error {
	c.refreshed <- struct{}{}
	return nil
}

func TestRecordingServiceRejectsInvalidConfigurationBeforeDAO(t *testing.T) {
	cases := []struct {
		name       string
		metric     string
		expression string
		labels     model.StringList
		enable     int8
	}{
		{name: "blank", metric: " ", expression: "up", enable: 1},
		{name: "braces", metric: "bad{rule}", expression: "up", enable: 1},
		{name: "syntax", metric: "valid", expression: "sum(", enable: 1},
		{name: "string result", metric: "valid", expression: `("hello")`, enable: 1},
		{name: "matrix result", metric: "valid", expression: "up[5m]", enable: 1},
		{name: "bad label", metric: "valid", expression: "up", labels: model.StringList{"team"}, enable: 1},
		{name: "reserved label", metric: "valid", expression: "up", labels: model.StringList{"__name__=bad"}, enable: 1},
		{name: "invalid state", metric: "valid", expression: "up", enable: 99},
		{name: "negative state", metric: "valid", expression: "up", enable: -1},
	}
	for _, tc := range cases {
		for _, update := range []bool{false, true} {
			mode := "create"
			if update {
				mode = "update"
			}
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				dao := &recordingValidationDAO{}
				svc := NewAlertManagerRecordService(dao, nil, zap.NewNop(), &recordingValidationCache{refreshed: make(chan struct{}, 1)})
				var err error
				if update {
					err = svc.UpdateMonitorRecordRule(context.Background(), &model.UpdateMonitorRecordRuleReq{ID: 17, Name: tc.metric, Expr: tc.expression, Labels: tc.labels, Enable: tc.enable, PoolID: 3})
				} else {
					err = svc.CreateMonitorRecordRule(context.Background(), &model.CreateMonitorRecordRuleReq{Name: tc.metric, Expr: tc.expression, Labels: tc.labels, Enable: tc.enable, PoolID: 3})
				}
				if err == nil || dao.reads != 0 || dao.checks != 0 || dao.creates != 0 || dao.updates != 0 {
					t.Fatal("invalid configuration reached storage")
				}
			})
		}
	}
}

func TestRecordingServicePreservesPayloadAndOptionalState(t *testing.T) {
	for _, update := range []bool{false, true} {
		for _, enable := range []int8{0, 1, 2} {
			for _, labelCase := range []struct {
				name   string
				values model.StringList
			}{
				{name: "omitted", values: nil}, {name: "clear", values: model.StringList{}}, {name: "values", values: model.StringList{"team=ops", "url=a=b"}},
			} {
				labels := labelCase.values
				name := "create"
				if update {
					name = "update"
				}
				t.Run(fmt.Sprintf("%s/enable_%d/labels_%s", name, enable, labelCase.name), func(t *testing.T) {
					dao := &recordingValidationDAO{}
					refreshed := make(chan struct{}, 1)
					svc := NewAlertManagerRecordService(dao, nil, zap.NewNop(), &recordingValidationCache{refreshed: refreshed})
					var err error
					if update {
						err = svc.UpdateMonitorRecordRule(context.Background(), &model.UpdateMonitorRecordRuleReq{ID: 17, Name: "服务:up", Expr: "42", Labels: labels, Enable: enable, PoolID: 3, IpAddress: ""})
					} else {
						err = svc.CreateMonitorRecordRule(context.Background(), &model.CreateMonitorRecordRuleReq{Name: "服务:up", Expr: "42", Labels: labels, Enable: enable, PoolID: 3, UserID: 5, CreateUserName: "fixture"})
					}
					if err != nil || dao.stored == nil {
						t.Fatalf("valid write failed: %v", err)
					}
					wantEnable := enable
					if !update && enable == 0 {
						wantEnable = 1
					}
					if dao.stored.Enable != wantEnable || dao.stored.Name != "服务:up" || dao.stored.Expr != "42" || !reflect.DeepEqual(dao.stored.Labels, labels) {
						t.Fatal("payload, empty/nil labels, or optional state changed")
					}
					// Update's omitted zero is still passed through for GORM's existing
					// zero-value omission; this does not claim a real DB write was run.
					if update && dao.stored.ID != 17 || !update && (dao.stored.UserID != 5 || dao.stored.CreateUserName != "fixture") {
						t.Fatal("saved identity changed")
					}
					select {
					case <-refreshed:
					case <-time.After(2 * time.Second):
						t.Fatal("successful write did not request refresh")
					}
				})
			}
		}
	}
}

func TestRecordingServiceStorageFailuresDoNotRefresh(t *testing.T) {
	for _, failure := range []string{"check", "conflict", "missing", "write"} {
		for _, update := range []bool{false, true} {
			if !update && failure == "missing" {
				continue
			}
			t.Run(failure, func(t *testing.T) {
				dao := &recordingValidationDAO{failure: failure}
				refreshed := make(chan struct{}, 1)
				svc := NewAlertManagerRecordService(dao, nil, zap.NewNop(), &recordingValidationCache{refreshed: refreshed})
				var err error
				if update {
					err = svc.UpdateMonitorRecordRule(context.Background(), &model.UpdateMonitorRecordRuleReq{ID: 17, Name: "updated", Expr: "up", Enable: 1, PoolID: 3})
				} else {
					err = svc.CreateMonitorRecordRule(context.Background(), &model.CreateMonitorRecordRuleReq{Name: "updated", Expr: "up", Enable: 1, PoolID: 3})
				}
				if err == nil || dao.stored != nil {
					t.Fatal("storage failure was accepted")
				}
				select {
				case <-refreshed:
					t.Fatal("failed write refreshed cache")
				default:
				}
			})
		}
	}
}

func TestRecordingCreateBindingStateAndNameBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		metric  string
		enable  int8
		invalid bool
	}{
		{name: "omitted state", metric: "r", enable: 0},
		{name: "enabled", metric: "r", enable: 1},
		{name: "disabled", metric: "r", enable: 2},
		{name: "invalid state", metric: "r", enable: 99, invalid: true},
		{name: "negative state", metric: "r", enable: -1, invalid: true},
		{name: "empty name", metric: "", invalid: true},
		{name: "maximum ASCII name", metric: strings.Repeat("a", 50)},
		{name: "too long ASCII name", metric: strings.Repeat("a", 51), invalid: true},
		{name: "maximum UTF8 name", metric: strings.Repeat("名", 50)},
		{name: "too long UTF8 name", metric: strings.Repeat("名", 51), invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &model.CreateMonitorRecordRuleReq{Name: tc.metric, PoolID: 3, Expr: "up", Enable: tc.enable}
			err := binding.Validator.ValidateStruct(req)
			if (err != nil) != tc.invalid {
				t.Fatal("binding contract mismatch")
			}
		})
	}
}
