package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type monitorPromTask struct {
	PrometheusConfigCache
	calls   atomic.Int32
	failure error
}

func (task *monitorPromTask) GenerateMainConfig(context.Context) error {
	task.calls.Add(1)
	return task.failure
}

type monitorAlertTask struct {
	AlertManagerConfigCache
	calls   atomic.Int32
	failure error
}

func (task *monitorAlertTask) GenerateMainConfig(context.Context) error {
	task.calls.Add(1)
	return task.failure
}

type monitorRuleTask struct {
	AlertRuleConfigCache
	calls   atomic.Int32
	failure error
}

func (task *monitorRuleTask) GenerateMainConfig(context.Context) error {
	task.calls.Add(1)
	return task.failure
}

type monitorRecordTask struct {
	RecordRuleConfigCache
	calls   atomic.Int32
	failure error
}

func (task *monitorRecordTask) GenerateMainConfig(context.Context) error {
	task.calls.Add(1)
	return task.failure
}

func TestMonitorCacheUsesInjectedTaskSwitches(t *testing.T) {
	for _, test := range []struct {
		name                  string
		alert, record         int
		wantAlert, wantRecord int32
	}{
		{"both disabled", 0, 0, 0, 0},
		{"alert only", 1, 0, 1, 0},
		{"record only", 0, 1, 0, 1},
		{"both enabled", 1, 1, 1, 1},
		{"non-one remains disabled", 2, -1, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			// Deliberately contradict the typed flags, including the normal global
			// default 0 when the injected configuration enables a task.
			globalAlert, globalRecord := 1, 1
			if test.alert == 1 {
				globalAlert = 0
			}
			if test.record == 1 {
				globalRecord = 0
			}
			viper.Set("prometheus.enable_alert", globalAlert)
			viper.Set("prometheus.enable_record", globalRecord)
			cfg := &config.Config{Prometheus: config.PrometheusConfig{EnableAlert: test.alert, EnableRecord: test.record}}
			prom, alert, rule, record := &monitorPromTask{}, &monitorAlertTask{}, &monitorRuleTask{}, &monitorRecordTask{}
			cache, err := NewMonitorCache(cfg, prom, alert, rule, record, zap.NewNop())
			if err != nil {
				t.Fatal(err)
			}
			for generation := int32(1); generation <= 2; generation++ {
				if generation == 2 {
					cfg.Prometheus.EnableAlert = globalAlert
					cfg.Prometheus.EnableRecord = globalRecord
					viper.Set("prometheus.enable_alert", globalAlert+10)
					viper.Set("prometheus.enable_record", globalRecord+10)
				}
				if err := cache.MonitorCacheManager(context.Background()); err != nil {
					t.Fatal(err)
				}
				if prom.calls.Load() != generation || alert.calls.Load() != generation {
					t.Fatal("main configuration generation was gated by optional switches")
				}
				if rule.calls.Load() != test.wantAlert*generation || record.calls.Load() != test.wantRecord*generation {
					t.Fatal("optional task selection differs from constructor configuration snapshot")
				}
			}
		})
	}
}

func TestMonitorCachePreservesTaskFailures(t *testing.T) {
	failure := errors.New("fixture task failure")
	for _, test := range []struct {
		name        string
		enabled     int
		wantFailure bool
	}{{"disabled failing rule", 0, false}, {"enabled failing rule", 1, true}} {
		t.Run(test.name, func(t *testing.T) {
			cache, err := NewMonitorCache(&config.Config{Prometheus: config.PrometheusConfig{EnableAlert: test.enabled}}, &monitorPromTask{}, &monitorAlertTask{}, &monitorRuleTask{failure: failure}, &monitorRecordTask{}, zap.NewNop())
			if err != nil {
				t.Fatal(err)
			}
			err = cache.MonitorCacheManager(context.Background())
			if errors.Is(err, failure) != test.wantFailure {
				t.Fatalf("task failure contract differs: %v", err)
			}
		})
	}
}

func TestMonitorCacheRequiresConfiguration(t *testing.T) {
	for _, test := range []struct {
		name  string
		cfg   *config.Config
		valid bool
	}{{"missing", nil, false}, {"provided", &config.Config{}, true}} {
		t.Run(test.name, func(t *testing.T) {
			cache, err := NewMonitorCache(test.cfg, nil, nil, nil, nil, zap.NewNop())
			if (err == nil) != test.valid {
				t.Fatalf("constructor error differs: %v", err)
			}
			if !test.valid && cache != nil {
				t.Fatal("failed constructor returned usable cache")
			}
		})
	}
}
