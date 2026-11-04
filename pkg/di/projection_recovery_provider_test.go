package di

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

type recoveryProviderJournal struct{}

func (recoveryProviderJournal) CurrentProjectionRecoveryHighWater(context.Context) (uint64, error) {
	return 0, nil
}
func (recoveryProviderJournal) ListCurrentUnprojectedPoolOperations(context.Context, uint64, uint64, int) ([]configdao.PoolReservation, error) {
	return nil, nil
}
func (recoveryProviderJournal) ReadPoolProjectionSourceVersion(context.Context, int8, int) (uint64, error) {
	return 0, nil
}
func (recoveryProviderJournal) ReadAMPoolSourceVersion(context.Context, int) (configdao.AMSourceVersion, error) {
	return configdao.AMSourceVersion{}, nil
}

type recoveryProviderDispatcher struct{}

func (recoveryProviderDispatcher) Recover(context.Context, configdao.PoolReservation) error {
	return nil
}
func TestProjectionRecoveryProviderDefaultsAndIndependentSwitches(t *testing.T) {
	cfg := &config.Config{}
	defaults := cache.DefaultProjectionRecoveryOptions()
	if actual := ProvideProjectionRecoveryOptions(cfg); !reflect.DeepEqual(actual, defaults) {
		t.Fatal("zero config lost bounded defaults", actual)
	}
	cfg.Prometheus.EnableAlert = 0
	cfg.Prometheus.EnableRecord = 0
	options := ProvideProjectionRecoveryOptions(cfg)
	worker, err := cache.NewProjectionRecoveryWorker(recoveryProviderJournal{}, recoveryProviderDispatcher{}, zap.NewNop(), options)
	if err != nil || worker == nil {
		t.Fatal("normal generation switches disabled recovery", err)
	}
	cfg.Prometheus.ProjectionRecoveryInterval = 2 * time.Second
	cfg.Prometheus.ProjectionRecoveryItemTimeout = 3 * time.Second
	cfg.Prometheus.ProjectionRecoveryCycleTimeout = 4 * time.Second
	cfg.Prometheus.ProjectionRecoveryPageSize = 2
	cfg.Prometheus.ProjectionRecoveryMaxPerCycle = 3
	expected := cache.ProjectionRecoveryOptions{Interval: 2 * time.Second, ItemTimeout: 3 * time.Second, CycleTimeout: 4 * time.Second, PageSize: 2, MaxPerCycle: 3}
	if actual := ProvideProjectionRecoveryOptions(cfg); !reflect.DeepEqual(actual, expected) {
		t.Fatal("explicit recovery limits lost", actual)
	}
}
func TestProjectionRecoveryProviderRejectsUnsafeLimits(t *testing.T) {
	for _, mode := range []string{"negative interval", "huge timeout", "huge page", "zero cap override"} {
		t.Run(mode, func(t *testing.T) {
			cfg := &config.Config{}
			switch mode {
			case "negative interval":
				cfg.Prometheus.ProjectionRecoveryInterval = -time.Second
			case "huge timeout":
				cfg.Prometheus.ProjectionRecoveryItemTimeout = time.Hour
			case "huge page":
				cfg.Prometheus.ProjectionRecoveryPageSize = 101
			case "zero cap override":
				cfg.Prometheus.ProjectionRecoveryMaxPerCycle = -1
			}
			if _, err := cache.NewProjectionRecoveryWorker(recoveryProviderJournal{}, recoveryProviderDispatcher{}, zap.NewNop(), ProvideProjectionRecoveryOptions(cfg)); err == nil {
				t.Fatal("invalid configured limits accepted")
			}
		})
	}
}
